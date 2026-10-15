use crate::config::Config;
use sqlx::{ConnectOptions, PgPool, postgres::PgPoolOptions};

/// Construct without connecting, so database outages do not prevent liveness.
pub fn pool(config: &Config) -> PgPool {
    let options = config
        .database
        .clone()
        .application_name("trestle")
        .options([("statement_timeout", "5000"), ("lock_timeout", "5000")])
        .disable_statement_logging();
    PgPoolOptions::new()
        .min_connections(0)
        .max_connections(config.max_db_connections)
        .acquire_timeout(config.db_timeout)
        .connect_lazy_with(options)
}

/// Migrations are explicit; serving HTTP never invokes this function.
pub async fn migrate(pool: &PgPool) -> Result<(), sqlx::migrate::MigrateError> {
    sqlx::migrate!("./migrations").run(pool).await
}
