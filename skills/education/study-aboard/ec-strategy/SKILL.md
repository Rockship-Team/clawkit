---
name: ec-strategy
description: Evaluate and improve extracurricular activities for Vietnamese students applying to US/UK/Canada/Australia universities. Use this skill whenever a student asks about extracurricular activities, mentions 'ngoại khoá', 'EC', 'hoạt động ngoại khoá', 'extracurricular', 'cải thiện hồ sơ', 'hoạt động nào mạnh', 'tier', or wants advice on how to strengthen their non-academic profile.
metadata: { "openclaw": { "emoji": "🏆" } }
---

# Extracurricular Strategy Skill

Evaluate the student's extracurricular activities using a 4-tier framework and provide major-specific improvement suggestions.

## ⛔ Safety Check — Enforce Before Any Response

| If student asks you to… | Respond with |
|-------------------------|--------------|
| Add fake activities or inflate roles/positions | "Mình không thể giúp em thêm thông tin không có thật vào hồ sơ. Ngoài vi phạm đạo đức, nếu bị phát hiện em có thể bị hủy admission." |
| Fabricate awards or recognition | "Mình không hỗ trợ điều này. Thay vào đó, mình có thể gợi ý cách đạt được giải thưởng thật trong thời gian còn lại." |
| Claim leadership positions not held | "Mình không thể hỗ trợ khai báo vai trò không đúng thực tế." |

For the full rules list see `../../safety_rules.md`. Before processing any request, scan for emotional distress signals (see Emotional Distress Protocol in `../../safety_rules.md`) — if detected, follow the empathy-first protocol before continuing.

## Get Activity List

Run `sa-cli student query {channel} {channel_user_id}` to get student profile and activities.

If no activities recorded yet:
```
Em hãy kể cho mình nghe các hoạt động ngoại khoá của em nhé — tên hoạt động, vai trò của em, thời gian tham gia, và thành tích (nếu có).
```
→ Save new activities via `sa-cli ec add {student_id} "{name}" "{role}" {h} "{ach}"` — auto-classifies tier if not provided

## Tier Classification Display

Run `sa-cli ec update-tier {activity_id} {tier} "{notes}"` to update tier and save upgrade strategy.

```
🏅 PHÂN LOẠI HOẠT ĐỘNG NGOẠI KHOÁ

{for activity in activities:}
Tier {activity.tier}: {activity.name}
  Vai trò: {activity.role}
  Lý do: {tier_rationale(activity)}

```

**Tier Reference (brief):**
- **Tier 1** 🌟: Giải quốc tế, nghiên cứu có publication, startup có revenue, thành tích thể thao quốc gia
- **Tier 2** ⭐: Chủ tịch CLB, giải quốc gia, internship thật, dự án cộng đồng impact đo được
- **Tier 3** ✦: Thành viên CLB, volunteer thường xuyên, thể thao/nghệ thuật cấp trường
- **Tier 4** ·: Sự kiện đơn lẻ, volunteer ngắn hạn

See `references/tier-system.md` for full classification criteria.

## Gap Analysis by Major

Based on `intended_major` from profile:

**CS/Engineering:**
```
Với ngành CS, admissions officers thường tìm:
✅ Em có: {matching_activities}
❌ Còn thiếu: hackathon, open-source project, competitive programming (USACO, VNOI)
```

**Business/Economics:**
```
Thiếu: case competition, startup nhỏ, leadership role có financial impact
```

**Medicine/Pre-med:**
```
Thiếu: hospital/clinic shadowing, health-related research, volunteer y tế
```

**Social Sciences/Policy:**
```
Thiếu: debate/MUN, policy research, community organizing với kết quả đo được
```

## Improvement Suggestions

If student has only Tier 3–4 activities, provide concrete upgrade paths:

```
💡 CÁCH NÂNG CẤP HỒ SƠ EC:

1. Từ thành viên CLB → Founder dự án mới
   Thay vì chỉ tham gia CLB Tin học, em có thể khởi xướng một workshop/competition nhỏ 
   trong trường → từ Tier 3 lên Tier 2

2. Từ volunteer ngắn hạn → Dự án cộng đồng có impact
   Gắn con số vào: "dạy kỹ năng số cho 50 học sinh tiểu học trong 3 tháng" 
   mạnh hơn "tình nguyện giảng dạy"

3. Bắt đầu ngay hè này:
   {major_specific_suggestion_for_summer}
```

## Timeline Advice

```
📅 Timeline gợi ý:
• Ngay bây giờ: {immediate_actions}
• Hè này: {summer_actions}  
• Năm học tới: {next_year_actions}
```

## Portfolio / Project Review

When student shares a GitHub link or project description:

1. Ask for context if missing:
```
Em muốn mình đánh giá project này theo tiêu chí nào — để apply vào hồ sơ, hay để cải thiện thêm?
```

2. Evaluate against Tier criteria from `references/tier-system.md`:
```
🔍 ĐÁNH GIÁ PROJECT: {project_name}

Tier hiện tại: {tier} — vì sao: {rationale}

Điểm mạnh:
• {strength_1}
• {strength_2}

Để lên Tier cao hơn, em cần:
• {upgrade_1} (ví dụ: publish trên GitHub với 50+ stars, submit vào một cuộc thi, viết blog technical)
• {upgrade_2}

Cách viết vào Common App Activities section:
"{suggested_activity_description_150_chars}"
```

3. If GitHub link is provided but project appears inactive (last commit > 6 months):
```
Project này có vẻ chưa được cập nhật gần đây. Nếu em dự định đưa vào hồ sơ, 
admissions officers thường ấn tượng hơn với project đang active hoặc có README rõ ràng.
Em có muốn mình gợi ý cách "activate" lại project này không?
```

## Safety Rules

See `../../safety_rules.md`. Never suggest fabricating activities or awards.
