use axum::{
    Json, Router,
    extract::{MatchedPath, Request, State},
    http::{HeaderValue, StatusCode},
    middleware::{self, Next},
    response::{IntoResponse, Response},
    routing::get,
};
use serde::Serialize;
use sqlx::PgPool;
use std::time::{Duration, Instant};
use tracing::Instrument;

#[derive(Serialize)]
struct Health {
    status: &'static str,
}

#[derive(Clone)]
struct AppState {
    pool: PgPool,
    timeout: Duration,
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

pub fn router(pool: PgPool, timeout: Duration) -> Router {
    Router::new()
        .merge(crate::assets::router())
        .route(
            "/health/live",
            get(|| async { Json(Health { status: "ok" }) }),
        )
        .route("/health/ready", get(readiness))
        .fallback(|| async { HttpError::NotFound })
        .method_not_allowed_fallback(|| async { HttpError::MethodNotAllowed })
        .with_state(AppState { pool, timeout })
        .layer(middleware::from_fn(request_diagnostics_and_headers))
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
