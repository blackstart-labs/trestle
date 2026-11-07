<div align="center">

# Trestle

**An open-source, self-hosted productivity platform unifying projects, workflows, task tracking, and knowledge management in a single lightweight binary.**

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![Svelte Version](https://img.shields.io/badge/Svelte-5.57+-FF3E00?style=flat-square&logo=svelte)](https://svelte.dev)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg?style=flat-square)](LICENSE)
[![WCAG 2.2 AA](https://img.shields.io/badge/Accessibility-WCAG_2.2_AA-green.svg?style=flat-square)](docs/design/design-principles.md)
[![Self-Hosted](https://img.shields.io/badge/Deployment-Self--Hosted-purple.svg?style=flat-square)](#-production-self-hosting)
[![Docker Hub](https://img.shields.io/badge/Docker_Hub-marufsarker%2Ftrestle-2496ED?style=flat-square&logo=docker&logoColor=white)](https://hub.docker.com/r/marufsarker/trestle)
[![Docs Live](https://img.shields.io/badge/Docs-tresstle.vercel.app-000000?style=flat-square&logo=vercel&logoColor=white)](https://tresstle.vercel.app)

[Quickstart](#-quickstart-one-command-setup) • [Features](#-features) • [Self-Hosting](#-production-self-hosting) • [Live Documentation](https://tresstle.vercel.app) • [Operator CLI](#-operator-cli) • [REST API](#-rest-api-reference)

---

<p align="center">
  <img src="assets/demo.png" alt="Trestle Application Interface" width="100%" />
</p>

</div>

## 📌 Overview

**Trestle** is built for developers, indie creators, consultants, and teams who want complete data ownership over their everyday work without the weight or telemetry of SaaS alternatives. 

Trestle couples a high-performance **Go (`net/http`) backend** with a reactive **Svelte 5 SPA frontend**, compiling everything into a **single static binary** via `go:embed`. At runtime, it requires only a PostgreSQL 18 database: no Node.js runtime or asset bundles to host separately. Optional project integrations send outbound notifications to Telegram, Slack, and Discord; core work requires no external service.

```
┌─────────────────────────────────────────────────────────────────┐
│                      Trestle Architecture                       │
│                                                                 │
│  Browser / Client ───HTTP (Strict CSP)───► Go HTTP Monolith     │
│                                             │ (embeds Svelte 5) │
│                                             ▼                   │
│                                    PostgreSQL 18 (pgx/v5)       │
└─────────────────────────────────────────────────────────────────┘
```

---

## ✨ Features

### 🏢 Multi-Tenant Workspaces & Identity
* **Workspace Isolation:** Clean tenant boundaries with role-based permissions (`Owner`, `Admin`, `Member`).
* **Ownership Lifecycle:** Transfer workspace ownership with re-authentication safeguards and last-owner preservation.
* **Hardened Local Identity:** Argon2id password hashing, constant-time verification, session tokens, and in-memory brute-force rate limiting.

### 📋 Projects & Canonical Tasks
* **Atomic Human Keys:** Tasks automatically receive project-scoped human-readable identifiers (e.g. `WEB-42`, `CORE-101`).
* **Dual Views:** Switch seamlessly between high-density **List view** and interactive **Kanban Board**.
* **Project Archiving:** Archive inactive projects while maintaining data integrity and historical references.

### 🔄 Custom Workflow Statuses
* **Canonical Category Mapping:** Define custom workflow statuses mapped to five fundamental categories: `Backlog`, `Unstarted`, `Started`, `Completed`, and `Cancelled`.
* **Safe Transitions & Atomic Remap:** Reorder status columns freely; deleting an in-use status automatically prompts for a target remap status so tasks are never orphaned.

### 🤝 Deep Collaboration
* **Hierarchical Subtasks:** Break work down into manageable subtask checklists with instant keyboard-friendly toggling.
* **Multi-Assignees & Colored Labels:** Tag tasks with customizable labels and assign one or more team members.
* **Threaded Comments & Audit Log:** Write Markdown comments and inspect an immutable audit activity stream detailing all status transitions and property updates.

### 🎨 Executive-Tier UI & Accessibility
* **Themeable Workbench:** Dark-first aesthetic powered by modern OKLCH tokens, with 5 accent presets (*Iris, Sky, Tide, Moss, Ember*).
* **Apple-Fluid Motion:** Spring-based motion (`web/src/lib/motion.ts`), symmetric exits, and view transitions.
* **WCAG 2.2 AA Compliant:** 44px minimum touch targets, visible focus rings, full keyboard shortcuts, and contrast verified automatically across light and dark modes.

### 🔒 Enterprise Security by Default
* **Strict Content Security Policy (CSP):** `default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self'; ...` prevents XSS and blocks third-party CDNs.
* **Anti-CSRF Verification:** State-mutating requests require cryptographic `X-CSRF-Token` headers.
* **Zero Secret Leakage:** Database connection strings, stack traces, and internal errors are never exposed over the API or in server diagnostics.

---

## 🚀 Quickstart (One-Command Setup)

Run the entire application locally with a single command across **Linux, macOS, and Windows**. The runner automatically installs matching local tools (Go/Node) into an isolated `.tools/` directory, checks ports, launches PostgreSQL, builds the frontend, applies migrations, bootstraps the initial admin account, and launches Trestle.

### Linux / macOS / WSL
```bash
git clone https://github.com/blackstart-labs/trestle.git && cd trestle && ./run.sh
```

### Windows (PowerShell)
```powershell
git clone https://github.com/blackstart-labs/trestle.git; cd trestle; .\run.ps1
```

### Windows (Command Prompt)
```cmd
git clone https://github.com/blackstart-labs/trestle.git && cd trestle && run.cmd
```

### What the runner handles automatically:
1. **Tool Verification:** Detects Go, Node.js, and container runtimes (Docker/Podman). If Go or Node are missing, it installs isolated copies into `.tools/` without touching system packages.
2. **Port Conflict Resolution:** If port `5432` (PostgreSQL) or `3000` (web app) is busy, it automatically binds to the next available free port without colliding.
3. **Database Container:** Starts PostgreSQL 18 via Docker Compose and waits for readiness.
4. **Production Build:** Installs locked frontend dependencies when `web/package.json` or `web/package-lock.json` changes, then rebuilds `web/dist`. Unchanged dependencies are reused on later starts.
5. **Database Migrations:** Applies all database schema migrations.
6. **Admin Bootstrap:** Provisions the initial operator (`admin` / `adminpassword123`). Pass `--interactive` to enter custom credentials.
7. **Application Launch:** Serves the app at `http://127.0.0.1:3000` (or auto-assigned port).

To stop the background database service:
```bash
./run.sh stop
```

---

## 🛠 Manual Development Setup

If you prefer to run each step manually:

```bash
# 1. Install frontend dependencies and build assets
npm ci --prefix web --ignore-scripts
npm run build --prefix web

# 2. Launch PostgreSQL 18
export POSTGRES_PASSWORD="$(openssl rand -hex 24)"
export DATABASE_URL="postgres://trestle:${POSTGRES_PASSWORD}@127.0.0.1:5432/trestle"
docker compose up -d --wait db

# 3. Apply migrations and initialize bootstrap
go run . migrate
go run . bootstrap

# 4. Start the Trestle server
go run .
```

Open `http://127.0.0.1:3000/`. For hot-reloading frontend development, start the backend, then run `npm run dev --prefix web` (Vite proxies API calls to port 3000).

---

## 🌐 Production Self-Hosting

Trestle is designed for simple, reliable production deployment.

### Pull from Docker Hub

The official production container image is hosted on Docker Hub:

```bash
# Pull latest stable release
docker pull marufsarker/trestle:latest

# Or pull a specific release tag
docker pull marufsarker/trestle:0.1.0
```

### Option A: Docker Compose (Recommended)

Create a `docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  db:
    image: postgres:18-bookworm
    restart: always
    environment:
      POSTGRES_DB: trestle
      POSTGRES_USER: trestle
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?Database password required}
    volumes:
      - trestle_pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U trestle -d trestle"]
      interval: 5s
      timeout: 3s
      retries: 10

  app:
    image: marufsarker/trestle:latest
    restart: always
    depends_on:
      db:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://trestle:${POSTGRES_PASSWORD}@db:5432/trestle
      TRESTLE_BIND_ADDR: 0.0.0.0:3000
      TRESTLE_DEV_MODE: "false"
      TRESTLE_MAX_DB_CONNECTIONS: 10
      TRESTLE_DB_TIMEOUT_MS: 3000
    ports:
      - "127.0.0.1:3000:3000"

volumes:
  trestle_pgdata:
```

Launch with:
```bash
docker compose -f docker-compose.prod.yml up -d
```

### Option B: Standalone Binary (Systemd)

Compile the static binary:
```bash
npm ci --prefix web && npm run build --prefix web
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o trestle .
```

Create `/etc/systemd/system/trestle.service`:

```ini
[Unit]
Description=Trestle Workspace Server
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=trestle
Group=trestle
WorkingDirectory=/opt/trestle
EnvironmentFile=/etc/trestle/trestle.env
ExecStart=/opt/trestle/trestle
Restart=always
RestartSec=5s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

Reload and enable:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now trestle
```

---

### Reverse Proxy Configuration

#### Caddy (Automatic HTTPS)
```caddy
trestle.yourdomain.com {
    reverse_proxy 127.0.0.1:3000
}
```

#### Nginx
```nginx
server {
    listen 80;
    server_name trestle.yourdomain.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name trestle.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/trestle.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/trestle.yourdomain.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## ⚙️ Configuration Reference

Trestle is configured through environment variables:

| Setting | Default | Description |
| --- | --- | --- |
| `DATABASE_URL` | *Required* | PostgreSQL connection string (`postgres://user:pass@host:5432/db`). |
| `TRESTLE_BIND_ADDR` | `127.0.0.1:3000` | IP socket address and port to bind the HTTP server. |
| `TRESTLE_MAX_DB_CONNECTIONS` | `5` | Maximum active database connections (range: 1–32). |
| `TRESTLE_DB_TIMEOUT_MS` | `2000` | Database pool acquisition and query readiness timeout (ms). |
| `TRESTLE_DEV_MODE` | `false` | When `true`, relaxes `Secure` cookie flag for loopback HTTP development. |
| `TRESTLE_ARGON2_M_COST` | `65536` | Argon2id memory cost in KiB (default: 64 MiB). |
| `TRESTLE_ARGON2_T_COST` | `2` | Argon2id iterations count. |
| `TRESTLE_ARGON2_PARALLELISM` | `1` | Argon2id threads / lanes parallelism degree. |

---

## 💻 Operator CLI

The `trestle` binary provides operational CLI subcommands:

```bash
# Apply pending database schema migrations
./trestle migrate

# Run one-time initial operator account and workspace setup
./trestle bootstrap

# Provision a user with a workspace role (owner, admin, member)
./trestle adduser

# Reset a user's password securely via interactive prompt
./trestle resetpassword

# Disable or re-enable user accounts
./trestle disableuser
./trestle enableuser
```

---

## 🔌 REST API Reference

All product endpoints are served under `/api/v1/`. Authenticated requests require session cookies, and mutating requests enforce `X-CSRF-Token` headers.

| Endpoint | Method | Description |
| --- | --- | --- |
| `/health/live` | `GET` | Process liveness check (`200 {"status":"ok"}`). |
| `/health/ready` | `GET` | PostgreSQL connectivity probe (`200 {"status":"ok"}`). |
| `/api/v1/session` | `POST` / `DELETE` | Authenticate with Argon2id / Invalidate session. |
| `/api/v1/me` | `GET` | Retrieve the authenticated user profile. |
| `/api/v1/workspaces` | `GET` | List accessible workspaces for the user. |
| `/api/v1/workspaces/{id}/members` | `GET` / `POST` | List and add workspace members. |
| `/api/v1/workspaces/{id}/members/{userId}` | `PATCH` / `DELETE` | Update member role or remove member. |
| `/api/v1/workspaces/{id}/transfer-ownership` | `POST` | Transfer workspace ownership with re-authentication. |
| `/api/v1/workspaces/{id}/projects` | `GET` / `POST` | List and create workspace projects. |
| `/api/v1/workspaces/{id}/projects/{pId}` | `GET` | Retrieve project details. |
| `/api/v1/workspaces/{id}/projects/{pId}/archive` | `POST` | Archive project. |
| `/api/v1/workspaces/{id}/projects/{pId}/tasks` | `GET` / `POST` | List tasks (with filter & pagination) and create tasks. |
| `/api/v1/.../tasks/{taskId}` | `GET` / `PATCH` / `DELETE` | Retrieve, update properties, or delete task. |
| `/api/v1/.../statuses` | `GET` / `POST` | List statuses and create custom status. |
| `/api/v1/.../statuses/positions` | `PUT` | Reorder custom status positions. |
| `/api/v1/.../statuses/{sId}` | `PATCH` / `DELETE` | Edit status or delete with atomic task remap. |
| `/api/v1/.../tasks/{taskId}/labels` | `POST` / `DELETE` | Attach or detach labels. |
| `/api/v1/.../tasks/{taskId}/assignees` | `POST` / `DELETE` | Add or remove assignees. |
| `/api/v1/.../tasks/{taskId}/subtasks` | `GET` / `POST` | List and create checklist subtasks. |
| `/api/v1/.../tasks/{taskId}/comments` | `GET` / `POST` | List and create threaded comments. |
| `/api/v1/.../tasks/{taskId}/activity` | `GET` | Retrieve task activity and audit stream. |

---

## 📚 Documentation

<p align="center">
  <a href="https://tresstle.vercel.app" target="_blank" rel="noopener noreferrer">
    <img src="assets/docs.webp" alt="Trestle Documentation Portal" width="100%" />
  </a>
</p>

Visit the live documentation and interactive user manual at **[tresstle.vercel.app](https://tresstle.vercel.app)**.

* **[Live Documentation Portal](https://tresstle.vercel.app)**: Complete interactive user manual with global search (`⌘K`), feature walkthroughs, and responsive showcases.
* **[Documentation Source Code](docs-site/)**: Svelte 5 + Vite 8 frontend project powering the documentation site.
* **[Product Vision & Requirements](docs/product/product-requirements.md)**: Product purpose, user personas, and scope.
* **[Architecture Overview](docs/architecture.md)**: Modular monolith structure, data flow, and boundaries.
* **[Design Tokens & Principles](docs/design/design-tokens.md)**: Color science, typography, motion curves, and components.
* **[Threat Model & Security](docs/security/threat-model.md)**: Assets, threats, safeguards, and permission matrix.
* **[Operations & Troubleshooting](docs/operations.md)**: Production deployment, migrations, backups, and disaster recovery.

---

## 🧪 Verification & Checks

Run the automated test suite from repository root:

```bash
# Frontend verification (from web/)
npm run lint:design --prefix web    # Design system tokens and WCAG guard
npm run check --prefix web          # Svelte 5 and TypeScript check (0 errors)
npm test --prefix web               # Vitest component unit tests
npm run format:check --prefix web   # Prettier code style
npm run build --prefix web          # Production bundle build

# Backend verification
gofmt -l internal main.go migrations # Go format check
go vet ./internal/... .             # Go vet diagnostics
go test ./internal/...              # Go tests (runs DB tests when DATABASE_URL is set)
```

---

## 📄 License

Trestle is licensed under the [GNU Affero General Public License v3 (AGPL-3.0)](LICENSE). Network deployments of modified versions must make their corresponding source code available.
