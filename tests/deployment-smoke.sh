#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

export TRESTLE_IMAGE=${1:?Pass the Trestle image to test}
export POSTGRES_PASSWORD=ci-database-password
export TRESTLE_PORT=13000
fixture=$(mktemp -d)
suffix=${fixture##*.}
stack=(docker compose --project-name "trestle-smoke-${suffix,,}" --file deploy/compose.yaml)
dump="$fixture/trestle.dump"
cleanup() {
    status=$?
    if (( status != 0 )); then
        "${stack[@]}" ps -a
        "${stack[@]}" logs
    fi
    "${stack[@]}" down -v
    rm -f "$dump"
    rmdir "$fixture"
}
trap cleanup EXIT

bash -n deploy/install.sh
"${stack[@]}" up -d --wait --wait-timeout 120 app
"${stack[@]}" exec -T \
    -e TRESTLE_BOOTSTRAP_USERNAME=smoke-user \
    -e TRESTLE_BOOTSTRAP_PASSWORD=ci-login-password \
    app /app/trestle bootstrap

login() {
    curl --fail --silent --show-error --max-time 10 \
        -H 'Content-Type: application/json' \
        -d '{"username":"smoke-user","password":"ci-login-password"}' \
        "http://127.0.0.1:$TRESTLE_PORT/api/v1/session" >/dev/null
}
login

"${stack[@]}" down
"${stack[@]}" up -d --wait --wait-timeout 120 app
login

"${stack[@]}" exec -T db pg_dump -U trestle -Fc trestle > "$dump"
"${stack[@]}" down -v
"${stack[@]}" up -d --wait --wait-timeout 120 db
"${stack[@]}" exec -T db pg_restore -U trestle --no-owner --exit-on-error -d trestle < "$dump"
"${stack[@]}" up -d --wait --wait-timeout 120 app
login

# A database from a newer build must keep the replacement app stopped.
"${stack[@]}" rm -sf app migrate
"${stack[@]}" exec -T db psql -v ON_ERROR_STOP=1 -U trestle -d trestle -c \
    "INSERT INTO _sqlx_migrations VALUES (999999999999, 'future', now(), true, ''::bytea, 0)"
if "${stack[@]}" up -d --wait --wait-timeout 30 app; then
    echo "Compose started the app after an incompatible migration." >&2
    exit 1
fi
[[ -z $("${stack[@]}" ps --status running -q app) ]]
