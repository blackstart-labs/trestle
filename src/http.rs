pub mod me;
pub mod middleware;
pub mod projects;
pub mod session;
pub mod workspaces;

use axum::{
    Json, Router,
    extract::{MatchedPath, Request, State},
    http::{HeaderValue, StatusCode},
    middleware::Next,
    response::{IntoResponse, Response},
    routing::{delete, get, post},
};
use serde::Serialize;
use sqlx::PgPool;
use std::time::{Duration, Instant};
use tracing::Instrument;

use crate::config::Config;
use crate::identity::{Hasher, LoginLimiter};

#[derive(Serialize)]
struct Health {
    status: &'static str,
}

/// Shared application state injected into all handlers.
#[derive(Clone)]
pub struct AppState {
    pub pool: PgPool,
    pub timeout: Duration,
    pub config: Config,
    pub hasher: Hasher,
    pub limiter: LoginLimiter,
}

#[derive(Serialize)]
struct ErrorBody {
    error: ErrorDetail,
}

#[derive(Serialize)]
struct ErrorDetail {
    code: &'static str,
    message: &'static str,
}

/// Public errors deliberately carry no raw database or secret-bearing messages.
pub enum HttpError {
    NotFound,
    MethodNotAllowed,
    Unavailable,
    Internal,
    Forbidden,
    Conflict(&'static str),
    BadRequest(&'static str),
}

impl IntoResponse for HttpError {
    fn into_response(self) -> Response {
        let (status, code, message) = match self {
            Self::NotFound => (StatusCode::NOT_FOUND, "not_found", "Not found"),
            Self::MethodNotAllowed => (
                StatusCode::METHOD_NOT_ALLOWED,
                "method_not_allowed",
                "Method not allowed",
            ),
            Self::Unavailable => (
                StatusCode::SERVICE_UNAVAILABLE,
                "unavailable",
                "Service unavailable",
            ),
            Self::Internal => (
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal_error",
                "Internal server error",
            ),
            Self::Forbidden => (StatusCode::FORBIDDEN, "forbidden", "Forbidden"),
            Self::Conflict(msg) => (StatusCode::CONFLICT, "conflict", msg),
            Self::BadRequest(msg) => (StatusCode::BAD_REQUEST, "bad_request", msg),
        };
        (
            status,
            Json(ErrorBody {
                error: ErrorDetail { code, message },
            }),
        )
            .into_response()
    }
}

pub fn router(pool: PgPool, timeout: Duration, config: Config) -> Router {
    let hasher = Hasher::new(
        config.argon2_m_cost,
        config.argon2_t_cost,
        config.argon2_parallelism,
    )
    .unwrap_or_else(|_| {
        tracing::warn!("argon2 parameters invalid, using defaults");
        Hasher::default_params()
    });

    let state = AppState {
        pool: pool.clone(),
        timeout,
        config,
        hasher,
        limiter: LoginLimiter::new(),
    };

    Router::new()
        .merge(crate::assets::router())
        .route(
            "/health/live",
            get(|| async { Json(Health { status: "ok" }) }),
        )
        .route("/health/ready", get(readiness))
        // Identity API
        .route("/api/v1/session", post(session_post))
        .route("/api/v1/session", delete(session_delete))
        .route("/api/v1/me", get(me::me))
        .route("/api/v1/workspaces", get(workspaces::list_workspaces))
        .route(
            "/api/v1/workspaces/{workspace_id}/members",
            get(workspaces::list_members).post(workspaces::add_member),
        )
        .route(
            "/api/v1/workspaces/{workspace_id}/members/{user_id}",
            axum::routing::patch(workspaces::update_member_role).delete(workspaces::remove_member),
        )
        .route(
            "/api/v1/workspaces/{workspace_id}/transfer-ownership",
            post(workspaces::transfer_ownership),
        )
        // Projects & Tasks API
        .route(
            "/api/v1/workspaces/{workspace_id}/projects",
            get(projects::list_projects).post(projects::create_project),
        )
        .route(
            "/api/v1/workspaces/{workspace_id}/projects/{project_id}",
            get(projects::get_project),
        )
        .route(
            "/api/v1/workspaces/{workspace_id}/projects/{project_id}/archive",
            post(projects::archive_project),
        )
        .route(
            "/api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks",
            get(projects::list_tasks).post(projects::create_task),
        )
        .route(
            "/api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}",
            get(projects::get_task)
                .patch(projects::update_task)
                .delete(projects::delete_task),
        )
        .fallback(|| async { HttpError::NotFound })
        .method_not_allowed_fallback(|| async { HttpError::MethodNotAllowed })
        .with_state(state.clone())
        .layer(axum::middleware::from_fn_with_state(
            state.clone(),
            self::middleware::csrf_middleware,
        ))
        .layer(axum::middleware::from_fn_with_state(
            state.clone(),
            self::middleware::resolve_session_middleware,
        ))
        .layer(axum::middleware::from_fn(request_diagnostics_and_headers))
}

pub struct ClientIp(pub std::net::IpAddr);

impl<S: Sync> axum::extract::FromRequestParts<S> for ClientIp {
    type Rejection = std::convert::Infallible;

    async fn from_request_parts(
        parts: &mut axum::http::request::Parts,
        _state: &S,
    ) -> Result<Self, Self::Rejection> {
        let ip = parts
            .extensions
            .get::<axum::extract::ConnectInfo<std::net::SocketAddr>>()
            .map(|ci| ci.0.ip())
            .unwrap_or(std::net::IpAddr::V4(std::net::Ipv4Addr::LOCALHOST));
        Ok(ClientIp(ip))
    }
}

async fn session_post(
    State(state): State<AppState>,
    ClientIp(peer_ip): ClientIp,
    headers: axum::http::HeaderMap,
    body: axum::extract::Json<session::LoginRequest>,
) -> Response {
    session::login(State(state), peer_ip, headers, body).await
}

/// Logout: requires authenticated session (validated by extractor).
async fn session_delete(
    State(state): State<AppState>,
    authenticated: self::middleware::AuthenticatedSession,
) -> Response {
    session::logout(State(state), authenticated).await
}

async fn readiness(State(state): State<AppState>) -> Response {
    match tokio::time::timeout(state.timeout, sqlx::query("SELECT 1").execute(&state.pool)).await {
        Ok(Ok(_)) => Json(Health { status: "ok" }).into_response(),
        _ => {
            tracing::warn!("database readiness check failed");
            HttpError::Unavailable.into_response()
        }
    }
}

async fn request_diagnostics_and_headers(request: Request, next: Next) -> Response {
    let route = request
        .extensions()
        .get::<MatchedPath>()
        .map(MatchedPath::as_str)
        .unwrap_or("<unmatched>");
    let span = tracing::info_span!("http_request", method = %request.method(), route);
    let started = Instant::now();
    let mut response = next.run(request).instrument(span.clone()).await;
    tracing::info!(parent: &span, status = response.status().as_u16(), elapsed_ms = started.elapsed().as_millis() as u64, "request complete");
    response
        .headers_mut()
        .insert("cache-control", HeaderValue::from_static("no-store"));
    response.headers_mut().insert(
        "x-content-type-options",
        HeaderValue::from_static("nosniff"),
    );
    response.headers_mut().insert(
        "content-security-policy",
        HeaderValue::from_static("default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'none'"),
    );
    response
}
