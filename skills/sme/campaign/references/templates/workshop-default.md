---
template_id: workshop-default
event_type: workshop
placeholders:
  required: [title, date_ict, duration, venue, capacity, price_vnd, luma_url, takeaways, agenda]
  optional: [what_to_bring, organizer_email, brand_name, payment_info]
---

## Event description (for Luma upload)

# {{title}}

**Ngày giờ**: {{date_ict}} · Duration: {{duration}} · Venue: {{venue}} · Giới hạn: {{capacity}} slot

## Bạn sẽ học được gì?

{{takeaways}}

## Agenda

{{agenda}}

## Cần mang theo

{{what_to_bring}}

## Đầu tư

**{{price_vnd}} VND** / người — bao gồm tài liệu, F&B, hands-on exercises.

## Đăng ký

Slot giới hạn. Register tại: {{luma_url}}

Sau khi đăng ký, em sẽ gửi email hướng dẫn chuyển khoản. Slot chỉ confirm sau khi nhận được payment.

---

## Facebook caption (250-400 chars)

🎓 **{{title}}** — {{date_short}}

Hands-on workshop — {{capacity}} slot duy nhất. Học trực tiếp tại {{venue}}.

{{takeaways_short}}

Đầu tư: {{price_vnd}}đ
Register: {{luma_url}}

#Workshop #{{brand_name}}

---

## LinkedIn caption (500-800 chars)

📅 {{date_ict}} | Hands-on Workshop — {{title}}

Workshop giới hạn {{capacity}} người, hands-on exercises.

{{takeaways}}

**Agenda:**
{{agenda}}

**Venue**: {{venue}}
**Đầu tư**: {{price_vnd}} VND / người

Register: {{luma_url}}

#Workshop #{{brand_name}}

---

## Confirmation email (sau khi registered) — workshop paid: ASK payment

**Subject**: Đăng ký {{title}} — Xác nhận + hướng dẫn chuyển khoản

Chào {{first_name}},

Cảm ơn bạn đã đăng ký workshop **{{title}}** ({{date_ict}}).

Workshop giới hạn {{capacity}} slot, để giữ chỗ vui lòng chuyển khoản:

{{payment_info}}

Nội dung: `WORKSHOP {{title_short}} {{first_name}}`

Số tiền: **{{price_vnd}} VND**

Sau khi nhận được payment, em confirm slot + gửi Zoom/venue link chi tiết. Hạn chuyển khoản: trước ngày {{payment_deadline}}.

Nếu cần hỗ trợ, reply email này.

{{signature}}

---

## Confirmed-paid email (sau khi anh confirm payment)

**Subject**: ✅ Confirmed slot — {{title}} · {{date_ict}}

Chào {{first_name}},

Slot của bạn cho workshop **{{title}}** đã confirmed.

**Chi tiết:**
- Ngày giờ: {{date_ict}}
- Venue: {{venue}}
- Zoom link: {{zoom_url}}
- Duration: {{duration}}
- Cần mang: {{what_to_bring}}

Tôi sẽ gửi reminder email 1 ngày trước + 1 giờ trước event.

Hẹn gặp bạn tại workshop!

{{signature}}
