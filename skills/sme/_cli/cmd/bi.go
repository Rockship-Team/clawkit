package main

func cmdDashboard(args []string) {
	if len(args) == 0 {
		errOut("usage: dashboard summary")
	}
	org := defaultOrgID()
	today := vnToday()
	monthPrefix := today[:7]

	switch args[0] {
	case "summary":
		// Revenue this month
		revRow, _ := queryOne(`SELECT COALESCE(SUM(total),0) as revenue, COUNT(*) as count
			FROM invoices WHERE org_id=? AND direction='outbound' AND status NOT IN ('cancelled','draft')
			AND substr(issued_date,1,7)=?`, org, monthPrefix)

		// Expenses this month
		expRow, _ := queryOne(`SELECT COALESCE(SUM(total),0) as expenses, COUNT(*) as count
			FROM invoices WHERE org_id=? AND direction='inbound' AND status NOT IN ('cancelled','draft')
			AND substr(issued_date,1,7)=?`, org, monthPrefix)

		// AR overdue
		arRow, _ := queryOne(`SELECT COALESCE(SUM(amount_due),0) as overdue, COUNT(*) as count
			FROM invoices WHERE org_id=? AND direction='outbound' AND status='overdue'`, org)

		// AP due this week
		weekEnd := vnNow().AddDate(0, 0, 7).Format("2006-01-02")
		apRow, _ := queryOne(`SELECT COALESCE(SUM(amount_due),0) as due, COUNT(*) as count
			FROM invoices WHERE org_id=? AND direction='inbound' AND status IN ('confirmed','partial')
			AND due_date BETWEEN ? AND ?`, org, today, weekEnd)

		// Active tasks
		taskRow, _ := queryOne(`SELECT COUNT(*) as total,
			SUM(CASE WHEN status='todo' THEN 1 ELSE 0 END) as todo,
			SUM(CASE WHEN status='in_progress' THEN 1 ELSE 0 END) as in_progress,
			SUM(CASE WHEN priority='high' AND status IN ('todo','in_progress') THEN 1 ELSE 0 END) as high_priority
			FROM tasks WHERE org_id=?`, org)

		// Pipeline
		pipeRow, _ := queryOne(`SELECT COUNT(*) as leads, COALESCE(SUM(weighted_value),0) as weighted
			FROM leads WHERE org_id=? AND stage NOT IN ('won','lost')`, org)

		// Employees
		empRow, _ := queryOne("SELECT COUNT(*) as total FROM employees WHERE org_id=? AND status='active'", org)

		// Tax deadlines next 30 days
		thirtyDays := vnNow().AddDate(0, 0, 30).Format("2006-01-02")
		taxRows, _ := queryRows(`SELECT tax_type,period_label,deadline_date,status FROM tax_deadlines
			WHERE org_id=? AND deadline_date BETWEEN ? AND ? ORDER BY deadline_date LIMIT 5`,
			org, today, thirtyDays)

		// Pending approvals
		approvalRow, _ := queryOne("SELECT COUNT(*) as count FROM approval_requests WHERE org_id=? AND status='pending'", org)

		okOut(map[string]interface{}{
			"period":            monthPrefix,
			"revenue":           revRow,
			"expenses":          expRow,
			"ar_overdue":        arRow,
			"ap_due_week":       apRow,
			"tasks":             taskRow,
			"pipeline":          pipeRow,
			"employees":         empRow,
			"tax_upcoming":      taxRows,
			"pending_approvals": approvalRow,
		})

	default:
		errOut("unknown dashboard command: " + args[0])
	}
}

func cmdReport(args []string) {
	if len(args) == 0 {
		errOut("usage: report pnl|cashflow|ar-aging|ap-aging|revenue-monthly")
	}
	org := defaultOrgID()

	switch args[0] {
	case "pnl":
		// Simplified P&L for a period
		period := vnToday()[:7]
		if len(args) > 1 {
			period = args[1]
		}
		revRow, _ := queryOne(`SELECT COALESCE(SUM(subtotal),0) as revenue, COALESCE(SUM(vat_amount),0) as vat_collected
			FROM invoices WHERE org_id=? AND direction='outbound' AND status NOT IN ('cancelled','draft')
			AND substr(issued_date,1,7)=?`, org, period)
		expRow, _ := queryOne(`SELECT COALESCE(SUM(subtotal),0) as expenses, COALESCE(SUM(vat_amount),0) as vat_paid
			FROM invoices WHERE org_id=? AND direction='inbound' AND status NOT IN ('cancelled','draft')
			AND substr(issued_date,1,7)=?`, org, period)
		payrollRow, _ := queryOne(`SELECT COALESCE(SUM(total_gross),0) as payroll_cost, COALESCE(SUM(total_employer_cost),0) as total_cost
			FROM payroll_runs WHERE org_id=? AND period=? AND status IN ('approved','paid')`, org, period)

		revenue := int64(0)
		expenses := int64(0)
		payroll := int64(0)
		if revRow != nil {
			revenue, _ = revRow["revenue"].(int64)
		}
		if expRow != nil {
			expenses, _ = expRow["expenses"].(int64)
		}
		if payrollRow != nil {
			payroll, _ = payrollRow["total_cost"].(int64)
		}
		grossProfit := revenue - expenses
		netProfit := grossProfit - payroll

		okOut(map[string]interface{}{
			"period":        period,
			"revenue":       revenue,
			"cost_of_goods": expenses,
			"gross_profit":  grossProfit,
			"payroll":       payroll,
			"net_profit":    netProfit,
			"margin_pct":    0, // would calculate if revenue > 0
		})

	case "revenue-monthly":
		// Last 12 months revenue trend
		rows, _ := queryRows(`SELECT substr(issued_date,1,7) as month,
			COALESCE(SUM(CASE WHEN direction='outbound' THEN total ELSE 0 END),0) as revenue,
			COALESCE(SUM(CASE WHEN direction='inbound' THEN total ELSE 0 END),0) as expenses
			FROM invoices WHERE org_id=? AND status NOT IN ('cancelled','draft')
			AND issued_date >= date(?,'-12 months')
			GROUP BY substr(issued_date,1,7) ORDER BY month`, org, vnToday())
		okOut(map[string]interface{}{"monthly": rows, "count": len(rows)})

	case "ar-aging":
		rows, _ := queryRows(`SELECT
			seller_name as contact,
			SUM(CASE WHEN due_date >= ? THEN amount_due ELSE 0 END) as current,
			SUM(CASE WHEN due_date < ? AND due_date >= date(?,'- 30 days') THEN amount_due ELSE 0 END) as days_1_30,
			SUM(CASE WHEN due_date < date(?,'- 30 days') AND due_date >= date(?,'- 60 days') THEN amount_due ELSE 0 END) as days_31_60,
			SUM(CASE WHEN due_date < date(?,'- 60 days') AND due_date >= date(?,'- 90 days') THEN amount_due ELSE 0 END) as days_61_90,
			SUM(CASE WHEN due_date < date(?,'- 90 days') THEN amount_due ELSE 0 END) as over_90,
			SUM(amount_due) as total
			FROM invoices WHERE org_id=? AND direction='outbound' AND amount_due > 0 AND status NOT IN ('cancelled','paid')
			GROUP BY seller_name ORDER BY total DESC`,
			vnToday(), vnToday(), vnToday(), vnToday(), vnToday(), vnToday(), vnToday(), vnToday(), org)
		okOut(map[string]interface{}{"ar_aging": rows, "count": len(rows)})

	case "ap-aging":
		rows, _ := queryRows(`SELECT
			seller_name as contact,
			SUM(CASE WHEN due_date >= ? THEN amount_due ELSE 0 END) as current,
			SUM(CASE WHEN due_date < ? THEN amount_due ELSE 0 END) as overdue,
			SUM(amount_due) as total
			FROM invoices WHERE org_id=? AND direction='inbound' AND amount_due > 0 AND status NOT IN ('cancelled','paid')
			GROUP BY seller_name ORDER BY total DESC`,
			vnToday(), vnToday(), org)
		okOut(map[string]interface{}{"ap_aging": rows, "count": len(rows)})

	case "cashflow":
		// Monthly cashflow statement
		period := vnToday()[:7]
		if len(args) > 1 {
			period = args[1]
		}
		inRow, _ := queryOne(`SELECT COALESCE(SUM(amount),0) as total FROM payments
			WHERE org_id=? AND direction='in' AND status='completed' AND substr(paid_at,1,7)=?`, org, period)
		outRow, _ := queryOne(`SELECT COALESCE(SUM(amount),0) as total FROM payments
			WHERE org_id=? AND direction='out' AND status='completed' AND substr(paid_at,1,7)=?`, org, period)
		inAmt := int64(0)
		outAmt := int64(0)
		if inRow != nil {
			inAmt, _ = inRow["total"].(int64)
		}
		if outRow != nil {
			outAmt, _ = outRow["total"].(int64)
		}
		okOut(map[string]interface{}{
			"period":   period,
			"cash_in":  inAmt,
			"cash_out": outAmt,
			"net_flow": inAmt - outAmt,
		})

	default:
		errOut("unknown report command: " + args[0])
	}
}
