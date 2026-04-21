---
name: sme-reminder
description: "Daily BD outreach coach — khi user noi 'nhac toi', 'nhac toi contacts', 'ai can follow-up', 'hom nay outreach ai', 'daily outreach', 'remind me' → fetch contacts tu COSMO, categorize vao pipeline cells, suggest action cu the. Ke ca user hoi ngan gon ('nhac toi contacts', 'nhac toi khach hang') van phai trigger. KHONG phai memory search, KHONG phai cron scheduler."
metadata: { "openclaw": { "emoji": "🎯" } }
---

# SME Reminder — Ai Can Outreach Hom Nay, Va LAM GI

## Trigger — MATCH RONG, KHONG WAIT CLARIFICATION

Kich hoat skill nay NGAY khi user message khop bat ky pattern:

**Tieng Viet:**
- Chua tu `nhac toi` (bat ky ket hop: "nhac toi", "nhac toi contacts",
  "nhac toi khach hang", "nhac toi outreach", "nhac toi follow up")
- Chua `ai can follow up` / `ai dang stuck` / `ai can lien he`
- Chua `hom nay outreach` / `hom nay nen lien he` / `hom nay lam gi`
- Chua `con contact nao chua lam` / `stale leads`
- Chua `outreach ai` / `lien he ai`

**English:**
- Contains `remind me` / `who to contact` / `daily outreach` /
  `suggest outreach` / `list stale leads` / `outreach reminder`

**Cron:**
- Payload chua `DAILY_MORNING_BRIEFING` / `DAILY_EVENING_REVIEW`

### QUY TAC BAT BUOC

1. **KHONG DOC MEMORY** khi trigger. User noi "nhac toi contacts" =
   fetch COSMO live, khong phai grep trong memory/*.md. Memory co the
   co noi dung cu — KHONG relevant.

2. **KHONG HOI CLARIFY** kieu "ban muon nhac ve gi?". Nguyen tac: user
   da trigger = fetch live + hien ket qua. Neu user muon narrow xuong
   (vd "chi nhac proposal hot"), ho se noi them o reply sau.

3. **KHONG trigger skill nay** khi:
   - User noi "nhac toi <time>" (vd "nhac toi 3h chieu", "nhac toi
     mai 9am") — do la cron scheduler request, khong phai skill nay.
   - User hoi ve 1 contact cu the ("contact X la ai?") — do la CRM
     search, chuyen sang sme-crm skill.

## LUONG

### Step 1 — Goi CLI de lay du lieu da categorize

```bash
# Mode "all" — default, tat ca cell
sme-cli cosmo daily-plan

# Mode "morning" — chi HOT/STUCK/POST_MEETING/QUALIFIED (urgent today)
sme-cli cosmo daily-plan --mode morning

# Mode "evening" — chi POST_MEETING/HOT/QUALIFIED (ton lai hom nay + prep mai)
sme-cli cosmo daily-plan --mode evening
```

Output la JSON structured. Khong can tu categorize. Chon mode dua tren
trigger:
- User noi "nhac toi" mo ho → `--mode all`
- Cron DAILY_MORNING_BRIEFING → `--mode morning`
- Cron DAILY_EVENING_REVIEW → `--mode evening`

### Step 2 — Format ra chat (DATA-AWARE, khong lap template)

Output JSON co field quan trong:
- `cells[].enrichment_summary: {enriched, partial, needed}` — count contacts
  co business context day du vs chi co email/ten.
- `cells[].contacts[].enrichment_status`: `"enriched"` | `"partial"` | `"needed"`.
- `cells[].contacts[]`: name, company, job_title, industry, email, idle_days,
  last_outcome, next_step, interactions_30d, ...

### 🔑 QUY TAC VANG: personalize theo context co san

Khi render moi contact, ap dung nguyen tac sau:

**Neu contact `enrichment_status = "enriched"` (co company + role):**

Compose action **CU THE** dua tren:
- Company + job_title (vd "anh Tuan — CTO Acme")
- Last_outcome / next_step neu co (vd "proposal gui 4 ngay, chua reply")
- Industry / pain point (infer tu company)

Vi du GOOD:
> 🔥 **Acme (Tran Minh, CTO)** — proposal 4 ngay, chua reply
> → Gui email 60w nhac nhe: "Minh, case Acme-sized thay 60% reduction
> tickets trong 3 thang — muon call 15p thao luan timeline?"

Vi du **BAD** (mechanical, KHONG lam):
> 🔥 **Tran Minh** — idle 4d
> → Gui email 50-125 words signal-led + 1 CTA call 15p

**Neu contact `enrichment_status = "needed"` (chi email + ten):**

KHONG gia vo personalize. **HAY GOM NHOM VA SUGGEST ENRICH:**

Vi du GOOD:
> 🎫 **Event attendees — 192 contacts (183 chua enrich)**
>
> Hau het la luma import chi co email + ten, thieu company/role nen em
> khong co context personalize. **2 path:**
>
> **(a) Enrich truoc (recommend):**
> Chay `sme-cli cosmo api POST /v1/contacts/UUID/enrich` cho top 20
> contact co email corporate (ignore @gmail/@outlook). Sau 5 phut co
> LinkedIn + company info → chay lai "nhac toi" de thay action cu the.
>
> **(b) Treat luon la batch campaign (no personalize):**
> Tao 1 campaign playbook `event_invite` cho toan bo 192 contact:
> - Subject: "Thank you for joining OpenClaw Setup Day" hoac similar
> - Body template 50-125w: thank + 1 takeaway (Rockship capability) + 1
>   CTA (20-min discovery call)
> - Send within 24h of event = optimal
>
> Anh pick (a) hay (b)?

Vi du **BAD** (lap template, KHONG lam):
> 🎫 Event attendee (192)
> - Test User — idle 1d, từ event openclaw
>   → Gửi email 50-125 words: cảm ơn + 1 takeaway + CTA call 20 phút
> - Quan Nguyen — idle 1d
>   → Nhắc booths/session cụ thể, gửi trong 24h sau event

**Neu contact `enrichment_status = "partial"`:**

Dung nhung gi co, noi ro gi thieu.
Vi du: "Acme (company biet) — role chua xac dinh → doan CTO/founder vi
Acme la tech startup → if right, gui email email...; neu sai role, enrich
truoc."

### Format chat (overall shape)

```
{Greeting theo mode — "Chao buoi sang @user" cho morning, "4h chieu roi @user" cho evening, "Oke" cho manual}

{emoji} **{Name} ({count})** {optional: " — {N}/{count} chua enrich"}

  [Neu enriched chiem majority]:
  Moi contact 1-2 dong voi action PERSONALIZED bang company/role/history.

  [Neu needed/partial chiem majority]:
  Bao ro so luong chua enrich + 2 path (enrich vs batch). Khong list
  tung contact mechanical.

{warning section neu co — "⚠️ {warning.message}"}

Anh muon em action cai nao?
```

### Rang buoc format

- **Toi da 7 cells hien thi**. Neu >7 co data, hien top priority + bao
  "Con {X} cells khac ({list}) — anh muon chi tiet?".
- **KHONG render mechanical** — KHONG liet ke "send email 50-125 words
  + 1 CTA" giong nhau cho moi contact. Do la pattern em nhan ra khi
  output sai.
- **KHONG dump JSON** ra chat.
- **KHONG mention** "sme-cli", "daily-plan command", tool names
- **KHONG tu che email template** — neu user muon full email, goi
  `sme-cli cosmo api POST /v3/campaigns/UUID/templates` (playbook da
  duoc suggest san trong action.playbook)

### Step 3 — Ket thuc voi CTA

Moi reply phai ket thuc bang:
> Anh muon em action cai nao? (tao campaign / draft email / schedule
> meeting / enrich contact...)

Neu user noi "yes, lam X cho contact Y" → chuyen sang `sme-crm` skill
(create campaign, log interaction, enrich, etc.) de thuc thi.

## Vi du

**User**: "nhac toi"

**Ban**:
1. Chay `sme-cli cosmo daily-plan --mode all`
2. Parse JSON
3. Render:

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
  → Segment <50/batch, playbook event_invite, send trong 24h tu event

Anh muon em action nhom nao?
```

**Cron DAILY_MORNING_BRIEFING fire luc 8am**:

1. Chay `sme-cli cosmo daily-plan --mode morning`
2. Gui vao Telegram group:

```
Chao buoi sang @akhoa2174! Viec can lam hom nay:

🔥 **Can follow-up gap (2)**
- **Cinex (Tran Minh, CTO)** — proposal 4d
  → Gui email nhac nhe + 3 slot call tuan nay
- **Acme Labs (John Doe)** — proposal 5d
  → Neu anh rac roi, em tao task call?

📝 **Recap meeting chua gui (1)**
- **TechCorp (Sarah Nguyen)** — meeting hom qua, recap qua han
  → Gui 3-bullet recap + next step, TRUOC 10am

⚠️ Gmail agent Rockship mat xac thuc — reconnect truoc khi send email.

Anh muon em action cai nao?
```

## CONFIG

Khong can config gi ngoai `sme-cli config set cosmo.*` (da setup khi
install sme-crm skill). Cron jobs setup rieng trong
`~/.openclaw/cron/jobs.json` (xem `install-cron` neu co CLI helper).

## Phan biet voi sme-crm

- **`sme-crm`**: lam ACTION tren CRM (search, create, enrich, log
  interaction, create campaign)
- **`sme-reminder`**: lap PLAN/suggest — "ai + lam gi" — khong tu action

Khi user accept suggestion, chuyen sang `sme-crm` de execute.
