<div align="center">

# Trestle

**An open-source, self-hosted productivity platform unifying projects, workflows, task tracking, and knowledge management in a single lightweight binary.**

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![Svelte Version](https://img.shields.io/badge/Svelte-5.57+-FF3E00?style=flat-square&logo=svelte)](https://svelte.dev)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg?style=flat-square)](LICENSE)
[![Docker Hub](https://img.shields.io/badge/Docker_Hub-marufsarker%2Ftrestle-2496ED?style=flat-square&logo=docker&logoColor=white)](https://hub.docker.com/r/marufsarker/trestle)
[![Docs Live](https://img.shields.io/badge/Docs-tresstle.vercel.app-000000?style=flat-square&logo=vercel&logoColor=white)](https://tresstle.vercel.app)

[Features](#-features) • [Quickstart](#-quickstart) • [Production Deployment](#-production-deployment) • [Configuration](#-configuration) • [Documentation](https://tresstle.vercel.app) • [Star History](#star-history)

---

<p align="center">
  <img src="assets/demo.png" alt="Trestle Application Interface" width="100%" />
</p>

</div>

## 📌 Overview

**Trestle** is built for developers, indie creators, and teams who want complete data ownership over their everyday work without SaaS bloat or telemetry.

It couples a high-performance **Go (`net/http`) backend** with a reactive **Svelte 5 SPA frontend**, compiled into a **single static binary** via `go:embed`. At runtime, it only requires a PostgreSQL database — zero Node.js runtime, zero CDN dependencies, and strict Content Security Policy.

---

## ✨ Features

- **🏢 Multi-Tenant Workspaces:** Isolated workspaces with role-based permissions (`Owner`, `Admin`, `Member`), custom workflow pipelines, and seamless switching.
- **📋 Canonical Projects & Tasks:** Atomic project-scoped keys (`API-1`, `WEB-42`), custom priorities, due dates, estimates, and Markdown task descriptions.
- **🔄 Dual Views (List & Kanban):** Switch instantly between high-density tabular list views and visual drag-and-drop Kanban boards with inline status editing.
- **⏱️ Live Task Time Tracking:** Persistent live stopwatch that survives browser restarts, header readout, timesheet logging, and board workload badges.
- **🗺️ Milestone Roadmaps:** Interactive Gantt timelines, milestone completion math, and progress tracking rails.
- **🤖 Chat & Webhook Alerts:** Real-time task notification cards sent to Telegram (with forum topic ID support), Discord, and Slack.
- **📦 Complete Data Portability:** 1-click full JSON export for workspaces and individual projects for safe backups and zero lock-in.
- **🎨 Themeable & Accessible:** Dark and light modes with 5 accent presets, custom avatars, spring motion, and WCAG 2.2 AA compliance.

---

## 🚀 Quickstart

Run Trestle locally with a single command. The runner automatically verifies prerequisites, checks ports, launches PostgreSQL, applies migrations, provisions the admin account, and starts the app:

### Linux / macOS / WSL
```bash
git clone https://github.com/blackstart-labs/trestle.git && cd trestle && ./run.sh
```

### Windows (PowerShell)
```powershell
git clone https://github.com/blackstart-labs/trestle.git; cd trestle; .\run.ps1
```

Access the app at **`http://127.0.0.1:3000`**. The local runner creates an
operator account named `admin` with a random password and prints it once; copy
it from the terminal output and change it after signing in. Run
`./run.sh --interactive` to choose your own credentials instead.

To stop the background database:
```bash
./run.sh stop
```

---

## 🌐 Production Deployment

The repository ships `compose.prod.yaml`, a VPS-ready stack that runs PostgreSQL,
applies migrations, then starts Trestle. The database is never published, and
Trestle binds to loopback so a reverse proxy can terminate TLS.

```bash
cp .env.production.example .env
# Replace the placeholder password in POSTGRES_PASSWORD and DATABASE_URL.
docker compose -f compose.prod.yaml up -d --build
docker compose -f compose.prod.yaml run --rm app bootstrap
```

Put Caddy, nginx or Traefik in front of `127.0.0.1:3000` and give it a public
HTTPS hostname. Outside `TRESTLE_DEV_MODE`, cookies are `Secure`, so plain HTTP
on a public address will not keep you signed in.

```caddyfile
trestle.example.com {
    reverse_proxy 127.0.0.1:3000
}
```

### ARM servers and Raspberry Pi

The pinned base images are multi-arch. Build natively on the host, or cross-build
from an amd64 machine with buildx and QEMU:

```bash
docker buildx build --platform linux/arm64,linux/amd64 \
  -t your-registry/trestle:latest --push .
```

Then set `TRESTLE_IMAGE_TAG` in `.env` and drop the `build:` key from your
Compose file, or keep building on the server itself. The prebuilt Hub image
`marufsarker/trestle:latest` is also usable.

See [docs/operations.md](docs/operations.md) for the full environment table,
health probes, backups and troubleshooting.

---

## ⚙️ Configuration

Configure Trestle via environment variables:

| Setting | Default | Description |
| --- | --- | --- |
| `DATABASE_URL` | *Required* | PostgreSQL connection URL (`postgres://user:pass@host:5432/db`). |
| `TRESTLE_BIND_ADDR` | `127.0.0.1:3000` | Socket address and port to bind the HTTP server. |
| `TRESTLE_MAX_DB_CONNECTIONS` | `5` | Maximum active database connections (range: 1–32). |
| `TRESTLE_DB_TIMEOUT_MS` | `2000` | Database pool query timeout in milliseconds. |
| `TRESTLE_DEV_MODE` | `false` | When `true`, relaxes `Secure` cookie flag for loopback HTTP testing only. |
| `SESSION_TTL_HOURS` | `336` | Absolute session lifetime in hours (1–8760); the idle limit stays 12 hours. |

---

## 💻 Operator CLI

The static binary provides operator CLI commands:

```bash
./trestle migrate        # Apply database schema migrations
./trestle bootstrap      # Run initial admin account setup
./trestle adduser        # Provision a new user with a workspace role
./trestle resetpassword  # Securely reset a user's password
./trestle disableuser    # Deactivate a user account
./trestle enableuser     # Re-enable a user account
```

---

## 📚 Documentation

Visit the interactive user manual at **[tresstle.vercel.app](https://tresstle.vercel.app)** for in-depth walkthroughs, shortcuts (`⌘K`), and guides:

- **[Architecture & Design](docs/architecture.md)** — Monolith structure, boundaries, and data flow.
- **[REST API Reference](docs/api.md)** — Complete endpoint specifications, CSRF defense, and envelopes.
- **[Database Schema](docs/database.md)** — PostgreSQL tables, indices, and migration lifecycles.
- **[Operations & Backups](docs/operations.md)** — Reverse proxies (Caddy, Nginx), backups, and recovery.
- **[Design System & Tokens](docs/design/design-tokens.md)** — Semantic tokens, motion curves, and accessibility.

---

## Star History

<a href="https://www.star-history.com/?repos=blackstart-labs%2Ftrestle&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=blackstart-labs/trestle&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=blackstart-labs/trestle&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=blackstart-labs/trestle&type=date&legend=top-left" />
 </picture>
</a>

---

## 📄 License

Trestle is licensed under the [GNU Affero General Public License v3 (AGPL-3.0)](LICENSE).
