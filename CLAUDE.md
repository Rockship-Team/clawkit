# CLAUDE.md

Guidance for Claude Code when working in this repository. Pair with [ARCHITECTURE.md](ARCHITECTURE.md) for deeper detail.

## Commands

```bash
make build          # Build binary for current platform → ./clawkit
make test           # Run all tests
make test-race      # Tests with the race detector (CGO required; also run in CI)
make fmt            # go fmt + go vet
make generate       # Regenerate registry.json from skills/**/{SKILL.md,_config.json}
make check-generate # Verify registry.json is in sync (CI check)
make dist           # Cross-compile darwin/linux/windows → dist/
make release-check  # fmt + check-generate + test + dist — dry run of the release workflow
make bump V=x.y.z   # Sync VERSION across Makefile and npm/package.json
```

Run a single test package:

```bash
CGO_ENABLED=0 go test -v ./internal/archive/...
```

**Always run `make generate` after editing any `SKILL.md` frontmatter or `_config.json`.** CI fails if `registry.json` drifts (`make check-generate`).

## What clawkit is

A CLI skill manager for OpenClaw AI agents. A "skill" is `SKILL.md` (AI prompt with optional `{key}` placeholders) plus optional persona files (`_bootstrap/`), an optional runtime (`_cli/`), and dev metadata (`_config.json`). Distributed via npm, which wraps platform-specific Go binaries in `npm/binaries/`.

## Install destinations

`clawkit install` splits files across three places by purpose:

| Destination | Contents | Lifetime |
|---|---|---|
| `<workspace>/skills/<skill>/` | `SKILL.md` (placeholders baked), `clawkit.json` | Removed on uninstall |
| `<workspace>/` (root) | `_bootstrap/*.md` (IDENTITY.md, SOUL.md, safety_rules.md, …) | Overwritten every install |
| `~/.clawkit/runtimes/<key>/` | `_cli/` payload (binary, DB, …) | Shared; survives uninstall; removed only by `clawkit purge <key>` |

`~/.clawkit/bin/` holds symlinks to the runtime's `bins`, added to `PATH` via `ensureInPath`.

Runtime `key` = group name (grouped skill) or skill name (flat skill with its own `_cli/`). Every member of a group shares one runtime — one binary, one database.

## Skill layout

**Flat skill:**

```text
skills/<skill>/
  _bootstrap/           Persona .md → workspace root on install
  _cli/                 Runtime payload (binary, data, …)
  _cli.json             { exclude, data_paths, bins }
  _config.json          { version, setup_prompts }
  SKILL.md              Frontmatter + agent prompt
```

**Grouped skills** share `_bootstrap/`, `_cli/`, `_cli.json` at the group level:

```text
skills/<group>/
  _bootstrap/
  _cli/
  _cli.json
  <skill-a>/
    _config.json
    SKILL.md
  <skill-b>/
    _config.json
    SKILL.md
```

Shared artifacts exist **only at the group level**. The child skill dir holds only `_config.json` + `SKILL.md`.

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

**`_config.json`** (dev-only) — consumed only by `gen-registry`. Never copied to the install:

```json
{
  "version": "1.0.0",
  "setup_prompts": [{"key": "shop_name", "label": "Shop name"}]
}
```

**`_cli.json`** — runtime install rules, consumed by `internal/runtime`:

```json
{
  "exclude":    ["cmd"],
  "data_paths": ["sa-data"],
  "bins":       ["sa-cli"]
}
```

- `exclude` — paths inside `_cli/` skipped on runtime install (source dirs like `cmd/`, tests, …).
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

1. **Build-time:** `cmd/gen-registry` walks `skills/` and produces `internal/installer/registry.json` from each `SKILL.md` frontmatter + `_config.json`. A directory is recorded as a group when it holds `_cli/` *and* child `SKILL.md`s. `_cli/` is never scanned into. Directory name is the canonical key; `name:` in frontmatter is informational.
2. **Install-time:** `internal/installer` fetches the registry (embedded → remote → local override), copies the skill dir (skipping `_config.json`, `_cli/`, `_cli.json`, `_bootstrap/`), installs the shared runtime, links bins, prompts for `setup_prompts`, updates the OpenClaw allowlist, bakes `{key}` placeholders, copies `_bootstrap/*.md` to the workspace root, and writes `clawkit.json`.
3. **Runtime:** the agent reads `SKILL.md`; invocations in the prompt resolve `sa-cli` (or whatever) through `PATH`, backed by `~/.clawkit/bin/<bin>` → `~/.clawkit/runtimes/<key>/<bin>`.

## Key packages

- **`cmd/clawkit/main.go`** — CLI dispatcher: `list`, `install`, `update`, `uninstall`, `purge`, `status`, `web`, `dashboard`, `version`. `install`/`update` accept `<name> [<member>…]` where `name` resolves to either a flat skill or a group and trailing args select specific members.
- **`cmd/gen-registry/main.go`** — Scans `skills/**/SKILL.md` + `_config.json`. Hand-written indent-aware YAML parser. Emits `registry.json` with `skills` and `groups` sections.
- **`internal/installer/commands.go`** — Orchestrates install / update / uninstall / purge / status. Install flow: preflight → download → runtime install + link bins → prompt → allowlist → template → bootstrap → `clawkit.json`.
- **`internal/installer/registry.go`** — Registry load (embedded + remote + local override), source resolution (`findLocalSkill`, `skills.FindSkill`), `downloadSkill`, `installLocalRuntime` / `installEmbeddedRuntime`, `alwaysExclude`.
- **`internal/installer/lockdown.go`** — Allowlist only. `SetupWorkspace` appends to `agents.defaults.skills`; `RemoveFromWorkspace` removes the entry and clears it when empty.
- **`internal/runtime/`** — Shared-runtime install/purge under `~/.clawkit/runtimes/<key>/`, `_cli.json` parsing, bin symlinking into `~/.clawkit/bin/`, exclude / data_paths logic.
- **`internal/archive/`** — `tar.gz` / `zip`; strips top-level dir.
- **`internal/config/`** — `SkillConfig { Version, Group, UserInputs }`, OpenClaw detection, `Preflight`.
- **`internal/template/`** — `Process()` replaces `{key}` placeholders in the installed `SKILL.md`.
- **`internal/dashboard/`** — Web dashboard served by `clawkit dashboard`.
- **`internal/ui/`** — ANSI terminal helpers.
- **`skills/`** — Built-in skills grouped by vertical (`ecommerce`, `finance`, `real-estate`, `sme`, `study-aboard`, `utilities`). Each vertical must be in the `//go:embed` directive at [skills/skills.go](skills/skills.go).
- **`TEMPLATE.md`** — Reference for authoring new skills (layout, file purposes).

## Adding a skill

1. Create `skills/<name>/` (flat) or `skills/<group>/<name>/` (grouped).
2. Author `SKILL.md` and `_config.json`. For grouped skills, add `_bootstrap/`, `_cli/`, `_cli.json` at the group level (shared by every member).
3. `make generate`.
4. If you added a new top-level vertical directory, add it to the `//go:embed` directive in [skills/skills.go](skills/skills.go).

See [TEMPLATE.md](TEMPLATE.md) for each file's shape.

## Non-obvious invariants

- **Never edit `internal/installer/registry.json` by hand** — regenerated by `make generate`.
- **`_config.json`, `_cli/`, `_cli.json`, `_bootstrap/` never land in the installed skill dir** — all four are in `alwaysExclude` in [internal/installer/registry.go](internal/installer/registry.go).
- **Uninstall does NOT purge the shared runtime.** Data (SQLite DBs etc.) survives; use `clawkit purge <key>` for explicit cleanup.
- **Runtime update preserves `data_paths`.** Re-running `clawkit install` or `clawkit update` overwrites binaries and code but leaves any path listed in `data_paths` untouched.
- **On Windows, runtime bins are copied, not symlinked.** Symlinks there usually require admin.
- **`registry.json` is embedded into the binary via `//go:embed`.** The binary ships a snapshot; a newer remote registry at `raw.githubusercontent.com/.../registry.json` overlays it at runtime.

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

1. `make release-check` — local dry run (`fmt + check-generate + test + dist`).
2. `make bump V=x.y.z` — update VERSION in `Makefile` and `npm/package.json` in one shot (prevents drift).
3. Commit, tag `vx.y.z`, push tag.

Pushing the `v*` tag triggers the release workflow: cross-compile all binaries, upload per-arch `.tar.gz` (macOS/Linux) + `.exe` (Windows) to the GitHub Release, and `npm publish` as `@rockship/clawkit`. The workflow also `sed`s the Makefile VERSION in-place at runtime as a safety net; always bump first so the repo and the release agree.
