package main

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Transaction struct {
	ID        int    `json:"id"`
	Date      string `json:"date"`
	Place     string `json:"place"`
	Amount    int64  `json:"amount"`
	Category  string `json:"category"`
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"`
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

	okOut(map[string]interface{}{
		"period":      period,
		"label":       label,
		"total":       total,
		"count":       len(filtered),
		"by_category": byCat,
		"recent":      recent,
	})
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

// weekStart returns Monday of the current week.
func weekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -(weekday - 1))
}
