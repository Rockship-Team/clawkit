---
name: gog
description: "Google Workspace CLI cho Gmail / Calendar / Drive / Contacts / Sheets / Docs. KICH HOAT khi user noi: 'check mail', 'doc email', 'search inbox', 'tim mail tu X', 'co mail moi khong', 'mail ve [chu de]', 'xem lich', 'meeting hom nay', 'event tuan nay', 'tao event', 'gui mail', 'reply mail X', 'tai file Drive', 'search Drive'. Tren bot-clawkit: account default rockship17.co@gmail.com da auth san — KHONG can hoi user setup OAuth."
metadata: {"openclaw":{"requires":{"bins":["gog"]}},"clawdbot":{"emoji":"🎮","install":[{"id":"brew","kind":"brew","formula":"steipete/tap/gogcli","bins":["gog"],"label":"Install gog (macOS/brew)"},{"id":"clawkit","kind":"clawkit","label":"Install via clawkit (Linux/Windows — auto-installed during setup)"}]}}
---

# gog

Use `gog` for Gmail/Calendar/Drive/Contacts/Sheets/Docs.

## ACCOUNT DEFAULT (bot-clawkit)

**Da auth san** account `rockship17.co@gmail.com` — keyring password `GOG_KEYRING_PASSWORD` set san trong env cua gateway, child `exec` inherit tu dong.

**MOI lenh phai co `-a rockship17.co@gmail.com`** — KHONG hoi user OAuth, KHONG xin credentials, KHONG bao "chua setup". Cu chay `gog` truc tiep qua `exec`.

Vi du chuan:
```
gog gmail search "from:supabase.com" -a rockship17.co@gmail.com --max 10 -j
```

## TAO GOOGLE DOC (cho long-form output)

**LƯU Ý:** `gog docs create` / `gog docs write` KHÔNG tồn tại. Subcommand `gog docs` chỉ có `export`. Để TẠO Google Doc, dùng workflow 3 bước qua Drive API:

```bash
# 1. Sinh content vào file md (markdown)
cat > /tmp/<slug>.md << 'EOF'
# Tiêu đề
Nội dung markdown ở đây
EOF

# 2. Upload + convert sang Google Doc
gog drive upload /tmp/<slug>.md \
  --name "<Tên doc>" \
  --convert-to doc \
  -a rockship17.co@gmail.com \
  --json
# Output có .file.id và .file.webViewLink

# 3. Share anyone-with-link writer (cho team review/edit)
gog drive share <fileId> \
  --to anyone --role writer --force \
  -a rockship17.co@gmail.com --json
```

**KHI NÀO dùng:** plan, proposal, blog, content calendar (>250 words). Output: paste URL vào chat + 3 bullet summary, KHÔNG paste full text.

**KHI NÀO KHÔNG dùng:** cold email body, FB post ngắn, reminder text, tin nhắn Telegram (paste trực tiếp dễ copy hơn).

## TRIGGER (khi nao kich hoat)

Kich hoat NGAY khi user noi bat ky pattern:
- "check mail", "co mail moi khong", "doc email", "search inbox"
- "tim mail tu X", "mail ve {topic}", "mail co {keyword}"
- "xem lich", "meeting hom nay", "event tuan nay", "{ngay} co lich gi"
- "gui mail cho X", "reply mail X" (PHAI confirm voi user noi dung truoc khi send)
- "tai file Drive", "search Drive {query}"
- "viet doc Google", "tao Sheet X"

KHONG hoi clarification gi cho cac yeu cau co du context — chay luon, return ket qua.

Setup (once)
1. Tạo OAuth2 credentials tại https://console.cloud.google.com/apis/credentials
   - Create Credentials → OAuth client ID → **Desktop app**
   - Download file credentials JSON (dạng `client_secret_*.json`)
2. Bật APIs cần dùng tại https://console.cloud.google.com/apis/library
   - Gmail API, Google Calendar API, Drive API, Contacts API, Sheets API, Docs API
3. Cấu hình OAuth consent screen tại https://console.cloud.google.com/apis/credentials/consent
   - Publishing status: nếu để "Testing" → phải thêm Gmail vào **Test users**
   - Hoặc publish app (chỉ cần cho tài khoản cá nhân, không cần review)
4. `gog auth credentials set /path/to/client_secret_*.json`
5. `gog auth add you@gmail.com --services gmail,calendar,drive,contacts,sheets,docs`
6. Kiểm tra: `gog auth list`

Troubleshooting
- `unauthorized_client`: OAuth app đang ở chế độ Testing → thêm email vào Test users (bước 3), hoặc publish app
- `access_denied`: chưa bật API tương ứng (bước 2)
- `invalid_client`: sai Client ID/Secret, hoặc chọn sai loại app (phải là Desktop app)

Gmail
- Search: `gog gmail search 'newer_than:7d' --max 10`
- Search (JSON): `gog gmail search 'from:boss@example.com is:unread' --max 5 --json`
- Get thread: `gog gmail thread get <threadId>`
- Download attachments: `gog gmail thread get <threadId> --download --out-dir ./attachments`
- Send plain text: `gog gmail send --to a@b.com --subject "Hi" --body "Hello"`
- Send HTML: `gog gmail send --to a@b.com --subject "Hi" --body-html "<p>Hello</p>"`
- Reply (with quote): `gog gmail send --reply-to-message-id <messageId> --quote --to a@b.com --subject "Re: Hi" --body "Reply"`
- Labels: `gog gmail labels list`

Calendar
- List calendars: `gog calendar calendars`
- Today: `gog calendar events <calendarId> --today`
- Tomorrow: `gog calendar events <calendarId> --tomorrow`
- This week: `gog calendar events <calendarId> --week`
- Next N days: `gog calendar events <calendarId> --days 7`
- Date range: `gog calendar events <calendarId> --from 2025-01-01T00:00:00Z --to 2025-01-08T00:00:00Z`
- All calendars: `gog calendar events --all`
- Search: `gog calendar search "meeting" --days 30`

Drive
- Search: `gog drive search "query" --max 10`
- Create folder: `gog drive mkdir "Folder Name" --parent <parentFolderId>`
- Rename: `gog drive rename <fileId> "New Name"`
- Move: `gog drive move <fileId> --parent <destinationFolderId>`
- Delete (trash): `gog drive delete <fileId>`
- Permanent delete: `gog drive delete <fileId> --permanent`
- Share with user: `gog drive share <fileId> --to user --email user@example.com --role reader`
- Share public: `gog drive share <fileId> --to anyone --role reader`
- List permissions: `gog drive permissions <fileId>`

Contacts
- List: `gog contacts list --max 50`
- Search: `gog contacts search "Name" --max 20`
- Get by email: `gog contacts get user@example.com`
- Create: `gog contacts create --given "John" --family "Doe" --email "john@example.com" --phone "+1234567890"`
- Delete: `gog contacts delete people/<resourceName>`

Sheets
- Get: `gog sheets get <sheetId> 'Tab!A1:D10'`
- Metadata: `gog sheets metadata <sheetId>`
- Update (JSON): `gog sheets update <sheetId> 'A1' --values-json '[["A","B"],["1","2"]]'`
- Append (JSON): `gog sheets append <sheetId> 'Tab!A:C' --values-json '[["x","y","z"]]'`
- Clear: `gog sheets clear <sheetId> 'Tab!A2:Z'`
- Create: `gog sheets create "My Sheet" --sheets "Sheet1,Sheet2"`
- Export Excel: `gog sheets export <sheetId> --format xlsx --out ./sheet.xlsx`

Docs
- Read: `gog docs cat <docId>`
- Export: `gog docs export <docId> --format txt --out <output_path>` (e.g. `/tmp/doc.txt` on macOS/Linux, `%TEMP%\doc.txt` on Windows)

Notes
- Set `GOG_ACCOUNT=you@gmail.com` to avoid repeating `--account`.
- For scripting, prefer `--json` and `--no-input`.
- Confirm before sending mail or creating/modifying events.
