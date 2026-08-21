# COSMO — What It Is & How It Works (for BD users)

## COSMO là gì?

COSMO là nền tảng CRM + Sales Automation của Rockship, giúp đội BD (Business Development) quản lý contacts, chạy outreach campaigns, và theo dõi pipeline bán hàng — tất cả có AI hỗ trợ.

## Các khái niệm chính

### Contacts (Danh bạ)
- Mỗi contact là 1 người/lead trong hệ thống: tên, email, công ty, chức vụ, LinkedIn, v.v.
- Contact có **business_stage** theo pipeline: `INTRO` → `SEND` → `REPLIED` → `MEETING` → `QUALIFIED` → `WON`/`LOST`
- AI có thể **enrich** contact: tự động tìm thêm thông tin từ LinkedIn, website, v.v.
- Contact có **ai_insights**: pain points, interests, talking points do AI phân tích

### Contact Lists (Danh sách)
- Nhóm contacts lại theo mục đích: "Khách mời webinar tháng 4", "Leads SaaS Q2", v.v.
- Dùng làm **đầu vào cho campaigns** — mỗi campaign cần 1 contact list

### Campaigns (Chiến dịch email)
- Gửi email tự động đến 1 danh sách contacts
- Mỗi campaign có:
  - **Playbook** (chiến lược): cold_outreach, event_invite, revive_dormant_leads, upsell_existing_customers, content_offering, webinar_follow_up
  - **Templates** (mẫu email): AI tự generate dựa trên playbook + thông tin contact
  - **Agent** (tài khoản gửi): email account dùng để gửi (ví dụ: son@rockship.co)
- Flow tạo campaign: Chọn contacts → Tạo list → Chọn playbook → AI tạo templates → Activate → Emails tự gửi

### Templates (Mẫu email)
- Mỗi campaign có nhiều templates: First Email + Follow-up 1, 2, 3...
- AI generate nội dung dựa trên playbook và contact data
- **send_after**: thời gian chờ (tính bằng giờ) trước khi gửi template tiếp theo
- Templates có merge tags: {{first_name}}, {{company}}, v.v. — tự thay bằng data thật

### Outreach Pipeline
- Theo dõi trạng thái outreach từng contact: `COLD` → `NO_REPLY` → `REPLIED` → `POST_MEETING` → `DROPPED`
- AI có thể **suggest** ai nên outreach tiếp, **draft** email cá nhân hóa
- Khác với campaign (gửi hàng loạt), outreach là 1-1 cá nhân hóa

### Events (Sự kiện)
- Tạo events (webinar, workshop, meetup) với trang đăng ký public
- Có schedule, venue, capacity, takeaways, audience info
- Public page tự động tại `/events/{slug}`

### Meetings (Cuộc họp)
- Lịch họp với contacts: thời gian, channel (Google Meet, Zoom, in-person)
- AI generate **meeting brief**: tóm tắt contact, talking points, preparation notes

### Interactions (Lịch sử tương tác)
- Log mọi tương tác với contact: email, call, meeting, LinkedIn message, note
- Dùng để theo dõi relationship history và AI phân tích

### Knowledge Base (Cơ sở kiến thức)
- Tài liệu nội bộ: product info, case studies, pricing, FAQs
- AI search để trả lời câu hỏi hoặc soạn content (RAG)

### Daily Actions (Việc cần làm hôm nay)
- AI tự động suggest: ai cần follow-up, meeting nào sắp tới, lead nào nóng
- Briefing hàng ngày cho BD team

### Segmentations (Phân khúc)
- Chia contacts theo tiêu chí: industry, company size, engagement level
- AI tính **segment scores** để xếp hạng leads theo ICP (Ideal Customer Profile)

### Agents (Tài khoản email)
- Email accounts kết nối với COSMO để gửi campaigns
- Mỗi agent có daily_limit (giới hạn gửi/ngày) để tránh spam

## Playbooks — Khi nào dùng gì?

| Playbook | Dùng khi |
|----------|----------|
| `cold_outreach` | Contact mới, chưa biết mình |
| `event_invite` | Mời tham gia event/webinar |
| `revive_dormant_leads` | Contact cũ, lâu không tương tác |
| `upsell_existing_customers` | Khách hàng hiện tại, giới thiệu thêm |
| `content_offering` | Chia sẻ content (blog, whitepaper, case study) |
| `webinar_follow_up` | Follow-up sau webinar |

## Flow điển hình của BD

1. **Tìm leads**: Search contacts hoặc dùng Apollo enrichment
2. **Tạo list**: Nhóm leads theo campaign mục tiêu
3. **Chạy campaign**: Chọn playbook phù hợp → AI tạo emails → Activate
4. **Theo dõi**: Check replies, update outreach state, log interactions
5. **Meeting**: Schedule meeting với leads quan tâm, AI prep briefing
6. **Close**: Update business_stage → WON/LOST
