/// Workspace and membership HTTP handlers.
use axum::{
    Json,
    extract::{Path, State},
    http::StatusCode,
    response::{IntoResponse, Response},
};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::http::{AppState, ErrorBody, ErrorDetail, HttpError, middleware::AuthenticatedSession};
use crate::identity::{
    Username,
    store::{self, MemberProjection, StoreError, WorkspaceProjection},
};

#[derive(Serialize)]
pub struct WorkspaceListResponse {
    pub workspaces: Vec<WorkspaceItem>,
}

#[derive(Serialize)]
pub struct WorkspaceItem {
    pub id: String,
    pub name: String,
    pub slug: String,
    pub role: String,
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

#[derive(Serialize)]
pub struct MemberListResponse {
    pub members: Vec<MemberItem>,
}

#[derive(Serialize)]
pub struct MemberItem {
    pub user_id: String,
    pub username: String,
    pub display_name: String,
    pub role: String,
}

impl From<MemberProjection> for MemberItem {
    fn from(m: MemberProjection) -> Self {
        Self {
            user_id: m.user_id.to_string(),
            username: m.username,
            display_name: m.display_name,
            role: m.role,
        }
    }
}

#[derive(Deserialize)]
pub struct AddMemberRequest {
    pub username: String,
    pub role: String,
}

#[derive(Deserialize)]
pub struct UpdateMemberRoleRequest {
    pub role: String,
}

#[derive(Deserialize)]
pub struct TransferOwnershipRequest {
    pub target_user_id: Uuid,
    pub password: String,
}

/// GET /api/v1/workspaces — return list of workspaces for the current user.
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
            HttpError::Internal.into_response()
        }
    }
}

/// GET /api/v1/workspaces/{workspace_id}/members — list workspace members.
///
/// Conceals 404 if the workspace does not exist or caller is not a member.
pub async fn list_members(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path(workspace_id): Path<Uuid>,
) -> Response {
    let caller_role =
        match store::get_user_membership(&state.pool, workspace_id, session.user.id).await {
            Ok(Some(role)) => role,
            Ok(None) => return HttpError::NotFound.into_response(),
            Err(e) => {
                tracing::error!(error = %e, "failed to check workspace membership");
                return HttpError::Internal.into_response();
            }
        };

    let _ = caller_role;
    match store::list_workspace_members(&state.pool, workspace_id).await {
        Ok(members) => Json(MemberListResponse {
            members: members.into_iter().map(MemberItem::from).collect(),
        })
        .into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to list workspace members");
            HttpError::Internal.into_response()
        }
    }
}

/// POST /api/v1/workspaces/{workspace_id}/members — add a member to the workspace.
pub async fn add_member(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path(workspace_id): Path<Uuid>,
    Json(payload): Json<AddMemberRequest>,
) -> Response {
    let caller_role =
        match store::get_user_membership(&state.pool, workspace_id, session.user.id).await {
            Ok(Some(role)) => role,
            Ok(None) => return HttpError::NotFound.into_response(),
            Err(e) => {
                tracing::error!(error = %e, "failed to check caller membership");
                return HttpError::Internal.into_response();
            }
        };

    if caller_role != "owner" && caller_role != "admin" {
        return HttpError::Forbidden.into_response();
    }

    if payload.role != "admin" && payload.role != "member" && payload.role != "guest" {
        return HttpError::BadRequest("Role must be admin, member, or guest").into_response();
    }

    let canonical_username = match Username::parse(&payload.username) {
        Ok(u) => u,
        Err(_) => return HttpError::BadRequest("Invalid username format").into_response(),
    };

    let target_user =
        match store::find_user_projection_by_username(&state.pool, canonical_username.as_str())
            .await
        {
            Ok(Some(u)) => u,
            Ok(None) => return HttpError::NotFound.into_response(),
            Err(e) => {
                tracing::error!(error = %e, "failed to find user by username");
                return HttpError::Internal.into_response();
            }
        };

    if target_user.disabled {
        return HttpError::BadRequest("Cannot add a disabled user").into_response();
    }

    match store::add_workspace_member(
        &state.pool,
        Some(session.user.id),
        workspace_id,
        target_user.id,
        &payload.role,
    )
    .await
    {
        Ok(member) => (StatusCode::CREATED, Json(MemberItem::from(member))).into_response(),
        Err(StoreError::Conflict) => {
            HttpError::Conflict("User is already a member of this workspace").into_response()
        }
        Err(StoreError::Forbidden) => HttpError::Forbidden.into_response(),
        Err(StoreError::NotFound) => HttpError::NotFound.into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to add member");
            HttpError::Internal.into_response()
        }
    }
}

/// PATCH /api/v1/workspaces/{workspace_id}/members/{user_id} — update a member's role.
pub async fn update_member_role(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path((workspace_id, user_id)): Path<(Uuid, Uuid)>,
    Json(payload): Json<UpdateMemberRoleRequest>,
) -> Response {
    let caller_role =
        match store::get_user_membership(&state.pool, workspace_id, session.user.id).await {
            Ok(Some(role)) => role,
            Ok(None) => return HttpError::NotFound.into_response(),
            Err(e) => {
                tracing::error!(error = %e, "failed to check caller membership");
                return HttpError::Internal.into_response();
            }
        };

    if caller_role != "owner" && caller_role != "admin" {
        return HttpError::Forbidden.into_response();
    }

    if payload.role != "admin" && payload.role != "member" && payload.role != "guest" {
        return HttpError::BadRequest("Role must be admin, member, or guest").into_response();
    }

    if caller_role == "admin" {
        let target_role = match store::get_user_membership(&state.pool, workspace_id, user_id).await
        {
            Ok(Some(role)) => role,
            Ok(None) => return HttpError::NotFound.into_response(),
            Err(e) => {
                tracing::error!(error = %e, "failed to check target membership");
                return HttpError::Internal.into_response();
            }
        };

        if target_role == "owner" {
            return HttpError::Forbidden.into_response();
        }
    }

    match store::update_workspace_member_role(
        &state.pool,
        Some(session.user.id),
        workspace_id,
        user_id,
        &payload.role,
    )
    .await
    {
        Ok(member) => (StatusCode::OK, Json(MemberItem::from(member))).into_response(),
        Err(StoreError::Conflict) => {
            HttpError::Conflict("Cannot demote the last active owner").into_response()
        }
        Err(StoreError::NotFound) => HttpError::NotFound.into_response(),
        Err(StoreError::Forbidden) => HttpError::Forbidden.into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to update member role");
            HttpError::Internal.into_response()
        }
    }
}

/// DELETE /api/v1/workspaces/{workspace_id}/members/{user_id} — remove a member.
pub async fn remove_member(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path((workspace_id, user_id)): Path<(Uuid, Uuid)>,
) -> Response {
    let caller_role =
        match store::get_user_membership(&state.pool, workspace_id, session.user.id).await {
            Ok(Some(role)) => role,
            Ok(None) => return HttpError::NotFound.into_response(),
            Err(e) => {
                tracing::error!(error = %e, "failed to check caller membership");
                return HttpError::Internal.into_response();
            }
        };

    let is_self = session.user.id == user_id;
    if !is_self {
        if caller_role != "owner" && caller_role != "admin" {
            return HttpError::Forbidden.into_response();
        }

        if caller_role == "admin" {
            let target_role =
                match store::get_user_membership(&state.pool, workspace_id, user_id).await {
                    Ok(Some(role)) => role,
                    Ok(None) => return HttpError::NotFound.into_response(),
                    Err(e) => {
                        tracing::error!(error = %e, "failed to check target membership");
                        return HttpError::Internal.into_response();
                    }
                };

            if target_role == "owner" {
                return HttpError::Forbidden.into_response();
            }
        }
    }

    match store::remove_workspace_member(&state.pool, Some(session.user.id), workspace_id, user_id)
        .await
    {
        Ok(()) => StatusCode::NO_CONTENT.into_response(),
        Err(StoreError::Conflict) => {
            HttpError::Conflict("Cannot remove the last active owner").into_response()
        }
        Err(StoreError::NotFound) => HttpError::NotFound.into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to remove member");
            HttpError::Internal.into_response()
        }
    }
}

/// POST /api/v1/workspaces/{workspace_id}/transfer-ownership — transfer ownership.
pub async fn transfer_ownership(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path(workspace_id): Path<Uuid>,
    Json(payload): Json<TransferOwnershipRequest>,
) -> Response {
    let caller_role =
        match store::get_user_membership(&state.pool, workspace_id, session.user.id).await {
            Ok(Some(role)) => role,
            Ok(None) => return HttpError::NotFound.into_response(),
            Err(e) => {
                tracing::error!(error = %e, "failed to check caller membership");
                return HttpError::Internal.into_response();
            }
        };

    if caller_role != "owner" {
        return HttpError::Forbidden.into_response();
    }

    // Reauthentication: verify caller password
    let user_row = match store::find_user_by_username(&state.pool, &session.user.username).await {
        Ok(Some(row)) => row,
        _ => return HttpError::Internal.into_response(),
    };

    let password_valid = state
        .hasher
        .verify(payload.password.as_bytes(), &user_row.1)
        .unwrap_or_default();

    if !password_valid {
        return (
            StatusCode::UNAUTHORIZED,
            Json(ErrorBody {
                error: ErrorDetail {
                    code: "unauthorized",
                    message: "Invalid password confirmation",
                },
            }),
        )
            .into_response();
    }

    match store::transfer_workspace_ownership(
        &state.pool,
        Some(session.user.id),
        workspace_id,
        session.user.id,
        payload.target_user_id,
    )
    .await
    {
        Ok(()) => (StatusCode::OK, Json(serde_json::json!({ "status": "ok" }))).into_response(),
        Err(StoreError::NotFound) => HttpError::NotFound.into_response(),
        Err(StoreError::Forbidden) => HttpError::Forbidden.into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to transfer ownership");
            HttpError::Internal.into_response()
        }
    }
}
