---
name: sme-events
description: "Event lifecycle manager cho SME — tao event (workshop/webinar/networking/demo-day/conference-booth/internal-kickoff), checklist prep 1-2 ngay truoc, post-event actions (thank-you email qua sme-campaign + feedback survey). Hoat dong voi COSMO /v1/events + Luma (Phase 2) + Google Forms (Phase 3). Trigger khi user noi 'tao event', 'chuan bi event', 'sau event', 'danh gia event'."
metadata: { "openclaw": { "emoji": "📅" } }
---

# SME Events — Event Lifecycle Manager

## Trigger

Bat ky message nao co y:
- "tao event" / "dang ky event" / "set up event" — CREATE flow
- "chuan bi event X" / "event {ngay mai} can lam gi" — PREP flow
- "event X vua xong" / "sau event" / "post event" / "thank-you attendees" — POST flow
- "list event" / "event sap toi" / "event da qua" — LIST flow
- "danh gia event" / "feedback form event" — SURVEY flow (Phase 3)
- "import Luma event" / "sync Luma" — LUMA (Phase 2)

KHONG trigger:
- Nhac ai outreach hom nay → sme-reminder skill
- Search contact → sme-crm skill

## 6 EVENT TYPES

| ID | Emoji | Best for |
|---|---|---|
| `workshop` | 🎓 | Hands-on training, 10-30 nguoi |
| `webinar` | 💻 | Online, 50-500 nguoi, lead gen |
| `networking` | 🤝 | Gathering 30-100 nguoi, relationship |
| `demo-day` | 🎯 | Sales demo 5-10 prospect |
| `conference-booth` | 🏢 | Booth industry conf, lead capture rong |
| `internal-kickoff` | 🎬 | Kickoff project/quarter, 10-50 stakeholder |

Mỗi type co san: prep_tasks (5 items), day_of_tasks, post_tasks, survey_prompt.

---

## FLOW 1 — Create event

User noi "tao workshop AI day 15/5 o Rockship office":

1. Hoi thieu info:
   - Type (workshop / webinar / etc.)
   - Title
   - Date + time (ICT)
   - Venue (neu offline) or platform (neu online)
   - Capacity (target attendee)
2. Neu user chua biet type, present 6 options + best_for → user chon.
3. Neu info chua du (vd "tao event AI" khong co date):
   - **Recommend prep roadmap truoc** dua tren type dang nhac:
     - Workshop → "Can chot noi dung 2 tuan truoc, venue 1 tuan, marketing 10 ngay..."
     - Webinar → "Confirm speaker 2 tuan, slides 1 tuan, reminder email..."
   - KHONG bat buoc create event ngay; hoi them info.
4. Khi du info:
   ```bash
   sme-cli event create \
     --type workshop --title "AI Workshop" \
     --date "2026-05-15T14:00:00+07:00" \
     --venue "Rockship office" --capacity 25
   ```
5. Response: event_id + recommend chay `prep-checklist <event_id>` de xem task cu the.

**Neu user co Luma link**: pass `--luma-url https://lu.ma/...`. Bot luu vao
external_urls.luma_url cua event. (Phase 2 se co `sme-cli event sync-luma`
de auto-fetch attendees vao COSMO.)

---

## FLOW 2 — Prep checklist (1-2 ngay truoc event)

User noi "event X ngay mai can chuan bi gi":

1. Tim event_id (neu user chua provide):
   - `sme-cli event list --filter upcoming`
   - Match theo title user noi
2. Lay checklist:
   ```bash
   sme-cli event prep-checklist <event_id>
   ```
3. Render friendly:
   > 🎓 **Workshop "AI Workshop" — ngay mai 2pm o Rockship office**
   >
   > Checklist:
   > - [ ] Venue + AV: check mic, projector, extension cord
   > - [ ] Tai lieu: in handout, exercises, name tag
   > - [ ] Attendee list: confirm so luong + dietary preferences
   > - [ ] Facilitator brief: flow, timing per section, fallback plan
   > - [ ] Logistics: snack/drink, sign-in sheet
   >
   > Anh da lam cai nao roi? Em track giup.

4. User check-off tung task (manual conversation — bot theo doi qua
   interaction log hoac set event metadata.prep_done=[task_1, task_2]).

**Tich hop voi sme-reminder**: Moi sang 8am neu event trong 1-3 ngay,
reminder auto surface `EVENT_PREP_SOON` cell nhac task con thieu.

---

## FLOW 3 — Post-event actions

User noi "event X vua xong" hoac cron detect event.date < now va
thank_you_sent = false:

1. Lay suggestions:
   ```bash
   sme-cli event post-actions <event_id>
   ```
2. Output co:
   - `post_tasks`: danh sach action can lam
   - `campaign_handoff`: playbook + audience de trigger sme-campaign
   - `survey_handoff`: command cho Phase 3, fallback manual link
3. Render friendly + ASK permission:
   > 📮 **Event "AI Workshop" vua xong (hom qua). Can lam tiep:**
   >
   > 1. Gui thank-you email cho ~25 attendees (<24h). Playbook:
   >    `content_offering`. Em chuyen sang sme-campaign skill draft luon
   >    nhe?
   > 2. Tao feedback form qua Google Forms (hien manual — Phase 3 se
   >    auto-gen). Em soan 5 cau survey template, anh tu tao form?
   > 3. Log 25 attendees vao CRM voi tag `workshop_{event_id}`.
   > 4. Video edit + share social.
   >
   > Anh muon em lam cai nao truoc?

4. Neu user approve → **trigger sme-campaign skill** de thuc thi
   campaign. Su dung command:
   ```bash
   sme-cli cosmo api POST /v1/campaigns \
     '{"name":"Thank You — AI Workshop","playbook":"content_offering",
       "list_contact_id":"<event_attendee_list>","status":"draft"}'
   ```
   Sau do PATCH status=active de trigger sending.

5. Sau khi campaign tao, set event.metadata.thank_you_sent = true:
   ```bash
   sme-cli cosmo api PATCH /v1/events/<event_id> \
     '{"metadata":{"thank_you_sent":true,"thank_you_campaign_id":"<campaign_id>"}}'
   ```

---

## FLOW 4 — Abstract event (chua du info)

User noi "toi dinh to chuc webinar AI thang sau nhung chua biet lam gi":

KHONG bat buoc create event. Recommend prep roadmap tu type:

> 💻 Webinar chuan bi theo timeline sau (giup anh khoi scramble vao phut cuoi):
>
> **4 tuan truoc:**
> - Chot topic + speaker (neu external)
> - Dang ky platform (Zoom/GoToWebinar)
> - Tao landing page + registration form
>
> **3 tuan truoc:**
> - Publish announcement → email list + social
> - Tao outline slides
>
> **2 tuan truoc:**
> - Slides final, rehearsal lan 1
> - Reminder email lan 1 cho registered
>
> **1 tuan truoc:**
> - Rehearsal lan 2 + Q&A prep
> - Reminder email lan 2
> - Backup host brief
>
> **1 ngay truoc:**
> - Test platform, audio, demo
> - Reminder email lan 3
>
> **Hom event:**
> - Join 30p truoc, record, monitor chat
>
> **Sau event:**
> - Recording + slides email (<24h)
> - Feedback form (em soan 5 cau)
> - Nurture campaign (playbook webinar_follow_up)
>
> Anh muon em tao event entry voi target date truoc, hay van dang nghi?

---

## INTEGRATION VOI skills khac

- **sme-reminder**: daily-plan auto include event signals:
  - `EVENT_PREP_SOON` (event 1-3 ngay toi)
  - `EVENT_POSTMORTEM` (event <3 ngay truoc, chua gui thank-you)
- **sme-campaign**: trigger khi user approve post-event email. Pass
  playbook + list_contact_id cua attendee list.
- **sme-crm**: log attendees sau event (tag, interaction, business_stage).

---

## QUY TAC (plain language, giong sme-reminder)

| CAM dung | Noi the nay |
|---|---|
| "event_type_id" | "loai event" |
| "list_contact_id" | "danh sach khach hang" |
| "playbook" | "kich ban email" / "mau email" |
| "metadata.thank_you_sent" | (khong noi — check silent) |
| "campaign_handoff" | "em chuyen sang skill gui email" |

---

## CLI commands

```bash
sme-cli event types                              # list 6 types + checklist
sme-cli event list [--filter upcoming|recent]    # fetch events
sme-cli event create --type X --title Y --date Z # tao event
sme-cli event prep-checklist <event_id>          # tasks 1-2 ngay truoc
sme-cli event post-actions <event_id>            # thank-you + survey suggestions
sme-cli event sync-luma                          # Phase 2 (Luma API)
sme-cli event create-survey <event_id>           # Phase 3 (Google Forms API)
```
