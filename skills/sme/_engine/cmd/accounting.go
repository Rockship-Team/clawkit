package main

import (
	"fmt"
	"strconv"
	"strings"
)

func cmdInvoice(args []string) {
	if len(args) == 0 {
		errOut("usage: invoice add|list|get|update")
	}
	org := defaultOrgID()
	switch args[0] {
	case "add":
		// invoice add <direction> <contact_name> <amount> <type> [due_date] [notes]
		if len(args) < 5 {
			errOut("usage: invoice add <inbound|outbound> <contact_name> <amount> <type> [due_date] [notes]")
		}
		dir, cname := args[1], args[2]
		amt := parseVND(args[3])
		itype := args[4]
		due := ""
		if len(args) > 5 {
			due = args[5]
		}
		notes := ""
		if len(args) > 6 {
			notes = args[6]
		}
		vat := int64(float64(amt) * 0.1)
		total := amt + vat
		id := newID()
		now := vnNowISO()
		_, err := exec(`INSERT INTO invoices (id,org_id,direction,invoice_type,status,subtotal,vat_rate,vat_amount,total,amount_due,seller_name,due_date,notes,issued_date,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, org, dir, itype, "draft", amt, 10.0, vat, total, total, cname, due, notes, vnToday(), now, now)
		if err != nil {
			errOut("insert failed: " + err.Error())
		}
		okOut(map[string]interface{}{"id": id, "direction": dir, "contact": cname, "total": total, "status": "draft"})

	case "list":
		filter := "all"
		if len(args) > 1 {
			filter = args[1]
		}
		q := "SELECT id,invoice_number,direction,invoice_type,status,seller_name,total,amount_due,due_date,created_at FROM invoices WHERE org_id=?"
		qargs := []interface{}{org}
		switch filter {
		case "overdue":
			q += " AND status IN ('confirmed','partial') AND due_date < ?"
			qargs = append(qargs, vnToday())
		case "draft":
			q += " AND status='draft'"
		case "inbound":
			q += " AND direction='inbound'"
		case "outbound":
			q += " AND direction='outbound'"
		case "unpaid":
			q += " AND amount_due > 0 AND status NOT IN ('cancelled','paid')"
		}
		q += " ORDER BY created_at DESC LIMIT 50"
		rows, err := queryRows(q, qargs...)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"invoices": rows, "count": len(rows), "filter": filter})

	case "get":
		if len(args) < 2 {
			errOut("usage: invoice get <id>")
		}
		row, err := queryOne("SELECT * FROM invoices WHERE id=? AND org_id=?", args[1], org)
		if err != nil {
			errOut(err.Error())
		}
		if row == nil {
			errOut("invoice not found")
		}
		items, _ := queryRows("SELECT * FROM invoice_items WHERE invoice_id=?", args[1])
		row["items"] = items
		okOut(map[string]interface{}{"invoice": row})

	case "update":
		if len(args) < 4 {
			errOut("usage: invoice update <id> <field> <value>")
		}
		id, field, val := args[1], args[2], args[3]
		allowed := map[string]bool{"status": true, "amount_paid": true, "notes": true, "invoice_number": true, "due_date": true}
		if !allowed[field] {
			errOut("cannot update field: " + field)
		}
		q := fmt.Sprintf("UPDATE invoices SET %s=?, updated_at=? WHERE id=? AND org_id=?", field)
		_, err := exec(q, val, vnNowISO(), id, org)
		if err != nil {
			errOut(err.Error())
		}
		// Recalculate amount_due if amount_paid changed
		if field == "amount_paid" {
			exec("UPDATE invoices SET amount_due = total - CAST(? AS INTEGER), status = CASE WHEN total - CAST(? AS INTEGER) <= 0 THEN 'paid' ELSE status END WHERE id=?", val, val, id)
		}
		okOut(map[string]interface{}{"updated": id, "field": field})

	case "ar-aging":
		// Accounts Receivable aging report
		rows, _ := queryRows(`SELECT
			seller_name as contact,
			COUNT(*) as count,
			SUM(amount_due) as total_due,
			MIN(due_date) as oldest_due,
			SUM(CASE WHEN due_date < ? THEN amount_due ELSE 0 END) as overdue_amount
		FROM invoices
		WHERE org_id=? AND direction='outbound' AND amount_due > 0 AND status NOT IN ('cancelled','paid')
		GROUP BY seller_name ORDER BY total_due DESC`, vnToday(), org)
		total := int64(0)
		overdue := int64(0)
		for _, r := range rows {
			if v, ok := r["total_due"].(int64); ok {
				total += v
			}
			if v, ok := r["overdue_amount"].(int64); ok {
				overdue += v
			}
		}
		okOut(map[string]interface{}{"aging": rows, "total_receivable": total, "total_overdue": overdue})

	case "ap-aging":
		rows, _ := queryRows(`SELECT
			seller_name as contact,
			COUNT(*) as count,
			SUM(amount_due) as total_due,
			MIN(due_date) as oldest_due
		FROM invoices
		WHERE org_id=? AND direction='inbound' AND amount_due > 0 AND status NOT IN ('cancelled','paid')
		GROUP BY seller_name ORDER BY total_due DESC`, org)
		total := int64(0)
		for _, r := range rows {
			if v, ok := r["total_due"].(int64); ok {
				total += v
			}
		}
		okOut(map[string]interface{}{"aging": rows, "total_payable": total})

	default:
		errOut("unknown invoice command: " + args[0])
	}
}

func cmdPayment(args []string) {
	if len(args) == 0 {
		errOut("usage: payment add|list")
	}
	org := defaultOrgID()
	switch args[0] {
	case "add":
		// payment add <direction> <amount> <method> [contact] [invoice_id] [notes]
		if len(args) < 4 {
			errOut("usage: payment add <in|out> <amount> <method> [contact] [invoice_id] [notes]")
		}
		dir := args[1]
		amt := parseVND(args[2])
		method := args[3]
		contact := ""
		if len(args) > 4 {
			contact = args[4]
		}
		invID := ""
		if len(args) > 5 {
			invID = args[5]
		}
		notes := ""
		if len(args) > 6 {
			notes = args[6]
		}
		id := newID()
		// Find contact_id by name
		contactID := ""
		if contact != "" {
			row, _ := queryOne("SELECT id FROM contacts WHERE org_id=? AND full_name LIKE ?", org, "%"+contact+"%")
			if row != nil {
				contactID = fmt.Sprintf("%v", row["id"])
			}
		}
		_, err := exec(`INSERT INTO payments (id,org_id,invoice_id,contact_id,direction,amount,method,status,paid_at,notes,created_at)
			VALUES (?,?,NULLIF(?,''),(NULLIF(?,'')  ),?,?,?,?,?,?,?)`,
			id, org, invID, contactID, dir, amt, method, "completed", vnNowISO(), notes, vnNowISO())
		if err != nil {
			errOut(err.Error())
		}
		// Update invoice if linked
		if invID != "" {
			exec("UPDATE invoices SET amount_paid = amount_paid + ?, amount_due = amount_due - ?, updated_at = ? WHERE id = ?",
				amt, amt, vnNowISO(), invID)
			exec("UPDATE invoices SET status = 'paid' WHERE id = ? AND amount_due <= 0", invID)
		}
		okOut(map[string]interface{}{"id": id, "direction": dir, "amount": amt, "method": method})

	case "list":
		q := "SELECT p.id,p.direction,p.amount,p.method,p.status,p.paid_at,p.notes FROM payments p WHERE p.org_id=? ORDER BY p.paid_at DESC LIMIT 50"
		rows, _ := queryRows(q, org)
		okOut(map[string]interface{}{"payments": rows, "count": len(rows)})

	default:
		errOut("unknown payment command: " + args[0])
	}
}

func cmdBank(args []string) {
	if len(args) == 0 {
		errOut("usage: bank import|reconcile|unmatched")
	}
	org := defaultOrgID()
	switch args[0] {
	case "import":
		// bank import <bank_name> <account> <date> <ref> <description> <amount> [balance]
		if len(args) < 7 {
			errOut("usage: bank import <bank> <account> <date> <ref> <desc> <amount> [balance]")
		}
		id := newID()
		amt := parseVND(args[6])
		bal := int64(0)
		if len(args) > 7 {
			bal = parseVND(args[7])
		}
		_, err := exec(`INSERT INTO bank_transactions (id,org_id,bank_name,account_number,txn_date,txn_ref,description,amount,balance,match_status,imported_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			id, org, args[1], args[2], args[3], args[4], args[5], amt, bal, "unmatched", vnNowISO())
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"id": id, "amount": amt, "status": "unmatched"})

	case "unmatched":
		rows, _ := queryRows(`SELECT id,bank_name,txn_date,txn_ref,description,amount,balance
			FROM bank_transactions WHERE org_id=? AND match_status='unmatched' ORDER BY txn_date DESC LIMIT 50`, org)
		okOut(map[string]interface{}{"unmatched": rows, "count": len(rows)})

	case "reconcile":
		// Auto-match by amount + date proximity
		unmatched, _ := queryRows("SELECT id,amount,txn_date,description FROM bank_transactions WHERE org_id=? AND match_status='unmatched'", org)
		matched := 0
		for _, txn := range unmatched {
			amt, _ := txn["amount"].(int64)
			date := fmt.Sprintf("%v", txn["txn_date"])
			var inv map[string]interface{}
			if amt > 0 {
				inv, _ = queryOne("SELECT id FROM invoices WHERE org_id=? AND direction='outbound' AND total=? AND status NOT IN ('paid','cancelled') ORDER BY ABS(julianday(due_date)-julianday(?)) LIMIT 1", org, amt, date)
			} else {
				inv, _ = queryOne("SELECT id FROM invoices WHERE org_id=? AND direction='inbound' AND total=? AND status NOT IN ('paid','cancelled') ORDER BY ABS(julianday(due_date)-julianday(?)) LIMIT 1", org, -amt, date)
			}
			if inv != nil {
				exec("UPDATE bank_transactions SET match_status='auto_matched', matched_invoice_id=? WHERE id=?", inv["id"], txn["id"])
				matched++
			}
		}
		okOut(map[string]interface{}{"processed": len(unmatched), "matched": matched})

	default:
		errOut("unknown bank command: " + args[0])
	}
}

func cmdCashflow(args []string) {
	if len(args) == 0 {
		errOut("usage: cashflow weekly|forecast")
	}
	org := defaultOrgID()
	today := vnToday()

	switch args[0] {
	case "weekly":
		// 7-day forward cashflow
		weekEnd := vnNow().AddDate(0, 0, 7).Format("2006-01-02")

		// AR due this week
		arRows, _ := queryRows(`SELECT seller_name,SUM(amount_due) as total FROM invoices
			WHERE org_id=? AND direction='outbound' AND status IN ('confirmed','partial','overdue') AND due_date BETWEEN ? AND ?
			GROUP BY seller_name`, org, today, weekEnd)
		arTotal := int64(0)
		for _, r := range arRows {
			if v, ok := r["total"].(int64); ok {
				arTotal += v
			}
		}

		// AP due this week
		apRows, _ := queryRows(`SELECT seller_name,SUM(amount_due) as total FROM invoices
			WHERE org_id=? AND direction='inbound' AND status IN ('confirmed','partial') AND due_date BETWEEN ? AND ?
			GROUP BY seller_name`, org, today, weekEnd)
		apTotal := int64(0)
		for _, r := range apRows {
			if v, ok := r["total"].(int64); ok {
				apTotal += v
			}
		}

		// Tax due
		taxRows, _ := queryRows(`SELECT tax_type,period_label,amount_due FROM tax_deadlines
			WHERE org_id=? AND status='upcoming' AND deadline_date BETWEEN ? AND ?`, org, today, weekEnd)
		taxTotal := int64(0)
		for _, r := range taxRows {
			if v, ok := r["amount_due"].(int64); ok {
				taxTotal += v
			}
		}

		net := arTotal - apTotal - taxTotal
		var alerts []string
		if net < 0 {
			alerts = append(alerts, fmt.Sprintf("Thieu %d VND trong 7 ngay toi", -net))
		}

		// Overdue AR for collection suggestions
		overdueAR, _ := queryRows(`SELECT seller_name,SUM(amount_due) as total FROM invoices
			WHERE org_id=? AND direction='outbound' AND status='overdue'
			GROUP BY seller_name ORDER BY total DESC LIMIT 3`, org)
		for _, r := range overdueAR {
			alerts = append(alerts, fmt.Sprintf("Thu no: %v con no %v VND", r["seller_name"], r["total"]))
		}

		okOut(map[string]interface{}{
			"period":       today + " → " + weekEnd,
			"inflows":      arRows,
			"outflows_ap":  apRows,
			"outflows_tax": taxRows,
			"total_in":     arTotal,
			"total_out":    apTotal + taxTotal,
			"net":          net,
			"alerts":       alerts,
		})

	case "forecast":
		// 30-day forecast
		days := 30
		if len(args) > 1 {
			days, _ = strconv.Atoi(args[1])
		}
		end := vnNow().AddDate(0, 0, days).Format("2006-01-02")

		arRow, _ := queryOne("SELECT COALESCE(SUM(amount_due),0) as total FROM invoices WHERE org_id=? AND direction='outbound' AND status NOT IN ('cancelled','paid') AND due_date BETWEEN ? AND ?", org, today, end)
		apRow, _ := queryOne("SELECT COALESCE(SUM(amount_due),0) as total FROM invoices WHERE org_id=? AND direction='inbound' AND status NOT IN ('cancelled','paid') AND due_date BETWEEN ? AND ?", org, today, end)
		taxRow, _ := queryOne("SELECT COALESCE(SUM(amount_due),0) as total FROM tax_deadlines WHERE org_id=? AND status='upcoming' AND deadline_date BETWEEN ? AND ?", org, today, end)

		ar := int64(0)
		ap := int64(0)
		tax := int64(0)
		if arRow != nil {
			ar, _ = arRow["total"].(int64)
		}
		if apRow != nil {
			ap, _ = apRow["total"].(int64)
		}
		if taxRow != nil {
			tax, _ = taxRow["total"].(int64)
		}

		okOut(map[string]interface{}{
			"period_days":      days,
			"expected_in":      ar,
			"expected_out":     ap + tax,
			"expected_out_ap":  ap,
			"expected_out_tax": tax,
			"net_forecast":     ar - ap - tax,
		})

	default:
		errOut("unknown cashflow command: " + args[0])
	}
}

func cmdExpense(args []string) {
	if len(args) == 0 {
		errOut("usage: expense add|list|approve")
	}
	org := defaultOrgID()
	switch args[0] {
	case "add":
		if len(args) < 4 {
			errOut("usage: expense add <category> <amount> <description> [submitted_by]")
		}
		id := newID()
		amt := parseVND(args[2])
		by := "system"
		if len(args) > 4 {
			by = args[4]
		}
		_, err := exec(`INSERT INTO expense_claims (id,org_id,submitted_by,category,amount,description,status,created_at)
			VALUES (?,?,?,?,?,?,?,?)`, id, org, by, args[1], amt, args[3], "pending", vnNowISO())
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"id": id, "amount": amt, "status": "pending"})

	case "list":
		status := "all"
		if len(args) > 1 {
			status = args[1]
		}
		q := "SELECT id,category,amount,description,status,created_at FROM expense_claims WHERE org_id=?"
		qargs := []interface{}{org}
		if status != "all" {
			q += " AND status=?"
			qargs = append(qargs, status)
		}
		q += " ORDER BY created_at DESC LIMIT 50"
		rows, _ := queryRows(q, qargs...)
		okOut(map[string]interface{}{"expenses": rows, "count": len(rows)})

	case "approve":
		if len(args) < 2 {
			errOut("usage: expense approve <id>")
		}
		_, err := exec("UPDATE expense_claims SET status='approved', approved_at=? WHERE id=? AND org_id=?",
			vnNowISO(), args[1], org)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"approved": args[1]})

	default:
		errOut("unknown expense command: " + args[0])
	}
}

// Ensure these are used
var _ = strings.TrimSpace
var _ = strconv.Atoi
