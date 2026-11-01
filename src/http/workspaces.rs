/// GET /api/v1/workspaces — authorized workspace list for the current user.
use axum::{Json, extract::State, response::IntoResponse};
use serde::Serialize;

use crate::http::{AppState, middleware::AuthenticatedSession};
use crate::identity::store::{self, WorkspaceProjection};

#[derive(Serialize)]
struct WorkspaceListResponse {
    workspaces: Vec<WorkspaceItem>,
}

#[derive(Serialize)]
struct WorkspaceItem {
    id: String,
    name: String,
    slug: String,
    role: String,
}

impl From<WorkspaceProjection> for WorkspaceItem {
    fn from(w: WorkspaceProjection) -> Self {
        Self {
            id: w.id.to_string(),
            name: w.name,
            slug: w.slug,
            role: w.role,
        }
    }
}

/// Return the list of workspaces the authenticated user is a member of.
///
/// Returns 401 if no valid session is present (via extractor).
/// Returns an empty list if the user has no workspace memberships.
/// Does not disclose workspaces the user is not a member of.
pub async fn list_workspaces(
    State(state): State<AppState>,
    session: AuthenticatedSession,
) -> impl IntoResponse {
    match store::list_user_workspaces(&state.pool, session.user.id).await {
        Ok(workspaces) => Json(WorkspaceListResponse {
            workspaces: workspaces.into_iter().map(WorkspaceItem::from).collect(),
        })
        .into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to list workspaces");
            crate::http::HttpError::Internal.into_response()
        }
    }
}
