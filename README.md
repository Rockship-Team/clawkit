# clawkit

CLI skill manager for [OpenClaw](https://docs.openclaw.ai) AI agents. Install, configure, and manage AI skills with one command.

```bash
npm install -g @rockship/clawkit
```

The package ships platform binaries, skill files, and `registry.json`. The `clawkit` command (a tiny Node wrapper) resolves the right binary for your OS/arch and points it at the skills shipped in the same package. Skill runtimes (the `_engine/` payloads) still land in `~/.clawkit/engines/` and `~/.clawkit/bin` is added to your `PATH` at first install.

Built by [Rockship](https://rockship.co) | [Architecture](./ARCHITECTURE.md) | [Template](./TEMPLATE.md)

---

## Requirements

- **OpenClaw** — [install guide](https://docs.openclaw.ai/installation)
- Any runtime your skill's `_engine/` needs (Node.js, Go, Python, …)

---

## Quick Start

```bash
clawkit list                           # See available skills and groups
clawkit install ecom-bot               # Install a flat skill
clawkit install study-aboard           # Install every member of a group
clawkit install study-aboard essay-review profile-assessment
clawkit status                         # Check installed skills
clawkit update ecom-bot                # Update, keep stored setup values
clawkit uninstall ecom-bot
clawkit purge study-aboard             # Delete a shared engine (incl. data)
```

---

## Commands

| Command | Description |
|---------|-------------|
| `clawkit list` | List available skills and groups |
| `clawkit install <name> [<member>…]` | Install a flat skill, a whole group, or selected members |
| `clawkit update <name> [<member>…]` | Update (same resolution as install); stored `user_inputs` are kept |
| `clawkit uninstall <skill>` | Remove a skill and its allowlist entry (shared engine is preserved) |
| `clawkit purge <key>` | Delete `~/.clawkit/engines/<key>` and its symlinked bins |
| `clawkit status` | Show installed skills |
| `clawkit dashboard [--port N]` | Start the local web dashboard |
| `clawkit web <skill>` | Serve a skill's `web/` directory |
| `clawkit version` | Print version |

---

## Skill Layout

A skill owns an AI prompt (`SKILL.md`) and optionally a runtime (`_engine/`), persona files (`_bootstrap/`), and dev-time metadata (`config.json`).

**Flat skill:**

```
skills/<skill>/
  _bootstrap/           Persona .md files copied to the workspace root on install
  _engine/                 Runtime payload (binary, data, …) — installed to ~/.clawkit
  engine.json             Runtime metadata: { exclude, data_paths, bins }
  config.json          Dev metadata: { version, setup_prompts }
  SKILL.md              Frontmatter + agent prompt
```

**Grouped skills** share `_bootstrap/`, `_engine/`, and `engine.json` at the group level:

```
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

All four shared files live **only** at the group level; the installed skill directory never contains a copy.

---

## Install Mechanics

`clawkit install` does three distinct things:

1. **Skill files** → `<OpenClaw workspace>/skills/<skill>/` — contains `SKILL.md` (with `{key}` placeholders baked in) and `clawkit.json` (version + group + user_inputs).
2. **`_bootstrap/` .md files** → workspace root — overwriting any existing files of the same name.
3. **`_engine/` payload** → `~/.clawkit/engines/<key>/` — one shared copy per skill (key = skill name) or per group (key = group name). Binaries listed in `engine.json#bins` are symlinked into `~/.clawkit/bin`, which is added to `PATH`. Paths listed in `data_paths` (e.g. a SQLite DB) are preserved across re-installs so user state survives updates.

The result: every member of a group sees the *same* `sa-cli` binary and the *same* `sa.db` — no duplicated runtimes, no diverged databases.

---

## Metadata

Skill metadata lives in three files, each read by a different consumer:

**`SKILL.md` frontmatter** — OpenClaw-native, consumed by the agent runtime and by `gen-registry`:

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

**`config.json`** — clawkit dev-time metadata (never copied to the install):

```json
{
  "version": "1.0.0",
  "setup_prompts": [{"key": "shop_name", "label": "Shop name"}]
}
```

**`engine.json`** — runtime install rules (colocated with `_engine/`):

```json
{
  "exclude":    ["cmd"],
  "data_paths": ["sa-data"],
  "bins":       ["sa-cli"]
}
```

**`registry.json`** — generated from `SKILL.md` + `config.json` by `make generate`. CI enforces sync (`make check-generate`).

**`clawkit.json`** — written into each installed skill dir by the installer:

```json
{
  "version":     "1.0.0",
  "group":       "study-aboard",
  "user_inputs": { "shop_name": "Hoa Xuan" }
}
```

Used by `clawkit update` to re-bake placeholders without re-prompting.

---

## Creating a New Skill

```bash
mkdir -p skills/my-skill                    # flat
# or
mkdir -p skills/my-group/my-skill           # grouped (shared engine at skills/my-group/)

# Author SKILL.md, config.json, and (optionally) engine.json + _engine/.

make generate                               # Refresh registry.json
make build                                  # Build the CLI
./clawkit install my-skill                  # Try it
```

See [TEMPLATE.md](TEMPLATE.md) for each file's shape and purpose.

---

## Project Structure

```text
cmd/
  clawkit/              CLI entry point
  gen-registry/         Registry generator (SKILL.md + config.json → registry.json)
internal/
  archive/              tar.gz / zip
  config/               SkillConfig (clawkit.json), OpenClaw detection
  installer/            Install, update, uninstall, purge, registry, allowlist
  engine/               Shared engine management (~/.clawkit/engines + ~/.clawkit/bin)
  template/             {key} placeholder substitution in SKILL.md
  dashboard/            Web dashboard
  ui/                   Terminal output helpers
skills/                 Built-in skills, grouped by vertical (ecommerce, finance, …)
npm/                    npm package — wrapper, binaries, and staged skills/
```

---

## Development

```bash
make build          # Build binary → ./clawkit
make test           # Run tests
make test-race      # Run tests with the race detector (CGO required)
make fmt            # go fmt + go vet
make generate       # Regenerate registry.json from skills/
make check-generate # CI check: registry.json is in sync
make dist           # Cross-compile for all platforms into dist/
make release-check  # fmt + check-generate + test + dist (dry run)
make help           # List every target
```

### Key Constraints

- **Zero external Go dependencies** — stdlib only (the YAML frontmatter parser is hand-written).
- **Cross-platform** — macOS, Linux, Windows (arm64 + amd64).

---

## Release

```bash
make release-check          # fmt + check-generate + test + npm-stage (dry run)
make bump V=1.2.0           # sync VERSION in Makefile and npm/package.json
git commit -am 'Release v1.2.0'
git tag v1.2.0
git push && git push --tags
```

Pushing the `v*` tag triggers GitHub Actions: cross-compile all binaries, stage `npm/` (binaries + `skills/` + `registry.json`), and `npm publish` as `@rockship/clawkit`. Distribution is npm-only; the GitHub repo can stay private because npm handles the auth.

---

## License

MIT
