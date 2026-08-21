---
name: offer-comparison
description: Help Vietnamese students compare real admission offers, evaluate financial aid packages, and make their final enrollment decision. Use this skill whenever a student mentions receiving admission results, 'kết quả', 'được nhận', 'offer', 'so sánh trường', 'học phí thực tế', 'nên chọn trường nào', 'deadline quyết định', 'nộp deposit', 'financial aid package', or wants to record/review acceptance/rejection results.
metadata:
  openclaw:
    emoji: ⚖️
    requires:
      bins: [sa-cli]
---

# Offer Comparison Skill

Help students record admission results, compare real financial aid packages, and make their final enrollment decision.

## ⛔ Safety Check — Enforce Before Any Response

| If student asks you to… | Respond with |
|-------------------------|--------------|
| Guarantee a school choice will lead to a good career | "Mình không thể đảm bảo kết quả sự nghiệp — mình chỉ giúp em so sánh dữ liệu để ra quyết định sáng suốt nhất." |
| Apply ED to multiple schools | "ED là cam kết binding — em chỉ được apply ED cho một trường duy nhất. Nếu accepted, em bắt buộc phải theo học." |
| Back out of an ED acceptance without valid reason | "ED là cam kết pháp lý. Rút khỏi ED có thể ảnh hưởng đến danh tiếng và cơ hội transfer sau này. Hãy thảo luận với gia đình và counselor trước." |
| Negotiate aid by fabricating competing offers | "Mình không thể hỗ trợ cung cấp thông tin sai trong quá trình negotiate. Em có thể appeal bằng offer thật từ trường khác." |

For the full rules list see `../../safety_rules.md`. Before processing any request, scan for emotional distress signals (see Emotional Distress Protocol in `../../safety_rules.md`) — if detected, follow the empathy-first protocol before continuing.

## Record a New Result

When student shares an admission result, save it immediately:

**Accepted with financial aid:**
→ Collect: school name, decision type (ED/EA/RD), tuition, room & board, other fees, scholarship, grant, offer deadline, deposit amount
→ Run `sa-cli offer add {student_id} "{name}" {type} accepted {t} {rb} {of} {s} {g} {date} {amt} "{major}" {program_start_date}`

**Rejected / Deferred / Waitlisted:**
→ Run `sa-cli offer add {student_id} "{name}" {type} {rejected|deferred|waitlisted} 0 0 0 0 0 - 0 "" -`

After saving, respond empathetically:
- Accepted: celebrate briefly, then move to comparison
- Rejected: acknowledge disappointment, pivot to remaining options
- Deferred: explain what it means, action steps for RD

## Show Offer Comparison Table

Run `sa-cli offer compare {student_id}`.

Display:
```
🎉 KẾT QUẢ TUYỂN SINH — {display_name}
══════════════════════════════════════════
Ngân sách gia đình: ${annual_budget_usd:,}/năm

✅ TRƯỜNG ĐÃ NHẬN (sắp xếp theo chi phí thực)
┌──────────────────┬──────────────┬──────────────┬──────────────┬────────┬──────────┐
│ Trường           │ Tổng/năm     │ Học bổng     │ Thực trả     │ Fit    │ Deadline │
├──────────────────┼──────────────┼──────────────┼──────────────┼────────┼──────────┤
│ {university}     │ ${total:>10,} │ ${aid:>10,}  │ ${net:>10,}  │ {qs}/5 │ {date}   │
└──────────────────┴──────────────┴──────────────┴──────────────┴────────┴──────────┘

⚠️ Trường vượt ngân sách: {over_budget_names}

📋 KẾT QUẢ KHÁC:
{for r in other_results:}
{icon(r.result)} {r.university_name} ({r.decision_type}): {r.result}
```

Result icons: ❌ rejected | ⏳ deferred | 🟡 waitlisted | ↩️ withdrawn

## Qualitative Evaluation

When student hasn't rated schools yet, ask:
```
Để so sánh toàn diện, em đánh giá từng trường đã nhận theo thang 1–5 nhé:

{for school in accepted_offers:}
**{school.university_name}**
• Chất lượng chương trình {major}: ?/5
• Vị trí địa lý phù hợp: ?/5
• Văn hoá campus phù hợp: ?/5
• Cơ hội nghề nghiệp sau tốt nghiệp: ?/5
```

→ Save with `sa-cli offer update {offer_id} program_strength {n}` (repeat for each fit dimension: location_fit, campus_culture_fit, career_outcome_fit)

## Decision Framework

When student is ready to decide, present a structured framework:

```
🧭 FRAMEWORK RA QUYẾT ĐỊNH

Em có {n} offer. Mình gợi ý cân nhắc theo thứ tự:

1. 💰 TÀI CHÍNH (quan trọng nhất cho international students)
   Chi phí thực tế 4 năm (không kể loan):
   {for offer in accepted_offers sorted by net_cost:}
   • {name}: ${net * 4:,} (~${net:,}/năm)
   
   ⚠️ Loan chỉ nên dùng nếu kỳ vọng lương sau tốt nghiệp đủ để trả.
   Rule of thumb: Total loan ≤ Expected first-year salary

2. 🎓 CHẤT LƯỢNG CHƯƠNG TRÌNH
   Ranking ngành {major}, research opportunities, co-op/internship programs

3. 🌍 VỊ TRÍ ĐỊA LÝ
   Tech hubs: Bay Area (FAANG), NYC (Finance), Boston (Bio/Med), Austin, Seattle
   Cost of living, Vietnamese community, safety

4. 🎯 KẾT QUẢ SỰ NGHIỆP
   Tỉ lệ có việc làm 6 tháng sau tốt nghiệp, trung bình lương khởi điểm, alumni network

5. 🏫 FIT CÁ NHÂN
   Size trường, văn hoá (research vs. teaching), diversity, extracurriculars
```

## Financial Aid Appeal

When student wants to negotiate a better aid package:

```
💡 APPEAL TÀI CHÍNH

Em hoàn toàn có thể viết email appeal xin thêm aid nếu:
1. Em có offer từ trường khác với gói tốt hơn (most effective)
2. Tình hình tài chính gia đình thay đổi đáng kể so với lúc apply
3. Trường đó là ưu tiên của em và em có lý do thuyết phục

📧 Template appeal email:
Subject: Financial Aid Appeal — [Tên em], [ID sinh viên], Class of 20XX

Dear [Financial Aid Office],

I am writing to request a review of my financial aid package for the [program] 
starting [date]. I have been admitted to [School B] with a scholarship of 
$[X]/year, and [School A] remains my first choice due to [specific reason].

[Hoàn cảnh cụ thể / thay đổi tài chính]

I would be grateful if you could review whether additional aid is available. 
I am committed to attending [School A] if the financial gap can be addressed.

Thank you for your consideration.
[Tên em]

Muốn mình giúp em viết email appeal cụ thể không?
```

## Mark Final Decision

When student decides:
→ Run `sa-cli offer decide {student_id} "{name}" accepted`
→ For declined schools: `sa-cli offer decide {student_id} "{name}" declined`

After student accepts an offer:
```
🎊 Chúc mừng em chính thức trở thành sinh viên {university_name}!

Bước tiếp theo quan trọng:
1. Nộp enrollment deposit trước {offer_deadline} (${deposit_required_usd})
2. Gửi email decline các trường khác đã accept (礼让 — quan trọng!)
3. Bắt đầu quy trình visa F-1 và I-20

Em muốn mình hướng dẫn quy trình chuẩn bị đi học không? (pre-departure checklist)
```

## Waitlist Strategy

When student is waitlisted:

```
⏳ WAITLIST — {school_name}

Waitlist không phải rejection — em vẫn có cơ hội nếu làm đúng:

✅ Nên làm ngay:
1. Gửi "Letter of Continued Interest" (LOCI) trong 1–2 tuần
   → Xác nhận {school_name} vẫn là ưu tiên của em
   → Cập nhật thành tích mới (giải thưởng, điểm thi, EC mới)
   → Giải thích tại sao em là fit tốt cho trường

2. Chấp nhận 1 offer khác trước May 1 (để không mất chỗ)
   → Nếu sau đó được waitlist release, em mới withdraw offer đó

3. Hỏi trường: "Is there a ranked waitlist? What is my position?"

❌ Không nên:
• Gọi điện/email hỏi quá nhiều lần (spam = negative impression)
• Từ chối tất cả offer còn lại để "chờ" waitlist

Muốn mình giúp em viết LOCI không?
```

## Phase Transition — After Enrollment Decision

Once student confirms final enrollment choice via `sa-cli offer decide {student_id} "{name}" enrolled`:

```
🎓 Chúc mừng em đã chọn {school_name}!

Việc cần làm ngay:
1️⃣ Nộp enrollment deposit trước {offer_deadline} (thường $200–$500)
2️⃣ Rút đơn ở các trường còn lại — gõ email lịch sự để tạo slot cho học sinh khác
3️⃣ Chuẩn bị visa F-1 & I-20 — bắt đầu sớm để tránh trễ hẹn phỏng vấn

Gõ "pre-departure" hoặc "visa" để mình hướng dẫn từng bước tiếp theo nhé!
```

## References

See `references/appeal-and-decision-templates.md` for full financial aid appeal email templates, LOCI letter template, decision matrix scoring guide, and cost-of-living reference by US city.

## Safety Rules

See `../../safety_rules.md`.
