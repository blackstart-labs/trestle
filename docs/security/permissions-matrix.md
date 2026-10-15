# Planned permissions matrix

Status: proposed product policy for identity/project specifications. **No
authentication or authorization is implemented.** Roles label explicit operations;
do not implement one global numeric-role comparison.

## Scope model

An instance operator administers deployment, not an implicit application grant
to tenant resources. Host/database administrators can access underlying storage;
Trestle cannot promise secrecy from infrastructure owners. Workspace owners and
admins have explicit access to all projects in their workspace, including
private projects. Private means hidden from ordinary nonmembers, not from the
workspace administrators.

Workspace roles: owner, admin, member, guest. Project roles: manager, contributor,
viewer. Team membership, assignment and customer associations do not grant access
on their own. Initial implementation may expose fewer roles, but must not silently
merge their security semantics.

The maintainer approved owner-only transfer with password reauthentication fresh
within five minutes, and operator CLI provisioning/recovery without implicit
workspace membership. See [identity policy](identity-policy.md). These remain
requirements, not implemented authorization controls.

| Operation | Workspace owner | Workspace admin | Project manager | Project contributor | Project viewer / workspace guest |
| --- | --- | --- | --- | --- | --- |
| Workspace settings | Yes | Yes, excluding ownership/security-root changes | No | No | No |
| Transfer ownership | Explicit verified-owner operation | No | No | No | No |
| Invite/remove workspace members | Yes, preserve an owner | Yes, cannot remove/demote owner | No | No | No |
| Create/archive projects | Yes | Yes | Manage own assigned project only if granted | No | No |
| Manage project membership/workflow | Yes | Yes | Assigned project | No | No |
| Read private project content | Yes | Yes | Assigned project | Assigned project | Only explicit project-viewer membership |
| Create/edit/transition tasks | Yes | Yes | Assigned project | Assigned project | No |
| Read project documents/reports | Yes | Yes | Assigned project, excluding separately restricted reports | Same; not private financial/team details | Explicitly shared read scope only |
| Log/edit own time | Yes, subject to time policy | Same | Same in assigned project | Same in assigned project | No |
| Edit another user's time / view financial reports | Separate named grant | Separate named grant | Separate named grant | No | No |
| Manage integration credentials/automation | Explicit delegated management scope | Same | Only project-scoped explicit grant | No | No |
| Revoke own sessions | Yes | Yes | Yes | Yes | Yes |

Workspace member may discover workspace-visible projects; project mutation
rights require the applicable project grant. Guest is not a general browse-all
membership. The projects feature must define visibility defaults and project
creation grants precisely before shipping.

## Enforcement and revocation

- Resolve actor from an active session, never a user ID supplied in a request.
- Scope every resource lookup and derived query to the authorized tenant/project.
- For a resource the actor cannot discover, use a consistent concealed `404`.
  Use `403` only where existence is already authorized and the action is denied.
- Check current membership on each protected request. No permission cache in the
  first implementation. Define transaction locking/version checks for revocation
  racing with writes, including exports, automation and integrations.
- Enforce last-owner preservation, owner-transfer and self-demotion rules under
  concurrency, not only in UI controls.
- Audit grants/revocations/role changes and sensitive credential operations with
  redacted metadata; project activity and security audit have distinct audiences.

Required fixtures include two workspaces, private and visible projects, mixed
roles, nonmembers, disabled users and revoked sessions. Test guessed IDs,
cross-project associations, aggregates/search/exports and concurrent revocation.
