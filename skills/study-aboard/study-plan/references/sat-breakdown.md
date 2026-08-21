# SAT / ACT Study Plan Reference

Section breakdown, weekly task templates, and score gap logic for SAT and ACT plans.

## Table of Contents
- [SAT Score Structure](#sat-score-structure)
- [ACT Score Structure](#act-score-structure)
- [Weekly Task Templates by Phase](#weekly-task-templates-by-phase)
- [Section-Level Gap → Focus Mapping](#section-level-gap--focus-mapping)
- [SAT Math Topic Ladder](#sat-math-topic-ladder)
- [SAT Reading & Writing Topic Ladder](#sat-reading--writing-topic-ladder)
- [Adjusted Plan Rules](#adjusted-plan-rules)
- [Common Resources](#common-resources)

---

## SAT Score Structure

Total: 400–1600 (two sections × 800 each)

| Section | Score Range | Subsections |
|---------|------------|-------------|
| Evidence-Based Reading & Writing (EBRW) | 200–800 | Reading comprehension + Grammar/Usage |
| Math | 200–800 | Algebra, Advanced Math, Problem Solving & Data Analysis, Geometry |

**Target thresholds for US universities:**

| School tier | Minimum SAT |
|-------------|------------|
| Top 10 (MIT, Harvard, Princeton) | 1520–1580+ |
| Top 20 (CMU, Georgetown, UMich) | 1450–1520 |
| Top 50 (Purdue, UMN, Case Western) | 1350–1450 |
| Safety schools | 1200–1350 |

**Test-optional note (2025–2026):** Some schools are reverting to test-required. Always check the specific school's current policy in `universities.db` (`test_policy_2026` field). Never assume test-optional unless confirmed.

---

## ACT Score Structure

Total: 1–36 (composite average of 4 sections)

| Section | Score Range | Notes |
|---------|-----------|-------|
| English | 1–36 | Grammar, rhetorical skills |
| Math | 1–36 | Pre-algebra through trigonometry |
| Reading | 1–36 | 4 passages, 35 min |
| Science | 1–36 | Data interpretation, experimental design |

**SAT ↔ ACT conversion (rough):**

| SAT | ACT |
|-----|-----|
| 1600 | 36 |
| 1520 | 34 |
| 1450 | 32–33 |
| 1380 | 30–31 |
| 1300 | 28–29 |
| 1200 | 25–26 |

Recommend the test the student is more likely to score higher on. If unsure, suggest taking one diagnostic of each.

---

## Weekly Task Templates by Phase

`create_plan.py` uses `SAT_WEEKLY` dict for section rotation. Use these templates when displaying the plan in detail.

### Phase 1 — Foundation (Weeks 1 – 25% of total)

**Goal**: Diagnostic → identify section and topic gaps. No full tests yet.

| Day | SAT | ACT |
|-----|-----|-----|
| Mon | EBRW: 1 Reading passage (12 min timed) + mark unknown vocab | English: 1 grammar passage, categorize error types |
| Tue | Math: Algebra warm-up (20 problems, no calculator) | Math: Pre-algebra + Algebra review |
| Wed | EBRW: Grammar drill — punctuation, subject-verb agreement | Reading: 2 passages (compare passage type difficulty) |
| Thu | Math: Problem Solving & Data Analysis (charts, ratios) | Science: 1 data representation set |
| Fri | EBRW: 1 full Reading section timed (27 min) | Full English section timed |
| Sat | Math: Advanced Math — quadratics, functions | Math: 30-question timed drill |
| Sun | **Diagnostic section** (weakest section) → report section score | **Diagnostic section** → report section score |

### Phase 2 — Skill Building (Middle 50% of weeks)

**Goal**: Section-targeted drilling. Alternate weak section (60%) + strong section maintenance (40%).

| Day | Focus |
|-----|-------|
| Mon | Weak section intensive — 45-min drill |
| Tue | Weak section — error analysis from Mon |
| Wed | Strong section maintenance — 30-min drill |
| Thu | Mixed section — 1 module timed |
| Fri | Math: Advanced Math or Geometry (alternate weeks) |
| Sat | EBRW: Full module timed OR Math: Full module timed (alternate) |
| Sun | **Full practice test** (2 modules each section) → report total + section scores |

### Phase 3 — Test Simulation (Final 25% of weeks)

**Goal**: Stamina, pacing, test-day strategy.

| Day | Focus |
|-----|-------|
| Mon–Tue | Weak topic drills (based on latest practice test error log) |
| Wed | Full-length timed test (all 4 modules) |
| Thu | Full error analysis — categorize mistakes: careless vs knowledge gap |
| Fri | Light: 20 problems from most-missed topics only |
| Sat | Rest or very light review |
| Sun | **Final mock** or actual test day |

---

## Section-Level Gap → Focus Mapping

Use when student reports individual section scores from a practice test.

### EBRW (Reading & Writing)

| Score below target by | Recommended focus |
|----------------------|------------------|
| > 60 pts | Vocabulary in context + passage structure basics |
| 30–60 pts | Command of Evidence questions + transitions/rhetoric |
| < 30 pts | Pacing strategy + final review of error patterns |

**Common Vietnamese student weak areas in EBRW:**
- "Words in Context" — academic vocabulary range
- Inference questions — reading between the lines in complex passages
- Craft & Structure — author's purpose, tone analysis

### Math

| Score below target by | Recommended focus |
|----------------------|------------------|
| > 80 pts | Algebra foundations (linear equations, systems) |
| 40–80 pts | Advanced Math (polynomials, quadratics, exponentials) |
| < 40 pts | Problem Solving & Data Analysis + Geometry (most test-specific) |

**Topic rotation rule**: Alternate Math modules — don't spend 2+ consecutive days on the same topic category.

---

## SAT Math Topic Ladder

Progress through these in order — don't skip ahead:

```
Level 1 — Foundation (target: 550–650)
├── Linear equations (1 variable)
├── Linear equations (2 variables / systems)
├── Ratios, rates, proportions
├── Percentages
└── Basic statistics (mean, median, range)

Level 2 — Intermediate (target: 650–750)
├── Quadratic equations (factoring, quadratic formula)
├── Functions — domain, range, f(x) notation
├── Exponential growth/decay
├── Geometry (area, perimeter, circles, triangles)
└── Data analysis (scatterplots, two-way tables)

Level 3 — Advanced (target: 750–800)
├── Polynomial division, remainder theorem
├── Trigonometry (sin/cos/tan, unit circle basics)
├── Complex numbers
├── Nonlinear systems
└── Geometric and arithmetic sequences
```

---

## SAT Reading & Writing Topic Ladder

```
Level 1 — Foundation
├── Main idea / central claim questions
├── Vocabulary: high-frequency Academic Word List
└── Grammar: subject-verb agreement, pronoun reference

Level 2 — Intermediate
├── Command of Evidence (citing textual support)
├── Words in Context (tone + register)
├── Transitions and logical flow
└── Grammar: punctuation (commas, semicolons, colons)

Level 3 — Advanced
├── Cross-text connections (Passage 1 vs Passage 2)
├── Author's purpose / rhetorical strategy
├── Quantitative evidence interpretation (graphs in passages)
└── Craft & Structure — nuanced inference
```

---

## Adjusted Plan Rules

Same rules as TOEFL/IELTS (see `toefl-breakdown.md`) with SAT-specific thresholds:

| Gap (target - current total SAT) | Phase adjustment |
|----------------------------------|-----------------|
| ≤ 50 pts | On track — increase difficulty (move up one topic level) |
| 51–150 pts | Slightly behind — add 1 extra section drill/day |
| > 150 pts | Significantly behind — reset to Phase 1 intensity |

**If < 4 weeks to test and gap > 100 pts:**
```
Với khoảng cách {gap} điểm và chỉ còn {weeks_left} tuần, 
khả năng cao em sẽ chưa đạt mục tiêu {target_score} trong lần thi này.

Mình gợi ý 2 lựa chọn:
1. Dời ngày thi thêm 6–8 tuần để có thêm thời gian — mình điều chỉnh lộ trình ngay.
2. Giữ ngày thi này (lấy kinh nghiệm thật) + lên kế hoạch thi lại với mục tiêu cao hơn.

Em muốn làm theo hướng nào?
```

---

## Common Resources

Free official SAT/ACT resources only (do not recommend paid platforms by name):

| Resource | For | Notes |
|----------|-----|-------|
| Khan Academy + College Board | SAT | 8 full-length official practice tests, free |
| College Board Bluebook app | SAT | Official digital practice, adaptive |
| ACT.org practice tests | ACT | 5 free official tests |
| SAT daily practice (Khan Academy) | SAT | 10-min/day adaptive drills |
| College Board Score Reports | SAT | Shows exact missed questions after real test |

**Strategy for Vietnamese students specifically:**
- EBRW: Read English news (BBC, NYT) daily — builds passage familiarity faster than drilling alone
- Math: Concepts are the same as Vietnamese high school math, but word problems require careful English parsing — practice translating word problems before solving
- Time management: SAT digital is adaptive — don't skip Module 1 questions, they determine Module 2 difficulty
