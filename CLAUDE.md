# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make build          # Build binary for current platform → ./clawkit
make test           # Run all tests
make fmt            # go fmt + go vet
make generate       # Regenerate registry.json from skills/**/SKILL.md
make check-generate # Verify registry.json is in sync (CI check)
make dist           # Cross-compile for darwin/linux/windows
make package SKILL=<name>  # Package a skill to .tar.gz
```

Run a single test package:
```bash
CGO_ENABLED=0 go test -v ./internal/archive/...
```

**Important:** After editing any `SKILL.md` frontmatter, always run `make generate` to keep `registry.json` in sync. CI will fail if they diverge (`make check-generate`).

## Architecture

Clawkit is a CLI skill manager for OpenClaw AI agents. Skills are AI prompt files with an install lifecycle (OAuth, config, schema-driven DB, templates). The binary is distributed via npm wrapping platform-specific Go binaries (`npm/binaries/`).

### Standard flow

Go (installer) + Node.js (runtime) + schema.json (data model). No Python.

### Data flow

1. `registry.json` is generated from `skills/**/SKILL.md` YAML frontmatter by `cmd/gen-registry`. Never edit it by hand.
2. At install time, `internal/installer` fetches the registry, downloads the skill package, applies profile overlay, runs OAuth, initializes DB from `schema.json`, processes templates, and saves `clawkit.json` per installed skill.
3. At runtime, `cli.js` (generic, schema-driven) reads `schema.json` + `clawkit.json` and performs CRUD operations against the configured store backend (local JSON, Supabase, or custom API).

### Key packages

- **`cmd/clawkit/main.go`** — CLI dispatcher (list, install, update, uninstall, status, package, version)
- **`internal/installer/commands.go`** — All command implementations; install flow: preflight → download → profile overlay → OAuth → lockdown → schema init → config save → template processing
- **`internal/installer/schema.go`** — Schema parsing, validation, multi-table merge, DB initialization, credential collection. Constants: `DBTargetLocal`, `DBTargetSupabase`, `DBTargetAPI`
- **`internal/installer/profile.go`** — Profile overlay: catalog, schema (with extend-merge), images, bootstrap-files
- **`internal/installer/registry.go`** — Registry loading (remote + embedded + local) and skill package downloading. Supports nested vertical dirs via `findLocalSkill()`
- **`internal/installer/lockdown.go`** — 1-skill-at-a-time workspace lockdown: remove prior, backup, override, reset sessions, set allowlist
- **`internal/archive/`** — tar.gz / zip extraction and creation; strips top-level directory from archives
- **`internal/config/`** — `SkillConfig` struct (skill_name, profile, version, db_target, oauth_done, tokens, user_inputs), config file read/write, OpenClaw detection
- **`internal/template/`** — SKILL.md placeholder substitution; `catalog.json` loading; `EnsureImageDirs` (reads images_dir from schema.json)
- **`internal/ui/`** — ANSI terminal output helpers (Info/Ok/Warn/Fatal) and `PromptInput`
- **`oauth/`** — OAuth providers; each self-registers via `init()`. Add a new provider by creating a new file and calling `Register()` in its `init()`
- **`skills/`** — Built-in skills grouped by vertical (ecommerce/, utilities/, tools/)
- **`templates/`** — Generic `cli.js` and per-vertical schemas (ecommerce, education, consulting, gold, food-distribution)

### Skills directory structure

Skills are grouped by vertical under `skills/`:

```
skills/
  ecommerce/
    shop-hoa/
    carehub-baby/
  utilities/
    finance-tracker/
  tools/
    gog/
```

The registry generator (`cmd/gen-registry`) scans recursively. The installer (`findLocalSkill`) searches one level of nesting. The embed directive in `skills/skills.go` uses vertical-level `all:` directives.

### Adding a skill

1. Pick a vertical or create a new one under `skills/`.
2. Copy from `templates/verticals/<vertical>/` or create `SKILL.md` + `config.json` + `schema.json` + copy `templates/cli.js`.
3. Run `make generate`.
4. Update the `//go:embed` directive in `skills/skills.go` if adding a new vertical.
5. Add any OAuth providers to `oauth/` if they don't exist.

### Skill metadata split

Skill metadata is split between two files:

- **`SKILL.md` frontmatter** — OpenClaw-native fields only: `name`, `description`, `metadata` (including `metadata.openclaw.emoji`, `metadata.openclaw.requires.bins`, etc.)
- **`config.json`** (dev source) — Clawkit-specific fields: `version`, `requires_bins`, `setup_prompts`, `exclude`. After installation, this becomes `clawkit.json` in the installed skill directory.

`registry.json` is generated from both sources by `cmd/gen-registry`. The `name` and `description` come from SKILL.md; everything else comes from `config.json`.

Example `config.json`:
```json
{
  "version": "1.0.0",
  "requires_bins": ["gog"],
  "setup_prompts": [{"key": "name", "label": "Your name"}],
  "exclude": ["cmd", "tools", "*.tmp"]
}
```

The `exclude` patterns use `filepath.Match` syntax and are applied during `clawkit install` (copyDir, copyEmbeddedSkill) and `clawkit package` (CreateTarGz). Patterns match against both full relative paths and individual path components.

### Schema system

`schema.json` defines the data model. Supports multi-table:

```json
{
  "tables": {
    "orders": { "fields": [...], "statuses": [...] },
    "contacts": { "fields": [...] }
  },
  "primary": "orders",
  "timezone": "Asia/Ho_Chi_Minh",
  "images_dir": "products"
}
```

Field types: `text`, `integer`. Auto values: `increment`, `timestamp`. Roles: `owner`, `status`, `price`, `timestamp`. Ref: `"ref": "other_table"` (documentation only, not enforced).

Profile schemas can use `"extend": true` to add fields/tables to a base schema.

### Store backends

- `local` — JSON files (1 per table), created at install time
- `supabase` — Supabase REST API, credentials prompted at install
- `api` — Generic REST API, customer provides endpoint + auth header

`cli.js` uses `--table <name>` to target non-primary tables.

### Profile system

`clawkit install <skill> --profile <name>` overlays domain-specific files from `profiles/<name>/` onto the base skill:

- `profile.yaml` — key-value pairs merged into template placeholders
- `catalog.json` — product catalog override
- `schema.json` — schema override (supports extend-merge)
- Images directory — product images override
- `bootstrap-files/` — agent persona override

### Adding an OAuth provider

Implement the `oauth.Provider` interface (`Name()`, `Display()`, `Authenticate() (map[string]string, error)`) and call `oauth.Register(yourProvider{})` in `init()`. The returned map is merged into the skill's `clawkit.json` tokens.

### Cross-platform rules

| Concern | Do | Don't |
|---|---|---|
| File paths | `filepath.Join(a, b)` | `a + "/" + b` |
| Temp directory | `os.TempDir()` | Hardcode `/tmp` |
| Binary names | `name := "gog"; if goos == "windows" { name += ".exe" }` | Assume no `.exe` |
| Open browser | `oauth.OpenBrowser(url)` (handles all 3 platforms) | Call `open`/`xdg-open` directly |
| Unix-only syscalls | Guard with `if goos != "windows"` | Call `chmod`, `sudo` unconditionally |
| Archive format | `.zip` on Windows, `.tar.gz` elsewhere | Assume tar.gz |

### Zero external dependencies

The Go module uses only the standard library. The YAML frontmatter parser in `cmd/gen-registry` is hand-written. The Node.js `cli.js` uses only built-in modules. Do not add external dependencies without discussion.

### Release

Releases are triggered by pushing a `v*` tag. The release workflow cross-compiles all platform binaries (`make dist`), packages skills, creates a GitHub Release, and publishes to npm as `@rockship/clawkit`.
