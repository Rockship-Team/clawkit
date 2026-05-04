# Google Doc Output — Workflow chuẩn

## KHI NÀO dùng Google Doc thay vì paste text vào chat

**BẮT BUỘC** dùng Google Doc khi sinh:
- Pilot plan / project plan (bất kỳ "plan.md" task nào)
- Proposal / báo giá / quote (long-form, >200 words)
- Blog post / landing copy / case study
- Content kế hoạch tháng (marketing calendar, content calendar)
- Bất kỳ document nào user cần **edit / comment / collab** với team
- Bất kỳ output > 250 words mà user có thể muốn share/review

**KHÔNG** dùng Google Doc cho:
- Cold email body / subject (ngắn, paste trực tiếp dễ copy)
- Reminder text / cron payload (1-2 câu)
- Tin nhắn Telegram reply (in-chat)
- Code snippet / config sample (paste trực tiếp)
- Quick answer < 100 words

## Workflow CLI (4 lệnh)

```bash
# 1. Sinh content vào file md tạm
cat > /tmp/<slug>.md << 'EOF'
# <Title>

<Markdown content here — supports H1/H2/H3, bullet, table, bold, italic>
EOF

# 2. Upload + convert sang Google Doc
gog drive upload /tmp/<slug>.md \
  --name "<Tiêu đề doc — date>" \
  --convert-to doc \
  -a rockship17.co@gmail.com \
  --json

# Output JSON có .file.id và .file.webViewLink

# 3. Share anyone-with-link (writer)
gog drive share <fileId> \
  --to anyone --role writer --force \
  -a rockship17.co@gmail.com --json

# 4. Return webViewLink cho user
```

## Output format khi return cho user

KHÔNG paste content full vào chat. Thay vào đó:

```
Đã tạo Google Doc:
📄 <Title>
🔗 https://docs.google.com/document/d/<id>/edit

Anh review + edit trực tiếp trong doc. Em chờ feedback.
```

Optionally show 2-3 dòng summary đầu doc để user biết bot làm đúng hướng:

```
Đã tạo Google Doc: https://docs.google.com/document/d/<id>/edit

Tóm tắt nội dung doc:
- Mục tiêu: ...
- Scope: ...
- Output: ...

Anh review + edit, em chờ feedback.
```

## Khi user yêu cầu sửa doc đã tạo

User: "đổi mở bài thành Y" / "thêm phần Z"

Bot KHÔNG tạo doc mới. Thay vào đó:
1. Đọc lại doc hiện tại: `gog drive download <fileId> -e markdown` (export lại md)
2. Apply sửa local
3. Re-upload qua `gog drive upload --update <fileId>` HOẶC update bằng Docs API

(Nếu phức tạp → tạo doc mới + bỏ doc cũ, vẫn OK nhưng lose comment history.)

## Naming convention

Tên doc = `<Loại> — <Subject> — <YYYY-MM-DD>`

Ví dụ:
- `Pilot Plan — Coolmate D2C Q2 2026 — 2026-05-04`
- `Proposal — Vinasun AI Customer Service — 2026-05-04`
- `Content Calendar — LogicX FB tháng 5 — 2026-05-04`

## Edge cases

- **Drive quota full** → fail upload → fallback paste markdown vào chat + báo user
- **Network down** → fail → fallback paste
- **Doc trùng tên** → vẫn tạo được (Google Drive cho phép duplicate name, file ID khác)
