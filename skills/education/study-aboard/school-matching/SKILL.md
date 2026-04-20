---
name: school-matching
description: Help Vietnamese students find and evaluate universities matching their profile across USA, UK, Canada, and Australia. Use this skill whenever a student asks about choosing schools, wants school recommendations, mentions 'chọn trường', 'gợi ý trường', 'danh sách trường', 'trường nào phù hợp', 'nên apply đâu', 'Reach Target Safety', 'UCAS', 'trường Anh', 'trường Canada', 'trường Úc', or asks about a specific university. Invoke proactively when the student has a complete profile but no school list yet.
metadata: { "openclaw": { "emoji": "🎓" } }
---

# School Matching Skill

Generate a personalized Reach / Target / Safety school list based on the student's academic profile, intended major, budget, and target countries.

## ⛔ Safety Check — Enforce Before Any Response

| If student asks you to… | Respond with |
|-------------------------|--------------|
| Guarantee admission to any school | "Mình không thể đảm bảo kết quả. Mình chỉ có thể đánh giá xác suất dựa trên dữ liệu lịch sử." |
| Provide false/inflated acceptance rate data | "Mình chỉ dùng dữ liệu từ nguồn chính thức. Mình sẽ không đưa ra con số không có căn cứ." |
| Suggest fabricating profile info to seem more competitive | "Mình không hỗ trợ khai báo thông tin sai trong hồ sơ apply." |

For the full rules list see `../../safety_rules.md`. Before processing any request, scan for emotional distress signals (see Emotional Distress Protocol in `../../safety_rules.md`) — if detected, follow the empathy-first protocol before continuing.

## Prerequisites

Run `sa-cli student query {channel} {channel_user_id}`. If no profile found, redirect:
```
Em chưa có hồ sơ với mình. Để mình gợi ý trường chính xác, cho mình biết một chút về em nhé — em đang học lớp mấy và GPA bao nhiêu?
```
(Then invoke profile-assessment skill.)

## Country System Awareness

Before generating school list, check `target_country` from student profile. If multiple countries or unclear, ask:

```
Em đang muốn apply nước nào? (có thể chọn nhiều)
🇺🇸 Mỹ  |  🇬🇧 Anh  |  🇨🇦 Canada  |  🇦🇺 Úc
```

Each country has a different application system — explain proactively:

### 🇺🇸 USA — Common App / Coalition
- Apply qua **Common App** hoặc **Coalition** (hoặc direct với một số trường)
- Không giới hạn số trường apply
- Hệ thống Reach / Target / Safety — chuẩn
- Có Early Decision (binding) và Early Action

### 🇬🇧 UK — UCAS
- Apply qua **UCAS** (ucas.com) — tối đa **5 trường** (hoặc 4 nếu có Oxford/Cambridge)
- Deadline UCAS: **15/10** cho Oxford/Cambridge/Medicine; **31/01** cho các trường còn lại (năm học kế tiếp)
- **Personal Statement**: 4,000 ký tự — về ngành học, không phải về cuộc sống cá nhân
- Không có Common App essay, không có extracurricular section riêng
- Trường offer dựa trên **predicted grades** (điểm dự đoán từ giáo viên) + Personal Statement
- Không có ED/EA — chỉ có một vòng duy nhất

### 🇨🇦 Canada — Direct / OUAC
- Ontario: qua **OUAC** (ouac.on.ca)
- BC, Alberta, các tỉnh khác: **direct application** qua website từng trường
- Không có nationwide platform như Common App
- Ít cạnh tranh hơn Mỹ với profile tương đương
- Deadline thường: **tháng 11–tháng 2** tuỳ trường và tỉnh

### 🇦🇺 Australia — Direct / UAC / VTAC
- NSW: **UAC** (uac.edu.au) | Victoria: **VTAC** (vtac.edu.au) | Còn lại: direct
- Rolling admissions cho nhiều trường
- Apply sớm thường có lợi (scholarship)
- Điểm IELTS tối thiểu thường khắt khe hơn Mỹ

---

## Generate School List

Run `sa-cli university match {student_id}` to score universities from the DB against the student's profile.

**If a school named by the student is not found in universities.db:**
```
Mình chưa có dữ liệu chi tiết về {school_name} trong hệ thống hiện tại (knowledge base đang có 201 trường).

Thông tin mình biết về trường này (từ training data): {general_info_if_available}

⚠️ Lưu ý: Dữ liệu này có thể chưa được cập nhật cho cycle {current_year}–{next_year}. 
Em nên xác nhận deadline và requirements trực tiếp trên website chính thức của trường.

Em vẫn muốn thêm {school_name} vào danh sách apply không? Mình có thể track deadline nếu em cung cấp thông tin thủ công.
```

**If knowledge base data for a matched school appears stale (last_updated > 12 months):**
```
⚠️ Dữ liệu {school_name} được cập nhật lần cuối {last_updated}. Vui lòng xác nhận deadline và học phí trực tiếp với trường trước khi apply.
```

Display result as — **format varies by country**:

### 🇺🇸 USA List

```
🎓 DANH SÁCH TRƯỜNG — 🇺🇸 MỸ
Mục tiêu: {intended_major} | Ngân sách: ${annual_budget_usd:,}/năm
══════════════════════════════════════════

🔴 REACH (cơ hội 15–25%)
┌─────────────────────────────────────────┐
│ {school_name}                           │
│ Acceptance rate (intl): {rate}%         │
│ SAT range: {sat_25}–{sat_75}           │
│ Chi phí: ~${cost:,}/năm                 │
│ Sau aid: ~${net_cost:,}/năm (ước tính)  │
│ Deadline EA: {ea_deadline} / RD: {rd_deadline} │
│ {if css_profile_required: "📋 Yêu cầu CSS Profile"} │
│ 💡 {fit_rationale}                      │
└─────────────────────────────────────────┘

🟡 TARGET (cơ hội 40–60%)
[same format]

🟢 SAFETY (cơ hội 70%+)
[same format]
```

---

### 🇬🇧 UK List

```
🎓 DANH SÁCH TRƯỜNG — 🇬🇧 ANH (UCAS)
Mục tiêu: {intended_major} | Ngân sách: ${annual_budget_usd:,}/năm
⚠️ UCAS: tối đa 5 trường — chọn kỹ hơn Mỹ!
══════════════════════════════════════════

🔴 AMBITIOUS
┌─────────────────────────────────────────────┐
│ {school_name}  {if russell_group: "Russell Group ⭐"} │
│ {city}                                       │
│ Yêu cầu: A-Level {a_level_requirement}       │
│          IB: {ib_requirement_points} điểm ({ib_requirement_hl} tại HL) │
│          {if a_level_requirement_subject: "Môn bắt buộc: {a_level_requirement_subject}"} │
│ UCAS deadline: {ucas_deadline_display}        │
│          {if oxbridge: "⚠️ Oxbridge: deadline 15/10 + interview bắt buộc"} │
│ IELTS tối thiểu: {ielts_minimum}             │
│ Học phí: £{tuition_local_currency:,}/năm     │
│ IHS (NHS): £{ihs_annual_gbp}/năm             │
│ Tổng ước tính: £{total_gbp:,}/năm (~${total_cost_usd_approx:,} USD) │
│ {if tb_test_required: "💉 TB test bắt buộc cho sinh viên từ VN"} │
│ CAS: ~{cas_processing_days} ngày sau đặt cọc │
│ BRP: lấy trong {brp_collection_days} ngày sau nhập học │
│ {if scholarship_available: "🎓 Học bổng có: {scholarship_names}"} │
│ 💡 {fit_rationale}                           │
└─────────────────────────────────────────────┘

🟡 BORDERLINE
[same format]

🟢 LIKELY OFFER
[same format]
```

---

### 🇨🇦 Canada List

```
🎓 DANH SÁCH TRƯỜNG — 🇨🇦 CANADA
Mục tiêu: {intended_major} | Ngân sách: ${annual_budget_usd:,}/năm
══════════════════════════════════════════

🔴 REACH
┌─────────────────────────────────────────────┐
│ {school_name}                               │
│ {city}, {province_fullname}                 │
│ GPA tối thiểu: {gpa_requirement_ca}/4.0     │
│   {gpa_requirement_ca_note}                 │
│ IELTS tối thiểu: {ielts_minimum}            │
│ Apply qua: {application_platform_ca_display} │
│ Deadline: {application_deadline_ca_display}  │
│ {if caq_required: "⚠️ Trường ở Québec — cần CAQ trước (~{caq_processing_weeks} tuần)"} │
│ Study Permit: ~{study_permit_processing_weeks} tuần xử lý │
│ {if biometrics_required: "💾 Biometrics bắt buộc tại VAC (CAD $85)"} │
│ Học phí: CAD ${tuition_local_currency:,}/năm │
│ Tổng ước tính: ~${total_cost_usd_approx:,} USD/năm │
│ {if co_op_available: "🔧 Có Co-op program"} │
│ {if pgwp_eligible: "📋 PGWP: ở lại làm việc {pgwp_duration_years} năm sau tốt nghiệp"} │
│ {if scholarship_available: "🎓 Học bổng có: {scholarship_names}"} │
│ 💡 {fit_rationale}                           │
└─────────────────────────────────────────────┘

🟡 TARGET
[same format]

🟢 SAFETY
[same format]
```

---

### 🇦🇺 Australia List

```
🎓 DANH SÁCH TRƯỜNG — 🇦🇺 ÚC
Mục tiêu: {intended_major} | Ngân sách: ${annual_budget_usd:,}/năm
══════════════════════════════════════════

🔴 REACH
┌─────────────────────────────────────────────┐
│ {school_name}  {if go8_member: "Go8 ⭐"}    │
│ {city}, {state}                             │
│ IELTS tối thiểu: {ielts_minimum_overall} overall, {ielts_minimum_band} mỗi band │
│   {ielts_note}                              │
│ Apply qua: {application_platform_au_display} │
│ Deadline: {application_deadline_au_display}  │
│ OSHC (bắt buộc): ~AUD ${oshc_approx_aud_per_year}/năm │
│   ({oshc_providers})                        │
│ CoE: ~{coe_processing_days} ngày sau đặt cọc │
│ Student Visa 500: ~{student_visa_500_processing_weeks} tuần xử lý │
│ {if visa_biometrics_required: "💾 Có thể yêu cầu biometrics tại VFS Global"} │
│ ⚠️ GTE: {gte_notes}                         │
│ Học phí: AUD ${tuition_local_currency:,}/năm │
│ Tổng ước tính: ~${total_cost_usd_approx:,} USD/năm │
│ {if graduate_visa_485_eligible: "📋 Visa 485: ở lại {graduate_visa_485_years} năm sau tốt nghiệp"} │
│ {if scholarship_available: "🎓 Học bổng có: {scholarship_names}"} │
│ 💡 {fit_rationale}                           │
└─────────────────────────────────────────────┘

🟡 TARGET
[same format]

🟢 SAFETY
[same format]
```

After displaying list, ask:
```
Em muốn thêm trường nào vào danh sách apply chính thức không? Mình sẽ theo dõi deadline và nhắc em đúng lúc.
```

If student confirms adding schools → run `sa-cli application add {student_id} {university_id} "{university_name}" {category} {ED|EA|RD|rolling} {deadline|-} {channel} {channel_user_id}` for each school (saves to DB and registers cron reminders automatically).

## Dream School Assessment

If student mentions a specific dream school:
```
Về {school_name}: acceptance rate cho international students là khoảng {rate}%. 
Với profile hiện tại của em ({gpa_value} GPA, {sat_score or "chưa có SAT"}), 
mình đánh giá đây là trường {category}.

Để tăng cơ hội:
{improvement_suggestions}
```

Be honest even if the assessment is unfavorable — false hope harms students.

## Budget Flagging

For schools where estimated net cost > annual_budget_usd:
```
⚠️ Lưu ý: {school_name} ước tính ~${net_cost:,}/năm sau aid — vượt ngân sách ${annual_budget_usd:,} của em. 
Vẫn có thể apply nếu em xin được scholarship thêm.
```

## UK Schools — UCAS-Specific Guidance

When target_country includes UK:

```
🇬🇧 LƯU Ý ĐẶC BIỆT — APPLY ANH (UCAS)

Em chỉ được chọn tối đa 5 trường. Mình gợi ý cân nhắc kỹ hơn so với apply Mỹ.

Về trường em đang xem xét:
• {school_name}: {ucas_tariff_range} — {acceptance_rate}%
  Yêu cầu A-Level: {a_level_requirement} | IB: {ib_requirement}
  Personal Statement: tập trung vào {relevant_topics_for_major}

⚠️ Oxford / Cambridge: deadline UCAS là 15/10 — sớm hơn 3,5 tháng so với các trường khác.
   Nếu apply Oxbridge: chỉ được chọn 1 trong 2 (Oxford hoặc Cambridge, không cả hai).
```

For UCAS school list, replace Reach/Target/Safety with likelihood based on predicted grades:
```
🟢 Likely offer (predicted grades đủ): [schools]
🟡 Borderline (predicted grades cận): [schools]
🔴 Ambitious (cần grades xuất sắc): [schools]
```

## Canada Schools — Province Awareness

When target_country includes Canada:

```
🇨🇦 LƯU Ý — APPLY CANADA

Em muốn học ở tỉnh nào? Hệ thống apply khác nhau:
• Ontario (Toronto, Waterloo, McMaster…) → OUAC
• BC (UBC, SFU, UVic…) → Direct
• Alberta (U of A, UCalgary…) → Direct
• Québec (McGill, Concordia…) → Direct + cần CAQ nếu được nhận

💡 Lợi thế so với Mỹ: với profile tương đương, khả năng nhận cao hơn và học phí thấp hơn.
Chi phí trung bình: CAD $25,000–$45,000/năm (including living expenses).
```

## Australia Schools — Ranking & IELTS

When target_country includes Australia:

```
🇦🇺 LƯU Ý — APPLY ÚC

Group of Eight (Go8) — top 8 universities:
Melbourne, ANU, Sydney, Queensland, UNSW, Monash, Adelaide, Western Australia

⚠️ IELTS tối thiểu thường cao hơn Mỹ: overall 6.5–7.0, không có band nào dưới 6.0.
   Em có kết quả IELTS chưa? Nếu chưa, mình nên lên kế hoạch thi trước khi apply.

Chi phí: AUD $30,000–$50,000/năm (tuition) + AUD $21,000 sinh hoạt.
```

---

## Hard Factors vs Soft Factors in fit_rationale

When generating `fit_rationale` for a school, consider both axes:

**Hard factors** (numeric, objective):
- GPA vs school's reported GPA range
- SAT/ACT vs school's 25th–75th percentile range
- TOEFL/IELTS vs school's minimum requirement
- Budget vs estimated net cost after aid

**Soft factors** (qualitative, holistic):
- EC strength and Tier distribution vs school's selectivity
- Major fit (does the school have strong programs in the student's intended major?)
- Essay potential (competitive schools place more weight on narrative)
- Recommendation quality (if on file)

The rationale should name the specific gap or strength — e.g., "SAT 1380 vs median 1450 → gap, nhưng EC mạnh bù đắp" — not a generic phrase.

## References

See `references/kb-schema.md` for the university data format used in display.

## Phase Transition — After School List Confirmed

Once the student confirms their school list (applications added via `sa-cli application add`), bridge to the EXECUTE phase.

**Next steps vary by country:**

**🇺🇸 USA:**
```
✅ Danh sách {n} trường 🇺🇸 đã được lưu! Mình sẽ theo dõi deadline tự động.

Bước tiếp theo:
1️⃣ Xem dashboard deadline (gõ "deadline")
2️⃣ Brainstorm Common App Personal Statement (gõ "essay")
3️⃣ Ôn thi SAT/TOEFL nếu chưa đạt target (gõ "lộ trình SAT")

Deadline gần nhất: {earliest_deadline} ({school_name}) — còn {days} ngày.
```

**🇬🇧 UK:**
```
✅ Danh sách {n} trường 🇬🇧 đã được lưu trên UCAS!

Bước tiếp theo:
1️⃣ Viết UCAS Personal Statement (gõ "personal statement") — khác hoàn toàn Common App
2️⃣ Xin predicted grades từ giáo viên — trường cần trước khi xét hồ sơ
3️⃣ {if any_oxbridge: "Ôn luyện phỏng vấn Oxbridge (gõ 'phỏng vấn Oxford/Cambridge')"}
4️⃣ {if tb_test_required: "Đặt lịch TB test — bắt buộc cho sinh viên từ VN"}

UCAS deadline: {earliest_ucas_deadline} — còn {days} ngày.
```

**🇨🇦 Canada:**
```
✅ Danh sách {n} trường 🇨🇦 đã được lưu!

Bước tiếp theo:
1️⃣ Chuẩn bị hồ sơ cho từng trường (mỗi trường có thể yêu cầu riêng)
2️⃣ {if any_quebec: "Nộp đơn CAQ ngay cho trường Québec (~{caq_weeks} tuần xử lý)"}
3️⃣ Ôn thi IELTS nếu chưa đạt minimum (gõ "lộ trình IELTS")

Deadline gần nhất: {earliest_deadline} ({school_name}) — còn {days} ngày.
```

**🇦🇺 Australia:**
```
✅ Danh sách {n} trường 🇦🇺 đã được lưu!

Bước tiếp theo:
1️⃣ Đảm bảo IELTS ≥ {max_ielts_requirement} overall — đây là điều kiện cứng
2️⃣ Chuẩn bị Statement of Purpose có yếu tố GTE (kế hoạch về nước sau tốt nghiệp)
3️⃣ Sau khi nhận offer: đặt cọc sớm để nhận CoE và apply Visa 500

Deadline gần nhất: {earliest_deadline} ({school_name}) — còn {days} ngày.
```

## Safety Rules

See `../../safety_rules.md`. Never guarantee admission results.
