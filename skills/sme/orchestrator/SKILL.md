---
name: sme-orchestrator
description: "GTM Orchestrator — CHỈ kích hoạt cho yêu cầu ĐA BƯỚC / MỤC TIÊU TỔNG QUÁT / MƠ HỒ cần phối hợp NHIỀU skill hoặc chưa rõ nên dùng skill nào (vd 'tìm công ty logistics ở Singapore rồi bắt đầu outreach', 'giúp anh build lại pipeline cho ngành F&B từ đầu', 'lo hết vụ khách này cho anh'). KHÔNG kích hoạt cho yêu cầu đơn giản đã rõ 1 skill cụ thể — những yêu cầu đó đi THẲNG vào skill tương ứng (vd 'viết proposal cho ABC' → sme-proposal trực tiếp, 'xem KPI outreach' → sme-analytics/sme-kpi trực tiếp, 'sync linkedin' → sme-outreach trực tiếp). Khi kích hoạt: hiểu goal → chia bước → chọn skill/tool theo thứ tự → tra approval policy trước khi thực thi hành động rủi ro → đánh giá kết quả từng bước trước khi sang bước kế."
metadata: { "openclaw": { "emoji": "🧭" } }
---

# SME Orchestrator — Goal → Plan → Route → Approve → Evaluate

Skill này **KHÔNG phải là điểm vào bắt buộc cho mọi request GTM**. Phần lớn request đơn giản đi thẳng vào
skill cụ thể mà không cần qua đây (xem "KHI NÀO KHÔNG KÍCH HOẠT" bên dưới). Orchestrator chỉ xử lý phần
**request phức tạp hơn 1 skill có thể tự quyết định** — đây là điểm khác biệt quan trọng nhất so với
`sme-reminder` (Briefing): reminder BÁO CÁO trạng thái theo lịch, orchestrator QUYẾT ĐỊNH cách xử lý 1 goal
mới do user đưa ra ngay lúc đó.

## KHI NÀO KÍCH HOẠT

- Goal cần **nhiều skill phối hợp theo thứ tự** (vd: tìm khách mới → outreach → theo dõi reply → chốt deal)
- Goal **mơ hồ**, chưa rõ nên dùng skill nào ("lo vụ khách X giúp anh", "làm sao để có thêm khách hàng ngành Y")
- User giao **1 mục tiêu tổng quát**, không phải 1 hành động cụ thể ("build lại chiến dịch cho quý này")
- Cần **quyết định workflow** (làm cái gì trước, cái gì sau, có cần hỏi user trước khi làm tiếp không)

## KHI NÀO KHÔNG KÍCH HOẠT — đi thẳng vào skill

| Ví dụ request | Đi thẳng vào |
|---|---|
| "Viết proposal cho ABC" | `sme-proposal` |
| "Xem KPI outreach tuần này" | `sme-kpi`/`sme-outreach funnel` (số đơn lẻ) hoặc `sme-analytics` (muốn tổng hợp + bottleneck) |
| "Sync linkedin" | `sme-outreach` |
| "Soạn bài FB tuần này" | `sme-marketing` |
| "Search contact tên X" | `sme-crm` |
| "Nhắc tôi lúc 6h chiều" | `sme-scheduler` |
| "Nhắc tôi" (báo cáo trạng thái hiện tại) | `sme-reminder` (Briefing) |
| "Deal này đang ở đâu / có nên gửi proposal chưa" (1 contact cụ thể) | `sme-opportunity` |
| "Phân tích công ty X, có đáng target không" (1 account cụ thể, chưa phải multi-step goal) | `sme-intelligence` |

**Nguyên tắc:** nếu 1 skill khác đã có trigger phrase khớp rõ ràng với request, để skill đó tự xử lý —
KHÔNG kích hoạt orchestrator "cho chắc". Tránh double-trigger (2 skill cùng trả lời 1 request).

## BƯỚC 1 — GOAL UNDERSTANDING

Trước khi làm gì, diễn giải lại goal thành 1 câu ngắn cho chính mình (không cần nói ra với user trừ khi
mơ hồ thật sự): "User muốn gì, kết quả cuối cùng trông như thế nào?"

Nếu sau bước này vẫn không rõ goal → hỏi lại NGẮN GỌN 1 câu, KHÔNG tự đoán rồi làm sai hướng.

## BƯỚC 2 — PLANNING (chia bước)

Chia goal thành chuỗi bước, mỗi bước gắn với 1 skill cụ thể. Ví dụ:

> "Tìm công ty logistics ở Singapore rồi bắt đầu outreach"
> 1. `sme-intelligence` (khi có) hoặc `sme-crm`/Apollo — research/enrich công ty logistics Singapore
> 2. `sme-campaign` — tạo campaign cold_reach cho segment này
> 3. `sme-outreach` — thực thi gửi connection request/message

Trình bày plan ngắn gọn cho user TRƯỚC khi thực thi bước có rủi ro (xem Approval Policy) — không cần hỏi
OK cho bước AUTO (vd bước research), chỉ dừng lại hỏi khi tới bước cần APPROVAL/HUMAN ONLY.

## BƯỚC 3 — SKILL ROUTING TABLE

Bảng định tuyến chính thức — nguồn duy nhất, các skill khác (kể cả `sme-reminder`) tham chiếu bảng này
thay vì tự giữ bản sao riêng:

| Nhu cầu | Skill | Ghi chú |
|---|---|---|
| Account nên target không, tại sao, buying signal, pain hypothesis (pre-conversation) | `sme-intelligence` | Facade — reuse ICP/relationship score + Apollo, xem `intelligence/SKILL.md`. KHÔNG tự gửi outreach |
| Campaign objective/segment/cadence/messaging, kích hoạt campaign | `sme-campaign` | CLI thực thi thật (Phase 2C) — `campaign create/activate/pause`, xem `campaign/SKILL.md`. Activate = APPROVAL, tạo ≠ activate |
| LinkedIn connection/message/reply, activity log | `sme-outreach` | |
| Reply analysis, intent/sentiment/objection, meeting prep/follow-up | `sme-engagement` | Chủ sở hữu Unified Taxonomy — xem `engagement/SKILL.md`. Qualification/stage/readiness → `sme-opportunity`, không còn ở đây |
| Qualification/opportunity stage/deal risk/next-step/proposal readiness/WON-LOST (deal-level) | `sme-opportunity` | Read-only aggregation view (`sme-cli opportunity view/risk-list`) — không phải bảng DB mới, xem `opportunity/SKILL.md` |
| Content strategy/generation, campaign asset | `sme-marketing` | |
| Sinh + gửi proposal | `sme-proposal` | |
| KPI/funnel/hiệu suất, bottleneck, recommendation | `sme-analytics` | Thin aggregation qua `sme-cli analytics summary` — xem `analytics/SKILL.md`. KHÔNG nhầm với `sme-bi` (agent `intel`, data khác hẳn) |
| Contact/company data, stage field, segment | `sme-crm` | Shared service, không phải business skill |
| Báo cáo Morning/EOD/Weekly, overdue, priority alert | `sme-reminder` (Briefing) | Không tự quyết định routing/approval |
| Nhắc theo giờ cụ thể user tự đặt | `sme-scheduler` | Time-based thuần, không fetch data |

## BƯỚC 4 — NEXT BEST ACTION (mức BASIC, Phase 1)

Khi cần gợi ý "nên làm gì tiếp theo" trong 1 workflow đa bước, gọi:

```bash
sme-cli cosmo daily-plan --mode all
```

Dùng NGUYÊN kết quả `classify()` đã có (cell `PROPOSAL_HOT`, `QUALIFIED_OPEN`, `NEW_*`...) làm gợi ý ưu
tiên — **KHÔNG** tự viết thêm logic scoring/ranking mới, **KHÔNG** mở rộng sâu hơn những gì `cosmo_plan.go`
đã trả về. Next Best Action nâng cao (kết hợp outreach/campaign/analytics thành 1 recommendation engine
thật) thuộc Phase 3.

## BƯỚC 5 — APPROVAL DECISION

Trước khi thực thi bất kỳ bước nào trong plan, tra `references/approval-policy.md` (AUTO / APPROVAL /
HUMAN ONLY). Bước AUTO → làm luôn, không hỏi. Bước APPROVAL → trình bày rõ sẽ làm gì, chờ user xác nhận
(1-key reply nếu có thể). Bước HUMAN ONLY → không tự quyết, trình bày lựa chọn có sẵn hoặc nói rõ cần
user tự quyết định.

## BƯỚC 6 — RESULT EVALUATION

Sau khi 1 bước trong plan hoàn tất (đặc biệt sau hand-off sang skill khác), kiểm tra kết quả có đúng kỳ
vọng không trước khi sang bước tiếp theo. Nếu có action đã thực thi đáng để đo hiệu quả, log qua
`sme-cli action-log suggest`/`action-log done` (đã có sẵn, tái dùng nguyên).

## PHÂN BIỆT VỚI CÁC SKILL KHÁC

- **`sme-reminder` (Briefing)** — chỉ báo cáo trạng thái theo lịch/khi được hỏi, KHÔNG quyết định workflow,
  KHÔNG quyết định approval, KHÔNG làm Next Best Action. Khi Briefing xong và user chốt 1 hành động ĐƠN
  GIẢN, RÕ RÀNG ngay sau đó → đi thẳng vào skill tương ứng (không cần quay lại orchestrator); chỉ quay lại
  orchestrator nếu hành động đó lại mơ hồ/đa bước.
- **8 core skill nghiệp vụ** (intelligence/campaign/outreach/engagement/opportunity/marketing/proposal/
  analytics) — tự thực thi domain logic của mình. Orchestrator không làm thay domain logic, chỉ quyết định
  THỨ TỰ và CÓ ĐƯỢC LÀM KHÔNG.
- **`sme-crm`** — shared service, không phải đối tượng orchestrator "điều phối", mà là nơi mọi skill khác
  (kể cả orchestrator) đọc/ghi contact data.
