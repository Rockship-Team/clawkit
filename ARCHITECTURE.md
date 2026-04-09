# Architecture

Technical reference for contributors and developers.

---

## System Overview

```mermaid
graph TB
    subgraph User Machine
        CLI["clawkit CLI<br/>(Go binary via npm)"]
        OC["OpenClaw Runtime"]
        GOG["gog CLI<br/>(Google Workspace)"]
        Skills["~/.openclaw/workspace/skills/"]
    end

    subgraph External Services
        Zalo[(Zalo)]
        GSheets[(Google Sheets)]
        Gmail[(Gmail / Calendar)]
        NPM[(npm registry<br/>@rockship/clawkit)]
        GHR[(GitHub Releases<br/>skill packages)]
    end

    subgraph Rockship
        Repo["GitHub Repo<br/>(private)"]
        CI["GitHub Actions CI"]
    end

    User -->|npm install -g| NPM
    NPM --> CLI
    User -->|clawkit install| CLI
    CLI -->|download .tar.gz| GHR
    CLI -->|OAuth flow| External Services
    CLI -->|write skill files| Skills
    OC -->|load skill prompt| Skills
    OC -->|chat channel| Zalo
    OC -->|gog sheets append| GOG
    GOG -->|Sheets API| GSheets
    GOG -->|Gmail API| Gmail
    Repo -->|git tag vX.Y.Z| CI
    CI -->|publish| NPM
    CI -->|upload packages| GHR
```

---

## Component Responsibilities

| Component | Role |
|-----------|------|
| **clawkit** | Install, update, and manage skills. Runs OAuth once at install time. |
| **OpenClaw** | AI runtime — loads skill prompts, manages chat channels, routes messages. |
| **gog CLI** | Google API proxy — handles Gmail, Sheets, Calendar with auto token refresh. |
| **Skills** | SKILL.md prompt files that define AI behavior and tool usage. |

---

## Install Flow

```mermaid
sequenceDiagram
    actor User
    participant clawkit
    participant Registry as registry.json
    participant OAuth as OAuth Provider
    participant Browser
    participant Google as Google / Zalo
    participant OpenClaw

    User->>clawkit: clawkit install finance-tracker
    clawkit->>Registry: lookup skill metadata
    Registry-->>clawkit: version, requires_oauth, description

    clawkit->>OpenClaw: Preflight check (openclaw in PATH?)
    OpenClaw-->>clawkit: ✓

    clawkit->>clawkit: Download skill package (.tar.gz)
    clawkit->>clawkit: Extract to ~/.openclaw/workspace/skills/

    clawkit->>OAuth: Run google_sheets provider
    OAuth->>clawkit: Load credentials from gog config?
    clawkit-->>OAuth: client_id, client_secret, gmail_account

    OAuth->>Browser: Open Google auth URL
    User->>Browser: Login + grant permission
    Browser->>OAuth: Callback with auth code
    OAuth->>Google: Exchange code for tokens
    Google-->>OAuth: access_token, refresh_token

    OAuth->>Google: Create Finance Tracker spreadsheet
    Google-->>OAuth: spreadsheet_id, spreadsheet_url
    OAuth->>Google: Setup sheets + pie chart (batchUpdate)
    OAuth->>Google: Verify access

    OAuth-->>clawkit: tokens (spreadsheet_id, gmail_account, ...)
    clawkit->>clawkit: Save config.json (preserves tokens)

    clawkit-->>User: ✓ Installed! Spreadsheet: https://...
```

---

## Daily Usage Flow (finance-tracker)

```mermaid
sequenceDiagram
    actor User
    participant Zalo
    participant OpenClaw
    participant AI as AI Model
    participant gog
    participant Sheets as Google Sheets

    User->>Zalo: [sends receipt photo]
    Zalo->>OpenClaw: incoming message + image
    OpenClaw->>AI: skill prompt + image + message history

    AI->>AI: Extract: amount, merchant, date
    AI->>AI: Classify category (Ăn uống / Cafe / ...)

    AI-->>OpenClaw: "55,000đ - Cafe Highlands. Lưu nhé?"
    OpenClaw->>Zalo: send reply
    Zalo->>User: "55,000đ - Cafe Highlands. Lưu nhé?"

    User->>Zalo: "ừ"
    Zalo->>OpenClaw: incoming message
    OpenClaw->>AI: skill prompt + "ừ"

    AI->>gog: gog sheets append <spreadsheetId> "Giao dịch!A:E" "09/04/2026|Cafe Highlands|55000|Cafe|"
    gog->>gog: Check token expiry → auto refresh if needed
    gog->>Sheets: Sheets API append
    Sheets-->>gog: ✓
    gog-->>AI: ✓

    AI-->>OpenClaw: "Đã lưu! Tổng hôm nay: 155,000đ"
    OpenClaw->>Zalo: send reply
    Zalo->>User: "Đã lưu! Tổng hôm nay: 155,000đ"
```

---

## Zalo Personal Auth Flow

```mermaid
sequenceDiagram
    actor User
    participant clawkit
    participant OpenClaw
    participant ZaloPlugin as @openclaw/zalouser
    participant Zalo as Zalo Mobile App

    clawkit->>OpenClaw: Check openclaw in PATH
    clawkit->>OpenClaw: openclaw plugins list
    OpenClaw-->>clawkit: zalouser installed? (yes/no)

    alt Plugin not installed
        clawkit->>OpenClaw: openclaw plugins install @openclaw/zalouser
    end

    clawkit->>OpenClaw: openclaw channels login --channel zalouser (background)
    OpenClaw->>ZaloPlugin: start login session
    ZaloPlugin->>clawkit: write QR image to /tmp/openclaw/qr.png

    clawkit->>clawkit: Detect terminal (iTerm2 / Kitty / fallback)
    clawkit-->>User: [QR code displayed in terminal]

    User->>Zalo: Scan QR code
    Zalo->>ZaloPlugin: authenticate session
    ZaloPlugin-->>OpenClaw: login complete

    clawkit->>OpenClaw: openclaw config set channels.zalouser.enabled true
    clawkit-->>User: ✓ Zalo connected
```

---

## OAuth Provider Architecture

Each OAuth provider is a self-registering Go struct. No central registry needed.

```mermaid
classDiagram
    class Provider {
        <<interface>>
        +Name() string
        +Display() string
        +Authenticate() map~string~string~, error
    }

    class ZaloPersonal {
        +Name() "zalo_personal"
        +Authenticate() QR scan via OpenClaw
    }

    class Gmail {
        +Name() "gmail"
        +Authenticate() OAuth2 + gog CLI setup
    }

    class GoogleSheets {
        +Name() "google_sheets"
        +Authenticate() OAuth2 + create spreadsheet
    }

    class ZaloOA {
        +Name() "zalo_oa"
        +Authenticate() OAuth2 browser flow
    }

    class Facebook {
        +Name() "facebook"
        +Authenticate() OAuth2 browser flow
    }

    Provider <|.. ZaloPersonal
    Provider <|.. Gmail
    Provider <|.. GoogleSheets
    Provider <|.. ZaloOA
    Provider <|.. Facebook
```

Each provider calls `Register()` in its `init()` function — adding a new provider requires only creating a new file.

---

## Token Management

| Provider | Token storage | Refresh handled by |
|----------|-------------|-------------------|
| `zalo_personal` | OpenClaw internal | OpenClaw runtime |
| `gmail` | gog CLI keyring | gog CLI (automatic) |
| `google_sheets` | gog CLI keyring (via gog) | gog CLI (automatic) |
| `zalo_oa` | config.json | Not implemented (short-lived) |
| `facebook` | config.json | Not implemented (short-lived) |

> **Design principle:** clawkit uses OAuth tokens only once at install time (e.g. creating a spreadsheet). After install, all API calls go through gog CLI or OpenClaw, both of which handle token refresh automatically.

---

## Skill Package Format

A skill is a directory with a required `SKILL.md`:

```
skills/your-skill/
├── SKILL.md         # YAML frontmatter + OpenClaw prompt (required)
├── catalog.json     # Product/service catalog (optional)
├── init_db.py       # Database initialization (optional)
└── [assets]         # Images, scripts, etc.
```

`SKILL.md` frontmatter drives everything — `registry.json` is auto-generated from it:

```yaml
---
name: finance-tracker
description: "Receipt scan → categorize → Google Sheets"
version: "1.0.0"
requires_oauth:
  - google_sheets     # OAuth providers to run at install
setup_prompts: []     # Deprecated — use SKILL.md placeholders instead
metadata:
  openclaw:
    emoji: "💰"
    requires:
      bins: ["gog"]   # External binaries required at runtime
---
```

---

## Release Pipeline

```mermaid
graph LR
    Dev -->|git tag v1.2.0| GitHub
    GitHub -->|triggers| CI[GitHub Actions]
    CI --> Build[Build binaries<br/>macOS · Linux · Windows]
    CI --> Package[Package skills<br/>.tar.gz]
    CI --> Release[Create GitHub Release]
    CI --> NPM[Publish @rockship/clawkit<br/>to npm]
    NPM -->|npm install -g| User
```

Triggered by pushing a version tag:

```bash
git tag v1.2.0
git push origin v1.2.0
```
