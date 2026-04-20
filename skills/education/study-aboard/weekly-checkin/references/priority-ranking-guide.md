# Weekly Check-in — Priority Ranking & Cron Guide

## Priority Ranking Logic

Apply in strict order — the first matching rule sets the priority tier for that item:

| Rank | Condition | Tier | Action |
|------|-----------|------|--------|
| 1 | Any deadline ≤ 7 days | 🔴 CRITICAL | Put at top of action list; do not bury in summary |
| 2 | Essay `not_started` with deadline ≤ 30 days | 🔴 HIGH | Prompt to start this week, no exceptions |
| 3 | Study plan score not reported this week | 🟡 MEDIUM | Ask student to share latest practice score |
| 4 | Rec letters pending or CSS/FAFSA undone | 🟡 MEDIUM | Named item with specific follow-up |
| 5 | EC not updated > 30 days with upcoming interview/competition | 🟢 LOW | Note in EC section |
| 6 | Essay `draft` not updated > 7 days | 🟢 LOW | Gentle nudge to continue |

**Cap the Priority Action Block at 3 items.** If more than 3 conditions are met, pick the top 3 by rank above. Never list more than 3 — it overwhelms students.

---

## Cron Schedule

| Job Name Pattern | Fires | Condition |
|-----------------|-------|-----------|
| `weekly_checkin_{student_id}` | Every Sunday 10:00 local time | Active student (has at least 1 app or 1 plan) |

- Cron is registered automatically when `profile-assessment` creates a new student record.
- To pause: `sa-cli cron pause {student_id} weekly_checkin {weeks}` — `weeks` can be 1, 2, or `0` (until student resumes manually).
- Hard override: even if paused, if any application has `days_until_deadline ≤ 7`, the system fires a one-off critical reminder regardless. Do NOT suppress this.

---

## Section-by-Section Data Rules

### 📚 ÔN THI (Study Plans)
- Source: `sa-cli plan list {student_id}`
- Show all active plans (status = `active`), skip `completed` or `paused`
- If `last_checkin_score` is null this week: always show the nudge line regardless of other conditions
- Score delta colour: +50 pts → "tiến bộ tốt", +0–49 → "đang cải thiện", negative → flag for plan adjustment

### ✍️ ESSAY
- Source: `sa-cli essay list {student_id}`
- `days_since_update` = today − `last_updated_date`
- Show Common App PS first, then supplementals ordered by deadline
- Do NOT show essays for schools already `submitted` or `removed`

### 📅 DEADLINE SẮP TỚI
- Source: `sa-cli application dashboard {student_id}`
- Only show apps with `days_until_deadline ≤ 30` and `submission_status = not_submitted`
- Use urgency icons from `deadline-tracker/references/urgency-and-checklist-guide.md`

### 📋 CHECKLIST CHUNG
- Source: `sa-cli checklist get {student_id}`
- Display aggregate `done/total` — do not list every item unless student asks
- Surface only the blocking items: rec letters, CSS Profile, FAFSA

### 🏆 HOẠT ĐỘNG NGOẠI KHOÁ
- Source: `sa-cli ec list {student_id}`
- Show active ECs (status = `active` or `in_progress`), skip `completed`
- Flag if `days_since_update > 30` — EC strategy may need revisit

---

## No Active Data Fallback

| Condition | Response |
|-----------|----------|
| No plans + no apps | Redirect to onboarding: profile-assessment → school-matching → study-plan |
| Plans exist but no apps | Remind to build school list: "Em đã có lộ trình ôn thi nhưng chưa có danh sách trường — gõ 'gợi ý trường' để mình giúp chọn." |
| Apps exist but no plans | Prompt to create study plan: "Em có danh sách trường nhưng chưa có lộ trình ôn thi — gõ 'lộ trình SAT' để bắt đầu." |

---

## Phase Transition Detection

At the end of weekly check-in, check `sa-cli application dashboard`:

| Condition | Action |
|-----------|--------|
| All apps `submitted` | Trigger offer-comparison prompt (see `deadline-tracker` urgency guide for exact text) |
| At least 1 offer received (`offer_received = true`) | Mention: "Em đã có kết quả từ {school} — gõ 'so sánh offer' để mình giúp ra quyết định." |
| Pre-departure triggered (deposit paid) | Handoff to `pre-departure` skill |

---

## Emotional Distress Protocol

Before rendering any data, scan the student's message (or the absence of a message — cron-triggered) for distress signals listed in `../../safety_rules.md`. For cron-triggered check-ins, check the last 7 days of messages via context if available; if none, proceed normally.
