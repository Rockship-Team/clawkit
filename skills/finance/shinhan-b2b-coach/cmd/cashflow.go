package main

import (
	"fmt"
	"strconv"
)

func cmdCashflow(args []string) {
	if len(args) == 0 {
		errOut("usage: cashflow <forecast|weekly|gap>")
	}

	switch args[0] {
	case "forecast":
		cmdCashflowForecast(args[1:])
	case "weekly":
		cmdCashflowWeekly()
	case "gap":
		cmdCashflowGap()
	default:
		errOut("unknown cashflow subcommand: " + args[0])
	}
}

func cmdCashflowForecast(args []string) {
	days := 30
	if len(args) > 0 {
		d, err := strconv.Atoi(args[0])
		if err == nil && d > 0 {
			days = d
		}
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	today := vnToday()
	now := vnNowISO()

	// 1. Get company cash_reserve as opening_balance
	companyRow, err := queryOne(db,
		`SELECT COALESCE(cash_reserve, 0) AS cash_reserve FROM company_profile WHERE company_id = ?`, companyID)
	if err != nil {
		errOut("failed to query company: " + err.Error())
	}
	openingBalance := toInt64(companyRow["cash_reserve"])

	// 2. Query outstanding receivables → inflows
	arRows, err := queryRows(db,
		`SELECT amount, COALESCE(collection_probability, 85) AS prob
		FROM receivables
		WHERE company_id = ? AND status = 'outstanding' AND due_date <= date(?, '+' || ? || ' days')`,
		companyID, today, days)
	if err != nil {
		errOut("failed to query receivables: " + err.Error())
	}

	totalARFull := int64(0)
	totalARWeighted := int64(0)
	for _, row := range arRows {
		amt := toInt64(row["amount"])
		prob := toInt64(row["prob"])
		totalARFull += amt
		totalARWeighted += amt * prob / 100
	}

	// 3. Query outstanding payables → outflows
	apRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount), 0) AS total
		FROM payables
		WHERE company_id = ? AND status = 'outstanding' AND due_date <= date(?, '+' || ? || ' days')`,
		companyID, today, days)
	if err != nil {
		errOut("failed to query payables: " + err.Error())
	}
	totalAP := toInt64(apRow["total"])

	// 4. Get avg monthly recurring costs from business_transactions (last 3 months)
	recurringRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount), 0) AS total, COUNT(DISTINCT substr(date, 1, 7)) AS months
		FROM business_transactions
		WHERE company_id = ? AND direction = 'out'
		AND category IN ('payroll', 'rent', 'other')
		AND date >= date(?, '-3 months')`,
		companyID, today)
	if err != nil {
		errOut("failed to query recurring costs: " + err.Error())
	}

	recurringTotal := toInt64(recurringRow["total"])
	months := toInt64(recurringRow["months"])
	if months == 0 {
		months = 1
	}
	monthlyRecurring := recurringTotal / months
	// Pro-rate to forecast period
	recurringForPeriod := monthlyRecurring * int64(days) / 30

	// 5. Calculate period_end
	periodEndRow, err := queryOne(db,
		`SELECT date(?, '+' || ? || ' days') AS period_end`, today, days)
	if err != nil {
		errOut("failed to calc period_end: " + err.Error())
	}
	periodEnd, _ := periodEndRow["period_end"].(string)

	// 6. Create 3 scenarios
	type scenario struct {
		Name           string
		Inflows        int64
		Outflows       int64
		Recurring      int64
		NetCashflow    int64
		ClosingBalance int64
		GapAlert       string
	}

	scenarios := []scenario{
		{
			Name:      "optimistic",
			Inflows:   totalARFull,
			Outflows:  totalAP,
			Recurring: 0,
		},
		{
			Name:      "base",
			Inflows:   totalARFull * 85 / 100,
			Outflows:  totalAP,
			Recurring: recurringForPeriod,
		},
		{
			Name:      "pessimistic",
			Inflows:   totalARFull * 70 / 100,
			Outflows:  totalAP,
			Recurring: recurringForPeriod * 110 / 100,
		},
	}

	results := []map[string]interface{}{}

	for i := range scenarios {
		s := &scenarios[i]
		s.NetCashflow = s.Inflows - s.Outflows - s.Recurring
		s.ClosingBalance = openingBalance + s.NetCashflow
		s.GapAlert = "false"
		if s.ClosingBalance < 0 {
			s.GapAlert = "true"
		}

		// Save to cashflow_forecast table
		_, err := exec(db,
			`INSERT INTO cashflow_forecast
				(company_id, forecast_date, period_start, period_end, scenario,
				 opening_balance, inflows_expected, outflows_expected,
				 net_cashflow, closing_balance, gap_alert, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			companyID, today, today, periodEnd, s.Name,
			openingBalance, s.Inflows, s.Outflows+s.Recurring,
			s.NetCashflow, s.ClosingBalance, s.GapAlert, now,
		)
		if err != nil {
			errOut("failed to save forecast: " + err.Error())
		}

		results = append(results, map[string]interface{}{
			"scenario":        s.Name,
			"opening_balance": openingBalance,
			"inflows":         s.Inflows,
			"outflows":        s.Outflows,
			"recurring_costs": s.Recurring,
			"net_cashflow":    s.NetCashflow,
			"closing_balance": s.ClosingBalance,
			"gap_alert":       s.GapAlert,
		})
	}

	jsonOut(map[string]interface{}{
		"forecast_days": days,
		"period_start":  today,
		"period_end":    periodEnd,
		"scenarios":     results,
	})
}

func cmdCashflowWeekly() {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	today := vnToday()

	// 1. AR due this week (today to today+7)
	arRows, err := queryRows(db,
		`SELECT *, COALESCE(collection_probability, 85) AS prob
		FROM receivables
		WHERE company_id = ? AND status = 'outstanding'
		AND due_date >= ? AND due_date <= date(?, '+7 days')
		ORDER BY due_date`,
		companyID, today, today,
	)
	if err != nil {
		errOut("failed to query AR this week: " + err.Error())
	}

	arTotal := int64(0)
	for _, row := range arRows {
		arTotal += toInt64(row["amount"])
	}

	// 2. AP due this week
	apRows, err := queryRows(db,
		`SELECT *
		FROM payables
		WHERE company_id = ? AND status = 'outstanding'
		AND due_date >= ? AND due_date <= date(?, '+7 days')
		ORDER BY due_date`,
		companyID, today, today,
	)
	if err != nil {
		errOut("failed to query AP this week: " + err.Error())
	}

	apTotal := int64(0)
	for _, row := range apRows {
		apTotal += toInt64(row["amount"])
	}

	// 3. Net for the week
	netWeek := arTotal - apTotal

	// 4. Warnings about customers with low collection_probability (< 80)
	lowScoreRows, err := queryRows(db,
		`SELECT DISTINCT customer_name, collection_probability
		FROM receivables
		WHERE company_id = ? AND status = 'outstanding'
		AND collection_probability < 80
		AND due_date >= ? AND due_date <= date(?, '+7 days')`,
		companyID, today, today,
	)
	if err != nil {
		errOut("failed to query low-score customers: " + err.Error())
	}

	warnings := []map[string]interface{}{}
	for _, row := range lowScoreRows {
		warnings = append(warnings, map[string]interface{}{
			"customer":               row["customer_name"],
			"collection_probability": row["collection_probability"],
			"warning":                fmt.Sprintf("Customer '%s' has low collection probability (%v%%)", row["customer_name"], row["collection_probability"]),
		})
	}

	// Calculate week end date
	weekEndRow, err := queryOne(db, `SELECT date(?, '+7 days') AS week_end`, today)
	if err != nil {
		errOut("failed to calc week end: " + err.Error())
	}
	weekEnd, _ := weekEndRow["week_end"].(string)

	jsonOut(map[string]interface{}{
		"week_start":    today,
		"week_end":      weekEnd,
		"ar_due":        map[string]interface{}{"count": len(arRows), "total": arTotal, "items": arRows},
		"ap_due":        map[string]interface{}{"count": len(apRows), "total": apTotal, "items": apRows},
		"net_week":      netWeek,
		"warnings":      warnings,
		"warning_count": len(warnings),
	})
}

func cmdCashflowGap() {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	rows, err := queryRows(db,
		`SELECT * FROM cashflow_forecast
		WHERE company_id = ? AND gap_alert = 'true'
		ORDER BY period_start`,
		companyID,
	)
	if err != nil {
		errOut("failed to query gap alerts: " + err.Error())
	}

	jsonOut(map[string]interface{}{
		"count":      len(rows),
		"gap_alerts": rows,
	})
}
