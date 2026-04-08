# clawkit

CLI tool for installing and managing OpenClaw skills. Built by [Rockship](https://rockship.co).

clawkit handles the entire skill deployment lifecycle: downloading skill templates, Zalo QR code authentication, collecting client configuration, and installing skills into the correct OpenClaw directory — all in a single command.

## Installation

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/Rockship-Team/clawkit/main/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/Rockship-Team/clawkit/main/install.ps1 | iex
```

### Build from source (all platforms)

```bash
git clone git@github.com:Rockship-Team/clawkit.git
cd clawkit
make build
```

> Requires Go 1.22+. On Windows, use `CGO_ENABLED=0 go build -o clawkit.exe ./cmd/clawkit`.

## Quick Start

```bash
# List available skills
clawkit list

# Install a skill (includes Zalo QR login + configuration)
clawkit install shop-hoa-zalo
```

## Requirements

- **OpenClaw** installed on the target machine ([install guide](https://docs.openclaw.ai/installation))
- **Go 1.22+** only if building from source

clawkit auto-detects your OpenClaw installation and installs skills to the appropriate directory.

## Platform Support

| Platform | Install method | Config directory | Binary install path |
|----------|---------------|------------------|---------------------|
| macOS | `install.sh` / Homebrew | `~/Library/Application Support/clawkit` | `/usr/local/bin` |
| Linux | `install.sh` | `~/.config/clawkit` | `/usr/local/bin` or `~/.local/bin` |
| Windows | `install.ps1` | `%APPDATA%\clawkit` | `%LOCALAPPDATA%\clawkit\bin` |

## Commands

| Command | Description |
|---------|-------------|
| `clawkit list` | List available skills and their install status |
| `clawkit install <skill>` | Install a skill with Zalo QR login and configuration |
| `clawkit update <skill>` | Update a skill while preserving tokens and config |
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
  ├─ 3. Check/install Zalo plugin, show QR code for user to scan
  ├─ 4. Collect client config (shop name, email, price list, etc.)
  ├─ 5. Process SKILL.md — replace placeholders with client values
  ├─ 6. Initialize database (if init_db.py exists)
  └─ 7. Save config.json
```

### Zalo Personal Authentication

clawkit uses OpenClaw's built-in `zca-js` for Zalo Personal login. No App ID or App Secret needed — the user simply scans a QR code:

```
✓ Zalo plugin found
Opening QR code — scan it with your Zalo app.
QR code saved at: /tmp/openclaw/qr.png
Waiting for you to scan... (press Enter after scanning)
```

### Template System

SKILL.md contains placeholders that clawkit replaces at install time:

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

After installation, clients can customize the price list directly in SKILL.md.

## Build

```bash
make build          # Build for current platform
make test           # Run tests
make fmt            # Format and vet code
make lint           # Run golangci-lint
make coverage       # Test coverage report
make dist           # Cross-compile for macOS, Linux, Windows
make package SKILL=shop-hoa-zalo   # Package a skill as .tar.gz
make help           # Show all commands
```

## Project Structure

```
clawkit/
├── cmd/clawkit/           # CLI entry point
│   └── main.go
├── internal/
│   ├── archive/           # tar.gz create/extract
│   ├── config/            # OpenClaw detection, skill config
│   ├── installer/         # install, update, list, status, package commands
│   ├── template/          # SKILL.md placeholder processing + catalog
│   └── ui/                # Terminal output (colors, prompts)
├── oauth/                 # Pluggable OAuth providers
│   ├── oauth.go           # Provider interface + registry
│   ├── zalo_personal.go   # Zalo QR code login (via OpenClaw)
│   ├── zalo_oa.go         # Zalo Official Account OAuth
│   ├── gmail.go           # Gmail OAuth (+ gog CLI integration)
│   ├── google.go          # Google OAuth (generic)
│   └── facebook.go        # Facebook OAuth (Pages, Messenger)
├── skills/                # Skill templates
│   ├── shop-hoa-zalo/
│   ├── carehub-baby/
│   └── gog/
│       ├── SKILL.md       # Skill with placeholders
│       ├── catalog.json   # Product categories and prices
│       ├── init_db.py     # Database initialization
│       └── flowers/       # Sample product images
├── registry.json          # Available skills manifest
├── install.sh             # Installer for macOS/Linux
├── install.ps1            # Installer for Windows (PowerShell)
├── Makefile               # Build, test, lint, dist
├── .github/workflows/     # CI pipeline
├── .golangci.yml          # Linter config
├── .editorconfig          # Code formatting
└── LICENSE                # MIT
```

## Contributing

### Adding a New Skill

1. Create a directory under `skills/`:

```
skills/your-skill-name/
├── SKILL.md           # Use {placeholders} for client-specific values
├── catalog.json       # Optional: product/service catalog
├── init_db.py         # Optional: database setup
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

3. Test:

```bash
make build
./clawkit install your-skill-name --skip-oauth
cat ~/.openclaw/workspace/skills/your-skill-name/SKILL.md
```

### Adding a New OAuth Provider

Create a new file in `oauth/` — it self-registers via `init()`:

```go
// oauth/your_provider.go
package oauth

func init() { Register(&YourProvider{}) }

type YourProvider struct{}

func (p *YourProvider) Name() string    { return "your_provider" }
func (p *YourProvider) Display() string { return "Your Provider" }
func (p *YourProvider) Authenticate() (map[string]string, error) {
    // Implement OAuth flow
    // Use WaitForCallback() for browser redirect
    // Use OpenBrowser() to open auth URL
}
```

No other files need to be modified.

### Development Workflow

```bash
git clone git@github.com:Rockship-Team/clawkit.git
cd clawkit
make build
make test
./clawkit install shop-hoa-zalo --skip-oauth
```

### Creating a Release

```bash
make dist
gh release create v0.1.0 dist/* --title "v0.1.0" --notes "Initial release"
```

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
