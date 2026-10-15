use axum::{
    body::Body,
    http::{Request, StatusCode},
};
use http_body_util::BodyExt;
use tower::ServiceExt;
use trestle::{config::Config, database, http};

fn app() -> axum::Router {
    let config = Config::from_pairs([
        ("DATABASE_URL", "postgres://user:secret@127.0.0.1:1/trestle"),
        ("TRESTLE_DB_TIMEOUT_MS", "100"),
    ])
    .unwrap();
    http::router(database::pool(&config), config.db_timeout)
}

#[tokio::test]
async fn liveness_succeeds_without_a_database() {
    let response = app()
        .oneshot(
            Request::builder()
                .uri("/health/live")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::OK);
    assert_eq!(response.headers()["content-type"], "application/json");
    assert_eq!(response.headers()["cache-control"], "no-store");
    assert_eq!(response.headers()["x-content-type-options"], "nosniff");
    let body = response.into_body().collect().await.unwrap().to_bytes();
    assert_eq!(&body[..], br#"{"status":"ok"}"#);
}

#[tokio::test]
async fn unavailable_database_returns_bounded_secret_safe_readiness_failure() {
    let response = tokio::time::timeout(
        std::time::Duration::from_secs(2),
        app().oneshot(
            Request::builder()
                .uri("/health/ready")
                .body(Body::empty())
                .unwrap(),
        ),
    )
    .await
    .expect("readiness must be bounded")
    .unwrap();
    assert_eq!(response.status(), StatusCode::SERVICE_UNAVAILABLE);
    assert_eq!(response.headers()["cache-control"], "no-store");
    let body = response.into_body().collect().await.unwrap().to_bytes();
    assert_eq!(
        &body[..],
        br#"{"error":{"code":"unavailable","message":"Service unavailable"}}"#
    );
}

#[tokio::test]
async fn unknown_routes_and_unsupported_methods_return_consistent_errors() {
    for (method, path, status, expected) in [
        (
            "GET",
            "/unknown?secret=never-log-this",
            StatusCode::NOT_FOUND,
            r#"{"error":{"code":"not_found","message":"Not found"}}"#,
        ),
        (
            "POST",
            "/health/live",
            StatusCode::METHOD_NOT_ALLOWED,
            r#"{"error":{"code":"method_not_allowed","message":"Method not allowed"}}"#,
        ),
    ] {
        let response = app()
            .oneshot(
                Request::builder()
                    .method(method)
                    .uri(path)
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), status);
        assert_eq!(response.headers()["content-type"], "application/json");
        assert_eq!(response.headers()["cache-control"], "no-store");
        let body = response.into_body().collect().await.unwrap().to_bytes();
        assert_eq!(&body[..], expected.as_bytes());
    }
}

#[tokio::test]
async fn request_diagnostics_include_status_but_not_raw_url_or_headers() {
    use std::{
        io::Write,
        sync::{Arc, Mutex},
    };
    #[derive(Clone)]
    struct Writer(Arc<Mutex<Vec<u8>>>);
    impl Write for Writer {
        fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
            self.0.lock().unwrap().extend_from_slice(bytes);
            Ok(bytes.len())
        }
        fn flush(&mut self) -> std::io::Result<()> {
            Ok(())
        }
    }
    let writer = Writer(Arc::default());
    let output = writer.0.clone();
    let subscriber = tracing_subscriber::fmt()
        .json()
        .with_writer(move || writer.clone())
        .finish();
    // Match the executable's process-wide subscriber. Scoped subscribers race
    // tracing callsite interest caching when other HTTP tests run concurrently.
    tracing::subscriber::set_global_default(subscriber).unwrap();
    app()
        .oneshot(
            Request::builder()
                .uri("/private-path?token=private-query")
                .header("authorization", "Bearer private-header")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    let logs = String::from_utf8(output.lock().unwrap().clone()).unwrap();
    assert!(logs.contains("404"));
    assert!(logs.contains("<unmatched>"));
    for secret in ["private-path", "private-query", "private-header"] {
        assert!(!logs.contains(secret));
    }
}

#[tokio::test]
async fn internal_errors_have_a_fixed_public_response() {
    use axum::response::IntoResponse;
    let response = http::HttpError::Internal.into_response();
    assert_eq!(response.status(), StatusCode::INTERNAL_SERVER_ERROR);
    let body = response.into_body().collect().await.unwrap().to_bytes();
    assert_eq!(
        &body[..],
        br#"{"error":{"code":"internal_error","message":"Internal server error"}}"#
    );
}
