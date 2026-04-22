# CLAUDE.md

Guidance for Claude Code when working in this repository. Pair with [ARCHITECTURE.md](ARCHITECTURE.md) for deeper detail.

## Commands

```bash
make build          # Build binary for current platform → ./clawkit
make test           # Run all tests
make test-race      # Tests with the race detector (CGO required; also run in CI)
make fmt            # go fmt + go vet
make generate       # Regenerate registry.json from skills/**/{SKILL.md,config.json}
make check-generate # Verify registry.json is in sync (CI check)
make dist           # Cross-compile darwin/linux/windows → dist/
make npm-stage      # dist + copy binaries + skills/ + registry.json into npm/ (ready to publish)
make npm-pack       # npm-stage + `npm pack` (local tarball smoke test)
make release-check  # fmt + check-generate + test + npm-stage — dry run of the release workflow
make bump V=x.y.z   # Sync VERSION across Makefile and npm/package.json
```

Run a single test package:

```bash
CGO_ENABLED=0 go test -v ./internal/archive/...
```

**Always run `make generate` after editing any `SKILL.md` frontmatter or `config.json`.** CI fails if `registry.json` drifts (`make check-generate`).

## What clawkit is

A CLI skill manager for OpenClaw AI agents. A "skill" is `SKILL.md` (AI prompt with optional `{key}` placeholders) plus optional persona files (`_bootstrap/`), an optional runtime (`_engine/`), and dev metadata (`config.json`). Distributed as a single npm package `@rockship/clawkit` containing platform binaries (`binaries/`), skill files (`skills/`), and `registry.json`. A tiny Node wrapper (`bin/clawkit.js`) picks the right binary per OS/arch and points it at the packaged skills via `CLAWKIT_SKILLS_DIR` and `CLAWKIT_REGISTRY` env vars. The GitHub repo can stay private — npm (or GitHub Packages) handles auth.

## Install destinations

`clawkit install` splits files across three places by purpose:

| Destination | Contents | Lifetime |
|---|---|---|
| `<workspace>/skills/<skill>/` | `SKILL.md` (placeholders baked), `clawkit.json` | Removed on uninstall |
| `<workspace>/` (root) | `_bootstrap/*.md` (IDENTITY.md, SOUL.md, safety_rules.md, …) | Overwritten every install |
| `~/.clawkit/engines/<key>/` | `_engine/` payload (binary, DB, …) | Shared; survives uninstall; removed only by `clawkit purge <key>` |

`~/.clawkit/bin/` holds symlinks to the runtime's `bins`, added to `PATH` via `ensureInPath`.

Runtime `key` = group name (grouped skill) or skill name (flat skill with its own `_engine/`). Every member of a group shares one runtime — one binary, one database.

## Skill layout

**Flat skill:**

```text
skills/<skill>/
  _bootstrap/           Persona .md → workspace root on install
  _engine/                 Runtime payload (binary, data, …)
  engine.json             { exclude, data_paths, bins }
  config.json          { version, setup_prompts }
  SKILL.md              Frontmatter + agent prompt
```

**Grouped skills** share `_bootstrap/`, `_engine/`, `engine.json` at the group level:

```text
skills/<group>/
  _bootstrap/
  _engine/
  engine.json
  <skill-a>/
    config.json
    SKILL.md
  <skill-b>/
    config.json
    SKILL.md
```

Shared artifacts exist **only at the group level**. The child skill dir holds only `config.json` + `SKILL.md`.

## Metadata files

Four files, four consumers:

**`SKILL.md` frontmatter** — OpenClaw-native, consumed by the agent and by `gen-registry`:

```yaml
---
name: my-skill
description: One-line purpose
metadata:
  openclaw:
    os: [darwin, linux, windows]
    requires:
      bins: [sa-cli]
      config: []
---
```

**`config.json`** (dev-only) — consumed only by `gen-registry`. Never copied to the install:

```json
{
  "version": "1.0.0",
  "setup_prompts": [{"key": "shop_name", "label": "Shop name"}]
}
```

**`engine.json`** — runtime install rules, consumed by `internal/engine`:

```json
{
  "exclude":    ["cmd"],
  "data_paths": ["sa-data"],
  "bins":       ["sa-cli"]
}
```

- `exclude` — paths inside `_engine/` skipped on runtime install (source dirs like `cmd/`, tests, …).
- `data_paths` — paths preserved across re-installs (shared DBs, user-written state).
- `bins` — names chmodded `+x` and symlinked into `~/.clawkit/bin/`.

**`clawkit.json`** — written into each installed skill dir:

```json
{
  "version":     "1.0.0",
  "group":       "study-aboard",
  "user_inputs": { "shop_name": "Hoa Xuan" }
}
```

`user_inputs` survives `clawkit update` so placeholders are re-baked without re-prompting. `group` records the group a skill was installed from (empty for flat).

## Data flow

1. **Build-time:** `cmd/gen-registry` walks `skills/` and produces `internal/installer/registry.json` from each `SKILL.md` frontmatter + `config.json`. A directory is recorded as a group when it holds `_engine/` *and* child `SKILL.md`s. `_engine/` is never scanned into. Directory name is the canonical key; `name:` in frontmatter is informational.
2. **Install-time:** `internal/installer` loads the registry (local override → `CLAWKIT_REGISTRY` → embedded), copies the skill dir (skipping `config.json`, `_engine/`, `engine.json`, `_bootstrap/`), installs the shared engine, links bins, prompts for `setup_prompts`, updates the OpenClaw allowlist, bakes `{key}` placeholders, copies `_bootstrap/*.md` to the workspace root, and writes `clawkit.json`.
3. **Runtime:** the agent reads `SKILL.md`; invocations in the prompt resolve `sa-cli` (or whatever) through `PATH`, backed by `~/.clawkit/bin/<bin>` → `~/.clawkit/engines/<key>/<bin>`.

## Key packages

- **`cmd/clawkit/main.go`** — CLI dispatcher: `list`, `install`, `update`, `uninstall`, `purge`, `status`, `web`, `dashboard`, `version`. `install`/`update` accept `<name> [<member>…]` where `name` resolves to either a flat skill or a group and trailing args select specific members.
- **`cmd/gen-registry/main.go`** — Scans `skills/**/SKILL.md` + `config.json`. Hand-written indent-aware YAML parser. Emits `registry.json` with `skills` and `groups` sections.
- **`internal/installer/commands.go`** — Orchestrates install / update / uninstall / purge / status. Install flow: preflight → download → engine install + link bins → prompt → allowlist → template → bootstrap → `clawkit.json`.
- **`internal/installer/registry.go`** — Registry load (local override → `CLAWKIT_REGISTRY` env set by the npm wrapper → embedded fallback), source resolution (`findLocalSkill` for dev tree, `findSkillIn(CLAWKIT_SKILLS_DIR)` for the packaged skills), `downloadSkill`, `installEngine`, `alwaysExclude`.
- **`internal/installer/lockdown.go`** — Allowlist only. `SetupWorkspace` appends to `agents.defaults.skills`; `RemoveFromWorkspace` removes the entry and clears it when empty.
- **`internal/engine/`** — Shared-engine install/purge under `~/.clawkit/engines/<key>/`, `engine.json` parsing, bin symlinking into `~/.clawkit/bin/`, exclude / data_paths logic.
- **`internal/archive/`** — `tar.gz` / `zip`; strips top-level dir.
- **`internal/config/`** — `SkillConfig { Version, Group, UserInputs }`, OpenClaw detection, `Preflight`.
- **`internal/template/`** — `Process()` replaces `{key}` placeholders in the installed `SKILL.md`.
- **`internal/dashboard/`** — Web dashboard served by `clawkit dashboard`.
- **`internal/ui/`** — ANSI terminal helpers.
- **`skills/`** — Built-in skills grouped by vertical (`ecommerce`, `finance`, `self-improving-agent`, `sme`, `study-aboard`, `utilities`). These are copied into `npm/skills/` by `make npm-stage` so they ship with every published npm package.
- **`TEMPLATE.md`** — Reference for authoring new skills (layout, file purposes).

## Adding a skill

1. Create `skills/<name>/` (flat) or `skills/<group>/<name>/` (grouped).
2. Author `SKILL.md` and `config.json`. For grouped skills, add `_bootstrap/`, `_engine/`, `engine.json` at the group level (shared by every member).
3. `make generate`.

See [TEMPLATE.md](TEMPLATE.md) for each file's shape.

## Non-obvious invariants

- **Never edit `internal/installer/registry.json` by hand** — regenerated by `make generate`.
- **`config.json`, `_engine/`, `engine.json`, `_bootstrap/` never land in the installed skill dir** — all four are in `alwaysExclude` in [internal/installer/registry.go](internal/installer/registry.go).
- **Uninstall does NOT purge the shared engine.** Data (SQLite DBs etc.) survives; use `clawkit purge <key>` for explicit cleanup.
- **Engine update preserves `data_paths`.** Re-running `clawkit install` or `clawkit update` overwrites binaries and code but leaves any path listed in `data_paths` untouched.
- **On Windows, engine bins are copied, not symlinked.** Symlinks there usually require admin.
- **`registry.json` is embedded into the binary via `//go:embed`** as an offline fallback only. At runtime the npm wrapper points `CLAWKIT_REGISTRY` at the fresh `registry.json` shipped in the package, which takes priority. A local `./registry.json` in cwd always wins (dev override).

## Cross-platform rules

| Concern | Do | Don't |
|---|---|---|
| File paths | `filepath.Join(a, b)` | `a + "/" + b` |
| Temp directory | `os.TempDir()` | Hardcode `/tmp` |
| Binary names | Append `.exe` on Windows | Assume no `.exe` |
| Unix-only syscalls | Guard with `runtime.GOOS != "windows"` | Call `chmod` / `sudo` unconditionally |
| Archive format | `.zip` on Windows, `.tar.gz` elsewhere | Assume `.tar.gz` |
| Bin linking | Symlink on Unix, copy on Windows | Symlink unconditionally |

## Zero external dependencies

The Go module uses only the standard library. The YAML frontmatter parser is hand-written. Do not add dependencies without discussion.

## Release

1. `make release-check` — local dry run (`fmt + check-generate + test + npm-stage`).
2. `make bump V=x.y.z` — sync VERSION across `Makefile` and `npm/package.json` so dev view and published view can't drift.
3. Commit, tag `vx.y.z`, push tag.

Pushing the `v*` tag triggers `.github/workflows/release.yml`: cross-compile all binaries, run `make npm-stage` (copy binaries into `npm/binaries/`, skills into `npm/skills/`, `registry.json` into `npm/`), then `npm publish --access restricted` using `${{ secrets.NPM_TOKEN }}`. The workflow also `sed`s the Makefile VERSION in-place at runtime as a safety net; always bump first.

### How skills are located at install time

The binary does **not** embed skills. `clawkit install <skill>` resolves its source in this order:

1. Local `skills/<...>/<skill>/SKILL.md` in the current working directory — for development inside the repo.
2. The packaged skills dir pointed to by `CLAWKIT_SKILLS_DIR` — set by the npm wrapper to `<pkg>/skills`.
3. No network, no cache, no auth.

`registry.json` follows the same pattern: `./registry.json` → `CLAWKIT_REGISTRY` env → small embedded fallback. The wrapper sets `CLAWKIT_REGISTRY=<pkg>/registry.json` automatically.
