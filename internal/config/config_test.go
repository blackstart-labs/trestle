package config

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestConfigurationRequiresADatabaseURLWithoutEchoingInput(t *testing.T) {
	_, err := FromPairs(map[string]string{})
	if err == nil || err.Error() != "DATABASE_URL is required" {
		t.Fatalf("got %v", err)
	}
}

func TestValidConfigurationHasSafeDefaultsAndRedactsDatabaseOptions(t *testing.T) {
	cfg, err := FromPairs(map[string]string{
		"DATABASE_URL": "postgres://example:private-password@localhost/trestle",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.BindAddr.String(); got != "127.0.0.1:3000" {
		t.Errorf("bind addr %q", got)
	}
	if cfg.MaxDBConnections != 5 {
		t.Errorf("max connections %d", cfg.MaxDBConnections)
	}
	if cfg.DBTimeout.Milliseconds() != 2000 {
		t.Errorf("timeout %v", cfg.DBTimeout)
	}
	for _, format := range []string{"%v", "%+v", "%s"} {
		if s := fmt.Sprintf(format, cfg); strings.Contains(s, "private-password") {
			t.Errorf("%s leaks the password: %s", format, s)
		}
	}
}

func TestConfigurationRejectsInvalidURLsAndOutOfRangeSettingsWithoutValues(t *testing.T) {
	cases := []struct{ key, value, expected string }{
		{"DATABASE_URL", "mysql://user:secret@localhost/db", "DATABASE_URL must be a valid PostgreSQL URL"},
		{"DATABASE_URL", "", "DATABASE_URL must be a valid PostgreSQL URL"},
		{"DATABASE_URL", "postgres://user:secret@localhost/db?port=invalid", "DATABASE_URL must be a valid PostgreSQL URL"},
		{"TRESTLE_BIND_ADDR", "secret", "TRESTLE_BIND_ADDR must be an IP socket address with a nonzero port"},
		{"TRESTLE_BIND_ADDR", "127.0.0.1:0", "TRESTLE_BIND_ADDR must be an IP socket address with a nonzero port"},
		{"TRESTLE_MAX_DB_CONNECTIONS", "0", "TRESTLE_MAX_DB_CONNECTIONS must be an integer from 1 to 32"},
		{"TRESTLE_MAX_DB_CONNECTIONS", "33", "TRESTLE_MAX_DB_CONNECTIONS must be an integer from 1 to 32"},
		{"TRESTLE_MAX_DB_CONNECTIONS", "secret", "TRESTLE_MAX_DB_CONNECTIONS must be an integer from 1 to 32"},
		{"TRESTLE_DB_TIMEOUT_MS", "99", "TRESTLE_DB_TIMEOUT_MS must be an integer from 100 to 10000"},
		{"TRESTLE_DB_TIMEOUT_MS", "10001", "TRESTLE_DB_TIMEOUT_MS must be an integer from 100 to 10000"},
		{"TRESTLE_DB_TIMEOUT_MS", "", "TRESTLE_DB_TIMEOUT_MS must be an integer from 100 to 10000"},
	}
	for _, c := range cases {
		_, err := FromPairs(map[string]string{
			"DATABASE_URL": "postgres://user:secret@localhost/db",
			c.key:          c.value,
		})
		if err == nil {
			t.Errorf("%s=%q: accepted", c.key, c.value)
			continue
		}
		if err.Error() != c.expected {
			t.Errorf("%s=%q: got %q, want %q", c.key, c.value, err.Error(), c.expected)
		}
		if s := fmt.Sprintf("%#v", err); strings.Contains(s, "secret") {
			t.Errorf("%s=%q: error leaks input: %s", c.key, c.value, s)
		}
	}
}

func TestConfigurationAcceptsExplicitIPv6AndBoundaryValues(t *testing.T) {
	for _, c := range []struct{ connections, timeout string }{{"1", "100"}, {"32", "10000"}} {
		cfg, err := FromPairs(map[string]string{
			"DATABASE_URL":               "postgresql://user:secret@localhost/db",
			"TRESTLE_BIND_ADDR":          "[::1]:4000",
			"TRESTLE_MAX_DB_CONNECTIONS": c.connections,
			"TRESTLE_DB_TIMEOUT_MS":      c.timeout,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := cfg.BindAddr.String(); got != "[::1]:4000" {
			t.Errorf("bind addr %q", got)
		}
		if got := strconv.FormatInt(int64(cfg.MaxDBConnections), 10); got != c.connections {
			t.Errorf("connections %s, want %s", got, c.connections)
		}
		if got := strconv.FormatInt(cfg.DBTimeout.Milliseconds(), 10); got != c.timeout {
			t.Errorf("timeout %s, want %s", got, c.timeout)
		}
	}
}
