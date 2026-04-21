---
name: sme-reminder
description: "Daily BD coach cho SME — tu dong goi y 'hom nay/ngay mai nen outreach ai va lam gi' dua tren pipeline cell (research-backed). Kich hoat khi user noi 'nhac toi', 'ai can follow-up', 'daily outreach', 'remind me', hoac khi cron fire morning/evening briefing."
metadata: { "openclaw": { "emoji": "🎯" } }
---

# SME Reminder — Ai Can Outreach Hom Nay, Va LAM GI

## Trigger

Kich hoat skill nay khi user noi bat ky tin nhan nao co y:
- "nhac toi" / "nhac toi outreach"
- "ai can follow-up" / "ai dang stuck"
- "hom nay outreach ai" / "hom nay nen lien he ai" / "gio lam gi"
- "con contact nao chua lam" / "list stale leads"
- "remind me" / "who to contact today" / "daily outreach" / "suggest outreach"

Hoac khi cron payload co "DAILY_MORNING_BRIEFING" / "DAILY_EVENING_REVIEW".

**KHONG** kich hoat khi user muon schedule cron moi ("nhac lan 3h" /
"moi ngay 9h"). Do la task cua scheduler skill khac, khong phai skill nay.

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

### Step 2 — Format ra chat (user-friendly)

Output JSON co shape:
```json
{
  "ok": true,
  "mode": "morning",
  "generated_at": "2026-04-21T08:00:00+07:00",
  "total_contacts": 182,
  "cells": [
    {
      "id": "PROPOSAL_HOT",
      "emoji": "🔥",
      "name": "Can follow-up gap",
      "priority": 1,
      "why": "Day 3 sweet spot cho touch #2 (+31% reply...)",
      "count": 2,
      "action": {
        "playbook": "cold_outreach",
        "subject_hint": "...",
        "length": "50-125 words",
        "cta": "..."
      },
      "contacts": [{"id":"...","name":"John","company":"Acme","idle_days":4,...}]
    }
  ],
  "warnings": [{"type": "email_agent_invalid_cred", "message": "..."}]
}
```

Format chat theo template:

```
{Greeting theo mode — "Chao buoi sang @user" cho morning, "4h chieu roi @user" cho evening, "Oke" cho manual}

{emoji} **{Name} ({count})**
- **{company} ({contact.name}, {contact.job_title})** — idle {idle_days}d
  → {1-line custom action dua tren action.subject_hint + cta, viet gon}
- ... (max 5 morning / 3 evening / 7 all)

{next cell...}

{warning section neu co — "⚠️ {warning.message}"}

Anh muon em action cai nao?
```

### Rang buoc format

- **Toi da 7 cells hien thi**. Neu >7 co data, hien top priority + bao
  "Con {X} cells khac ({list}) — anh muon chi tiet?"
- **Per contact 1-2 dong max** — khong dai dong
- **Suggest action ngan** (1 cau, dua tren `action.subject_hint` +
  `action.cta` cua cell — ko copy nguyen van action.length etc.)
- **KHONG dump JSON** ra chat
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
