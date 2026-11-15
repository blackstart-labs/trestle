#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

export TRESTLE_IMAGE=${1:?Pass the Trestle image to test}
export POSTGRES_PASSWORD=ci-database-password
export TRESTLE_DOMAIN=trestle.example
fixture=$(mktemp -d)
suffix=${fixture##*.}
stack=(docker compose --project-name "trestle-caddy-smoke-${suffix,,}"
    --file "$fixture/compose.yaml" --file "$fixture/compose.override.yaml" --file "$fixture/ports.yaml")

cleanup() {
    local status=$?
    if (( status != 0 )); then
        "${stack[@]}" logs || true
    fi
    "${stack[@]}" down -v
    rm -rf -- "$fixture"
}
trap cleanup EXIT

cp deploy/compose.yaml deploy/caddy/compose.override.yaml "$fixture"
sed -e '/admin off/a\	skip_install_trust' -e '/reverse_proxy/i\	tls internal' \
    deploy/caddy/Caddyfile > "$fixture/Caddyfile"
# Keep the test listener on loopback and leave the host HTTPS ports alone.
cat > "$fixture/ports.yaml" <<'EOF'
services:
  caddy:
    ports: !override
      - "127.0.0.1::80"
      - "127.0.0.1::443"
EOF

"${stack[@]}" config --format json | jq -e '
  (.services.app.ports // [] | length) == 0 and
  (.services.caddy.networks | keys) == ["ingress"] and
  (.services.db.networks | keys) == ["default"]
'
"${stack[@]}" up -d --wait --wait-timeout 180 caddy
endpoint=$("${stack[@]}" port caddy 443)
port=${endpoint##*:}
for attempt in {1..15}; do
    if "${stack[@]}" cp caddy:/data/caddy/pki/authorities/local/root.crt "$fixture/root.crt" 2>/dev/null; then break; fi
    sleep 1
done
test -s "$fixture/root.crt"
curl --fail --silent --show-error --noproxy '*' --max-time 8 --retry 5 --retry-all-errors \
    --cacert "$fixture/root.crt" --resolve "$TRESTLE_DOMAIN:$port:127.0.0.1" \
    "https://$TRESTLE_DOMAIN:$port/health/ready" | jq -e '.status == "ok"'
