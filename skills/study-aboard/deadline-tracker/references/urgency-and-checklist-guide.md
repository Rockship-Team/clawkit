# Deadline Tracker — Urgency Levels & Post-Submission Checklist Reference

## Urgency Level Definitions

| Icon | Days Until Deadline | Response Required |
|------|--------------------|--------------------|
| 🔴 | ≤ 7 days | Immediate action — check essay status, confirm submission readiness |
| 🟡 | 8–14 days | Escalate — finalize essay, check score submissions |
| 🟠 | 15–30 days | Monitor — prompt update on essay progress |
| ⚪ | > 30 days | Low urgency — note in check-in |
| ⛕ | Past deadline (days < 0) | Missed — cancel reminders, offer alternatives |

---

## Cron Reminder Schedule (per application)

Four cron jobs are registered per application at time of `add_application.py`:

| Job Suffix | Fires | Message Tone |
|-----------|-------|--------------|
| `-30d-` | 30 days before deadline | Nhắc nhẹ — essay status check |
| `-14d-` | 14 days before deadline | Nhắc mạnh — finalize draft |
| `-7d-` | 7 days before deadline | Cảnh báo đỏ — submit this week |
| `-1d-` | 1 day before deadline | Final reminder — last chance |

All active cron jobs for an application are **auto-cancelled** when:
- `submission_status` is set to `submitted` or `removed`
- `update_application.py` handles this automatically

Naming pattern: `deadline-{Nd}-{student_id[:8]}-{app_id[:8]}`

---

## Essay Status Flow

```
not_started → draft → revised → final
```

| Status | Meaning | Agent Action |
|--------|---------|--------------|
| `not_started` | Student hasn't started writing | Prompt to begin, especially if deadline ≤ 30 days |
| `draft` | First draft submitted to essay-review | Encourage revision; check if feedback received |
| `revised` | At least one revision done | Ask if another round of feedback needed |
| `final` | Essay finalized and ready | Confirm submission checklist items |

---

## Submission Status Flow

```
not_submitted → submitted
             ↘ removed (withdrew school from list)
             ↘ missed  (deadline passed without submitting)
```

| Status | Cron Jobs | Phase Transition |
|--------|-----------|-----------------|
| `not_submitted` | Active | Continue reminders |
| `submitted` | Cancel all | Trigger post-submission checklist; check if all apps done → offer-comparison |
| `removed` | Cancel all | Confirm removal; adjust school list |
| `missed` | Cancel all | Show missed deadline options (RD round, alternative schools) |

---

## Post-Submission Checklist (per school)

After each submission, prompt student to verify:

| Item | Notes |
|------|-------|
| Application portal shows "Received" | Check within 24–48 hours |
| SAT/ACT scores sent to school | Must be sent directly from College Board/ACT — not self-reported |
| TOEFL/IELTS scores sent | Must be sent directly from ETS/British Council |
| Recommendation letters submitted | Check all recommenders' submission status in portal |
| Financial aid forms (if applicable) | FAFSA or CSS Profile — check school-specific deadline (may differ from app deadline) |
| School report / Counselor recommendation | Counselor submits separately; follow up with school counselor |
| Mid-year report | Sent after semester 1 grades are final (usually January–February) |

---

## Student-Wide Checklist Items

These apply once across the entire application cycle (not per school):

| Key | Item | When to Mark Done |
|-----|------|------------------|
| `common_app_submitted` | Common App account finalized | After first application submitted |
| `sat_scores_sent` | SAT scores sent to all schools | After College Board sends scores |
| `toefl_scores_sent` | TOEFL scores sent to all schools | After ETS sends scores |
| `ielts_scores_sent` | IELTS scores sent | After British Council sends scores |
| `css_profile_done` | CSS Profile completed | Before financial aid deadline |
| `fafsa_done` | FAFSA completed | Before financial aid deadline |
| `rec_letter_1_submitted` | Recommender 1 submitted | Check via Common App portal |
| `rec_letter_2_submitted` | Recommender 2 submitted | Check via Common App portal |
| `rec_letter_3_submitted` | Recommender 3 submitted (if applicable) | Check via Common App portal |
| `transcript_sent` | Official transcript sent | Sent by school counselor |
| `school_report_submitted` | School report sent | Sent by counselor |
| `mid_year_report_submitted` | Mid-year report sent | Sent by counselor after semester 1 |

---

## Phase Transition Trigger

When ALL applications in the student's list have `submission_status = submitted`:

→ Check with `get_dashboard.py` — if no schools remain `not_submitted`, invoke Phase Transition prompt:

```
🎉 Em đã nộp đủ {n} trường! Giai đoạn tiếp theo là chờ kết quả.

Trong lúc chờ:
• Nếu có trường Waitlist → mình giúp em viết Letter of Continued Interest
• Khi có kết quả đầu tiên → gõ "kết quả" để mình giúp so sánh offer
• Nếu được nhận → mình giúp em chuẩn bị visa F-1 và pre-departure

Mình sẽ nhắc em kiểm tra portal định kỳ nhé!
```

`update_application.py` returns `"phase_transition": "offer-comparison"` in its JSON output when this condition is met — use it to trigger the prompt above.
