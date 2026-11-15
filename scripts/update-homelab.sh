#!/usr/bin/env bash
# ==============================================================================
# Trestle Repository Helper: Build & Update Homelab
# Location: scripts/update-homelab.sh
#
# Delegates directly to the homelab update orchestrator:
# /home/maruf/homelab/scripts/update-trestle.sh
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
HOMELAB_SCRIPT="/home/maruf/homelab/scripts/update-trestle.sh"

if [[ ! -f "${HOMELAB_SCRIPT}" ]]; then
    echo "[!] Homelab update script not found at ${HOMELAB_SCRIPT}" >&2
    exit 1
fi

export TRESTLE_REPO_DIR="${REPO_DIR}"
exec "${HOMELAB_SCRIPT}" "$@"

