// Package config reads and validates runtime configuration from the environment.
package config

import (
	"errors"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Argon2id parameter defaults (benchmarked on reference host 2026-11-01).
// m=65536 KiB (64 MiB), t=2, p=1 → ~122ms per hash on this host.
const (
	DefaultArgon2MCost       = 65_536
	DefaultArgon2TCost       = 2
	DefaultArgon2Parallelism = 1
)

type Config struct {
	BindAddr         netip.AddrPort
	MaxDBConnections int32
	DBTimeout        time.Duration
	// Database holds the parsed connection settings; it is never logged.
	Database *pgxpool.Config
	// DevMode relaxes the cookie Secure requirement for loopback HTTP only.
	DevMode           bool
	Argon2MCost       uint32
	Argon2TCost       uint32
	Argon2Parallelism uint32
}

// String redacts the database settings.
func (c Config) String() string {
	return "Config{BindAddr:" + c.BindAddr.String() + " Database:[REDACTED]}"
}

var keys = []string{
	"DATABASE_URL",
	"TRESTLE_BIND_ADDR",
	"TRESTLE_MAX_DB_CONNECTIONS",
	"TRESTLE_DB_TIMEOUT_MS",
	"TRESTLE_DEV_MODE",
	"TRESTLE_ARGON2_M_COST",
	"TRESTLE_ARGON2_T_COST",
	"TRESTLE_ARGON2_PARALLELISM",
}

func FromEnv() (Config, error) {
	values := map[string]string{}
	for _, key := range keys {
		if v, ok := os.LookupEnv(key); ok {
			if !utf8.ValidString(v) {
				return Config{}, errors.New(key + " must be valid Unicode")
			}
			values[key] = v
		}
	}
	return FromPairs(values)
}

func FromPairs(values map[string]string) (Config, error) {
	url, ok := values["DATABASE_URL"]
	if !ok {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	badURL := errors.New("DATABASE_URL must be a valid PostgreSQL URL")
	if !strings.HasPrefix(url, "postgres://") && !strings.HasPrefix(url, "postgresql://") {
		return Config{}, badURL
	}
	database, err := pgxpool.ParseConfig(url)
	if err != nil {
		return Config{}, badURL
	}

	bind := "127.0.0.1:3000"
	if v, ok := values["TRESTLE_BIND_ADDR"]; ok {
		bind = v
	}
	bindAddr, err := netip.ParseAddrPort(bind)
	if err != nil || bindAddr.Port() == 0 {
		return Config{}, errors.New("TRESTLE_BIND_ADDR must be an IP socket address with a nonzero port")
	}

	maxConns, err := boundedInt32(values, "TRESTLE_MAX_DB_CONNECTIONS", 5, 1, 32,
		"TRESTLE_MAX_DB_CONNECTIONS must be an integer from 1 to 32")
	if err != nil {
		return Config{}, err
	}
	timeoutMS, err := bounded(values, "TRESTLE_DB_TIMEOUT_MS", 2000, 100, 10000,
		"TRESTLE_DB_TIMEOUT_MS must be an integer from 100 to 10000")
	if err != nil {
		return Config{}, err
	}

	devMode := values["TRESTLE_DEV_MODE"] == "true"
	if devMode && !bindAddr.Addr().IsLoopback() {
		return Config{}, errors.New("TRESTLE_DEV_MODE=true is only permitted with a loopback bind address")
	}

	// OWASP minimum memory is 19456 KiB; maximum 512 MiB.
	mCost, err := bounded(values, "TRESTLE_ARGON2_M_COST", DefaultArgon2MCost, 19_456, 524_288,
		"TRESTLE_ARGON2_M_COST must be an integer from 19456 to 524288")
	if err != nil {
		return Config{}, err
	}
	tCost, err := bounded(values, "TRESTLE_ARGON2_T_COST", DefaultArgon2TCost, 1, 16,
		"TRESTLE_ARGON2_T_COST must be an integer from 1 to 16")
	if err != nil {
		return Config{}, err
	}
	parallelism, err := bounded(values, "TRESTLE_ARGON2_PARALLELISM", DefaultArgon2Parallelism, 1, 8,
		"TRESTLE_ARGON2_PARALLELISM must be an integer from 1 to 8")
	if err != nil {
		return Config{}, err
	}

	return Config{
		BindAddr:          bindAddr,
		MaxDBConnections:  maxConns,
		DBTimeout:         time.Duration(timeoutMS) * time.Millisecond,
		Database:          database,
		DevMode:           devMode,
		Argon2MCost:       mCost,
		Argon2TCost:       tCost,
		Argon2Parallelism: parallelism,
	}, nil
}

func bounded(values map[string]string, key string, def, min, max uint32, message string) (uint32, error) {
	raw, ok := values[key]
	if !ok {
		return def, nil
	}
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || uint32(n) < min || uint32(n) > max {
		return 0, errors.New(message)
	}
	return uint32(n), nil
}

func boundedInt32(values map[string]string, key string, def, min, max int32, message string) (int32, error) {
	raw, ok := values[key]
	if !ok {
		return def, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || int32(n) < min || int32(n) > max {
		return 0, errors.New(message)
	}
	return int32(n), nil
}
