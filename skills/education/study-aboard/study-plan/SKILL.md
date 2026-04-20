---
name: study-plan
description: Create and manage personalized SAT/TOEFL/IELTS study plans for Vietnamese students. Use this skill whenever a student asks about test preparation, mentions 'lộ trình SAT', 'ôn thi', 'kế hoạch học', 'study plan', 'SAT prep', 'TOEFL plan', 'luyện thi', 'điểm SAT', 'thi lại', or shares a practice test score. Also use when a weekly check-in cron job fires for a student with an active study plan. Also invoke when a student asks about GPA improvement, course selection, 'nên chọn môn gì', 'học AP hay IB', 'GPA đang giảm', or wants advice on which courses to take for their target major.
metadata: { "openclaw": { "emoji": "📚" } }
---

# Study Plan Skill

Create week-by-week standardized test preparation plans that adapt based on weekly practice scores.

## ⛔ Safety Check — Enforce Before Any Response

| If student asks you to… | Respond with |
|-------------------------|--------------|
| Guarantee a target score | "Mình không thể đảm bảo điểm số. Lộ trình này được thiết kế để tối đa hoá khả năng đạt mục tiêu, nhưng kết quả phụ thuộc vào cả quá trình luyện tập của em." |
| Provide fake practice scores to report to schools | "Mình không hỗ trợ khai báo điểm không trung thực." |

For the full rules list see `../../safety_rules.md`. Before processing any request, scan for emotional distress signals (see Emotional Distress Protocol in `../../safety_rules.md`) — if detected, follow the empathy-first protocol before continuing.

## Mode Detection

**Initial request**: Student asks for a new study plan
**Check-in mode**: Message contains a practice score or is triggered by weekly cron job
**GPA / Course mode**: Student asks about GPA improvement, course selection, or AP/IB choices

## Initial Plan Request

Run `sa-cli student query {channel} {channel_user_id}` to get student profile.

Ask if not already clear:
```
Em muốn ôn thi môn gì? (SAT / TOEFL / IELTS / ACT)
Mục tiêu điểm của em là bao nhiêu?
Em dự định thi ngày nào? (để mình tính số tuần còn lại)
Điểm hiện tại (nếu đã thi thử) là bao nhiêu?
```

Run `sa-cli plan create {student_id} {type} {target_score} {test_date} {current_score} {channel} {channel_user_id}` — saves plan and registers weekly check-in cron.

Display the generated plan:

```
📚 LỘ TRÌNH {TEST_TYPE} — Mục tiêu: {target_score}
Thời gian: {total_weeks} tuần (thi ngày {test_date})
──────────────────────────────────────────────

{for each week_group in plan_weeks:}
Tuần {week_range}: {focus_areas}
  → {daily_tasks}
  {if practice_test: "📝 Thi thử cuối tuần"}

CHECK-IN HÀNG TUẦN:
Mình sẽ hỏi thăm em mỗi Chủ nhật lúc 10h sáng. 
Hãy gửi điểm thi thử để mình điều chỉnh lộ trình nhé!
```

Weekly check-in cron job is registered automatically by create_plan.py.

## Weekly Check-in Mode

When triggered by cron or student shares a score:

```
Tuần này em học thế nào? 📊 Điểm thi thử gần nhất của em là bao nhiêu?
```

On receiving score → run `sa-cli plan checkin {plan_id} {score} "{notes}"`.

Display adjusted plan based on `adjustment.status` from `checkin.py`:
```
✅ Mình đã ghi nhận {score} điểm.

{if adjustment.status == "on_track":}
🎉 {adjustment.message}
Mình tăng độ khó một chút cho các tuần tới:

{if adjustment.status == "close":}
💪 {adjustment.message}
Lộ trình giữ nguyên nhịp — chỉ tăng cường phần yếu nhất:

{if adjustment.status == "behind":}
⚠️ {adjustment.message}
Mình điều chỉnh lộ trình để tập trung vào điểm yếu:

📅 CÁC TUẦN CÒN LẠI (đã cập nhật):
{adjusted remaining weeks}
```

## SAT/ACT Specifics

For SAT/ACT plans, see `references/sat-breakdown.md` for section-level breakdown (EBRW + Math), topic ladders, phase templates, and score gap logic.

## TOEFL/IELTS Specifics

For TOEFL/IELTS plans, see `references/toefl-breakdown.md` for the 4-skills (Reading, Listening, Speaking, Writing) breakdown format and weekly task templates.

## GPA Monitoring & Course Selection

When student asks about GPA or course choices — run `sa-cli student query {channel} {channel_user_id}` to get current GPA and curriculum type.

### GPA Trend Alert

If student shares a new GPA lower than what's on file:

```
Mình thấy GPA em đang giảm từ {previous_gpa} xuống {current_gpa}.

Nếu em đang apply trong 1–2 cycle tới, điều này cần chú ý vì:
- Mid-year report sẽ được gửi cho các trường đang xét hồ sơ
- GPA giảm đáng kể có thể dẫn đến rescind offer (dù hiếm)

Nguyên nhân chủ yếu thường là: môn học quá nặng, áp lực thi, hoặc cân bằng EC vs học tập.
Em muốn mình giúp gì — điều chỉnh lộ trình ôn thi để nhẹ hơn, hay tập trung ổn định GPA trước?
```

### Course Selection by Major

Recommend AP/IB courses based on `intended_major` from profile:

| Major | Recommended AP/IB | Avoid overloading |
|-------|-------------------|-------------------|
| CS / Engineering | AP Calc BC, AP CS A, AP Physics C | Don't drop core Math for electives |
| Business / Econ | AP Calc AB, AP Micro/Macro, AP Stats | Balance with humanities for holistic review |
| Pre-med / Biology | AP Biology, AP Chemistry, AP Calc AB | AP Physics recommended but not required |
| Social Sciences | AP US History, AP Lang, AP Gov, AP Econ | Depth > breadth |
| Math / Pure Science | AP Calc BC, AP Stats, AP Physics C | Olympiad prep > AP count |

**Script response format:**
```
Với ngành {intended_major}, các môn em nên ưu tiên:

✅ Quan trọng nhất: {top_courses}
⭐ Tốt nếu có: {secondary_courses}
⚠️ Lưu ý: Đừng lấy quá nhiều AP/IB cùng lúc — 
   4–5 AP tổng là đủ. Điểm GPA vẫn quan trọng hơn số lượng AP.

Trường mục tiêu của em ({dream_school}) thường thấy applicant với 
{typical_course_rigor} — em đang {on_track_or_gap}.
```

### Course Rigor Context

US admissions evaluates "course rigor" — choosing the hardest available courses at your school. If a student's school doesn't offer AP/IB, acknowledge this:

```
Nếu trường em không có AP/IB, đó không phải bất lợi — 
admissions officers đánh giá theo những gì có sẵn tại trường của em.
Điều quan trọng là em đang chọn những môn khó nhất mà trường cung cấp.
```

## Test Date Recommendation Based on Deadlines

When creating a plan (or when the student asks "nên thi ngày nào?"):

1. Run `sa-cli student query {channel} {channel_user_id}` to get student ID, then run `sa-cli application dashboard {student_id}` for the earliest EA/ED deadline in their school list:
   - If no school list yet: use November 1 as the default EA target.

2. Back-calculate the recommended test date:
   | Earliest EA/ED deadline | Latest recommended test date | Register by |
   |-------------------------|------------------------------|-------------|
   | Nov 1 (ED/EA Round 1)  | October SAT (mid-Oct)        | Mid-August  |
   | Nov 15                  | October SAT                  | Mid-August  |
   | Jan 1 (RD/EA II)       | December SAT                 | Late-Oct    |
   | Rolling admissions      | December SAT                 | Late-Oct    |

3. Display recommendation:
```
📅 KHI NÀO NÊN THI?

Deadline sớm nhất của em: {earliest_deadline} ({school_name} {app_type})

→ Em nên có điểm SAT trước {score_deadline_date}.
→ Kỳ thi phù hợp: {recommended_test_date}
→ Hạn đăng ký: {registration_deadline}

Nếu kỳ thi đó không khả dụng hoặc em chưa sẵn sàng, 
kỳ thi dự phòng gần nhất là: {backup_test_date}

⚠️ Đăng ký sớm — các phòng thi ở Hà Nội/TP.HCM thường đầy trước hạn 4–6 tuần.
```

For SAT/ACT test date calendar see `references/sat-test-dates.md`.

## Safety Rules

See `../../safety_rules.md`.
