use std::{process::ExitCode, time::Duration};
use tokio::net::TcpListener;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};
use trestle::{config::Config, database, http};

#[tokio::main]
async fn main() -> ExitCode {
    tracing_subscriber::registry()
        .with(tracing_subscriber::filter::filter_fn(|metadata| {
            metadata.target().starts_with("trestle") && *metadata.level() <= tracing::Level::INFO
        }))
        .with(tracing_subscriber::fmt::layer().json())
        .init();
    match run().await {
        Ok(()) => ExitCode::SUCCESS,
        Err(message) => {
            tracing::error!(message, "application stopped with an error");
            ExitCode::FAILURE
        }
    }
}

async fn run() -> Result<(), String> {
    let args: Vec<_> = std::env::args_os().skip(1).collect();
    let migrate = match args.as_slice() {
        [] => false,
        [arg] if arg == "migrate" => true,
        _ => return Err("usage: trestle [migrate]".into()),
    };
    let config = Config::from_env().map_err(|error| error.to_string())?;
    let pool = database::pool(&config);
    if migrate {
        let result = tokio::time::timeout(Duration::from_secs(60), database::migrate(&pool)).await;
        pool.close().await;
        return match result {
            Ok(Ok(())) => {
                tracing::info!("database migrations complete");
                Ok(())
            }
            _ => Err("database migration failed or timed out; verify connectivity, permissions and migration history".into()),
        };
    }

    // Register termination handlers before binding; failures must not be ignored.
    #[cfg(unix)]
    let mut terminate = tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
        .map_err(|_| "could not register termination handler")?;
    let listener = TcpListener::bind(config.bind_addr)
        .await
        .map_err(|_| "could not bind HTTP listener")?;
    tracing::info!(address = %config.bind_addr, "HTTP server listening");

    let (shutdown_started, shutdown_received) = tokio::sync::oneshot::channel();
    let shutdown = async move {
        #[cfg(unix)]
        tokio::select! {
            _ = tokio::signal::ctrl_c() => {},
            _ = terminate.recv() => {},
        }
        #[cfg(not(unix))]
        let _ = tokio::signal::ctrl_c().await;
        tracing::info!("shutdown requested");
        let _ = shutdown_started.send(());
    };
    let server = axum::serve(listener, http::router(pool.clone(), config.db_timeout))
        .with_graceful_shutdown(shutdown);
    let server = std::future::IntoFuture::into_future(server);
    tokio::pin!(server);
    let result = tokio::select! {
        result = &mut server => result.map_err(|_| "HTTP server failed".to_owned()),
        _ = shutdown_received => {
            match tokio::time::timeout(Duration::from_secs(10), &mut server).await {
                Ok(result) => result.map_err(|_| "HTTP server failed during shutdown".to_owned()),
                Err(_) => Err("HTTP shutdown drain timed out".into()),
            }
        }
    };
    pool.close().await;
    result
}
