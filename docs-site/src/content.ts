export interface DocSection {
  id: string;
  title: string;
  category: string;
  badge?: string;
  summary: string;
}

export const SECTIONS: DocSection[] = [
  { id: "overview", title: "Overview & Architecture", category: "Getting Started", summary: "Product purpose, modular monolith architecture, and core design principles." },
  { id: "quickstart", title: "Quickstart Setup", category: "Getting Started", badge: "1-Command", summary: "One-command instant setup across Linux, macOS, WSL, and Windows." },
  { id: "self-hosting", title: "Production Self-Hosting", category: "Operations", badge: "Docker / Binary", summary: "Docker Compose, standalone static binary, systemd, Caddy/Nginx reverse proxies, and backups." },
  { id: "workspaces", title: "Workspaces & Roles", category: "Core Features", summary: "Multi-tenant workspace isolation, membership management, and atomic ownership transfer." },
  { id: "projects-tasks", title: "Projects & Tasks", category: "Core Features", summary: "Projects, atomic human keys (<KEY>-<NUM>), and dual List/Kanban board views." },
  { id: "custom-statuses", title: "Custom Workflow Statuses", category: "Core Features", summary: "Custom statuses mapped to 5 canonical categories, ordering, and atomic delete remap." },
  { id: "collaboration", title: "Task Collaboration", category: "Core Features", summary: "Hierarchical subtasks, multi-assignees, colored labels, comments, and audit activity streams." },
  { id: "operator-cli", title: "Operator CLI Reference", category: "Operations", summary: "Commands for migration, initial bootstrap, provisioning, and password resets." },
  { id: "api-reference", title: "REST API Reference", category: "Developer", badge: "v1", summary: "Complete endpoints specification for sessions, workspaces, projects, tasks, and activity." },
  { id: "security", title: "Security & Threat Model", category: "Architecture", summary: "Argon2id hashing, session token verification, rate limiting, CSRF, and strict same-origin CSP." },
  { id: "design-system", title: "Design System & AGENTS.md", category: "Developer", summary: "Semantic OKLCH tokens, accessible shared components, motion curves, and WCAG 2.2 AA." },
  { id: "contributing", title: "Development & Testing", category: "Developer", summary: "Running verification tests, svelte-check, Vitest, Go vet, and definition of done." }
];

export const CATEGORIES = ["Getting Started", "Core Features", "Operations", "Architecture", "Developer"];

