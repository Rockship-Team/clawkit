package main

import (
	"fmt"
	"strconv"
	"time"
)

// cmdGoal dispatches sme-goal — Goal persistence (Phase 3A) + reliable
// progress measurement (Phase 3B). The single dedicated `goals` table
// remains from 3A (weekly_kpis only supports 5 fixed weekly metrics, wrong
// shape for an arbitrary goal like "5 qualified leads this month").
//
// Phase 3B adds exactly one new column: baseline_value. COSMO has NO
// reliable, timestamped record of when a contact's business_stage changed
// (audited directly against cosmo-backend source: no stage-transition
// timestamp field, updated_at is a blanket any-field-touch marker shared
// with ~15 unrelated fields, no audit/history table, no GORM hooks, and
// contact-update never touches the interactions table). So "progress"
// cannot be derived from a time-windowed count — the only reliable
// strategy left is: snapshot the current count at goal-creation time
// (baseline_value), and report progress = current - baseline. This is
// Strategy B from the Phase 3B brief, chosen because Strategy A (real
// stage-transition history) does not exist and was confirmed not to exist,
// not assumed.
//
//	sme-cli goal set --text "..." [--metric qualified_leads|proposals|contracts|custom] [--target N] [--deadline YYYY-MM-DD]
//	sme-cli goal list [--status active|completed|cancelled]
//	sme-cli goal view <id>
//	sme-cli goal check <id>          # alias of view — same output, same discipline
//	sme-cli goal complete <id>
//	sme-cli goal cancel <id>
//	sme-cli goal next-action <id>    # Phase 3B — goal-aware NBA, see goal_nba.go
func cmdGoal(args []string) {
	if len(args) == 0 {
		errOut("usage: goal set|list|view|check|complete|cancel|next-action")
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
	case "next-action":
		goalNextAction(args[1:])
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
	// Additive-only (Phase 3B): nullable, no backfill, no default that could
	// fabricate a baseline for goals created before this column existed —
	// those simply have baseline_value NULL and fall back to "unknown"
	// progress, same as any metric_type without a reliable source.
	ensureColumn("goals", "baseline_value", "INTEGER")
}

// goalMetricStages: metric types with a reliable, already-built progress
// source (COSMO business_stage via the exact same fetchAllContacts
// primitive sme-opportunity/sme-analytics already use). Any other
// metric_type (including "custom") has NO source here and MUST report
// "unknown" — never a guessed/derived number.
//
// Phase 3B hardening: business_stage is a single mutable column, and
// crm/SKILL.md documents its lifecycle as
// NEW → ENGAGED → QUALIFIED → PROPOSAL → NEGOTIATION → WON/LOST
// (confirmed against cosmo-backend: this scheme is GTM's own convention —
// COSMO's original domain enum, PRE_SALES/SALES/POST_SALES, is unused by
// this pipeline). Counting only the EXACT stage (e.g. == "QUALIFIED") is
// wrong for a milestone goal like "5 qualified leads this month": once a
// newly-qualified contact progresses QUALIFIED → PROPOSAL, they leave the
// exact-match set and progress falls even though they're further along,
// not less qualified (opportunity.go already treats this the same way —
// "business_stage đã là PROPOSAL/WON → readiness = ready", i.e. later
// stages already imply the earlier milestone). So each metric here counts
// its stage CUMULATIVELY ("at-or-beyond") — the one shared definition used
// for both the baseline snapshot and the current count, so they can never
// drift apart.
var goalMetricStages = map[string][]string{
	"qualified_leads": {"QUALIFIED", "PROPOSAL", "NEGOTIATION", "WON"},
	"proposals":       {"PROPOSAL", "NEGOTIATION", "WON"},
	"contracts":       {"WON"},
}

const goalMaxPages = 8 // on-demand only, same default as opportunity/analytics — never in a hot/cron path

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

	// Capture baseline NOW, at goal-creation time — the only moment we can
	// reliably say "this many already existed before the goal started".
	// nil (SQL NULL) for metric types without a reliable source, so
	// goalProgress() can tell "no baseline captured" apart from "baseline
	// was legitimately 0".
	var baseline interface{}
	baselineNote := ""
	if stages, ok := goalMetricStages[metric]; ok {
		count, err := countContactsByBusinessStages(stages, goalMaxPages)
		if err == nil {
			baseline = count
			baselineNote = fmt.Sprintf(" (baseline hiện tại: %d %s có sẵn trước khi tạo goal — tính theo business_stage ∈ %v, không tính vào progress)", count, metric, stages)
		}
	}

	orgID := defaultOrgID()
	now := vnNowISO()
	id := newID()
	_, err := mustDB().Exec(`
		INSERT INTO goals (id, org_id, goal_text, metric_type, target_value, deadline, status, created_at, updated_at, baseline_value)
		VALUES (?, ?, ?, ?, ?, ?, 'active', ?, ?, ?)
	`, id, orgID, text, metric, target, nullableString(deadline), now, now, baseline)
	if err != nil {
		errOut(err.Error())
		return
	}
	okOut(map[string]interface{}{
		"id": id, "goal_text": text, "metric_type": metric, "target_value": target,
		"deadline": deadline, "status": "active",
		"message": "Đã lưu goal — sẽ nhớ lại kể cả sau khi hết phiên chat này." + baselineNote,
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

// goalRecord is the persisted row plus everything goalProgress derives —
// shared by goalView and goal_nba.go so the two never disagree on numbers.
type goalRecord struct {
	ID            string
	GoalText      string
	MetricType    string
	TargetValue   int64
	Deadline      string
	Status        string
	CreatedAt     string
	HasBaseline   bool
	BaselineValue int
}

func loadGoal(id string) (*goalRecord, error) {
	orgID := defaultOrgID()
	row, err := queryOne(`SELECT id, goal_text, metric_type, target_value, deadline, status, created_at, baseline_value FROM goals WHERE org_id = ? AND id = ?`, orgID, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	g := &goalRecord{
		ID:          toString(row["id"]),
		GoalText:    toString(row["goal_text"]),
		MetricType:  toString(row["metric_type"]),
		TargetValue: toInt64(row["target_value"]),
		Deadline:    toString(row["deadline"]),
		Status:      toString(row["status"]),
		CreatedAt:   toString(row["created_at"]),
	}
	if row["baseline_value"] != nil {
		g.HasBaseline = true
		g.BaselineValue = int(toInt64(row["baseline_value"]))
	}
	return g, nil
}

// goalProgress is THE single place that computes progress for a goal —
// used by both `goal view/check` and `goal next-action` so they can never
// report different numbers for the same goal. Returns ok=false (progress
// unmeasurable) whenever metric_type has no reliable source OR no baseline
// was captured (e.g. a metric added to goalMetricStages after this goal was
// created) — never guesses.
type goalProgressResult struct {
	OK                bool
	CurrentValue      int
	Progress          int
	Remaining         int
	MeasurementMethod string
	ScopeStatus       string
}

// calculateGoalProgress is the PURE arithmetic core — progress = current -
// baseline (clamped >= 0, since a contact can also leave a stage, e.g. move
// to LOST), remaining = target - progress (clamped >= 0). Extracted so
// Test 1/2 from the Phase 3B brief (baseline correctness) exercise the
// exact same code the CLI runs, not a re-typed copy in a test file.
func calculateGoalProgress(current, baseline int, target int64) (progress, remaining int) {
	progress = current - baseline
	if progress < 0 {
		progress = 0
	}
	remaining = int(target) - progress
	if remaining < 0 {
		remaining = 0
	}
	return progress, remaining
}

func computeGoalProgress(g *goalRecord) goalProgressResult {
	stages, hasSource := goalMetricStages[g.MetricType]
	if !hasSource || !g.HasBaseline {
		reason := "chưa có nguồn dữ liệu tin cậy cho metric này"
		if hasSource && !g.HasBaseline {
			reason = "goal này được tạo trước khi có baseline tracking, hoặc lấy baseline lúc tạo thất bại"
		}
		return goalProgressResult{OK: false, MeasurementMethod: "unknown — " + reason}
	}
	current, err := countContactsByBusinessStages(stages, goalMaxPages)
	if err != nil {
		return goalProgressResult{OK: false, MeasurementMethod: "unknown — không lấy được dữ liệu từ COSMO lúc này (" + err.Error() + ")"}
	}
	progress, remaining := calculateGoalProgress(current, g.BaselineValue, g.TargetValue)
	return goalProgressResult{
		OK:           true,
		CurrentValue: current,
		Progress:     progress,
		Remaining:    remaining,
		// Honest wording (Phase 3B hardening, Part 5): this is cumulative
		// "at-or-beyond" counting, and COSMO has NO stage-transition history
		// (audited directly — confirmed in Phase 3B), so a contact that was
		// QUALIFIED/PROPOSAL and later moved to LOST silently drops out of
		// this count with no trace it ever counted. The honest claim is
		// "net new contacts currently at-or-beyond <stage> since goal
		// creation" — NEVER "total leads ever qualified", which this
		// mechanism cannot support.
		MeasurementMethod: fmt.Sprintf(
			"COSMO business_stage ∈ %v (cumulative \"at-or-beyond\", không phải chỉ đúng 1 stage): current=%d, baseline (lúc tạo goal, cùng stage-set)=%d, progress=current-baseline=%d. "+
				"Đây là \"net new hiện đang ở mức %s trở lên kể từ lúc tạo goal\" — KHÔNG phải \"tổng số từng đạt %s\": COSMO không lưu lịch sử chuyển stage, nên 1 contact rớt xuống LOST sau khi đã qua mức này sẽ không còn được tính, không để lại dấu vết.",
			stages, current, g.BaselineValue, progress, g.MetricType, g.MetricType),
		ScopeStatus: "unverified — COSMO không có field gán contact vào 1 offering/segment cụ thể (vd \"AI Automation\"), nên progress chỉ đếm theo business_stage tổng, KHÔNG xác nhận được có đúng scope goal_text mô tả hay không",
	}
}

func goalView(args []string) {
	if len(args) == 0 {
		errOut("usage: goal view <id>")
		return
	}
	g, err := loadGoal(args[0])
	if err != nil {
		errOut(err.Error())
		return
	}
	if g == nil {
		errOut("không tìm thấy goal id " + args[0])
		return
	}

	progress := computeGoalProgress(g)
	out := map[string]interface{}{
		"id": g.ID, "goal_text": g.GoalText, "metric_type": g.MetricType,
		"target_value": g.TargetValue, "deadline": g.Deadline, "status": g.Status,
		"created_at": g.CreatedAt, "measurement_method": progress.MeasurementMethod,
	}
	if progress.OK {
		out["current_value"] = progress.CurrentValue
		out["baseline_value"] = g.BaselineValue
		out["progress"] = progress.Progress
		out["remaining"] = progress.Remaining
		out["scope_status"] = progress.ScopeStatus
		if progress.Progress >= int(g.TargetValue) && g.Status == "active" {
			out["recommended_status"] = "completed — progress đã đạt/vượt target, nhưng KHÔNG tự động complete. Gọi `goal complete " + g.ID + "` nếu anh xác nhận."
		}
	} else {
		out["current_value"] = "unknown"
		out["progress"] = "unknown"
		out["remaining"] = "unknown"
	}
	if g.Deadline != "" {
		if dl, err := time.Parse("2006-01-02", g.Deadline); err == nil {
			days := int(time.Until(dl).Hours() / 24)
			out["days_remaining"] = days
		}
	}
	okOut(out)
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

// countContactsByBusinessStages reuses the exact same fetchAllContacts scan
// sme-analytics/sme-opportunity already rely on — never a second COSMO
// pagination implementation. On-demand only (not in any hot/cron path).
// Takes a stage SET (not a single stage) — this is the one shared
// definition goalSet (baseline) and computeGoalProgress (current) both
// call, so baseline and current can never use a different stage-set by
// accident (Phase 3B hardening, Part 3).
func countContactsByBusinessStages(stages []string, maxPages int) (int, error) {
	contacts, _, err := fetchAllContacts(maxPages)
	if err != nil {
		return 0, err
	}
	match := make(map[string]bool, len(stages))
	for _, s := range stages {
		match[s] = true
	}
	count := 0
	for _, c := range contacts {
		if match[c.BusinessStage] {
			count++
		}
	}
	return count, nil
}
