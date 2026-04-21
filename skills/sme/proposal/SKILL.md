---
name: sme-proposal
description: "Proposal generator cho SME — viet proposal chuyen nghiep, export PDF qua Manus AI, 7-step pipeline tu client data → outline → approve → PDF."
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

## Step 7 — Generate & Export PDF via Manus AI (CHI SAU KHI user approve)

> **MANDATORY PRE-CHECK — failing any of these = hard failure:**
>
> 1. **APPROVAL GATE (HARD STOP).** Tin nhan CUOI CUNG cua user phai chua mot
>    trong cac tu khoa: `approve`, `duyet`, `OK`, `duoc roi`, `dong y`, `gen di`,
>    `chot`. Khong duoc suy dien — neu thieu tu khoa → **STOP, quay lai Step 6**
>    va present outline + hoi "Em chot luon nhe?". Vi pham = burn API credits ko ly do.
> 2. **NO AUTO-RETRY.** Neu `sme-cli manus generate-proposal` tra error
>    (timeout / rate limit / 5xx / "busy") → **STOP NGAY**. Bao user:
>    "Manus dang loi, anh thu lai sau 2-3 phut nha." **TUYET DOI khong**
>    chay lai cau lenh — moi lan retry = task moi tren Manus = double credits.
> 2b. **HANDLE status=pending.** Neu output co `"status": "pending"` va `"task_id"`,
>    NGHIA LA task van dang chay tren Manus (chua xong). **TUYET DOI khong**
>    chay lai `manus generate-proposal`. Phai poll bang:
>    `sme-cli manus get-task <task_id>` (tu output truoc).
>    Cho ~1 phut roi check lai. Lap lai check toi da 5 lan, neu van pending
>    thi bao user "Manus busy, em check lai sau 5 phut" — KHONG tao task moi.
> 3. **PRESERVE FULL URL.** Khi share PDF link, paste **RAW URL day du**
>    (bao gom `?Policy=...&Key-Pair-Id=...&Signature=...`). KHONG dung markdown
>    `[text](url)` — Telegram parser cat query string. Cat query = link 403 broken.
> 4. **BANNED tools** — NEU dinh dung BAT KY cong cu nay, STOP IMMEDIATELY:
>    Canva, WeasyPrint, pandoc, md-to-pdf, NotebookLM, Puppeteer, generate_pdf.sh,
>    wkhtmltopdf, Prince, Chrome headless, BAT KY HTML-to-PDF converter nao khac.
>    Dung banned tool = **hard failure**.

**How to generate PDF — ONE command only:**

1. Save approved outline vao temp file (TUYET DOI khong noi "saved at /tmp/..." voi user — im lang, day la thao tac noi bo):

   ```bash
   cat > /tmp/proposal_outline.md << 'OUTLINE_EOF'
   {paste approved outline}
   OUTLINE_EOF
   ```

2. Chay **deterministic wrapper** (validate tier + contact_id + outline size, roi goi Manus):

   ```bash
   sme-cli proposal generate "{Company_Name}" "{contact_id}" "{Starter|Pro|Enterprise}" /tmp/proposal_outline.md
   ```

   - `contact_id` = UUID tu `sme-cli cosmo search-contact` (Step 1). CLI reject neu khong phai UUID.
   - Tier phai la 1 trong 3 ten chinh xac. CLI reject "Enterprise Plus", "Pro Plus", "Custom", v.v.
   - Outline phai > 200 bytes. CLI reject placeholder.
   - Neu CLI return `"ok": false` → **STOP**, bao user roi dung (khong retry).

   Fallback cu (khong khuyen): `sme-cli manus generate-proposal "{Company}" /tmp/proposal_outline.md` — bo qua guards.

   Script tu dong:
   - Load `.env` (API keys)
   - Encode `assets/templates/style_template.pdf` as base64 attachment
   - Build Manus prompt (hardcoded, KHONG modify)
   - Create Manus task
   - Poll den khi complete
   - In PDF download URL

   **Do NOT** build JSON tay. Do NOT upload file thu cong. Do NOT modify prompt. Do NOT sua script.

3. Share PDF URL tu script output voi user.

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
| [manus-api.md](references/manus-api.md)                         | Manus AI endpoints                                   |

## Assets

| File                                  | Purpose                                            |
| ------------------------------------- | -------------------------------------------------- |
| `assets/templates/`                   | 5 proposal outlines by type + `style_template.pdf` |
| `assets/templates/style_template.pdf` | PDF style reference — Manus MUST match exactly     |

## Data Updates

User co the update bat ky file nao trong skill:

| User provides…               | Update this file                      |
| ---------------------------- | ------------------------------------- |
| Pricing, packages, discounts | `references/pricing-packages.md`      |
| Pricing strategy             | `references/pricing-strategy.md`      |
| Brand colors, fonts, layout  | `assets/templates/style_template.pdf` |
| Template changes             | `assets/templates/{type}.md`          |
| COSMO API changes            | `references/cosmo-api.md`             |
| Apollo API changes           | `references/apollo-api.md`            |
| Normalization rules          | `references/normalize-client-data.md` |
| Outline quality rules        | `references/outline-guide.md`         |

Sau khi update: confirm file nao da thay + summarize diff.

## Rules

- **PDF = Manus AI ONLY.** Preferred: `sme-cli proposal generate "{Company}" "{contact_id}" "{tier}" /tmp/outline.md` (validates tier + CRM + outline). Fallback: `sme-cli manus generate-proposal "{Company}" /tmp/outline.md` (no guards). ONE command. NO EXCEPTIONS.
- **Pricing = `sme-cli proposal pricing` ONLY.** Khong tu viet gia tu tri nho. Khong doc markdown pricing.
- **BANNED:** Canva, WeasyPrint, pandoc, md-to-pdf, NotebookLM, Puppeteer, wkhtmltopdf, Prince, Chrome headless. Using ANY banned tool = skill failure.
- **Do NOT** build Manus JSON / prompt tay. Script handles everything.
- **Step 6** = present outline → WAIT for approval.
- **Step 7** = save outline to file → run `manus_generate_proposal.sh` → share URL.
- Never invent client info — chi dung API data / user input.
- Always read `pricing-packages.md` truoc khi dinh gia — never invent prices.
- Never expose walk-away prices or competitor data in client-facing output.
- Flag missing info de BD biet hoi follow-up.
- Always check CRM truoc Apollo (tranh duplicate contacts).
- Respond in same language user writes in.
- Sau khi gui proposal, log interaction + PATCH `business_stage = PROPOSAL`.
