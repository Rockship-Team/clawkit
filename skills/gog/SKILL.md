---
name: gog
description: Google Workspace CLI - Gmail, Calendar, Drive, Contacts, Sheets, Docs
version: "1.0.0"
requires_oauth:
  - gmail
setup_prompts: []
homepage: https://gogcli.sh
metadata: {"clawdbot":{"emoji":"🎮","install":[{"id":"brew","kind":"brew","formula":"steipete/tap/gogcli","bins":["gog"],"label":"Install gog (macOS/brew)"},{"id":"clawkit","kind":"clawkit","label":"Install via clawkit (Linux/Windows — auto-installed during setup)"}]}}
---

# gog

Use `gog` for Gmail/Calendar/Drive/Contacts/Sheets/Docs. Requires OAuth setup.

Setup (once)
1. Tạo OAuth2 credentials tại https://console.cloud.google.com/apis/credentials
2. Download file credentials JSON (dạng `client_secret_*.json`)
3. `gog auth credentials set /path/to/client_secret_*.json`
4. `gog auth add you@gmail.com --services gmail,calendar,drive,contacts,sheets,docs`
5. Kiểm tra: `gog auth list`

Common commands
- Gmail search: `gog gmail search 'newer_than:7d' --max 10`
- Gmail send: `gog gmail send --to a@b.com --subject "Hi" --body "Hello"`
- Calendar: `gog calendar events <calendarId> --from <iso> --to <iso>`
- Drive search: `gog drive search "query" --max 10`
- Contacts: `gog contacts list --max 20`
- Sheets get: `gog sheets get <sheetId> "Tab!A1:D10" --json`
- Sheets update: `gog sheets update <sheetId> "Tab!A1:B2" --values-json '[["A","B"],["1","2"]]' --input USER_ENTERED`
- Sheets append: `gog sheets append <sheetId> "Tab!A:C" --values-json '[["x","y","z"]]' --insert INSERT_ROWS`
- Sheets clear: `gog sheets clear <sheetId> "Tab!A2:Z"`
- Sheets metadata: `gog sheets metadata <sheetId> --json`
- Docs export: `gog docs export <docId> --format txt --out /tmp/doc.txt`
- Docs cat: `gog docs cat <docId>`

Notes
- Set `GOG_ACCOUNT=you@gmail.com` to avoid repeating `--account`.
- For scripting, prefer `--json` plus `--no-input`.
- Sheets values can be passed via `--values-json` (recommended) or as inline rows.
- Docs supports export/cat/copy. In-place edits require a Docs API client (not in gog).
- Confirm before sending mail or creating events.

