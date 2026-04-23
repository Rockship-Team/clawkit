package main

import (
	"fmt"
	"strconv"
)

func cmdContact(args []string) {
	if len(args) == 0 {
		errOut("usage: contact add|list|get|update|search")
	}
	org := defaultOrgID()
	switch args[0] {
	case "add":
		// contact add <type> <name> [company] [phone] [email] [tax_code]
		if len(args) < 3 {
			errOut("usage: contact add <customer|vendor|partner> <name> [company] [phone] [email] [tax_code]")
		}
		id := newID()
		now := vnNowISO()
		company, phone, email, tax := "", "", "", ""
		if len(args) > 3 {
			company = args[3]
		}
		if len(args) > 4 {
			phone = args[4]
		}
		if len(args) > 5 {
			email = args[5]
		}
		if len(args) > 6 {
			tax = args[6]
		}
		_, err := exec(`INSERT INTO contacts (id,org_id,type,full_name,company_name,phone,email,tax_code,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?)`, id, org, args[1], args[2], company, phone, email, tax, now, now)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"id": id, "type": args[1], "name": args[2]})

	case "list":
		ctype := "all"
		if len(args) > 1 {
			ctype = args[1]
		}
		q := "SELECT id,type,full_name,company_name,phone,email,tax_code FROM contacts WHERE org_id=?"
		qargs := []interface{}{org}
		if ctype != "all" {
			q += " AND type=?"
			qargs = append(qargs, ctype)
		}
		q += " ORDER BY full_name LIMIT 100"
		rows, _ := queryRows(q, qargs...)
		okOut(map[string]interface{}{"contacts": rows, "count": len(rows)})

	case "get":
		if len(args) < 2 {
			errOut("usage: contact get <id>")
		}
		row, _ := queryOne("SELECT * FROM contacts WHERE id=? AND org_id=?", args[1], org)
		if row == nil {
			errOut("contact not found")
		}
		okOut(map[string]interface{}{"contact": row})

	case "search":
		if len(args) < 2 {
			errOut("usage: contact search <query>")
		}
		q := "%" + args[1] + "%"
		rows, _ := queryRows(`SELECT id,type,full_name,company_name,phone,email FROM contacts
			WHERE org_id=? AND (full_name LIKE ? OR company_name LIKE ? OR phone LIKE ? OR email LIKE ?)
			LIMIT 20`, org, q, q, q, q)
		okOut(map[string]interface{}{"contacts": rows, "count": len(rows)})

	case "update":
		if len(args) < 4 {
			errOut("usage: contact update <id> <field> <value>")
		}
		allowed := map[string]bool{
			"full_name": true, "company_name": true, "phone": true, "email": true,
			"address": true, "tax_code": true, "bank_account": true, "bank_name": true, "type": true,
		}
		if !allowed[args[2]] {
			errOut("cannot update: " + args[2])
		}
		exec(fmt.Sprintf("UPDATE contacts SET %s=?, updated_at=? WHERE id=? AND org_id=?", args[2]),
			args[3], vnNowISO(), args[1], org)
		okOut(map[string]interface{}{"updated": args[1], "field": args[2]})

	default:
		errOut("unknown contact command: " + args[0])
	}
}

func cmdLead(args []string) {
	if len(args) == 0 {
		errOut("usage: lead add|list|update|pipeline")
	}
	org := defaultOrgID()
	switch args[0] {
	case "add":
		// lead add <contact_name> <source> <estimated_value> [assigned_to] [expected_close]
		if len(args) < 4 {
			errOut("usage: lead add <contact_name> <source> <estimated_value> [assigned_to] [expected_close]")
		}
		val := parseVND(args[3])
		assigned := ""
		if len(args) > 4 {
			assigned = args[4]
		}
		expClose := ""
		if len(args) > 5 {
			expClose = args[5]
		}
		// Find or create contact
		contactID := ""
		row, _ := queryOne("SELECT id FROM contacts WHERE org_id=? AND full_name LIKE ?", org, "%"+args[1]+"%")
		if row != nil {
			contactID = fmt.Sprintf("%v", row["id"])
		}

		id := newID()
		now := vnNowISO()
		weighted := int64(float64(val) * 0.1) // 10% default probability
		_, err := exec(`INSERT INTO leads (id,org_id,contact_id,assigned_to,source,stage,estimated_value,probability_pct,weighted_value,expected_close,last_activity,created_at,updated_at)
			VALUES (?,?,NULLIF(?,''),NULLIF(?,''),?,?,?,?,?,NULLIF(?,''),?,?,?)`,
			id, org, contactID, assigned, args[2], "new", val, 10, weighted, expClose, now, now, now)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"id": id, "contact": args[1], "value": val, "stage": "new"})

	case "list":
		stage := "all"
		if len(args) > 1 {
			stage = args[1]
		}
		q := `SELECT l.id,l.stage,l.source,l.estimated_value,l.probability_pct,l.weighted_value,l.expected_close,l.last_activity,
			c.full_name as contact_name FROM leads l LEFT JOIN contacts c ON l.contact_id=c.id WHERE l.org_id=?`
		qargs := []interface{}{org}
		if stage != "all" {
			q += " AND l.stage=?"
			qargs = append(qargs, stage)
		}
		q += " ORDER BY l.updated_at DESC LIMIT 50"
		rows, _ := queryRows(q, qargs...)
		okOut(map[string]interface{}{"leads": rows, "count": len(rows)})

	case "update":
		// lead update <id> <field> <value>
		if len(args) < 4 {
			errOut("usage: lead update <id> stage|probability_pct|notes|won_lost_reason <value>")
		}
		id, field, val := args[1], args[2], args[3]
		allowed := map[string]bool{"stage": true, "probability_pct": true, "notes": true, "won_lost_reason": true, "estimated_value": true}
		if !allowed[field] {
			errOut("cannot update: " + field)
		}
		exec(fmt.Sprintf("UPDATE leads SET %s=?, last_activity=?, updated_at=? WHERE id=? AND org_id=?", field),
			val, vnNowISO(), vnNowISO(), id, org)
		// Recalculate weighted value if probability changed
		if field == "probability_pct" || field == "estimated_value" {
			exec("UPDATE leads SET weighted_value = estimated_value * probability_pct / 100 WHERE id=?", id)
		}
		okOut(map[string]interface{}{"updated": id, "field": field})

	case "pipeline":
		rows, _ := queryRows(`SELECT stage, COUNT(*) as count, COALESCE(SUM(estimated_value),0) as total_value,
			COALESCE(SUM(weighted_value),0) as weighted_total
			FROM leads WHERE org_id=? AND stage NOT IN ('won','lost')
			GROUP BY stage ORDER BY CASE stage WHEN 'new' THEN 1 WHEN 'contacted' THEN 2 WHEN 'qualified' THEN 3
			WHEN 'proposal' THEN 4 WHEN 'negotiation' THEN 5 END`, org)
		// Total pipeline
		totalRow, _ := queryOne(`SELECT COUNT(*) as total_leads, COALESCE(SUM(estimated_value),0) as total_value,
			COALESCE(SUM(weighted_value),0) as weighted_total
			FROM leads WHERE org_id=? AND stage NOT IN ('won','lost')`, org)
		okOut(map[string]interface{}{"stages": rows, "summary": totalRow})

	default:
		errOut("unknown lead command: " + args[0])
	}
}

func cmdQuote(args []string) {
	if len(args) == 0 {
		errOut("usage: quote create|list|update")
	}
	org := defaultOrgID()
	switch args[0] {
	case "create":
		// quote create <contact_id> <items_json> [valid_days] [lead_id]
		if len(args) < 3 {
			errOut("usage: quote create <contact_id> <items_json> [valid_days] [lead_id]")
		}
		validDays := 30
		if len(args) > 3 {
			validDays, _ = strconv.Atoi(args[3])
		}
		leadID := ""
		if len(args) > 4 {
			leadID = args[4]
		}

		id := newID()
		qnum := "QT-" + vnNow().Format("060102") + "-" + id[:6]
		validUntil := vnNow().AddDate(0, 0, validDays).Format("2006-01-02")

		// Parse items to calculate totals (items_json is already JSON)
		subtotal := int64(0) // simplified, real impl would parse items
		vatTotal := int64(float64(subtotal) * 0.1)
		grand := subtotal + vatTotal

		_, err := exec(`INSERT INTO quotations (id,org_id,quote_number,contact_id,lead_id,items,subtotal,vat_total,grand_total,status,valid_until,created_by,created_at)
			VALUES (?,?,?,?,NULLIF(?,''),?,?,?,?,?,?,?,?)`,
			id, org, qnum, args[1], leadID, args[2], subtotal, vatTotal, grand, "draft", validUntil, "system", vnNowISO())
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"id": id, "quote_number": qnum, "valid_until": validUntil})

	case "list":
		rows, _ := queryRows(`SELECT q.id,q.quote_number,q.grand_total,q.status,q.valid_until,c.full_name as contact
			FROM quotations q LEFT JOIN contacts c ON q.contact_id=c.id WHERE q.org_id=? ORDER BY q.created_at DESC LIMIT 50`, org)
		okOut(map[string]interface{}{"quotations": rows, "count": len(rows)})

	case "update":
		if len(args) < 4 {
			errOut("usage: quote update <id> status <draft|sent|accepted|rejected>")
		}
		exec("UPDATE quotations SET status=? WHERE id=? AND org_id=?", args[3], args[1], org)
		okOut(map[string]interface{}{"updated": args[1], "status": args[3]})

	default:
		errOut("unknown quote command: " + args[0])
	}
}

func cmdOrder(args []string) {
	if len(args) == 0 {
		errOut("usage: order add|list|update")
	}
	org := defaultOrgID()
	switch args[0] {
	case "add":
		// order add <contact_id> <items_json> <total> [payment_terms] [delivery_date]
		if len(args) < 4 {
			errOut("usage: order add <contact_id> <items_json> <total> [payment_terms] [delivery_date]")
		}
		total := parseVND(args[3])
		terms := "cod"
		if len(args) > 4 {
			terms = args[4]
		}
		delivery := ""
		if len(args) > 5 {
			delivery = args[5]
		}
		id := newID()
		onum := "SO-" + vnNow().Format("060102") + "-" + id[:6]
		now := vnNowISO()
		_, err := exec(`INSERT INTO orders (id,org_id,order_number,contact_id,items,total,status,order_date,delivery_date,payment_terms,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?,?,NULLIF(?,''),?,?,?)`,
			id, org, onum, args[1], args[2], total, "confirmed", vnToday(), delivery, terms, now, now)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"id": id, "order_number": onum, "total": total})

	case "list":
		status := "all"
		if len(args) > 1 {
			status = args[1]
		}
		q := `SELECT o.id,o.order_number,o.total,o.status,o.order_date,o.delivery_date,c.full_name as contact
			FROM orders o LEFT JOIN contacts c ON o.contact_id=c.id WHERE o.org_id=?`
		qargs := []interface{}{org}
		if status != "all" {
			q += " AND o.status=?"
			qargs = append(qargs, status)
		}
		q += " ORDER BY o.created_at DESC LIMIT 50"
		rows, _ := queryRows(q, qargs...)
		okOut(map[string]interface{}{"orders": rows, "count": len(rows)})

	case "update":
		if len(args) < 4 {
			errOut("usage: order update <id> status <confirmed|processing|shipped|delivered|completed|cancelled>")
		}
		now := vnNowISO()
		exec("UPDATE orders SET status=?, updated_at=? WHERE id=? AND org_id=?", args[3], now, args[1], org)
		if args[3] == "delivered" {
			exec("UPDATE orders SET delivered_at=? WHERE id=?", now, args[1])
		}
		okOut(map[string]interface{}{"updated": args[1], "status": args[3]})

	default:
		errOut("unknown order command: " + args[0])
	}
}
