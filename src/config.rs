use sqlx::postgres::PgConnectOptions;
use std::{collections::BTreeMap, fmt, net::SocketAddr, time::Duration};

/// Argon2id parameter defaults (benchmarked on reference host 2026-11-01).
/// m=65536 KiB (64 MiB), t=2, p=1 → ~122ms per hash on this host.
pub const DEFAULT_ARGON2_M_COST: u32 = 65_536;
pub const DEFAULT_ARGON2_T_COST: u32 = 2;
pub const DEFAULT_ARGON2_PARALLELISM: u32 = 1;

#[derive(Clone)]
pub struct Config {
    pub bind_addr: SocketAddr,
    pub max_db_connections: u32,
    pub db_timeout: Duration,
    pub(crate) database: PgConnectOptions,
    /// Enable development mode: relaxes cookie Secure requirement for loopback HTTP.
    /// Must only be used with a loopback bind address.
    pub dev_mode: bool,
    /// Argon2id memory cost in KiB.
    pub argon2_m_cost: u32,
    /// Argon2id time cost (number of passes).
    pub argon2_t_cost: u32,
    /// Argon2id parallelism factor.
    pub argon2_parallelism: u32,
}

impl fmt::Debug for Config {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("Config")
            .field("bind_addr", &self.bind_addr)
            .field("max_db_connections", &self.max_db_connections)
            .field("db_timeout", &self.db_timeout)
            .field("database", &"[REDACTED]")
            .field("dev_mode", &self.dev_mode)
            .field("argon2_m_cost", &self.argon2_m_cost)
            .field("argon2_t_cost", &self.argon2_t_cost)
            .field("argon2_parallelism", &self.argon2_parallelism)
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
            ("TRESTLE_DEV_MODE", "TRESTLE_DEV_MODE must be valid Unicode"),
            (
                "TRESTLE_ARGON2_M_COST",
                "TRESTLE_ARGON2_M_COST must be valid Unicode",
            ),
            (
                "TRESTLE_ARGON2_T_COST",
                "TRESTLE_ARGON2_T_COST must be valid Unicode",
            ),
            (
                "TRESTLE_ARGON2_PARALLELISM",
                "TRESTLE_ARGON2_PARALLELISM must be valid Unicode",
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

        // Development mode: only valid when bind address is loopback.
        let dev_mode = values
            .get("TRESTLE_DEV_MODE")
            .map(String::as_str)
            .unwrap_or("false")
            == "true";
        if dev_mode && !bind_addr.ip().is_loopback() {
            return Err(ConfigError(
                "TRESTLE_DEV_MODE=true is only permitted with a loopback bind address",
            ));
        }

        let argon2_m_cost = bounded_number(
            &values,
            "TRESTLE_ARGON2_M_COST",
            DEFAULT_ARGON2_M_COST,
            // OWASP minimum: 19456. Allow lower only for test builds.
            19_456,
            524_288, // 512 MiB maximum
            "TRESTLE_ARGON2_M_COST must be an integer from 19456 to 524288",
        )?;
        let argon2_t_cost = bounded_number(
            &values,
            "TRESTLE_ARGON2_T_COST",
            DEFAULT_ARGON2_T_COST,
            1,
            16,
            "TRESTLE_ARGON2_T_COST must be an integer from 1 to 16",
        )?;
        let argon2_parallelism = bounded_number(
            &values,
            "TRESTLE_ARGON2_PARALLELISM",
            DEFAULT_ARGON2_PARALLELISM,
            1,
            8,
            "TRESTLE_ARGON2_PARALLELISM must be an integer from 1 to 8",
        )?;

        Ok(Self {
            bind_addr,
            max_db_connections,
            db_timeout: Duration::from_millis(u64::from(timeout_ms)),
            database,
            dev_mode,
            argon2_m_cost,
            argon2_t_cost,
            argon2_parallelism,
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
