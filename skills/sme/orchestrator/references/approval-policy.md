# GTM Approval Policy (Phase 1)

Đây là chính sách phê duyệt DÙNG CHUNG cho toàn bộ GTM skill. Mọi skill (campaign, outreach, engagement,
proposal, marketing...) và `sme-orchestrator` đều tham chiếu văn bản này thay vì tự định nghĩa rule riêng.

Thiết kế theo hướng **bounded autonomy** — KHÔNG phải "mọi hành động đều phải hỏi user". Mục tiêu: agent tự
làm được nhiều việc nhất có thể trong phạm vi an toàn, chỉ dừng lại hỏi khi hành động có rủi ro thật (không
thể hoàn tác, ảnh hưởng bên ngoài, hoặc liên quan cam kết thương mại/pháp lý).

## AUTO — tự làm, không cần hỏi trước

Điều kiện: read-only, hoặc có thể hoàn tác dễ dàng, hoặc có evidence rõ ràng hỗ trợ quyết định.

- Search / read (CRM, contact, event, content, KPI, outreach data...)
- Research / enrichment (Apollo, Gmail history, web search cho context)
- Analysis (tính bottleneck, phân loại cell, tổng hợp báo cáo)
- Intent/sentiment classification (đọc reply, gắn nhãn — KHÔNG tự ý ghi stage dựa trên đó)
- Draft generation (soạn email, proposal, content — CHƯA gửi/publish)
- Activity logging (`sme-cli outreach log-event`, `action-log suggest`)
- Reminder / thông báo nội bộ cho user (không phải gửi ra ngoài cho khách)
- Factual CRM update khi có evidence rõ ràng và không mơ hồ (vd: log 1 interaction user vừa mô tả cụ thể,
  cập nhật `next_step` dựa trên nội dung user vừa xác nhận) — KHÔNG áp dụng cho đổi `business_stage` nếu
  không rõ ràng (xem mục APPROVAL bên dưới)

## APPROVAL — phải hỏi OK trước khi thực thi

Điều kiện: hành động visible ra bên ngoài (khách hàng/công chúng thấy), khó hoàn tác hoàn toàn, hoặc ảnh
hưởng tới reputation.

- Gửi tin nhắn/email outbound thật ra ngoài (LinkedIn message, email campaign, reply khách)
- Activate campaign (bulk send, kích hoạt sequence)
- Đặt lịch meeting với khách hàng bên ngoài (không phải nhắc nội bộ)
- Gửi proposal (file/PDF) cho khách
- Chuyển `business_stage` khi tín hiệu MƠ HỒ (vd intent=unclear, hoặc chỉ dựa suy đoán không có evidence
  rõ trong nội dung reply)
- Bất kỳ hành động nào hiển thị công khai / ảnh hưởng tới hình ảnh công ty ra bên ngoài

Khi ở nhóm này: agent PHẢI trình bày draft/kế hoạch trước, chờ user xác nhận (kể cả 1-key reply shortcut),
KHÔNG tự thực thi trước rồi báo sau.

## HUMAN ONLY — không tự quyết, chỉ con người mới được quyết

Điều kiện: cam kết có tính ràng buộc thương mại/pháp lý, hoặc thay đổi có ảnh hưởng tài chính trực tiếp.

- Thay đổi giá / pricing tier
- Discount / ưu đãi đặc biệt cho khách cụ thể
- Điều khoản thương mại (payment terms, SLA, cam kết hợp đồng)
- Cam kết pháp lý (legal commitment)
- Cam kết scope/deliverable ngoài những gì đã định nghĩa sẵn (vd hứa thêm tính năng chưa có)
- Cam kết timeline cụ thể với khách khi chưa xác nhận nội bộ khả thi

Ở nhóm này, agent **không được tự đề xuất số cụ thể thay cho user quyết định** — chỉ có thể trình bày lựa
chọn đã được định nghĩa sẵn (vd 3 tier giá đã cấu hình) và hỏi user chọn, hoặc nói rõ "việc này cần anh/chị
quyết định trực tiếp, em không tự đưa ra được."

## Cách dùng trong skill

Khi 1 skill (vd `campaign`, `proposal`, `engagement`) cần biết 1 hành động cụ thể thuộc nhóm nào, tra theo
ví dụ ở trên theo tinh thần (không cần liệt kê hết mọi trường hợp) — nếu hành động không khớp rõ ví dụ nào,
mặc định xếp vào **APPROVAL** (an toàn hơn AUTO khi không chắc).

`sme-orchestrator` là nơi ra quyết định approval khi điều phối multi-step goal. Với request đơn giản đi
thẳng vào 1 skill cụ thể (không qua orchestrator), skill đó tự tra policy này để quyết định.

## Ghi chú Phase 3 (chưa làm ở Phase 1)

Policy này được thiết kế để mở rộng thành rule-based autonomy — vd sau khi tích luỹ đủ dữ liệu action-log
(bao nhiêu suggestion được user đồng ý), có thể hạ 1 số action từ APPROVAL xuống AUTO có điều kiện (vd "gửi
follow-up lần 1 cho contact đã từng approve loại follow-up này 3 lần liên tiếp"). Không tự động hạ cấp bất
kỳ action nào ở Phase 1 — mọi phân loại trên là cố định cho tới khi có quyết định rõ ràng mở rộng.
