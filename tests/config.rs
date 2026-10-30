use trestle::config::Config;

#[test]
fn configuration_requires_a_database_url_without_echoing_input() {
    let error = Config::from_pairs([] as [(&str, &str); 0]).unwrap_err();
    assert_eq!(error.to_string(), "DATABASE_URL is required");
}

#[test]
fn valid_configuration_has_safe_defaults_and_redacts_database_options() {
    let config = Config::from_pairs([(
        "DATABASE_URL",
        "postgres://example:private-password@localhost/trestle",
    )])
    .unwrap();
    assert_eq!(config.bind_addr.to_string(), "127.0.0.1:3000");
    assert_eq!(config.max_db_connections, 5);
    assert_eq!(config.db_timeout.as_millis(), 2000);
    assert!(!format!("{config:?}").contains("private-password"));
}

#[test]
fn configuration_rejects_invalid_urls_and_out_of_range_settings_without_values() {
    for (key, value, expected) in [
        (
            "DATABASE_URL",
            "mysql://user:secret@localhost/db",
            "DATABASE_URL must be a valid PostgreSQL URL",
        ),
        (
            "DATABASE_URL",
            "",
            "DATABASE_URL must be a valid PostgreSQL URL",
        ),
        (
            "DATABASE_URL",
            "postgres://user:secret@localhost/db?port=invalid",
            "DATABASE_URL must be a valid PostgreSQL URL",
        ),
        (
            "TRESTLE_BIND_ADDR",
            "secret",
            "TRESTLE_BIND_ADDR must be an IP socket address with a nonzero port",
        ),
        (
            "TRESTLE_BIND_ADDR",
            "127.0.0.1:0",
            "TRESTLE_BIND_ADDR must be an IP socket address with a nonzero port",
        ),
        (
            "TRESTLE_MAX_DB_CONNECTIONS",
            "0",
            "TRESTLE_MAX_DB_CONNECTIONS must be an integer from 1 to 32",
        ),
        (
            "TRESTLE_MAX_DB_CONNECTIONS",
            "33",
            "TRESTLE_MAX_DB_CONNECTIONS must be an integer from 1 to 32",
        ),
        (
            "TRESTLE_MAX_DB_CONNECTIONS",
            "secret",
            "TRESTLE_MAX_DB_CONNECTIONS must be an integer from 1 to 32",
        ),
        (
            "TRESTLE_DB_TIMEOUT_MS",
            "99",
            "TRESTLE_DB_TIMEOUT_MS must be an integer from 100 to 10000",
        ),
        (
            "TRESTLE_DB_TIMEOUT_MS",
            "10001",
            "TRESTLE_DB_TIMEOUT_MS must be an integer from 100 to 10000",
        ),
        (
            "TRESTLE_DB_TIMEOUT_MS",
            "",
            "TRESTLE_DB_TIMEOUT_MS must be an integer from 100 to 10000",
        ),
    ] {
        let error = Config::from_pairs([
            ("DATABASE_URL", "postgres://user:secret@localhost/db"),
            (key, value),
        ])
        .unwrap_err();
        assert_eq!(error.to_string(), expected);
        assert!(!format!("{error:?}").contains("secret"));
    }
}

#[test]
fn configuration_accepts_explicit_ipv6_and_boundary_values() {
    for (connections, timeout) in [("1", "100"), ("32", "10000")] {
        let config = Config::from_pairs([
            ("DATABASE_URL", "postgresql://user:secret@localhost/db"),
            ("TRESTLE_BIND_ADDR", "[::1]:4000"),
            ("TRESTLE_MAX_DB_CONNECTIONS", connections),
            ("TRESTLE_DB_TIMEOUT_MS", timeout),
        ])
        .unwrap();
        assert_eq!(config.bind_addr.to_string(), "[::1]:4000");
        assert_eq!(config.max_db_connections.to_string(), connections);
        assert_eq!(config.db_timeout.as_millis().to_string(), timeout);
    }
}

#[test]
fn executable_rejects_bad_environment_without_logging_secret_values() {
    let output = std::process::Command::new(env!("CARGO_BIN_EXE_trestle"))
        .env_clear()
        .env(
            "DATABASE_URL",
            "postgres://user:private-password@localhost/db",
        )
        .env("TRESTLE_MAX_DB_CONNECTIONS", "private-password")
        .output()
        .unwrap();
    assert!(!output.status.success());
    let diagnostics = format!(
        "{}{}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(diagnostics.contains("TRESTLE_MAX_DB_CONNECTIONS"));
    assert!(!diagnostics.contains("private-password"));
}

#[cfg(unix)]
#[test]
fn non_unicode_configuration_names_the_setting_without_exposing_bytes() {
    use std::{ffi::OsStr, os::unix::ffi::OsStrExt};
    let output = std::process::Command::new(env!("CARGO_BIN_EXE_trestle"))
        .env_clear()
        .env("DATABASE_URL", OsStr::from_bytes(b"private-password\xff"))
        .output()
        .unwrap();
    assert!(!output.status.success());
    let diagnostics = format!(
        "{}{}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(diagnostics.contains("DATABASE_URL must be valid Unicode"));
    assert!(!diagnostics.contains("private-password"));
}

#[test]
fn unavailable_database_makes_migration_command_fail_without_credentials() {
    let output = std::process::Command::new(env!("CARGO_BIN_EXE_trestle"))
        .arg("migrate")
        .env_clear()
        .env(
            "DATABASE_URL",
            "postgres://user:private-password@127.0.0.1:1/trestle",
        )
        .env("TRESTLE_DB_TIMEOUT_MS", "100")
        .output()
        .unwrap();
    assert!(!output.status.success());
    let diagnostics = format!(
        "{}{}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(diagnostics.contains("database migration failed or timed out"));
    assert!(!diagnostics.contains("private-password"));
    assert!(!diagnostics.contains("postgres://"));
}
