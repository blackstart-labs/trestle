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

#[sqlx::test(migrations = false)]
async fn operator_user_provisioning_and_recovery(pool: PgPool) {
    database::migrate(&pool).await.unwrap();
    let config = test_config(&pool);
    let app = http::router(pool.clone(), Duration::from_secs(2), config);

    let hasher = Hasher::new(4096, 1, 1).unwrap();
    let hash1 = hasher.hash(b"initial-password-123").unwrap();

    // 1. Provision user via store::create_user
    let user = store::create_user(&pool, "operator_user", &hash1, "Operator User")
        .await
        .unwrap();
    assert_eq!(user.username, "operator_user");
    assert!(!user.disabled);

    // 2. Provisioning duplicate username fails with Conflict
    let dup_err = store::create_user(&pool, "operator_user", &hash1, "Another User").await;
    assert!(matches!(dup_err, Err(StoreError::Conflict)));

    // 3. User has no workspace memberships
    let user_ws = store::list_user_workspaces(&pool, user.id).await.unwrap();
    assert!(user_ws.is_empty());

    // 4. User can log in
    let login_body = serde_json::json!({
        "username": "operator_user",
        "password": "initial-password-123"
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

    let cookies: Vec<_> = login_res
        .headers()
        .get_all(header::SET_COOKIE)
        .into_iter()
        .collect();
    let session_cookie = cookies
        .iter()
        .find(|c| c.to_str().unwrap().contains("trestle-session"))
        .unwrap()
        .to_str()
        .unwrap();

    // Session is valid
    let me_res = app
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/me")
                .header(header::COOKIE, session_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(me_res.status(), StatusCode::OK);

    // 5. Operator resets password -> revokes all active sessions
    let hash2 = hasher.hash(b"recovered-password-456").unwrap();
    let reset_user = store::reset_user_password(&pool, "operator_user", &hash2)
        .await
        .unwrap();
    assert_eq!(reset_user.id, user.id);

    // Previous session is now revoked!
    let me_after_reset = app
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/me")
                .header(header::COOKIE, session_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(me_after_reset.status(), StatusCode::UNAUTHORIZED);

    // Old password fails to log in
    let old_login = app
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
    assert_eq!(old_login.status(), StatusCode::UNAUTHORIZED);

    // New password succeeds
    let new_login_body = serde_json::json!({
        "username": "operator_user",
        "password": "recovered-password-456"
    });
    let new_login_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/session")
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&new_login_body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(new_login_res.status(), StatusCode::OK);

    let new_session_cookie = new_login_res
        .headers()
        .get_all(header::SET_COOKIE)
        .into_iter()
        .find(|c| c.to_str().unwrap().contains("trestle-session"))
        .unwrap()
        .to_str()
        .unwrap();

    // 6. Disable user -> invalidates active session & prevents login
    store::set_user_disabled(&pool, "operator_user", true)
        .await
        .unwrap();

    let me_disabled = app
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/me")
                .header(header::COOKIE, new_session_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(me_disabled.status(), StatusCode::UNAUTHORIZED);

    let login_when_disabled = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/session")
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&new_login_body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(login_when_disabled.status(), StatusCode::UNAUTHORIZED);

    // 7. Re-enable user
    store::set_user_disabled(&pool, "operator_user", false)
        .await
        .unwrap();

    let login_reenabled = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/session")
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&new_login_body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(login_reenabled.status(), StatusCode::OK);
}

#[sqlx::test(migrations = false)]
async fn workspace_memberships_lifecycle_and_authorization(pool: PgPool) {
    database::migrate(&pool).await.unwrap();
    let config = test_config(&pool);
    let app = http::router(pool.clone(), Duration::from_secs(2), config);

    let hasher = Hasher::new(4096, 1, 1).unwrap();
    let password_hash = hasher.hash(b"owner-password-123").unwrap();

    // 1. Bootstrap initial owner and workspace
    let owner = store::bootstrap(
        &pool,
        "owner_user",
        &password_hash,
        "Workspace Owner",
        "Trestle HQ",
        "trestle-hq",
    )
    .await
    .unwrap();

    let workspaces = store::list_user_workspaces(&pool, owner.id).await.unwrap();
    let ws_id = workspaces[0].id;

    // Provision alice (to be admin) and bob (to be member) and charlie (non-member)
    let alice_hash = hasher.hash(b"alice-password-123").unwrap();
    let alice = store::create_user(&pool, "alice", &alice_hash, "Alice Cooper")
        .await
        .unwrap();

    let bob_hash = hasher.hash(b"bob-password-123").unwrap();
    let bob = store::create_user(&pool, "bob", &bob_hash, "Bob Builder")
        .await
        .unwrap();

    let charlie_hash = hasher.hash(b"charlie-password-123").unwrap();
    let _charlie = store::create_user(&pool, "charlie", &charlie_hash, "Charlie Outsider")
        .await
        .unwrap();

    // Helper to log in and get (session_cookie, csrf_token)
    let login = |user: &'static str, pass: &'static str| {
        let app = app.clone();
        async move {
            let body = serde_json::json!({ "username": user, "password": pass });
            let res = app
                .oneshot(
                    Request::builder()
                        .method("POST")
                        .uri("/api/v1/session")
                        .header(header::CONTENT_TYPE, "application/json")
                        .body(Body::from(serde_json::to_vec(&body).unwrap()))
                        .unwrap(),
                )
                .await
                .unwrap();
            assert_eq!(res.status(), StatusCode::OK);
            let cookies: Vec<_> = res
                .headers()
                .get_all(header::SET_COOKIE)
                .into_iter()
                .collect();
            let sess = cookies
                .iter()
                .find(|c| c.to_str().unwrap().contains("trestle-session"))
                .unwrap()
                .to_str()
                .unwrap()
                .to_owned();
            let csrf_raw = cookies
                .iter()
                .find(|c| c.to_str().unwrap().contains("trestle-csrf"))
                .unwrap()
                .to_str()
                .unwrap();
            let csrf = extract_cookie(csrf_raw, "trestle-csrf").unwrap().to_owned();
            (sess, csrf)
        }
    };

    let (owner_sess, owner_csrf) = login("owner_user", "owner-password-123").await;
    let (charlie_sess, _) = login("charlie", "charlie-password-123").await;

    // 2. Charlie (non-member) gets concealed 404 when querying workspace members
    let charlie_list = app
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!("/api/v1/workspaces/{ws_id}/members"))
                .header(header::COOKIE, &charlie_sess)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(charlie_list.status(), StatusCode::NOT_FOUND);

    // 3. Owner lists members -> 1 member (owner)
    let owner_list = app
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!("/api/v1/workspaces/{ws_id}/members"))
                .header(header::COOKIE, &owner_sess)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(owner_list.status(), StatusCode::OK);

    // 4. Owner adds Alice as 'admin'
    let add_alice_body = serde_json::json!({
        "username": "alice",
        "role": "admin"
    });
    let add_alice_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/workspaces/{ws_id}/members"))
                .header(header::COOKIE, &owner_sess)
                .header("X-CSRF-Token", &owner_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&add_alice_body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(add_alice_res.status(), StatusCode::CREATED);

    // Adding Alice again -> 409 Conflict
    let add_alice_dup = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/workspaces/{ws_id}/members"))
                .header(header::COOKIE, &owner_sess)
                .header("X-CSRF-Token", &owner_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&add_alice_body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(add_alice_dup.status(), StatusCode::CONFLICT);

    // 5. Alice logs in as admin
    let (alice_sess, alice_csrf) = login("alice", "alice-password-123").await;

    // Alice (admin) adds Bob as 'member'
    let add_bob_body = serde_json::json!({
        "username": "bob",
        "role": "member"
    });
    let add_bob_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/workspaces/{ws_id}/members"))
                .header(header::COOKIE, &alice_sess)
                .header("X-CSRF-Token", &alice_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&add_bob_body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(add_bob_res.status(), StatusCode::CREATED);

    // Alice (admin) attempts to add Charlie as 'owner' -> 400 Bad Request (cannot add as owner)
    let add_charlie_owner = serde_json::json!({
        "username": "charlie",
        "role": "owner"
    });
    let add_charlie_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/workspaces/{ws_id}/members"))
                .header(header::COOKIE, &alice_sess)
                .header("X-CSRF-Token", &alice_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&add_charlie_owner).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(add_charlie_res.status(), StatusCode::BAD_REQUEST);

    // 6. Alice (admin) attempts to demote Owner -> 403 Forbidden
    let demote_owner_body = serde_json::json!({ "role": "member" });
    let demote_owner_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("PATCH")
                .uri(format!("/api/v1/workspaces/{ws_id}/members/{}", owner.id))
                .header(header::COOKIE, &alice_sess)
                .header("X-CSRF-Token", &alice_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&demote_owner_body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(demote_owner_res.status(), StatusCode::FORBIDDEN);

    // Alice (admin) attempts to remove Owner -> 403 Forbidden
    let remove_owner_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!("/api/v1/workspaces/{ws_id}/members/{}", owner.id))
                .header(header::COOKIE, &alice_sess)
                .header("X-CSRF-Token", &alice_csrf)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(remove_owner_res.status(), StatusCode::FORBIDDEN);

    // 7. Last Active Owner Protection:
    // Owner attempts to demote themselves while sole owner -> 409 Conflict
    let owner_self_demote = app
        .clone()
        .oneshot(
            Request::builder()
                .method("PATCH")
                .uri(format!("/api/v1/workspaces/{ws_id}/members/{}", owner.id))
                .header(header::COOKIE, &owner_sess)
                .header("X-CSRF-Token", &owner_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&demote_owner_body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(owner_self_demote.status(), StatusCode::CONFLICT);

    // Owner attempts to remove themselves while sole owner -> 409 Conflict
    let owner_self_remove = app
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!("/api/v1/workspaces/{ws_id}/members/{}", owner.id))
                .header(header::COOKIE, &owner_sess)
                .header("X-CSRF-Token", &owner_csrf)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(owner_self_remove.status(), StatusCode::CONFLICT);

    // Owner attempts to disable self via CLI while sole owner -> Conflict
    let disable_sole_owner = store::set_user_disabled(&pool, "owner_user", true).await;
    assert!(matches!(disable_sole_owner, Err(StoreError::Conflict)));

    // 8. Ownership Transfer with Reauthentication
    // Bad password -> 401 Unauthorized
    let bad_transfer_body = serde_json::json!({
        "target_user_id": alice.id,
        "password": "wrong-password"
    });
    let bad_transfer_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/workspaces/{ws_id}/transfer-ownership"))
                .header(header::COOKIE, &owner_sess)
                .header("X-CSRF-Token", &owner_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&bad_transfer_body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(bad_transfer_res.status(), StatusCode::UNAUTHORIZED);

    // Good password -> 200 OK
    let good_transfer_body = serde_json::json!({
        "target_user_id": alice.id,
        "password": "owner-password-123"
    });
    let good_transfer_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/workspaces/{ws_id}/transfer-ownership"))
                .header(header::COOKIE, &owner_sess)
                .header("X-CSRF-Token", &owner_csrf)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(serde_json::to_vec(&good_transfer_body).unwrap()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(good_transfer_res.status(), StatusCode::OK);

    // Verify roles after transfer: Alice is owner, previous owner is admin
    let alice_membership = store::get_user_membership(&pool, ws_id, alice.id)
        .await
        .unwrap();
    assert_eq!(alice_membership.as_deref(), Some("owner"));

    let prev_owner_membership = store::get_user_membership(&pool, ws_id, owner.id)
        .await
        .unwrap();
    assert_eq!(prev_owner_membership.as_deref(), Some("admin"));

    // 9. Alice (new owner) removes Bob
    let remove_bob_res = app
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!("/api/v1/workspaces/{ws_id}/members/{}", bob.id))
                .header(header::COOKIE, &alice_sess)
                .header("X-CSRF-Token", &alice_csrf)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(remove_bob_res.status(), StatusCode::NO_CONTENT);

    let bob_membership = store::get_user_membership(&pool, ws_id, bob.id)
        .await
        .unwrap();
    assert_eq!(bob_membership, None);
}

#[sqlx::test(migrations = false)]
async fn concurrent_last_owner_removal_race_prevention(pool: PgPool) {
    database::migrate(&pool).await.unwrap();

    let hasher = Hasher::new(4096, 1, 1).unwrap();
    let hash = hasher.hash(b"password-123").unwrap();

    // 1. Bootstrap workspace with owner 1
    let owner1 = store::bootstrap(
        &pool,
        "owner_one",
        &hash,
        "Owner One",
        "Dual Owner Workspace",
        "dual-owner",
    )
    .await
    .unwrap();

    let workspaces = store::list_user_workspaces(&pool, owner1.id).await.unwrap();
    let ws_id = workspaces[0].id;

    // 2. Create owner 2 and insert as second owner
    let owner2 = store::create_user(&pool, "owner_two", &hash, "Owner Two")
        .await
        .unwrap();
    sqlx::query(
        "INSERT INTO workspace_memberships (workspace_id, user_id, role)
         VALUES ($1, $2, 'owner')",
    )
    .bind(ws_id)
    .bind(owner2.id)
    .execute(&pool)
    .await
    .unwrap();

    // 3. Two concurrent attempts: Task A tries to remove Owner 1, Task B tries to remove Owner 2
    let pool_a = pool.clone();
    let task_a = tokio::spawn(async move {
        store::remove_workspace_member(&pool_a, Some(owner1.id), ws_id, owner1.id).await
    });

    let pool_b = pool.clone();
    let task_b = tokio::spawn(async move {
        store::remove_workspace_member(&pool_b, Some(owner2.id), ws_id, owner2.id).await
    });

    let (res_a, res_b) = tokio::join!(task_a, task_b);
    let res_a = res_a.unwrap();
    let res_b = res_b.unwrap();

    // Exactly one must succeed (Ok) and one must fail (Conflict)
    let ok_count = [res_a.is_ok(), res_b.is_ok()]
        .into_iter()
        .filter(|&b| b)
        .count();
    let conflict_count = [
        matches!(res_a, Err(StoreError::Conflict)),
        matches!(res_b, Err(StoreError::Conflict)),
    ]
    .into_iter()
    .filter(|&b| b)
    .count();

    assert_eq!(ok_count, 1, "exactly one removal should succeed");
    assert_eq!(
        conflict_count, 1,
        "one removal must fail with Conflict to preserve the last owner"
    );

    // Verify exactly one active owner remains in the workspace
    let remaining_members = store::list_workspace_members(&pool, ws_id).await.unwrap();
    let active_owners: Vec<_> = remaining_members
        .iter()
        .filter(|m| m.role == "owner")
        .collect();
    assert_eq!(
        active_owners.len(),
        1,
        "workspace must retain exactly 1 active owner"
    );
}
