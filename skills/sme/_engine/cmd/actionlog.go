package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// cmdActionLog tracks suggestions the bot made and whether they were acted on.
//
//	sme-cli action-log suggest --contact-name NAME --action TEXT [--contact-id ID] [--source morning|evening|user]
//	sme-cli action-log done --contact-name NAME [--note TEXT] [--by NAME]
//	sme-cli action-log skip --contact-name NAME [--reason TEXT]
//	sme-cli action-log rate [--days N]
//	sme-cli action-log pending
//	sme-cli action-log auto-check
func cmdActionLog(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: action-log suggest|done|skip|rate|pending|auto-check")
		os.Exit(1)
	}
	ensureActionLogTable()
	switch args[0] {
	case "suggest":
		actionSuggest(args[1:])
	case "done":
		actionDone(args[1:])
	case "skip":
		actionSkip(args[1:])
	case "rate":
		actionRate(args[1:])
	case "pending":
		actionPending(args[1:])
	case "auto-check":
		actionAutoCheck(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "usage: action-log suggest|done|skip|rate|pending|auto-check\n")
		os.Exit(1)
	}
}

func ensureActionLogTable() {
	mustDB().Exec(`CREATE TABLE IF NOT EXISTS action_suggestions (
		id           TEXT PRIMARY KEY,
		contact_id   TEXT DEFAULT '',
		contact_name TEXT NOT NULL,
		action_text  TEXT NOT NULL,
		source       TEXT DEFAULT 'morning',
		suggested_at TEXT NOT NULL,
		week_label   TEXT NOT NULL,
		status       TEXT DEFAULT 'pending',
		done_at      TEXT,
		done_by      TEXT,
		note         TEXT,
		created_at   TEXT NOT NULL
	)`)
	mustDB().Exec(`CREATE INDEX IF NOT EXISTS idx_action_week ON action_suggestions(week_label, status)`)
	mustDB().Exec(`CREATE INDEX IF NOT EXISTS idx_action_contact ON action_suggestions(contact_name, status)`)
}

func actionSuggest(args []string) {
	var contactID, contactName, action, source string
	source = "morning"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--contact-id":
			i++; contactID = args[i]
		case "--contact-name":
			i++; contactName = args[i]
		case "--action":
			i++; action = args[i]
		case "--source":
			i++; source = args[i]
		}
	}
	if contactName == "" || action == "" {
		errOut("--contact-name và --action là bắt buộc")
		return
	}
	now := vnNowISO()
	week := isoWeekLabel(vnNow())
	_, err := mustDB().Exec(`
		INSERT INTO action_suggestions
			(id, contact_id, contact_name, action_text, source, suggested_at, week_label, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', ?)
	`, newID(), contactID, contactName, action, source, now, week, now)
	if err != nil {
		errOut(err.Error())
		return
	}
	okOut(map[string]interface{}{
		"contact": contactName,
		"action":  action,
		"week":    week,
		"message": fmt.Sprintf("Logged: %s → %s", contactName, action),
	})
}

func actionDone(args []string) {
	var contactName, note, doneBy string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--contact-name":
			i++; contactName = args[i]
		case "--note":
			i++; note = args[i]
		case "--by":
			i++; doneBy = args[i]
		}
	}
	if contactName == "" {
		errOut("--contact-name là bắt buộc")
		return
	}
	now := vnNowISO()
	rows, _ := queryRows(`
		SELECT id, contact_name, action_text FROM action_suggestions
		WHERE status = 'pending'
		  AND lower(contact_name) LIKE lower('%'||?||'%')
		ORDER BY suggested_at DESC LIMIT 5
	`, contactName)

	if len(rows) == 0 {
		// Tạo done log trực tiếp dù không có suggestion trước
		mustDB().Exec(`INSERT INTO action_suggestions
			(id,contact_id,contact_name,action_text,source,suggested_at,week_label,status,done_at,done_by,note,created_at)
			VALUES (?,''	,?,'manual done','user',?,?,?,?,?,?,?)`,
			newID(), contactName, now, isoWeekLabel(vnNow()), "done", now, doneBy, note, now)
		okOut(map[string]interface{}{"contact": contactName, "status": "done", "note": "manual (no prior suggestion found)"})
		return
	}
	for _, r := range rows {
		mustDB().Exec(`UPDATE action_suggestions SET status='done',done_at=?,done_by=?,note=? WHERE id=?`,
			now, doneBy, note, toString(r["id"]))
	}
	okOut(map[string]interface{}{
		"contact": contactName,
		"updated": len(rows),
		"message": fmt.Sprintf("Marked done: %d item(s) cho %s", len(rows), contactName),
	})
}

func actionSkip(args []string) {
	var contactName, reason string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--contact-name":
			i++; contactName = args[i]
		case "--reason":
			i++; reason = args[i]
		}
	}
	if contactName == "" {
		errOut("--contact-name là bắt buộc")
		return
	}
	now := vnNowISO()
	res, _ := mustDB().Exec(`
		UPDATE action_suggestions SET status='skipped', done_at=?, note=?
		WHERE status='pending' AND lower(contact_name) LIKE lower('%'||?||'%')
	`, now, reason, contactName)
	n, _ := res.RowsAffected()
	okOut(map[string]interface{}{"contact": contactName, "skipped": n, "reason": reason})
}

func actionRate(args []string) {
	days := 7
	for i := 0; i < len(args); i++ {
		if args[i] == "--days" && i+1 < len(args) {
			i++; fmt.Sscanf(args[i], "%d", &days)
		}
	}
	since := vnNow().AddDate(0, 0, -days).Format("2006-01-02")

	rows, _ := queryRows(`
		SELECT status, COUNT(*) as count FROM action_suggestions
		WHERE suggested_at >= ? GROUP BY status
	`, since)

	counts := map[string]int64{"pending": 0, "done": 0, "skipped": 0}
	for _, r := range rows {
		counts[toString(r["status"])] = toInt64(r["count"])
	}
	total := counts["pending"] + counts["done"] + counts["skipped"]
	rate := 0.0
	if total > 0 {
		rate = float64(counts["done"]) / float64(total) * 100
	}
	resolved := counts["done"] + counts["skipped"]
	resolvedRate := 0.0
	if resolved > 0 {
		resolvedRate = float64(counts["done"]) / float64(resolved) * 100
	}

	weekRows, _ := queryRows(`
		SELECT week_label,
		       SUM(CASE WHEN status='done' THEN 1 ELSE 0 END) as done,
		       SUM(CASE WHEN status='skipped' THEN 1 ELSE 0 END) as skipped,
		       COUNT(*) as total
		FROM action_suggestions WHERE suggested_at >= ?
		GROUP BY week_label ORDER BY week_label DESC
	`, since)

	interp := "Chưa đủ data (cần ít nhất 5 suggestions)"
	if total >= 5 {
		if rate >= 70 {
			interp = "Tốt — team đang act on suggestions của bot"
		} else if rate >= 40 {
			interp = "Trung bình — khoảng một nửa suggestions được thực hiện"
		} else {
			interp = "Thấp — suggestions chưa phù hợp hoặc team chưa dùng bot thường xuyên"
		}
	}

	okOut(map[string]interface{}{
		"period_days":       days,
		"since":             since,
		"total_suggestions": total,
		"done":              counts["done"],
		"skipped":           counts["skipped"],
		"pending":           counts["pending"],
		"action_rate_pct":   fmt.Sprintf("%.1f%%", rate),
		"resolved_rate_pct": fmt.Sprintf("%.1f%%", resolvedRate),
		"by_week":           weekRows,
		"interpretation":    interp,
	})
}

func actionPending(args []string) {
	rows, _ := queryRows(`
		SELECT contact_name, action_text, source, suggested_at,
		       CAST(julianday('now') - julianday(suggested_at) AS INTEGER) as days_old
		FROM action_suggestions WHERE status='pending'
		ORDER BY suggested_at ASC LIMIT 20
	`)
	okOut(map[string]interface{}{
		"pending": rows,
		"count":   len(rows),
		"message": fmt.Sprintf("%d suggestions chưa được xử lý", len(rows)),
	})
}

// actionAutoCheck queries COSMO for new interactions on pending contacts.
func actionAutoCheck(args []string) {
	pending, _ := queryRows(`
		SELECT id, contact_id, contact_name, action_text, suggested_at
		FROM action_suggestions
		WHERE status='pending' AND contact_id != ''
		ORDER BY suggested_at ASC LIMIT 20
	`)
	if len(pending) == 0 {
		okOut(map[string]interface{}{"checked": 0, "auto_done": 0,
			"message": "Không có pending suggestion có contact_id để check"})
		return
	}
	autoDone := 0
	results := []map[string]interface{}{}
	for _, p := range pending {
		cid := toString(p["contact_id"])
		cname := toString(p["contact_name"])
		suggestedAt := toString(p["suggested_at"])
		id := toString(p["id"])
		if cid == "" {
			continue
		}
		raw, code, err := cosmoRequest("GET",
			fmt.Sprintf("/v1/contacts/%s/interactions?limit=5", cid), nil)
		if err != nil || code != 200 {
			continue
		}
		var resp struct {
			Data []struct {
				CreatedAt string `json:"created_at"`
				Type      string `json:"type"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			continue
		}
		for _, interaction := range resp.Data {
			if len(interaction.CreatedAt) >= 10 && interaction.CreatedAt > suggestedAt[:10] {
				now := vnNowISO()
				mustDB().Exec(`UPDATE action_suggestions SET status='done',done_at=?,done_by='auto-cosmo',note=? WHERE id=?`,
					now, fmt.Sprintf("Auto: %s interaction detected", interaction.Type), id)
				autoDone++
				results = append(results, map[string]interface{}{
					"contact":     cname,
					"interaction": interaction.Type,
					"auto_done":   true,
				})
				break
			}
		}
	}
	okOut(map[string]interface{}{
		"checked":   len(pending),
		"auto_done": autoDone,
		"results":   results,
	})
}

// actionWeekSummary returns a 1-line summary for embedding in briefings.
func actionWeekSummary() string {
	week := isoWeekLabel(vnNow())
	rows, _ := queryRows(`
		SELECT status, COUNT(*) as count FROM action_suggestions
		WHERE week_label=? GROUP BY status
	`, week)
	counts := map[string]int64{}
	for _, r := range rows {
		counts[toString(r["status"])] = toInt64(r["count"])
	}
	total := counts["done"] + counts["skipped"] + counts["pending"]
	if total == 0 {
		return ""
	}
	week = strings.TrimPrefix(week, fmt.Sprintf("%d-", vnNow().Year()))
	return fmt.Sprintf("%s: %d/%d suggestions done (%.0f%%)",
		week, counts["done"], total,
		float64(counts["done"])/float64(total)*100)
}
