/// Projects and Tasks HTTP handlers.
use axum::{
    Json,
    extract::{Path, Query, State},
    http::StatusCode,
    response::{IntoResponse, Response},
};
use serde::{Deserialize, Deserializer, Serialize};
use time::{Date, format_description::well_known::Rfc3339};
use uuid::Uuid;

use crate::http::{AppState, HttpError, middleware::AuthenticatedSession};
use crate::identity::store::{self, StoreError};
use crate::projects::{
    CreateProjectParams, CreateTaskParams, Project, Task, UpdateTaskParams, store as projects_store,
};

#[derive(Serialize)]
pub struct ProjectItem {
    pub id: String,
    pub workspace_id: String,
    pub name: String,
    pub slug: String,
    pub key: String,
    pub description: String,
    pub task_counter: i64,
    pub archived: bool,
    pub created_at: String,
    pub updated_at: String,
}

impl From<Project> for ProjectItem {
    fn from(p: Project) -> Self {
        Self {
            id: p.id.to_string(),
            workspace_id: p.workspace_id.to_string(),
            name: p.name,
            slug: p.slug,
            key: p.key,
            description: p.description,
            task_counter: p.task_counter,
            archived: p.archived,
            created_at: p
                .created_at
                .format(&Rfc3339)
                .unwrap_or_else(|_| p.created_at.to_string()),
            updated_at: p
                .updated_at
                .format(&Rfc3339)
                .unwrap_or_else(|_| p.updated_at.to_string()),
        }
    }
}

#[derive(Serialize)]
pub struct ProjectListResponse {
    pub projects: Vec<ProjectItem>,
}

#[derive(Deserialize)]
pub struct ProjectListQuery {
    pub include_archived: Option<bool>,
}

#[derive(Deserialize)]
pub struct CreateProjectRequest {
    pub name: String,
    pub slug: String,
    pub key: String,
    pub description: Option<String>,
}

#[derive(Serialize)]
pub struct TaskItem {
    pub id: String,
    pub workspace_id: String,
    pub project_id: String,
    pub project_key: String,
    pub task_number: i64,
    pub human_key: String,
    pub title: String,
    pub description: String,
    pub status: String,
    pub priority: String,
    pub due_date: Option<String>,
    pub created_by: Option<String>,
    pub created_at: String,
    pub updated_at: String,
}

impl From<Task> for TaskItem {
    fn from(t: Task) -> Self {
        Self {
            id: t.id.to_string(),
            workspace_id: t.workspace_id.to_string(),
            project_id: t.project_id.to_string(),
            project_key: t.project_key,
            task_number: t.task_number,
            human_key: t.human_key,
            title: t.title,
            description: t.description,
            status: t.status,
            priority: t.priority,
            due_date: t.due_date.map(|d| d.to_string()),
            created_by: t.created_by.map(|u| u.to_string()),
            created_at: t
                .created_at
                .format(&Rfc3339)
                .unwrap_or_else(|_| t.created_at.to_string()),
            updated_at: t
                .updated_at
                .format(&Rfc3339)
                .unwrap_or_else(|_| t.updated_at.to_string()),
        }
    }
}

#[derive(Serialize)]
pub struct PaginatedTasksResponse {
    pub tasks: Vec<TaskItem>,
    pub total: i64,
    pub limit: i64,
    pub offset: i64,
}

#[derive(Deserialize, Default)]
pub struct TaskListQuery {
    pub status: Option<String>,
    pub limit: Option<i64>,
    pub offset: Option<i64>,
}

#[derive(Deserialize)]
pub struct CreateTaskRequest {
    pub title: String,
    pub description: Option<String>,
    pub status: Option<String>,
    pub priority: Option<String>,
    pub due_date: Option<String>,
}

fn deserialize_some<'de, T, D>(deserializer: D) -> Result<Option<Option<T>>, D::Error>
where
    T: Deserialize<'de>,
    D: Deserializer<'de>,
{
    Deserialize::deserialize(deserializer).map(Some)
}

#[derive(Deserialize, Default)]
pub struct UpdateTaskRequest {
    pub title: Option<String>,
    pub description: Option<String>,
    pub status: Option<String>,
    pub priority: Option<String>,
    #[serde(default, deserialize_with = "deserialize_some")]
    pub due_date: Option<Option<String>>,
}

// ---------------------------------------------------------------------------
// Helpers & Validation
// ---------------------------------------------------------------------------

async fn check_membership(
    state: &AppState,
    workspace_id: Uuid,
    user_id: Uuid,
) -> Result<String, HttpError> {
    match store::get_user_membership(&state.pool, workspace_id, user_id).await {
        Ok(Some(role)) => Ok(role),
        Ok(None) => Err(HttpError::NotFound),
        Err(e) => {
            tracing::error!(error = %e, "failed to check workspace membership");
            Err(HttpError::Internal)
        }
    }
}

fn validate_project_slug(slug: &str) -> bool {
    let s = slug.trim();
    if s.len() < 2 || s.len() > 48 {
        return false;
    }
    if s.starts_with('-') || s.ends_with('-') {
        return false;
    }
    s.chars()
        .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == '-')
}

fn validate_project_key(key: &str) -> bool {
    let k = key.trim();
    if k.len() < 2 || k.len() > 8 {
        return false;
    }
    k.chars()
        .all(|c| c.is_ascii_uppercase() || c.is_ascii_digit())
}

fn validate_status(status: &str) -> bool {
    matches!(
        status,
        "backlog" | "unstarted" | "started" | "completed" | "cancelled"
    )
}

fn validate_priority(priority: &str) -> bool {
    matches!(priority, "none" | "low" | "medium" | "high" | "urgent")
}

fn parse_due_date(s: &str) -> Result<Date, HttpError> {
    let format = time::macros::format_description!("[year]-[month]-[day]");
    Date::parse(s.trim(), &format)
        .map_err(|_| HttpError::BadRequest("Invalid due_date format, expected YYYY-MM-DD"))
}

// ---------------------------------------------------------------------------
// Project Handlers
// ---------------------------------------------------------------------------

/// GET /api/v1/workspaces/{workspace_id}/projects
pub async fn list_projects(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path(workspace_id): Path<Uuid>,
    Query(query): Query<ProjectListQuery>,
) -> Response {
    if let Err(err) = check_membership(&state, workspace_id, session.user.id).await {
        return err.into_response();
    }

    let include_archived = query.include_archived.unwrap_or(false);
    match projects_store::list_projects(&state.pool, workspace_id, include_archived).await {
        Ok(projects) => Json(ProjectListResponse {
            projects: projects.into_iter().map(ProjectItem::from).collect(),
        })
        .into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to list projects");
            HttpError::Internal.into_response()
        }
    }
}

/// POST /api/v1/workspaces/{workspace_id}/projects
pub async fn create_project(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path(workspace_id): Path<Uuid>,
    Json(payload): Json<CreateProjectRequest>,
) -> Response {
    let caller_role = match check_membership(&state, workspace_id, session.user.id).await {
        Ok(role) => role,
        Err(err) => return err.into_response(),
    };

    if caller_role != "owner" && caller_role != "admin" {
        return HttpError::Forbidden.into_response();
    }

    let name = payload.name.trim();
    if name.is_empty() || name.len() > 120 {
        return HttpError::BadRequest("Project name must be between 1 and 120 characters")
            .into_response();
    }

    let slug = payload.slug.trim().to_lowercase();
    if !validate_project_slug(&slug) {
        return HttpError::BadRequest(
            "Project slug must be 2-48 lowercase alphanumeric characters or hyphens",
        )
        .into_response();
    }

    let key = payload.key.trim().to_uppercase();
    if !validate_project_key(&key) {
        return HttpError::BadRequest("Project key must be 2-8 uppercase alphanumeric characters")
            .into_response();
    }

    let description = payload.description.unwrap_or_default();
    if description.len() > 2000 {
        return HttpError::BadRequest("Project description cannot exceed 2000 characters")
            .into_response();
    }

    let params = CreateProjectParams {
        name: name.to_string(),
        slug,
        key,
        description,
    };

    match projects_store::create_project(&state.pool, Some(session.user.id), workspace_id, params)
        .await
    {
        Ok(project) => (StatusCode::CREATED, Json(ProjectItem::from(project))).into_response(),
        Err(StoreError::Conflict) => {
            HttpError::Conflict("A project with this slug or key already exists in the workspace")
                .into_response()
        }
        Err(e) => {
            tracing::error!(error = %e, "failed to create project");
            HttpError::Internal.into_response()
        }
    }
}

/// GET /api/v1/workspaces/{workspace_id}/projects/{project_id}
pub async fn get_project(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path((workspace_id, project_id)): Path<(Uuid, Uuid)>,
) -> Response {
    if let Err(err) = check_membership(&state, workspace_id, session.user.id).await {
        return err.into_response();
    }

    match projects_store::get_project(&state.pool, workspace_id, project_id).await {
        Ok(Some(project)) => Json(ProjectItem::from(project)).into_response(),
        Ok(None) => HttpError::NotFound.into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to get project");
            HttpError::Internal.into_response()
        }
    }
}

/// POST /api/v1/workspaces/{workspace_id}/projects/{project_id}/archive
pub async fn archive_project(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path((workspace_id, project_id)): Path<(Uuid, Uuid)>,
) -> Response {
    let caller_role = match check_membership(&state, workspace_id, session.user.id).await {
        Ok(role) => role,
        Err(err) => return err.into_response(),
    };

    if caller_role != "owner" && caller_role != "admin" {
        return HttpError::Forbidden.into_response();
    }

    match projects_store::archive_project(
        &state.pool,
        Some(session.user.id),
        workspace_id,
        project_id,
    )
    .await
    {
        Ok(project) => Json(ProjectItem::from(project)).into_response(),
        Err(StoreError::NotFound) => HttpError::NotFound.into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to archive project");
            HttpError::Internal.into_response()
        }
    }
}

// ---------------------------------------------------------------------------
// Task Handlers
// ---------------------------------------------------------------------------

/// GET /api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks
pub async fn list_tasks(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path((workspace_id, project_id)): Path<(Uuid, Uuid)>,
    Query(query): Query<TaskListQuery>,
) -> Response {
    if let Err(err) = check_membership(&state, workspace_id, session.user.id).await {
        return err.into_response();
    }

    if let Some(ref s) = query.status
        && !validate_status(s)
    {
        return HttpError::BadRequest("Invalid status filter").into_response();
    }

    let limit = query.limit.unwrap_or(20);
    let offset = query.offset.unwrap_or(0);

    match projects_store::list_tasks(
        &state.pool,
        workspace_id,
        project_id,
        query.status.as_deref(),
        limit,
        offset,
    )
    .await
    {
        Ok(paginated) => Json(PaginatedTasksResponse {
            tasks: paginated.tasks.into_iter().map(TaskItem::from).collect(),
            total: paginated.total,
            limit: paginated.limit,
            offset: paginated.offset,
        })
        .into_response(),
        Err(StoreError::NotFound) => HttpError::NotFound.into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to list tasks");
            HttpError::Internal.into_response()
        }
    }
}

/// POST /api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks
pub async fn create_task(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path((workspace_id, project_id)): Path<(Uuid, Uuid)>,
    Json(payload): Json<CreateTaskRequest>,
) -> Response {
    if let Err(err) = check_membership(&state, workspace_id, session.user.id).await {
        return err.into_response();
    }

    let title = payload.title.trim();
    if title.is_empty() || title.len() > 255 {
        return HttpError::BadRequest("Task title must be between 1 and 255 characters")
            .into_response();
    }

    let status = payload.status.unwrap_or_else(|| "unstarted".to_string());
    if !validate_status(&status) {
        return HttpError::BadRequest("Invalid task status").into_response();
    }

    let priority = payload.priority.unwrap_or_else(|| "none".to_string());
    if !validate_priority(&priority) {
        return HttpError::BadRequest("Invalid task priority").into_response();
    }

    let description = payload.description.unwrap_or_default();
    if description.len() > 10000 {
        return HttpError::BadRequest("Task description cannot exceed 10000 characters")
            .into_response();
    }

    let due_date = match payload.due_date {
        Some(s) if !s.trim().is_empty() => match parse_due_date(&s) {
            Ok(d) => Some(d),
            Err(err) => return err.into_response(),
        },
        _ => None,
    };

    let params = CreateTaskParams {
        title: title.to_string(),
        description,
        status,
        priority,
        due_date,
    };

    match projects_store::create_task(
        &state.pool,
        Some(session.user.id),
        workspace_id,
        project_id,
        params,
    )
    .await
    {
        Ok(task) => (StatusCode::CREATED, Json(TaskItem::from(task))).into_response(),
        Err(StoreError::NotFound) => HttpError::NotFound.into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to create task");
            HttpError::Internal.into_response()
        }
    }
}

/// GET /api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}
pub async fn get_task(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path((workspace_id, project_id, task_id)): Path<(Uuid, Uuid, Uuid)>,
) -> Response {
    if let Err(err) = check_membership(&state, workspace_id, session.user.id).await {
        return err.into_response();
    }

    match projects_store::get_task(&state.pool, workspace_id, project_id, task_id).await {
        Ok(Some(task)) => Json(TaskItem::from(task)).into_response(),
        Ok(None) => HttpError::NotFound.into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to get task");
            HttpError::Internal.into_response()
        }
    }
}

/// PATCH /api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}
pub async fn update_task(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path((workspace_id, project_id, task_id)): Path<(Uuid, Uuid, Uuid)>,
    Json(payload): Json<UpdateTaskRequest>,
) -> Response {
    if let Err(err) = check_membership(&state, workspace_id, session.user.id).await {
        return err.into_response();
    }

    if let Some(ref title) = payload.title {
        let t = title.trim();
        if t.is_empty() || t.len() > 255 {
            return HttpError::BadRequest("Task title must be between 1 and 255 characters")
                .into_response();
        }
    }

    if let Some(ref s) = payload.status
        && !validate_status(s)
    {
        return HttpError::BadRequest("Invalid task status").into_response();
    }

    if let Some(ref p) = payload.priority
        && !validate_priority(p)
    {
        return HttpError::BadRequest("Invalid task priority").into_response();
    }

    if let Some(ref d) = payload.description
        && d.len() > 10000
    {
        return HttpError::BadRequest("Task description cannot exceed 10000 characters")
            .into_response();
    }

    let parsed_due_date = match payload.due_date {
        None => None,
        Some(None) => Some(None),
        Some(Some(s)) => {
            if s.trim().is_empty() {
                Some(None)
            } else {
                match parse_due_date(&s) {
                    Ok(d) => Some(Some(d)),
                    Err(err) => return err.into_response(),
                }
            }
        }
    };

    let params = UpdateTaskParams {
        title: payload.title.map(|t| t.trim().to_string()),
        description: payload.description,
        status: payload.status,
        priority: payload.priority,
        due_date: parsed_due_date,
    };

    match projects_store::update_task(
        &state.pool,
        Some(session.user.id),
        workspace_id,
        project_id,
        task_id,
        params,
    )
    .await
    {
        Ok(task) => Json(TaskItem::from(task)).into_response(),
        Err(StoreError::NotFound) => HttpError::NotFound.into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to update task");
            HttpError::Internal.into_response()
        }
    }
}

/// DELETE /api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}
pub async fn delete_task(
    State(state): State<AppState>,
    session: AuthenticatedSession,
    Path((workspace_id, project_id, task_id)): Path<(Uuid, Uuid, Uuid)>,
) -> Response {
    let caller_role = match check_membership(&state, workspace_id, session.user.id).await {
        Ok(role) => role,
        Err(err) => return err.into_response(),
    };

    if caller_role != "owner" && caller_role != "admin" {
        return HttpError::Forbidden.into_response();
    }

    match projects_store::delete_task(
        &state.pool,
        Some(session.user.id),
        workspace_id,
        project_id,
        task_id,
    )
    .await
    {
        Ok(()) => StatusCode::NO_CONTENT.into_response(),
        Err(StoreError::NotFound) => HttpError::NotFound.into_response(),
        Err(e) => {
            tracing::error!(error = %e, "failed to delete task");
            HttpError::Internal.into_response()
        }
    }
}
