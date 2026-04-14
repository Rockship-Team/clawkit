# clawkit

CLI skill manager for [OpenClaw](https://docs.openclaw.ai) AI agents. Install, configure, and manage AI skills with one command.

```bash
npm install -g @rockship/clawkit
```

Built by [Rockship](https://rockship.co) | [Architecture](./ARCHITECTURE.md) | [Templates](./templates/README.md)

---

## Requirements

- **Node.js 18+** — [nodejs.org](https://nodejs.org)
- **OpenClaw** — [install guide](https://docs.openclaw.ai/installation)

---

## Quick Start

```bash
clawkit list                              # See available skills
clawkit install shop-hoa                  # Install a skill
clawkit install ecom-bot --profile bakery # Install with a domain profile
clawkit status                            # Check installed skills
```

---

## Commands

| Command | Description |
|---------|-------------|
| `clawkit list` | List available skills |
| `clawkit install <skill> [--profile <name>] [--skip-oauth]` | Install a skill |
| `clawkit update <skill>` | Update, preserving config and tokens |
| `clawkit uninstall <skill>` | Uninstall and restore workspace |
| `clawkit status` | Show installed skills with profile and OAuth status |
| `clawkit package <skill>` | Package a skill for distribution |
| `clawkit version` | Print version |

---

## Architecture

```
clawkit (Go CLI)
  │
  ├── Install flow: preflight → download → profile overlay → OAuth → lockdown → schema init → config save
  │
  ├── schema.json       Declarative data model (multi-table, field roles, statuses)
  ├── cli.js            Generic Node.js runtime (CRUD, images, Telegram upload)
  └── profile.yaml      Domain-specific overrides (catalog, images, persona)
```

### Storage Backends

Skills support three database targets, configured via `db_target` in profile.yaml:

| Target | Storage | Use Case |
|--------|---------|----------|
| `local` | JSON files (1 per table) | Development, small shops |
| `supabase` | Supabase REST API | Cloud database, no server needed |
| `api` | Customer's own REST API | Existing backend integration |

### Project Structure

```
cmd/
  clawkit/              CLI entry point
  gen-registry/         Registry generator (scans SKILL.md frontmatter)
internal/
  archive/              tar.gz / zip extraction and creation
  config/               SkillConfig struct, OpenClaw detection
  installer/            Install/update/uninstall commands, schema, profiles
  template/             SKILL.md placeholder substitution, catalog processing
  ui/                   Terminal output helpers (Info/Ok/Warn/Fatal)
oauth/                  OAuth providers (self-registering via init())
skills/                 Built-in skills grouped by vertical
  ecommerce/            shop-hoa, carehub-baby
  utilities/            finance-tracker
  tools/                gog (Google Workspace CLI)
templates/              Reusable templates for new skills
  cli.js                Generic schema-driven CLI
  verticals/            Pre-built schemas per business vertical
    ecommerce/          Orders, products, contacts (4 tables)
    education/          Enrollments, courses, contacts (3 tables)
    consulting/         Students, applications, test scores (4 tables)
    gold/               Transactions, products, price board (4 tables)
    food-distribution/  Orders, inventory, products (5 tables)
```

---

## Creating a New Skill

```bash
# 1. Copy a vertical template
cp -r templates/verticals/ecommerce skills/ecommerce/my-shop

# 2. Copy the generic CLI
cp templates/cli.js skills/ecommerce/my-shop/cli.js

# 3. Customize SKILL.md (AI prompt) and schema.json (data model)

# 4. Register and build
make generate
make build
```

See [templates/README.md](templates/README.md) for detailed guides per vertical.

### Schema Format

```json
{
  "tables": {
    "orders": {
      "fields": [
        {"name": "id", "type": "integer", "auto": "increment"},
        {"name": "status", "type": "text", "default": "new", "role": "status"},
        {"name": "customer", "type": "text", "required": true},
        {"name": "total", "type": "integer", "role": "price"},
        {"name": "sender_id", "type": "text", "role": "owner"},
        {"name": "created_at", "type": "text", "auto": "timestamp", "role": "timestamp"}
      ],
      "statuses": ["new", "completed", "cancelled"]
    }
  },
  "primary": "orders",
  "timezone": "Asia/Ho_Chi_Minh"
}
```

### Profiles

Profiles enable one skill base to serve multiple domains:

```bash
clawkit install ecom-bot --profile shop-hoa   # Flower shop
clawkit install ecom-bot --profile bakery     # Bakery
```

Each profile overrides: `catalog.json`, product images, `workspace-overrides/`, `schema.json` (with extend support), and template placeholders via `profile.yaml`.

---

## Development

```bash
make build          # Build binary → ./clawkit
make test           # Run all tests
make fmt            # go fmt + go vet
make generate       # Regenerate registry.json from skills
make check-generate # Verify registry.json is in sync (CI check)
make dist           # Cross-compile for all platforms
```

### Key Constraints

- **Zero external Go dependencies** — stdlib only
- **Cross-platform** — macOS, Linux, Windows (arm64 + amd64)
- **No Python** — all runtime is Go (install) + Node.js (skill CLI)

---

## Release

```bash
git tag v1.2.0
git push origin v1.2.0
```

GitHub Actions cross-compiles, creates a Release, and publishes to npm as `@rockship/clawkit`.

---

## License

MIT
