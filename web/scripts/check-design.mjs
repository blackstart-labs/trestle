#!/usr/bin/env node
// Design-system guard. Fails when frontend code bypasses the system:
// raw colours, native pickers, inline style attributes, ad-hoc motion,
// fonts, layers or icon sets. Run: npm run lint:design
//
// A line may opt out with a reason in a trailing comment:
//   design-allow: <rule> <why>
// Reviewers should question every allow.

import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative, sep } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../src", import.meta.url));

// The system itself is allowed to define what components may only consume.
const SYSTEM_FILES = new Set(["tokens.css", "app.css"]);
const isTest = (f) => /\.test\.ts$/.test(f);

const rules = [
  {
    id: "raw-colour",
    scope: "style",
    test: (line) => {
      // Neutral light and shadow are physics, not palette: pure white/black
      // with alpha for glints and drop shadows, and #000 as a mask stencil.
      const l = line
        .replace(/oklch\((0|1) 0 0( \/ [\d.]+)?\)/g, "")
        .replace(/(mask-image|mask):[^;]*/g, "")
        .replace(/^\s*#000\b[^;]*$/, "");
      return (
        /#[0-9a-fA-F]{3,8}\b/.test(l) ||
        /\b(rgb|rgba|hsl|hsla|oklch|oklab|lab|lch|color)\((?![^)]*var\(--)/.test(
          l,
        )
      );
    },
    message:
      "Raw colour in component styles. Use a semantic token from tokens.css (var(--text), var(--accent), …) or derive from one with var(--…).",
  },
  {
    id: "native-select",
    scope: "markup",
    test: (line) => /<select(\s|>|$)/.test(line),
    message:
      "Native <select>. Use ui/Select.svelte (field, chip or ghost variant).",
  },
  {
    id: "native-date",
    scope: "markup",
    test: (line) => /type=["']date["']/.test(line),
    message: 'Native <input type="date">. Use ui/DatePicker.svelte.',
  },
  {
    id: "inline-style",
    scope: "markup",
    test: (line) => /\sstyle=["{]/.test(line),
    message:
      'style="…" attribute. The CSP forbids inline styles; use a style:prop={value} directive or a class.',
  },
  {
    id: "transition-all",
    scope: "style",
    test: (line) => /transition:\s*all\b/.test(line),
    message:
      "transition: all. Name the properties (transform, opacity, colours) instead.",
  },
  {
    id: "raw-easing",
    scope: "style",
    test: (line) =>
      /cubic-bezier\(|linear\(\s*0/.test(line) ||
      /(transition|animation)[^;]*(?<![-\w])(ease|ease-in|ease-out|ease-in-out)(?![-\w])/.test(
        line,
      ),
    message:
      "Raw easing. Use --ease-spring (default), --ease-out, --ease-sheet, --ease-exit or --ease-bounce from tokens.css.",
  },
  {
    id: "font-family",
    scope: "style",
    test: (line) => /font-family:\s*(?!var\(|inherit)/.test(line),
    message:
      "font-family in a component. Use var(--font-body) / var(--font-mono).",
  },
  {
    id: "raw-z-index",
    scope: "style",
    test: (line) => /z-index:\s*\d{2,}/.test(line),
    message:
      "Raw z-index. Use a --layer-* token (sticky, overlay, palette, popover, skip).",
  },
  {
    id: "svelte-transition",
    scope: "script",
    test: (line) =>
      /from\s+["']svelte\/transition["']/.test(line) &&
      /\b(fly|fade|scale|slide|blur|draw)\b/.test(line),
    message:
      "Built-in Svelte transition. Use lib/motion.ts (pop/shrink, dialogIn/Out, drawerIn/Out, appear, fadeIn) so motion stays critically damped and consistent.",
  },
  {
    id: "icon-library",
    scope: "script",
    test: (line) =>
      /from\s+["'](lucide|@lucide|svelte-icons|@iconify|heroicons|@fortawesome|phosphor)/.test(
        line,
      ),
    message:
      "Icon package import. Add the path to ui/Icon.svelte instead (one outline family).",
  },
  {
    id: "external-asset",
    scope: "any",
    test: (line) =>
      /https?:\/\/(fonts\.googleapis|fonts\.gstatic|cdn\.|unpkg|cdnjs)/.test(
        line,
      ),
    message:
      "Third-party asset URL. The CSP allows only same-origin assets; self-host through npm.",
  },
];

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (/\.(svelte|css|ts)$/.test(name)) out.push(p);
  }
  return out;
}

/** Split a file into lines tagged with the region they belong to. */
function regions(file, text) {
  const lines = text.split(/\r?\n/);
  if (file.endsWith(".css")) return lines.map((l) => ({ l, region: "style" }));
  if (file.endsWith(".ts")) return lines.map((l) => ({ l, region: "script" }));
  let region = "markup";
  return lines.map((l) => {
    if (/<script[\s>]/.test(l)) region = "script";
    else if (/<style[\s>]/.test(l)) region = "style";
    const tagged = { l, region };
    if (/<\/script>/.test(l) || /<\/style>/.test(l)) region = "markup";
    return tagged;
  });
}

const problems = [];
for (const file of walk(root)) {
  const rel = relative(root, file).split(sep).join("/");
  const base = rel.split("/").pop();
  if (SYSTEM_FILES.has(base) || isTest(base)) continue;
  const text = readFileSync(file, "utf8");
  regions(file, text).forEach(({ l, region }, i) => {
    if (/^\s*(\/\/|\*|\/\*)/.test(l) && !/design-allow/.test(l)) return;
    for (const rule of rules) {
      if (rule.scope !== "any" && rule.scope !== region) continue;
      if (!rule.test(l)) continue;
      if (new RegExp(`design-allow:\\s*${rule.id}\\b`).test(l)) continue;
      problems.push(`web/src/${rel}:${i + 1}  [${rule.id}] ${rule.message}`);
    }
  });
}

if (problems.length) {
  console.error(`Design system check failed (${problems.length}):\n`);
  for (const p of problems) console.error("  " + p);
  console.error(
    "\nSee AGENTS.md § Design system and docs/design/. Fix the cause, or add " +
      "`design-allow: <rule> <reason>` on the line if the exception is deliberate.",
  );
  process.exit(1);
}
console.log("Design system check passed.");
