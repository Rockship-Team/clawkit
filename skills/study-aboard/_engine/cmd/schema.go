package main

import (
	"fmt"
	"os"
)

// cmdInit creates all SQLite tables. Safe to run multiple times (IF NOT EXISTS).
func cmdInit() {
	d := mustDB()

	migrations := []string{

		// ====== PROFILE ======
		`CREATE TABLE IF NOT EXISTS student_profile (
			id TEXT PRIMARY KEY,
			channel TEXT NOT NULL,
			channel_user_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			grade_level INTEGER NOT NULL,
			school_name TEXT NOT NULL,
			curriculum_type TEXT NOT NULL DEFAULT 'VN',
			gpa_value REAL,
			gpa_scale REAL,
			sat_score INTEGER,
			act_score INTEGER,
			toefl_score INTEGER,
			ielts_score REAL,
			ap_scores TEXT DEFAULT '[]',
			intended_major TEXT,
			target_countries TEXT DEFAULT '[]',
			annual_budget_usd INTEGER,
			needs_financial_aid INTEGER NOT NULL DEFAULT 0,
			dream_schools TEXT,
			consent_student INTEGER NOT NULL DEFAULT 0,
			consent_guardian INTEGER NOT NULL DEFAULT 0,
			onboarding_completed_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS ux_student_channel ON student_profile(channel, channel_user_id)`,

		// ====== EC STRATEGY ======
		`CREATE TABLE IF NOT EXISTS extracurricular_activity (
			id TEXT PRIMARY KEY,
			student_id TEXT NOT NULL REFERENCES student_profile(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			role TEXT NOT NULL,
			start_date TEXT,
			end_date TEXT,
			hours_per_week INTEGER,
			achievements TEXT,
			tier INTEGER,
			upgrade_notes TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS ix_ec_student ON extracurricular_activity(student_id)`,

		// ====== SCHOOL MATCHING ======
		`CREATE TABLE IF NOT EXISTS university_record (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			location_city TEXT NOT NULL,
			location_state TEXT,
			location_country TEXT NOT NULL DEFAULT 'US',
			type TEXT NOT NULL,
			usnews_ranking INTEGER,
			acceptance_rate_overall REAL,
			acceptance_rate_international REAL,
			sat_25 INTEGER,
			sat_75 INTEGER,
			act_25 INTEGER,
			act_75 INTEGER,
			gpa_avg_admitted REAL,
			test_policy TEXT NOT NULL DEFAULT 'optional',
			toefl_minimum INTEGER,
			ielts_minimum REAL,
			strong_programs TEXT,
			application_platform TEXT NOT NULL DEFAULT 'Common App',
			ed_available INTEGER NOT NULL DEFAULT 0,
			ea_available INTEGER NOT NULL DEFAULT 0,
			ed_deadline TEXT,
			ea_deadline TEXT,
			rd_deadline TEXT,
			financial_aid_international TEXT NOT NULL DEFAULT 'none',
			avg_annual_cost_usd INTEGER,
			essay_prompts TEXT,
			admits_by_major INTEGER NOT NULL DEFAULT 0,
			cycle_year INTEGER NOT NULL DEFAULT 2026,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS ix_uni_name ON university_record(name)`,
		`CREATE INDEX IF NOT EXISTS ix_uni_country ON university_record(location_country)`,

		// ====== DEADLINE TRACKER ======
		`CREATE TABLE IF NOT EXISTS application (
			id TEXT PRIMARY KEY,
			student_id TEXT NOT NULL REFERENCES student_profile(id) ON DELETE CASCADE,
			university_id TEXT NOT NULL,
			university_name TEXT NOT NULL,
			category TEXT NOT NULL,
			application_type TEXT NOT NULL,
			deadline TEXT,
			essay_status TEXT NOT NULL DEFAULT 'not_started',
			submission_status TEXT NOT NULL DEFAULT 'not_submitted',
			scores_sent INTEGER NOT NULL DEFAULT 0,
			recs_submitted INTEGER NOT NULL DEFAULT 0,
			financial_aid_forms_done INTEGER NOT NULL DEFAULT 0,
			notes TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS ix_app_student ON application(student_id)`,
		`CREATE INDEX IF NOT EXISTS ix_app_deadline ON application(deadline)`,

		`CREATE TABLE IF NOT EXISTS student_checklist (
			id TEXT PRIMARY KEY,
			student_id TEXT NOT NULL UNIQUE REFERENCES student_profile(id) ON DELETE CASCADE,
			common_app_submitted INTEGER NOT NULL DEFAULT 0,
			sat_scores_sent INTEGER NOT NULL DEFAULT 0,
			toefl_scores_sent INTEGER NOT NULL DEFAULT 0,
			ielts_scores_sent INTEGER NOT NULL DEFAULT 0,
			css_profile_done INTEGER NOT NULL DEFAULT 0,
			fafsa_done INTEGER NOT NULL DEFAULT 0,
			rec_letter_1_submitted INTEGER NOT NULL DEFAULT 0,
			rec_letter_2_submitted INTEGER NOT NULL DEFAULT 0,
			rec_letter_3_submitted INTEGER NOT NULL DEFAULT 0,
			transcript_sent INTEGER NOT NULL DEFAULT 0,
			school_report_sent INTEGER NOT NULL DEFAULT 0,
			mid_year_report_sent INTEGER NOT NULL DEFAULT 0,
			notes TEXT,
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS ix_checklist_student ON student_checklist(student_id)`,

		// ====== STUDY PLAN ======
		`CREATE TABLE IF NOT EXISTS study_plan (
			id TEXT PRIMARY KEY,
			student_id TEXT NOT NULL REFERENCES student_profile(id) ON DELETE CASCADE,
			test_type TEXT NOT NULL,
			current_score INTEGER,
			target_score INTEGER NOT NULL,
			test_date TEXT NOT NULL,
			plan_weeks TEXT NOT NULL DEFAULT '[]',
			active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS ix_sp_student ON study_plan(student_id)`,

		`CREATE TABLE IF NOT EXISTS study_plan_checkin (
			id TEXT PRIMARY KEY,
			study_plan_id TEXT NOT NULL REFERENCES study_plan(id) ON DELETE CASCADE,
			checkin_date TEXT NOT NULL DEFAULT (datetime('now')),
			reported_score INTEGER,
			adjustment_notes TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS ix_checkin_plan ON study_plan_checkin(study_plan_id)`,

		// ====== ESSAY REVIEW ======
		`CREATE TABLE IF NOT EXISTS essay_draft (
			id TEXT PRIMARY KEY,
			student_id TEXT NOT NULL REFERENCES student_profile(id) ON DELETE CASCADE,
			application_id TEXT REFERENCES application(id),
			essay_type TEXT NOT NULL,
			prompt_text TEXT,
			version INTEGER NOT NULL DEFAULT 1,
			content TEXT NOT NULL,
			word_count INTEGER NOT NULL,
			score_authenticity REAL,
			score_structure REAL,
			score_specificity REAL,
			score_voice REAL,
			score_so_what REAL,
			score_grammar REAL,
			feedback_json TEXT,
			ai_generated_flag INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS ix_essay_student ON essay_draft(student_id)`,

		// ====== PRE-DEPARTURE ======
		`CREATE TABLE IF NOT EXISTS visa_checklist (
			id TEXT PRIMARY KEY,
			student_id TEXT NOT NULL UNIQUE REFERENCES student_profile(id) ON DELETE CASCADE,
			school_name TEXT,
			program_start_date TEXT,
			i20_received INTEGER NOT NULL DEFAULT 0,
			sevis_fee_paid INTEGER NOT NULL DEFAULT 0,
			sevis_id TEXT,
			ds160_completed INTEGER NOT NULL DEFAULT 0,
			visa_appointment_booked INTEGER NOT NULL DEFAULT 0,
			visa_appointment_date TEXT,
			visa_interview_done INTEGER NOT NULL DEFAULT 0,
			visa_approved INTEGER NOT NULL DEFAULT 0,
			visa_approved_date TEXT,
			financial_proof_ready INTEGER NOT NULL DEFAULT 0,
			bank_statement_ready INTEGER NOT NULL DEFAULT 0,
			sponsor_letter_ready INTEGER NOT NULL DEFAULT 0,
			housing_arranged INTEGER NOT NULL DEFAULT 0,
			housing_type TEXT,
			housing_address TEXT,
			health_insurance_arranged INTEGER NOT NULL DEFAULT 0,
			vaccinations_done INTEGER NOT NULL DEFAULT 0,
			medical_records_ready INTEGER NOT NULL DEFAULT 0,
			flight_booked INTEGER NOT NULL DEFAULT 0,
			flight_date TEXT,
			orientation_registered INTEGER NOT NULL DEFAULT 0,
			orientation_date TEXT,
			course_registration_done INTEGER NOT NULL DEFAULT 0,
			notes TEXT,
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS ix_visa_student ON visa_checklist(student_id)`,

		// ====== OFFER COMPARISON ======
		`CREATE TABLE IF NOT EXISTS admission_offer (
			id TEXT PRIMARY KEY,
			student_id TEXT NOT NULL REFERENCES student_profile(id) ON DELETE CASCADE,
			university_name TEXT NOT NULL,
			decision_type TEXT NOT NULL,
			result TEXT NOT NULL,
			tuition_fees_usd INTEGER,
			room_board_usd INTEGER,
			other_fees_usd INTEGER,
			scholarship_usd INTEGER NOT NULL DEFAULT 0,
			grant_usd INTEGER NOT NULL DEFAULT 0,
			loan_offered_usd INTEGER NOT NULL DEFAULT 0,
			work_study_usd INTEGER NOT NULL DEFAULT 0,
			major TEXT,
			program_name TEXT,
			program_start_date TEXT,
			offer_deadline TEXT,
			deposit_required_usd INTEGER,
			student_decision TEXT NOT NULL DEFAULT 'pending',
			program_strength INTEGER,
			location_fit INTEGER,
			campus_culture_fit INTEGER,
			career_outcome_fit INTEGER,
			notes TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS ix_offer_student ON admission_offer(student_id)`,
		`CREATE INDEX IF NOT EXISTS ix_offer_result ON admission_offer(result)`,

		// ====== CRON JOBS ======
		`CREATE TABLE IF NOT EXISTS cron_job (
			id TEXT PRIMARY KEY,
			student_id TEXT NOT NULL REFERENCES student_profile(id) ON DELETE CASCADE,
			openclaw_job_name TEXT NOT NULL UNIQUE,
			job_type TEXT NOT NULL,
			application_id TEXT REFERENCES application(id) ON DELETE SET NULL,
			study_plan_id TEXT REFERENCES study_plan(id) ON DELETE SET NULL,
			urgency_days INTEGER,
			scheduled_at TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS ix_cron_student ON cron_job(student_id)`,
		`CREATE INDEX IF NOT EXISTS ix_cron_status ON cron_job(status)`,
	}

	for _, m := range migrations {
		if _, err := d.Exec(m); err != nil {
			fmt.Fprintf(os.Stderr, "migration failed: %v\nSQL: %.120s...\n", err, m)
			os.Exit(1)
		}
	}

	okOut(map[string]interface{}{
		"database":    dbPath(),
		"tables":      12,
		"initialized": true,
	})
}
