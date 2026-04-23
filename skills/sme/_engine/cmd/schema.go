package main

import (
	"fmt"
	"os"
)

// cmdInit creates all SQLite tables. Safe to run multiple times (IF NOT EXISTS).
func cmdInit() {
	d := mustDB()

	migrations := []string{
		// ====== CORE ======
		`CREATE TABLE IF NOT EXISTS organizations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT UNIQUE NOT NULL,
			tax_code TEXT,
			address TEXT,
			phone TEXT,
			email TEXT,
			settings TEXT DEFAULT '{}',
			misa_app_id TEXT,
			misa_access_token TEXT,
			misa_token_expiry TEXT,
			misa_org_code TEXT,
			zalo_oa_id TEXT,
			zalo_oa_token TEXT,
			llm_provider TEXT DEFAULT 'claude',
			llm_api_key TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			email TEXT,
			phone TEXT,
			full_name TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'staff',
			zalo_user_id TEXT,
			is_active INTEGER DEFAULT 1,
			last_seen TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_org ON users(org_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_zalo ON users(zalo_user_id)`,

		`CREATE TABLE IF NOT EXISTS contacts (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			type TEXT NOT NULL,
			full_name TEXT NOT NULL,
			company_name TEXT,
			tax_code TEXT,
			email TEXT,
			phone TEXT,
			address TEXT,
			bank_account TEXT,
			bank_name TEXT,
			tags TEXT DEFAULT '[]',
			custom_fields TEXT DEFAULT '{}',
			misa_object_id TEXT,
			misa_object_code TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_contacts_org ON contacts(org_id)`,
		`CREATE INDEX IF NOT EXISTS idx_contacts_type ON contacts(org_id, type)`,

		`CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL,
			user_id TEXT,
			action TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			before_data TEXT,
			after_data TEXT,
			source TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_org ON audit_logs(org_id, created_at)`,

		// ====== ACCOUNTING ======
		`CREATE TABLE IF NOT EXISTS invoices (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			invoice_number TEXT,
			direction TEXT NOT NULL,
			contact_id TEXT REFERENCES contacts(id),
			invoice_type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'draft',
			subtotal INTEGER DEFAULT 0,
			vat_rate REAL DEFAULT 10.0,
			vat_amount INTEGER DEFAULT 0,
			total INTEGER DEFAULT 0,
			amount_paid INTEGER DEFAULT 0,
			amount_due INTEGER DEFAULT 0,
			issued_date TEXT,
			due_date TEXT,
			paid_date TEXT,
			seller_tax_code TEXT,
			seller_name TEXT,
			ocr_source TEXT,
			ocr_confidence REAL,
			ocr_image_url TEXT,
			misa_voucher_id TEXT,
			misa_ref_no TEXT,
			misa_sync_status TEXT DEFAULT 'pending',
			notes TEXT,
			created_by TEXT REFERENCES users(id),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_org ON invoices(org_id)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(org_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_due ON invoices(org_id, due_date)`,

		`CREATE TABLE IF NOT EXISTS invoice_items (
			id TEXT PRIMARY KEY,
			invoice_id TEXT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
			description TEXT NOT NULL,
			quantity REAL DEFAULT 1,
			unit TEXT DEFAULT 'pcs',
			unit_price INTEGER DEFAULT 0,
			vat_rate REAL DEFAULT 10.0,
			vat_amount INTEGER DEFAULT 0,
			line_total INTEGER DEFAULT 0,
			account_code TEXT,
			misa_item_id TEXT,
			created_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS payments (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			invoice_id TEXT REFERENCES invoices(id),
			contact_id TEXT REFERENCES contacts(id),
			direction TEXT NOT NULL,
			amount INTEGER NOT NULL,
			method TEXT NOT NULL,
			bank_ref TEXT,
			status TEXT NOT NULL DEFAULT 'completed',
			paid_at TEXT,
			notes TEXT,
			misa_voucher_id TEXT,
			created_by TEXT REFERENCES users(id),
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_org ON payments(org_id, paid_at)`,

		`CREATE TABLE IF NOT EXISTS bank_transactions (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			bank_name TEXT,
			account_number TEXT,
			txn_date TEXT NOT NULL,
			txn_ref TEXT,
			description TEXT,
			amount INTEGER NOT NULL,
			balance INTEGER,
			matched_payment_id TEXT REFERENCES payments(id),
			matched_invoice_id TEXT REFERENCES invoices(id),
			match_status TEXT DEFAULT 'unmatched',
			match_confidence REAL,
			imported_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bank_txn_org ON bank_transactions(org_id, txn_date)`,

		`CREATE TABLE IF NOT EXISTS cashflow_snapshots (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			snapshot_date TEXT NOT NULL,
			period_start TEXT NOT NULL,
			period_end TEXT NOT NULL,
			granularity TEXT NOT NULL,
			confirmed_inflows INTEGER DEFAULT 0,
			expected_inflows INTEGER DEFAULT 0,
			payroll_due INTEGER DEFAULT 0,
			vendor_payments_due INTEGER DEFAULT 0,
			tax_due INTEGER DEFAULT 0,
			operating_expenses INTEGER DEFAULT 0,
			opening_balance INTEGER DEFAULT 0,
			net_cashflow INTEGER DEFAULT 0,
			closing_balance INTEGER DEFAULT 0,
			scenario TEXT DEFAULT 'base',
			line_items TEXT DEFAULT '[]',
			created_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS expense_claims (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			submitted_by TEXT NOT NULL REFERENCES users(id),
			category TEXT NOT NULL,
			amount INTEGER NOT NULL,
			description TEXT,
			receipt_url TEXT,
			ocr_data TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			approved_by TEXT REFERENCES users(id),
			approved_at TEXT,
			invoice_id TEXT REFERENCES invoices(id),
			created_at TEXT NOT NULL
		)`,

		// ====== TAX ======
		`CREATE TABLE IF NOT EXISTS tax_deadlines (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			tax_type TEXT NOT NULL,
			period_type TEXT NOT NULL,
			period_label TEXT,
			deadline_date TEXT NOT NULL,
			checklist TEXT DEFAULT '[]',
			status TEXT NOT NULL DEFAULT 'upcoming',
			filed_date TEXT,
			amount_due INTEGER,
			amount_paid INTEGER,
			reminder_7d INTEGER DEFAULT 0,
			reminder_3d INTEGER DEFAULT 0,
			reminder_1d INTEGER DEFAULT 0,
			notes TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tax_deadline_org ON tax_deadlines(org_id, deadline_date)`,

		`CREATE TABLE IF NOT EXISTS tax_calculations (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			tax_type TEXT NOT NULL,
			period_label TEXT NOT NULL,
			input_data TEXT NOT NULL,
			calculated_amount INTEGER NOT NULL,
			calculation_breakdown TEXT NOT NULL,
			xml_export_path TEXT,
			calculated_by TEXT REFERENCES users(id),
			calculated_at TEXT NOT NULL
		)`,

		// ====== HR ======
		`CREATE TABLE IF NOT EXISTS employees (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			user_id TEXT REFERENCES users(id),
			employee_code TEXT,
			full_name TEXT NOT NULL,
			phone TEXT,
			email TEXT,
			id_number TEXT,
			department TEXT,
			position TEXT,
			contract_type TEXT NOT NULL,
			contract_start TEXT,
			contract_end TEXT,
			probation_end TEXT,
			base_salary INTEGER DEFAULT 0,
			allowances TEXT DEFAULT '{}',
			social_ins_salary INTEGER,
			social_ins_no TEXT,
			tax_code TEXT,
			dependents INTEGER DEFAULT 0,
			bank_account TEXT,
			bank_name TEXT,
			annual_leave_days REAL DEFAULT 12,
			used_leave_days REAL DEFAULT 0,
			remaining_leave REAL DEFAULT 12,
			misa_employee_id TEXT,
			status TEXT DEFAULT 'active',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_employees_org ON employees(org_id)`,

		`CREATE TABLE IF NOT EXISTS payroll_runs (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			period TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'draft',
			total_gross INTEGER DEFAULT 0,
			total_deductions INTEGER DEFAULT 0,
			total_net INTEGER DEFAULT 0,
			total_employer_cost INTEGER DEFAULT 0,
			approved_by TEXT REFERENCES users(id),
			paid_at TEXT,
			created_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS payroll_items (
			id TEXT PRIMARY KEY,
			payroll_id TEXT NOT NULL REFERENCES payroll_runs(id) ON DELETE CASCADE,
			employee_id TEXT NOT NULL REFERENCES employees(id),
			base_salary INTEGER DEFAULT 0,
			overtime INTEGER DEFAULT 0,
			commission INTEGER DEFAULT 0,
			bonus INTEGER DEFAULT 0,
			allowances INTEGER DEFAULT 0,
			gross_total INTEGER DEFAULT 0,
			bhxh_employee INTEGER DEFAULT 0,
			bhyt_employee INTEGER DEFAULT 0,
			bhtn_employee INTEGER DEFAULT 0,
			pit_amount INTEGER DEFAULT 0,
			other_deductions INTEGER DEFAULT 0,
			total_deductions INTEGER DEFAULT 0,
			net_pay INTEGER DEFAULT 0,
			bhxh_employer INTEGER DEFAULT 0,
			bhyt_employer INTEGER DEFAULT 0,
			bhtn_employer INTEGER DEFAULT 0,
			pit_breakdown TEXT,
			created_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS leave_requests (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			employee_id TEXT NOT NULL REFERENCES employees(id),
			leave_type TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			days REAL NOT NULL,
			reason TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			approved_by TEXT REFERENCES users(id),
			approved_at TEXT,
			created_at TEXT NOT NULL
		)`,

		// ====== SALES ======
		`CREATE TABLE IF NOT EXISTS leads (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			contact_id TEXT REFERENCES contacts(id),
			assigned_to TEXT REFERENCES users(id),
			source TEXT,
			stage TEXT NOT NULL DEFAULT 'new',
			estimated_value INTEGER,
			probability_pct INTEGER DEFAULT 10,
			weighted_value INTEGER,
			expected_close TEXT,
			actual_close TEXT,
			won_lost_reason TEXT,
			notes TEXT,
			last_activity TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_leads_org ON leads(org_id)`,
		`CREATE INDEX IF NOT EXISTS idx_leads_stage ON leads(org_id, stage)`,

		`CREATE TABLE IF NOT EXISTS quotations (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			quote_number TEXT UNIQUE,
			lead_id TEXT REFERENCES leads(id),
			contact_id TEXT NOT NULL REFERENCES contacts(id),
			items TEXT NOT NULL DEFAULT '[]',
			subtotal INTEGER DEFAULT 0,
			vat_total INTEGER DEFAULT 0,
			grand_total INTEGER DEFAULT 0,
			status TEXT DEFAULT 'draft',
			valid_until TEXT,
			pdf_url TEXT,
			created_by TEXT REFERENCES users(id),
			created_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS orders (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			order_number TEXT UNIQUE,
			contact_id TEXT NOT NULL REFERENCES contacts(id),
			quotation_id TEXT REFERENCES quotations(id),
			lead_id TEXT REFERENCES leads(id),
			items TEXT NOT NULL DEFAULT '[]',
			total INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'confirmed',
			order_date TEXT NOT NULL,
			delivery_date TEXT,
			delivered_at TEXT,
			payment_terms TEXT DEFAULT 'cod',
			invoice_id TEXT REFERENCES invoices(id),
			notes TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS engagement_rules (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			name TEXT NOT NULL,
			trigger_type TEXT NOT NULL,
			trigger_config TEXT NOT NULL,
			message_template TEXT NOT NULL,
			channel TEXT DEFAULT 'zalo',
			is_active INTEGER DEFAULT 1,
			created_at TEXT NOT NULL
		)`,

		// ====== OPS ======
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			title TEXT NOT NULL,
			description TEXT,
			assigned_to TEXT REFERENCES users(id),
			created_by TEXT REFERENCES users(id),
			status TEXT NOT NULL DEFAULT 'todo',
			priority TEXT DEFAULT 'medium',
			due_date TEXT,
			completed_at TEXT,
			source_type TEXT,
			source_id TEXT,
			tags TEXT DEFAULT '[]',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_org ON tasks(org_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_assigned ON tasks(assigned_to, status)`,

		`CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			name TEXT NOT NULL,
			category TEXT NOT NULL,
			file_url TEXT NOT NULL,
			file_type TEXT,
			file_size INTEGER,
			ocr_text TEXT,
			entity_type TEXT,
			entity_id TEXT,
			expires_at TEXT,
			expiry_reminded INTEGER DEFAULT 0,
			uploaded_by TEXT REFERENCES users(id),
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_documents_org ON documents(org_id)`,

		`CREATE TABLE IF NOT EXISTS licenses (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			license_type TEXT NOT NULL,
			license_number TEXT,
			issued_by TEXT,
			issued_date TEXT,
			expiry_date TEXT,
			document_id TEXT REFERENCES documents(id),
			status TEXT DEFAULT 'active',
			reminder_90d INTEGER DEFAULT 0,
			reminder_30d INTEGER DEFAULT 0,
			created_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			user_id TEXT NOT NULL REFERENCES users(id),
			title TEXT NOT NULL,
			body TEXT,
			type TEXT NOT NULL,
			channel TEXT DEFAULT 'zalo',
			is_read INTEGER DEFAULT 0,
			entity_type TEXT,
			entity_id TEXT,
			sent_at TEXT,
			read_at TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notif_user ON notifications(user_id, is_read, created_at)`,

		`CREATE TABLE IF NOT EXISTS approval_requests (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			requested_by TEXT NOT NULL REFERENCES users(id),
			current_step INTEGER DEFAULT 1,
			approval_chain TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			created_at TEXT NOT NULL,
			resolved_at TEXT
		)`,

		// ====== RAG ======
		`CREATE TABLE IF NOT EXISTS knowledge_chunks (
			id TEXT PRIMARY KEY,
			org_id TEXT,
			domain TEXT NOT NULL,
			source_name TEXT NOT NULL,
			source_ref TEXT,
			chunk_text TEXT NOT NULL,
			chunk_index INTEGER DEFAULT 0,
			metadata TEXT DEFAULT '{}',
			is_active INTEGER DEFAULT 1,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_domain ON knowledge_chunks(domain)`,

		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id),
			user_id TEXT NOT NULL REFERENCES users(id),
			channel TEXT NOT NULL,
			skill_name TEXT,
			messages TEXT NOT NULL DEFAULT '[]',
			actions_taken TEXT DEFAULT '[]',
			created_at TEXT NOT NULL,
			ended_at TEXT
		)`,
	}

	for _, m := range migrations {
		if _, err := d.Exec(m); err != nil {
			fmt.Fprintf(os.Stderr, "migration failed: %v\nSQL: %.80s...\n", err, m)
			os.Exit(1)
		}
	}

	// Seed default organization if none exists
	var count int
	d.QueryRow("SELECT COUNT(*) FROM organizations").Scan(&count)
	if count == 0 {
		cfg := loadConnections()
		orgID := cfg.Org.ID
		if orgID == "" {
			orgID = "default"
		}
		orgName := cfg.Org.Name
		if orgName == "" {
			orgName = "Doanh nghiep cua toi"
		}
		slug := "default"
		now := vnNowISO()
		d.Exec(`INSERT INTO organizations (id,name,slug,tax_code,created_at,updated_at) VALUES (?,?,?,?,?,?)`,
			orgID, orgName, slug, cfg.Org.TaxCode, now, now)
	}

	okOut(map[string]interface{}{
		"database":    dbPath(),
		"tables":      31,
		"initialized": true,
	})
}
