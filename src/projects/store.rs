/// Persistence and query logic for projects and tasks.
use serde::Serialize;
use sqlx::PgPool;
use time::{Date, OffsetDateTime};
use uuid::Uuid;

use crate::identity::store::StoreError;

#[derive(Debug, Clone, Serialize)]
pub struct Project {
    pub id: Uuid,
    pub workspace_id: Uuid,
    pub name: String,
    pub slug: String,
    pub key: String,
    pub description: String,
    pub task_counter: i64,
    pub archived: bool,
    pub created_at: OffsetDateTime,
    pub updated_at: OffsetDateTime,
}

#[derive(Debug, Clone, Serialize)]
pub struct Task {
    pub id: Uuid,
    pub workspace_id: Uuid,
    pub project_id: Uuid,
    pub project_key: String,
    pub task_number: i64,
    pub human_key: String,
    pub title: String,
    pub description: String,
    pub status: String,
    pub priority: String,
    pub due_date: Option<Date>,
    pub created_by: Option<Uuid>,
    pub created_at: OffsetDateTime,
    pub updated_at: OffsetDateTime,
}

#[derive(Debug, Clone, Serialize)]
pub struct PaginatedTasks {
    pub tasks: Vec<Task>,
    pub total: i64,
    pub limit: i64,
    pub offset: i64,
}

#[derive(Debug, Clone)]
pub struct CreateProjectParams {
    pub name: String,
    pub slug: String,
    pub key: String,
    pub description: String,
}

#[derive(Debug, Clone)]
pub struct CreateTaskParams {
    pub title: String,
    pub description: String,
    pub status: String,
    pub priority: String,
    pub due_date: Option<Date>,
}

#[derive(Debug, Clone, Default)]
pub struct UpdateTaskParams {
    pub title: Option<String>,
    pub description: Option<String>,
    pub status: Option<String>,
    pub priority: Option<String>,
    pub due_date: Option<Option<Date>>,
}

fn map_sqlx(err: sqlx::Error) -> StoreError {
    if let Some(db_err) = err.as_database_error()
        && db_err.code().as_deref() == Some("23505")
    {
        return StoreError::Conflict;
    }
    StoreError::Internal(err)
}

// ---------------------------------------------------------------------------
// Projects
// ---------------------------------------------------------------------------

pub async fn create_project(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    params: CreateProjectParams,
) -> Result<Project, StoreError> {
    let project_id = Uuid::new_v4();
    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    let now = OffsetDateTime::now_utc();

    sqlx::query(
        "INSERT INTO projects (id, workspace_id, name, slug, key, description, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $7)",
    )
    .bind(project_id)
    .bind(workspace_id)
    .bind(&params.name)
    .bind(&params.slug)
    .bind(&params.key)
    .bind(&params.description)
    .bind(now)
    .execute(&mut *tx)
    .await
    .map_err(map_sqlx)?;

    crate::identity::store::record_audit_event_tx(
        &mut tx,
        actor_id,
        "project.created",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "project_id": project_id,
            "slug": params.slug,
            "key": params.key,
        }),
    )
    .await?;

    tx.commit().await.map_err(StoreError::Internal)?;

    Ok(Project {
        id: project_id,
        workspace_id,
        name: params.name,
        slug: params.slug,
        key: params.key,
        description: params.description,
        task_counter: 0,
        archived: false,
        created_at: now,
        updated_at: now,
    })
}

pub async fn list_projects(
    pool: &PgPool,
    workspace_id: Uuid,
    include_archived: bool,
) -> Result<Vec<Project>, StoreError> {
    let rows = sqlx::query_as::<
        _,
        (
            Uuid,
            Uuid,
            String,
            String,
            String,
            String,
            i64,
            Option<OffsetDateTime>,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "SELECT id, workspace_id, name, slug, key, description, task_counter, archived_at, created_at, updated_at
         FROM projects
         WHERE workspace_id = $1 AND ($2 OR archived_at IS NULL)
         ORDER BY name ASC",
    )
    .bind(workspace_id)
    .bind(include_archived)
    .fetch_all(pool)
    .await
    .map_err(StoreError::Internal)?;

    Ok(rows
        .into_iter()
        .map(|r| Project {
            id: r.0,
            workspace_id: r.1,
            name: r.2,
            slug: r.3,
            key: r.4,
            description: r.5,
            task_counter: r.6,
            archived: r.7.is_some(),
            created_at: r.8,
            updated_at: r.9,
        })
        .collect())
}

pub async fn get_project(
    pool: &PgPool,
    workspace_id: Uuid,
    project_id: Uuid,
) -> Result<Option<Project>, StoreError> {
    let row = sqlx::query_as::<
        _,
        (
            Uuid,
            Uuid,
            String,
            String,
            String,
            String,
            i64,
            Option<OffsetDateTime>,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "SELECT id, workspace_id, name, slug, key, description, task_counter, archived_at, created_at, updated_at
         FROM projects
         WHERE id = $1 AND workspace_id = $2",
    )
    .bind(project_id)
    .bind(workspace_id)
    .fetch_optional(pool)
    .await
    .map_err(StoreError::Internal)?;

    Ok(row.map(|r| Project {
        id: r.0,
        workspace_id: r.1,
        name: r.2,
        slug: r.3,
        key: r.4,
        description: r.5,
        task_counter: r.6,
        archived: r.7.is_some(),
        created_at: r.8,
        updated_at: r.9,
    }))
}

pub async fn archive_project(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
) -> Result<Project, StoreError> {
    let now = OffsetDateTime::now_utc();
    let row = sqlx::query_as::<
        _,
        (
            Uuid,
            Uuid,
            String,
            String,
            String,
            String,
            i64,
            Option<OffsetDateTime>,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "UPDATE projects
         SET archived_at = $1, updated_at = $1
         WHERE id = $2 AND workspace_id = $3
         RETURNING id, workspace_id, name, slug, key, description, task_counter, archived_at, created_at, updated_at",
    )
    .bind(now)
    .bind(project_id)
    .bind(workspace_id)
    .fetch_optional(pool)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    crate::identity::store::record_audit_event(
        pool,
        actor_id,
        "project.archived",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "project_id": project_id,
        }),
    )
    .await?;

    Ok(Project {
        id: row.0,
        workspace_id: row.1,
        name: row.2,
        slug: row.3,
        key: row.4,
        description: row.5,
        task_counter: row.6,
        archived: row.7.is_some(),
        created_at: row.8,
        updated_at: row.9,
    })
}

// ---------------------------------------------------------------------------
// Tasks
// ---------------------------------------------------------------------------

pub async fn create_task(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    params: CreateTaskParams,
) -> Result<Task, StoreError> {
    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    // Increment task_counter atomically on the project
    let project_row = sqlx::query_as::<_, (String, i64)>(
        "UPDATE projects
         SET task_counter = task_counter + 1, updated_at = now()
         WHERE id = $1 AND workspace_id = $2 AND archived_at IS NULL
         RETURNING key, task_counter",
    )
    .bind(project_id)
    .bind(workspace_id)
    .fetch_optional(&mut *tx)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    let project_key = project_row.0;
    let task_number = project_row.1;
    let task_id = Uuid::new_v4();
    let now = OffsetDateTime::now_utc();

    sqlx::query(
        "INSERT INTO tasks
            (id, workspace_id, project_id, task_number, title, description, status, priority, due_date, created_by, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)",
    )
    .bind(task_id)
    .bind(workspace_id)
    .bind(project_id)
    .bind(task_number)
    .bind(&params.title)
    .bind(&params.description)
    .bind(&params.status)
    .bind(&params.priority)
    .bind(params.due_date)
    .bind(actor_id)
    .bind(now)
    .execute(&mut *tx)
    .await
    .map_err(map_sqlx)?;

    let human_key = format!("{}-{}", project_key, task_number);

    crate::identity::store::record_audit_event_tx(
        &mut tx,
        actor_id,
        "task.created",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "project_id": project_id,
            "task_id": task_id,
            "human_key": human_key,
        }),
    )
    .await?;

    tx.commit().await.map_err(StoreError::Internal)?;

    Ok(Task {
        id: task_id,
        workspace_id,
        project_id,
        project_key,
        task_number,
        human_key,
        title: params.title,
        description: params.description,
        status: params.status,
        priority: params.priority,
        due_date: params.due_date,
        created_by: actor_id,
        created_at: now,
        updated_at: now,
    })
}

pub async fn list_tasks(
    pool: &PgPool,
    workspace_id: Uuid,
    project_id: Uuid,
    status_filter: Option<&str>,
    limit: i64,
    offset: i64,
) -> Result<PaginatedTasks, StoreError> {
    // Verify project exists in workspace
    let project = get_project(pool, workspace_id, project_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let bounded_limit = limit.clamp(1, 100);
    let bounded_offset = offset.max(0);

    let total: i64 = sqlx::query_scalar(
        "SELECT COUNT(*) FROM tasks
         WHERE project_id = $1 AND workspace_id = $2
           AND ($3::TEXT IS NULL OR status = $3)",
    )
    .bind(project_id)
    .bind(workspace_id)
    .bind(status_filter)
    .fetch_one(pool)
    .await
    .map_err(StoreError::Internal)?;

    let rows = sqlx::query_as::<
        _,
        (
            Uuid,
            Uuid,
            Uuid,
            i64,
            String,
            String,
            String,
            String,
            Option<Date>,
            Option<Uuid>,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "SELECT id, workspace_id, project_id, task_number, title, description, status, priority, due_date, created_by, created_at, updated_at
         FROM tasks
         WHERE project_id = $1 AND workspace_id = $2
           AND ($3::TEXT IS NULL OR status = $3)
         ORDER BY task_number DESC
         LIMIT $4 OFFSET $5",
    )
    .bind(project_id)
    .bind(workspace_id)
    .bind(status_filter)
    .bind(bounded_limit)
    .bind(bounded_offset)
    .fetch_all(pool)
    .await
    .map_err(StoreError::Internal)?;

    let tasks = rows
        .into_iter()
        .map(|r| {
            let human_key = format!("{}-{}", project.key, r.3);
            Task {
                id: r.0,
                workspace_id: r.1,
                project_id: r.2,
                project_key: project.key.clone(),
                task_number: r.3,
                human_key,
                title: r.4,
                description: r.5,
                status: r.6,
                priority: r.7,
                due_date: r.8,
                created_by: r.9,
                created_at: r.10,
                updated_at: r.11,
            }
        })
        .collect();

    Ok(PaginatedTasks {
        tasks,
        total,
        limit: bounded_limit,
        offset: bounded_offset,
    })
}

pub async fn get_task(
    pool: &PgPool,
    workspace_id: Uuid,
    project_id: Uuid,
    task_id: Uuid,
) -> Result<Option<Task>, StoreError> {
    let row = sqlx::query_as::<
        _,
        (
            Uuid,
            Uuid,
            Uuid,
            String,
            i64,
            String,
            String,
            String,
            String,
            Option<Date>,
            Option<Uuid>,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "SELECT t.id, t.workspace_id, t.project_id, p.key, t.task_number, t.title, t.description, t.status, t.priority, t.due_date, t.created_by, t.created_at, t.updated_at
         FROM tasks t
         JOIN projects p ON p.id = t.project_id
         WHERE t.id = $1 AND t.project_id = $2 AND t.workspace_id = $3",
    )
    .bind(task_id)
    .bind(project_id)
    .bind(workspace_id)
    .fetch_optional(pool)
    .await
    .map_err(StoreError::Internal)?;

    Ok(row.map(|r| {
        let human_key = format!("{}-{}", r.3, r.4);
        Task {
            id: r.0,
            workspace_id: r.1,
            project_id: r.2,
            project_key: r.3,
            task_number: r.4,
            human_key,
            title: r.5,
            description: r.6,
            status: r.7,
            priority: r.8,
            due_date: r.9,
            created_by: r.10,
            created_at: r.11,
            updated_at: r.12,
        }
    }))
}

pub async fn update_task(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    task_id: Uuid,
    params: UpdateTaskParams,
) -> Result<Task, StoreError> {
    let now = OffsetDateTime::now_utc();
    let has_due_date = params.due_date.is_some();
    let due_date_val = params.due_date.flatten();

    let res = sqlx::query(
        "UPDATE tasks
         SET updated_at = $1,
             title = COALESCE($2, title),
             description = COALESCE($3, description),
             status = COALESCE($4, status),
             priority = COALESCE($5, priority),
             due_date = CASE WHEN $6 THEN $7 ELSE due_date END
         WHERE id = $8 AND project_id = $9 AND workspace_id = $10",
    )
    .bind(now)
    .bind(params.title)
    .bind(params.description)
    .bind(params.status)
    .bind(params.priority)
    .bind(has_due_date)
    .bind(due_date_val)
    .bind(task_id)
    .bind(project_id)
    .bind(workspace_id)
    .execute(pool)
    .await
    .map_err(map_sqlx)?;

    if res.rows_affected() == 0 {
        return Err(StoreError::NotFound);
    }

    crate::identity::store::record_audit_event(
        pool,
        actor_id,
        "task.updated",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "project_id": project_id,
            "task_id": task_id,
        }),
    )
    .await?;

    get_task(pool, workspace_id, project_id, task_id)
        .await?
        .ok_or(StoreError::NotFound)
}

pub async fn delete_task(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    task_id: Uuid,
) -> Result<(), StoreError> {
    let res =
        sqlx::query("DELETE FROM tasks WHERE id = $1 AND project_id = $2 AND workspace_id = $3")
            .bind(task_id)
            .bind(project_id)
            .bind(workspace_id)
            .execute(pool)
            .await
            .map_err(StoreError::Internal)?;

    if res.rows_affected() == 0 {
        return Err(StoreError::NotFound);
    }

    crate::identity::store::record_audit_event(
        pool,
        actor_id,
        "task.deleted",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "project_id": project_id,
            "task_id": task_id,
        }),
    )
    .await?;

    Ok(())
}
