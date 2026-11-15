#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

fixture=$(mktemp -d)
trap 'rm -rf -- "$fixture"' EXIT
export fixture
export PATH="$fixture/bin:$PATH"
mkdir -p "$fixture/bin" "$fixture/deploy"
cp deploy/compose.yaml "$fixture/deploy/compose.yaml"
cp -r deploy/quadlet "$fixture/deploy/quadlet"
cp -r deploy/caddy "$fixture/deploy/caddy"
cat > "$fixture/deploy/install.sh" <<'EOF'
set -euo pipefail
test -s deploy/compose.yaml
test -s deploy/quadlet/trestle-app.container
if [[ ${legacy_bundle:-} != yes ]]; then
    test -s deploy/caddy/Caddyfile
    test -s deploy/caddy/compose.override.yaml
    test -s deploy/caddy/quadlet/trestle-caddy.container
fi
printf '%s\n' "$1" > "$fixture/installed"
EOF
printf 'docker.io/marufsarker/trestle@sha256:%064d\n' 0 > "$fixture/image.txt"
(
    cd "$fixture"
    tar -czf trestle-deploy.tar.gz deploy/install.sh deploy/compose.yaml deploy/quadlet deploy/caddy image.txt
    sha256sum --binary trestle-deploy.tar.gz > trestle-deploy.tar.gz.sha256
)
cat > "$fixture/bin/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
url=${!#}
base=https://github.com/blackstart-labs/trestle/releases
case $url in
    "$base/$release_path/trestle-deploy.tar.gz"|"$base/$release_path/trestle-deploy.tar.gz.sha256") ;;
    *) echo "Unexpected release URL: $url" >&2; exit 1 ;;
esac
cp "$fixture/${url##*/}" "${@: -2:1}"
EOF
chmod +x "$fixture/bin/curl"

release_path=latest/download bash deploy/bootstrap.sh
cmp "$fixture/image.txt" "$fixture/installed"
rm "$fixture/installed"

release_path=download/v1.2.3 bash deploy/bootstrap.sh v1.2.3
cmp "$fixture/image.txt" "$fixture/installed"
rm "$fixture/installed"

printf 'corrupt' >> "$fixture/trestle-deploy.tar.gz"
if release_path=latest/download bash deploy/bootstrap.sh >"$fixture/error" 2>&1; then
    echo "The installer accepted a corrupt deployment bundle." >&2
    exit 1
fi
grep -Fq 'checksum mismatch' "$fixture/error"
test ! -e "$fixture/installed"

(
    cd "$fixture"
    tar -czf trestle-deploy.tar.gz deploy/install.sh deploy/compose.yaml deploy/quadlet image.txt
    sha256sum --binary trestle-deploy.tar.gz > trestle-deploy.tar.gz.sha256
)
legacy_bundle=yes release_path=download/v1.2.3 bash deploy/bootstrap.sh v1.2.3
cmp "$fixture/image.txt" "$fixture/installed"
rm "$fixture/installed"
if TRESTLE_DOMAIN=trestle.example legacy_bundle=yes release_path=download/v1.2.3 bash deploy/bootstrap.sh v1.2.3 >"$fixture/error" 2>&1; then
    echo "The launcher enabled Caddy with an older bundle that has no Caddy files." >&2
    exit 1
fi
grep -Fq 'does not include bundled Caddy' "$fixture/error"
test ! -e "$fixture/installed"
