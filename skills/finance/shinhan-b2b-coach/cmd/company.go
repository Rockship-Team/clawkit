package main

import (
	"fmt"
	"strconv"
	"strings"
)

// getCompanyID returns the company_id of the first (only) company in company_profile.
func getCompanyID() string {
	db := mustDB()
	defer db.Close()
	row, err := queryOne(db, "SELECT company_id FROM company_profile LIMIT 1")
	if err != nil {
		errOut("failed to query company: " + err.Error())
	}
	if row == nil {
		errOut("no company profile found — run 'init' or 'company add' first")
	}
	cid, ok := row["company_id"].(string)
	if !ok || cid == "" {
		errOut("company_id is empty — run 'company add' first")
	}
	return cid
}

func cmdCompany(args []string) {
	if len(args) == 0 {
		errOut("usage: company <add|get|update|onboard>")
	}

	switch args[0] {
	case "add":
		cmdCompanyAdd(args[1:])
	case "get":
		cmdCompanyGet()
	case "update":
		cmdCompanyUpdate(args[1:])
	case "onboard":
		cmdCompanyOnboard()
	default:
		errOut("unknown company subcommand: " + args[0])
	}
}

func cmdCompanyAdd(args []string) {
	if len(args) < 2 {
		errOut("usage: company add <name> <industry> [tax_code] [employee_count] [monthly_revenue_avg]")
	}

	name := args[0]
	industry := args[1]

	taxCode := ""
	if len(args) > 2 {
		taxCode = args[2]
	}
	employeeCount := 0
	if len(args) > 3 {
		employeeCount, _ = strconv.Atoi(args[3])
	}
	var monthlyRevenueAvg int64
	if len(args) > 4 {
		monthlyRevenueAvg = parseVND(args[4])
	}

	db := mustDB()
	defer db.Close()

	// Delete existing company (single-tenant: one company per install)
	_, _ = exec(db, "DELETE FROM company_profile")

	companyID := newID()
	now := vnNowISO()

	_, err := exec(db,
		`INSERT INTO company_profile (company_id, name, industry, tax_code, employee_count, monthly_revenue_avg, onboarded, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'false', ?)`,
		companyID, name, industry, taxCode, employeeCount, monthlyRevenueAvg, now,
	)
	if err != nil {
		errOut("failed to add company: " + err.Error())
	}

	row, err := queryOne(db, "SELECT * FROM company_profile WHERE company_id = ?", companyID)
	if err != nil {
		errOut("failed to read back company: " + err.Error())
	}

	okOut(map[string]interface{}{
		"message": fmt.Sprintf("Company '%s' created", name),
		"company": row,
	})
}

func cmdCompanyGet() {
	db := mustDB()
	defer db.Close()

	row, err := queryOne(db, "SELECT * FROM company_profile LIMIT 1")
	if err != nil {
		errOut("failed to query company: " + err.Error())
	}
	if row == nil {
		errOut("no company profile found — run 'company add' first")
	}

	jsonOut(row)
}

var allowedCompanyFields = map[string]bool{
	"name":                  true,
	"industry":              true,
	"tax_code":              true,
	"employee_count":        true,
	"monthly_revenue_avg":   true,
	"monthly_expense_avg":   true,
	"cash_reserve":          true,
	"accounting_software":   true,
	"fiscal_year_start":     true,
	"current_bank_products": true,
}

var numericCompanyFields = map[string]bool{
	"employee_count":      true,
	"monthly_revenue_avg": true,
	"monthly_expense_avg": true,
	"cash_reserve":        true,
	"fiscal_year_start":   true,
}

func cmdCompanyUpdate(args []string) {
	if len(args) < 2 {
		errOut("usage: company update <field> <value>")
	}

	field := strings.ToLower(args[0])
	value := strings.Join(args[1:], " ")

	if !allowedCompanyFields[field] {
		allowed := make([]string, 0, len(allowedCompanyFields))
		for k := range allowedCompanyFields {
			allowed = append(allowed, k)
		}
		errOut(fmt.Sprintf("field '%s' not allowed. Allowed: %s", field, strings.Join(allowed, ", ")))
	}

	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	var err error
	if numericCompanyFields[field] {
		numVal := parseVND(value)
		_, err = exec(db, fmt.Sprintf("UPDATE company_profile SET %s = ? WHERE company_id = ?", field), numVal, companyID)
	} else {
		_, err = exec(db, fmt.Sprintf("UPDATE company_profile SET %s = ? WHERE company_id = ?", field), value, companyID)
	}
	if err != nil {
		errOut("failed to update company: " + err.Error())
	}

	row, err := queryOne(db, "SELECT * FROM company_profile WHERE company_id = ?", companyID)
	if err != nil {
		errOut("failed to read back company: " + err.Error())
	}

	okOut(map[string]interface{}{
		"message": fmt.Sprintf("Updated %s", field),
		"company": row,
	})
}

func cmdCompanyOnboard() {
	companyID := getCompanyID()

	db := mustDB()
	defer db.Close()

	_, err := exec(db, "UPDATE company_profile SET onboarded = 'true' WHERE company_id = ?", companyID)
	if err != nil {
		errOut("failed to onboard company: " + err.Error())
	}

	okOut(map[string]interface{}{
		"message": "Company onboarded successfully",
	})
}
