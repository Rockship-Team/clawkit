---
name: weekly-checkin
description: Proactively check in with Vietnamese students every week to review overall study abroad progress — SAT/TOEFL scores, essay drafts, EC updates, upcoming deadlines, and next priority actions. Use this skill whenever a weekly cron job fires for a student, or when a student asks for a general progress update, mentions 'tuần này', 'cập nhật tiến độ', 'tổng hợp', 'nhìn lại tuần này', 'mình đang ở đâu rồi', 'còn phải làm gì', or 'check-in'.
metadata:
  openclaw:
    emoji: 📆
    requires:
      bins: [sa-cli]
---

# Weekly Check-in Skill

Proactively review all active workstreams and surface the top 1–3 priority actions for the week. This skill is the primary driver of the bot's "chủ động push" behaviour — do not wait for the student to ask.

## ⛔ Safety Check — Enforce Before Any Response

| If student asks you to… | Respond with |
|-------------------------|--------------|
| Guarantee admission outcome | "Mình không thể đảm bảo kết quả. Check-in này giúp em không bỏ lỡ bước nào quan trọng." |
| Skip a week's check-in permanently | "Mình có thể tạm dừng nhắc nhở — nhưng thật sự khuyến khích em duy trì nhịp hàng tuần vì giai đoạn apply rất quan trọng." |

For the full rules list see `../../safety_rules.md`. Before processing any request, scan for emotional distress signals (see Emotional Distress Protocol in `../../safety_rules.md`) — if detected, follow the empathy-first protocol **before** showing any progress data.

---

## Trigger Modes

| Mode | Condition |
|------|-----------|
| **Cron (auto)** | Weekly cron job fires on Sunday 10:00 AM student's timezone |
| **On-demand** | Student explicitly asks for a progress summary or check-in |

---

## Data Collection

Run the following commands in sequence to build a full picture before composing the response:

1. `sa-cli student query {channel} {channel_user_id}` → student profile + timezone
2. `sa-cli application dashboard {student_id}` → application list with essay status & urgency flags
3. `sa-cli plan list {student_id}` → active study plans with latest check-in score
4. `sa-cli essay list {student_id}` → essay drafts with current round and last-updated date
5. `sa-cli ec list {student_id}` → EC activities with tier, status, and last-updated date
6. `sa-cli checklist get {student_id}` → student-wide checklist (rec letters, scores, FAFSA/CSS)

---

## Weekly Summary Format

```
📆 CHECK-IN TUẦN — {display_name}
{current_date} | Còn {days_to_earliest_deadline} ngày đến deadline gần nhất ({earliest_deadline_school})
══════════════════════════════════════════

📚 ÔN THI
{for plan in active_plans:}
  {plan.type}: {plan.current_score or "chưa cập nhật"} → mục tiêu {plan.target_score}
  Tuần này: {plan.current_week_tasks_summary}
  {if plan.last_checkin_score is None: "⚠️ Em chưa báo điểm tuần này — thi thử xong gửi mình nhé!"}

✍️ ESSAY
{for essay in active_essays:}
  {essay.school_name or "Common App"} — {essay.essay_type}: {essay.status} (vòng {essay.round})
  Cập nhật gần nhất: {days_since_update} ngày trước
  {if days_since_update > 7: "⚠️ Chưa cập nhật 1 tuần — tuần này cố gắng sửa thêm 1 vòng nhé"}

📅 DEADLINE SẮP TỚI
{for app in urgent_apps (days_until_deadline ≤ 30):}
  {urgency_icon} {app.university_name} {app.application_type}: còn {app.days_until_deadline} ngày
    Essay: {essay_status_icon} {app.essay_status}

📋 CHECKLIST CHUNG
  Tổng: {done}/{total} hoàn thành
  {if pending_rec_letters > 0: "⚠️ Còn {pending_rec_letters} thư giới thiệu chưa submit"}
  {if css_profile == 0 and has_css_school: "⚠️ CSS Profile chưa điền"}
  {if fafsa == 0 and has_fafsa_school: "⚠️ FAFSA chưa điền"}

🏆 HOẠT ĐỘNG NGOẠI KHOÁ
{for ec in active_ecs:}
  {ec.name} ({ec.tier}): {ec.status}
  {if ec.days_since_update > 30: "⚠️ Chưa cập nhật 30 ngày"}
```

---

## Priority Action Block

After the summary, always close with a focused action list (max 3 items, ordered by urgency):

```
🎯 ƯU TIÊN TUẦN NÀY:
1. {highest_priority_action}
2. {second_priority_action}
3. {third_priority_action}

Em muốn bắt đầu từ đâu? Mình sẵn sàng hỗ trợ ngay!
```

**Priority ranking logic (apply in order):**
1. Any deadline ≤ 7 days → CRITICAL — put at top
2. Any essay `not_started` with deadline ≤ 30 days → HIGH
3. Study plan score not reported this week → MEDIUM
4. Checklist items: rec letters pending, CSS/FAFSA undone → MEDIUM
5. EC not updated > 30 days with upcoming interview/competition → LOW
6. Essay `draft` not updated > 7 days → LOW

---

## No Active Plan or No Applications Yet

If `active_plans` is empty **and** `applications` is empty:

```
Mình thấy em chưa bắt đầu giai đoạn apply chính thức.

Các bước nên làm ngay:
1. Đánh giá hồ sơ hiện tại → gõ "đánh giá hồ sơ"
2. Chọn trường mục tiêu → gõ "gợi ý trường"
3. Lên lộ trình SAT/TOEFL → gõ "lộ trình SAT"
```

---

## Snooze / Pause Reminders

If student asks to pause weekly check-in:

```
Mình sẽ tạm dừng nhắc tuần. Em muốn tạm nghỉ bao lâu?
• 1 tuần
• 2 tuần
• Cho đến khi em nhắn lại

⚠️ Lưu ý: Nếu có deadline ≤ 7 ngày, mình vẫn sẽ nhắc — đây là cảnh báo không thể tắt.
```

→ On confirmation: run `sa-cli cron list {student_id}` to find the `weekly_checkin_{student_id}` job id, then `sa-cli cron cancel {job_name}`. Re-register by asking the student to trigger any action that creates a new plan. Deadline reminder crons (7-day and below) are never paused.

---

## Cron Registration

Weekly check-in cron is registered automatically the first time `sa-cli plan create` runs. The job fires every Sunday at 10:00 local time and delivers a `weekly_checkin` trigger to this skill.

Cron job name pattern: `weekly_checkin_{student_id}`

---

## Safety Rules

See `../../safety_rules.md`.
