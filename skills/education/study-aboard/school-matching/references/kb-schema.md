# University Knowledge Base Schema

Documents the data format stored in `universities.db` and used by `get_school_list.py` for display.

## Table of Contents
- [University Record Fields — Core (all countries)](#university-record-fields--core-all-countries)
- [US-Specific Fields](#us-specific-fields)
- [UK-Specific Fields](#uk-specific-fields)
- [Canada-Specific Fields](#canada-specific-fields)
- [Australia-Specific Fields](#australia-specific-fields)
- [Display Field Mapping by Country](#display-field-mapping-by-country)
- [Reach / Target / Safety Thresholds by Country](#reach--target--safety-thresholds-by-country)
- [budget_flag Logic](#budget_flag-logic)
- [last_updated Staleness Check](#last_updated-staleness-check)
- [Seed Data Countries](#seed-data-countries)

---

## University Record Fields — Core (all countries)

| Field | Type | Description |
|-------|------|-------------|
| `id` | INTEGER PK | Internal ID |
| `name` | TEXT | Full university name |
| `country` | TEXT | ISO 2-letter code: US, UK, CA, AU, SG, HK, JP, KR, DE, NL |
| `city` | TEXT | City / campus location |
| `acceptance_rate_overall` | REAL | Overall acceptance rate (0.0–1.0) |
| `acceptance_rate_intl` | REAL | International student acceptance rate (0.0–1.0), NULL if unknown |
| `toefl_minimum` | INTEGER | Minimum TOEFL iBT score required (NULL = not required / test-flexible) |
| `ielts_minimum` | REAL | Minimum IELTS overall score required (NULL = not required) |
| `tuition_local_currency` | INTEGER | Annual tuition in local currency |
| `tuition_currency` | TEXT | ISO currency code: USD, GBP, CAD, AUD |
| `tuition_usd_approx` | INTEGER | Approximate tuition in USD (for cross-country cost comparison) |
| `living_cost_local_currency` | INTEGER | Estimated annual living cost in local currency |
| `living_cost_usd_approx` | INTEGER | Approximate living cost in USD |
| `total_cost_usd_approx` | INTEGER | `tuition_usd_approx + living_cost_usd_approx` |
| `strong_majors` | TEXT | JSON array of majors this school is known for |
| `ranking_qs_2026` | INTEGER | QS World University Ranking 2026 (NULL if unranked) |
| `last_updated` | TEXT | ISO date of last data update (YYYY-MM-DD) |

---

## US-Specific Fields

| Field | Type | Description |
|-------|------|-------------|
| `sat_25` | INTEGER | 25th percentile SAT score |
| `sat_75` | INTEGER | 75th percentile SAT score |
| `act_25` | INTEGER | 25th percentile ACT score |
| `act_75` | INTEGER | 75th percentile ACT score |
| `gpa_avg` | REAL | Average admitted GPA (4.0 scale) |
| `test_policy` | TEXT | `required` / `optional` / `blind` |
| `financial_aid_international` | INTEGER | Median financial aid for intl students (USD/yr), 0 if no aid |
| `need_blind_intl` | INTEGER | 1 = need-blind, 0 = need-aware, -1 = no aid |
| `css_profile_required` | INTEGER | 1 = required, 0 = not |
| `ea_deadline` | TEXT | Early Action deadline (YYYY-MM-DD), NULL if none |
| `ed_deadline` | TEXT | Early Decision deadline (YYYY-MM-DD), NULL if none |
| `rd_deadline` | TEXT | Regular Decision deadline (YYYY-MM-DD) |
| `application_platform` | TEXT | `common_app` / `coalition` / `direct` |

---

## UK-Specific Fields

| Field | Type | Description |
|-------|------|-------------|
| `ucas_code` | TEXT | 4-letter UCAS institution code (e.g. `OXFD`, `CMBG`, `LNDN`) |
| `ucas_deadline` | TEXT | UCAS submission deadline: `2026-10-15` (Oxbridge/Medicine) or `2026-01-31` (others) |
| `ucas_tariff_min` | INTEGER | Minimum UCAS Tariff Points (e.g. 128 = AAA at A-Level) |
| `a_level_requirement` | TEXT | Typical A-Level offer, e.g. `"AAA"`, `"AAB"`, `"A*AA"` |
| `a_level_requirement_subject` | TEXT | Required subjects, e.g. `"Maths + Physics"`, `"Chemistry + Biology"` |
| `ib_requirement_points` | INTEGER | Minimum IB Diploma total points, e.g. `38`, `36`, `34` |
| `ib_requirement_hl` | TEXT | IB Higher Level requirement, e.g. `"766"`, `"655"` |
| `interview_required` | INTEGER | 1 = interview is part of selection (Oxbridge, Medicine), 0 = no |
| `interview_format` | TEXT | e.g. `"Oxford tutorial-style"`, `"MMI (Medicine)"`, NULL if none |
| `oxbridge` | INTEGER | 1 = Oxford or Cambridge, 0 = other |
| `russell_group` | INTEGER | 1 = member of Russell Group, 0 = other |
| `personal_statement_focus` | TEXT | Key content the personal statement should address for this school/major |
| `cas_processing_days` | INTEGER | Typical days to receive CAS after enrollment deposit (range: 7–30) |
| `ihs_annual_gbp` | INTEGER | Immigration Health Surcharge per year (standard: £776 as of 2025) |
| `tb_test_required` | INTEGER | 1 = TB test required for Vietnamese nationals, 0 = not required |
| `brp_collection_days` | INTEGER | Days to collect BRP after arriving in UK (standard: 10) |
| `scholarship_available` | TEXT | JSON array of available scholarships, e.g. `[{"name":"Global Excellence","value":"£5000/yr"}]` |
| `foundation_year_available` | INTEGER | 1 = school offers Foundation Year for students who don't meet direct entry |

---

## Canada-Specific Fields

| Field | Type | Description |
|-------|------|-------------|
| `province` | TEXT | Province code: `ON`, `BC`, `QC`, `AB`, `MB`, `SK`, `NS`, `NB`, `PE`, `NL` |
| `application_platform_ca` | TEXT | `ouac` (Ontario), `direct` (BC/AB/others), `uapply` (some BC), `bac` (QC) |
| `ouac_code` | TEXT | OUAC code for Ontario universities (NULL for other provinces) |
| `application_deadline_ca` | TEXT | Main application deadline (YYYY-MM-DD), e.g. `2026-01-15` |
| `gpa_requirement_ca` | REAL | Minimum GPA on 4.0 scale for admission consideration |
| `gpa_requirement_ca_note` | TEXT | Context note, e.g. `"Waterloo Engineering: 90%+ average in strong Grade 12 subjects"` |
| `caq_required` | INTEGER | 1 = school is in Québec, requires CAQ (Certificat d'acceptation du Québec) |
| `caq_processing_weeks` | INTEGER | Typical CAQ processing time in weeks (usually 4–8) |
| `study_permit_processing_weeks` | INTEGER | Typical IRCC Study Permit processing time for Vietnamese nationals (weeks) |
| `biometrics_required` | INTEGER | 1 = biometrics required at VAC (standard for CA Study Permit) |
| `biometrics_fee_cad` | INTEGER | Biometrics fee in CAD (standard: $85) |
| `co_op_available` | INTEGER | 1 = co-op / work-integrated learning program available |
| `pgwp_eligible` | INTEGER | 1 = graduates eligible for Post-Graduation Work Permit |
| `pgwp_duration_years` | REAL | PGWP duration in years (1–3, based on program length) |
| `scholarship_available` | TEXT | JSON array of available scholarships |

---

## Australia-Specific Fields

| Field | Type | Description |
|-------|------|-------------|
| `state` | TEXT | State code: `NSW`, `VIC`, `QLD`, `WA`, `SA`, `TAS`, `ACT`, `NT` |
| `application_platform_au` | TEXT | `uac` (NSW), `vtac` (VIC), `qtac` (QLD), `satac` (SA), `tisc` (WA), `direct` |
| `application_deadline_au` | TEXT | Main application deadline (YYYY-MM-DD) |
| `ielts_minimum_overall` | REAL | IELTS overall minimum (typically 6.0–7.0) |
| `ielts_minimum_band` | REAL | IELTS minimum per-band score (no band below this) |
| `ielts_note` | TEXT | e.g. `"No band below 6.0"`, `"Writing 7.0 for Engineering"` |
| `atar_equivalent_min` | REAL | Minimum ATAR (or equivalent) for direct entry, NULL if not published |
| `go8_member` | INTEGER | 1 = Group of Eight member, 0 = other |
| `coe_processing_days` | INTEGER | Typical days to receive CoE after enrollment deposit (range: 3–14) |
| `oshc_required` | INTEGER | 1 = OSHC mandatory (always 1 for international students in AU) |
| `oshc_approx_aud_per_year` | INTEGER | Approximate OSHC cost per year in AUD (range: 600–750) |
| `oshc_providers` | TEXT | Comma-separated list of OSHC providers the school accepts |
| `gte_notes` | TEXT | GTE (Genuine Temporary Entrant) — factors this school's visa officers typically scrutinise |
| `student_visa_500_processing_weeks` | INTEGER | Typical Student Visa subclass 500 processing time for Vietnamese nationals |
| `visa_biometrics_required` | INTEGER | 1 = biometrics at VFS Global may be requested |
| `graduate_visa_485_eligible` | INTEGER | 1 = graduates eligible for Temporary Graduate visa (subclass 485) |
| `graduate_visa_485_years` | REAL | Duration of 485 visa in years (2 for Bachelor, 2–4 for postgrad) |
| `scholarship_available` | TEXT | JSON array of available scholarships |
| `foundation_pathway_available` | INTEGER | 1 = school offers or partners with a Foundation Studies provider |

---

## Display Field Mapping by Country

### 🇺🇸 USA Display Card

| Template placeholder | DB field |
|---------------------|----------|
| `{school.name}` | `name` |
| `{rate}%` | `acceptance_rate_intl` or `acceptance_rate_overall` |
| `{sat_25}–{sat_75}` | `sat_25`, `sat_75` |
| `{cost:,}` | `total_cost_usd_approx` |
| `{net_cost:,}` | `total_cost_usd_approx - financial_aid_international` |
| `{ea_deadline}` | `ea_deadline` (DD/MM/YYYY, "N/A" if NULL) |
| `{rd_deadline}` | `rd_deadline` |
| `{fit_rationale}` | `strong_majors` matched against `intended_major` |

---

### 🇬🇧 UK Display Card

```
┌─────────────────────────────────────────────┐
│ {school.name}  {if russell_group: "Russell Group ⭐"}  │
│ {city}                                       │
│ Yêu cầu: A-Level {a_level_requirement}       │
│          IB: {ib_requirement_points} điểm    │
│          ({ib_requirement_hl} tại HL)         │
│          {if a_level_requirement_subject: "Môn bắt buộc: {a_level_requirement_subject}"} │
│ UCAS deadline: {ucas_deadline_display}        │
│   {if oxbridge: "⚠️ Oxford/Cambridge: deadline 15/10, interview bắt buộc"} │
│ IELTS tối thiểu: {ielts_minimum}             │
│ Học phí: £{tuition_local_currency:,}/năm     │
│ Sinh hoạt: £{living_cost_local_currency:,}/năm│
│ IHS (bảo hiểm NHS): £{ihs_annual_gbp}/năm    │
│ Tổng ước tính: £{total_gbp:,}/năm            │
│   (~${total_cost_usd_approx:,} USD)          │
│                                              │
│ {if tb_test_required: "💉 TB test bắt buộc cho sinh viên từ VN"} │
│ CAS: nhận sau ~{cas_processing_days} ngày sau khi đặt cọc │
│ BRP: lấy trong {brp_collection_days} ngày sau khi nhập học │
│                                              │
│ {if scholarship_available: "🎓 Học bổng: {scholarship_names}"} │
│ 💡 {fit_rationale}                           │
└─────────────────────────────────────────────┘
```

**Likelihood classification for UK** (thay vì Reach/Target/Safety):

| Label | Condition |
|-------|-----------|
| 🟢 Likely offer | Predicted IB ≥ `ib_requirement_points` + 2, or A-Level ≥ requirement |
| 🟡 Borderline | Predicted IB = `ib_requirement_points` ± 1 |
| 🔴 Ambitious | Predicted IB < `ib_requirement_points` - 1, or missing required subject |

Predicted grades sourced from `student.predicted_ib` or `student.predicted_alevel` (set during profile-assessment).

---

### 🇨🇦 Canada Display Card

```
┌─────────────────────────────────────────────┐
│ {school.name}                               │
│ {city}, {province}                          │
│ GPA tối thiểu: {gpa_requirement_ca}/4.0     │
│   {gpa_requirement_ca_note}                 │
│ IELTS tối thiểu: {ielts_minimum}            │
│ Apply qua: {application_platform_ca_display} │
│ Deadline: {application_deadline_ca_display}  │
│                                             │
│ {if caq_required: "⚠️ Trường ở Québec — cần xin CAQ trước (~{caq_processing_weeks} tuần)"} │
│ Study Permit: ~{study_permit_processing_weeks} tuần xử lý │
│ {if biometrics_required: "💾 Biometrics bắt buộc tại VAC (CAD ${biometrics_fee_cad})"} │
│                                             │
│ Học phí: CAD ${tuition_local_currency:,}/năm│
│ Sinh hoạt: CAD ${living_cost_local_currency:,}/năm │
│ Tổng ước tính: ~${total_cost_usd_approx:,} USD/năm │
│                                             │
│ {if co_op_available: "🔧 Có Co-op program"} │
│ {if pgwp_eligible: "📋 PGWP: ở lại làm việc {pgwp_duration_years} năm sau tốt nghiệp"} │
│ {if scholarship_available: "🎓 Học bổng: {scholarship_names}"} │
│ 💡 {fit_rationale}                           │
└─────────────────────────────────────────────┘
```

---

### 🇦🇺 Australia Display Card

```
┌─────────────────────────────────────────────┐
│ {school.name}  {if go8_member: "Go8 ⭐"}    │
│ {city}, {state}                             │
│ IELTS tối thiểu: {ielts_minimum_overall} overall │
│   Không band nào dưới {ielts_minimum_band}  │
│   {ielts_note}                              │
│ Apply qua: {application_platform_au_display} │
│ Deadline: {application_deadline_au_display}  │
│                                             │
│ OSHC (bảo hiểm bắt buộc): ~AUD ${oshc_approx_aud_per_year}/năm │
│   Nhà cung cấp: {oshc_providers}           │
│ CoE: nhận sau ~{coe_processing_days} ngày sau khi đặt cọc │
│ Student Visa 500: ~{student_visa_500_processing_weeks} tuần xử lý │
│ {if visa_biometrics_required: "💾 Có thể yêu cầu biometrics tại VFS Global"} │
│                                             │
│ ⚠️ GTE (Genuine Temporary Entrant):         │
│   {gte_notes}                               │
│                                             │
│ Học phí: AUD ${tuition_local_currency:,}/năm│
│ Sinh hoạt: AUD ${living_cost_local_currency:,}/năm │
│ Tổng ước tính: ~${total_cost_usd_approx:,} USD/năm │
│                                             │
│ {if graduate_visa_485_eligible: "📋 Visa 485: ở lại {graduate_visa_485_years} năm sau tốt nghiệp"} │
│ {if scholarship_available: "🎓 Học bổng: {scholarship_names}"} │
│ 💡 {fit_rationale}                           │
└─────────────────────────────────────────────┘
```

---

## Reach / Target / Safety Thresholds by Country

### 🇺🇸 USA

`get_school_list.py` classifies each US school using a composite score:
1. GPA gap: `school.gpa_avg - student.gpa_4scale`
2. SAT gap: `school.sat_75 - student.sat_score` (if SAT available)
3. Acceptance rate: `school.acceptance_rate_intl`

| Category | Acceptance rate (intl) | GPA gap | Typical chance |
|----------|----------------------|---------|----------------|
| Reach 🔴 | < 20% OR GPA gap > 0.5 | Significant | 15–25% |
| Target 🟡 | 20–50% AND GPA gap ≤ 0.3 | Moderate | 40–60% |
| Safety 🟢 | > 50% AND GPA gap ≤ 0 | Minor/none | 70%+ |

Minimum recommended list: 2 Reach, 3 Target, 2 Safety.
If student has no SAT: classify based on GPA + acceptance rate only.

---

### 🇬🇧 UK

UK classification is based on **entry requirements vs student's predicted grades** (not acceptance rate, which is less meaningful given UCAS structure).

```python
def uk_likelihood(school, student):
    predicted_ib = student.predicted_ib  # from profile
    predicted_al = student.predicted_alevel  # e.g. "AAB"
    
    ib_gap = (school.ib_requirement_points or 0) - (predicted_ib or 0)
    
    if ib_gap <= -2:   # student predicted 2+ points above requirement
        return "likely"
    elif -2 < ib_gap <= 1:
        return "borderline"
    else:              # student predicted more than 1 point below requirement
        return "ambitious"
```

Display label:
- `likely` → 🟢 Likely offer
- `borderline` → 🟡 Borderline
- `ambitious` → 🔴 Ambitious

⚠️ If student is on VN curriculum (not IB/A-Level), display:
```
⚠️ {school_name} yêu cầu bằng quốc tế (A-Level / IB). Em nên:
1. Thi A-Level / IB trước khi apply, HOẶC
2. Xem xét Foundation Year tại trường (nếu có: {foundation_year_available})
```

---

### 🇨🇦 Canada

Classification based on GPA comparison:

| Category | GPA gap (school.gpa_req - student.gpa_4scale) | Label |
|----------|----------------------------------------------|-------|
| Very likely 🟢 | ≤ -0.2 (student above req) | Safety-equivalent |
| Likely 🟡 | -0.2 to +0.2 | Target-equivalent |
| Reach 🔴 | > +0.2 | Reach |

Note: Canadian universities are generally less selective than US counterparts for international students with strong GPA + IELTS.

---

### 🇦🇺 Australia

Classification based on IELTS + ATAR equivalent:

| Category | IELTS gap | ATAR equivalent | Label |
|----------|-----------|-----------------|-------|
| Very likely 🟢 | Student IELTS ≥ school minimum + 0.5 | Above published min | Safety-equivalent |
| Likely 🟡 | Student IELTS ≥ school minimum | Meets published min | Target-equivalent |
| Reach 🔴 | Student IELTS < school minimum | Below published min | Reach (IELTS is hard gate) |

⚠️ IELTS is a **hard gate** in Australia — if student IELTS is below minimum, even strong academics cannot compensate. Always surface this first.

---

## budget_flag Logic

Works across all countries using `total_cost_usd_approx`:

```python
def budget_flag(school, annual_budget_usd):
    net_cost = school.total_cost_usd_approx - (school.financial_aid_international or 0)
    if net_cost <= annual_budget_usd:
        return "✅"
    elif net_cost <= annual_budget_usd * 1.2:
        return "⚠️ Hơi vượt ngân sách"
    else:
        return "❌ Vượt ngân sách"
```

For UK: also add IHS to net_cost: `net_cost += school.ihs_annual_gbp * GBP_TO_USD`
For AU: also add OSHC: `net_cost += school.oshc_approx_aud_per_year * AUD_TO_USD`

---

## last_updated Staleness Check

```python
from datetime import date

STALE_THRESHOLD_DAYS = 365

def is_stale(last_updated_str: str) -> bool:
    last_updated = date.fromisoformat(last_updated_str)
    return (date.today() - last_updated).days > STALE_THRESHOLD_DAYS
```

If `is_stale(school.last_updated)` → append staleness warning to school card.

---

## Seed Data Countries

The `data/` directory contains seed files:

| File | Country | # Universities (approx) | Key fields added |
|------|---------|------------------------|-----------------|
| `seed_US.json` | United States | ~80 | sat_25/75, gpa_avg, ea/ed/rd_deadline, css_profile_required |
| `seed_UK.json` | United Kingdom | ~30 | ucas_code, a_level_requirement, ib_requirement_points, ib_requirement_hl, ucas_deadline, interview_required, oxbridge, russell_group, cas_processing_days, ihs_annual_gbp, tb_test_required, brp_collection_days |
| `seed_CA.json` | Canada | ~20 | province, application_platform_ca, ouac_code, application_deadline_ca, gpa_requirement_ca, caq_required, caq_processing_weeks, study_permit_processing_weeks, biometrics_required, co_op_available, pgwp_eligible, pgwp_duration_years |
| `seed_AU.json` | Australia | ~20 | state, application_platform_au, ielts_minimum_overall, ielts_minimum_band, ielts_note, go8_member, coe_processing_days, oshc_required, oshc_approx_aud_per_year, oshc_providers, gte_notes, student_visa_500_processing_weeks, graduate_visa_485_eligible, graduate_visa_485_years |
| `seed_SG.json` | Singapore | ~5 | — |
| `seed_HK.json` | Hong Kong | ~5 | — |
| `seed_JP.json` | Japan | ~10 | — |
| `seed_KR.json` | South Korea | ~10 | — |
| `seed_DE.json` | Germany | ~10 | — |
| `seed_NL.json` | Netherlands | ~10 | — |

Seed files use the same field names as the DB schema above. Run `scripts/seed_universities.py` to reload from seed files.

---

## Sample Seed Records

### UK — University of Edinburgh

```json
{
  "name": "University of Edinburgh",
  "country": "UK",
  "city": "Edinburgh",
  "acceptance_rate_overall": 0.42,
  "acceptance_rate_intl": 0.38,
  "toefl_minimum": 100,
  "ielts_minimum": 6.5,
  "tuition_local_currency": 27600,
  "tuition_currency": "GBP",
  "tuition_usd_approx": 34900,
  "living_cost_local_currency": 13000,
  "living_cost_usd_approx": 16400,
  "total_cost_usd_approx": 51300,
  "strong_majors": ["Medicine", "Law", "Philosophy", "Computer Science", "Engineering"],
  "ranking_qs_2026": 27,
  "ucas_code": "EDINB",
  "ucas_deadline": "2026-01-31",
  "ucas_tariff_min": 120,
  "a_level_requirement": "AAA",
  "a_level_requirement_subject": null,
  "ib_requirement_points": 37,
  "ib_requirement_hl": "666",
  "interview_required": 0,
  "interview_format": null,
  "oxbridge": 0,
  "russell_group": 1,
  "personal_statement_focus": "Academic motivation for chosen subject, relevant reading and activities",
  "cas_processing_days": 14,
  "ihs_annual_gbp": 776,
  "tb_test_required": 1,
  "brp_collection_days": 10,
  "scholarship_available": [{"name": "Global Scholarships", "value": "£5,000 one-off"}],
  "foundation_year_available": 0,
  "last_updated": "2025-08-01"
}
```

### Canada — University of Waterloo

```json
{
  "name": "University of Waterloo",
  "country": "CA",
  "city": "Waterloo",
  "province": "ON",
  "acceptance_rate_overall": 0.53,
  "acceptance_rate_intl": 0.45,
  "toefl_minimum": 90,
  "ielts_minimum": 6.5,
  "tuition_local_currency": 48000,
  "tuition_currency": "CAD",
  "tuition_usd_approx": 35000,
  "living_cost_local_currency": 16000,
  "living_cost_usd_approx": 11700,
  "total_cost_usd_approx": 46700,
  "strong_majors": ["Computer Science", "Engineering", "Mathematics", "Accounting"],
  "ranking_qs_2026": 154,
  "application_platform_ca": "ouac",
  "ouac_code": "WAT",
  "application_deadline_ca": "2026-02-01",
  "gpa_requirement_ca": 3.7,
  "gpa_requirement_ca_note": "Engineering/CS: thường yêu cầu 90%+ trung bình Grade 12 các môn chính",
  "caq_required": 0,
  "caq_processing_weeks": null,
  "study_permit_processing_weeks": 8,
  "biometrics_required": 1,
  "biometrics_fee_cad": 85,
  "co_op_available": 1,
  "pgwp_eligible": 1,
  "pgwp_duration_years": 3,
  "scholarship_available": [{"name": "International Student Merit Scholarship", "value": "CAD $2,000–$5,000"}],
  "last_updated": "2025-08-01"
}
```

### Australia — University of Melbourne

```json
{
  "name": "University of Melbourne",
  "country": "AU",
  "city": "Melbourne",
  "state": "VIC",
  "acceptance_rate_overall": 0.70,
  "acceptance_rate_intl": 0.65,
  "toefl_minimum": 79,
  "ielts_minimum": 6.5,
  "tuition_local_currency": 44736,
  "tuition_currency": "AUD",
  "tuition_usd_approx": 28200,
  "living_cost_local_currency": 22000,
  "living_cost_usd_approx": 13900,
  "total_cost_usd_approx": 42100,
  "strong_majors": ["Medicine", "Law", "Engineering", "Commerce", "Arts"],
  "ranking_qs_2026": 13,
  "application_platform_au": "vtac",
  "application_deadline_au": "2025-09-30",
  "ielts_minimum_overall": 6.5,
  "ielts_minimum_band": 6.0,
  "ielts_note": "No band below 6.0; Medicine requires 7.0 overall",
  "atar_equivalent_min": 80.0,
  "go8_member": 1,
  "coe_processing_days": 7,
  "oshc_required": 1,
  "oshc_approx_aud_per_year": 680,
  "oshc_providers": "Medibank, Bupa, Allianz, nib",
  "gte_notes": "Strong ties to Vietnam (family, property, career plan) expected. Academic motivation for chosen program required in SOP.",
  "student_visa_500_processing_weeks": 5,
  "visa_biometrics_required": 1,
  "graduate_visa_485_eligible": 1,
  "graduate_visa_485_years": 2,
  "scholarship_available": [
    {"name": "Melbourne International Undergraduate Scholarship", "value": "AUD $10,000/năm"},
    {"name": "Melbourne Research Scholarship", "value": "Toàn phần — bậc PhD"}
  ],
  "foundation_pathway_available": 1,
  "last_updated": "2025-08-01"
}
```
