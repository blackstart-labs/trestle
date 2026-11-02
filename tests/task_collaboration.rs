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

    for (k, v) in res.headers() {
        if k == header::SET_COOKIE {
            let cookie_str = v.to_str().unwrap();
            if let Some(val) = extract_cookie(cookie_str, "trestle-session") {
                session_token = Some(val.to_string());
            }
            if let Some(val) = extract_cookie(cookie_str, "trestle-csrf") {
                csrf_token = Some(val.to_string());
            }
        }
    }

    (session_token.unwrap(), csrf_token.unwrap())
}

#[sqlx::test(migrations = false)]
async fn task_collaboration_end_to_end(pool: PgPool) {
    database::migrate(&pool).await.unwrap();

    let config = test_config(&pool);
    let hasher = Hasher::new(4096, 1, 1).unwrap();
    let password_hash = hasher.hash(b"test-collab-pw").unwrap();

    // 1. Bootstrap Workspace A with Owner Alice
    let alice = store::bootstrap(
        &pool,
        "alice_collab",
        &password_hash,
        "Alice Smith",
        "Collab Workspace",
        "collab-ws",
    )
    .await
    .unwrap();
    let ws_a = store::list_user_workspaces(&pool, alice.id).await.unwrap()[0].id;

    // 2. Add Member Bob to Workspace A
    let bob = store::create_user(&pool, "bob_collab", &password_hash, "Bob Developer")
        .await
        .unwrap();
    store::add_workspace_member(&pool, Some(alice.id), ws_a, bob.id, "member")
        .await
        .unwrap();

    // 3. Add Mallory in separate Workspace B
    let mallory = store::create_user(&pool, "mallory_collab", &password_hash, "Mallory Attacker")
        .await
        .unwrap();
    let ws_b = uuid::Uuid::new_v4();
    sqlx::query(
        "INSERT INTO workspaces (id, name, slug)
         VALUES ($1, $2, $3)",
    )
    .bind(ws_b)
    .bind("Mallory Workspace")
    .bind("mallory-ws")
    .execute(&pool)
    .await
    .unwrap();
    sqlx::query(
        "INSERT INTO workspace_memberships (workspace_id, user_id, role)
         VALUES ($1, $2, 'owner')",
    )
    .bind(ws_b)
    .bind(mallory.id)
    .execute(&pool)
    .await
    .unwrap();

    let app = http::router(pool.clone(), Duration::from_secs(5), config.clone());

    let (alice_sess, alice_csrf) = login_user(&app, "alice_collab", "test-collab-pw").await;
    let (bob_sess, bob_csrf) = login_user(&app, "bob_collab", "test-collab-pw").await;
    let (mallory_sess, mallory_csrf) = login_user(&app, "mallory_collab", "test-collab-pw").await;

    let alice_cookie = format!("trestle-session={alice_sess}; trestle-csrf={alice_csrf}");
    let bob_cookie = format!("trestle-session={bob_sess}; trestle-csrf={bob_csrf}");
    let mallory_cookie = format!("trestle-session={mallory_sess}; trestle-csrf={mallory_csrf}");

    // 4. Alice creates a Project
    let create_proj_req = Request::builder()
        .method("POST")
        .uri(format!("/api/v1/workspaces/{ws_a}/projects"))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "name": "Platform Redesign",
                "slug": "platform-redesign",
                "key": "PLAT",
                "description": "Modern frontend & collaboration tools"
            }))
            .unwrap(),
        ))
        .unwrap();

    let res = app.clone().oneshot(create_proj_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let proj_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    let proj_id = proj_json["id"].as_str().unwrap();

    // 5. Label Management: Create Labels
    let create_label_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/labels"
        ))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "name": "frontend",
                "color": "#315b3b"
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(create_label_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let label1_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    let label1_id = label1_json["id"].as_str().unwrap();
    assert_eq!(label1_json["name"], "frontend");
    assert_eq!(label1_json["color"], "#315b3b");

    // Duplicate label returns 409 Conflict
    let dup_label_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/labels"
        ))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "name": "frontend",
                "color": "#112233"
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(dup_label_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CONFLICT);

    // Create second label: "bug"
    let create_label2_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/labels"
        ))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "name": "bug",
                "color": "#9f302b"
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(create_label2_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let label2_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    let label2_id = label2_json["id"].as_str().unwrap();

    // List labels returns both
    let list_labels_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/labels"
        ))
        .header(header::COOKIE, &alice_cookie)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(list_labels_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let labels_list: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(labels_list["labels"].as_array().unwrap().len(), 2);

    // 6. Create Main Task
    let create_task_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks"
        ))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "title": "Implement task detail drawer",
                "description": "Rich drawer showing details, assignees, and comments",
                "status": "started",
                "priority": "high"
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(create_task_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let task_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    let task_id = task_json["id"].as_str().unwrap();
    assert_eq!(task_json["human_key"], "PLAT-1");

    // 7. Attach Label to Task
    let attach_label_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/labels"
        ))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "label_id": label1_id
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(attach_label_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::NO_CONTENT);

    // 8. Assign Member Bob to Task
    let assign_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/assignees"
        ))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "user_id": bob.id
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(assign_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let assignee_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(assignee_json["username"], "bob_collab");

    // Attempting to assign non-member Mallory returns 404 (concealed boundary)
    let assign_invalid_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/assignees"
        ))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "user_id": mallory.id
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(assign_invalid_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::NOT_FOUND);

    // Inspect task: verify labels and assignees are populated
    let get_task_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}"
        ))
        .header(header::COOKIE, &alice_cookie)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(get_task_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let task_detail: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(task_detail["labels"].as_array().unwrap().len(), 1);
    assert_eq!(task_detail["labels"][0]["name"], "frontend");
    assert_eq!(task_detail["assignees"].as_array().unwrap().len(), 1);
    assert_eq!(task_detail["assignees"][0]["username"], "bob_collab");

    // 9. Subtasks: Create Subtasks
    let create_subtask1_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/subtasks"
        ))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "title": "Build drawer layout",
                "status": "completed",
                "priority": "medium"
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(create_subtask1_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let subtask1_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(subtask1_json["human_key"], "PLAT-2");
    assert_eq!(subtask1_json["parent_id"], task_id);

    let create_subtask2_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/subtasks"
        ))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "title": "Add keyboard focus trap and ESC handler",
                "status": "unstarted",
                "priority": "high"
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(create_subtask2_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);

    // List subtasks returns 2 subtasks
    let list_subtasks_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/subtasks"
        ))
        .header(header::COOKIE, &alice_cookie)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(list_subtasks_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let subtasks: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(subtasks.as_array().unwrap().len(), 2);

    // 10. Threaded Comments
    // Bob posts a comment
    let bob_comment_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/comments"
        ))
        .header(header::COOKIE, &bob_cookie)
        .header("X-CSRF-Token", &bob_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "content": "I am working on the drawer subtasks today."
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(bob_comment_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let bob_comment_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    let bob_comment_id = bob_comment_json["id"].as_str().unwrap();
    assert_eq!(bob_comment_json["author_username"], "bob_collab");

    // Alice posts a comment
    let alice_comment_req = Request::builder()
        .method("POST")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/comments"
        ))
        .header(header::COOKIE, &alice_cookie)
        .header("X-CSRF-Token", &alice_csrf)
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(
            serde_json::to_vec(&serde_json::json!({
                "content": "Sounds good! Make sure to verify keyboard ESC behavior."
            }))
            .unwrap(),
        ))
        .unwrap();
    let res = app.clone().oneshot(alice_comment_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::CREATED);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let alice_comment_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    let alice_comment_id = alice_comment_json["id"].as_str().unwrap();

    // List comments
    let list_comments_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/comments"
        ))
        .header(header::COOKIE, &alice_cookie)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(list_comments_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let comments_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(comments_json["comments"].as_array().unwrap().len(), 2);

    // Bob cannot delete Alice's comment (returns 403 Forbidden)
    let bob_delete_alice_comment = Request::builder()
        .method("DELETE")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/comments/{alice_comment_id}"
        ))
        .header(header::COOKIE, &bob_cookie)
        .header("X-CSRF-Token", &bob_csrf)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(bob_delete_alice_comment).await.unwrap();
    assert_eq!(res.status(), StatusCode::FORBIDDEN);

    // Bob can delete his own comment
    let bob_delete_own_comment = Request::builder()
        .method("DELETE")
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/comments/{bob_comment_id}"
        ))
        .header(header::COOKIE, &bob_cookie)
        .header("X-CSRF-Token", &bob_csrf)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(bob_delete_own_comment).await.unwrap();
    assert_eq!(res.status(), StatusCode::NO_CONTENT);

    // 11. Activity Feed
    let activity_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/activity"
        ))
        .header(header::COOKIE, &alice_cookie)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(activity_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let act_json: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    let activities = act_json["activities"].as_array().unwrap();
    assert!(!activities.is_empty());
    let actions: Vec<&str> = activities
        .iter()
        .map(|a| a["action"].as_str().unwrap())
        .collect();
    assert!(actions.contains(&"task.created"));
    assert!(actions.contains(&"label.attached"));
    assert!(actions.contains(&"user.assigned"));
    assert!(actions.contains(&"subtask.created"));
    assert!(actions.contains(&"comment.added"));

    // 12. Multi-Dimensional Filtering on task list
    // Filter by label_id=label1_id (frontend)
    let filter_label_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks?label_id={label1_id}"
        ))
        .header(header::COOKIE, &alice_cookie)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(filter_label_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let filtered: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(filtered["total"], 1);
    assert_eq!(filtered["tasks"][0]["id"], task_id);

    // Filter by label_id=label2_id (bug, which has 0 tasks attached)
    let filter_bug_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks?label_id={label2_id}"
        ))
        .header(header::COOKIE, &alice_cookie)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(filter_bug_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let filtered_bug: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(filtered_bug["total"], 0);

    // Filter by assignee_id=bob.id
    let filter_bob_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks?assignee_id={}",
            bob.id
        ))
        .header(header::COOKIE, &alice_cookie)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(filter_bob_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::OK);
    let bytes = res.into_body().collect().await.unwrap().to_bytes();
    let filtered_bob: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(filtered_bob["total"], 1);

    // 13. Tenant Isolation: Mallory cannot access Workspace A resources (all return 404)
    let mallory_task_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}"
        ))
        .header(header::COOKIE, &mallory_cookie)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(mallory_task_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::NOT_FOUND);

    let mallory_labels_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/labels"
        ))
        .header(header::COOKIE, &mallory_cookie)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(mallory_labels_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::NOT_FOUND);

    let mallory_subtasks_req = Request::builder()
        .uri(format!(
            "/api/v1/workspaces/{ws_a}/projects/{proj_id}/tasks/{task_id}/subtasks"
        ))
        .header(header::COOKIE, &mallory_cookie)
        .body(Body::empty())
        .unwrap();
    let res = app.clone().oneshot(mallory_subtasks_req).await.unwrap();
    assert_eq!(res.status(), StatusCode::NOT_FOUND);
}
