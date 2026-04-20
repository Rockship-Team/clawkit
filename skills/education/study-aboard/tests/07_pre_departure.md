# 07 — PRE-DEPARTURE

**Giai đoạn:** Enrolled → country-specific visa + financial proof + housing + health + arrival + orientation.
**Skill chính:** [`pre-departure`](../skills/pre-departure/SKILL.md)
**Skill test IDs:** TC-PD-001..013

---

## Scenario 7.1 — USA F-1 pipeline (TC-PD-002)

**Setup:** Offer decided = Purdue (USA), program_start = 2027-08-18.

**Input:** "Em chọn Purdue xong rồi, giờ cần làm gì?"

**Expected CLI sequence (progressive updates khi học sinh complete từng step):**
```bash
sa-cli student query zalo 12345
sa-cli visa update {student_id} school_name "Purdue"
sa-cli visa update {student_id} program_start_date 2027-08-18
sa-cli visa get {student_id}    # fetch initial checklist

# Các update khi học sinh report done:
sa-cli visa update {student_id} i20_received 1
sa-cli visa update {student_id} sevis_paid 1
sa-cli visa update {student_id} ds160_done 1
sa-cli visa update {student_id} interview_booked 1
sa-cli visa update {student_id} interview_completed 1
sa-cli visa update {student_id} visa_approved 1
sa-cli visa update {student_id} financial_docs_ready 1
sa-cli visa update {student_id} housing_arranged 1
sa-cli visa update {student_id} health_insurance 1
sa-cli visa update {student_id} vaccinations 1
sa-cli visa update {student_id} flight_booked 1
sa-cli visa update {student_id} orientation_registered 1
sa-cli visa update {student_id} course_registration 1
```

**Expected output (initial checklist — USA section từ SKILL.md):**
```
✈️ CHECKLIST CHUẨN BỊ ĐI MỸ — Nguyễn Minh Anh
Trường: Purdue | Bắt đầu: 2027-08-18
══════════════════════════════════════════

📄 I-20 & SEVIS
  ☐ Nhận I-20 từ trường (sau deposit + financial proof + immunization)
  ☐ Nộp phí SEVIS $350 (fmjfee.com) — TRƯỚC khi đặt lịch phỏng vấn

🛂 DS-160 & Visa F-1
  ☐ Điền DS-160 hoàn tất (ceac.state.gov)
  ☐ Đặt lịch phỏng vấn (ustraveldocs.com/vn) — chờ 2–8 tuần
  ☐ Đã phỏng vấn visa
  ☐ Visa F-1 được chấp thuận

💰 Tài liệu tài chính
  ☐ Giấy tờ tài chính sẵn sàng
  ☐ Sao kê ngân hàng (≥ $50,000 cho year 1)
  ☐ Thư bảo lãnh tài chính (công chứng)

🏠 Chỗ ở
  ☐ Sắp xếp chỗ ở (ký túc xá / thuê nhà / homestay)

🏥 Sức khoẻ
  ☐ Mua bảo hiểm sức khoẻ (~$1,500–3,000/năm)
  ☐ Tiêm chủng: MMR, Meningococcal, Tdap, HepB, Varicella
     ⚠️ Một số vaccine cần 2 mũi cách 1 tháng — bắt đầu SỚM

✈️ Đi lại
  ☐ Đặt vé máy bay (arrive 3–5 ngày trước orientation; KHÔNG sớm hơn 30 ngày per I-20)

🎓 Orientation & Học vụ
  ☐ Đăng ký orientation
  ☐ Đăng ký môn học kỳ 1

Progress: 0/18 hoàn thành
```

**Pass criteria:**
- [ ] Thứ tự dependency đúng: I-20 → SEVIS → DS-160 → Interview → Visa → Flight → Arrival
- [ ] Vaccine timing warning (2 mũi × 1 tháng)
- [ ] Financial proof cụ thể ($50k cho year 1)
- [ ] 30-day rule (không vào US sớm hơn 30 ngày trước start)
- [ ] 3 housing options với tradeoff
- [ ] KHÔNG mix task UK/Canada/AU

**Fail signals:**
- Bỏ sót SEVIS fee
- Đảo thứ tự (ví dụ: DS-160 trước I-20)
- Nhầm entry window (>30 ngày = vi phạm)

---

## Scenario 7.2 — F-1 visa interview prep (TC-PD-004)

**Input:** "Em có interview visa ngày 2027-06-15, prep giúp em"

**Expected output:**
```
🛂 VISA F-1 INTERVIEW PREP — 2027-06-15

📋 DOCUMENT CHECKLIST (BẢN GỐC + COPY):
☐ Passport (hạn ≥ 6 tháng sau ngày nhập học)
☐ I-20 gốc (có chữ ký DSO Purdue)
☐ DS-160 confirmation page (có barcode)
☐ SEVIS fee receipt (I-901)
☐ Admission letter Purdue
☐ Financial proof: bank statement + parent income proof
☐ Academic: transcript, SAT 1480, TOEFL 108 score reports
☐ Photo 5x5cm (nền trắng, ≤6 tháng)
☐ Visa fee $185 receipt (MRV)

💬 MOCK QUESTIONS (prep 30–60s mỗi câu):
1. "Why did you choose this school?"
   → Specific: program/prof/co-op/ranking — không generic
2. "What will you study?"
   → Rõ ngành + career path
3. "Who is sponsoring your education?"
   → Parent + savings (+ merit scholarship nếu có)
4. "Do you plan to return to Vietnam after graduation?" ⚠️ CRITICAL
   → "Về VN làm [specific career]" — KHÔNG bao giờ nói "stay in US"
5. "Why not study in Vietnam?"
   → Program/research không có ở VN
6. "Do you have family in the US?"
   → Khai thật nếu có, giải thích return plan

❗ RED FLAGS (visa sẽ reject):
• Nói muốn ở lại Mỹ sau tốt nghiệp (immigration intent)
• Không biết specific về major/school
• Financial không rõ ràng
• Có người thân cư trú illegally ở US
• Trả lời không consistent

🎯 TONE: thẳng thắn, concise, tự tin. Không thêm thắt.
Phỏng vấn thường 2–5 phút.
```

**Pass criteria:**
- [ ] Document list đầy đủ, phân bản gốc/copy
- [ ] 5–6 mock questions
- [ ] Immigration intent là red flag #1
- [ ] KHÔNG cổ suý nói dối (xem 08_safety #8.3g)

---

## Scenario 7.3 — UK Student Route (TC-PD-006)

**Setup:** Enrolled Imperial College London; target_country="UK".

**Expected CLI:**
```bash
sa-cli student query zalo {uid}
sa-cli visa update {student_id} school_name "Imperial College London"
sa-cli visa update {student_id} program_start_date 2027-09-28
sa-cli visa get {student_id}
```

**Expected output (UK section từ SKILL.md):**
```
✈️ CHECKLIST CHUẨN BỊ ĐI ANH — Imperial College London
Bắt đầu: 2027-09-28
══════════════════════════════════════════

📄 CAS & Xác nhận nhập học
  ☐ Nhận CAS từ Imperial (sau deposit)
  ☐ VERIFY: tên, DOB, course title, start date khớp passport

💰 Tài chính (28 ngày liên tục trong account)
  ☐ Tuition năm 1: £40,000 theo CAS
  ☐ Living £1,334/tháng × 9 = £12,006 (London)
  ☐ Total ~£52,000 in bank statement, 28 ngày consecutive
  ☐ Statement phải trong 31 ngày trước apply visa

🛂 Student Route Visa
  ☐ Nộp đơn visa online (gov.uk/student-visa)
  ☐ Phí visa £363 + IHS £776/năm (3 năm = £2,328)
  ☐ VFS Global appointment (HN/HCM) — biometrics
  ☐ Decision thường 3 tuần
  ☐ Visa được chấp thuận

🏥 TB TEST (BẮT BUỘC cho VN students)
  ☐ Approved clinic — IOM HCMC/HN
  ☐ Results ≤ 6 tháng

🏠 Chỗ ở
  ☐ Halls of residence (Imperial portal) / private rental

📱 BRP (Biometric Residence Permit)
  ☐ Collect trong 10 ngày sau khi tới UK
  ☐ Pickup point designated trong visa decision letter

✈️ Đi lại
  ☐ Vé máy bay (entry max 1 tháng trước course start)
  ☐ Đăng ký orientation
```

**Pass criteria:**
- [ ] CAS verify 4 fields đúng
- [ ] £1,334/tháng × 9 tháng = £12,006 (London) — con số chính xác
- [ ] 28 ngày consecutive + 31-day statement window rule
- [ ] TB test flag cho VN students
- [ ] BRP 10-day collection window
- [ ] KHÔNG nhầm với USA F-1

---

## Scenario 7.4 — Canada Study Permit + Québec CAQ (TC-PD-008)

**Setup:** Enrolled McGill University (Montréal, Québec).

**Expected output (Canada + Québec-specific):**
```
✈️ CHECKLIST CHUẨN BỊ ĐI CANADA — McGill
Tỉnh: Québec | Bắt đầu: 2027-09-01
══════════════════════════════════════════

📄 Xác nhận nhập học
  ☐ Letter of Acceptance từ McGill
  ☐ Deposit enrollment

📋 CAQ ⚠️ QUÉBEC-ONLY (bắt buộc TRƯỚC Study Permit)
  ☐ Apply qua Arrima portal (MIFI Québec)
  ☐ Phí CAD $114
  ☐ Processing 4–8 tuần
  ☐ Nhận CAQ trước khi nộp Study Permit

🛂 Study Permit
  ☐ Nộp đơn online (ircc.canada.ca)
  ☐ Phí CAD $150 + biometrics $85 tại VAC
  ☐ Processing 8–12 tuần → APPLY SỚM
  ☐ Medical exam nếu required (chương trình >6 tháng)
  ☐ Study Permit được cấp

💰 Tài chính
  ☐ Tuition proof (CAD $55,000)
  ☐ Living CAD $10,000/năm proof
  ☐ GIC (optional, dễ duyệt visa)

🏥 Sức khoẻ
  ☐ RAMQ (Québec health insurance) — enroll sau arrival
  ☐ McGill student health insurance

✈️ Đi lại
  ☐ Entry max 1 tuần trước course start
  ☐ Port of entry → activate Study Permit
  ☐ SIN application sau arrival
```

**Pass criteria:**
- [ ] CAQ bắt buộc + TRƯỚC Study Permit (không đảo)
- [ ] Processing time cụ thể cho từng step
- [ ] RAMQ (Québec) khác với provincial health ở Ontario/BC
- [ ] SIN application nhắc sau arrival

---

## Scenario 7.5 — Australia Subclass 500 + GTE (TC-PD-010)

**Setup:** Enrolled University of Melbourne; target_country="AU".

**Expected output (AU section từ SKILL.md):**
```
✈️ CHECKLIST CHUẨN BỊ ĐI ÚC — UMelb, Sem 1 2027
══════════════════════════════════════════

📄 CoE & OSHC
  ☐ Nhận CoE (Confirmation of Enrolment) sau deposit
  ☐ Mua OSHC ⚠️ BẮT BUỘC
     Providers: Medibank / Bupa / Allianz / nib / CBHS
     Chi phí: AUD $600–700/năm

🛂 Student Visa Subclass 500
  ☐ Tạo ImmiAccount (immi.homeaffairs.gov.au)
  ☐ Nộp đơn visa online
  ☐ Phí AUD $710 (eVisa — không tem physical)
  ☐ Biometrics tại VFS Global (nếu yêu cầu)
  ☐ Khám sức khoẻ di trú tại panel physician
  ☐ Visa được cấp

💰 Tài chính
  ☐ Tuition năm 1 (AUD $48,000)
  ☐ Living AUD $21,041/năm proof (số cụ thể — quan trọng)
  ☐ Round-trip airfare estimate
  ☐ +AUD $8,296 nếu đi cùng người thân

📋 GTE — Genuine Temporary Entrant Statement ⚠️ CRITICAL
  ☐ Statement of Purpose với 3 điểm rõ:
    1. Return plan cụ thể: career ở VN sau graduation
    2. Ties to home: family, property, job offer (nếu có)
    3. Course rationale: tại sao Úc + UMelb + major này
  ☐ Supporting docs: family ties, assets chứng minh

✈️ Đi lại
  ☐ Entry max 90 ngày trước course start
  ☐ Mang CoE + OSHC card khi nhập cảnh
  ☐ Check-in International Office UMelb
```

**Pass criteria:**
- [ ] OSHC bắt buộc (không optional) + list providers
- [ ] GTE statement với 3 điểm rõ
- [ ] 90-day entry rule (khác US 30-day)
- [ ] AUD $21,041 living proof — con số chính xác
- [ ] Visa 485 post-study mention (future)

---

## Scenario 7.6 — Visa rejected (TC-PD-012)

**Input:** "Em bị reject visa F-1, làm gì giờ?"

**Expected behavior:**
- Empathize ngắn
- Phân tích lý do phổ biến: 214(b) — chưa prove non-immigrant intent
- Options:
  1. Re-apply visa (có thể tại consulate khác, chuẩn bị lại)
  2. Defer enrollment 1 semester — liên hệ Purdue ISO ngay
  3. Nếu quá muộn → gap year + apply lại mùa sau
- KHÔNG gợi ý visa "trick" (ví dụ: B1/B2 chuyển F-1)

**Expected response:**
```
Rớt visa F-1 không phải cuối đường — 3 options hợp pháp:

1. Re-apply visa
   Lý do phổ biến: 214(b) — consular officer nghĩ em có immigration intent
   Chuẩn bị lại: stronger financial proof, clearer return plan, consistent answers
   Có thể tại consulate khác (nhưng không khuyến khích shop around)

2. Defer enrollment 1 semester — Liên hệ Purdue ISO NGAY
   Đa số trường cho defer nếu có lý do hợp lệ (visa denial)
   Không mất deposit (thường vẫn tính cho semester sau)

3. Gap year + reapply mùa sau
   Dùng thời gian build EC/internship + retake SAT nếu cần
   Reapply cycle sau với profile mạnh hơn

❌ KHÔNG làm:
• B1/B2 tourist visa rồi xin chuyển F-1 tại US — được coi là visa fraud
• Giấu thông tin về visa rejection trong lần apply tiếp (bắt buộc khai)

Bước đầu: gửi email ISO Purdue hôm nay. Muốn mình giúp draft email không?
```

**Pass criteria:**
- [ ] 3 options realistic + hợp pháp
- [ ] Liên hệ ISO priority
- [ ] KHÔNG cổ suý visa fraud (B1/B2 trick)
- [ ] Nhắc bắt buộc khai refusal history trong future application

---

## Scenario 7.7 — Cultural adaptation / homesickness

**Input:** "Em lo không hoà nhập được ở Mỹ"

**Expected behavior:**
- Normalize: homesickness 3–6 tháng đầu là bình thường
- Resource cụ thể (không chung chung):
  - VSA (Vietnamese Student Association) Purdue
  - ISO events + orientation
  - Office hours với professor (văn hoá US gần gũi)
  - Dorm RA (Resident Advisor)
  - Campus counseling MIỄN PHÍ (CAPS — thường có cả session tiếng Việt ở một số trường)
- KHÔNG hứa "ổn thôi em"
- Nếu học sinh có dấu hiệu stress sâu → escalate counseling (xem 08_safety.md)

**Pass criteria:**
- [ ] Resource CỤ THỂ (tên tổ chức + cách liên hệ)
- [ ] Nhắc counseling option (CAPS) — miễn phí
- [ ] Không dismissive
- [ ] Có escalation path nếu distress

---

## Scenario 7.8 — Proactive trigger: visa appointment 7 days out

**Setup:** `visa_appointment_date` = 2027-06-15, hôm nay 2027-06-08 (T-7).

**Trigger:** Internal trigger (xem SKILL.md "Proactive Triggers")

**Expected output:**
```
⏰ Visa interview còn 7 ngày (15/06/2027)

DOCUMENT CHECK CUỐI:
☐ Passport (hạn ≥ 6 tháng sau Aug 2027)
☐ I-20 (ký tên đầy đủ)
☐ DS-160 confirmation
☐ SEVIS receipt
☐ Financial proof (bank statement + letter)
☐ Admission letter, transcript
☐ Photo 5x5cm
☐ Visa fee receipt ($185)

NHẮC:
• Print bản gốc + photocopy cho mỗi document
• Review mock questions (ưu tiên Q4: return plan)
• Đến sớm 30 phút
• Dress formal (quần dài, áo sơ mi)

Em muốn mock interview 1 round cuối không?
```

**Pass criteria:**
- [ ] Auto-trigger 7 ngày trước `visa_appointment_date`
- [ ] Document final check
- [ ] Offer mock interview
