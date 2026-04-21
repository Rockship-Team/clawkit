---
template_id: webinar-default
event_type: webinar
placeholders:
  required: [title, date_ict, duration, luma_url, takeaways, agenda]
  optional: [speakers, organizer_email, brand_name]
---

## Event description (for Luma upload)

# {{title}}

**Ngày giờ**: {{date_ict}} · Duration: {{duration}} · Format: Webinar online

## Bạn sẽ học được gì?

{{takeaways}}

## Agenda

{{agenda}}

## Ai nên tham gia?

{{audience}}

## Đăng ký

Miễn phí. Register tại: {{luma_url}}

{{speakers}}

---

## Facebook caption (250-400 chars)

🎓 **{{title}}** — {{date_short}} lúc {{time}}

{{takeaways_short}}

Miễn phí — đăng ký: {{luma_url}}

#AI #Webinar #{{brand_name}}

---

## LinkedIn caption (500-800 chars, professional)

📅 {{date_ict}} | Webinar — {{title}}

{{takeaways}}

**Ai nên tham gia:**
{{audience}}

**Agenda:**
{{agenda}}

Đăng ký: {{luma_url}}

#AI #Webinar #{{brand_name}}

---

## Email announcement

**Subject**: {{title}} — {{date_short}}

Hi {{first_name}},

Tôi viết để mời bạn tham gia webinar **{{title}}** vào {{date_ict}}.

**Bạn sẽ học được:**
{{takeaways}}

**Chi tiết:**
- Date: {{date_ict}}
- Duration: {{duration}}
- Format: Online (link gửi sau khi đăng ký)
- Cost: Miễn phí

Đăng ký: {{luma_url}}

Rất mong gặp bạn.

{{signature}}

---

## Thank-you email (sau khi registered) — free event auto-send

**Subject**: Đã nhận đăng ký — {{title}}

Chào {{first_name}},

Cảm ơn bạn đã đăng ký webinar **{{title}}**.

**Chi tiết event:**
- Ngày giờ: {{date_ict}}
- Zoom link: {{zoom_url}}
- Duration: {{duration}}

Tôi sẽ gửi reminder email 1 ngày trước + 1 giờ trước event. Nếu có thắc mắc, reply email này.

Hẹn gặp bạn tại event!

{{signature}}
