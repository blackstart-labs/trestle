use axum::{
    body::Body,
    http::{Request, StatusCode, header},
};
use http_body_util::BodyExt;
use sqlx::{ConnectOptions, PgPool};
use std::time::Duration;
use tower::ServiceExt;
use trestle::{
    config::Config,
    database, http,
    identity::{Hasher, store},
};

fn test_config(pool: &PgPool) -> Config {
    let url = pool.connect_options().to_url_lossy();
    Config::from_pairs([
        ("DATABASE_URL", url.as_str()),
        ("TRESTLE_DEV_MODE", "true"),
        ("TRESTLE_ARGON2_MEMORY_KIB", "4096"),
        ("TRESTLE_ARGON2_TIME_COST", "1"),
        ("TRESTLE_ARGON2_PARALLELISM", "1"),
    ])
    .unwrap()
}

fn extract_cookie<'a>(header_value: &'a str, name: &str) -> Option<&'a str> {
    for pair in header_value.split(';') {
        let pair = pair.trim();
        if let Some((k, v)) = pair.split_once('=')
            && k.trim() == name
        {
            return Some(v.trim());
        }
    }
    None
}

async fn login_user(app: &axum::Router, username: &str, password: &str) -> (String, String) {
    let login_body = serde_json::json!({
        "username": username,
        "password": password
    });

    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/session")
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&login_body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(res.status(), StatusCode::OK);

    let mut session_token = None;
    let mut csrf_token = None;

    for cookie_header in res.headers().get_all(header::SET_COOKIE) {
        let cookie_str = cookie_header.to_str().unwrap();
        if let Some(val) = extract_cookie(cookie_str, "trestle-session") {
            session_token = Some(val.to_owned());
        }
        if let Some(val) = extract_cookie(cookie_str, "trestle-csrf") {
            csrf_token = Some(val.to_owned());
        }
    }

    (
        session_token.expect("session cookie"),
        csrf_token.expect("csrf cookie"),
    )
}

#[sqlx::test(migrations = false)]
async fn project_and_task_lifecycle_end_to_end(pool: PgPool) {
    database::migrate(&pool).await.unwrap();

    let config = test_config(&pool);
    let hasher = Hasher::new(4096, 1, 1).unwrap();
    let password_hash = hasher.hash(b"secure-pass-123").unwrap();

    let user = store::bootstrap(
        &pool,
        "alice",
        &password_hash,
        "Alice Smith",
        "Acme Engineering",
        "acme-eng",
    )
    .await
    .unwrap();

    let workspaces = store::list_user_workspaces(&pool, user.id).await.unwrap();
    let ws_id = workspaces[0].id;

    let app = http::router(pool.clone(), Duration::from_secs(2), config.clone());
    let (session_tok, csrf_tok) = login_user(&app, "alice", "secure-pass-123").await;
    let cookie_header = format!("trestle-session={session_tok}; trestle-csrf={csrf_tok}");

    // 1. List projects initially — empty.
    let list_req = Request::builder()
        .uri(format!("/api/v1/workspaces/{ws_id}/projects"))
        .header(header::COOKIE, &cookie_header)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(list_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(json["projects"].as_array().unwrap().len(), 0);

    // 2. Create project.
    let create_proj_payload = serde_json::json!({
        "name": "Core Platform",
        "slug": "core-platform",
        "key": "CORE",
        "description": "Main platform infrastructure"
    });
    let create_req = Request::builder()
        .method("POST")
        .uri(format!("/api/v1/workspaces/{ws_id}/projects"))
        .header(header::COOKIE, &cookie_header)
        .header("X-CSRF-Token", &csrf_tok)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&create_proj_payload).unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(create_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let proj_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    let proj_id = proj_json["id"].as_str().unwrap().to_string();
    assert_eq!(proj_json["key"], "CORE");
    assert_eq!(proj_json["slug"], "core-platform");
    assert_eq!(proj_json["task_counter"], 0);

    // 3. Conflict on duplicate slug or key.
    let dup_req = Request::builder()
        .method("POST")
        .uri(format!("/api/v1/workspaces/{ws_id}/projects"))
        .header(header::COOKIE, &cookie_header)
        .header("X-CSRF-Token", &csrf_tok)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&create_proj_payload).unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(dup_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CONFLICT);

    // 4. Get project by ID.
    let get_req = Request::builder()
        .uri(format!("/api/v1/workspaces/{ws_id}/projects/{proj_id}"))
        .header(header::COOKIE, &cookie_header)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(get_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let get_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(get_json["name"], "Core Platform");

    // 5. Create Task 1.
    let task1_payload = serde_json::json!({
        "title": "Set up database connection pool",
        "description": "Configure pooling and timeouts",
        "status": "unstarted",
        "priority": "high",
        "due_date": "2026-11-15"
    });
    let t1_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_id}/projects/{proj_id}/tasks"
        ))
        .header(header::COOKIE, &cookie_header)
        .header("X-CSRF-Token", &csrf_tok)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(serde_json::to_vec(&task1_payload).unwrap()))
        .unwrap();
    let res = app.clone().oneshot(t1_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let t1_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    let t1_id = t1_json["id"].as_str().unwrap().to_string();
    assert_eq!(t1_json["human_key"], "CORE-1");
    assert_eq!(t1_json["task_number"], 1);
    assert_eq!(t1_json["priority"], "high");
    assert_eq!(t1_json["status"], "unstarted");
    assert_eq!(t1_json["due_date"], "2026-11-15");

    // 6. Create Task 2.
    let task2_payload = serde_json::json!({
        "title": "Implement healthcheck endpoints",
        "description": "Live and ready checks",
        "status": "backlog",
        "priority": "medium"
    });
    let t2_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_id}/projects/{proj_id}/tasks"
        ))
        .header(header::COOKIE, &cookie_header)
        .header("X-CSRF-Token", &csrf_tok)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(serde_json::to_vec(&task2_payload).unwrap()))
        .unwrap();
    let res = app.clone().oneshot(t2_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let t2_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    let t2_id = t2_json["id"].as_str().unwrap().to_string();
    assert_eq!(t2_json["human_key"], "CORE-2");
    assert_eq!(t2_json["task_number"], 2);

    // 7. List Tasks — paginated and ordered by task_number DESC.
    let list_tasks_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_id}/projects/{proj_id}/tasks"
        ))
        .header(header::COOKIE, &cookie_header)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(list_tasks_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let tasks_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(tasks_json["total"], 2);
    let tasks = tasks_json["tasks"].as_array().unwrap();
    assert_eq!(tasks.len(), 2);
    assert_eq!(tasks[0]["human_key"], "CORE-2");
    assert_eq!(tasks[1]["human_key"], "CORE-1");

    // 8. Filter by status=backlog.
    let filter_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_id}/projects/{proj_id}/tasks?status=backlog"
        ))
        .header(header::COOKIE, &cookie_header)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(filter_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let filtered_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(filtered_json["total"], 1);
    assert_eq!(filtered_json["tasks"][0]["human_key"], "CORE-2");

    // 9. Update Task 1 (PATCH status, priority).
    let update_payload = serde_json::json!({
        "status": "started",
        "priority": "urgent"
    });
    let patch_req = Request::builder()
        .method("PATCH")
        .uri(format!(
            "/api/v1/workspaces/{ws_id}/projects/{proj_id}/tasks/{t1_id}"
        ))
        .header(header::COOKIE, &cookie_header)
        .header("X-CSRF-Token", &csrf_tok)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(serde_json::to_vec(&update_payload).unwrap()))
        .unwrap();
    let res = app.clone().oneshot(patch_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let updated_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(updated_json["status"], "started");
    assert_eq!(updated_json["priority"], "urgent");
    // Verify title and description remained unchanged
    assert_eq!(updated_json["title"], "Set up database connection pool");

    // 10. Delete Task 2.
    let del_req = Request::builder()
        .method("DELETE")
        .uri(format!(
            "/api/v1/workspaces/{ws_id}/projects/{proj_id}/tasks/{t2_id}"
        ))
        .header(header::COOKIE, &cookie_header)
        .header("X-CSRF-Token", &csrf_tok)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(del_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::NO_CONTENT);

    // Verify task count is now 1.
    let count_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_id}/projects/{proj_id}/tasks"
        ))
        .header(header::COOKIE, &cookie_header)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(count_req).await.unwrap();
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let count_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(count_json["total"], 1);

    // 11. Archive Project.
    let archive_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_id}/projects/{proj_id}/archive"
        ))
        .header(header::COOKIE, &cookie_header)
        .header("X-CSRF-Token", &csrf_tok)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(archive_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let arch_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(arch_json["archived"], true);

    // Listing active projects without include_archived returns 0 projects.
    let list_active = Request::builder()
        .uri(format!("/api/v1/workspaces/{ws_id}/projects"))
        .header(header::COOKIE, &cookie_header)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(list_active).await.unwrap();
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let list_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(list_json["projects"].as_array().unwrap().len(), 0);

    // Listing with include_archived=true returns 1 project.
    let list_all = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_id}/projects?include_archived=true"
        ))
        .header(header::COOKIE, &cookie_header)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(list_all).await.unwrap();
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let all_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(all_json["projects"].as_array().unwrap().len(), 1);
}

#[sqlx::test(migrations = false)]
async fn project_and_task_authorization_and_isolation(pool: PgPool) {
    database::migrate(&pool).await.unwrap();

    let config = test_config(&pool);
    let hasher = Hasher::new(4096, 1, 1).unwrap();
    let password_hash = hasher.hash(b"test-pass-word").unwrap();

    // 1. Bootstrap Workspace A with Owner Alice
    let alice = store::bootstrap(
        &pool,
        "alice",
        &password_hash,
        "Alice Smith",
        "Workspace A",
        "ws-a",
    )
    .await
    .unwrap();
    let ws_a = store::list_user_workspaces(&pool, alice.id).await.unwrap()[0].id;

    // 2. Add Bob to the database and create a separate Workspace B for Bob
    let bob = store::create_user(&pool, "bob", &password_hash, "Bob Jones")
        .await
        .unwrap();
    let ws_b = uuid::Uuid::new_v4();
    sqlx::query("INSERT INTO workspaces (id, name, slug) VALUES ($1, $2, $3)")
        .bind(ws_b)
        .bind("Workspace B")
        .bind("ws-b")
        .execute(&pool)
        .await
        .unwrap();
    sqlx::query(
        "INSERT INTO workspace_memberships (workspace_id, user_id, role)
         VALUES ($1, $2, 'owner')",
    )
    .bind(ws_b)
    .bind(bob.id)
    .execute(&pool)
    .await
    .unwrap();

    // 3. Add Charlie to Workspace A as a "member"
    let charlie = store::create_user(&pool, "charlie", &password_hash, "Charlie Brown")
        .await
        .unwrap();
    store::add_workspace_member(&pool, Some(alice.id), ws_a, charlie.id, "member")
        .await
        .unwrap();

    let app = http::router(pool.clone(), Duration::from_secs(2), config.clone());

    // Alice creates Project Alpha in Workspace A
    let (alice_session, alice_csrf) = login_user(&app, "alice", "test-pass-word").await;
    let alice_cookie = format!("trestle-session={alice_session}; trestle-csrf={alice_csrf}");

    let create_req = Request::builder()
        .method("POST")
        .uri(format!("/api/v1/workspaces/{ws_a}/projects"))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "name": "Project Alpha",
                "slug": "proj-alpha",
                "key": "ALPHA",
                "description": "Secret Workspace A Project"
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(create_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let proj_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    let proj_a_id = proj_json["id"].as_str().unwrap().to_string();

    // Create a task in Project Alpha
    let create_task_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_a_id}/tasks"
        ))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "title": "Secret Task in A"
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(create_task_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let task_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    let task_a_id = task_json["id"].as_str().unwrap().to_string();

    // --- Bob's perspective (Non-member of Workspace A) ---
    let (bob_session, bob_csrf) = login_user(&app, "bob", "test-pass-word").await;
    let bob_cookie = format!("trestle-session={bob_session}; trestle-csrf={bob_csrf}");

    // Bob trying to list projects in Workspace A -> concealed 404
    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!("/api/v1/workspaces/{ws_a}/projects"))
                .header(header::COOKIE, &bob_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(res.status(), StatusCode::NOT_FOUND);

    // Bob trying to get Project Alpha -> concealed 404
    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!("/api/v1/workspaces/{ws_a}/projects/{proj_a_id}"))
                .header(header::COOKIE, &bob_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(res.status(), StatusCode::NOT_FOUND);

    // Bob trying to list tasks in Project Alpha -> concealed 404
    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!(
                    "/api/v1/workspaces/{ws_a}/projects/{proj_a_id}/tasks"
                ))
                .header(header::COOKIE, &bob_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(res.status(), StatusCode::NOT_FOUND);

    // Bob trying to create a task in Project Alpha -> concealed 404
    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/workspaces/{ws_a}/projects/{proj_a_id}/tasks"
                ))
                .header(header::COOKIE, &bob_cookie)
                .header("X-CSRF-Token", &bob_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(r#"{"title": "Intruder task"}"#))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(res.status(), StatusCode::NOT_FOUND);

    // Bob trying to patch task in Project Alpha -> concealed 404
    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("PATCH")
                .uri(format!(
                    "/api/v1/workspaces/{ws_a}/projects/{proj_a_id}/tasks/{task_a_id}"
                ))
                .header(header::COOKIE, &bob_cookie)
                .header("X-CSRF-Token", &bob_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(r#"{"title": "Hacked"}"#))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(res.status(), StatusCode::NOT_FOUND);

    // Bob trying to delete task in Project Alpha -> concealed 404
    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!(
                    "/api/v1/workspaces/{ws_a}/projects/{proj_a_id}/tasks/{task_a_id}"
                ))
                .header(header::COOKIE, &bob_cookie)
                .header("X-CSRF-Token", &bob_csrf)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(res.status(), StatusCode::NOT_FOUND);

    // --- Charlie's perspective (Member in Workspace A, not Owner/Admin) ---
    let (charlie_session, charlie_csrf) = login_user(&app, "charlie", "test-pass-word").await;
    let charlie_cookie = format!("trestle-session={charlie_session}; trestle-csrf={charlie_csrf}");

    // Charlie can list projects
    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!("/api/v1/workspaces/{ws_a}/projects"))
                .header(header::COOKIE, &charlie_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(res.status(), StatusCode::OK);

    // Charlie cannot create projects (Forbidden 403)
    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/workspaces/{ws_a}/projects"))
                .header(header::COOKIE, &charlie_cookie)
                .header("X-CSRF-Token", &charlie_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(
                    r#"{"name": "Charlie Proj", "slug": "charlie-p", "key": "CP"}"#,
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(res.status(), StatusCode::FORBIDDEN);

    // Charlie cannot archive project (Forbidden 403)
    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/workspaces/{ws_a}/projects/{proj_a_id}/archive"
                ))
                .header(header::COOKIE, &charlie_cookie)
                .header("X-CSRF-Token", &charlie_csrf)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(res.status(), StatusCode::FORBIDDEN);

    // Charlie CAN create a task in Project Alpha
    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/workspaces/{ws_a}/projects/{proj_a_id}/tasks"
                ))
                .header(header::COOKIE, &charlie_cookie)
                .header("X-CSRF-Token", &charlie_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(r#"{"title": "Charlie's task"}"#))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);

    // Charlie CAN update a task
    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("PATCH")
                .uri(format!(
                    "/api/v1/workspaces/{ws_a}/projects/{proj_a_id}/tasks/{task_a_id}"
                ))
                .header(header::COOKIE, &charlie_cookie)
                .header("X-CSRF-Token", &charlie_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(r#"{"status": "started"}"#))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(res.status(), StatusCode::OK);

    // Charlie CANNOT delete a task (Forbidden 403)
    let res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!(
                    "/api/v1/workspaces/{ws_a}/projects/{proj_a_id}/tasks/{task_a_id}"
                ))
                .header(header::COOKIE, &charlie_cookie)
                .header("X-CSRF-Token", &charlie_csrf)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(res.status(), StatusCode::FORBIDDEN);
}

#[sqlx::test(migrations = false)]
async fn concurrent_task_counter_allocation(pool: PgPool) {
    database::migrate(&pool).await.unwrap();

    let hasher = Hasher::new(4096, 1, 1).unwrap();
    let password_hash = hasher.hash(b"test-pass-word").unwrap();

    let user = store::bootstrap(
        &pool,
        "dave",
        &password_hash,
        "Dave Miller",
        "Concurrent Org",
        "conc-org",
    )
    .await
    .unwrap();

    let ws_id = store::list_user_workspaces(&pool, user.id).await.unwrap()[0].id;

    let project = trestle::projects::store::create_project(
        &pool,
        Some(user.id),
        ws_id,
        trestle::projects::CreateProjectParams {
            name: "Concurrency Test".to_string(),
            slug: "conc-test".to_string(),
            key: "CONC".to_string(),
            description: "Testing concurrent task key counter".to_string(),
        },
    )
    .await
    .unwrap();

    // Spawn 20 tasks concurrently
    let task_count = 20;
    let mut handles = Vec::new();

    for i in 0..task_count {
        let pool = pool.clone();
        let user_id = user.id;
        let proj_id = project.id;
        let handle = tokio::spawn(async move {
            trestle::projects::store::create_task(
                &pool,
                Some(user_id),
                ws_id,
                proj_id,
                trestle::projects::CreateTaskParams {
                    title: format!("Task {i}"),
                    description: "Concurrent creation".to_string(),
                    status: "unstarted".to_string(),
                    priority: "none".to_string(),
                    due_date: None,
                    parent_id: None,
                    status_id: None,
                },
            )
            .await
        });
        handles.push(handle);
    }

    let mut numbers = Vec::new();
    for handle in handles {
        let result = handle.await.unwrap().unwrap();
        numbers.push(result.task_number);
    }

    numbers.sort_unstable();
    assert_eq!(numbers.len(), task_count);
    let expected: Vec<i64> = (1..=(task_count as i64)).collect();
    assert_eq!(numbers, expected);

    // Verify project record reflects counter = 20
    let updated_proj = trestle::projects::store::get_project(&pool, ws_id, project.id)
        .await
        .unwrap()
        .unwrap();
    assert_eq!(updated_proj.task_counter, task_count as i64);
}
