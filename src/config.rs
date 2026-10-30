use sqlx::postgres::PgConnectOptions;
use std::{collections::BTreeMap, fmt, net::SocketAddr, time::Duration};

pub struct Config {
    pub bind_addr: SocketAddr,
    pub max_db_connections: u32,
    pub db_timeout: Duration,
    pub(crate) database: PgConnectOptions,
}

impl fmt::Debug for Config {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("Config")
            .field("bind_addr", &self.bind_addr)
            .field("max_db_connections", &self.max_db_connections)
            .field("db_timeout", &self.db_timeout)
            .field("database", &"[REDACTED]")
            .finish()
    }
}

#[derive(Debug, PartialEq, Eq)]
pub struct ConfigError(&'static str);

impl fmt::Display for ConfigError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(self.0)
    }
}

impl std::error::Error for ConfigError {}

impl Config {
    pub fn from_env() -> Result<Self, ConfigError> {
        let mut values = Vec::new();
        for (key, unicode_error) in [
            ("DATABASE_URL", "DATABASE_URL must be valid Unicode"),
            (
                "TRESTLE_BIND_ADDR",
                "TRESTLE_BIND_ADDR must be valid Unicode",
            ),
            (
                "TRESTLE_MAX_DB_CONNECTIONS",
                "TRESTLE_MAX_DB_CONNECTIONS must be valid Unicode",
            ),
            (
                "TRESTLE_DB_TIMEOUT_MS",
                "TRESTLE_DB_TIMEOUT_MS must be valid Unicode",
            ),
        ] {
            match std::env::var(key) {
                Ok(value) => values.push((key, value)),
                Err(std::env::VarError::NotPresent) => {}
                Err(std::env::VarError::NotUnicode(_)) => {
                    return Err(ConfigError(unicode_error));
                }
            }
        }
        Self::from_pairs(values)
    }

    pub fn from_pairs<K: Into<String>, V: Into<String>>(
        pairs: impl IntoIterator<Item = (K, V)>,
    ) -> Result<Self, ConfigError> {
        let values: BTreeMap<String, String> = pairs
            .into_iter()
            .map(|(key, value)| (key.into(), value.into()))
            .collect();
        let url = values
            .get("DATABASE_URL")
            .ok_or(ConfigError("DATABASE_URL is required"))?;
        if !(url.starts_with("postgres://") || url.starts_with("postgresql://")) {
            return Err(ConfigError("DATABASE_URL must be a valid PostgreSQL URL"));
        }
        let database = url
            .parse()
            .map_err(|_| ConfigError("DATABASE_URL must be a valid PostgreSQL URL"))?;
        let bind_addr: SocketAddr = values
            .get("TRESTLE_BIND_ADDR")
            .map(String::as_str)
            .unwrap_or("127.0.0.1:3000")
            .parse()
            .map_err(|_| {
                ConfigError("TRESTLE_BIND_ADDR must be an IP socket address with a nonzero port")
            })?;
        if bind_addr.port() == 0 {
            return Err(ConfigError(
                "TRESTLE_BIND_ADDR must be an IP socket address with a nonzero port",
            ));
        }
        let max_db_connections = bounded_number(
            &values,
            "TRESTLE_MAX_DB_CONNECTIONS",
            5,
            1,
            32,
            "TRESTLE_MAX_DB_CONNECTIONS must be an integer from 1 to 32",
        )?;
        let timeout_ms = bounded_number(
            &values,
            "TRESTLE_DB_TIMEOUT_MS",
            2000,
            100,
            10000,
            "TRESTLE_DB_TIMEOUT_MS must be an integer from 100 to 10000",
        )?;
        Ok(Self {
            bind_addr,
            max_db_connections,
            db_timeout: Duration::from_millis(u64::from(timeout_ms)),
            database,
        })
    }
}

fn bounded_number(
    values: &BTreeMap<String, String>,
    key: &str,
    default: u32,
    min: u32,
    max: u32,
    message: &'static str,
) -> Result<u32, ConfigError> {
    let Some(value) = values.get(key) else {
        return Ok(default);
    };
    let value = value.parse().map_err(|_| ConfigError(message))?;
    if !(min..=max).contains(&value) {
        return Err(ConfigError(message));
    }
    Ok(value)
}
