---
name: sme-kpi
description: "Weekly KPI tracking cho BD/Sales — đặt target contract mỗi tuần, nhắc nếu chưa đặt, monitor actual vs target, diagnose bottleneck pipeline, đề xuất action cụ thể. Trigger khi: user nói 'KPI tuần này', 'target tuần này', 'đặt KPI', 'bao nhiêu contract tuần này', 'tiến độ KPI', 'còn thiếu bao nhiêu'. Cũng chạy khi cron trigger WEEKLY_KPI_PROMPT, WEEKLY_KPI_REMINDER, WEEKLY_KPI_MONITOR."
metadata: { "openclaw": { "emoji": "🎯" } }
---

# SME KPI — Weekly Contract Tracker

Skill theo dõi vòng lặp 4 lớp: **đặt KPI → monitor → diagnose → đề xuất action**.

## COMMANDS

```bash
sme-cli kpi check                    # KPI tuần hiện tại + needs_reminder flag
sme-cli kpi set --contracts N [--revenue N] [--set-by TEN] [--week YYYY-Www]
sme-cli kpi get [--week YYYY-Www]
sme-cli kpi list                     # 8 tuần gần nhất
```

"Contract" = contact chuyển sang stage WON trong COSMO.

## TRIGGER PATTERNS

### User-initiated (conversational)

Kích hoạt khi user nói bất kỳ:
- `KPI tuần này / target tuần này / mục tiêu tuần này`
- `đặt KPI / set KPI / KPI là X contract`
- `tiến độ KPI / còn thiếu bao nhiêu / đã đạt chưa`
- `tuần này target bao nhiêu / bao nhiêu hợp đồng`

→ Nếu chưa có KPI: hỏi user đặt target  
→ Nếu đã có KPI: chạy WEEKLY_KPI_MONITOR (xem bên dưới)

### Cron triggers

| Payload | Thời điểm | Action |
|---|---|---|
| `WEEKLY_KPI_PROMPT` | Thứ Hai 9h | Hỏi đặt KPI tuần mới |
| `WEEKLY_KPI_REMINDER` | Thứ Hai 11h | Nhắc lại nếu chưa đặt |
| `WEEKLY_KPI_MONITOR` | Trong morning briefing hoặc khi user hỏi | Monitor + diagnose + action |

---

## FLOW 1 — ĐẶT KPI (User hoặc cron WEEKLY_KPI_PROMPT)

### Khi nhận WEEKLY_KPI_PROMPT

Gọi `sme-cli kpi check`. Không output gì về kết quả lệnh.

- `kpi_set = true` → dừng, không gửi gì.
- `kpi_set = false` → gửi đúng 1 dòng này (không thêm bất cứ thứ gì):

`@Hans_Dang thứ Hai rồi anh ơi 👋 Tuần này mình target bao nhiêu contract?`

### Khi user reply với con số

```
Parse số từ message:
  "5 contract", "5", "target 5", "khoảng 3-5" → contracts = 5
  "10 triệu" hoặc "100tr" → revenue (nếu không có contracts)

Gọi: sme-cli kpi set --contracts N --set-by "tên user nếu biết"

Confirm ngắn gọn:
  "Noted ✓ Target tuần này: 5 contract (2026-W23, 08-14/06).
   Em sẽ báo tiến độ trong briefing sáng nhé."
```

---

## FLOW 2 — NHẮC NẾU CHƯA ĐẶT (cron WEEKLY_KPI_REMINDER)

Gọi `sme-cli kpi check`. Không output gì về kết quả lệnh.

- `kpi_set = true` → dừng, không gửi gì.
- `kpi_set = false` → gửi đúng 1 dòng này (không thêm bất cứ thứ gì):

`@Hans_Dang anh ơi, em vẫn chưa có KPI tuần này 🙏 Đặt nhanh để em theo dõi cho anh được không? Reply số là được, ví dụ: "3 contract"`

---

## FLOW 3 — MONITOR (Lớp 2 + 3 + 4)

Chạy khi: user hỏi tiến độ KPI, hoặc embed vào morning briefing.

### Bước 1 — Lấy KPI và actual

```bash
# Target
sme-cli kpi check

# Actual: contacts chuyển WON trong tuần này
sme-cli cosmo search-contact "stage:WON" 50
# Lọc thủ công: chỉ lấy contact có last_interaction >= week_start
# Đếm số contact thỏa điều kiện = actual_won
```

### Bước 2 — Tính gap và pipeline health

```bash
# Xem pipeline hiện tại
sme-cli cosmo daily-plan --mode morning
# Đếm theo stage: PROPOSAL + NEGOTIATION = "trong tầm với"
# QUALIFIED = "tiềm năng"
```

### Bước 3 — Tổng hợp output theo 4 lớp

**Format tin nhắn:**

```
📊 KPI tuần này (W23 · 08-14/06)

Target: 5 contract
Đã đạt: 2 ✓
Còn thiếu: 3

🔍 Pipeline hiện tại:
• NEGOTIATION: 1 deal → có thể close tuần này
• PROPOSAL sent: 2 deal → cần follow-up
• QUALIFIED: 4 → cần push lên PROPOSAL

💡 Bottleneck: [xem Diagnosis bên dưới]

⚡ Cần làm ngay:
• [Action 1]
• [Action 2]
• [Action 3]
```

---

## DIAGNOSIS RULES (Lớp 3)

Sau khi có actual + pipeline data, diagnose theo logic:

| Tình huống | Bottleneck | Gợi ý diagnosis |
|---|---|---|
| actual < target VÀ NEGOTIATION = 0 | Không có deal sắp close | "Pipeline thiếu deal ở cuối funnel" |
| PROPOSAL nhiều nhưng không ai reply | Email/proposal không convert | "Proposal chưa được follow-up đủ" |
| QUALIFIED nhiều nhưng ít PROPOSAL | Chuyển đổi Qualified→Proposal chậm | "BD chưa push khách lên bước tiếp" |
| Ít QUALIFIED | Thiếu lead đầu vào | "Cần thêm outreach / campaign mới" |
| actual >= target | Đạt rồi | Chúc mừng, gợi ý stretch goal |

---

## ACTION RECOMMENDATION (Lớp 4)

Mỗi action PHẢI có: **số cụ thể + hành động cụ thể**.

❌ Sai: "Cần follow-up các deal"
✅ Đúng: "2 proposal gửi >5 ngày chưa reply — cần follow-up ngay hôm nay"

❌ Sai: "Thêm contact mới"
✅ Đúng: "Pipeline còn 3 QUALIFIED nhưng thiếu 3 contract — cần push 3 khách này lên PROPOSAL trong 2 ngày tới"

Số actions tối đa: 3 việc, ưu tiên theo impact.

---

## INTEGRATION VỚI MORNING BRIEFING

Khi cron `DAILY_MORNING_BRIEFING` chạy, embed section KPI vào cuối briefing nếu KPI đã được đặt:

```
Gọi: sme-cli kpi check
Nếu kpi_set = true VÀ days_left <= 5:
  Thêm vào briefing: [output của FLOW 3 rút gọn — chỉ target/actual/3 action]
Nếu kpi_set = false VÀ ngày là thứ Hai:
  Nhắc 1 dòng: "💡 Chưa có KPI tuần này — reply số contract target để em theo dõi"
```

---

## QUY TẮC GIAO TIẾP

- Casual, ngắn. Không dùng header **Phần 1/2/3**.
- Số liệu KHÔNG bịa — chưa có data thì nói "em chưa kiểm tra".
- Khi nhắc KPI: tối đa 2 lần/tuần (prompt + reminder). Không spam.
- Confirm action trong 1 dòng, không giải thích dài.

---

## BD TEAM KPI — Per-member tracking

### Set KPI cho từng người

```bash
sme-cli kpi set --member "Hans" --proposals 5 --meetings 3 --contacts 20
sme-cli kpi set --member "Minh" --proposals 3 --meetings 2 --contacts 15
```

Fields: `--proposals` (số proposal cần gửi), `--meetings` (meetings cần book), `--contacts` (contact mới cần add), `--contracts` (deal cần close), `--revenue` (doanh thu target)

### Xem toàn team

```bash
sme-cli kpi team              # KPI tuần hiện tại của cả team
sme-cli kpi team --week 2026-W22  # tuần trước
```

### Check 1 người

```bash
sme-cli kpi check --member "Hans"
```

### Actual từ COSMO

```bash
sme-cli kpi actual             # cả team
sme-cli kpi actual --member "Hans"  # 1 người
```

Actual = interactions được log trong COSMO tuần này (proposal_sent, meeting, contact_created, stage_changed→WON).

### TRIGGER PATTERNS cho team BD KPI

User hỏi:
- "KPI tuần này của team" / "team đang ở đâu" → `sme-cli kpi team`
- "Minh đã gửi bao nhiêu proposal" → `sme-cli kpi actual --member Minh`  
- "Đặt KPI cho Linh: 4 proposal" → `sme-cli kpi set --member Linh --proposals 4`
- "So sánh target vs actual" → gọi cả `kpi team` + `kpi actual`, render bảng so sánh

### Format render bảng so sánh (khi user hỏi tiến độ team)

```
📊 KPI team tuần này (W23 · 08-14/06):

Member    | Proposal | Meeting | Contact
----------|----------|---------|--------
Hans      | 2/5 ✗   | 1/3 ✗  | 12/20 ✗
Minh      | 3/3 ✓   | 2/2 ✓  | 10/15 ✗
Linh      | 1/4 ✗   | 3/3 ✓  | 5/10 ✗

Còn 6 ngày. @Hans_Dang — Hans và Linh đang dưới target, cần push.
```

Actual data lấy từ `sme-cli kpi actual` hoặc `sme-cli cosmo search-interactions`.
Nếu chưa có actual data → nói rõ "em chưa có data actual từ COSMO tuần này".
