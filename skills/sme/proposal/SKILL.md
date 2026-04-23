---
name: sme-proposal
description: "Proposal generator cho SME — viet proposal chuyen nghiep cho BAT KY loai dich vu (AI agent, consulting, custom-dev, SaaS, managed-services, data engineering, training...), render PDF tai cho (chromium headless), gui file thang vao chat. 6-step pipeline tu client data → outline → approve → PDF. Dung sme-crm lam gateway CRM."
metadata: { "openclaw": { "emoji": "📝" } }
---

# Proposal — SME Vietnam

## CRITICAL RULES — READ FIRST

### How to Talk (USER-FRIENDLY FIRST)

- Talk like a human colleague on Slack. Short, warm, natural. NO technical jargon (token, API, UUID, JSON, endpoint, JWT, 401, refresh, etc.).
- **NEVER mention**: himalaya, gog, IMAP, SMTP, MML, JSON, API, endpoint, email provider, cau hinh account, UUID, campaign ID, contact list ID, playbook name, agent ID, token, JWT, refresh, 401/403/500 error, curl, bash, script.
- Maximum 2-3 casual questions if missing info — then DO IT.
- Bias toward action.

### If Something Goes Wrong

- NEVER say: "Token expired", "401", "API error", "JWT invalid".
- INSTEAD: "Xin loi, he thong dang ket noi lai. De minh thu lai..."

### How to Reply After Completing a Task

- 1-3 cau max. Nhu texting teammate.
- NEVER dump IDs, UUIDs, chi tiet ky thuat.
- NEVER "Chi tiet:", "Campaign ID:", "Trang thai:" format.
- NEVER "saved to /tmp/...", "file written to..." — user khong can biet file path.
- NEVER "Step 1 ✅ Step 2 ✅ Step 3..." — chay silent, chi report ket qua cuoi.
- Good: "Done, proposal sent qua email cho Son roi nha — co link PDF attach."
- Bad: "Campaign ID dfbf48b7... Status: active..."

---

## SCOPE

Skill nay chi lam proposal. Neu user hoi:

- "Tao campaign", "gui email hang loat" → `sme-campaign`.
- "Meeting prep", "daily action", "reply cua khach" → `sme-engagement`.
- "Search contact", "enrich info", "log interaction" → `sme-crm`.
- "Tao order", "chot don hang" → `sme-sales`.

---

## 6-STEP PROPOSAL PIPELINE

Khi user noi "proposal", "quote", "bao gia", "viet de xuat" — theo dung 6 buoc.

## Step 1 — Collect Client Input

1. **Check CRM truoc** → delegate sang `sme-crm`:

   > "sme-crm: search contact {company}"

   - Co → dung data tu CRM (profile, `ai_insights`, past interactions).
   - Khong co → hoi user hoac research Apollo (delegate `sme-crm`: "apollo search company X").

2. Neu user cung cap meeting notes, extract: requirements, budget, timeline, decision maker, pain points, missing info.

3. Neu chi co ten cong ty → delegate sme-crm: "apollo search company + search people c_suite/vp".

## Step 2 — Normalize Information

Standardize data vao unified client brief.
→ Doc [normalize-client-data.md](references/normalize-client-data.md)

Output: company profile, requirements, commercial terms, signals, missing info. Flag gi thieu.

Neu client chua trong CRM → delegate sme-crm: "create contact {name, email, company, ...}".

## Step 3 — Get Pricing (BAT BUOC TRUOC KHI viet outline)

```bash
sme-cli proposal pricing
```

> **HARD RULE:** Step 4 KHONG duoc bat dau truoc khi chay command nay va doc xong output. Viet outline voi gia tu tri nho = hallucinate. Command return JSON 3 tier (Starter 15M / Pro 400M / Enterprise 800M) + add-ons. CHI dung so tu CLI output, CAM doc `references/pricing-packages.md`.

Optional: [pricing-strategy.md](references/pricing-strategy.md) — guidance match tier voi client size / pain level.

## Step 4 — Research + Compose Outline

Proposal co the la BAT KY loai dich vu — khong bat buoc fit vao 5 template.

1. **Research** (always): WebSearch 2-3 query tim reference proposals + case studies cung industry + scale. Research tot = numbers co thuc, argument specific, tang credibility.

2. **Read principles**: [outline-guide.md](references/outline-guide.md) — cach viet section, level detail, format rules. Luon doc file nay.

3. **Check templates** (optional head start):

   | Template | Fits when client asks for |
   |---|---|
   | [ai-agent.md](assets/templates/ai-agent.md) | AI chatbot / agent |
   | [consulting.md](assets/templates/consulting.md) | IT / tech consulting |
   | [custom-dev.md](assets/templates/custom-dev.md) | Custom software dev |
   | [saas.md](assets/templates/saas.md) | SaaS subscription |
   | [managed-services.md](assets/templates/managed-services.md) | Managed services retainer |
   | (none — compose freely) | Data engineering, training, hardware, research... |

   Template la **example**, KHONG phai schema bat buoc. Section can add/remove tuy client need. Vi du client can on-prem → add "Deployment Architecture" du template khong co.

4. **Generate outline**: Fill sections voi client-specific content tu client brief + research. Moi section phai map toi client need — khong viet section vi "template bao phai co".

5. **Pricing**: Chon TIER phu hop tu Step 3 (Starter / Pro / Enterprise). Budget > Enterprise → recommend Enterprise + list add-ons tu CLI. **CAM bia tier moi** (no "Enterprise Plus", "Custom", "Premium").

## Step 5 — User Review

1. Display outline day du, ro rang.
2. Hoi user confirm hoac request changes.
3. Sua neu can, present lai.
4. **Doi user approve** ("OK", "duyet", "duoc roi"...) truoc khi qua Step 6.

## Step 6 — Render PDF + Send File (CHI SAU user approve)

> **MANDATORY PRE-CHECK:**
>
> 1. **APPROVAL GATE (HARD STOP).** Tin nhan CUOI cua user phai chua mot tu: `approve`, `duyet`, `OK`, `duoc roi`, `dong y`, `gen di`, `chot`. Khong suy dien — neu thieu → **STOP, quay lai Step 5**.
> 2. **NO AUTO-RETRY.** Neu `sme-cli proposal generate` error → **STOP** va bao user. Moi retry se ghi de file. Doc error, sua root cause (outline qua ngan / contact_id sai / chromium chua cai), chay lai **1 lan**.

**How to generate PDF — ONE command only:**

1. Save approved outline vao temp file (IM LANG):

   ```bash
   cat > /tmp/proposal_outline.md << 'OUTLINE_EOF'
   {paste approved outline}
   OUTLINE_EOF
   ```

2. Chay deterministic wrapper:

   ```bash
   sme-cli proposal generate "{Company_Name}" "{contact_id}" "{Starter|Pro|Enterprise}" /tmp/proposal_outline.md
   ```

   Validation:
   - `contact_id` = UUID tu sme-crm (Step 1). CLI reject sai format.
   - Tier phai la 1 trong 3. CLI reject "Enterprise Plus", "Custom", etc.
   - Outline > 200 bytes. CLI reject placeholder.
   - CLI return `"ok": false` → **STOP**, doc error, sua root cause.

   Output: `{"pdf_path": "/tmp/proposal_<company>_<timestamp>.pdf", "engine": "...", "status": "completed"}`.

   Dependencies: chromium (hoac chromium-browser / google-chrome) trong PATH. Fallback: chromium → chromium-browser → google-chrome → google-chrome-stable → wkhtmltopdf → pandoc.

3. **Gui PDF file vao chat** — KHONG share link, KHONG paste URL:

   ```bash
   sme-cli channel send-file "<pdf_path>" \
     --chat-id "<CHAT_ID>" \
     --caption "Proposal cho {Company}"
   ```

   CLI doc Telegram bot token tu `~/.openclaw/openclaw.json`, POST `sendDocument`. Output: `{"ok": true, "message_id": N, ...}`.

   ### Cach xac dinh `CHAT_ID` — bulletproof rules

   Doc block `Conversation info (untrusted metadata)` dau message context:
   - `sender_id` — luon co, vd `"7142847127"`
   - `is_group_chat` — true/false
   - `conversation_label` — vd `"Rockship | Business Development id:-5147613854"`

   **DEFAULT rule — luon gui DM (sender_id):**

   Proposal la document nhay cam (gia, nhan vien khach hang). **MAC DINH DM**:

   ```
   CHAT_ID = sender_id
   ```

   Vi du: `--chat-id "7142847127"`.

   **EXCEPTION — dung group_id khi user EXPLICITLY noi:** "gui vao group", "post cho ca team xem", "share public".

   Khi do extract group_id tu `conversation_label`:
   ```
   label = "Rockship | Business Development id:-5147613854"
   CHAT_ID = "-5147613854"
   ```

   **CAM:**
   - Hardcode chat_id tu vi du trong code.
   - Luan qua lai giua sender_id / group_id — pick 1 va di tiep.
   - Neu khong chac → chon sender_id (safer).

   Neu CLI report `"chat not found"` → sai format (vd `"akhoa2174"` thay vi numeric). Re-read sender_id, retry 1 lan. Fail → STOP + hoi user.

   Sau CLI return ok, noi ngan: "Em gui proposal cho anh roi nha — file .pdf trong chat." KHONG paste pdf_path.

4. **Log interaction + update stage** — DELEGATE sang sme-crm:

   > "sme-crm: log interaction proposal_sent voi contact UUID + patch stage contact UUID → PROPOSAL"

   sme-crm chay `sme-cli cosmo log-interaction UUID "proposal_sent"` + `PATCH /v1/contacts/UUID '{"business_stage":"PROPOSAL"}'`.

## References

Doc khi can — khong preload.

| File | Read when… |
|---|---|
| [normalize-client-data.md](references/normalize-client-data.md) | Step 2 |
| [outline-guide.md](references/outline-guide.md) | Step 4 |
| [pricing-packages.md](references/pricing-packages.md) | Reference only (use `sme-cli proposal pricing`) |
| [pricing-strategy.md](references/pricing-strategy.md) | Step 4 |

## Assets

| File | Purpose |
|---|---|
| `assets/templates/` | 5 proposal outlines by type |

## Data Updates

User co the update bat ky file nao:

| User provides… | Update this file |
|---|---|
| Pricing, packages, discounts | `references/pricing-packages.md` |
| Pricing strategy | `references/pricing-strategy.md` |
| Template changes | `assets/templates/{type}.md` |
| Normalization rules | `references/normalize-client-data.md` |
| Outline quality rules | `references/outline-guide.md` |

Sau update: confirm file nao thay + summarize diff.

## CRM OPERATIONS — DELEGATE SANG sme-crm

KHONG goi COSMO API truc tiep cho CRM action:

| Can lam | Delegate intent |
|---|---|
| Search contact theo company | "sme-crm: search contact {company}" |
| Apollo research company | "sme-crm: apollo search company X" |
| Apollo research people | "sme-crm: apollo search people X c_suite/vp" |
| Enrich contact | "sme-crm: enrich contact UUID" |
| Create contact | "sme-crm: create contact {...}" |
| Log interaction proposal_sent | "sme-crm: log interaction proposal_sent contact UUID" |
| Patch stage PROPOSAL | "sme-crm: patch stage contact UUID → PROPOSAL" |

**sme-cli proposal pricing / generate + sme-cli channel send-file van goi truc tiep** vi do la CLI cua skill nay, khong phai CRM action.

## Rules

- **PDF render = `sme-cli proposal generate` ONLY.** Validates tier + contact_id UUID + outline size > 200 bytes, builds branded HTML, shells out to chromium (fallback chain wkhtmltopdf/pandoc). ONE command, local, no external API.
- **Pricing = `sme-cli proposal pricing` ONLY.** Khong tu viet gia tu tri nho. Khong doc markdown pricing.
- **Gui PDF qua file attachment**, khong paste link.
- **Step 5** = present outline → WAIT for approval.
- **Step 6** = save outline → `sme-cli proposal generate` → `sme-cli channel send-file` (default DM sender_id).
- **CRM operations delegate sme-crm** — khong goi COSMO API truc tiep.
- Never invent client info — chi dung sme-crm data / user input.
- Always use `sme-cli proposal pricing` output — never invent prices / tiers.
- Never expose walk-away prices hoac competitor data trong client-facing output.
- Flag missing info de BD biet hoi follow-up.
- Always check CRM (qua sme-crm) truoc Apollo (tranh duplicate).
- Respond in same language user writes in.
- Sau gui proposal, delegate sme-crm log interaction + PATCH `business_stage = PROPOSAL`.
