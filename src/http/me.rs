/// GET /api/v1/me — current authenticated user.
use axum::{Json, response::IntoResponse};

use crate::http::{middleware::AuthenticatedSession, session::UserResponse};

/// Return the current authenticated user's safe projection.
/// Returns 401 if there is no valid session (via the extractor).
pub async fn me(session: AuthenticatedSession) -> impl IntoResponse {
    Json(UserResponse::from(session.user))
}
