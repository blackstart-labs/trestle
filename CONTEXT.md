# Trestle

A shared language for projects, recorded work, and connected knowledge.

## People and ownership

**Instance**: One independently operated Trestle installation.

**User**: A person with an identity in an instance, potentially belonging to
several workspaces. _Avoid_: Account when referring to workspace ownership.

**Workspace**: A tenant containing members and their projects, work, and
knowledge. _Avoid_: Organization as a second, overlapping tenant level.

**Membership**: A user's association with a workspace or project, including
their role at that scope.

**Team**: A named group of workspace members. A team is not a tenant or an
implicit grant of unrestricted access.

**Customer**: A person or business for whom billable work is performed.
_Avoid_: Workspace, client when it could mean a browser application.

## Work and planning

**Project**: Work organized within one workspace with its own settings,
members, tasks, and workflow.

**Task**: The canonical actionable work item, including bugs and stories as
classifications. _Avoid_: Separate issue, todo, and ticket identities.

**Subtask**: A task with a parent task, not an independent kind of resource.

**Status**: A named, ordered workflow position configured for a project.

**Status category**: A stable semantic grouping of statuses, such as completed
or cancelled. _Avoid_: Inferring completion from a status's display name.

**Milestone**: A planning target grouping tasks toward an outcome and date.
_Avoid_: Treating every milestone as an actionable task.

**Cycle**: A bounded planning period for project work. _Avoid_: Sprint as a
second resource type with identical meaning.

**Saved view**: A named presentation and query configuration over existing
resources, not a copy of those resources.

## Time and knowledge

**Time entry**: Recorded work by one user with a project, time interval or
duration, and optional task association.

**Running timer**: A time entry whose work interval has started but not ended.

**Timesheet**: A period-based view or submission of a user's time entries.
_Avoid_: Using timesheet to mean every individual time entry.

**Activity**: A reusable classification of work performed, distinct from a task.

**Document**: An identifiable knowledge page containing portable Markdown and
associated metadata. _Avoid_: A folder path as its sole identity.

**Structured collection**: A set of user-defined records and typed properties.
_Avoid_: Database when confused with Trestle's persistence database.

**Automation rule**: An authorized trigger, conditions, and actions with
recorded execution. It is not unrestricted user code.
