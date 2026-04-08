# clawkit

CLI tool for installing and managing OpenClaw skills. Built by [Rockship](https://rockship.co).

clawkit handles the entire skill deployment lifecycle: downloading skill templates, running OAuth authorization, collecting client configuration, and installing skills into the correct OpenClaw directory — all in a single command.

## Quick Start

```bash
git clone git@github.com:Rockship-Team/clawkit.git
cd clawkit
make build

# List available skills
./clawkit list

# Install a skill
./clawkit install shop-hoa-zalo
```

## Requirements

- **Go 1.22+** (for building from source)
- **OpenClaw** installed on the target machine ([install guide](https://docs.openclaw.ai/installation))

clawkit auto-detects your OpenClaw installation and installs skills to `~/.openclaw/workspace/skills/`.

## Commands

| Command | Description |
|---------|-------------|
| `clawkit list` | List available skills and their install status |
| `clawkit install <skill>` | Install a skill with OAuth setup and configuration |
| `clawkit update <skill>` | Update a skill while preserving OAuth tokens and config |
| `clawkit status` | Show all installed skills |
| `clawkit package <skill>` | Package a skill into .tar.gz for distribution (dev) |
| `clawkit version` | Print version |

### Install flags

```bash
# Skip OAuth for testing (dev only)
clawkit install shop-hoa-zalo --skip-oauth
```

## How It Works

```
clawkit install shop-hoa-zalo
  │
  ├─ 1. Detect OpenClaw installation
  ├─ 2. Download skill (remote) or copy from local skills/ directory
  ├─ 3. Run OAuth flow (opens browser for Zalo/Google authorization)
  ├─ 4. Collect client config (shop name, email, etc.)
  ├─ 5. Process SKILL.md.tmpl → replace placeholders → generate SKILL.md
  └─ 6. Save config.json
```

### Template System

Skills use a template system for reusable deployment. A `SKILL.md.tmpl` file contains placeholders that clawkit replaces at install time:

| Placeholder | Source |
|-------------|--------|
| `{shopName}` | User input during install |
| `{notifyEmailFrom}` | User input |
| `{notifyEmailTo}` | User input |
| `{notifyEmailAppPassword}` | User input |
| `{catalogSection}` | Auto-generated from `catalog.json` |
| `{baseDir}` | Handled by OpenClaw runtime (not replaced by clawkit) |

### Catalog System

Each skill can include a `catalog.json` that defines product categories and price tiers. clawkit generates the catalog listing in SKILL.md from this file:

```json
{
  "categories": [
    {"folder": "hoa-hong", "label": "roses"},
    {"folder": "hoa-huong-duong", "label": "sunflowers"}
  ],
  "price_tiers": [280000, 300000, 350000, 450000],
  "best_seller": true
}
```

## Build

```bash
make build          # Build for current platform
make test           # Run tests
make dist           # Cross-compile for macOS, Linux, Windows
make package SKILL=shop-hoa-zalo   # Package a skill as .tar.gz
make help           # Show all commands
```

Cross-compile targets:
- macOS ARM64 (Apple Silicon)
- macOS AMD64 (Intel)
- Linux AMD64
- Windows AMD64

## Project Structure

```
clawkit/
├── main.go            # CLI entry point and command routing
├── installer.go       # install, update, list, status, package commands
├── oauth.go           # Zalo Personal/OA OAuth flows
├── template.go        # SKILL.md template processing + catalog generation
├── config.go          # OpenClaw detection, skill config read/write
├── registry.go        # Skill registry loader (remote + local fallback)
├── archive.go         # tar.gz create/extract utilities
├── ui.go              # Terminal output helpers (colors, prompts)
├── *_test.go          # Unit tests
├── registry.json      # Available skills manifest
├── Makefile           # Build, test, dist, package commands
├── .github/workflows/ # CI pipeline
├── .editorconfig      # Code formatting rules
├── LICENSE            # MIT
└── skills/            # Skill templates
    └── shop-hoa-zalo/
        ├── SKILL.md.tmpl    # Template with placeholders
        ├── catalog.json     # Product categories and prices
        ├── init_db.py       # Database initialization script
        └── flowers/         # Sample product images
```

## Contributing

### Adding a New Skill

1. Create a directory under `skills/`:

```
skills/your-skill-name/
├── SKILL.md.tmpl      # Template (use placeholders for client-specific values)
├── catalog.json       # Optional: product/service catalog
└── [other files]      # Scripts, assets, etc.
```

2. Add the skill to `registry.json`:

```json
{
  "skills": {
    "your-skill-name": {
      "version": "1.0.0",
      "description": "What this skill does",
      "requires_oauth": ["zalo_personal"],
      "setup_prompts": [
        {"key": "shop_name", "label": "Shop name"},
        {"key": "phone", "label": "Phone number"}
      ]
    }
  }
}
```

3. If your skill needs a new OAuth provider, add the flow in `oauth.go`:

```go
func oauthYourProvider(skillDir string) error {
    // Implement OAuth flow
}
```

And register it in `runOAuthFlow()`.

4. Test:

```bash
make build
./clawkit install your-skill-name --skip-oauth

# Verify generated SKILL.md
cat ~/.openclaw/workspace/skills/your-skill-name/SKILL.md
```

### Development Workflow

```bash
# Clone and build
git clone git@github.com:Rockship-Team/clawkit.git
cd clawkit
make build

# Run tests
make test

# Test install (skip OAuth)
./clawkit install shop-hoa-zalo --skip-oauth

# Verify result
cat ~/.openclaw/workspace/skills/shop-hoa-zalo/SKILL.md
cat ~/.openclaw/workspace/skills/shop-hoa-zalo/config.json
```

### Adding a New OAuth Provider

1. Add the OAuth function in `oauth.go`
2. Register the provider name in `runOAuthFlow()` switch
3. Use `waitForOAuthCallback()` for the local callback server (shared across all providers)
4. Save tokens to `SkillConfig.Tokens` map

### Commit Convention

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add Gmail OAuth support
fix: handle empty catalog.json gracefully
refactor: simplify template processing
docs: update README with new skill guide
```

## License

MIT
