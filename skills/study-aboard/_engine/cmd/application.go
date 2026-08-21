package main

import (
	"fmt"
	osexec "os/exec"
	"time"
)

func cmdApplication(args []string) {
	if len(args) == 0 {
		errOut("usage: application add|list|dashboard|update")
	}
	switch args[0] {
	case "add":
		// application add <student_id> <university_id> <university_name> <category> <reach|target|safety> <ED|EA|RD|rolling> <deadline|-> <channel> <channel_user_id>
		if len(args) < 9 {
			errOut("usage: application add <student_id> <university_id> <university_name> <category> <type> <deadline|-> <channel> <channel_user_id>")
		}
		studentID, uniID, uniName := args[1], args[2], args[3]
		category, appType, deadline := args[4], args[5], args[6]
		channel, channelUserID := args[7], args[8]
		if deadline == "-" {
			deadline = ""
		}

		appID := newID()
		now := vnNowISO()
		var deadlineVal interface{} = nil
		if deadline != "" {
			deadlineVal = deadline
		}
		_, err := exec(
			`INSERT INTO application (id,student_id,university_id,university_name,category,application_type,deadline,created_at,updated_at)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			appID, studentID, uniID, uniName, category, appType, deadlineVal, now, now,
		)
		if err != nil {
			errOut("insert failed: " + err.Error())
		}

		// Register cron reminder jobs for 30/14/7/1 days before deadline
		cronJobs := []map[string]interface{}{}
		if deadline != "" {
			dl, err2 := time.Parse("2006-01-02", deadline)
			if err2 == nil {
				for _, days := range []int{30, 14, 7, 1} {
					remindDate := dl.AddDate(0, 0, -days)
					if remindDate.After(vnNow()) {
						jobName := fmt.Sprintf("deadline-%dd-%s-%s", days, studentID[:8], appID[:8])
						schedule := fmt.Sprintf("0 8 %d %d *", remindDate.Day(), int(remindDate.Month()))
						message := fmt.Sprintf(
							"Deadline reminder: %s %s — còn %d ngày! Channel: %s, User: %s",
							uniName, appType, days, channel, channelUserID,
						)
						result := triggerCron(jobName, schedule, message)
						result["days"] = days
						result["date"] = remindDate.Format("2006-01-02")
						cronJobs = append(cronJobs, result)

						cronID := newID()
						exec( //nolint
							`INSERT INTO cron_job (id,student_id,openclaw_job_name,job_type,application_id,urgency_days,scheduled_at)
							 VALUES (?,?,?,?,?,?,?)`,
							cronID, studentID, jobName, "deadline_reminder", appID, days, remindDate.Format("2006-01-02"),
						)
					}
				}
			}
		}

		okOut(map[string]interface{}{
			"application_id": appID,
			"university":     uniName,
			"type":           appType,
			"deadline":       deadline,
			"cron_jobs":      cronJobs,
		})

	case "list":
		// application list <student_id>
		if len(args) < 2 {
			errOut("usage: application list <student_id>")
		}
		rows, err := queryRows(
			`SELECT id,university_name,category,application_type,deadline,
			 essay_status,submission_status,scores_sent,recs_submitted,financial_aid_forms_done,notes
			 FROM application WHERE student_id=? ORDER BY deadline ASC`,
			args[1],
		)
		if err != nil {
			errOut(err.Error())
		}
		today := vnToday()
		for _, r := range rows {
			dl, _ := r["deadline"].(string)
			r["urgency"] = urgencyFlag(dl, today)
			r["days_until_deadline"] = daysUntil(dl, today)
		}
		okOut(map[string]interface{}{"applications": rows, "count": len(rows), "student_id": args[1]})

	case "dashboard":
		// application dashboard <student_id>
		if len(args) < 2 {
			errOut("usage: application dashboard <student_id>")
		}
		profile, _ := queryOne(
			"SELECT display_name,intended_major,annual_budget_usd FROM student_profile WHERE id=?",
			args[1],
		)
		rows, err := queryRows(
			`SELECT id,university_name,category,application_type,deadline,essay_status,submission_status,notes
			 FROM application WHERE student_id=? ORDER BY deadline ASC`,
			args[1],
		)
		if err != nil {
			errOut(err.Error())
		}
		today := vnToday()
		var urgent []map[string]interface{}
		for _, r := range rows {
			dl, _ := r["deadline"].(string)
			r["urgency"] = urgencyFlag(dl, today)
			d := daysUntil(dl, today)
			r["days_until_deadline"] = d
			if d != nil && *d <= 30 && r["submission_status"] != "submitted" {
				urgent = append(urgent, r)
			}
		}
		okOut(map[string]interface{}{
			"student_id": args[1], "profile": profile,
			"applications": rows, "urgent": urgent, "count": len(rows),
		})

	case "update":
		// application update <app_id> <field> <value>
		if len(args) < 4 {
			errOut("usage: application update <app_id> <field> <value>")
		}
		appID, field, val := args[1], args[2], args[3]
		allowed := map[string]bool{
			"essay_status": true, "submission_status": true, "notes": true,
			"deadline": true, "scores_sent": true, "recs_submitted": true,
			"financial_aid_forms_done": true,
		}
		if !allowed[field] {
			errOut("cannot update field: " + field)
		}
		q := fmt.Sprintf("UPDATE application SET %s=?, updated_at=? WHERE id=?", field)
		_, err := exec(q, val, vnNowISO(), appID)
		if err != nil {
			errOut(err.Error())
		}
		// Auto-cancel cron reminders when submitted / removed / missed
		if field == "submission_status" && (val == "submitted" || val == "removed" || val == "missed") {
			cancelCronsForApp(appID)
		}
		okOut(map[string]interface{}{"updated": appID, "field": field, "value": val})

	default:
		errOut("unknown application command: " + args[0])
	}
}

// --- Helpers ---

func urgencyFlag(deadline, today string) string {
	d := daysUntil(deadline, today)
	if d == nil {
		return "GREEN"
	}
	switch {
	case *d < 0:
		return "PASSED"
	case *d <= 7:
		return "RED"
	case *d <= 14:
		return "ORANGE"
	case *d <= 30:
		return "YELLOW"
	default:
		return "GREEN"
	}
}

func daysUntil(deadline, today string) *int {
	if deadline == "" {
		return nil
	}
	dl, err := time.Parse("2006-01-02", deadline)
	if err != nil {
		return nil
	}
	td, err := time.Parse("2006-01-02", today)
	if err != nil {
		return nil
	}
	d := int(dl.Sub(td).Hours() / 24)
	return &d
}

func triggerCron(jobName, schedule, message string) map[string]interface{} {
	c := osexec.Command(
		"openclaw", "cron", "add", jobName,
		"--schedule", schedule,
		"--session", "isolated",
		"--delete-after-run",
		"--tz", "Asia/Ho_Chi_Minh",
		"--message", message,
	)
	err := c.Run()
	return map[string]interface{}{"job_name": jobName, "ok": err == nil}
}

func cancelCronsForApp(appID string) {
	jobs, _ := queryRows("SELECT openclaw_job_name FROM cron_job WHERE application_id=? AND status='active'", appID)
	for _, j := range jobs {
		jobName := fmt.Sprintf("%v", j["openclaw_job_name"])
		osexec.Command("openclaw", "cron", "delete", jobName).Run() //nolint
		exec("UPDATE cron_job SET status='cancelled' WHERE openclaw_job_name=?", jobName) //nolint
	}
}
