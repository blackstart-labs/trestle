#!/usr/bin/env bash
set -euo pipefail
umask 077

base=https://github.com/blackstart-labs/trestle/releases
case $# in
    0) release=$base/latest/download ;;
    1)
        if [[ ! $1 =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            echo "Use a release tag in the form v1.2.3." >&2
            exit 2
        fi
        release=$base/download/$1
        ;;
    *) echo "Pass at most one release tag." >&2; exit 2 ;;
esac

tmp=$(mktemp -d)
trap 'rm -rf -- "$tmp"' EXIT
cd "$tmp"

archive=trestle-deploy.tar.gz
curl_options=(--fail --silent --show-error --location --proto '=https' --proto-redir '=https' --max-time 60)
curl "${curl_options[@]}" --output "$archive" "$release/$archive"
curl "${curl_options[@]}" --output "$archive.sha256" "$release/$archive.sha256"
read -r checksum filename < "$archive.sha256"
if [[ ! $checksum =~ ^[a-f0-9]{64}$ || $filename != "*$archive" ]]; then
    echo "Invalid checksum file for the Trestle deployment bundle." >&2
    exit 1
fi
printf '%s *%s\n' "$checksum" "$archive" | sha256sum --check --status || {
    echo "Trestle deployment bundle checksum mismatch. Download it again." >&2
    exit 1
}

tar -xzf "$archive" deploy/install.sh deploy/compose.yaml deploy/quadlet image.txt
if tar -tzf "$archive" deploy/caddy >/dev/null 2>&1; then
    tar -xzf "$archive" deploy/caddy
elif [[ -n ${TRESTLE_DOMAIN:-} ]]; then
    echo "This release does not include bundled Caddy. Use an external HTTPS proxy or select a newer release." >&2
    exit 1
fi
bash deploy/install.sh "$(<image.txt)"
