// Package database builds the connection pool and runs migrations.
package database

import (
	"bytes"
	"context"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maruf-pfc/trestle/internal/config"
	"github.com/maruf-pfc/trestle/migrations"
)

// Pool is constructed without connecting, so database outages do not prevent liveness.
func Pool(cfg config.Config) *pgxpool.Pool {
	c := cfg.Database.Copy()
	c.MinConns = 0
	c.MaxConns = cfg.MaxDBConnections
	c.ConnConfig.ConnectTimeout = cfg.DBTimeout
	c.ConnConfig.RuntimeParams["application_name"] = "trestle"
	c.ConnConfig.RuntimeParams["statement_timeout"] = "5000"
	c.ConnConfig.RuntimeParams["lock_timeout"] = "5000"
	pool, err := pgxpool.NewWithConfig(context.Background(), c)
	if err != nil {
		// Only reachable with an invalid config, which config.FromPairs already rejects.
		panic(err)
	}
	return pool
}

type migration struct {
	version      int64
	description  string
	sql          string
	checksum     []byte
	crlfChecksum []byte
}

var load = sync.OnceValues(loadMigrations)

func loadMigrations() ([]migration, error) {
	names, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	var out []migration
	for _, name := range names {
		stem := strings.TrimSuffix(name, ".sql")
		v, desc, ok := strings.Cut(stem, "_")
		version, err := strconv.ParseInt(v, 10, 64)
		if !ok || err != nil {
			return nil, fmt.Errorf("invalid migration filename %q", name)
		}
		body, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			return nil, err
		}
		sql := strings.ReplaceAll(string(body), "\r\n", "\n")
		sum := sha512.Sum384([]byte(sql))
		// Earlier builds hashed checkout bytes, which could have CRLF line endings.
		crlf := sha512.Sum384([]byte(strings.ReplaceAll(sql, "\n", "\r\n")))
		out = append(out, migration{version, strings.ReplaceAll(desc, "_", " "), sql, sum[:], crlf[:]})
	}
	return out, nil
}

// CheckSchema verifies the database's migration history against this build.
func CheckSchema(ctx context.Context, pool *pgxpool.Pool) error {
	all, err := load()
	if err != nil {
		return err
	}
	applied, err := appliedMigrations(ctx, pool, all)
	if err != nil {
		return err
	}
	if len(applied) != len(all) {
		return errors.New("database schema does not match this build")
	}
	return nil
}

type migrationReader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func appliedMigrations(ctx context.Context, db migrationReader, all []migration) (map[int64]bool, error) {
	expected := make(map[int64]migration, len(all))
	for _, m := range all {
		expected[m.version] = m
	}
	rows, err := db.Query(ctx, "SELECT version, success, checksum FROM _sqlx_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied := make(map[int64]bool, len(all))
	for rows.Next() {
		var version int64
		var success bool
		var checksum []byte
		if err := rows.Scan(&version, &success, &checksum); err != nil {
			return nil, err
		}
		m, known := expected[version]
		switch {
		case !success:
			return nil, fmt.Errorf("migration %d is partially applied; fix and remove it from _sqlx_migrations", version)
		case !known:
			return nil, fmt.Errorf("migration %d was applied but is missing from this build", version)
		case !bytes.Equal(checksum, m.checksum) && !bytes.Equal(checksum, m.crlfChecksum):
			return nil, fmt.Errorf("migration %d was modified after it was applied", version)
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

// Migrate applies pending migrations. Serving HTTP never invokes this function.
// It keeps the `_sqlx_migrations` history table and advisory lock key used by the
// previous SQLx-based backend, so existing databases migrate forward unchanged.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	all, err := load()
	if err != nil {
		return err
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	var dbName string
	if err := conn.QueryRow(ctx, "SELECT current_database()").Scan(&dbName); err != nil {
		return err
	}
	lockID := int64(0x3d32ad9e) * int64(crc32.ChecksumIEEE([]byte(dbName)))
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
		return err
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", lockID)

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS _sqlx_migrations (
		version BIGINT PRIMARY KEY,
		description TEXT NOT NULL,
		installed_on TIMESTAMPTZ NOT NULL DEFAULT now(),
		success BOOLEAN NOT NULL,
		checksum BYTEA NOT NULL,
		execution_time BIGINT NOT NULL
	)`); err != nil {
		return err
	}

	applied, err := appliedMigrations(ctx, conn, all)
	if err != nil {
		return err
	}

	for _, m := range all {
		if applied[m.version] {
			continue
		}
		started := time.Now()
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, m.sql); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %d failed: %w", m.version, err)
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO _sqlx_migrations (version, description, success, checksum, execution_time) VALUES ($1, $2, TRUE, $3, -1)",
			m.version, m.description, m.checksum); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		if _, err := conn.Exec(ctx, "UPDATE _sqlx_migrations SET execution_time = $1 WHERE version = $2",
			time.Since(started).Nanoseconds(), m.version); err != nil {
			return err
		}
	}
	return nil
}
