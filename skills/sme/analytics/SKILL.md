---
name: sme-analytics
description: "Báo cáo tổng hợp KPI + outreach funnel + so sánh hiệu suất kênh + bottleneck trong pipeline + recommendation dựa trên số liệu đo được. Trigger khi user hỏi 'hiệu suất outreach thế nào', 'kênh nào tốt hơn', 'bottleneck ở đâu', 'nên cải thiện gì'. Đây là THIN AGGREGATION layer — KHÔNG phải analytics platform mới, không tự tính số khác với sme-kpi/sme-outreach funnel. KHÔNG PHẢI sme-bi (agent intel khác, data khác hẳn, KHÔNG merge)."
metadata: { "openclaw": { "emoji": "📊" } }
---

# SME Analytics — Thin Aggregation (Phase 2B + Campaign Analytics Phase 2C)

Analytics **KHÔNG tính số liệu mới** — mọi con số đều reuse trực tiếp từ `sme-cli kpi`/`sme-cli outreach
funnel`/COSMO `business_stage` (qua cơ chế `fetchAllContacts` đã dùng ở `cosmo daily-plan`/`sme-opportunity`).
Nếu số trong Analytics khác với gọi `kpi team`/`outreach funnel` riêng lẻ — đó là bug, báo lại ngay.

## LỆNH CHÍNH

```bash
sme-cli analytics summary [--days N] [--week YYYY-Www] [--member NAME] [--max-pages N]
sme-cli analytics campaigns [--max-pages N]     # Phase 2C — campaign metrics thật
```

## OUTPUT FIELDS

| Field | Nguồn | Ghi chú |
|---|---|---|
| `kpi_summary.targets` | `kpi team` (nguyên) | |
| `kpi_summary.actuals` | `kpi actual` (nguyên) | Nếu bảng `interactions` local chưa có data → trả string `insufficient_data`, KHÔNG trả mảng rỗng giả |
| `outreach_funnel` | `outreach funnel` (nguyên) | |
| `channel_comparison` | Pivot lại `outreach_funnel` theo channel | `reply_rate` chỉ tính khi có `messages > 0`, nếu không → `"insufficient_data"` |
| `bottleneck` | Outreach/Reply từ `outreach_events`; Qualified/Proposal từ COSMO `business_stage` | **KHÔNG có stage "Research"** — hệ thống hiện chưa có nguồn dữ liệu này, cố tình bỏ qua thay vì giả 0 |
| `recommendations` | So sánh reply_rate GIỮA các channel (relative), không so với benchmark tuyệt đối bịa ra | Chỉ xuất hiện khi có ≥2 channel đủ volume (≥5 tin nhắn) VÀ chênh lệch ≥10 điểm % |
| `campaign_performance` | `sme-cli analytics campaigns` (xem bảng riêng bên dưới) — KHÔNG dùng `analytics summary` cho campaign | Campaign Engine đã build (Phase 2C) |

## CAMPAIGN ANALYTICS (Phase 2C) — `sme-cli analytics campaigns`

| Field | Nguồn | Ghi chú |
|---|---|---|
| `campaigns_created`, `by_status`, `active_campaigns` | `GET /v1/campaigns` (COSMO thật) | |
| `messages_sent`, `replies`, `reply_rate` | Sum `sent`/`reply` COSMO trả sẵn per-campaign (GetByID/List) | KHÔNG tự tính lại — 0% khi denominator thật = 0, không phải `insufficient_data` (khác `channel_comparison` của `analytics summary`, ở đây `messages_sent` luôn ≥0 sản phẩm của COSMO tính sẵn) |
| `channel_performance` | Đọc `cmetadata.client.channel` per-campaign (qua `campaignChannel()` — cùng hàm `campaign activate` dùng) | Không tự suy channel từ tên/playbook |
| `qualified_opportunities`, `proposals` | Reuse `analyticsBottleneck`'s COSMO `business_stage` scan (KHÔNG chạy lại lần 2) | |

**Chưa hỗ trợ (audit Phase 2C, để rõ chứ không giả lập):** sent/reply chi tiết theo
từng recipient (chỉ có aggregate count qua COSMO GetByID, không có event feed
per-recipient trong phạm vi API đã audit) — nếu cần drill-down từng người, phải audit
sâu hơn `relations.CampaignWithEmails`/`CampaignWithConversations` (chưa làm ở Phase 2C).

## QUY TẮC "KHÔNG BỊA SỐ"

- Reply rate không tính được (thiếu denominator) → `insufficient_data`, KHÔNG phải `0%`.
- Recommendation KHÔNG so với 1 con số benchmark tuyệt đối bịa ra (vd "reply rate nên >10%") — hệ thống hiện
  chưa có target reply-rate nào được cấu hình (`weekly_kpis` không có field này). Thay vào đó so sánh
  TƯƠNG ĐỐI giữa các channel đang có — chỉ 1 channel có volume thật → KHÔNG có gì để so sánh → không đưa
  recommendation.
- Bottleneck chỉ hiện stage có nguồn dữ liệu thật — không cố hiện đủ "Research → Outreach → Reply →
  Qualified → Proposal" nếu 1 stage không đo được.
- `analytics campaigns` chỉ đọc số COSMO đã tính (sent/reply/reply_rate) — không tự suy diễn per-recipient
  detail khi API không cung cấp.

## RANH GIỚI

| Câu hỏi | Skill xử lý |
|---|---|
| Hiệu suất outreach/KPI/bottleneck tổng hợp | **`sme-analytics`** (skill này) |
| Target KPI tuần này bao nhiêu | `sme-kpi` |
| Số liệu outreach hôm nay/tuần này (raw) | `sme-outreach` |
| Account này có đáng target không | `sme-intelligence` |
| `sme-bi` (agent `intel`, bảng `sales` riêng) | **KHÔNG merge** — data nguồn khác hẳn, xem `GTM_ARCHITECTURE.md` |

## VÍ DỤ

**User:** "Hiệu suất outreach tháng này thế nào, có bottleneck ở đâu không?"
→ `sme-cli analytics summary --days 30` → trình bày channel_comparison + bottleneck + recommendations (nếu
có), nói rõ nếu 1 phần "insufficient_data" thay vì lờ đi.
