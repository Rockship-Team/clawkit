package main

import (
	"fmt"
	"strconv"
	"strings"
)

func cmdDiscount(args []string) {
	if len(args) == 0 {
		errOut("usage: discount <analyze|simulate|list>")
	}

	switch args[0] {
	case "analyze":
		cmdDiscountAnalyze()
	case "simulate":
		cmdDiscountSimulate(args[1:])
	case "list":
		cmdDiscountList(args[1:])
	default:
		errOut("unknown discount subcommand: " + args[0])
	}
}

func cmdDiscountAnalyze() {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	now := vnNowISO()

	// 1. Current gross margin
	revRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND direction='in' AND category='revenue'`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	revenue := toInt64(revRow["total"])

	cogsRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND direction='out' AND category='cogs'`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	cogs := toInt64(cogsRow["total"])

	grossMarginPct := 0.0
	if revenue > 0 {
		grossMarginPct = float64(revenue-cogs) * 100.0 / float64(revenue)
	}

	// 2. Customer segmentation by total revenue (direction='in')
	custRows, err := queryRows(db,
		`SELECT counterparty, SUM(amount) AS total_revenue, COUNT(*) AS txn_count
		 FROM business_transactions
		 WHERE company_id=? AND direction='in'
		 GROUP BY counterparty
		 ORDER BY total_revenue DESC`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	// Assign segments: top 20% = A, next 30% = B, rest = C
	totalCustomers := len(custRows)
	topA := totalCustomers * 20 / 100
	if topA < 1 && totalCustomers > 0 {
		topA = 1
	}
	topB := totalCustomers * 50 / 100 // top 50% boundary (A+B)
	if topB < topA {
		topB = topA
	}

	type segInfo struct {
		Count   int   `json:"count"`
		Revenue int64 `json:"revenue"`
	}
	segments := map[string]*segInfo{"A": {}, "B": {}, "C": {}}
	customerSegments := []map[string]interface{}{}

	for i, row := range custRows {
		seg := "C"
		if i < topA {
			seg = "A"
		} else if i < topB {
			seg = "B"
		}
		rev := toInt64(row["total_revenue"])
		segments[seg].Count++
		segments[seg].Revenue += rev
		customerSegments = append(customerSegments, map[string]interface{}{
			"counterparty":  row["counterparty"],
			"total_revenue": rev,
			"txn_count":     toInt64(row["txn_count"]),
			"segment":       seg,
		})
	}

	// 3. Seasonal dips: GROUP BY month, find months with revenue < 80% of average
	monthRows, err := queryRows(db,
		`SELECT substr(date, 1, 7) AS month, SUM(amount) AS monthly_revenue
		 FROM business_transactions
		 WHERE company_id=? AND direction='in' AND category='revenue'
		 GROUP BY substr(date, 1, 7)
		 ORDER BY month`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	monthlyAvg := int64(0)
	if len(monthRows) > 0 {
		totalMonthlyRev := int64(0)
		for _, mr := range monthRows {
			totalMonthlyRev += toInt64(mr["monthly_revenue"])
		}
		monthlyAvg = totalMonthlyRev / int64(len(monthRows))
	}

	threshold := monthlyAvg * 80 / 100
	seasonalDips := []map[string]interface{}{}
	for _, mr := range monthRows {
		rev := toInt64(mr["monthly_revenue"])
		if rev < threshold {
			seasonalDips = append(seasonalDips, map[string]interface{}{
				"month":   mr["month"],
				"revenue": rev,
				"avg":     monthlyAvg,
				"pct_of_avg": func() string {
					if monthlyAvg == 0 {
						return "0.0"
					}
					return fmt.Sprintf("%.1f", float64(rev)*100.0/float64(monthlyAvg))
				}(),
			})
		}
	}

	// 4. Generate strategy suggestions and save
	strategies := []map[string]interface{}{}

	// Strategy 1: Volume discount for B segment to grow them to A
	if segments["B"].Count > 0 {
		projectedImpact := segments["B"].Revenue * 5 / 100 // 5% growth estimate
		_, err := exec(db,
			`INSERT INTO discount_strategies
			 (company_id, strategy_type, target_segment, discount_pct, condition, projected_revenue_impact, projected_margin_impact, ai_recommendation, status, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'proposed', ?)`,
			companyID, "volume_discount", "B", 5,
			"Order volume increase >15%",
			projectedImpact,
			int64(grossMarginPct*100)-500, // margin impact in basis points
			"Offer 5% volume discount to B-segment customers to encourage larger orders and grow them toward A-segment",
			now,
		)
		if err != nil {
			errOut("failed to save strategy: " + err.Error())
		}
		strategies = append(strategies, map[string]interface{}{
			"type":        "volume_discount",
			"segment":     "B",
			"discount":    5,
			"description": "5% volume discount for B-segment customers",
		})
	}

	// Strategy 2: Early-pay discount for slow-paying A customers
	if segments["A"].Count > 0 {
		_, err := exec(db,
			`INSERT INTO discount_strategies
			 (company_id, strategy_type, target_segment, discount_pct, condition, projected_revenue_impact, projected_margin_impact, ai_recommendation, status, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'proposed', ?)`,
			companyID, "early_payment", "A", 2,
			"Payment within 10 days of invoice",
			int64(0), // revenue neutral
			int64(-200),
			"Offer 2% early-payment discount to A-segment customers to improve cash conversion cycle",
			now,
		)
		if err != nil {
			errOut("failed to save strategy: " + err.Error())
		}
		strategies = append(strategies, map[string]interface{}{
			"type":        "early_payment",
			"segment":     "A",
			"discount":    2,
			"description": "2% early-payment discount for A-segment to speed up collections",
		})
	}

	// Strategy 3: Seasonal promotion if dips detected
	if len(seasonalDips) > 0 {
		dipMonths := []string{}
		for _, d := range seasonalDips {
			dipMonths = append(dipMonths, fmt.Sprintf("%v", d["month"]))
		}
		_, err := exec(db,
			`INSERT INTO discount_strategies
			 (company_id, strategy_type, target_segment, discount_pct, condition, projected_revenue_impact, projected_margin_impact, ai_recommendation, status, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'proposed', ?)`,
			companyID, "seasonal_promotion", "all", 10,
			fmt.Sprintf("Active during months: %s", strings.Join(dipMonths, ", ")),
			monthlyAvg*15/100, // target 15% lift
			int64(-1000),
			fmt.Sprintf("10%% seasonal discount during slow months (%s) to smooth revenue curve", strings.Join(dipMonths, ", ")),
			now,
		)
		if err != nil {
			errOut("failed to save strategy: " + err.Error())
		}
		strategies = append(strategies, map[string]interface{}{
			"type":        "seasonal_promotion",
			"segment":     "all",
			"discount":    10,
			"months":      dipMonths,
			"description": "10% seasonal promotion during slow months",
		})
	}

	jsonOut(map[string]interface{}{
		"gross_margin_pct": fmt.Sprintf("%.1f", grossMarginPct),
		"revenue":          revenue,
		"cogs":             cogs,
		"customer_segments": map[string]interface{}{
			"A": map[string]interface{}{"count": segments["A"].Count, "revenue": segments["A"].Revenue},
			"B": map[string]interface{}{"count": segments["B"].Count, "revenue": segments["B"].Revenue},
			"C": map[string]interface{}{"count": segments["C"].Count, "revenue": segments["C"].Revenue},
		},
		"total_customers":     totalCustomers,
		"customers":           customerSegments,
		"seasonal_dips":       seasonalDips,
		"monthly_avg_revenue": monthlyAvg,
		"strategies":          strategies,
		"strategy_count":      len(strategies),
	})
}

func cmdDiscountSimulate(args []string) {
	if len(args) < 2 {
		errOut("usage: discount simulate <discount_pct> <target_segment> [volume_increase_pct]")
	}

	discountPct, err := strconv.ParseFloat(args[0], 64)
	if err != nil || discountPct <= 0 || discountPct >= 100 {
		errOut("discount_pct must be a number between 0 and 100")
	}

	targetSegment := strings.ToUpper(args[1])
	if targetSegment != "A" && targetSegment != "B" && targetSegment != "C" && targetSegment != "ALL" {
		errOut("target_segment must be A, B, C, or all")
	}

	volumeIncreasePct := 0.0
	if len(args) > 2 {
		volumeIncreasePct, _ = strconv.ParseFloat(args[2], 64)
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	now := vnNowISO()

	// Get all customers with revenue totals for segmentation
	custRows, err := queryRows(db,
		`SELECT counterparty, SUM(amount) AS total_revenue
		 FROM business_transactions
		 WHERE company_id=? AND direction='in'
		 GROUP BY counterparty
		 ORDER BY total_revenue DESC`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	// Segment boundaries
	totalCustomers := len(custRows)
	topA := totalCustomers * 20 / 100
	if topA < 1 && totalCustomers > 0 {
		topA = 1
	}
	topB := totalCustomers * 50 / 100
	if topB < topA {
		topB = topA
	}

	// Sum revenue for target segment
	currentRevenue := int64(0)
	for i, row := range custRows {
		seg := "C"
		if i < topA {
			seg = "A"
		} else if i < topB {
			seg = "B"
		}
		if targetSegment == "ALL" || seg == targetSegment {
			currentRevenue += toInt64(row["total_revenue"])
		}
	}

	// Get total COGS
	totalRevRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND direction='in'`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	totalRevenue := toInt64(totalRevRow["total"])

	cogsRow, err := queryOne(db,
		`SELECT COALESCE(SUM(amount),0) AS total FROM business_transactions
		 WHERE company_id=? AND direction='out' AND category='cogs'`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}
	totalCOGS := toInt64(cogsRow["total"])

	// Calculate COGS proportional to segment revenue
	cogsProportional := int64(0)
	if totalRevenue > 0 {
		cogsProportional = totalCOGS * currentRevenue / totalRevenue
	}

	// Simulation
	newRevenue := int64(float64(currentRevenue) * (1 - discountPct/100) * (1 + volumeIncreasePct/100))
	newCOGS := int64(float64(cogsProportional) * (1 + volumeIncreasePct/100))

	currentMargin := 0.0
	if currentRevenue > 0 {
		currentMargin = float64(currentRevenue-cogsProportional) * 100.0 / float64(currentRevenue)
	}
	newMargin := 0.0
	if newRevenue > 0 {
		newMargin = float64(newRevenue-newCOGS) * 100.0 / float64(newRevenue)
	}

	netImpact := newRevenue - currentRevenue

	// Save as proposed strategy
	_, err = exec(db,
		`INSERT INTO discount_strategies
		 (company_id, strategy_type, target_segment, discount_pct, condition, projected_revenue_impact, projected_margin_impact, ai_recommendation, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'proposed', ?)`,
		companyID, "simulation", targetSegment, int(discountPct),
		fmt.Sprintf("Discount %.1f%% with expected volume increase %.1f%%", discountPct, volumeIncreasePct),
		netImpact,
		int64(newMargin*100)-int64(currentMargin*100),
		fmt.Sprintf("Simulated: %.1f%% discount on segment %s, volume +%.1f%% => net impact %d VND", discountPct, targetSegment, volumeIncreasePct, netImpact),
		now,
	)
	if err != nil {
		errOut("failed to save simulation: " + err.Error())
	}

	jsonOut(map[string]interface{}{
		"target_segment":       targetSegment,
		"discount_pct":         discountPct,
		"volume_increase_pct":  volumeIncreasePct,
		"current_revenue":      currentRevenue,
		"projected_revenue":    newRevenue,
		"current_margin_pct":   fmt.Sprintf("%.1f", currentMargin),
		"projected_margin_pct": fmt.Sprintf("%.1f", newMargin),
		"net_impact":           netImpact,
		"recommendation": func() string {
			if netImpact > 0 {
				return "Positive impact — consider implementing"
			}
			return "Negative impact — review volume assumptions"
		}(),
	})
}

func cmdDiscountList(args []string) {
	filter := "all"
	if len(args) > 0 {
		filter = strings.ToLower(args[0])
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	var query string
	var qargs []interface{}

	switch filter {
	case "proposed":
		query = `SELECT * FROM discount_strategies WHERE company_id=? AND status='proposed' ORDER BY created_at DESC`
		qargs = []interface{}{companyID}
	case "active":
		query = `SELECT * FROM discount_strategies WHERE company_id=? AND status='active' ORDER BY created_at DESC`
		qargs = []interface{}{companyID}
	case "all":
		query = `SELECT * FROM discount_strategies WHERE company_id=? ORDER BY created_at DESC`
		qargs = []interface{}{companyID}
	default:
		errOut("invalid filter: " + filter + ". Valid: proposed, active, all")
	}

	rows, err := queryRows(db, query, qargs...)
	if err != nil {
		errOut("failed to list strategies: " + err.Error())
	}

	jsonOut(map[string]interface{}{
		"filter":     filter,
		"count":      len(rows),
		"strategies": rows,
	})
}

// ---------- Pricing Analyzer ----------

func cmdPricing(args []string) {
	if len(args) == 0 {
		errOut("usage: pricing <analyze>")
	}

	switch args[0] {
	case "analyze":
		cmdPricingAnalyze()
	default:
		errOut("unknown pricing subcommand: " + args[0])
	}
}

func cmdPricingAnalyze() {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	today := vnToday()

	// Category breakdown for revenue (direction='in')
	catRows, err := queryRows(db,
		`SELECT category, COUNT(*) AS txn_count, AVG(amount) AS avg_amount, SUM(amount) AS total_revenue
		 FROM business_transactions
		 WHERE company_id=? AND direction='in'
		 GROUP BY category
		 ORDER BY total_revenue DESC`, companyID)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	// Last 3 months vs prior 3 months comparison
	// last 3 months: from today-3months to today
	// prior 3 months: from today-6months to today-3months
	recentRows, err := queryRows(db,
		`SELECT category, COUNT(*) AS txn_count, SUM(amount) AS total_revenue
		 FROM business_transactions
		 WHERE company_id=? AND direction='in' AND date >= date(?, '-3 months')
		 GROUP BY category`, companyID, today)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	priorRows, err := queryRows(db,
		`SELECT category, COUNT(*) AS txn_count, SUM(amount) AS total_revenue
		 FROM business_transactions
		 WHERE company_id=? AND direction='in'
		 AND date >= date(?, '-6 months') AND date < date(?, '-3 months')
		 GROUP BY category`, companyID, today, today)
	if err != nil {
		errOut("query failed: " + err.Error())
	}

	recentMap := map[string]map[string]int64{}
	for _, row := range recentRows {
		cat := fmt.Sprintf("%v", row["category"])
		recentMap[cat] = map[string]int64{
			"count":   toInt64(row["txn_count"]),
			"revenue": toInt64(row["total_revenue"]),
		}
	}
	priorMap := map[string]map[string]int64{}
	for _, row := range priorRows {
		cat := fmt.Sprintf("%v", row["category"])
		priorMap[cat] = map[string]int64{
			"count":   toInt64(row["txn_count"]),
			"revenue": toInt64(row["total_revenue"]),
		}
	}

	categories := []map[string]interface{}{}
	for _, row := range catRows {
		cat := fmt.Sprintf("%v", row["category"])
		entry := map[string]interface{}{
			"category":      cat,
			"txn_count":     toInt64(row["txn_count"]),
			"avg_amount":    toInt64(row["avg_amount"]),
			"total_revenue": toInt64(row["total_revenue"]),
		}

		// Trend analysis
		recent := recentMap[cat]
		prior := priorMap[cat]
		trend := "stable"

		if prior != nil && prior["count"] > 0 && recent != nil {
			volumeChange := float64(recent["count"]-prior["count"]) * 100.0 / float64(prior["count"])
			revenueChange := float64(recent["revenue"]-prior["revenue"]) * 100.0 / float64(prior["revenue"])

			entry["recent_3mo_count"] = recent["count"]
			entry["prior_3mo_count"] = prior["count"]
			entry["recent_3mo_revenue"] = recent["revenue"]
			entry["prior_3mo_revenue"] = prior["revenue"]
			entry["volume_change_pct"] = fmt.Sprintf("%.1f", volumeChange)
			entry["revenue_change_pct"] = fmt.Sprintf("%.1f", revenueChange)

			if volumeChange < -10 {
				trend = "declining"
			} else if revenueChange > 10 {
				trend = "growing"
			}
		}

		entry["trend"] = trend
		categories = append(categories, entry)
	}

	jsonOut(map[string]interface{}{
		"as_of":      today,
		"categories": categories,
		"count":      len(categories),
	})
}
