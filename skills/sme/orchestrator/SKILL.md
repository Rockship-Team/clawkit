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

### GOAL PERSISTENCE (Phase 3A) — khi nào lưu lại goal để nhớ về sau

Trước Phase 3A, mọi goal user đưa ra chỉ tồn tại trong đúng lượt chat đó — hôm sau agent không biết lại.
Giờ có `sme-cli goal set/list/view/complete/cancel` (xem `_engine/cmd/goal.go`) để lưu goal **1 bảng nhỏ
riêng** — KHÔNG nhét vào `weekly_kpis` (chỉ hỗ trợ 5 metric cố định theo tuần, không hợp cho goal tự do).

**Nhận diện goal nên persist** — user đưa ra 1 outcome có ít nhất 1 trong 2:
- **measurable target** (số cụ thể: "5 qualified leads", "10 contract")
- **deadline** (mốc thời gian: "tháng này", "cuối quý")

→ Tự động persist, **AUTO, không cần hỏi xin phép** (đây là internal action, không phải external send):

```bash
sme-cli goal set --text "<nguyên văn goal>" --metric <qualified_leads|proposals|contracts|custom> --target N --deadline YYYY-MM-DD
```

Chọn `--metric`:
- Goal nói rõ "qualified lead(s)" → `qualified_leads`
- Goal nói rõ "proposal" → `proposals`
- Goal nói rõ "contract"/"hợp đồng"/"chốt deal" → `contracts`
- Không khớp loại nào trong 3 loại trên (vd "10 event", "20 demo") → `custom` — vẫn lưu target/deadline,
  nhưng `goal check` sẽ trả progress = `unknown` vì chưa có nguồn dữ liệu tin cậy cho loại này (KHÔNG suy
  diễn/bịa số — đây là kỷ luật bắt buộc của Phase 3A).
- Không tự tính `--deadline` nếu user chỉ nói "tháng này" mà không rõ ngày cụ thể → dùng ngày cuối tháng
  hiện tại (deterministic, không đoán mơ hồ hơn).

**Baseline (Phase 3B):** `goal set` tự động chụp lại số lượng hiện có (vd 4 QUALIFIED) NGAY LÚC TẠO GOAL làm
baseline — COSMO không có cách nào biết chính xác "khi nào" 1 contact chuyển sang stage này (đã audit source
COSMO backend, xác nhận không có), nên đây là cách duy nhất đáng tin cậy để KHÔNG tính nhầm số có sẵn từ
trước thành progress của goal mới. Vì vậy nếu CRM đã có 4 qualified trước khi tạo goal "5 qualified leads",
`goal check`/`next-action` sẽ báo progress = 0/5 (KHÔNG phải 4/5) — đây là hành vi ĐÚNG, không phải bug.

**Goal MƠ HỒ** (không có target lẫn deadline, vd "tăng doanh số", "làm marketing tốt hơn") → **KHÔNG tự
bịa số/deadline**. Hỏi lại 1 câu ngắn ("Cụ thể là bao nhiêu / tới khi nào?"). Nếu user vẫn không cho số cụ
thể sau khi hỏi → có thể lưu với `--metric custom --target 0` (không target), không được tự chọn 1 con số
thay user.

User **không cần biết** `sme-cli goal` tồn tại — đây là internal action, chỉ cần confirm ngắn gọn: "Em ghi
nhận goal này rồi, sẽ theo dõi cho anh."

### GOAL-AWARE NBA (Phase 3B) — khi user hỏi "giờ nên làm gì" mà đang có goal active

Trigger: user hỏi 1 trong các dạng sau **VÀ** đang có ít nhất 1 goal `status=active`:
- "giờ nên làm gì?" / "tiếp theo làm gì?"
- "tình hình goal sao rồi?" / "goal của tôi đang tới đâu?"
- "hôm nay ưu tiên gì?"

Flow (AUTO — không cần hỏi xin phép để tính toán/đề xuất, chỉ dừng lại khi tới bước gửi/activate thật):

```bash
sme-cli goal list --status active           # nếu chưa biết goal_id
sme-cli goal next-action <goal_id>
```

`goal next-action` đã tự làm: check progress (baseline vs current, KHÔNG bao giờ tính all-time count làm
progress — xem `goal.go`), phát hiện bottleneck (Case A-E, xem code comment trong `goal_nba.go` để biết thứ
tự ưu tiên), trả về `recommended_action` + `recommended_skill` + `approval_required`.

Trình bày cho user:
1. Progress hiện tại (current/target, KHÔNG phải all-time count).
2. Bottleneck phát hiện được (nếu có).
3. Recommended action — **nếu `approval_required=true`, KHÔNG tự thực thi phần external send/activate**, chỉ
   chuẩn bị (research qua Intelligence, draft campaign) rồi dừng lại hỏi. Nếu `approval_required=false`
   (internal-only, vd đề xuất xem lại proposal risk-list), có thể trình bày luôn không cần hỏi OK trước.
4. Nếu `detected_bottleneck` là "none — goal đã đạt target" → hỏi user có muốn `goal complete <id>` không,
   **KHÔNG tự động complete**.
5. Nếu progress = "unknown" → nói rõ giới hạn (metric này chưa đo được), KHÔNG bịa số.

**Chưa làm ở Phase 3B** (để tương lai): tự động chain thực thi Intelligence→Campaign mà không dừng hỏi;
dùng action_rate/kết quả NBA trước đó để tự thay đổi recommendation tương lai (đó là learning, thuộc
Phase 3C/Future — hiện `goal next-action` chỉ log vào ActionLog để có traceability, không tự học từ đó).

### ACTIONLOG CONTINUITY (Phase 3C) — tránh lặp lại đề xuất y hệt

`goal next-action`/`goal review-check` giờ trả thêm 3 field: `previously_suggested`, `previous_status`,
`continuity_note` (xem `checkGoalNBAContinuity` trong `goal_nba.go`) — đây là 1 lookup ActionLog đơn thuần
(action_text lần gần nhất cho đúng goal này, có giống recommendation hiện tại không), **KHÔNG phải
self-learning, KHÔNG đổi decision tree**. Khi trình bày recommendation cho user:

- Nếu `continuity_note` không rỗng và `previous_status = "done"` → recommendation này y hệt lần trước đã
  làm xong, chưa có evidence mới → **đừng trình bày như 1 priority mới**, chỉ nhắc ngắn ("việc này em từng
  báo rồi, anh đã xử lý, hiện chưa có gì mới") hoặc bỏ qua nếu không ai hỏi trực tiếp.
- Nếu `previous_status = "skipped"` → có thể vẫn đề xuất lại (KHÔNG tự động bỏ qua), nhưng PHẢI nhắc rõ đã
  từng bị skip trước đó, để user tự quyết định có đổi ý không — không lặp lại mù quáng như thể lần đầu.
- Nếu `previous_status = "pending"` → nhắc việc này vẫn đang chờ xử lý từ lần trước, không tạo thêm bản ghi
  ActionLog trùng.
- Nếu rỗng (không có gì trước đó, hoặc recommendation khác lần trước) → trình bày bình thường như Phase 3B.

### MEMORY-AWARE NBA (Phase 3C) — dùng context/preference đã học, không đổi CRM

Trước khi trình bày 1 recommendation liên quan tới **1 account/contact cụ thể được nêu tên, chiến lược
campaign/message, hoặc 1 action đang định lặp lại** — gọi `memory_search` (native tool memory-core, KHÔNG
phải sme-cli) với query CỤ THỂ (vd tên account + "outreach"/"preference"/"hold"/"correction"), KHÔNG dump
toàn bộ memory vào context.

- **Có kết quả liên quan** (vd "ABC hẹn liên hệ lại tháng 9", "user không muốn follow-up dồn dập") → điều
  chỉnh CÁCH trình bày/đề xuất recommendation cho đúng — vd không đề xuất outreach ABC ngay, hoặc đổi tone
  follow-up sang nhẹ nhàng hơn. **KHÔNG được dùng để đổi `business_stage` hay bất kỳ field CRM nào** — đó là
  CRM's job (`sme-crm`), memory chỉ là context/lý do, không phải fact.
- **Không có kết quả liên quan** → tiếp tục bình thường, y hệt Phase 3B, không cần nói gì thêm.
- **`memory_search` lỗi/timeout/không khả dụng** → **PHẢI tiếp tục luồng chính bình thường**, không chờ,
  không crash, không hỏi lại user — chỉ đơn giản là recommendation không có thêm memory context lần này.
  Memory KHÔNG BAO GIỜ được phép chặn GTM flow.

**Memory WRITE — khi nào ghi lại:** Chỉ ghi khi trong hội thoại có 1 trong các tín hiệu rõ ràng: preference
tường minh của user, correction tường minh, "đừng liên hệ X cho tới khi...", 1 bài học campaign có bằng
chứng thật, hoặc lý do quyết định quan trọng sẽ cần lại sau. **KHÔNG ghi mọi reply/activity thường
xuyên** — những cái đó đã có sẵn ở CRM/ActionLog/event data, ghi lại vào Memory là trùng lặp không cần
thiết. Dùng ĐÚNG format/vị trí đã định nghĩa sẵn trong `skills/learn/SKILL.md` (per-user file
`~/workspace-gtm/memory/users/{username}.md`, team file `team-bd.md`, cập nhật `MEMORY.md` index) — KHÔNG
tự nghĩ ra format/writer mới.

**Ownership quyết định (Phase 3C audit — xem thêm câu trả lời đầy đủ trong báo cáo Phase 3C):** `sme-learn`
tồn tại trên disk (`skills/learn/SKILL.md`) nhưng KHÔNG có trong danh sách skill active của agent `gtm`
(`openclaw.json agents.list[].skills` không có `"learn"`) — đây là gap vận hành (bị rớt khỏi config, không
phải bị thay thế có chủ đích), memory-core plugin (native, đã bật, `dreaming` REM phase enabled) vẫn tự động
promote các pattern LẶP LẠI ≥3 lần vào MEMORY.md, nhưng KHÔNG bắt được 1 câu nói tường minh, quan trọng,
chỉ nói 1 lần (memory-core promotion cần `minRecallCount=3`). Quyết định: **IMPROVE, không blanket
reactivate** — thay vì bật lại `sme-learn` như 1 background behavior chạy sau MỌI turn của TẤT CẢ skill
(blast radius rộng, khó verify trong 1 phase), phần ghi memory tường minh giờ được kích hoạt trực tiếp ngay
tại đây (orchestrator, trong luồng Goal/NBA) và trong GOAL_REVIEW MODE (`reminder/SKILL.md`) — dùng ĐÚNG
format `sme-learn` đã định nghĩa, không phải 1 writer song song mới.

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
| User đặt 1 mục tiêu có target/deadline ("tháng này cần X"), hoặc hỏi lại goal đang track | `sme-orchestrator` (persist qua `sme-cli goal set/view` trực tiếp) | Xem "GOAL PERSISTENCE" ở BƯỚC 1 — KHÔNG có skill riêng cho goal, đây là 1 phần của orchestrator |

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
