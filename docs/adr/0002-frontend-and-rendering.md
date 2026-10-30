---
status: accepted
---
# Svelte with static delivery and selective rich interactions

Use Svelte, Vite and TypeScript as the primary frontend direction, approved after
the initial audit. Serve build assets from the Rust application's origin; do
not require a production Node server. M01b implements this with pinned build
dependencies, a readiness foundation page and assets embedded in the binary.
Product UI/routing remains planned; see its feature specification and evidence.

Server-rendered forms/partial updates reduce client machinery for simple pages
but complicate shared rich board/table/editor state; a full SSR application
server adds operational cost and private-content caching risks. A static Svelte
application fits authenticated interactive workflows with one framework and
one deployment origin. Keep ordinary forms semantic and load board/editor/graph
code by route or feature, not in every initial bundle.

This is not a claim that Svelte is fastest. Measure actual transfer/execution and
interaction costs. Public landing/docs/release pages can be separately
prerendered with titles, canonical and Open Graph metadata, sitemap/robots rules.
Private routes use noindex and private/no-store behavior as appropriate, but
authorization—not robots directives—protects data. SSR/hybrid rendering can be
revisited with evidence; do not introduce a second interactive framework.
