package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// cmdKPI dispatches weekly KPI subcommands.
//
//	sme-cli kpi set --contracts N [--proposals N] [--meetings N] [--contacts N] [--member NAME] [--week YYYY-Www]
//	sme-cli kpi get [--member NAME] [--week YYYY-Www]
//	sme-cli kpi list [--limit N]
//	sme-cli kpi check [--member NAME] [--week YYYY-Www]
//	sme-cli kpi team [--week YYYY-Www]        # tất cả BD members
//	sme-cli kpi actual [--member NAME] [--week YYYY-Www]  # từ COSMO
func cmdKPI(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: kpi set|get|list|check|team|actual")
		os.Exit(1)
	}
	ensureKPITable()
	switch args[0] {
	case "set":
		kpiSet(args[1:])
	case "get":
		kpiGet(args[1:])
	case "list":
		kpiList(args[1:])
	case "check":
		kpiCheck(args[1:])
	case "team":
		kpiTeam(args[1:])
	case "actual":
		kpiActual(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "usage: kpi set|get|list|check\n")
		os.Exit(1)
	}
}

func ensureKPITable() {
	mustDB().Exec(`CREATE TABLE IF NOT EXISTS weekly_kpis (
		id               TEXT PRIMARY KEY,
		org_id           TEXT NOT NULL DEFAULT 'default',
		week_label       TEXT NOT NULL,
		week_start       TEXT NOT NULL,
		week_end         TEXT NOT NULL,
		member           TEXT NOT NULL DEFAULT 'team',
		contracts_target INTEGER NOT NULL DEFAULT 0,
		proposals_target INTEGER NOT NULL DEFAULT 0,
		meetings_target  INTEGER NOT NULL DEFAULT 0,
		contacts_target  INTEGER NOT NULL DEFAULT 0,
		revenue_target   INTEGER NOT NULL DEFAULT 0,
		set_by           TEXT,
		notes            TEXT,
		created_at       TEXT NOT NULL,
		updated_at       TEXT NOT NULL,
		UNIQUE(org_id, week_label, member)
	)`)
	// Migrate: thêm cột mới nếu chưa có
	mustDB().Exec(`ALTER TABLE weekly_kpis ADD COLUMN member TEXT NOT NULL DEFAULT 'team'`)
	mustDB().Exec(`ALTER TABLE weekly_kpis ADD COLUMN proposals_target INTEGER NOT NULL DEFAULT 0`)
	mustDB().Exec(`ALTER TABLE weekly_kpis ADD COLUMN meetings_target INTEGER NOT NULL DEFAULT 0`)
	mustDB().Exec(`ALTER TABLE weekly_kpis ADD COLUMN contacts_target INTEGER NOT NULL DEFAULT 0`)
}

// isoWeekLabel returns "YYYY-Www" for a given time.
func isoWeekLabel(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// weekBounds returns Monday and Sunday (VN locale) for an ISO week label.
func weekBounds(label string) (start, end string, err error) {
	var year, week int
	_, err = fmt.Sscanf(label, "%d-W%d", &year, &week)
	if err != nil {
		return
	}
	// Jan 4 is always in week 1 per ISO 8601
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.UTC)
	// Monday of week 1
	weekOneMonday := jan4.AddDate(0, 0, -int(jan4.Weekday()-time.Monday))
	monday := weekOneMonday.AddDate(0, 0, (week-1)*7)
	sunday := monday.AddDate(0, 0, 6)
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	start = monday.In(loc).Format("2006-01-02")
	end = sunday.In(loc).Format("2006-01-02")
	return
}

func kpiSet(args []string) {
	week := isoWeekLabel(vnNow())
	member := "team"
	var contracts, proposals, meetings, contacts, revenue int64
	var setBy, notes string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--week":
			i++; week = args[i]
		case "--member":
			i++; member = args[i]
		case "--contracts":
			i++; contracts, _ = strconv.ParseInt(args[i], 10, 64)
		case "--proposals":
			i++; proposals, _ = strconv.ParseInt(args[i], 10, 64)
		case "--meetings":
			i++; meetings, _ = strconv.ParseInt(args[i], 10, 64)
		case "--contacts":
			i++; contacts, _ = strconv.ParseInt(args[i], 10, 64)
		case "--revenue":
			i++; revenue = parseVND(args[i])
		case "--set-by":
			i++; setBy = args[i]
		case "--notes":
			i++; notes = args[i]
		}
	}

	if contracts == 0 && proposals == 0 && meetings == 0 && contacts == 0 && revenue == 0 {
		errOut("cần ít nhất 1 target: --contracts, --proposals, --meetings, --contacts, --revenue")
		return
	}

	start, end, err := weekBounds(week)
	if err != nil {
		errOut("week format không hợp lệ, dùng YYYY-Www (vd: 2026-W23)")
		return
	}

	now := vnNowISO()
	_, err = mustDB().Exec(`
		INSERT INTO weekly_kpis
			(id, org_id, week_label, week_start, week_end, member,
			 contracts_target, proposals_target, meetings_target, contacts_target,
			 revenue_target, set_by, notes, created_at, updated_at)
		VALUES (?, 'default', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(org_id, week_label, member) DO UPDATE SET
			contracts_target = excluded.contracts_target,
			proposals_target = excluded.proposals_target,
			meetings_target  = excluded.meetings_target,
			contacts_target  = excluded.contacts_target,
			revenue_target   = excluded.revenue_target,
			set_by           = excluded.set_by,
			notes            = excluded.notes,
			updated_at       = excluded.updated_at
	`, newID(), week, start, end, member,
		contracts, proposals, meetings, contacts,
		revenue, setBy, notes, now, now)
	if err != nil {
		errOut(err.Error())
		return
	}

	summary := fmt.Sprintf("KPI tuần %s [%s] đã lưu", week, member)
	if contracts > 0 { summary += fmt.Sprintf(": %d contract", contracts) }
	if proposals > 0 { summary += fmt.Sprintf(", %d proposal", proposals) }
	if meetings > 0  { summary += fmt.Sprintf(", %d meeting", meetings) }
	if contacts > 0  { summary += fmt.Sprintf(", %d contact mới", contacts) }

	okOut(map[string]interface{}{
		"week": week, "member": member,
		"week_start": start, "week_end": end,
		"contracts_target": contracts, "proposals_target": proposals,
		"meetings_target": meetings, "contacts_target": contacts,
		"revenue_target": revenue, "set_by": setBy,
		"message": summary,
	})
}

func kpiGet(args []string) {
	week := isoWeekLabel(vnNow())
	for i := 0; i < len(args); i++ {
		if args[i] == "--week" && i+1 < len(args) {
			i++
			week = args[i]
		}
	}

	row, err := queryOne(`
		SELECT week_label, week_start, week_end,
		       contracts_target, revenue_target, set_by, notes, created_at, updated_at
		FROM weekly_kpis
		WHERE org_id = 'default' AND week_label = ?
	`, week)
	if err != nil {
		errOut(err.Error())
		return
	}
	if row == nil {
		jsonOut(map[string]interface{}{
			"ok":      true,
			"week":    week,
			"kpi_set": false,
			"message": fmt.Sprintf("Chưa có KPI cho tuần %s", week),
		})
		return
	}
	row["ok"] = true
	row["kpi_set"] = true
	jsonOut(row)
}

func kpiList(args []string) {
	limit := 8
	for i := 0; i < len(args); i++ {
		if args[i] == "--limit" && i+1 < len(args) {
			i++
			fmt.Sscanf(args[i], "%d", &limit)
		}
	}
	rows, err := queryRows(`
		SELECT week_label, week_start, week_end,
		       contracts_target, revenue_target, set_by, created_at
		FROM weekly_kpis
		WHERE org_id = 'default'
		ORDER BY week_label DESC
		LIMIT ?
	`, limit)
	if err != nil {
		errOut(err.Error())
		return
	}
	jsonOut(map[string]interface{}{
		"ok":    true,
		"weeks": rows,
		"count": len(rows),
	})
}

// kpiCheck is designed for cron jobs. Returns KPI status + needs_reminder flag.
func kpiCheck(args []string) {
	week := isoWeekLabel(vnNow())
	member := "team"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--week":
			i++; week = args[i]
		case "--member":
			i++; member = args[i]
		}
	}

	start, end, _ := weekBounds(week)
	now := vnNow()
	weekday := now.Weekday()
	daysLeft := int(time.Sunday-weekday+7) % 7

	row, _ := queryOne(`
		SELECT week_label, week_start, week_end, member,
		       contracts_target, proposals_target, meetings_target,
		       contacts_target, revenue_target, set_by, notes, created_at
		FROM weekly_kpis
		WHERE org_id = 'default' AND week_label = ? AND member = ?
	`, week, member)

	if row == nil {
		jsonOut(map[string]interface{}{
			"ok": true, "week": week, "member": member,
			"week_start": start, "week_end": end,
			"kpi_set": false, "needs_reminder": true, "days_left": daysLeft,
			"message": fmt.Sprintf("Chưa có KPI tuần %s [%s] — còn %d ngày", week, member, daysLeft),
		})
		return
	}

	row["ok"] = true
	row["kpi_set"] = true
	row["needs_reminder"] = false
	row["days_left"] = daysLeft
	row["week_start"] = start
	row["week_end"] = end
	jsonOut(row)
}

// kpiTeam — tổng hợp KPI của toàn team BD trong tuần
func kpiTeam(args []string) {
	week := isoWeekLabel(vnNow())
	for i := 0; i < len(args); i++ {
		if args[i] == "--week" && i+1 < len(args) {
			i++; week = args[i]
		}
	}
	start, end, _ := weekBounds(week)

	rows, err := queryRows(`
		SELECT member, contracts_target, proposals_target, meetings_target,
		       contacts_target, revenue_target, set_by, updated_at
		FROM weekly_kpis
		WHERE org_id = 'default' AND week_label = ?
		ORDER BY member
	`, week)
	if err != nil {
		errOut(err.Error())
		return
	}

	now := vnNow()
	daysLeft := int(time.Sunday-now.Weekday()+7) % 7

	okOut(map[string]interface{}{
		"week": week, "week_start": start, "week_end": end,
		"days_left": daysLeft,
		"members": rows,
		"count": len(rows),
		"message": fmt.Sprintf("KPI tuần %s: %d thành viên đã đặt target", week, len(rows)),
	})
}

// kpiActual — lấy actual từ COSMO interactions trong tuần
func kpiActual(args []string) {
	week := isoWeekLabel(vnNow())
	member := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--week":
			i++; week = args[i]
		case "--member":
			i++; member = args[i]
		}
	}
	start, end, _ := weekBounds(week)

	// Query local DB cho interactions trong tuần
	// (COSMO interactions được log qua sme-cli cosmo log-interaction)
	q := `
		SELECT
			COALESCE(json_extract(raw_data, '$.assigned_to'), 'unknown') as member,
			SUM(CASE WHEN interaction_type = 'proposal_sent' THEN 1 ELSE 0 END) as proposals,
			SUM(CASE WHEN interaction_type IN ('meeting', 'call') THEN 1 ELSE 0 END) as meetings,
			SUM(CASE WHEN interaction_type = 'contact_created' THEN 1 ELSE 0 END) as contacts_added,
			SUM(CASE WHEN interaction_type = 'stage_changed'
			         AND json_extract(raw_data,'$.to_stage') = 'WON' THEN 1 ELSE 0 END) as contracts_won
		FROM interactions
		WHERE created_at >= ? AND created_at <= ?
	`
	qargs := []interface{}{start, end + "T23:59:59"}
	if member != "" {
		q += ` AND json_extract(raw_data, '$.assigned_to') = ?`
		qargs = append(qargs, member)
	}
	q += ` GROUP BY member ORDER BY proposals DESC`

	rows, err := queryRows(q, qargs...)
	if err != nil {
		// Fallback: table không tồn tại hoặc không có data
		jsonOut(map[string]interface{}{
			"ok": true, "week": week, "member": member,
			"week_start": start, "week_end": end,
			"actual": []interface{}{},
			"note": "Chưa có interaction data trong local DB. Dùng sme-cli cosmo search-interactions để xem từ COSMO.",
		})
		return
	}

	okOut(map[string]interface{}{
		"week": week, "week_start": start, "week_end": end,
		"member": member, "actual": rows,
	})
}
