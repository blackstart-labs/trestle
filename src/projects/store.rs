/// Persistence and query logic for projects, tasks, labels, assignees, subtasks, comments, and activities.
use serde::{Deserialize, Serialize};
use sqlx::PgPool;
use time::{Date, OffsetDateTime};
use uuid::Uuid;

use crate::identity::store::StoreError;

#[derive(Debug, Clone, Serialize, Deserialize)]
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Label {
    pub id: Uuid,
    pub workspace_id: Uuid,
    pub project_id: Uuid,
    pub name: String,
    pub color: String,
    pub created_at: OffsetDateTime,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TaskAssignee {
    pub task_id: Uuid,
    pub user_id: Uuid,
    pub username: String,
    pub display_name: String,
    pub avatar_url: Option<String>,
    pub assigned_by: Option<Uuid>,
    pub created_at: OffsetDateTime,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TaskComment {
    pub id: Uuid,
    pub workspace_id: Uuid,
    pub task_id: Uuid,
    pub user_id: Uuid,
    pub author_name: String,
    pub author_username: String,
    pub content: String,
    pub created_at: OffsetDateTime,
    pub updated_at: OffsetDateTime,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TaskActivity {
    pub id: Uuid,
    pub workspace_id: Uuid,
    pub task_id: Uuid,
    pub actor_id: Option<Uuid>,
    pub actor_name: Option<String>,
    pub action: String,
    pub details: serde_json::Value,
    pub created_at: OffsetDateTime,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
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
    pub parent_id: Option<Uuid>,
    #[serde(default)]
    pub labels: Vec<Label>,
    #[serde(default)]
    pub assignees: Vec<TaskAssignee>,
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
    pub parent_id: Option<Uuid>,
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

    // If parent_id is specified, verify it exists in the same project and workspace
    if let Some(pid) = params.parent_id {
        let parent_exists = sqlx::query_scalar::<_, bool>(
            "SELECT EXISTS (SELECT 1 FROM tasks WHERE id = $1 AND project_id = $2 AND workspace_id = $3)",
        )
        .bind(pid)
        .bind(project_id)
        .bind(workspace_id)
        .fetch_one(&mut *tx)
        .await
        .map_err(StoreError::Internal)?;

        if !parent_exists {
            return Err(StoreError::NotFound);
        }
    }

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
            (id, workspace_id, project_id, task_number, title, description, status, priority, due_date, created_by, parent_id, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)",
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
    .bind(params.parent_id)
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
            "parent_id": params.parent_id,
        }),
    )
    .await?;

    record_task_activity_tx(
        &mut tx,
        workspace_id,
        task_id,
        actor_id,
        "task.created",
        &serde_json::json!({
            "human_key": human_key,
            "title": params.title,
            "status": params.status,
            "priority": params.priority,
        }),
    )
    .await?;

    if let Some(pid) = params.parent_id {
        record_task_activity_tx(
            &mut tx,
            workspace_id,
            pid,
            actor_id,
            "subtask.created",
            &serde_json::json!({
                "subtask_id": task_id,
                "subtask_key": human_key,
                "title": params.title,
            }),
        )
        .await?;
    }

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
        parent_id: params.parent_id,
        labels: Vec::new(),
        assignees: Vec::new(),
        created_at: now,
        updated_at: now,
    })
}

async fn fetch_labels_for_tasks(
    pool: &PgPool,
    task_ids: &[Uuid],
) -> Result<std::collections::HashMap<Uuid, Vec<Label>>, StoreError> {
    let mut map: std::collections::HashMap<Uuid, Vec<Label>> = std::collections::HashMap::new();
    if task_ids.is_empty() {
        return Ok(map);
    }

    let rows = sqlx::query_as::<_, (Uuid, Uuid, Uuid, Uuid, String, String, OffsetDateTime)>(
        "SELECT tla.task_id, l.id, l.workspace_id, l.project_id, l.name, l.color, l.created_at
         FROM task_label_assignments tla
         JOIN labels l ON l.id = tla.label_id
         WHERE tla.task_id = ANY($1)
         ORDER BY l.name ASC",
    )
    .bind(task_ids)
    .fetch_all(pool)
    .await
    .map_err(StoreError::Internal)?;

    for r in rows {
        map.entry(r.0).or_default().push(Label {
            id: r.1,
            workspace_id: r.2,
            project_id: r.3,
            name: r.4,
            color: r.5,
            created_at: r.6,
        });
    }

    Ok(map)
}

async fn fetch_assignees_for_tasks(
    pool: &PgPool,
    task_ids: &[Uuid],
) -> Result<std::collections::HashMap<Uuid, Vec<TaskAssignee>>, StoreError> {
    let mut map: std::collections::HashMap<Uuid, Vec<TaskAssignee>> =
        std::collections::HashMap::new();
    if task_ids.is_empty() {
        return Ok(map);
    }

    let rows = sqlx::query_as::<_, (Uuid, Uuid, String, String, Option<Uuid>, OffsetDateTime)>(
        "SELECT ta.task_id, ta.user_id, u.username, u.display_name, ta.assigned_by, ta.created_at
         FROM task_assignees ta
         JOIN users u ON u.id = ta.user_id
         WHERE ta.task_id = ANY($1)
         ORDER BY u.display_name ASC",
    )
    .bind(task_ids)
    .fetch_all(pool)
    .await
    .map_err(StoreError::Internal)?;

    for r in rows {
        map.entry(r.0).or_default().push(TaskAssignee {
            task_id: r.0,
            user_id: r.1,
            username: r.2,
            display_name: r.3,
            avatar_url: None,
            assigned_by: r.4,
            created_at: r.5,
        });
    }

    Ok(map)
}

#[derive(Debug, Clone, Default)]
pub struct ListTasksParams<'a> {
    pub status_filter: Option<&'a str>,
    pub label_filter: Option<Uuid>,
    pub assignee_filter: Option<Uuid>,
    pub limit: i64,
    pub offset: i64,
}

pub async fn list_tasks(
    pool: &PgPool,
    workspace_id: Uuid,
    project_id: Uuid,
    params: ListTasksParams<'_>,
) -> Result<PaginatedTasks, StoreError> {
    // Verify project exists in workspace
    let project = get_project(pool, workspace_id, project_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let bounded_limit = params.limit.clamp(1, 100);
    let bounded_offset = params.offset.max(0);

    let total: i64 = sqlx::query_scalar(
        "SELECT COUNT(*) FROM tasks t
         WHERE t.project_id = $1 AND t.workspace_id = $2
           AND ($3::TEXT IS NULL OR t.status = $3)
           AND ($4::UUID IS NULL OR EXISTS (
               SELECT 1 FROM task_label_assignments tla WHERE tla.task_id = t.id AND tla.label_id = $4
           ))
           AND ($5::UUID IS NULL OR EXISTS (
               SELECT 1 FROM task_assignees ta WHERE ta.task_id = t.id AND ta.user_id = $5
           ))",
    )
    .bind(project_id)
    .bind(workspace_id)
    .bind(params.status_filter)
    .bind(params.label_filter)
    .bind(params.assignee_filter)
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
            Option<Uuid>,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "SELECT t.id, t.workspace_id, t.project_id, t.task_number, t.title, t.description, t.status, t.priority, t.due_date, t.created_by, t.parent_id, t.created_at, t.updated_at
         FROM tasks t
         WHERE t.project_id = $1 AND t.workspace_id = $2
           AND ($3::TEXT IS NULL OR t.status = $3)
           AND ($4::UUID IS NULL OR EXISTS (
               SELECT 1 FROM task_label_assignments tla WHERE tla.task_id = t.id AND tla.label_id = $4
           ))
           AND ($5::UUID IS NULL OR EXISTS (
               SELECT 1 FROM task_assignees ta WHERE ta.task_id = t.id AND ta.user_id = $5
           ))
         ORDER BY t.task_number DESC
         LIMIT $6 OFFSET $7",
    )
    .bind(project_id)
    .bind(workspace_id)
    .bind(params.status_filter)
    .bind(params.label_filter)
    .bind(params.assignee_filter)
    .bind(bounded_limit)
    .bind(bounded_offset)
    .fetch_all(pool)
    .await
    .map_err(StoreError::Internal)?;

    let task_ids: Vec<Uuid> = rows.iter().map(|r| r.0).collect();
    let labels_map = fetch_labels_for_tasks(pool, &task_ids).await?;
    let assignees_map = fetch_assignees_for_tasks(pool, &task_ids).await?;

    let tasks = rows
        .into_iter()
        .map(|r| {
            let human_key = format!("{}-{}", project.key, r.3);
            let labels = labels_map.get(&r.0).cloned().unwrap_or_default();
            let assignees = assignees_map.get(&r.0).cloned().unwrap_or_default();
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
                parent_id: r.10,
                labels,
                assignees,
                created_at: r.11,
                updated_at: r.12,
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
            Option<Uuid>,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "SELECT t.id, t.workspace_id, t.project_id, p.key, t.task_number, t.title, t.description, t.status, t.priority, t.due_date, t.created_by, t.parent_id, t.created_at, t.updated_at
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

    match row {
        Some(r) => {
            let human_key = format!("{}-{}", r.3, r.4);
            let mut labels_map = fetch_labels_for_tasks(pool, &[r.0]).await?;
            let mut assignees_map = fetch_assignees_for_tasks(pool, &[r.0]).await?;
            let labels = labels_map.remove(&r.0).unwrap_or_default();
            let assignees = assignees_map.remove(&r.0).unwrap_or_default();
            Ok(Some(Task {
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
                parent_id: r.11,
                labels,
                assignees,
                created_at: r.12,
                updated_at: r.13,
            }))
        }
        None => Ok(None),
    }
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
    .bind(&params.title)
    .bind(&params.description)
    .bind(&params.status)
    .bind(&params.priority)
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

    if let Some(ref new_status) = params.status {
        let _ = record_task_activity(
            pool,
            workspace_id,
            task_id,
            actor_id,
            "task.status_changed",
            &serde_json::json!({ "status": new_status }),
        )
        .await;
    }

    if let Some(ref new_priority) = params.priority {
        let _ = record_task_activity(
            pool,
            workspace_id,
            task_id,
            actor_id,
            "task.priority_changed",
            &serde_json::json!({ "priority": new_priority }),
        )
        .await;
    }

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

// ---------------------------------------------------------------------------
// Labels
// ---------------------------------------------------------------------------

pub async fn create_label(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    name: String,
    color: String,
) -> Result<Label, StoreError> {
    let _ = get_project(pool, workspace_id, project_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let label_id = Uuid::new_v4();
    let now = OffsetDateTime::now_utc();

    sqlx::query(
        "INSERT INTO labels (id, workspace_id, project_id, name, color, created_at)
         VALUES ($1, $2, $3, $4, $5, $6)",
    )
    .bind(label_id)
    .bind(workspace_id)
    .bind(project_id)
    .bind(&name)
    .bind(&color)
    .bind(now)
    .execute(pool)
    .await
    .map_err(map_sqlx)?;

    crate::identity::store::record_audit_event(
        pool,
        actor_id,
        "label.created",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "project_id": project_id,
            "label_id": label_id,
            "name": name,
        }),
    )
    .await?;

    Ok(Label {
        id: label_id,
        workspace_id,
        project_id,
        name,
        color,
        created_at: now,
    })
}

pub async fn list_labels(
    pool: &PgPool,
    workspace_id: Uuid,
    project_id: Uuid,
) -> Result<Vec<Label>, StoreError> {
    let _ = get_project(pool, workspace_id, project_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let rows = sqlx::query_as::<_, (Uuid, Uuid, Uuid, String, String, OffsetDateTime)>(
        "SELECT id, workspace_id, project_id, name, color, created_at
         FROM labels
         WHERE workspace_id = $1 AND project_id = $2
         ORDER BY name ASC",
    )
    .bind(workspace_id)
    .bind(project_id)
    .fetch_all(pool)
    .await
    .map_err(StoreError::Internal)?;

    Ok(rows
        .into_iter()
        .map(|r| Label {
            id: r.0,
            workspace_id: r.1,
            project_id: r.2,
            name: r.3,
            color: r.4,
            created_at: r.5,
        })
        .collect())
}

pub async fn delete_label(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    label_id: Uuid,
) -> Result<(), StoreError> {
    let res = sqlx::query(
        "DELETE FROM labels
         WHERE id = $1 AND workspace_id = $2 AND project_id = $3",
    )
    .bind(label_id)
    .bind(workspace_id)
    .bind(project_id)
    .execute(pool)
    .await
    .map_err(StoreError::Internal)?;

    if res.rows_affected() == 0 {
        return Err(StoreError::NotFound);
    }

    crate::identity::store::record_audit_event(
        pool,
        actor_id,
        "label.deleted",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "project_id": project_id,
            "label_id": label_id,
        }),
    )
    .await?;

    Ok(())
}

pub async fn attach_label(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    task_id: Uuid,
    label_id: Uuid,
) -> Result<(), StoreError> {
    let task = get_task(pool, workspace_id, project_id, task_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let label = sqlx::query_as::<_, (String, String)>(
        "SELECT name, color FROM labels WHERE id = $1 AND project_id = $2 AND workspace_id = $3",
    )
    .bind(label_id)
    .bind(project_id)
    .bind(workspace_id)
    .fetch_optional(pool)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    sqlx::query(
        "INSERT INTO task_label_assignments (task_id, label_id, created_at)
         VALUES ($1, $2, now())
         ON CONFLICT (task_id, label_id) DO NOTHING",
    )
    .bind(task_id)
    .bind(label_id)
    .execute(pool)
    .await
    .map_err(StoreError::Internal)?;

    record_task_activity(
        pool,
        workspace_id,
        task_id,
        actor_id,
        "label.attached",
        &serde_json::json!({
            "label_id": label_id,
            "name": label.0,
            "color": label.1,
            "task_key": task.human_key,
        }),
    )
    .await?;

    Ok(())
}

pub async fn detach_label(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    task_id: Uuid,
    label_id: Uuid,
) -> Result<(), StoreError> {
    let task = get_task(pool, workspace_id, project_id, task_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let res = sqlx::query(
        "DELETE FROM task_label_assignments
         WHERE task_id = $1 AND label_id = $2",
    )
    .bind(task_id)
    .bind(label_id)
    .execute(pool)
    .await
    .map_err(StoreError::Internal)?;

    if res.rows_affected() > 0 {
        record_task_activity(
            pool,
            workspace_id,
            task_id,
            actor_id,
            "label.detached",
            &serde_json::json!({
                "label_id": label_id,
                "task_key": task.human_key,
            }),
        )
        .await?;
    }

    Ok(())
}

// ---------------------------------------------------------------------------
// Assignees
// ---------------------------------------------------------------------------

pub async fn assign_user(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    task_id: Uuid,
    user_id: Uuid,
) -> Result<TaskAssignee, StoreError> {
    let task = get_task(pool, workspace_id, project_id, task_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let user_info = sqlx::query_as::<_, (String, String)>(
        "SELECT u.username, u.display_name
         FROM workspace_memberships wm
         JOIN users u ON u.id = wm.user_id
         WHERE wm.workspace_id = $1 AND wm.user_id = $2",
    )
    .bind(workspace_id)
    .bind(user_id)
    .fetch_optional(pool)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    let now = OffsetDateTime::now_utc();

    sqlx::query(
        "INSERT INTO task_assignees (task_id, user_id, assigned_by, created_at)
         VALUES ($1, $2, $3, $4)
         ON CONFLICT (task_id, user_id) DO UPDATE SET assigned_by = $3",
    )
    .bind(task_id)
    .bind(user_id)
    .bind(actor_id)
    .bind(now)
    .execute(pool)
    .await
    .map_err(StoreError::Internal)?;

    record_task_activity(
        pool,
        workspace_id,
        task_id,
        actor_id,
        "user.assigned",
        &serde_json::json!({
            "user_id": user_id,
            "display_name": user_info.1,
            "username": user_info.0,
            "task_key": task.human_key,
        }),
    )
    .await?;

    Ok(TaskAssignee {
        task_id,
        user_id,
        username: user_info.0,
        display_name: user_info.1,
        avatar_url: None,
        assigned_by: actor_id,
        created_at: now,
    })
}

pub async fn unassign_user(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    task_id: Uuid,
    user_id: Uuid,
) -> Result<(), StoreError> {
    let task = get_task(pool, workspace_id, project_id, task_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let res = sqlx::query("DELETE FROM task_assignees WHERE task_id = $1 AND user_id = $2")
        .bind(task_id)
        .bind(user_id)
        .execute(pool)
        .await
        .map_err(StoreError::Internal)?;

    if res.rows_affected() > 0 {
        record_task_activity(
            pool,
            workspace_id,
            task_id,
            actor_id,
            "user.unassigned",
            &serde_json::json!({
                "user_id": user_id,
                "task_key": task.human_key,
            }),
        )
        .await?;
    }

    Ok(())
}

// ---------------------------------------------------------------------------
// Subtasks
// ---------------------------------------------------------------------------

pub async fn list_subtasks(
    pool: &PgPool,
    workspace_id: Uuid,
    project_id: Uuid,
    parent_id: Uuid,
) -> Result<Vec<Task>, StoreError> {
    let _ = get_task(pool, workspace_id, project_id, parent_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let project = get_project(pool, workspace_id, project_id)
        .await?
        .ok_or(StoreError::NotFound)?;

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
            Option<Uuid>,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "SELECT id, workspace_id, project_id, task_number, title, description, status, priority, due_date, created_by, parent_id, created_at, updated_at
         FROM tasks
         WHERE parent_id = $1 AND project_id = $2 AND workspace_id = $3
         ORDER BY task_number ASC",
    )
    .bind(parent_id)
    .bind(project_id)
    .bind(workspace_id)
    .fetch_all(pool)
    .await
    .map_err(StoreError::Internal)?;

    let task_ids: Vec<Uuid> = rows.iter().map(|r| r.0).collect();
    let labels_map = fetch_labels_for_tasks(pool, &task_ids).await?;
    let assignees_map = fetch_assignees_for_tasks(pool, &task_ids).await?;

    let tasks = rows
        .into_iter()
        .map(|r| {
            let human_key = format!("{}-{}", project.key, r.3);
            let labels = labels_map.get(&r.0).cloned().unwrap_or_default();
            let assignees = assignees_map.get(&r.0).cloned().unwrap_or_default();
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
                parent_id: r.10,
                labels,
                assignees,
                created_at: r.11,
                updated_at: r.12,
            }
        })
        .collect();

    Ok(tasks)
}

// ---------------------------------------------------------------------------
// Comments
// ---------------------------------------------------------------------------

pub async fn create_comment(
    pool: &PgPool,
    actor_id: Uuid,
    workspace_id: Uuid,
    project_id: Uuid,
    task_id: Uuid,
    content: String,
) -> Result<TaskComment, StoreError> {
    let task = get_task(pool, workspace_id, project_id, task_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let user_info = sqlx::query_as::<_, (String, String)>(
        "SELECT username, display_name FROM users WHERE id = $1",
    )
    .bind(actor_id)
    .fetch_optional(pool)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    let comment_id = Uuid::new_v4();
    let now = OffsetDateTime::now_utc();

    sqlx::query(
        "INSERT INTO task_comments (id, workspace_id, task_id, user_id, content, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $6)",
    )
    .bind(comment_id)
    .bind(workspace_id)
    .bind(task_id)
    .bind(actor_id)
    .bind(&content)
    .bind(now)
    .execute(pool)
    .await
    .map_err(StoreError::Internal)?;

    record_task_activity(
        pool,
        workspace_id,
        task_id,
        Some(actor_id),
        "comment.added",
        &serde_json::json!({
            "comment_id": comment_id,
            "author_name": user_info.1,
            "task_key": task.human_key,
        }),
    )
    .await?;

    Ok(TaskComment {
        id: comment_id,
        workspace_id,
        task_id,
        user_id: actor_id,
        author_name: user_info.1,
        author_username: user_info.0,
        content,
        created_at: now,
        updated_at: now,
    })
}

pub async fn list_comments(
    pool: &PgPool,
    workspace_id: Uuid,
    project_id: Uuid,
    task_id: Uuid,
) -> Result<Vec<TaskComment>, StoreError> {
    let _ = get_task(pool, workspace_id, project_id, task_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let rows = sqlx::query_as::<
        _,
        (
            Uuid,
            Uuid,
            Uuid,
            Uuid,
            String,
            String,
            String,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "SELECT c.id, c.workspace_id, c.task_id, c.user_id, u.display_name, u.username, c.content, c.created_at, c.updated_at
         FROM task_comments c
         JOIN users u ON u.id = c.user_id
         WHERE c.task_id = $1 AND c.workspace_id = $2
         ORDER BY c.created_at ASC",
    )
    .bind(task_id)
    .bind(workspace_id)
    .fetch_all(pool)
    .await
    .map_err(StoreError::Internal)?;

    Ok(rows
        .into_iter()
        .map(|r| TaskComment {
            id: r.0,
            workspace_id: r.1,
            task_id: r.2,
            user_id: r.3,
            author_name: r.4,
            author_username: r.5,
            content: r.6,
            created_at: r.7,
            updated_at: r.8,
        })
        .collect())
}

pub async fn delete_comment(
    pool: &PgPool,
    actor_id: Uuid,
    workspace_id: Uuid,
    project_id: Uuid,
    task_id: Uuid,
    comment_id: Uuid,
    is_admin_or_owner: bool,
) -> Result<(), StoreError> {
    let _ = get_task(pool, workspace_id, project_id, task_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let comment_row = sqlx::query_as::<_, (Uuid, Uuid)>(
        "SELECT user_id, workspace_id FROM task_comments WHERE id = $1 AND task_id = $2",
    )
    .bind(comment_id)
    .bind(task_id)
    .fetch_optional(pool)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    if comment_row.1 != workspace_id {
        return Err(StoreError::NotFound);
    }

    if !is_admin_or_owner && comment_row.0 != actor_id {
        return Err(StoreError::Forbidden);
    }

    sqlx::query("DELETE FROM task_comments WHERE id = $1")
        .bind(comment_id)
        .execute(pool)
        .await
        .map_err(StoreError::Internal)?;

    Ok(())
}

// ---------------------------------------------------------------------------
// Activities
// ---------------------------------------------------------------------------

pub async fn record_task_activity(
    pool: &PgPool,
    workspace_id: Uuid,
    task_id: Uuid,
    actor_id: Option<Uuid>,
    action: &str,
    details: &serde_json::Value,
) -> Result<(), StoreError> {
    sqlx::query(
        "INSERT INTO task_activities (id, workspace_id, task_id, actor_id, action, details, created_at)
         VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, now())",
    )
    .bind(workspace_id)
    .bind(task_id)
    .bind(actor_id)
    .bind(action)
    .bind(details)
    .execute(pool)
    .await
    .map_err(StoreError::Internal)?;
    Ok(())
}

pub async fn record_task_activity_tx(
    tx: &mut sqlx::Transaction<'_, sqlx::Postgres>,
    workspace_id: Uuid,
    task_id: Uuid,
    actor_id: Option<Uuid>,
    action: &str,
    details: &serde_json::Value,
) -> Result<(), StoreError> {
    sqlx::query(
        "INSERT INTO task_activities (id, workspace_id, task_id, actor_id, action, details, created_at)
         VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, now())",
    )
    .bind(workspace_id)
    .bind(task_id)
    .bind(actor_id)
    .bind(action)
    .bind(details)
    .execute(&mut **tx)
    .await
    .map_err(StoreError::Internal)?;
    Ok(())
}

pub async fn list_task_activities(
    pool: &PgPool,
    workspace_id: Uuid,
    project_id: Uuid,
    task_id: Uuid,
) -> Result<Vec<TaskActivity>, StoreError> {
    let _ = get_task(pool, workspace_id, project_id, task_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let rows = sqlx::query_as::<
        _,
        (
            Uuid,
            Uuid,
            Uuid,
            Option<Uuid>,
            Option<String>,
            String,
            serde_json::Value,
            OffsetDateTime,
        ),
    >(
        "SELECT a.id, a.workspace_id, a.task_id, a.actor_id, u.display_name, a.action, a.details, a.created_at
         FROM task_activities a
         LEFT JOIN users u ON u.id = a.actor_id
         WHERE a.task_id = $1 AND a.workspace_id = $2
         ORDER BY a.created_at DESC
         LIMIT 50",
    )
    .bind(task_id)
    .bind(workspace_id)
    .fetch_all(pool)
    .await
    .map_err(StoreError::Internal)?;

    Ok(rows
        .into_iter()
        .map(|r| TaskActivity {
            id: r.0,
            workspace_id: r.1,
            task_id: r.2,
            actor_id: r.3,
            actor_name: r.4,
            action: r.5,
            details: r.6,
            created_at: r.7,
        })
        .collect())
}
