---
name: deadline-tracker
description: Track application deadlines and send proactive alerts for Vietnamese students applying to universities. Use this skill whenever a student asks about deadlines, application status, mentions 'deadline', 'hạn nộp', 'lịch nộp hồ sơ', 'trạng thái hồ sơ', 'khi nào nộp', 'còn bao lâu nữa', 'đã submit chưa', or when proactively checking on upcoming deadlines. Also handles post-submission checklists.
metadata:
  openclaw:
    emoji: 📅
    requires:
      bins: [sa-cli]
---

# Deadline Tracker Skill

Manage the student's application dashboard and ensure no deadline is missed.

## ⛔ Safety Check — Enforce Before Any Response

| If student asks you to… | Respond with |
|-------------------------|--------------|
| Guarantee admission outcome | "Mình không thể đảm bảo kết quả — deadline tracking giúp em không bỏ lỡ cơ hội, nhưng kết quả phụ thuộc vào cả hồ sơ." |
| Fabricate submission status / create false documents | "Mình không thể hỗ trợ khai báo thông tin không đúng trong hồ sơ nộp." |

For the full rules list see `../../safety_rules.md`. Before processing any request, scan for emotional distress signals (see Emotional Distress Protocol in `../../safety_rules.md`) — if detected, follow the empathy-first protocol before continuing. Works with OpenClaw cron jobs for automated reminders.

## Show Dashboard

Run `sa-cli student query {channel} {channel_user_id}` to get student profile, then run `sa-cli application dashboard {student_id}` to fetch the application list with urgency flags.

```
📋 DASHBOARD HỒ SƠ — {display_name}
══════════════════════════════════════════
Mục tiêu: {intended_major} | Ngân sách: ${annual_budget_usd:,}/năm

┌────────────────────┬────────┬───────────┬──────────┬──────────┐
│ Trường             │ Loại   │ Deadline  │ Essay    │ Trạng thái│
├────────────────────┼────────┼───────────┼──────────┼──────────┤
{for app in applications:}
│ {app.university_name:<18} │ {app.category:<6} │ {app.application_type} {app.deadline} │ {essay_status_icon} {app.essay_status:<6} │ {status_icon} {app.submission_status} │
└────────────────────┴────────┴───────────┴──────────┴──────────┘

CẢNH BÁO:
{for app in urgent_apps:}
{urgency_icon(app.days_until_deadline)} {app.university_name} {app.application_type} — còn {app.days_until_deadline} ngày
  Essay: {app.essay_status} | Cần làm: {app.next_action}
```

Urgency icons: 🔴 ≤ 7 days | 🟡 ≤ 14 days | 🟠 ≤ 30 days | ⚪ > 30 days | ⛕ PASSED (days_until_deadline < 0)
Essay status icons: ✅ final | ⚠️ draft | ❌ not_started

## Missed Deadline Handling

When `days_until_deadline < 0` for an application with `submission_status != submitted`:

```
⛕ ĐÃ QUÁ HẠN: {school_name} {app_type} — deadline {deadline} đã qua {abs(days_until_deadline)} ngày.

Một số lựa chọn em có thể xem xét:
• Rolling admissions: Một số trường (thường là ít selective) vẫn nhận hồ sơ sau deadline — mình kiểm tra được cho em.
• RD thay ED/EA: Nếu em bỏ lỡ ED/EA, mình kiểm tra xem trường có vòng RD không.
• Cycle tiếp theo: Nếu em chưa sẵn sàng, apply cycle mùa sau với hồ sơ mạnh hơn cũng là lựa chọn tốt.
• Waitlist/Transfer: Một số trường có intake mùa hè hoặc chương trình transfer.

Em muốn mình tìm trường thay thế có deadline phù hợp hơn không?
```

Do NOT show cron reminder for missed deadlines — auto-cancel any remaining reminders for that application by running `sa-cli application update {app_id} submission_status missed`.

## Remove School from Application List

Before removing any school, enforce the confirmation gate from `safety_rules.md`:

1. Respond: `"Đây là quyết định quan trọng — em có chắc chắn muốn bỏ {school_name} ra khỏi danh sách không?"`
2. Wait for explicit confirmation ("có", "chắc rồi", "xoá đi"). Do NOT proceed on ambiguous messages.
3. On confirmed → run `sa-cli application update {app_id} submission_status removed` — this also auto-cancels any remaining cron reminders for that application.
4. Confirm:
```
✅ Đã xoá {school_name} khỏi danh sách. Các nhắc nhở deadline cũng đã được huỷ.
```

## Add School to Application List

When student wants to add a school:
1. Confirm school name and application type (ED/EA/RD)
2. Run `sa-cli application add {student_id} {university_id} "{university_name}" {reach|target|safety} {ED|EA|RD|rolling} {deadline|-} {channel} {channel_user_id}` — saves to DB and registers cron reminders automatically
4. Confirm:
```
✅ Đã thêm {school_name} vào danh sách! Mình sẽ nhắc em:
• 30 ngày trước deadline: nhắc sớm
• 14 ngày: nhắc mạnh hơn
• 7 ngày: cảnh báo đỏ
• 1 ngày: nhắc cuối
```

## Update Essay Status

When student says they finished/updated an essay:
→ Run `sa-cli application update {app_id} essay_status {not_started|draft|final}`
```
✅ Đã cập nhật essay {school_name}: {new_status}
```

## Mark as Submitted

When student confirms submission:
1. Run `sa-cli application update {app_id} submission_status submitted` — saves status and auto-cancels remaining cron reminders
3. Display post-submission checklist:

```
🎉 Chúc mừng em đã nộp đơn {school_name}!

📋 CHECKLIST SAU KHI SUBMIT:
☐ Gửi SAT/ACT score chính thức (nếu chưa)
☐ Gửi TOEFL/IELTS score chính thức (nếu chưa)  
☐ Kiểm tra application portal — xác nhận "Application Received"
☐ Thư giới thiệu ({rec_letters_submitted}/total submitted?)
☐ Financial aid forms (FAFSA/CSS Profile) — deadline: {financial_aid_deadline}

Mình sẽ nhắc em kiểm tra portal sau 1 tuần nhé!
```

## Cron Reminder Templates

When a cron job fires and delivers a reminder message to this skill:

**30-day reminder:**
```
📅 Nhắc nhẹ: {school_name} {app_type} còn 30 ngày (deadline {deadline}).
Essay của em đang ở trạng thái: {essay_status}.
```

**7-day reminder:**
```
🔴 Còn 7 NGÀY! {school_name} {app_type} deadline {deadline}.
Essay: {essay_status}. Cần submit tuần này rồi em ơi!
```

**1-day reminder:**
```
🚨 NGÀY MAI là deadline {school_name}!
✅ Checklist cuối: essay done? scores sent? financial aid?
```

**14-day reminder:**
```
🟡 Còn 14 ngày! {school_name} {app_type} deadline {deadline}.
Essay: {essay_status}. Đây là thời điểm lý tưởng để finalize draft nhé.
```

## Student-Wide Checklist

Run `sa-cli checklist get {student_id}` to fetch the global checklist (not per-school — these are items that apply once across the entire application cycle).

Display format:
```
📋 CHECKLIST CHUNG — {display_name}
══════════════════════════════════════════
☑ Common App submitted
☑ SAT scores sent to schools
☑ TOEFL scores sent to schools (108)
☐ IELTS scores sent to schools
☐ CSS Profile completed
☐ FAFSA completed
☑ Recommendation letter 1 submitted
⚠️ Recommendation letter 2 submitted  ← còn thiếu theo notes
☐ Recommendation letter 3 submitted
☑ Official transcript sent
☐ School report submitted
☐ Mid-year report submitted

{done}/{total} hoàn thành
📝 Ghi chú: {notes}
```

When student says a checklist item is done:
→ Run `sa-cli checklist update {student_id} {item_key} 1`

When student wants to add a note (e.g. "cô Thuỷ chưa submit"):
→ Run `sa-cli checklist notes {student_id} "{text}"`

Trigger proactively: Show checklist when student asks about overall status, or when dashboard shows ≥ 2 schools with deadline ≤ 30 days.

## Phase Transition — After All Schools Submitted

When `submission_status == "submitted"` for ALL applications in the list:

```
🎉 Em đã nộp đủ {n} trường rồi! Giai đoạn tiếp theo là chờ kết quả.

Trong lúc chờ:
• Nếu có trường Waitlist → mình giúp em viết Letter of Continued Interest
• Khi có kết quả đầu tiên → gõ "kết quả" để mình giúp so sánh offer
• Nếu được nhận → mình giúp em chuẩn bị visa F-1 và pre-departure

Mình sẽ nhắc em kiểm tra portal định kỳ nhé!
```

## References

See `references/urgency-and-checklist-guide.md` for urgency level definitions, cron job naming patterns, essay/submission status flows, and phase transition trigger details.

## Safety Rules

See `../../safety_rules.md`.
