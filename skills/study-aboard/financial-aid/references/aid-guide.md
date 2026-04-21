# Financial Aid Reference Guide

Data sources, scholarship programs, and university-specific aid policies for Vietnamese international students.

## Table of Contents
- [Aid Application Timelines](#aid-application-timelines)
- [CSS Profile vs FAFSA](#css-profile-vs-fafsa)
- [Need-Blind vs Need-Aware](#need-blind-vs-need-aware)
- [Merit Aid Programs](#merit-aid-programs)
- [Vietnam-Specific Scholarships](#vietnam-specific-scholarships)
- [Aid Estimate Formula](#aid-estimate-formula)
- [ED + Financial Aid Interaction](#ed--financial-aid-interaction)

---

## Aid Application Timelines

| School Type | CSS Profile Deadline | FAFSA Deadline | Aid Decision |
|-------------|---------------------|----------------|--------------|
| Early Decision / EA | 2–4 weeks before ED/EA deadline | Same | With admission |
| Regular Decision | Nov 1 – Jan 15 (varies) | Feb 1 – Mar 1 | March–April |
| CSS Priority | Usually Nov 1 | N/A (intl students) | March |

**Rule of thumb**: CSS Profile deadline is always 2–3 weeks before the school's application deadline. Tell students to submit CSS *before* the Common App deadline, not after.

---

## CSS Profile vs FAFSA

| | CSS Profile | FAFSA |
|-|------------|-------|
| Who uses it | Private colleges, ~450 schools | Public colleges, some private |
| International students | YES — most need-aware schools require it | NO — for US citizens/residents only |
| Cost | ~$25 first school, $16 each additional | Free |
| What it covers | Detailed family assets, home equity, business | Income and basic assets only |
| Managed by | College Board (collegeboard.org/css-profile) | Federal Student Aid (studentaid.gov) |

**For Vietnamese international students**: FAFSA is almost never required. CSS Profile is the standard form for private US schools that offer aid to international students.

---

## Need-Blind vs Need-Aware

| Policy | Meaning | Schools (examples) |
|--------|---------|-------------------|
| **Need-blind for internationals** | Admission not affected by ability to pay | MIT, Harvard, Yale, Princeton, Amherst, Dartmouth |
| **Need-aware for internationals** | Financial need *can* affect admission | Most schools (~80%) |
| **No aid for internationals** | School does not offer aid to intl students | Most public universities (UC, UNC, etc.) |

When a student applies to a need-aware school, their financial need is a real admission factor. Acknowledge this honestly.

---

## Merit Aid Programs

Schools with significant merit aid available to international students (no financial need required):

| School | Program | Max Amount | Notes |
|--------|---------|-----------|-------|
| Vanderbilt | Cornelius Vanderbilt Scholarship | Full tuition | Highly competitive, separate application |
| USC | Presidential Scholarship | Up to full tuition | GPA + SAT based |
| Northeastern | Merit-based | Up to $20,000/yr | Auto-considered |
| Tulane | Dean's Honor Scholarship | $25,000/yr | Auto-considered |
| Case Western | Presidential Scholarship | Up to full tuition | Separate application |
| Rochester | Various merit awards | $15,000–$20,000 | Auto-considered |
| Emory | Various merit awards | Up to $20,000 | Auto-considered |

**Caveat**: Merit aid amounts and availability change annually. Always verify on the official school website.

---

## Vietnam-Specific Scholarships

| Program | Amount | Eligibility | Link |
|---------|--------|------------|------|
| VEF (Vietnam Education Foundation) | Full funding | STEM graduate programs | vef.gov (check current status) |
| EducationUSA Opportunity Funds | Varies | High-need students, top PSAT/SAT | educationusa.state.gov |
| ASEAN Scholarships (Singapore) | Full scholarship | For ASEAN nationals | moe.gov.sg |
| MEXT (Japan) | Full scholarship | Japanese government program | studyinjapan.go.jp |
| Chevening (UK) | Full scholarship | Graduate level, leadership focus | chevening.org |
| Australia Awards | Full scholarship | Graduate level | australiaawards.gov.au |

---

## Aid Estimate Formula

Used by `scripts/get_cost_comparison.py` to estimate net cost. This is approximate — actual aid depends on family financials submitted via CSS Profile.

```
Estimated Aid = financial_aid_international * (1 - need_awareness_penalty)

where:
  financial_aid_international: from universities.db (median aid for intl students)
  need_awareness_penalty:
    - need_blind → 0.0 (full aid estimate)
    - need_aware → 0.15 (reduce by 15% as buffer)
    - no_aid → 1.0 (no aid)

Net Cost = (tuition + room_board + fees) - Estimated Aid
```

Display `⚠️ Ước tính` label whenever showing net cost — never present as confirmed.

---

## ED + Financial Aid Interaction

Early Decision applicants typically receive their financial aid package at the same time as the admission decision.

**Important rules to tell students:**
1. If the aid package is insufficient, the student CAN withdraw from ED without penalty — but they must have documentation showing financial hardship.
2. "Insufficient" means family contribution required > what family can actually pay.
3. The threshold is not arbitrary — the student should discuss with family *before* applying ED what the maximum acceptable net cost is.

**Script for confirming ED + Aid understanding:**
```
Trước khi em apply ED cho {school_name}, mình cần em xác nhận:
• Gia đình em có thể chi tối đa ${max_family_contribution:,}/năm không?
• {school_name} ước tính net cost ~${net_cost:,}/năm sau aid.
• Nếu được nhận ED nhưng aid không đủ, em có quyền rút — nhưng phải có bằng chứng tài chính.

Em và gia đình đã thống nhất con số này chưa?
```
