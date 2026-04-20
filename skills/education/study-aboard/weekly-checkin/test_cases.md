# Test Cases - Weekly Check-in Skill

## 1. Trigger Mode Tests

### TC-WC-001: Cron Auto-Trigger
| Condition | Sunday 10:00 AM student timezone |
|-----------|----------------------------------|
| Source | weekly_checkin_{student_id} cron job |
| Data Collection | 6 CLI commands in sequence |
| Response | Full weekly summary |

### TC-WC-002: On-Demand Trigger
| Input | "Tuần này em thế nào" / "cập nhật tiến độ" / "check-in" |
|-------|----------------------------------------------------------|
| Action | Same data collection as cron |
| Response | Full weekly summary |

## 2. Data Collection Tests

### TC-WC-003: CLI Sequence
| Order | Command | Data Retrieved |
|-------|---------|----------------|
| 1 | `sa-cli student query {channel} {uid}` | Profile + timezone |
| 2 | `sa-cli application dashboard {student_id}` | Applications + urgency |
| 3 | `sa-cli plan list {student_id}` | Active study plans |
| 4 | `sa-cli essay list {student_id}` | Essay drafts + rounds |
| 5 | `sa-cli ec list {student_id}` | EC activities + tiers |
| 6 | `sa-cli checklist get {student_id}` | Student-wide checklist |

## 3. Weekly Summary Display Tests

### TC-WC-004: Summary Header
| Element | Format |
|---------|--------|
| Title | 📆 CHECK-IN TUẦN — {display_name} |
| Date | {current_date} |
| Urgency | Còn {days} ngày đến deadline gần nhất ({school}) |

### TC-WC-005: Section Display
| Section | Content | Warning |
|---------|---------|---------|
| 📚 ÔN THI | Plan type, current → target, weekly tasks | "⚠️ Em chưa báo điểm tuần này" if missing |
| ✍️ ESSAY | School, type, status, round, last update | "⚠️ Chưa cập nhật 1 tuần" if >7 days |
| 📅 DEADLINE | Urgent apps (≤30 days), essay status | Urgency icons 🔴🟡🟠 |
| 📋 CHECKLIST | {done}/{total} + flagged items | ⚠️ for pending rec letters, CSS, FAFSA |
| 🏆 HOẠT ĐỘNG | EC name, tier, status | "⚠️ Chưa cập nhật 30 ngày" if stale |

## 4. Priority Action Tests

### TC-WC-006: Priority Ranking Logic
| Priority | Condition | Max |
|----------|-----------|-----|
| 1. CRITICAL | Deadline ≤ 7 days | 1st |
| 2. HIGH | Essay not_started + deadline ≤ 30 | 2nd |
| 3. MEDIUM | Study plan score not reported | 3rd |
| 4. MEDIUM | Checklist: rec letters, CSS/FAFSA | 4th |
| 5. LOW | EC not updated > 30 days + upcoming event | 5th |
| 6. LOW | Essay draft not updated > 7 days | 6th |

### TC-WC-007: Priority Display Format
```
🎯 ƯU TIÊN TUẦN NÀY:
1. {highest_priority_action}
2. {second_priority_action}
3. {third_priority_action}

Em muốn bắt đầu từ đâu? Mình sẵn sàng hỗ trợ ngay!
```

## 5. No Active Workstream Tests

### TC-WC-008: Empty State
| Condition | active_plans empty AND applications empty |
|-----------|-------------------------------------------|
| Message | "Mình thấy em chưa bắt đầu giai đoạn apply..." |
| Suggestions | 1️⃣ Đánh giá hồ sơ 2️⃣ Chọn trường 3️⃣ Lộ trình SAT |

## 6. Snooze/Pause Tests

### TC-WC-009: Pause Request
| Input | "Tạm dừng nhắc tuần" |
|-------|----------------------|
| Options | • 1 tuần • 2 tuần • Cho đến khi em nhắn lại |
| Warning | "⚠️ Nếu có deadline ≤ 7 ngày, mình vắn nhắc" |
| CLI | `sa-cli cron pause {student_id} weekly_checkin {weeks}` |

## 7. Safety Check Tests

### TC-WC-SAFETY-001: Protected Actions
| Request | Response |
|---------|----------|
| "Đảm bảo em đỗ" | "Mình không thể đảm bảo kết quả..." |
| "Tắt nhắc nhở vĩnh viễn" | "Mình có thể tạm dừng — nhưng khuyến khích duy trì nhịp..." |

### TC-WC-SAFETY-002: Emotional Distress
| Trigger | Action |
|---------|--------|
| Distress in check-in message | Empathy-first BEFORE showing any progress data |
| Severe distress | Extended support message + resources |

## 8. Integration Tests

### TC-WC-010: Cross-Skill Data Merge
| Skill | Data Used in Check-in |
|-------|----------------------|
| deadline-tracker | Application urgency, essay status |
| study-plan | Plan progress, last check-in score |
| essay-review | Draft versions, last updated |
| ec-strategy | Activity tiers, update staleness |
| profile-assessment | Budget, target countries |

### TC-WC-011: Cron Registration
| Registration Point | Command |
|-------------------|---------|
| Profile created (profile-assessment) | `sa-cli cron register {student_id} weekly_checkin "0 10 * * 0" {channel} {uid}` |
| Job Name | `weekly_checkin_{student_id}` |
| Fire Time | Every Sunday 10:00 AM local |
