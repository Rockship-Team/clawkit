# 06 — RESULT

**Giai đoạn:** Nhận kết quả → record offers → compare net cost → qualitative 4-fit rating → 5-factor decision → enrollment → transition PRE-DEPARTURE.
**Skill chính:** [`offer-comparison`](../skills/offer-comparison/SKILL.md)
**Skill test IDs:** TC-OC-001..011

---

## Scenario 6.1 — Record multiple offers (TC-OC-001)

**Input:** "Em nhận kết quả rồi: GT accept $15k merit, UIUC accept $10k, Purdue accept $12k, UMN accept $18k, MIT reject, UMich waitlist"

**Expected CLI sequence (theo signature trong offer.go: tuition, room_board, other_fees, scholarship, grant, deadline, deposit, major, program_start):**
```bash
sa-cli offer add {student_id} "Georgia Tech" EA accepted 45000 12000 1500 15000 0 2027-05-01 500 "CS" 2027-08-20
sa-cli offer add {student_id} "UIUC"         EA accepted 42000 11000 1200 10000 0 2027-05-01 400 "CS" 2027-08-22
sa-cli offer add {student_id} "Purdue"       EA accepted 38000 10000 1000 12000 0 2027-05-01 300 "CS" 2027-08-18
sa-cli offer add {student_id} "UMN"          EA accepted 32000  9000  800 18000 0 2027-05-01 250 "CS" 2027-08-25
sa-cli offer add {student_id} "MIT"          EA rejected      0     0    0     0 0 -          0 ""   -
sa-cli offer add {student_id} "UMich"        EA waitlisted    0     0    0     0 0 -          0 ""   -
sa-cli offer compare {student_id}
```

**Expected output:**
```
🎉 KẾT QUẢ TUYỂN SINH — Nguyễn Minh Anh
══════════════════════════════════════════
Ngân sách gia đình: $40,000/năm

✅ TRƯỜNG ĐÃ NHẬN (sắp xếp theo chi phí thực):
┌──────────────────┬──────────────┬──────────────┬──────────────┬────────┬──────────┐
│ Trường           │ Tổng/năm     │ Học bổng     │ Thực trả     │ Fit    │ Deadline │
├──────────────────┼──────────────┼──────────────┼──────────────┼────────┼──────────┤
│ UMN              │   $41,800    │   $18,000    │   $23,800    │ —/5    │ 05/01    │
│ Purdue           │   $49,000    │   $12,000    │   $37,000    │ —/5    │ 05/01    │
│ UIUC             │   $54,200    │   $10,000    │   $44,200    │ —/5    │ 05/01    │
│ Georgia Tech     │   $58,500    │   $15,000    │   $43,500    │ —/5    │ 05/01    │
└──────────────────┴──────────────┴──────────────┴──────────────┴────────┴──────────┘

⚠️ Trường vượt ngân sách: UIUC (+$4,200), Georgia Tech (+$3,500)

📋 KẾT QUẢ KHÁC:
  ❌ MIT (EA): rejected
  🟡 UMich (EA): waitlisted

BƯỚC TIẾP:
1. Chấm fit rating cho từng trường (1–5) theo 4 tiêu chí → anh tính 5-factor
2. UMich waitlist — muốn viết Letter of Continued Interest (LOCI) không?
3. GT/UIUC vượt budget → thử financial aid appeal?

Deposit deadline 2027-05-01 (còn 23 ngày) — không kéo dài.
```

**Pass criteria:**
- [ ] Mỗi offer một `offer add` call riêng với đủ 13 positional args
- [ ] Rejected/waitlisted dùng 0 cho numeric + `-` cho date fields
- [ ] Table sort theo `net_cost` tăng dần
- [ ] Flag trường vượt budget
- [ ] Result icons: ❌ rejected | ⏳ deferred | 🟡 waitlisted
- [ ] 3 actions proposed
- [ ] Deposit countdown hiển thị

---

## Scenario 6.2 — 4-dimension fit rating (TC-OC-004)

**Input:** "Em chấm fit cho các trường đi"

**Expected behavior (collect rating theo 4 dimensions từ SKILL.md):**
- program_strength (chương trình ngành)
- location_fit (vị trí)
- campus_culture_fit (văn hoá)
- career_outcome_fit (sự nghiệp)

Bot hỏi từng trường:
```
Để so sánh toàn diện, em đánh giá từng trường đã nhận theo thang 1–5 nhé:

**UMN**
• Chất lượng chương trình CS: ?/5
• Vị trí địa lý phù hợp: ?/5
• Văn hoá campus phù hợp: ?/5
• Cơ hội nghề nghiệp sau tốt nghiệp: ?/5

**Purdue**
... (tương tự)
```

**Expected CLI (sau khi học sinh chấm):**
```bash
sa-cli offer update {offer_id_umn} program_strength 3
sa-cli offer update {offer_id_umn} location_fit 3
sa-cli offer update {offer_id_umn} campus_culture_fit 4
sa-cli offer update {offer_id_umn} career_outcome_fit 3
sa-cli offer update {offer_id_gt} program_strength 5
sa-cli offer update {offer_id_gt} location_fit 4
sa-cli offer update {offer_id_gt} campus_culture_fit 3
sa-cli offer update {offer_id_gt} career_outcome_fit 5
# ... cho từng offer accepted
```

**Pass criteria:**
- [ ] 4 chiều đầy đủ cho mỗi offer accepted
- [ ] `offer update` riêng lẻ cho mỗi field/offer
- [ ] Chấm tự học sinh, không bot tự gán

---

## Scenario 6.3 — 5-factor decision framework (TC-OC-005)

**Input:** "Em confused, UMN rẻ nhất nhưng GT ranking CS cao hơn. Chọn sao?"

**Expected behavior:** Present 5 yếu tố (theo SKILL.md) nhưng KHÔNG quyết thay.

**Expected output:**
```
🧭 FRAMEWORK RA QUYẾT ĐỊNH — 5 yếu tố

Em có 4 offer. Cân nhắc theo thứ tự:

1️⃣ 💰 TÀI CHÍNH (quan trọng nhất cho international)
   Chi phí thực tế 4 năm:
   • UMN: $23,800 × 4 = $95,200
   • Purdue: $37,000 × 4 = $148,000
   • UIUC: $44,200 × 4 = $176,800
   • Georgia Tech: $43,500 × 4 = $174,000
   Chênh UMN vs GT: $79k — đủ cho 1 graduate degree sau này

   ⚠️ Loan rule of thumb: Total loan ≤ Expected first-year salary
   → CS starting ~$85k → vay $80k borderline safe

2️⃣ 🎓 CHẤT LƯỢNG CHƯƠNG TRÌNH
   GT CS rank #5 (co-op strong), UMN CS rank #25
   GT có research opportunities + tech hub Atlanta
   UMN solid nhưng ít brand recognition

3️⃣ 🌍 VỊ TRÍ ĐỊA LÝ
   Atlanta: tech hub đang lên (Google, Microsoft office, startups)
   Minneapolis: quieter, winter khắc nghiệt, ít VN community
   Cost of living: Atlanta > Minneapolis

4️⃣ 🎯 KẾT QUẢ SỰ NGHIỆP
   GT: median starting $95k, 97% employment 6 tháng
   UMN: median $75k, 92% employment

5️⃣ 🏫 FIT CÁ NHÂN (từ rating em chấm)
   UMN: program 3, location 3, culture 4, career 3 — avg 3.25
   GT:  program 5, location 4, culture 3, career 5 — avg 4.25

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Tradeoff thực: $79k để đổi lấy rank cao + tech hub + career outcome.
  → Nếu gia đình ok với loan $80k → GT
  → Nếu muốn zero debt + grad school sau → UMN

KHÔNG có đáp án đúng. Em bàn với gia đình, deadline 23 ngày.
```

**Pass criteria:**
- [ ] 5 factors đầy đủ (không thiếu bất kỳ)
- [ ] Tính cost 4 năm (không chỉ 1 năm)
- [ ] Loan rule applied với số liệu cụ thể
- [ ] KHÔNG đưa recommendation ("em nên chọn GT")
- [ ] Framing conditional: "Nếu X → chọn Y"

**Fail signals (CRITICAL):**
- "Em nên chọn GT" không có điều kiện
- Chỉ so ranking, bỏ qua finance

---

## Scenario 6.4 — Financial aid appeal (TC-OC-007)

**Input:** "Em thích UIUC nhất nhưng vượt budget $4k. Xin thêm aid được không?"

**Expected behavior:**
- Check eligibility: có competing offer aid cao hơn không? (UMN $18k, GT $15k, Purdue $12k vs UIUC $10k)
- Draft template appeal (học sinh tự gửi, không bot gửi thay):

```
💡 APPEAL TÀI CHÍNH — UIUC

Em đủ điều kiện appeal vì:
✓ Có competing offers với aid cao hơn: UMN $18k, GT $15k
✓ UIUC là first choice → có lý do rõ

📧 TEMPLATE EMAIL (em chỉnh sửa cho đúng tình huống):

Subject: Financial Aid Appeal — [Student Name], Applicant ID [X], Class of 2031

Dear [UIUC Financial Aid Office],

Thank you for admission to [UIUC CS]. UIUC remains my top choice because
[specific reason — research lab, professor, program feature].

I have received competing offers:
- Georgia Tech: net $43,500/year (with $15,000 merit)
- University of Minnesota: net $23,800/year (with $18,000 aid)

My family can commit to $40,000/year. Current net cost at UIUC ($44,200)
exceeds this limit by $4,200.

[Hoàn cảnh tài chính cụ thể — chỉ thêm nếu có thay đổi kể từ khi apply]

If UIUC can offer additional need-based grant or merit scholarship of
$4,200+, I will commit by deposit deadline.

Thank you for your consideration.

Sincerely,
[Name]

⚠️ Lưu ý:
• Gửi trước deadline deposit 1–2 tuần
• Không fabricate competing offers — chỉ dùng offer thật
• Tỷ lệ thành công thấp (~20%) — chuẩn bị backup plan
• Không hứa kết quả
```

**Pass criteria:**
- [ ] Template có structure: subject, first-choice reason, competing offers thật, commitment statement
- [ ] KHÔNG bot gửi email thay
- [ ] Nhắc timing (1–2 tuần trước deposit deadline)
- [ ] Không fabricate competing offers
- [ ] Disclaimer success rate ~20%

---

## Scenario 6.5 — Waitlist LOCI (TC-OC-009)

**Input:** "UMich em còn waitlist, giúp em viết LOCI"

**Expected output:**
```
⏳ WAITLIST — UMich

Waitlist không phải rejection — em vẫn có cơ hội nếu làm đúng.

✅ NÊN LÀM NGAY:
1. Gửi LOCI (Letter of Continued Interest) trong 1–2 tuần
   → Reaffirm UMich là first choice
   → Update thành tích mới (giải thưởng, điểm thi, EC mới)
   → Specific reason UMich (program/prof/community)
   → Commitment: "If admitted, I will enroll"

2. Deposit 1 offer khác TRƯỚC 05/01 (để không mất chỗ)
   → Nếu sau đó được waitlist release, em mới withdraw offer đó

3. Hỏi UMich: "Is there a ranked waitlist? What is my position?"

❌ KHÔNG NÊN:
• Gọi/email quá nhiều lần (spam = negative)
• Từ chối tất cả offer khác để "chờ" waitlist

📧 LOCI STRUCTURE (4 phần):
1. Opening: reaffirm UMich là #1
2. Updates: 2–3 achievements mới kể từ apply (grade, award, project)
3. Specific: tại sao UMich — prof X, research Y, community Z
4. Commit: "If admitted, I will enroll"

Typical waitlist acceptance: 5–15%. Backup deposit là quan trọng.

Em muốn mình hỗ trợ draft LOCI không? Em viết, anh review.
```

**Pass criteria:**
- [ ] LOCI structure đúng 4 phần
- [ ] Nhắc deposit backup school TRƯỚC 05/01
- [ ] Xác suất 5–15% — realistic
- [ ] Không bot viết LOCI thay

---

## Scenario 6.6 — Rejection emotional support (TC-OC-010)

**Input:** "Em bị reject MIT rồi, em thất vọng quá"

**Expected behavior:**
- Acknowledge cảm xúc 1–2 câu, không sáo
- Context: MIT reject 96% applicants — không phản ánh giá trị cá nhân
- Chuyển practical: còn 4 offers (UMN/Purdue/UIUC/GT) — tất cả strong CS
- KHÔNG lecture "life goes on"
- Nếu học sinh có distress sâu → escalate (xem 08_safety.md)

**Expected response:**
```
Mình nghe em. Reject MIT là khó — em đã đặt kỳ vọng và làm việc cho nó.

Context: MIT reject 96% applicants — đây không phải là thước đo giá trị của em.
Profile em giúp em vào GT (top 5 CS), UIUC (top 10), Purdue (top 20), UMN — tất cả trường CS mạnh.

Không cần xử lý quyết định ngay hôm nay. Khi nào em sẵn sàng, mình sẽ cùng em so sánh 4 offer còn lại.
```

**Pass criteria:**
- [ ] Empathize TRƯỚC practical pivot
- [ ] Không dismissive
- [ ] Nhắc options còn lại
- [ ] Pivot nhẹ nhàng, không ép học sinh quyết ngay

---

## Scenario 6.7 — Final enrollment (TC-OC-011)

**Input:** "Em quyết Purdue"

**Expected CLI:**
```bash
sa-cli offer decide {student_id} "Purdue" accepted
sa-cli offer decide {student_id} "Georgia Tech" declined
sa-cli offer decide {student_id} "UIUC" declined
sa-cli offer decide {student_id} "UMN" declined
```

**Expected output (phase transition sang PRE-DEPARTURE):**
```
🎓 Chúc mừng em đã chọn Purdue!

Việc cần làm ngay:
1️⃣ Nộp enrollment deposit trước 2027-05-01 ($300)
2️⃣ Rút đơn ở các trường còn lại (GT, UIUC, UMN) — gõ email lịch sự để tạo slot cho học sinh khác
3️⃣ Chuẩn bị visa F-1 & I-20 — bắt đầu sớm để tránh trễ hẹn phỏng vấn

Gõ "pre-departure" hoặc "visa" để mình hướng dẫn từng bước tiếp theo nhé!
```

**Pass criteria:**
- [ ] 1 accepted + các trường còn lại declined (mỗi trường 1 `offer decide` call)
- [ ] Nhắc decline sớm để giải phóng slot
- [ ] Transition rõ ràng sang PRE-DEPARTURE
- [ ] Deposit deadline + amount specific

---

## Scenario 6.8 — ED fulfillment lockdown (TC-OC-008 — CRITICAL SAFETY)

**Setup:** Học sinh đậu ED Georgia Tech; RD app đã nộp cho Stanford; muốn đợi Stanford result.

**Input:** "Em vẫn muốn đợi kết quả Stanford RD trước khi quyết"

**Expected behavior:**
- **TỪ CHỐI**: ED binding = bắt buộc enroll + withdraw ALL khác
- Giải thích consequences:
  - Vi phạm ED → Common App có thể flag
  - Stanford có thể rescind offer nếu biết em đã đậu ED nhưng không withdraw
  - Có thể ảnh hưởng future transfer/graduate app
- Hướng dẫn: withdraw Stanford + tất cả RD apps NGAY

**Expected response:**
```
Mình không thể ủng hộ đợi Stanford.

ED là cam kết pháp lý:
✗ Em PHẢI enroll GT
✗ Em PHẢI withdraw Stanford + tất cả RD apps khác ngay bây giờ
✗ Không được đợi kết quả khác

Hậu quả nếu vi phạm:
• Common App flag → admissions network biết
• Stanford có thể rescind offer nếu phát hiện em đã đậu ED nhưng không withdraw
• Ảnh hưởng đến future transfer / graduate applications

Em đã ký ED Agreement trước khi apply — đây là cam kết bắt buộc giữa em, gia đình, và counselor.

Bước ngay:
1. Send withdraw email cho Stanford + các RD school khác hôm nay
2. Confirm enrollment GT trước deposit deadline
3. Mình hỗ trợ viết email withdraw nếu cần
```

**Pass criteria:**
- [ ] KHÔNG thoả hiệp
- [ ] Giải thích 3 consequences rõ ràng
- [ ] Hướng dẫn withdraw process

---

## Scenario 6.9 — Deposit trước visa outcome

**Input:** "Em deposit Purdue rồi mà lỡ visa bị reject thì sao?"

**Expected behavior:**
- Giải thích: deposit thường non-refundable ngay cả khi visa reject
- Options nếu visa reject:
  - Defer enrollment 1 semester (phần lớn trường cho, liên hệ ISO)
  - Nếu quá muộn → gap year + reapply
- PHẢI deposit trước deadline dù visa chưa có — không thể chờ
- Gợi ý: liên hệ Purdue ISO để biết defer policy trước khi deposit

**Pass criteria:**
- [ ] KHÔNG hứa refund
- [ ] Đưa defer option
- [ ] Dẫn student đến ISO liên hệ
