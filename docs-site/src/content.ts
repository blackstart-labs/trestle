export interface DocSection {
  id: string;
  title: string;
  category: string;
  badge?: string;
  summary: string;
  metaTitle: string;
  metaDescription: string;
  keywords: string[];
}

export const SECTIONS: DocSection[] = [
  {
    id: "home",
    title: "Home Showcase",
    category: "Getting Started",
    badge: "Showcase",
    summary: "Visual product showcase and overview inspired by the Trestle design language.",
    metaTitle: "Trestle — Self-Hosted Workspace for Projects, Tasks & Knowledge",
    metaDescription: "Open-source, privacy-first team workspace combining Kanban boards, live task time tracking, milestone roadmaps, and chat integrations without SaaS complexity.",
    keywords: [
      "trestle",
      "self-hosted project management",
      "open source task manager",
      "kanban board",
      "svelte 5",
      "go project management",
      "privacy-first issue tracker",
      "self-hosted linear alternative",
      "self-hosted jira alternative"
    ]
  },
  {
    id: "getting-started",
    title: "Welcome & Core Concepts",
    category: "Getting Started",
    summary: "Understanding Trestle: instances, workspaces, projects, and the unified task identity.",
    metaTitle: "Welcome & Core Architecture Concepts | Trestle Documentation",
    metaDescription: "Explore Trestle's core architecture: independent instances, multi-tenant workspaces, canonical prefix keys, and atomic task identifiers without SaaS telemetry.",
    keywords: [
      "trestle architecture",
      "core concepts",
      "self-hosted instance",
      "multi-tenant workspaces",
      "atomic task identity",
      "task hierarchy"
    ]
  },
  {
    id: "quickstart-guide",
    title: "First Steps & Sign-In",
    category: "Getting Started",
    badge: "Quickstart",
    summary: "Logging in, initial admin setup, first workspace creation, and navigating the interface.",
    metaTitle: "Quickstart Guide: First Steps & Sign-In | Trestle Documentation",
    metaDescription: "Step-by-step setup guide for Trestle: initial administrator login, password security, creating your first workspace, and defining project prefixes.",
    keywords: [
      "trestle quickstart",
      "first steps",
      "admin login",
      "create workspace",
      "initial setup",
      "project onboarding"
    ]
  },
  {
    id: "workspaces",
    title: "Workspace Management",
    category: "Workspaces & Teams",
    summary: "Creating, switching, and organizing isolated workspaces for teams and clients.",
    metaTitle: "Workspace Management & Team Isolation | Trestle Documentation",
    metaDescription: "Learn how to manage isolated workspaces in Trestle. Partition projects, tasks, custom statuses, and members for distinct teams, clients, and departments.",
    keywords: [
      "workspace management",
      "team isolation",
      "multi-tenant workspaces",
      "workspace switcher",
      "client workspaces",
      "workspace settings"
    ]
  },
  {
    id: "members-roles",
    title: "Members, Roles & Access",
    category: "Workspaces & Teams",
    badge: "Roles",
    summary: "Managing team members, permission tiers (Owner, Admin, Member, Guest), and ownership transfer.",
    metaTitle: "Team Members, Permission Tiers & Role Access | Trestle Documentation",
    metaDescription: "Detailed guide to Trestle's role-based access control (RBAC). Configure Owner, Admin, Member, and Guest permission tiers and manage team invitations.",
    keywords: [
      "workspace members",
      "role based access control",
      "rbac",
      "permission tiers",
      "workspace owner",
      "team invites",
      "ownership transfer"
    ]
  },
  {
    id: "projects",
    title: "Projects & Prefix Keys",
    category: "Projects & Planning",
    summary: "Setting up projects, configuring unique prefixes (e.g. TR-1), and project lifecycles.",
    metaTitle: "Projects & Canonical Prefix Keys | Trestle Documentation",
    metaDescription: "Configure projects in Trestle with atomic human-readable prefix keys (e.g. API-1, WEB-42). Manage project lifecycles, statuses, and team assignments.",
    keywords: [
      "project management",
      "prefix keys",
      "atomic task keys",
      "issue identifiers",
      "project planning",
      "project lifecycle"
    ]
  },
  {
    id: "project-dashboard",
    title: "Dashboard & Progress Rails",
    category: "Projects & Planning",
    summary: "Visual progress rails, honest deadline tracking, completion math, and workload overviews.",
    metaTitle: "Project Dashboard & Progress Rails | Trestle Documentation",
    metaDescription: "Understand project health at a glance with Trestle's visual progress rails, deadline milestone completion math, honest velocity metrics, and team workload.",
    keywords: [
      "project dashboard",
      "progress rails",
      "completion math",
      "milestone velocity",
      "deadline tracking",
      "workload overview"
    ]
  },
  {
    id: "tasks-overview",
    title: "Tasks & Human Keys",
    category: "Tasks & Workflows",
    summary: "Creating tasks, atomic human identifiers, priorities, due dates, and estimates.",
    metaTitle: "Tasks & Atomic Human Identifiers | Trestle Documentation",
    metaDescription: "Master task tracking in Trestle. Create tasks with atomic keys, customize urgency priorities, due dates, time estimates, and rich Markdown specifications.",
    keywords: [
      "task management",
      "atomic human keys",
      "task priorities",
      "due dates",
      "markdown task editor",
      "task estimates",
      "issue tracking"
    ]
  },
  {
    id: "list-and-board",
    title: "List & Kanban Views",
    category: "Tasks & Workflows",
    badge: "Dual View",
    summary: "Switching between dense tabular list and visual Kanban board with inline editing.",
    metaTitle: "Interactive List & Kanban Board Views | Trestle Documentation",
    metaDescription: "Switch seamlessly between dense tabular lists and visual drag-and-drop Kanban boards with inline status editing, live timers, and keyboard navigation.",
    keywords: [
      "kanban board",
      "tabular list view",
      "drag and drop tasks",
      "dual view",
      "inline editing",
      "task triage",
      "agile board"
    ]
  },
  {
    id: "custom-statuses",
    title: "Workflow Statuses",
    category: "Tasks & Workflows",
    summary: "Custom status pipelines mapped to 5 canonical categories with delete remap safeguards.",
    metaTitle: "Workflow Statuses & Category Pipelines | Trestle Documentation",
    metaDescription: "Create custom workflow statuses mapped to 5 canonical categories (Backlog, Todo, Started, Completed, Cancelled) with safe delete remapping safeguards.",
    keywords: [
      "workflow statuses",
      "custom pipelines",
      "status categories",
      "status remapping",
      "backlog todo in-progress done",
      "lifecycle mapping"
    ]
  },
  {
    id: "task-details",
    title: "Task Drawer & Subtasks",
    category: "Tasks & Workflows",
    summary: "Side drawer details, checklist subtasks, multi-assignees, and tag labels.",
    metaTitle: "Task Detail Drawer, Checklists & Subtasks | Trestle Documentation",
    metaDescription: "Inspect tasks in a non-disruptive slide-out drawer. Manage checklist subtasks, multi-assignees, tag labels, and file attachments without losing context.",
    keywords: [
      "task detail drawer",
      "subtask checklist",
      "task assignees",
      "labels badges",
      "task slideout",
      "inline editing"
    ]
  },
  {
    id: "time-tracking",
    title: "Live Time Tracking",
    category: "Tasks & Workflows",
    badge: "Live",
    summary: "Live timer and timesheet logging with persistent background state and board indicators.",
    metaTitle: "Live Time Tracking, Stopwatch & Timesheets | Trestle Documentation",
    metaDescription: "Track work in real-time with Trestle's persistent live stopwatch, top-bar status pill, timesheet history logging, work notes, and board status indicators.",
    keywords: [
      "live time tracking",
      "task stopwatch",
      "persistent timer",
      "timesheet logging",
      "work notes",
      "billable hours",
      "project time tracking"
    ]
  },
  {
    id: "collaboration",
    title: "Comments & Activity Stream",
    category: "Tasks & Workflows",
    summary: "Real-time task discussions, change history audit trail, and team updates.",
    metaTitle: "Comments, Activity Stream & Webhook Alerts | Trestle Documentation",
    metaDescription: "Collaborate with real-time task discussions, complete audit activity streams, and instant webhook notifications to Telegram forum topics, Slack, and Discord.",
    keywords: [
      "task comments",
      "activity stream",
      "audit trail",
      "team collaboration",
      "telegram bot topic",
      "discord webhook",
      "slack notifications"
    ]
  },
  {
    id: "command-palette",
    title: "Command Palette & Search",
    category: "Productivity",
    badge: "⌘K",
    summary: "Instant keyboard navigation, global search, and keyboard shortcuts.",
    metaTitle: "Command Palette (⌘K) & Keyboard Shortcuts | Trestle Documentation",
    metaDescription: "Navigate your workspace at the speed of thought. Use Trestle's universal Command Palette (Ctrl+K / Cmd+K) for instant global search and keyboard workflows.",
    keywords: [
      "command palette",
      "keyboard shortcuts",
      "cmd k",
      "quick switcher",
      "global search",
      "power user shortcuts",
      "keyboard-first"
    ]
  },
  {
    id: "personalization",
    title: "Themes & Appearance",
    category: "Productivity",
    summary: "Customizing your workspace: light/dark modes, accent color presets, and accessibility.",
    metaTitle: "Themes, Accent Presets & Accessibility | Trestle Documentation",
    metaDescription: "Personalize your workspace with high-contrast light and dark themes, 6 curated accent color presets, custom user/workspace avatars, and WCAG 2.2 AA standards.",
    keywords: [
      "themes",
      "dark mode",
      "light mode",
      "accent color presets",
      "custom avatars",
      "wcag 2.2 aa accessibility",
      "workspace styling"
    ]
  },
  {
    id: "self-hosting",
    title: "Self-Hosting & Docker",
    category: "Deployment",
    badge: "Docker",
    summary: "Concise operator guide: 1-command setup and official Docker Hub deployment.",
    metaTitle: "Self-Hosting & Docker Deployment Guide | Trestle Documentation",
    metaDescription: "Deploy Trestle in production with 1 command. Complete operator guide for Docker, Docker Compose, PostgreSQL, reverse proxies, and Cloudflare Tunnels.",
    keywords: [
      "self-hosting trestle",
      "docker deployment",
      "docker compose",
      "postgresql setup",
      "cloudflare tunnel",
      "homelab project management",
      "production deployment"
    ]
  }
];

export const CATEGORIES = [
  "Getting Started",
  "Workspaces & Teams",
  "Projects & Planning",
  "Tasks & Workflows",
  "Productivity",
  "Deployment"
];

export function getSectionSEO(id: string): {
  title: string;
  description: string;
  keywords: string;
  canonical: string;
  category: string;
  ogType: string;
} {
  const section = SECTIONS.find(s => s.id === id) || SECTIONS[0];
  const isHome = section.id === "home";
  return {
    title: section.metaTitle,
    description: section.metaDescription,
    keywords: section.keywords.join(", "),
    canonical: isHome ? "https://tresstle.vercel.app/" : `https://tresstle.vercel.app/#${section.id}`,
    category: section.category,
    ogType: isHome ? "website" : "article"
  };
}
