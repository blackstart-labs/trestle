// Package database builds the connection pool and runs migrations.
package database

import (
	"context"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"sort"
	"strconv"
	"strings"
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
	version     int64
	description string
	sql         string
	checksum    []byte
}

func load() ([]migration, error) {
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
		sum := sha512.Sum384(body)
		out = append(out, migration{version, strings.ReplaceAll(desc, "_", " "), string(body), sum[:]})
	}
	return out, nil
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

	var dirty int64
	err = conn.QueryRow(ctx, "SELECT version FROM _sqlx_migrations WHERE success = false ORDER BY version LIMIT 1").Scan(&dirty)
	if err == nil {
		return fmt.Errorf("migration %d is partially applied; fix and remove it from _sqlx_migrations", dirty)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	rows, err := conn.Query(ctx, "SELECT version, checksum FROM _sqlx_migrations ORDER BY version")
	if err != nil {
		return err
	}
	applied := map[int64][]byte{}
	for rows.Next() {
		var v int64
		var sum []byte
		if err := rows.Scan(&v, &sum); err != nil {
			rows.Close()
			return err
		}
		applied[v] = sum
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	known := map[int64]bool{}
	for _, m := range all {
		known[m.version] = true
	}
	for v := range applied {
		if !known[v] {
			return fmt.Errorf("migration %d was applied but is missing from this build", v)
		}
	}

	for _, m := range all {
		if sum, ok := applied[m.version]; ok {
			if string(sum) != string(m.checksum) {
				return fmt.Errorf("migration %d was modified after it was applied", m.version)
			}
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
