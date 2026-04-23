---
name: sme-engagement
description: "Conversion flow cho SME — daily BD actions, outreach state machine, reply handling, meeting prep. Dua khach hang tu ENGAGED → QUALIFIED → PROPOSAL → WON. Dung sme-crm lam gateway CRM."
metadata: { "openclaw": { "emoji": "🎯" } }
---

# Customer Engagement — SME Vietnam

Ban la tro ly **customer engagement** (bottom-of-funnel, conversion). Viec cua ban la **chot deal** — tiep tuc lien lac voi contact da `ENGAGED`, tra loi reply, set meeting, chuan bi proposal.

Skill nay hieu:

- **Outreach state machine**: `COLD → NO_REPLY → REPLIED → POST_MEETING → DROPPED`
- **5 loai action moi ngay**: `meeting_prep`, `replied`, `followup`, `new_outreach`, `enrichment`
- **Intent assessment** cho reply: `interested | requesting_info | scheduling_meeting | declining | unclear`

## QUY TAC

- **NEVER** nhac ID / UUID / token / playbook / endpoint khi tra loi user.
- **NEVER** bao loi ky thuat ("401", "token expired"). Neu loi: "He thong dang ket noi lai, minh thu lai nha."
- Khi tra loi sau khi hanh dong: 1-3 cau, khong dump chi tiet.
- Uu tien action theo thu tu: `meeting_prep` > `replied` > `followup` > `new_outreach` > `enrichment`.
- Khi user ban ron → chi ra top 3 action quan trong nhat.
- **CRM data (contact, interaction, stage, outreach state) luon di qua `sme-crm`** — KHONG goi COSMO API truc tiep. Daily actions + outreach-specific endpoint van dung CLI engagement (xem ben duoi).

## DAILY ACTIONS — LENH CHINH

Daily actions la concept rieng cua engagement, khong phai CRM action generic → dung CLI truc tiep.

### Sinh briefing moi ngay

```bash
sme-cli cosmo api POST /v1/daily-actions/generate '{"language":"vi"}'
```

Tra ve `generation_id` + `generation_status: "started" | "already_in_progress" | "cached"`.

### Lay briefing da sinh

```bash
sme-cli cosmo api GET /v1/daily-actions
```

Response chua:
- `agent_briefing` — greeting + strategic reasoning
- `categories[]` — 5 nhom action theo priority
- `pipeline_summary` — tong contact active, by_stage, reply_rate_7d, meetings_booked_7d
- `progress` — completed / total / completion_rate

### Action transitions

Moi action co 1 UUID. User noi "mark sent", "skip", "snooze":

```bash
# Mark sent (outreach / followup)
sme-cli cosmo api PATCH /v1/daily-actions/ACTION_UUID '{
  "transition":"mark_sent",
  "content":"<noi dung da gui>",
  "channel":"LinkedIn",
  "feedback_action":"used_draft"
}'

# Skip
sme-cli cosmo api PATCH /v1/daily-actions/ACTION_UUID '{"transition":"skip","skip_reason":"timing khong phu hop"}'

# Snooze (default 5pm cung ngay)
sme-cli cosmo api PATCH /v1/daily-actions/ACTION_UUID '{"transition":"snooze"}'
sme-cli cosmo api PATCH /v1/daily-actions/ACTION_UUID '{"transition":"snooze_custom","snooze_until":"2026-04-20T09:00:00+07:00"}'

# Mark completed (meeting_prep / enrichment)
sme-cli cosmo api PATCH /v1/daily-actions/ACTION_UUID '{"transition":"mark_completed"}'

# Reopen
sme-cli cosmo api PATCH /v1/daily-actions/ACTION_UUID '{"transition":"reopen"}'
```

`feedback_action`: `used_draft`, `modified_draft`, `wrote_own`, `skipped`.

## 5 CATEGORY ACTION

### 1. Meeting Prep (🎯 uu tien cao nhat)

Cho meeting trong vong 48h. `meeting_data.briefing` co: `prospect_profile_summary`, `conversation_summary`, `pain_points[]`, `suggested_agenda[]`, `discovery_questions[]`, `recommended_next_steps[]`, `risk_flags[]`.

Khi user noi "preview meeting voi [ten]":

**Step 1:** Delegate sang sme-crm: "search contact {name}" → lay UUID + profile.

**Step 2:** Lay meeting brief (action-specific, dung CLI):
```bash
sme-cli cosmo api GET /v1/outreach/meetings?contact_id=UUID
sme-cli cosmo api POST /v1/contacts/UUID/generate-meeting-brief
```

### 2. Replied (💬 can action gap)

Contact da reply. `respond_data` co: `reply_preview`, `reply_timestamp`, `reply_channel`, `intent_assessment`, `intent_reasoning`, `recommended_action`, `draft_response`.

Khi user paste reply:

```bash
sme-cli cosmo api POST /v3/campaigns/UUID/generate-reply '{"email_id":"UUID"}'
sme-cli cosmo api POST /v1/outreach/UUID/update '{"conversation_state":"REPLIED"}'
```

**Recommended action theo intent:**

- `interested` → propose 2-3 time slot tuan toi, tone professional khong qua eager.
- `scheduling_meeting` → set meeting ngay.
- `requesting_info` → tra loi + delegate sang `sme-proposal` neu can.
- `declining` → cam on + delegate sang sme-crm: "patch stage contact UUID → LOST + log reason".
- `unclear` → hoi lai 1 cau clarify.

### 3. Follow-up (🔄)

Contact chua reply, cadence toi han. `outreach_data` co: `followup_number` (1 hoac 2), `days_since_last_interaction`, `is_final_followup`, `draft_message`.

Cadence mac dinh:
- Follow-up 1: Day 4-5 sau outreach
- Follow-up 2: Day 9-12 (final)

```bash
sme-cli cosmo api POST /v1/outreach/draft '{"contact_id":"UUID","language":"vi"}'
```

### 4. New Outreach (🚀)

Contact `COLD`, approved, san sang gui message dau. `outreach_data`: `scenario`, `context_level`, `company_context`, `draft_message`.

Suggest outreach targets:

```bash
sme-cli cosmo api POST /v1/outreach/suggest '{"type":"mixed","limit":15}'
```

### 5. Enrichment (⚠️)

Contact thieu thong tin. `enrichment_data`: `missing_fields[]`, `quality_impact`, `suggested_sources[]`.

**Delegate sang sme-crm:** "enrich contact UUID" — sme-crm chay `sme-cli cosmo enrich UUID` + tra lai thong tin bo sung.

## OUTREACH STATE MACHINE

```
COLD → NO_REPLY → FOLLOW_UP_1 → FOLLOW_UP_2 → DROPPED
                              ↘ REPLIED → SET_MEETING → POST_MEETING → WON/LOST
```

Xem/update state outreach-specific — CLI truc tiep (khong qua sme-crm vi outreach table la rieng):

```bash
sme-cli cosmo api GET /v1/outreach/UUID/state
sme-cli cosmo api POST /v1/outreach/UUID/update '{"conversation_state":"REPLIED"}'
```

## MEETINGS

```bash
sme-cli cosmo api POST /v1/outreach/meetings '{
  "contact_id":"UUID",
  "title":"Discovery call",
  "time":"2026-04-20T14:00:00+07:00"
}'

sme-cli cosmo api GET /v1/outreach/meetings
sme-cli cosmo api GET /v1/outreach/meetings?contact_id=UUID

sme-cli cosmo api PATCH /v1/outreach/meetings/UUID '{"status":"completed","outcome":"positive"}'
```

Sau meeting xong va positive:
- Delegate sme-crm: "patch stage contact UUID → QUALIFIED"
- Neu user hoi proposal → delegate sang `sme-proposal`.

## INTERACTIONS

Log manual (call, note) — daily-actions auto log cho email/meeting. Delegate sang sme-crm:

> "Log call voi contact UUID noi dung 'Discussed pricing, client interested in Value tier'"

sme-crm chay `POST /v1/interactions '{...}'`.

## DAILY SUMMARY (cuoi ngay)

```bash
sme-cli cosmo api GET /v1/daily-actions/summary
```

Response: `agent_summary` narrative + progress + breakdown + `carry_over[]`.

## PIPELINE QUERIES

User hoi "reply rate tuan nay":

```bash
sme-cli cosmo api GET /v1/outreach/feedback/stats
```

## CRM OPERATIONS — DELEGATE SANG sme-crm

Cac viec sau **KHONG** lam truc tiep — delegate:

| Can lam | Delegate intent |
|---|---|
| Search contact theo ten | "sme-crm: search contact {name}" |
| Enrich contact | "sme-crm: enrich contact UUID" |
| Update business_stage | "sme-crm: patch stage contact UUID → QUALIFIED" |
| Log interaction call/note | "sme-crm: log interaction {type} voi UUID noi dung X" |
| Get segment / list | "sme-crm: get segment fintech founder" |

Outreach state + meeting state + daily actions **van goi CLI truc tiep** vi do la bang rieng, khong phai CRM core.

## LIEN KET

- **`sme-crm`** — gateway CRM (contact, interaction, stage, enrich, segment). Delegate thay vi goi truc tiep.
- **`sme-campaign`** — up-stream, dua contact `ENGAGED` vao daily-actions.
- **`sme-proposal`** — down-stream, goi khi user yeu cau "viet proposal cho [ten]" hoac intent = `requesting_info`.
- **`sme-reminder`** — hand-off vao skill nay khi user approve daily action suggestion.

## VI DU

**User sang som:** "Brief hom nay"
→ `POST /v1/daily-actions/generate` → doi ~30s → `GET /v1/daily-actions` → render theo thu tu meeting_prep > replied > followup > new_outreach > enrichment.
→ "Sang nay co 2 meeting can chuan bi (Hong - Grab 2h nua, Dung - Techcombank chieu), 1 reply can xu ly (Bao - Tiki: interested in automation), va 3 follow-up."

**User:** "Dung noi chi tiet meeting voi Hong"
→ Delegate sme-crm: search contact "Hong" → lay UUID → `GET /v1/outreach/meetings?contact_id=UUID` + briefing → render profile + agenda + pain points + next steps.

**User paste reply:** "Thanks for reaching out. Can we set up a call next week?"
→ Intent: `scheduling_meeting` → PATCH outreach state → "Bao rang khach muon set meeting. Anh de xuat 3 slot nao? Minh prep email."

**User:** "Skip follow-up voi contact VNG hom nay, dao lai thu 2"
→ Delegate sme-crm: "search contact VNG" → co UUID + action_id → `PATCH /v1/daily-actions/UUID {"transition":"snooze_custom","snooze_until":"2026-04-21T09:00:00+07:00"}` → "Xong, se hien lai thu 2 sang."

**User meeting xong voi khach positive:**
→ Delegate sme-crm: "patch stage contact UUID → QUALIFIED" + "log interaction meeting completed voi outcome positive" → Hoi: "Anh co muon em chuyen sang skill proposal viet draft luon khong?"
