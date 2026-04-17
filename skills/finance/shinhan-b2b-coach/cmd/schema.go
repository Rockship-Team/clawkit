package main

import "fmt"

// cmdInit creates all tables and seeds default data.
func cmdInit() {
	db := mustDB()
	defer db.Close()

	tables := []string{
		`CREATE TABLE IF NOT EXISTS company_profile (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id TEXT UNIQUE,
			name TEXT NOT NULL,
			tax_code TEXT,
			industry TEXT,
			employee_count INTEGER,
			monthly_revenue_avg INTEGER,
			monthly_expense_avg INTEGER,
			cash_reserve INTEGER,
			current_bank_products TEXT,
			accounting_software TEXT,
			fiscal_year_start INTEGER DEFAULT 1,
			owner_user_id TEXT,
			health_score INTEGER,
			risk_grade TEXT,
			onboarded TEXT DEFAULT 'false',
			created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS business_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id TEXT NOT NULL,
			date TEXT NOT NULL,
			type TEXT NOT NULL,
			direction TEXT NOT NULL,
			counterparty TEXT,
			amount INTEGER NOT NULL,
			category TEXT NOT NULL,
			invoice_number TEXT,
			due_date TEXT,
			paid_date TEXT,
			status TEXT DEFAULT 'pending',
			note TEXT,
			created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS cashflow_forecast (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id TEXT NOT NULL,
			forecast_date TEXT NOT NULL,
			period_start TEXT NOT NULL,
			period_end TEXT NOT NULL,
			scenario TEXT DEFAULT 'base',
			opening_balance INTEGER,
			inflows_confirmed INTEGER,
			inflows_expected INTEGER,
			outflows_confirmed INTEGER,
			outflows_expected INTEGER,
			net_cashflow INTEGER,
			closing_balance INTEGER,
			gap_alert TEXT DEFAULT 'false',
			line_items TEXT,
			created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS receivables (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id TEXT NOT NULL,
			customer_name TEXT NOT NULL,
			invoice_number TEXT,
			amount INTEGER NOT NULL,
			issued_date TEXT,
			due_date TEXT NOT NULL,
			paid_date TEXT,
			status TEXT DEFAULT 'outstanding',
			days_overdue INTEGER DEFAULT 0,
			aging_bucket TEXT,
			collection_probability INTEGER DEFAULT 100,
			created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS payables (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id TEXT NOT NULL,
			vendor_name TEXT NOT NULL,
			invoice_number TEXT,
			amount INTEGER NOT NULL,
			issued_date TEXT,
			due_date TEXT NOT NULL,
			paid_date TEXT,
			status TEXT DEFAULT 'outstanding',
			early_pay_discount_pct INTEGER,
			early_pay_deadline TEXT,
			created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS discount_strategies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id TEXT NOT NULL,
			strategy_type TEXT NOT NULL,
			target_segment TEXT,
			discount_pct INTEGER,
			condition TEXT,
			projected_revenue_impact INTEGER,
			projected_margin_impact INTEGER,
			ai_recommendation TEXT,
			status TEXT DEFAULT 'proposed',
			created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS bank_product_recommendations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id TEXT NOT NULL,
			product_type TEXT NOT NULL,
			product_name TEXT NOT NULL,
			trigger_reason TEXT NOT NULL,
			trigger_data TEXT,
			estimated_amount INTEGER,
			priority TEXT DEFAULT 'medium',
			status TEXT DEFAULT 'new',
			assigned_rm TEXT,
			contacted_at TEXT,
			outcome TEXT,
			created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS business_health_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id TEXT NOT NULL,
			period TEXT NOT NULL,
			revenue INTEGER,
			cogs INTEGER,
			gross_margin_pct INTEGER,
			operating_expenses INTEGER,
			net_profit INTEGER,
			net_margin_pct INTEGER,
			current_ratio TEXT,
			quick_ratio TEXT,
			dso_days INTEGER,
			dpo_days INTEGER,
			cash_conversion_cycle INTEGER,
			burn_rate INTEGER,
			runway_months INTEGER,
			health_score INTEGER,
			created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS coach_conversations_b2b (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			question TEXT NOT NULL,
			topic TEXT NOT NULL,
			intent TEXT,
			product_signal TEXT,
			created_at TEXT
		)`,
	}

	for _, ddl := range tables {
		if _, err := db.Exec(ddl); err != nil {
			errOut("failed to create table: " + err.Error())
		}
	}

	// Indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_txn_company_date ON business_transactions(company_id, date)`,
		`CREATE INDEX IF NOT EXISTS idx_recv_company_status ON receivables(company_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_pay_company_status ON payables(company_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_rec_company_status ON bank_product_recommendations(company_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_health_company_period ON business_health_metrics(company_id, period)`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			errOut("failed to create index: " + err.Error())
		}
	}

	// Seed a default company if none exists
	row, err := queryOne(db, "SELECT COUNT(*) AS cnt FROM company_profile")
	if err != nil {
		errOut("failed to check company_profile: " + err.Error())
	}
	cnt := int64(0)
	if row != nil {
		switch v := row["cnt"].(type) {
		case int64:
			cnt = v
		case float64:
			cnt = int64(v)
		}
	}

	if cnt == 0 {
		companyID := newID()
		now := vnNowISO()
		_, err := exec(db,
			`INSERT INTO company_profile (company_id, name, onboarded, created_at) VALUES (?, ?, ?, ?)`,
			companyID, "My Company", "false", now,
		)
		if err != nil {
			errOut("failed to seed default company: " + err.Error())
		}
		fmt.Printf("Seeded default company: %s (id=%s)\n", "My Company", companyID)
	}

	okOut(map[string]interface{}{
		"message": "Database initialized with 9 tables",
		"db_path": dbPath(),
		"tables":  9,
		"indexes": 5,
	})
}
