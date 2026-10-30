use axum::{
    body::Body,
    http::{Request, StatusCode},
};
use http_body_util::BodyExt;
use sqlx::PgPool;
use std::time::Duration;
use tower::ServiceExt;
use trestle::{database, http};

#[sqlx::test(migrations = false)]
async fn fresh_database_migrates_and_repeat_execution_succeeds(pool: PgPool) {
    database::migrate(&pool).await.unwrap();
    database::migrate(&pool).await.unwrap();
    let response = http::router(pool, Duration::from_secs(2))
        .oneshot(
            Request::builder()
                .uri("/health/ready")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::OK);
    let body = response.into_body().collect().await.unwrap().to_bytes();
    assert_eq!(&body[..], br#"{"status":"ok"}"#);
}

#[sqlx::test(migrations = false)]
async fn concurrent_migration_commands_are_serialized_safely(pool: PgPool) {
    let (first, second) = tokio::join!(database::migrate(&pool), database::migrate(&pool));
    first.unwrap();
    second.unwrap();
    database::migrate(&pool).await.unwrap();
}

#[sqlx::test(migrations = false)]
async fn saturated_application_pool_returns_bounded_readiness_failure(pool: PgPool) {
    use sqlx::ConnectOptions;
    use trestle::config::Config;
    let url = pool.connect_options().to_url_lossy();
    let config = Config::from_pairs([
        ("DATABASE_URL", url.as_str()),
        ("TRESTLE_MAX_DB_CONNECTIONS", "1"),
        ("TRESTLE_DB_TIMEOUT_MS", "2000"),
    ])
    .unwrap();
    let application_pool = database::pool(&config);
    let held_connection = application_pool.acquire().await.unwrap();
    // Establish the connection before testing saturation. A 100ms authentication
    // budget is not reliable on loaded CI hosts. Keep the HTTP deadline shorter
    // than pool acquisition to verify that it bounds the entire readiness check.
    let app = http::router(application_pool.clone(), Duration::from_millis(100));
    let response = tokio::time::timeout(
        Duration::from_secs(2),
        app.clone().oneshot(
            Request::builder()
                .uri("/health/ready")
                .body(Body::empty())
                .unwrap(),
        ),
    )
    .await
    .expect("pool exhaustion must not hang readiness")
    .unwrap();
    assert_eq!(response.status(), StatusCode::SERVICE_UNAVAILABLE);
    drop(held_connection);
    let response = app
        .oneshot(
            Request::builder()
                .uri("/health/ready")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::OK);
    application_pool.close().await;
}
