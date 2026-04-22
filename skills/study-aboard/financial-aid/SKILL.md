---
name: financial-aid
description: Help Vietnamese students and families understand university costs, financial aid, and scholarships for studying abroad in USA, UK, Canada, and Australia. Use this skill whenever a student asks about costs, tuition, financial aid, scholarships, mentions 'học phí', 'financial aid', 'chi phí', 'cost', 'học bổng', 'scholarship', 'merit aid', 'CSS Profile', 'FAFSA', 'budget', 'Chevening', 'học bổng Anh', 'học bổng Canada', 'học bổng Úc', or wants to compare costs across schools. Also handles Early Decision (ED) strategy questions with required confirmation.
metadata:
  openclaw:
    emoji: 💵
    requires:
      bins: [sa-cli]
---

# Financial Aid Advisor Skill

Help students and families understand the true cost of education after financial aid, and navigate aid application processes.

## ⛔ Safety Check — Enforce Before Any Response

| If student asks you to… | Respond with |
|-------------------------|--------------|
| Guarantee a scholarship amount | "Mình chỉ cung cấp ước tính dựa trên dữ liệu. Con số thực tế phụ thuộc vào profile em và quyết định của từng trường." |
| Provide false financial information on FAFSA/CSS Profile | "Mình không hỗ trợ khai báo thông tin tài chính không đúng — đây là hành vi gian lận liên bang có thể dẫn đến hủy aid và bị truy tố." |
| Claim ED decision without real intent | "Early Decision là cam kết ràng buộc. Mình không thể giúp em apply ED nếu không có ý định thực sự theo học." |

For the full rules list see `../../safety_rules.md`. Before processing any request, scan for emotional distress signals (see Emotional Distress Protocol in `../../safety_rules.md`) — if detected, follow the empathy-first protocol before continuing.

## Cost Comparison Table

Run `sa-cli student query {channel} {channel_user_id}` to get student profile, then run `sa-cli financial cost-compare {student_id}` — reads applications and university cost data from the DB.

```
💰 SO SÁNH CHI PHÍ — SAU FINANCIAL AID
Ngân sách gia đình: ${annual_budget_usd:,}/năm
══════════════════════════════════════════

┌──────────────────┬──────────────┬──────────┬──────────────┬────────┐
│ Trường           │ Tổng/năm     │ Aid ước tính│ Thực trả  │ Status │
├──────────────────┼──────────────┼──────────┼──────────────┼────────┤
{for school in cost_comparison:}
│ {school.name:<16} │ ${school.total_cost:>10,} │ ${school.aid:>8,} │ ${school.net_cost:>10,} │ {budget_flag(school)} │
└──────────────────┴──────────────┴──────────┴──────────────┴────────┘

{budget_flag legend: ✅ within budget | ⚠️ over budget}

⚠️ Lưu ý 2025–2026: Một số trường đang cắt giảm financial aid cho international students.
Xem thêm: references/aid-guide.md
```

## CSS Profile Guidance

For schools requiring CSS Profile:
```
📋 {school_name} yêu cầu CSS Profile (ngoài FAFSA).

CSS Profile là gì? Biểu mẫu tài chính chi tiết hơn FAFSA, do College Board quản lý.
Deadline CSS Profile: thường sớm hơn deadline apply 2–4 tuần.
Cần làm: Lên collegeboard.org/css-profile, điền thông tin tài chính gia đình.
Chi phí nộp: ~$25 cho trường đầu tiên.

Em có muốn mình nhắc deadline CSS Profile không?
```

## Cost Reference by Country

When student asks about costs for a specific country, use these benchmarks:

### 🇺🇸 USA
| Loại trường | Học phí/năm | Sinh hoạt | Tổng ước tính |
|---|---|---|---|
| Private elite (MIT, Harvard…) | $58,000–$62,000 | $18,000–$22,000 | $76,000–$84,000 |
| Public flagship (UMich, UCLA…) | $40,000–$58,000 | $16,000–$20,000 | $56,000–$78,000 |
| State university (Purdue, ASU…) | $28,000–$45,000 | $12,000–$16,000 | $40,000–$61,000 |

⚠️ Xu hướng 2025–2026: Một số trường Mỹ đang cắt giảm need-based aid cho international students. Merit scholarship vẫn ổn định hơn.

### 🇬🇧 UK
| Loại trường | Học phí/năm | Sinh hoạt (London) | Sinh hoạt (ngoài London) |
|---|---|---|---|
| Russell Group (Oxford, Imperial…) | £25,000–£40,000 | £15,000–£18,000 | £12,000–£14,000 |
| Other UK universities | £15,000–£25,000 | £15,000–£18,000 | £10,000–£13,000 |

💡 Lợi thế: chương trình đại học chỉ 3 năm (tiết kiệm 1 năm so với Mỹ). Master 1 năm.
💰 IHS (bảo hiểm y tế NHS): £776/năm — bắt buộc, tính trước khi so sánh.

### 🇨🇦 Canada
| Loại trường | Học phí/năm | Sinh hoạt | Tổng ước tính |
|---|---|---|---|
| Top universities (UofT, UBC, McGill…) | CAD $35,000–$55,000 | CAD $15,000–$20,000 | CAD $50,000–$75,000 |
| Other universities | CAD $20,000–$35,000 | CAD $12,000–$16,000 | CAD $32,000–$51,000 |

💡 Quy đổi: 1 CAD ≈ 0.73 USD. Chi phí thực thấp hơn Mỹ đáng kể.
💡 Post-Graduation Work Permit (PGWP): sau khi tốt nghiệp có thể ở lại làm việc 1–3 năm — lợi thế lớn cho international students.

### 🇦🇺 Australia
| Loại trường | Học phí/năm | Sinh hoạt | Tổng ước tính |
|---|---|---|---|
| Go8 (Melbourne, ANU, Sydney…) | AUD $35,000–$50,000 | AUD $21,000–$25,000 | AUD $56,000–$75,000 |
| Other universities | AUD $25,000–$38,000 | AUD $18,000–$22,000 | AUD $43,000–$60,000 |

💡 Quy đổi: 1 AUD ≈ 0.63 USD. 
💡 Temporary Graduate visa (subclass 485): sau tốt nghiệp có thể ở lại 2–4 năm tuỳ trình độ.

---

## Scholarship Information

Provide scholarship suggestions based on profile. Common sources for Vietnamese international students:

### 🇺🇸 USA
- Merit scholarships từ individual universities (check `financial_aid_international` field)
- EducationUSA Opportunity Funds
- Vietnam Education Foundation
- University-specific: see `references/aid-guide.md`

### 🇬🇧 UK
- **Chevening Scholarship** — học bổng toàn phần của Chính phủ Anh (1 năm Master). Yêu cầu: 2 năm kinh nghiệm làm việc → không phải cho học sinh lớp 12, nhưng nên biết để plan sau đại học.
- **Commonwealth Scholarship** — cho công dân các nước Commonwealth (VN không eligible, nhưng nhiều học sinh biết lầm).
- **University-specific scholarships**: nhiều Russell Group university có merit scholarship cho international undergrad — thường 20–50% học phí.
  - University of Edinburgh: Global Scholarships
  - University of Glasgow: Global Excellence Scholarship
  - University of Birmingham: International Achievement Scholarship
- **GREAT Scholarship** — học bổng của British Council + universities (~£10,000).

⚠️ UK scholarship cho undergraduate international students ít hơn US. Hầu hết là partial (không toàn phần).

### 🇨🇦 Canada
- **Vanier Canada Graduate Scholarships** — cho bậc PhD (không áp dụng undergraduate).
- **University-specific entrance scholarships**: nhiều trường tự động xét khi apply
  - University of Toronto: Lester B. Pearson International Scholarship (toàn phần — cực kỳ cạnh tranh)
  - UBC: International Major Entrance Scholarship (lên đến CAD $80,000 / 4 năm)
  - McGill University: Entrance Scholarship
  - University of Waterloo: International Student Merit Scholarship
- **Province scholarships**: Alberta Scholarship Program, Ontario Graduate Scholarship (graduate level)

💡 Canada có nhiều entrance scholarship tự động — không cần apply riêng nếu GPA và EC đủ mạnh.

### 🇦🇺 Australia
- **Australia Awards** — học bổng toàn phần của Chính phủ Úc cho sinh viên sau đại học từ VN. Apply qua dfat.gov.au.
- **Endeavour Scholarships** — đã kết thúc (thay bởi Australia Awards).
- **University-specific**:
  - University of Melbourne: Melbourne International Undergraduate Scholarship (lên đến AUD $10,000/năm)
  - ANU: ANU Chancellor's International Scholarship (50% học phí)
  - UNSW: UNSW International Scholarship (lên đến AUD $10,000)
  - Monash University: Monash International Merit Scholarship (lên đến AUD $10,000/năm)
- **Research Training Program (RTP)** — cho bậc PhD.

💡 Australia Awards cho undergraduate rất hiếm — chủ yếu graduate/postgraduate. Scholarship tốt nhất là university-specific merit.

## Early Decision (ED) Confirmation — REQUIRED

If student asks about ED strategy:

**Step 1: Show confirmation prompt (ALWAYS — do not skip)**
```
⚠️ Early Decision (ED) là cam kết ràng buộc (binding commitment).

Nếu em apply ED và được nhận:
✓ Em PHẢI theo học trường đó
✓ Em phải rút tất cả đơn ở các trường khác
✓ Không thể từ chối trừ khi financial aid không đủ

Em và gia đình đã hiểu rõ và đồng ý với cam kết này chưa? 
Gõ "Đồng ý — tôi hiểu ED là binding" để tiếp tục.
```

**Step 2: Only proceed with ED advice after confirmation received.**

## Safety Rules

See `../../safety_rules.md`. Never guarantee scholarship amounts. Always remind that aid estimates are approximate.
