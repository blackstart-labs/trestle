use axum::{
    body::Body,
    http::{Request, StatusCode},
};
use http_body_util::BodyExt;
use tower::ServiceExt;
use trestle::{config::Config, database, http};

fn app() -> axum::Router {
    let config =
        Config::from_pairs([("DATABASE_URL", "postgres://user:secret@127.0.0.1:1/trestle")])
            .unwrap();
    http::router(database::pool(&config), config.db_timeout)
}

#[tokio::test]
async fn referenced_assets_have_correct_types_and_security_headers() {
    let response = app()
        .oneshot(Request::builder().uri("/").body(Body::empty()).unwrap())
        .await
        .unwrap();
    let bytes = response.into_body().collect().await.unwrap().to_bytes();
    let html = std::str::from_utf8(&bytes).unwrap();
    let paths: Vec<_> = html
        .split('"')
        .filter(|value| value.starts_with("/assets/"))
        .collect();
    assert!(paths.iter().any(|path| path.ends_with(".js")));
    assert!(paths.iter().any(|path| path.ends_with(".css")));
    for path in paths {
        let response = app()
            .oneshot(Request::builder().uri(path).body(Body::empty()).unwrap())
            .await
            .unwrap();
        assert_eq!(response.status(), StatusCode::OK);
        assert_eq!(
            response.headers()["content-type"],
            if path.ends_with(".js") {
                "text/javascript; charset=utf-8"
            } else {
                "text/css; charset=utf-8"
            }
        );
        assert_eq!(response.headers()["cache-control"], "no-store");
        assert_eq!(response.headers()["x-content-type-options"], "nosniff");
        assert_eq!(
            response.headers()["content-security-policy"],
            "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'none'"
        );
        assert!(
            !response
                .into_body()
                .collect()
                .await
                .unwrap()
                .to_bytes()
                .is_empty()
        );
        let head = app()
            .oneshot(
                Request::builder()
                    .method("HEAD")
                    .uri(path)
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(head.status(), StatusCode::OK);
        assert!(
            head.into_body()
                .collect()
                .await
                .unwrap()
                .to_bytes()
                .is_empty()
        );
        let post = app()
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri(path)
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(post.status(), StatusCode::METHOD_NOT_ALLOWED);
    }
}

#[tokio::test]
async fn unknown_paths_never_fall_back_to_html() {
    for path in [
        "/assets/missing.js",
        "/api/v1/workspaces",
        "/src/App.svelte",
        "/assets/../index.html",
        "/assets/%2e%2e/index.html",
        "/assets/%2fetc/passwd",
        "/index.html",
    ] {
        let response = app()
            .oneshot(Request::builder().uri(path).body(Body::empty()).unwrap())
            .await
            .unwrap();
        assert_eq!(response.status(), StatusCode::NOT_FOUND, "{path}");
        assert_eq!(response.headers()["content-type"], "application/json");
        assert_eq!(
            &response.into_body().collect().await.unwrap().to_bytes()[..],
            br#"{"error":{"code":"not_found","message":"Not found"}}"#
        );
    }
}

#[tokio::test]
async fn root_supports_head_but_not_mutation() {
    let head = app()
        .oneshot(
            Request::builder()
                .method("HEAD")
                .uri("/")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(head.status(), StatusCode::OK);
    assert!(
        head.into_body()
            .collect()
            .await
            .unwrap()
            .to_bytes()
            .is_empty()
    );
    let post = app()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(post.status(), StatusCode::METHOD_NOT_ALLOWED);
}

#[tokio::test]
async fn root_delivers_the_built_frontend_without_a_database() {
    let response = app()
        .oneshot(Request::builder().uri("/").body(Body::empty()).unwrap())
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::OK);
    assert_eq!(
        response.headers()["content-type"],
        "text/html; charset=utf-8"
    );
    assert_eq!(response.headers()["cache-control"], "no-store");
    assert_eq!(response.headers()["x-content-type-options"], "nosniff");
    let body = response.into_body().collect().await.unwrap().to_bytes();
    let html = std::str::from_utf8(&body).unwrap();
    assert!(html.contains("<title>Trestle — foundation</title>"));
    assert!(html.contains("/assets/"));
}
