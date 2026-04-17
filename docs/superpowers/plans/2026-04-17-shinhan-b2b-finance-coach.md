# Shinhan B2B Finance Coach Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a B2B Finance Coach skill for Shinhan that helps SME owners manage cashflow, receivables, payables, discount strategy, and financial health — while surfacing Shinhan product recommendations based on real-time financial signals.

**Architecture:** A new clawkit skill `shinhan-b2b-coach` under `skills/finance/` with a Go CLI (`b2b-cli`) that manages 9 SQLite tables (company profile, transactions, cashflow forecasts, receivables, payables, discount strategies, bank product recommendations, health metrics, conversations). The SKILL.md prompt covers all 12 skill capabilities from the spec organized as 5 groups (Cashflow Intelligence, Revenue Strategy, Financial Health, Bank Product Intelligence, Banker Dashboard). The CLI uses the same patterns as sol-finance-coach (JSON output, VND parsing, Vietnam timezone).

**Tech Stack:** Go 1.22 (CLI binary), SQLite via modernc.org/sqlite, clawkit skill system (SKILL.md + config.json + schema.json)

---

## Scope Note

The spec defines 12 sub-skills across 5 groups. Rather than 12 separate clawkit skills, we implement **one skill** (`shinhan-b2b-coach`) with a single Go CLI binary (`b2b-cli`) that handles all 12 capabilities via subcommands. This matches the pattern established by `sol-finance-coach` (one skill, one CLI, multiple capabilities).

The 12 capabilities map to CLI command groups:

| Spec Skill                  | CLI Command                                         | Task   |
| --------------------------- | --------------------------------------------------- | ------ |
| A1 biz-cashflow-forecaster  | `b2b-cli cashflow forecast\|weekly\|gap`            | Task 4 |
| A2 biz-ar-optimizer         | `b2b-cli ar add\|list\|aging\|remind\|score`        | Task 5 |
| A3 biz-ap-strategist        | `b2b-cli ap add\|list\|schedule\|discount-roi`      | Task 5 |
| B1 biz-discount-advisor     | `b2b-cli discount analyze\|simulate\|list`          | Task 6 |
| B2 biz-pricing-analyzer     | `b2b-cli pricing analyze`                           | Task 6 |
| C1 biz-health-dashboard     | `b2b-cli health calculate\|show\|history`           | Task 7 |
| C2 biz-report-generator     | `b2b-cli report pnl\|cashflow\|aging\|summary`      | Task 7 |
| C3 biz-tax-estimator        | `b2b-cli tax estimate-vat\|estimate-cit\|deadlines` | Task 7 |
| D1 biz-product-recommender  | `b2b-cli recommend evaluate\|list\|update`          | Task 8 |
| D2 biz-loan-readiness       | `b2b-cli recommend loan-readiness`                  | Task 8 |
| E1 banker-intelligence-feed | `b2b-cli banker portfolio\|pipeline\|alerts`        | Task 8 |

---

## File Structure

### New files to create

```
skills/finance/shinhan-b2b-coach/
  SKILL.md                          # AI prompt covering all 12 capabilities
  config.json                       # Clawkit metadata
  schema.json                       # 9 B2B tables from spec
  bootstrap-files/
    IDENTITY.md                     # Shinhan B2B Coach persona
  cmd/
    go.mod                          # Module: b2b-cli, dep: modernc.org/sqlite
    main.go                         # CLI dispatcher
    store.go                        # DB open, helpers, VND parsing, time
    schema.go                       # cmdInit — create all 9 tables
    company.go                      # company add|get|update|onboard
    transactions.go                 # txn add|list|report|import
    cashflow.go                     # cashflow forecast|weekly|gap
    arap.go                         # ar/ap add|list|aging|remind|score|schedule|discount-roi
    discount.go                     # discount analyze|simulate|list + pricing analyze
    health.go                       # health calculate|show|history + report commands + tax estimate
    recommend.go                    # recommend evaluate|list|update|loan-readiness + banker commands
    health_score.go                 # Health score calculation (pure logic, testable)
    health_score_test.go            # Tests for health score
  data/
    shinhan_products.json           # Shinhan product catalog (name, rate, min/max, description)
    industry_benchmarks.json        # VN SME benchmarks by industry (margins, DSO, DPO)
    tax_calendar_b2b.json           # Business tax deadlines (VAT monthly, CIT quarterly)
```

### Files to modify

```
skills/skills.go:21                 # Verify all:finance is in embed (already present)
```

---

## Task 1: Go Module + Store Layer + Schema

**Files:**

- Create: `skills/finance/shinhan-b2b-coach/cmd/go.mod`
- Create: `skills/finance/shinhan-b2b-coach/cmd/main.go`
- Create: `skills/finance/shinhan-b2b-coach/cmd/store.go`
- Create: `skills/finance/shinhan-b2b-coach/cmd/schema.go`

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p skills/finance/shinhan-b2b-coach/{cmd,data,bootstrap-files}
```

- [ ] **Step 2: Write go.mod**

```go
// skills/finance/shinhan-b2b-coach/cmd/go.mod
module b2b-cli

go 1.22

require modernc.org/sqlite v1.34.5
```

- [ ] **Step 3: Write store.go — shared helpers**

```go
// skills/finance/shinhan-b2b-coach/cmd/store.go
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func skillDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openclaw", "workspace", "skills", "shinhan-b2b-coach")
}

func dbPath() string { return filepath.Join(skillDir(), "b2b.db") }

func openDB() (*sql.DB, error) {
	if db != nil {
		return db, nil
	}
	os.MkdirAll(skillDir(), 0o755)
	var err error
	db, err = sql.Open("sqlite", dbPath()+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.Exec("PRAGMA foreign_keys = ON")
	return db, nil
}

func mustDB() *sql.DB {
	d, err := openDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
	return d
}

// --- JSON output ---
func jsonOut(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func okOut(extra map[string]interface{}) {
	out := map[string]interface{}{"ok": true}
	for k, v := range extra {
		out[k] = v
	}
	jsonOut(out)
}

func errOut(msg string) {
	jsonOut(map[string]interface{}{"ok": false, "error": msg})
	os.Exit(1)
}

// --- Time ---
func vnNow() time.Time {
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	if loc == nil {
		return time.Now().UTC().Add(7 * time.Hour)
	}
	return time.Now().In(loc)
}

func vnToday() string  { return vnNow().Format("2006-01-02") }
func vnNowISO() string { return vnNow().Format("2006-01-02T15:04:05+07:00") }

func newID() string {
	b := make([]byte, 16)
	n := time.Now().UnixNano()
	for i := range b {
		b[i] = byte(n >> (i * 4))
		n = n*6364136223846793005 + 1442695040888963407
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// --- VND parsing (50tr, 1.5ty, 200k, 50.000.000) ---
func parseVND(s string) int64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimRight(s, "đdVNDvnd ")
	// Dots as thousands separator: "50.000.000" → "50000000"
	if strings.Count(s, ".") >= 2 {
		s = strings.ReplaceAll(s, ".", "")
	}
	if strings.HasSuffix(s, "ty") {
		s = strings.TrimSuffix(s, "ty")
		var v float64
		fmt.Sscanf(s, "%f", &v)
		return int64(v * 1e9)
	}
	if strings.HasSuffix(s, "tr") || strings.HasSuffix(s, "trieu") {
		s = strings.TrimSuffix(s, "trieu")
		s = strings.TrimSuffix(s, "tr")
		var v float64
		fmt.Sscanf(s, "%f", &v)
		return int64(v * 1e6)
	}
	if strings.HasSuffix(s, "k") || strings.HasSuffix(s, "K") {
		s = s[:len(s)-1]
		var v float64
		fmt.Sscanf(s, "%f", &v)
		return int64(v * 1e3)
	}
	var v int64
	fmt.Sscanf(s, "%d", &v)
	return v
}

// --- Query helpers ---
func queryRows(q string, args ...interface{}) ([]map[string]interface{}, error) {
	d := mustDB()
	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)
		row := map[string]interface{}{}
		for i, c := range cols {
			if b, ok := vals[i].([]byte); ok {
				row[c] = string(b)
			} else {
				row[c] = vals[i]
			}
		}
		out = append(out, row)
	}
	return out, nil
}

func queryOne(q string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := queryRows(q, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func exec(q string, args ...interface{}) (sql.Result, error) {
	return mustDB().Exec(q, args...)
}
```

- [ ] **Step 4: Write schema.go — cmdInit with all 9 tables**

Create all 9 tables from the spec's schema section (company_profile, business_transactions, cashflow_forecast, receivables, payables, discount_strategies, bank_product_recommendations, business_health_metrics, coach_conversations_b2b). Use `CREATE TABLE IF NOT EXISTS`. All UUIDs as TEXT, timestamps as TEXT, amounts as INTEGER (VND). Add appropriate indexes.

After creating tables, seed a default company profile if none exists.

- [ ] **Step 5: Write main.go — dispatcher with stubs**

Dispatch to: init, company, txn, cashflow, ar, ap, discount, pricing, health, report, tax, recommend, banker. Create stub functions that call `errOut("not implemented")` for all commands except `cmdInit`.

- [ ] **Step 6: Run go mod tidy + build**

```bash
cd skills/finance/shinhan-b2b-coach/cmd && go mod tidy && go build -o ../b2b-cli .
```

- [ ] **Step 7: Test init**

```bash
../b2b-cli init
# Expected: {"ok":true, "database":"...", "tables":9}
```

- [ ] **Step 8: Commit**

```bash
git add skills/finance/shinhan-b2b-coach/
git commit -m "feat(shinhan-b2b): foundation — go module, store layer, 9-table schema"
```

---

## Task 2: Company Profile + Business Transactions

**Files:**

- Create: `skills/finance/shinhan-b2b-coach/cmd/company.go`
- Create: `skills/finance/shinhan-b2b-coach/cmd/transactions.go`
- Modify: `skills/finance/shinhan-b2b-coach/cmd/main.go` (replace stubs)

- [ ] **Step 1: Write company.go**

`cmdCompany(args)` subcommands:

- `add <name> <industry> [tax_code] [employee_count] [monthly_revenue_avg]` — create company profile
- `get` — show current company profile
- `update <field> <value>` — update profile field (name, industry, employee_count, monthly_revenue_avg, monthly_expense_avg, cash_reserve, accounting_software, fiscal_year_start)
- `onboard` — mark company as onboarded

The company_id should be auto-generated on first `add` and stored. Subsequent commands use the stored company_id (single-tenant: one company per installation).

- [ ] **Step 2: Write transactions.go**

`cmdTxn(args)` subcommands:

- `add <date> <type> <direction> <counterparty> <amount> <category> [invoice_number] [due_date] [note]`
  - type: `sale`, `purchase`, `salary`, `tax`, `rent`, `utility`, `other`
  - direction: `in`, `out`
  - category: `revenue`, `cogs`, `opex`, `payroll`, `tax`, `rent`, `other`
- `list [period] [direction] [category]` — filter by YYYY-MM period, in/out, category
- `report <period>` — monthly P&L summary: total revenue, COGS, gross margin, opex, net profit
- `import <csv_path>` — bulk import from CSV (date,type,direction,counterparty,amount,category)

All amounts in VND using `parseVND()`. Dates as YYYY-MM-DD.

- [ ] **Step 3: Update main.go — replace company and txn stubs**

- [ ] **Step 4: Build + test**

```bash
go build -o ../b2b-cli .
../b2b-cli company add "ABC Trading" "wholesale" "0312345678" 25 1200000000
../b2b-cli company get
../b2b-cli txn add 2026-04-01 sale in "Customer X" 150tr revenue INV-001 2026-04-30
../b2b-cli txn add 2026-04-05 purchase out "Vendor Y" 80tr cogs PO-001
../b2b-cli txn list 2026-04
../b2b-cli txn report 2026-04
```

- [ ] **Step 5: Commit**

```bash
git add skills/finance/shinhan-b2b-coach/cmd/company.go skills/finance/shinhan-b2b-coach/cmd/transactions.go
git commit -m "feat(shinhan-b2b): company profile + business transactions with P&L report"
```

---

## Task 3: Receivables + Payables (AR/AP)

**Files:**

- Create: `skills/finance/shinhan-b2b-coach/cmd/arap.go`
- Modify: `skills/finance/shinhan-b2b-coach/cmd/main.go`

- [ ] **Step 1: Write arap.go**

`cmdAR(args)` subcommands:

- `add <customer> <amount> <due_date> [invoice_number] [issued_date]` — add receivable
- `list [outstanding|overdue|paid|all]` — filter by status
- `aging` — aging report: current (0-30), 31-60, 61-90, >90 days buckets with totals
- `score <customer>` — payment score based on history: avg days to pay, late frequency, collection probability
- `remind <id>` — mark receivable for collection reminder, output reminder text
- `pay <id> [paid_date]` — mark as paid

`cmdAP(args)` subcommands:

- `add <vendor> <amount> <due_date> [invoice_number] [early_pay_discount_pct] [early_pay_deadline]`
- `list [outstanding|overdue|paid|all]`
- `schedule` — payment schedule: which to pay now, which to defer (sorted by due_date, flagging early-pay opportunities)
- `discount-roi <id>` — calculate annualized return of early-payment discount: `discount_pct / (remaining_days / 365) * 100`. If ROI > 20%, recommend taking it.
- `pay <id> [paid_date]`

Both should auto-calculate `days_overdue` and `aging_bucket` on list/aging queries.

- [ ] **Step 2: Update main.go stubs**

- [ ] **Step 3: Build + test**

```bash
go build -o ../b2b-cli .
../b2b-cli ar add "Customer Alpha" 200tr 2026-04-15 INV-101
../b2b-cli ar add "Customer Beta" 150tr 2026-03-01 INV-088
../b2b-cli ar list outstanding
../b2b-cli ar aging
../b2b-cli ap add "Vendor Gamma" 80tr 2026-04-20 PO-055 2 2026-04-10
../b2b-cli ap discount-roi 1
../b2b-cli ap schedule
```

- [ ] **Step 4: Commit**

```bash
git commit -am "feat(shinhan-b2b): AR/AP management with aging, scoring, early-pay ROI"
```

---

## Task 4: Cashflow Forecaster

**Files:**

- Create: `skills/finance/shinhan-b2b-coach/cmd/cashflow.go`

- [ ] **Step 1: Write cashflow.go**

`cmdCashflow(args)` subcommands:

- `forecast [days]` — 30/60/90 day forecast with 3 scenarios (optimistic/base/pessimistic):
  1. Query receivables (outstanding) as expected inflows — apply collection_probability
  2. Query payables (outstanding) as expected outflows
  3. Get company cash_reserve as opening balance
  4. Estimate recurring costs from transaction history (avg monthly payroll, rent, utilities)
  5. Optimistic: 100% collection, no unexpected costs
  6. Base: historical collection rate, normal costs
  7. Pessimistic: 70% collection, 10% cost overrun
  8. Calculate closing balance per scenario. If any scenario < 0, set gap_alert=true
  9. Save to cashflow_forecast table

- `weekly` — "Tuan thanh toan" report for current week:
  1. What's due to come in (AR due this week)
  2. What's due to go out (AP due this week + recurring costs)
  3. Net for the week
  4. Warnings about customers likely to pay late (from AR score)

- `gap` — show all forecasts where gap_alert=true, sorted by severity

Each forecast stores `line_items` as JSON array with individual inflow/outflow entries.

- [ ] **Step 2: Build + test**

```bash
go build -o ../b2b-cli .
../b2b-cli cashflow forecast 60
../b2b-cli cashflow weekly
../b2b-cli cashflow gap
```

- [ ] **Step 3: Commit**

```bash
git commit -am "feat(shinhan-b2b): cashflow forecaster — 3-scenario forecast, weekly report, gap alerts"
```

---

## Task 5: Discount Strategy + Pricing

**Files:**

- Create: `skills/finance/shinhan-b2b-coach/cmd/discount.go`

- [ ] **Step 1: Write discount.go**

`cmdDiscount(args)` subcommands:

- `analyze` — analyze transaction data to suggest discount strategies:
  1. Calculate current gross margin from transactions
  2. Segment customers by revenue (top 20% = "A", next 30% = "B", rest = "C")
  3. Identify seasonal dips from historical monthly revenue
  4. Suggest discount strategies with projected impact (save to discount_strategies table)

- `simulate <discount_pct> <target_segment> [volume_increase_pct]` — what-if simulator:
  1. Take current revenue for segment
  2. Apply discount
  3. Apply estimated volume increase
  4. Calculate: new revenue, new margin, net impact vs. no discount
  5. Save as proposed strategy

- `list [proposed|active|all]` — list strategies by status

`cmdPricing(args)` subcommands:

- `analyze` — analyze pricing by product/service category:
  1. Group transactions by category
  2. Calculate: avg order value, transaction count, total revenue, margin contribution
  3. Flag categories with declining order frequency (potential overpricing)
  4. Flag categories with growing volume but low margin (potential underpricing)

- [ ] **Step 2: Build + test**

```bash
go build -o ../b2b-cli .
../b2b-cli discount analyze
../b2b-cli discount simulate 8 A 15
../b2b-cli pricing analyze
```

- [ ] **Step 3: Commit**

```bash
git commit -am "feat(shinhan-b2b): discount advisor + pricing analyzer"
```

---

## Task 6: Health Score + Reports + Tax

**Files:**

- Create: `skills/finance/shinhan-b2b-coach/cmd/health.go`
- Create: `skills/finance/shinhan-b2b-coach/cmd/health_score.go`
- Create: `skills/finance/shinhan-b2b-coach/cmd/health_score_test.go`

- [ ] **Step 1: Write health_score.go — pure calculation logic**

```go
// skills/finance/shinhan-b2b-coach/cmd/health_score.go
package main

// HealthInput contains the raw metrics for health score calculation.
type HealthInput struct {
	Revenue           int64
	COGS              int64
	OperatingExpenses int64
	NetProfit         int64
	TotalAR           int64 // outstanding receivables
	TotalAP           int64 // outstanding payables
	CashReserve       int64
	MonthlyExpenseAvg int64
	RevenueLastMonth  int64
	RevenueThisMonth  int64
	Revenue3MonthsAgo int64
	RevenueStdDev     float64 // revenue volatility
	TopCustomerPct    float64 // % of revenue from top customer
}

// HealthResult is the calculated health metrics.
type HealthResult struct {
	GrossMarginPct       int   `json:"gross_margin_pct"`
	NetMarginPct         int   `json:"net_margin_pct"`
	DSODays              int   `json:"dso_days"`
	DPODays              int   `json:"dpo_days"`
	CashConversionCycle  int   `json:"cash_conversion_cycle"`
	BurnRate             int64 `json:"burn_rate"`
	RunwayMonths         int   `json:"runway_months"`
	HealthScore          int   `json:"health_score"`
	RiskGrade            string `json:"risk_grade"`
	ProfitabilityScore   int   `json:"profitability_score"`
	LiquidityScore       int   `json:"liquidity_score"`
	EfficiencyScore      int   `json:"efficiency_score"`
	GrowthScore          int   `json:"growth_score"`
	StabilityScore       int   `json:"stability_score"`
}

// CalculateHealth computes all metrics and the composite health score.
func CalculateHealth(in HealthInput) HealthResult {
	r := HealthResult{}

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
	// CCC = DSO - DPO (simplified, no DIO)
	r.CashConversionCycle = r.DSODays - r.DPODays

	// Burn rate & runway
	if in.NetProfit < 0 {
		r.BurnRate = -in.NetProfit
		if r.BurnRate > 0 {
			r.RunwayMonths = int(in.CashReserve / r.BurnRate)
		}
	} else {
		r.RunwayMonths = 99 // profitable = infinite runway
	}

	// --- Component scores (each 0-100) ---

	// Profitability (25%): net margin vs benchmark (10% good, 20% excellent)
	r.ProfitabilityScore = clamp(r.NetMarginPct*5, 0, 100) // 20% margin = 100

	// Liquidity (25%): based on runway
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

	// Efficiency (20%): DSO (lower = better, benchmark ~30 days)
	if r.DSODays <= 15 {
		r.EfficiencyScore = 100
	} else if r.DSODays <= 30 {
		r.EfficiencyScore = 80
	} else if r.DSODays <= 45 {
		r.EfficiencyScore = 60
	} else if r.DSODays <= 60 {
		r.EfficiencyScore = 40
	} else {
		r.EfficiencyScore = 20
	}

	// Growth (15%): month-over-month revenue trend
	if in.RevenueLastMonth > 0 && in.RevenueThisMonth > 0 {
		growthPct := int((in.RevenueThisMonth - in.RevenueLastMonth) * 100 / in.RevenueLastMonth)
		r.GrowthScore = clamp(50+growthPct*2, 0, 100) // 0% growth = 50, +25% = 100
	} else {
		r.GrowthScore = 50 // neutral if no data
	}

	// Stability (15%): low volatility + low concentration
	r.StabilityScore = 50 // default
	if in.RevenueStdDev > 0 && in.Revenue > 0 {
		cv := in.RevenueStdDev / float64(in.Revenue) // coefficient of variation
		if cv < 0.1 {
			r.StabilityScore = 90
		} else if cv < 0.2 {
			r.StabilityScore = 70
		} else if cv < 0.4 {
			r.StabilityScore = 40
		} else {
			r.StabilityScore = 20
		}
	}
	if in.TopCustomerPct > 50 {
		r.StabilityScore -= 20 // concentration risk penalty
	}
	r.StabilityScore = clamp(r.StabilityScore, 0, 100)

	// Composite score: weighted average
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

func clamp(v, min, max int) int {
	if v < min { return min }
	if v > max { return max }
	return v
}
```

- [ ] **Step 2: Write health_score_test.go**

```go
// skills/finance/shinhan-b2b-coach/cmd/health_score_test.go
package main

import "testing"

func TestCalculateHealth_Profitable(t *testing.T) {
	r := CalculateHealth(HealthInput{
		Revenue: 1000000000, COGS: 650000000, OperatingExpenses: 200000000,
		NetProfit: 150000000, TotalAR: 200000000, TotalAP: 100000000,
		CashReserve: 500000000, MonthlyExpenseAvg: 80000000,
		RevenueLastMonth: 900000000, RevenueThisMonth: 1000000000,
	})
	if r.GrossMarginPct != 35 {
		t.Errorf("gross margin = %d, want 35", r.GrossMarginPct)
	}
	if r.NetMarginPct != 15 {
		t.Errorf("net margin = %d, want 15", r.NetMarginPct)
	}
	if r.DSODays != 6 { // 200M/1000M*30 = 6
		t.Errorf("DSO = %d, want 6", r.DSODays)
	}
	if r.RunwayMonths != 99 { // profitable
		t.Errorf("runway = %d, want 99", r.RunwayMonths)
	}
	if r.RiskGrade != "A" && r.RiskGrade != "B" {
		t.Errorf("grade = %s, want A or B", r.RiskGrade)
	}
}

func TestCalculateHealth_Struggling(t *testing.T) {
	r := CalculateHealth(HealthInput{
		Revenue: 500000000, COGS: 400000000, OperatingExpenses: 150000000,
		NetProfit: -50000000, TotalAR: 300000000, TotalAP: 200000000,
		CashReserve: 100000000, MonthlyExpenseAvg: 50000000,
		RevenueLastMonth: 600000000, RevenueThisMonth: 500000000,
	})
	if r.NetMarginPct >= 0 {
		t.Errorf("net margin should be negative, got %d", r.NetMarginPct)
	}
	if r.BurnRate != 50000000 {
		t.Errorf("burn rate = %d, want 50M", r.BurnRate)
	}
	if r.RunwayMonths != 2 { // 100M / 50M = 2
		t.Errorf("runway = %d, want 2", r.RunwayMonths)
	}
	if r.RiskGrade == "A" || r.RiskGrade == "B" {
		t.Errorf("should not be A or B, got %s", r.RiskGrade)
	}
}

func TestCalculateHealth_ZeroRevenue(t *testing.T) {
	r := CalculateHealth(HealthInput{CashReserve: 200000000, MonthlyExpenseAvg: 50000000})
	if r.GrossMarginPct != 0 {
		t.Errorf("gross margin should be 0, got %d", r.GrossMarginPct)
	}
	if r.HealthScore > 50 {
		t.Errorf("score should be low with no revenue, got %d", r.HealthScore)
	}
}
```

- [ ] **Step 3: Run tests**

```bash
cd skills/finance/shinhan-b2b-coach/cmd && go test -v -run TestCalculateHealth
```

- [ ] **Step 4: Write health.go — commands using the health score logic**

`cmdHealth(args)` subcommands:

- `calculate [period]` — compute health metrics from transaction + AR/AP data, save to business_health_metrics
- `show` — display latest health score with all components
- `history [count]` — show health score trend

`cmdReport(args)` subcommands:

- `pnl <period>` — P&L from transactions (revenue - COGS - opex = net profit)
- `cashflow <period>` — cash in vs cash out from transactions
- `aging` — combined AR + AP aging report
- `summary` — one-page business overview (health + financials + AR/AP + forecast)

`cmdTax(args)` subcommands:

- `estimate-vat <period>` — VAT = output VAT (10% of revenue) - input VAT (10% of purchases)
- `estimate-cit <period>` — CIT = 20% of net profit (simplified)
- `deadlines` — upcoming tax deadlines from data/tax_calendar_b2b.json

- [ ] **Step 5: Build + test all commands**

```bash
go build -o ../b2b-cli .
../b2b-cli health calculate 2026-04
../b2b-cli health show
../b2b-cli report pnl 2026-04
../b2b-cli report summary
../b2b-cli tax estimate-vat 2026-04
```

- [ ] **Step 6: Commit**

```bash
git commit -am "feat(shinhan-b2b): health score, reports, tax estimator — with unit tests"
```

---

## Task 7: Bank Product Recommender + Loan Readiness + Banker Feed

**Files:**

- Create: `skills/finance/shinhan-b2b-coach/cmd/recommend.go`
- Create: `skills/finance/shinhan-b2b-coach/data/shinhan_products.json`

- [ ] **Step 1: Write shinhan_products.json**

```json
[
  {
    "type": "working_capital",
    "name": "Shinhan Revolving Credit Line",
    "rate": "8.5%/nam",
    "min": 200000000,
    "max": 5000000000,
    "description": "Han muc tin dung quay vong, giai ngan trong 24h"
  },
  {
    "type": "invoice_financing",
    "name": "Shinhan Invoice Factoring",
    "rate": "7-9%/nam",
    "min": 500000000,
    "max": 20000000000,
    "description": "Chiet khau cong no phai thu, nhan tien ngay"
  },
  {
    "type": "supply_chain",
    "name": "Shinhan Supply Chain Financing",
    "rate": "6.5-8%/nam",
    "min": 300000000,
    "max": 10000000000,
    "description": "Thanh toan NCC truoc, doanh nghiep tra sau"
  },
  {
    "type": "expansion_loan",
    "name": "Shinhan Business Growth Loan",
    "rate": "7.9%/nam",
    "min": 500000000,
    "max": 50000000000,
    "description": "Vay mo rong kinh doanh, thoi han 1-5 nam"
  },
  {
    "type": "equipment",
    "name": "Shinhan Equipment Financing",
    "rate": "8-10%/nam",
    "min": 200000000,
    "max": 10000000000,
    "description": "Tai tro mua thiet bi, tai san co dinh"
  },
  {
    "type": "deposit",
    "name": "Shinhan Business Term Deposit",
    "rate": "5-6.5%/nam",
    "min": 100000000,
    "max": 0,
    "description": "Gui tiet kiem doanh nghiep, lai suat uu dai"
  },
  {
    "type": "trade_finance",
    "name": "Shinhan Trade Finance / LC",
    "rate": "varies",
    "min": 500000000,
    "max": 0,
    "description": "Thu tin dung, bao lanh XNK"
  },
  {
    "type": "fx_hedging",
    "name": "Shinhan FX Hedging Account",
    "rate": "varies",
    "min": 0,
    "max": 0,
    "description": "Phong ngua rui ro ti gia cho DN nhap khau"
  },
  {
    "type": "insurance",
    "name": "Shinhan Group Insurance",
    "rate": "varies",
    "min": 0,
    "max": 0,
    "description": "Bao hiem nhom nhan vien"
  },
  {
    "type": "payroll",
    "name": "Shinhan Payroll Service",
    "rate": "free",
    "min": 0,
    "max": 0,
    "description": "Dich vu chi luong tu dong + bao hiem"
  },
  {
    "type": "premium_account",
    "name": "Shinhan Premium Business Account",
    "rate": "varies",
    "min": 0,
    "max": 0,
    "description": "Tai khoan DN cao cap voi uu dai dac biet"
  }
]
```

- [ ] **Step 2: Write recommend.go**

`cmdRecommend(args)` subcommands:

- `evaluate` — run all product triggers against current company data (from the spec's trigger table):
  1. Cashflow gap detected → working_capital
  2. AR > 60 days overdue, amount > 500M → invoice_financing
  3. Consistent early-pay discount usage → supply_chain
  4. Revenue growing > 20% QoQ → expansion_loan
  5. Import/export transactions → trade_finance / fx_hedging
  6. Cash reserve > 6 months expenses → deposit
  7. Equipment purchase > 200M → equipment
  8. Employee count growing > 20% → payroll + insurance
  9. Health score > 80 + growth → premium_account
     Save each recommendation to bank_product_recommendations table with trigger_reason and trigger_data.

- `list [new|contacted|converted|all]` — list recommendations by status
- `update <id> <status> [assigned_rm] [outcome]` — update recommendation status (contacted, converted, declined)
- `loan-readiness [target_amount]` — assess company's loan eligibility:
  1. Score revenue stability (12-month trend)
  2. Score profitability (net margin)
  3. Score payment history (% on-time from AR/AP)
  4. Score existing debt ratio
  5. Composite score 0-100
  6. List strengths (✅), warnings (⚠️), gaps (❌)
  7. Suggest improvements to increase score

`cmdBanker(args)` subcommands:

- `portfolio` — portfolio summary: total companies, health distribution, total revenue
- `pipeline` — product opportunity pipeline: grouped by product type, sorted by priority
- `alerts` — companies with deteriorating health (score dropped >10 in last period) or high overdue AR

- [ ] **Step 3: Build + test**

```bash
go build -o ../b2b-cli .
../b2b-cli recommend evaluate
../b2b-cli recommend list
../b2b-cli recommend loan-readiness 2000000000
../b2b-cli banker portfolio
../b2b-cli banker pipeline
```

- [ ] **Step 4: Commit**

```bash
git commit -am "feat(shinhan-b2b): product recommender, loan readiness, banker dashboard"
```

---

## Task 8: schema.json + SKILL.md + config.json + IDENTITY.md

**Files:**

- Create: `skills/finance/shinhan-b2b-coach/schema.json`
- Create: `skills/finance/shinhan-b2b-coach/SKILL.md`
- Create: `skills/finance/shinhan-b2b-coach/config.json`
- Create: `skills/finance/shinhan-b2b-coach/bootstrap-files/IDENTITY.md`
- Create: `skills/finance/shinhan-b2b-coach/data/industry_benchmarks.json`
- Create: `skills/finance/shinhan-b2b-coach/data/tax_calendar_b2b.json`

- [ ] **Step 1: Copy schema.json from spec**

Use the 9-table schema directly from section 2 of the spec document. Set `"primary": "business_transactions"` and `"timezone": "Asia/Ho_Chi_Minh"`.

- [ ] **Step 2: Write SKILL.md**

YAML frontmatter:

```yaml
---
name: shinhan-b2b-coach
description: "Tu van tai chinh doanh nghiep 24/7 — du bao dong tien, quan ly cong no, chien luoc gia, suc khoe tai chinh, goi y san pham Shinhan."
metadata: { "openclaw": { "emoji": "🏦" } }
---
```

The prompt body should:

1. Introduce the coach persona: "Tu van vien tai chinh AI cho doanh nghiep SME Viet Nam. Phan tich dong tien, cong no, chien luoc kinh doanh, va goi y giai phap tai chinh tu Shinhan."
2. Document ALL b2b-cli commands with exact syntax (company, txn, cashflow, ar, ap, discount, pricing, health, report, tax, recommend, banker)
3. Include the 3-tier confirmation system from BEST_PRACTICES.md:
   - Tier 1 (double confirm): evaluate product recommendations, bulk import transactions
   - Tier 2 (single confirm): add receivable/payable > 500M, mark AR as paid, run payroll-related tax
   - Tier 3 (immediate): view reports, calculate health, list/search
4. Include Vietnamese example interactions from the spec (cashflow forecast, discount strategy, loan readiness)
5. Include the Shinhan product trigger table for reference
6. Rules: all VND, check ok:true, no fabricating data, always cite data source
7. When recommending Shinhan products, include specific terms (rate, min/max, description) from shinhan_products.json

- [ ] **Step 3: Write config.json**

```json
{
  "version": "1.0.0",
  "requires_bins": [],
  "setup_prompts": [
    {
      "key": "company_name",
      "label": "Ten doanh nghiep",
      "placeholder": "Cong ty TNHH ABC"
    },
    {
      "key": "industry",
      "label": "Nganh nghe",
      "placeholder": "wholesale/manufacturing/retail/service"
    },
    { "key": "tax_code", "label": "Ma so thue", "placeholder": "0312345678" }
  ],
  "exclude": ["cmd", "*.go", "*.tmp"]
}
```

- [ ] **Step 4: Write IDENTITY.md**

```markdown
Ten: Tai — Tu van tai chinh doanh nghiep
Doi tac: Ngan hang Shinhan Viet Nam

Ban la chuyen gia tai chinh cho doanh nghiep vua va nho. Ban hieu biet sau ve:

- Quan ly dong tien va cong no
- Chien luoc gia va chiet khau
- Bao cao tai chinh va thue
- San pham ngan hang (Shinhan) phu hop cho doanh nghiep

Tone: Chuyen nghiep nhung than thien. Dung so lieu cu the, khong chung chung.
Luon kem giai phap sau moi phan tich.
```

- [ ] **Step 5: Write industry_benchmarks.json**

```json
{
  "wholesale": {
    "gross_margin_pct": 15,
    "net_margin_pct": 5,
    "dso_days": 35,
    "dpo_days": 25
  },
  "manufacturing": {
    "gross_margin_pct": 25,
    "net_margin_pct": 8,
    "dso_days": 40,
    "dpo_days": 30
  },
  "retail": {
    "gross_margin_pct": 30,
    "net_margin_pct": 7,
    "dso_days": 10,
    "dpo_days": 20
  },
  "service": {
    "gross_margin_pct": 45,
    "net_margin_pct": 15,
    "dso_days": 30,
    "dpo_days": 15
  },
  "construction": {
    "gross_margin_pct": 20,
    "net_margin_pct": 6,
    "dso_days": 60,
    "dpo_days": 45
  },
  "fmcg": {
    "gross_margin_pct": 20,
    "net_margin_pct": 5,
    "dso_days": 20,
    "dpo_days": 30
  },
  "technology": {
    "gross_margin_pct": 50,
    "net_margin_pct": 12,
    "dso_days": 35,
    "dpo_days": 20
  }
}
```

- [ ] **Step 6: Write tax_calendar_b2b.json**

```json
{
  "deadlines": [
    {
      "tax_type": "vat",
      "period": "monthly",
      "rule": "Ngay 20 thang sau",
      "note": "Doanh nghiep DT >50 ty"
    },
    {
      "tax_type": "vat",
      "period": "quarterly",
      "rule": "Ngay cuoi thang dau quy sau",
      "note": "DN DT <=50 ty"
    },
    {
      "tax_type": "cit",
      "period": "quarterly",
      "rule": "Ngay 30 thang dau quy sau",
      "note": "Tam nop TNDN"
    },
    {
      "tax_type": "cit",
      "period": "annually",
      "rule": "Thang 3 nam sau",
      "note": "Quyet toan TNDN"
    },
    {
      "tax_type": "pit",
      "period": "monthly",
      "rule": "Ngay 20 thang sau",
      "note": "Khau tru TNCN"
    },
    {
      "tax_type": "financial_report",
      "period": "annually",
      "rule": "Thang 3 nam sau",
      "note": "BCTC nam"
    },
    {
      "tax_type": "license_fee",
      "period": "annually",
      "rule": "Ngay 30/01",
      "note": "Le phi mon bai"
    }
  ]
}
```

- [ ] **Step 7: Run make generate + build + test**

```bash
cd /Users/ngoctran/src/clawkit
make generate   # Should include shinhan-b2b-coach
make build
make test
make check-generate
./clawkit list | grep shinhan
```

- [ ] **Step 8: Commit**

```bash
git commit -am "feat(shinhan-b2b): SKILL.md, schema, config, data files — complete B2B Finance Coach skill"
```

---

## Task 9: End-to-End Integration Test

**Files:** None created — this is a verification task.

- [ ] **Step 1: Clean state test**

```bash
rm -f ~/.openclaw/workspace/skills/shinhan-b2b-coach/b2b.db
b2b=skills/finance/shinhan-b2b-coach/b2b-cli
$b2b init
$b2b company add "Viet Anh Trading" wholesale 0312345678 25 1200000000
$b2b company update cash_reserve 500000000
$b2b company update monthly_expense_avg 850000000
```

- [ ] **Step 2: Seed realistic transaction data**

```bash
# Revenue transactions (4 months)
$b2b txn add 2026-01-15 sale in "KH Alpha" 350tr revenue INV-001
$b2b txn add 2026-01-20 sale in "KH Beta" 280tr revenue INV-002
$b2b txn add 2026-02-10 sale in "KH Alpha" 400tr revenue INV-003
$b2b txn add 2026-02-18 sale in "KH Gamma" 150tr revenue INV-004
$b2b txn add 2026-03-05 sale in "KH Alpha" 380tr revenue INV-005
$b2b txn add 2026-03-22 sale in "KH Beta" 300tr revenue INV-006
$b2b txn add 2026-04-08 sale in "KH Alpha" 420tr revenue INV-007
$b2b txn add 2026-04-15 sale in "KH Delta" 200tr revenue INV-008

# Expense transactions
$b2b txn add 2026-04-01 salary out "Payroll" 250tr payroll
$b2b txn add 2026-04-05 purchase out "NCC Hoang Anh" 180tr cogs PO-001
$b2b txn add 2026-04-10 purchase out "NCC Minh Duc" 120tr cogs PO-002
$b2b txn add 2026-04-12 rent out "Van phong" 45tr rent
$b2b txn add 2026-04-15 utility out "Dien nuoc" 15tr opex
```

- [ ] **Step 3: Seed AR/AP data**

```bash
# Receivables (some overdue)
$b2b ar add "KH Alpha" 420tr 2026-04-30 INV-007
$b2b ar add "KH Delta" 200tr 2026-04-20 INV-008
$b2b ar add "KH Beta" 300tr 2026-03-15 INV-006  # overdue

# Payables (with early-pay discount)
$b2b ap add "NCC Hoang Anh" 180tr 2026-04-25 PO-001 2 2026-04-15
$b2b ap add "NCC Minh Duc" 120tr 2026-05-01 PO-002
```

- [ ] **Step 4: Test all major features**

```bash
# Cashflow
$b2b cashflow forecast 60
$b2b cashflow weekly

# AR/AP
$b2b ar aging
$b2b ap schedule
$b2b ap discount-roi 1

# Health
$b2b health calculate 2026-04
$b2b health show

# Reports
$b2b report pnl 2026-04
$b2b report summary

# Tax
$b2b tax estimate-vat 2026-04

# Discount strategy
$b2b discount analyze
$b2b discount simulate 5 A 10

# Bank products
$b2b recommend evaluate
$b2b recommend list
$b2b recommend loan-readiness 2000000000

# Banker view
$b2b banker portfolio
$b2b banker pipeline
```

Verify:

- All commands return `"ok": true`
- Health score is a reasonable number (0-100)
- Cashflow forecast shows expected gap if AR collection slow
- Product recommendations trigger for cashflow gap and/or overdue AR
- P&L report shows correct revenue - expenses
- VAT estimate = ~10% of (revenue - purchases)

- [ ] **Step 5: Commit verification notes**

```bash
git commit --allow-empty -m "test: verified shinhan-b2b end-to-end — all 12 capabilities working"
```

---

## Self-Review Checklist

**Spec coverage:**

- [x] A1 biz-cashflow-forecaster → Task 4 (cashflow forecast/weekly/gap)
- [x] A2 biz-ar-optimizer → Task 3 (ar add/list/aging/score/remind/pay)
- [x] A3 biz-ap-strategist → Task 3 (ap add/list/schedule/discount-roi/pay)
- [x] B1 biz-discount-advisor → Task 5 (discount analyze/simulate)
- [x] B2 biz-pricing-analyzer → Task 5 (pricing analyze)
- [x] C1 biz-health-dashboard → Task 6 (health calculate/show/history)
- [x] C2 biz-report-generator → Task 6 (report pnl/cashflow/aging/summary)
- [x] C3 biz-tax-estimator → Task 6 (tax estimate-vat/estimate-cit/deadlines)
- [x] D1 biz-product-recommender → Task 7 (recommend evaluate/list/update)
- [x] D2 biz-loan-readiness → Task 7 (recommend loan-readiness)
- [x] E1 banker-intelligence-feed → Task 7 (banker portfolio/pipeline/alerts)
- [x] Company profile + onboarding → Task 2
- [x] Business transactions + P&L → Task 2
- [x] Schema (9 tables) → Task 1
- [x] Health score calculation with tests → Task 6
- [x] Shinhan product trigger table → Task 7
- [x] SKILL.md with Vietnamese examples → Task 8
- [x] End-to-end verification → Task 9

**Placeholder scan:** No TBD/TODO found. All code blocks complete.

**Type consistency:** `HealthInput`/`HealthResult` defined in Task 6, used in `health.go` (same task). `parseVND()` defined in Task 1 `store.go`, used throughout. CLI command names consistent across tasks and SKILL.md.
