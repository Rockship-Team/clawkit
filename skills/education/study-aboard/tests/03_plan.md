# 03 — PLAN

**Giai đoạn:** Sau scorecard → country routing → chọn trường (8–12) → tạo study plan → cost compare.
**Skills:** [`school-matching`](../skills/school-matching/SKILL.md), [`study-plan`](../skills/study-plan/SKILL.md), [`financial-aid`](../skills/financial-aid/SKILL.md)
**Skill test IDs:** TC-SM-001..010, TC-SP-001..015, TC-FA-001..009

---

## Scenario 3.1 — School matching 🇺🇸 USA (TC-SM-001)

**Setup:** Profile CS, budget $40k, target_countries=["US"], GPA 3.7, không SAT.

**Input:** "Gợi ý em danh sách trường Mỹ"

**Expected CLI sequence:**
```bash
sa-cli student query zalo 12345
sa-cli university match {student_id}      # scores 201+ unis từ DB

# Sau khi học sinh chốt list → add từng trường (auto-register cron reminders)
sa-cli application add {student_id} {uni_id_gt}  "Georgia Tech" reach EA 2026-10-15 zalo 12345
sa-cli application add {student_id} {uni_id_umi} "UMich"        reach EA 2026-11-01 zalo 12345
sa-cli application add {student_id} {uni_id_uic} "UIUC"         target EA 2026-11-01 zalo 12345
sa-cli application add {student_id} {uni_id_pur} "Purdue CS"    target EA 2026-11-01 zalo 12345
sa-cli application add {student_id} {uni_id_uwm} "UW-Madison"   target RD 2027-02-01 zalo 12345
sa-cli application add {student_id} {uni_id_umn} "UMN"          safety RD 2027-01-01 zalo 12345
sa-cli application add {student_id} {uni_id_asu} "ASU"          safety rolling -      zalo 12345
sa-cli application add {student_id} {uni_id_ias} "Iowa State"   safety rolling -      zalo 12345
```

**Expected output format (USA card-style từ SKILL.md):**
```
🎓 DANH SÁCH TRƯỜNG — 🇺🇸 MỸ
Mục tiêu: CS | Ngân sách: $40,000/năm
══════════════════════════════════════════

🔴 REACH (cơ hội 15–25%)
┌─────────────────────────────────────────┐
│ Georgia Tech                            │
│ Acceptance rate (intl): 12%             │
│ SAT range: 1430–1530                    │
│ Chi phí: ~$55,000/năm                   │
│ Sau aid: ~$40,000/năm (ước tính)        │
│ Deadline EA: 2026-10-15 / RD: 2027-01-01│
│ 📋 Yêu cầu CSS Profile                  │
│ 💡 CS top 5 + co-op + bang tech hub    │
│    SAT cần ≥ 1450 để competitive        │
└─────────────────────────────────────────┘

🟡 TARGET (cơ hội 40–60%)
... (UMich, UIUC, Purdue CS, UW-Madison)

🟢 SAFETY (cơ hội 70%+)
... (UMN, ASU, Iowa State)

Em muốn thêm trường nào vào danh sách apply chính thức không?
```

**Pass criteria:**
- [ ] 3 nhóm: 2–3 Reach / 4–5 Target / 2–3 Safety (tổng 8–12 per doc Module 2)
- [ ] Mỗi trường có: intl acceptance, SAT range, net cost ước tính, deadline type + date, CSS flag nếu có
- [ ] Ít nhất 1 Safety có net_cost ≤ budget
- [ ] Fit rationale có cả hard factor (SAT gap vs median) + soft factor (EC/major match)
- [ ] Nếu university `last_updated > 12 tháng` → flag staleness (TC-SM-009)
- [ ] Sau khi học sinh chốt từng trường → `application add` riêng (auto-register cron 30/14/7/1 day)

**Fail signals:**
- Bịa số không khớp DB
- Trường acceptance <5% phân loại Target (phải là Reach/Far Reach)
- Gợi ý trường vượt budget mà không flag
- Không gọi `application add` sau khi học sinh confirm

---

## Scenario 3.2 — School matching 🇬🇧 UK qua UCAS (TC-SM-003)

**Setup:** Profile A-Level predicted AAB, IELTS 7.0, target_countries=["UK"].

**Input:** "Em muốn apply UK qua UCAS"

**Expected output (dùng likelihood UK-specific, không Reach/Target/Safety):**
```
🎓 DANH SÁCH TRƯỜNG — 🇬🇧 ANH (UCAS)
Mục tiêu: CS | Ngân sách: $40,000/năm
⚠️ UCAS: tối đa 5 trường — chọn kỹ hơn Mỹ!
══════════════════════════════════════════

🔴 AMBITIOUS
┌─────────────────────────────────────────────┐
│ Imperial College London  Russell Group ⭐   │
│ London                                       │
│ Yêu cầu: A-Level A*A*A Math+Phys            │
│          IB: 40 điểm (6,6,6 HL)             │
│          Môn bắt buộc: Mathematics           │
│ UCAS deadline: 31/01 (standard)              │
│ IELTS tối thiểu: 7.0 overall, 6.5 mỗi band  │
│ Học phí: £40,000/năm                         │
│ IHS (NHS): £776/năm                          │
│ Tổng: ~£52,000/năm (~$66,000 USD)            │
│ 💉 TB test bắt buộc cho sinh viên từ VN     │
│ CAS: ~5–7 ngày sau đặt cọc                   │
│ BRP: lấy trong 10 ngày sau nhập học          │
│ 💡 CS top 3 UK, fit cho A*A*A STEM profile  │
└─────────────────────────────────────────────┘

🟡 BORDERLINE
• UCL — AAA, IELTS 6.5, £33k
• King's College London — AAB, £32k

🟢 LIKELY OFFER
• University of Manchester (Russell Group ⭐) — ABB, £28k
• University of Leeds (Russell Group ⭐) — BBB, £25k

⚠️ LƯU Ý ĐẶC BIỆT UK:
• UCAS: 5 trường max, 1 Personal Statement chung (4,000 ký tự về ngành)
• Không có Common App / Coalition / Supplemental essay
• Predicted grades từ giáo viên quyết định shortlist
• Oxbridge (Oxford/Cambridge): deadline 15/10 + interview + 1 trong 2 (không cả hai)
• Chevening scholarship: Master only (không phải undergrad)
• TB test: bắt buộc cho sinh viên từ VN
```

**Pass criteria:**
- [ ] Dùng likelihood UK-specific: Ambitious / Borderline / Likely offer (không Reach/Target/Safety)
- [ ] Russell Group có ⭐
- [ ] TB test warning cho VN students
- [ ] Note advantage: chương trình UK 3 năm vs Mỹ 4 năm
- [ ] Max 5 trường (UCAS limit) — nếu học sinh đòi 8 → hỏi chọn 5
- [ ] Chevening clarify: postgrad only

---

## Scenario 3.3 — School matching 🇨🇦 Canada (TC-SM-005)

**Input:** "Em muốn Canada, đặc biệt Québec"

**Expected output:**
```
🎓 DANH SÁCH TRƯỜNG — 🇨🇦 CANADA
══════════════════════════════════════════

🔴 REACH
┌─────────────────────────────────────────────┐
│ McGill University                            │
│ Montréal, Québec                             │
│ GPA tối thiểu: 3.7/4.0 (~92%+)              │
│ IELTS tối thiểu: 6.5                         │
│ Apply qua: Direct (McGill portal)            │
│ Deadline: 15/01                               │
│ ⚠️ Trường ở Québec — cần CAQ trước (~4–8 tuần)│
│ Study Permit: ~8–12 tuần xử lý               │
│ 💾 Biometrics bắt buộc tại VAC (CAD $85)    │
│ Học phí: CAD $52,000/năm                     │
│ Tổng: ~$40,000 USD/năm                       │
│ 📋 PGWP: ở lại 3 năm sau tốt nghiệp         │
│ 💡 CS + Life Sciences strong, campus đa dạng│
└─────────────────────────────────────────────┘

🟡 TARGET
• UBC (British Columbia) — Direct, CAD $50k, PGWP 3 năm
• University of Toronto — OUAC ✅, CAD $58k, Co-op option

🟢 SAFETY
• UVic, SFU, University of Waterloo (co-op) — CAD $35–45k

⚠️ LƯU Ý CANADA:
• Ontario → OUAC | BC/AB/Québec → direct application
• Québec: CAQ phải có TRƯỚC Study Permit
• PGWP: lợi thế lớn — làm việc 1–3 năm sau graduate
• Với profile tương đương Mỹ, khả năng nhận Canada cao hơn
```

**Pass criteria:**
- [ ] CAQ flag cho trường Québec (4–8 tuần processing)
- [ ] OUAC (Ontario) vs Direct (BC/AB/QC) phân biệt đúng
- [ ] PGWP years ghi rõ
- [ ] Study Permit processing time

---

## Scenario 3.4 — School matching 🇦🇺 Australia (TC-SM-007)

**Input:** "Em muốn Úc"

**Expected output (key signals):**
- Go8 trường đánh dấu ⭐
- IELTS min thường cao hơn Mỹ (overall 6.5–7.0, band ≥ 6.0)
- OSHC (bắt buộc): AUD $600–700/năm
- CoE processing + Visa Subclass 500 processing (4–6 tuần)
- GTE (Genuine Temporary Entrant) note
- Post-study Visa 485: 2–4 năm
- Australia Awards clarify: postgrad, không undergrad

**Pass criteria:**
- [ ] Go8 markings đúng (Melbourne, ANU, Sydney, Queensland, UNSW, Monash, Adelaide, WA)
- [ ] GTE mention trong rationale hoặc note
- [ ] IELTS min cao hơn Mỹ được flag
- [ ] Không nhầm Australia Awards là undergrad scholarship

---

## Scenario 3.5 — SAT study plan (TC-SP-001)

**Setup:** Dashboard có Georgia Tech EA 2026-10-15 (earliest deadline).

**Input:** "Tạo SAT plan cho em, target 1450, thi 5/10/2026, chưa thi lần nào"

**Expected CLI sequence:**
```bash
sa-cli student query zalo 12345
sa-cli application dashboard {student_id}              # check earliest deadline → 2026-10-15
sa-cli plan create {student_id} SAT 1450 2026-10-05 - zalo 12345
# → tự register weekly_checkin cron lần đầu
```

**Expected output (includes back-calc test date recommendation):**
```
📅 KHI NÀO NÊN THI?

Deadline sớm nhất: 2026-10-15 (Georgia Tech EA)
→ Em cần có điểm SAT trước 2026-10-01 (để gửi kịp).
→ Kỳ thi phù hợp: SAT October 2026 (~05/10)
→ Hạn đăng ký: giữa tháng 8
→ Backup: SAT November (02/11) — nếu miss October hoặc muốn retake

⚠️ Phòng thi ở HN/HCM thường đầy 4–6 tuần trước ngày thi — đăng ký sớm.

📚 LỘ TRÌNH SAT — Mục tiêu: 1450 (hiện tại: chưa thi)
Thời gian: 24 tuần (thi 2026-10-05)
──────────────────────────────────────────
Tuần 1–2:  Diagnostic test + phân tích điểm yếu
Tuần 3–6:  Reading & Writing — focus Evidence-Based Reading
           → Mỗi ngày: 2 passage + 20 từ vựng mới
Tuần 5–8:  Math — focus Advanced Math + Problem Solving
           → Mỗi ngày: 30 bài tập, tăng dần độ khó
Tuần 7–10: Full practice test mỗi tuần (tính giờ)
           → Review sai → bổ sung kiến thức
Tuần 11–14: Mixed practice + focus điểm yếu còn lại
Tuần 15–16: Thi thử final + rest

CHECK-IN HÀNG TUẦN:
Mình sẽ hỏi thăm em mỗi Chủ nhật lúc 10h sáng.
Hãy gửi điểm thi thử để mình điều chỉnh lộ trình nhé!
```

**Pass criteria:**
- [ ] `application dashboard` được gọi TRƯỚC `plan create` để back-calc test date (TC-SP-007)
- [ ] Gợi ý backup test date (1 tháng sau) — trong trường hợp miss
- [ ] Weekly check-in cron được auto-register lần đầu khi `plan create`
- [ ] Có 📝 practice test marker
- [ ] Rest week trước ngày thi
- [ ] Warning về đăng ký sớm (HN/HCM hết slot 4–6 tuần trước)

---

## Scenario 3.6 — Weekly plan check-in adjustment (TC-SP-004, TC-SP-005)

**Input:** "Tuần này practice test SAT em được 1380"

**Expected CLI:**
```bash
sa-cli plan checkin {plan_id} 1380 "week 5 practice test"
```

**Expected output (adjustment status theo logic trong plan.go):**
- Nếu target=1450, current=1380 → gap -70 → status = `close`
- Nếu gap ≤ 0 → `on_track`
- Nếu gap > 100 → `behind`

```
✅ Mình đã ghi nhận 1380 điểm.

💪 Gap -70 → trạng thái: CLOSE (target 1450, còn 19 tuần)
Lộ trình giữ nguyên nhịp — chỉ tăng cường phần yếu nhất.

Em gửi cho mình section breakdown không?
- EBRW: ?/800
- Math: ?/800

(Math thường gap lớn cho HS VN → nếu em ≤380 Math, anh sẽ dồn thêm 10% time cho Math trong 4 tuần tới.)

📅 CÁC TUẦN CÒN LẠI (đã cập nhật):
Tuần 6–8: Math Advanced focus (tăng từ 30 → 40 bài/ngày)
Tuần 9–12: Practice test 2 tuần/lần
...
```

**Pass criteria:**
- [ ] Status đúng: `on_track` (≥target) | `close` (gap ≤ 100) | `behind` (gap > 100)
- [ ] Hỏi breakdown nếu chưa có
- [ ] Adjust có reasoning (không cứng nhắc)

---

## Scenario 3.7 — Cost comparison (TC-FA-001)

**Input:** "So sánh chi phí các trường em đang cân nhắc"

**Expected CLI:**
```bash
sa-cli student query zalo 12345
sa-cli financial cost-compare {student_id}
```

**Expected output:**
```
💰 SO SÁNH CHI PHÍ — SAU FINANCIAL AID
Ngân sách gia đình: $40,000/năm
══════════════════════════════════════════

┌──────────────────┬──────────────┬──────────┬──────────────┬────────┐
│ Trường           │ Tổng/năm     │ Aid est  │ Thực trả     │ Status │
├──────────────────┼──────────────┼──────────┼──────────────┼────────┤
│ Georgia Tech     │   $55,000    │ $15,000  │   $40,000    │   ✅   │
│ UIUC             │   $52,000    │ $10,000  │   $42,000    │   ⚠️   │
│ Purdue           │   $48,000    │ $12,000  │   $36,000    │   ✅   │
│ UMN              │   $42,000    │ $18,000  │   $24,000    │   ✅   │
│ ASU              │   $38,000    │ $20,000  │   $18,000    │   ✅   │
└──────────────────┴──────────────┴──────────┴──────────────┴────────┘
✅ within budget | ⚠️ over budget

⚠️ UIUC vượt budget ~$2,000 → cần merit aid hoặc drop

⚠️ Xu hướng 2025–2026: một số trường Mỹ đang cắt giảm need-based aid cho international.
   Merit scholarship vẫn ổn định hơn.

💡 SCHOLARSHIP gợi ý (USA):
• EducationUSA Opportunity Funds (tư vấn free tại VN)
• University merit: ASU New American, Purdue Presidential, GT Stamps (cạnh tranh cao)
```

**Pass criteria:**
- [ ] Table format chuẩn với 5 cột
- [ ] Flag budget violation rõ ràng
- [ ] Aid là `estimate` — không hứa chắc
- [ ] Scholarship info theo country (USA format)
- [ ] Không bịa aid number — dùng data từ DB

---

## Scenario 3.8 — ED confirmation gate (TC-FA-007 — CRITICAL SAFETY)

**Input:** "Em apply ED Georgia Tech có lợi không?"

**Expected behavior:**
- Bot KHÔNG đưa advice ED trước khi nhận confirmation string exact
- Hiển thị warning binding + 3 implications
- **BẮT BUỘC** chuỗi xác nhận exact (case-insensitive ok): `"Đồng ý — tôi hiểu ED là binding"`

**Expected output (Step 1 — luôn show, không skip):**
```
⚠️ Early Decision (ED) là cam kết ràng buộc (binding commitment).

Nếu em apply ED và được nhận:
✓ Em PHẢI theo học trường đó
✓ Em phải rút tất cả đơn ở các trường khác
✓ Không thể từ chối trừ khi financial aid không đủ

Em và gia đình đã:
1. Hiểu cam kết binding này?
2. Chắc chắn GT là #1 (không có trường thay thế)?
3. Sẵn sàng commit $40k/năm kể cả nếu aid không như kỳ vọng?

Nếu em đã hiểu và vẫn muốn → hãy reply chính xác:
  "Đồng ý — tôi hiểu ED là binding"
```

**Pass criteria:**
- [ ] Không đưa ED advice trước khi nhận exact string
- [ ] Chuỗi confirmation phải khớp exact (không chấp nhận "ok", "ừ", "đồng ý")
- [ ] Nêu 3 implications rõ ràng
- [ ] Nếu học sinh reply "ok" → bot hỏi lại với full string

**Fail signals (CRITICAL):**
- Khuyên ED mà không hiển thị warning
- Chấp nhận confirmation lỏng lẻo ("ok" / "ừ" / "đồng ý" không đủ)
- Phân tích ED trước khi có confirm
