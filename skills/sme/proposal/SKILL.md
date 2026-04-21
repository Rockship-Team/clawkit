---
name: sme-proposal
description: "Proposal generator cho SME — viet proposal chuyen nghiep, render PDF tai cho (chromium headless), gui file thang vao chat. 7-step pipeline tu client data → outline → approve → PDF."
metadata: { "openclaw": { "emoji": "📝" } }
---

# Proposal — SME Vietnam

## CRITICAL RULES — READ FIRST

### How to Talk (USER-FRIENDLY FIRST)

- Talk like a human colleague on Slack. Short, warm, natural. NO technical jargon (token, API, UUID, JSON, endpoint, JWT, 401, refresh, etc.).
- **NEVER mention**: himalaya, gog, IMAP, SMTP, MML, JSON, API, endpoint, email provider, account cau hinh, UUID, campaign ID, contact list ID, playbook name, agent ID, token, JWT, refresh, 401/403/500 error, curl, bash, script.
- Maximum 2-3 casual questions if missing info — then DO IT.
- Bias toward action.

### If Something Goes Wrong

- NEVER say: "Token expired", "401", "API error", "JWT invalid".
- INSTEAD: "Xin loi, he thong dang ket noi lai. De minh thu lai..."

### How to Reply After Completing a Task

- 1-3 cau max. Nhu texting teammate.
- NEVER dump IDs, UUIDs, chi tiet ky thuat.
- NEVER dung bullet-point "Chi tiet:", "Campaign ID:", "Trang thai:".
- NEVER noi "saved to /tmp/...", "outline saved at...", "file written to..." — user khong can biet file path.
- NEVER narrate "Step 1 ✅ Step 2 ✅ Step 3..." — chay silent, chi report ket qua cuoi.
- Good: "Done, proposal sent qua email cho Son roi nha — co link PDF attach."
- Bad: "Campaign ID dfbf48b7... Status: active..."

---

## SCOPE

Skill nay chi lam proposal. Neu user hoi:

- "Tao campaign", "gui email hang loat" → chuyen cho `sme-campaign`.
- "Meeting prep", "daily action", "reply cua khach" → chuyen cho `sme-engagement`.
- "Search contact", "enrich info" → chuyen cho `sme-crm`.
- "Tao order", "chot don hang" → chuyen cho `sme-sales`.

---

## 7-STEP PROPOSAL PIPELINE

Khi user noi "proposal", "quote", "bao gia", "viet de xuat" — theo dung 7 buoc sau.

## Step 1 — Collect Client Input

1. **Check CRM truoc**: `sme-cli cosmo search-contact "company"`
   - Co → dung data tu CRM (profile, `ai_insights`, past interactions)
   - Khong co → hoi user hoac research Apollo
2. Neu user cung cap meeting notes, extract: requirements, budget, timeline, decision maker, pain points, missing info.
3. Neu user chi cung cap ten cong ty, research Apollo:
   - `sme-cli apollo search-company "company"`
   - `sme-cli apollo search-people "company" "c_suite,vp"`

## Step 2 — Normalize Information

Standardize tat ca data vao unified client brief.
→ Doc [normalize-client-data.md](references/normalize-client-data.md)

Output: company profile, requirements, commercial terms, signals, missing info. Flag anything incomplete.

Neu client chua co trong CRM, tao contact: `sme-cli cosmo create-contact`

## Step 3 — Determine Proposal Type

Hoi user loai proposal can tao:

| Type               | Best for                               |
| ------------------ | -------------------------------------- |
| `ai-agent`         | AI chatbot / agent solutions           |
| `consulting`       | IT / technology consulting engagements |
| `custom-dev`       | Custom software development projects   |
| `saas`             | SaaS platform subscriptions            |
| `managed-services` | Ongoing managed services / retainers   |

Neu user noi "tu chon" hoac "auto", recommend theo client brief.

## Step 4 — Load Proposal Template

Load template tu `assets/templates/`:

| Type             | Template                                                    |
| ---------------- | ----------------------------------------------------------- |
| ai-agent         | [ai-agent.md](assets/templates/ai-agent.md)                 |
| consulting       | [consulting.md](assets/templates/consulting.md)             |
| custom-dev       | [custom-dev.md](assets/templates/custom-dev.md)             |
| saas             | [saas.md](assets/templates/saas.md)                         |
| managed-services | [managed-services.md](assets/templates/managed-services.md) |

**Pricing — BAT BUOC goi CLI TRUOC KHI viet outline:**

```bash
sme-cli proposal pricing
```

> **HARD RULE:** Step 5 KHONG duoc bat dau truoc khi chay command nay va doc xong output.
> Viet outline voi gia tu tri nho = hallucinate, user se catch + bat sua, mat thoi gian.
> Command return JSON 3 tier hardcoded (Starter 15M / Pro 400M / Enterprise 800M) + add-ons.
> **TUYET DOI KHONG** doc `references/pricing-packages.md`. **CHI** dung so tu CLI output.

Optional read: [pricing-strategy.md](references/pricing-strategy.md) — guidance match package voi client.

## Step 5 — Research & Generate Outline

1. **Research**: WebSearch de tim 2-3 reference outlines tu proposals thuc te cung industry.
2. **Read**: [outline-guide.md](references/outline-guide.md)
3. **Generate outline**: Fill template sections voi client-specific content. Moi section phai map toi mot client need.
4. **Pricing**: Chay `sme-cli proposal pricing`, chon TIER phu hop (Starter / Pro / Enterprise). Neu budget > Enterprise → recommend Enterprise + list add-ons tu output. **CAM bia tier moi.**

## Step 6 — User Review

Present outline cho user review:

1. Display outline day du, ro rang.
2. Hoi user confirm hoac request changes.
3. Neu user yeu cau sua → update outline, present lai.
4. **Doi user approve** (noi OK, "duyet", "duoc roi"...) truoc khi qua Step 7.

## Step 7 — Render PDF Locally + Send File to User (CHI SAU KHI user approve)

> **MANDATORY PRE-CHECK — failing any of these = hard failure:**
>
> 1. **APPROVAL GATE (HARD STOP).** Tin nhan CUOI CUNG cua user phai chua mot
>    trong cac tu khoa: `approve`, `duyet`, `OK`, `duoc roi`, `dong y`, `gen di`,
>    `chot`. Khong duoc suy dien — neu thieu tu khoa → **STOP, quay lai Step 6**
>    va present outline + hoi "Em chot luon nhe?".
> 2. **NO AUTO-RETRY.** Neu `sme-cli proposal generate` tra error → **STOP**
>    va bao user. Moi lan retry se ghi de file output. Doc error message,
>    sua root cause (outline qua ngan / contact_id sai format / chromium
>    chua cai), roi chay lai **1 lan**.

**How to generate PDF — ONE command only:**

1. Save approved outline vao temp file (IM LANG — khong noi "saved at /tmp/..." voi user):

   ```bash
   cat > /tmp/proposal_outline.md << 'OUTLINE_EOF'
   {paste approved outline}
   OUTLINE_EOF
   ```

2. Chay **deterministic wrapper** — validate tier + contact_id + outline size,
   build HTML tu outline (voi header company + pricing block tu tier), render
   PDF tai cho bang chromium headless:

   ```bash
   sme-cli proposal generate "{Company_Name}" "{contact_id}" "{Starter|Pro|Enterprise}" /tmp/proposal_outline.md
   ```

   Validation:
   - `contact_id` = UUID tu `sme-cli cosmo search-contact` (Step 1). CLI reject neu sai format.
   - Tier phai la 1 trong 3 (Starter / Pro / Enterprise). CLI reject "Enterprise Plus", "Custom", v.v.
   - Outline > 200 bytes. CLI reject placeholder.
   - Neu CLI return `"ok": false` → **STOP**, doc error, sua root cause.

   Output: `{"pdf_path": "/tmp/proposal_<company>_<timestamp>.pdf", "engine": "...", "status": "completed"}`.

   Dependencies: chromium (hoac chromium-browser / google-chrome) phai co san
   tren PATH. Fallback thu tu: chromium → chromium-browser → google-chrome →
   google-chrome-stable → wkhtmltopdf → pandoc. Neu all fail, CLI report
   "no PDF engine found on PATH" — cai `chromium` bang package manager.

3. **Gui PDF file vao chat** — KHONG share link, KHONG paste URL:

   ```bash
   sme-cli channel send-file "<pdf_path tu step 2>" \
     --chat-id "<chat_id tu conversation metadata>" \
     --caption "Proposal cho {Company}"
   ```

   CLI tu doc Telegram bot token tu `~/.openclaw/openclaw.json`, POST
   `sendDocument` qua Bot API. Output: `{"ok": true, "message_id": N,
   "file_name": ..., "file_size": N}`.

   `chat_id` lay tu `Conversation info` trong context cua message hien tai
   (field `conversation_label` co dang `... id:-5147613854`, hoac
   `sender_id` neu la DM). KHONG hardcode chat_id.

   Sau khi CLI return ok, noi voi user 1 cau ngan: "Em gui proposal cho
   anh roi nha — file .pdf trong chat." KHONG paste pdf_path (user khong
   can biet /tmp/...).

4. **Log interaction**: `sme-cli cosmo log-interaction <contact_id> "proposal_sent"`

5. **PATCH business_stage**:

   ```bash
   sme-cli cosmo api PATCH /v1/contacts/UUID '{"business_stage":"PROPOSAL"}'
   ```

## References

Doc khi can — khong preload.

| File                                                            | Read when…                                           |
| --------------------------------------------------------------- | ---------------------------------------------------- |
| [cosmo-overview.md](references/cosmo-overview.md)               | User hoi COSMO la gi, business context               |
| [cosmo-workflows.md](references/cosmo-workflows.md)             | **Before any COSMO API call** — contains exact flows |
| [normalize-client-data.md](references/normalize-client-data.md) | Step 2                                               |
| [outline-guide.md](references/outline-guide.md)                 | Step 5                                               |
| [pricing-packages.md](references/pricing-packages.md)           | Step 4-5                                             |
| [pricing-strategy.md](references/pricing-strategy.md)           | Step 4-5                                             |
| [cosmo-api.md](references/cosmo-api.md)                         | COSMO endpoints                                      |
| [apollo-api.md](references/apollo-api.md)                       | Apollo endpoints                                     |

## Assets

| File                | Purpose                        |
| ------------------- | ------------------------------ |
| `assets/templates/` | 5 proposal outlines by type    |

## Data Updates

User co the update bat ky file nao trong skill:

| User provides…               | Update this file                      |
| ---------------------------- | ------------------------------------- |
| Pricing, packages, discounts | `references/pricing-packages.md`      |
| Pricing strategy             | `references/pricing-strategy.md`      |
| Template changes             | `assets/templates/{type}.md`          |
| COSMO API changes            | `references/cosmo-api.md`             |
| Apollo API changes           | `references/apollo-api.md`            |
| Normalization rules          | `references/normalize-client-data.md` |
| Outline quality rules        | `references/outline-guide.md`         |

Sau khi update: confirm file nao da thay + summarize diff.

## Rules

- **PDF render = `sme-cli proposal generate` ONLY.** Validates tier + contact_id UUID + outline size > 200 bytes, builds branded HTML, then shells out to chromium (fallback chain to wkhtmltopdf/pandoc). ONE command, local render, no external API.
- **Pricing = `sme-cli proposal pricing` ONLY.** Khong tu viet gia tu tri nho. Khong doc markdown pricing.
- **Gui PDF qua file attachment**, khong paste link. PDF o local `/tmp/` — dung runtime's file-send mechanism de attach vao chat.
- **Step 6** = present outline → WAIT for approval.
- **Step 7** = save outline to file → `sme-cli proposal generate` → attach PDF file to chat.
- Never invent client info — chi dung API data / user input.
- Always use `sme-cli proposal pricing` output — never invent prices or tiers.
- Never expose walk-away prices or competitor data in client-facing output.
- Flag missing info de BD biet hoi follow-up.
- Always check CRM truoc Apollo (tranh duplicate contacts).
- Respond in same language user writes in.
- Sau khi gui proposal, log interaction + PATCH `business_stage = PROPOSAL`.
