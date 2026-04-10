# clawkit

The official CLI for installing and managing [OpenClaw](https://docs.openclaw.ai) skills.

```bash
npm install -g @rockship/clawkit
```

Built by [Rockship](https://rockship.co) · [Tiếng Việt](./README.vi.md)

---

## Requirements

**Node.js 16 or higher** is required. If you don't have it:

- **Download:** [nodejs.org](https://nodejs.org) — install the LTS version
- **macOS (Homebrew):** `brew install node`
- **Windows (winget):** `winget install OpenJS.NodeJS.LTS`
- **Linux:** `sudo apt install nodejs npm` or use [nvm](https://github.com/nvm-sh/nvm)

Verify your installation:

```bash
node --version   # should be v16 or higher
npm --version
```

**OpenClaw** must also be installed and running on your machine. See the [OpenClaw install guide](https://docs.openclaw.ai/installation).

---

## Installation

```bash
npm install -g @rockship/clawkit
```

Supports macOS (Apple Silicon & Intel), Linux, and Windows.

Verify:

```bash
clawkit version
```

---

## Quick Start

```bash
# See all available skills
clawkit list

# Install a skill
clawkit install shop-hoa

# Check installed skills
clawkit status
```

---

## Available Skills

| Skill | Description |
|-------|-------------|
| `shop-hoa` | Flower shop assistant for OpenClaw web chat / TUI — consult, quote, send images, take orders, look up history |
| `carehub-baby` | Blackmores baby nutrition consultant for CareHub via Zalo |
| `gog` | Google Workspace assistant — Gmail, Calendar, Drive, Contacts |

---

## Commands

| Command | Description |
|---------|-------------|
| `clawkit list` | List available skills and install status |
| `clawkit install <skill>` | Install a skill (runs OAuth + configuration) |
| `clawkit update <skill>` | Update a skill, preserving tokens and config |
| `clawkit status` | Show all installed skills |
| `clawkit version` | Print version |

---

## How It Works

When you run `clawkit install`, it:

1. Detects your OpenClaw installation
2. Downloads the skill package
3. Runs OAuth (e.g. Zalo QR scan, Gmail login)
4. Applies your configuration to the skill template
5. Initializes the database if needed
6. Registers the skill in your OpenClaw workspace

For architecture diagrams and technical deep-dives, see [ARCHITECTURE.md](./ARCHITECTURE.md).

### Zalo Authentication

No App ID or App Secret required. clawkit uses OpenClaw's built-in Zalo integration. You scan a QR code once from the Zalo mobile app:

```
[1/3] Checking OpenClaw...         ✓
[2/3] Loading Zalo plugin...       ✓
[3/3] Scan the QR code with Zalo

██████████████████████████
█ ▄▄▄▄▄ █▀█▄▄▀▄█ ▄▄▄▄▄ █
█ █   █ █ ▀▄▄▄█ █   █ █
...

Waiting for scan... (3 min timeout)
✓ Zalo connected
```

---

## Development

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
description: "Short description"
requires_oauth:
  - zalo_personal
setup_prompts: []
---
```

3. Regenerate the registry and test:

```bash
make generate
make build
./clawkit install your-skill --skip-oauth
```

> `registry.json` is auto-generated from SKILL.md frontmatter. Never edit it manually.

### Releasing a New Version

Push a version tag — GitHub Actions handles everything:

```bash
git tag v1.2.0
git push origin v1.2.0
```

The CI will build binaries for all platforms, create a GitHub Release, and publish `@rockship/clawkit` to npm automatically.

---

## License

MIT
