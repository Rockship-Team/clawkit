package main

import (
	"fmt"
	"strconv"
	"time"
)

// cmdGoal dispatches sme-goal — minimal Goal persistence (Phase 3A). This
// is the ONLY new state this phase adds: a small dedicated table, because
// weekly_kpis only supports 5 fixed metric types on a weekly scope and
// can't hold an arbitrary goal like "5 qualified leads this month" (audited
// during the Phase 3 pre-flight — this was the single confirmed real gap
// blocking "does the agent still know the goal tomorrow").
//
// Deliberately NOT built here (Phase 3B/3C): Goal-aware NBA, delta-driven
// recommendation chaining, memory-aware goal reasoning. `goal check` only
// loads the persisted definition and — ONLY for metric types with an
// already-reliable source — shows current progress. Everything else stays
// "unknown" rather than guessed.
//
//	sme-cli goal set --text "..." [--metric qualified_leads|proposals|contracts|custom] [--target N] [--deadline YYYY-MM-DD]
//	sme-cli goal list [--status active|completed|cancelled]
//	sme-cli goal view <id>
//	sme-cli goal check <id>      # alias of view — same output, same discipline
//	sme-cli goal complete <id>
//	sme-cli goal cancel <id>
func cmdGoal(args []string) {
	if len(args) == 0 {
		errOut("usage: goal set|list|view|check|complete|cancel")
		return
	}
	ensureGoalsTable()
	switch args[0] {
	case "set":
		goalSet(args[1:])
	case "list":
		goalList(args[1:])
	case "view", "check":
		goalView(args[1:])
	case "complete":
		goalSetStatus(args[1:], "completed")
	case "cancel":
		goalSetStatus(args[1:], "cancelled")
	default:
		errOut("unknown goal command: " + args[0])
	}
}

func ensureGoalsTable() {
	mustDB().Exec(`CREATE TABLE IF NOT EXISTS goals (
		id           TEXT PRIMARY KEY,
		org_id       TEXT NOT NULL DEFAULT 'default',
		goal_text    TEXT NOT NULL,
		metric_type  TEXT NOT NULL DEFAULT 'custom',
		target_value INTEGER NOT NULL DEFAULT 0,
		deadline     TEXT,
		status       TEXT NOT NULL DEFAULT 'active',
		created_at   TEXT NOT NULL,
		updated_at   TEXT NOT NULL
	)`)
}

// goalMetricTypes with a reliable, already-built progress source (COSMO
// business_stage via the exact same fetchAllContacts primitive
// sme-opportunity/sme-analytics already use — never a second data path).
// Any other metric_type (including "custom") has NO source here and MUST
// report "unknown" — never a guessed/derived number.
var goalMetricStage = map[string]string{
	"qualified_leads": "QUALIFIED",
	"proposals":       "PROPOSAL",
	"contracts":       "WON",
}

func goalSet(args []string) {
	var text, metric, deadline string
	var target int64
	metric = "custom"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--text":
			i++
			text = args[i]
		case "--metric":
			i++
			metric = args[i]
		case "--target":
			i++
			target, _ = strconv.ParseInt(args[i], 10, 64)
		case "--deadline":
			i++
			deadline = args[i]
		}
	}
	if text == "" {
		errOut("usage: goal set --text \"...\" [--metric qualified_leads|proposals|contracts|custom] [--target N] [--deadline YYYY-MM-DD]")
		return
	}
	if deadline != "" {
		if _, err := time.Parse("2006-01-02", deadline); err != nil {
			errOut("--deadline phải đúng format YYYY-MM-DD, vd 2026-08-31")
			return
		}
	}

	orgID := defaultOrgID()
	now := vnNowISO()
	id := newID()
	_, err := mustDB().Exec(`
		INSERT INTO goals (id, org_id, goal_text, metric_type, target_value, deadline, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'active', ?, ?)
	`, id, orgID, text, metric, target, nullableString(deadline), now, now)
	if err != nil {
		errOut(err.Error())
		return
	}
	okOut(map[string]interface{}{
		"id": id, "goal_text": text, "metric_type": metric, "target_value": target,
		"deadline": deadline, "status": "active",
		"message": "Đã lưu goal — sẽ nhớ lại kể cả sau khi hết phiên chat này.",
	})
}

func goalList(args []string) {
	status := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--status" && i+1 < len(args) {
			i++
			status = args[i]
		}
	}
	orgID := defaultOrgID()
	q := `SELECT id, goal_text, metric_type, target_value, deadline, status, created_at FROM goals WHERE org_id = ?`
	qargs := []interface{}{orgID}
	if status != "" {
		q += ` AND status = ?`
		qargs = append(qargs, status)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := queryRows(q, qargs...)
	if err != nil {
		errOut(err.Error())
		return
	}
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	okOut(map[string]interface{}{"goals": rows, "count": len(rows)})
}

func goalView(args []string) {
	if len(args) == 0 {
		errOut("usage: goal view <id>")
		return
	}
	id := args[0]
	orgID := defaultOrgID()
	row, err := queryOne(`SELECT id, goal_text, metric_type, target_value, deadline, status, created_at, updated_at FROM goals WHERE org_id = ? AND id = ?`, orgID, id)
	if err != nil {
		errOut(err.Error())
		return
	}
	if row == nil {
		errOut("không tìm thấy goal id " + id)
		return
	}

	metricType := toString(row["metric_type"])
	currentValue := "unknown"
	progressSource := "unknown — chưa có nguồn dữ liệu tin cậy cho metric này (chỉ hỗ trợ qualified_leads/proposals/contracts qua COSMO business_stage)"
	if stage, ok := goalMetricStage[metricType]; ok {
		count, cErr := countContactsByBusinessStage(stage, 8)
		if cErr == nil {
			currentValue = fmt.Sprint(count)
			progressSource = "COSMO business_stage=" + stage + " (đếm trực tiếp, không suy diễn)"
		} else {
			progressSource = "unknown — không lấy được dữ liệu từ COSMO lúc này (" + cErr.Error() + ")"
		}
	}

	row["current_value"] = currentValue
	row["progress_source"] = progressSource
	okOut(row)
}

func goalSetStatus(args []string, status string) {
	if len(args) == 0 {
		errOut("usage: goal " + status + " <id>")
		return
	}
	id := args[0]
	orgID := defaultOrgID()
	res, err := mustDB().Exec(`UPDATE goals SET status = ?, updated_at = ? WHERE org_id = ? AND id = ?`,
		status, vnNowISO(), orgID, id)
	if err != nil {
		errOut(err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		errOut("không tìm thấy goal id " + id)
		return
	}
	okOut(map[string]interface{}{"id": id, "status": status})
}

// countContactsByBusinessStage reuses the exact same fetchAllContacts scan
// sme-analytics/sme-opportunity already rely on — never a second COSMO
// pagination implementation. On-demand only (not in any hot/cron path).
func countContactsByBusinessStage(stage string, maxPages int) (int, error) {
	contacts, _, err := fetchAllContacts(maxPages)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, c := range contacts {
		if c.BusinessStage == stage {
			count++
		}
	}
	return count, nil
}
