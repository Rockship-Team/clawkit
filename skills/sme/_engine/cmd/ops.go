package main

import "fmt"

func cmdTask(args []string) {
	if len(args) == 0 {
		errOut("usage: task add|list|update|done|cancel")
	}
	org := defaultOrgID()
	switch args[0] {
	case "add":
		// task add <title> [assigned_to] [due_date] [priority] [description]
		if len(args) < 2 {
			errOut("usage: task add <title> [assigned_to] [due_date] [priority] [description]")
		}
		id := newID()
		now := vnNowISO()
		assigned, due, priority, desc := "", "", "medium", ""
		if len(args) > 2 {
			assigned = args[2]
		}
		if len(args) > 3 {
			due = args[3]
		}
		if len(args) > 4 {
			priority = args[4]
		}
		if len(args) > 5 {
			desc = args[5]
		}
		_, err := exec(`INSERT INTO tasks (id,org_id,title,description,assigned_to,status,priority,due_date,created_at,updated_at)
			VALUES (?,?,?,?,NULLIF(?,''),?,?,NULLIF(?,''),?,?)`,
			id, org, args[1], desc, assigned, "todo", priority, due, now, now)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"id": id, "title": args[1], "status": "todo"})

	case "list":
		status := "active"
		if len(args) > 1 {
			status = args[1]
		}
		q := "SELECT id,title,assigned_to,status,priority,due_date,created_at FROM tasks WHERE org_id=?"
		qargs := []interface{}{org}
		if status == "active" {
			q += " AND status IN ('todo','in_progress')"
		} else if status != "all" {
			q += " AND status=?"
			qargs = append(qargs, status)
		}
		q += " ORDER BY CASE priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END, due_date LIMIT 50"
		rows, _ := queryRows(q, qargs...)
		okOut(map[string]interface{}{"tasks": rows, "count": len(rows)})

	case "update":
		if len(args) < 4 {
			errOut("usage: task update <id> <field> <value>")
		}
		allowed := map[string]bool{"title": true, "description": true, "assigned_to": true, "status": true, "priority": true, "due_date": true}
		if !allowed[args[2]] {
			errOut("cannot update: " + args[2])
		}
		exec(fmt.Sprintf("UPDATE tasks SET %s=?, updated_at=? WHERE id=? AND org_id=?", args[2]),
			args[3], vnNowISO(), args[1], org)
		okOut(map[string]interface{}{"updated": args[1], "field": args[2]})

	case "done":
		if len(args) < 2 {
			errOut("usage: task done <id>")
		}
		exec("UPDATE tasks SET status='done', completed_at=?, updated_at=? WHERE id=? AND org_id=?",
			vnNowISO(), vnNowISO(), args[1], org)
		okOut(map[string]interface{}{"done": args[1]})

	case "cancel":
		if len(args) < 2 {
			errOut("usage: task cancel <id>")
		}
		exec("UPDATE tasks SET status='cancelled', updated_at=? WHERE id=? AND org_id=?",
			vnNowISO(), args[1], org)
		okOut(map[string]interface{}{"cancelled": args[1]})

	default:
		errOut("unknown task command: " + args[0])
	}
}

func cmdDocument(args []string) {
	if len(args) == 0 {
		errOut("usage: document add|list|search")
	}
	org := defaultOrgID()
	switch args[0] {
	case "add":
		// document add <name> <category> <file_url> [file_type] [expires_at] [entity_type] [entity_id]
		if len(args) < 4 {
			errOut("usage: document add <name> <category> <file_url> [file_type] [expires_at]")
		}
		id := newID()
		ftype, expires := "", ""
		if len(args) > 4 {
			ftype = args[4]
		}
		if len(args) > 5 {
			expires = args[5]
		}
		_, err := exec(`INSERT INTO documents (id,org_id,name,category,file_url,file_type,expires_at,created_at)
			VALUES (?,?,?,?,?,?,NULLIF(?,''),?)`, id, org, args[1], args[2], args[3], ftype, expires, vnNowISO())
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"id": id, "name": args[1], "category": args[2]})

	case "list":
		cat := "all"
		if len(args) > 1 {
			cat = args[1]
		}
		q := "SELECT id,name,category,file_type,expires_at,created_at FROM documents WHERE org_id=?"
		qargs := []interface{}{org}
		if cat != "all" {
			q += " AND category=?"
			qargs = append(qargs, cat)
		}
		q += " ORDER BY created_at DESC LIMIT 50"
		rows, _ := queryRows(q, qargs...)
		okOut(map[string]interface{}{"documents": rows, "count": len(rows)})

	case "search":
		if len(args) < 2 {
			errOut("usage: document search <query>")
		}
		q := "%" + args[1] + "%"
		rows, _ := queryRows(`SELECT id,name,category,file_type FROM documents
			WHERE org_id=? AND (name LIKE ? OR ocr_text LIKE ?) LIMIT 20`, org, q, q)
		okOut(map[string]interface{}{"documents": rows, "count": len(rows)})

	case "expiring":
		thirtyDays := vnNow().AddDate(0, 0, 30).Format("2006-01-02")
		rows, _ := queryRows(`SELECT id,name,category,expires_at FROM documents
			WHERE org_id=? AND expires_at IS NOT NULL AND expires_at <= ? AND expires_at >= ?
			ORDER BY expires_at`, org, thirtyDays, vnToday())
		okOut(map[string]interface{}{"expiring": rows, "count": len(rows)})

	default:
		errOut("unknown document command: " + args[0])
	}
}

func cmdLicense(args []string) {
	if len(args) == 0 {
		errOut("usage: license add|list|expiring")
	}
	org := defaultOrgID()
	switch args[0] {
	case "add":
		// license add <type> <number> <issued_by> [issued_date] [expiry_date]
		if len(args) < 4 {
			errOut("usage: license add <type> <number> <issued_by> [issued_date] [expiry_date]")
		}
		id := newID()
		issued, expiry := "", ""
		if len(args) > 4 {
			issued = args[4]
		}
		if len(args) > 5 {
			expiry = args[5]
		}
		_, err := exec(`INSERT INTO licenses (id,org_id,license_type,license_number,issued_by,issued_date,expiry_date,status,created_at)
			VALUES (?,?,?,?,?,NULLIF(?,''),NULLIF(?,''),?,?)`,
			id, org, args[1], args[2], args[3], issued, expiry, "active", vnNowISO())
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"id": id, "type": args[1], "number": args[2]})

	case "list":
		rows, _ := queryRows("SELECT id,license_type,license_number,issued_by,expiry_date,status FROM licenses WHERE org_id=? ORDER BY expiry_date", org)
		okOut(map[string]interface{}{"licenses": rows, "count": len(rows)})

	case "expiring":
		ninetyDays := vnNow().AddDate(0, 0, 90).Format("2006-01-02")
		rows, _ := queryRows(`SELECT id,license_type,license_number,expiry_date,status FROM licenses
			WHERE org_id=? AND expiry_date IS NOT NULL AND expiry_date <= ? AND expiry_date >= ?
			ORDER BY expiry_date`, org, ninetyDays, vnToday())
		okOut(map[string]interface{}{"expiring": rows, "count": len(rows)})

	default:
		errOut("unknown license command: " + args[0])
	}
}
