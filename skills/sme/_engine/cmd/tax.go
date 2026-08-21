package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

// Vietnam PIT brackets (progressive tax)
type pitBracket struct {
	Upper int64   // cumulative upper limit
	Rate  float64 // percentage
}

var pitBrackets = []pitBracket{
	{5000000, 5},
	{10000000, 10},
	{18000000, 15},
	{32000000, 20},
	{52000000, 25},
	{80000000, 30},
	{math.MaxInt64, 35},
}

const (
	selfDeduction      = 11000000
	dependentDeduction = 4400000
	bhxhRate           = 0.08
	bhytRate           = 0.015
	bhtnRate           = 0.01
)

func cmdTax(args []string) {
	if len(args) == 0 {
		errOut("usage: tax calendar|pit|pit-contractor|vat|deadlines|add-deadline")
	}
	org := defaultOrgID()

	switch args[0] {
	case "pit":
		// tax pit <gross_salary> [insurance_salary] [dependents] [allowances]
		if len(args) < 2 {
			errOut("usage: tax pit <gross> [ins_salary] [dependents] [allowances]")
		}
		gross := parseVND(args[1])
		insSalary := gross
		if len(args) > 2 {
			insSalary = parseVND(args[2])
		}
		deps := 0
		if len(args) > 3 {
			deps, _ = strconv.Atoi(args[3])
		}
		allowances := int64(0)
		if len(args) > 4 {
			allowances = parseVND(args[4])
		}

		// Insurance deductions (employee share)
		bhxh := int64(float64(insSalary) * bhxhRate)
		bhyt := int64(float64(insSalary) * bhytRate)
		bhtn := int64(float64(insSalary) * bhtnRate)
		insurance := bhxh + bhyt + bhtn

		// Gross income
		grossIncome := gross + allowances

		// Deductions
		selfDed := int64(selfDeduction)
		depDed := int64(dependentDeduction) * int64(deps)
		totalDed := insurance + selfDed + depDed

		// Taxable income
		taxable := grossIncome - totalDed
		if taxable < 0 {
			taxable = 0
		}

		// Progressive calculation
		type bracketCalc struct {
			Rate    float64 `json:"rate"`
			Taxable int64   `json:"taxable"`
			Tax     int64   `json:"tax"`
		}
		var breakdown []bracketCalc
		remaining := taxable
		prevLimit := int64(0)
		totalTax := int64(0)

		for _, b := range pitBrackets {
			if remaining <= 0 {
				break
			}
			bracketSize := b.Upper - prevLimit
			if b.Upper == math.MaxInt64 {
				bracketSize = remaining
			}
			if remaining < bracketSize {
				bracketSize = remaining
			}
			tax := int64(float64(bracketSize) * b.Rate / 100)
			totalTax += tax
			breakdown = append(breakdown, bracketCalc{b.Rate, bracketSize, tax})
			remaining -= bracketSize
			prevLimit = b.Upper
		}

		effectiveRate := 0.0
		if grossIncome > 0 {
			effectiveRate = float64(totalTax) / float64(grossIncome) * 100
		}

		netPay := grossIncome - insurance - totalTax

		okOut(map[string]interface{}{
			"gross_income":        grossIncome,
			"insurance_deduction": insurance,
			"bhxh":                bhxh,
			"bhyt":                bhyt,
			"bhtn":                bhtn,
			"self_deduction":      selfDed,
			"dependent_deduction": depDed,
			"total_deductions":    totalDed,
			"taxable_income":      taxable,
			"pit_amount":          totalTax,
			"effective_rate_pct":  math.Round(effectiveRate*100) / 100,
			"net_pay":             netPay,
			"breakdown":           breakdown,
		})

	case "pit-contractor":
		// Flat 10% if >= 2,000,000
		if len(args) < 2 {
			errOut("usage: tax pit-contractor <payment_amount>")
		}
		amt := parseVND(args[1])
		tax := int64(0)
		if amt >= 2000000 {
			tax = amt / 10
		}
		okOut(map[string]interface{}{"payment": amt, "pit": tax, "net": amt - tax, "rate": "10%"})

	case "calendar":
		// Show upcoming tax deadlines
		rows, _ := queryRows(`SELECT id,tax_type,period_type,period_label,deadline_date,status,amount_due
			FROM tax_deadlines WHERE org_id=? AND deadline_date >= ? ORDER BY deadline_date LIMIT 20`,
			org, vnToday())
		okOut(map[string]interface{}{"deadlines": rows, "count": len(rows)})

	case "deadlines":
		// Show all deadlines with status
		status := "all"
		if len(args) > 1 {
			status = args[1]
		}
		q := "SELECT * FROM tax_deadlines WHERE org_id=?"
		qargs := []interface{}{org}
		if status != "all" {
			q += " AND status=?"
			qargs = append(qargs, status)
		}
		q += " ORDER BY deadline_date LIMIT 50"
		rows, _ := queryRows(q, qargs...)
		okOut(map[string]interface{}{"deadlines": rows, "count": len(rows)})

	case "add-deadline":
		// tax add-deadline <type> <period_type> <period_label> <date> [amount]
		if len(args) < 5 {
			errOut("usage: tax add-deadline <vat|cit|pit|license_fee> <monthly|quarterly|annually> <label> <date> [amount]")
		}
		id := newID()
		amt := int64(0)
		if len(args) > 5 {
			amt = parseVND(args[5])
		}
		_, err := exec(`INSERT INTO tax_deadlines (id,org_id,tax_type,period_type,period_label,deadline_date,amount_due,status,created_at)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			id, org, args[1], args[2], args[3], args[4], amt, "upcoming", vnNowISO())
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"id": id, "tax_type": args[1], "deadline": args[4]})

	case "vat":
		// Prepare VAT summary for a period
		period := time.Now().Format("2006-01")
		if len(args) > 1 {
			period = args[1]
		}
		outRow, _ := queryOne(`SELECT COALESCE(SUM(vat_amount),0) as vat_out, COALESCE(SUM(subtotal),0) as sales
			FROM invoices WHERE org_id=? AND direction='outbound' AND status NOT IN ('cancelled','draft')
			AND substr(issued_date,1,7)=?`, org, period)
		inRow, _ := queryOne(`SELECT COALESCE(SUM(vat_amount),0) as vat_in, COALESCE(SUM(subtotal),0) as purchases
			FROM invoices WHERE org_id=? AND direction='inbound' AND status NOT IN ('cancelled','draft')
			AND substr(issued_date,1,7)=?`, org, period)

		vatOut := int64(0)
		vatIn := int64(0)
		sales := int64(0)
		purchases := int64(0)
		if outRow != nil {
			vatOut, _ = outRow["vat_out"].(int64)
			sales, _ = outRow["sales"].(int64)
		}
		if inRow != nil {
			vatIn, _ = inRow["vat_in"].(int64)
			purchases, _ = inRow["purchases"].(int64)
		}
		vatDue := vatOut - vatIn

		// Save calculation
		calcID := newID()
		input, _ := json.Marshal(map[string]interface{}{"period": period, "vat_out": vatOut, "vat_in": vatIn})
		breakdown, _ := json.Marshal(map[string]interface{}{"sales": sales, "purchases": purchases, "vat_output": vatOut, "vat_input": vatIn, "vat_payable": vatDue})
		exec(`INSERT INTO tax_calculations (id,org_id,tax_type,period_label,input_data,calculated_amount,calculation_breakdown,calculated_at)
			VALUES (?,?,?,?,?,?,?,?)`, calcID, org, "vat", period, string(input), vatDue, string(breakdown), vnNowISO())

		okOut(map[string]interface{}{
			"period":      period,
			"sales":       sales,
			"purchases":   purchases,
			"vat_output":  vatOut,
			"vat_input":   vatIn,
			"vat_payable": vatDue,
			"calc_id":     calcID,
		})

	case "seed-calendar":
		// Seed standard VN tax deadlines for a year
		year := time.Now().Year()
		if len(args) > 1 {
			year, _ = strconv.Atoi(args[1])
		}
		count := 0
		// Monthly VAT: 20th of next month
		for m := 1; m <= 12; m++ {
			deadline := fmt.Sprintf("%d-%02d-20", year, m+1)
			if m == 12 {
				deadline = fmt.Sprintf("%d-01-20", year+1)
			}
			label := fmt.Sprintf("T%02d/%d", m, year)
			exec(`INSERT OR IGNORE INTO tax_deadlines (id,org_id,tax_type,period_type,period_label,deadline_date,status,created_at)
				VALUES (?,?,?,?,?,?,?,?)`, newID(), org, "vat", "monthly", label, deadline, "upcoming", vnNowISO())
			count++
		}
		// Quarterly CIT: 30th of next month after quarter end
		for q := 1; q <= 4; q++ {
			m := q*3 + 1
			if m > 12 {
				m = 1
			}
			y := year
			if q == 4 {
				y = year + 1
			}
			deadline := fmt.Sprintf("%d-%02d-30", y, m)
			label := fmt.Sprintf("Q%d/%d", q, year)
			exec(`INSERT OR IGNORE INTO tax_deadlines (id,org_id,tax_type,period_type,period_label,deadline_date,status,created_at)
				VALUES (?,?,?,?,?,?,?,?)`, newID(), org, "cit", "quarterly", label, deadline, "upcoming", vnNowISO())
			count++
		}
		// Annual PIT finalization: March 31
		exec(`INSERT OR IGNORE INTO tax_deadlines (id,org_id,tax_type,period_type,period_label,deadline_date,status,created_at)
			VALUES (?,?,?,?,?,?,?,?)`, newID(), org, "pit", "annually", fmt.Sprintf("Nam %d", year), fmt.Sprintf("%d-03-31", year+1), "upcoming", vnNowISO())
		count++
		// License fee: Jan 30
		exec(`INSERT OR IGNORE INTO tax_deadlines (id,org_id,tax_type,period_type,period_label,deadline_date,status,created_at)
			VALUES (?,?,?,?,?,?,?,?)`, newID(), org, "license_fee", "annually", fmt.Sprintf("Nam %d", year), fmt.Sprintf("%d-01-30", year), "upcoming", vnNowISO())
		count++

		okOut(map[string]interface{}{"year": year, "deadlines_created": count})

	default:
		errOut("unknown tax command: " + args[0])
	}
}
