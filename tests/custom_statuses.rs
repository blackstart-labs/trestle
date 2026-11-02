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
async fn custom_statuses_lifecycle_and_remap(pool: PgPool) {
    database::migrate(&pool).await.unwrap();

    let config = test_config(&pool);
    let hasher = Hasher::new(4096, 1, 1).unwrap();
    let password_hash = hasher.hash(b"test-status-pw").unwrap();

    // 1. Bootstrap Workspace with Owner Alice
    let alice = store::bootstrap(
        &pool,
        "alice_status",
        &password_hash,
        "Alice Admin",
        "Status Workspace",
        "status-ws",
    )
    .await
    .unwrap();
    let ws_id = store::list_user_workspaces(&pool, alice.id).await.unwrap()[0].id;

    // 2. Add Member Bob
    let bob = store::create_user(&pool, "bob_status", &password_hash, "Bob Member")
        .await
        .unwrap();
    store::add_workspace_member(&pool, Some(alice.id), ws_id, bob.id, "member")
        .await
        .unwrap();

    let app = http::router(pool.clone(), Duration::from_secs(5), config);

    let (alice_session, alice_csrf) = login_user(&app, "alice_status", "test-status-pw").await;
    let (bob_session, bob_csrf) = login_user(&app, "bob_status", "test-status-pw").await;

    // 3. Create a project
    let create_proj_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/workspaces/{}/projects", ws_id))
                .header(header::COOKIE, format!("trestle-session={}", alice_session))
                .header("x-csrf-token", &alice_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(
                    serde_json::to_vec(&serde_json::json!({
                        "name": "Status Test Project",
                        "slug": "status-proj",
                        "key": "STAT"
                    }))
                    .unwrap(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(create_proj_res.status(), StatusCode::CREATED);
    let proj_body = create_proj_res
        .into_body()
        .collect()
        .await
        .unwrap()
        .to_bytes();
    let proj_json: serde_json::Value = serde_json::from_slice(&proj_body).unwrap();
    let proj_id = proj_json["id"].as_str().unwrap();

    // 4. Verify 5 default seeded statuses
    let list_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("GET")
                .uri(format!(
                    "/api/v1/workspaces/{}/projects/{}/statuses",
                    ws_id, proj_id
                ))
                .header(header::COOKIE, format!("trestle-session={}", bob_session))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(list_res.status(), StatusCode::OK);
    let list_bytes = list_res.into_body().collect().await.unwrap().to_bytes();
    let list_json: serde_json::Value = serde_json::from_slice(&list_bytes).unwrap();
    let statuses = list_json["statuses"].as_array().unwrap();
    assert_eq!(statuses.len(), 5);

    let default_names: Vec<&str> = statuses
        .iter()
        .map(|s| s["name"].as_str().unwrap())
        .collect();
    assert_eq!(
        default_names,
        vec!["Backlog", "To Do", "In Progress", "Done", "Cancelled"]
    );

    let default_todo_id = statuses[1]["id"].as_str().unwrap().to_string();
    let default_done_id = statuses[3]["id"].as_str().unwrap().to_string();

    // 5. Member Bob attempts to create status -> 403 Forbidden
    let bob_create = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/workspaces/{}/projects/{}/statuses",
                    ws_id, proj_id
                ))
                .header(header::COOKIE, format!("trestle-session={}", bob_session))
                .header("x-csrf-token", &bob_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(
                    serde_json::to_vec(&serde_json::json!({
                        "name": "Blocked",
                        "category": "started",
                        "color": "#9f302b"
                    }))
                    .unwrap(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(bob_create.status(), StatusCode::FORBIDDEN);

    // 6. Alice creates custom status "In QA"
    let create_status_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/workspaces/{}/projects/{}/statuses",
                    ws_id, proj_id
                ))
                .header(header::COOKIE, format!("trestle-session={}", alice_session))
                .header("x-csrf-token", &alice_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(
                    serde_json::to_vec(&serde_json::json!({
                        "name": "In QA",
                        "category": "started",
                        "color": "#775300"
                    }))
                    .unwrap(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(create_status_res.status(), StatusCode::CREATED);
    let qa_bytes = create_status_res
        .into_body()
        .collect()
        .await
        .unwrap()
        .to_bytes();
    let qa_json: serde_json::Value = serde_json::from_slice(&qa_bytes).unwrap();
    let qa_id = qa_json["id"].as_str().unwrap().to_string();
    assert_eq!(qa_json["name"], "In QA");
    assert_eq!(qa_json["category"], "started");
    assert_eq!(qa_json["position"], 5);

    // 7. Duplicate status name returns 409 Conflict
    let dup_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/workspaces/{}/projects/{}/statuses",
                    ws_id, proj_id
                ))
                .header(header::COOKIE, format!("trestle-session={}", alice_session))
                .header("x-csrf-token", &alice_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(
                    serde_json::to_vec(&serde_json::json!({
                        "name": "In QA",
                        "category": "started"
                    }))
                    .unwrap(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(dup_res.status(), StatusCode::CONFLICT);

    // 8. Update status name and color
    let update_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("PATCH")
                .uri(format!(
                    "/api/v1/workspaces/{}/projects/{}/statuses/{}",
                    ws_id, proj_id, qa_id
                ))
                .header(header::COOKIE, format!("trestle-session={}", alice_session))
                .header("x-csrf-token", &alice_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(
                    serde_json::to_vec(&serde_json::json!({
                        "name": "QA Testing",
                        "color": "#8c6200"
                    }))
                    .unwrap(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(update_res.status(), StatusCode::OK);
    let updated_bytes = update_res.into_body().collect().await.unwrap().to_bytes();
    let updated_json: serde_json::Value = serde_json::from_slice(&updated_bytes).unwrap();
    assert_eq!(updated_json["name"], "QA Testing");
    assert_eq!(updated_json["color"], "#8c6200");

    // 9. Create task assigned to QA Testing
    let create_task_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/workspaces/{}/projects/{}/tasks",
                    ws_id, proj_id
                ))
                .header(header::COOKIE, format!("trestle-session={}", bob_session))
                .header("x-csrf-token", &bob_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(
                    serde_json::to_vec(&serde_json::json!({
                        "title": "Verify status workflow",
                        "status_id": qa_id
                    }))
                    .unwrap(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(create_task_res.status(), StatusCode::CREATED);
    let task_bytes = create_task_res
        .into_body()
        .collect()
        .await
        .unwrap()
        .to_bytes();
    let task_json: serde_json::Value = serde_json::from_slice(&task_bytes).unwrap();
    let task_id = task_json["id"].as_str().unwrap().to_string();
    assert_eq!(task_json["status_id"], qa_id);
    assert_eq!(task_json["status_name"], "QA Testing");
    assert_eq!(task_json["status"], "started");

    // 10. Deleting status with active tasks fails without remap_to
    let delete_fail = app
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!(
                    "/api/v1/workspaces/{}/projects/{}/statuses/{}",
                    ws_id, proj_id, qa_id
                ))
                .header(header::COOKIE, format!("trestle-session={}", alice_session))
                .header("x-csrf-token", &alice_csrf)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(delete_fail.status(), StatusCode::CONFLICT);

    // 11. Deleting status with valid remap_to reassigns tasks and deletes status
    let delete_ok = app
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!(
                    "/api/v1/workspaces/{}/projects/{}/statuses/{}?remap_to={}",
                    ws_id, proj_id, qa_id, default_done_id
                ))
                .header(header::COOKIE, format!("trestle-session={}", alice_session))
                .header("x-csrf-token", &alice_csrf)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(delete_ok.status(), StatusCode::NO_CONTENT);

    // 12. Check task has been remapped to default_done_id and status is "completed"
    let check_task = app
        .clone()
        .oneshot(
            Request::builder()
                .method("GET")
                .uri(format!(
                    "/api/v1/workspaces/{}/projects/{}/tasks/{}",
                    ws_id, proj_id, task_id
                ))
                .header(header::COOKIE, format!("trestle-session={}", bob_session))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(check_task.status(), StatusCode::OK);
    let check_bytes = check_task.into_body().collect().await.unwrap().to_bytes();
    let check_json: serde_json::Value = serde_json::from_slice(&check_bytes).unwrap();
    assert_eq!(check_json["status_id"], default_done_id);
    assert_eq!(check_json["status_name"], "Done");
    assert_eq!(check_json["status"], "completed");

    // 13. Reordering statuses
    let reorder_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("PUT")
                .uri(format!(
                    "/api/v1/workspaces/{}/projects/{}/statuses/positions",
                    ws_id, proj_id
                ))
                .header(header::COOKIE, format!("trestle-session={}", alice_session))
                .header("x-csrf-token", &alice_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(
                    serde_json::to_vec(&serde_json::json!({
                        "status_ids": [default_done_id, default_todo_id]
                    }))
                    .unwrap(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(reorder_res.status(), StatusCode::NO_CONTENT);
}
