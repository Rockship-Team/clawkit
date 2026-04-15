package main

import (
	"math"
	"os"
	"strconv"
)

func cmdSimulate(args []string) {
	if len(args) == 0 {
		errOut("usage: simulate compound|loan|goal")
		os.Exit(1)
	}

	switch args[0] {
	case "compound":
		simulateCompound(args[1:])
	case "loan":
		simulateLoan(args[1:])
	case "goal":
		simulateGoal(args[1:])
	default:
		errOut("unknown simulate command: " + args[0])
		os.Exit(1)
	}
}

// simulateCompound calculates compound interest with monthly contributions.
// Args: <principal> <monthly> <annual_rate_pct> <years>
func simulateCompound(args []string) {
	if len(args) < 4 {
		errOut("usage: simulate compound <principal> <monthly> <rate_pct> <years>")
		os.Exit(1)
	}

	principal, _ := parseAmount(args[0])
	monthly, _ := parseAmount(args[1])
	ratePct, _ := strconv.ParseFloat(args[2], 64)
	years, _ := strconv.Atoi(args[3])

	if years <= 0 || years > 50 {
		errOut("years must be 1-50")
		os.Exit(1)
	}

	monthlyRate := ratePct / 100.0 / 12.0
	months := years * 12
	totalDeposited := float64(principal) + float64(monthly)*float64(months)

	// FV = P*(1+r)^n + M*((1+r)^n - 1)/r
	balance := float64(principal)
	var yearlyBreakdown []map[string]interface{}

	for m := 1; m <= months; m++ {
		balance = balance*(1+monthlyRate) + float64(monthly)
		if m%12 == 0 {
			y := m / 12
			deposited := float64(principal) + float64(monthly)*float64(m)
			interest := balance - deposited
			yearlyBreakdown = append(yearlyBreakdown, map[string]interface{}{
				"year":      y,
				"balance":   int64(math.Round(balance)),
				"deposited": int64(math.Round(deposited)),
				"interest":  int64(math.Round(interest)),
			})
		}
	}

	totalInterest := balance - totalDeposited
	multiplier := 1.0
	if totalDeposited > 0 {
		multiplier = balance / totalDeposited
	}

	okOut(map[string]interface{}{
		"principal":       principal,
		"monthly":         monthly,
		"annual_rate_pct": ratePct,
		"years":           years,
		"final_balance":   int64(math.Round(balance)),
		"total_deposited": int64(math.Round(totalDeposited)),
		"total_interest":  int64(math.Round(totalInterest)),
		"multiplier":      math.Round(multiplier*100) / 100,
		"yearly":          yearlyBreakdown,
	})
}

// simulateLoan calculates monthly payment for a fixed-rate loan.
// Args: <loan_amount> <annual_rate_pct> <years>
func simulateLoan(args []string) {
	if len(args) < 3 {
		errOut("usage: simulate loan <amount> <rate_pct> <years>")
		os.Exit(1)
	}

	amount, _ := parseAmount(args[0])
	ratePct, _ := strconv.ParseFloat(args[1], 64)
	years, _ := strconv.Atoi(args[2])

	if years <= 0 || years > 50 {
		errOut("years must be 1-50")
		os.Exit(1)
	}

	monthlyRate := ratePct / 100.0 / 12.0
	months := years * 12
	P := float64(amount)

	// Monthly payment = P * r * (1+r)^n / ((1+r)^n - 1)
	var monthlyPayment float64
	if monthlyRate == 0 {
		monthlyPayment = P / float64(months)
	} else {
		pow := math.Pow(1+monthlyRate, float64(months))
		monthlyPayment = P * monthlyRate * pow / (pow - 1)
	}

	totalPayment := monthlyPayment * float64(months)
	totalInterest := totalPayment - P

	// Yearly amortization
	balance := P
	var yearlyBreakdown []map[string]interface{}
	for y := 1; y <= years; y++ {
		yearPrincipal := 0.0
		yearInterest := 0.0
		for m := 0; m < 12; m++ {
			interest := balance * monthlyRate
			principal := monthlyPayment - interest
			yearPrincipal += principal
			yearInterest += interest
			balance -= principal
		}
		if balance < 0 {
			balance = 0
		}
		yearlyBreakdown = append(yearlyBreakdown, map[string]interface{}{
			"year":              y,
			"remaining_balance": int64(math.Round(balance)),
			"principal_paid":    int64(math.Round(yearPrincipal)),
			"interest_paid":     int64(math.Round(yearInterest)),
		})
	}

	okOut(map[string]interface{}{
		"loan_amount":     amount,
		"annual_rate_pct": ratePct,
		"years":           years,
		"monthly_payment": int64(math.Round(monthlyPayment)),
		"total_payment":   int64(math.Round(totalPayment)),
		"total_interest":  int64(math.Round(totalInterest)),
		"yearly":          yearlyBreakdown,
	})
}

// simulateGoal calculates monthly savings needed to reach a target.
// Args: <target> <years> [current_savings]
func simulateGoal(args []string) {
	if len(args) < 2 {
		errOut("usage: simulate goal <target> <years> [current]")
		os.Exit(1)
	}

	target, _ := parseAmount(args[0])
	years, _ := strconv.Atoi(args[1])
	current := int64(0)
	if len(args) > 2 {
		current, _ = parseAmount(args[2])
	}

	if years <= 0 || years > 50 {
		errOut("years must be 1-50")
		os.Exit(1)
	}

	needed := float64(target - current)
	if needed <= 0 {
		okOut(map[string]interface{}{
			"target":  target,
			"current": current,
			"message": "You already have enough!",
		})
		return
	}

	months := years * 12

	// Scenarios with different rates
	rates := []struct {
		Name string
		Rate float64
	}{
		{"Gui tiet kiem", 6.0},
		{"Quy trai phieu", 8.0},
		{"Ket hop (50/50 tiet kiem + quy co phieu)", 10.0},
	}

	var scenarios []map[string]interface{}
	for _, r := range rates {
		monthlyRate := r.Rate / 100.0 / 12.0
		// Solve for monthly contribution M:
		// needed = FV(current, rate, months) + M * ((1+r)^n - 1)/r - current*(1+r)^n
		// Actually: target = current*(1+r)^n + M*((1+r)^n - 1)/r
		// So M = (target - current*(1+r)^n) * r / ((1+r)^n - 1)
		pow := math.Pow(1+monthlyRate, float64(months))
		futureOfCurrent := float64(current) * pow
		stillNeeded := float64(target) - futureOfCurrent

		var monthlySavings float64
		if stillNeeded <= 0 {
			monthlySavings = 0
		} else if monthlyRate == 0 {
			monthlySavings = stillNeeded / float64(months)
		} else {
			monthlySavings = stillNeeded * monthlyRate / (pow - 1)
		}

		scenarios = append(scenarios, map[string]interface{}{
			"name":            r.Name,
			"annual_rate_pct": r.Rate,
			"monthly_savings": int64(math.Round(monthlySavings)),
			"total_deposited": int64(math.Round(monthlySavings*float64(months))) + current,
			"interest_earned": int64(math.Round(float64(target) - monthlySavings*float64(months) - float64(current))),
		})
	}

	okOut(map[string]interface{}{
		"target":    target,
		"current":   current,
		"years":     years,
		"needed":    int64(needed),
		"scenarios": scenarios,
	})
}
