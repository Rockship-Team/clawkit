package main

import (
	"os"
	"sort"
	"strconv"
	"strings"
)

type Transaction struct {
	ID          int    `json:"id"`
	Date        string `json:"date"`
	Place       string `json:"place"`
	Amount      int64  `json:"amount"`
	Category    string `json:"category"`
	Note        string `json:"note"`
	IsRecurring bool   `json:"is_recurring,omitempty"`
	CreatedAt   string `json:"created_at"`
}

var validCategories = map[string]bool{
	"food": true, "cafe": true, "shopping": true, "transport": true,
	"health": true, "entertainment": true, "education": true,
	"home": true, "bills": true, "other": true,
}

func loadTransactions() []Transaction {
	var txs []Transaction
	readJSON(userPath("transactions.json"), &txs)
	return txs
}

func saveTransactions(txs []Transaction) error {
	return writeJSON(userPath("transactions.json"), txs)
}

func cmdSpend(args []string) {
	ensureInit()
	if len(args) == 0 {
		errOut("usage: spend add|report|last|undo")
		os.Exit(1)
	}

	switch args[0] {
	case "add":
		spendAdd(args[1:])
	case "report":
		period := "month"
		if len(args) > 1 {
			period = args[1]
		}
		spendReport(period)
	case "last":
		n := 5
		if len(args) > 1 {
			n, _ = strconv.Atoi(args[1])
			if n <= 0 {
				n = 5
			}
		}
		spendLast(n)
	case "undo":
		spendUndo()
	case "budget":
		spendBudget(args[1:])
	case "compare":
		if len(args) < 3 {
			errOut("usage: spend compare <period1> <period2>")
			os.Exit(1)
		}
		spendCompare(args[1], args[2])
	default:
		errOut("unknown spend command: " + args[0])
		os.Exit(1)
	}
}

func spendAdd(args []string) {
	// args: <place> <amount> <category> [note] [date]
	if len(args) < 3 {
		errOut("usage: spend add <place> <amount> <category> [note] [date]")
		os.Exit(1)
	}

	place := args[0]
	amount, err := parseAmount(args[1])
	if err != nil {
		errOut("invalid amount: " + args[1])
		os.Exit(1)
	}

	category := strings.ToLower(args[2])
	if !validCategories[category] {
		category = "other"
	}

	note := ""
	if len(args) > 3 {
		note = args[3]
	}

	date := vnToday()
	if len(args) > 4 && args[4] != "" {
		date = args[4]
	}

	txs := loadTransactions()
	maxID := 0
	todayTotal := int64(0)
	for _, t := range txs {
		if t.ID > maxID {
			maxID = t.ID
		}
		if t.Date == vnToday() {
			todayTotal += t.Amount
		}
	}

	tx := Transaction{
		ID:        maxID + 1,
		Date:      date,
		Place:     place,
		Amount:    amount,
		Category:  category,
		Note:      note,
		CreatedAt: vnNow().Format("2006-01-02T15:04:05-07:00"),
	}

	txs = append(txs, tx)
	if err := saveTransactions(txs); err != nil {
		errOut("failed to save: " + err.Error())
		os.Exit(1)
	}

	okOut(map[string]interface{}{
		"saved":       tx,
		"today_total": todayTotal + amount,
	})
}

func spendReport(period string) {
	txs := loadTransactions()
	now := vnNow()

	var filtered []Transaction
	var label string

	switch period {
	case "today":
		today := now.Format("2006-01-02")
		label = today
		for _, t := range txs {
			if t.Date == today {
				filtered = append(filtered, t)
			}
		}
	case "week":
		weekAgo := now.AddDate(0, 0, -7).Format("2006-01-02")
		label = "7 ngay qua"
		for _, t := range txs {
			if t.Date >= weekAgo {
				filtered = append(filtered, t)
			}
		}
	case "month":
		monthPrefix := now.Format("2006-01")
		label = now.Format("2006-01")
		for _, t := range txs {
			if strings.HasPrefix(t.Date, monthPrefix) {
				filtered = append(filtered, t)
			}
		}
	case "all":
		label = "tat ca"
		filtered = txs
	default:
		label = period
		for _, t := range txs {
			if strings.HasPrefix(t.Date, period) {
				filtered = append(filtered, t)
			}
		}
	}

	total := int64(0)
	catTotals := map[string]int64{}
	for _, t := range filtered {
		total += t.Amount
		catTotals[t.Category] += t.Amount
	}

	type catEntry struct {
		Category string `json:"category"`
		Amount   int64  `json:"amount"`
		Pct      int    `json:"pct"`
	}

	var byCat []catEntry
	for cat, amt := range catTotals {
		pct := 0
		if total > 0 {
			pct = int(amt * 100 / total)
		}
		byCat = append(byCat, catEntry{cat, amt, pct})
	}
	sort.Slice(byCat, func(i, j int) bool { return byCat[i].Amount > byCat[j].Amount })

	// Recent 5
	recent := filtered
	if len(recent) > 5 {
		recent = recent[len(recent)-5:]
	}
	// Reverse for most recent first
	for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
		recent[i], recent[j] = recent[j], recent[i]
	}

	result := map[string]interface{}{
		"period":      period,
		"label":       label,
		"total":       total,
		"count":       len(filtered),
		"by_category": byCat,
		"recent":      recent,
	}

	// Include budget info if set
	p := loadProfile()
	if p.MonthlyBudget > 0 && (period == "month" || strings.HasPrefix(period, vnNow().Format("2006-01"))) {
		remaining := p.MonthlyBudget - total
		pctUsed := 0
		if p.MonthlyBudget > 0 {
			pctUsed = int(total * 100 / p.MonthlyBudget)
		}
		result["budget"] = p.MonthlyBudget
		result["remaining"] = remaining
		result["pct_used"] = pctUsed
	}

	okOut(result)
}

func spendLast(n int) {
	txs := loadTransactions()
	if len(txs) > n {
		txs = txs[len(txs)-n:]
	}
	// Reverse
	for i, j := 0, len(txs)-1; i < j; i, j = i+1, j-1 {
		txs[i], txs[j] = txs[j], txs[i]
	}
	okOut(map[string]interface{}{"transactions": txs, "count": len(txs)})
}

func spendUndo() {
	txs := loadTransactions()
	if len(txs) == 0 {
		errOut("no transactions to undo")
		os.Exit(1)
	}
	removed := txs[len(txs)-1]
	txs = txs[:len(txs)-1]
	if err := saveTransactions(txs); err != nil {
		errOut("failed to save: " + err.Error())
		os.Exit(1)
	}
	okOut(map[string]interface{}{"removed": removed})
}

func spendBudget(args []string) {
	if len(args) == 0 {
		errOut("usage: spend budget set|get")
		os.Exit(1)
	}
	switch args[0] {
	case "set":
		if len(args) < 2 {
			errOut("usage: spend budget set <amount>")
			os.Exit(1)
		}
		v, err := parseAmount(args[1])
		if err != nil {
			errOut("invalid amount: " + args[1])
			os.Exit(1)
		}
		p := loadProfile()
		p.MonthlyBudget = v
		if err := saveProfile(p); err != nil {
			errOut("failed to save: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{"monthly_budget": v})
	case "get":
		p := loadProfile()
		okOut(map[string]interface{}{"monthly_budget": p.MonthlyBudget})
	default:
		errOut("usage: spend budget set|get")
		os.Exit(1)
	}
}

func spendCompare(period1, period2 string) {
	txs := loadTransactions()

	aggregate := func(prefix string) (int64, []map[string]interface{}) {
		total := int64(0)
		catTotals := map[string]int64{}
		for _, t := range txs {
			if strings.HasPrefix(t.Date, prefix) {
				total += t.Amount
				catTotals[t.Category] += t.Amount
			}
		}
		var byCat []map[string]interface{}
		for cat, amt := range catTotals {
			pct := 0
			if total > 0 {
				pct = int(amt * 100 / total)
			}
			byCat = append(byCat, map[string]interface{}{
				"category": cat,
				"amount":   amt,
				"pct":      pct,
			})
		}
		sort.Slice(byCat, func(i, j int) bool {
			return byCat[i]["amount"].(int64) > byCat[j]["amount"].(int64)
		})
		return total, byCat
	}

	total1, byCat1 := aggregate(period1)
	total2, byCat2 := aggregate(period2)

	// Build delta by category
	catMap1 := map[string]int64{}
	for _, c := range byCat1 {
		catMap1[c["category"].(string)] = c["amount"].(int64)
	}
	catMap2 := map[string]int64{}
	for _, c := range byCat2 {
		catMap2[c["category"].(string)] = c["amount"].(int64)
	}
	allCats := map[string]bool{}
	for k := range catMap1 {
		allCats[k] = true
	}
	for k := range catMap2 {
		allCats[k] = true
	}
	var deltaCat []map[string]interface{}
	for cat := range allCats {
		deltaCat = append(deltaCat, map[string]interface{}{
			"category": cat,
			"delta":    catMap2[cat] - catMap1[cat],
		})
	}

	okOut(map[string]interface{}{
		"period1":     map[string]interface{}{"period": period1, "total": total1, "by_category": byCat1},
		"period2":     map[string]interface{}{"period": period2, "total": total2, "by_category": byCat2},
		"total_delta": total2 - total1,
		"delta":       deltaCat,
	})
}
