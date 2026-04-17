package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ---------- Health Commands ----------

func cmdHealth(args []string) {
	if len(args) == 0 {
		errOut("usage: health <calculate|show|history>")
	}

	switch args[0] {
	case "calculate":
		period := ""
		if len(args) > 1 {
			period = args[1]
		}
		cmdHealthCalculate(period)
	case "show":
		cmdHealthShow()
	case "history":
		count := 6
		if len(args) > 1 {
			c, err := strconv.Atoi(args[1])
			if err == nil && c > 0 {
				count = c
			}
		}
		cmdHealthHistory(count)
	default:
		errOut("unknown health subcommand: " + args[0])
	}
}

func cmdHealthCalculate(period string) {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	now := vnNowISO()
	today := vnToday()

	if period == "" {
		// Default to current month YYYY-MM
		period = today[:7]
	}
	dateLike := period + "%"

	// Revenue
	revRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='in' AND category='revenue'`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	revenue := toInt64(revRow["total"])

	// COGS
	cogsRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='out' AND category='cogs'`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	cogs := toInt64(cogsRow["total"])

	// Operating expenses
	opexRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='out' AND category IN ('opex','rent','payroll','tax')`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	opex := toInt64(opexRow["total"])

	netProfit := revenue - cogs - opex

	// Total AR outstanding
	arRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM receivables
		 WHERE company_id=? AND status='outstanding'`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	totalAR := toInt64(arRow["total"])

	// Total AP outstanding
	apRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM payables
		 WHERE company_id=? AND status='outstanding'`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	totalAP := toInt64(apRow["total"])

	// Cash reserve from company profile
	companyRow, err := queryOne(db,
		`SELECT COALESCE(cash_reserve,0) AS cash_reserve, COALESCE(monthly_expense_avg,0) AS monthly_expense_avg
		 FROM company_profile WHERE company_id=?`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	cashReserve := toInt64(companyRow["cash_reserve"])
	monthlyExpenseAvg := toInt64(companyRow["monthly_expense_avg"])

	// Revenue this month vs last month (for growth calculation)
	thisMonthRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='in' AND category='revenue'`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	revenueThisMonth := toInt64(thisMonthRow["total"])

	// Derive previous month from period
	prevPeriod := prevMonth(period)
	prevLike := prevPeriod + "%"
	prevRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='in' AND category='revenue'`,
		companyID, prevLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	revenueLastMonth := toInt64(prevRow["total"])

	// Calculate health
	input := HealthInput{
		Revenue:           revenue,
		COGS:              cogs,
		OperatingExpenses: opex,
		NetProfit:         netProfit,
		TotalAR:           totalAR,
		TotalAP:           totalAP,
		CashReserve:       cashReserve,
		MonthlyExpenseAvg: monthlyExpenseAvg,
		RevenueLastMonth:  revenueLastMonth,
		RevenueThisMonth:  revenueThisMonth,
	}

	result := CalculateHealth(input)

	// Save to business_health_metrics
	_, err = exec(db,
		`INSERT INTO business_health_metrics
		 (company_id, period, revenue, cogs, gross_margin_pct, operating_expenses, net_profit, net_margin_pct,
		  dso_days, dpo_days, cash_conversion_cycle, burn_rate, runway_months, health_score, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		companyID, period, revenue, cogs, result.GrossMarginPct, opex, netProfit, result.NetMarginPct,
		result.DSODays, result.DPODays, result.CashConversionCycle, result.BurnRate, result.RunwayMonths, result.HealthScore, now,
	)
	if err != nil {
		errOut("failed to save health metrics: " + err.Error())
	}

	// Update company profile
	_, err = exec(db,
		`UPDATE company_profile SET health_score=?, risk_grade=? WHERE company_id=?`,
		result.HealthScore, result.RiskGrade, companyID)
	if err != nil {
		errOut("failed to update company profile: " + err.Error())
	}

	jsonOut(map[string]interface{}{
		"period":                period,
		"revenue":               revenue,
		"cogs":                  cogs,
		"gross_margin_pct":      result.GrossMarginPct,
		"operating_expenses":    opex,
		"net_profit":            netProfit,
		"net_margin_pct":        result.NetMarginPct,
		"dso_days":              result.DSODays,
		"dpo_days":              result.DPODays,
		"cash_conversion_cycle": result.CashConversionCycle,
		"burn_rate":             result.BurnRate,
		"runway_months":         result.RunwayMonths,
		"health_score":          result.HealthScore,
		"risk_grade":            result.RiskGrade,
		"components": map[string]interface{}{
			"profitability": result.ProfitabilityScore,
			"liquidity":     result.LiquidityScore,
			"efficiency":    result.EfficiencyScore,
			"growth":        result.GrowthScore,
			"stability":     result.StabilityScore,
		},
	})
}

func cmdHealthShow() {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	row, err := queryOne(db,
		`SELECT * FROM business_health_metrics
		 WHERE company_id=?
		 ORDER BY created_at DESC LIMIT 1`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	if row == nil {
		errOut("no health metrics found — run 'health calculate' first")
	}

	// Also get company risk grade
	companyRow, err := queryOne(db,
		`SELECT health_score, risk_grade FROM company_profile WHERE company_id=?`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	row["risk_grade"] = companyRow["risk_grade"]
	jsonOut(row)
}

func cmdHealthHistory(count int) {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	rows, err := queryRows(db,
		`SELECT period, health_score, revenue, net_profit, net_margin_pct, dso_days, runway_months, created_at
		 FROM business_health_metrics
		 WHERE company_id=?
		 ORDER BY period DESC LIMIT ?`, companyID, count)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	jsonOut(map[string]interface{}{
		"count":   len(rows),
		"history": rows,
	})
}

// prevMonth returns the YYYY-MM for the month before the given YYYY-MM period.
func prevMonth(period string) string {
	if len(period) < 7 {
		return period
	}
	y, _ := strconv.Atoi(period[:4])
	m, _ := strconv.Atoi(period[5:7])
	m--
	if m < 1 {
		m = 12
		y--
	}
	return fmt.Sprintf("%04d-%02d", y, m)
}

// ---------- Report Commands ----------

func cmdReport(args []string) {
	if len(args) == 0 {
		errOut("usage: report <pnl|cashflow|aging|summary>")
	}

	switch args[0] {
	case "pnl":
		if len(args) < 2 {
			errOut("usage: report pnl <YYYY-MM>")
		}
		cmdReportPNL(args[1])
	case "cashflow":
		if len(args) < 2 {
			errOut("usage: report cashflow <YYYY-MM>")
		}
		cmdReportCashflow(args[1])
	case "aging":
		cmdReportAging()
	case "summary":
		cmdReportSummary()
	default:
		errOut("unknown report subcommand: " + args[0])
	}
}

func cmdReportPNL(period string) {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	dateLike := period + "%"

	revRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='in' AND category='revenue'`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	revenue := toInt64(revRow["total"])

	cogsRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='out' AND category='cogs'`,
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

	opexRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='out' AND category IN ('opex','rent','payroll','tax')`,
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

	jsonOut(map[string]interface{}{
		"report":             "P&L",
		"period":             period,
		"revenue":            revenue,
		"cogs":               cogs,
		"gross_profit":       grossProfit,
		"gross_margin_pct":   fmt.Sprintf("%.1f", grossMarginPct),
		"operating_expenses": opex,
		"net_profit":         netProfit,
		"net_margin_pct":     fmt.Sprintf("%.1f", netMarginPct),
	})
}

func cmdReportCashflow(period string) {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	dateLike := period + "%"

	inRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total, COUNT(*) AS cnt FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='in'`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	cashIn := toInt64(inRow["total"])
	inCount := toInt64(inRow["cnt"])

	outRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total, COUNT(*) AS cnt FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='out'`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	cashOut := toInt64(outRow["total"])
	outCount := toInt64(outRow["cnt"])

	netCashflow := cashIn - cashOut

	jsonOut(map[string]interface{}{
		"report":         "Cashflow",
		"period":         period,
		"cash_in":        cashIn,
		"cash_in_count":  inCount,
		"cash_out":       cashOut,
		"cash_out_count": outCount,
		"net_cashflow":   netCashflow,
	})
}

func cmdReportAging() {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	today := vnToday()

	// AR aging
	arRows, err := queryRows(db,
		`SELECT CAST(julianday(?) - julianday(due_date) AS INTEGER) AS days_overdue, amount
		 FROM receivables WHERE company_id=? AND status='outstanding'`, today, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	arBuckets := map[string]int64{"current_0_30": 0, "days_31_60": 0, "days_61_90": 0, "over_90": 0}
	arCounts := map[string]int{"current_0_30": 0, "days_31_60": 0, "days_61_90": 0, "over_90": 0}
	arTotal := int64(0)
	for _, row := range arRows {
		days := toInt64(row["days_overdue"])
		amt := toInt64(row["amount"])
		arTotal += amt
		switch {
		case days <= 30:
			arBuckets["current_0_30"] += amt
			arCounts["current_0_30"]++
		case days <= 60:
			arBuckets["days_31_60"] += amt
			arCounts["days_31_60"]++
		case days <= 90:
			arBuckets["days_61_90"] += amt
			arCounts["days_61_90"]++
		default:
			arBuckets["over_90"] += amt
			arCounts["over_90"]++
		}
	}

	// AP aging
	apRows, err := queryRows(db,
		`SELECT CAST(julianday(?) - julianday(due_date) AS INTEGER) AS days_overdue, amount
		 FROM payables WHERE company_id=? AND status='outstanding'`, today, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	apBuckets := map[string]int64{"current_0_30": 0, "days_31_60": 0, "days_61_90": 0, "over_90": 0}
	apCounts := map[string]int{"current_0_30": 0, "days_31_60": 0, "days_61_90": 0, "over_90": 0}
	apTotal := int64(0)
	for _, row := range apRows {
		days := toInt64(row["days_overdue"])
		amt := toInt64(row["amount"])
		apTotal += amt
		switch {
		case days <= 30:
			apBuckets["current_0_30"] += amt
			apCounts["current_0_30"]++
		case days <= 60:
			apBuckets["days_31_60"] += amt
			apCounts["days_31_60"]++
		case days <= 90:
			apBuckets["days_61_90"] += amt
			apCounts["days_61_90"]++
		default:
			apBuckets["over_90"] += amt
			apCounts["over_90"]++
		}
	}

	jsonOut(map[string]interface{}{
		"report": "Aging",
		"as_of":  today,
		"ar_aging": map[string]interface{}{
			"total": arTotal,
			"buckets": map[string]interface{}{
				"current_0_30": map[string]interface{}{"count": arCounts["current_0_30"], "total": arBuckets["current_0_30"]},
				"days_31_60":   map[string]interface{}{"count": arCounts["days_31_60"], "total": arBuckets["days_31_60"]},
				"days_61_90":   map[string]interface{}{"count": arCounts["days_61_90"], "total": arBuckets["days_61_90"]},
				"over_90":      map[string]interface{}{"count": arCounts["over_90"], "total": arBuckets["over_90"]},
			},
		},
		"ap_aging": map[string]interface{}{
			"total": apTotal,
			"buckets": map[string]interface{}{
				"current_0_30": map[string]interface{}{"count": apCounts["current_0_30"], "total": apBuckets["current_0_30"]},
				"days_31_60":   map[string]interface{}{"count": apCounts["days_31_60"], "total": apBuckets["days_31_60"]},
				"days_61_90":   map[string]interface{}{"count": apCounts["days_61_90"], "total": apBuckets["days_61_90"]},
				"over_90":      map[string]interface{}{"count": apCounts["over_90"], "total": apBuckets["over_90"]},
			},
		},
	})
}

func cmdReportSummary() {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	today := vnToday()
	currentPeriod := today[:7]

	// Health score
	healthRow, err := queryOne(db,
		`SELECT health_score, risk_grade, name FROM company_profile WHERE company_id=?`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	healthScore := toInt64(healthRow["health_score"])
	riskGrade, _ := healthRow["risk_grade"].(string)
	companyName, _ := healthRow["name"].(string)

	// P&L highlights for current period
	dateLike := currentPeriod + "%"
	revRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='in' AND category='revenue'`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	revenue := toInt64(revRow["total"])

	expRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='out'`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	expenses := toInt64(expRow["total"])

	// AR/AP totals
	arRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total, COUNT(*) AS cnt FROM receivables
		 WHERE company_id=? AND status='outstanding'`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	arTotal := toInt64(arRow["total"])
	arCount := toInt64(arRow["cnt"])

	apRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total, COUNT(*) AS cnt FROM payables
		 WHERE company_id=? AND status='outstanding'`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	apTotal := toInt64(apRow["total"])
	apCount := toInt64(apRow["cnt"])

	// Overdue AR
	overdueRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total, COUNT(*) AS cnt FROM receivables
		 WHERE company_id=? AND status='outstanding' AND due_date < ?`, companyID, today)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	overdueAR := toInt64(overdueRow["total"])
	overdueARCount := toInt64(overdueRow["cnt"])

	// Cashflow gap alerts
	gapRow, err := queryOne(db,
		`SELECT COUNT(*) AS cnt FROM cashflow_forecast
		 WHERE company_id=? AND gap_alert='true'`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	gapAlerts := toInt64(gapRow["cnt"])

	// Top alerts
	alerts := []string{}
	if overdueAR > 0 {
		alerts = append(alerts, fmt.Sprintf("AR overdue: %d invoices, total %d VND", overdueARCount, overdueAR))
	}
	if gapAlerts > 0 {
		alerts = append(alerts, fmt.Sprintf("Cashflow gap alerts: %d forecasts show negative balance", gapAlerts))
	}
	if riskGrade == "D" || riskGrade == "F" {
		alerts = append(alerts, fmt.Sprintf("Low health score: %d (grade %s)", healthScore, riskGrade))
	}

	jsonOut(map[string]interface{}{
		"report":       "Executive Summary",
		"company":      companyName,
		"period":       currentPeriod,
		"health_score": healthScore,
		"risk_grade":   riskGrade,
		"pnl": map[string]interface{}{
			"revenue":    revenue,
			"expenses":   expenses,
			"net_profit": revenue - expenses,
		},
		"ar": map[string]interface{}{
			"outstanding_total": arTotal,
			"outstanding_count": arCount,
			"overdue_total":     overdueAR,
			"overdue_count":     overdueARCount,
		},
		"ap": map[string]interface{}{
			"outstanding_total": apTotal,
			"outstanding_count": apCount,
		},
		"cashflow_gap_alerts": gapAlerts,
		"alerts":              alerts,
		"alert_count":         len(alerts),
	})
}

// ---------- Tax Commands ----------

func cmdTax(args []string) {
	if len(args) == 0 {
		errOut("usage: tax <estimate-vat|estimate-cit|deadlines>")
	}

	switch args[0] {
	case "estimate-vat":
		if len(args) < 2 {
			errOut("usage: tax estimate-vat <YYYY-MM>")
		}
		cmdTaxEstimateVAT(args[1])
	case "estimate-cit":
		if len(args) < 2 {
			errOut("usage: tax estimate-cit <YYYY-MM>")
		}
		cmdTaxEstimateCIT(args[1])
	case "deadlines":
		cmdTaxDeadlines()
	default:
		errOut("unknown tax subcommand: " + args[0])
	}
}

func cmdTaxEstimateVAT(period string) {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	dateLike := period + "%"

	// Output VAT = 10% of revenue
	revRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='in' AND category='revenue'`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	revenue := toInt64(revRow["total"])

	// Input VAT = 10% of purchases (direction='out' AND category IN ('cogs','other'))
	purchRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='out' AND category IN ('cogs','other')`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	purchases := toInt64(purchRow["total"])

	outputVAT := revenue * 10 / 100
	inputVAT := purchases * 10 / 100
	vatPayable := outputVAT - inputVAT

	jsonOut(map[string]interface{}{
		"period":      period,
		"revenue":     revenue,
		"purchases":   purchases,
		"output_vat":  outputVAT,
		"input_vat":   inputVAT,
		"vat_payable": vatPayable,
		"note":        "VAT rate 10%. VAT payable = output VAT - input VAT",
	})
}

func cmdTaxEstimateCIT(period string) {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	dateLike := period + "%"

	// Revenue
	revRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='in' AND category='revenue'`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	revenue := toInt64(revRow["total"])

	// All expenses
	expRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND date LIKE ? AND direction='out'`,
		companyID, dateLike)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	expenses := toInt64(expRow["total"])

	netProfit := revenue - expenses
	cit := int64(0)
	if netProfit > 0 {
		cit = netProfit * 20 / 100
	}

	jsonOut(map[string]interface{}{
		"period":     period,
		"revenue":    revenue,
		"expenses":   expenses,
		"net_profit": netProfit,
		"cit":        cit,
		"cit_rate":   "20%",
		"note":       "CIT = 20% of net profit (simplified). Actual CIT may differ based on deductions.",
	})
}

func cmdTaxDeadlines() {
	// Try multiple paths for the tax calendar JSON
	paths := []string{
		filepath.Join(skillDir(), "data", "tax_calendar_b2b.json"),
		filepath.Join("data", "tax_calendar_b2b.json"),
	}

	// Also try relative to executable
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		paths = append(paths,
			filepath.Join(exeDir, "data", "tax_calendar_b2b.json"),
			filepath.Join(exeDir, "..", "data", "tax_calendar_b2b.json"),
		)
	}

	var data []byte
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}

	if data == nil {
		errOut("cannot find tax_calendar_b2b.json. Searched: " + strings.Join(paths, ", "))
	}

	var calendar struct {
		Deadlines []map[string]interface{} `json:"deadlines"`
	}
	if err := json.Unmarshal(data, &calendar); err != nil {
		errOut("failed to parse tax calendar: " + err.Error())
	}

	jsonOut(map[string]interface{}{
		"deadlines": calendar.Deadlines,
		"count":     len(calendar.Deadlines),
	})
}
