package main

import (
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"strings"
	"time"
)

// cmdCronHealth dispatches cron health subcommands.
//
//	sme-cli cron-health log          # pull latest run từ openclaw, lưu vào DB
//	sme-cli cron-health check        # tổng hợp health + đề xuất fix
//	sme-cli cron-health autofix      # apply known fixes tự động
//	sme-cli cron-health list         # xem lịch sử lỗi gần đây
func cmdCronHealth(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: cron-health log|check|autofix|list")
		os.Exit(1)
	}
	ensureCronHealthTable()
	switch args[0] {
	case "log":
		cronHealthLog(args[1:])
	case "check":
		cronHealthCheck(args[1:])
	case "autofix":
		cronHealthAutofix(args[1:])
	case "list":
		cronHealthList(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "usage: cron-health log|check|autofix|list\n")
		os.Exit(1)
	}
}

func ensureCronHealthTable() {
	mustDB().Exec(`CREATE TABLE IF NOT EXISTS cron_health_log (
		id            TEXT PRIMARY KEY,
		job_id        TEXT NOT NULL,
		job_name      TEXT NOT NULL,
		run_id        TEXT,
		run_at        TEXT NOT NULL,
		status        TEXT NOT NULL,
		error_type    TEXT,
		error_detail  TEXT,
		duration_ms   INTEGER,
		model         TEXT,
		delivered     INTEGER DEFAULT 0,
		fix_applied   TEXT,
		fix_at        TEXT,
		created_at    TEXT NOT NULL
	)`)
	mustDB().Exec(`CREATE INDEX IF NOT EXISTS idx_cron_health_job ON cron_health_log(job_name, run_at)`)
	mustDB().Exec(`CREATE INDEX IF NOT EXISTS idx_cron_health_status ON cron_health_log(status, error_type)`)
}

// ── Error classification ──────────────────────────────────────────────────────

type errorType string

const (
	errTimeout         errorType = "timeout"
	errSessionConflict errorType = "session_conflict"
	errModelUnavail    errorType = "model_unavailable"
	errDeliveryFailed  errorType = "delivery_failed"
	errReasoningLeak   errorType = "reasoning_leak"
	errUnknown         errorType = "unknown"
)

var reasoningLeakKeywords = []string{
	"kpi_set =", "needs_reminder =", "cần gửi reminder",
	"Gửi đúng dòng", "Bước 1:", "Bước 2:",
	"kpi_set = true", "kpi_set = false",
}

func classifyError(errMsg, summary string, timedOut bool) errorType {
	if timedOut {
		return errTimeout
	}
	if strings.Contains(errMsg, "EmbeddedAttemptSessionTakeoverError") ||
		strings.Contains(errMsg, "session file changed") {
		return errSessionConflict
	}
	if strings.Contains(errMsg, "FallbackSummaryError") ||
		strings.Contains(errMsg, "All models failed") ||
		strings.Contains(errMsg, "No API key found") {
		return errModelUnavail
	}
	for _, kw := range reasoningLeakKeywords {
		if strings.Contains(summary, kw) {
			return errReasoningLeak
		}
	}
	if errMsg != "" {
		return errUnknown
	}
	return ""
}

// ── openclaw cron runs reader ─────────────────────────────────────────────────

type cronRunEntry struct {
	JobID        string
	JobName      string
	RunID        string
	RunAt        string
	Status       string
	Error        string
	DurationMs   int64
	Model        string
	Delivered    bool
	Summary      string
	TimedOut     bool
}

func readCronRuns(jobID string) ([]cronRunEntry, error) {
	out, err := osexec.Command("openclaw", "cron", "runs", "--id", jobID).Output()
	if err != nil {
		// fallback: might need -u rockship context — try with full path
		out, err = osexec.Command("/usr/bin/openclaw", "cron", "runs", "--id", jobID).Output()
		if err != nil {
			return nil, fmt.Errorf("openclaw cron runs: %w", err)
		}
	}
	var result struct {
		Entries []struct {
			JobID      string `json:"jobId"`
			RunID      string `json:"runId"`
			Action     string `json:"action"`
			Status     string `json:"status"`
			Error      string `json:"error"`
			DurationMs int64  `json:"durationMs"`
			Model      string `json:"model"`
			Delivered  bool   `json:"delivered"`
			Summary    string `json:"summary"`
			RunAtMs    int64  `json:"runAtMs"`
			Diagnostics struct {
				Entries []struct {
					Message string `json:"message"`
				} `json:"entries"`
			} `json:"diagnostics"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parse cron runs: %w", err)
	}
	var runs []cronRunEntry
	for _, e := range result.Entries {
		if e.Action != "finished" {
			continue
		}
		runAt := time.UnixMilli(e.RunAtMs).In(func() *time.Location {
			l, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
			return l
		}()).Format(time.RFC3339)

		timedOut := strings.Contains(e.Error, "timed out") || strings.Contains(e.Error, "timeout")
		runs = append(runs, cronRunEntry{
			JobID:      e.JobID,
			RunID:      e.RunID,
			RunAt:      runAt,
			Status:     e.Status,
			Error:      e.Error,
			DurationMs: e.DurationMs,
			Model:      e.Model,
			Delivered:  e.Delivered,
			Summary:    e.Summary,
			TimedOut:   timedOut,
		})
	}
	return runs, nil
}

// ── cron-health log ───────────────────────────────────────────────────────────

func cronHealthLog(args []string) {
	// Đọc danh sách jobs từ openclaw
	out, err := osexec.Command("openclaw", "cron", "list", "--json").Output()
	if err != nil {
		out, err = osexec.Command("/usr/bin/openclaw", "cron", "list", "--json").Output()
	}

	var jobList []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err == nil {
		var result struct {
			Jobs []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"jobs"`
		}
		if json.Unmarshal(out, &result) == nil {
			jobList = result.Jobs
		}
	}

	// Fallback: đọc trực tiếp từ jobs.json
	if len(jobList) == 0 {
		for _, je := range readJobsFromFile() {
			jobList = append(jobList, struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: je.ID, Name: je.Name})
		}
	}

	logged := 0
	errors := 0
	now := vnNowISO()

	for _, job := range jobList {
		runs, err := readCronRuns(job.ID)
		if err != nil {
			continue
		}
		for _, run := range runs {
			// Skip nếu đã log rồi
			existing, _ := queryOne(`SELECT id FROM cron_health_log WHERE run_id = ?`, run.RunID)
			if existing != nil {
				continue
			}

			et := classifyError(run.Error, run.Summary, run.TimedOut)
			delivered := 0
			if run.Delivered {
				delivered = 1
			}

			// Detect delivery failure even when status=ok
			if run.Status == "ok" && !run.Delivered {
				et = errDeliveryFailed
			}

			_, err = osexec.Command("true").Output() // noop
			_ = err
			mustDB().Exec(`
				INSERT OR IGNORE INTO cron_health_log
					(id, job_id, job_name, run_id, run_at, status, error_type, error_detail, duration_ms, model, delivered, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, newID(), job.ID, job.Name, run.RunID, run.RunAt,
				run.Status, string(et), run.Error,
				run.DurationMs, run.Model, delivered, now)
			logged++
			if run.Status == "error" {
				errors++
			}
		}
	}

	okOut(map[string]interface{}{
		"logged": logged,
		"errors": errors,
		"jobs":   len(jobList),
	})
}

type jobEntry struct {
	ID   string
	Name string
}

func readJobsFromFile() []jobEntry {
	home, _ := os.UserHomeDir()
	path := home + "/.openclaw/cron/jobs.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg struct {
		Jobs []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil
	}
	var out []jobEntry
	for _, j := range cfg.Jobs {
		out = append(out, jobEntry{ID: j.ID, Name: j.Name})
	}
	return out
}

// ── cron-health check ─────────────────────────────────────────────────────────

type jobHealth struct {
	JobName        string  `json:"job_name"`
	Total          int64   `json:"total"`
	Errors         int64   `json:"errors"`
	Undelivered    int64   `json:"undelivered"`
	ErrorRate      float64 `json:"error_rate_pct"`
	ErrorTypes     string  `json:"error_types"`
	LastRun        string  `json:"last_run"`
	AvgDurationMs  float64 `json:"avg_duration_ms"`
	NeedsAttention bool    `json:"needs_attention"`
	SuggestedFix   string  `json:"suggested_fix"`
}

func cronHealthCheck(args []string) {
	rows, _ := queryRows(`
		SELECT job_name,
		       COUNT(*) as total,
		       SUM(CASE WHEN status='error' THEN 1 ELSE 0 END) as errors,
		       SUM(CASE WHEN status='ok' AND delivered=0 THEN 1 ELSE 0 END) as undelivered,
		       GROUP_CONCAT(DISTINCT error_type) as error_types,
		       MAX(run_at) as last_run,
		       AVG(duration_ms) as avg_duration_ms
		FROM cron_health_log
		WHERE run_at >= datetime('now', '-7 days')
		GROUP BY job_name
		ORDER BY errors DESC
	`)

	var jobs []jobHealth
	for _, r := range rows {
		total := toInt64(r["total"])
		errors := toInt64(r["errors"])
		undelivered := toInt64(r["undelivered"])
		errorTypes := toString(r["error_types"])
		rate := 0.0
		if total > 0 {
			rate = float64(errors) / float64(total) * 100
		}
		fix := suggestFix(errorTypes, rate)
		jobs = append(jobs, jobHealth{
			JobName:       toString(r["job_name"]),
			Total:         total,
			Errors:        errors,
			Undelivered:   undelivered,
			ErrorRate:     rate,
			ErrorTypes:    errorTypes,
			LastRun:       toString(r["last_run"]),
			AvgDurationMs: toFloat64(r["avg_duration_ms"]),
			NeedsAttention: rate >= 50 || undelivered > 0,
			SuggestedFix:  fix,
		})
	}

	okOut(map[string]interface{}{
		"period":   "7d",
		"jobs":     jobs,
		"fixable":  countFixable(jobs),
		"jobs_count": len(jobs),
	})
}

func suggestFix(errorTypes string, errorRate float64) string {
	if errorRate == 0 {
		return "healthy"
	}
	fixes := []string{}
	if strings.Contains(errorTypes, string(errTimeout)) {
		fixes = append(fixes, "increase timeoutSeconds (double current)")
	}
	if strings.Contains(errorTypes, string(errSessionConflict)) {
		fixes = append(fixes, "offset schedule by +3min to avoid concurrent jobs")
	}
	if strings.Contains(errorTypes, string(errModelUnavail)) {
		fixes = append(fixes, "switch primary model to deepseek/deepseek-chat")
	}
	if strings.Contains(errorTypes, string(errDeliveryFailed)) {
		fixes = append(fixes, "check Telegram bot token and chat_id")
	}
	if strings.Contains(errorTypes, string(errReasoningLeak)) {
		fixes = append(fixes, "rewrite cron payload to remove conditional reasoning; use shell script instead")
	}
	if len(fixes) == 0 {
		return "manual review needed"
	}
	return strings.Join(fixes, "; ")
}

func countFixable(jobs []jobHealth) int {
	n := 0
	for _, j := range jobs {
		if j.SuggestedFix != "healthy" && j.SuggestedFix != "manual review needed" {
			n++
		}
	}
	return n
}

// ── cron-health autofix ───────────────────────────────────────────────────────

func cronHealthAutofix(args []string) {
	// Đọc jobs có lỗi gần nhất
	rows, _ := queryRows(`
		SELECT job_id, job_name, error_type, error_detail, duration_ms,
		       COUNT(*) as occurrences
		FROM cron_health_log
		WHERE status = 'error'
		  AND fix_applied IS NULL
		  AND run_at >= datetime('now', '-7 days')
		GROUP BY job_id, error_type
		HAVING occurrences >= 1
		ORDER BY occurrences DESC
	`)

	applied := []map[string]interface{}{}
	skipped := []map[string]interface{}{}

	home, _ := os.UserHomeDir()
	jobsPath := home + "/.openclaw/cron/jobs.json"
	rawJobs, err := os.ReadFile(jobsPath)
	if err != nil {
		errOut("cannot read cron/jobs.json: " + err.Error())
		return
	}
	var jobsCfg map[string]interface{}
	json.Unmarshal(rawJobs, &jobsCfg)
	jobs := jobsCfg["jobs"].([]interface{})

	changed := false

	for _, r := range rows {
		jobID := toString(r["job_id"])
		jobName := toString(r["job_name"])
		et := errorType(toString(r["error_type"]))
		durationMs := toInt64(r["duration_ms"])

		fix := applyFix(jobs, jobID, et, durationMs)
		if fix != "" {
			changed = true
			applied = append(applied, map[string]interface{}{
				"job": jobName, "error_type": et, "fix": fix,
			})
			// Mark as fixed in DB
			mustDB().Exec(`
				UPDATE cron_health_log SET fix_applied = ?, fix_at = ?
				WHERE job_id = ? AND error_type = ? AND fix_applied IS NULL
			`, fix, vnNowISO(), jobID, string(et))
		} else {
			skipped = append(skipped, map[string]interface{}{
				"job": jobName, "error_type": et,
				"reason": "no automatic fix available — manual review needed",
			})
		}
	}

	if changed {
		out, _ := json.MarshalIndent(jobsCfg, "", "  ")
		os.WriteFile(jobsPath, out, 0o600)
	}

	okOut(map[string]interface{}{
		"applied": applied,
		"skipped": skipped,
		"jobs_file_updated": changed,
	})
}

func applyFix(jobs []interface{}, jobID string, et errorType, durationMs int64) string {
	for _, j := range jobs {
		job, ok := j.(map[string]interface{})
		if !ok {
			continue
		}
		if toString(job["id"]) != jobID {
			continue
		}

		payload, _ := job["payload"].(map[string]interface{})
		if payload == nil {
			continue
		}

		switch et {
		case errTimeout:
			current := int64(0)
			if v, ok := payload["timeoutSeconds"]; ok {
				current = toInt64(v)
			}
			if current == 0 {
				current = durationMs / 1000
			}
			newTimeout := current * 2
			if newTimeout > 600 {
				newTimeout = 600
			}
			payload["timeoutSeconds"] = newTimeout
			return fmt.Sprintf("timeoutSeconds: %d → %d", current, newTimeout)

		case errSessionConflict:
			sched, _ := job["schedule"].(map[string]interface{})
			if sched == nil {
				continue
			}
			expr := toString(sched["expr"])
			newExpr := offsetCronByMinutes(expr, 3)
			if newExpr != expr {
				sched["expr"] = newExpr
				return fmt.Sprintf("schedule: '%s' → '%s' (+3min offset)", expr, newExpr)
			}

		case errModelUnavail:
			// Đổi model ở openclaw.json — nằm ngoài jobs.json
			return applyModelFix(jobID)
		}
	}
	return ""
}

// offsetCronByMinutes thêm N phút vào cron expression (chỉ hỗ trợ simple format)
func offsetCronByMinutes(expr string, offset int) string {
	parts := strings.Fields(expr)
	if len(parts) < 5 {
		return expr
	}
	minStr := parts[0]
	var min int
	if _, err := fmt.Sscanf(minStr, "%d", &min); err != nil {
		return expr // complex expression, skip
	}
	newMin := (min + offset) % 60
	parts[0] = fmt.Sprintf("%d", newMin)
	return strings.Join(parts, " ")
}

func applyModelFix(jobID string) string {
	// Đọc openclaw.json, tìm agent của job này, đổi primary model
	home, _ := os.UserHomeDir()
	path := home + "/.openclaw/openclaw.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return ""
	}

	agents, _ := cfg["agents"].(map[string]interface{})
	list, _ := agents["list"].([]interface{})
	for _, a := range list {
		agent, _ := a.(map[string]interface{})
		model, _ := agent["model"].(map[string]interface{})
		if model == nil {
			continue
		}
		primary := toString(model["primary"])
		// Nếu đang dùng free tier → đổi sang deepseek
		if strings.Contains(primary, ":free") || strings.Contains(primary, "nemotron") {
			model["primary"] = "deepseek/deepseek-chat"
			out, _ := json.MarshalIndent(cfg, "", "  ")
			os.WriteFile(path, out, 0o600)
			return fmt.Sprintf("primary model: %s → deepseek/deepseek-chat", primary)
		}
	}
	return ""
}

// ── cron-health list ──────────────────────────────────────────────────────────

func cronHealthList(args []string) {
	limit := 20
	jobName := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--limit":
			i++
			fmt.Sscanf(args[i], "%d", &limit)
		case "--job":
			i++
			jobName = args[i]
		}
	}

	q := `SELECT job_name, run_at, status, error_type, duration_ms, model, delivered, fix_applied
	      FROM cron_health_log`
	var qargs []interface{}
	if jobName != "" {
		q += ` WHERE job_name = ?`
		qargs = append(qargs, jobName)
	}
	q += ` ORDER BY run_at DESC LIMIT ?`
	qargs = append(qargs, limit)

	rows, _ := queryRows(q, qargs...)
	okOut(map[string]interface{}{
		"logs":  rows,
		"count": len(rows),
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toInt64(v interface{}) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case float64:
		return int64(x)
	case int:
		return int64(x)
	}
	return 0
}

func toFloat64(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	}
	return 0
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
