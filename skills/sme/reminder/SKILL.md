---
name: sme-reminder
description: "Proactive GTM Briefing cho 5 lifecycle SME: marketing (content cadence slips), engagement (daily BD outreach qua COSMO), sales (proposal follow-up / meeting prep), event (prep checklist 1-3 ngay truoc + post-event thank-you overdue), outreach (LinkedIn connection/message activity qua sme-outreach). Khi user noi 'nhac toi', 'ai can follow-up', 'hom nay outreach ai', 'event sap toi can lam gi', 'content tuan nay con gi', 'hom nay outreach duoc bao nhieu', 'tom tat tin nhan hom nay' → fetch live data, categorize, BAO CAO trang thai. KHONG phai memory search, KHONG phai cron scheduler, KHONG quyet dinh workflow/approval (xem sme-orchestrator)."
metadata: { "openclaw": { "emoji": "🎯" } }
---

# SME Reminder — Proactive GTM Briefing

Skill này là **Briefing** (Morning Brief / EOD Brief / Weekly Review / overdue detection / priority alert
/ proactive reminder) — KHÔNG phải orchestrator. Nó fetch live state từ các lifecycle + trình bày rõ ràng
cho user, nhưng **KHÔNG tự quyết định workflow đa bước, KHÔNG tự quyết định approval, KHÔNG làm Next Best
Action** — 3 việc đó thuộc về `sme-orchestrator` (skill riêng).

**Sở hữu (Own):**
- Morning Brief / EOD Brief / Weekly Review
- Overdue detection (event chưa attendee, proposal stuck, follow-up quá hạn)
- Priority alerts (PIPELINE_WATCH)
- Proactive reminders theo lịch (cron)
- KPI/status summary

**KHÔNG sở hữu (Does NOT own) — đã MOVE sang `sme-orchestrator`:**
- Skill routing cho goal đa bước/mơ hồ
- Workflow planning
- Approval decision cho hành động rủi ro
- Next Best Action logic

Khi user, ngay sau khi xem briefing, chốt 1 hành động **đơn giản, rõ ràng** (vd "ok soạn proposal cho X")
→ vẫn có thể đi thẳng vào skill tương ứng (xem TRIGGER section, không cần vòng qua orchestrator). Chỉ khi
hành động đó lại mơ hồ/đa bước thì mới cần `sme-orchestrator`.

5 lifecycle mà skill theo dõi (chỉ để BÁO CÁO, không phải để quyết định phải làm gì):

| Lifecycle | Data source | Hand-off toi |
|---|---|---|
| **Engagement** (BD outreach daily qua COSMO) | `sme-cli cosmo daily-plan` | `sme-engagement` / `sme-crm` |
| **Sales** (proposal / meeting) | `sme-cli cosmo daily-plan` cells `PROPOSAL_*` / `MEETING_*` | `sme-proposal` / `sme-engagement` |
| **Event** (prep + post) | `sme-cli event list` + event metadata | `sme-campaign` (event flow A) |
| **Marketing** (content cadence) | `sme-cli social upcoming` | `sme-marketing` |
| **Outreach** (LinkedIn connection/message activity) | `sme-cli outreach today` / `pending` / `list` / `funnel` | `sme-outreach` |

### ⚠️ Phân biệt "outreach" — 2 nghĩa khác nhau trong skill này

Từ "outreach" xuất hiện ở CẢ Engagement lẫn lifecycle mới "Outreach", dễ nhầm. **Điểm phân biệt mấu chốt:
có chữ "ai" (hỏi NGƯỜI) hay không (hỏi SỐ):**

- **"hôm nay outreach ai" / "ai cần liên hệ hôm nay" / "nên outreach ai"** (có chữ "ai" — câu hỏi GỢI Ý
  chưa biết làm gì, hỏi bot đề xuất NGƯỜI nào cần liên hệ) → đây là **Engagement**, dùng
  `sme-cli cosmo daily-plan` như cũ, KHÔNG liên quan `sme-outreach`. **Test case đã xác nhận SAI trước
  đây:** "hôm nay outreach ai" từng bị route nhầm sang LinkedIn data — PHẢI trả lời bằng gợi ý contact từ
  COSMO (vd "Cinex — Trần Minh, CTO, proposal 4 ngày chưa reply"), TUYỆT ĐỐI KHÔNG trả lời bằng số
  connection request/tin nhắn LinkedIn.
- **"hôm nay outreach được bao nhiêu người" / "tóm tắt tin nhắn hôm nay" / "sync linkedin" / "connection request tuần này" / "ai đã reply"** (câu hỏi SỐ LIỆU HOẠT ĐỘNG đã làm, đặc biệt nhắc LinkedIn/connection/tin nhắn) → đây là lifecycle **Outreach**, dùng `sme-cli outreach ...`, hand-off `sme-outreach` khi cần chi tiết/action.

Nếu không chắc câu hỏi thuộc loại nào, ưu tiên hỏi lại 1 câu ngắn thay vì đoán sai lifecycle.

## TRIGGER — Match rong, khong wait clarification

Kich hoat NGAY khi message khop bat ky pattern:

### Engagement / Sales triggers (tieng Viet)

- Chua `nhac toi` (bat ky: "nhac toi", "nhac toi contacts", "nhac toi khach hang", "nhac toi outreach", "nhac toi follow up")
- Chua `ai can follow up` / `ai dang stuck` / `ai can lien he`
- Chua `hom nay outreach` / `hom nay nen lien he` / `hom nay lam gi`
- Chua `con contact nao chua lam` / `stale leads`
- Chua `outreach ai` / `lien he ai`
- Chua `low priority` / `deprioritize` / `ai im lang lau` / `ai khong reply` (→ render cell `LOW_PRIORITY`, xem QUY TAC LOW_PRIORITY ben duoi)

### Event triggers

- Chua `event sap toi` / `event tuan nay` / `event can lam gi`
- Chua `sau event` + "chua gui" / "chua thank-you"
- Chua `event nao con viec`

### Marketing triggers

- Chua `content tuan nay` / `bai dang tuan nay`
- Chua `post nao chua` / `slot nao trong`
- Chua `marketing hom nay` / `content hom nay`

### Outreach triggers (LinkedIn — xem ⚠️ phan biet o tren truoc khi match)

- Chua `hom nay outreach duoc bao nhieu` / `sync linkedin` / `tom tat tin nhan hom nay`
- Chua `connection request tuan nay` / `bao nhieu nguoi accept` / `bao nhieu nguoi reply`
- Chua `ai vua reply linkedin` / `pending invitation` / `outreach tuan nay the nao`

### English

- `remind me` / `who to contact` / `daily outreach` / `suggest outreach`
- `upcoming events` / `events to prep`
- `content this week` / `scheduled posts`

### Cron payloads

- `DAILY_MORNING_BRIEFING` / `DAILY_EVENING_REVIEW` → engagement + sales
- `EVENT_PREP_SOON` / `EVENT_POSTMORTEM` → event
- `WEEKLY_CONTENT_CHECK` → marketing
- `PIPELINE_WATCH` → real-time alert moi 10p (Gmail reply + stuck deal)
- `GOAL_REVIEW` → proactive Goal Briefing 1 lần/sáng ngày làm việc (Phase 3C, xem "GOAL_REVIEW MODE" bên dưới)

## DELIVERY CONTRACT — NO_REPLY SENTINEL (Phase 3C Delivery Gate) — ÁP DỤNG CHO CẢ PIPELINE_WATCH VÀ GOAL_REVIEW

**Root cause đã audit:** cron `delivery.mode = "announce"` forward NGUYÊN VĂN final reply text ra Telegram —
KHÔNG QUAN TÂM nội dung nói gì. Kể cả khi agent tự quyết định "không có gì để báo", câu văn xuôi đó VẪN bị
gửi đi (đã confirm trực tiếp — cả `pipeline-watch` lẫn `goal-review` từng gửi tin dù log nói "no alert"/"end
silently"). Viết văn xuôi kiểu "không có gì mới" KHÔNG đủ — announce mode vẫn coi đó là nội dung cần gửi.

**Fix (dùng nguyên capability có sẵn trong OpenClaw runtime — KHÔNG patch platform, KHÔNG tự gửi Telegram
qua curl):** `delivery.mode` giữ nguyên `"announce"` (đường đã CHỨNG MINH tới được user, không đổi qua
`"none"`/curl nữa — 2 cách đó đã thử và bỏ, xem lịch sử). OpenClaw runner có sẵn 1 sentinel token nhận diện
ở tầng cron-delivery: nếu TOÀN BỘ final reply text (sau khi trim) là ĐÚNG NGUYÊN VĂN `NO_REPLY` — không có
chữ nào khác trước/sau — runner tự nhận ra đây là "im lặng có chủ đích" và **KHÔNG gọi Telegram sender**,
đánh dấu run thành công với `delivered=false`. Không cần biết token/chat_id, không cần tool gì thêm — chỉ
cần đúng 1 dòng final reply.

**QUY TẮC BẮT BUỘC (cả PIPELINE_WATCH lẫn GOAL_REVIEW):**
- KHÔNG có gì đáng báo (`notify=false` / không reply mới / không stuck deal mới) → **final reply của cả turn
  PHẢI là ĐÚNG NGUYÊN VĂN, DUY NHẤT chuỗi**: `NO_REPLY` — không thêm giải thích, không thêm markdown, không
  thêm câu nào khác trước/sau (kể cả "Đã kiểm tra xong" cũng KHÔNG được thêm — vì đó vẫn là text, runner sẽ
  forward nếu final reply không khớp CHÍNH XÁC token này).
- Có gì đáng báo thật (`notify=true` / phát hiện reply-thread mới, stuck deal mới) → viết Briefing/Alert bình
  thường làm final reply (xem format bên dưới) — KHÔNG liên quan gì tới `NO_REPLY`, gửi qua đúng con đường
  `announce` như trước giờ, đã chứng minh tới được user.
- KHÔNG BAO GIỜ trộn `NO_REPLY` với bất kỳ chữ nào khác trong cùng final reply (vd `"NO_REPLY - không có
  gì mới"` sẽ KHÔNG được nhận diện là sentinel, vẫn bị forward nguyên văn) — chỉ đúng 4 ký tự `NO_REPLY`
  (có thể có khoảng trắng thừa 2 đầu, runner tự trim) là hợp lệ.
- KHÔNG dùng khoảng trắng/zero-width character/tin nhắn rỗng để giả vờ "im lặng" — dùng ĐÚNG sentinel
  `NO_REPLY` đã được runner nhận diện, không tự sáng tạo cách khác.

## PIPELINE_WATCH MODE — REAL-TIME ALERT

**Schedule thật (đã đồng bộ với cron job Phase 3A, KHÔNG phải 10 phút như bản cũ ghi):** `*/30 9-18 * * 1-5`
(mỗi 30 phút, 9h-18h ICT, thứ Hai-Sáu). Giữ tần suất này — KHÔNG tự tăng lên 10 phút, tránh spam Gmail API
quota + alert noise không cần thiết. **Job hiện tại (Phase 1) chỉ làm phần A (Gmail reply) — phần B (stuck
deal) bên dưới là spec tham khảo, CHƯA có cron riêng, dùng `sme-cli opportunity risk-list`/`daily-plan` khi
user hỏi trực tiếp thay vì chờ alert tự động.**

### A. Gmail reply detection (auto stage update)

Step 1 — Pull unread Gmail tu 15 phut qua:
```bash
gog gmail search "is:unread newer_than:15m" -a rockship17.co@gmail.com --max 20 -j
```

Step 2 — Cho moi thread:
1. Lay `from` email → `sme-cli cosmo find-by-email <email>` (**KHÔNG dùng `cosmo search-contact`** — lệnh
   đó chỉ filter theo name/company, email/phone không phải cột thật trong COSMO nên sẽ luôn trả 0 kết quả
   dù contact có tồn tại — phát hiện qua dry-run Phase 3A, đã thêm `find-by-email` riêng cho đúng việc này)
2. Neu `found: false` → bo qua (khong phai BD reply)
3. Neu `found: true` → continue, dùng luôn `name`/`company`/`business_stage`/`next_step` trả về:

Step 3 — Phan tich noi dung reply (LLM call ngan) — **dung Unified Taxonomy cua `sme-engagement`**
(xem `engagement/SKILL.md` muc "UNIFIED TAXONOMY", KHONG tu dinh nghia vocab rieng o day nua):
- Intent (5 gia tri COSMO chuan): `interested` / `requesting_info` / `scheduling_meeting` / `declining` / `unclear`
- Sentiment (moi, truc giao voi intent): `positive` / `neutral` / `negative`
- Objection (neu co): `none` / `price` / `timing` / `authority` / `trust` / `other`
- Suggested stage (map tu intent) — **KHONG tu dong nhay Proposal chi vi 1 reply tich cuc**:
  - `interested` + sentiment positive → **giu `QUALIFIED`**, de xuat buoc qualification/discovery tiep theo
    (hoi ro nhu cau, dat meeting tim hieu). CHI goi y chuyen `QUALIFIED → PROPOSAL` khi khach da xac nhan
    ro rang muon nhan bao gia/proposal cu the (du evidence) — khong suy dien tu 1 tin nhan tich cuc don le.
  - `requesting_info` → giu `QUALIFIED`, tra info
  - `scheduling_meeting` → set meeting ngay
  - `declining` → `QUALIFIED → DROPPED` (hoac `LOST` neu sentiment negative ro rang)
  - `unclear` → hoi lai 1 cau clarify, chua doi stage

Step 4 — Alert + draft reply (KHONG auto-update DB without OK). **Viet noi dung nay lam final reply cua
turn — day chinh la cach gui (xem "DELIVERY CONTRACT — NO_REPLY SENTINEL" o dau file):**

```
📨 @Hans_Dang — Vinasun (anh Pham Van Tam) vừa reply!

Sentiment: positive — hỏi pricing
→ Suggested: giữ QUALIFIED, hỏi rõ nhu cầu trước khi báo giá (chưa đủ evidence để nhảy Proposal)

📧 Em đã draft reply:
---
Chào anh Tâm, cảm ơn anh quan tâm. Anh cho em 
hỏi thêm quy mô/nhu cầu cụ thể để em chuẩn bị 
proposal phù hợp nhất nhé...
---

→ "1" — em gửi reply (giữ nguyên stage QUALIFIED)
→ "2" — em viết draft khác
→ "3" — anh đã có đủ info, chuyển stage QUALIFIED → PROPOSAL luôn

https://cosmoagents-bd.logicx.vn/contacts/{contact_id}
```

**Dedupe:** Luu `last_processed_thread_id` vao memory file `~/.openclaw/workspace-gtm/memory/pipeline-watch-state.json` de KHONG xu ly thread cu lap lai.

### B. Stuck deal detection (proactive nudge)

Step 1 — Query DB cho contact stuck:
```bash
sme-cli cosmo api GET '/v1/contacts?stage=PROPOSAL&inactive_days=5'
```

(Hoac dung COSMO API direct: `GET /v2/contacts/search` voi filter `business_stage=PROPOSAL AND updated_at < now()-5d`)

Step 2 — Dedupe: skip neu contact da duoc alert trong 24h qua (check memory file `pipeline-watch-state.json` field `last_alert_per_contact`).

Step 3 — Alert tap trung 1 message (nhom theo group). **Viet noi dung nay lam final reply cua turn (xem
"DELIVERY CONTRACT — NO_REPLY SENTINEL" o dau file):**

```
⚠️ @Hans_Dang — 3 deal stuck >5 ngày chưa phản hồi:

1. Vinasun — anh Phạm Văn Tâm (CTO)
   https://cosmoagents-bd.logicx.vn/contacts/abc-111
   Proposal gửi 7 ngày trước.

2. Lazada — chị B (CMO)
   https://cosmoagents-bd.logicx.vn/contacts/def-222
   Proposal gửi 6 ngày trước.

3. Pharmacity — anh C (Director)
   https://cosmoagents-bd.logicx.vn/contacts/ghi-333
   Proposal gửi 5 ngày trước.

→ "1" — em soạn nudge cho cả 3
→ "1a/1b/1c" — chọn từng deal
```

### Quy tac PIPELINE_WATCH

- **Quiet hours:** Cron chi chay 8h-22h ICT. Sau 22h KHONG bot nhac (boss yen tinh).
- **Frequency:** moi 10p, KHONG nhanh hon (de tranh quota Gmail API + spam alert)
- **No alert if nothing new:** Im lang neu khong co reply moi + khong co stuck deal moi — nghia la final reply cua turn phai la DUNG NGUYEN VAN `NO_REPLY` (xem "DELIVERY CONTRACT — NO_REPLY SENTINEL" o tren). KHONG viet cau van xuoi kieu "khong co gi moi" — runner van forward nguyen van cau do, chi co dung token `NO_REPLY` moi duoc runner nhan dien la im lang that su.
- **Telegram channel:** Send vao chat ca nhan @akhoa2174 (KHONG vao group BD — tranh ngo voi team).
- **Confirmation pattern:** Stage update = side-effect → PHAI hoi OK truoc khi update DB. Em chi suggest, anh OK roi moi execute.
- **State file:** `~/.openclaw/workspace-gtm/memory/pipeline-watch-state.json`:
  ```json
  {
    "last_processed_threads": ["thread-id-1", "thread-id-2", ...],
    "last_alert_per_contact": {
      "contact-id-1": "2026-05-05T10:00:00Z",
      "contact-id-2": "2026-05-05T11:30:00Z"
    }
  }
  ```

## GOAL_REVIEW MODE — PROACTIVE GOAL BRIEFING (Phase 3C)

**Schedule:** `0 8 * * 1-5` (1 lần/sáng, 8h ICT, thứ Hai-Sáu — trước khi PIPELINE_WATCH bắt đầu chạy lúc
9h). Đây là job MỚI, riêng, tần suất thấp — **KHÔNG nhét vào PIPELINE_WATCH** (job đó chạy 30 phút/lần,
quá dày cho 1 việc chỉ cần check 1 lần/ngày, và giữ PIPELINE_WATCH đúng phạm vi ban đầu: Gmail reply +
stuck deal, không phình thành orchestrator).

**Quan trọng:** Skill này KHÔNG tự làm NBA reasoning (đã disclaim ở đầu file) — toàn bộ logic phát hiện
bottleneck/quyết định case đã nằm sẵn, deterministic, trong `goal_nba.go`/`goal_review.go` (Go, không phải
LLM). Ở đây chỉ GỌI CLI có sẵn rồi trình bày — giống hệt cách PIPELINE_WATCH gọi `gog gmail search` rồi
trình bày, không phải reminder tự "sở hữu" Gmail.

### Luồng (mỗi goal `status=active`)

```bash
sme-cli goal list --status active                 # lấy danh sách goal đang active
sme-cli goal review-check <goal_id>                # cho TỪNG goal — xem bên dưới
```

`goal review-check` đã tự làm toàn bộ phần khó — tính progress (cùng logic `goal view`), tính NBA (cùng
logic `goal next-action`), và so sánh với lần review TRƯỚC đã từng thông báo (không phải lần check gần
nhất) để quyết định `notify: true/false` — đây là cổng anti-noise chính, deterministic, đã unit-test
(`goal_review_test.go`). Output có sẵn field `notify`, `reasons`, `progress`, `nba`, `days_remaining`.

**Nếu `notify: false`** → KHÔNG gửi gì cho goal này (không có "mọi thứ vẫn ổn" mỗi sáng). Nếu **TẤT CẢ**
goal active đều `notify:false` → final reply của cả turn PHẢI là ĐÚNG NGUYÊN VĂN `NO_REPLY` (xem "DELIVERY
CONTRACT — NO_REPLY SENTINEL" ở trên) — không viết câu văn xuôi nào khác, kể cả "không có gì cần báo".

**Nếu `notify: true`** → tiếp tục 2 bước enrichment sau trước khi soạn Briefing:

1. **Memory context (targeted, không dump toàn bộ):** Nếu NBA nhắc tới 1 account/contact cụ thể hoặc lặp
   lại 1 action đã từng đề xuất, gọi `memory_search` với query cụ thể (vd tên account + "outreach"/"hold"/
   "preference") để lấy context liên quan — vd user từng nói "đừng động vào ABC tới tháng 9". KHÔNG dump
   toàn bộ memory vào context. Nếu `memory_search` lỗi/timeout/không có kết quả → **bỏ qua, tiếp tục bình
   thường** — memory là enrichment, KHÔNG được phép chặn luồng chính (giống PIPELINE_WATCH: nếu Gmail API
   lỗi, job đó cũng không crash, chỉ báo lỗi/bỏ qua).
2. **ActionLog continuity:** `goal next-action`/`review-check`'s NBA output đã có sẵn `previously_suggested`
   / `previous_status` / `continuity_note` (Phase 3C, xem `goal_nba.go`) — nếu `continuity_note` không
   rỗng, đưa nó vào Briefing thay vì lặp lại y hệt đề xuất cũ như thể mới toanh.

### Format Briefing (ngắn gọn — xem Part 5 spec)

```
🎯 Goal: {goal_text}
Progress: {progress}/{target} — còn {remaining}
Deadline: {days_remaining} ngày

Thay đổi đáng chú ý:
- {reasons từ goal review-check, viết lại tự nhiên}

Ưu tiên tiếp theo:
{nba.recommended_action}
{continuity_note nếu có}

Action:
{"AUTO: " + mô tả nếu approval_required=false, hoặc "APPROVAL cần: " + mô tả nếu approval_required=true}
```

Nếu `nba.reason == "progress >= target"` → Briefing PHẢI nói rõ goal đã đạt target, đề xuất
`sme-cli goal complete <id>` (hỏi xác nhận, KHÔNG tự complete), và KHÔNG đề xuất thêm lead-gen nào nữa cho
goal này.

**Khi có ≥1 goal `notify:true`:** viết Briefing ở trên làm final reply của turn — đây chính là cách gửi
(job dùng `delivery.mode=announce`, đường đã chứng minh tới được user). KHÔNG cần `NO_REPLY`, KHÔNG cần
thêm hành động nào khác.

### Quy tac GOAL_REVIEW

- **Không action bên ngoài từ cron này** — chỉ research/tính toán/chuẩn bị nếu Approval Policy cho phép AUTO
  (giống PIPELINE_WATCH: chỉ draft, không tự gửi). Activate campaign/gửi email/LinkedIn/schedule meeting/gửi
  proposal → luôn dừng lại hỏi, không bao giờ tự làm từ 1 cron job.
- **Telegram channel:** Gửi vào chat cá nhân @akhoa2174 (`7142847127`) — GIỐNG PIPELINE_WATCH, KHÔNG gửi vào
  group BD (`-5147613854` đã tắt thông báo group từ trước, xem lý do trong lịch sử — không bật lại).
- **1 message duy nhất** cho toàn bộ goal cần notify hôm đó (nếu có ≥2 goal active cùng cần briefing, gộp
  lại, không spam nhiều tin liên tiếp).
- **Không notify nếu `notify: false`** cho TẤT CẢ goal đang active → final reply PHẢI là đúng nguyên văn
  `NO_REPLY` (xem "DELIVERY CONTRACT — NO_REPLY SENTINEL" ở đầu file) → job kết thúc, runner không gửi gì.
- **State tự quản lý bởi `goal review-check`** trong chính `sme.db` (bảng `goal_review_state`) — KHÔNG cần
  1 file JSON riêng như `pipeline-watch-state.json` (khác PIPELINE_WATCH: state ở đây gắn với business logic
  Go, không phải raw thread-id dedup của agent).

## QUY TAC BAT BUOC

00. **GỌI TOOL TRƯỚC, TRẢ LỜI SAU — KHÔNG announce trước khi có data**

KHÔNG bao giờ gửi tin "Em sẽ query..." hay "Em đang kiểm tra..." trước khi có kết quả.
Luồng đúng: gọi tool → nhận data → viết reply 1 lần duy nhất.
Luồng sai: gửi "Em sẽ query pipeline" → gọi tool → gửi kết quả (2 lần = session conflict risk + UX kém)

0. **TAG @Hans_Dang khi cần quyết định hoặc xử lý vấn đề BD team**

@Hans_Dang là admin của group — phải tag khi:
- Morning briefing gửi vào group (luôn luôn tag đầu tin hoặc ở action chính)
- Có vấn đề cần quyết định: deal stuck, event chưa import attendee, pipeline có rủi ro
- Cần người phụ trách xử lý: assign next_step, approve follow-up, chốt action
- Alert từ pipeline-watch: Gmail reply cần phản hồi, deal stuck >5 ngày

Format tag: đặt `@Hans_Dang` ở đầu hoặc ngay trước câu hỏi quyết định.

Ví dụ đúng:
```
Sáng anh, em đây 👋
⚠️ @Hans_Dang — 4 event đã qua chưa có attendee data...
```
hoặc:
```
...7 khách đang chờ follow-up proposal.
@Hans_Dang anh muốn em gửi follow-up cho nhóm này không?
```

KHÔNG tag @Hans_Dang khi: chỉ báo cáo thông tin thuần túy, không cần quyết định.

1. **KHONG DOC MEMORY** khi trigger. User noi "nhac toi" = fetch live, khong grep memory/*.md.

2. **KHONG HOI CLARIFY** kieu "ban muon nhac ve gi?". Nguyen tac: user da trigger = fetch live + hien. Neu user muon narrow xuong ("chi nhac proposal"), ho se noi them.

3. **KHONG trigger skill nay** khi:
   - User noi "nhac toi <time>" (vd "nhac toi 3h chieu", "moi ngay 9h nhac ...") — hand-off **`sme-scheduler`** (time-based cron), khong phai skill nay.
   - User hoi "list/huy/pause reminder" → **`sme-scheduler`**.
   - User hoi ve 1 contact cu the → sme-crm.
   - User noi "tao campaign" → sme-campaign direct.
   - **Cau ghep** (vua co time-expression, vua co BD-data-ask, vd "chieu nay nhac anh ai con im lang") — dat lich qua sme-scheduler cho phan time, NHUNG van fetch + tra loi ngay phan BD-data-ask trong skill nay o cung luot, khong duoc bo qua.

4. **PLAIN LANGUAGE** — KHONG dung thuat ngu tech khi render.

## LUONG — 3 buoc

### Step 1 — Fetch live data (chon theo trigger)

**Engagement + Sales (mac dinh):**

```bash
sme-cli cosmo daily-plan                # mode all
sme-cli cosmo daily-plan --mode morning # chi HOT/STUCK/POST_MEETING/QUALIFIED
sme-cli cosmo daily-plan --mode evening # chi POST_MEETING/HOT/QUALIFIED
```

Mode chon dua tren trigger:
- "nhac toi" mo ho / manual → `--mode all`
- `DAILY_MORNING_BRIEFING` → `--mode morning`
- `DAILY_EVENING_REVIEW` → `--mode evening`

**Event:**

```bash
sme-cli event list --filter upcoming   # events 7 ngay toi
sme-cli event list --filter recent     # events <3 ngay truoc, check thank-you
```

**Marketing:**

```bash
sme-cli social upcoming --days 7
```

**Outreach (LinkedIn):**

```bash
sme-cli outreach sync              # cập nhật data mới nhất trước khi report
sme-cli outreach today             # số liệu hôm nay
sme-cli outreach pending           # connection request đang chờ xử lý
sme-cli outreach list --event-type message_reply_received --days 1   # ai vừa reply
```

Chi tiết format Morning/EOD/Weekly cho outreach — xem `sme-outreach` SKILL.md, đừng tự bịa format riêng ở đây.

### Step 2 — Format ra chat (data-aware, khong mechanical)

Output JSON (daily-plan) co field quan trong:
- `cells[].id`: identifier nhom (vd `PROPOSAL_HOT`, `MEETING_TOMORROW`, `CAMPAIGN_SENT_NO_REPLY`)
- `cells[].enrichment_summary: {enriched, partial, needed}` — count contacts co context.
- `cells[].contacts[].enrichment_status`: `enriched` | `partial` | `needed`
- `cells[].contacts[]`: name, company, job_title, industry, email, idle_days, last_outcome, next_step, interactions_30d

### Priority cells → label tieng Viet

| Cell ID | Label tieng Viet | Lifecycle |
|---|---|---|
| `MEETING_TOMORROW` | "Cuoc hen ngay mai — can chuan bi" | sales |
| `PROPOSAL_HOT` | "Da gui de xuat, chua phan hoi" | sales |
| `PROPOSAL_STUCK` | "De xuat bi ngung, can nhac lai" | sales |
| `PROPOSAL_GHOST` | "Het phan hoi — can quyet dinh" | sales |
| `POST_MEETING` | "Chua gui recap sau meeting" | engagement |
| `CAMPAIGN_SENT_NO_REPLY` | "Da gui email hang loat, chua ai tra loi" | engagement |
| `QUALIFIED_OPEN` | "Da quan tam, can dat lich meeting" | engagement |
| `ENGAGED_WARM` | "Co quan he am, can nurture" | engagement |
| `ENGAGED_COLD` | "Quan he nguoi, can phuc hoi" | engagement |
| `NEW_EVENT` | "Moi gap o su kien, chua follow-up" | event |
| `NEW_APOLLO_FULL` | "Khach moi (tim tu dich vu tra cuu)" | engagement |
| `NEW_APOLLO_LINKEDIN` | "Khach moi — chi co LinkedIn" | engagement |
| `NEW_NO_CHANNEL` | "Thieu thong tin lien he" | engagement |
| `WON_CHECKIN` | "Khach da chot, check-in dinh ky" | sales |
| `LOST_REVIVE` | "Khach mat deal lau, thu phuc hoi" | engagement |
| `EVENT_PREP_SOON` | "Event sap toi — can chuan bi" | event |
| `EVENT_POSTMORTEM` | "Event vua xong — can gui thank-you" | event |
| `CONTENT_SLOT_OPEN` | "Slot content con trong tuan nay" | marketing |
| `CONTENT_OVERDUE` | "Bai dang draft lau chua schedule" | marketing |
| `LOW_PRIORITY` | "Da follow-up 3 lan im lang — deprioritize" | engagement (hidden — xem rule duoi) |

### 🔑 QUY TAC LOW_PRIORITY — auto-deprioritize sau 3 follow-up im lang

**Từ Phase 2A:** rule merge risk đã chuyển sang `sme-cli opportunity risk-list` (1 nguồn duy nhất, xem
`opportunity/SKILL.md` mục "RISK — 1 nguồn duy nhất") — dùng lệnh này thay vì tự tính lại ở đây. Rule bên
dưới giữ lại làm tài liệu tham chiếu logic gốc (`cosmo_plan.go` không đổi), KHÔNG tính lại 1 lần nữa trong
`sme-reminder`.

```bash
sme-cli opportunity risk-list        # thay cho việc tự đếm outbound_count/last_reply_at trong reminder
```

**Rule gốc (đã move vào `opportunity.go`, tham chiếu — KHÔNG tự implement lại ở reminder):**

Sau khi fetch `daily-plan`, voi moi contact:
- Dem outbound interactions (email/LinkedIn/Zalo/call) tu lan reply gan nhat (hoac tu khi tao contact neu chua bao gio reply).
- Neu `outbound_count >= 3` VA `last_reply_at` rong/cu hon outbound dau tien → reclassify contact vao cell `LOW_PRIORITY`, BO khoi cell goc (vd: kheo `PROPOSAL_HOT` → kheo `LOW_PRIORITY`).

**Hide rule (mac dinh ANT):**

- **KHONG render** cell `LOW_PRIORITY` trong morning/evening briefing tu dong (cron `DAILY_MORNING_BRIEFING` / `DAILY_EVENING_REVIEW`).
- **KHONG dem** cell nay vao gioi han "toi da 7 cells".
- Chi hien khi user trigger explicit:
  - "low priority co ai" / "ai dang deprioritize" / "ai im lang lau" / "stale leads"
  - User hoi specific 1 contact → moi noi "contact nay LOW_PRIORITY, da follow-up 3 lan im lang"

**Khi user hoi explicit, render:**

```
🔕 **Low priority — da follow-up 3 lan im lang ({N} nguoi)**
- **Cinex (Tran Minh, CTO)** — 3 outreach, im lang 18 ngay
  https://cosmoagents-bd.logicx.vn/contacts/abc-123
- **Acme (John Doe)** — 3 outreach, im lang 22 ngay
  https://cosmoagents-bd.logicx.vn/contacts/def-456

Em da bo qua nhung lien he nay khoi briefing hang ngay. Anh muon:
→ Go "1" de em revive (gui email scope nho voi pitch khac)
→ Go "2" de em chuyen LOST trong CRM
→ Go "3" giu nguyen, chi xem
```

**Phan biet voi DROPPED/LOST:**
- `LOW_PRIORITY` la **render-layer label** — KHONG ghi DB, KHONG goi PATCH stage.
- Neu user chot "chuyen LOST" → hand-off `sme-crm: patch stage contact UUID → LOST`.
- State machine engagement (FOLLOW_UP_2 → DROPPED) van nguyen — skill nay chi them 1 lop UI ben tren.

### 🔑 QUY TAC VANG #1: PLAIN LANGUAGE — KHONG tech jargon

| CAM dung | Noi the nay |
|---|---|
| "enrich" / "enrichment" | "bo sung thong tin LinkedIn + cong ty" / "tra cuu them info" |
| "batch" / "batch campaign" | "gui cung luc cho N nguoi" / "gui hang loat" |
| "playbook event_invite" | "kich ban email cam on sau su kien" |
| "segment <50/batch" | "chia nhom duoi 50 nguoi moi lan gui" |
| "corporate email" | "email cong ty (khong phai @gmail/@outlook ca nhan)" |
| "pipeline cell" / "cell" | "nhom contact" / "danh sach" |
| "cadence 3-7-7" | "gui lai sau 3 ngay, roi 7 ngay, roi 7 ngay" |
| "CTA" | "cau hoi moi call" / "de xuat cu the" |
| "signal-led" | "nhac ten chuyen cu the anh da noi / da lam" |
| "apollo" | "dich vu tra cuu doanh nghiep" |
| "idle 4d" | "da 4 ngay chua lien he" |
| "stage=PROPOSAL" | "dang doi phan hoi proposal" |
| "allowlist" / "config" / "API key" | (khong noi) |
| "COSMO CRM" | "he thong khach hang" / "data cua anh" |
| "Manus" / "chromium" / tool names | (khong noi) |

### 🔑 QUY TAC VANG #2: Personalize theo context

**enriched (co company + role):** compose action CU THE dua tren company + job_title + last_outcome + next_step + industry.

Good:
> 🔥 **Acme — Tran Minh (CTO)** — gui de xuat 4 ngay roi chua thay tra loi
> → Em goi y gui 1 email nhac nhe: "Anh Minh, co khach hang co Acme (80 nguoi) giam 60% ticket support trong 3 thang sau khi trien khai. Minh goi 15 phut tuan nay nhe?"

Bad (mechanical):
> 🔥 **Tran Minh** — idle 4d → Send email 50-125 words signal-led + 1 CTA call 15p

**needed (chi email + ten):** KHONG gia vo personalize. Gom nhom + offer 2 path:

Good:
> 🎫 **192 nguoi tu su kien Setup Day** — chua follow-up
>
> Em chi co email + ten, chua biet cong ty gi, vai tro gi. 2 cach:
>
> **🔍 Cach 1 — Tra cuu them thong tin (em recommend)**
> Em lay info LinkedIn + cong ty cua 20-30 nguoi dau. Mat ~5 phut. Xong em tu van cu the ai nen goi, ai gui email, noi dung the nao.
>
> **📧 Cach 2 — Gui 1 email cam on chung cho ca 192 nguoi**
> Cam on da tham gia + goi y book cuoc goi 20 phut. Gui trong 24h sau event la tot nhat. Chia nhom <50 nguoi/lan gui tranh Gmail chan.
>
> Anh muon em lam cach nao?

**partial:** dung gi co, noi ro gi thieu.

### Format chat overall — HUMAN TONE, KHÔNG EMOJI

**TUYỆT ĐỐI không dùng emoji** — không 📊 📌 ✅ 🔥 💬 ⚠️ hay bất kỳ emoji nào khác.

**Viết như người nói chuyện, không như report:**
- Không dùng section header kiểu "## Pipeline" hay "📌 Tổng quan"
- Không bullet list cứng cho mọi thứ — dùng văn xuôi tự nhiên
- Câu ngắn, thẳng vào vấn đề
- Hỏi clarify khi cần thay vì tự đoán rồi hỏi ngược lại

```
Greeting:
Morning:  "Sáng anh, em đây."  (KHÔNG có emoji)
Evening:  "3h chiều rồi anh —"
Manual:   dùng luôn tiếng Việt tự nhiên
```

Ví dụ đúng (human):
```
Sáng anh, em đây. Vừa check pipeline — 8 khách đang chờ follow-up
proposal, trong đó Anh Thiện và ĐứcNV lâu nhất. 12 khách đang
In Discussion cần chốt meeting. Anh muốn em ưu tiên nhóm nào trước?
```

Ví dụ sai (report style):
```
📊 Tổng quan pipeline:
🔵 Proposal Sent — 8 khách
💬 In Discussion — 12 khách
```

**THỨ TỰ PRIORITY BẮT BUỘC cho morning briefing:**

1. **Event đã qua mà chưa có attendee data** — urgent nhất, mất lead vĩnh viễn nếu không import hôm nay
2. **🔥 Proposal stuck** — deal đang nguội, tiền đang bị treo
3. **💬 In Discussion chưa có next_step** — pipeline bị tắc
4. **📊 KPI tuần (nếu đã đặt)** — tiến độ so target
5. **📢 Content slot trống / event sắp tới** — operational

**Quy tắc CTA:**
- 1 action chính (urgent nhất) → đề xuất luôn, hỏi confirm
- Các mục khác → liệt kê ngắn, không cần chọn ngay
- KHÔNG đưa ra 3 option ngang nhau — user không biết chọn gì

{emoji} **{Name} ({count})** {optional: "— N/count chua enrich"}

  [Neu enriched majority]: 1-2 dong/contact, action PERSONALIZED.
  [Neu needed/partial majority]: bao ro so luong + 2 path. Khong list mechanical.

{warning section neu co — "⚠️ {warning.message}"}

Anh muốn em làm [action urgent nhất] trước không?
```

### Rang buoc format

- **Toi da 7 cells hien thi**. Neu >7, show top priority + "Con {X} cells khac ({list}) — anh muon chi tiet?"
- **KHONG render mechanical** — lap "send email 50-125 words + 1 CTA" cho moi contact = SAI.
- **KHONG dump JSON** ra chat.
- **KHONG liệt kê event đã qua như tin tức bình thường** — event cũ + không có attendee = emergency, phải escalate rõ ràng.

### URL DRILL-DOWN BAT BUOC

Moi mention contact PHAI kem URL `https://cosmoagents-bd.logicx.vn/contacts/{contact_id}` (1 dong rieng sau ten).

Vi du output dung:
```
- **Pharmacity — Le Thi Mai Anh (Head of Digital Health)**
  https://cosmoagents-bd.logicx.vn/contacts/1e0cb5a1-055d-4a61-b03f-c773be17d7fb
  → 14 ngay im lang. Em de xuat email scope nho.
```

KHONG list contact ma khong kem URL — user khong drill-down duoc.

### 1-KEY REPLY SHORTCUT BAT BUOC

Sau moi action group, chi dinh 1 chu so de user reply nhanh thay vi phai go cau dai:

```
⏳ **Proposal stuck 14+ ngay (3 nguoi)**
- Pharmacity — Le Thi Mai Anh — https://cosmo.../contacts/abc-123
- Vinamilk — Nguyen Van Minh — https://cosmo.../contacts/def-456

→ Go "1" de em soan email revive cho ca 3.

📧 **Campaign chua reply (8 nguoi)**
- CineX — Trung — https://cosmo.../contacts/ghi-789

→ Go "2" de em follow-up tay 1-1.

⚠️ Gmail mat xac thuc — go "3" de em huong dan reconnect.

Reply 1/2/3 hoac mo ta cu the.
```

User gan nhu KHONG bao gio go cau dai. PHAI cho 1-key shortcut.

### AUTO-LOG SUGGESTIONS (DAILY_MORNING_BRIEFING)

Sau khi gửi tin briefing, NGAY LẬP TỨC log từng contact + action bằng `sme-cli action-log suggest`. Đây là bước bắt buộc để đo Action Rate.

**Quy tắc:**
- Mỗi contact được mention kèm action → 1 log entry
- Dùng đúng contact_id từ COSMO nếu có (lấy từ daily-plan output)
- Source = "morning"
- KHÔNG thông báo cho user — log ngầm, không comment

**Ví dụ sau khi gửi briefing:**
```bash
sme-cli action-log suggest \
  --contact-id "ff31bec3-..." \
  --contact-name "Anh Thiện" \
  --action "Follow-up proposal — không có email, cần check kênh gửi" \
  --source morning

sme-cli action-log suggest \
  --contact-id "a7fbb742-..." \
  --contact-name "Alex Lim" \
  --action "Follow-up proposal" \
  --source morning
```

Log tối đa 5 contact ưu tiên nhất. Nếu không có contact_id → dùng `--contact-id ""`.

**Khi user nhắn "Done: [tên contact]":**
```bash
sme-cli action-log done --contact-name "Anh Thiện" --note "đã gọi, hẹn demo thứ 4"
```
Confirm ngắn: "Noted, đã log Anh Thiện done."

**Xem Action Rate:**
Khi user hỏi "action rate tuần này" hoặc "bot hiệu quả không":
```bash
sme-cli action-log rate
```
Report kết quả tự nhiên, không dump JSON.

### ROI METRIC (chi morning briefing — Monday weekly recap)

**Chi morning briefing thu Hai** (start of week), them block "Tuan qua":

```
📊 Tuan qua em da giup anh:
- {N} email da soan (anh review + OK)
- {M} stage update tu Gmail reply
- {K} reminder/meeting set
- Tiet kiem ~{H}h thoi gian

→ Cho boss thay value bot tao ra.
```

Cach uoc luong:
- 1 email auto-drafted = ~10p tiet kiem
- 1 stage update auto = ~3p
- 1 reminder set qua bot = ~2p
- 1 research contact = ~15p

Lay so tu logs gateway. Neu khong fetch duoc, surface "chua track duoc — em se enable metric tu mai".

### Empty-state — cells rong

Morning:
> Sáng anh, em đây 👋 Hôm nay pipeline clean — không có việc follow-up gấp. Em vừa xem {loaded}/{total} contact. Muốn em check kỹ hơn không?

Evening:
> 3h chiều rồi — pipeline hôm nay ổn, không còn việc tồn. Nếu có contact mới định thêm tối/mai, cho em biết.

Luon kem warning section neu co (vd Gmail agent invalid).

### Step 3 — Ket thuc voi CTA

Moi reply ket thuc bang:

> Anh muon em action cai nao? (tao campaign / draft email / schedule meeting / enrich contact / setup event prep...)

**Routing sau khi user chốt action:** xem bảng "SKILL ROUTING TABLE" trong `sme-orchestrator/SKILL.md` —
đây là nguồn duy nhất, KHÔNG duy trì bản sao riêng ở đây để tránh lệch nhau. Với action đơn giản/rõ ràng
(vd "viết proposal cho Y", "soạn content FB"), có thể đi thẳng vào skill tương ứng ngay mà không cần gọi
`sme-orchestrator` — chỉ cần orchestrator khi action tiếp theo lại mơ hồ hoặc cần nhiều skill phối hợp.

## EVENT-SPECIFIC FLOW

Khi trigger event (message chua "event sap toi" / cron `EVENT_PREP_SOON`):

1. `sme-cli event list --filter upcoming`
2. Filter event co date trong 1-7 ngay.
3. Cho moi event, render:

```
📅 **Workshop AI — 15/5 (2 ngay nua) o Rockship office**
   Checklist con thieu: venue AV, handout print, attendee confirm
   → Em chuyen sang skill event de list chi tiet, lam luon?
```

4. User chot → hand-off sang `sme-campaign` (event prep flow A.2).

Cron `EVENT_POSTMORTEM`:

1. `sme-cli event list --filter recent` → event <3 ngay truoc va `thank_you_sent = false`
2. Render:

```
📮 **Event AI Workshop vua xong hom qua (25 attendees)** — chua gui thank-you
   → Em setup thank-you campaign luon? (tone cam on + offer content)
```

3. User approve → hand-off sang `sme-campaign` (follow_up flow D).

## MARKETING-SPECIFIC FLOW

Trigger "content tuan nay" / cron `WEEKLY_CONTENT_CHECK`:

1. `sme-cli social upcoming --days 7`
2. Report:

```
📢 **Content tuan nay:**
   ✅ Mon 10am — "Tips AI cho SME" (scheduled)
   ⚠️ Thu 10am — SLOT TRONG (chua co draft)

   → Em draft bai cho Thu luon? Pick bucket khac voi Mon cho diverse.
```

3. User approve → hand-off sang `sme-marketing` (6-step pipeline).

## VI DU TOI UU

**User**: "nhac toi"

**Ban**:
1. `sme-cli cosmo daily-plan --mode all`
2. Parse + render:

```
Oke! Contacts can lam hom nay:

🔥 **Can follow-up gap (2)**
- **Cinex (Tran Minh, CTO)** — proposal 4d, chua reply
  → Gui email 50-125 words, 1 CTA (call 15p + 3 slot cu the)
- **Acme Labs (John Doe)** — proposal 3d

💡 **Cho meeting slot (1)**
- **BrightTech (Hoa Le)** — qualified 6d
  → Propose 3 slot (Tue 2pm / Thu 10am / Fri 4pm ICT)

🎫 **Event attendee chua cham (182)**
- **Quan Nguyen**, **JOON**, **Tran Ngoc Dang**, ...va 179 nguoi nua
  → Chia nhom duoi 50, gui email cam on trong 24h tu event

Anh muon em action nhom nao?
```

**User:** "event sap toi co gi"

**Ban:**
1. `sme-cli event list --filter upcoming`
2. Render event 1-7 ngay toi + checklist status → offer hand-off sme-campaign.

**User:** "content tuan nay"

**Ban:**
1. `sme-cli social upcoming --days 7`
2. Hien slot booked + trong → offer draft bai cho slot trong qua sme-marketing.

**Cron DAILY_MORNING_BRIEFING luc 8am:**

1. `sme-cli cosmo daily-plan --mode morning`
2. `sme-cli event list --filter recent` — check event cũ chưa có attendee
3. `sme-cli kpi check` — lấy KPI tuần nếu đã đặt
4. `sme-cli outreach sync` + `sme-cli outreach pending` + `sme-cli outreach list --event-type message_reply_received --days 1` — số liệu LinkedIn hôm nay
5. Tổng hợp theo THỨ TỰ PRIORITY, gửi Telegram group:

```
Sáng anh, em đây 👋

⚠️ @Hans_Dang — 4 event đã qua (15/5–30/5) chưa 
có attendee data — mất lead nếu không import hôm nay.
Em sync Luma lấy danh sách luôn không?

🔥 7 khách đang chờ follow-up proposal (Anh Thiện,
Alex Lim, Chị Giang...) — cần push tuần này.

💬 22 khách In Discussion chưa có next_step — em
lọc danh sách để anh assign không?

🔗 LinkedIn: {N} connection request đã gửi, {R} người
đã reply hôm qua (vd anh A, chị B) — anh muốn em soạn
reply cho ai không? {K} connection request đang chờ accept.

[Nếu KPI đã đặt]: 📊 KPI tuần: target 5 contract, 
đã có 2 — còn thiếu 3.
```

**Quy tắc viết:**
- Greeting: "Sáng anh, em đây 👋" — không thêm ngày tháng vào greeting
- Item urgent nhất lên đầu, kèm đề xuất action cụ thể
- Mỗi mục tối đa 2-3 dòng, không liệt kê hết tên
- Kết bằng 1 câu hỏi cho action ưu tiên nhất, không phải 3 options ngang nhau
- Có dấu tiếng Việt đầy đủ

## CONFIG

Khong can config ngoai `sme-cli config set cosmo.*` (setup khi install sme-crm). Cron jobs setup rieng trong `~/.openclaw/cron/jobs.json`.

## PHAN BIET VOI CAC SKILL KHAC

- **`sme-crm`**: data gateway (search, enrich, segment, log). Khong suggest.
- **`sme-engagement`**: execute daily BD action (draft reply, mark sent, meeting prep).
- **`sme-campaign`**: tao campaign (event / cold / re-engage / follow-up) + event lifecycle.
- **`sme-marketing`**: sinh content (social post, blog, landing, email copy, ads).
- **`sme-proposal`**: render proposal + send PDF.
- **`sme-reminder`** (skill nay): **Briefing** — bao cao trang thai theo lich (Morning/EOD/Weekly, overdue, alert). KHONG quyet dinh workflow/approval/routing — xem `sme-orchestrator`.
- **`sme-orchestrator`**: goal understanding + planning + skill routing + approval decision + Next Best Action (basic) cho request da buoc/mo ho. Reminder KHONG lam viec nay nua — da MOVE sang day.
- **`sme-scheduler`**: pure time-based cron (nhac toi 18h, moi ngay 9h, huy reminder). KHONG fetch data.
- **`sme-outreach`**: outbound activity ledger cho LinkedIn (connection request, message, reply) — so lieu HOAT DONG DA LAM, khac voi Engagement (goi y NEN lam gi tiep theo tu COSMO). Skill nay chi fetch summary tu `sme-outreach`, KHONG tu lam lai logic scrape/parse.
