# Trestle Technical Documentation

This directory contains the canonical technical notes, architecture specifications, and operational guides for Trestle. User guides, onboarding, and marketing documentation are hosted on the public website.

---

## Technical Notes

| Document | Description |
| --- | --- |
| [Architecture & System Design](architecture.md) | Modular monolith architecture, single-origin Go + Svelte 5 runtime, module boundaries, and resource model. |
| [Database Schema & Migrations](database.md) | PostgreSQL schema, `pgxpool` configuration, transactions, advisory locks, composite keys, and forward migration lifecycle. |
| [REST API Specification](api.md) | REST `/api/v1` conventions, cookie session authentication, `X-CSRF-Token` protection, error envelopes, pagination, and endpoints. |
| [Threat Model & Security Policy](threat-model.md) | Trust boundaries, role-based access control (RBAC) matrix, Argon2id password policy, tenant isolation, and outbound webhook SSRF defenses. |
| [Operations & Deployment Guide](operations.md) | Production setup, environment variables, Docker/Podman container builds, CLI commands (`trestle migrate`, `trestle bootstrap`), health probes, and backup/restore. |

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
- [ADR-0002: Svelte 5 Embedded Frontend](adr/0002-svelte-5-embedded-frontend.md)
- [ADR-0003: Mona Sans and Typography](adr/0003-mona-sans-and-typography.md)
- [ADR-0004: Local Identity and Operator Recovery](adr/0004-local-identity-and-operator-recovery.md)
- [ADR-0005: Go HTTP Backend](adr/0005-go-backend.md)
