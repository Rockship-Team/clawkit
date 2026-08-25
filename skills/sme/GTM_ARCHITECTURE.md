# GTM Agent Architecture (Phase 1 — Foundation)

Tài liệu kỹ thuật cho engineer/maintainer — KHÔNG phải SKILL.md, không được clawkit cài vào workspace của
agent nào. Ghi lại ranh giới trách nhiệm sau audit + revised architecture cho `gtm` agent, để capability
mới không tiếp tục bị đặt sai layer (như case cadence/messaging-angle từng nằm lạc trong `cosmo_plan.go`).

Xem thêm: `skills/sme/orchestrator/SKILL.md` (routing table chính thức), `skills/sme/orchestrator/references/approval-policy.md` (approval policy), `skills/sme/reminder/SKILL.md` (Briefing), `skills/sme/engagement/SKILL.md` (Unified Taxonomy).

## Agent Architecture Components (không phải core skill nghiệp vụ)

| Component | Sở hữu bởi | Vai trò |
|---|---|---|
| **Orchestrator** | `sme-orchestrator` (skill mới, Phase 1) | Goal understanding, planning, skill routing, approval decision, Next Best Action (basic), result evaluation — CHỈ cho request đa bước/mơ hồ. |
| **Briefing** | `sme-reminder` (tái định vị, Phase 1) | Morning/EOD/Weekly, overdue, priority alert, proactive reminder. KHÔNG quyết định workflow/approval. |
| **CRM (shared service)** | `sme-crm` | COSMO gateway — search/enrich/segment/log contact. KHÔNG phải business-reasoning skill — mọi skill khác PHẢI delegate qua đây thay vì gọi COSMO trực tiếp. Enforcement hiện tại: quy ước ở tầng prompt (chưa code-level guard — rủi ro cao nếu làm ở Phase 1, hoãn). |
| **Memory (shared infra)** | OpenClaw native `memory-core` (dream consolidation, embedding search) | Nguồn conversation-history CHÍNH THỨC. `sme-learn` KHÔNG phải memory chính — chỉ ghi structured preference/correction note (per-user), thu hẹp phạm vi để tránh nhầm với memory-core. KHÔNG build memory system mới. |
| **State** | Phân mảnh có chủ đích, CHƯA hợp nhất ở Phase 1/2 | 3 cơ chế: (1) COSMO `contacts.business_stage` (remote, nguồn sự thật cho deal stage), (2) `pipeline-watch-state.json` (local file, dedup Gmail thread + alert cooldown), (3) SQLite (`outreach_events`, `weekly_kpis` — local). Document rõ để không tái phân mảnh thêm; hợp nhất chỉ làm nếu có pain point thật (Future Phase). |
| **Scheduler (service/tool)** | `sme-scheduler` | Wrapper quanh native tool `cron`, thuần time-based. Không fetch data, không suggest — tách biệt rõ với Briefing (data-based) và Orchestrator (goal-based). |
| **Tools/Integrations** | Từng skill sở hữu implementation riêng | LinkedIn (CDP client tự viết trong `outreach.go`, read-only), Google Workspace (`gog`, binary riêng — Gmail/Calendar/Drive, Calendar CHƯA tách khỏi gog), COSMO/Apollo (`cosmo.go`/`cosmo_ai.go`), Document/PDF (`proposal.go`, chromium headless). KHÔNG coi là "core GTM business skill". |

## 8 Core GTM Business Skills

### 1. `sme-intelligence` *(Phase 2 — chưa có skill riêng)*
- **Responsibility:** ICP definition, account/contact research, enrichment, qualification criteria, scoring, pain hypothesis — tín hiệu **account-level, trước/ngoài hội thoại**.
- **Input:** company/contact identifier, industry/segment.
- **Output:** ICP fit score, enrichment data, pain hypothesis, qualification criteria.
- **Owns:** logic scoring/pain-hypothesis (khi build ở Phase 2).
- **Does NOT own:** conversation-level intent/sentiment (→ `sme-engagement`), contact identity storage (→ `sme-crm`).
- **Dependency:** `sme-crm`, COSMO/Apollo (external).
- **Hiện tại (tạm thời):** logic nằm rải rác trong `cosmo_ai.go` (ICP/relationship score proxy COSMO) + Apollo enrich qua `sme-crm` delegate. Chưa MOVE, chưa CREATE skill riêng ở Phase 1.

### 2. `sme-campaign`
- **Responsibility:** campaign objective, segment, channel, cadence/sequence, messaging angle, follow-up strategy, campaign KPI.
- **Input:** campaign objective (event_outreach/cold_reach/re_engage/follow_up), segment.
- **Output:** campaign record, sequence lịch trình, messaging draft theo cadence.
- **Owns:** campaign lifecycle definition (4 loại), segment.* (qua `sme-crm`, đã chạy thật).
- **Does NOT own:** thực thi gửi LinkedIn/email (→ `sme-outreach`/`sme-marketing`), reply handling (→ `sme-engagement`).
- **Dependency:** `sme-crm` (segment), `sme-intelligence` (messaging input, Phase 2), `sme-outreach` (thực thi kênh).
- **Gap đã biết:** CLI thực thi (`create`/`gen-templates`/`activate`/`stats`) chưa tồn tại — Phase 2.

### 3. `sme-outreach`
- **Responsibility:** LinkedIn (và kênh khác trong tương lai) outbound activity — connection request, message send/follow-up/re-engagement, activity logging.
- **Input:** target profile/segment (từ campaign), draft message (từ marketing/intelligence).
- **Output:** `outreach_events` (connection_request_sent/received, message_sent/reply_received); từ Phase 2A
  thêm `outreach_reply_classifications` (kết quả phân loại do `sme-engagement` thực hiện, `sme-outreach` chỉ
  validate + lưu — xem mục 4).
- **Owns:** LinkedIn CDP client, dedup/fingerprint ledger, stale/follow-up derived view (`opportunity view`/`risk-list` không, mà `outreach stale` — reuse `outreach_events`, không entity mới).
- **Does NOT own:** phân loại intent/sentiment của reply (→ `sme-engagement`) — **Phase 2A đã build pipeline code-level** (`outreach reply-context` → agent classify → `outreach log-classification`), thay cho hành vi chỉ prompt-following trước đây. Xem `outreach/SKILL.md` mục PIPELINE.
- **Dependency:** `sme-intelligence` (messaging angle), `sme-engagement` (hand-off khi có reply), `sme-crm`, `sme-opportunity` (đọc lại classification làm Evidence).

### 4. `sme-engagement`
- **Responsibility (long-term):** reply analysis, intent/sentiment/objection (Unified Taxonomy — xem `engagement/SKILL.md`), buying signal trong hội thoại, suggested response, meeting prep/follow-up.
- **Input:** reply content (mọi kênh), contact/stage hiện tại.
- **Output:** intent/sentiment/objection classification, draft response, meeting brief/recap.
- **Owns:** Unified Taxonomy (chủ sở hữu duy nhất, mọi kênh khác reuse — kể cả code-level pipeline mới của `sme-outreach`, xem mục 3).
- **Does NOT own:** contact identity/`business_stage` storage (→ `sme-crm`), account-level scoring trước hội thoại (→ `sme-intelligence`), gửi outbound đầu tiên (→ `sme-campaign`/`sme-outreach`), **qualification/opportunity stage/deal risk/next-step/proposal-readiness/WON-LOST** (→ `sme-opportunity`, **Phase 2A đã build**, không còn TEMPORARY). `interested`+`positive` KHÔNG tự nhảy `PROPOSAL` — Opportunity đánh giá readiness, `sme-crm` mới thực thi PATCH stage sau khi user xác nhận.
- **Dependency:** `sme-crm`, `sme-outreach` (nguồn reply LinkedIn), `sme-opportunity` (cross-check readiness trước khi đề xuất đổi stage).

### 5. `sme-opportunity` *(Phase 2A — đã build, xem `opportunity/SKILL.md`)*
- **Responsibility:** hợp nhất qualification/pain/next-step/risk/proposal-readiness thành 1 view nhất quán (lightweight, không entity mới); tương lai (Phase 3+, chỉ nếu cần): BANT đầy đủ.
- **Input:** `business_stage`, `next_step`, `stage_label`, `relationship.*` (COSMO qua `/v2/contacts/search`, cùng cơ chế `cosmo_plan.go`), risk cell (`classify()`), LinkedIn reply classification (`outreach_reply_classifications`).
- **Output:** `sme-cli opportunity view`/`risk-list` — 1 view risk/next-step/readiness thống nhất (đã merge `cosmo_plan.go` cell + `LOW_PRIORITY` render-rule cũ của reminder thành 1 nguồn duy nhất). Budget/Authority/Timeline/Known Pain luôn `unknown` nếu chưa có evidence — không suy diễn.
- **Does NOT own:** entity/database riêng (KHÔNG có bảng `opportunities` — pure aggregation); tự generate/gửi proposal (→ `sme-proposal`); phân loại reply (→ `sme-engagement`, Opportunity chỉ đọc kết quả).
- **Dependency:** `sme-crm`, `sme-engagement`, `sme-outreach` (classification data).

### 6. `sme-marketing`
- **Responsibility:** content strategy, content generation, campaign asset, (Phase 2+) market signal tổng hợp.
- **Owns:** `social.go` pipeline (7 bucket, cadence, 6-step content generation) — đã tốt, giữ nguyên.
- **Does NOT own:** audience/segment definition (delegate `sme-crm`), gửi content thật (chỉ sinh, không tự publish).
- **Dependency:** `sme-crm` (segment), `sme-analytics` (feedback loop hiệu suất content, Phase 3+).

### 7. `sme-proposal`
- **Responsibility:** sinh + render (PDF) + gửi proposal, có approval gate.
- **Owns:** `proposal.go` (chromium chain — không đổi); pricing tier/add-on/discount **từ Phase 2A đã config-hoá**
  (`Connections.Proposal.*` trong `config.go`, fallback đúng 100% giá trị hardcode cũ nếu config rỗng —
  KHÔNG đổi business pricing, KHÔNG đổi HTML/PDF).
- **Does NOT own:** quyết định deal context/next-step (link `sme-opportunity` — đánh giá readiness trước khi gửi, hiện contact_id vẫn là khóa chính).
- **Dependency:** `sme-crm`, `sme-opportunity`.

### 8. `sme-analytics` *(Phase 2 — chưa có skill riêng)*
- **Responsibility:** hợp nhất KPI + funnel + hiệu suất campaign/channel + recommendation.
- **Owns (khi build):** báo cáo tổng hợp dùng lại `sme-cli kpi` + `sme-cli outreach funnel` nguyên trạng.
- **Does NOT own:** `sme-bi` (skill khác hẳn, gán agent `intel`, dùng bảng `sales` cục bộ — KHÔNG PHẢI cùng nguồn dữ liệu, KHÔNG merge vào analytics này).
- **Dependency:** `sme-kpi`, `sme-outreach`.
- **Hiện tại (tạm thời):** user tự gọi `sme-cli kpi check`/`sme-cli outreach funnel` riêng lẻ, chưa có 1 report hợp nhất.

## Ghi chú Phase 1

Không có bảng nào ở trên yêu cầu tạo database mới, migration, hay implement 3 skill Phase 2 (intelligence/
opportunity/analytics). Đây thuần là tài liệu ranh giới để Phase 2/3 build đúng chỗ.

## Ghi chú Phase 2A (Deal Lifecycle — đã build)

`sme-opportunity` đã build (mục 5) — lightweight aggregation view, KHÔNG bảng mới. Pipeline code-level
LinkedIn reply → `sme-outreach` → `sme-engagement` taxonomy → `sme-opportunity` đã nối (mục 3, 4, 5).
Proposal pricing đã config-hoá (mục 7), fallback y hệt giá trị cũ. `sme-intelligence`, `sme-analytics`,
`sme-campaign` (CLI thực thi), LinkedIn accepted detection, autonomous outreach **vẫn CHƯA implement** —
để Phase 2B/2C/Future, không nằm trong phạm vi Phase 2A.
