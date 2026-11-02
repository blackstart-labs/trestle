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
    pub status_id: Option<Uuid>,
    pub status_name: Option<String>,
    pub status_color: Option<String>,
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
    pub status_id: Option<Uuid>,
}

#[derive(Debug, Clone, Default)]
pub struct UpdateTaskParams {
    pub title: Option<String>,
    pub description: Option<String>,
    pub status: Option<String>,
    pub status_id: Option<Uuid>,
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

    // Seed 5 default statuses with stable categories
    sqlx::query(
        "INSERT INTO task_statuses (id, workspace_id, project_id, name, category, color, position, created_at, updated_at)
         VALUES
            (gen_random_uuid(), $1, $2, 'Backlog', 'backlog', '#4c544c', 0, $3, $3),
            (gen_random_uuid(), $1, $2, 'To Do', 'unstarted', '#1e4d79', 1, $3, $3),
            (gen_random_uuid(), $1, $2, 'In Progress', 'started', '#775300', 2, $3, $3),
            (gen_random_uuid(), $1, $2, 'Done', 'completed', '#26633b', 3, $3, $3),
            (gen_random_uuid(), $1, $2, 'Cancelled', 'cancelled', '#9f302b', 4, $3, $3)",
    )
    .bind(workspace_id)
    .bind(project_id)
    .bind(now)
    .execute(&mut *tx)
    .await
    .map_err(StoreError::Internal)?;

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

    // Resolve status_id and canonical status category
    let status_row = if let Some(sid) = params.status_id {
        sqlx::query_as::<_, (Uuid, String, String, String)>(
            "SELECT id, name, category, color FROM task_statuses WHERE id = $1 AND project_id = $2 AND workspace_id = $3",
        )
        .bind(sid)
        .bind(project_id)
        .bind(workspace_id)
        .fetch_optional(&mut *tx)
        .await
        .map_err(StoreError::Internal)?
    } else {
        sqlx::query_as::<_, (Uuid, String, String, String)>(
            "SELECT id, name, category, color FROM task_statuses WHERE project_id = $1 AND workspace_id = $2 AND (category = $3 OR name = $3) ORDER BY position ASC LIMIT 1",
        )
        .bind(project_id)
        .bind(workspace_id)
        .bind(&params.status)
        .fetch_optional(&mut *tx)
        .await
        .map_err(StoreError::Internal)?
    };

    let (status_id, status_name, status_category, status_color) = match status_row {
        Some(s) => (Some(s.0), Some(s.1), s.2, Some(s.3)),
        None => (None, None, params.status.clone(), None),
    };

    sqlx::query(
        "INSERT INTO tasks
            (id, workspace_id, project_id, task_number, title, description, status, priority, due_date, created_by, parent_id, status_id, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)",
    )
    .bind(task_id)
    .bind(workspace_id)
    .bind(project_id)
    .bind(task_number)
    .bind(&params.title)
    .bind(&params.description)
    .bind(&status_category)
    .bind(&params.priority)
    .bind(params.due_date)
    .bind(actor_id)
    .bind(params.parent_id)
    .bind(status_id)
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
            "status_id": status_id,
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
            "status": status_category,
            "status_id": status_id,
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
        status: status_category,
        status_id,
        status_name,
        status_color,
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
    pub status_id: Option<Uuid>,
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
           AND ($4::UUID IS NULL OR t.status_id = $4)
           AND ($5::UUID IS NULL OR EXISTS (
               SELECT 1 FROM task_label_assignments tla WHERE tla.task_id = t.id AND tla.label_id = $5
           ))
           AND ($6::UUID IS NULL OR EXISTS (
               SELECT 1 FROM task_assignees ta WHERE ta.task_id = t.id AND ta.user_id = $6
           ))",
    )
    .bind(project_id)
    .bind(workspace_id)
    .bind(params.status_filter)
    .bind(params.status_id)
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
            Option<Uuid>,
            Option<String>,
            Option<String>,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "SELECT t.id, t.workspace_id, t.project_id, t.task_number, t.title, t.description,
                t.status, t.priority, t.due_date, t.created_by, t.parent_id,
                t.status_id, ts.name, ts.color, t.created_at, t.updated_at
         FROM tasks t
         LEFT JOIN task_statuses ts ON ts.id = t.status_id
         WHERE t.project_id = $1 AND t.workspace_id = $2
           AND ($3::TEXT IS NULL OR t.status = $3)
           AND ($4::UUID IS NULL OR t.status_id = $4)
           AND ($5::UUID IS NULL OR EXISTS (
               SELECT 1 FROM task_label_assignments tla WHERE tla.task_id = t.id AND tla.label_id = $5
           ))
           AND ($6::UUID IS NULL OR EXISTS (
               SELECT 1 FROM task_assignees ta WHERE ta.task_id = t.id AND ta.user_id = $6
           ))
         ORDER BY t.task_number DESC
         LIMIT $7 OFFSET $8",
    )
    .bind(project_id)
    .bind(workspace_id)
    .bind(params.status_filter)
    .bind(params.status_id)
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
                status_id: r.11,
                status_name: r.12,
                status_color: r.13,
                labels,
                assignees,
                created_at: r.14,
                updated_at: r.15,
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

#[derive(sqlx::FromRow)]
struct GetTaskRow {
    id: Uuid,
    workspace_id: Uuid,
    project_id: Uuid,
    project_key: String,
    task_number: i64,
    title: String,
    description: String,
    status: String,
    priority: String,
    due_date: Option<Date>,
    created_by: Option<Uuid>,
    parent_id: Option<Uuid>,
    status_id: Option<Uuid>,
    status_name: Option<String>,
    status_color: Option<String>,
    created_at: OffsetDateTime,
    updated_at: OffsetDateTime,
}

pub async fn get_task(
    pool: &PgPool,
    workspace_id: Uuid,
    project_id: Uuid,
    task_id: Uuid,
) -> Result<Option<Task>, StoreError> {
    let row = sqlx::query_as::<_, GetTaskRow>(
        "SELECT t.id, t.workspace_id, t.project_id, p.key AS project_key, t.task_number, t.title, t.description,
                t.status, t.priority, t.due_date, t.created_by, t.parent_id,
                t.status_id, ts.name AS status_name, ts.color AS status_color, t.created_at, t.updated_at
         FROM tasks t
         JOIN projects p ON p.id = t.project_id
         LEFT JOIN task_statuses ts ON ts.id = t.status_id
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
            let human_key = format!("{}-{}", r.project_key, r.task_number);
            let mut labels_map = fetch_labels_for_tasks(pool, &[r.id]).await?;
            let mut assignees_map = fetch_assignees_for_tasks(pool, &[r.id]).await?;
            let labels = labels_map.remove(&r.id).unwrap_or_default();
            let assignees = assignees_map.remove(&r.id).unwrap_or_default();
            Ok(Some(Task {
                id: r.id,
                workspace_id: r.workspace_id,
                project_id: r.project_id,
                project_key: r.project_key,
                task_number: r.task_number,
                human_key,
                title: r.title,
                description: r.description,
                status: r.status,
                priority: r.priority,
                due_date: r.due_date,
                created_by: r.created_by,
                parent_id: r.parent_id,
                status_id: r.status_id,
                status_name: r.status_name,
                status_color: r.status_color,
                labels,
                assignees,
                created_at: r.created_at,
                updated_at: r.updated_at,
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

    // Check if status or status_id is updated
    let (target_status_id, target_category) = if let Some(sid) = params.status_id {
        let status_row = sqlx::query_as::<_, (Uuid, String)>(
            "SELECT id, category FROM task_statuses WHERE id = $1 AND project_id = $2 AND workspace_id = $3",
        )
        .bind(sid)
        .bind(project_id)
        .bind(workspace_id)
        .fetch_optional(pool)
        .await
        .map_err(StoreError::Internal)?
        .ok_or(StoreError::NotFound)?;
        (Some(status_row.0), Some(status_row.1))
    } else if let Some(ref st) = params.status {
        let status_row = sqlx::query_as::<_, (Uuid, String)>(
            "SELECT id, category FROM task_statuses WHERE project_id = $1 AND workspace_id = $2 AND (category = $3 OR name = $3) ORDER BY position ASC LIMIT 1",
        )
        .bind(project_id)
        .bind(workspace_id)
        .bind(st)
        .fetch_optional(pool)
        .await
        .map_err(StoreError::Internal)?;
        match status_row {
            Some(s) => (Some(s.0), Some(s.1)),
            None => (None, Some(st.clone())),
        }
    } else {
        (None, None)
    };

    let has_status = target_category.is_some();
    let has_status_id = target_status_id.is_some();

    let res = sqlx::query(
        "UPDATE tasks
         SET updated_at = $1,
              title = COALESCE($2, title),
              description = COALESCE($3, description),
              status = CASE WHEN $4 THEN $5 ELSE status END,
              status_id = CASE WHEN $6 THEN $7 ELSE status_id END,
              priority = COALESCE($8, priority),
              due_date = CASE WHEN $9 THEN $10 ELSE due_date END
         WHERE id = $11 AND project_id = $12 AND workspace_id = $13",
    )
    .bind(now)
    .bind(&params.title)
    .bind(&params.description)
    .bind(has_status)
    .bind(&target_category)
    .bind(has_status_id)
    .bind(target_status_id)
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

    if let Some(ref new_status) = target_category {
        let _ = record_task_activity(
            pool,
            workspace_id,
            task_id,
            actor_id,
            "task.status_changed",
            &serde_json::json!({
                "status": new_status,
                "status_id": target_status_id,
            }),
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
            Option<Uuid>,
            Option<String>,
            Option<String>,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "SELECT t.id, t.workspace_id, t.project_id, t.task_number, t.title, t.description,
                t.status, t.priority, t.due_date, t.created_by, t.parent_id,
                t.status_id, ts.name, ts.color, t.created_at, t.updated_at
         FROM tasks t
         LEFT JOIN task_statuses ts ON ts.id = t.status_id
         WHERE t.parent_id = $1 AND t.project_id = $2 AND t.workspace_id = $3
         ORDER BY t.task_number ASC",
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
                status_id: r.11,
                status_name: r.12,
                status_color: r.13,
                labels,
                assignees,
                created_at: r.14,
                updated_at: r.15,
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

// ---------------------------------------------------------------------------
// Task Statuses (M05a)
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct TaskStatus {
    pub id: Uuid,
    pub workspace_id: Uuid,
    pub project_id: Uuid,
    pub name: String,
    pub category: String,
    pub color: String,
    pub position: i32,
    pub created_at: OffsetDateTime,
    pub updated_at: OffsetDateTime,
}

#[derive(Debug, Clone)]
pub struct CreateStatusParams {
    pub name: String,
    pub category: String,
    pub color: Option<String>,
}

#[derive(Debug, Clone)]
pub struct UpdateStatusParams {
    pub name: Option<String>,
    pub color: Option<String>,
}

pub async fn list_project_statuses(
    pool: &PgPool,
    workspace_id: Uuid,
    project_id: Uuid,
) -> Result<Vec<TaskStatus>, StoreError> {
    let _ = get_project(pool, workspace_id, project_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let rows = sqlx::query_as::<
        _,
        (
            Uuid,
            Uuid,
            Uuid,
            String,
            String,
            String,
            i32,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "SELECT id, workspace_id, project_id, name, category, color, position, created_at, updated_at
         FROM task_statuses
         WHERE workspace_id = $1 AND project_id = $2
         ORDER BY position ASC, created_at ASC",
    )
    .bind(workspace_id)
    .bind(project_id)
    .fetch_all(pool)
    .await
    .map_err(StoreError::Internal)?;

    Ok(rows
        .into_iter()
        .map(|r| TaskStatus {
            id: r.0,
            workspace_id: r.1,
            project_id: r.2,
            name: r.3,
            category: r.4,
            color: r.5,
            position: r.6,
            created_at: r.7,
            updated_at: r.8,
        })
        .collect())
}

pub async fn create_project_status(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    params: CreateStatusParams,
) -> Result<TaskStatus, StoreError> {
    let _ = get_project(pool, workspace_id, project_id)
        .await?
        .ok_or(StoreError::NotFound)?;

    let trimmed_name = params.name.trim();
    if trimmed_name.is_empty() || trimmed_name.len() > 50 {
        return Err(StoreError::Conflict);
    }

    let valid_categories = ["backlog", "unstarted", "started", "completed", "cancelled"];
    if !valid_categories.contains(&params.category.as_str()) {
        return Err(StoreError::Conflict);
    }

    let color = params.color.unwrap_or_else(|| "#315b3b".to_string());
    let id = Uuid::new_v4();
    let now = OffsetDateTime::now_utc();

    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    let max_pos: Option<i32> = sqlx::query_scalar(
        "SELECT MAX(position) FROM task_statuses WHERE workspace_id = $1 AND project_id = $2",
    )
    .bind(workspace_id)
    .bind(project_id)
    .fetch_one(&mut *tx)
    .await
    .map_err(StoreError::Internal)?;

    let next_pos = max_pos.map_or(0, |p| p + 1);

    sqlx::query(
        "INSERT INTO task_statuses (id, workspace_id, project_id, name, category, color, position, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)",
    )
    .bind(id)
    .bind(workspace_id)
    .bind(project_id)
    .bind(trimmed_name)
    .bind(&params.category)
    .bind(&color)
    .bind(next_pos)
    .bind(now)
    .execute(&mut *tx)
    .await
    .map_err(map_sqlx)?;

    crate::identity::store::record_audit_event_tx(
        &mut tx,
        actor_id,
        "status.created",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "project_id": project_id,
            "status_id": id,
            "name": trimmed_name,
            "category": params.category,
        }),
    )
    .await?;

    tx.commit().await.map_err(StoreError::Internal)?;

    Ok(TaskStatus {
        id,
        workspace_id,
        project_id,
        name: trimmed_name.to_string(),
        category: params.category,
        color,
        position: next_pos,
        created_at: now,
        updated_at: now,
    })
}

pub async fn update_project_status(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    status_id: Uuid,
    params: UpdateStatusParams,
) -> Result<TaskStatus, StoreError> {
    let now = OffsetDateTime::now_utc();
    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    let current = sqlx::query_as::<
        _,
        (
            Uuid,
            Uuid,
            Uuid,
            String,
            String,
            String,
            i32,
            OffsetDateTime,
            OffsetDateTime,
        ),
    >(
        "SELECT id, workspace_id, project_id, name, category, color, position, created_at, updated_at
         FROM task_statuses
         WHERE id = $1 AND workspace_id = $2 AND project_id = $3",
    )
    .bind(status_id)
    .bind(workspace_id)
    .bind(project_id)
    .fetch_optional(&mut *tx)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    let name = params
        .name
        .map(|n| n.trim().to_string())
        .filter(|n| !n.is_empty())
        .unwrap_or(current.3);

    let color = params
        .color
        .map(|c| c.trim().to_string())
        .filter(|c| !c.is_empty())
        .unwrap_or(current.5);

    sqlx::query(
        "UPDATE task_statuses
         SET name = $1, color = $2, updated_at = $3
         WHERE id = $4 AND workspace_id = $5 AND project_id = $6",
    )
    .bind(&name)
    .bind(&color)
    .bind(now)
    .bind(status_id)
    .bind(workspace_id)
    .bind(project_id)
    .execute(&mut *tx)
    .await
    .map_err(map_sqlx)?;

    crate::identity::store::record_audit_event_tx(
        &mut tx,
        actor_id,
        "status.updated",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "project_id": project_id,
            "status_id": status_id,
            "name": name,
            "color": color,
        }),
    )
    .await?;

    tx.commit().await.map_err(StoreError::Internal)?;

    Ok(TaskStatus {
        id: status_id,
        workspace_id,
        project_id,
        name,
        category: current.4,
        color,
        position: current.6,
        created_at: current.7,
        updated_at: now,
    })
}

pub async fn reorder_project_statuses(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    status_ids: &[Uuid],
) -> Result<(), StoreError> {
    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    for (pos, sid) in status_ids.iter().enumerate() {
        let res = sqlx::query(
            "UPDATE task_statuses
             SET position = $1, updated_at = now()
             WHERE id = $2 AND workspace_id = $3 AND project_id = $4",
        )
        .bind(pos as i32)
        .bind(sid)
        .bind(workspace_id)
        .bind(project_id)
        .execute(&mut *tx)
        .await
        .map_err(StoreError::Internal)?;

        if res.rows_affected() == 0 {
            return Err(StoreError::NotFound);
        }
    }

    crate::identity::store::record_audit_event_tx(
        &mut tx,
        actor_id,
        "status.reordered",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "project_id": project_id,
            "status_ids": status_ids,
        }),
    )
    .await?;

    tx.commit().await.map_err(StoreError::Internal)?;
    Ok(())
}

pub async fn delete_project_status(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    project_id: Uuid,
    status_id: Uuid,
    remap_to: Option<Uuid>,
) -> Result<(), StoreError> {
    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    // 1. Verify status exists
    let status_to_delete = sqlx::query_as::<_, (Uuid, String)>(
        "SELECT id, name FROM task_statuses WHERE id = $1 AND workspace_id = $2 AND project_id = $3",
    )
    .bind(status_id)
    .bind(workspace_id)
    .bind(project_id)
    .fetch_optional(&mut *tx)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    // 2. Total statuses check (cannot delete if it's the only one left)
    let count: i64 = sqlx::query_scalar(
        "SELECT COUNT(*) FROM task_statuses WHERE workspace_id = $1 AND project_id = $2",
    )
    .bind(workspace_id)
    .bind(project_id)
    .fetch_one(&mut *tx)
    .await
    .map_err(StoreError::Internal)?;

    if count <= 1 {
        return Err(StoreError::Conflict);
    }

    // 3. Check active tasks
    let task_count: i64 = sqlx::query_scalar(
        "SELECT COUNT(*) FROM tasks WHERE status_id = $1 AND workspace_id = $2 AND project_id = $3",
    )
    .bind(status_id)
    .bind(workspace_id)
    .bind(project_id)
    .fetch_one(&mut *tx)
    .await
    .map_err(StoreError::Internal)?;

    if task_count > 0 {
        let dest_id = remap_to.ok_or(StoreError::Conflict)?;
        if dest_id == status_id {
            return Err(StoreError::Conflict);
        }

        // Verify remap_to exists in same project
        let dest = sqlx::query_as::<_, (Uuid, String)>(
            "SELECT id, category FROM task_statuses WHERE id = $1 AND workspace_id = $2 AND project_id = $3",
        )
        .bind(dest_id)
        .bind(workspace_id)
        .bind(project_id)
        .fetch_optional(&mut *tx)
        .await
        .map_err(StoreError::Internal)?
        .ok_or(StoreError::NotFound)?;

        // Reassign all tasks
        sqlx::query(
            "UPDATE tasks
             SET status_id = $1, status = $2, updated_at = now()
             WHERE status_id = $3 AND workspace_id = $4 AND project_id = $5",
        )
        .bind(dest.0)
        .bind(&dest.1)
        .bind(status_id)
        .bind(workspace_id)
        .bind(project_id)
        .execute(&mut *tx)
        .await
        .map_err(StoreError::Internal)?;
    }

    // 4. Delete the status
    sqlx::query(
        "DELETE FROM task_statuses WHERE id = $1 AND workspace_id = $2 AND project_id = $3",
    )
    .bind(status_id)
    .bind(workspace_id)
    .bind(project_id)
    .execute(&mut *tx)
    .await
    .map_err(StoreError::Internal)?;

    crate::identity::store::record_audit_event_tx(
        &mut tx,
        actor_id,
        "status.deleted",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "project_id": project_id,
            "status_id": status_id,
            "name": status_to_delete.1,
            "remap_to": remap_to,
        }),
    )
    .await?;

    tx.commit().await.map_err(StoreError::Internal)?;
    Ok(())
}
