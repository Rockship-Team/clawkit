---
name: sme-intelligence
description: "Pre-conversation, account-level research: who đáng target, tại sao, tại sao lúc này, pain có thể là gì. Trigger khi user hỏi 'phân tích công ty X', 'account Y có đáng outreach không', 'ai nên target ngành Z', 'công ty này có signal gì đáng chú ý'. Đây KHÔNG phải conversation-level intent/sentiment (dùng sme-engagement), KHÔNG phải deal-stage/risk của 1 deal đã có (dùng sme-opportunity), KHÔNG tự gửi outreach hay tạo campaign — chỉ trả về research + recommendation."
metadata: { "openclaw": { "emoji": "🔭" } }
---

# SME Intelligence — Pre-conversation Account Research (Phase 2B)

Intelligence trả lời **"nên target ai, tại sao, tại sao lúc này, pain có thể là gì"** ở mức
**account/company, TRƯỚC hoặc NGOÀI hội thoại** — khác hẳn `sme-engagement` (phân tích 1 reply cụ thể
trong hội thoại) và `sme-opportunity` (đánh giá 1 deal đã tồn tại trong CRM).

**Đây là FACADE, không phải engine chấm điểm mới.** Không viết lại ICP scoring, relationship scoring, hay
Apollo research — tất cả đều reuse nguyên các lệnh đã có (xem RANH GIỚI bên dưới).

## LỆNH CHÍNH

```bash
sme-cli intelligence account "<tên công ty>" [--org-id <apollo_org_id>]
```

- Không truyền `--org-id`: nếu Apollo trả nhiều công ty khớp tên → trả về danh sách candidate
  (`ambiguous: true`), **KHÔNG tự đoán** công ty nào đúng.
- Sau khi user chọn 1 candidate → gọi lại kèm `--org-id` để lấy report đầy đủ.

## NGUỒN DỮ LIỆU (REUSE, không tạo mới)

| Capability | Lệnh gốc | Vai trò trong Intelligence |
|---|---|---|
| ICP score | `sme-cli cosmo score-icp <contact_id>` | Chỉ có nếu company đã có contact trong CRM. Nếu COSMO chưa cấu hình ICP segment nào (`segments_evaluated=0`) → luôn trả `unknown`, KHÔNG coi priority_score=0 là "fit kém". |
| Relationship score | `sme-cli cosmo score-relationship <contact_id>` | Tương tự — chỉ có nếu đã có contact hiện hữu. |
| Account research | `sme-cli apollo search-company <tên>` | Nguồn signal chính: headcount growth, public trading, buying-intent (Apollo), ownership structure. |
| Contact/persona research | `sme-cli apollo search-people <tên_công_ty> [seniorities]` | Danh sách title thực tế tại công ty — so khớp với danh sách "typical buyer persona" (CEO/COO/CTO/VP Operations/...). |
| Enrichment (full identity) | `sme-cli apollo enrich-person`, `sme-cli cosmo enrich` | Delegate — Intelligence không tự enrich thêm. |
| Relevant Rockship offering | `sme-cli proposal pricing` (tiers hiện có) | Dùng nguyên tier Starter/Pro/Enterprise đã có, không tạo danh sách offering riêng. |

## BUYING SIGNAL — account-level, KHÔNG phải conversation-level

**Phân biệt rõ:** signal ở đây là tín hiệu account/company (hiring, expansion, public activity...) — khác
hẳn buying signal trong hội thoại của `sme-engagement` (interest/objection khi khách đã reply).

Mỗi signal LUÔN có đủ: `signal_type`, `evidence` (dữ liệu thật, trích dẫn nguyên văn con số/field), `source`
("Apollo"), `relevance`, `confidence`. **Không có evidence → không tạo signal.** Nếu Apollo/COSMO không trả
field nào đáng chú ý → `signals: []`.

Signal type hiện hỗ trợ (dựa trên field thật Apollo trả về, xác nhận qua audit Phase 2B):
- `headcount_growth` — chỉ tạo khi tăng trưởng nhân sự (6/12/24 tháng) ≥ 5%. Dưới ngưỡng này coi là nhiễu vì
  Apollo trả `0` cho cả 2 trường hợp "không tăng trưởng" VÀ "field không có dữ liệu" — không phân biệt được.
- `public_company_activity` — công ty niêm yết công khai (có mã cổ phiếu).
- `buying_intent` — Apollo tự phát hiện tín hiệu quan tâm (`has_intent_signal_account`), nhưng KHÔNG biết cụ
  thể đang tìm sản phẩm/vấn đề gì → confidence thấp dù relevance cao.
- `ownership_structure` — thuộc 1 tập đoàn mẹ.

**KHÔNG hỗ trợ** (chưa có nguồn dữ liệu thật): tin tuyển dụng cụ thể theo role, tin tức mở rộng thị trường,
thay đổi lãnh đạo, technology stack. Nếu user hỏi các loại signal này → trả lời trung thực "chưa có nguồn dữ
liệu cho loại signal này", KHÔNG suy diễn từ dữ liệu khác.

## PAIN HYPOTHESIS — luôn là HYPOTHESIS, không phải fact

Mỗi hypothesis là 1 template cố định gắn với đúng 1 loại signal (không tự viết prose mới), luôn kèm
`supporting_evidence` trích dẫn lại evidence gốc, `relevant_offering`, `confidence`. Câu chữ luôn có marker
rõ ràng ("HYPOTHESIS", "chưa xác nhận qua hội thoại") để KHÔNG bị hiểu nhầm là fact đã xác nhận.

Ví dụ đối chiếu:
- FACT (từ signal): "Wilson Sons tăng 5.1% nhân sự trong 12 tháng (Apollo)."
- HYPOTHESIS (suy luận): "Có thể đang chịu áp lực vận hành do mở rộng nhân sự nhanh — CHƯA xác nhận qua hội thoại."

## QUALIFICATION — high/medium/low/insufficient_data

Đánh giá dựa trên: có contact hiện hữu trong CRM không, ICP score đã tính chưa (không phải chỉ "có priority_score"
mà phải `segments_evaluated > 0`), persona điển hình có khớp không, có signal đáng chú ý (medium/high relevance)
không. **`unknown` KHÔNG BAO GIỜ bị coi là "disqualified"** — 1 cold account thiếu dữ liệu chỉ dừng ở
`insufficient_data`, không phải mức thấp nhất mang tính phủ định.

BANT (Budget/Authority/Timeline) không bắt buộc cho cold account — COSMO/Apollo không có field này, luôn trả
`unknown` nếu chưa có evidence tường minh từ hội thoại thật.

## RANH GIỚI

| Câu hỏi | Skill xử lý |
|---|---|
| Account này có đáng target không, pain có thể là gì | **`sme-intelligence`** (skill này) |
| Đã có reply rồi, phân loại intent/sentiment/objection | `sme-engagement` |
| Deal này đang ở stage nào, risk gì, proposal readiness | `sme-opportunity` |
| Thực thi gửi LinkedIn/outreach | `sme-outreach` |
| Tạo/kích hoạt campaign | `sme-campaign` |
| Contact identity, business_stage lưu trữ | `sme-crm` |

## VÍ DỤ

**User:** "Phân tích công ty Wilson Sons giúp em, có đáng outreach không?"
→ `sme-cli intelligence account "Wilson Sons"` (kèm `--org-id` nếu đã biết) → trình bày signals + pain
hypotheses + qualification + recommended_angle. **KHÔNG tự gửi outreach** — chỉ trả recommendation, hỏi user
có muốn chuyển sang `sme-campaign`/`sme-outreach` không.

**User:** "Account X đáng outreach, chuẩn bị angle giúp em"
→ Dùng `recommended_angle` đã có từ lần gọi trước (hoặc gọi lại nếu cần) → trình bày angle, **KHÔNG** tự
kích hoạt campaign hay gửi message — dừng lại ở đề xuất.
