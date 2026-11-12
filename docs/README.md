# Trestle Technical Documentation

This directory contains the canonical technical notes, architecture specifications, and operational guides for Trestle. User guides, onboarding, and marketing documentation are hosted on the public website.

**Reference accuracy:** Phase 0 found outdated claims in the API, database,
operations and security references. Until those references are reconciled, use
the [verified corrections](phase-0-gap-analysis.md#8-documentation-corrections-and-decisions)
and the cited runtime source for session settings, configuration, routes,
permissions and secret storage. The Phase 0 plan does not implement new features.

---

## Technical Notes

| Document | Description |
| --- | --- |
| [Architecture & System Design](architecture.md) | Modular monolith architecture, single-origin Go + Svelte 5 runtime, module boundaries, and resource model. |
| [Database Schema & Migrations](database.md) | PostgreSQL schema, `pgxpool` configuration, transactions, advisory locks, composite keys, and forward migration lifecycle. |
| [REST API Specification](api.md) | REST `/api/v1` conventions, cookie session authentication, `X-CSRF-Token` protection, error envelopes, pagination, and endpoints. |
| [Threat Model & Security Policy](threat-model.md) | Trust boundaries, role-based access control (RBAC) matrix, Argon2id password policy, tenant isolation, and outbound webhook SSRF defenses. |
| [Operations & Deployment Guide](operations.md) | Production setup, environment variables, Docker/Podman container builds, CLI commands (`trestle migrate`, `trestle bootstrap`), health probes, and backup/restore. |
| [Phase 0: Feature Inventory & Implementation Plan](phase-0-gap-analysis.md) | Source-backed implemented/partial/missing capabilities, risks, migration proposals, delivery order and acceptance gates for the expanded product brief. |
| [Boards: Diagrams & Whiteboards](boards.md) | Native visual editing, permissions, persistence, JSON/SVG export, API contracts and real browser/database testing. |

---

## Design System

The design system specification is maintained under [`docs/design/`](design/):

- [Design Tokens](design/design-tokens.md): Semantic color palette (OKLCH), typography (Mona Sans), spacing, and motion easing.
- [Component Contracts](design/components.md): Reusable UI primitives (`Button`, `Select`, `DatePicker`, `MultiPicker`, `Modal`).
- [Design Principles](design/design-principles.md): Interaction rules, form validation, dialog behaviors, and responsive policies.

---

## Architecture Decision Records (ADRs)

Engineering decisions and technology selections are recorded under [`docs/adr/`](adr/):

- [ADR-0001: Modular Monolith and PostgreSQL](adr/0001-modular-monolith-and-postgresql.md)
- [ADR-0002: Frontend and Rendering](adr/0002-frontend-and-rendering.md)
- [ADR-0003: Sessions and Tenancy](adr/0003-sessions-and-tenancy.md)
- [ADR-0004: Local Identity and Operator Recovery](adr/0004-local-identity-and-operator-recovery.md)
- [ADR-0005: Go HTTP Backend](adr/0005-go-backend.md)
