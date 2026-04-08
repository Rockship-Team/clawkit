# clawkit

**clawkit** is the official CLI for installing and managing [OpenClaw](https://docs.openclaw.ai) skills. It handles the full deployment lifecycle — downloading skill packages, authenticating via Zalo QR, applying configuration, and registering into OpenClaw — in a single command.

Built by [Rockship](https://rockship.co).

---

## Requirements

- [Node.js 16+](https://nodejs.org)
- [OpenClaw](https://docs.openclaw.ai/installation) installed and running

---

## Installation

```bash
npm install -g @rockship/clawkit
```

Supports macOS (Apple Silicon & Intel), Linux, and Windows.

Verify the installation:

```bash
clawkit version
```

---

## Quick Start

```bash
# See available skills
clawkit list

# Install a skill
clawkit install shop-hoa-zalo

# Check what's installed
clawkit status
```

---

## Available Skills

| Skill | Description |
|-------|-------------|
| `shop-hoa-zalo` | Bot bán hoa qua Zalo cá nhân — tự động trả lời, báo giá, gửi ảnh, chốt đơn |
| `carehub-baby` | Trợ lý tư vấn sữa Blackmores cho CareHub Baby & Family qua Zalo |
| `gog` | Google Workspace CLI — Gmail, Calendar, Drive, Contacts, Sheets, Docs |

---

## Commands

| Command | Description |
|---------|-------------|
| `clawkit list` | List available skills and their install status |
| `clawkit install <skill>` | Install a skill (runs OAuth + configuration) |
| `clawkit update <skill>` | Update a skill, preserving tokens and existing config |
| `clawkit status` | Show all installed skills |
| `clawkit version` | Print version |

---

## How It Works

```
clawkit install shop-hoa-zalo
  │
  ├─ 1. Detect OpenClaw installation
  ├─ 2. Download skill package
  ├─ 3. Run OAuth (Zalo QR scan, Gmail, etc.)
  ├─ 4. Process SKILL.md — apply configuration placeholders
  ├─ 5. Initialize database (if init_db.py exists)
  └─ 6. Register skill in OpenClaw workspace
```

### Zalo Authentication

No App ID or App Secret required. clawkit uses OpenClaw's built-in Zalo integration — the user simply scans a QR code from the Zalo mobile app:

```
[1/3] Checking OpenClaw...         ✓
[2/3] Loading Zalo plugin...       ✓
[3/3] Scan the QR code with Zalo

██████████████████████████
█ ▄▄▄▄▄ █▀▄▄▀▄█ ▄▄▄▄▄ █
█ █   █ █▄▀▀▀▄█ █   █ █
...

Waiting for scan... (3 min timeout)
✓ Zalo connected successfully
```

---

## Platform Support

| Platform | Architecture | Config directory |
|----------|-------------|------------------|
| macOS | arm64, amd64 | `~/Library/Application Support/clawkit` |
| Linux | amd64 | `~/.config/clawkit` |
| Windows | amd64 | `%APPDATA%\clawkit` |

---

## Development

### Project Structure

```
clawkit/
├── cmd/
│   ├── clawkit/           # CLI entry point
│   └── gen-registry/      # Generates registry.json from SKILL.md frontmatter
├── internal/
│   ├── archive/           # tar.gz create/extract
│   ├── config/            # OpenClaw path detection, skill config
│   ├── installer/         # install, update, list, status, package logic
│   ├── template/          # SKILL.md placeholder processing + catalog
│   └── ui/                # Terminal output (colors, symbols, prompts)
├── oauth/                 # OAuth providers (self-registering via init())
│   ├── oauth.go
│   ├── zalo_personal.go
│   ├── zalo_oa.go
│   ├── gmail.go
│   ├── google.go
│   └── facebook.go
├── skills/                # Skill templates
│   ├── shop-hoa-zalo/
│   ├── carehub-baby/
│   └── gog/
├── npm/                   # npm package wrapper
│   ├── bin/clawkit.js     # Platform-detection shim
│   └── binaries/          # Bundled binaries (4 platforms)
├── registry.json          # Auto-generated — do not edit manually
└── Makefile
```

### Adding a New Skill

1. Create a directory under `skills/`:

```
skills/your-skill/
├── SKILL.md        # Required: YAML frontmatter + OpenClaw prompt
├── catalog.json    # Optional: product/service catalog
├── init_db.py      # Optional: database initialization
└── [assets]
```

2. Add YAML frontmatter to `SKILL.md`:

```yaml
---
version: "1.0.0"
description: "Short description of what this skill does"
requires_oauth:
  - zalo_personal
setup_prompts: []
---

Your OpenClaw skill prompt here...
```

3. Regenerate the registry and test:

```bash
make generate   # updates registry.json
make build
./clawkit install your-skill --skip-oauth
```

> `registry.json` is auto-generated from SKILL.md frontmatter. Never edit it directly.

### Adding a New OAuth Provider

Create a file in `oauth/` — it self-registers via `init()`, no other files need changing:

```go
// oauth/your_provider.go
package oauth

func init() { Register(&YourProvider{}) }

type YourProvider struct{}

func (p *YourProvider) Name() string    { return "your_provider" }
func (p *YourProvider) Display() string { return "Your Provider" }
func (p *YourProvider) Authenticate() (map[string]string, error) {
    // implement OAuth flow
    return map[string]string{"token": "..."}, nil
}
```

### Makefile Targets

```bash
make build        # Build binary for current platform
make test         # Run tests
make fmt          # Format and vet
make lint         # Run golangci-lint
make coverage     # Coverage report
make dist         # Cross-compile for macOS, Linux, Windows
make generate     # Regenerate registry.json from SKILL.md frontmatter
make npm-pack     # Build + pack npm tarball locally
```

### Releasing

Releases are fully automated. Push a version tag and GitHub Actions will build all platform binaries, create a GitHub Release, and publish to npm:

```bash
git tag v1.2.0
git push origin v1.2.0
```

---

## License

MIT
