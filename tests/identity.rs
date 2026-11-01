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
    identity::{
        Hasher,
        store::{self, StoreError},
    },
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

#[sqlx::test(migrations = false)]
async fn bootstrap_creates_user_workspace_and_owner_membership_atomically(pool: PgPool) {
    database::migrate(&pool).await.unwrap();

    assert!(!store::any_user_exists(&pool).await.unwrap());

    let hasher = Hasher::new(4096, 1, 1).unwrap();
    let password_hash = hasher.hash(b"correct-horse-battery-staple").unwrap();

    let bootstrap_res = store::bootstrap(
        &pool,
        "admin",
        &password_hash,
        "Administrator",
        "Acme Corp",
        "acme",
    )
    .await
    .unwrap();

    assert_eq!(bootstrap_res.username, "admin");
    assert_eq!(bootstrap_res.display_name, "Administrator");

    let workspaces = store::list_user_workspaces(&pool, bootstrap_res.id)
        .await
        .unwrap();
    assert_eq!(workspaces.len(), 1);
    assert_eq!(workspaces[0].slug, "acme");
    assert_eq!(workspaces[0].name, "Acme Corp");
    assert_eq!(workspaces[0].role, "owner");

    assert!(store::any_user_exists(&pool).await.unwrap());

    // Second bootstrap attempt must fail with Conflict.
    let second_res = store::bootstrap(
        &pool,
        "operator",
        &password_hash,
        "Operator",
        "Second Org",
        "second",
    )
    .await;

    assert!(matches!(second_res, Err(StoreError::Conflict)));
}

#[sqlx::test(migrations = false)]
async fn session_creation_resolution_and_revocation(pool: PgPool) {
    database::migrate(&pool).await.unwrap();

    let hasher = Hasher::new(4096, 1, 1).unwrap();
    let password_hash = hasher.hash(b"password123").unwrap();

    let bootstrapped = store::bootstrap(
        &pool,
        "alice",
        &password_hash,
        "Alice Smith",
        "Alice Workspace",
        "alice-ws",
    )
    .await
    .unwrap();

    let (session_token, csrf_token) = store::create_session(&pool, bootstrapped.id).await.unwrap();

    // Resolving with valid token returns user and CSRF hash.
    let resolved = store::resolve_session(&pool, &session_token)
        .await
        .unwrap()
        .expect("session must resolve");

    assert_eq!(resolved.user.id, bootstrapped.id);
    assert_eq!(resolved.user.username, "alice");
    let expected_csrf_hash = trestle::identity::hash_client_token(&csrf_token).unwrap();
    assert_eq!(resolved.csrf_token_hash, expected_csrf_hash.as_str());

    // Revoking session makes it unresolvable.
    store::revoke_session(&pool, resolved.session_id)
        .await
        .unwrap();

    let after_revoke = store::resolve_session(&pool, &session_token).await.unwrap();
    assert!(after_revoke.is_none());
}

#[sqlx::test(migrations = false)]
async fn workspace_list_is_scoped_to_user_memberships(pool: PgPool) {
    database::migrate(&pool).await.unwrap();

    let hasher = Hasher::new(4096, 1, 1).unwrap();
    let password_hash = hasher.hash(b"password123").unwrap();

    let bootstrapped = store::bootstrap(
        &pool,
        "bob",
        &password_hash,
        "Bob Jones",
        "Bob Org",
        "bob-org",
    )
    .await
    .unwrap();

    let workspaces = store::list_user_workspaces(&pool, bootstrapped.id)
        .await
        .unwrap();
    assert_eq!(workspaces.len(), 1);
    assert_eq!(workspaces[0].slug, "bob-org");
    assert_eq!(workspaces[0].role, "owner");
}

#[sqlx::test(migrations = false)]
async fn http_login_success_and_logout_flow(pool: PgPool) {
    database::migrate(&pool).await.unwrap();
    let config = test_config(&pool);

    let hasher = Hasher::new(4096, 1, 1).unwrap();
    let password_hash = hasher.hash(b"valid-password-123").unwrap();

    store::bootstrap(
        &pool,
        "carol",
        &password_hash,
        "Carol Danvers",
        "Carol Base",
        "carol-base",
    )
    .await
    .unwrap();

    let app = http::router(pool.clone(), Duration::from_secs(2), config.clone());

    // 1. Unauthenticated requests to /api/v1/me return 401.
    let unauth_res = app
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/me")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(unauth_res.status(), StatusCode::UNAUTHORIZED);

    // 2. Unauthenticated requests to /api/v1/workspaces return 401.
    let unauth_ws_res = app
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/workspaces")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(unauth_ws_res.status(), StatusCode::UNAUTHORIZED);

    // 3. Login with correct credentials returns 200 and cookies.
    let login_body = serde_json::json!({
        "username": "Carol", // mixed-case should normalize to lowercase
        "password": "valid-password-123"
    });

    let login_res = app
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

    assert_eq!(login_res.status(), StatusCode::OK);

    // Extract cookies from response headers.
    let mut session_cookie_val = None;
    let mut csrf_cookie_val = None;

    for cookie_header in login_res.headers().get_all(header::SET_COOKIE) {
        let cookie_str = cookie_header.to_str().unwrap();
        if let Some(val) = extract_cookie(cookie_str, "trestle-session") {
            session_cookie_val = Some(val.to_owned());
        }
        if let Some(val) = extract_cookie(cookie_str, "trestle-csrf") {
            csrf_cookie_val = Some(val.to_owned());
        }
    }

    let session_token = session_cookie_val.expect("session cookie must be set");
    let csrf_token = csrf_cookie_val.expect("csrf cookie must be set");

    let cookie_header = format!("trestle-session={session_token}; trestle-csrf={csrf_token}");

    // 4. Authenticated request to /api/v1/me returns user details.
    let me_res = app
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/me")
                .header(header::COOKIE, &cookie_header)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(me_res.status(), StatusCode::OK);
    let me_bytes = me_res.into_body().collect().await.unwrap().to_bytes();
    let me_json: serde_json::Value = serde_json::from_slice(&me_bytes).unwrap();
    assert_eq!(me_json["username"], "carol");
    assert_eq!(me_json["display_name"], "Carol Danvers");

    // 5. Authenticated request to /api/v1/workspaces returns workspace list.
    let ws_res = app
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/workspaces")
                .header(header::COOKIE, &cookie_header)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(ws_res.status(), StatusCode::OK);
    let ws_bytes = ws_res.into_body().collect().await.unwrap().to_bytes();
    let ws_json: serde_json::Value = serde_json::from_slice(&ws_bytes).unwrap();
    assert_eq!(ws_json["workspaces"][0]["slug"], "carol-base");
    assert_eq!(ws_json["workspaces"][0]["role"], "owner");

    // 6. Logout without CSRF header returns 403 Forbidden.
    let logout_no_csrf = app
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri("/api/v1/session")
                .header(header::COOKIE, &cookie_header)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(logout_no_csrf.status(), StatusCode::FORBIDDEN);

    // 7. Logout with valid CSRF header returns 200/204.
    let logout_with_csrf = app
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri("/api/v1/session")
                .header(header::COOKIE, &cookie_header)
                .header("x-csrf-token", &csrf_token)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert!(
        logout_with_csrf.status() == StatusCode::OK
            || logout_with_csrf.status() == StatusCode::NO_CONTENT
    );

    // 8. Subsequent request to /api/v1/me fails with 401 because session is revoked.
    let me_after_logout = app
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/me")
                .header(header::COOKIE, &cookie_header)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(me_after_logout.status(), StatusCode::UNAUTHORIZED);
}

#[sqlx::test(migrations = false)]
async fn http_login_rate_limiting(pool: PgPool) {
    database::migrate(&pool).await.unwrap();
    let config = test_config(&pool);

    let app = http::router(pool.clone(), Duration::from_secs(2), config);

    let bad_body = serde_json::json!({
        "username": "ratelimited_user",
        "password": "wrong-password"
    });

    // 5 failures should be 401.
    for _ in 0..5 {
        let res = app
            .clone()
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/api/v1/session")
                    .header(header::CONTENT_TYPE, "application/json")
                    .body(Body::from(serde_json::to_vec(&bad_body).unwrap()))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::UNAUTHORIZED);
    }

    // 6th attempt must be 429 Too Many Requests with Retry-After header.
    let sixth_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/session")
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&bad_body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(sixth_res.status(), StatusCode::TOO_MANY_REQUESTS);
    assert!(sixth_res.headers().contains_key(header::RETRY_AFTER));
}

fn extract_cookie<'a>(cookie_str: &'a str, name: &str) -> Option<&'a str> {
    for part in cookie_str.split(';') {
        let part = part.trim();
        if let Some(val) = part.strip_prefix(name).and_then(|v| v.strip_prefix('=')) {
            return Some(val);
        }
    }
    None
}
