package main

import (
	osexec "os/exec"
)

func cmdPlan(args []string) {
	if len(args) == 0 {
		errOut("usage: plan create|list|checkin")
	}
	switch args[0] {
	case "create":
		// plan create <student_id> <sat|toefl|ielts|act> <current_score|-> <target_score> <test_date>
		if len(args) < 6 {
			errOut("usage: plan create <student_id> <test_type> <current_score|-> <target_score> <test_date>")
		}
		studentID := args[1]
		testType := args[2]
		var currentScore interface{}
		if args[3] != "-" {
			currentScore = args[3]
		}
		targetScore := args[4]
		testDate := args[5]

		// Deactivate any previous active plan for same test type
		exec("UPDATE study_plan SET active=0 WHERE student_id=? AND test_type=? AND active=1", studentID, testType)

		planID := newID()
		now := vnNowISO()
		_, err := exec(
			`INSERT INTO study_plan (id,student_id,test_type,current_score,target_score,test_date,plan_weeks,active,created_at,updated_at)
			 VALUES (?,?,?,?,?,?,?,1,?,?)`,
			planID, studentID, testType, currentScore, targetScore, testDate, "[]", now, now,
		)
		if err != nil {
			errOut("insert failed: " + err.Error())
		}

		// Register weekly check-in cron
		jobName := "studyplan-weekly-" + planID[:8]
		cronID := newID()
		schedule := "0 9 * * 1" // every Monday 9am
		message := "Weekly study plan check-in: How is your " + testType + " prep going? Gõ điểm practice test gần nhất nhé!"
		cronResult := triggerCronStudy(jobName, schedule, message)

		exec(
			`INSERT INTO cron_job (id,student_id,openclaw_job_name,job_type,study_plan_id,scheduled_at)
			 VALUES (?,?,?,?,?,?)`,
			cronID, studentID, jobName, "weekly_checkin", planID, vnToday(),
		)

		okOut(map[string]interface{}{
			"plan_id":       planID,
			"test_type":     testType,
			"target_score":  targetScore,
			"test_date":     testDate,
			"weekly_cron":   cronResult,
		})

	case "list":
		// plan list <student_id>
		if len(args) < 2 {
			errOut("usage: plan list <student_id>")
		}
		rows, err := queryRows(
			`SELECT p.id,p.test_type,p.current_score,p.target_score,p.test_date,p.active,p.created_at,
			 (SELECT reported_score FROM study_plan_checkin WHERE study_plan_id=p.id ORDER BY checkin_date DESC LIMIT 1) as latest_score
			 FROM study_plan p WHERE p.student_id=? ORDER BY p.active DESC, p.created_at DESC`,
			args[1],
		)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"plans": rows, "count": len(rows)})

	case "checkin":
		// plan checkin <plan_id> <score> [notes]
		if len(args) < 3 {
			errOut("usage: plan checkin <plan_id> <score> [notes]")
		}
		planID := args[1]
		score := args[2]
		notes := ""
		if len(args) > 3 {
			notes = args[3]
		}
		checkinID := newID()
		_, err := exec(
			`INSERT INTO study_plan_checkin (id,study_plan_id,checkin_date,reported_score,adjustment_notes)
			 VALUES (?,?,?,?,?)`,
			checkinID, planID, vnNowISO(), score, notes,
		)
		if err != nil {
			errOut(err.Error())
		}
		// Update current_score on the plan
		exec("UPDATE study_plan SET current_score=?, updated_at=? WHERE id=?", score, vnNowISO(), planID)

		plan, _ := queryOne("SELECT test_type,target_score FROM study_plan WHERE id=?", planID)
		okOut(map[string]interface{}{
			"checkin_id":    checkinID,
			"plan_id":       planID,
			"reported_score": score,
			"plan":          plan,
		})

	default:
		errOut("unknown plan command: " + args[0])
	}
}

func triggerCronStudy(jobName, schedule, message string) map[string]interface{} {
	c := osexec.Command(
		"openclaw", "cron", "add", jobName,
		"--schedule", schedule,
		"--session", "isolated",
		"--tz", "Asia/Ho_Chi_Minh",
		"--message", message,
	)
	err := c.Run()
	return map[string]interface{}{"job_name": jobName, "ok": err == nil}
}
