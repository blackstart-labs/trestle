/// Session HTTP handlers: POST /api/v1/session (login) and DELETE /api/v1/session (logout).
use axum::{
    Json,
    extract::State,
    http::{HeaderMap, HeaderValue, StatusCode, header},
    response::{IntoResponse, Response},
};
use serde::{Deserialize, Serialize};
use std::net::IpAddr;

use crate::http::AppState;
use crate::identity::{
    LimitResult, Username,
    store::{self, UserProjection},
};

/// Maximum request body size for login (enforced by tower-http layer on the router).
/// Declared here as a constant for documentation; enforcement is in the router.
pub const MAX_LOGIN_BODY_BYTES: usize = 4096;

/// Login request body.
#[derive(Deserialize)]
pub struct LoginRequest {
    pub username: String,
    pub password: String,
}

/// Safe current-user projection for the login response and /me endpoint.
#[derive(Serialize)]
pub struct UserResponse {
    pub id: String,
    pub username: String,
    pub display_name: String,
    pub locale: String,
    pub timezone: String,
}

impl From<UserProjection> for UserResponse {
    fn from(u: UserProjection) -> Self {
        Self {
            id: u.id.to_string(),
            username: u.username,
            display_name: u.display_name,
            locale: u.locale,
            timezone: u.timezone,
        }
    }
}

/// POST /api/v1/session — login
///
/// Returns 200 with user projection and sets HttpOnly session cookie and
/// non-HttpOnly CSRF cookie on success.
/// Returns 401 generically for wrong credentials (same response for unknown
/// user and wrong password; dummy verification equalises timing).
/// Returns 422 for invalid request shape.
/// Returns 429 when the rate limit is exceeded.
pub async fn login(
    State(state): State<AppState>,
    peer_ip: IpAddr,
    headers: HeaderMap,
    body: axum::extract::Json<LoginRequest>,
) -> Response {
    let _ = &headers; // headers available for future use (e.g., origin validation)

    // Validate username format (normalizes to lowercase).
    let username = match Username::parse(&body.username) {
        Ok(u) => u,
        Err(_) => {
            // Invalid username format is a credential failure, not a 422 shape error.
            // Return 401 to avoid disclosing what usernames look like.
            perform_dummy_verification(&state);
            return unauthorized_response();
        }
    };

    // Check rate limit before attempting any credential work.
    if state.limiter.check(peer_ip, username.as_str()) == LimitResult::Exceeded {
        return rate_limited_response();
    }

    // Look up user by canonical username.
    let user_row = match store::find_user_by_username(&state.pool, username.as_str()).await {
        Ok(row) => row,
        Err(_) => {
            perform_dummy_verification(&state);
            return unauthorized_response();
        }
    };

    let (user_id, password_hash) = match user_row {
        Some(row) => row,
        None => {
            // User not found. Perform dummy verification to equalise timing.
            perform_dummy_verification(&state);
            state.limiter.record_failure(peer_ip, username.as_str());
            return unauthorized_response();
        }
    };

    // Verify password (blocking CPU work; run on spawn_blocking to avoid starving Tokio).
    let password_bytes = body.password.as_bytes().to_vec();
    let hasher = state.hasher.clone();
    let hash_copy = password_hash.clone();
    let verified =
        tokio::task::spawn_blocking(move || hasher.verify(&password_bytes, &hash_copy)).await;

    match verified {
        Ok(Ok(true)) => {}
        Ok(Ok(false)) | Ok(Err(_)) | Err(_) => {
            state.limiter.record_failure(peer_ip, username.as_str());
            return unauthorized_response();
        }
    }

    // Create session.
    let (session_token, csrf_token) = match store::create_session(&state.pool, user_id).await {
        Ok(tokens) => tokens,
        Err(e) => {
            tracing::error!(error = %e, "failed to create session");
            return crate::http::HttpError::Internal.into_response();
        }
    };

    // Fetch user projection for response.
    let user = match store::find_user_by_id(&state.pool, user_id).await {
        Ok(u) => u,
        Err(e) => {
            tracing::error!(error = %e, "failed to fetch user after login");
            return crate::http::HttpError::Internal.into_response();
        }
    };

    // Audit successful login.
    let _ = store::record_audit_event(
        &state.pool,
        Some(user_id),
        "session.created",
        &serde_json::json!({"method": "password"}),
    )
    .await;

    // Build cookie values.
    // In production: Secure; HttpOnly; SameSite=Lax; Path=/
    // Development: no Secure flag (when TRESTLE_DEV_MODE=true and loopback)
    let session_cookie = build_session_cookie(&session_token, &state.config);
    let csrf_cookie = build_csrf_cookie(&csrf_token, &state.config);

    let mut response = (StatusCode::OK, Json(UserResponse::from(user))).into_response();
    let headers = response.headers_mut();
    if let Ok(v) = HeaderValue::from_str(&session_cookie) {
        headers.append(header::SET_COOKIE, v);
    }
    if let Ok(v) = HeaderValue::from_str(&csrf_cookie) {
        headers.append(header::SET_COOKIE, v);
    }
    response
}

/// DELETE /api/v1/session — logout (CSRF-protected)
///
/// Revokes the current session. Returns 204 whether or not the session
/// existed (idempotent). CSRF token is validated by the middleware layer.
pub async fn logout(
    State(state): State<AppState>,
    session: crate::http::middleware::AuthenticatedSession,
) -> Response {
    let _ = store::revoke_session(&state.pool, session.session_id).await;
    let _ = store::record_audit_event(
        &state.pool,
        Some(session.user.id),
        "session.revoked",
        &serde_json::json!({}),
    )
    .await;

    // Clear cookies.
    let clear_session = clear_session_cookie(&state.config);
    let clear_csrf = clear_csrf_cookie(&state.config);

    let mut response = StatusCode::NO_CONTENT.into_response();
    let headers = response.headers_mut();
    if let Ok(v) = HeaderValue::from_str(&clear_session) {
        headers.append(header::SET_COOKIE, v);
    }
    if let Ok(v) = HeaderValue::from_str(&clear_csrf) {
        headers.append(header::SET_COOKIE, v);
    }
    response
}

// ---------------------------------------------------------------------------
// Cookie construction
// ---------------------------------------------------------------------------

fn build_session_cookie(token: &str, config: &crate::config::Config) -> String {
    let secure = if config.dev_mode { "" } else { "; Secure" };
    format!("trestle-session={token}; HttpOnly{secure}; SameSite=Lax; Path=/; Max-Age=604800")
}

fn build_csrf_cookie(token: &str, config: &crate::config::Config) -> String {
    // Not HttpOnly: JavaScript reads this and forwards in X-CSRF-Token.
    let secure = if config.dev_mode { "" } else { "; Secure" };
    format!("trestle-csrf={token}{secure}; SameSite=Lax; Path=/; Max-Age=604800")
}

fn clear_session_cookie(config: &crate::config::Config) -> String {
    let secure = if config.dev_mode { "" } else { "; Secure" };
    format!("trestle-session=; HttpOnly{secure}; SameSite=Lax; Path=/; Max-Age=0")
}

fn clear_csrf_cookie(config: &crate::config::Config) -> String {
    let secure = if config.dev_mode { "" } else { "; Secure" };
    format!("trestle-csrf=; SameSite=Lax{secure}; Path=/; Max-Age=0")
}

// ---------------------------------------------------------------------------
// Standard error responses
// ---------------------------------------------------------------------------

fn unauthorized_response() -> Response {
    (
        StatusCode::UNAUTHORIZED,
        Json(serde_json::json!({
            "error": {
                "code": "unauthorized",
                "message": "Invalid credentials"
            }
        })),
    )
        .into_response()
}

fn rate_limited_response() -> Response {
    let mut response = (
        StatusCode::TOO_MANY_REQUESTS,
        Json(serde_json::json!({
            "error": {
                "code": "rate_limited",
                "message": "Too many failed login attempts, please wait before trying again"
            }
        })),
    )
        .into_response();
    // Provide a generic retry indication. 900 seconds = 15 minutes.
    response
        .headers_mut()
        .insert("retry-after", HeaderValue::from_static("900"));
    response
}

fn perform_dummy_verification(state: &AppState) {
    state.hasher.dummy_verify();
}
