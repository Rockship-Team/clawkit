# clawkit

CLI skill manager for [OpenClaw](https://docs.openclaw.ai) AI agents. Install, configure, and manage AI skills with one command.

```bash
npm install -g @rockship/clawkit
```

Built by [Rockship](https://rockship.co) | [Architecture](./ARCHITECTURE.md) | [Templates](./templates/README.md)

---

## Requirements

- **OpenClaw** — [install guide](https://docs.openclaw.ai/installation)
- Any runtime your skill's `_cli/` needs (Node.js, Go, Python, …)

---

## Quick Start

```bash
clawkit list                    # See available skills
clawkit install ecom-bot        # Install (prompts for setup values)
clawkit status                  # Check installed skills
clawkit update ecom-bot         # Update to latest, keep your settings
clawkit uninstall ecom-bot
```

---

## Commands

| Command | Description |
|---------|-------------|
| `clawkit list` | List available skills and groups |
| `clawkit install <skill>` | Install a flat skill |
| `clawkit install <group>` | Install every skill in a group |
| `clawkit install <group> <member>...` | Install selected members of a group |
| `clawkit update <name> [<member>...]` | Update (same resolution as install) |
| `clawkit uninstall <skill>` | Uninstall and remove from the allowlist |
| `clawkit status` | Show installed skills |
| `clawkit dashboard` | Start the local web dashboard |
| `clawkit web <skill>` | Serve a skill's `web/` directory |
| `clawkit version` | Print version |

---

## Skill Layout

Flat skill:

```
skills/<skill>/
  _cli/                 cli.js or any runtime helpers
  config.json           { version, setup_prompts, exclude }
  SKILL.md              frontmatter + agent prompt
```

Grouped skills share one `_cli/`:

```
skills/<group>/
  _cli/                 shared runtime for every child skill
  <skill-a>/
    config.json
    SKILL.md
  <skill-b>/
    config.json
    SKILL.md
```

The installer merges the group's `_cli/` into each child at install time.

---

## Metadata

Skill metadata is split into two files so each consumer only sees what it
needs:

**`SKILL.md` frontmatter** (OpenClaw-native):

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

**`config.json`** (clawkit-only dev metadata):

```json
{
  "version": "1.0.0",
  "setup_prompts": [{"key": "shop_name", "label": "Shop name"}],
  "exclude": ["*.tmp"]
}
```

**`registry.json`** is generated from both by `make generate`. CI enforces
sync via `make check-generate`.

**`clawkit.json`** is written into the installed skill directory and holds
`{ version, user_inputs }` — used on update to re-bake placeholders without
re-prompting.

---

## Creating a New Skill

```bash
cp -r templates/skill skills/my-skill       # flat
# or
cp -r templates/group skills/my-group       # grouped

# Edit SKILL.md, config.json, and _cli/ to taste.

make generate                               # Refresh registry.json
make build                                  # Build the CLI
./clawkit install my-skill                  # Try it
```

See [templates/README.md](templates/README.md) for the scaffold layout.

---

## Project Structure

```
cmd/
  clawkit/              CLI entry point
  gen-registry/         Registry generator (frontmatter + config.json → registry.json)
internal/
  archive/              tar.gz / zip
  config/               SkillConfig, OpenClaw detection
  installer/            Install, update, uninstall, registry, allowlist
  template/             {key} placeholder substitution in SKILL.md
  dashboard/            Web dashboard
  ui/                   Terminal output helpers
skills/                 Built-in skills (grouped by vertical)
templates/              Skill scaffolding (flat + grouped examples)
npm/                    npm package wrapper with platform binaries
```

---

## Development

```bash
make build          # Build binary → ./clawkit
make test           # Run tests
make fmt            # go fmt + go vet
make generate       # Regenerate registry.json from skills/
make check-generate # CI check: registry.json is in sync
make dist           # Cross-compile for all platforms
```

### Key Constraints

- **Zero external Go dependencies** — stdlib only (the frontmatter parser
  is hand-written)
- **Cross-platform** — macOS, Linux, Windows (arm64 + amd64)

---

## Release

```bash
git tag v1.2.0
git push origin v1.2.0
```

GitHub Actions cross-compiles, creates a Release, and publishes to npm as
`@rockship/clawkit`.

---

## License

MIT
