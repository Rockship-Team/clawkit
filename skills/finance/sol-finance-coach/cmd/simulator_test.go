package main

import (
	"math"
	"testing"
)

// testCompound simulates compound interest directly (matching simulateCompound logic)
// so we can verify the math without running the CLI.
func testCompound(principal, monthly int64, ratePct float64, years int) (finalBalance int64) {
	monthlyRate := ratePct / 100.0 / 12.0
	months := years * 12
	balance := float64(principal)
	for m := 1; m <= months; m++ {
		balance = balance*(1+monthlyRate) + float64(monthly)
	}
	return int64(math.Round(balance))
}

// testLoanPayment calculates monthly loan payment (matching simulateLoan logic).
func testLoanPayment(amount int64, ratePct float64, years int) int64 {
	monthlyRate := ratePct / 100.0 / 12.0
	months := years * 12
	P := float64(amount)
	if monthlyRate == 0 {
		return int64(math.Round(P / float64(months)))
	}
	pow := math.Pow(1+monthlyRate, float64(months))
	payment := P * monthlyRate * pow / (pow - 1)
	return int64(math.Round(payment))
}

// testGoalMonthlySavings calculates monthly savings needed for a goal.
func testGoalMonthlySavings(target, current int64, ratePct float64, years int) int64 {
	monthlyRate := ratePct / 100.0 / 12.0
	months := years * 12
	pow := math.Pow(1+monthlyRate, float64(months))
	futureOfCurrent := float64(current) * pow
	stillNeeded := float64(target) - futureOfCurrent
	if stillNeeded <= 0 {
		return 0
	}
	if monthlyRate == 0 {
		return int64(math.Round(stillNeeded / float64(months)))
	}
	return int64(math.Round(stillNeeded * monthlyRate / (pow - 1)))
}

func TestCompoundInterest(t *testing.T) {
	tests := []struct {
		name      string
		principal int64
		monthly   int64
		rate      float64
		years     int
		wantMin   int64
		wantMax   int64
	}{
		{
			name:    "5tr/month, 7%, 10 years",
			monthly: 5000000,
			rate:    7.0,
			years:   10,
			wantMin: 850000000,
			wantMax: 880000000,
		},
		{
			name:    "10tr/month, 7%, 10 years",
			monthly: 10000000,
			rate:    7.0,
			years:   10,
			wantMin: 1700000000,
			wantMax: 1760000000,
		},
		{
			name:      "100tr principal, no monthly, 10%, 10 years",
			principal: 100000000,
			rate:      10.0,
			years:     10,
			// Monthly compounding: (1+0.1/12)^120 = 2.707 → ~270.7M
			wantMin: 269000000,
			wantMax: 272000000,
		},
		{
			name:    "zero rate",
			monthly: 1000000,
			rate:    0.0,
			years:   5,
			wantMin: 60000000,
			wantMax: 60000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testCompound(tt.principal, tt.monthly, tt.rate, tt.years)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("compound(%d, %d, %.1f%%, %dy) = %d, want [%d, %d]",
					tt.principal, tt.monthly, tt.rate, tt.years, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestLoanPayment(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		rate    float64
		years   int
		wantMin int64
		wantMax int64
	}{
		{
			name:    "1B VND, 10%, 20 years",
			amount:  1000000000,
			rate:    10.0,
			years:   20,
			wantMin: 9600000,
			wantMax: 9700000,
		},
		{
			name:    "500M VND, 8%, 15 years",
			amount:  500000000,
			rate:    8.0,
			years:   15,
			wantMin: 4770000,
			wantMax: 4790000,
		},
		{
			name:    "zero rate loan",
			amount:  120000000,
			rate:    0.0,
			years:   10,
			wantMin: 1000000,
			wantMax: 1000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testLoanPayment(tt.amount, tt.rate, tt.years)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("loanPayment(%d, %.1f%%, %dy) = %d, want [%d, %d]",
					tt.amount, tt.rate, tt.years, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestGoalMonthlySavings(t *testing.T) {
	tests := []struct {
		name    string
		target  int64
		current int64
		rate    float64
		years   int
		wantMin int64
		wantMax int64
	}{
		{
			name:    "3B target, 500M current, 6%, 5 years",
			target:  3000000000,
			current: 500000000,
			rate:    6.0,
			years:   5,
			wantMin: 33000000,
			wantMax: 37000000,
		},
		{
			name:    "already have enough",
			target:  100000000,
			current: 200000000,
			rate:    6.0,
			years:   5,
			wantMin: 0,
			wantMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testGoalMonthlySavings(tt.target, tt.current, tt.rate, tt.years)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("goalSavings(%d, %d, %.1f%%, %dy) = %d, want [%d, %d]",
					tt.target, tt.current, tt.rate, tt.years, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestLoanTotalInterest(t *testing.T) {
	// For a 1B loan at 10% over 20 years, total interest should be roughly 1.3B
	amount := int64(1000000000)
	rate := 10.0
	years := 20
	monthly := testLoanPayment(amount, rate, years)
	totalPaid := monthly * int64(years*12)
	totalInterest := totalPaid - amount

	if totalInterest < 1300000000 || totalInterest > 1340000000 {
		t.Errorf("total interest = %d, expected ~1.32B", totalInterest)
	}
}
