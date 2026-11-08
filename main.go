// Command trestle serves the workspace and provides operator commands.
package main

import (
	"bufio"
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/maruf-pfc/trestle/internal/config"
	"github.com/maruf-pfc/trestle/internal/database"
	"github.com/maruf-pfc/trestle/internal/httpapi"
	"github.com/maruf-pfc/trestle/internal/identity"
)

// The frontend must be built first: npm ci --prefix web && npm run build --prefix web.
//
//go:embed web/dist/index.html web/dist/assets
var dist embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(os.Args[1:]); err != nil {
		slog.Error("application stopped with an error", "message", err.Error())
		os.Exit(1)
	}
}

func loadDotEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			if _, exists := os.LookupEnv(k); !exists {
				_ = os.Setenv(k, v)
			}
		}
	}
}

func run(args []string) error {
	loadDotEnv()
	commands := map[string]func(config.Config) error{
		"migrate":       runMigrate,
		"bootstrap":     runBootstrap,
		"adduser":       runAddUser,
		"resetpassword": runResetPassword,
		"disableuser":   func(c config.Config) error { return runSetDisabled(c, true) },
		"enableuser":    func(c config.Config) error { return runSetDisabled(c, false) },
	}
	var cmd func(config.Config) error = runServer
	switch {
	case len(args) == 1 && commands[args[0]] != nil:
		cmd = commands[args[0]]
	case len(args) != 0:
		return errors.New("usage: trestle [migrate | bootstrap | adduser | resetpassword | disableuser | enableuser]")
	}
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	return cmd(cfg)
}

func runServer(cfg config.Config) error {
	pool := database.Pool(cfg)
	defer pool.Close()
	assets, err := fs.Sub(dist, "web/dist")
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              cfg.BindAddr.String(),
		Handler:           httpapi.Router(pool, cfg.DBTimeout, cfg, assets),
		ReadHeaderTimeout: 10 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errs := make(chan error, 1)
	go func() { errs <- server.ListenAndServe() }()
	slog.Info("HTTP server listening", "address", cfg.BindAddr.String())

	select {
	case err := <-errs:
		return fmt.Errorf("HTTP server failed: %w", err)
	case <-ctx.Done():
	}
	slog.Info("shutdown requested")
	drain, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(drain); err != nil {
		return errors.New("HTTP shutdown drain timed out")
	}
	return nil
}

func runMigrate(cfg config.Config) error {
	pool := database.Pool(cfg)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	// The raw error can name the database user and host, so it is never printed.
	if err := database.Migrate(ctx, pool); err != nil {
		errMsg := err.Error()
		if !strings.Contains(errMsg, "://") && !strings.Contains(errMsg, "@") {
			slog.Error("database migration failed details", "error", errMsg)
		} else {
			slog.Error("database migration failed details (connection error)")
		}
		return errors.New("database migration failed or timed out; verify connectivity, permissions and migration history")
	}
	slog.Info("database migrations complete")
	return nil
}

// ---------------------------------------------------------------------------
// Operator commands. Each reads env vars first and falls back to prompts.
// ---------------------------------------------------------------------------

var stdin = bufio.NewReader(os.Stdin)

func promptPlain(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	line, err := stdin.ReadString('\n')
	if err != nil && line == "" {
		return "", errors.New("failed to read input")
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return "", errors.New("Input cannot be empty.")
	}
	return line, nil
}

func promptHidden(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", errors.New("failed to read password")
	}
	p := strings.TrimRight(string(b), "\r\n")
	if p == "" {
		return "", errors.New("Password cannot be empty.")
	}
	return p, nil
}

// promptNewPassword asks twice and requires a match.
func promptNewPassword(prompt string) (string, error) {
	p, err := promptHidden(prompt)
	if err != nil {
		return "", err
	}
	again, err := promptHidden("Confirm password: ")
	if err != nil {
		return "", err
	}
	if p != again {
		return "", errors.New("Passwords do not match.")
	}
	return p, nil
}

func hashPassword(cfg config.Config, password string) (string, error) {
	if len(password) < identity.MinPasswordBytes {
		return "", errors.New("Password must be at least 8 characters.")
	}
	h, err := identity.NewHasher(cfg.Argon2MCost, cfg.Argon2TCost, cfg.Argon2Parallelism)
	if err != nil {
		return "", fmt.Errorf("argon2 configuration error: %w", err)
	}
	hash, err := h.Hash([]byte(password))
	if err != nil {
		return "", fmt.Errorf("password error: %w", err)
	}
	return hash, nil
}

func parseUsername(raw string) (string, error) {
	u, err := identity.ParseUsername(raw)
	if err != nil {
		return "", fmt.Errorf("Invalid username: %w", err)
	}
	return u, nil
}

func validWorkspaceSlug(slug string) bool {
	if len(slug) < 2 || len(slug) > 48 || slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	for _, c := range slug {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

func runBootstrap(cfg config.Config) error {
	pool := database.Pool(cfg)
	defer pool.Close()
	ctx := context.Background()

	exists, err := identity.AnyUserExists(ctx, pool)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if exists {
		fmt.Fprintln(os.Stderr, "This instance is already initialized. Skipping bootstrap.")
		return nil
	}

	username, uok := os.LookupEnv("TRESTLE_BOOTSTRAP_USERNAME")
	password, pok := os.LookupEnv("TRESTLE_BOOTSTRAP_PASSWORD")
	wsName, wsSlug := "Main Workspace", "main"
	if uok && pok {
		if v, ok := os.LookupEnv("TRESTLE_BOOTSTRAP_WORKSPACE_NAME"); ok {
			wsName = v
		}
		if v, ok := os.LookupEnv("TRESTLE_BOOTSTRAP_WORKSPACE_SLUG"); ok {
			wsSlug = v
		}
	} else {
		fmt.Fprintln(os.Stderr, "Trestle bootstrap: create the initial operator account.")
		fmt.Fprintln(os.Stderr, "This can only be run once. The account is not a workspace member until")
		fmt.Fprintln(os.Stderr, "workspace membership is granted.")
		fmt.Fprintln(os.Stderr)
		if username, err = promptPlain("Username (3-64 characters, letters/digits/._-): "); err != nil {
			return err
		}
		if password, err = promptNewPassword("Password (at least 8 characters): "); err != nil {
			return err
		}
		if wsName, err = promptPlain("Initial workspace name: "); err != nil {
			return err
		}
		if wsSlug, err = promptPlain("Workspace slug (URL identifier, 2-48 chars, lowercase alphanumeric + hyphens): "); err != nil {
			return err
		}
	}

	username, err = parseUsername(username)
	if err != nil {
		return err
	}
	if len(password) < identity.MinPasswordBytes {
		return errors.New("Password must be at least 8 characters.")
	}
	if strings.TrimSpace(wsName) == "" || len(wsName) > 120 {
		return errors.New("Workspace name must be 1-120 characters.")
	}
	wsSlug = strings.ToLower(strings.TrimSpace(wsSlug))
	if !validWorkspaceSlug(wsSlug) {
		return errors.New("Invalid workspace slug. Use 2-48 lowercase alphanumeric characters and hyphens.")
	}
	hash, err := hashPassword(cfg, password)
	if err != nil {
		return err
	}
	user, err := identity.Bootstrap(ctx, pool, username, hash, username, strings.TrimSpace(wsName), wsSlug)
	if errors.Is(err, identity.ErrConflict) {
		return errors.New("Another bootstrap completed concurrently. This instance is already initialized.")
	} else if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Bootstrap complete.")
	fmt.Fprintf(os.Stderr, "  Username:  %s\n", user.Username)
	fmt.Fprintf(os.Stderr, "  Workspace: %s (slug: %s)\n", wsName, wsSlug)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Sign in at the application URL with your username and password.")
	return nil
}

func runAddUser(cfg config.Config) error {
	pool := database.Pool(cfg)
	defer pool.Close()

	username, uok := os.LookupEnv("TRESTLE_ADDUSER_USERNAME")
	password, pok := os.LookupEnv("TRESTLE_ADDUSER_PASSWORD")
	displayName := username
	var err error
	if uok && pok {
		if v, ok := os.LookupEnv("TRESTLE_ADDUSER_DISPLAY_NAME"); ok {
			displayName = v
		}
	} else {
		fmt.Fprintln(os.Stderr, "Trestle operator user provisioning:")
		fmt.Fprintln(os.Stderr)
		if username, err = promptPlain("Username (3-64 characters, letters/digits/._-): "); err != nil {
			return err
		}
		if displayName, err = promptPlain("Display name: "); err != nil {
			return err
		}
		if password, err = promptNewPassword("Password (at least 8 characters): "); err != nil {
			return err
		}
	}
	if username, err = parseUsername(username); err != nil {
		return err
	}
	if len(password) < identity.MinPasswordBytes {
		return errors.New("Password must be at least 8 characters.")
	}
	if strings.TrimSpace(displayName) == "" || len(displayName) > 120 {
		return errors.New("Display name must be 1-120 characters.")
	}
	hash, err := hashPassword(cfg, password)
	if err != nil {
		return err
	}
	user, err := identity.CreateUser(context.Background(), pool, username, hash, strings.TrimSpace(displayName))
	if errors.Is(err, identity.ErrConflict) {
		return errors.New("User already exists.")
	} else if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "User provisioned successfully.")
	fmt.Fprintf(os.Stderr, "  Username:     %s\n", user.Username)
	fmt.Fprintf(os.Stderr, "  Display name: %s\n", user.DisplayName)
	fmt.Fprintln(os.Stderr, "(Note: Workspace membership must be granted separately.)")
	return nil
}

func runResetPassword(cfg config.Config) error {
	pool := database.Pool(cfg)
	defer pool.Close()

	username, uok := os.LookupEnv("TRESTLE_RESET_USERNAME")
	password, pok := os.LookupEnv("TRESTLE_RESET_PASSWORD")
	var err error
	if !uok || !pok {
		fmt.Fprintln(os.Stderr, "Trestle operator password reset & session revocation:")
		fmt.Fprintln(os.Stderr)
		if username, err = promptPlain("Username to reset: "); err != nil {
			return err
		}
		if password, err = promptNewPassword("New password (at least 8 characters): "); err != nil {
			return err
		}
	}
	if username, err = parseUsername(username); err != nil {
		return err
	}
	hash, err := hashPassword(cfg, password)
	if err != nil {
		return err
	}
	user, err := identity.ResetUserPassword(context.Background(), pool, username, hash)
	if errors.Is(err, identity.ErrNotFound) {
		return errors.New("User not found.")
	} else if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Password reset successfully for user '%s'.\n", user.Username)
	fmt.Fprintln(os.Stderr, "All existing active sessions for this user have been revoked.")
	return nil
}

func runSetDisabled(cfg config.Config, disabled bool) error {
	pool := database.Pool(cfg)
	defer pool.Close()

	envKey, prompt, verb := "TRESTLE_ENABLE_USERNAME", "Username to enable: ", "enabled"
	if disabled {
		envKey, prompt, verb = "TRESTLE_DISABLE_USERNAME", "Username to disable: ", "disabled"
	}
	username, ok := os.LookupEnv(envKey)
	var err error
	if !ok {
		if username, err = promptPlain(prompt); err != nil {
			return err
		}
	}
	if username, err = parseUsername(username); err != nil {
		return err
	}
	user, err := identity.SetUserDisabled(context.Background(), pool, username, disabled)
	switch {
	case errors.Is(err, identity.ErrNotFound):
		return errors.New("User not found.")
	case errors.Is(err, identity.ErrConflict):
		return errors.New("Cannot disable user: this user is the sole active owner of one or more workspaces.")
	case err != nil:
		return fmt.Errorf("database error: %w", err)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "User '%s' has been %s.\n", user.Username, verb)
	if disabled {
		fmt.Fprintln(os.Stderr, "All existing active sessions have been revoked.")
	}
	return nil
}
