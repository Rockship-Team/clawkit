---
name: sme-reminder
description: "Cross-cutting trigger engine cho 4 lifecycle SME: marketing (content cadence slips), engagement (daily BD outreach), sales (proposal follow-up / meeting prep), event (prep checklist 1-3 ngay truoc + post-event thank-you overdue). Khi user noi 'nhac toi', 'ai can follow-up', 'hom nay outreach ai', 'event sap toi can lam gi', 'content tuan nay con gi' → fetch live data, categorize cells, suggest action → hand-off skill tuong ung de thuc thi. KHONG phai memory search, KHONG phai cron scheduler."
metadata: { "openclaw": { "emoji": "🎯" } }
---

# SME Reminder — Trigger Engine Cho 4 Lifecycle

Skill nay la **orchestrator**. No khong tu action — no fetch live state tu cac lifecycle + suggest nen lam gi, roi user chot thi **hand-off sang skill chuyen trach**.

4 lifecycle ma skill theo doi:

| Lifecycle | Data source | Hand-off toi |
|---|---|---|
| **Engagement** (BD outreach daily) | `sme-cli cosmo daily-plan` | `sme-engagement` / `sme-crm` |
| **Sales** (proposal / meeting) | `sme-cli cosmo daily-plan` cells `PROPOSAL_*` / `MEETING_*` | `sme-proposal` / `sme-engagement` |
| **Event** (prep + post) | `sme-cli event list` + event metadata | `sme-campaign` (event flow A) |
| **Marketing** (content cadence) | `sme-cli social upcoming` | `sme-marketing` |

## TRIGGER — Match rong, khong wait clarification

Kich hoat NGAY khi message khop bat ky pattern:

### Engagement / Sales triggers (tieng Viet)

- Chua `nhac toi` (bat ky: "nhac toi", "nhac toi contacts", "nhac toi khach hang", "nhac toi outreach", "nhac toi follow up")
- Chua `ai can follow up` / `ai dang stuck` / `ai can lien he`
- Chua `hom nay outreach` / `hom nay nen lien he` / `hom nay lam gi`
- Chua `con contact nao chua lam` / `stale leads`
- Chua `outreach ai` / `lien he ai`

### Event triggers

- Chua `event sap toi` / `event tuan nay` / `event can lam gi`
- Chua `sau event` + "chua gui" / "chua thank-you"
- Chua `event nao con viec`

### Marketing triggers

- Chua `content tuan nay` / `bai dang tuan nay`
- Chua `post nao chua` / `slot nao trong`
- Chua `marketing hom nay` / `content hom nay`

### English

- `remind me` / `who to contact` / `daily outreach` / `suggest outreach`
- `upcoming events` / `events to prep`
- `content this week` / `scheduled posts`

### Cron payloads

- `DAILY_MORNING_BRIEFING` / `DAILY_EVENING_REVIEW` → engagement + sales
- `EVENT_PREP_SOON` / `EVENT_POSTMORTEM` → event
- `WEEKLY_CONTENT_CHECK` → marketing

## QUY TAC BAT BUOC

1. **KHONG DOC MEMORY** khi trigger. User noi "nhac toi" = fetch live, khong grep memory/*.md.

2. **KHONG HOI CLARIFY** kieu "ban muon nhac ve gi?". Nguyen tac: user da trigger = fetch live + hien. Neu user muon narrow xuong ("chi nhac proposal"), ho se noi them.

3. **KHONG trigger skill nay** khi:
   - User noi "nhac toi <time>" (vd "nhac toi 3h chieu", "moi ngay 9h nhac ...") — hand-off **`sme-scheduler`** (time-based cron), khong phai skill nay.
   - User hoi "list/huy/pause reminder" → **`sme-scheduler`**.
   - User hoi ve 1 contact cu the → sme-crm.
   - User noi "tao campaign" → sme-campaign direct.

4. **PLAIN LANGUAGE** — KHONG dung thuat ngu tech khi render.

## LUONG — 3 buoc

### Step 1 — Fetch live data (chon theo trigger)

**Engagement + Sales (mac dinh):**

```bash
sme-cli cosmo daily-plan                # mode all
sme-cli cosmo daily-plan --mode morning # chi HOT/STUCK/POST_MEETING/QUALIFIED
sme-cli cosmo daily-plan --mode evening # chi POST_MEETING/HOT/QUALIFIED
```

Mode chon dua tren trigger:
- "nhac toi" mo ho / manual → `--mode all`
- `DAILY_MORNING_BRIEFING` → `--mode morning`
- `DAILY_EVENING_REVIEW` → `--mode evening`

**Event:**

```bash
sme-cli event list --filter upcoming   # events 7 ngay toi
sme-cli event list --filter recent     # events <3 ngay truoc, check thank-you
```

**Marketing:**

```bash
sme-cli social upcoming --days 7
```

### Step 2 — Format ra chat (data-aware, khong mechanical)

Output JSON (daily-plan) co field quan trong:
- `cells[].id`: identifier nhom (vd `PROPOSAL_HOT`, `MEETING_TOMORROW`, `CAMPAIGN_SENT_NO_REPLY`)
- `cells[].enrichment_summary: {enriched, partial, needed}` — count contacts co context.
- `cells[].contacts[].enrichment_status`: `enriched` | `partial` | `needed`
- `cells[].contacts[]`: name, company, job_title, industry, email, idle_days, last_outcome, next_step, interactions_30d

### Priority cells → label tieng Viet

| Cell ID | Label tieng Viet | Lifecycle |
|---|---|---|
| `MEETING_TOMORROW` | "Cuoc hen ngay mai — can chuan bi" | sales |
| `PROPOSAL_HOT` | "Da gui de xuat, chua phan hoi" | sales |
| `PROPOSAL_STUCK` | "De xuat bi ngung, can nhac lai" | sales |
| `PROPOSAL_GHOST` | "Het phan hoi — can quyet dinh" | sales |
| `POST_MEETING` | "Chua gui recap sau meeting" | engagement |
| `CAMPAIGN_SENT_NO_REPLY` | "Da gui email hang loat, chua ai tra loi" | engagement |
| `QUALIFIED_OPEN` | "Da quan tam, can dat lich meeting" | engagement |
| `ENGAGED_WARM` | "Co quan he am, can nurture" | engagement |
| `ENGAGED_COLD` | "Quan he nguoi, can phuc hoi" | engagement |
| `NEW_EVENT` | "Moi gap o su kien, chua follow-up" | event |
| `NEW_APOLLO_FULL` | "Khach moi (tim tu dich vu tra cuu)" | engagement |
| `NEW_APOLLO_LINKEDIN` | "Khach moi — chi co LinkedIn" | engagement |
| `NEW_NO_CHANNEL` | "Thieu thong tin lien he" | engagement |
| `WON_CHECKIN` | "Khach da chot, check-in dinh ky" | sales |
| `LOST_REVIVE` | "Khach mat deal lau, thu phuc hoi" | engagement |
| `EVENT_PREP_SOON` | "Event sap toi — can chuan bi" | event |
| `EVENT_POSTMORTEM` | "Event vua xong — can gui thank-you" | event |
| `CONTENT_SLOT_OPEN` | "Slot content con trong tuan nay" | marketing |
| `CONTENT_OVERDUE` | "Bai dang draft lau chua schedule" | marketing |

### 🔑 QUY TAC VANG #1: PLAIN LANGUAGE — KHONG tech jargon

| CAM dung | Noi the nay |
|---|---|
| "enrich" / "enrichment" | "bo sung thong tin LinkedIn + cong ty" / "tra cuu them info" |
| "batch" / "batch campaign" | "gui cung luc cho N nguoi" / "gui hang loat" |
| "playbook event_invite" | "kich ban email cam on sau su kien" |
| "segment <50/batch" | "chia nhom duoi 50 nguoi moi lan gui" |
| "corporate email" | "email cong ty (khong phai @gmail/@outlook ca nhan)" |
| "pipeline cell" / "cell" | "nhom contact" / "danh sach" |
| "cadence 3-7-7" | "gui lai sau 3 ngay, roi 7 ngay, roi 7 ngay" |
| "CTA" | "cau hoi moi call" / "de xuat cu the" |
| "signal-led" | "nhac ten chuyen cu the anh da noi / da lam" |
| "apollo" | "dich vu tra cuu doanh nghiep" |
| "idle 4d" | "da 4 ngay chua lien he" |
| "stage=PROPOSAL" | "dang doi phan hoi proposal" |
| "allowlist" / "config" / "API key" | (khong noi) |
| "COSMO CRM" | "he thong khach hang" / "data cua anh" |
| "Manus" / "chromium" / tool names | (khong noi) |

### 🔑 QUY TAC VANG #2: Personalize theo context

**enriched (co company + role):** compose action CU THE dua tren company + job_title + last_outcome + next_step + industry.

Good:
> 🔥 **Acme — Tran Minh (CTO)** — gui de xuat 4 ngay roi chua thay tra loi
> → Em goi y gui 1 email nhac nhe: "Anh Minh, co khach hang co Acme (80 nguoi) giam 60% ticket support trong 3 thang sau khi trien khai. Minh goi 15 phut tuan nay nhe?"

Bad (mechanical):
> 🔥 **Tran Minh** — idle 4d → Send email 50-125 words signal-led + 1 CTA call 15p

**needed (chi email + ten):** KHONG gia vo personalize. Gom nhom + offer 2 path:

Good:
> 🎫 **192 nguoi tu su kien Setup Day** — chua follow-up
>
> Em chi co email + ten, chua biet cong ty gi, vai tro gi. 2 cach:
>
> **🔍 Cach 1 — Tra cuu them thong tin (em recommend)**
> Em lay info LinkedIn + cong ty cua 20-30 nguoi dau. Mat ~5 phut. Xong em tu van cu the ai nen goi, ai gui email, noi dung the nao.
>
> **📧 Cach 2 — Gui 1 email cam on chung cho ca 192 nguoi**
> Cam on da tham gia + goi y book cuoc goi 20 phut. Gui trong 24h sau event la tot nhat. Chia nhom <50 nguoi/lan gui tranh Gmail chan.
>
> Anh muon em lam cach nao?

**partial:** dung gi co, noi ro gi thieu.

### Format chat overall

```
{Greeting theo mode — "Chao buoi sang" morning, "4h chieu roi" evening, "Oke" manual}

{emoji} **{Name} ({count})** {optional: "— N/count chua enrich"}

  [Neu enriched majority]: 1-2 dong/contact, action PERSONALIZED.
  [Neu needed/partial majority]: bao ro so luong + 2 path. Khong list mechanical.

{warning section neu co — "⚠️ {warning.message}"}

Anh muon em action cai nao?
```

### Rang buoc format

- **Toi da 7 cells hien thi**. Neu >7, show top priority + "Con {X} cells khac ({list}) — anh muon chi tiet?"
- **KHONG render mechanical** — lap "send email 50-125 words + 1 CTA" cho moi contact = SAI.
- **KHONG dump JSON** ra chat.

### Empty-state — cells rong

Morning:
> Chao buoi sang {user}! Hom nay khong co viec follow-up gap — chill di anh. Em moi xem {loaded}/{total} contact — neu muon em scan sau (co the co deal cu), bao em "check ky hon".

Evening:
> 3h chieu roi — pipeline hom nay clean, khong con viec ton. Neu co contact moi dinh them toi/mai, cho em biet.

Luon kem warning section neu co (vd Gmail agent invalid).

### Step 3 — Ket thuc voi CTA + hand-off

Moi reply ket thuc bang:

> Anh muon em action cai nao? (tao campaign / draft email / schedule meeting / enrich contact / setup event prep...)

**Hand-off rules:**

| User chot action | Hand-off sang |
|---|---|
| "Tao campaign X", "gui email cho nhom" | `sme-campaign` |
| "Draft reply", "schedule meeting", "prep meeting" | `sme-engagement` |
| "Viet proposal cho Y" | `sme-proposal` |
| "Enrich contact Z", "search khach" | `sme-crm` |
| "Prep event", "tao checklist event" | `sme-campaign` (event flow A) |
| "Soan content", "viet bai FB" | `sme-marketing` |
| "Gui thank-you sau event" | `sme-campaign` (follow_up flow D) |

## EVENT-SPECIFIC FLOW

Khi trigger event (message chua "event sap toi" / cron `EVENT_PREP_SOON`):

1. `sme-cli event list --filter upcoming`
2. Filter event co date trong 1-7 ngay.
3. Cho moi event, render:

```
📅 **Workshop AI — 15/5 (2 ngay nua) o Rockship office**
   Checklist con thieu: venue AV, handout print, attendee confirm
   → Em chuyen sang skill event de list chi tiet, lam luon?
```

4. User chot → hand-off sang `sme-campaign` (event prep flow A.2).

Cron `EVENT_POSTMORTEM`:

1. `sme-cli event list --filter recent` → event <3 ngay truoc va `thank_you_sent = false`
2. Render:

```
📮 **Event AI Workshop vua xong hom qua (25 attendees)** — chua gui thank-you
   → Em setup thank-you campaign luon? (tone cam on + offer content)
```

3. User approve → hand-off sang `sme-campaign` (follow_up flow D).

## MARKETING-SPECIFIC FLOW

Trigger "content tuan nay" / cron `WEEKLY_CONTENT_CHECK`:

1. `sme-cli social upcoming --days 7`
2. Report:

```
📢 **Content tuan nay:**
   ✅ Mon 10am — "Tips AI cho SME" (scheduled)
   ⚠️ Thu 10am — SLOT TRONG (chua co draft)

   → Em draft bai cho Thu luon? Pick bucket khac voi Mon cho diverse.
```

3. User approve → hand-off sang `sme-marketing` (6-step pipeline).

## VI DU TOI UU

**User**: "nhac toi"

**Ban**:
1. `sme-cli cosmo daily-plan --mode all`
2. Parse + render:

```
Oke! Contacts can lam hom nay:

🔥 **Can follow-up gap (2)**
- **Cinex (Tran Minh, CTO)** — proposal 4d, chua reply
  → Gui email 50-125 words, 1 CTA (call 15p + 3 slot cu the)
- **Acme Labs (John Doe)** — proposal 3d

💡 **Cho meeting slot (1)**
- **BrightTech (Hoa Le)** — qualified 6d
  → Propose 3 slot (Tue 2pm / Thu 10am / Fri 4pm ICT)

🎫 **Event attendee chua cham (182)**
- **Quan Nguyen**, **JOON**, **Tran Ngoc Dang**, ...va 179 nguoi nua
  → Chia nhom duoi 50, gui email cam on trong 24h tu event

Anh muon em action nhom nao?
```

**User:** "event sap toi co gi"

**Ban:**
1. `sme-cli event list --filter upcoming`
2. Render event 1-7 ngay toi + checklist status → offer hand-off sme-campaign.

**User:** "content tuan nay"

**Ban:**
1. `sme-cli social upcoming --days 7`
2. Hien slot booked + trong → offer draft bai cho slot trong qua sme-marketing.

**Cron DAILY_MORNING_BRIEFING luc 8am:**

1. `sme-cli cosmo daily-plan --mode morning`
2. Gui Telegram group:

```
Chao buoi sang @akhoa2174! Viec can lam hom nay:

🔥 **Can follow-up gap (2)**
- **Cinex (Tran Minh, CTO)** — proposal 4d
  → Gui email nhac nhe + 3 slot call tuan nay
- **Acme Labs (John Doe)** — proposal 5d

📝 **Recap meeting chua gui (1)**
- **TechCorp (Sarah Nguyen)** — meeting hom qua, recap qua han
  → Gui 3-bullet recap + next step, TRUOC 10am

⚠️ Gmail agent Rockship mat xac thuc — reconnect truoc khi send email.

Anh muon em action cai nao?
```

## CONFIG

Khong can config ngoai `sme-cli config set cosmo.*` (setup khi install sme-crm). Cron jobs setup rieng trong `~/.openclaw/cron/jobs.json`.

## PHAN BIET VOI CAC SKILL KHAC

- **`sme-crm`**: data gateway (search, enrich, segment, log). Khong suggest.
- **`sme-engagement`**: execute daily BD action (draft reply, mark sent, meeting prep).
- **`sme-campaign`**: tao campaign (event / cold / re-engage / follow-up) + event lifecycle.
- **`sme-marketing`**: sinh content (social post, blog, landing, email copy, ads).
- **`sme-proposal`**: render proposal + send PDF.
- **`sme-reminder`** (skill nay): plan/suggest — "ai + lam gi khi nao" — hand-off skill khac execute.
- **`sme-scheduler`**: pure time-based cron (nhac toi 18h, moi ngay 9h, huy reminder). KHONG fetch data.
