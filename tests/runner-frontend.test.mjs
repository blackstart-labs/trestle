import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
  utimesSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";

const bash =
  process.platform === "win32"
    ? join(process.env.ProgramFiles, "Git/bin/bash.exe")
    : "bash";
const shells = [
  { command: bash, file: "run.sh", args: ["stage.sh"] },
  {
    command: "pwsh",
    file: "run.ps1",
    args: ["-NoProfile", "-File", "stage.ps1"],
  },
];

for (const shell of shells) {
  const available = spawnSync(shell.command, ["--version"], {
    encoding: "utf8",
  });
  test(
    `${shell.file}: refresh dependencies before rebuilding and retry failures`,
    { skip: available.error?.code === "ENOENT" },
    () => {
      const dir = mkdtempSync(join(tmpdir(), "trestle-runner-"));
      const write = (path, value) => writeFileSync(join(dir, path), value);
      const read = (path) => readFileSync(join(dir, path), "utf8");
      try {
        for (const path of [
          "web/src",
          "web/dist",
          "web/node_modules/.bin",
          "web/node_modules/vite",
          "bin",
        ])
          mkdirSync(join(dir, path), { recursive: true });
        write("web/package.json", '{"dependencies":{"marked":"1.0.0"}}');
        write("web/package-lock.json", '{"lockfileVersion":3}');
        // Reproduce an existing Vite install and bundle from before a dependency update.
        write("web/node_modules/.bin/vite", "");
        write("web/dist/index.html", "old bundle");
        write("web/src/app.svelte", "source");
        utimesSync(join(dir, "web/src/app.svelte"), new Date(0), new Date(0));
        const stage = readFileSync(
          new URL(`../${shell.file}`, import.meta.url),
          "utf8",
        )
          .replaceAll("\r\n", "\n")
          .split(/# 4\. Frontend[^\n]*\n/)[1]
          .split("# 5. Database Migrations")[0];
        write(
          "bin/npm",
          `#!/usr/bin/env bash
echo "$*" >> calls
if [ "$1" = ci ]; then
  [ ! -e fail-install ] || exit 42
  mkdir -p web/node_modules/.bin web/node_modules/vite
  touch web/node_modules/.bin/vite
else
  [ ! -e fail-build ] || exit 43
  mkdir -p web/dist
  echo bundle > web/dist/index.html
fi
`,
        );
        write(
          "stage.sh",
          `set -eo pipefail\nexport PATH="$PWD/bin:$PATH"\ngreen() { :; }\n${stage}`,
        );
        write(
          "stage.ps1",
          `$ErrorActionPreference = "Stop"
function npm {
  Add-Content calls ($args -join " ")
  if ($args[0] -eq "ci") {
    if (Test-Path fail-install) { $global:LASTEXITCODE = 42; return }
    New-Item -ItemType Directory -Force web/node_modules/vite | Out-Null
  } else {
    if (Test-Path fail-build) { $global:LASTEXITCODE = 43; return }
    New-Item -ItemType Directory -Force web/dist | Out-Null
    Set-Content web/dist/index.html bundle
  }
  $global:LASTEXITCODE = 0
}
${stage}`,
        );
        if (process.platform !== "win32")
          spawnSync("chmod", ["+x", join(dir, "bin/npm")]);

        const run = (success = true) => {
          write("calls", "");
          const result = spawnSync(shell.command, shell.args, {
            cwd: dir,
            encoding: "utf8",
            timeout: 30000,
          });
          assert.equal(
            result.status === 0,
            success,
            result.stdout + result.stderr,
          );
          return read("calls").trim().split(/\r?\n/).filter(Boolean);
        };
        const rebuild = [
          "ci --prefix web --ignore-scripts",
          "run build --prefix web",
        ];
        assert.deepEqual(run(), rebuild);
        assert.deepEqual(run(), []);
        write("web/package-lock.json", '{"lockfileVersion":3,"changed":true}');
        utimesSync(
          join(dir, "web/package-lock.json"),
          new Date(0),
          new Date(0),
        );
        assert.deepEqual(run(), rebuild);
        write("web/package.json", '{"dependencies":{"marked":"2.0.0"}}');
        assert.deepEqual(run(), rebuild);

        // Failed installs/builds must not be cached as a successful setup.
        for (const failure of ["fail-install", "fail-build"]) {
          const installed = read("web/node_modules/.trestle-package-lock.json");
          write("web/package-lock.json", JSON.stringify({ failure }));
          write(failure, "");
          assert.deepEqual(
            run(false),
            failure === "fail-install" ? rebuild.slice(0, 1) : rebuild,
          );
          assert.equal(
            read("web/node_modules/.trestle-package-lock.json"),
            installed,
          );
          rmSync(join(dir, failure));
          assert.deepEqual(run(), rebuild);
        }
        rmSync(join(dir, "web/dist/index.html"));
        assert.deepEqual(run(), rebuild.slice(1));
        utimesSync(join(dir, "web/dist/index.html"), new Date(0), new Date(0));
        utimesSync(
          join(dir, "web/src/app.svelte"),
          new Date(1000),
          new Date(1000),
        );
        assert.deepEqual(run(), rebuild.slice(1));
        assert.ok(existsSync(join(dir, "web/dist/index.html")));
      } finally {
        rmSync(dir, { recursive: true, force: true });
      }
    },
  );
}
