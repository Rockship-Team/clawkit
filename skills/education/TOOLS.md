# TOOLS

> Skills define _how_ tools work.
> File này chứa **setup cụ thể của deployment này** — paths, env vars, conventions.

---

## CLI Binary — `sa-cli`

Tất cả logic nghiệp vụ nằm trong một Go binary duy nhất.

| | |
|---|---|
| Source | `skills/_cli/cmd/` |
| Binary | `skills/_cli/sa-cli` |
| Database | `skills/_cli/sa-data/sa.db` (SQLite, WAL mode) |

Build:
```bash
cd skills/_cli/cmd && go build -o ../sa-cli .
```

Init DB (tạo tất cả tables):
```bash
skills/_cli/sa-cli init
```

Reset DB:
```bash
rm skills/_cli/sa-data/sa.db && skills/_cli/sa-cli init
```

---

## Database

Một file SQLite duy nhất với tất cả tables — không còn per-skill DB:

| Table | Dùng bởi |
|-------|----------|
| `student_profile` | profile-assessment |
| `extracurricular_activity` | ec-strategy, profile-assessment |
| `university_record` | school-matching, financial-aid |
| `application` | deadline-tracker, school-matching, financial-aid |
| `student_checklist` | deadline-tracker |
| `study_plan` + `study_plan_checkin` | study-plan |
| `essay_draft` | essay-review |
| `visa_checklist` | pre-departure |
| `admission_offer` | offer-comparison |
| `cron_job` | deadline-tracker, study-plan |

---

## CLI Commands

```
sa-cli init                              Tạo DB và schema

sa-cli student query <channel> <uid>     Lấy profile + activities
sa-cli student save '<json>'             Tạo/cập nhật profile
sa-cli student list
sa-cli student update <id> <field> <val>
sa-cli student scorecard <id>

sa-cli application add <student_id> <uni_id> "<uni_name>" <category> <type> <deadline|-> <channel> <uid>
sa-cli application list <student_id>
sa-cli application dashboard <student_id>
sa-cli application update <app_id> <field> <value>

sa-cli checklist get <student_id>
sa-cli checklist update <student_id> <item_key> <0|1>
sa-cli checklist notes <student_id> "<text>"

sa-cli university list [country]
sa-cli university get <id>
sa-cli university search <query> [country]
sa-cli university seed <json_file>
sa-cli university match <student_id> [limit]

sa-cli financial cost-compare <student_id>

sa-cli plan create <student_id> <type> <target> <test_date> <current_score> <channel> <uid>
sa-cli plan list <student_id>
sa-cli plan checkin <plan_id> <score> "<notes>"

sa-cli essay submit <student_id> "<type>" "<prompt>" "<content>"
sa-cli essay save-scores <draft_id> '<scores_json>' '<feedback_json>' <true|false>
sa-cli essay list <student_id>
sa-cli essay get <draft_id>

sa-cli ec add <student_id> "<name>" "<role>" <hours_per_week> "<achievements>"
sa-cli ec list <student_id>
sa-cli ec update-tier <activity_id> <tier> "<notes>"

sa-cli visa get <student_id>
sa-cli visa update <student_id> <field> <value>

sa-cli offer add <student_id> "<uni_name>" <type> <result> <tuition> <room_board> <other_fees> <scholarship> <grant> <deadline> <deposit> "<major>" <start_date>
sa-cli offer update <offer_id> <field> <value>
sa-cli offer list <student_id>
sa-cli offer decide <student_id> "<uni_name>" <accepted|declined|enrolled>
sa-cli offer compare <student_id>

sa-cli cron list [student_id]
sa-cli cron cancel <cron_name>
sa-cli cron register <student_id> <job_name> "<cron_expr>" <channel> <uid>
sa-cli cron pause <student_id> <job_name> <weeks>    # weeks=0 → pause indefinitely

sa-cli config show|set|get
```

---

## University Knowledge Base

Seed data: `skills/school-matching/data/seed_{COUNTRY}.json` (US, UK, CA, AU). US có thêm batch files: `seed_US_batch3.json`, `seed_US_batch4.json`, `seed_US_extended.json`.

Load vào DB:
```bash
skills/_cli/sa-cli university seed skills/school-matching/data/seed_US.json
```

Cập nhật hàng năm vào tháng 8 (sau khi các trường release Common Data Set mới).

---

## Skill → Workflow Phase Mapping

```
ONBOARD/ASSESS  →  profile-assessment   (lưu profile + scorecard)
       ↓
PLAN            →  school-matching      (danh sách Reach/Target/Safety)
                →  study-plan           (lộ trình SAT/TOEFL/IELTS + GPA)
                →  ec-strategy          (đánh giá và nâng cấp EC)
       ↓
EXECUTE         →  essay-review         (brainstorm → draft → polish)
                →  deadline-tracker     (dashboard + cron reminders)
                →  financial-aid        (so sánh chi phí + CSS Profile)
                →  weekly-checkin       (tổng hợp tiến độ hàng tuần, Chủ nhật 10:00)
       ↓
RESULT          →  offer-comparison     (so sánh offer + ra quyết định)
       ↓
PRE-DEP         →  pre-departure        (visa F-1, nhà ở, orientation)
```

Mỗi skill kết thúc bằng **Phase Transition prompt** gợi ý skill tiếp theo.

---

## Channels

| Channel | Identifier |
|---------|-----------|
| Telegram | `channel = "telegram"` |
| Zalo OA | `channel = "zalo"` |
| Web | `channel = "web"` |

Zalo messages chỉ gửi trong khung **06:00–22:00 VNT** (`Asia/Ho_Chi_Minh`).

---

## Cron Jobs

### Deadline reminders (deadline-tracker)
Pattern: `deadline-{days}d-{student_id[:8]}-{app_id[:8]}`

Triggers: 30d → 14d → 7d → 1d trước deadline.
Auto-cancelled khi `submission_status = submitted`, `missed`, hoặc `removed`.

```bash
openclaw cron add "deadline-30d-abc12345-def67890" \
  --schedule "0 8 * * *" \
  --session isolated \
  --delete-after-run \
  --tz "Asia/Ho_Chi_Minh"
```

### Weekly check-in (weekly-checkin)
Pattern: `weekly_checkin_{student_id}`
Schedule: Chủ nhật 10:00 VNT (`0 10 * * 0`)

```bash
sa-cli cron register {student_id} weekly_checkin "0 10 * * 0" {channel} {channel_user_id}
```

Registered automatically by `sa-cli student save` (profile-assessment) when a new profile is created.
Pause: `sa-cli cron pause {student_id} weekly_checkin {weeks}` (weeks=0 → đến khi student tự resume).
Hard override: nếu có deadline ≤ 7 ngày, cron này **không thể bị tắt** — deadline-tracker gửi riêng.

### Weekly study check-in (study-plan)
Pattern: `checkin-weekly-{student_id[:8]}`
Schedule: Thứ Hai 08:00 VNT (`0 8 * * 1`)

```bash
sa-cli cron register {student_id} checkin-weekly "0 8 * * 1" {channel} {channel_user_id}
```

Registered automatically by `sa-cli plan create`.

---

## AP Scores Format

Lưu trong `student_profile.ap_scores` dạng JSON array string:
```json
[{"subject": "Calculus BC", "score": 5}, {"subject": "CS A", "score": 4}]
```

Pass qua `sa-cli student save` với key `ap_scores`. Giá trị rỗng: `"[]"`.

---

## Safety Rules

`safety_rules.md` tại project root — hard limits cho tất cả skills.

Tất cả SKILL.md đều có: `For the full rules list see ../../safety_rules.md.`
