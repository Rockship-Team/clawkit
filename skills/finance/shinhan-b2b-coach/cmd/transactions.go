package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

var validTxnTypes = map[string]bool{
	"sale": true, "purchase": true, "salary": true, "tax": true,
	"rent": true, "utility": true, "other": true,
}

var validDirections = map[string]bool{
	"in": true, "out": true,
}

var validCategories = map[string]bool{
	"revenue": true, "cogs": true, "opex": true, "payroll": true,
	"tax": true, "rent": true, "other": true,
}

func cmdTxn(args []string) {
	if len(args) == 0 {
		errOut("usage: txn <add|list|report|import>")
	}

	switch args[0] {
	case "add":
		cmdTxnAdd(args[1:])
	case "list":
		cmdTxnList(args[1:])
	case "report":
		cmdTxnReport(args[1:])
	case "import":
		cmdTxnImport(args[1:])
	default:
		errOut("unknown txn subcommand: " + args[0])
	}
}

func cmdTxnAdd(args []string) {
	if len(args) < 6 {
		errOut("usage: txn add <date> <type> <direction> <counterparty> <amount> <category> [invoice_number] [due_date] [note]")
	}

	date := args[0]
	txnType := strings.ToLower(args[1])
	direction := strings.ToLower(args[2])
	counterparty := args[3]
	amount := parseVND(args[4])
	category := strings.ToLower(args[5])

	invoiceNumber := ""
	if len(args) > 6 {
		invoiceNumber = args[6]
	}
	dueDate := ""
	if len(args) > 7 {
		dueDate = args[7]
	}
	note := ""
	if len(args) > 8 {
		note = strings.Join(args[8:], " ")
	}

	if !validTxnTypes[txnType] {
		errOut(fmt.Sprintf("invalid type '%s'. Valid: sale, purchase, salary, tax, rent, utility, other", txnType))
	}
	if !validDirections[direction] {
		errOut(fmt.Sprintf("invalid direction '%s'. Valid: in, out", direction))
	}
	if !validCategories[category] {
		errOut(fmt.Sprintf("invalid category '%s'. Valid: revenue, cogs, opex, payroll, tax, rent, other", category))
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	now := vnNowISO()
	result, err := exec(db,
		`INSERT INTO business_transactions (company_id, date, type, direction, counterparty, amount, category, invoice_number, due_date, note, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'recorded', ?)`,
		companyID, date, txnType, direction, counterparty, amount, category, invoiceNumber, dueDate, note, now,
	)
	if err != nil {
		errOut("failed to add transaction: " + err.Error())
	}

	id, _ := result.LastInsertId()

	row, err := queryOne(db, "SELECT * FROM business_transactions WHERE id = ?", id)
	if err != nil {
		errOut("failed to read back transaction: " + err.Error())
	}

	okOut(map[string]interface{}{
		"message":     "Transaction recorded",
		"transaction": row,
	})
}

func cmdTxnList(args []string) {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	query := "SELECT * FROM business_transactions WHERE company_id = ?"
	var qargs []interface{}
	qargs = append(qargs, companyID)

	// Parse optional filters: period, direction, category
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}
		lower := strings.ToLower(arg)
		if validDirections[lower] {
			query += " AND direction = ?"
			qargs = append(qargs, lower)
		} else if validCategories[lower] {
			query += " AND category = ?"
			qargs = append(qargs, lower)
		} else if len(arg) == 7 && arg[4] == '-' {
			// YYYY-MM period filter
			query += " AND date LIKE ?"
			qargs = append(qargs, arg+"%")
		}
	}

	query += " ORDER BY date DESC LIMIT 50"

	rows, err := queryRows(db, query, qargs...)
	if err != nil {
		errOut("failed to list transactions: " + err.Error())
	}

	jsonOut(map[string]interface{}{
		"count":        len(rows),
		"transactions": rows,
	})
}

func cmdTxnReport(args []string) {
	if len(args) < 1 {
		errOut("usage: txn report <YYYY-MM>")
	}

	period := args[0]
	if len(period) != 7 || period[4] != '-' {
		errOut("period must be YYYY-MM format, e.g. 2026-04")
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	dateLike := period + "%"

	// Revenue
	revRow, err := queryOne(db,
		"SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions WHERE company_id=? AND date LIKE ? AND direction='in' AND category='revenue'",
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	revenue := toInt64(revRow["total"])

	// COGS
	cogsRow, err := queryOne(db,
		"SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions WHERE company_id=? AND date LIKE ? AND direction='out' AND category='cogs'",
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	cogs := toInt64(cogsRow["total"])

	grossProfit := revenue - cogs
	grossMarginPct := 0.0
	if revenue > 0 {
		grossMarginPct = float64(grossProfit) * 100.0 / float64(revenue)
	}

	// Operating expenses: opex + rent + payroll + tax
	opexRow, err := queryOne(db,
		"SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions WHERE company_id=? AND date LIKE ? AND direction='out' AND category IN ('opex','rent','payroll','tax')",
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	opex := toInt64(opexRow["total"])

	netProfit := grossProfit - opex
	netMarginPct := 0.0
	if revenue > 0 {
		netMarginPct = float64(netProfit) * 100.0 / float64(revenue)
	}

	// Transaction count
	cntRow, err := queryOne(db,
		"SELECT COUNT(*) AS cnt FROM business_transactions WHERE company_id=? AND date LIKE ?",
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	txnCount := toInt64(cntRow["cnt"])

	jsonOut(map[string]interface{}{
		"period":             period,
		"revenue":            revenue,
		"cogs":               cogs,
		"gross_profit":       grossProfit,
		"gross_margin_pct":   fmt.Sprintf("%.1f", grossMarginPct),
		"operating_expenses": opex,
		"net_profit":         netProfit,
		"net_margin_pct":     fmt.Sprintf("%.1f", netMarginPct),
		"transaction_count":  txnCount,
	})
}

func cmdTxnImport(args []string) {
	if len(args) < 1 {
		errOut("usage: txn import <csv_path>")
	}

	csvPath := args[0]
	f, err := os.Open(csvPath)
	if err != nil {
		errOut("cannot open CSV file: " + err.Error())
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		errOut("failed to parse CSV: " + err.Error())
	}

	if len(records) < 2 {
		errOut("CSV file must have a header row and at least one data row")
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	now := vnNowISO()
	imported := 0

	// Skip header row (date,type,direction,counterparty,amount,category)
	for i, rec := range records[1:] {
		if len(rec) < 6 {
			fmt.Fprintf(os.Stderr, "skipping row %d: expected 6 columns, got %d\n", i+2, len(rec))
			continue
		}

		date := strings.TrimSpace(rec[0])
		txnType := strings.ToLower(strings.TrimSpace(rec[1]))
		direction := strings.ToLower(strings.TrimSpace(rec[2]))
		counterparty := strings.TrimSpace(rec[3])
		amount := parseVND(strings.TrimSpace(rec[4]))
		category := strings.ToLower(strings.TrimSpace(rec[5]))

		if !validTxnTypes[txnType] || !validDirections[direction] || !validCategories[category] {
			fmt.Fprintf(os.Stderr, "skipping row %d: invalid type/direction/category\n", i+2)
			continue
		}

		_, err := exec(db,
			`INSERT INTO business_transactions (company_id, date, type, direction, counterparty, amount, category, status, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 'recorded', ?)`,
			companyID, date, txnType, direction, counterparty, amount, category, now,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error inserting row %d: %s\n", i+2, err.Error())
			continue
		}
		imported++
	}

	okOut(map[string]interface{}{
		"message":  fmt.Sprintf("Imported %d transactions from %s", imported, csvPath),
		"imported": imported,
		"total":    len(records) - 1,
	})
}

// toInt64 converts an interface{} (from SQL query result) to int64.
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case int:
		return int64(val)
	case string:
		n, _ := fmt.Sscanf(val, "%d", new(int64))
		if n > 0 {
			return 0 // fallback
		}
		return 0
	default:
		return 0
	}
}
