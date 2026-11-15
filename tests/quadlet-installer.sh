#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

fixture=$(mktemp -d)
trap 'rm -rf -- "$fixture"' EXIT
export fixture
export XDG_DATA_HOME="$fixture/data" XDG_CONFIG_HOME="$fixture/config"
export PATH="$fixture/bin:$PATH" TRESTLE_RUNTIME=podman
export image
image=$(printf 'docker.io/example/trestle@sha256:%064d' 1)
mkdir "$fixture/bin"

cat > "$fixture/bin/podman" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
    'info --format {{.Host.Security.Rootless}} {{.Host.CgroupsVersion}}') echo 'true v2' ;;
    'volume ls --format {{.Name}}') printf '%s\n' "${fixture_volume:-}" ;;
    'exec --interactive --tty trestle-app /app/trestle bootstrap') echo bootstrap >> "$fixture/calls" ;;
    *) exit 1 ;;
esac
EOF
cat > "$fixture/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
    '--user show-environment') ;;
    '--user daemon-reload') echo reload >> "$fixture/calls" ;;
    '--user start trestle-app.service') echo start >> "$fixture/calls" ;;
    '--user start trestle-caddy.service') echo proxy >> "$fixture/calls" ;;
    *) exit 1 ;;
esac
EOF
printf '#!/usr/bin/env bash\necho yes\n' > "$fixture/bin/loginctl"
printf '#!/usr/bin/env bash\necho "${fixture_min_port:-80}"\n' > "$fixture/bin/sysctl"
chmod +x "$fixture/bin/"*

if fixture_volume=trestle-postgres-data script -qec 'bash deploy/install.sh "$image"' /dev/null > "$fixture/refusal"; then
    echo "Installer would reuse a database without its credentials." >&2
    exit 1
fi
grep -Fq 'Found an existing Trestle database volume' "$fixture/refusal"
test ! -e "$XDG_DATA_HOME/trestle/.env"

script -qec 'bash deploy/install.sh "$image"' /dev/null
env_file="$XDG_DATA_HOME/trestle/.env"
units="$XDG_CONFIG_HOME/containers/systemd/trestle"
test "$(stat -c %a "$env_file")" = 600
test "$units/trestle.env" -ef "$env_file"
grep -Fxq 'TRESTLE_RUNTIME=podman' "$env_file"
grep -Eq '^DATABASE_URL=postgres://trestle:[a-f0-9]{64}@trestle-db:5432/trestle\?sslmode=disable$' "$env_file"
grep -Fxq "Image=$image" "$units/trestle-app.container"
grep -Fxq "Image=$image" "$units/trestle-migrate.container"
postgres_image=$(sed -n 's/^    image: \(postgres:.*\)$/\1/p' deploy/compose.yaml)
grep -Fxq "Image=docker.io/library/$postgres_image" "$units/trestle-db.container"
printf 'reload\nstart\nbootstrap\n' | cmp - "$fixture/calls"

cp "$env_file" "$fixture/original-env"
script -qec 'bash deploy/install.sh "$image"' /dev/null
cmp "$fixture/original-env" "$env_file"

cat > "$fixture/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
    'compose version'|'info') ;;
    'compose --project-name trestle-production --env-file .env --file compose.yaml pull') echo pull >> "$fixture/docker-calls" ;;
    'compose --project-name trestle-production --env-file .env --file compose.yaml up -d --wait --wait-timeout 180 app') echo start >> "$fixture/docker-calls" ;;
    'compose --project-name trestle-production --env-file .env --file compose.yaml exec app /app/trestle bootstrap') echo bootstrap >> "$fixture/docker-calls" ;;
    *) exit 1 ;;
esac
EOF
chmod +x "$fixture/bin/docker"
mkdir -p "$fixture/docker-data/trestle"
cp deploy/compose.yaml "$fixture/docker-data/trestle/compose.yaml"
printf 'TRESTLE_IMAGE=%s\nPOSTGRES_PASSWORD=old-credential\n' "$image" > "$fixture/legacy-env"
cp "$fixture/legacy-env" "$fixture/docker-data/trestle/.env"
TRESTLE_RUNTIME=docker XDG_DATA_HOME="$fixture/docker-data" script -qec 'bash deploy/install.sh "$image"' /dev/null
cmp "$fixture/legacy-env" "$fixture/docker-data/trestle/.env"
printf 'pull\nstart\nbootstrap\n' | cmp - "$fixture/docker-calls"

if TRESTLE_DOMAIN='https://trestle.example/path' script -qec 'bash deploy/install.sh "$image"' /dev/null > "$fixture/refusal"; then
    echo "Installer accepted a URL in place of a hostname." >&2
    exit 1
fi
grep -Fq 'without a scheme, port or path' "$fixture/refusal"

export XDG_DATA_HOME="$fixture/managed-data" XDG_CONFIG_HOME="$fixture/managed-config"
export TRESTLE_DOMAIN=Trestle.Example
if fixture_min_port=1024 script -qec 'bash deploy/install.sh "$image"' /dev/null > "$fixture/refusal"; then
    echo "Rootless installer accepted inaccessible HTTPS ports." >&2
    exit 1
fi
grep -Fq 'rootless access to ports 80 and 443' "$fixture/refusal"
test ! -e "$XDG_DATA_HOME/trestle/.env"

rm "$fixture/calls"
script -qec 'bash deploy/install.sh "$image"' /dev/null
env_file="$XDG_DATA_HOME/trestle/.env"
units="$XDG_CONFIG_HOME/containers/systemd/trestle"
grep -Fxq 'TRESTLE_DOMAIN=trestle.example' "$env_file"
grep -Fxq 'Environment=TRESTLE_DOMAIN=trestle.example' "$units/trestle-caddy.container"
grep -Fxq 'Network=trestle-ingress.network' "$units/trestle-app.container"
if grep -q '^PublishPort=' "$units/trestle-app.container" || grep -q '^EnvironmentFile=' "$units/trestle-caddy.container"; then
    echo "Managed Quadlets published the app or attached the database environment to Caddy." >&2
    exit 1
fi
cmp deploy/caddy/Caddyfile "$units/Caddyfile"
printf 'reload\nproxy\nbootstrap\n' | cmp - "$fixture/calls"

# A rerun with a different hostname must preserve the active configuration.
cp "$env_file" "$fixture/managed-env"
if TRESTLE_DOMAIN=other.example script -qec 'bash deploy/install.sh "$image"' /dev/null > "$fixture/refusal"; then
    echo "Installer replaced the managed hostname during a rerun." >&2
    exit 1
fi
grep -Fq 'TRESTLE_DOMAIN differs' "$fixture/refusal"
cmp "$fixture/managed-env" "$env_file"

export TRESTLE_RUNTIME=docker XDG_DATA_HOME="$fixture/managed-docker-data"
cat > "$fixture/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
    'compose version'|'info'|'volume ls --format {{.Name}}') ;;
    'compose --project-name trestle-production --env-file .env --file compose.yaml --file compose.override.yaml config --quiet') echo config >> "$fixture/managed-docker-calls" ;;
    'compose --project-name trestle-production --env-file .env --file compose.yaml --file compose.override.yaml pull') echo pull >> "$fixture/managed-docker-calls" ;;
    'compose --project-name trestle-production --env-file .env --file compose.yaml --file compose.override.yaml up -d --wait --wait-timeout 180 caddy') echo proxy >> "$fixture/managed-docker-calls" ;;
    'compose --project-name trestle-production --env-file .env --file compose.yaml --file compose.override.yaml exec app /app/trestle bootstrap') echo bootstrap >> "$fixture/managed-docker-calls" ;;
    *) exit 1 ;;
esac
EOF
script -qec 'bash deploy/install.sh "$image"' /dev/null
cmp deploy/caddy/Caddyfile "$XDG_DATA_HOME/trestle/Caddyfile"
cmp deploy/caddy/compose.override.yaml "$XDG_DATA_HOME/trestle/compose.override.yaml"
printf 'config\npull\nproxy\nbootstrap\n' | cmp - "$fixture/managed-docker-calls"

cat > "$fixture/bin/sed" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ ${fixture_render_failure:-} == yes && ${!#} == */quadlet/trestle-app.container ]]; then
    printf 'partial unit\n'
    exit 1
fi
exec /usr/bin/sed "$@"
EOF
chmod +x "$fixture/bin/sed"
unset TRESTLE_DOMAIN
export TRESTLE_RUNTIME=podman XDG_DATA_HOME="$fixture/failed-data" XDG_CONFIG_HOME="$fixture/failed-config"
if fixture_render_failure=yes script -qec 'bash deploy/install.sh "$image"' /dev/null > "$fixture/refusal"; then
    echo "Installer ignored a Quadlet render failure." >&2
    exit 1
fi
test ! -e "$XDG_CONFIG_HOME/containers/systemd/trestle/trestle-app.container"
script -qec 'bash deploy/install.sh "$image"' /dev/null
grep -Fxq "Image=$image" "$XDG_CONFIG_HOME/containers/systemd/trestle/trestle-app.container"
