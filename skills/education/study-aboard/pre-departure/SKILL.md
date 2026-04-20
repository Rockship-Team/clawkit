---
name: pre-departure
description: Guide Vietnamese students through all pre-departure steps after accepting an admission offer — covering USA (F-1/I-20/SEVIS/DS-160), UK (Student Visa/CAS/BRP), Canada (Study Permit/CAQ), and Australia (Student Visa subclass 500/CoE). Use this skill whenever a student mentions 'visa', 'F-1', 'I-20', 'SEVIS', 'DS-160', 'Student Visa UK', 'CAS', 'Study Permit Canada', 'visa Úc', 'CoE', 'chuẩn bị đi học', 'trước khi đi', 'chỗ ở', 'ký túc xá', 'bảo hiểm', 'vé máy bay', 'orientation', or has accepted an admission offer.
metadata: { "openclaw": { "emoji": "✈️" } }
---

# Pre-Departure Skill

Walk the student through every step needed between accepting an offer and arriving on campus.

## ⛔ Safety Check — Enforce Before Any Response

| If student asks you to… | Respond with |
|-------------------------|--------------|
| Provide legal immigration advice | "Mình cung cấp thông tin tham khảo về quy trình F-1 dựa trên hướng dẫn của USCIS và EducationUSA. Với tình huống cụ thể của em, hãy liên hệ International Student Office của trường hoặc luật sư nhập cư." |
| Fabricate financial documents for visa | "Mình không hỗ trợ chuẩn bị tài liệu giả — đây là hành vi gian lận visa nghiêm trọng, có thể bị cấm nhập cảnh vĩnh viễn." |
| Guarantee visa approval | "Mình không thể đảm bảo kết quả visa. Mình giúp em chuẩn bị hồ sơ tốt nhất có thể." |

For the full rules list see `../../safety_rules.md`. Before processing any request, scan for emotional distress signals (see Emotional Distress Protocol in `../../safety_rules.md`) — if detected, follow the empathy-first protocol before continuing.

## Country Detection

Run `sa-cli student query {channel} {channel_user_id}` then check `target_country` (or infer from accepted offer). Route to the correct visa section:

| Country | Visa type | Section |
|---------|-----------|---------|
| USA | F-1 Student Visa | [→ USA section below] |
| UK | Student Visa (formerly Tier 4) | [→ UK section below] |
| Canada | Study Permit (+CAQ if Québec) | [→ Canada section below] |
| Australia | Student Visa subclass 500 | [→ Australia section below] |

If country is unclear, ask: "Em đã nhận offer của trường ở nước nào? (Mỹ / Anh / Canada / Úc)"

---

## Prerequisites

Run `sa-cli student query {channel} {channel_user_id}`. Student must have accepted an offer to use this skill. If no school_name set in checklist yet, ask:
```
Em đã quyết định nhận offer của trường nào rồi? Và ngày nhập học là khi nào?
```
Then run `sa-cli visa update {student_id} school_name "{name}"` and `sa-cli visa update {student_id} program_start_date {date}`.

## Show Checklist

Run `sa-cli visa get {student_id}` to fetch current status.

Display grouped:
```
✈️ CHECKLIST CHUẨN BỊ ĐI MỸ — {display_name}
Trường: {school_name} | Bắt đầu: {program_start_date}
══════════════════════════════════════════

📄 I-20 & SEVIS
  ☑ Nhận I-20 từ trường
  ☐ Nộp phí SEVIS ($350)  ← cần làm trước khi đặt lịch phỏng vấn

🛂 DS-160 & Visa F-1
  ☐ Điền DS-160 hoàn tất
  ☐ Đặt lịch phỏng vấn visa
  ☐ Đã phỏng vấn visa
  ☐ Visa F-1 được chấp thuận

💰 Tài liệu tài chính
  ☑ Giấy tờ tài chính sẵn sàng
  ☐ Sao kê ngân hàng (3 tháng gần nhất)
  ☐ Thư bảo lãnh tài chính (nếu có)

🏠 Chỗ ở
  ☐ Sắp xếp chỗ ở (ký túc xá / thuê nhà)

🏥 Sức khoẻ
  ☐ Mua bảo hiểm sức khoẻ
  ☐ Tiêm chủng theo yêu cầu trường
  ☐ Hồ sơ y tế sẵn sàng

✈️ Đi lại
  ☐ Đặt vé máy bay

🎓 Orientation & Học vụ
  ☐ Đăng ký orientation
  ☐ Đăng ký môn học kỳ 1

{done}/{total} hoàn thành
```

When student completes an item → run `sa-cli visa update {student_id} {key} 1`

## Visa F-1 Step-by-Step Guide

When student asks about the visa process in detail:

### Bước 1 — Nhận I-20
```
I-20 là tài liệu trường cấp xác nhận em được nhận vào học và đủ điều kiện xin visa F-1.

📋 Trường sẽ gửi I-20 sau khi em:
1. Nộp enrollment deposit (thường $300–$500)
2. Nộp financial proof (sao kê ngân hàng chứng minh em có đủ tiền 1 năm học)
3. Hoàn thành immunization records

Thời gian: 1–4 tuần sau khi nộp đủ giấy tờ.
→ Kiểm tra portal của trường thường xuyên.
```

### Bước 2 — Nộp phí SEVIS
```
SEVIS (Student and Exchange Visitor Information System) là hệ thống theo dõi sinh viên quốc tế của Mỹ.

💰 Phí SEVIS: $350 (một lần, không hoàn trả)
→ Nộp tại: fmjfee.com
→ Cần: SEVIS ID từ I-20 của em (dạng N + 10 số)
→ Nộp TRƯỚC khi đặt lịch phỏng vấn visa

Lưu receipt — cần mang theo khi phỏng vấn.
```

### Bước 3 — Điền DS-160
```
DS-160 là đơn xin visa trực tuyến của Bộ Ngoại giao Mỹ.

🔗 Điền tại: ceac.state.gov
⏱️ Thời gian: 1–2 giờ
📌 Tips quan trọng:
• Trả lời trung thực — sai sót có thể dẫn đến từ chối visa
• Lưu Application ID thường xuyên (hết 20 phút không hoạt động sẽ mất)
• Photo: nền trắng, 5x5cm, chụp trong 6 tháng gần nhất
• Sau khi submit: in confirmation page (có barcode)
```

### Bước 4 — Đặt lịch phỏng vấn
```
Đặt lịch tại Đại sứ quán Mỹ tại Hà Nội hoặc Lãnh sự quán tại TP.HCM.

🔗 Đặt lịch tại: ustraveldocs.com/vn
⏳ Thời gian chờ hiện tại: thường 2–8 tuần (2025–2026)
   → Đặt sớm nhất có thể sau khi có I-20!

📋 Hồ sơ cần mang khi phỏng vấn:
• Hộ chiếu (còn hạn > 6 tháng sau ngày nhập học)
• I-20 gốc (có chữ ký DSO của trường)
• DS-160 confirmation page
• SEVIS fee receipt
• Ảnh thẻ (5x5cm, nền trắng)
• Sao kê ngân hàng / bằng chứng tài chính
• Thư nhập học (offer letter)
• Bằng tốt nghiệp / học bạ
• Thư giải trình mục đích học tập (nếu có)
```

### Bước 5 — Phỏng vấn visa
```
🎯 Tips phỏng vấn F-1:
• Trả lời ngắn gọn, tự tin, bằng tiếng Anh nếu được
• Câu hỏi thường gặp:
  - "Why did you choose this school?"
  - "What will you study?"
  - "Who is sponsoring your education?"
  - "Do you plan to return to Vietnam after graduation?"
  - "Do you have family or friends in the US?"
• Nhấn mạnh: em có kế hoạch học xong về nước (ties to home country)
• Đừng nói em muốn ở lại Mỹ sau khi tốt nghiệp

⏱️ Phỏng vấn thường chỉ 2–5 phút.
Nếu được chấp thuận: hộ chiếu giữ lại 3–5 ngày để dán visa.
```

## Housing Guidance

When student asks about housing:
```
🏠 CÁC LỰA CHỌN CHỖ Ở

1. KÝ TÚC XÁ (On-campus dorm) — khuyến khích năm 1
   ✅ Ưu: an toàn, tiện lợi, dễ kết bạn, không cần xe
   ⚠️ Hạn: thường tốn hơn, ít riêng tư, deadline đăng ký sớm
   → Đăng ký trên portal trường sớm nhất có thể (May–June)

2. THUÊ NHÀ (Off-campus apartment)
   ✅ Ưu: rẻ hơn (nếu ở ghép), riêng tư, linh hoạt
   ⚠️ Hạn: cần xe/phương tiện, hợp đồng 12 tháng, đặt cọc
   → Tìm qua: Zillow, Apartments.com, Facebook groups của sinh viên VN tại {city}

3. HOMESTAY
   ✅ Ưu: cải thiện tiếng Anh, có bữa ăn, an toàn
   ⚠️ Hạn: ít tự do, xa trường, tốn kém
   → Thông qua International Student Office

Em muốn mình tìm thêm thông tin cụ thể về housing của {school_name} không?
```

## Health Insurance & Vaccinations

```
🏥 BẢO HIỂM SỨC KHOẺ

Hầu hết trường bắt buộc mua bảo hiểm — thường tự động tính vào học phí (~$1,500–$3,000/năm).
Em có thể waive (từ chối) nếu có bảo hiểm tương đương từ nguồn khác.

💉 TIÊM CHỦNG THƯỜNG YÊU CẦU:
• MMR (Sởi, Quai bị, Rubella) — 2 mũi
• Meningococcal (Viêm màng não) — bắt buộc ở nhiều trường
• Tdap (Bạch hầu, Ho gà, Uốn ván)
• Hepatitis B — 3 mũi
• Varicella (Thuỷ đậu) — hoặc chứng minh đã mắc

📋 Kiểm tra yêu cầu cụ thể tại Student Health Center của {school_name}.
Nếu thiếu mũi nào: tiêm tại Việt Nam trước khi đi (rẻ hơn nhiều so với Mỹ).
```

## Proactive Triggers

- When visa_appointment_date is set and today ≥ (visa_appointment_date - 7 days): remind about documents to bring
- When visa_approved = 1 and flight_booked = 0: suggest booking flights
- When program_start_date is set and (program_start_date - today) ≤ 60 days and summary.pending > 5: show full checklist with urgency

---

## 🇬🇧 UK — Student Visa (Student Route)

### Show UK Checklist

```
✈️ CHECKLIST CHUẨN BỊ ĐI ANH — {display_name}
Trường: {school_name} | Bắt đầu: {program_start_date}
══════════════════════════════════════════

📄 CAS & Xác nhận nhập học
  ☐ Nhận CAS (Confirmation of Acceptance for Studies) từ trường
  ☐ Kiểm tra số CAS và course details chính xác

💰 Tài chính
  ☐ Chứng minh tài chính: đủ tiền học phí năm 1 + £1,334/tháng sinh hoạt (9 tháng)
  ☐ Tiền trong tài khoản ≥ 28 ngày liên tiếp trước khi nộp visa

🛂 Visa Student UK
  ☐ Nộp đơn visa online (gov.uk)
  ☐ Nộp phí visa: £363 + Immigration Health Surcharge £776/năm
  ☐ Đặt lịch appointment tại Visa Application Centre (Hà Nội / TP.HCM)
  ☐ Sinh trắc học (biometrics) đã hoàn thành
  ☐ Visa được chấp thuận

🏠 Chỗ ở & Sức khoẻ
  ☐ Sắp xếp chỗ ở (halls of residence / private rental)
  ☐ TB test (nếu trường yêu cầu — bắt buộc cho sinh viên từ Việt Nam)

✈️ Đi lại & Nhập học
  ☐ Đặt vé máy bay
  ☐ Đăng ký orientation
  ☐ Đăng ký môn học kỳ 1
  ☐ Nhận BRP (Biometric Residence Permit) sau khi tới UK — lấy tại bưu điện trong 10 ngày

{done}/{total} hoàn thành
```

### UK Visa Step-by-Step

#### Bước 1 — Nhận CAS
```
CAS (Confirmation of Acceptance for Studies) là mã 14 ký tự do trường cấp khi em đã đặt cọc.
Cần có CAS số TRƯỚC khi nộp đơn visa.

📋 Trường gửi CAS sau khi em:
1. Nộp enrollment deposit
2. Đáp ứng điều kiện nhập học (nếu có — ví dụ: kết quả tốt nghiệp)
3. Cung cấp ảnh hộ chiếu hợp lệ

Kiểm tra CAS: tên, ngày sinh, course title, start date — sai là bị từ chối visa.
```

#### Bước 2 — Chứng minh tài chính
```
💰 Yêu cầu tối thiểu (2025–2026):
• Học phí năm 1: theo CAS
• Sinh hoạt phí: £1,334/tháng × 9 tháng = £12,006 (nếu học ở London)
                  £1,023/tháng × 9 tháng = £9,207 (ngoài London)

Tiền phải có trong tài khoản ≥ 28 ngày liên tiếp (ngày nộp đơn tính ngày 0).
Chấp nhận: sao kê ngân hàng, thư xác nhận từ ngân hàng.
```

#### Bước 3 — Nộp đơn visa & Immigration Health Surcharge
```
🔗 Nộp đơn tại: gov.uk/student-visa
💰 Phí visa: £363 (nộp từ ngoài UK)
💰 IHS (Immigration Health Surcharge): £776/năm × số năm học
   → Cho phép dùng NHS (dịch vụ y tế công của UK) — rất đáng

Sau khi nộp đơn online → đặt lịch appointment tại VFS Global (Hà Nội / TP.HCM) để nộp tài liệu và lấy sinh trắc học.
```

#### Bước 4 — Nhận BRP sau khi tới UK
```
BRP (Biometric Residence Permit) = thẻ cư trú tạm thời — quan trọng như visa.

📮 Trường sẽ thông báo địa chỉ bưu điện để lấy BRP.
⏰ Phải lấy trong 10 ngày đầu sau khi nhập học. Quá hạn → phải nộp lại.

BRP cần cho: mở tài khoản ngân hàng, thuê nhà, đăng ký GP (bác sĩ gia đình).
```

---

## 🇨🇦 Canada — Study Permit

### Show Canada Checklist

```
✈️ CHECKLIST CHUẨN BỊ ĐI CANADA — {display_name}
Trường: {school_name} | Bắt đầu: {program_start_date}
══════════════════════════════════════════

📄 Xác nhận nhập học
  ☐ Nhận Letter of Acceptance (LOA) từ trường
  ☐ Nộp enrollment deposit

🛂 Study Permit
  ☐ Nộp đơn Study Permit online (ircc.canada.ca)
  ☐ Nộp phí: CAD $150
  ☐ Sinh trắc học (biometrics) tại VAC — CAD $85
  ☐ Study Permit được cấp

{if province == "Quebec":}
  📋 CAQ (Certificat d'acceptation du Québec)
    ☐ Nộp đơn CAQ trước khi nộp Study Permit
    ☐ Nhận CAQ — thêm ~1–2 tháng xử lý

💰 Tài chính
  ☐ Chứng minh tài chính: học phí năm 1 + CAD $10,000 sinh hoạt

🏥 Sức khoẻ
  ☐ Khám sức khoẻ di trú (nếu được yêu cầu — thường với chương trình > 6 tháng)

✈️ Đi lại & Nhập học
  ☐ Đặt vé máy bay
  ☐ Đăng ký orientation
  ☐ Đăng ký môn học

{done}/{total} hoàn thành
```

### Canada Study Permit Step-by-Step

#### Bước 1 — Nhận LOA và nộp đơn
```
Study Permit cho phép em học tại Canada. Xin trước khi nhập cảnh.

🔗 Nộp tại: ircc.canada.ca/en/immigration-refugees-citizenship/services/study-canada
💰 Phí: CAD $150 + biometrics CAD $85
⏱️ Thời gian xử lý: 4–12 tuần (vary theo quốc tịch — kiểm tra IRCC website)

Tài liệu cần thiết:
• Passport còn hạn
• Letter of Acceptance từ DLI (Designated Learning Institution)
• Bằng chứng tài chính (học phí + CAD $10,000 sinh hoạt)
• Statement of Purpose (giải thích lý do học, kế hoạch về nước)
• Ảnh hộ chiếu
```

#### Đặc biệt — Québec (CAQ)
```
Nếu trường ở Québec (Montréal, Québec City...):
Em cần xin CAQ (Certificat d'acceptation du Québec) TRƯỚC khi nộp Study Permit.

🔗 Nộp CAQ tại: immigration-quebec.gouv.qc.ca
💰 Phí: CAD $114
⏱️ Xử lý: 4–8 tuần

Sau khi có CAQ → nộp Study Permit liên bang như thông thường.
```

---

## 🇦🇺 Australia — Student Visa (Subclass 500)

### Show Australia Checklist

```
✈️ CHECKLIST CHUẨN BỊ ĐI ÚC — {display_name}
Trường: {school_name} | Bắt đầu: {program_start_date}
══════════════════════════════════════════

📄 CoE & Xác nhận nhập học
  ☐ Nhận CoE (Confirmation of Enrolment) từ trường sau khi đóng học phí
  ☐ Đăng ký OSHC (Overseas Student Health Cover) — bắt buộc

🛂 Student Visa 500
  ☐ Tạo tài khoản ImmiAccount (immi.homeaffairs.gov.au)
  ☐ Nộp đơn visa Student 500 online
  ☐ Nộp phí visa: AUD $710
  ☐ Sinh trắc học (biometrics) tại VFS Global nếu được yêu cầu
  ☐ Khám sức khoẻ di trú tại IOM hoặc panel physician được chỉ định
  ☐ Visa được cấp (eVisa — không có tem vật lý)

💰 Tài chính
  ☐ Chứng minh tài chính: học phí năm 1 + AUD $21,041 sinh hoạt + AUD $8,296 nếu có 1 người thân đi cùng

✈️ Đi lại & Nhập học
  ☐ Đặt vé máy bay (nhập cảnh sớm nhất 90 ngày trước ngày học)
  ☐ Đăng ký orientation
  ☐ Mang CoE và OSHC card khi nhập cảnh

{done}/{total} hoàn thành
```

### Australia Visa Step-by-Step

#### Bước 1 — Nhận CoE và mua OSHC
```
CoE (Confirmation of Enrolment) = xác nhận đã đăng ký học — do trường cấp sau khi em đóng học phí (một phần hoặc toàn bộ).

OSHC (Overseas Student Health Cover) là bảo hiểm y tế bắt buộc trong thời gian học tại Úc.
→ Thường mua qua trường hoặc trực tiếp với các nhà cung cấp được chấp thuận (Medibank, Bupa, Allianz, nib, CBHS).
💰 Chi phí: khoảng AUD $600–$700/năm.
```

#### Bước 2 — Nộp đơn visa Student 500
```
🔗 Nộp tại: immi.homeaffairs.gov.au (tạo ImmiAccount)
💰 Phí visa: AUD $710
⏱️ Thời gian xử lý: 4–6 tuần (có thể nhanh hơn nếu hồ sơ đầy đủ)

Tài liệu cần thiết:
• CoE từ trường
• OSHC certificate
• Passport còn hạn ≥ 6 tháng sau ngày học
• Bằng chứng tài chính
• Kết quả tiếng Anh (IELTS/TOEFL)
• Học bạ / bằng tốt nghiệp
• Statement of Purpose

Visa Úc là eVisa (điện tử) — không có tem trong passport. 
Kiểm tra tình trạng visa tại: border.gov.au/Visa-enquiries
```

#### GEO (Genuine Temporary Entrant)
```
Quan trọng: Visa Student Úc yêu cầu chứng minh "Genuine Temporary Entrant" — em thực sự có kế hoạch về nước sau khi học xong.

Trong Statement of Purpose, nên đề cập:
• Kế hoạch nghề nghiệp tại Việt Nam sau khi tốt nghiệp
• Mối quan hệ gia đình / tài sản / tài chính tại VN (ties to home)
• Lý do cụ thể chọn chương trình này tại trường này
```

---

## References

See `references/visa-interview-prep.md` for complete F-1 interview Q&A (30+ questions with coaching notes), document checklist, common refusal reasons and prevention, and key English phrases to memorize.

## Safety Rules

See `../../safety_rules.md`.
