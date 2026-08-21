---
name: sme-learn
description: "Học từ người dùng — tự động capture preferences, corrections, validated patterns từ mọi cuộc trò chuyện. Trigger sau mỗi turn: phát hiện learning moment → ghi vào memory file đúng chỗ. KHÔNG chờ user yêu cầu. Đây là background behavior luôn chạy."
metadata: { "openclaw": { "emoji": "🧠" } }
---

# SME Learn — Continuous Learning từ mọi User

Skill này chạy **ngầm sau mỗi turn**. Không cần user trigger. Không thông báo khi ghi memory.

## Nhận diện Learning Moment

### Correction (user sửa bot)
- User nói: "không phải vậy", "đừng làm thế", "lần sau...", "thay vì...", "anh muốn..."
- User edit lại output của bot
- User yêu cầu làm lại theo cách khác
- User phàn nàn về format/tone/nội dung

### Validation (user xác nhận cách làm tốt)
- User nói: "đúng rồi", "perfect", "làm thế này nữa nha", "chuẩn", "y chang"
- User approve một cách tiếp cận không standard
- User dùng lại output của bot mà không sửa

### New Pattern (workflow mới)
- User yêu cầu làm điều gì chưa từng làm
- Task cần ≥4 tool calls → pattern có thể tái sử dụng
- User giải thích quy trình nội bộ của team

---

## Cách ghi Memory

### 1. Per-user file

File path: `~/workspace-gtm/memory/users/{username}.md`

Username lấy từ conversation metadata (`sender` field). Normalize: lowercase, thay space bằng `-`.

Ví dụ: `Hans Dang` → `hans-dang.md`, `Anh Khoa` → `anh-khoa.md`

Format ghi:
```markdown
## Preferences
- [PREF] {rule} — vì {lý do nếu biết}

## Corrections
- [FIX] Đừng {hành vi cũ} → {hành vi đúng} (từ {ngày})

## Validated Patterns  
- [OK] {pattern} hoạt động tốt với user này (từ {ngày})

## Context
- Role: {vai trò trong team nếu biết}
- Style: {communication style}
```

### 2. Shared team file

File path: `~/workspace-gtm/memory/users/team-bd.md`

Ghi khi: cùng correction/preference xuất hiện từ ≥2 user khác nhau.

Format:
```markdown
## Team BD — Shared Preferences
- [TEAM] {rule} — confirmed by {user1}, {user2}
```

### 3. MEMORY.md index

File path: `~/workspace-gtm/memory/MEMORY.md`

Sau khi ghi per-user file, append 1 dòng vào MEMORY.md:
```
- [{username}]({relative-path}) — {1-line summary of latest learning}
```

---

## Quy tắc ghi

**GHI ngay** — không delay, không hỏi user. Ghi trong cùng turn phát hiện learning moment.

**KHÔNG ghi** khi:
- Chỉ là câu hỏi thông tin bình thường
- User hỏi data (contacts, events, proposals) — không phải feedback về bot
- Trùng với ghi chú đã có trong file

**Format ngắn** — 1 dòng per learning. KHÔNG giải thích dài.

**Timestamp** — luôn ghi ngày: `(2026-06-01)`.

---

## Skill Evolution Trigger

Sau task có ≥4 tool calls VÀ thành công:
1. Tự hỏi: "Pattern này có tái sử dụng không?"
2. Nếu có → check SKILL.md liên quan:
   - Có section "Pitfalls" không? Nếu gặp lỗi → add pitfall
   - Có ví dụ tương tự không? Nếu tìm được cách tốt hơn → propose update
3. Ghi 1 dòng vào `~/workspace-gtm/memory/skill-suggestions.md`:
   ```
   - [{skill-name}] {gợi ý cải thiện} — từ task {mô tả ngắn} ({ngày})
   ```

Bot KHÔNG tự sửa SKILL.md — chỉ ghi suggestions. Human review trước khi apply.

---

## Ví dụ

**User Minh sửa:** "Em đừng liệt kê hết tên, 3 cái là đủ"
→ Ghi vào `users/minh.md`:
```
- [FIX] Tối đa 3 tên trong 1 bullet list (2026-06-01)
```

**User Hans validate:** "Đúng rồi, cứ tag @Hans_Dang khi cần quyết định"
→ Ghi vào `users/hans-dang.md`:
```
- [OK] Tag @Hans_Dang cho mọi decision point — confirmed (2026-06-01)
```

**Xuất hiện ở 2 users:**
→ Ghi vào `users/team-bd.md`:
```
- [TEAM] Không liệt kê >3 tên — confirmed by Minh, Hans (2026-06-01)
```
