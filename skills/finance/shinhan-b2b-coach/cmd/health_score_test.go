package main

import "testing"

func TestCalculateHealth_Profitable(t *testing.T) {
	in := HealthInput{
		Revenue:           1000000000, // 1 billion
		COGS:              600000000,  // 600M
		OperatingExpenses: 200000000,  // 200M
		NetProfit:         200000000,  // 200M => 20% margin
		TotalAR:           150000000,  // 150M
		TotalAP:           100000000,  // 100M
		CashReserve:       500000000,  // 500M
		MonthlyExpenseAvg: 80000000,   // 80M/month
		RevenueLastMonth:  80000000,
		RevenueThisMonth:  90000000, // growing ~12.5%
	}

	r := CalculateHealth(in)

	if r.RiskGrade != "A" && r.RiskGrade != "B" {
		t.Errorf("expected risk grade A or B for profitable company, got %s (score=%d)", r.RiskGrade, r.HealthScore)
	}
	if r.HealthScore < 60 {
		t.Errorf("expected health score >= 60 for profitable company, got %d", r.HealthScore)
	}
	if r.NetMarginPct != 20 {
		t.Errorf("expected net margin 20%%, got %d%%", r.NetMarginPct)
	}
	if r.GrossMarginPct != 40 {
		t.Errorf("expected gross margin 40%%, got %d%%", r.GrossMarginPct)
	}
	if r.BurnRate != 0 {
		t.Errorf("expected burn rate 0 for profitable company, got %d", r.BurnRate)
	}
	if r.RunwayMonths < 12 {
		t.Errorf("expected long runway for profitable company with cash, got %d months", r.RunwayMonths)
	}
}

func TestCalculateHealth_Struggling(t *testing.T) {
	in := HealthInput{
		Revenue:           200000000, // 200M
		COGS:              180000000, // 180M
		OperatingExpenses: 50000000,  // 50M
		NetProfit:         -30000000, // -30M (loss)
		TotalAR:           100000000, // 100M — high relative to revenue
		TotalAP:           50000000,
		CashReserve:       20000000, // only 20M cash
		MonthlyExpenseAvg: 30000000,
		RevenueLastMonth:  25000000,
		RevenueThisMonth:  20000000, // declining
	}

	r := CalculateHealth(in)

	if r.RiskGrade != "C" && r.RiskGrade != "D" {
		t.Errorf("expected risk grade C or D for struggling company, got %s (score=%d)", r.RiskGrade, r.HealthScore)
	}
	if r.HealthScore > 59 {
		t.Errorf("expected health score <= 59 for struggling company, got %d", r.HealthScore)
	}
	if r.BurnRate != 30000000 {
		t.Errorf("expected burn rate 30M, got %d", r.BurnRate)
	}
	if r.NetMarginPct >= 0 {
		t.Errorf("expected negative net margin for struggling company, got %d%%", r.NetMarginPct)
	}
}

func TestCalculateHealth_ZeroRevenue(t *testing.T) {
	in := HealthInput{
		Revenue:           0,
		COGS:              0,
		OperatingExpenses: 0,
		NetProfit:         0,
		TotalAR:           0,
		TotalAP:           0,
		CashReserve:       0,
		MonthlyExpenseAvg: 0,
		RevenueLastMonth:  0,
		RevenueThisMonth:  0,
	}

	r := CalculateHealth(in)

	// Should not panic and should produce a low score
	if r.HealthScore > 50 {
		t.Errorf("expected low health score for zero-revenue company, got %d", r.HealthScore)
	}
	if r.GrossMarginPct != 0 {
		t.Errorf("expected 0%% gross margin, got %d%%", r.GrossMarginPct)
	}
	if r.NetMarginPct != 0 {
		t.Errorf("expected 0%% net margin, got %d%%", r.NetMarginPct)
	}
}
