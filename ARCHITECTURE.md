# Architecture

Technical reference for contributors and developers.

---

## System Overview

```text
User Machine                              External
┌────────────────────────────────────┐   ┌──────────────────┐
│ clawkit CLI (Go)                   │   │ npm registry     │
│   ├── install / update / uninstall │◄──│ @rockship/       │
│   ├── purge (shared engine)       │   │   clawkit        │
│   ├── registry lookup              │   │ (binaries +      │
│   └── template render              │   │  skills/)        │
│                                    │   └──────────────────┘
│                                    │
│ Workspace (OpenClaw)               │
│   ├── skills/<skill>/              │
│   │     SKILL.md (baked), clawkit.json
│   └── IDENTITY.md, SOUL.md, …      │   ← from _bootstrap/
│                                    │
│ ~/.clawkit/                        │
│   ├── engines/<key>/              │   ← shared _engine/ payload
│   │     binary, sa-data/sa.db, …   │
│   └── bin/                         │   ← symlinks on PATH
│                                    │
│ OpenClaw runtime                   │
│   ├── AI agent (reads SKILL.md)    │
│   └── Channels                     │
└────────────────────────────────────┘
```

Three install destinations, each with a different lifetime:

| Destination | What lands there | Lifetime |
|-------------|------------------|----------|
| `<workspace>/skills/<skill>/` | `SKILL.md` (with baked placeholders), `clawkit.json` | Per skill; removed on uninstall |
| `<workspace>/` (root) | `_bootstrap/*.md` (IDENTITY, SOUL, …) | Overwritten every install |
| `~/.clawkit/engines/<key>/` | `_engine/` payload (binaries, data) | Shared; survives uninstall; removed only by `clawkit purge` |

`~/.clawkit/bin/` holds symlinks to `bins` in each runtime and is added to `PATH`.

---

## Install Flow

```text
clawkit install <name> [<member>…]
  │
  ├── resolveInstallTargets: flat skill? whole group? selected members?
  │
  └── for each target skill:
      ├── 1. Preflight: detect OpenClaw (config.Preflight)
      ├── 2. Registry lookup: load SkillInfo from registry
      ├── 3. downloadSkill:
      │       ├── copy skills/<skill>/ into <workspace>/skills/<skill>/
      │       │   skipping: config.json, _engine/, engine.json, _bootstrap/
      │       └── install the shared engine (installEngine):
      │              ├── key = group name (grouped) or skill name (flat)
      │              ├── engine.Install copies _engine/ → ~/.clawkit/engines/<key>/
      │              │     honoring spec.Exclude and spec.DataPaths
      │              └── engine.LinkBins symlinks each spec.Bins entry into
      │                    ~/.clawkit/bin and ensures it is on PATH
      ├── 4. installRequiredBins: for bins declared in SKILL.md that don't
      │       come from the engine (e.g. `gog`), fetch from GitHub Releases
      ├── 5. Prompt: collect user_inputs interactively
      ├── 6. Allowlist: openclaw config set agents.defaults.skills […]
      ├── 7. Template: replace {key} placeholders in the installed SKILL.md
      ├── 8. applyBootstrap: copy _bootstrap/*.md → workspace root
      └── 9. Save clawkit.json: { version, group, user_inputs }
```

**Update** re-runs steps 3, 7, 9 and reuses the stored `user_inputs` — setup prompts are not asked again. The shared engine is re-copied (non-data files) but `data_paths` entries are preserved, so a SQLite DB or anything else the CLI persists survives updates untouched.

**Uninstall** removes `<workspace>/skills/<skill>/` and pulls the skill from the allowlist. The shared engine is **not** touched — other group members may still need it, and blowing it away would destroy user data. Use `clawkit purge <key>` to remove it explicitly.

---

## Skill Layout

**Flat skill:**

```text
skills/<skill>/
  _bootstrap/           Persona .md files copied to workspace root on install
  _engine/                 Runtime payload (binary, data, …) — shared
  engine.json             Runtime metadata: { exclude, data_paths, bins }
  config.json          Dev metadata: { version, setup_prompts }
  SKILL.md              Frontmatter + agent prompt
```

**Grouped skills** share `_bootstrap/`, `_engine/`, and `engine.json` at the group level:

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

The installer searches flat first, then one level of group nesting. For grouped skills, all three shared artifacts are read from the group level; the child skill directory only contains `config.json` + `SKILL.md`.

---

## Metadata

Metadata lives in **four** files, each read by a different consumer:

### `SKILL.md` frontmatter (OpenClaw-native)

```yaml
---
name: my-skill
description: What this skill does
metadata:
  openclaw:
    os: [darwin, linux, windows]
    requires:
      bins: [node]
      config: []
---
```

Consumed by the agent runtime at read-time and by `gen-registry` at build-time. Fields `os`, `requires.bins`, `requires.config` flow into the registry.

### `config.json` (clawkit dev-time)

```json
{
  "version": "1.0.0",
  "setup_prompts": [{"key": "shop_name", "label": "Shop name"}]
}
```

Consumed only by `gen-registry`. Never copied into the installed skill directory.

### `engine.json` (runtime install rules)

```json
{
  "exclude":    ["cmd"],
  "data_paths": ["sa-data"],
  "bins":       ["sa-cli"]
}
```

Consumed by the engine installer:

- `exclude` — paths inside `_engine/` skipped on install (e.g. source directories like `cmd/`).
- `data_paths` — paths preserved across re-installs (shared databases, user-written files).
- `bins` — names to chmod `+x` and symlink into `~/.clawkit/bin/`.

### `registry.json` (generated)

Generated from `SKILL.md` + `config.json` by `cmd/gen-registry`. Each entry:
`{ description, os, requires_bins, requires_config, version, setup_prompts }`.
Regenerate with `make generate`; CI enforces sync via `make check-generate`. Never edit by hand.

### `clawkit.json` (install-time)

Written into every installed skill directory:

```json
{
  "version":     "1.0.0",
  "group":       "study-aboard",
  "user_inputs": { "shop_name": "Hoa Xuan" }
}
```

`user_inputs` is preserved across `clawkit update` so placeholders are re-baked without re-prompting. `group` records the group the skill was installed from (empty for flat skills).

---

## Registry Generation

`cmd/gen-registry` walks `skills/` recursively. For each directory containing a `SKILL.md`:

1. Parse the YAML frontmatter with a hand-written indent-aware parser (zero deps) — supports nested maps and inline flow arrays `[a, b, c]`.
2. Flatten `metadata.openclaw.os`, `metadata.openclaw.requires.bins`, `metadata.openclaw.requires.config` into the entry.
3. Read the sibling `config.json` for `version` and `setup_prompts`.
4. Emit the merged record keyed by directory name.

A directory is additionally recorded as a **group** when it holds a `_engine/` *and* at least one child directory contains a `SKILL.md`. `_engine/` directories themselves are never scanned.

The directory name is the canonical key; the `name:` field in frontmatter is informational only.

---

## Skill Resolution Order

When installing `<name>`:

1. **Local dev** — `findLocalSkill` checks `skills/<name>/`, then one level of nesting (`skills/<vertical>/<name>/`), returning the first match with a `SKILL.md`.
2. **Packaged** — `findSkillIn(os.Getenv("CLAWKIT_SKILLS_DIR"), name)` — the npm wrapper points this at `<npm-package>/skills/` (same nesting rule).

The engine source is determined independently: if the skill directory contains `_engine/`, key = skill name; else if the parent directory contains `_engine/`, key = parent directory name (the group); else no engine is installed. No network fetch.

---

## Shared-Engine Lifecycle

| Event | Effect on `~/.clawkit/engines/<key>/` |
|-------|----------------------------------------|
| First install of any skill with runtime key `<key>` | Full copy from source (honoring `exclude`, skipping `data_paths` if already populated) |
| Subsequent install / update of another member | Same copy; non-data files overwritten; `data_paths` preserved |
| Uninstall of any skill | Untouched — other members may still reference it |
| `clawkit purge <key>` | Directory removed, symlinks in `~/.clawkit/bin/` removed |

Data preservation is a design choice: the shared engine owns user state (databases, generated files), and the installer must not stomp on it during routine updates. Explicit purge is the only way to reset.

`~/.clawkit/bin/` is added to the user's `PATH` via the same `ensureInPath` helper that installs the `gog` CLI — shell profile append on Unix, `setx` on Windows.

---

## Workspace Allowlist

clawkit updates OpenClaw's skill allowlist so installed skills appear in `<available_skills>`:

- **Install** appends the skill to `agents.defaults.skills`.
- **Uninstall** removes it; when the last skill is removed, the allowlist entry is cleared (`openclaw config unset agents.defaults.skills`).

---

## Key Packages

| Package | Responsibility |
|---------|----------------|
| `cmd/clawkit` | CLI dispatcher: `list`, `install`, `update`, `uninstall`, `purge`, `status`, `web`, `dashboard`, `version` |
| `cmd/gen-registry` | Registry generator + hand-written YAML frontmatter parser |
| `internal/archive` | `tar.gz` / `zip` extraction and creation (strips top-level dir) |
| `internal/config` | `SkillConfig { Version, Group, UserInputs }`, OpenClaw detection, `Preflight` |
| `internal/installer` | All commands. `commands.go` is the orchestrator; `registry.go` handles source resolution and download; `lockdown.go` manages the allowlist |
| `internal/engine` | Shared-engine install/purge, `engine.json` parsing, bin symlinking |
| `internal/template` | `Process()` — replaces `{key}` placeholders in installed SKILL.md |
| `internal/dashboard` | Web dashboard served by `clawkit dashboard` |
| `internal/ui` | ANSI terminal output helpers |

---

## Directory Structure

```text
clawkit/
  cmd/
    clawkit/                CLI entry point
    gen-registry/           Registry generator + frontmatter parser
  internal/
    archive/                tar.gz / zip
    config/                 SkillConfig, OpenClaw detection
    installer/              Commands, registry, allowlist
    runtime/                Shared runtime (~/.clawkit/engines, ~/.clawkit/bin)
    template/               {key} placeholder substitution
    dashboard/              Web dashboard
    ui/                     Terminal output helpers
  skills/                   Built-in skills, grouped by vertical
  TEMPLATE.md               Reference for authoring a new skill
  npm/                      npm package: wrapper, binaries/, staged skills/
```

---

## Cross-Platform Rules

| Concern | Do | Don't |
|---------|----|-------|
| File paths | `filepath.Join(a, b)` | `a + "/" + b` |
| Temp directory | `os.TempDir()` | Hardcode `/tmp` |
| Binary names | Append `.exe` on Windows | Assume no `.exe` |
| Unix-only syscalls | Guard with `runtime.GOOS != "windows"` | Call `chmod`, `sudo` unconditionally |
| Archive format | `.zip` on Windows, `.tar.gz` elsewhere | Assume `.tar.gz` |
| Bin linking | Symlink on Unix, copy on Windows | Symlink unconditionally |

---

## Zero External Dependencies

The Go module uses only the standard library. The YAML frontmatter parser in `cmd/gen-registry` is hand-written. Do not add external dependencies without discussion.

---

## Release

1. `make release-check` — local dry run: `fmt + check-generate + test + npm-stage`.
2. `make bump V=1.2.0` — syncs VERSION in `Makefile` and `npm/package.json` so dev view and published view can't drift.
3. Commit, tag `v1.2.0`, push tag:

```bash
git commit -am 'Release v1.2.0'
git tag v1.2.0
git push && git push --tags
```

The `v*` tag triggers `.github/workflows/release.yml`:

- Re-runs `make check-generate` and `make test` on the tag.
- Runs `make npm-stage`, which in turn:
  - `make dist` — cross-compiles 5 binaries into `dist/`.
  - Copies `dist/clawkit-*` into `npm/binaries/`.
  - Copies `skills/` into `npm/skills/` (dropping `skills.go`).
  - Copies `internal/installer/registry.json` into `npm/registry.json`.
  - Runs `npm version` so `npm/package.json` matches the tag.
- Runs `npm publish --access restricted` from `npm/` using `${{ secrets.NPM_TOKEN }}`.

Distribution is npm-only, which means the GitHub repo can stay **private** — npm (or GitHub Packages if you swap the registry) handles auth for end users. Users install via `npm install -g @rockship/clawkit`; the binary ends up under `<node>/lib/node_modules/@rockship/clawkit/binaries/…` and is invoked through the wrapper at `<node>/bin/clawkit`. Skill files live in the same npm package (`<pkg>/skills/`) and are located at runtime via the `CLAWKIT_SKILLS_DIR` env var set by the wrapper — no network fetch, no cache layer.

The workflow also `sed`s the Makefile VERSION in-place at runtime as a safety net, but the canonical flow is `make bump` first.
