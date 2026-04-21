package main

import (
	osexec "os/exec"
)

func cmdCron(args []string) {
	if len(args) == 0 {
		errOut("usage: cron list|cancel")
	}
	switch args[0] {
	case "list":
		// cron list <student_id>
		if len(args) < 2 {
			errOut("usage: cron list <student_id>")
		}
		rows, err := queryRows(
			`SELECT c.id,c.openclaw_job_name,c.job_type,c.urgency_days,c.scheduled_at,c.status,
			 a.university_name as application_name
			 FROM cron_job c
			 LEFT JOIN application a ON a.id=c.application_id
			 WHERE c.student_id=? ORDER BY c.scheduled_at ASC`,
			args[1],
		)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"cron_jobs": rows, "count": len(rows)})

	case "cancel":
		// cron cancel <job_name>
		if len(args) < 2 {
			errOut("usage: cron cancel <job_name>")
		}
		jobName := args[1]
		osexec.Command("openclaw", "cron", "delete", jobName).Run() //nolint
		_, err := exec("UPDATE cron_job SET status='cancelled' WHERE openclaw_job_name=?", jobName)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"cancelled": jobName})

	default:
		errOut("unknown cron command: " + args[0])
	}
}
