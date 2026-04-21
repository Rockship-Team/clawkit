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

## FLOW 1 — Create event (BIAS TO ACTION — khong phong van user)

User noi "tao workshop AI day 15/5 o Rockship office cho 25 nguoi":

### Required fields de CREATE

Chi co 5 field can de tao event:

| Field | Default neu user khong noi | Ask 1 cau neu thieu |
|---|---|---|
| type | — | Hoi: "Day la workshop, webinar, networking, demo-day, conference-booth, hay internal-kickoff?" |
| title | Suy ra tu context (vd "AI Workshop") | Khong hoi — tu dat |
| date | Parse tu context | Hoi: "Ngay may anh muon? Neu da chot 15/5, cho em gio bat dau (vd 14:00)?" |
| venue | Neu user noi "online" → "Online"; khong thi hoi | Hoi: "O dau anh? Office / external venue / online?" |
| capacity | Khong noi → khong set (optional) | Khong hoi — co the bo qua |

### NICE-TO-HAVE — **TUYET DOI KHONG hoi o tao event**

De user update sau qua edit command hoac manually. Cac field nay KHONG
lam skill blocked:
- Chu de / agenda chi tiet
- Budget
- External speakers
- Equipment / AV requirements cu the
- Marketing channels
- Guest list VIP
- Dress code / dietary

**Ly do**: user da bao "workshop AI" la du context. Chi tiet agenda
lam sau. Hoi 5 cau cung 1 luc = cv consultant, KHONG phai BD coach.

### Flow cu the

**Case 1 — User cho du 4-5 required**:

User: "tao workshop AI ngay 15/5 o Rockship office cho 25 nguoi"

Bot suy ra:
- Type: workshop (explicit)
- Title: "AI Workshop" (tu "workshop AI")
- Venue: "Rockship office" (explicit)
- Capacity: 25 (explicit)
- Date: 15/5 → **missing time + year**. Default year = nam hien tai (2026).
  Default time = 14:00 ICT (office hours, after lunch).

**Bot chi hoi 1 cau**:
> "OK, em ghi nhan workshop AI ngay 15/5/2026 o Rockship office cho 25
> nguoi. Gio bat dau em de mac dinh 2 gio chieu, anh co can doi khac
> khong? (vd 9am, 1pm, 3pm...)"

Sau khi user confirm hoac doi gio, chay CREATE ngay.

**Case 2 — User abstract ("toi dinh to chuc webinar AI thang sau")**:

KHONG bat buoc create. Dung flow 4 (prep roadmap theo type).

**Case 3 — User thieu date**:

> "Workshop AI nghe hay! Anh dinh to chuc ngay bao nhieu? Em can biet
> de lap prep checklist. Neu chua chot duoc, em co the gui roadmap
> chuan bi theo timeline 4 tuan / 3 tuan / 2 tuan / 1 tuan truoc."

### CREATE command

```bash
sme-cli event create \
  --type workshop --title "AI Workshop" \
  --date "2026-05-15T14:00:00+07:00" \
  --venue "Rockship office" --capacity 25
```

Response → bot noi (plain Vietnamese, tu giai thich tai sao can):
> ✅ Xong! Em tao event "AI Workshop" 15/5/2026 2pm o Rockship office
> cho 25 nguoi.
>
> Tuan sau co workshop — em co danh sach viec can chuan bi truoc
> (venue/AV, in tai lieu, confirm attendee, brief facilitator, snack).
> Thuong xong truoc 1-2 ngay la chay om. Muon em list ra cho anh
> track luon, hay de sau?

Khi user OK → chay `sme-cli event prep-checklist <event_id>` va render
tung task ro rang:
> Checklist cho workshop 15/5:
>
> 1. **Venue + AV** — kiem tra mic, projector, o cam dien
> 2. **Tai lieu** — in handout, exercises, name tag cho 25 nguoi
> 3. **Attendee list** — confirm so nguoi den + dietary preferences
> 4. **Facilitator brief** — ai dan workshop, flow, backup plan
> 5. **Logistics** — snack, nuoc, sign-in sheet
>
> Anh tick dan tung cai khi lam xong nhe — em co the nhac lai ngay
> truoc workshop.

**QUY TAC**: khi bot offer tinh nang, phai kem 1 cau giai thich TAI SAO
cai do co ich. KHONG hoi "muon X khong?" trong (user se confused).
Hoi kieu "co X, giup anh Y, muon em lam khong?".

### Luma link (optional)

Neu user co link Luma: pass `--luma-url https://lu.ma/...`. Bot luu vao
external_urls.luma_url. (Phase 2 se co `sme-cli event sync-luma` de
auto-fetch attendees vao CRM.)

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
