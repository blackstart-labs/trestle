export interface DocSection {
  id: string;
  title: string;
  category: string;
  badge?: string;
  summary: string;
}

export const SECTIONS: DocSection[] = [
  { id: "home", title: "Home Showcase", category: "Getting Started", badge: "Showcase", summary: "Visual product showcase and overview inspired by the Trestle design language." },
  { id: "getting-started", title: "Welcome & Core Concepts", category: "Getting Started", summary: "Understanding Trestle: instances, workspaces, projects, and the unified task identity." },
  { id: "quickstart-guide", title: "First Steps & Sign-In", category: "Getting Started", badge: "Quickstart", summary: "Logging in, initial admin setup, first workspace creation, and navigating the interface." },

  { id: "workspaces", title: "Workspace Management", category: "Workspaces & Teams", summary: "Creating, switching, and organizing isolated workspaces for teams and clients." },
  { id: "members-roles", title: "Members, Roles & Access", category: "Workspaces & Teams", badge: "Roles", summary: "Managing team members, permission tiers (Owner, Admin, Member, Guest), and ownership transfer." },

  { id: "projects", title: "Projects & Prefix Keys", category: "Projects & Planning", summary: "Setting up projects, configuring unique prefixes (e.g. TR-1), and project lifecycles." },
  { id: "project-dashboard", title: "Dashboard & Progress Rails", category: "Projects & Planning", summary: "Visual progress rails, honest deadline tracking, completion math, and workload overviews." },

  { id: "tasks-overview", title: "Tasks & Human Keys", category: "Tasks & Workflows", summary: "Creating tasks, atomic human identifiers, priorities, due dates, and estimates." },
  { id: "list-and-board", title: "List & Kanban Views", category: "Tasks & Workflows", badge: "Dual View", summary: "Switching between dense tabular list and visual Kanban board with inline editing." },
  { id: "custom-statuses", title: "Workflow Statuses", category: "Tasks & Workflows", summary: "Custom status pipelines mapped to 5 canonical categories with delete remap safeguards." },
  { id: "task-details", title: "Task Drawer & Subtasks", category: "Tasks & Workflows", summary: "Side drawer details, checklist subtasks, multi-assignees, and tag labels." },
  { id: "collaboration", title: "Comments & Activity Stream", category: "Tasks & Workflows", summary: "Real-time task discussions, change history audit trail, and team updates." },

  { id: "command-palette", title: "Command Palette & Search", category: "Productivity", badge: "⌘K", summary: "Instant keyboard navigation, global search, and keyboard shortcuts." },
  { id: "personalization", title: "Themes & Appearance", category: "Productivity", summary: "Customizing your workspace: light/dark modes, accent color presets, and accessibility." },

  { id: "self-hosting", title: "Self-Hosting & Docker", category: "Deployment", badge: "Docker", summary: "Concise operator guide: 1-command setup and official Docker Hub deployment." }
];

export const CATEGORIES = [
  "Getting Started",
  "Workspaces & Teams",
  "Projects & Planning",
  "Tasks & Workflows",
  "Productivity",
  "Deployment"
];
