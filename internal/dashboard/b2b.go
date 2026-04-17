package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rockship-co/clawkit/internal/config"
)

func b2bDBPath() string {
	return filepath.Join(config.GetSkillsDir(), "shinhan-b2b-coach", "b2b.db")
}

func handleB2B(w http.ResponseWriter, r *http.Request) {
	dbPath := b2bDBPath()
	if _, err := os.Stat(dbPath); err != nil {
		http.Error(w, "B2B database not found. Install shinhan-b2b-coach first.", http.StatusNotFound)
		return
	}

	action := strings.TrimPrefix(r.URL.Path, "/api/b2b/")

	switch action {
	case "company":
		result, err := runQuery(dbPath, "SELECT * FROM company_profile LIMIT 1")
		if err != nil {
			httpErr(w, err)
			return
		}
		jsonOK(w, result)

	case "health":
		result, err := runQuery(dbPath, "SELECT * FROM business_health_metrics ORDER BY period DESC LIMIT 6")
		if err != nil {
			httpErr(w, err)
			return
		}
		jsonOK(w, result)

	case "cashflow":
		result, err := runQuery(dbPath, "SELECT * FROM cashflow_forecast ORDER BY period_start DESC LIMIT 30")
		if err != nil {
			httpErr(w, err)
			return
		}
		jsonOK(w, result)

	case "receivables":
		result, err := runQuery(dbPath, "SELECT *, CAST(julianday('now') - julianday(due_date) AS INTEGER) as calc_days_overdue FROM receivables WHERE status='outstanding' ORDER BY due_date")
		if err != nil {
			httpErr(w, err)
			return
		}
		jsonOK(w, result)

	case "payables":
		result, err := runQuery(dbPath, "SELECT * FROM payables WHERE status='outstanding' ORDER BY due_date")
		if err != nil {
			httpErr(w, err)
			return
		}
		jsonOK(w, result)

	case "transactions":
		period := r.URL.Query().Get("period")
		q := "SELECT * FROM business_transactions"
		if period != "" {
			q += fmt.Sprintf(" WHERE substr(date,1,7)='%s'", period)
		}
		q += " ORDER BY date DESC LIMIT 100"
		result, err := runQuery(dbPath, q)
		if err != nil {
			httpErr(w, err)
			return
		}
		jsonOK(w, result)

	case "recommendations":
		result, err := runQuery(dbPath, "SELECT * FROM bank_product_recommendations ORDER BY CASE priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END, created_at DESC")
		if err != nil {
			httpErr(w, err)
			return
		}
		jsonOK(w, result)

	case "discounts":
		result, err := runQuery(dbPath, "SELECT * FROM discount_strategies ORDER BY created_at DESC")
		if err != nil {
			httpErr(w, err)
			return
		}
		jsonOK(w, result)

	case "summary":
		type summaryData struct {
			Company         json.RawMessage `json:"company"`
			Health          json.RawMessage `json:"health"`
			AR              json.RawMessage `json:"ar"`
			AP              json.RawMessage `json:"ap"`
			Recommendations json.RawMessage `json:"recommendations"`
			RecentTxns      json.RawMessage `json:"recent_transactions"`
		}
		sd := summaryData{}
		sd.Company = marshalAny(queryOrEmpty(dbPath, "SELECT * FROM company_profile LIMIT 1"))
		sd.Health = marshalAny(queryOrEmpty(dbPath, "SELECT * FROM business_health_metrics ORDER BY period DESC LIMIT 1"))
		sd.AR = marshalAny(queryOrEmpty(dbPath, "SELECT status, COUNT(*) as count, COALESCE(SUM(amount),0) as total FROM receivables GROUP BY status"))
		sd.AP = marshalAny(queryOrEmpty(dbPath, "SELECT status, COUNT(*) as count, COALESCE(SUM(amount),0) as total FROM payables GROUP BY status"))
		sd.Recommendations = marshalAny(queryOrEmpty(dbPath, "SELECT * FROM bank_product_recommendations WHERE status='new' ORDER BY CASE priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END LIMIT 10"))
		sd.RecentTxns = marshalAny(queryOrEmpty(dbPath, "SELECT * FROM business_transactions ORDER BY date DESC LIMIT 10"))
		jsonOK(w, sd)

	case "pnl":
		period := r.URL.Query().Get("period")
		if period == "" {
			httpErr(w, fmt.Errorf("missing ?period=YYYY-MM"))
			return
		}
		revenue := queryOrEmpty(dbPath, fmt.Sprintf("SELECT COALESCE(SUM(amount),0) as total FROM business_transactions WHERE direction='in' AND category='revenue' AND substr(date,1,7)='%s'", period))
		cogs := queryOrEmpty(dbPath, fmt.Sprintf("SELECT COALESCE(SUM(amount),0) as total FROM business_transactions WHERE direction='out' AND category='cogs' AND substr(date,1,7)='%s'", period))
		opex := queryOrEmpty(dbPath, fmt.Sprintf("SELECT COALESCE(SUM(amount),0) as total FROM business_transactions WHERE direction='out' AND category IN ('opex','rent','payroll','tax') AND substr(date,1,7)='%s'", period))
		jsonOK(w, map[string]any{"period": period, "revenue": revenue, "cogs": cogs, "opex": opex})

	case "ar-aging":
		result, err := runQuery(dbPath, `SELECT
			CASE
				WHEN julianday('now') - julianday(due_date) <= 0 THEN 'current'
				WHEN julianday('now') - julianday(due_date) <= 30 THEN '1-30d'
				WHEN julianday('now') - julianday(due_date) <= 60 THEN '31-60d'
				ELSE '>60d'
			END as bucket,
			COUNT(*) as count,
			SUM(amount) as total
			FROM receivables WHERE status='outstanding'
			GROUP BY bucket
			ORDER BY CASE bucket WHEN 'current' THEN 1 WHEN '1-30d' THEN 2 WHEN '31-60d' THEN 3 ELSE 4 END`)
		if err != nil {
			httpErr(w, err)
			return
		}
		jsonOK(w, result)

	case "monthly-revenue":
		result, err := runQuery(dbPath, `SELECT substr(date,1,7) as month,
			SUM(CASE WHEN direction='in' THEN amount ELSE 0 END) as revenue,
			SUM(CASE WHEN direction='out' THEN amount ELSE 0 END) as expense
			FROM business_transactions
			GROUP BY substr(date,1,7) ORDER BY month DESC LIMIT 12`)
		if err != nil {
			httpErr(w, err)
			return
		}
		jsonOK(w, result)

	default:
		http.NotFound(w, r)
	}
}

func httpErr(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func queryOrEmpty(dbPath, sql string) any {
	result, err := runQuery(dbPath, sql)
	if err != nil {
		return []map[string]any{}
	}
	return result
}

func marshalAny(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("null")
	}
	return json.RawMessage(b)
}
