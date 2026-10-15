# Trestle documentation

This is the project documentation entry point. Read status labels literally:
requirements are not evidence that functionality exists.

## Authoritative sources

| Subject | Source |
| --- | --- |
| Canonical terminology | [Root glossary](../CONTEXT.md) |
| Vision and scope | [Product requirements](product/product-requirements.md), [vision](product/vision.md) |
| Capability status and evidence | [Feature matrix](product/feature-matrix.md) |
| Work order and blockers | [Roadmap and implementation tracker](product/roadmap.md) |
| Reference extraction | [Feature inventory](product/feature-inventory.md) |
| Current executable architecture | [Backend foundation](architecture.md) |
| System direction | [Architecture overview](architecture/architecture-overview.md), [ADRs](adr/0001-modular-monolith-and-postgresql.md) |
| Domain ownership | [Domain model](architecture/domain-model.md), [module interfaces](architecture/module-boundaries.md) |
| Current health/error behavior | [Root README](../README.md), [error contract](api/error-contract.md) |
| Current deployment/configuration | [Operations guide](operations.md) |
| Threats and permissions | [Current threat model](threat-model.md), [permissions matrix](security/permissions-matrix.md) |
| Planned product security | [Product threat model](security/threat-model.md) |
| Deployment/platform evidence | [Deployment architecture](architecture/deployment-architecture.md), [Termux evaluation](operations/android-termux.md) |
| Design direction | [Design principles](design/design-principles.md) |
| Test and release policy | [Testing strategy](testing/testing-strategy.md), [release checklist](testing/release-checklist.md) |
| Executed checks | [Verification record](verification.md) |
| Next identity feature specification | [Identity and workspaces](features/identity-and-workspaces.md) |
| Frontend foundation | [Specification](features/frontend-foundation.md), [verification](verification-frontend.md) |

## Structure and change policy

Create focused documents when a decision or feature needs them; never create
empty placeholder files to imply completeness. The requested product,
architecture, database, API, design, security, features, operations, testing and
contributing categories are the target organization, not a demand to duplicate
every existing document immediately.

The existing operational and foundation documents remain authoritative for
implemented behavior. Split them when their size or feature ownership warrants
it, preserving inbound links. ADRs use `docs/adr/`, the engineering skill
convention, rather than introducing a second decision-record directory.

Every feature specification must address the twenty sections demonstrated in
the identity specification, or explain why a section is not applicable. Later
capabilities have inventory entries and dependencies, not pretend-complete specs.
Feature implementation cannot start until its applicable contracts and material
security/scope decisions are resolved.

Requirements use stable identifiers. Link acceptance criteria to implementation
and tests in the feature matrix. A passing test proves the tested behavior,
not an entire future product category. Update the matrix and relevant sources
in the same change as implementation.
