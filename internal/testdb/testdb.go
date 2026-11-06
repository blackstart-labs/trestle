// Package testdb gives each test a fresh, migrated PostgreSQL database.
// Tests skip when DATABASE_URL is unset.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maruf-pfc/trestle/internal/config"
	"github.com/maruf-pfc/trestle/internal/database"
)

// URL creates an empty database and returns its URL; it is dropped when the test ends.
func URL(t *testing.T) string {
	t.Helper()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		t.Skip("DATABASE_URL not set; skipping database test")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("connect to DATABASE_URL: %v", err)
	}
	b := make([]byte, 8)
	rand.Read(b)
	name := "trestle_test_" + hex.EncodeToString(b)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		admin.Close(context.Background())
	})
	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	return u.String()
}

// Config returns a dev-mode config with fast Argon2 parameters for the given database URL.
func Config(t *testing.T, dbURL string, extra map[string]string) config.Config {
	t.Helper()
	pairs := map[string]string{"DATABASE_URL": dbURL, "TRESTLE_DEV_MODE": "true"}
	for k, v := range extra {
		pairs[k] = v
	}
	cfg, err := config.FromPairs(pairs)
	if err != nil {
		t.Fatal(err)
	}
	// Below the production minimum, to keep tests fast.
	cfg.Argon2MCost, cfg.Argon2TCost = 4096, 1
	return cfg
}

// Pool returns a pool on a fresh database; migrate controls whether migrations run first.
func Pool(t *testing.T, migrate bool) (*pgxpool.Pool, config.Config) {
	t.Helper()
	cfg := Config(t, URL(t), nil)
	pool := database.Pool(cfg)
	t.Cleanup(pool.Close)
	if migrate {
		if err := database.Migrate(context.Background(), pool); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}
	return pool, cfg
}
