package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ---------- Recommend Commands ----------

func cmdRecommend(args []string) {
	if len(args) == 0 {
		errOut("usage: recommend <evaluate|list|update|loan-readiness>")
	}

	switch args[0] {
	case "evaluate":
		cmdRecommendEvaluate()
	case "list":
		filter := "all"
		if len(args) > 1 {
			filter = strings.ToLower(args[1])
		}
		cmdRecommendList(filter)
	case "update":
		cmdRecommendUpdate(args[1:])
	case "loan-readiness":
		targetAmount := int64(0)
		if len(args) > 1 {
			targetAmount = parseVND(args[1])
		}
		cmdLoanReadiness(targetAmount)
	default:
		errOut("unknown recommend subcommand: " + args[0])
	}
}

// loadProducts loads the Shinhan product catalog from JSON.
func loadProducts() map[string]map[string]interface{} {
	paths := []string{
		filepath.Join(skillDir(), "data", "shinhan_products.json"),
		filepath.Join("data", "shinhan_products.json"),
	}
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		paths = append(paths,
			filepath.Join(exeDir, "data", "shinhan_products.json"),
			filepath.Join(exeDir, "..", "data", "shinhan_products.json"),
		)
	}

	var data []byte
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}

	products := map[string]map[string]interface{}{}
	if data == nil {
		return products
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(data, &items); err != nil {
		return products
	}
	for _, item := range items {
		ptype, _ := item["type"].(string)
		products[ptype] = item
	}
	return products
}

func cmdRecommendEvaluate() {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	now := vnNowISO()
	today := vnToday()
	products := loadProducts()
	newRecs := []map[string]interface{}{}

	insertRec := func(productType, triggerReason, triggerData string, estimatedAmount int64, priority string) {
		productName := productType
		if p, ok := products[productType]; ok {
			if n, ok := p["name"].(string); ok {
				productName = n
			}
		}
		_, err := exec(db,
			`INSERT INTO bank_product_recommendations
			 (company_id, product_type, product_name, trigger_reason, trigger_data, estimated_amount, priority, status, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 'new', ?)`,
			companyID, productType, productName, triggerReason, triggerData, estimatedAmount, priority, now,
		)
		if err != nil {
			return // skip silently on duplicate/error
		}
		newRecs = append(newRecs, map[string]interface{}{
			"product_type":     productType,
			"product_name":     productName,
			"trigger_reason":   triggerReason,
			"estimated_amount": estimatedAmount,
			"priority":         priority,
		})
	}

	// 1. Cashflow gap => working_capital
	gapRow, err := queryOne(db,
		`SELECT COUNT(*) AS cnt FROM cashflow_forecast
		 WHERE company_id=? AND gap_alert='true'`, companyID)
	if err == nil && toInt64(gapRow["cnt"]) > 0 {
		// Get the largest gap
		gapAmtRow, _ := queryOne(db,
			`SELECT MIN(closing_balance) AS worst_gap FROM cashflow_forecast
			 WHERE company_id=? AND gap_alert='true'`, companyID)
		gapAmt := int64(0)
		if gapAmtRow != nil {
			gapAmt = -toInt64(gapAmtRow["worst_gap"])
			if gapAmt < 0 {
				gapAmt = 0
			}
		}
		insertRec("working_capital", "Cashflow gap detected in forecast",
			fmt.Sprintf("gap_alerts=%d", toInt64(gapRow["cnt"])),
			gapAmt, "high")
	}

	// 2. AR > 60 days overdue, total > 500M => invoice_financing
	arOverdueRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total, COUNT(*) AS cnt FROM receivables
		 WHERE company_id=? AND status='outstanding'
		 AND CAST(julianday(?)-julianday(due_date) AS INTEGER) > 60`,
		companyID, today)
	if err == nil {
		overdueTotal := toInt64(arOverdueRow["total"])
		if overdueTotal > 500000000 {
			insertRec("invoice_financing", "AR overdue > 60 days exceeds 500M VND",
				fmt.Sprintf("overdue_total=%d,count=%d", overdueTotal, toInt64(arOverdueRow["cnt"])),
				overdueTotal, "high")
		}
	}

	// 3. Consistent early-pay discount usage in AP => supply_chain
	earlyPayRow, err := queryOne(db,
		`SELECT COUNT(*) AS cnt FROM payables
		 WHERE company_id=? AND early_pay_discount_pct > 0`, companyID)
	if err == nil && toInt64(earlyPayRow["cnt"]) >= 3 {
		insertRec("supply_chain", "Consistent early-pay discount usage detected",
			fmt.Sprintf("early_pay_count=%d", toInt64(earlyPayRow["cnt"])),
			0, "medium")
	}

	// 4. Revenue growing > 20% QoQ => expansion_loan
	// Compare last 2 quarters of revenue
	currentPeriod := today[:7]
	q1End := currentPeriod
	q1Start := prevMonth(prevMonth(prevMonth(currentPeriod)))
	q2End := prevMonth(q1Start)
	q2Start := prevMonth(prevMonth(prevMonth(q2End)))

	q1RevRow, _ := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND direction='in' AND category='revenue'
		 AND date >= ? AND date < ?`, companyID, q1Start+"-01", q1End+"-32")
	q2RevRow, _ := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND direction='in' AND category='revenue'
		 AND date >= ? AND date < ?`, companyID, q2Start+"-01", q2End+"-32")

	q1Rev := int64(0)
	q2Rev := int64(0)
	if q1RevRow != nil {
		q1Rev = toInt64(q1RevRow["total"])
	}
	if q2RevRow != nil {
		q2Rev = toInt64(q2RevRow["total"])
	}
	if q2Rev > 0 {
		growthPct := (q1Rev - q2Rev) * 100 / q2Rev
		if growthPct > 20 {
			insertRec("expansion_loan", fmt.Sprintf("Revenue growing %d%% QoQ", growthPct),
				fmt.Sprintf("q1_rev=%d,q2_rev=%d,growth=%d%%", q1Rev, q2Rev, growthPct),
				q1Rev, "medium")
		}
	}

	// 5. Import/export transactions => trade_finance
	tradeRow, err := queryOne(db,
		`SELECT COUNT(*) AS cnt FROM business_transactions
		 WHERE company_id=? AND (
			LOWER(category) LIKE '%import%' OR LOWER(category) LIKE '%export%'
			OR LOWER(counterparty) LIKE '%usd%' OR LOWER(counterparty) LIKE '%eur%'
			OR LOWER(note) LIKE '%import%' OR LOWER(note) LIKE '%export%'
		 )`, companyID)
	if err == nil && toInt64(tradeRow["cnt"]) > 0 {
		insertRec("trade_finance", "Import/export transactions detected",
			fmt.Sprintf("trade_txn_count=%d", toInt64(tradeRow["cnt"])),
			0, "medium")
	}

	// 6. Cash reserve > 6 months expenses => deposit
	companyRow, _ := queryOne(db,
		`SELECT COALESCE(cash_reserve,0) AS cash_reserve, COALESCE(monthly_expense_avg,0) AS monthly_expense_avg
		 FROM company_profile WHERE company_id=?`, companyID)
	cashReserve := int64(0)
	monthlyExpenseAvg := int64(0)
	if companyRow != nil {
		cashReserve = toInt64(companyRow["cash_reserve"])
		monthlyExpenseAvg = toInt64(companyRow["monthly_expense_avg"])
	}
	if monthlyExpenseAvg > 0 && cashReserve > monthlyExpenseAvg*6 {
		excess := cashReserve - monthlyExpenseAvg*3 // keep 3 months, invest the rest
		insertRec("deposit", "Cash reserve exceeds 6 months of expenses",
			fmt.Sprintf("cash_reserve=%d,monthly_expense=%d", cashReserve, monthlyExpenseAvg),
			excess, "low")
	}

	// 7. Equipment purchase > 200M => equipment
	equipRow, err := queryOne(db,
		`SELECT COUNT(*) AS cnt, MAX(amount) AS max_amount FROM business_transactions
		 WHERE company_id=? AND direction='out'
		 AND (LOWER(category)='other' OR LOWER(type)='purchase')
		 AND amount > 200000000`, companyID)
	if err == nil && toInt64(equipRow["cnt"]) > 0 {
		insertRec("equipment", "Large equipment/asset purchase detected",
			fmt.Sprintf("count=%d,max=%d", toInt64(equipRow["cnt"]), toInt64(equipRow["max_amount"])),
			toInt64(equipRow["max_amount"]), "medium")
	}

	// 8. Health score > 80 + growth => premium_account
	healthRow, _ := queryOne(db,
		`SELECT COALESCE(health_score,0) AS hs FROM company_profile WHERE company_id=?`, companyID)
	hs := int64(0)
	if healthRow != nil {
		hs = toInt64(healthRow["hs"])
	}
	if hs > 80 {
		insertRec("premium_account", "High health score indicates premium eligibility",
			fmt.Sprintf("health_score=%d", hs),
			0, "low")
	}

	// 9. Employee count > 50 => payroll + insurance
	empRow, _ := queryOne(db,
		`SELECT COALESCE(employee_count,0) AS ec FROM company_profile WHERE company_id=?`, companyID)
	ec := int64(0)
	if empRow != nil {
		ec = toInt64(empRow["ec"])
	}
	if ec > 50 {
		insertRec("payroll", "Employee count > 50, payroll automation recommended",
			fmt.Sprintf("employee_count=%d", ec),
			0, "medium")
		insertRec("insurance", "Employee count > 50, group insurance recommended",
			fmt.Sprintf("employee_count=%d", ec),
			0, "medium")
	}

	// 10. Foreign transactions => fx_hedging
	fxRow, err := queryOne(db,
		`SELECT COUNT(*) AS cnt FROM business_transactions
		 WHERE company_id=? AND (
			LOWER(note) LIKE '%usd%' OR LOWER(note) LIKE '%eur%'
			OR LOWER(note) LIKE '%foreign%'
			OR LOWER(counterparty) LIKE '%usd%' OR LOWER(counterparty) LIKE '%eur%'
		 )`, companyID)
	if err == nil && toInt64(fxRow["cnt"]) > 0 {
		insertRec("fx_hedging", "Foreign currency transactions detected",
			fmt.Sprintf("fx_txn_count=%d", toInt64(fxRow["cnt"])),
			0, "medium")
	}

	// 11. Investment topic in coach conversations => wealth_management
	investRow, err := queryOne(db,
		`SELECT COUNT(*) AS cnt FROM coach_conversations_b2b
		 WHERE company_id=? AND LOWER(topic)='investment'`, companyID)
	if err == nil && toInt64(investRow["cnt"]) > 0 {
		insertRec("wealth_management", "Owner inquired about investment",
			fmt.Sprintf("conversation_count=%d", toInt64(investRow["cnt"])),
			0, "low")
	}

	jsonOut(map[string]interface{}{
		"evaluated_triggers":  11,
		"new_recommendations": len(newRecs),
		"recommendations":     newRecs,
	})
}

func cmdRecommendList(filter string) {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	var query string
	var qargs []interface{}

	switch filter {
	case "new":
		query = `SELECT * FROM bank_product_recommendations WHERE company_id=? AND status='new' ORDER BY created_at DESC`
		qargs = []interface{}{companyID}
	case "contacted":
		query = `SELECT * FROM bank_product_recommendations WHERE company_id=? AND status='contacted' ORDER BY created_at DESC`
		qargs = []interface{}{companyID}
	case "converted":
		query = `SELECT * FROM bank_product_recommendations WHERE company_id=? AND status='converted' ORDER BY created_at DESC`
		qargs = []interface{}{companyID}
	case "all":
		query = `SELECT * FROM bank_product_recommendations WHERE company_id=? ORDER BY created_at DESC`
		qargs = []interface{}{companyID}
	default:
		errOut("invalid filter: " + filter + ". Valid: new, contacted, converted, all")
	}

	rows, err := queryRows(db, query, qargs...)
	if err != nil {
		errOut("failed to list recommendations: " + err.Error())
	}

	jsonOut(map[string]interface{}{
		"filter":          filter,
		"count":           len(rows),
		"recommendations": rows,
	})
}

func cmdRecommendUpdate(args []string) {
	if len(args) < 2 {
		errOut("usage: recommend update <id> <status> [assigned_rm] [outcome]")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		errOut("invalid id: " + args[0])
	}

	status := strings.ToLower(args[1])
	validStatuses := map[string]bool{"new": true, "contacted": true, "converted": true, "declined": true}
	if !validStatuses[status] {
		errOut("invalid status. Valid: new, contacted, converted, declined")
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	now := vnNowISO()

	// Check exists
	row, err := queryOne(db,
		`SELECT * FROM bank_product_recommendations WHERE id=? AND company_id=?`, id, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	if row == nil {
		errOut(fmt.Sprintf("recommendation #%d not found", id))
	}

	assignedRM := ""
	if len(args) > 2 {
		assignedRM = args[2]
	}
	outcome := ""
	if len(args) > 3 {
		outcome = strings.Join(args[3:], " ")
	}

	contactedAt := ""
	if status == "contacted" || status == "converted" {
		contactedAt = now
	}

	_, err = exec(db,
		`UPDATE bank_product_recommendations
		 SET status=?, assigned_rm=?, outcome=?, contacted_at=?
		 WHERE id=? AND company_id=?`,
		status, assignedRM, outcome, contactedAt, id, companyID)
	if err != nil {
		errOut("failed to update recommendation: " + err.Error())
	}

	updated, err := queryOne(db, "SELECT * FROM bank_product_recommendations WHERE id=?", id)
	if err != nil {
		errOut("failed to read back recommendation: " + err.Error())
	}

	okOut(map[string]interface{}{
		"message":        fmt.Sprintf("Recommendation #%d updated to %s", id, status),
		"recommendation": updated,
	})
}

func cmdLoanReadiness(targetAmount int64) {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	// 1. Revenue stability: check available months of revenue
	monthRows, err := queryRows(db,
		`SELECT substr(date,1,7) AS month, SUM(amount) AS monthly_revenue
		 FROM business_transactions
		 WHERE company_id=? AND direction='in' AND category='revenue'
		 GROUP BY substr(date,1,7)
		 ORDER BY month DESC LIMIT 12`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	revenueStabilityScore := 50 // default
	avgRevenue := int64(0)
	if len(monthRows) > 0 {
		totalRev := int64(0)
		growing := 0
		for _, row := range monthRows {
			totalRev += toInt64(row["monthly_revenue"])
		}
		avgRevenue = totalRev / int64(len(monthRows))

		// Check if growing: compare first half vs second half
		if len(monthRows) >= 4 {
			recentHalf := int64(0)
			olderHalf := int64(0)
			mid := len(monthRows) / 2
			for i, row := range monthRows {
				if i < mid {
					recentHalf += toInt64(row["monthly_revenue"])
				} else {
					olderHalf += toInt64(row["monthly_revenue"])
				}
			}
			if olderHalf > 0 {
				changePct := (recentHalf - olderHalf) * 100 / olderHalf
				if changePct > 10 {
					growing = 1
					revenueStabilityScore = 90
				} else if changePct >= 0 {
					revenueStabilityScore = 70
				} else {
					revenueStabilityScore = 30
				}
			}
		}
		_ = growing
	}

	// 2. Profitability from latest health metrics
	profitabilityScore := 50
	healthRow, _ := queryOne(db,
		`SELECT net_margin_pct, health_score FROM business_health_metrics
		 WHERE company_id=? ORDER BY created_at DESC LIMIT 1`, companyID)
	netMargin := int64(0)
	if healthRow != nil {
		netMargin = toInt64(healthRow["net_margin_pct"])
		if netMargin > 15 {
			profitabilityScore = 90
		} else if netMargin > 10 {
			profitabilityScore = 75
		} else if netMargin > 5 {
			profitabilityScore = 60
		} else if netMargin > 0 {
			profitabilityScore = 40
		} else {
			profitabilityScore = 15
		}
	}

	// 3. Payment history
	paymentHistoryScore := 50
	arPaidRow, _ := queryOne(db,
		`SELECT COUNT(*) AS total,
		        SUM(CASE WHEN days_overdue <= 0 THEN 1 ELSE 0 END) AS on_time
		 FROM receivables WHERE company_id=? AND status='paid'`, companyID)
	arOnTimePct := 0.0
	if arPaidRow != nil && toInt64(arPaidRow["total"]) > 0 {
		arOnTimePct = float64(toInt64(arPaidRow["on_time"])) * 100.0 / float64(toInt64(arPaidRow["total"]))
	}

	apPaidRow, _ := queryOne(db,
		`SELECT COUNT(*) AS total FROM payables WHERE company_id=? AND status='paid'`, companyID)
	apPaidTotal := int64(0)
	if apPaidRow != nil {
		apPaidTotal = toInt64(apPaidRow["total"])
	}
	// For AP, assume on-time if paid (we don't track overdue on AP payments in this simplified model)
	apOnTimePct := 0.0
	if apPaidTotal > 0 {
		apOnTimePct = 85.0 // default assumption
	}

	paymentHistoryScore = int((arOnTimePct + apOnTimePct) / 2)
	if paymentHistoryScore > 100 {
		paymentHistoryScore = 100
	}

	// 4. Existing debt burden
	debtScore := 80 // default: no known debt = good
	existingPayables, _ := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM payables
		 WHERE company_id=? AND status='outstanding'`, companyID)
	outstandingDebt := int64(0)
	if existingPayables != nil {
		outstandingDebt = toInt64(existingPayables["total"])
	}
	if avgRevenue > 0 {
		debtRatio := float64(outstandingDebt) / float64(avgRevenue)
		if debtRatio > 3 {
			debtScore = 20
		} else if debtRatio > 2 {
			debtScore = 40
		} else if debtRatio > 1 {
			debtScore = 60
		}
	}

	// Composite score
	compositeScore := (revenueStabilityScore*30 + profitabilityScore*25 + paymentHistoryScore*25 + debtScore*20) / 100

	// Build strengths, warnings, gaps
	strengths := []string{}
	warnings := []string{}
	gaps := []string{}

	if revenueStabilityScore >= 70 {
		strengths = append(strengths, "Stable/growing revenue trend")
	} else if revenueStabilityScore >= 40 {
		warnings = append(warnings, "Revenue trend is flat or slightly declining")
	} else {
		gaps = append(gaps, "Revenue is declining — banks may require additional collateral")
	}

	if profitabilityScore >= 60 {
		strengths = append(strengths, fmt.Sprintf("Net margin %d%% — healthy profitability", netMargin))
	} else if profitabilityScore >= 30 {
		warnings = append(warnings, fmt.Sprintf("Net margin %d%% — below ideal for unsecured lending", netMargin))
	} else {
		gaps = append(gaps, fmt.Sprintf("Net margin %d%% — loss-making, loan approval unlikely without collateral", netMargin))
	}

	if paymentHistoryScore >= 60 {
		strengths = append(strengths, fmt.Sprintf("Good payment history (AR on-time: %.0f%%)", arOnTimePct))
	} else {
		warnings = append(warnings, "Payment history needs improvement")
	}

	if debtScore >= 60 {
		strengths = append(strengths, "Low existing debt burden")
	} else {
		gaps = append(gaps, fmt.Sprintf("High existing payables: %d VND", outstandingDebt))
	}

	// Suggest Shinhan product
	products := loadProducts()
	suggestedProduct := "working_capital" // default
	if targetAmount > 500000000 {
		suggestedProduct = "expansion_loan"
	}

	productInfo := map[string]interface{}{}
	if p, ok := products[suggestedProduct]; ok {
		productInfo = p
	}

	jsonOut(map[string]interface{}{
		"loan_readiness_score": compositeScore,
		"target_amount":        targetAmount,
		"components": map[string]interface{}{
			"revenue_stability": revenueStabilityScore,
			"profitability":     profitabilityScore,
			"payment_history":   paymentHistoryScore,
			"debt_burden":       debtScore,
		},
		"strengths":           strengths,
		"warnings":            warnings,
		"gaps":                gaps,
		"suggested_product":   productInfo,
		"avg_monthly_revenue": avgRevenue,
		"outstanding_debt":    outstandingDebt,
	})
}

// ---------- Banker Commands ----------

func cmdBanker(args []string) {
	if len(args) == 0 {
		errOut("usage: banker <portfolio|pipeline|alerts>")
	}

	switch args[0] {
	case "portfolio":
		cmdBankerPortfolio()
	case "pipeline":
		cmdBankerPipeline()
	case "alerts":
		cmdBankerAlerts()
	default:
		errOut("unknown banker subcommand: " + args[0])
	}
}

func cmdBankerPortfolio() {
	db := mustDB()
	defer db.Close()

	// Company summary
	companyRows, err := queryRows(db,
		`SELECT company_id, name, industry, health_score, risk_grade, monthly_revenue_avg, employee_count
		 FROM company_profile`)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	totalRevenue := int64(0)
	for _, row := range companyRows {
		totalRevenue += toInt64(row["monthly_revenue_avg"])
	}

	// Product count
	prodRow, err := queryOne(db,
		`SELECT COUNT(*) AS cnt FROM bank_product_recommendations WHERE status IN ('new','contacted')`)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	activeProducts := toInt64(prodRow["cnt"])

	jsonOut(map[string]interface{}{
		"total_companies":       len(companyRows),
		"companies":             companyRows,
		"total_monthly_revenue": totalRevenue,
		"active_product_recs":   activeProducts,
	})
}

func cmdBankerPipeline() {
	db := mustDB()
	defer db.Close()

	rows, err := queryRows(db,
		`SELECT product_type, COUNT(*) AS count, SUM(estimated_amount) AS total_estimated
		 FROM bank_product_recommendations
		 WHERE status='new'
		 GROUP BY product_type
		 ORDER BY total_estimated DESC`)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	totalEstimated := int64(0)
	totalCount := int64(0)
	for _, row := range rows {
		totalEstimated += toInt64(row["total_estimated"])
		totalCount += toInt64(row["count"])
	}

	jsonOut(map[string]interface{}{
		"pipeline":        rows,
		"total_products":  totalCount,
		"total_estimated": totalEstimated,
		"product_types":   len(rows),
	})
}

func cmdBankerAlerts() {
	db := mustDB()
	defer db.Close()

	today := vnToday()
	alerts := []map[string]interface{}{}

	// 1. Companies with health score drop (compare last two health metrics)
	companyRows, err := queryRows(db, `SELECT company_id, name FROM company_profile`)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	for _, company := range companyRows {
		cid, _ := company["company_id"].(string)
		cname, _ := company["name"].(string)

		healthRows, err := queryRows(db,
			`SELECT health_score, period FROM business_health_metrics
			 WHERE company_id=? ORDER BY created_at DESC LIMIT 2`, cid)
		if err == nil && len(healthRows) >= 2 {
			current := toInt64(healthRows[0]["health_score"])
			previous := toInt64(healthRows[1]["health_score"])
			if current < previous-10 {
				alerts = append(alerts, map[string]interface{}{
					"type":     "health_score_drop",
					"company":  cname,
					"detail":   fmt.Sprintf("Health score dropped from %d to %d", previous, current),
					"severity": "high",
				})
			}
		}
	}

	// 2. High overdue AR (> 60 days)
	overdueRows, err := queryRows(db,
		`SELECT r.company_id, cp.name AS company_name,
		        SUM(r.amount) AS overdue_total, COUNT(*) AS overdue_count
		 FROM receivables r
		 JOIN company_profile cp ON r.company_id = cp.company_id
		 WHERE r.status='outstanding'
		 AND CAST(julianday(?)-julianday(r.due_date) AS INTEGER) > 60
		 GROUP BY r.company_id`, today)
	if err == nil {
		for _, row := range overdueRows {
			cname, _ := row["company_name"].(string)
			alerts = append(alerts, map[string]interface{}{
				"type":     "high_overdue_ar",
				"company":  cname,
				"detail":   fmt.Sprintf("%d invoices overdue >60 days, total %d VND", toInt64(row["overdue_count"]), toInt64(row["overdue_total"])),
				"severity": "high",
			})
		}
	}

	// 3. Cashflow gap forecasts
	gapRows, err := queryRows(db,
		`SELECT cf.company_id, cp.name AS company_name,
		        COUNT(*) AS gap_count, MIN(cf.closing_balance) AS worst_balance
		 FROM cashflow_forecast cf
		 JOIN company_profile cp ON cf.company_id = cp.company_id
		 WHERE cf.gap_alert='true'
		 GROUP BY cf.company_id`)
	if err == nil {
		for _, row := range gapRows {
			cname, _ := row["company_name"].(string)
			alerts = append(alerts, map[string]interface{}{
				"type":     "cashflow_gap",
				"company":  cname,
				"detail":   fmt.Sprintf("%d cashflow gap forecasts, worst balance %d VND", toInt64(row["gap_count"]), toInt64(row["worst_balance"])),
				"severity": "medium",
			})
		}
	}

	jsonOut(map[string]interface{}{
		"alert_count": len(alerts),
		"alerts":      alerts,
	})
}
