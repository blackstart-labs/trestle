#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 || ! $1 =~ ^[a-z0-9][a-z0-9./:_-]*@sha256:[a-f0-9]{64}$ ]]; then
    echo "Usage: bash deploy/install.sh registry/repository@sha256:<release image index digest>" >&2
    exit 2
fi
image=$1
domain=${TRESTLE_DOMAIN:-}
domain=${domain,,}
if [[ -n $domain && (${#domain} -gt 253 || ! $domain =~ ^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]([a-z0-9-]{0,61}[a-z0-9])?$) ]]; then
    echo "Set TRESTLE_DOMAIN to a public DNS hostname, without a scheme, port or path." >&2
    exit 2
fi
case "$(uname -s)/$(uname -m)" in
    Linux/x86_64|Linux/aarch64|Linux/riscv64) ;;
    *) echo "The installer requires Linux on AMD64, ARM64, or RISC-V 64." >&2; exit 1 ;;
esac
if [[ ! -t 0 || ! -t 1 ]]; then
    echo "Run the installer in a terminal to use interactive bootstrap." >&2
    exit 1
fi
runtime=${TRESTLE_RUNTIME:-docker}
case $runtime in
    docker)
        docker compose version >/dev/null
        docker info >/dev/null
        volume=trestle-production_postgres-data
        ;;
    podman)
        if (( EUID == 0 )) || [[ $(podman info --format '{{.Host.Security.Rootless}} {{.Host.CgroupsVersion}}') != 'true v2' ]]; then
            echo "Use rootless Podman with cgroup v2 for this installation." >&2
            exit 1
        fi
        systemctl --user show-environment >/dev/null
        if [[ $(loginctl show-user "$(id -u)" --property=Linger --value) != yes ]]; then
            echo "Enable user lingering with loginctl enable-linger \"$(id -un)\" so Trestle starts at boot, then rerun." >&2
            exit 1
        fi
        if [[ -n $domain ]]; then
            min_port=$(sysctl -n net.ipv4.ip_unprivileged_port_start)
            if [[ ! $min_port =~ ^[0-9]+$ ]] || (( min_port > 80 )); then
                echo "Managed HTTPS needs rootless access to ports 80 and 443. Configure the host or use an external HTTPS proxy." >&2
                exit 1
            fi
        fi
        quadlet_dir=${XDG_CONFIG_HOME:-"$HOME/.config"}/containers/systemd/trestle
        volume=trestle-postgres-data
        ;;
    *) echo "TRESTLE_RUNTIME must be docker or podman." >&2; exit 2 ;;
esac

source_dir=$(dirname "$(realpath "${BASH_SOURCE[0]}")")
install_dir=${XDG_DATA_HOME:-"$HOME/.local/share"}/trestle
mkdir -p "$install_dir"
cd "$install_dir"
install_dir=$(pwd -P)
exec 9>.install.lock
flock -n 9 || { echo "A Trestle installation is already running." >&2; exit 1; }

if [[ -f .env ]]; then
    installed_runtime=$(sed -n 's/^TRESTLE_RUNTIME=//p' .env)
    if [[ ${installed_runtime:-docker} != "$runtime" ]] || ! grep -Fxq "TRESTLE_IMAGE=$image" .env; then
        echo "Deployment files differ from this release. Use the upgrade procedure." >&2
        exit 1
    fi
    if [[ $(sed -n 's/^TRESTLE_DOMAIN=//p' .env) != "$domain" ]]; then
        echo "TRESTLE_DOMAIN differs from this installation. Rerun with the original hostname or use the proxy change procedure." >&2
        exit 1
    fi
else
    volumes=$("$runtime" volume ls --format '{{.Name}}')
    if grep -Fxq "$volume" <<< "$volumes"; then
        echo "Found an existing Trestle database volume. Recover its configuration before continuing." >&2
        exit 1
    fi
    password=$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')
    [[ $password =~ ^[a-f0-9]{64}$ ]]
    tmp=$(mktemp .env.XXXXXX)
    printf 'TRESTLE_IMAGE=%s\nTRESTLE_RUNTIME=%s\nPOSTGRES_PASSWORD=%s\n' "$image" "$runtime" "$password" > "$tmp"
    if [[ -n $domain ]]; then
        printf 'TRESTLE_DOMAIN=%s\n' "$domain" >> "$tmp"
    fi
    if [[ $runtime == podman ]]; then
        printf 'DATABASE_URL=postgres://trestle:%s@trestle-db:5432/trestle?sslmode=disable\n' "$password" >> "$tmp"
    fi
    mv "$tmp" .env
fi

install_config() {
    local source=$1 destination=$2 staged
    shift 2
    staged=$(mktemp "$(dirname "$destination")/.trestle.XXXXXX")
    if ! {
        if (( $# )); then
            sed "$@" -- "$source"
        else
            cat -- "$source"
        fi
    } > "$staged"; then
        rm -f "$staged"
        return 1
    fi
    if [[ -e $destination || -L $destination ]]; then
        if ! cmp -s "$staged" "$destination"; then
            rm -f "$staged"
            echo "The installed file differs from this release: $destination. Use the upgrade procedure." >&2
            return 1
        fi
        rm -f "$staged"
    else
        mv "$staged" "$destination"
    fi
}

if [[ $runtime == docker ]]; then
    install_config "$source_dir/compose.yaml" compose.yaml
    # Compose gives exported shell variables precedence over the deployment file.
    unset TRESTLE_IMAGE POSTGRES_PASSWORD TRESTLE_PORT TRESTLE_DOMAIN
    stack=(docker compose --project-name trestle-production --env-file .env --file compose.yaml)
    target=app
    if [[ -n $domain ]]; then
        install_config "$source_dir/caddy/compose.override.yaml" compose.override.yaml
        install_config "$source_dir/caddy/Caddyfile" Caddyfile
        stack+=(--file compose.override.yaml)
        target=caddy
        "${stack[@]}" config --quiet
    fi
    "${stack[@]}" pull
    "${stack[@]}" up -d --wait --wait-timeout 180 "$target"
    "${stack[@]}" exec app /app/trestle bootstrap
else
    mkdir -p "$quadlet_dir"
    if [[ ! -e $quadlet_dir/trestle.env && ! -L $quadlet_dir/trestle.env ]]; then
        ln -s "$install_dir/.env" "$quadlet_dir/trestle.env"
    fi
    if [[ $(readlink "$quadlet_dir/trestle.env") != "$install_dir/.env" ]]; then
        echo "The Quadlet environment file points to a different installation." >&2
        exit 1
    fi
    for unit in "$source_dir"/quadlet/*; do
        edits=(-e "s|@TRESTLE_IMAGE@|$image|g")
        if [[ -n $domain && ${unit##*/} == trestle-app.container ]]; then
            edits+=(-e 's/^PublishPort=.*/Network=trestle-ingress.network/')
        fi
        install_config "$unit" "$quadlet_dir/${unit##*/}" "${edits[@]}"
    done
    target=trestle-app.service
    if [[ -n $domain ]]; then
        install_config "$source_dir/caddy/Caddyfile" "$quadlet_dir/Caddyfile"
        for unit in "$source_dir"/caddy/quadlet/*; do
            install_config "$unit" "$quadlet_dir/${unit##*/}" -e "s|@TRESTLE_DOMAIN@|$domain|g"
        done
        target=trestle-caddy.service
    fi
    systemctl --user daemon-reload
    if ! systemctl --user start "$target"; then
        echo "Inspect startup logs with journalctl --user -u trestle-db -u trestle-migrate -u trestle-app -u trestle-caddy." >&2
        exit 1
    fi
    podman exec --interactive --tty trestle-app /app/trestle bootstrap
fi
if [[ -n $domain ]]; then
    printf '\nCaddy is running for https://%s. Verify HTTPS after DNS and certificate issuance.\n' "$domain"
else
    printf '\nTrestle is ready at the local upstream 127.0.0.1:3000. Configure your HTTPS reverse proxy for that upstream.\n'
fi
printf 'Deployment files: %s\n' "$install_dir"
