<script lang="ts">
  import CodeBlock from "../components/CodeBlock.svelte";
  import Callout from "../components/Callout.svelte";

  interface Props {
    sectionId: string;
  }

  let { sectionId }: Props = $props();
</script>

<div class="prose">
  {#if sectionId === "getting-started"}
    <h1>Welcome & Core Concepts</h1>
    <p class="lead">Trestle is a self-hosted workspace for projects, tasks, workflows, and team collaboration — built to give you complete data ownership without SaaS complexity.</p>

    <Callout type="tip" title="THE TRESTLE PRINCIPLE">
      A single task identity (e.g. <code>TR-42</code>) connects your lists, boards, planning, and documentation. Trestle displays real state only — no artificial metrics, fake team counts, or SaaS telemetry.
    </Callout>

    <h2>The Core Hierarchy</h2>
    <p>Trestle organizes work through four clear tiers:</p>
    <ul>
      <li><strong>Instance:</strong> Your running Trestle server, operated independently on your own infrastructure.</li>
      <li><strong>Workspaces:</strong> Multi-tenant containers. Workspaces partition all projects, tasks, members, and custom workflows.</li>
      <li><strong>Projects:</strong> Focused initiatives within a workspace with their own prefix keys (e.g. <code>API</code>, <code>APP</code>, <code>DOCS</code>).</li>
      <li><strong>Tasks:</strong> The canonical work item. Every bug, feature, or story is a task with an atomic identifier (<code>&lt;KEY&gt;-&lt;NUM&gt;</code>).</li>
    </ul>

    <h2>Interface Overview</h2>
    <p>The application interface consists of three primary zones:</p>
    <ul>
      <li><strong>Navigation Sidebar:</strong> Quick workspace switcher, project links, global search, and workspace settings.</li>
      <li><strong>Main Canvas:</strong> Your primary working area — switch between the <em>Project Dashboard</em>, dense <em>List View</em>, and visual <em>Kanban Board</em>.</li>
      <li><strong>Task Detail Drawer:</strong> A non-disruptive slide-out panel allowing you to inspect subtasks, assignees, labels, and activity without losing your scroll position.</li>
    </ul>

  {:else if sectionId === "quickstart-guide"}
    <h1>First Steps & Sign-In</h1>
    <p class="lead">Getting started with your Trestle instance: logging in, creating your first workspace, and setting up initial projects.</p>

    <h2>1. Signing In</h2>
    <p>Open your browser and navigate to your Trestle instance URL (typically <code>http://localhost:3000</code> in local development or your domain in production).</p>
    <p>Enter your account credentials:</p>
    <ul>
      <li><strong>Default Username:</strong> <code>admin</code></li>
      <li><strong>Default Password:</strong> <code>adminpassword123</code> (or the custom password created during bootstrap)</li>
    </ul>

    <Callout type="note" title="SECURITY RECOMMENDATION">
      After your initial sign-in, change your password in <strong>Workspace Settings → Profile</strong> or run the operator CLI command <code>go run . resetpassword</code>.
    </Callout>

    <h2>2. Choosing or Creating a Workspace</h2>
    <p>After authentication, Trestle presents the <strong>Workspace Chooser</strong>:</p>
    <ol>
      <li>Select an existing workspace from your list to enter immediately.</li>
      <li>Click <strong>+ New Workspace</strong> to create an isolated workspace for a new company, client, or initiative.</li>
      <li>Give your workspace a friendly name (e.g. <code>Acme Studio</code>) and save.</li>
    </ol>

    <h2>3. Creating Your First Project</h2>
    <p>Inside your workspace, click <strong>+ New Project</strong> from the sidebar:</p>
    <ul>
      <li><strong>Project Name:</strong> A descriptive title (e.g. <code>Web Application</code>).</li>
      <li><strong>Prefix Key:</strong> A short 2 to 5 character code (e.g. <code>WEB</code>). All tasks in this project will receive atomic identifiers like <code>WEB-1</code>, <code>WEB-2</code>.</li>
      <li><strong>Description:</strong> Brief overview of the project scope.</li>
    </ul>

  {:else if sectionId === "workspaces"}
    <h1>Workspace Management</h1>
    <p class="lead">Workspaces provide clean, isolated boundaries for teams, departments, or separate client engagements.</p>

    <h2>Workspace Isolation</h2>
    <p>Everything in Trestle belongs to a workspace:</p>
    <ul>
      <li>Projects, tasks, subtasks, and comments are fully partitioned.</li>
      <li>Custom workflow statuses and category mappings are configured per workspace.</li>
      <li>Team memberships and roles are scoped exclusively to each workspace.</li>
    </ul>

    <h2>Switching Between Workspaces</h2>
    <p>You can belong to multiple workspaces with a single user account:</p>
    <ol>
      <li>Click the workspace name at the top of the left sidebar.</li>
      <li>Select any workspace from the dropdown list to switch instantly without re-authenticating.</li>
      <li>Use the global shortcut <code>⌘K</code> (or <code>Ctrl+K</code>) and type <em>Switch Workspace</em>.</li>
    </ol>

    <h2>Workspace Settings</h2>
    <p>Workspace Owners and Admins can access workspace configuration by clicking <strong>Settings</strong> in the sidebar:</p>
    <ul>
      <li><strong>General:</strong> Update workspace display name and identity hue.</li>
      <li><strong>Members:</strong> Invite team members, change roles, and revoke access.</li>
      <li><strong>Workflow:</strong> Manage status categories and project pipelines.</li>
    </ul>

  {:else if sectionId === "members-roles"}
    <h1>Members, Roles & Access</h1>
    <p class="lead">Trestle uses role-based access control (RBAC) to ensure team members have the appropriate level of authority.</p>

    <h2>Permission Roles</h2>
    <table class="manual-table">
      <thead>
        <tr>
          <th>Role</th>
          <th>Capabilities</th>
          <th>Best For</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td><strong>Owner</strong></td>
          <td>Full workspace control, member management, workspace deletion, and ownership transfer.</td>
          <td>Workspace creators, founders, lead operators.</td>
        </tr>
        <tr>
          <td><strong>Admin</strong></td>
          <td>Create/edit projects, manage custom workflow statuses, invite members, and update settings.</td>
          <td>Project managers, team leads.</td>
        </tr>
        <tr>
          <td><strong>Member</strong></td>
          <td>Create, edit, assign, and complete tasks; log time, post comments, and manage subtasks.</td>
          <td>Active contributors, engineers, designers.</td>
        </tr>
        <tr>
          <td><strong>Guest</strong></td>
          <td>Read-only viewing of assigned projects and public tasks. Cannot change project settings.</td>
          <td>Clients, external contractors, stakeholders.</td>
        </tr>
      </tbody>
    </table>

    <h2>Inviting Team Members</h2>
    <ol>
      <li>Navigate to <strong>Settings → Members</strong> in the sidebar.</li>
      <li>Click <strong>Invite Member</strong>.</li>
      <li>Enter the user's username or email address and select their assigned role (<em>Member</em>, <em>Admin</em>, or <em>Guest</em>).</li>
      <li>The member will immediately see the workspace upon their next sign-in.</li>
    </ol>

    <h2>Ownership Transfer</h2>
    <p>A workspace has exactly one Owner. To transfer ownership:</p>
    <ul>
      <li>Navigate to <strong>Settings → Members</strong> as the current Owner.</li>
      <li>Select an Admin member and choose <strong>Transfer Ownership</strong>.</li>
      <li>Confirm the prompt. This action is atomic — your role will automatically transition to Admin.</li>
    </ul>

  {:else if sectionId === "projects"}
    <h1>Projects & Prefix Keys</h1>
    <p class="lead">Projects organize tasks toward specific deliverables with dedicated human-readable prefix keys.</p>

    <h2>The Power of Prefix Keys</h2>
    <p>Every project in Trestle defines a unique prefix key (e.g. <code>API</code>, <code>UI</code>, <code>OPS</code>). This prefix establishes the human-readable identifier for all tasks:</p>
    <div class="key-example-badge">
      <span class="prefix">API</span><span class="divider">-</span><span class="num">104</span>
    </div>
    <p>Why human keys matter:</p>
    <ul>
      <li><strong>Clarity:</strong> Mentioning <code>API-104</code> in chat or commit messages immediately conveys context.</li>
      <li><strong>Atomic Counting:</strong> Task numbers increment sequentially per project without collision.</li>
      <li><strong>Fast Search:</strong> Typing any key into the Command Palette (<code>⌘K</code>) opens the task instantly.</li>
    </ul>

    <h2>Archiving Projects</h2>
    <p>When an initiative wraps up, archive the project rather than deleting it. Archiving preserves all task history, comments, and audit trails while hiding the project from daily sidebar lists.</p>

  {:else if sectionId === "project-dashboard"}
    <h1>Dashboard & Progress Rails</h1>
    <p class="lead">Every project features a dedicated dashboard providing honest, real-time summaries of work velocity and deadlines.</p>

    <h2>The Continuous Progress Rail</h2>
    <p>The Progress Rail spans every status category in a single visual bar at the top of the project. It provides an immediate visual breakdown of where work stands:</p>
    <ul>
      <li><strong>Backlog:</strong> Ideas and deferred tasks.</li>
      <li><strong>Unstarted:</strong> Ready for execution.</li>
      <li><strong>Started:</strong> Actively being worked on.</li>
      <li><strong>Completed:</strong> Delivered and verified.</li>
      <li><strong>Cancelled:</strong> Intentionally stopped work.</li>
    </ul>
    <p>Clicking any segment on the rail instantly filters the task list to that category.</p>

    <h2>Honest Completion Math</h2>
    <Callout type="note" title="NO ARTIFICIAL METRICS">
      Trestle calculates completion strictly from real task counts:
      <br /><br />
      <code>Completion Rate = Completed ÷ (Total Tasks − Cancelled)</code>
      <br /><br />
      Cancelled tasks are excluded from the denominator so dropped work never artificially inflates your progress!
    </Callout>

    <h2>The 7-Day Deadline Timeline</h2>
    <p>The dashboard displays a rolling 7-day calendar bar highlighting upcoming commitments. Overdue tasks are prominently flagged with an alert dot to prevent deadline slips.</p>

  {:else if sectionId === "tasks-overview"}
    <h1>Tasks & Human Keys</h1>
    <p class="lead">The task is Trestle's canonical work item — flexible enough for bugs, user stories, chores, and high-level milestones.</p>

    <h2>Anatomy of a Task</h2>
    <ul>
      <li><strong>Identifier:</strong> The immutable human key (e.g. <code>TR-45</code>).</li>
      <li><strong>Title:</strong> Clear, action-oriented task summary.</li>
      <li><strong>Description:</strong> Rich markdown content supporting lists, links, quotes, and code blocks.</li>
      <li><strong>Status:</strong> Current position in your workflow (e.g. <em>In Progress</em>, <em>Testing</em>).</li>
      <li><strong>Priority:</strong> Four distinct levels — <code>Urgent</code>, <code>High</code>, <code>Medium</code>, and <code>Low</code>.</li>
      <li><strong>Due Date:</strong> Target delivery date with relative recency indicators (e.g. <em>Due in 2 days</em>, <em>Overdue by 1 day</em>).</li>
      <li><strong>Estimate:</strong> Estimated duration or points (e.g. <code>2h</code>, <code>4h</code>, <code>1d</code>).</li>
      <li><strong>Assignees:</strong> One or more team members responsible for execution.</li>
      <li><strong>Labels:</strong> Reusable color-coded tags for categorization (e.g. <code>Frontend</code>, <code>Bug</code>, <code>Security</code>).</li>
    </ul>

    <h2>Quick Task Creation</h2>
    <p>Create tasks with zero friction:</p>
    <ul>
      <li>Click <strong>+ New Task</strong> in the project toolbar.</li>
      <li>Or use the keyboard shortcut <code>C</code> anywhere inside a project view.</li>
      <li>Press <code>Enter</code> to save and create another task immediately.</li>
    </ul>

  {:else if sectionId === "list-and-board"}
    <h1>List & Kanban Views</h1>
    <p class="lead">Switch effortlessly between high-density tabular lists and visual Kanban boards depending on how you prefer to work.</p>

    <h2>List View</h2>
    <p>The List View is designed for high-efficiency daily scanning and bulk organization:</p>
    <ul>
      <li><strong>Inline Property Changes:</strong> Click any status chip, priority badge, or due date directly on the row to change it immediately — no modal required.</li>
      <li><strong>Full Keyboard Navigation:</strong> Use arrow keys (<code>↑</code> / <code>↓</code>) to move between tasks, and <code>Enter</code> to open the detail drawer.</li>
      <li><strong>Sorting & Grouping:</strong> Group tasks by Status Category, Priority, or Assignee with a single click.</li>
    </ul>

    <h2>Kanban Board View</h2>
    <p>The Board View visualizes your workflow columns grouped across categories:</p>
    <ul>
      <li><strong>Drag and Drop:</strong> Drag task cards smoothly between columns to update status.</li>
      <li><strong>Keyboard Moves:</strong> Focus any card and press <code>Alt + ←</code> or <code>Alt + →</code> to advance work without a mouse.</li>
      <li><strong>Quick Add Slots:</strong> Click <strong>+ Add Task</strong> at the bottom of any column to create tasks directly in that workflow stage.</li>
    </ul>

  {:else if sectionId === "custom-statuses"}
    <h1>Workflow Statuses</h1>
    <p class="lead">Configure custom workflow pipelines tailored to your team's development lifecycle while retaining stable semantic categories.</p>

    <h2>The 5 Canonical Categories</h2>
    <p>Every custom status is mapped to one of five stable categories so dashboard math and filters always remain accurate:</p>
    <ul>
      <li><strong>Backlog:</strong> Ideas, triage items, or deferred work.</li>
      <li><strong>Unstarted:</strong> To Do, Planned, Ready for Sprint.</li>
      <li><strong>Started:</strong> In Progress, In Review, QA / Testing.</li>
      <li><strong>Completed:</strong> Done, Shipped, Released.</li>
      <li><strong>Cancelled:</strong> Won't Fix, Duplicate, Abandoned.</li>
    </ul>

    <h2>Configuring Statuses</h2>
    <ol>
      <li>In any project, click <strong>Workflow Settings</strong> (or the gear icon on the board).</li>
      <li>Click <strong>+ Add Status</strong>.</li>
      <li>Choose a display name (e.g. <code>Code Review</code>), select a category (e.g. <code>Started</code>), and pick a color tint.</li>
      <li>Drag statuses to reorder column order on your Kanban board.</li>
    </ol>

    <h2>Safe Delete Remap Safeguard</h2>
    <p>Trestle prevents orphaned tasks. When you delete a status that currently contains tasks, Trestle prompts you to select a replacement status:</p>
    <Callout type="tip" title="DATA SAFETY">
      All affected tasks are atomically reassigned to your chosen replacement status before the old status is removed. No task is ever lost or left in an invalid state.
    </Callout>

  {:else if sectionId === "task-details"}
    <h1>Task Drawer & Subtasks</h1>
    <p class="lead">The Task Detail Drawer lets you inspect and edit complex task metadata without losing your place on the board or list.</p>

    <h2>The Slide-Out Drawer</h2>
    <p>Clicking any task opens the side drawer from the right edge:</p>
    <ul>
      <li><strong>Inline Editing:</strong> Title, description, status, priority, and due dates save automatically as you make changes.</li>
      <li><strong>Context Preservation:</strong> The underlying list or board remains visible and responsive behind the drawer.</li>
      <li><strong>Quick Close:</strong> Press <code>Esc</code> or click outside to dismiss the drawer.</li>
    </ul>

    <h2>Checklist Subtasks</h2>
    <p>Break complex work into actionable, checklist items:</p>
    <ul>
      <li>Click <strong>+ Add Subtask</strong> inside the drawer.</li>
      <li>Enter subtask items and press <code>Enter</code> to rapidly create a checklist.</li>
      <li>Checking off subtasks updates the progress indicator in real-time (e.g. <code>3/5 completed</code>).</li>
    </ul>

    <h2>Multi-Assignees & Labels</h2>
    <p>Assign tasks to multiple contributors. Each person receives an avatar with an identity hue generated deterministically from their user ID. Add color-coded labels to classify work across functional areas (e.g. <code>Backend</code>, <code>Security</code>, <code>v1.0</code>).</p>

  {:else if sectionId === "collaboration"}
    <h1>Comments & Activity Stream</h1>
    <p class="lead">Communicate with teammates directly inside tasks and maintain an immutable audit trail of all project changes.</p>

    <h2>Task Discussions</h2>
    <p>Post updates and collaborate in context:</p>
    <ul>
      <li>Write comments using standard GitHub-flavored Markdown.</li>
      <li>Embed code snippets with syntax blocks, bullet lists, and links.</li>
      <li>Comments are timestamped and attributed with author avatar badges.</li>
    </ul>

    <h2>The Audit Activity Stream</h2>
    <p>Every task maintains an automatic audit timeline documenting all changes over time:</p>
    <ul>
      <li>Status transitions (e.g. <em>In Progress → Code Review</em>).</li>
      <li>Assignee additions and removals.</li>
      <li>Priority escalations and due date revisions.</li>
      <li>Subtask creations and completions.</li>
    </ul>

  {:else if sectionId === "command-palette"}
    <h1>Command Palette & Search</h1>
    <p class="lead">Move through your workspace at the speed of thought with global keyboard navigation and instant search.</p>

    <h2>Opening the Palette</h2>
    <p>Press <kbd>⌘K</kbd> on macOS or <kbd>Ctrl+K</kbd> on Windows/Linux from anywhere in the application. You can also click the search icon in the top header.</p>

    <h2>What You Can Search</h2>
    <ul>
      <li><strong>Tasks by Key:</strong> Type <code>TR-12</code> to jump directly to that task.</li>
      <li><strong>Tasks by Title:</strong> Search task titles across all projects in the workspace.</li>
      <li><strong>Projects:</strong> Jump to any project dashboard, list, or board.</li>
      <li><strong>Actions:</strong> Quick actions like <em>Create Task</em>, <em>Toggle Theme</em>, or <em>Switch Workspace</em>.</li>
    </ul>

    <h2>Essential Keyboard Shortcuts</h2>
    <table class="manual-table">
      <thead>
        <tr>
          <th>Shortcut</th>
          <th>Action</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td><kbd>⌘K</kbd> / <kbd>Ctrl+K</kbd></td>
          <td>Open Command Palette & Global Search</td>
        </tr>
        <tr>
          <td><kbd>C</kbd></td>
          <td>Create a new task in active project</td>
        </tr>
        <tr>
          <td><kbd>Esc</kbd></td>
          <td>Close top drawer, modal, or dropdown</td>
        </tr>
        <tr>
          <td><kbd>↑</kbd> / <kbd>↓</kbd></td>
          <td>Navigate through task list rows</td>
        </tr>
        <tr>
          <td><kbd>Enter</kbd></td>
          <td>Open focused task in detail drawer</td>
        </tr>
        <tr>
          <td><kbd>Alt + ←</kbd> / <kbd>Alt + →</kbd></td>
          <td>Move focused card between Kanban columns</td>
        </tr>
      </tbody>
    </table>

  {:else if sectionId === "personalization"}
    <h1>Themes & Appearance</h1>
    <p class="lead">Tailor Trestle to your personal work style with dark and light themes, custom brand accents, and accessible contrast.</p>

    <h2>Themes</h2>
    <p>Trestle ships with two carefully calibrated visual modes:</p>
    <ul>
      <li><strong>Night-Shift Dark:</strong> A deep, near-black workbench designed for evening productivity and low eye fatigue. Resting panels carry subtle luminous edges rather than harsh drop shadows.</li>
      <li><strong>Studio Light:</strong> A warm, cool off-white aesthetic mirroring the same surface structure with crisp clarity.</li>
    </ul>
    <p>Toggle themes instantly using the sun/moon icon in the header or via the Command Palette (<kbd>⌘K</kbd> → <em>Toggle Theme</em>).</p>

    <h2>Brand Accent Presets</h2>
    <p>Customize the primary action and focus hue in <strong>Settings → Appearance</strong>:</p>
    <ul>
      <li><strong>Lilac (Default):</strong> Soft, focused brand purple.</li>
      <li><strong>Cyan:</strong> Electric technical blue.</li>
      <li><strong>Emerald:</strong> Crisp balanced green.</li>
      <li><strong>Amber:</strong> Warm honey gold.</li>
      <li><strong>Rose:</strong> Vivid crimson red.</li>
    </ul>

    <Callout type="tip" title="ACCESSIBILITY FIRST">
      Every theme and accent combination satisfies WCAG 2.2 AA contrast standards (minimum 4.5:1 for body copy and 3:1 for interactive controls).
    </Callout>

  {:else if sectionId === "self-hosting"}
    <h1>Self-Hosting & Docker</h1>
    <p class="lead">Concise instructions to run Trestle on your own server or local machine using Docker or the automated runner.</p>

    <h2>Option 1: One-Command Setup (Linux / macOS / WSL)</h2>
    <p>Clone the repository and run the automated launcher:</p>
    <CodeBlock lang="bash" code={`# Run setup and start application
./run.sh

# On Windows PowerShell:
# .\\run.ps1`} />
    <p>The runner automatically handles database readiness, frontend assets, database migrations, and provisions the initial admin account.</p>

    <h2>Option 2: Docker Hub & Docker Compose (Production)</h2>
    <p>Pull the official pre-built image (~32MB) from Docker Hub:</p>
    <CodeBlock lang="bash" code={`docker pull marufsarker/trestle:latest`} />

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

    <p>Start the services in background mode:</p>
    <CodeBlock lang="bash" code={`docker compose up -d`} />
    <p>Your Trestle instance is now accessible at <code>http://localhost:3000</code>.</p>
  {/if}
</div>

<style>
  .lead {
    font-size: 1.15rem;
    line-height: 1.6;
    color: var(--text-secondary);
    margin-bottom: 1.75rem;
  }

  .manual-table {
    width: 100%;
    border-collapse: collapse;
    margin: 1.5rem 0 2rem;
    font-size: 0.92rem;
  }

  .manual-table th {
    text-align: left;
    padding: 0.75rem 1rem;
    background: var(--surface-sunken);
    border-bottom: 2px solid var(--line);
    color: var(--text);
    font-weight: 600;
  }

  .manual-table td {
    padding: 0.85rem 1rem;
    border-bottom: 1px solid var(--line);
    color: var(--text-secondary);
    vertical-align: top;
  }

  .manual-table tr:hover td {
    background: var(--hover);
  }

  .key-example-badge {
    display: inline-flex;
    align-items: center;
    padding: 0.35rem 0.85rem;
    border-radius: var(--radius-sm);
    background: var(--surface-raised);
    border: 1px solid var(--border);
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: 1.1rem;
    margin: 0.5rem 0 1rem;
    box-shadow: 0 2px 8px -2px rgba(0, 0, 0, 0.1);
  }

  .key-example-badge .prefix {
    color: var(--accent);
  }

  .key-example-badge .divider {
    color: var(--text-muted);
    margin: 0 0.15rem;
  }

  .key-example-badge .num {
    color: var(--text);
  }

  kbd {
    display: inline-block;
    padding: 0.15rem 0.45rem;
    font-size: 0.82em;
    font-family: var(--font-mono);
    background: var(--surface-raised);
    border: 1px solid var(--border);
    border-radius: 4px;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
    color: var(--text);
  }
</style>
