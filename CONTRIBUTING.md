# Contributing

Read [README.md](README.md), [architecture](docs/architecture.md), and the
[threat model](docs/threat-model.md) first. Any frontend change, whether by hand
or with an AI agent, must follow [AGENTS.md](AGENTS.md) (the design-system
contract) and pass `npm run lint:design`. Check the current branch and working
tree; preserve other contributors' uncommitted work.

- Keep changes small and specific. Describe expected behavior and acceptance
  criteria before substantial changes.
- Add deterministic tests through the affected public interfaces. Use real
  PostgreSQL for database behavior and never a production database.
- Run gofmt, go vet, tests, govulncheck and release build as listed
  in the README. Report checks you could not run accurately.
- Build `web/dist` before any Go build/test; run the frontend format/type/DOM/browser checks
  too. Commit the npm lockfile, not node_modules/dist. Review generated bundled
  license notices and `npm audit`; do not bypass advisories to make CI green.
- Add forward migrations; never edit applied migrations or delete existing data
  without explicit approval and a recovery plan.
- Keep domain rules out of HTTP handlers. Do not add speculative abstractions,
  dependencies, database engines or product capabilities.
- Update documentation for behavior/configuration changes. Review security,
  accessibility, resource bounds and error paths, not just the happy path.
- Preserve the license and upstream relationship. No release/push is implied
  by a local contribution. Use conventional commit messages when committing.

Track non-sensitive bugs in the repository's GitHub Issues. Do not publish
secrets or security vulnerabilities in a public issue or PR; follow SECURITY.md.
