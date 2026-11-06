#!/usr/bin/env bash
# ==============================================================================
# Trestle — Unified One-Command Setup & Runner
# ==============================================================================
# Automatically handles:
#  1. Self-bootstrap (clones repo if run externally via curl/pipe)
#  2. Prerequisite checking (Go, Node, Docker/Podman)
#  3. Dynamic port collision detection & resolution for DB and Web ports
#  4. Environment (.env) initialization with secure credentials
#  5. PostgreSQL container launch & readiness verification
#  6. Frontend dependency installation and production build
#  7. Database migration execution
#  8. Operator bootstrap initialization
#  9. Application startup and graceful shutdown
# ==============================================================================

set -eo pipefail

# Ensure we are in the repository root. If run outside (e.g. curl ... | bash), clone it.
if [ ! -f "go.mod" ] || [ ! -d "web" ]; then
    echo "==> Trestle repository not detected in current directory."
    if ! command -v git >/dev/null 2>&1; then
        echo "[ERROR] 'git' is required to clone the repository. Please install git and try again." >&2
        exit 1
    fi
    echo "==> Cloning https://github.com/maruf-pfc/trestle.git..."
    git clone https://github.com/maruf-pfc/trestle.git trestle
    cd trestle
    exec ./run.sh "$@"
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$REPO_ROOT"

# --- Helper Utilities ---

bold() { printf "\033[1m%s\033[0m\n" "$*"; }
green() { printf "\033[32m%s\033[0m\n" "$*"; }
yellow() { printf "\033[33m%s\033[0m\n" "$*"; }
cyan() { printf "\033[36m%s\033[0m\n" "$*"; }
red() { printf "\033[31m%s\033[0m\n" "$*"; }

# Check if a port is actively listening on host
port_is_open() {
    local host="$1"
    local port="$2"
    if command -v node >/dev/null 2>&1; then
        node -e "
            const net = require('net');
            const server = net.createServer();
            server.once('error', (err) => {
                if (err.code === 'EADDRINUSE') process.exit(0);
                process.exit(1);
            });
            server.once('listening', () => {
                server.close(() => process.exit(1));
            });
            server.listen($port, '$host');
        " >/dev/null 2>&1
        return $?
    elif command -v nc >/dev/null 2>&1; then
        nc -z -w 1 "$host" "$port" >/dev/null 2>&1
        return $?
    else
        (exec 3<>"/dev/tcp/$host/$port") >/dev/null 2>&1 && { exec 3>&-; return 0; } || return 1
    fi
}

# Find next available port starting from start_port
find_free_port() {
    local host="$1"
    local start_port="$2"
    local port="$start_port"
    while [ "$port" -le 65535 ]; do
        if ! port_is_open "$host" "$port"; then
            echo "$port"
            return 0
        fi
        port=$((port + 1))
    done
    echo "$start_port"
}

# Detect container runtime
detect_container_runtime() {
    if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD="docker compose"
        CONTAINER_CLI="docker"
    elif command -v podman >/dev/null 2>&1 && podman compose version >/dev/null 2>&1; then
        COMPOSE_CMD="podman compose"
        CONTAINER_CLI="podman"
    elif command -v podman-compose >/dev/null 2>&1; then
        COMPOSE_CMD="podman-compose"
        CONTAINER_CLI="podman"
    else
        COMPOSE_CMD=""
        CONTAINER_CLI=""
    fi
}

# Check if our trestle-db container is already running
is_trestle_db_running() {
    if [ -n "$CONTAINER_CLI" ]; then
        local status
        status=$($CONTAINER_CLI inspect trestle-db-1 --format '{{.State.Status}}' 2>/dev/null || true)
        [ "$status" = "running" ] && return 0
    fi
    return 1
}

# Get bound host port of running trestle-db container
get_trestle_db_port() {
    if [ -n "$CONTAINER_CLI" ]; then
        $CONTAINER_CLI port trestle-db-1 5432/tcp 2>/dev/null | head -n1 | awk -F: '{print $NF}'
    fi
}

# Inspect password from existing container if present
get_trestle_db_existing_password() {
    if [ -n "$CONTAINER_CLI" ]; then
        $CONTAINER_CLI inspect trestle-db-1 --format '{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null | grep '^POSTGRES_PASSWORD=' | cut -d= -f2-
    fi
}

# --- Action: Stop ---
action_stop() {
    detect_container_runtime
    echo "Stopping Trestle services..."
    if [ -n "$COMPOSE_CMD" ]; then
        $COMPOSE_CMD stop db || true
        green "✓ Database container stopped."
    else
        yellow "No container engine detected."
    fi
    exit 0
}

# Handle CLI arguments
case "${1:-}" in
    stop)
        action_stop
        ;;
    --help|-h|help)
        echo "Usage: ./run.sh [command | option]"
        echo ""
        echo "Commands:"
        echo "  (none)        Setup environment, database, frontend, migrate, and start Trestle"
        echo "  stop          Stop the background database container"
        echo "  db            Start only the PostgreSQL database container"
        echo "  migrate       Run database migrations"
        echo "  bootstrap     Run operator account bootstrap"
        echo "  help          Display this help message"
        echo ""
        echo "Options:"
        echo "  --interactive Prompt for custom operator credentials during bootstrap"
        exit 0
        ;;
    --interactive|-i)
        INTERACTIVE_BOOTSTRAP="true"
        ;;
    db)
        detect_container_runtime
        if [ -z "$COMPOSE_CMD" ]; then
            red "[ERROR] Docker or Podman with Compose is required to start the database."
            exit 1
        fi
        $COMPOSE_CMD up -d db
        green "✓ Database service started."
        exit 0
        ;;
    migrate)
        [ -f ".env" ] && set -a && source .env && set +a
        go run . migrate
        exit 0
        ;;
    bootstrap)
        [ -f ".env" ] && set -a && source .env && set +a
        go run . bootstrap
        exit 0
        ;;
esac

# ==============================================================================
# MAIN SETUP & RUN FLOW
# ==============================================================================

echo ""
bold "=================================================================="
bold "                 TRESTLE — ONE-COMMAND SETUP                      "
bold "=================================================================="
echo ""

# 1. Check prerequisites
echo "[1/6] Checking system tools and prerequisites..."

if ! command -v go >/dev/null 2>&1; then
    red "[ERROR] Go toolchain (go) is not installed."
    echo "Install Go (the version in go.mod or newer): https://go.dev/dl/"
    exit 1
fi
GO_VERSION=$(go env GOVERSION)
green "  ✓ Go toolchain found ($GO_VERSION)"

if ! command -v node >/dev/null 2>&1 || ! command -v npm >/dev/null 2>&1; then
    red "[ERROR] Node.js and npm are required to build the frontend."
    echo "Please install Node.js (v24 recommended): https://nodejs.org/"
    exit 1
fi
NODE_VERSION=$(node --version)
green "  ✓ Node.js & npm found ($NODE_VERSION)"

detect_container_runtime
if [ -n "$COMPOSE_CMD" ]; then
    green "  ✓ Container runtime found ($COMPOSE_CMD)"
else
    yellow "  ! Docker/Podman Compose not detected. An existing local PostgreSQL database must be configured in .env."
fi

# 2. Port Collision Detection & Environment Setup
echo ""
echo "[2/6] Resolving ports and configuring environment..."

# Load existing .env if present
if [ -f ".env" ]; then
    set -a
    source .env
    set +a
fi

# Determine PostgreSQL Port
TARGET_PG_PORT="${POSTGRES_PORT:-5432}"

if is_trestle_db_running; then
    EXISTING_PORT=$(get_trestle_db_port)
    if [ -n "$EXISTING_PORT" ]; then
        POSTGRES_PORT="$EXISTING_PORT"
        green "  ✓ Reusing running Trestle database on 127.0.0.1:$POSTGRES_PORT"
    fi
else
    # Check if target port is occupied by another service
    if port_is_open "127.0.0.1" "$TARGET_PG_PORT"; then
        RESOLVED_PG_PORT=$(find_free_port "127.0.0.1" 5433)
        yellow "  ! Port $TARGET_PG_PORT is in use by another service/database."
        cyan   "  ➜ Auto-selected free port $RESOLVED_PG_PORT for Trestle PostgreSQL."
        POSTGRES_PORT="$RESOLVED_PG_PORT"
    else
        POSTGRES_PORT="$TARGET_PG_PORT"
        green "  ✓ Database port 127.0.0.1:$POSTGRES_PORT is available"
    fi
fi

# Determine Web Application Port
DEFAULT_WEB_PORT=3000
if [ -n "${TRESTLE_BIND_ADDR:-}" ]; then
    CURRENT_WEB_PORT=$(echo "$TRESTLE_BIND_ADDR" | awk -F: '{print $NF}')
else
    CURRENT_WEB_PORT=$DEFAULT_WEB_PORT
fi

if port_is_open "127.0.0.1" "$CURRENT_WEB_PORT"; then
    RESOLVED_WEB_PORT=$(find_free_port "127.0.0.1" $((CURRENT_WEB_PORT + 1)))
    yellow "  ! Port $CURRENT_WEB_PORT is in use by another application."
    cyan   "  ➜ Auto-selected free port $RESOLVED_WEB_PORT for Trestle web server."
    TRESTLE_BIND_ADDR="127.0.0.1:$RESOLVED_WEB_PORT"
else
    TRESTLE_BIND_ADDR="127.0.0.1:$CURRENT_WEB_PORT"
    green "  ✓ Web application port $TRESTLE_BIND_ADDR is available"
fi

# Determine Database Password
if [ -z "${POSTGRES_PASSWORD:-}" ]; then
    EXISTING_PASS=$(get_trestle_db_existing_password)
    if [ -n "$EXISTING_PASS" ]; then
        POSTGRES_PASSWORD="$EXISTING_PASS"
    elif command -v openssl >/dev/null 2>&1; then
        POSTGRES_PASSWORD="$(openssl rand -hex 24)"
    else
        POSTGRES_PASSWORD="$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')"
    fi
fi

DATABASE_URL="postgres://trestle:${POSTGRES_PASSWORD}@127.0.0.1:${POSTGRES_PORT}/trestle"
TRESTLE_DEV_MODE="true"

# Write / update .env
cat <<EOF > .env
# Trestle local configuration
# Generated automatically by run.sh — preserves custom values when present.
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_PORT=${POSTGRES_PORT}
DATABASE_URL=${DATABASE_URL}
TRESTLE_BIND_ADDR=${TRESTLE_BIND_ADDR}
TRESTLE_DEV_MODE=${TRESTLE_DEV_MODE}
TRESTLE_MAX_DB_CONNECTIONS=${TRESTLE_MAX_DB_CONNECTIONS:-5}
TRESTLE_DB_TIMEOUT_MS=${TRESTLE_DB_TIMEOUT_MS:-2000}
EOF

export POSTGRES_PASSWORD POSTGRES_PORT DATABASE_URL TRESTLE_BIND_ADDR TRESTLE_DEV_MODE
green "  ✓ Configuration saved to .env"

# 3. Start Database
echo ""
echo "[3/6] Starting PostgreSQL database..."

if [ -n "$COMPOSE_CMD" ]; then
    $COMPOSE_CMD up -d db
    printf "  ➜ Waiting for PostgreSQL on port %s to be ready..." "$POSTGRES_PORT"
    DB_READY=false
    for _ in $(seq 1 30); do
        if $COMPOSE_CMD exec -T db pg_isready -U trestle -d trestle >/dev/null 2>&1; then
            DB_READY=true
            break
        fi
        sleep 1
        printf "."
    done
    echo ""
    if [ "$DB_READY" != "true" ]; then
        red "[ERROR] PostgreSQL did not become ready within 30 seconds."
        exit 1
    fi
    green "  ✓ PostgreSQL is healthy and accepting connections."
else
    # Verify local PostgreSQL
    if ! port_is_open "127.0.0.1" "$POSTGRES_PORT"; then
        red "[ERROR] No database is running at 127.0.0.1:$POSTGRES_PORT."
        echo "Please start Docker/Podman or start your local PostgreSQL 18 service."
        exit 1
    fi
    green "  ✓ Local PostgreSQL detected on 127.0.0.1:$POSTGRES_PORT."
fi

# 4. Frontend Dependencies & Build
echo ""
echo "[4/6] Verifying frontend assets..."

if [ ! -d "web/node_modules" ]; then
    echo "  ➜ Installing frontend dependencies..."
    npm ci --prefix web --ignore-scripts || npm install --prefix web --ignore-scripts
fi

if [ ! -d "web/dist" ] || [ ! -f "web/dist/index.html" ]; then
    echo "  ➜ Building frontend production bundle..."
    npm run build --prefix web
else
    # Check if any web/src file is newer than web/dist
    if [ -n "$(find web/src -newer web/dist/index.html 2>/dev/null | head -n1)" ]; then
        echo "  ➜ Frontend source changes detected. Rebuilding..."
        npm run build --prefix web
    fi
fi
green "  ✓ Frontend assets ready in web/dist."

# 5. Database Migrations
echo ""
echo "[5/6] Applying database migrations..."
go run . migrate
green "  ✓ Database schema is up to date."

# 6. Instance Bootstrap (Initial Operator Account)
echo ""
echo "[6/6] Checking instance bootstrap..."

# Set default operator credentials for frictionless 1-command setup unless --interactive was requested
if [ "${INTERACTIVE_BOOTSTRAP:-false}" != "true" ] && [ -z "${TRESTLE_BOOTSTRAP_USERNAME:-}" ]; then
    export TRESTLE_BOOTSTRAP_USERNAME="admin"
    export TRESTLE_BOOTSTRAP_PASSWORD="adminpassword123"
    export TRESTLE_BOOTSTRAP_WORKSPACE_NAME="Main Workspace"
    export TRESTLE_BOOTSTRAP_WORKSPACE_SLUG="main"
fi

# Run bootstrap (idempotent: exits 0 if already initialized)
go run . bootstrap || true

# --- Launch Server ---
APP_URL="http://${TRESTLE_BIND_ADDR}"

echo ""
bold "=================================================================="
green "  🚀 TRESTLE IS READY AND RUNNING!"
bold "=================================================================="
cyan "  Web Application:  ${APP_URL}"
cyan "  Database:         127.0.0.1:${POSTGRES_PORT} (db: trestle)"
if [ -n "${TRESTLE_BOOTSTRAP_USERNAME:-}" ]; then
cyan "  Default Admin:    username: ${TRESTLE_BOOTSTRAP_USERNAME} / password: ${TRESTLE_BOOTSTRAP_PASSWORD}"
fi
bold "=================================================================="
echo "Press Ctrl+C to stop the application server."
echo "(To stop the background database container later, run: ./run.sh stop)"
echo ""

# Handle graceful shutdown on Ctrl+C
cleanup() {
    echo ""
    yellow "==> Trestle application stopped."
    echo "    (Database container is still running in background. Run './run.sh stop' to stop it.)"
    exit 0
}
trap cleanup SIGINT SIGTERM

go run .
