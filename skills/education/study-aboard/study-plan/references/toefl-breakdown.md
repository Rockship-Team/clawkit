# TOEFL / IELTS Study Plan Reference

4-skills breakdown format and weekly task templates for `create_plan.py` and check-in display.

## Table of Contents
- [TOEFL Score Structure](#toefl-score-structure)
- [IELTS Score Structure](#ielts-score-structure)
- [Weekly Task Templates by Phase](#weekly-task-templates-by-phase)
- [Score Gap → Focus Area Mapping](#score-gap--focus-area-mapping)
- [Adjusted Plan Rules](#adjusted-plan-rules)
- [Common Resources](#common-resources)

---

## TOEFL Score Structure

Total: 0–120 (4 sections × 30 points each)

| Section | Score Range | Typical weak area for Vietnamese students |
|---------|------------|------------------------------------------|
| Reading | 0–30 | Vocabulary range, inference questions |
| Listening | 0–30 | Academic lecture speed, note-taking |
| Speaking | 0–30 | Integrated task, response timing (45s/60s) |
| Writing | 0–30 | Integrated essay (reading+listening synthesis) |

**Target thresholds for US universities:**

| School tier | Minimum TOEFL |
|-------------|--------------|
| Top 20 (MIT, Harvard, etc.) | 100–110+ |
| Top 50 | 90–100 |
| Top 100 | 80–90 |
| Safety schools | 70–80 |

---

## IELTS Score Structure

Total: 0–9 band (average of 4 sections)

| Section | Band Range | Notes |
|---------|-----------|-------|
| Reading | 0–9 | Academic module only (not General Training) |
| Listening | 0–9 | Same for both modules |
| Speaking | 0–9 | Face-to-face interview with examiner |
| Writing | 0–9 | Task 1 (graph/chart) + Task 2 (essay) |

**Target thresholds:**

| School tier | Minimum IELTS |
|-------------|--------------|
| Top 20 UK (Oxford, Imperial) | 7.0–7.5 overall, min 6.5 per section |
| Top UK / AU / CA | 6.5–7.0 overall |
| Safety | 6.0–6.5 overall |

---

## Weekly Task Templates by Phase

`create_plan.py` uses these templates to generate the week-by-week plan based on total weeks and test type.

### Phase 1 — Foundation (Weeks 1 – 30% of total)

**Goal**: Build core skills. Identify biggest gap section.

| Day | TOEFL | IELTS |
|-----|-------|-------|
| Mon | Reading: 1 practice passage + vocab list (20 words) | Reading: 1 Academic passage + highlight linking words |
| Tue | Listening: 1 lecture (TPO), note-taking drill | Listening: Section 3–4 practice, map/diagram |
| Wed | Writing: Integrated essay outline practice | Writing: Task 1 graph description (bar/line chart) |
| Thu | Speaking: Task 1+2 timed practice (record yourself) | Speaking: Part 2 long turn (2 min) practice |
| Fri | Full section drill: weakest section only | Full section drill: weakest section only |
| Sat | Rest or light vocab review | Rest or light vocab review |
| Sun | **Practice test** (1 section only) → report score | **Practice test** (1 section only) → report score |

### Phase 2 — Skill Building (Middle 40% of weeks)

**Goal**: Improve score on target sections. Simulate test conditions.

| Day | Focus |
|-----|-------|
| Mon–Tue | Weak section intensive (2 drills/day) |
| Wed | Timed full-section mock |
| Thu | Review errors from Wed mock |
| Fri | Strong section maintenance (1 drill) |
| Sat | Speaking + Writing combo day |
| Sun | **Full mock test** (all 4 sections) → report total score |

### Phase 3 — Test Simulation (Final 30% of weeks)

**Goal**: Build stamina. Replicate real test conditions.

| Day | Focus |
|-----|-------|
| Mon–Tue | Targeted weak section (based on latest mock) |
| Wed | Full timed mock test |
| Thu | Error analysis + strategy review |
| Fri | Light review only — no new material |
| Sat | Rest |
| Sun | **Final mock** or scheduled actual test |

---

## Score Gap → Focus Area Mapping

When a check-in score comes in, `checkin.py` uses this to determine focus adjustment:

| Gap (target - current) | Phase adjustment |
|------------------------|-----------------|
| ≤ 5 points (TOEFL) / ≤ 0.5 band (IELTS) | On track — increase difficulty |
| 6–15 points / 0.5–1.0 band | Slightly behind — add 1 extra drill/day on weakest section |
| > 15 points / > 1.0 band | Significantly behind — restructure plan to Phase 1 intensity |

**Per-section gap for TOEFL** (if student reports section scores):

| Section | Score below target by... | Recommended focus |
|---------|------------------------|------------------|
| Reading | > 3 pts | Vocabulary expansion (Academic Word List), skim/scan drills |
| Listening | > 3 pts | Slow down → speed up: start at 0.8x speed, increase to 1.2x |
| Speaking | > 3 pts | Template drilling for Integrated task; record + replay |
| Writing | > 3 pts | Integrated essay: 20-min outline → 20-min write drill |

---

## Adjusted Plan Rules

When the adjusted plan is displayed after check-in, follow these rules:

1. **Never reduce total week count** — only restructure remaining weeks.
2. **If on track**: Increase from Phase 1 → Phase 2, or Phase 2 → Phase 3 templates.
3. **If behind**: Stay in current phase for 1 more week before advancing.
4. **If significantly behind and < 4 weeks to test**: Recommend rescheduling the test. Use this message:
   ```
   Với khoảng cách {gap} điểm và chỉ còn {weeks_left} tuần, 
   mình nghĩ em nên cân nhắc dời ngày thi thêm 4–6 tuần để đảm bảo đạt mục tiêu.
   Em có muốn mình điều chỉnh lộ trình với ngày thi mới không?
   ```

---

## Common Resources

Tell students to use these free official resources (do not recommend paid courses unless student asks):

| Resource | For | Link description |
|----------|-----|-----------------|
| ETS TOEFL Practice Online (TPO) | TOEFL | Official practice tests on ets.org |
| IELTS.org practice materials | IELTS | Free sample tests on ielts.org |
| Khan Academy + College Board | SAT | Free SAT prep (8 full practice tests) |
| Magoosh Vocabulary | TOEFL vocab | Free word lists; app available |
| IELTS Liz (ieltsliz.com) | IELTS | Free lessons for all 4 sections |
| TED Talks | Listening (both) | Academic English, auto-subtitles available |

Do not recommend specific paid platforms by name (Magoosh paid, Princeton Review, Kaplan) unless the student explicitly asks about paid options.
