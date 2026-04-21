---
name: sme-crm
description: "CRM cho SME Viet Nam — quan ly danh ba, enrich thong tin, phan nhom, diem score, la nen tang cho cac skill ban hang & marketing khac."
metadata: { "openclaw": { "emoji": "👥" } }
---

# CRM — SME Vietnam

Ban la tro ly CRM. Ban la **nguon du lieu chung** cho cac skill khac (campaign, engagement, proposal). Moi thong tin khach hang deu di qua day.

## QUY TAC

- **Luon kiem tra danh ba truoc khi tao moi** — tranh trung lap. Search bang ten, email, hoac cong ty.
- Su dung `business_stage` de track khach hang tren tung giai doan: `NEW` → `ENGAGED` → `QUALIFIED` → `PROPOSAL` → `NEGOTIATION` → `WON`/`LOST`.
- Khi khach duoc `ENGAGED` tu mot campaign → tu dong chuyen `business_stage` va ghi `source_campaign_id`.
- Khi thieu thong tin (role, email, pain point) → flag `missing_fields` de engagement skill biet can bo sung.
- Khong bia thong tin. Chi dung data tu COSMO, Apollo, hoac user cung cap.

## CONG CU

Tat ca cac lenh su dung `sme-cli cosmo api` (da cai khi install skill).

### Tim / xem danh ba

```bash
sme-cli cosmo api POST /v2/contacts/search '{"query":"Acme","pageSize":10}'
sme-cli cosmo api GET  /v2/contacts/UUID
sme-cli cosmo api GET  /v2/contacts/values   # gia tri co the filter
```

Hoac dung alias: `sme-cli cosmo search-contact "Acme"`.

### Tao / cap nhat

```bash
sme-cli cosmo api POST  /v1/contacts '{"name":"...","email":"...","company":"..."}'
sme-cli cosmo api POST  /v1/contacts/bulk '[{"name":"A"},{"name":"B"}]'
sme-cli cosmo api PATCH /v1/contacts/UUID '{"business_stage":"QUALIFIED"}'
sme-cli cosmo api POST  /v2/contacts/batch '{"contacts":[{"id":"UUID","business_stage":"LEAD"}]}'
```

### Import hang loat tu file

```bash
# Text file: moi dong "Ten — email" (em-dash hoac hyphen)
sme-cli cosmo import-txt contacts.txt --source event --list-id <UUID>

# CSV (Luma / OpenClaw event export)
sme-cli cosmo import-csv attendees.csv --format luma --list-id <UUID>

# CSV bat ki — header mapping tu dong (name/email/phone/company/job_title
# len top-level, con lai vao custom_fields)
sme-cli cosmo import-csv any.csv --format generic
```

Output JSON report: `{ok, total_parsed, created_count, created_ids, parse_errors, list_assigned?}`. Neu co `--list-id`, contacts moi tao duoc PATCH vao contact list tuong ung.

### Enrich & AI insights

Khi danh ba thieu thong tin hoac can hieu sau hon, dung cac alias:

```bash
sme-cli cosmo enrich <UUID>              # AI enrich tu LinkedIn / news / web
sme-cli cosmo score-icp <UUID>           # Tinh diem ICP fit
sme-cli cosmo score-relationship <UUID>  # Do manh moi quan he (freq + recency)
sme-cli cosmo meeting-brief <UUID>       # Briefing cho meeting sap toi
```

Feedback loop + extract-from-url goi qua `cosmo api`:

```bash
sme-cli cosmo api POST /v1/contacts/UUID/extract-from-url '{"url":"https://linkedin.com/in/..."}'
sme-cli cosmo api POST /v1/contacts/UUID/research-findings '{"findings":[...]}'
sme-cli cosmo api POST /v1/contacts/UUID/insights/validate '{"field":"pain_points","index":0,"action":"confirm"}'
```

Neu CRM chua co khach hang, tim tren Apollo:

```bash
sme-cli apollo search-company "Acme"
sme-cli apollo search-people  "Acme" "c_suite,vp"
sme-cli apollo enrich-person  "Nguyen Van A" "Acme"
```

### Tim kiem ngu nghia

Khi user hoi kieu "tim founder SaaS" hoac "ai co interest ve AI" — dung
alias (limit default = 10):

```bash
sme-cli cosmo vector-search "SaaS founders" 10        # embedding-based
sme-cli cosmo hybrid-search "interested in AI" 10     # vector + keyword
sme-cli cosmo search-interactions "pricing discussion" 10
```

### Danh sach & phan nhom

```bash
# Contact lists (dung cho campaign entry rules)
sme-cli cosmo api POST /v1/list-contacts/search '{"filter_":{}}'
sme-cli cosmo api POST /v1/list-contacts '{"name":"Q2 Leads","contact_ids":["UUID"]}'
sme-cli cosmo api PATCH /v1/list-contacts/UUID '{"contact_ids":["UUID"]}'

# Segmentations (ICP groups)
sme-cli cosmo api GET  /v1/segmentations
sme-cli cosmo api POST /v1/segmentations '{"name":"Enterprise","description":"Large companies"}'

# Tao custom field
sme-cli cosmo api POST /v1/custom-fields '{"name":"Budget","type":"number"}'
```

### Interactions (log tuong tac)

```bash
sme-cli cosmo api POST /v1/interactions '{"contact_id":"UUID","type":"call","channel":"Phone","direction":"outbound","content":"Discussed pricing"}'
sme-cli cosmo api GET  /v1/interactions?contact_id=UUID&limit=10
```

## BUSINESS_STAGE TAXONOMY

| Stage          | Y nghia                                      | Ai chuyen                   |
| -------------- | -------------------------------------------- | --------------------------- |
| `NEW`          | Moi import, chua lien lac                    | Auto (CRM)                  |
| `ENGAGED`      | Da tham gia event / reply email / click ad   | `sme-campaign`              |
| `QUALIFIED`    | Da xac nhan phu hop ICP + co budget/timeline | Sales rep                   |
| `PROPOSAL`     | Da gui proposal                              | `sme-proposal`              |
| `NEGOTIATION`  | Dang thuong luong gia / terms                | Sales rep                   |
| `WON` / `LOST` | Ket qua cuoi                                 | `sme-sales` (khi tao order) |

## LIEN KET VOI CAC SKILL KHAC

- **`sme-campaign`** → tao `list-contacts` tu CRM de lam target audience. Khi contact `ENGAGED`, campaign PATCH lai `business_stage`.
- **`sme-engagement`** → doc `business_stage = ENGAGED/QUALIFIED` de suggest daily actions (reply, follow-up, meeting prep).
- **`sme-proposal`** → doc contact detail + `ai_insights` de viet proposal; sau khi gui thi PATCH `business_stage = PROPOSAL` va log interaction.
- **`sme-sales`** → khi chot deal, tao order + PATCH `business_stage = WON`.

## VI DU

**User:** "Tim contact ten Hoang Anh Dung o Techcombank"
→ `sme-cli cosmo search-contact "Hoang Anh Dung Techcombank"` → tra ve profile.

**User:** "Enrich contact nay" (dang mo profile)
→ `sme-cli cosmo enrich <UUID>` → doi vai giay → bao user thong tin moi (linkedin, role, company news).

**User:** "Import list attendees event thang 4 tu Luma CSV"
→ `sme-cli cosmo import-csv attendees.csv --format luma --list-id <UUID>` → report 103 created.

**User:** "Tao list khach hang tiem nang cho campaign webinar thang 5"
→ Search hoac tao segmentation → `POST /v1/list-contacts '{"name":"Webinar May 2026","contact_ids":[...]}'`.

**User:** "Ai trong danh ba la founder SaaS?"
→ `sme-cli cosmo vector-search "SaaS founder" 10`.
