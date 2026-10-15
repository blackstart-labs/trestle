use axum::{Router, routing::get};

include!(concat!(env!("OUT_DIR"), "/assets.rs"));

pub fn router<S: Clone + Send + Sync + 'static>() -> Router<S> {
    let mut router = Router::new();
    for &(path, mime, bytes) in ASSETS {
        router = router.route(
            path,
            get(move || async move { ([("content-type", mime)], bytes) }),
        );
    }
    router
}
