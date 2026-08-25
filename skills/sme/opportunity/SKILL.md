---
name: sme-opportunity
description: "Deal-level lightweight view: qualification status, opportunity stage, deal risk, next step, proposal readiness, WON/LOST interpretation cho 1 contact/deal cụ thể. Trigger khi user hỏi 'deal này đang ở đâu', 'có nên gửi proposal cho X chưa', 'contact Y rủi ro gì', 'ai đang low priority/stale'. Đây là READ-ONLY AGGREGATION view (KHÔNG có bảng DB riêng, KHÔNG phải nguồn dữ liệu mới) — tổng hợp business_stage/next_step từ COSMO (qua sme-crm) + risk cell từ cosmo_plan.go + classification LinkedIn từ sme-outreach. KHÔNG phải nơi lưu contact/stage (dùng sme-crm), KHÔNG phải nơi phân tích reply (dùng sme-engagement — Opportunity chỉ ĐỌC kết quả của Engagement, không tự phân loại)."
metadata: { "openclaw": { "emoji": "🎯" } }
---

# SME Opportunity — Deal Lifecycle View (lightweight, Phase 2A)

Opportunity là **long-term owner** của: qualification, opportunity stage, deal risk, next step, proposal
readiness, và cách diễn giải WON/LOST. `sme-engagement` KHÔNG còn giữ vai trò này lâu dài — xem
`engagement/SKILL.md` "RESPONSIBILITY BOUNDARY" (đã update để trỏ về đây).

**Đây KHÔNG phải 1 entity/database mới.** Không có bảng `opportunities`. Mọi field đều aggregate từ dữ
liệu đã tồn tại:
- `business_stage`, `next_step`, `stage_label`, `relationship.*` — COSMO, qua `/v2/contacts/search` (cùng
  cơ chế `cosmo_plan.go` đã dùng cho `daily-plan`)
- Risk cell — reuse trực tiếp `classify()`/`buildPlanCells` của `cosmo_plan.go` (KHÔNG viết lại logic này)
- LinkedIn reply classification — đọc từ bảng `outreach_reply_classifications` do `sme-outreach` ghi (qua
  pipeline Outreach → Engagement taxonomy, xem `outreach/SKILL.md`)

## LỆNH CHÍNH

```bash
sme-cli opportunity view <contact_id_hoặc_tên/công_ty>   # deep view cho 1 contact/deal
sme-cli opportunity risk-list [--max-pages N]            # batch: ai đang LOW_PRIORITY/stale
```

`opportunity view` tự resolve contact: nếu input là UUID → chỉ match chính xác ID đó (không đoán); nếu là
tên/công ty và có nhiều kết quả khớp → trả về danh sách candidate để user chọn, KHÔNG tự ý chọn 1 cái.

## OUTPUT FIELDS

| Field | Nguồn | Ghi chú |
|---|---|---|
| Contact/Company | COSMO | |
| Current Stage | `business_stage` + `stage_label` (COSMO) | |
| Qualification Status | suy ra từ `business_stage` (not_started/in_progress/qualified/closed_lost) | KHÔNG phải scoring model mới — chỉ map lại stage hiện có |
| Known Pain/Need | — | **Luôn `unknown`** — COSMO chưa có field này (xác nhận qua audit Phase 2), không suy diễn |
| Next Step | `next_step` (COSMO) | `unknown` nếu rỗng |
| Risk | merge `cosmo_plan.go` cell + LOW_PRIORITY override (xem dưới) | 1 nguồn duy nhất, không còn tách rời giữa Go code và rule trong `reminder/SKILL.md` |
| Proposal Readiness | xem QUY TẮC PROPOSAL READINESS | KHÔNG BAO GIỜ tự "ready" chỉ từ 1 reply tích cực |
| Evidence | list cụ thể mọi field đã dùng để kết luận | Nếu thiếu → liệt kê rõ đang thiếu gì, KHÔNG lấp bằng suy đoán |
| Recommended Action | action template của cell tương ứng (`cosmo_plan.go` `cellTemplates`) | Reuse nguyên, không viết thêm |
| BANT (budget/authority/timeline) | — | **Luôn `unknown`** trừ khi có evidence tường minh — COSMO không có field này |

## QUY TẮC PROPOSAL READINESS — KHÔNG auto-jump

`interested` + sentiment `positive` (dù từ Gmail hay LinkedIn) **KHÔNG đồng nghĩa proposal-ready**.

- Nếu `business_stage` đã là `PROPOSAL`/`WON` → readiness = `ready` (đã được xác lập từ trước bởi quy trình
  hiện có, không phải Opportunity tự quyết).
- Nếu chỉ có tín hiệu quan tâm (`interested`/`requesting_info`) mà CHƯA có xác nhận khách muốn nhận báo
  giá/proposal cụ thể → readiness = `not_ready`, nêu rõ lý do + bước qualification/discovery còn thiếu.
- KHÔNG BAO GIỜ tự chuyển `QUALIFIED → PROPOSAL` từ view này — Opportunity chỉ ĐÁNH GIÁ readiness, việc đổi
  stage vẫn qua `sme-crm` (delegate PATCH), và CHỈ sau khi user xác nhận.

## RISK — 1 nguồn duy nhất (không còn duplicate)

Trước đây risk bị tách giữa 2 nơi: `cosmo_plan.go`'s cell classification (PROPOSAL_HOT/STUCK/GHOST,
ENGAGED_COLD, LOST_REVIVE, WON_CHECKIN...) và `reminder/SKILL.md`'s rule LOW_PRIORITY tính ở render-time.
Từ Phase 2A, `sme-cli opportunity view`/`risk-list` là nơi DUY NHẤT merge 2 cái này:
- Base risk = cell từ `classify()` (nguyên trạng, không đổi logic gốc)
- LOW_PRIORITY override = khi outreach đang in-progress (PROPOSAL_HOT/STUCK/GHOST, ENGAGED_WARM/COLD,
  QUALIFIED_OPEN, CAMPAIGN_SENT_NO_REPLY) VÀ contact đã im lặng nhiều lần không reply

`sme-reminder`'s "low priority co ai" / "ai im lang lau" trigger giờ gọi `sme-cli opportunity risk-list`
thay vì tự tính riêng — xem `reminder/SKILL.md`. Cell-based classification bên trong `cosmo daily-plan`
(dùng cho briefing tự động mỗi 10 phút) **giữ nguyên, không đổi** — risk-list là 1 lệnh riêng, chạy on-demand,
không nằm trong vòng lặp PIPELINE_WATCH (tránh N+1 API call vào hot path).

## RANH GIỚI

| Câu hỏi | Skill xử lý |
|---|---|
| Contact identity, `business_stage` lưu trữ/PATCH | `sme-crm` |
| Phân loại reply (intent/sentiment/objection), soạn reply, meeting prep | `sme-engagement` |
| Deal này ở stage nào, risk gì, có nên gửi proposal chưa | **`sme-opportunity`** (skill này) |
| Sinh + gửi proposal thật | `sme-proposal` (Opportunity chỉ đánh giá readiness, không tự generate) |
| LinkedIn activity thô (connection/message counts) | `sme-outreach` |

## VÍ DỤ

**User:** "Deal Wilson AI đang ở đâu, có nên gửi proposal chưa?"
→ `sme-cli opportunity view "Wilson AI"` (hoặc contact_id nếu đã biết) → trình bày Current Stage / Next
Step / Risk / Proposal Readiness + Evidence, KHÔNG tự quyết định gửi proposal thay user.

**User:** "Ai đang bị deprioritize / im lặng lâu?"
→ `sme-cli opportunity risk-list` → liệt kê, hỏi user muốn revive hay chuyển LOST (delegate `sme-crm`).
