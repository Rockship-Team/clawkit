# Architecture

Technical reference for contributors and developers.

---

## System Overview

```
User Machine                          External
┌──────────────────────────┐         ┌──────────────────┐
│ clawkit CLI (Go)         │         │ npm registry     │
│   ├── install/update     │◄────────│ GitHub Releases  │
│   ├── schema.json parser │         │ Supabase         │
│   └── profile overlay    │         │ Customer API     │
│                          │         └──────────────────┘
│ cli.js (Node.js)         │
│   ├── JsonStore (local)  │
│   ├── RemoteStore (API)  │
│   └── Telegram upload    │
│                          │
│ OpenClaw Runtime         │
│   ├── AI agent           │
│   ├── Telegram channel   │
│   └── Zalo channel       │
└──────────────────────────┘
```

---

## Install Flow

```
clawkit install <skill> [--profile <name>]
  │
  ├── 1. Preflight: detect OpenClaw
  ├── 2. Registry lookup: load skill metadata
  ├── 3. Dependency check: verify requires_skills installed
  ├── 4. Download: local dev → embedded → GitHub Releases
  ├── 5. Profile overlay: catalog, schema, images, workspace-overrides
  ├── 6. Install bins: download required CLIs (e.g. gog)
  ├── 7. OAuth: run each provider in requires_oauth
  ├── 8. Lockdown: backup workspace → apply overrides → reset sessions → set allowlist
  ├── 9. Schema init: load schema.json → validate → create DB (local/supabase/api)
  ├── 10. Save config.json: skill_name, profile, version, db_target, tokens
  └── 11. Template processing: replace {placeholders} in SKILL.md
```

---

## Schema System

### Format

```json
{
  "tables": {
    "orders":      { "fields": [...], "statuses": [...] },
    "order_items": { "fields": [...] },
    "contacts":    { "fields": [...] }
  },
  "primary": "orders",
  "timezone": "Asia/Ho_Chi_Minh",
  "images_dir": "products"
}
```

Legacy single-table format (`"table"` + `"fields"`) is auto-normalized to multi-table on load.

### Field Properties

| Property | Values | Purpose |
|----------|--------|---------|
| `type` | `text`, `integer` | Data type |
| `auto` | `increment`, `timestamp` | Auto-generated at insert time |
| `default` | any string | Default value (excluded from positional args) |
| `required` | `true` | Validation: must be non-empty |
| `role` | `owner`, `status`, `price`, `timestamp` | Semantic role for CLI commands |
| `ref` | table name | Documents a relationship (not enforced) |

### Store Backends

```
cli.js --table <name> <command> <args>
  │
  ├── db_target: local    → <table>.json per table (read/write JSON files)
  ├── db_target: supabase → <supabase_url>/rest/v1/<table> (PostgREST API)
  └── db_target: api      → <base_url>/<table> (customer's REST endpoints)
```

API contract for remote backends:
- `GET /<table>` → `[{record}, ...]`
- `POST /<table>` → `{record}` (with server-assigned id)
- `PATCH /<table>/<id>` → `{record}` (updated)

---

## Profile System

```
skills/<vertical>/<skill>/
  profiles/
    shop-hoa/
      profile.yaml            Key-value pairs → template placeholders
      catalog.json            Overrides base catalog
      schema.json             Extends or replaces base schema
      products/               Product images
      workspace-overrides/    Agent persona files
```

Profile overlay order: catalog → schema (with extend-merge) → images → workspace-overrides → cleanup.

Schema extend: profile schema with `"extend": true` appends new fields/tables to base. Without extend, full replace.

---

## Workspace Lockdown

clawkit enforces a 1-skill-at-a-time model:

1. **Remove prior skills** — prompt user to confirm
2. **Backup workspace files** — AGENTS.md, SOUL.md, etc. to `.clawkit-backup/<timestamp>/`
3. **Apply workspace overrides** — copy skill's persona files to workspace root
4. **Delete generic files** — BOOTSTRAP.md, HEARTBEAT.md, TOOLS.md
5. **Reset sessions** — archive existing .jsonl, empty sessions.json
6. **Set allowlist** — `openclaw config set agents.defaults.skills '["<skill>"]'`

Reversible via `clawkit uninstall` which restores from backup.

---

## OAuth Provider Architecture

Self-registering pattern — each provider is a separate file:

```go
// oauth/my_provider.go
type myProvider struct{}
func (p myProvider) Name() string    { return "my_provider" }
func (p myProvider) Display() string { return "My Service" }
func (p myProvider) Authenticate() (map[string]string, error) { ... }
func init() { Register(myProvider{}) }
```

Tokens are saved to `config.json` and used for template placeholder substitution in SKILL.md.

---

## Directory Structure

```
clawkit/
  cmd/
    clawkit/                CLI entry point
    gen-registry/           Registry generator (recursive SKILL.md scan)
  internal/
    archive/                tar.gz / zip
    config/                 SkillConfig, OpenClaw detection
    installer/              Commands, schema, profiles, lockdown, registry
    template/               Placeholder substitution, catalog, image dirs
    ui/                     Terminal output (Info/Ok/Warn/Fatal)
  oauth/                    Self-registering OAuth providers
  skills/                   Built-in skills
    ecommerce/              shop-hoa, carehub-baby
    utilities/              finance-tracker
    tools/                  gog
  templates/                Reusable templates
    cli.js                  Generic schema-driven CLI
    verticals/              Per-vertical starter kits
      ecommerce/            4 tables: orders, order_items, products, contacts
      education/            3 tables: enrollments, courses, contacts
      consulting/           4 tables: students, applications, test_scores, contacts
      gold/                 4 tables: transactions, products, price_board, contacts
      food-distribution/    5 tables: orders, order_items, products, inventory, contacts
  npm/                      npm package wrapper with platform binaries
```

---

## Release

Push a version tag → GitHub Actions builds all platforms, packages skills, creates Release, publishes to npm:

```bash
git tag v1.2.0
git push origin v1.2.0
```
