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

### 1. `sme-intelligence` *(Phase 2B — đã build, xem `intelligence/SKILL.md`)*
- **Responsibility:** ICP interpretation, account/contact research orchestration, account-level buying signals,
  pain hypothesis, early qualification, recommended positioning angle — tín hiệu **account-level, trước/ngoài
  hội thoại**.
- **Input:** company name/domain (+ `--org-id` Apollo sau khi disambiguate).
- **Output:** `sme-cli intelligence account` — signals (evidence-gated), pain_hypotheses (luôn đánh dấu là
  hypothesis), qualification (high/medium/low/insufficient_data — `unknown` KHÔNG BAO GIỜ = disqualified),
  target_personas, recommended_angle.
- **Owns:** facade/orchestration logic nối các capability đã có lại với nhau — KHÔNG viết lại ICP scoring,
  relationship scoring, hay Apollo research (xem bảng REUSE trong `intelligence/SKILL.md`).
- **Does NOT own:** outreach execution (→ `sme-outreach`), campaign execution (→ `sme-campaign`), reply
  intent (→ `sme-engagement`), deal stage/readiness (→ `sme-opportunity`), CRM persistence (→ `sme-crm`), KPI
  (→ `sme-analytics`).
- **Dependency:** `sme-crm` (qua `resolveOpportunityContacts`/COSMO search), Apollo (`apollo.go`),
  `sme-opportunity` (proposal tiers làm "relevant offering" reference).
- **Known limitation (Phase 2B audit):** `/v1/intelligence/vector-search/*` path đã fix (trước đó gọi sai path,
  404) nhưng backend hiện 500 do OpenAI embedding key bị 401 — vấn đề COSMO backend, KHÔNG phải client bug,
  chưa dùng được. Intelligence KHÔNG phụ thuộc endpoint này — dùng Apollo + `cosmoContactsSearch` (filter-based)
  làm nguồn chính.

### 2. `sme-campaign` *(Phase 2C — đã build, xem `campaign/SKILL.md`)*
- **Responsibility:** campaign objective, segment reference, cadence/sequence, messaging angle, follow-up strategy, campaign status/KPI.
- **Input:** playbook (label, không phải enum COSMO), `list_contact_id` (từ `sme-crm`), template content (agent soạn theo `sme-marketing/SKILL.md`).
- **Output:** `sme-cli campaign create/list/view/stats/add-template/templates/activate/pause/delete` — gọi thẳng COSMO Campaign API thật (`/v1/campaigns`, `/v1/template`), không phải giả lập.
- **Owns:** campaign lifecycle state (`draft/scheduled/active/paused/ended` — enum COSMO thật, không tự đặt tên khác), cadence qua `send_after`/`position` trên từng template.
- **Does NOT own:** ICP/research/scoring (→ `sme-intelligence`), CRM contact/segment storage (→ `sme-crm`, chỉ tham chiếu `list_contact_id`), sinh nội dung (→ agent áp dụng `sme-marketing/SKILL.md` mục C, campaign.go KHÔNG có copywriting engine riêng), thực thi gửi LinkedIn (→ `sme-outreach` — read-only, KHÔNG có send), reply handling (→ `sme-engagement`), KPI aggregation (→ `sme-analytics`).
- **Dependency:** `sme-crm` (segment), `sme-intelligence` (messaging angle input), `sme-marketing` (content rules), `sme-outreach` (LinkedIn tracking thủ công), `sme-analytics` (`analytics campaigns`).
- **Channel semantics (quan trọng — không phải chuyện nhỏ):** COSMO Campaign KHÔNG có khái niệm channel — `agent_id` luôn nghĩa là "email identity sẽ gửi". `--channel email` (default) dùng path PATCH `status=active` thật, đã enqueue worker gửi email. `--channel linkedin` **KHÔNG BAO GIỜ** chạm COSMO `status=active` (tránh vô tình trigger gửi email cho campaign định là LinkedIn) — chỉ đánh dấu `cmetadata.client.linkedin_status=ready_for_manual_send`, con người tự gửi, log qua `sme-outreach`.
- **Google email auth (known issue, xác nhận live mỗi phiên):** agent duy nhất của org (`rockship17.co@gmail.com`) có `status: "invalid Google grant"` (xác nhận qua `POST /v1/agents/search`) — `campaign activate` tự kiểm tra và TỪ CHỐI nếu vẫn invalid, không bypass. Không block phần còn lại của Campaign engine (draft/template/segment/pause/delete/stats đều hoạt động bình thường).
- **Known backend bug (ngoài phạm vi sửa):** `DELETE /v1/template/:id` panic (500, nil pointer) — để lại vài template row mồ côi vô hại (campaign cha đã xoá) từ quá trình test phiên này.

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

### 8. `sme-analytics` *(Phase 2B — đã build, thin aggregation, xem `analytics/SKILL.md`)*
- **Responsibility:** hợp nhất KPI + outreach funnel + so sánh hiệu suất channel + bottleneck detection +
  recommendation dựa trên số liệu đo được thật.
- **Owns:** `sme-cli analytics summary` — pivot lại đúng data đã có (`kpiTeamData`/`kpiActualData`/
  `outreachFunnelData`, các hàm data-returning được extract ra từ `kpi.go`/`outreach.go` để tái dùng nguyên
  query, không viết lại). Bottleneck's Qualified/Proposal count lấy từ COSMO `business_stage` qua
  `fetchAllContacts` (cùng cơ chế pagination đã fix ở Phase 2B COSMO audit).
- **Does NOT own:** `sme-bi` (skill khác hẳn, gán agent `intel`, dùng bảng `sales` cục bộ — KHÔNG PHẢI cùng
  nguồn dữ liệu, KHÔNG merge vào analytics này); campaign performance thật (Campaign Engine chưa build —
  trả cố định `"unavailable_until_campaign_engine"`, Phase 2C).
- **Dependency:** `sme-kpi`, `sme-outreach`, `sme-opportunity`/COSMO (`business_stage` cho bottleneck).
- **Kỷ luật "không bịa số":** reply_rate/recommendation không có benchmark tuyệt đối bịa ra (không field
  reply-rate-target nào trong `weekly_kpis`) — recommendation chỉ so sánh TƯƠNG ĐỐI giữa các channel có đủ
  volume thật (≥5 tin nhắn, chênh lệch ≥10 điểm %). "Research" stage bị bỏ khỏi bottleneck có chủ đích — chưa
  có nguồn dữ liệu nào ghi nhận "đã research nhưng chưa liên hệ".

## Ghi chú Phase 1

Không có bảng nào ở trên yêu cầu tạo database mới, migration, hay implement 3 skill Phase 2 (intelligence/
opportunity/analytics). Đây thuần là tài liệu ranh giới để Phase 2/3 build đúng chỗ.

## Ghi chú Phase 2A (Deal Lifecycle — đã build)

`sme-opportunity` đã build (mục 5) — lightweight aggregation view, KHÔNG bảng mới. Pipeline code-level
LinkedIn reply → `sme-outreach` → `sme-engagement` taxonomy → `sme-opportunity` đã nối (mục 3, 4, 5).
Proposal pricing đã config-hoá (mục 7), fallback y hệt giá trị cũ.

## Ghi chú Phase 2B (Intelligence + Analytics — đã build)

`sme-intelligence` (mục 1) và `sme-analytics` (mục 8) đã build — cả 2 đều là facade/thin-aggregation, KHÔNG
tạo entity/database mới. Trong lúc audit trước khi build, phát hiện + fix 1 loạt bug client-side ở
`cosmo.go`/`cosmo_plan.go`/`cosmo_ai.go`/`opportunity.go` (COSMO `/v2/contacts/search` gửi sai request shape
— filter/pagination bị silently ignored, ảnh hưởng cả `daily-plan`/`sme-opportunity`; `/v1/intelligence/
vector-search/*` gọi sai path). Test isolation cũng đã fix (`SME_DATA_DIR` giờ điều khiển thật cả `sme.db`/
`connections.json`, không chỉ data JSON tĩnh). Xem `intelligence/SKILL.md`, `analytics/SKILL.md` cho chi tiết.

`sme-campaign` (CLI thực thi), LinkedIn accepted detection, autonomous outreach, Campaign performance thật
**vẫn CHƯA implement** — để Phase 2C/Future, không nằm trong phạm vi Phase 2A/2B.

## Ghi chú Phase 2C (Campaign Execution — đã build)

`sme-campaign` (mục 2) đã build — gọi thẳng COSMO Campaign API thật (`create/list/view/stats/add-template/
templates/activate/pause/delete`), audit kỹ trước khi build (capability matrix trong `campaign/SKILL.md`).
Không tạo core skill thứ 9, không migration DB mới (campaign/template/list-contact đều sống ở COSMO).
`sme-analytics` mở rộng thêm `analytics campaigns` (mục 8) — reuse `sent/reply/reply_rate` COSMO tính sẵn,
không tự tính lại.

Phát hiện quan trọng trong lúc audit: COSMO Campaign hoàn toàn không có khái niệm "channel" — `agent_id` luôn
là 1 email identity (Gmail/Outlook OAuth). Vì `sme-outreach` xác nhận (lại) KHÔNG có khả năng gửi LinkedIn
(chỉ đọc qua CDP `Runtime.evaluate`), channel LinkedIn của Campaign chỉ có thể là kế hoạch + tracking thủ
công — KHÔNG có "activate" thật cho LinkedIn. Agent email duy nhất của org đang `invalid Google grant` —
xác nhận live, KHÔNG bypass — nên channel email cũng chưa test được thật sự gửi trong phiên này (chỉ test
tới bước bị từ chối activate, đúng theo thiết kế an toàn).

LinkedIn accepted detection: audit lại (Phase 2C) không tìm thêm được signal đáng tin cậy nào ngoài những gì
đã biết ở Phase 1 (trang "Received invitations" trộn lẫn connection cá nhân + follow company) — vẫn
`unsupported/deferred`, không block Phase 2C.

Campaign engine chưa test được: gửi email thật (do Google auth), reply/sent event chi tiết theo từng người
nhận (chỉ có aggregate count qua COSMO GetByID) — cả 2 đều document rõ trong Known Issues, không giả lập số.
