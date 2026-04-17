package main

// HealthInput holds all raw financial data needed for health calculation.
type HealthInput struct {
	Revenue, COGS, OperatingExpenses, NetProfit      int64
	TotalAR, TotalAP, CashReserve, MonthlyExpenseAvg int64
	RevenueLastMonth, RevenueThisMonth               int64
}

// HealthResult holds all computed health metrics.
type HealthResult struct {
	GrossMarginPct, NetMarginPct                                                     int
	DSODays, DPODays, CashConversionCycle                                            int
	BurnRate                                                                         int64
	RunwayMonths, HealthScore                                                        int
	RiskGrade                                                                        string
	ProfitabilityScore, LiquidityScore, EfficiencyScore, GrowthScore, StabilityScore int
}

// CalculateHealth computes a composite business health score from financial inputs.
// All calculations are pure — no database or I/O access.
func CalculateHealth(in HealthInput) HealthResult {
	var r HealthResult

	// Gross margin
	if in.Revenue > 0 {
		r.GrossMarginPct = int((in.Revenue - in.COGS) * 100 / in.Revenue)
	}

	// Net margin
	if in.Revenue > 0 {
		r.NetMarginPct = int(in.NetProfit * 100 / in.Revenue)
	}

	// DSO = (AR / Revenue) * 30
	if in.Revenue > 0 {
		r.DSODays = int(in.TotalAR * 30 / in.Revenue)
	}

	// DPO = (AP / COGS) * 30
	if in.COGS > 0 {
		r.DPODays = int(in.TotalAP * 30 / in.COGS)
	}

	// CCC = DSO - DPO
	r.CashConversionCycle = r.DSODays - r.DPODays

	// Burn rate = -NetProfit if negative, else 0
	if in.NetProfit < 0 {
		r.BurnRate = -in.NetProfit
	}

	// Runway = CashReserve / BurnRate (months)
	if r.BurnRate > 0 {
		r.RunwayMonths = int(in.CashReserve / r.BurnRate)
	} else if in.CashReserve > 0 {
		// Profitable with cash — effectively unlimited; cap at 99
		r.RunwayMonths = 99
	}

	// --- Component scores ---

	// Profitability (25%): net_margin * 5, capped 0-100 (20% margin = 100)
	r.ProfitabilityScore = r.NetMarginPct * 5
	if r.ProfitabilityScore < 0 {
		r.ProfitabilityScore = 0
	}
	if r.ProfitabilityScore > 100 {
		r.ProfitabilityScore = 100
	}

	// Liquidity (25%): runway >= 12mo=100, >=6=70, >=3=40, else 10
	switch {
	case r.RunwayMonths >= 12:
		r.LiquidityScore = 100
	case r.RunwayMonths >= 6:
		r.LiquidityScore = 70
	case r.RunwayMonths >= 3:
		r.LiquidityScore = 40
	default:
		r.LiquidityScore = 10
	}

	// Efficiency (20%): DSO <= 15=100, <=30=80, <=45=60, <=60=40, else 20
	switch {
	case r.DSODays <= 15:
		r.EfficiencyScore = 100
	case r.DSODays <= 30:
		r.EfficiencyScore = 80
	case r.DSODays <= 45:
		r.EfficiencyScore = 60
	case r.DSODays <= 60:
		r.EfficiencyScore = 40
	default:
		r.EfficiencyScore = 20
	}

	// Growth (15%): 50 + (month-over-month growth% * 2), capped 0-100
	growthPct := 0
	if in.RevenueLastMonth > 0 {
		growthPct = int((in.RevenueThisMonth - in.RevenueLastMonth) * 100 / in.RevenueLastMonth)
	}
	r.GrowthScore = 50 + growthPct*2
	if r.GrowthScore < 0 {
		r.GrowthScore = 0
	}
	if r.GrowthScore > 100 {
		r.GrowthScore = 100
	}

	// Stability (15%): default 50, penalize high concentration
	// Without detailed concentration data, use default
	r.StabilityScore = 50

	// Composite health score (weighted)
	r.HealthScore = (r.ProfitabilityScore*25 +
		r.LiquidityScore*25 +
		r.EfficiencyScore*20 +
		r.GrowthScore*15 +
		r.StabilityScore*15) / 100

	// Risk grade
	switch {
	case r.HealthScore >= 80:
		r.RiskGrade = "A"
	case r.HealthScore >= 60:
		r.RiskGrade = "B"
	case r.HealthScore >= 40:
		r.RiskGrade = "C"
	case r.HealthScore >= 20:
		r.RiskGrade = "D"
	default:
		r.RiskGrade = "F"
	}

	return r
}
