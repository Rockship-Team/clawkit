package main

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"os"
	"strings"
)

// Tip is a savings tip entry.
type Tip struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Text     string `json:"text"`
	Season   string `json:"season,omitempty"`
}

// CreditCard is a VN credit card entry.
type CreditCard struct {
	ID          string `json:"id"`
	Bank        string `json:"bank"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	AnnualFee   string `json:"annual_fee"`
	Cashback    string `json:"cashback"`
	Rewards     string `json:"rewards"`
	MinIncome   int64  `json:"min_income"`
	InterestPct string `json:"interest_pct"`
	BestFor     string `json:"best_for"`
}

func loadTips() []Tip {
	var tips []Tip
	readJSON(dataPath("tips.json"), &tips)
	return tips
}

func loadCards() []CreditCard {
	var cards []CreditCard
	readJSON(dataPath("credit-cards.json"), &cards)
	return cards
}

func cmdTips(args []string) {
	if len(args) == 0 {
		errOut("usage: tips random|daily|seasonal [category]")
		os.Exit(1)
	}

	tips := loadTips()
	if len(tips) == 0 {
		errOut("no tips found in data/tips.json")
		os.Exit(1)
	}

	switch args[0] {
	case "random":
		category := ""
		if len(args) > 1 {
			category = args[1]
		}
		filtered := tips
		if category != "" {
			filtered = nil
			for _, t := range tips {
				if t.Category == category {
					filtered = append(filtered, t)
				}
			}
		}
		if len(filtered) == 0 {
			errOut("no tips for category: " + category)
			os.Exit(1)
		}
		tip := filtered[rand.Intn(len(filtered))]
		okOut(map[string]interface{}{"tip": tip})

	case "daily":
		// Deterministic tip based on date
		h := sha256.Sum256([]byte(vnToday()))
		idx := int(binary.BigEndian.Uint32(h[:4])) % len(tips)
		tip := tips[idx]
		okOut(map[string]interface{}{"tip": tip, "date": vnToday()})

	case "seasonal":
		now := vnNow()
		month := int(now.Month())
		day := now.Day()

		var seasons []string
		// Tet: Jan 1 – Feb 15
		if month == 1 || (month == 2 && day <= 15) {
			seasons = append(seasons, "tet")
		}
		// Back to school: Jul 15 – Sep 15
		if (month == 7 && day >= 15) || month == 8 || (month == 9 && day <= 15) {
			seasons = append(seasons, "back-to-school")
		}
		// Sale season: Nov 1 – Dec 31
		if month == 11 || month == 12 {
			seasons = append(seasons, "sale-season")
		}

		if len(seasons) == 0 {
			okOut(map[string]interface{}{"seasonal": false, "seasons": []string{}, "tips": []Tip{}, "count": 0})
			return
		}

		seasonSet := map[string]bool{}
		for _, s := range seasons {
			seasonSet[s] = true
		}
		var matched []Tip
		for _, t := range tips {
			if t.Season != "" && seasonSet[t.Season] {
				matched = append(matched, t)
			}
		}
		okOut(map[string]interface{}{"seasonal": true, "seasons": seasons, "tips": matched, "count": len(matched)})

	default:
		errOut("unknown tips command: " + args[0])
		os.Exit(1)
	}
}

func cmdCards(args []string) {
	if len(args) == 0 {
		errOut("usage: cards list|recommend|compare")
		os.Exit(1)
	}

	cards := loadCards()
	if len(cards) == 0 {
		errOut("no cards found in data/credit-cards.json")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		category := ""
		if len(args) > 1 {
			category = args[1]
		}
		filtered := cards
		if category != "" {
			filtered = nil
			for _, c := range cards {
				if c.Category == category {
					filtered = append(filtered, c)
				}
			}
		}
		okOut(map[string]interface{}{"cards": filtered, "count": len(filtered)})

	case "recommend":
		if len(args) < 2 {
			errOut("usage: cards recommend <spending_type> [income]")
			os.Exit(1)
		}
		spendType := args[1]
		income := int64(0)
		if len(args) > 2 {
			income, _ = parseAmount(args[2])
		}

		var matched []CreditCard
		for _, c := range cards {
			if income > 0 && c.MinIncome > income {
				continue
			}
			if strings.Contains(strings.ToLower(c.BestFor), spendType) {
				matched = append(matched, c)
			}
		}
		// If no specific match, return general cards
		if len(matched) == 0 {
			for _, c := range cards {
				if income > 0 && c.MinIncome > income {
					continue
				}
				matched = append(matched, c)
			}
		}
		if len(matched) > 5 {
			matched = matched[:5]
		}
		okOut(map[string]interface{}{"recommended": matched, "spending_type": spendType, "count": len(matched)})

	case "compare":
		if len(args) < 3 {
			errOut("usage: cards compare <card_id_1> <card_id_2>")
			os.Exit(1)
		}
		id1, id2 := args[1], args[2]
		var card1, card2 *CreditCard
		for i := range cards {
			if cards[i].ID == id1 {
				card1 = &cards[i]
			}
			if cards[i].ID == id2 {
				card2 = &cards[i]
			}
		}
		if card1 == nil {
			errOut("card not found: " + id1)
			os.Exit(1)
		}
		if card2 == nil {
			errOut("card not found: " + id2)
			os.Exit(1)
		}
		okOut(map[string]interface{}{"card1": card1, "card2": card2})

	default:
		errOut("unknown cards command: " + args[0])
		os.Exit(1)
	}
}
