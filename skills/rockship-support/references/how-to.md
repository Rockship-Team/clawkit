# OpenClaw Tutorial Content

Đây là nội dung thực tế của trang tutorial tại `https://openclaw.rockship.co/en/tutorial`.
Source files: `public/docs/en/`

---

## 1. Installing OpenClaw on macOS

### Install Required Tools

```bash
curl -fsSL https://openclaw.rockship.co/install.sh | bash
```

Tự động cài: Homebrew (nếu chưa có), Node.js v18+, Python 3.11+

### Install OpenClaw

```bash
npm install -g openclaw@latest
openclaw -v   # kiểm tra version
```

### Initial Setup

```bash
openclaw onboard --install-daemon
```

Các bước chọn:
1. Continue? → `Yes`
2. Setup mode → `QuickStart`
3. Model/auth provider → `OpenRouter`
4. Enter OpenRouter API key → `sk-or-v1-xxxxxxxxxx`
5. Default model → `openrouter/stepfun/step-3.5-flash`
6. Select channel → `Skip for now`
7. Search provider → `Skip for now`
8. Configure skills now? → `No`
9. Enable hooks? → `Skip for now`
10. How do you want to hatch your bot? → `Do this later`

Sau setup, gateway tự mở terminal mới. Nếu không tự mở:
```bash
openclaw gateway
```

### Dashboard

```bash
openclaw dashboard
```

### Kết nối Telegram

```bash
openclaw channels add --channel telegram --token <bot-token>
openclaw pairing approve telegram <code>
```

Lấy bot token: Telegram → BotFather → `/newbot` → đặt tên → lấy token.
Lấy pairing code: mở bot trên Telegram → `/start` → copy code.

---

## 2. Installing OpenClaw on Windows

### Install Required Tools

```cmd
curl -fsSL https://openclaw.rockship.co/install.ps1 | powershell -
```

### Install OpenClaw

```cmd
npm install -g openclaw@latest
openclaw -v
```

### Initial Setup

```bash
openclaw onboard --install-daemon
```

Các bước chọn giống macOS (1–9). Sau setup, gateway tự mở terminal mới. Nếu không:
```cmd
openclaw gateway
```

> Gateway phải chạy liên tục để OpenClaw hoạt động trên Windows.

### Dashboard

```cmd
openclaw dashboard
```

### Kết nối Telegram

```cmd
openclaw channels add --channel telegram --token <bot-token>
openclaw pairing approve telegram <code>
```

---

## 3. Installing Skills on macOS / Linux

### Quick Install

```bash
# vnstock
curl -fsSL https://openclaw.rockship.co/install-skill.sh | bash -s -- vnstock

# gold-price
curl -fsSL https://openclaw.rockship.co/install-skill.sh | bash -s -- gold-price

# vnnews
curl -fsSL https://openclaw.rockship.co/install-skill.sh | bash -s -- vnnews
```

Script tự động: cài Python (nếu thiếu), download skill ZIP, extract vào workspace, restart gateway.

### Verify & Restart

```bash
openclaw skills list
openclaw gateway restart
```

---

## 4. Installing Skills on Windows

### Quick Install

```cmd
# vnstock
curl -fsSL https://openclaw.rockship.co/install-skill.ps1 -o install-skill.ps1 && powershell -ExecutionPolicy Bypass -File install-skill.ps1 vnstock

# gold-price
curl -fsSL https://openclaw.rockship.co/install-skill.ps1 -o install-skill.ps1 && powershell -ExecutionPolicy Bypass -File install-skill.ps1 gold-price

# vnnews
curl -fsSL https://openclaw.rockship.co/install-skill.ps1 -o install-skill.ps1 && powershell -ExecutionPolicy Bypass -File install-skill.ps1 vnnews
```

### Verify & Restart

```cmd
openclaw skills list
openclaw gateway restart
```

---

## 5. Channel Connection

### Connect Telegram

1. Mở Telegram, tìm `@BotFather` → `/newbot`
2. Đặt tên bot và username (phải kết thúc bằng `bot`)
3. Copy bot token từ BotFather
4. Paste token vào chat OpenClaw:
   > "This is my Telegram bot token: `<token>`, please directly help me configure the connection to Telegram."

### Connect WhatsApp

1. Trong OpenClaw dashboard → click SSH để mở terminal
2. Enable WhatsApp channel (chỉ làm 1 lần):
```bash
openclaw config set channels.whatsapp.enabled true
```
3. Chạy lệnh login:
```bash
openclaw channels login --channel whatsapp
```
4. QR code sẽ xuất hiện trong terminal — kéo divider để phóng to nếu cần
5. Trên điện thoại: WhatsApp → ⋮ → Linked Devices → Link a Device → quét QR
6. Thành công khi thấy: "WhatsApp gateway connected as +xxxxxxxxx"

**Lỗi thường gặp:**
- "Unable to connect to this device" → WhatsApp → Settings → Linked Devices → xóa device cũ → chạy lại lệnh
- QR không hiện đủ → kéo divider terminal lên cao hơn

### Connect Discord

1. Vào discord.com/developers/applications → tạo application mới
2. Tab Bot → tạo bot → copy token
3. OAuth2 → URL Generator → chọn scope `bot` + permission `Send Messages`
4. Invite bot vào server bằng URL vừa tạo
5. Paste token vào OpenClaw → Channels → Discord

---

## 6. Self-Rescue Guide

### Bước 1 — Xác định triệu chứng

- Agent không phản hồi tin nhắn?
- Bị kẹt trong loop?
- Trả về lỗi liên tục?

### Bước 2 — Soft Restart (Khuyên dùng trước)

1. Dashboard → click tên instance
2. Menu ⋮ → Restart
3. Chờ 30–60 giây
4. Test bằng một tin nhắn đơn giản

> Giải quyết được phần lớn các trường hợp stuck/không phản hồi.

### Bước 3 — Clear Context (nếu Soft Restart không đủ)

Gửi lệnh trong chat với agent:
```
/reset
```
Reset context đang hoạt động, giữ nguyên long-term memory.

### Bước 4 — Full Deletion (Phương án cuối cùng)

⚠️ **Xóa instance sẽ mất toàn bộ memory và lịch sử hội thoại.**

1. Dashboard → chọn instance
2. Menu ⋮ → Delete Instance → xác nhận
3. Tạo instance mới từ đầu

> Liên hệ support trước khi dùng phương án này: rockship17@gmail.com

---

## 7. Connect Google Workspace (gog CLI)

`gog` là CLI tool kết nối Gmail, Calendar, Drive, Contacts, Sheets, Docs với OpenClaw qua OAuth.

### Bước 1 — Cài gog

**macOS:**
```bash
brew install steipete/tap/gogcli
```

**Linux (Ubuntu/Debian):**
```bash
git clone https://github.com/steipete/gogcli.git
cd gogcli && make
```

**Linux (Arch):**
```bash
yay -S gogcli
```

**Windows:**
```cmd
winget install -e --id steipete.gogcli
```

### Bước 2 — Tạo Google Cloud credentials

1. Vào [console.cloud.google.com](https://console.cloud.google.com)
2. Tạo project mới
3. Bật **Gmail API** và **Google Calendar API**
4. Tạo OAuth credentials → chọn loại **Desktop app**
5. Download file `client_secret.json`

### Bước 3 — Đăng ký credentials với gog

```bash
gog auth credentials /path/to/client_secret.json
```

### Bước 4 — Authorize tài khoản Google

```bash
gog auth add you@gmail.com --services gmail,calendar,drive,contacts,docs,sheets
```

Trình duyệt sẽ mở để bạn đăng nhập Google và cấp quyền.

### Bước 5 — Kiểm tra kết nối

```bash
gog auth list
```

### Bước 6 — Cài gog skill cho OpenClaw

```bash
curl -fsSL https://openclaw.rockship.co/install-skill.sh | bash -s -- gog
openclaw gateway restart
```

> Sau khi cài, Calendar tự động hoạt động cùng Gmail vì dùng chung OAuth.

---

### Các lệnh thường dùng

**Gmail:**
```bash
# Tìm email 7 ngày gần nhất
gog gmail search 'newer_than:7d' --max 10

# Gửi email
gog gmail send --to a@b.com --subject "Hi" --body "Hello"
```

**Calendar:**
```bash
# Xem sự kiện
gog calendar events <calendarId>
```

**Tips:**
- Set `GOG_ACCOUNT=you@gmail.com` để không cần truyền account mỗi lần
- Dùng `--json` cho output dễ xử lý tự động
- Dùng `--body-file` cho email nhiều dòng
