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
    match args.as_slice() {
        [] => run_server().await,
        [arg] if arg == "migrate" => run_migrate().await,
        [arg] if arg == "bootstrap" => run_bootstrap().await,
        [arg] if arg == "adduser" => run_adduser().await,
        [arg] if arg == "resetpassword" => run_resetpassword().await,
        [arg] if arg == "disableuser" => run_disableuser().await,
        [arg] if arg == "enableuser" => run_enableuser().await,
        _ => Err("usage: trestle [migrate | bootstrap | adduser | resetpassword | disableuser | enableuser]".into()),
    }
}

async fn run_server() -> Result<(), String> {
    let config = Config::from_env().map_err(|error| error.to_string())?;
    let pool = database::pool(&config);

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
    let timeout = config.db_timeout;
    let router = http::router(pool.clone(), timeout, config);
    let server = axum::serve(
        listener,
        router.into_make_service_with_connect_info::<std::net::SocketAddr>(),
    )
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

async fn run_migrate() -> Result<(), String> {
    let config = Config::from_env().map_err(|error| error.to_string())?;
    let pool = database::pool(&config);
    let result = tokio::time::timeout(Duration::from_secs(60), database::migrate(&pool)).await;
    pool.close().await;
    match result {
        Ok(Ok(())) => {
            tracing::info!("database migrations complete");
            Ok(())
        }
        _ => Err("database migration failed or timed out; verify connectivity, permissions and migration history".into()),
    }
}

async fn run_bootstrap() -> Result<(), String> {
    let config = Config::from_env().map_err(|error| error.to_string())?;
    let pool = database::pool(&config);

    // Check if already bootstrapped.
    let exists = trestle::identity::store::any_user_exists(&pool)
        .await
        .map_err(|e| format!("database error: {e}"))?;
    if exists {
        pool.close().await;
        eprintln!("This instance is already initialized. Skipping bootstrap.");
        return Ok(());
    }

    let username_env = std::env::var("TRESTLE_BOOTSTRAP_USERNAME").ok();
    let password_env = std::env::var("TRESTLE_BOOTSTRAP_PASSWORD").ok();
    let ws_name_env = std::env::var("TRESTLE_BOOTSTRAP_WORKSPACE_NAME").ok();
    let ws_slug_env = std::env::var("TRESTLE_BOOTSTRAP_WORKSPACE_SLUG").ok();

    let (username_str, password, workspace_name, workspace_slug) = match (
        username_env,
        password_env,
    ) {
        (Some(u), Some(p)) => {
            let wn = ws_name_env.unwrap_or_else(|| "Main Workspace".into());
            let ws = ws_slug_env.unwrap_or_else(|| "main".into());
            (u, p, wn, ws)
        }
        _ => {
            // Prompt for operator credentials (hidden at terminal).
            eprintln!("Trestle bootstrap: create the initial operator account.");
            eprintln!("This can only be run once. The account is not a workspace member until");
            eprintln!("workspace membership is granted.");
            eprintln!();

            let username = prompt_plain("Username (3-64 characters, letters/digits/._-): ")?;
            let password = prompt_hidden("Password (at least 8 characters): ")?;
            let password2 = prompt_hidden("Confirm password: ")?;
            if password != password2 {
                pool.close().await;
                return Err("Passwords do not match.".into());
            }

            let workspace_name = prompt_plain("Initial workspace name: ")?;
            let workspace_slug = prompt_plain(
                "Workspace slug (URL identifier, 2-48 chars, lowercase alphanumeric + hyphens): ",
            )?;
            (username, password, workspace_name, workspace_slug)
        }
    };

    let username = trestle::identity::Username::parse(&username_str)
        .map_err(|e| format!("Invalid username: {e}"))?;

    if password.len() < 8 {
        pool.close().await;
        return Err("Password must be at least 8 characters.".into());
    }

    if workspace_name.trim().is_empty() || workspace_name.len() > 120 {
        pool.close().await;
        return Err("Workspace name must be 1-120 characters.".into());
    }

    let workspace_slug = workspace_slug.trim().to_ascii_lowercase();
    if !validate_workspace_slug(&workspace_slug) {
        pool.close().await;
        return Err(
            "Invalid workspace slug. Use 2-48 lowercase alphanumeric characters and hyphens."
                .into(),
        );
    }

    // Hash password (uses Argon2id with configured parameters).
    let hasher = trestle::identity::Hasher::new(
        config.argon2_m_cost,
        config.argon2_t_cost,
        config.argon2_parallelism,
    )
    .map_err(|e| format!("argon2 configuration error: {e}"))?;

    let password_hash = tokio::task::spawn_blocking(move || hasher.hash(password.as_bytes()))
        .await
        .map_err(|_| "password hashing task failed")?
        .map_err(|e| format!("password error: {e}"))?;

    let display_name = username.as_str().to_owned();

    let user = trestle::identity::store::bootstrap(
        &pool,
        username.as_str(),
        &password_hash,
        &display_name,
        workspace_name.trim(),
        &workspace_slug,
    )
    .await
    .map_err(|e| match e {
        trestle::identity::store::StoreError::Conflict => {
            "Another bootstrap completed concurrently. This instance is already initialized.".into()
        }
        e => format!("database error: {e}"),
    })?;

    pool.close().await;

    eprintln!();
    eprintln!("Bootstrap complete.");
    eprintln!("  Username:  {}", user.username);
    eprintln!("  Workspace: {workspace_name} (slug: {workspace_slug})");
    eprintln!();
    eprintln!("Sign in at the application URL with your username and password.");
    Ok(())
}

async fn run_adduser() -> Result<(), String> {
    let config = Config::from_env().map_err(|error| error.to_string())?;
    let pool = database::pool(&config);

    let username_env = std::env::var("TRESTLE_ADDUSER_USERNAME").ok();
    let password_env = std::env::var("TRESTLE_ADDUSER_PASSWORD").ok();
    let display_name_env = std::env::var("TRESTLE_ADDUSER_DISPLAY_NAME").ok();

    let (username_str, password, display_name) = match (username_env, password_env) {
        (Some(u), Some(p)) => {
            let dn = display_name_env.unwrap_or_else(|| u.clone());
            (u, p, dn)
        }
        _ => {
            eprintln!("Trestle operator user provisioning:");
            eprintln!();
            let username = prompt_plain("Username (3-64 characters, letters/digits/._-): ")?;
            let display_name = prompt_plain("Display name: ")?;
            let password = prompt_hidden("Password (at least 8 characters): ")?;
            let password2 = prompt_hidden("Confirm password: ")?;
            if password != password2 {
                pool.close().await;
                return Err("Passwords do not match.".into());
            }
            (username, password, display_name)
        }
    };

    let username = trestle::identity::Username::parse(&username_str)
        .map_err(|e| format!("Invalid username: {e}"))?;

    if password.len() < 8 {
        pool.close().await;
        return Err("Password must be at least 8 characters.".into());
    }

    if display_name.trim().is_empty() || display_name.len() > 120 {
        pool.close().await;
        return Err("Display name must be 1-120 characters.".into());
    }

    let hasher = trestle::identity::Hasher::new(
        config.argon2_m_cost,
        config.argon2_t_cost,
        config.argon2_parallelism,
    )
    .map_err(|e| format!("argon2 configuration error: {e}"))?;

    let password_hash = tokio::task::spawn_blocking(move || hasher.hash(password.as_bytes()))
        .await
        .map_err(|_| "password hashing task failed")?
        .map_err(|e| format!("password error: {e}"))?;

    let user = trestle::identity::store::create_user(
        &pool,
        username.as_str(),
        &password_hash,
        display_name.trim(),
    )
    .await
    .map_err(|e| match e {
        trestle::identity::store::StoreError::Conflict => "User already exists.".into(),
        e => format!("database error: {e}"),
    })?;

    pool.close().await;

    eprintln!();
    eprintln!("User provisioned successfully.");
    eprintln!("  Username:     {}", user.username);
    eprintln!("  Display name: {}", user.display_name);
    eprintln!("(Note: Workspace membership must be granted separately.)");
    Ok(())
}

async fn run_resetpassword() -> Result<(), String> {
    let config = Config::from_env().map_err(|error| error.to_string())?;
    let pool = database::pool(&config);

    let username_env = std::env::var("TRESTLE_RESET_USERNAME").ok();
    let password_env = std::env::var("TRESTLE_RESET_PASSWORD").ok();

    let (username_str, password) = match (username_env, password_env) {
        (Some(u), Some(p)) => (u, p),
        _ => {
            eprintln!("Trestle operator password reset & session revocation:");
            eprintln!();
            let username = prompt_plain("Username to reset: ")?;
            let password = prompt_hidden("New password (at least 8 characters): ")?;
            let password2 = prompt_hidden("Confirm new password: ")?;
            if password != password2 {
                pool.close().await;
                return Err("Passwords do not match.".into());
            }
            (username, password)
        }
    };

    let username = trestle::identity::Username::parse(&username_str)
        .map_err(|e| format!("Invalid username: {e}"))?;

    if password.len() < 8 {
        pool.close().await;
        return Err("Password must be at least 8 characters.".into());
    }

    let hasher = trestle::identity::Hasher::new(
        config.argon2_m_cost,
        config.argon2_t_cost,
        config.argon2_parallelism,
    )
    .map_err(|e| format!("argon2 configuration error: {e}"))?;

    let password_hash = tokio::task::spawn_blocking(move || hasher.hash(password.as_bytes()))
        .await
        .map_err(|_| "password hashing task failed")?
        .map_err(|e| format!("password error: {e}"))?;

    let user =
        trestle::identity::store::reset_user_password(&pool, username.as_str(), &password_hash)
            .await
            .map_err(|e| match e {
                trestle::identity::store::StoreError::NotFound => "User not found.".into(),
                e => format!("database error: {e}"),
            })?;

    pool.close().await;

    eprintln!();
    eprintln!("Password reset successfully for user '{}'.", user.username);
    eprintln!("All existing active sessions for this user have been revoked.");
    Ok(())
}

async fn run_disableuser() -> Result<(), String> {
    let config = Config::from_env().map_err(|error| error.to_string())?;
    let pool = database::pool(&config);

    let username_env = std::env::var("TRESTLE_DISABLE_USERNAME").ok();
    let username_str = match username_env {
        Some(u) => u,
        None => prompt_plain("Username to disable: ")?,
    };

    let username = trestle::identity::Username::parse(&username_str)
        .map_err(|e| format!("Invalid username: {e}"))?;

    let user = trestle::identity::store::set_user_disabled(&pool, username.as_str(), true)
        .await
        .map_err(|e| match e {
            trestle::identity::store::StoreError::NotFound => "User not found.".into(),
            trestle::identity::store::StoreError::Conflict => {
                "Cannot disable user: this user is the sole active owner of one or more workspaces."
                    .into()
            }
            e => format!("database error: {e}"),
        })?;

    pool.close().await;

    eprintln!();
    eprintln!("User '{}' has been disabled.", user.username);
    eprintln!("All existing active sessions have been revoked.");
    Ok(())
}

async fn run_enableuser() -> Result<(), String> {
    let config = Config::from_env().map_err(|error| error.to_string())?;
    let pool = database::pool(&config);

    let username_env = std::env::var("TRESTLE_ENABLE_USERNAME").ok();
    let username_str = match username_env {
        Some(u) => u,
        None => prompt_plain("Username to enable: ")?,
    };

    let username = trestle::identity::Username::parse(&username_str)
        .map_err(|e| format!("Invalid username: {e}"))?;

    let user = trestle::identity::store::set_user_disabled(&pool, username.as_str(), false)
        .await
        .map_err(|e| match e {
            trestle::identity::store::StoreError::NotFound => "User not found.".into(),
            e => format!("database error: {e}"),
        })?;

    pool.close().await;

    eprintln!();
    eprintln!("User '{}' has been enabled.", user.username);
    Ok(())
}

fn prompt_plain(prompt: &str) -> Result<String, String> {
    eprint!("{prompt}");
    let mut line = String::new();
    std::io::stdin()
        .read_line(&mut line)
        .map_err(|_| "failed to read input")?;
    let trimmed = line.trim_end_matches(['\n', '\r']).to_owned();
    if trimmed.is_empty() {
        return Err("Input cannot be empty.".into());
    }
    Ok(trimmed)
}

fn prompt_hidden(prompt: &str) -> Result<String, String> {
    let password = rpassword::prompt_password(prompt).map_err(|_| "failed to read password")?;
    let trimmed = password.trim_end_matches(['\n', '\r']).to_owned();
    if trimmed.is_empty() {
        return Err("Password cannot be empty.".into());
    }
    Ok(trimmed)
}

fn validate_workspace_slug(slug: &str) -> bool {
    if slug.len() < 2 || slug.len() > 48 {
        return false;
    }
    if slug.starts_with('-') || slug.ends_with('-') {
        return false;
    }
    slug.chars()
        .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == '-')
}
