---
name: sme-kpi
description: "KPI tracking cho BD/Sales. 2 loại: (1) OUTREACH KPI — target LinkedIn tin nhắn MỚI/ngày, đặt 1 lần dùng mãi, báo cáo 2 mốc/ngày T2-T6 (9h finalize hôm qua + 18h provisional hôm nay) + 9h thứ Bảy (finalize thứ Sáu + weekend check); (2) BD Team KPI (cũ) — contract/proposal/meeting/contact mỗi tuần, đặt lại theo tuần. Trigger khi: user nói 'outreach hôm nay bao nhiêu', 'KPI tuần này', 'target tuần này', 'đặt KPI', 'còn thiếu bao nhiêu người/contract', 'tiến độ KPI'. Cũng chạy khi cron trigger MORNING_UPDATE, DAILY_PROGRESS, WEEKEND_KPI_CHECK, WEEKLY_KPI_PROMPT, WEEKLY_KPI_REMINDER, WEEKLY_KPI_MONITOR."
metadata: { "openclaw": { "emoji": "🎯" } }
---

# SME KPI — Outreach & Contract Tracker

Skill có 2 mảng tách biệt:
- **Outreach KPI** (LinkedIn, activity-based, đặt 1 lần) — xem mục đầu tiên, đây là mảng chính đang dùng hàng ngày.
- **BD Team KPI** (CRM, outcome-based: contract/proposal/meeting/contact, đặt lại mỗi tuần) — xem mục "BD TEAM KPI" cuối file, giữ lại cho ai còn cần track theo kiểu cũ.

---

## OUTREACH KPI — LinkedIn Daily Target (mảng chính)

Target **số profile có tin nhắn LinkedIn gửi đi trong ngày** — KHÔNG phải connection request, KHÔNG phải kết quả CRM. Đặt **1 LẦN DUY NHẤT**, áp dụng mãi cho tới khi user chủ động đổi lại — KHÔNG hỏi lại mỗi tuần như KPI kiểu cũ bên dưới.

**Định nghĩa chính xác 1 "outreach" được tính (chốt 2026-09-16):** mỗi PROFILE có tin nhắn (`message_sent`) trong 1 ngày được tính đúng 1 lần cho ngày đó — nhắn qua lại nhiều lần với cùng người trong CÙNG 1 ngày chỉ tính 1, nhưng nhắn cho cùng người ở 2 NGÀY KHÁC NHAU (vd follow-up hôm sau) được tính lại — mỗi ngày độc lập, không nhìn lịch sử trước đó. Connection request (lời mời kết nối) **KHÔNG** được tính vào số này — chỉ tin nhắn thật mới tính.

(Bản trước 2026-09-16 dùng định nghĩa "chỉ tính lần nhắn ĐẦU TIÊN trong toàn bộ lịch sử" — đã bỏ vì gây hiểu lầm là thiếu dữ liệu/lỗi khi thực ra do follow-up hợp lệ bị loại.)

Giới hạn đã biết: khớp theo TÊN hiển thị (data scrape từ inbox LinkedIn không có link profile riêng cho mỗi tin nhắn) — 2 người trùng tên hiển thị có thể bị gộp nhầm thành 1, hiếm gặp nhưng có thể xảy ra.

### COMMANDS

```bash
sme-cli kpi outreach-target set --count 30 [--set-by TEN]   # 30 người/ngày → tuần = 150 (T2-T6, weekly luôn = daily×5, không lưu riêng)
sme-cli kpi outreach-target get                              # xem target hiện tại
sme-cli kpi outreach-status [--date YYYY-MM-DD]              # tiến độ hôm nay + tuần này — CHỈ dùng khi user hỏi trực tiếp (ad-hoc), KHÔNG dùng cho cron
sme-cli kpi outreach-report --slot morning|evening|weekend [--date YYYY-MM-DD]  # lệnh DUY NHẤT cho cả 3 cron slot — xem FLOW bên dưới
```

`outreach-report` tự finalize/snapshot đúng ngữ nghĩa cho từng slot (xem bảng dưới) và đếm trực tiếp từ bảng `outreach_events` — KHÔNG cần ai nhập tay số liệu.

### BẮT BUỘC: sync trước khi báo cáo — CÓ GIỚI HẠN THỜI GIAN

Trước MỌI lần chạy `kpi outreach-report` cho 1 trong 3 cron dưới đây, PHẢI chạy sync với timeout riêng, KHÔNG chạy trần:

```bash
timeout 30 sme-cli outreach sync
```

Đọc CDP LinkedIn — xem "SYNC LINKEDIN" trong `outreach/SKILL.md`, chỉ đọc, KHÔNG click/type gì. Nếu lệnh trên timeout hoặc lỗi (Chrome/CDP treo hoặc down): **DỪNG NGAY, KHÔNG thử lại, KHÔNG chờ thêm** — chuyển sang `kpi outreach-report` luôn bằng data cũ đã có, nói rõ trong tin nhắn "dữ liệu tới lần sync gần nhất" — KHÔNG báo số liệu = 0 khi thực ra là sync fail, và **KHÔNG được để cả job treo tới hết timeout 300s** — đây chính là nguyên nhân đã gây lỗi thật (15/09/2026: sync treo → toàn bộ job timeout 300s, 4 lần liên tiếp không gửi được tin nào).

### THIẾT KẾ MỚI (16/09/2026) — chỉ 2 mốc/ngày, có FINAL và PROVISIONAL rõ ràng

Thiết kế cũ (3 mốc/ngày 9h/13h/18h, coi 18h là "tổng kết") có 1 lỗi thật: tin nhắn gửi sau 18h bị mất — LinkedIn đổi nhãn "hôm nay" thành "Yesterday" qua sáng hôm sau, và script quét cũ dừng lại ngay khi gặp nhãn khác "hôm nay". Đã vá ở tầng scrape (đọc thêm được nhãn "Yesterday"), nhưng thiết kế cron/report cũng đổi theo để phản ánh đúng: **18h KHÔNG BAO GIỜ là số cuối cùng** — số thật ("FINAL") chỉ chốt được vào sáng hôm sau, sau khi sync đã vét nốt phần tối hôm qua.

| Payload | Cron | Ý nghĩa |
|---|---|---|
| `MORNING_UPDATE` | `0 9 * * 1-5` | Finalize ngày làm việc liền trước (Thứ Hai finalize CẢ TUẦN TRƯỚC thay vì 1 ngày) + báo target hôm nay |
| `DAILY_PROGRESS` | `0 18 * * 1-5` | Snapshot **PROVISIONAL** — chỉ là ảnh chụp tạm, KHÔNG phải "tổng kết ngày" |
| `WEEKEND_KPI_CHECK` | `0 9 * * 6` | Finalize thứ Sáu + báo tiến độ tuần (tuần vẫn PROVISIONAL vì Sat/Sun còn phát sinh thêm) |

**KHÔNG có cron Chủ Nhật.** Hoạt động outreach thứ Bảy/Chủ Nhật vẫn được tính đủ — chỉ là chờ tới sáng Thứ Hai mới được cộng chính thức vào tuần trước, bằng cách tính lại trực tiếp theo khoảng ngày thật (Mon→Sun), không phụ thuộc cơ chế nhãn "Yesterday" (cơ chế đó chỉ để KHÔNG BỊ MẤT dữ liệu lúc sync, không phải cách tính tổng).

### FLOW — MORNING_UPDATE (9h, T2-T6)

1. `timeout 30 sme-cli outreach sync`
2. `sme-cli kpi outreach-report --slot morning`
3. Nếu `target_set = false` → gửi 1 dòng nhắc đặt target lần đầu, dừng.
4. **Nếu hôm nay là Thứ Hai** (`job_type = "WEEKLY_SUMMARY+MORNING_UPDATE"`) → gửi **2 phần trong 1 tin**, KHÔNG thêm gì khác:

```
@Hans_Dang 📊 Tổng kết tuần trước ({prev_week_start}-{prev_week_end}, FINAL): {prev_week_actual}/{prev_week_target} người ({prev_week_achievement_pct}%) — {prev_week_new} mới, {prev_week_followup} follow-up.

🌅 Sáng thứ Hai! Hôm nay target {day_target} người. Tuần mới bắt đầu.
```

5. **Thứ Ba → Thứ Sáu** (`job_type = "MORNING_UPDATE"`) → gửi đúng 1 tin:

`@Hans_Dang Sáng rồi anh 🌅 Hôm qua ({prev_day}, FINAL): {prev_day_actual}/{prev_day_target} người ({prev_day_new} mới, {prev_day_followup} follow-up). Hôm nay target {day_target} người. Tuần này đang {week_actual_so_far}/{week_target} (còn {week_remaining}).`

### FLOW — DAILY_PROGRESS (18h, T2-T6) — LUÔN LÀ PROVISIONAL

1. `timeout 30 sme-cli outreach sync`
2. `sme-cli kpi outreach-report --slot evening`
3. Gửi đúng 1 tin — **BẮT BUỘC có chữ "tạm"/"chưa final"**, KHÔNG được dùng chữ "tổng kết ngày":

`@Hans_Dang Cập nhật cuối giờ chiều (số tạm, chưa final) 🕕 Đã nhắn tin cho {day_actual}/{day_target} người hôm nay ({day_new} mới, {day_followup} follow-up). {replies_today} reply. Tuần này: {week_actual}/{week_target}. Số chính thức của hôm nay sẽ chốt vào sáng mai.`

### FLOW — WEEKEND_KPI_CHECK (9h, CHỈ Thứ Bảy — KHÔNG chạy Chủ Nhật)

1. `timeout 30 sme-cli outreach sync`
2. `sme-cli kpi outreach-report --slot weekend`
3. Gửi đúng 1 tin (LUÔN gửi, không áp dụng NO_REPLY — khác thiết kế cũ, vì đây còn là dịp finalize thứ Sáu, không chỉ là nhắc nhở):

`@Hans_Dang Thứ Sáu ({friday_date}) đã chốt FINAL: {friday_actual}/{friday_target} người ({friday_new} mới, {friday_followup} follow-up). Tuần này (tạm tính, còn Thứ Bảy/CN): {week_actual}/{week_target} người ({achievement_pct}%) — {week_new} mới, {week_followup} follow-up, còn thiếu {week_remaining} để đạt target tuần.`

### QUY TẮC — Outreach KPI

- Đúng 1 tin mỗi lần (Thứ Hai được phép 2 đoạn trong CÙNG 1 tin, không phải 2 tin riêng), có `@Hans_Dang`, không thêm markdown thừa/phân tích/gợi ý action (khác FLOW 3 MONITOR ở mục KPI cũ — đó là khi user CHỦ ĐỘNG hỏi mới phân tích sâu).
- Số liệu lấy nguyên từ `outreach-report`, KHÔNG tự tính lại/làm tròn khác đi.
- **KHÔNG BAO GIỜ** gọi số liệu 18h là "tổng kết"/"final"/"cuối cùng" — luôn phải kèm "tạm"/"provisional". Chỉ số liệu từ `outreach-report --slot morning` (field có hậu tố `_actual`, không phải `_so_far`) và `--slot weekend` (field `friday_actual`) mới là FINAL.
- KHÔNG tự chạy `outreach-report` slot nào ngoài 3 slot cron này để tránh ghi đè snapshot ngoài ý muốn — muốn xem tạm thời cho user hỏi ngang thì dùng `outreach-status` (ad-hoc, không ghi snapshot).
- **Metric phụ New/Follow-up:** `_new` = lần đầu tiên trong lịch sử nhắn cho người đó rơi vào đúng ngày/tuần đang báo cáo; `_followup` = người đó đã có tin nhắn trước đó, hôm nay/tuần này chỉ là tiếp tục. `_new + _followup` LUÔN bằng đúng `_actual` cùng field — đây là breakdown, không phải định nghĩa đếm khác.

---

## BD TEAM KPI (kiểu cũ) — Weekly Contract Tracker

Skill theo dõi vòng lặp 4 lớp: **đặt KPI → monitor → diagnose → đề xuất action**. Dựa trên kết quả CRM (contract/proposal/meeting/contact), đặt lại theo từng tuần — khác với Outreach KPI ở trên.

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

### RANH GIỚI — "goal" KHÁC "KPI", KHÔNG được dùng nhầm lệnh

**KPI (skill này)** = chỉ 1 loại số cố định mỗi tuần chuẩn (contract/proposal/meeting/contact mới), theo `sme-cli kpi set`.

**Goal (KHÔNG PHẢI skill này)** = mục tiêu tự do, bất kỳ metric nào, deadline tự chọn (không nhất thiết "tuần này") — vd
"đặt goal tuần này: 2 proposal mới", "mục tiêu tháng này 10 qualified lead", "goal quý này X hợp đồng".

**Cách phân biệt:** nếu user dùng đúng từ **"goal"/"mục tiêu"** (không nói "KPI"), hoặc target không phải 1 trong 4 field
cố định của KPI (contract/proposal/meeting/contact) — đây là **goal**, PHẢI dừng lại, KHÔNG gọi `sme-cli kpi set`.
Thay vào đó nói rõ: "Đây là goal chứ không phải KPI tuần — em dùng `sme-cli goal set` để lưu, không lẫn vào KPI"
và gọi đúng `sme-cli goal set --text "<nguyên văn>" --metric <qualified_leads|proposals|contracts> --target N --deadline YYYY-MM-DD`
(xem `orchestrator/SKILL.md` phần GOAL PERSISTENCE để biết cú pháp đầy đủ).

Case đã xảy ra thật (26/08/2026, E2E test): "Đặt goal tuần này: 2 proposal mới" bị nhầm gọi `kpi set --proposals 2`
— SAI, phải gọi `goal set --metric proposals --target 2`.

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
