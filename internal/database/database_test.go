package database_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maruf-pfc/trestle/internal/database"
	"github.com/maruf-pfc/trestle/internal/httpapi"
	"github.com/maruf-pfc/trestle/internal/testdb"
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
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	status, body := ready(t, httpapi.Router(pool, 2*time.Second, cfg, nil))
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

func TestSaturatedApplicationPoolReturnsBoundedReadinessFailure(t *testing.T) {
	cfg := testdb.Config(t, testdb.URL(t), map[string]string{
		"TRESTLE_MAX_DB_CONNECTIONS": "1",
		"TRESTLE_DB_TIMEOUT_MS":      "2000",
	})
	pool := database.Pool(cfg)
	defer pool.Close()
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
