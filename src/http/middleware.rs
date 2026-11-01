/// Session resolution and CSRF validation middleware.
///
/// Extracts the session cookie, resolves it against the database, and
/// injects an `AuthenticatedSession` into request extensions.
///
/// Handlers that require authentication use the `AuthenticatedSession` extractor.
/// If the session is absent or invalid, they return 401 automatically.
use axum::{
    extract::{FromRef, FromRequestParts, State},
    http::{Request, StatusCode, request::Parts},
    middleware::Next,
    response::{IntoResponse, Response},
};
use uuid::Uuid;

use crate::http::AppState;
use crate::identity::store::UserProjection;

/// An authenticated session injected by the session middleware.
///
/// Only present when a valid, unexpired, unrevoked session was resolved.
/// Handlers extract this to access the current user and session ID.
#[derive(Clone, Debug)]
pub struct AuthenticatedSession {
    pub session_id: Uuid,
    pub user: UserProjection,
    /// Raw CSRF token hash for validation in CSRF-protected handlers.
    pub csrf_token_hash: String,
}

/// Axum extractor: resolves the authenticated session from request extensions.
///
/// Returns 401 if no session is present, the session is expired/revoked,
/// or the user is disabled.
impl<S> FromRequestParts<S> for AuthenticatedSession
where
    AppState: FromRef<S>,
    S: Send + Sync,
{
    type Rejection = Response;

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        parts
            .extensions
            .get::<AuthenticatedSession>()
            .cloned()
            .ok_or_else(|| {
                (
                    StatusCode::UNAUTHORIZED,
                    axum::Json(serde_json::json!({
                        "error": {
                            "code": "unauthorized",
                            "message": "Authentication required"
                        }
                    })),
                )
                    .into_response()
            })
    }
}

pub async fn resolve_session_middleware(
    State(state): State<AppState>,
    mut request: Request<axum::body::Body>,
    next: Next,
) -> Response {
    let token = request
        .headers()
        .get(axum::http::header::COOKIE)
        .and_then(|h| h.to_str().ok())
        .and_then(|c| extract_cookie_value(c, "trestle-session"))
        .map(|s| s.to_owned());

    if let Some(token) = token {
        match crate::identity::store::resolve_session(&state.pool, &token).await {
            Ok(Some(resolved)) => {
                request.extensions_mut().insert(AuthenticatedSession {
                    session_id: resolved.session_id,
                    user: resolved.user,
                    csrf_token_hash: resolved.csrf_token_hash,
                });
            }
            Ok(None) => {}
            Err(e) => {
                tracing::warn!(error = %e, "session resolution database error");
            }
        }
    }

    next.run(request).await
}

/// Extract a named cookie value from the Cookie header string.
fn extract_cookie_value<'a>(cookie_str: &'a str, name: &str) -> Option<&'a str> {
    for part in cookie_str.split(';') {
        let part = part.trim();
        if let Some(value) = part.strip_prefix(name).and_then(|v| v.strip_prefix('=')) {
            return Some(value);
        }
    }
    None
}

/// Middleware: validate CSRF token for mutating cookie-authenticated requests.
///
/// Checks the `X-CSRF-Token` header against the stored CSRF token hash.
/// Must run after session resolution so the CSRF hash is available.
///
/// Only applied to routes that need CSRF protection (POST, PUT, PATCH, DELETE
/// on authenticated endpoints). GET/HEAD/OPTIONS are safe and not checked.
pub async fn csrf_middleware(
    State(state): State<AppState>,
    request: Request<axum::body::Body>,
    next: Next,
) -> Response {
    let method = request.method().clone();
    // Only validate CSRF for state-changing methods.
    if matches!(
        method,
        axum::http::Method::POST
            | axum::http::Method::PUT
            | axum::http::Method::PATCH
            | axum::http::Method::DELETE
    ) {
        // If there's no authenticated session, session-based CSRF doesn't apply.
        // The endpoint handler will reject unauthenticated requests.
        if let Some(session) = request.extensions().get::<AuthenticatedSession>() {
            let csrf_header = request
                .headers()
                .get("x-csrf-token")
                .and_then(|v| v.to_str().ok());

            let valid = match csrf_header {
                Some(client_token) => {
                    match crate::identity::hash_client_token(client_token) {
                        Ok(client_hash) => {
                            // Constant-time comparison to prevent timing attacks.
                            constant_time_eq(client_hash.as_str(), &session.csrf_token_hash)
                        }
                        Err(_) => false,
                    }
                }
                None => false,
            };

            if !valid {
                let _ = &state; // suppress unused warning
                return (
                    StatusCode::FORBIDDEN,
                    axum::Json(serde_json::json!({
                        "error": {
                            "code": "csrf_invalid",
                            "message": "Invalid or missing CSRF token"
                        }
                    })),
                )
                    .into_response();
            }
        }
    }
    next.run(request).await
}

/// Constant-time string comparison to prevent timing attacks on CSRF validation.
fn constant_time_eq(a: &str, b: &str) -> bool {
    if a.len() != b.len() {
        return false;
    }
    a.bytes()
        .zip(b.bytes())
        .fold(0u8, |acc, (x, y)| acc | (x ^ y))
        == 0
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn extract_cookie_value_finds_named_cookie() {
        let cookies = "other=abc; trestle-session=mytoken123; another=xyz";
        assert_eq!(
            extract_cookie_value(cookies, "trestle-session"),
            Some("mytoken123")
        );
    }

    #[test]
    fn extract_cookie_value_returns_none_when_absent() {
        let cookies = "other=abc; another=xyz";
        assert_eq!(extract_cookie_value(cookies, "trestle-session"), None);
    }

    #[test]
    fn constant_time_eq_matches_equal_strings() {
        assert!(constant_time_eq("abc123", "abc123"));
    }

    #[test]
    fn constant_time_eq_rejects_different_strings() {
        assert!(!constant_time_eq("abc123", "abc124"));
    }

    #[test]
    fn constant_time_eq_rejects_different_lengths() {
        assert!(!constant_time_eq("abc", "abcd"));
    }

    #[test]
    fn constant_time_eq_rejects_empty_vs_nonempty() {
        assert!(!constant_time_eq("", "a"));
        assert!(!constant_time_eq("a", ""));
    }

    #[test]
    fn constant_time_eq_accepts_empty_empty() {
        assert!(constant_time_eq("", ""));
    }
}
