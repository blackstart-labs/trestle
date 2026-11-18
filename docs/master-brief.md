# MASTER PROMPT: TRESTLE — ALL-IN-ONE PRODUCTIVITY & WORK MANAGEMENT PLATFORM

## 1. Your Role

Act as a Principal Software Architect, Senior Product Designer, Senior Full-Stack Engineer, UX Researcher, Database Architect, and DevOps Engineer.

Your task is to analyze, design, architect, and develop **Trestle**, an enterprise-quality, self-hosted, open-source productivity and work management platform.

You are not building a simple Kanban application.

You are building a unified workspace that combines the best and most practical capabilities of:

- Plane — Project Management, Issue Tracking, Agile Planning.
- Leantime — Strategic Planning, Goals, Objectives, Gantt Charts.
- Kimai — Time Tracking, Timesheets, Billable Hours.
- GitHub — Developer Workflow, Repository Integration, Pull Request Tracking.
- Notion — Documentation, Databases, Structured Knowledge.
- Obsidian — Markdown Knowledge Management, Backlinks, Graph View.

The objective is to create one coherent product instead of six disconnected applications.

**Product Philosophy:**

"Plan your work. Execute your projects. Track your time. Connect your knowledge. Automate your workflow."

---

# 2. CRITICAL: Analyze the Existing Repository First

Repository:

https://github.com/blackstart-labs/trestle

Before implementing anything:

1. Inspect the complete repository structure.
2. Read README.md, PRODUCT.md, CONTEXT.md, DESIGN.md, architecture documentation, and all relevant technical documentation.
3. Inspect existing frontend components, backend packages, API routes, database schemas, migrations, and tests.
4. Identify every implemented feature.
5. Identify partially implemented features.
6. Identify planned features.
7. Identify incomplete workflows, bugs, inconsistencies, and technical debt.
8. Understand existing design patterns and component architecture.
9. Understand the existing authentication, authorization, and workspace isolation mechanisms.
10. Produce a feature gap analysis before changing code.

### Strict rules

- Do not replace the existing application with a new scaffold.
- Do not rewrite the backend or frontend without a compelling technical reason.
- Do not remove existing features.
- Do not break existing API contracts unnecessarily.
- Do not duplicate existing functionality.
- Do not create mock implementations and present them as working features.
- Do not introduce unnecessary dependencies.
- Do not change the database architecture without documenting the reason.
- Do not use placeholder buttons or non-functional UI.
- Do not claim a feature is complete until its backend, frontend, database, authorization, and tests are implemented.

Preserve all existing functionality and extend it incrementally.

If documentation and implementation disagree, inspect the source code and tests. Treat verified implementation as the source of truth and update documentation accordingly.

---

# 3. Existing Technology Stack — Preserve It

Use the existing technology stack unless a technical limitation requires a justified change.

### Frontend
- Svelte 5
- TypeScript
- Vite
- Existing component architecture
- Existing design tokens and theme system
- Existing routing and state-management patterns

### Backend
- Go
- Standard `net/http` architecture
- Existing modular monolith structure
- Existing API conventions
- Existing middleware and authentication system

### Database
- PostgreSQL
- Existing database access layer
- Versioned SQL migrations
- Transaction-safe database operations

### Infrastructure
- Docker
- Go binary with embedded frontend assets
- GitHub Actions
- Existing deployment and configuration conventions

### Engineering principles

- Clean and modular architecture.
- Strong type safety.
- Secure-by-default design.
- Multi-tenant data isolation.
- Database integrity.
- Accessibility.
- Maintainability.
- Performance.
- Self-hosted deployment.
- No unnecessary microservices.

Do not introduce another framework just because it is popular.

---

# 4. Product Architecture

Organize Trestle into six interconnected product areas.

1. Plan
2. Execute
3. Track
4. Know
5. Automate
6. Integrate

All modules must share the same workspace, users, projects, permissions, search infrastructure, and activity system.

## Module A: PLAN — Strategic & Project Planning

Take inspiration from Plane and Leantime.

### A1. Workspace Management

Implement or improve:

- Multiple workspaces.
- Workspace settings.
- Workspace members.
- Workspace roles and permissions.
- Workspace invitations.
- Teams and groups.
- Workspace ownership transfer.
- Workspace-level dashboards.
- Workspace activity.
- Workspace-level settings.

### A2. Goals & Objectives

Implement strategic planning.

Features:

- Create organizational goals.
- Create measurable objectives.
- Define Key Results.
- Set target values.
- Assign goal owners.
- Define start and target dates.
- Link goals to projects.
- Link projects to objectives.
- Track goal progress.
- Show progress rollups.
- Track blocked or at-risk goals.
- Goal status history.
- Goal hierarchy.

Example:

Goal: Launch Trestle v1.0.

Objectives:
- Complete Project Management.
- Complete Time Tracking.
- Launch Documentation System.
- Release Developer Integrations.

Each objective must connect to actual projects and measurable progress.

### A3. Project Management

Support:

- Project creation and editing.
- Project templates.
- Project keys.
- Project descriptions.
- Project ownership.
- Project members.
- Project status.
- Project priority.
- Project start and due dates.
- Project progress.
- Project archiving.
- Project duplication.
- Project settings.
- Project activity.
- Project overview.

### A4. Advanced Issue & Task Management

Build a comprehensive task management system.

Task properties:

- Unique task identifier.
- Title.
- Rich description.
- Status.
- Priority.
- Assignees.
- Creator.
- Labels.
- Start date.
- Due date.
- Estimate.
- Spent time.
- Parent task.
- Project.
- Milestone.
- Cycle.
- Module.
- Dependencies.
- Attachments.
- Custom fields.

Task functionality:

- Create, edit, duplicate, archive, delete.
- Subtasks and nested tasks.
- Comments and threaded discussions.
- Mentions.
- Activity history.
- Task templates.
- Recurring tasks.
- Task dependencies.
- Blocked-by and blocking relationships.
- Task watchers.
- Task subscriptions.
- Bulk actions.
- Keyboard shortcuts.
- Inline editing.
- Custom statuses.
- Task relationship visualization.

### A5. Agile Planning

Implement:

**Cycles / Sprints**
- Create cycles.
- Define start and end dates.
- Assign tasks.
- Move incomplete tasks to another cycle.
- Track cycle progress.
- Cycle burndown.
- Cycle completion report.
- Previous and upcoming cycles.

**Modules / Epics**
- Group related tasks.
- Define module objectives.
- Set module deadlines.
- Assign owners.
- Track completion.
- Link modules to milestones.

**Roadmaps**
- Project roadmap.
- Workspace roadmap.
- Goal roadmap.
- Milestone timeline.
- Progress tracking.

### A6. Views

Implement a unified view system.

Required views:

- List View.
- Kanban View.
- Calendar View.
- Timeline View.
- Gantt View.
- Table View.
- My Tasks.
- Assigned to Me.
- Created by Me.
- Upcoming Tasks.
- Overdue Tasks.
- Completed Tasks.
- Blocked Tasks.

### A7. Saved Views & Filters

Allow users to:

- Filter by status.
- Filter by priority.
- Filter by assignee.
- Filter by labels.
- Filter by dates.
- Filter by project.
- Filter by cycle.
- Filter by milestone.
- Combine multiple filters.
- Save custom filters.
- Share views with workspace members.
- Configure sorting and grouping.
- Configure visible columns.

---

# 5. Module B: EXECUTE — Team Collaboration

Take inspiration from Plane, GitHub, and Notion.

Implement:

- Team management.
- Member profiles.
- Task assignment.
- Task comments.
- Threaded discussions.
- @Mentions.
- Activity feed.
- Project updates.
- Notifications.
- In-app notification center.
- Read/unread state.
- Notification preferences.
- User presence where appropriate.
- File attachments.
- Image uploads.
- Shared project resources.
- Task watchers.
- Project subscriptions.
- Workspace announcements.

### Collaboration rules

Every important action should produce an activity event.

Examples:

- Task created.
- Task assigned.
- Status changed.
- Priority changed.
- Comment added.
- Due date changed.
- Milestone completed.
- Pull Request linked.
- Document updated.

Use a consistent activity model rather than implementing separate history systems for every module.

---

# 6. Module C: TRACK — Professional Time Management

Take inspiration from Kimai.

This should become a complete time management system, not just a basic stopwatch.

### C1. Time Tracking

Implement:

- Start timer.
- Pause timer.
- Resume timer.
- Stop timer.
- Manual time entry.
- Timer description.
- Task association.
- Project association.
- Activity association.
- User association.
- Start and end timestamps.
- Duration calculation.
- Edit time entries.
- Delete time entries.
- Time entry approval.
- Timer history.

### C2. Reliable Timer Architecture

Important:

Do not rely exclusively on browser localStorage for active timer persistence.

Use a server-side running timer model.

Requirements:

- One active timer per user by default.
- Server-authoritative start time.
- Persisted timer state.
- Safe handling of multiple browser sessions.
- Correct duration calculation.
- Timezone-aware timestamps.
- Protection against duplicate time entries.
- Transaction-safe stop and log operations.
- Clear handling of browser refresh and network failures.

The frontend should synchronize with the server rather than treating its local clock as the authoritative source.

### C3. Timesheets

Implement:

- Daily timesheet.
- Weekly timesheet.
- Monthly timesheet.
- Calendar-based timesheet.
- User timesheet.
- Project timesheet.
- Client timesheet.
- Manual adjustments.
- Approval workflow.
- Timesheet export.
- Total hours.
- Billable hours.
- Non-billable hours.

### C4. Client & Activity Management

Implement:

- Customer management.
- Customer details.
- Customer contacts.
- Customer projects.
- Project activities.
- Activity categories.
- Internal and external projects.
- Customer-specific hourly rates.

### C5. Billing

Implement:

- Billable and non-billable time.
- Hourly rates.
- User-specific rates.
- Project-specific rates.
- Customer-specific rates.
- Billing summaries.
- Expense records.
- Invoice generation.
- Invoice status.
- Invoice PDF export.
- Tax configuration where appropriate.

Keep accounting functionality limited to time-based billing rather than trying to become a full accounting application.

### C6. Time Reports

Implement:

- Time by user.
- Time by project.
- Time by customer.
- Time by activity.
- Billable vs non-billable.
- Estimated vs actual time.
- Weekly and monthly summaries.
- Date range filtering.
- CSV and PDF exports.

---

# 7. Module D: KNOW — Documentation & Knowledge Management

Combine the best ideas from Notion and Obsidian.

This is a major product differentiator.

## D1. Document Management

Implement:

- Create documents.
- Edit documents.
- Delete documents.
- Archive documents.
- Nested pages.
- Document folders.
- Document templates.
- Document ownership.
- Document permissions.
- Document sharing.
- Document history.
- Version restoration.
- Comments.
- Mentions.
- Attachments.
- Favorites.
- Recently viewed documents.

## D2. Markdown Editor

Support:

- Headings.
- Paragraphs.
- Bold and italic.
- Lists.
- Checklists.
- Blockquotes.
- Code blocks.
- Syntax highlighting.
- Tables.
- Images.
- Links.
- File attachments.
- Horizontal rules.
- Inline code.
- Markdown import/export.

Preserve portable Markdown as much as possible.

Do not lock users into an undocumented proprietary document format.

## D3. Block Editor

Implement an extensible block-based editor.

Block types:

- Text.
- Heading.
- Bulleted list.
- Numbered list.
- Checklist.
- Quote.
- Code.
- Image.
- File.
- Table.
- Divider.
- Callout.
- Embed.
- Toggle.
- Database view.

Support:

- Drag and drop.
- Reordering.
- Nested blocks.
- Keyboard shortcuts.
- Slash commands.
- Copy and paste.
- Undo and redo.
- Autosave.
- Draft recovery.

Do not implement a complex real-time collaborative editor until the basic editor is reliable.

## D4. Connected Knowledge

Take inspiration from Obsidian.

Implement:

- Internal document links.
- Backlinks.
- Related documents.
- Wiki-style links.
- Tags.
- Properties.
- Document graph.
- Graph filtering.
- Orphaned document detection.
- Broken link detection.
- Global knowledge search.

Example:

A document named `Authentication Architecture` can link to `Database Schema`, `API Documentation`, and `Security Guidelines`.

Trestle should automatically display backlinks and connected documents.

## D5. Knowledge Graph

Build an interactive graph visualization.

Show:

- Documents as nodes.
- Links as edges.
- Project relationships.
- Tags and categories.
- Related knowledge.

Allow users to:

- Zoom.
- Pan.
- Search nodes.
- Filter by tags.
- Filter by workspace.
- Navigate to a document from the graph.

Do not make graph visualization the primary way to browse documents. It is a supplementary knowledge discovery tool.

## D6. Structured Collections

Take inspiration from Notion databases.

Users should be able to create custom collections.

Examples:

- Product Roadmap.
- Bug Database.
- Meeting Notes.
- Book Library.
- Research Database.
- Client Directory.
- Recruitment Tracker.

Support properties:

- Text.
- Number.
- Select.
- Multi-select.
- Date.
- Checkbox.
- URL.
- Email.
- Person.
- Relation.
- Formula.
- Status.

Collection views:

- Table.
- Board.
- Calendar.
- List.
- Gallery.

Support:

- Sorting.
- Filtering.
- Grouping.
- Relations.
- Rollups.
- Templates.
- Custom views.

---

# 8. Module E: AUTOMATE — Workflow Automation

Take inspiration from Plane's automation capabilities and general workflow automation systems.

Build an extensible event-driven automation engine.

### E1. Automation Rules

Users should be able to create rules using a visual interface.

Structure:

WHEN [Trigger]
IF [Condition]
THEN [Action]

Example:

WHEN a task is created.

IF priority is Urgent.

THEN:
- Assign to Project Lead.
- Send Telegram notification.
- Send Slack notification.
- Add an activity event.

### E2. Triggers

Support:

- Task created.
- Task updated.
- Task completed.
- Status changed.
- Priority changed.
- Assignee changed.
- Due date approaching.
- Task overdue.
- Comment added.
- Milestone completed.
- Cycle started.
- Cycle completed.
- Pull Request merged.
- Time entry created.
- Document updated.

### E3. Conditions

Support:

- Project.
- Status.
- Priority.
- Assignee.
- Labels.
- Due date.
- Task type.
- User role.
- Workspace.

### E4. Actions

Support:

- Change status.
- Assign user.
- Change priority.
- Add label.
- Create task.
- Add comment.
- Send notification.
- Send webhook.
- Create activity.
- Update related records.

### E5. Automation Management

Implement:

- Enable/disable rules.
- Rule history.
- Execution logs.
- Error handling.
- Retry policies.
- Idempotency.
- Permission validation.
- Execution limits.
- Audit trail.

Automation must never bypass workspace permissions.

---

# 9. Module F: INTEGRATE — Developer & External Integrations

Take inspiration from GitHub.

Do not build another Git hosting platform.

Connect Trestle to existing development tools.

## F1. GitHub Integration

Implement:

- GitHub OAuth or GitHub App integration.
- Connect repositories.
- Repository selection.
- Link repositories to projects.
- Link branches to tasks.
- Link commits to tasks.
- Link pull requests to tasks.
- Show pull request status.
- Show review status.
- Show merge status.
- Display CI/CD status.
- Display deployment information where available.
- GitHub webhook processing.
- GitHub event activity.

Example:

Task: `TRS-120 Implement Authentication`

Developer creates:

`feature/TRS-120-authentication`

Commits:

`feat(auth): implement login`

Pull Request:

`#42 Implement authentication`

Trestle should automatically associate these events with the task when a valid task identifier is present.

## F2. Notifications

Preserve existing integrations:

- Telegram.
- Slack.
- Discord.

Improve them with:

- Event-specific notification settings.
- Project-specific channels.
- User notification preferences.
- Test connection.
- Delivery status.
- Failure logs.
- Retry support.

## F3. Webhooks & API

Implement:

- Webhook subscriptions.
- Webhook signing.
- Secure webhook delivery.
- Retry mechanism.
- API authentication.
- API rate limiting.
- API documentation.
- Integration event logs.

---

# 10. DASHBOARD & ANALYTICS

Build dashboards for different user roles.

## Workspace Dashboard

Show:

- Total projects.
- Active projects.
- Completed projects.
- Total tasks.
- Overdue tasks.
- Task completion trends.
- Upcoming milestones.
- Team activity.
- Workspace goals.

## Personal Dashboard

Show:

- My tasks.
- Tasks due today.
- Upcoming deadlines.
- Overdue tasks.
- Active timer.
- Recent activity.
- Assigned projects.
- Weekly productivity.

## Project Dashboard

Show:

- Project progress.
- Task distribution.
- Milestone progress.
- Cycle progress.
- Team workload.
- Estimated vs actual hours.
- Recent activity.
- Upcoming deadlines.
- Blocked tasks.

## Time Dashboard

Show:

- Hours tracked.
- Billable hours.
- Non-billable hours.
- Time by project.
- Time by user.
- Time trends.

## Reports

Support:

- Date filters.
- Workspace filters.
- Project filters.
- User filters.
- Export to CSV.
- Export to PDF.

Use real backend data. Never ship hardcoded dashboard metrics as production data.

---

# 11. SEARCH SYSTEM

Build a unified search experience.

Search across:

- Projects.
- Tasks.
- Milestones.
- Cycles.
- Documents.
- Collections.
- Comments.
- Users.
- Clients.
- Activities.

Features:

- Global search.
- Keyboard shortcut.
- Recent searches.
- Search suggestions.
- Filters.
- Search by identifier.
- Search by title.
- Search by content.
- Permission-aware results.

Users must never receive search results from workspaces they are not authorized to access.

---

# 12. DESIGN SYSTEM & UX REQUIREMENTS

Design a premium, modern, dark-first productivity workbench.

The visual quality should be comparable to modern productivity products such as Linear, Plane, Notion, and GitHub.

### Design direction

- Professional.
- Minimal.
- Functional.
- Information-dense but readable.
- Consistent spacing.
- Strong typography.
- Subtle borders.
- Restrained shadows.
- Clear visual hierarchy.
- Smooth interactions.
- Excellent responsive behavior.

### Theme system

Support:

- Dark theme.
- Light theme.
- System preference.
- Custom accent colors.
- Consistent semantic color tokens.

Do not randomly change the existing Trestle branding or visual identity.

Preserve established design tokens unless there is a clear reason to improve them.

### Application Layout

Use a consistent application shell:

**Sidebar**
- Workspace switcher.
- Overview.
- My Tasks.
- Projects.
- Goals.
- Cycles.
- Milestones.
- Documents.
- Collections.
- Time Tracking.
- Reports.
- Integrations.
- Settings.

**Top Navigation**
- Breadcrumbs.
- Global search.
- Create button.
- Notifications.
- User menu.

**Main Content**
- Page title.
- Contextual actions.
- Filters.
- View switcher.
- Main content.

### Interaction quality

Implement:

- Command palette.
- Keyboard shortcuts.
- Context menus.
- Drag and drop where useful.
- Inline editing.
- Optimistic updates with rollback.
- Loading skeletons.
- Empty states.
- Error states.
- Confirmation dialogs.
- Toast notifications.
- Responsive drawers.
- Accessible modals.

### Accessibility

Target WCAG 2.2 AA.

Ensure:

- Keyboard navigation.
- Visible focus indicators.
- Screen-reader labels.
- Proper semantic HTML.
- Color contrast.
- Reduced motion support.
- Accessible forms.
- Accessible dialogs.

Do not use color alone to communicate task status.

### Responsive Design

Optimize for:

- Desktop.
- Laptop.
- Tablet.
- Mobile.

Mobile should support meaningful workflows, not just display a compressed desktop interface.

---

# 13. SHARED DOMAIN MODEL

Design the data model carefully.

Core entities should include:

- User.
- Workspace.
- Membership.
- Team.
- Customer.
- Project.
- ProjectMember.
- Goal.
- Objective.
- KeyResult.
- Task.
- Subtask.
- TaskDependency.
- TaskLabel.
- TaskAssignee.
- Status.
- Milestone.
- Cycle.
- Module.
- SavedView.
- TimeEntry.
- RunningTimer.
- Timesheet.
- Activity.
- Comment.
- Attachment.
- Document.
- DocumentBlock.
- DocumentLink.
- DocumentVersion.
- Tag.
- Collection.
- CollectionProperty.
- CollectionRecord.
- CollectionRelation.
- AutomationRule.
- AutomationExecution.
- Integration.
- Repository.
- PullRequest.
- Webhook.
- Notification.

These are conceptual entities, not mandatory one-to-one tables.

Normalize where appropriate, use JSON only for genuinely flexible data, and avoid creating unnecessary tables.

### Important relationships

- A Workspace contains Projects.
- A Workspace contains Members.
- A Project contains Tasks.
- A Project can belong to Goals.
- A Task can belong to a Cycle.
- A Task can belong to a Milestone.
- A Task can belong to a Module.
- A Task can have Dependencies.
- A Task can have Time Entries.
- A Task can link to Documents.
- A Project can link to GitHub Repositories.
- A Task can link to Branches, Commits, and Pull Requests.
- Automation Rules can listen to events from any supported module.

All tenant-owned records must have enforceable workspace isolation.

---

# 14. SECURITY & RELIABILITY

Treat security as a core product feature.

Implement:

- Strict authorization.
- Workspace isolation.
- Role-based access control.
- CSRF protection.
- Secure session management.
- Password hashing.
- Input validation.
- Output encoding.
- SQL injection protection.
- Rate limiting.
- Secure file uploads.
- MIME validation.
- File size restrictions.
- Webhook signature verification.
- Secret encryption at rest where appropriate.
- Audit logging.
- Safe error messages.
- Database transactions.
- Backup and restore documentation.

Never trust frontend permission checks.

Every protected operation must be authorized on the backend.

---

# 15. PERFORMANCE & SCALABILITY

Optimize for realistic small and medium-sized teams first, while keeping the architecture extensible.

Requirements:

- Efficient database queries.
- Proper indexing.
- Pagination.
- Avoid N+1 queries.
- Efficient filtering and sorting.
- Background job processing where necessary.
- Event-driven integrations.
- Cached derived statistics where justified.
- Efficient frontend rendering.
- Lazy loading for large datasets.
- Large document handling.
- Safe file storage.

Do not add Redis, Kafka, Elasticsearch, or additional infrastructure without demonstrating a concrete need.

Prefer a simple and reliable architecture.

---

# 16. TESTING REQUIREMENTS

Every major feature must include appropriate tests.

### Backend

- Unit tests.
- Integration tests.
- API tests.
- Authorization tests.
- Tenant isolation tests.
- Database transaction tests.
- Migration tests.
- Error handling tests.

### Frontend

- Component tests.
- State management tests.
- Form validation tests.
- Interaction tests.
- Responsive behavior tests.
- Accessibility tests.

### End-to-end

Test real workflows.

Example:

1. Create Workspace.
2. Invite Member.
3. Create Project.
4. Create Task.
5. Assign Task.
6. Start Timer.
7. Stop Timer.
8. Verify Time Entry.
9. Create Document.
10. Link Document to Task.
11. Connect GitHub Repository.
12. Link Pull Request.
13. Complete Task.
14. Verify Activity and Notification.

A feature is not complete simply because its page renders.

---

# 17. DEVELOPMENT STRATEGY

Do not attempt to implement every feature in one massive change.

Work in controlled phases.

### Phase 0 — Repository Audit

- Inspect the entire existing codebase.
- Document existing functionality.
- Create feature inventory.
- Identify technical debt.
- Create implementation plan.
- Identify migration requirements.

### Phase 1 — Core Project Management

- Complete advanced task management.
- Cycles.
- Modules.
- Dependencies.
- Saved views.
- Calendar.
- Gantt.
- Recurring tasks.

### Phase 2 — Time Management

- Server-side timer.
- Timesheets.
- Activities.
- Customers.
- Billable hours.
- Rates.
- Reports.

### Phase 3 — Knowledge Management

- Documents.
- Markdown editor.
- Nested pages.
- Internal links.
- Backlinks.
- Search.
- Document history.

### Phase 4 — Structured Collections

- Custom properties.
- Table.
- Board.
- Calendar.
- Relations.
- Rollups.
- Templates.

### Phase 5 — Developer Integrations

- GitHub App.
- Repository connection.
- Branches.
- Commits.
- Pull Requests.
- CI status.
- Webhooks.

### Phase 6 — Automation

- Event system.
- Automation builder.
- Conditions.
- Actions.
- Execution history.
- Retries.

### Phase 7 — Analytics & Product Polish

- Dashboards.
- Reports.
- Performance.
- Accessibility.
- Mobile experience.
- Security review.
- Documentation.
- Deployment improvements.

---

# 18. REQUIRED WORKFLOW FOR THE AI AGENT

For every development task:

1. Understand the feature and its user workflow.
2. Inspect relevant existing code.
3. Identify reusable components and services.
4. Design the database changes.
5. Design API contracts.
6. Implement backend logic.
7. Implement frontend interactions.
8. Add permissions and validation.
9. Add loading, empty, error, and success states.
10. Write tests.
11. Run relevant tests.
12. Run formatting and linting.
13. Review security and performance.
14. Update documentation.
15. Summarize changed files and implementation status.

Never skip backend implementation and build a frontend-only mockup.

Never mark incomplete work as complete.

If a task is too large, split it into smaller implementation tasks and complete them in dependency order.

Ask for clarification only when a genuine product decision cannot be inferred from the existing codebase or requirements.

---

# 19. FEATURE ACCEPTANCE CRITERIA

A feature is considered complete only when:

- Its user workflow is clearly defined.
- UI is implemented.
- Backend logic is implemented.
- Database persistence works.
- Authorization works.
- Validation works.
- Error handling works.
- Loading and empty states exist.
- Responsive behavior works.
- Tests pass.
- Documentation is updated.

For integrations, include connection setup, disconnection, credential handling, event processing, failure handling, and auditability.

For data-heavy features, verify pagination, filtering, and workspace isolation.

---

# 20. PRODUCT QUALITY STANDARD

Trestle must feel like one cohesive application.

Avoid creating a collection of unrelated modules with different UX patterns.

A user should be able to:

1. Set a goal.
2. Create a project.
3. Plan tasks and cycles.
4. Assign work to a team.
5. Track time.
6. Write project documentation.
7. Connect development activity.
8. Automate repetitive workflows.
9. Monitor project performance.
10. Understand everything from one workspace.

The user should not need to switch between five different applications to complete these workflows.

### Final Product Vision

Trestle should become:

**An open-source, self-hosted, all-in-one workspace for project management, team collaboration, time tracking, knowledge management, and developer productivity.**

Build it with the simplicity of a modern productivity application, the flexibility of Notion, the planning capabilities of Plane, the time tracking power of Kimai, the developer workflow of GitHub, and the connected knowledge model of Obsidian.

**Start with Phase 0: audit the existing repository and produce the implementation plan. Do not immediately start rewriting code. Preserve the existing product, then develop it incrementally into the vision described above.**
