package database_test

import (
	"bytes"
	"context"
	"crypto/sha512"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maruf-pfc/trestle/internal/database"
	"github.com/maruf-pfc/trestle/internal/httpapi"
	"github.com/maruf-pfc/trestle/internal/testdb"
	"github.com/maruf-pfc/trestle/migrations"
)

func ready(t *testing.T, h http.Handler) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/health/ready", nil))
	b, _ := io.ReadAll(rec.Result().Body)
	return rec.Code, string(b)
}

func TestFreshDatabaseMigratesAndRepeatExecutionSucceeds(t *testing.T) {
	pool, cfg := testdb.Pool(t, false)
	ctx := context.Background()
	app := httpapi.Router(pool, 2*time.Second, cfg, nil)
	if status, _ := ready(t, app); status != http.StatusServiceUnavailable {
		t.Fatalf("readiness before migration = %d", status)
	}
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	status, body := ready(t, app)
	if status != http.StatusOK || body != `{"status":"ok"}` {
		t.Fatalf("readiness %d %s", status, body)
	}
}

func TestConcurrentMigrationCommandsAreSerializedSafely(t *testing.T) {
	pool, _ := testdb.Pool(t, false)
	ctx := context.Background()
	errs := make(chan error, 2)
	for range 2 {
		go func() { errs <- database.Migrate(ctx, pool) }()
	}
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
}

func TestReadinessRejectsModifiedEarlierMigration(t *testing.T) {
	pool, cfg := testdb.Pool(t, true)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "UPDATE _sqlx_migrations SET checksum = '' WHERE version = (SELECT min(version) FROM _sqlx_migrations)"); err != nil {
		t.Fatal(err)
	}
	app := httpapi.Router(pool, 2*time.Second, cfg, nil)
	if status, _ := ready(t, app); status != http.StatusServiceUnavailable {
		t.Fatalf("readiness with modified history = %d, want 503", status)
	}
	if err := database.Migrate(ctx, pool); err == nil {
		t.Fatal("migrate accepted the modified history")
	}
}

func TestMigrationHistoryAcceptsEquivalentWindowsLineEndings(t *testing.T) {
	pool, cfg := testdb.Pool(t, true)
	ctx := context.Background()
	raw, err := fs.ReadFile(migrations.FS, "202610150001_foundation.sql")
	if err != nil {
		t.Fatal(err)
	}
	lf := bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	expected := sha512.Sum384(lf)
	var recorded []byte
	if err := pool.QueryRow(ctx, "SELECT checksum FROM _sqlx_migrations WHERE version = 202610150001").Scan(&recorded); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(recorded, expected[:]) {
		t.Fatal("new migration did not record a normalized checksum")
	}
	crlf := sha512.Sum384(bytes.ReplaceAll(lf, []byte("\n"), []byte("\r\n")))
	if _, err := pool.Exec(ctx, "UPDATE _sqlx_migrations SET checksum = $1 WHERE version = 202610150001", crlf[:]); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate rejected legacy checkout checksum: %v", err)
	}
	if status, _ := ready(t, httpapi.Router(pool, 2*time.Second, cfg, nil)); status != http.StatusOK {
		t.Fatalf("readiness with equivalent legacy history = %d", status)
	}
}

func TestSaturatedApplicationPoolReturnsBoundedReadinessFailure(t *testing.T) {
	cfg := testdb.Config(t, testdb.URL(t), map[string]string{
		"TRESTLE_MAX_DB_CONNECTIONS": "1",
		"TRESTLE_DB_TIMEOUT_MS":      "2000",
	})
	pool := database.Pool(cfg)
	defer pool.Close()
	if err := database.Migrate(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
	// Establish the connection before testing saturation; the HTTP deadline is
	// shorter than pool acquisition to verify it bounds the whole readiness check.
	held, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	app := httpapi.Router(pool, 100*time.Millisecond, cfg, nil)

	done := make(chan int, 1)
	go func() { status, _ := ready(t, app); done <- status }()
	select {
	case status := <-done:
		if status != http.StatusServiceUnavailable {
			t.Fatalf("saturated readiness %d", status)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pool exhaustion must not hang readiness")
	}

	held.Release()
	if status, body := ready(t, app); status != http.StatusOK {
		t.Fatalf("readiness after release %d %s", status, body)
	}
}
