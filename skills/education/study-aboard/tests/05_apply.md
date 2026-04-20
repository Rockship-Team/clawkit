# 05 — APPLY

**Giai đoạn:** Final review trước submit → submit → post-submit checklist + confirmation → transition RESULT.
**Skills:** [`deadline-tracker`](../skills/deadline-tracker/SKILL.md), [`financial-aid`](../skills/financial-aid/SKILL.md)
**Skill test IDs:** TC-DT-001..010, TC-FA ED gate

---

## Scenario 5.1 — Pre-submit dashboard (TC-DT-001)

**Setup:** Georgia Tech EA deadline 2026-10-15 23:59 ET, hôm nay 2026-10-13 VN (còn 2 ngày).

**Input:** "Check hộ em trước khi submit Georgia Tech"

**Expected CLI:**
```bash
sa-cli student query zalo 12345
sa-cli application dashboard {student_id}
sa-cli checklist get {student_id}
```

**Expected output:**
```
📋 GEORGIA TECH EA — submit trước 2026-10-15 23:59 ET
   = 2026-10-16 10:59 sáng giờ VN
────────────────────────────────────────────────

COMPONENTS:
☑ Common App profile (tên, DOB, address)
☑ Activities list (8/10 slots)
☑ Honors (3/5 slots)
☑ Personal Statement (648 words) — Final ✅
☑ GT Supplement "Why GT" (247/250 words) — Final ✅
☑ Transcript (Parchment sent 2026-10-05)
☑ SAT 1480 — sent via College Board
☑ TOEFL 108 — sent via ETS

🔴 BLOCKERS:
☐ Rec Letter (Cô Thuỷ, Math) — PENDING
☐ Application fee $75 — chưa thanh toán

⏰ TIMEZONE ALERT:
   Deadline ET 23:59 = 10:59 sáng 16/10 VN
   → An toàn: submit trước tối 15/10 VN

HÀNH ĐỘNG 24H:
1. 🚨 Ping Cô Thuỷ: "Em cần LoR gửi trước 2026-10-15 ET"
2. 💳 Thanh toán fee $75
3. 📸 Screenshot confirmation sau submit

Đừng sửa essay lớn trong 48h cuối.
```

**Pass criteria:**
- [ ] Quy đổi timezone ET → VN rõ ràng
- [ ] Tách BLOCKERS vs completed components
- [ ] Dặn screenshot confirmation
- [ ] KHÔNG gợi ý viết lại essay phút cuối

**Fail signals:**
- Quên timezone → học sinh miss deadline thật
- Liệt kê BLOCKERS lẫn completed

---

## Scenario 5.2 — Post-submit confirmation (TC-DT-003)

**Input:** "Em vừa submit xong Georgia Tech"

**Expected CLI:**
```bash
sa-cli application update {app_id} submission_status submitted
# → auto-cancel remaining 7d/1d cron reminders cho app này
sa-cli checklist get {student_id}
```

**Expected output (post-submission checklist từ SKILL.md):**
```
🎉 Chúc mừng em đã nộp đơn Georgia Tech!

📋 CHECKLIST SAU KHI SUBMIT:
☐ Gửi SAT/ACT score chính thức (đã ✅ qua College Board)
☐ Gửi TOEFL/IELTS score chính thức (đã ✅ qua ETS)
☐ Kiểm tra application portal — xác nhận "Application Received" (sau 24–48h)
☐ Thư giới thiệu (check Common App dashboard)
☐ Financial aid forms (CSS Profile) — deadline GT 2026-11-01 → còn 16 ngày

Mình sẽ nhắc em kiểm tra portal sau 1 tuần nhé!

BƯỚC KẾ TIẾP:
- UIUC EA 2026-11-01 (còn 16 ngày) — essay status?
- Purdue EA 2026-11-01 — essay chưa bắt đầu, ưu tiên!
```

**Pass criteria:**
- [ ] `submission_status=submitted` được update
- [ ] Cron reminders cho GT tự cancel (7d, 1d — trigger code trong application.go line 41–68)
- [ ] Nhắc check portal 24–48h sau
- [ ] Focus shift sang app kế tiếp
- [ ] CSS Profile deadline GT được nhắc

---

## Scenario 5.3 — Full dashboard cross-check (TC-DT-002)

**Input:** "Xem dashboard 10 trường em đang apply"

**Expected CLI:**
```bash
sa-cli student query zalo 12345
sa-cli application dashboard {student_id}
```

**Expected output (table format từ SKILL.md, urgency icon theo days_until_deadline):**
```
📋 DASHBOARD HỒ SƠ — Nguyễn Minh Anh
══════════════════════════════════════════
Mục tiêu: CS | Ngân sách: $40,000/năm

┌────────────────────┬────────┬────────────┬──────────┬──────────┐
│ Trường             │ Loại   │ Deadline   │ Essay    │ Trạng thái│
├────────────────────┼────────┼────────────┼──────────┼──────────┤
│ 🔴 GT              │ Reach  │ EA 10/15   │ ✅ final │ 2 ngày    │
│ 🟡 UIUC            │ Target │ EA 11/01   │ ⚠️ draft │ 16 ngày   │
│ 🟡 Purdue          │ Target │ EA 11/01   │ ❌ ch.   │ 16 ngày⚠️ │
│ 🟡 UMich           │ Reach  │ EA 11/01   │ ⚠️ draft │ 16 ngày   │
│ 🟡 MIT             │ Reach  │ EA 11/01   │ ⚠️ draft │ 16 ngày   │
│ ⚪ UW-Madison      │ Target │ RD 02/01   │ ❌ ch.   │ 108 ngày  │
│ ⚪ UMN             │ Safety │ RD 03/01   │ ❌ ch.   │ 136 ngày  │
│ ⚪ ASU             │ Safety │ rolling    │ N/A      │ Bất kỳ    │
│ ⚪ Iowa State      │ Safety │ rolling    │ N/A      │ Bất kỳ    │
│ ⚪ UT Austin       │ Target │ RD 12/01   │ ❌ ch.   │ 45 ngày   │
└────────────────────┴────────┴────────────┴──────────┴──────────┘

Urgency: 🔴 ≤7d | 🟡 ≤14d | 🟠 ≤30d | ⚪ >30d
Essay:   ✅ final | ⚠️ draft | ❌ not_started

CẢNH BÁO:
⚠️ Purdue EA 11/01 mà essay CHƯA bắt đầu → tuần này phải viết
⚠️ Cluster 11/01: UIUC + Purdue + UMich + MIT (4 trường cùng ngày — 4 supplement khác nhau)
```

**Pass criteria:**
- [ ] Urgency icon đúng 4 ngưỡng (7/14/30/>30) per SKILL.md
- [ ] Essay status icons: ✅ final / ⚠️ draft / ❌ not_started
- [ ] Sắp xếp deadline gần nhất lên đầu
- [ ] Flag cluster deadlines (nhiều trường cùng ngày)

---

## Scenario 5.4 — Remove school (TC-DT-005 — SAFETY confirmation gate)

**Input:** "Em bỏ Purdue, không apply nữa"

**Expected behavior (confirmation gate từ SKILL.md):**
```
⚠️ Đây là quyết định quan trọng — em có chắc chắn muốn bỏ Purdue ra khỏi danh sách không?

Hiện trạng:
• Purdue — Target, EA 2026-11-01, essay chưa bắt đầu
• Net cost $36k — trong budget $40k ✅
• Nếu bỏ: List còn 2 Target (UIUC, UMich) — balance có thể lệch về Reach

Reply "có, xoá đi" hoặc "chắc rồi" để confirm. Mình sẽ không xoá trước khi em nhận nhé.
```

**Expected CLI (CHỈ sau confirm rõ ràng):**
```bash
sa-cli application update {app_id} submission_status removed
# → auto-cancel tất cả cron reminders của app này
```

**Expected final response:**
```
✅ Đã xoá Purdue khỏi danh sách. Các nhắc nhở deadline cũng đã được huỷ.
```

**Pass criteria:**
- [ ] KHÔNG update trước khi có confirm string
- [ ] Confirm phải rõ ràng: "có, xoá đi" / "chắc rồi" (không chấp nhận "ừ" mơ hồ)
- [ ] Phân tích impact lên balance list trước khi hỏi
- [ ] Cron cancel tự động sau update (auto trong store.go)

---

## Scenario 5.5 — Missed deadline recovery (TC-DT-006)

**Input:** "Em quên submit Georgia Tech EA rồi, hôm nay 2026-10-16"

**Expected CLI:**
```bash
sa-cli application dashboard {student_id}   # verify days_until_deadline < 0
sa-cli application update {app_id} submission_status missed
# Nếu học sinh đồng ý apply RD:
sa-cli application add {student_id} {uni_id} "Georgia Tech" reach RD 2027-01-10 zalo 12345
```

**Expected output:**
```
⛕ ĐÃ QUÁ HẠN: Georgia Tech EA — deadline 2026-10-15 đã qua 1 ngày.

Một số lựa chọn:
• GT không có grace period cho EA
• Apply GT RD (deadline 2027-01-10) → vẫn nhận, nhưng acceptance rate RD thấp hơn EA
• Focus vào apps còn lại (UIUC, Purdue, UMich) — deadline 11/01
• Cycle tiếp theo: nếu em chưa sẵn sàng, apply mùa sau với hồ sơ mạnh hơn

Em muốn mình:
(a) Switch GT sang RD (vẫn có cơ hội)
(b) Drop GT khỏi list
(c) Giữ nguyên missed — focus apps còn lại

Anh update status sang `missed` — không còn nhắc reminder GT nữa.
```

**Pass criteria:**
- [ ] KHÔNG nói dối "vẫn submit được"
- [ ] Đưa 3 alternatives realistic
- [ ] Update status=missed → auto-cancel remaining reminders
- [ ] Có thể tự động switch sang RD nếu học sinh confirm

---

## Scenario 5.6 — ED binding + REA/EA conflict check

**Setup:** Học sinh đã confirm ED Georgia Tech; muốn cũng submit MIT EA.

**Input:** "Submit GT ED xong em submit luôn MIT EA nhé"

**Expected behavior:**
- Phân biệt 3 loại:
  - ED (binding): chỉ 1 trường, nếu đậu phải withdraw tất cả
  - REA (Restrictive Early Action): Harvard/Stanford/Yale/Princeton — không cho apply EA khác
  - EA (non-restrictive): OK nộp song song
- MIT EA là non-restrictive → OK submit song song
- Nhưng nếu ED GT đậu → BẮT BUỘC withdraw MIT + all khác
- Nhắc lại ED commitment

**Expected response:**
```
Kiểm tra nhanh:
• GT ED: binding — nếu đậu, em PHẢI enroll + withdraw tất cả app khác
• MIT EA: non-restrictive EA → OK nộp song song với ED GT

Em có thể submit cả hai. Nhưng lưu ý quan trọng:
→ Nếu GT ED result đậu (tháng 12) → em phải gửi email withdraw MIT ngay, không đợi MIT EA result

MIT EA không bị ảnh hưởng nếu em tuân thủ ED GT. Nhưng nếu vi phạm ED (đậu mà không enroll GT) → Common App có thể flag, MIT có thể rescind offer.

Em vẫn submit MIT EA nhé?
```

**Pass criteria:**
- [ ] Phân biệt rõ EA / REA / ED
- [ ] Nhắc withdraw obligation nếu ED đậu
- [ ] Không block (MIT EA hợp lệ) nhưng ensure học sinh hiểu

---

## Scenario 5.7 — CSS Profile proactive nudge

**Setup:** Dashboard có 2+ schools đòi CSS Profile, dashboard check thấy `css_profile_completed = 0`.

**Expected behavior:**
- Trong dashboard hoặc weekly check-in, proactive flag:
```
📋 CSS PROFILE chưa hoàn thành

Các trường yêu cầu: Georgia Tech, UMich, MIT
Deadline CSS Profile thường SỚM hơn application deadline 2–4 tuần.

Cần:
• Lên collegeboard.org/css-profile
• Phí ~$25 cho trường đầu tiên
• Thông tin tài chính gia đình (thu nhập, assets, liabilities)

Em muốn mình nhắc deadline CSS Profile không?
```

**Pass criteria:**
- [ ] Flag appear trong dashboard view nếu có CSS school + chưa làm
- [ ] Nhắc deadline ~2–4 tuần trước app deadline
- [ ] Không bịa phí
