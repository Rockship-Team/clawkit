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

### 🔑 QUY TAC VANG #1: PLAIN LANGUAGE — KHONG dung thuat ngu tech

User khong phai ky su. Khi render output, CAM dung cac tu sau. Cot trai
la cam, cot phai la thay the de hieu:

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
| "stage=PROPOSAL" / "QUALIFIED" | "dang doi phan hoi proposal" / "da quan tam, can gap" |
| "allowlist" / "config" / "API key" | (khong noi — do la chi tiet ky thuat) |
| "tier" / "playbook" / "stage" | dung ngon ngu thong thuong |
| "COSMO CRM" | "he thong khach hang" / "data cua anh" |
| "Manus" / "chromium" / ten tool | (khong noi) |

Neu anh thay minh dinh viet 1 tu tech, STOP va dich sang tieng Viet
de hieu. User khong can biet "em dung tool gi", ho can biet "em se lam gi".

### 🔑 QUY TAC VANG #2: personalize theo context co san

Khi render moi contact:

**Neu contact `enrichment_status = "enriched"` (co company + role):**

Compose action **CU THE** dua tren:
- Company + job_title (vd "anh Tuan — CTO Acme")
- Last_outcome / next_step neu co (vd "proposal gui 4 ngay, chua reply")
- Industry (infer tu company)

Vi du GOOD (tieng Viet friendly):
> 🔥 **Acme — Trần Minh (CTO)** — gửi đề xuất 4 ngày rồi chưa thấy trả lời
> → Em gợi ý gửi 1 email nhắc nhẹ: "Anh Minh, có khách hàng cỡ Acme
> (80 người) giảm 60% ticket support trong 3 tháng sau khi triển khai.
> Mình gọi 15 phút tuần này nhé?"

Vi du **BAD** (mechanical, KHONG lam):
> 🔥 **Tran Minh** — idle 4d
> → Send email 50-125 words signal-led + 1 CTA call 15p

**Neu contact `enrichment_status = "needed"` (chi email + ten):**

KHONG gia vo personalize. Gom nhom va hoi user chon huong di:

Vi du GOOD (tieng Viet, khong tech jargon):
> 🎫 **192 người từ sự kiện OpenClaw Setup Day** — chưa follow-up
>
> Em chỉ có email + tên của mọi người, chưa biết họ làm công ty gì, vai
> trò gì. 2 cách mình có thể làm:
>
> **🔍 Cách 1 — Tra cứu thêm thông tin trước (em recommend)**
> Em lấy info LinkedIn + công ty của 20-30 người đầu từ dịch vụ tra cứu
> doanh nghiệp. Mất ~5 phút. Xong em tư vấn cụ thể ai nên gọi, ai gửi
> email, nội dung như thế nào — kiểu "anh Minh làm CTO Acme, team 80
> người, nên gửi email nhắc case study A".
>
> **📧 Cách 2 — Gửi 1 email cám ơn chung cho cả 192 người**
> Nội dung: cảm ơn đã tham gia sự kiện + gợi ý book 1 cuộc gọi 20 phút
> thảo luận nhu cầu. Không nhắc tên cụ thể từng người. Gửi ngay trong
> 24h sau sự kiện là thời điểm tốt nhất. Em chia nhóm <50 người mỗi lần
> gửi để tránh bị Gmail chặn.
>
> ⚠️ Lưu ý: Gmail Rockship đang mất đăng nhập, phải fix trước khi gửi.
>
> Anh muốn em làm Cách 1 hay Cách 2?

Vi du **BAD** (quat ngon tech, KHONG lam):
> 🎫 Event attendee (192)
> - Test User — idle 1d, từ source openclaw_event
>   → Send email 50-125 words, 1 CTA, playbook event_invite, segment <50/batch

**Neu contact `enrichment_status = "partial"`:**

Dung nhung gi co, noi ro gi thieu — bang tieng Viet.
Vi du: "Acme — biet công ty nhưng chưa rõ vị trí của người này → đoán
là CTO hoặc founder vì Acme là startup công nghệ. Nếu đúng, gửi email
nhắc case study. Nếu không chắc, tra cứu thêm trước."

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
