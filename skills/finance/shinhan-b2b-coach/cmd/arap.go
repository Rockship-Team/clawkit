package main

import (
	"fmt"
	"strconv"
	"strings"
)

// ---------- Accounts Receivable ----------

func cmdAR(args []string) {
	if len(args) == 0 {
		errOut("usage: ar <add|list|aging|score|remind|pay>")
	}

	switch args[0] {
	case "add":
		cmdARAdd(args[1:])
	case "list":
		cmdARList(args[1:])
	case "aging":
		cmdARAging()
	case "score":
		cmdARScore(args[1:])
	case "remind":
		cmdARRemind(args[1:])
	case "pay":
		cmdARPay(args[1:])
	default:
		errOut("unknown ar subcommand: " + args[0])
	}
}

func cmdARAdd(args []string) {
	if len(args) < 3 {
		errOut("usage: ar add <customer> <amount> <due_date> [invoice_number] [issued_date]")
	}

	customer := args[0]
	amount := parseVND(args[1])
	dueDate := args[2]

	invoiceNumber := ""
	if len(args) > 3 {
		invoiceNumber = args[3]
	}
	issuedDate := vnToday()
	if len(args) > 4 {
		issuedDate = args[4]
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	now := vnNowISO()
	result, err := exec(db,
		`INSERT INTO receivables (company_id, customer_name, amount, due_date, invoice_number, issued_date, status, collection_probability, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'outstanding', 100, ?)`,
		companyID, customer, amount, dueDate, invoiceNumber, issuedDate, now,
	)
	if err != nil {
		errOut("failed to add receivable: " + err.Error())
	}

	id, _ := result.LastInsertId()

	row, err := queryOne(db, "SELECT * FROM receivables WHERE id = ?", id)
	if err != nil {
		errOut("failed to read back receivable: " + err.Error())
	}

	okOut(map[string]interface{}{
		"message":    fmt.Sprintf("Receivable from '%s' added", customer),
		"receivable": row,
	})
}

func cmdARList(args []string) {
	filter := "outstanding"
	if len(args) > 0 {
		filter = strings.ToLower(args[0])
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	today := vnToday()
	var query string
	var qargs []interface{}

	switch filter {
	case "outstanding":
		query = `SELECT *, CASE WHEN due_date < ? THEN CAST(julianday(?) - julianday(due_date) AS INTEGER) ELSE 0 END AS days_overdue
			FROM receivables WHERE company_id = ? AND status = 'outstanding' ORDER BY due_date`
		qargs = []interface{}{today, today, companyID}
	case "overdue":
		query = `SELECT *, CAST(julianday(?) - julianday(due_date) AS INTEGER) AS days_overdue
			FROM receivables WHERE company_id = ? AND status = 'outstanding' AND due_date < ? ORDER BY due_date`
		qargs = []interface{}{today, companyID, today}
	case "paid":
		query = `SELECT * FROM receivables WHERE company_id = ? AND status = 'paid' ORDER BY paid_date DESC`
		qargs = []interface{}{companyID}
	case "all":
		query = `SELECT *, CASE WHEN status = 'outstanding' AND due_date < ? THEN CAST(julianday(?) - julianday(due_date) AS INTEGER) ELSE 0 END AS days_overdue
			FROM receivables WHERE company_id = ? ORDER BY due_date`
		qargs = []interface{}{today, today, companyID}
	default:
		errOut("invalid filter: " + filter + ". Valid: outstanding, overdue, paid, all")
	}

	rows, err := queryRows(db, query, qargs...)
	if err != nil {
		errOut("failed to list receivables: " + err.Error())
	}

	jsonOut(map[string]interface{}{
		"filter":      filter,
		"count":       len(rows),
		"receivables": rows,
	})
}

func cmdARAging() {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	today := vnToday()

	// Query outstanding receivables with days overdue
	query := `SELECT
		CAST(julianday(?) - julianday(due_date) AS INTEGER) AS days_overdue,
		amount
		FROM receivables
		WHERE company_id = ? AND status = 'outstanding'`

	rows, err := queryRows(db, query, today, companyID)
	if err != nil {
		errOut("failed to query aging: " + err.Error())
	}

	type bucket struct {
		Count int64 `json:"count"`
		Total int64 `json:"total"`
	}

	current := bucket{} // 0-30 days (not yet overdue or overdue <= 30)
	days31_60 := bucket{}
	days61_90 := bucket{}
	over90 := bucket{}
	totalOutstanding := int64(0)

	for _, row := range rows {
		days := toInt64(row["days_overdue"])
		amt := toInt64(row["amount"])
		totalOutstanding += amt

		switch {
		case days <= 30:
			current.Count++
			current.Total += amt
		case days <= 60:
			days31_60.Count++
			days31_60.Total += amt
		case days <= 90:
			days61_90.Count++
			days61_90.Total += amt
		default:
			over90.Count++
			over90.Total += amt
		}
	}

	jsonOut(map[string]interface{}{
		"as_of": today,
		"buckets": map[string]interface{}{
			"current_0_30": map[string]interface{}{"count": current.Count, "total": current.Total},
			"days_31_60":   map[string]interface{}{"count": days31_60.Count, "total": days31_60.Total},
			"days_61_90":   map[string]interface{}{"count": days61_90.Count, "total": days61_90.Total},
			"over_90":      map[string]interface{}{"count": over90.Count, "total": over90.Total},
		},
		"total_outstanding": totalOutstanding,
		"total_invoices":    len(rows),
	})
}

func cmdARScore(args []string) {
	if len(args) < 1 {
		errOut("usage: ar score <customer>")
	}

	customer := args[0]
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	today := vnToday()

	rows, err := queryRows(db,
		`SELECT *, CASE
			WHEN status = 'paid' AND paid_date IS NOT NULL THEN CAST(julianday(paid_date) - julianday(due_date) AS INTEGER)
			WHEN status = 'outstanding' AND due_date < ? THEN CAST(julianday(?) - julianday(due_date) AS INTEGER)
			ELSE 0
		END AS calc_days
		FROM receivables WHERE company_id = ? AND customer_name = ?`,
		today, today, companyID, customer,
	)
	if err != nil {
		errOut("failed to query customer receivables: " + err.Error())
	}

	if len(rows) == 0 {
		errOut(fmt.Sprintf("no receivables found for customer '%s'", customer))
	}

	totalInvoices := len(rows)
	paidCount := 0
	paidOnTime := 0
	totalDaysToPay := int64(0)
	lateCount := 0

	for _, row := range rows {
		status, _ := row["status"].(string)
		calcDays := toInt64(row["calc_days"])

		if status == "paid" {
			paidCount++
			if calcDays <= 0 {
				paidOnTime++
			} else {
				lateCount++
			}
			totalDaysToPay += calcDays
		} else if status == "outstanding" {
			if calcDays > 0 {
				lateCount++
			}
		}
	}

	paidOnTimePct := 0.0
	avgDaysToPay := 0.0
	if paidCount > 0 {
		paidOnTimePct = float64(paidOnTime) * 100.0 / float64(paidCount)
		avgDaysToPay = float64(totalDaysToPay) / float64(paidCount)
	}

	lateRatio := float64(lateCount) / float64(totalInvoices)
	collectionProb := int(100.0 - lateRatio*50.0)
	if collectionProb < 0 {
		collectionProb = 0
	}

	// Update collection_probability for outstanding receivables of this customer
	_, _ = exec(db,
		`UPDATE receivables SET collection_probability = ? WHERE company_id = ? AND customer_name = ? AND status = 'outstanding'`,
		collectionProb, companyID, customer,
	)

	jsonOut(map[string]interface{}{
		"customer":               customer,
		"total_invoices":         totalInvoices,
		"paid_count":             paidCount,
		"paid_on_time_pct":       fmt.Sprintf("%.1f", paidOnTimePct),
		"avg_days_to_pay":        fmt.Sprintf("%.1f", avgDaysToPay),
		"late_count":             lateCount,
		"late_frequency":         fmt.Sprintf("%.1f%%", lateRatio*100),
		"collection_probability": collectionProb,
	})
}

func cmdARRemind(args []string) {
	if len(args) < 1 {
		errOut("usage: ar remind <id>")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		errOut("invalid id: " + args[0])
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	today := vnToday()

	row, err := queryOne(db,
		`SELECT *, CAST(julianday(?) - julianday(due_date) AS INTEGER) AS days_overdue
		FROM receivables WHERE id = ? AND company_id = ?`,
		today, id, companyID,
	)
	if err != nil {
		errOut("failed to query receivable: " + err.Error())
	}
	if row == nil {
		errOut(fmt.Sprintf("receivable #%d not found", id))
	}

	customer, _ := row["customer_name"].(string)
	invoice, _ := row["invoice_number"].(string)
	amount := toInt64(row["amount"])
	daysOverdue := toInt64(row["days_overdue"])

	if invoice == "" {
		invoice = fmt.Sprintf("#%d", id)
	}

	reminder := fmt.Sprintf(
		"Kinh gui %s, hoa don %s so tien %d VND da qua han %d ngay. Vui long thanh toan.",
		customer, invoice, amount, daysOverdue,
	)

	jsonOut(map[string]interface{}{
		"id":           id,
		"customer":     customer,
		"invoice":      invoice,
		"amount":       amount,
		"days_overdue": daysOverdue,
		"reminder":     reminder,
	})
}

func cmdARPay(args []string) {
	if len(args) < 1 {
		errOut("usage: ar pay <id> [paid_date]")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		errOut("invalid id: " + args[0])
	}

	paidDate := vnToday()
	if len(args) > 1 {
		paidDate = args[1]
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	// Check that receivable exists and is outstanding
	row, err := queryOne(db,
		`SELECT * FROM receivables WHERE id = ? AND company_id = ?`, id, companyID)
	if err != nil {
		errOut("failed to query receivable: " + err.Error())
	}
	if row == nil {
		errOut(fmt.Sprintf("receivable #%d not found", id))
	}

	dueDate, _ := row["due_date"].(string)

	// Calculate days overdue (positive = late, negative = early)
	daysRow, err := queryOne(db,
		`SELECT CAST(julianday(?) - julianday(?) AS INTEGER) AS days_overdue`, paidDate, dueDate)
	if err != nil {
		errOut("failed to calculate days: " + err.Error())
	}
	daysOverdue := toInt64(daysRow["days_overdue"])
	if daysOverdue < 0 {
		daysOverdue = 0
	}

	_, err = exec(db,
		`UPDATE receivables SET status = 'paid', paid_date = ?, days_overdue = ? WHERE id = ? AND company_id = ?`,
		paidDate, daysOverdue, id, companyID,
	)
	if err != nil {
		errOut("failed to update receivable: " + err.Error())
	}

	updated, err := queryOne(db, "SELECT * FROM receivables WHERE id = ?", id)
	if err != nil {
		errOut("failed to read back receivable: " + err.Error())
	}

	okOut(map[string]interface{}{
		"message":      fmt.Sprintf("Receivable #%d marked as paid", id),
		"days_overdue": daysOverdue,
		"receivable":   updated,
	})
}

// ---------- Accounts Payable ----------

func cmdAP(args []string) {
	if len(args) == 0 {
		errOut("usage: ap <add|list|schedule|discount-roi|pay>")
	}

	switch args[0] {
	case "add":
		cmdAPAdd(args[1:])
	case "list":
		cmdAPList(args[1:])
	case "schedule":
		cmdAPSchedule()
	case "discount-roi":
		cmdAPDiscountROI(args[1:])
	case "pay":
		cmdAPPay(args[1:])
	default:
		errOut("unknown ap subcommand: " + args[0])
	}
}

func cmdAPAdd(args []string) {
	if len(args) < 3 {
		errOut("usage: ap add <vendor> <amount> <due_date> [invoice_number] [early_pay_discount_pct] [early_pay_deadline]")
	}

	vendor := args[0]
	amount := parseVND(args[1])
	dueDate := args[2]

	invoiceNumber := ""
	if len(args) > 3 {
		invoiceNumber = args[3]
	}
	var earlyPayDiscountPct int64
	if len(args) > 4 {
		earlyPayDiscountPct, _ = strconv.ParseInt(args[4], 10, 64)
	}
	earlyPayDeadline := ""
	if len(args) > 5 {
		earlyPayDeadline = args[5]
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	now := vnNowISO()
	result, err := exec(db,
		`INSERT INTO payables (company_id, vendor_name, amount, due_date, invoice_number, issued_date, status, early_pay_discount_pct, early_pay_deadline, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'outstanding', ?, ?, ?)`,
		companyID, vendor, amount, dueDate, invoiceNumber, vnToday(), earlyPayDiscountPct, earlyPayDeadline, now,
	)
	if err != nil {
		errOut("failed to add payable: " + err.Error())
	}

	id, _ := result.LastInsertId()

	row, err := queryOne(db, "SELECT * FROM payables WHERE id = ?", id)
	if err != nil {
		errOut("failed to read back payable: " + err.Error())
	}

	okOut(map[string]interface{}{
		"message": fmt.Sprintf("Payable to '%s' added", vendor),
		"payable": row,
	})
}

func cmdAPList(args []string) {
	filter := "outstanding"
	if len(args) > 0 {
		filter = strings.ToLower(args[0])
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	today := vnToday()
	var query string
	var qargs []interface{}

	switch filter {
	case "outstanding":
		query = `SELECT *, CASE WHEN due_date < ? THEN CAST(julianday(?) - julianday(due_date) AS INTEGER) ELSE 0 END AS days_overdue
			FROM payables WHERE company_id = ? AND status = 'outstanding' ORDER BY due_date`
		qargs = []interface{}{today, today, companyID}
	case "overdue":
		query = `SELECT *, CAST(julianday(?) - julianday(due_date) AS INTEGER) AS days_overdue
			FROM payables WHERE company_id = ? AND status = 'outstanding' AND due_date < ? ORDER BY due_date`
		qargs = []interface{}{today, companyID, today}
	case "paid":
		query = `SELECT * FROM payables WHERE company_id = ? AND status = 'paid' ORDER BY paid_date DESC`
		qargs = []interface{}{companyID}
	case "all":
		query = `SELECT *, CASE WHEN status = 'outstanding' AND due_date < ? THEN CAST(julianday(?) - julianday(due_date) AS INTEGER) ELSE 0 END AS days_overdue
			FROM payables WHERE company_id = ? ORDER BY due_date`
		qargs = []interface{}{today, today, companyID}
	default:
		errOut("invalid filter: " + filter + ". Valid: outstanding, overdue, paid, all")
	}

	rows, err := queryRows(db, query, qargs...)
	if err != nil {
		errOut("failed to list payables: " + err.Error())
	}

	jsonOut(map[string]interface{}{
		"filter":   filter,
		"count":    len(rows),
		"payables": rows,
	})
}

func cmdAPSchedule() {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	today := vnToday()

	rows, err := queryRows(db,
		`SELECT *, CAST(julianday(due_date) - julianday(?) AS INTEGER) AS days_until_due
		FROM payables
		WHERE company_id = ? AND status = 'outstanding'
		ORDER BY due_date`,
		today, companyID,
	)
	if err != nil {
		errOut("failed to query payables: " + err.Error())
	}

	urgent := []map[string]interface{}{}   // due within 7 days
	upcoming := []map[string]interface{}{} // 8-30 days
	later := []map[string]interface{}{}    // > 30 days

	for _, row := range rows {
		daysUntil := toInt64(row["days_until_due"])

		// Flag early-pay discount availability
		discountPct := toInt64(row["early_pay_discount_pct"])
		earlyDeadline, _ := row["early_pay_deadline"].(string)
		hasDiscount := discountPct > 0 && earlyDeadline >= today
		row["early_pay_available"] = hasDiscount

		switch {
		case daysUntil <= 7:
			urgent = append(urgent, row)
		case daysUntil <= 30:
			upcoming = append(upcoming, row)
		default:
			later = append(later, row)
		}
	}

	jsonOut(map[string]interface{}{
		"as_of":    today,
		"urgent":   map[string]interface{}{"label": "Due within 7 days", "count": len(urgent), "items": urgent},
		"upcoming": map[string]interface{}{"label": "Due in 8-30 days", "count": len(upcoming), "items": upcoming},
		"later":    map[string]interface{}{"label": "Due after 30 days", "count": len(later), "items": later},
		"total":    len(rows),
	})
}

func cmdAPDiscountROI(args []string) {
	if len(args) < 1 {
		errOut("usage: ap discount-roi <id>")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		errOut("invalid id: " + args[0])
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	today := vnToday()

	row, err := queryOne(db,
		`SELECT *,
			CAST(julianday(due_date) - julianday(?) AS INTEGER) AS days_until_due,
			CAST(julianday(due_date) - julianday(early_pay_deadline) AS INTEGER) AS days_early
		FROM payables WHERE id = ? AND company_id = ?`,
		today, id, companyID,
	)
	if err != nil {
		errOut("failed to query payable: " + err.Error())
	}
	if row == nil {
		errOut(fmt.Sprintf("payable #%d not found", id))
	}

	discountPct := toInt64(row["early_pay_discount_pct"])
	if discountPct == 0 {
		errOut(fmt.Sprintf("payable #%d has no early-pay discount", id))
	}

	daysEarly := toInt64(row["days_early"])
	if daysEarly <= 0 {
		daysEarly = 1 // avoid division by zero
	}

	amount := toInt64(row["amount"])
	savings := amount * discountPct / 100
	annualizedROI := (float64(discountPct) / float64(daysEarly)) * 365.0

	recommend := "Khong nen"
	if annualizedROI > 20.0 {
		recommend = "Nen"
	}

	advice := fmt.Sprintf(
		"Chiet khau %d%%, tra som %d ngay, ROI quy nam = %.1f%%. %s lay.",
		discountPct, daysEarly, annualizedROI, recommend,
	)

	jsonOut(map[string]interface{}{
		"id":             id,
		"vendor":         row["vendor_name"],
		"amount":         amount,
		"discount_pct":   discountPct,
		"days_early":     daysEarly,
		"savings":        savings,
		"annualized_roi": fmt.Sprintf("%.1f", annualizedROI),
		"recommendation": recommend,
		"advice":         advice,
	})
}

func cmdAPPay(args []string) {
	if len(args) < 1 {
		errOut("usage: ap pay <id> [paid_date]")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		errOut("invalid id: " + args[0])
	}

	paidDate := vnToday()
	if len(args) > 1 {
		paidDate = args[1]
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	row, err := queryOne(db,
		`SELECT * FROM payables WHERE id = ? AND company_id = ?`, id, companyID)
	if err != nil {
		errOut("failed to query payable: " + err.Error())
	}
	if row == nil {
		errOut(fmt.Sprintf("payable #%d not found", id))
	}

	_, err = exec(db,
		`UPDATE payables SET status = 'paid', paid_date = ? WHERE id = ? AND company_id = ?`,
		paidDate, id, companyID,
	)
	if err != nil {
		errOut("failed to update payable: " + err.Error())
	}

	updated, err := queryOne(db, "SELECT * FROM payables WHERE id = ?", id)
	if err != nil {
		errOut("failed to read back payable: " + err.Error())
	}

	okOut(map[string]interface{}{
		"message": fmt.Sprintf("Payable #%d marked as paid", id),
		"payable": updated,
	})
}
