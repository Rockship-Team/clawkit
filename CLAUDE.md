# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make build          # Build binary for current platform → ./clawkit
make test           # Run all tests
make fmt            # go fmt + go vet
make generate       # Regenerate registry.json from skills/**/{SKILL.md,config.json}
make check-generate # Verify registry.json is in sync (CI check)
make dist           # Cross-compile for darwin/linux/windows
```

Run a single test package:

```bash
CGO_ENABLED=0 go test -v ./internal/archive/...
```

**Important:** After editing any `SKILL.md` frontmatter or `config.json`, always run `make generate` to keep `registry.json` in sync. CI will fail if they diverge (`make check-generate`).

## Architecture

Clawkit is a CLI skill manager for OpenClaw AI agents. Skills are AI prompt files (`SKILL.md`) plus a runtime directory (`_cli/`) in any language. The binary is distributed via npm wrapping platform-specific Go binaries (`npm/binaries/`).

### Data flow

1. `registry.json` is generated from `skills/**/SKILL.md` frontmatter and the sibling `config.json` by `cmd/gen-registry`. Never edit it by hand.
2. At install time, `internal/installer` fetches the registry, downloads the skill package, merges the group's `_cli/` when relevant, collects `setup_prompts` interactively, updates the OpenClaw allowlist, bakes user inputs into `SKILL.md`, and saves `clawkit.json` in the installed skill directory.
3. At runtime, whatever is in the skill's `_cli/` directory runs however the skill's `SKILL.md` tells the agent to invoke it.

### Key packages

- **`cmd/clawkit/main.go`** — CLI dispatcher (list, install, update, uninstall, status, package, web, dashboard, version). `install` / `update` accept `<name> [<member>...]`: name resolves to either a flat skill or a group; trailing args select specific group members.
- **`cmd/gen-registry/main.go`** — Scans `skills/**/SKILL.md` + `config.json`, emits `internal/installer/registry.json` with both `skills` and `groups` sections. Hand-written indent-aware YAML parser; skips `_cli/` directories. A directory is recorded as a group when it holds a `_cli/` and at least one child with a `SKILL.md`.
- **`internal/installer/commands.go`** — All command implementations. Install flow: preflight → download → group `_cli` merge → install bins → collect setup prompts → allowlist → template processing → save `clawkit.json`.
- **`internal/installer/registry.go`** — Registry loading (remote + embedded + local), skill package download, `findLocalSkill` (flat + one level of group nesting), `mergeGroupCLI` / `mergeEmbeddedGroupCLI`.
- **`internal/installer/lockdown.go`** — Allowlist management only. `SetupWorkspace` appends the skill to `agents.defaults.skills`; `RemoveFromWorkspace` removes it and clears the entry when the last skill is uninstalled.
- **`internal/archive/`** — tar.gz / zip extraction and creation; strips top-level directory from archives.
- **`internal/config/`** — `SkillConfig{ Version, UserInputs }`, config file read/write, OpenClaw detection.
- **`internal/template/`** — `Process()` — replaces `{key}` placeholders in `SKILL.md` with `user_inputs` values.
- **`internal/dashboard/`** — Web dashboard served by `clawkit dashboard`.
- **`internal/ui/`** — ANSI terminal output helpers.
- **`skills/`** — Built-in skills grouped by vertical.
- **`templates/`** — Scaffolding for new skills: `skill/` (flat) and `group/` (shared `_cli/`).

### Skills directory structure

Flat skill:

```
skills/<skill>/
  _cli/                   cli.js or any runtime helpers
  config.json
  SKILL.md
```

Grouped skills share one `_cli/` at the group level:

```
skills/<group>/
  _cli/                   merged into every child on install
  <skill-a>/
    config.json
    SKILL.md
  <skill-b>/
    config.json
    SKILL.md
```

The registry generator scans recursively and skips any `_cli/` directory. The installer searches flat first, then one level of group nesting, and merges the group's `_cli/` when the child has none.

### Adding a skill

1. Copy `templates/skill/` for a flat skill or `templates/group/` for a group.
2. Drop it under `skills/` (optionally under a vertical directory).
3. Edit `SKILL.md` (frontmatter + agent prompt) and `config.json`.
4. Run `make generate`.
5. Update the `//go:embed` directive in `skills/skills.go` if adding a new vertical directory.

### Metadata split

Metadata is split across three files:

**`SKILL.md` frontmatter** — OpenClaw-native fields:

```yaml
---
name: my-skill
description: One-line purpose
metadata:
  openclaw:
    os: [darwin, linux, windows]
    requires:
      bins: [node]
      config: []
---
```

**`config.json`** (dev-only) — clawkit-specific fields:

```json
{
  "version": "1.0.0",
  "setup_prompts": [{"key": "shop_name", "label": "Shop name"}],
  "exclude": ["*.tmp"]
}
```

**`clawkit.json`** (installed) — written by the installer into the installed skill directory:

```json
{
  "version": "1.0.0",
  "user_inputs": {"shop_name": "Hoa Xuan"}
}
```

`user_inputs` is preserved across `clawkit update` so placeholders are re-baked into the new `SKILL.md` without re-prompting. `config.json` is excluded from the installed directory — only `clawkit.json` lives there.

### Registry entry shape

Each entry in `registry.json` contains: `description`, `os`, `requires_bins`, `requires_config`, `version`, `setup_prompts`, `exclude`. `os` / `requires_bins` / `requires_config` come from `SKILL.md` frontmatter; `version` / `setup_prompts` / `exclude` come from `config.json`. The directory name is the registry key; the `name:` field in frontmatter is informational only.

### Exclude patterns

`exclude` in `config.json` uses `filepath.Match` syntax and is applied during `clawkit install` (`copyDir`, `copyEmbeddedSkill`). Patterns match against both full relative paths and individual path components; `**/` prefix supported.

### Cross-platform rules

| Concern | Do | Don't |
|---|---|---|
| File paths | `filepath.Join(a, b)` | `a + "/" + b` |
| Temp directory | `os.TempDir()` | Hardcode `/tmp` |
| Binary names | `name := "gog"; if goos == "windows" { name += ".exe" }` | Assume no `.exe` |
| Unix-only syscalls | Guard with `if goos != "windows"` | Call `chmod`, `sudo` unconditionally |
| Archive format | `.zip` on Windows, `.tar.gz` elsewhere | Assume tar.gz |

### Zero external dependencies

The Go module uses only the standard library. The YAML frontmatter parser in `cmd/gen-registry` is hand-written. Do not add external dependencies without discussion.

### Release

Releases are triggered by pushing a `v*` tag. The release workflow cross-compiles all platform binaries (`make dist`), packages skills, creates a GitHub Release, and publishes to npm as `@rockship/clawkit`.
