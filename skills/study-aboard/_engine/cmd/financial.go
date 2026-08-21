package main

func cmdFinancial(args []string) {
	if len(args) == 0 {
		errOut("usage: financial cost-compare")
	}
	switch args[0] {
	case "cost-compare":
		// financial cost-compare <student_id>
		if len(args) < 2 {
			errOut("usage: financial cost-compare <student_id>")
		}
		studentID := args[1]

		profile, err := queryOne("SELECT * FROM student_profile WHERE id=?", studentID)
		if err != nil || profile == nil {
			errOut("student not found: " + studentID)
		}

		apps, err := queryRows(
			`SELECT a.id,a.university_id,a.university_name,a.category,a.application_type,
			 u.avg_annual_cost_usd,u.financial_aid_international,u.ed_available
			 FROM application a
			 LEFT JOIN university_record u ON u.id=a.university_id
			 WHERE a.student_id=? AND a.submission_status != 'removed'
			 ORDER BY a.category`,
			studentID,
		)
		if err != nil {
			errOut(err.Error())
		}
		if len(apps) == 0 {
			errOut("no applications found — add schools first with: application add ...")
		}

		budget := floatFromRow(profile, "annual_budget_usd")
		needsAid := intFromRow(profile, "needs_financial_aid") == 1
		sat := intFromRow(profile, "sat_score")

		type costRow struct {
			UniversityName      string  `json:"university_name"`
			Category            string  `json:"category"`
			ApplicationType     string  `json:"application_type"`
			TotalAnnualCostUSD  int     `json:"total_annual_cost_usd"`
			EstimatedAidUSD     int     `json:"estimated_aid_usd"`
			NetAnnualCostUSD    int     `json:"net_annual_cost_usd"`
			FinancialAidType    string  `json:"financial_aid_type"`
			OverBudget          bool    `json:"over_budget"`
			EDAvailable         bool    `json:"ed_available"`
		}

		var costTable []costRow
		for _, app := range apps {
			total := intFromRow(app, "avg_annual_cost_usd")
			aidType := strFromRow(app, "financial_aid_international", "none")
			aid := estimateAid(aidType, total, int(budget), sat, needsAid)
			net := total - aid
			if net < 0 {
				net = 0
			}
			overBudget := budget > 0 && float64(net) > budget
			edAvail := intFromRow(app, "ed_available") == 1

			costTable = append(costTable, costRow{
				UniversityName:     strFromRow(app, "university_name", ""),
				Category:           strFromRow(app, "category", ""),
				ApplicationType:    strFromRow(app, "application_type", ""),
				TotalAnnualCostUSD: total,
				EstimatedAidUSD:    aid,
				NetAnnualCostUSD:   net,
				FinancialAidType:   aidType,
				OverBudget:         overBudget,
				EDAvailable:        edAvail,
			})
		}

		// Sort by net cost asc
		for i := 0; i < len(costTable)-1; i++ {
			for j := i + 1; j < len(costTable); j++ {
				if costTable[j].NetAnnualCostUSD < costTable[i].NetAnnualCostUSD {
					costTable[i], costTable[j] = costTable[j], costTable[i]
				}
			}
		}

		cheapest, mostExpensive, overBudgetCount := 0, 0, 0
		if len(costTable) > 0 {
			cheapest = costTable[0].NetAnnualCostUSD
			mostExpensive = costTable[len(costTable)-1].NetAnnualCostUSD
			for _, r := range costTable {
				if r.OverBudget {
					overBudgetCount++
				}
			}
		}

		okOut(map[string]interface{}{
			"student_id":         studentID,
			"annual_budget_usd":  budget,
			"cost_comparison":    costTable,
			"summary": map[string]interface{}{
				"cheapest_net":      cheapest,
				"most_expensive_net": mostExpensive,
				"over_budget_count": overBudgetCount,
			},
		})

	default:
		errOut("unknown financial command: " + args[0])
	}
}

func estimateAid(aidType string, cost, budget, sat int, needsAid bool) int {
	if aidType == "none" || (!needsAid && budget == 0) {
		return 0
	}
	switch aidType {
	case "merit":
		pct := 0.10 + float64(sat-1200)/400.0*0.20
		if pct < 0.10 {
			pct = 0.10
		}
		if pct > 0.30 {
			pct = 0.30
		}
		return int(float64(cost) * pct)
	case "need_based":
		gap := cost - budget
		if gap < 0 {
			gap = 0
		}
		aid := int(float64(gap) * 0.60)
		cap := int(float64(cost) * 0.50)
		if aid > cap {
			return cap
		}
		return aid
	case "limited":
		return int(float64(cost) * 0.08)
	}
	return 0
}
