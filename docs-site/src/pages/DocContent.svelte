<script lang="ts">
  import CodeBlock from "../components/CodeBlock.svelte";
  import Callout from "../components/Callout.svelte";

  interface Props {
    sectionId: string;
  }

  let { sectionId }: Props = $props();
</script>

<div class="prose">
  {#if sectionId === "overview"}
    <h1>Overview & Architecture</h1>
    <p class="lead">Trestle is a self-hosted workspace uniting projects, tasks, workflows, time tracking, and knowledge in a single, high-performance static binary.</p>

    <Callout type="tip" title="DATA OWNERSHIP">
      Trestle was created for teams and creators who need a modern Linear/Notion-caliber tool without SaaS lock-in, vendor telemetry, or external network dependencies.
    </Callout>

    <h2>Architectural Tenets</h2>
    <ul>
      <li><strong>Modular Monolith:</strong> A single Go (1.27+) HTTP server handles the REST API, session authentication, and database migrations.</li>
      <li><strong>Embedded Frontend:</strong> The reactive Svelte 5 single-page application is compiled into <code>web/dist</code> and baked directly into the Go executable using <code>go:embed</code>.</li>
      <li><strong>PostgreSQL 18:</strong> Leverages native PostgreSQL connection pooling (via <code>pgx/v5</code>) and strict relational constraints.</li>
      <li><strong>Zero External Runtime Requests:</strong> Self-hosted Mona Sans font, inline SVG icons, and a strict Content Security Policy (CSP) blocking external domains.</li>
    </ul>

    <h2>Component Topology</h2>
    <CodeBlock lang="text" code={`Browser Client (SPA)
        │
   HTTP / HTTPS (Strict CSP: default-src 'none')
        │
┌───────▼──────────────────────────────────────────────┐
│  Trestle Server (Single Go Executable)              │
│                                                      │
│  ├─ net/http ServeMux Router                         │
│  ├─ Argon2id Hasher & In-Memory Rate Limiter         │
│  ├─ Session & CSRF Middleware                        │
│  ├─ Embedded Svelte 5 Assets (/assets/*, index.html) │
│  └─ Database Migrator & Pool (pgx/v5)                │
└───────────────────────┬──────────────────────────────┘
                        │
                  PostgreSQL 18`} />

  {:else if sectionId === "quickstart"}
    <h1>Quickstart (One-Command Setup)</h1>
    <p class="lead">Launch the entire Trestle application with a single command across Linux, macOS, WSL, and Windows.</p>

    <Callout type="note" title="AUTOMATIC PREREQUISITES">
      The runner script automatically installs local copies of Go and Node.js into <code>.tools/</code> if your machine doesn't have them globally installed.
    </Callout>

    <h2>Linux / macOS / WSL</h2>
    <CodeBlock lang="bash" code={`git clone https://github.com/blackstart-labs/trestle.git && cd trestle && ./run.sh`} />

    <h2>Windows (PowerShell)</h2>
    <CodeBlock lang="powershell" code={`git clone https://github.com/blackstart-labs/trestle.git; cd trestle; .\\run.ps1`} />

    <h2>Windows (Command Prompt)</h2>
    <CodeBlock lang="cmd" code={`git clone https://github.com/blackstart-labs/trestle.git && cd trestle && run.cmd`} />

    <h2>What Happens Behind the Scenes:</h2>
    <ol>
      <li><strong>Port Auto-Discovery:</strong> Detects if ports <code>5432</code> (PostgreSQL) or <code>3000</code> (Web) are occupied, and dynamically assigns free ports.</li>
      <li><strong>Container Launch:</strong> Launches PostgreSQL 18 via Docker Compose and verifies database readiness.</li>
      <li><strong>Production Build:</strong> Builds the Svelte 5 frontend into <code>web/dist</code> embedded into Go.</li>
      <li><strong>Database Migration:</strong> Applies all forward-only migrations.</li>
      <li><strong>Admin Bootstrap:</strong> Creates default credentials: <code>admin</code> / <code>adminpassword123</code>.</li>
    </ol>

  {:else if sectionId === "self-hosting"}
    <h1>Production Self-Hosting</h1>
    <p class="lead">Deploy Trestle with Docker Hub, Docker Compose, standalone binaries, or systemd behind standard reverse proxies.</p>

    <h2>Docker Quickstart</h2>
    <p>Pull and inspect the official container image from Docker Hub:</p>
    <CodeBlock lang="bash" code={`# Pull the image
docker pull marufsarker/trestle:latest

# Or pull a specific release tag
docker pull marufsarker/trestle:0.1.0

# Print CLI help
docker run --rm marufsarker/trestle:latest --help`} />

    <h2>Option A: Docker Compose (Recommended)</h2>
    <p>Create a <code>docker-compose.yml</code> file:</p>
    <CodeBlock lang="yaml" code={`version: '3.8'

services:
  db:
    image: postgres:18-bookworm
    restart: always
    environment:
      POSTGRES_DB: trestle
      POSTGRES_USER: trestle
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD:?Password required}
    volumes:
      - pgdata:/var/lib/postgresql/data
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
      DATABASE_URL: postgres://trestle:\${POSTGRES_PASSWORD}@db:5432/trestle
      TRESTLE_BIND_ADDR: 0.0.0.0:3000
      TRESTLE_DEV_MODE: "false"
    ports:
      - "127.0.0.1:3000:3000"

volumes:
  pgdata:`} />

    <h2>Option B: Standalone Binary (Systemd)</h2>
    <p>Compile the binary:</p>
    <CodeBlock lang="bash" code={`npm ci --prefix web && npm run build --prefix web
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o trestle .`} />

    <p>Create <code>/etc/systemd/system/trestle.service</code>:</p>
    <CodeBlock lang="ini" code={`[Unit]
Description=Trestle Application Server
After=network.target postgresql.service

[Service]
Type=simple
User=trestle
WorkingDirectory=/opt/trestle
EnvironmentFile=/etc/trestle/trestle.env
ExecStart=/opt/trestle/trestle
Restart=always

[Install]
WantedBy=multi-user.target`} />

    <h2>Reverse Proxy Configuration</h2>
    <h3>Caddy (Automatic HTTPS)</h3>
    <CodeBlock lang="caddy" code={`trestle.yourdomain.com {
    reverse_proxy 127.0.0.1:3000
}`} />

    <h3>Nginx</h3>
    <CodeBlock lang="nginx" code={`server {
    listen 443 ssl http2;
    server_name trestle.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/trestle/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/trestle/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}`} />

  {:else if sectionId === "workspaces"}
    <h1>Workspaces & Tenancy</h1>
    <p class="lead">Trestle provides strict tenant isolation through multi-tenant workspaces, member roles, and ownership lifecycle guards.</p>

    <h2>Member Roles & Permissions</h2>
    <table>
      <thead>
        <tr><th>Role</th><th>Description</th><th>Permitted Actions</th></tr>
      </thead>
      <tbody>
        <tr><td><strong>Owner</strong></td><td>Primary workspace custodian</td><td>Everything, including ownership transfer, workspace deletion, and billing</td></tr>
        <tr><td><strong>Admin</strong></td><td>Team manager</td><td>Add/remove members, manage custom statuses, archive projects</td></tr>
        <tr><td><strong>Member</strong></td><td>Active contributor</td><td>Create/edit tasks, assign people, comment, manage subtasks</td></tr>
      </tbody>
    </table>

    <Callout type="warning" title="OWNERSHIP TRANSFER">
      Transferring workspace ownership requires entering the current owner's password again. The system enforces concurrency guards to guarantee a workspace never has zero owners.
    </Callout>

  {:else if sectionId === "projects-tasks"}
    <h1>Projects & Canonical Tasks</h1>
    <p class="lead">Manage projects with human-friendly atomic keys (&lt;KEY&gt;-&lt;NUM&gt;) and switch between high-density List and Kanban Board views.</p>

    <h2>Human Keys (e.g. CORE-101)</h2>
    <p>Every task is assigned an immutable, atomic human key composed of the project key and an incrementing sequence number. These keys remain consistent across list views, boards, URLs, and search.</p>

    <h2>Dual Viewing Modes</h2>
    <ul>
      <li><strong>List View:</strong> High-density table featuring multi-column sorting, overdue indicators, category filters, and quick inline editing.</li>
      <li><strong>Kanban Board:</strong> Columns grouped by workflow statuses, supporting both fluid drag-and-drop and accessible keyboard-driven card reordering.</li>
    </ul>

  {:else if sectionId === "custom-statuses"}
    <h1>Custom Workflow Statuses</h1>
    <p class="lead">Define bespoke project workflows while maintaining mathematical integrity across reporting.</p>

    <h2>The 5 Canonical Categories</h2>
    <p>Every custom status must map to exactly one of the five canonical categories:</p>
    <table>
      <thead>
        <tr><th>Category</th><th>Semantic Purpose</th><th>Default Color</th></tr>
      </thead>
      <tbody>
        <tr><td><code>backlog</code></td><td>Ideas, future requests, unprioritized items</td><td>Slate / Muted</td></tr>
        <tr><td><code>unstarted</code></td><td>Ready for work, queued for current sprint</td><td>Blue / Info</td></tr>
        <tr><td><code>started</code></td><td>Actively in progress or under review</td><td>Amber / Warning</td></tr>
        <tr><td><code>completed</code></td><td>Finished, verified, or deployed</td><td>Emerald / Success</td></tr>
        <tr><td><code>cancelled</code></td><td>Discarded, duplicate, or rejected</td><td>Zinc / Dim</td></tr>
      </tbody>
    </table>

    <Callout type="tip" title="SAFE DELETIONS">
      Deleting a status with assigned tasks requires selecting a <code>remap_to</code> target status so tasks are automatically and atomically migrated.
    </Callout>

  {:else if sectionId === "collaboration"}
    <h1>Task Collaboration</h1>
    <p class="lead">Granular task tracking with subtasks, assignees, labels, threaded Markdown discussions, and immutable audit logs.</p>

    <h2>Features in Task Detail Drawer:</h2>
    <ul>
      <li><strong>Checklist Subtasks:</strong> Instant keyboard-accessible completion toggles.</li>
      <li><strong>Multi-Assignees:</strong> Assign one or multiple workspace members with initials avatars.</li>
      <li><strong>Color-Coded Labels:</strong> Categorize issues with project-scoped label tags.</li>
      <li><strong>Markdown Comments:</strong> Formatted commentary with author-only deletion rights.</li>
      <li><strong>Activity Audit Stream:</strong> Complete history tracking who changed status, priority, or assignees, with timestamps.</li>
    </ul>

  {:else if sectionId === "operator-cli"}
    <h1>Operator CLI Reference</h1>
    <p class="lead">Manage your Trestle instance directly using terminal subcommands built into the Go executable.</p>

    <h2>Subcommands Overview</h2>
    <CodeBlock lang="bash" code={`# Run database schema migrations
./trestle migrate

# Run initial instance bootstrap (operator account + primary workspace)
./trestle bootstrap

# Provision a user with a workspace role
./trestle adduser

# Reset a user's password securely
./trestle resetpassword

# Disable or enable a user account
./trestle disableuser
./trestle enableuser`} />

  {:else if sectionId === "api-reference"}
    <h1>REST API Reference (v1)</h1>
    <p class="lead">Comprehensive HTTP endpoint reference for programmatic access and integrations.</p>

    <h2>Authentication & Session</h2>
    <table>
      <thead><tr><th>Method</th><th>Endpoint</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td><span class="badge-method badge-post">POST</span></td><td><code>/api/v1/session</code></td><td>Authenticate user (Argon2id) & issue cookie</td></tr>
        <tr><td><span class="badge-method badge-delete">DELETE</span></td><td><code>/api/v1/session</code></td><td>Invalidate current session</td></tr>
        <tr><td><span class="badge-method badge-get">GET</span></td><td><code>/api/v1/me</code></td><td>Retrieve current user profile & roles</td></tr>
      </tbody>
    </table>

    <h2>Workspaces & Projects</h2>
    <table>
      <thead><tr><th>Method</th><th>Endpoint</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td><span class="badge-method badge-get">GET</span></td><td><code>/api/v1/workspaces</code></td><td>List accessible workspaces</td></tr>
        <tr><td><span class="badge-method badge-get">GET</span></td><td><code>/api/v1/workspaces/{id}/projects</code></td><td>List active projects</td></tr>
        <tr><td><span class="badge-method badge-post">POST</span></td><td><code>/api/v1/workspaces/{id}/projects</code></td><td>Create project (requires admin/owner)</td></tr>
        <tr><td><span class="badge-method badge-post">POST</span></td><td><code>/api/v1/.../projects/{id}/archive</code></td><td>Archive project</td></tr>
      </tbody>
    </table>

    <h2>Tasks & Workflows</h2>
    <table>
      <thead><tr><th>Method</th><th>Endpoint</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td><span class="badge-method badge-get">GET</span></td><td><code>/api/v1/.../projects/{id}/tasks</code></td><td>List tasks with filters & pagination</td></tr>
        <tr><td><span class="badge-method badge-post">POST</span></td><td><code>/api/v1/.../projects/{id}/tasks</code></td><td>Create new task</td></tr>
        <tr><td><span class="badge-method badge-patch">PATCH</span></td><td><code>/api/v1/.../tasks/{id}</code></td><td>Update task title, description, status</td></tr>
        <tr><td><span class="badge-method badge-get">GET</span></td><td><code>/api/v1/.../tasks/{id}/activity</code></td><td>Retrieve task audit stream</td></tr>
      </tbody>
    </table>

  {:else if sectionId === "security"}
    <h1>Security & Threat Model</h1>
    <p class="lead">Trestle is built with defense-in-depth principles across transport, authentication, data isolation, and runtime boundaries.</p>

    <h2>Safeguards Implemented:</h2>
    <ul>
      <li><strong>Argon2id Hashing:</strong> State-of-the-art password hashing with configurable memory cost (default: 64 MiB), iterations, and parallelism.</li>
      <li><strong>Brute-Force Rate Limiting:</strong> In-memory sliding-window limiter prevents credential stuffing attacks.</li>
      <li><strong>Strict Same-Origin CSP:</strong> <code>default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self';</code></li>
      <li><strong>Information Concealment:</strong> Database errors and stack traces are suppressed; cross-tenant queries return concealed 404s.</li>
    </ul>

  {:else if sectionId === "design-system"}
    <h1>Design System & AGENTS.md</h1>
    <p class="lead">Trestle's UI adheres to a strict contract enforced by automated design linting and WCAG 2.2 AA accessibility standards.</p>

    <h2>Ten Non-Negotiables:</h2>
    <ol>
      <li><strong>No raw colors:</strong> Semantic tokens only (e.g. <code>var(--text)</code>, <code>var(--accent)</code>).</li>
      <li><strong>No native pickers:</strong> Custom accessible dropdowns and date pickers.</li>
      <li><strong>No inline styles:</strong> Dynamic CSSOM properties passed through Svelte directives (<code>style:--hue=...</code>).</li>
      <li><strong>Apple-Fluid Motion:</strong> Spring-based physics with no overshooting.</li>
      <li><strong>WCAG 2.2 AA:</strong> 44px minimum touch targets and visible focus indicators.</li>
    </ol>

  {:else if sectionId === "contributing"}
    <h1>Development & Testing Workflow</h1>
    <p class="lead">How to contribute, run test suites, and meet the Definition of Done.</p>

    <h2>Automated Verification Pipeline</h2>
    <CodeBlock lang="bash" code={`# Frontend verification
npm run lint:design --prefix web    # Verifies design-system contract
npm run check --prefix web          # svelte-check TypeScript validator
npm test --prefix web               # Vitest component unit tests
npm run format:check --prefix web   # Prettier format check
npm run build --prefix web          # Production asset bundle build

# Go verification
gofmt -l internal main.go migrations
go vet ./internal/... .
go test ./internal/...`} />
  {/if}
</div>

<style>
  .lead {
    font-size: 1.15rem;
    color: var(--text);
    margin-bottom: 1.5rem;
    line-height: 1.5;
  }
</style>
