package main

import (
	"fmt"
	"math"
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
//
// Outreach KPI (LinkedIn activity, đặt 1 lần dùng mãi — không phải kiểu weekly_kpis ở trên):
//
//	sme-cli kpi outreach-target set --count N [--set-by TEN]   # N người/ngày, tuần = N*5 (T2-T6)
//	sme-cli kpi outreach-target get
//	sme-cli kpi outreach-status [--date YYYY-MM-DD]            # tình hình hôm nay + tuần này (ad-hoc, không dùng cho cron)
//	sme-cli kpi outreach-report --slot morning|evening|weekend [--date YYYY-MM-DD]  # 3 cron slot thật (xem outreachReport)
func cmdKPI(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: kpi set|get|list|check|team|actual|outreach-target|outreach-status|outreach-report")
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
	case "outreach-target":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: kpi outreach-target set|get")
			os.Exit(1)
		}
		switch args[1] {
		case "set":
			outreachTargetSet(args[2:])
		case "get":
			outreachTargetGet(args[2:])
		default:
			fmt.Fprintln(os.Stderr, "usage: kpi outreach-target set|get")
			os.Exit(1)
		}
	case "outreach-status":
		outreachKPIStatus(args[1:])
	case "outreach-report":
		outreachReport(args[1:])
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
	// Monday of week 1. Go's time.Weekday is Sunday=0..Saturday=6, but ISO
	// weeks are Monday=1..Sunday=7 — must remap Sunday to 7 before
	// subtracting, or the "-int(...)" trick flips sign and lands a week late
	// whenever Jan 4 itself falls on a Sunday (e.g. 2026).
	isoWeekday := int(jan4.Weekday())
	if isoWeekday == 0 {
		isoWeekday = 7
	}
	weekOneMonday := jan4.AddDate(0, 0, -(isoWeekday - 1))
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

// kpiTeamData is the data-returning core of `kpi team` — extracted so
// sme-analytics can reuse the exact same targets query instead of a second,
// possibly-drifting copy.
func kpiTeamData(week string) (rows []map[string]interface{}, start, end string, err error) {
	start, end, _ = weekBounds(week)
	rows, err = queryRows(`
		SELECT member, contracts_target, proposals_target, meetings_target,
		       contacts_target, revenue_target, set_by, updated_at
		FROM weekly_kpis
		WHERE org_id = 'default' AND week_label = ?
		ORDER BY member
	`, week)
	return rows, start, end, err
}

// kpiTeam — tổng hợp KPI của toàn team BD trong tuần
func kpiTeam(args []string) {
	week := isoWeekLabel(vnNow())
	for i := 0; i < len(args); i++ {
		if args[i] == "--week" && i+1 < len(args) {
			i++; week = args[i]
		}
	}
	rows, start, end, err := kpiTeamData(week)
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

// kpiActualData is the data-returning core of `kpi actual` — same query
// sme-analytics reuses for the KPI-summary section, so the two commands can
// never silently disagree on actual numbers.
func kpiActualData(week, member string) (rows []map[string]interface{}, start, end string, hasData bool, err error) {
	start, end, _ = weekBounds(week)
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

	rows, err = queryRows(q, qargs...)
	if err != nil {
		// Table không tồn tại hoặc không có data — not a hard error, just
		// "no actuals available" (matches the CLI command's existing
		// fallback-to-empty behavior).
		return []map[string]interface{}{}, start, end, false, nil
	}
	return rows, start, end, true, nil
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
	rows, start, end, hasData, err := kpiActualData(week, member)
	if err != nil {
		errOut(err.Error())
		return
	}
	if !hasData {
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

// ── Outreach KPI (LinkedIn activity) ─────────────────────────────────────
//
// Khác hẳn weekly_kpis ở trên (contract/proposal/meeting/contact, đặt lại
// mỗi tuần): đây là 1 target ĐẶT MỘT LẦN áp dụng mãi cho tới khi đổi lại,
// đo bằng hoạt động outreach LinkedIn thật (connection_request_sent +
// message_sent trong bảng outreach_events, ghi bởi `outreach sync`/`log-event`).

func ensureOutreachTargetTable() {
	mustDB().Exec(`CREATE TABLE IF NOT EXISTS outreach_daily_target (
		org_id       TEXT PRIMARY KEY,
		daily_target INTEGER NOT NULL,
		set_by       TEXT,
		created_at   TEXT NOT NULL,
		updated_at   TEXT NOT NULL
	)`)
}

// outreachTargetSet — đặt target outreach/ngày (tuần = target*5, T2-T6). Ghi
// đè giá trị cũ, áp dụng cho mọi ngày/tuần kể từ giờ tới khi đổi lại — KHÔNG
// cần đặt lại mỗi tuần.
func outreachTargetSet(args []string) {
	ensureOutreachTargetTable()
	var count int64
	var setBy string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--count":
			i++
			count, _ = strconv.ParseInt(args[i], 10, 64)
		case "--set-by":
			i++
			setBy = args[i]
		}
	}
	if count <= 0 {
		errOut("cần --count > 0 (số người outreach/ngày)")
		return
	}
	now := vnNowISO()
	_, err := mustDB().Exec(`
		INSERT INTO outreach_daily_target (org_id, daily_target, set_by, created_at, updated_at)
		VALUES ('default', ?, ?, ?, ?)
		ON CONFLICT(org_id) DO UPDATE SET
			daily_target = excluded.daily_target,
			set_by       = excluded.set_by,
			updated_at   = excluded.updated_at
	`, count, setBy, now, now)
	if err != nil {
		errOut(err.Error())
		return
	}
	okOut(map[string]interface{}{
		"daily_target": count, "weekly_target": count * 5, "set_by": setBy,
		"message": fmt.Sprintf("Đã lưu target outreach: %d người/ngày (%d người/tuần T2-T6). Áp dụng mãi tới khi đổi lại.", count, count*5),
	})
}

// outreachTargetGet — đọc target hiện tại, không đổi gì.
func outreachTargetGet(args []string) {
	ensureOutreachTargetTable()
	row, _ := queryOne(`SELECT daily_target, set_by, updated_at FROM outreach_daily_target WHERE org_id = 'default'`)
	if row == nil {
		jsonOut(map[string]interface{}{
			"ok": true, "target_set": false,
			"message": "Chưa đặt target outreach — dùng: sme-cli kpi outreach-target set --count N",
		})
		return
	}
	row["ok"] = true
	row["target_set"] = true
	if dt, ok := row["daily_target"].(int64); ok {
		row["weekly_target"] = dt * 5
	}
	jsonOut(row)
}

// nextDay returns the calendar date after `date` (YYYY-MM-DD) as the same
// format. Used to build EXCLUSIVE upper bounds ("< nextDay+T00:00:00")
// instead of "<= date+T23:59:59" for occurred_at range queries — the
// inclusive form breaks when a stored value is exactly "date+T23:59:59" but
// with a timezone suffix appended (e.g. "...T23:59:59+07:00"), because that
// string sorts LEXICOGRAPHICALLY AFTER the bound "...T23:59:59" (a string is
// always "less than" a longer string sharing its full prefix) and gets
// wrongly excluded. Real bug hit 2026-09-16: resolveCardDate backdates
// stragglers to exactly "date+T23:59:59+07:00", so this excluded 49 of 52
// real messages from that day's count until fixed.
func nextDay(date string) string {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.UTC
	}
	t, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}

// countLinkedInEvents đếm số outreach_events kênh LinkedIn theo event_type,
// trong khoảng ngày [fromDate, toDate] (VN local date, inclusive cả 2 đầu).
func countLinkedInEvents(fromDate, toDate string, eventTypes []string) int64 {
	placeholders := ""
	qargs := make([]interface{}, 0, len(eventTypes)+2)
	for i, et := range eventTypes {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		qargs = append(qargs, et)
	}
	qargs = append(qargs, fromDate+"T00:00:00", nextDay(toDate)+"T00:00:00")
	row, _ := queryOne(`
		SELECT COUNT(*) as c FROM outreach_events
		WHERE org_id = 'default' AND channel = 'linkedin'
		  AND event_type IN (`+placeholders+`)
		  AND occurred_at >= ? AND occurred_at < ?
	`, qargs...)
	if row == nil {
		return 0
	}
	c, _ := row["c"].(int64)
	return c
}

// countDistinctProfilesForDay đếm số PROFILE riêng biệt (theo tên hiển thị
// LinkedIn) có tin nhắn "message_sent" trong đúng 1 ngày `date` — mỗi profile
// tính đúng 1 lần cho ngày đó, kể cả nhắn qua lại nhiều lần trong ngày.
//
// Đổi 2026-09-16: bỏ định nghĩa cũ "chỉ tính lần nhắn ĐẦU TIÊN trong toàn bộ
// lịch sử" — user xác nhận đó không phải điều cần track, và việc loại trừ
// follow-up khiến số liệu trông như thiếu (giống lỗi) dù thực ra là chủ đích
// cũ. Giờ CHỈ dedup trong phạm vi 1 ngày, không nhìn lịch sử trước đó.
//
// Giới hạn đã biết: khớp theo TÊN hiển thị (data từ scrape inbox không có
// profile_url riêng cho message), nên 2 người trùng tên hiển thị sẽ bị gộp
// nhầm thành 1 — chấp nhận được vì hiếm, nhưng cần biết trước.
func countDistinctProfilesForDay(date string) int64 {
	row, _ := queryOne(`
		SELECT COUNT(DISTINCT name) as c FROM outreach_events
		WHERE org_id = 'default' AND channel = 'linkedin' AND event_type = 'message_sent'
		  AND name != '' AND occurred_at >= ? AND occurred_at < ?
	`, date+"T00:00:00", nextDay(date)+"T00:00:00")
	if row == nil {
		return 0
	}
	c, _ := row["c"].(int64)
	return c
}

// countDistinctProfilesForRange đếm tổng số cặp (ngày, profile) riêng biệt
// trong [fromDate, toDate] — bằng tổng countDistinctProfilesForDay của từng
// ngày trong khoảng đó. 1 người nhắn ở 2 ngày KHÁC NHAU được tính 2 lần (mỗi
// ngày 1 lần, đúng ý "mỗi profile chỉ tính 1 lần/ngày"), nhưng nhắn nhiều lần
// TRONG CÙNG 1 NGÀY chỉ tính 1.
func countDistinctProfilesForRange(fromDate, toDate string) int64 {
	row, _ := queryOne(`
		SELECT COUNT(*) as c FROM (
			SELECT substr(occurred_at, 1, 10) as d, name
			FROM outreach_events
			WHERE org_id = 'default' AND channel = 'linkedin' AND event_type = 'message_sent'
			  AND name != '' AND occurred_at >= ? AND occurred_at < ?
			GROUP BY d, name
		) distinct_pairs
	`, fromDate+"T00:00:00", nextDay(toDate)+"T00:00:00")
	if row == nil {
		return 0
	}
	c, _ := row["c"].(int64)
	return c
}

// countNewVsFollowupForRange chia Daily/Weekly KPI thành 2 chỉ số phụ, cho mỗi
// (ngày, profile) riêng biệt trong [fromDate, toDate]:
//   - "New Outreach" — ngày đó là lần ĐẦU TIÊN trong toàn bộ lịch sử nhắn cho
//     profile này (global_first == đúng ngày đang xét).
//   - "Follow-up" — profile này đã có message_sent TRƯỚC ngày đang xét
//     (global_first < ngày đang xét) — tức đang tiếp tục 1 lead cũ.
//
// New + Follow-up LUÔN cộng lại đúng bằng countDistinctProfilesForDay/Range —
// đây chỉ là breakdown, KHÔNG phải định nghĩa đếm khác.
func countNewVsFollowupForRange(fromDate, toDate string) (newCount, followupCount int64) {
	row, _ := queryOne(`
		SELECT
			SUM(CASE WHEN global_first = d THEN 1 ELSE 0 END) as new_count,
			SUM(CASE WHEN global_first < d THEN 1 ELSE 0 END) as followup_count
		FROM (
			SELECT
				substr(oe.occurred_at, 1, 10) as d,
				oe.name as name,
				(SELECT substr(MIN(o2.occurred_at), 1, 10) FROM outreach_events o2
					WHERE o2.org_id = 'default' AND o2.channel = 'linkedin'
					  AND o2.event_type = 'message_sent' AND o2.name = oe.name) as global_first
			FROM outreach_events oe
			WHERE oe.org_id = 'default' AND oe.channel = 'linkedin' AND oe.event_type = 'message_sent'
			  AND oe.name != '' AND oe.occurred_at >= ? AND oe.occurred_at < ?
			GROUP BY d, oe.name
		) classified
	`, fromDate+"T00:00:00", nextDay(toDate)+"T00:00:00")
	if row == nil {
		return 0, 0
	}
	newCount, _ = row["new_count"].(int64)
	followupCount, _ = row["followup_count"].(int64)
	return newCount, followupCount
}

// outreachKPIStatus — tình hình outreach hôm nay + tuần này so với target đã
// đặt sẵn (outreachTargetSet). Dùng cho cả 4 mốc cron: 9h/13h/18h ngày
// thường và 9h cuối tuần (xem reminder/SKILL.md phần OUTREACH_KPI_*).
func outreachKPIStatus(args []string) {
	ensureOutreachTargetTable()
	date := vnToday()
	for i := 0; i < len(args); i++ {
		if args[i] == "--date" && i+1 < len(args) {
			i++
			date = args[i]
		}
	}

	targetRow, _ := queryOne(`SELECT daily_target FROM outreach_daily_target WHERE org_id = 'default'`)
	if targetRow == nil {
		jsonOut(map[string]interface{}{
			"ok": true, "target_set": false, "date": date,
			"message": "Chưa đặt target outreach — dùng: sme-cli kpi outreach-target set --count N",
		})
		return
	}
	dailyTarget, _ := targetRow["daily_target"].(int64)
	weeklyTarget := dailyTarget * 5

	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.UTC
	}
	t, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		errOut("date format không hợp lệ, dùng YYYY-MM-DD")
		return
	}
	weekLabel := isoWeekLabel(t)
	weekStart, weekEnd, _ := weekBounds(weekLabel)
	weekday := t.Weekday()
	isWeekend := weekday == time.Saturday || weekday == time.Sunday

	dayActual := countDistinctProfilesForDay(date)
	weekActual := countDistinctProfilesForRange(weekStart, date)
	repliesToday := countLinkedInEvents(date, date, []string{"message_reply_received"})

	dayRemaining := dailyTarget - dayActual
	if dayRemaining < 0 {
		dayRemaining = 0
	}
	weekRemaining := weeklyTarget - weekActual
	if weekRemaining < 0 {
		weekRemaining = 0
	}

	okOut(map[string]interface{}{
		"date": date, "is_weekend": isWeekend,
		"day_target": dailyTarget, "day_actual": dayActual, "day_remaining": dayRemaining,
		"week_start": weekStart, "week_end": weekEnd,
		"week_target": weeklyTarget, "week_actual": weekActual, "week_remaining": weekRemaining,
		"week_target_met": weekActual >= weeklyTarget,
		"replies_today":   repliesToday,
		"message": fmt.Sprintf("Hôm nay nhắn tin lần đầu cho %d/%d người mới, tuần %s→%s %d/%d người (còn thiếu %d), %d reply hôm nay",
			dayActual, dailyTarget, weekStart, weekEnd, weekActual, weeklyTarget, weekRemaining, repliesToday),
	})
}

// ── Outreach report: finalize/provisional cron slots ─────────────────────
//
// Chỉ 3 cron slot thật (không còn 13h/22h):
//   0 9 * * 1-5  -> --slot morning   (Monday: weekly summary + morning update;
//                                     Tue-Fri: finalize hôm qua + morning update)
//   0 18 * * 1-5 -> --slot evening   (PROVISIONAL only — KHÔNG phải final)
//   0 9 * * 6    -> --slot weekend   (finalize Friday + weekend KPI check,
//                                     PROVISIONAL ở cấp tuần)
// Không có cron Chủ Nhật. Hoạt động Sat/Sun vẫn được tính đủ vào tuần trước
// khi Monday 9h finalize lại bằng date-range thật (Mon->Sun), không phải suy
// từ nhãn "Yesterday" (đó chỉ là cơ chế lúc SYNC để không mất dữ liệu).

func ensureSnapshotTables() {
	mustDB().Exec(`CREATE TABLE IF NOT EXISTS outreach_day_snapshot (
		org_id      TEXT NOT NULL DEFAULT 'default',
		date        TEXT NOT NULL,
		day_actual  INTEGER NOT NULL,
		status      TEXT NOT NULL,
		recorded_at TEXT NOT NULL,
		PRIMARY KEY (org_id, date)
	)`)
	mustDB().Exec(`CREATE TABLE IF NOT EXISTS outreach_week_snapshot (
		org_id      TEXT NOT NULL DEFAULT 'default',
		week_label  TEXT NOT NULL,
		week_actual INTEGER NOT NULL,
		status      TEXT NOT NULL,
		recorded_at TEXT NOT NULL,
		PRIMARY KEY (org_id, week_label)
	)`)
}

// previousWorkingDay returns the last working day (Mon-Fri) before t.
// Monday's previous working day is Friday (skip the weekend); every other
// day is simply t-1. NEVER hard-code "yesterday = today - 1 day" for
// business-day logic — that's exactly what silently broke Monday's
// finalization before this existed.
func previousWorkingDay(t time.Time) time.Time {
	if t.Weekday() == time.Monday {
		return t.AddDate(0, 0, -3)
	}
	return t.AddDate(0, 0, -1)
}

func pctOf(actual, target int64) float64 {
	if target <= 0 {
		return 0
	}
	return math.Round(float64(actual)/float64(target)*1000) / 10
}

// finalizeDay computes the TRUE count for `date` straight from occurred_at
// date-ranges (not from any "Yesterday"-label reasoning) and records it as
// FINAL. Call this only once — the morning after `date`, after that
// morning's sync has already pulled in any stragglers LinkedIn had still
// labeled "Yesterday".
func finalizeDay(date string) int64 {
	ensureSnapshotTables()
	actual := countDistinctProfilesForDay(date)
	now := vnNowISO()
	mustDB().Exec(`
		INSERT INTO outreach_day_snapshot (org_id, date, day_actual, status, recorded_at)
		VALUES ('default', ?, ?, 'FINAL', ?)
		ON CONFLICT(org_id, date) DO UPDATE SET
			day_actual = excluded.day_actual, status = 'FINAL', recorded_at = excluded.recorded_at
	`, date, actual, now)
	return actual
}

// snapshotDayProvisional records today's count-so-far as PROVISIONAL — NOT
// final, since anything sent after this snapshot (e.g. after the 18h check)
// won't show up until finalizeDay runs the next working-day morning.
func snapshotDayProvisional(date string) int64 {
	ensureSnapshotTables()
	actual := countDistinctProfilesForDay(date)
	now := vnNowISO()
	mustDB().Exec(`
		INSERT INTO outreach_day_snapshot (org_id, date, day_actual, status, recorded_at)
		VALUES ('default', ?, ?, 'PROVISIONAL', ?)
		ON CONFLICT(org_id, date) DO UPDATE SET
			day_actual = excluded.day_actual, status = 'PROVISIONAL', recorded_at = excluded.recorded_at
	`, date, actual, now)
	return actual
}

// finalizeWeek computes the TRUE weekly total for [weekStart, weekEnd] from
// real occurred_at date-ranges and records it FINAL. Only Monday 9h calls
// this, for the week that just ended.
func finalizeWeek(weekLabel, weekStart, weekEnd string) int64 {
	ensureSnapshotTables()
	actual := countDistinctProfilesForRange(weekStart, weekEnd)
	now := vnNowISO()
	mustDB().Exec(`
		INSERT INTO outreach_week_snapshot (org_id, week_label, week_actual, status, recorded_at)
		VALUES ('default', ?, ?, 'FINAL', ?)
		ON CONFLICT(org_id, week_label) DO UPDATE SET
			week_actual = excluded.week_actual, status = 'FINAL', recorded_at = excluded.recorded_at
	`, weekLabel, actual, now)
	return actual
}

// snapshotWeekProvisional records the current week's total-so-far
// (weekStart..throughDate) as PROVISIONAL. Used by Saturday's weekend check
// — Sat/Sun can still add outreach, so the week isn't final yet.
func snapshotWeekProvisional(weekLabel, weekStart, throughDate string) int64 {
	ensureSnapshotTables()
	actual := countDistinctProfilesForRange(weekStart, throughDate)
	now := vnNowISO()
	mustDB().Exec(`
		INSERT INTO outreach_week_snapshot (org_id, week_label, week_actual, status, recorded_at)
		VALUES ('default', ?, ?, 'PROVISIONAL', ?)
		ON CONFLICT(org_id, week_label) DO UPDATE SET
			week_actual = excluded.week_actual, status = 'PROVISIONAL', recorded_at = excluded.recorded_at
	`, weekLabel, actual, now)
	return actual
}

// outreachReport implements the 3 real cron slots. Each slot has different
// finalize/provisional semantics — see kpi/SKILL.md "OUTREACH KPI" for the
// exact message templates the agent must send per slot.
//
//	sme-cli kpi outreach-report --slot morning|evening|weekend [--date YYYY-MM-DD]
func outreachReport(args []string) {
	ensureOutreachTargetTable()
	ensureSnapshotTables()
	slot := ""
	dateOverride := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--slot":
			i++
			slot = args[i]
		case "--date":
			i++
			dateOverride = args[i]
		}
	}
	if slot != "morning" && slot != "evening" && slot != "weekend" {
		errOut("cần --slot morning|evening|weekend")
		return
	}

	targetRow, _ := queryOne(`SELECT daily_target FROM outreach_daily_target WHERE org_id = 'default'`)
	if targetRow == nil {
		jsonOut(map[string]interface{}{
			"ok": true, "target_set": false, "slot": slot,
			"message": "Chưa đặt target outreach — dùng: sme-cli kpi outreach-target set --count N",
		})
		return
	}
	dailyTarget, _ := targetRow["daily_target"].(int64)
	weeklyTarget := dailyTarget * 5

	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.UTC
	}
	var today time.Time
	if dateOverride != "" {
		today, err = time.ParseInLocation("2006-01-02", dateOverride, loc)
		if err != nil {
			errOut("date format không hợp lệ, dùng YYYY-MM-DD")
			return
		}
	} else {
		today = vnNow()
	}
	todayStr := today.Format("2006-01-02")
	weekLabel := isoWeekLabel(today)
	weekStart, weekEnd, _ := weekBounds(weekLabel)
	weekActualSoFar := countDistinctProfilesForRange(weekStart, todayStr)
	weekRemaining := weeklyTarget - weekActualSoFar
	if weekRemaining < 0 {
		weekRemaining = 0
	}

	switch slot {
	case "morning":
		if today.Weekday() == time.Monday {
			prevWeekMonday := today.AddDate(0, 0, -7)
			prevWeekLabel := isoWeekLabel(prevWeekMonday)
			prevWeekStart, prevWeekEnd, _ := weekBounds(prevWeekLabel)
			prevWeekActual := finalizeWeek(prevWeekLabel, prevWeekStart, prevWeekEnd)
			prevWeekNew, prevWeekFollowup := countNewVsFollowupForRange(prevWeekStart, prevWeekEnd)

			todayActual := countDistinctProfilesForDay(todayStr)
			dayRemaining := dailyTarget - todayActual
			if dayRemaining < 0 {
				dayRemaining = 0
			}

			okOut(map[string]interface{}{
				"job_type": "WEEKLY_SUMMARY+MORNING_UPDATE", "date": todayStr, "weekday": "Monday",
				"prev_week_label": prevWeekLabel, "prev_week_start": prevWeekStart, "prev_week_end": prevWeekEnd,
				"prev_week_actual": prevWeekActual, "prev_week_target": weeklyTarget, "prev_week_status": "FINAL",
				"prev_week_new": prevWeekNew, "prev_week_followup": prevWeekFollowup,
				"prev_week_achievement_pct": pctOf(prevWeekActual, weeklyTarget),
				"day_target":                dailyTarget, "day_actual_so_far": todayActual, "day_remaining": dayRemaining,
				"week_label": weekLabel, "week_start": weekStart, "week_end": weekEnd,
				"week_target": weeklyTarget, "week_actual_so_far": weekActualSoFar, "week_remaining": weekRemaining,
				"message": fmt.Sprintf("[Weekly Summary tuần %s→%s, FINAL] %d/%d người (%.1f%%) — %d mới, %d follow-up. [Morning] hôm nay target %d, tuần mới %d/%d.",
					prevWeekStart, prevWeekEnd, prevWeekActual, weeklyTarget, pctOf(prevWeekActual, weeklyTarget), prevWeekNew, prevWeekFollowup, dailyTarget, weekActualSoFar, weeklyTarget),
			})
			return
		}

		// Tue-Fri: finalize working day trước đó qua previousWorkingDay(),
		// KHÔNG hard-code today-1 (Tue's prev day vẫn là Mon nên trùng, nhưng
		// hàm này là nơi DUY NHẤT quyết định "hôm qua làm việc" là ngày nào).
		prevDay := previousWorkingDay(today).Format("2006-01-02")
		prevDayActual := finalizeDay(prevDay)
		prevDayNew, prevDayFollowup := countNewVsFollowupForRange(prevDay, prevDay)

		todayActual := countDistinctProfilesForDay(todayStr)
		dayRemaining := dailyTarget - todayActual
		if dayRemaining < 0 {
			dayRemaining = 0
		}

		okOut(map[string]interface{}{
			"job_type": "MORNING_UPDATE", "date": todayStr,
			"prev_day": prevDay, "prev_day_actual": prevDayActual, "prev_day_target": dailyTarget, "prev_day_status": "FINAL",
			"prev_day_new": prevDayNew, "prev_day_followup": prevDayFollowup,
			"day_target": dailyTarget, "day_actual_so_far": todayActual, "day_remaining": dayRemaining,
			"week_label": weekLabel, "week_start": weekStart, "week_end": weekEnd,
			"week_target": weeklyTarget, "week_actual_so_far": weekActualSoFar, "week_remaining": weekRemaining,
			"message": fmt.Sprintf("[Morning Update, FINAL hôm qua %s] %d/%d người (%d mới, %d follow-up). Hôm nay target %d. Tuần %d/%d (còn %d).",
				prevDay, prevDayActual, dailyTarget, prevDayNew, prevDayFollowup, dailyTarget, weekActualSoFar, weeklyTarget, weekRemaining),
		})

	case "evening":
		todayActual := snapshotDayProvisional(todayStr)
		todayNew, todayFollowup := countNewVsFollowupForRange(todayStr, todayStr)
		repliesToday := countLinkedInEvents(todayStr, todayStr, []string{"message_reply_received"})
		dayRemaining := dailyTarget - todayActual
		if dayRemaining < 0 {
			dayRemaining = 0
		}

		okOut(map[string]interface{}{
			"job_type": "DAILY_PROGRESS", "status": "PROVISIONAL", "date": todayStr,
			"day_target": dailyTarget, "day_actual": todayActual, "day_remaining": dayRemaining,
			"day_new": todayNew, "day_followup": todayFollowup,
			"week_label": weekLabel, "week_start": weekStart, "week_end": weekEnd,
			"week_target": weeklyTarget, "week_actual": weekActualSoFar, "week_remaining": weekRemaining,
			"replies_today": repliesToday,
			"message": fmt.Sprintf("[Daily Progress — PROVISIONAL, chưa phải final] %d/%d người hôm nay (%d mới, %d follow-up). %d reply. Tuần %d/%d.",
				todayActual, dailyTarget, todayNew, todayFollowup, repliesToday, weekActualSoFar, weeklyTarget),
		})

	case "weekend":
		friday := previousWorkingDay(today).Format("2006-01-02")
		fridayActual := finalizeDay(friday)
		fridayNew, fridayFollowup := countNewVsFollowupForRange(friday, friday)
		snapshotWeekProvisional(weekLabel, weekStart, todayStr)
		weekNew, weekFollowup := countNewVsFollowupForRange(weekStart, todayStr)

		okOut(map[string]interface{}{
			"job_type": "WEEKEND_KPI_CHECK", "week_status": "PROVISIONAL", "date": todayStr,
			"friday_date": friday, "friday_actual": fridayActual, "friday_target": dailyTarget, "friday_status": "FINAL",
			"friday_new": fridayNew, "friday_followup": fridayFollowup,
			"week_label": weekLabel, "week_start": weekStart, "week_end": weekEnd,
			"week_target": weeklyTarget, "week_actual": weekActualSoFar, "week_remaining": weekRemaining,
			"week_new": weekNew, "week_followup": weekFollowup,
			"achievement_pct":  pctOf(weekActualSoFar, weeklyTarget),
			"week_target_met":  weekActualSoFar >= weeklyTarget,
			"message": fmt.Sprintf("[Weekend KPI Check, tuần vẫn PROVISIONAL] Thứ Sáu %d/%d (%d mới, %d follow-up). Tuần: %d/%d người (%.1f%%) — %d mới, %d follow-up, còn thiếu %d.",
				fridayActual, dailyTarget, fridayNew, fridayFollowup, weekActualSoFar, weeklyTarget, pctOf(weekActualSoFar, weeklyTarget), weekNew, weekFollowup, weekRemaining),
		})
	}
}
