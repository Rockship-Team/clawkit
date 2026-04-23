package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

func cmdEmployee(args []string) {
	if len(args) == 0 {
		errOut("usage: employee add|list|get|update")
	}
	org := defaultOrgID()
	switch args[0] {
	case "add":
		// employee add <name> <department> <position> <contract_type> <base_salary> [phone] [email]
		if len(args) < 6 {
			errOut("usage: employee add <name> <dept> <position> <fulltime|parttime|contractor> <salary> [phone] [email]")
		}
		id := newID()
		salary := parseVND(args[5])
		phone, email := "", ""
		if len(args) > 6 {
			phone = args[6]
		}
		if len(args) > 7 {
			email = args[7]
		}
		now := vnNowISO()
		_, err := exec(`INSERT INTO employees (id,org_id,full_name,department,position,contract_type,base_salary,phone,email,social_ins_salary,status,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, org, args[1], args[2], args[3], args[4], salary, phone, email, salary, "active", now, now)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"id": id, "name": args[1], "salary": salary})

	case "list":
		filter := "active"
		if len(args) > 1 {
			filter = args[1]
		}
		q := "SELECT id,employee_code,full_name,department,position,contract_type,base_salary,status FROM employees WHERE org_id=?"
		qargs := []interface{}{org}
		if filter != "all" {
			q += " AND status=?"
			qargs = append(qargs, filter)
		}
		q += " ORDER BY full_name LIMIT 100"
		rows, _ := queryRows(q, qargs...)
		okOut(map[string]interface{}{"employees": rows, "count": len(rows)})

	case "get":
		if len(args) < 2 {
			errOut("usage: employee get <id>")
		}
		row, _ := queryOne("SELECT * FROM employees WHERE id=? AND org_id=?", args[1], org)
		if row == nil {
			errOut("employee not found")
		}
		okOut(map[string]interface{}{"employee": row})

	case "update":
		if len(args) < 4 {
			errOut("usage: employee update <id> <field> <value>")
		}
		allowed := map[string]bool{
			"full_name": true, "department": true, "position": true, "base_salary": true,
			"phone": true, "email": true, "dependents": true, "bank_account": true,
			"bank_name": true, "status": true, "contract_end": true, "social_ins_salary": true,
		}
		if !allowed[args[2]] {
			errOut("cannot update field: " + args[2])
		}
		q := fmt.Sprintf("UPDATE employees SET %s=?, updated_at=? WHERE id=? AND org_id=?", args[2])
		exec(q, args[3], vnNowISO(), args[1], org)
		okOut(map[string]interface{}{"updated": args[1], "field": args[2]})

	default:
		errOut("unknown employee command: " + args[0])
	}
}

func cmdPayroll(args []string) {
	if len(args) == 0 {
		errOut("usage: payroll calculate|list|get|approve")
	}
	org := defaultOrgID()
	switch args[0] {
	case "calculate":
		// payroll calculate <period> e.g. 2026-04
		if len(args) < 2 {
			errOut("usage: payroll calculate <YYYY-MM>")
		}
		period := args[1]

		// Check if already exists
		existing, _ := queryOne("SELECT id FROM payroll_runs WHERE org_id=? AND period=?", org, period)
		if existing != nil {
			errOut("payroll already exists for " + period + ". Delete or use a new period.")
		}

		// Get active employees
		emps, _ := queryRows("SELECT * FROM employees WHERE org_id=? AND status='active'", org)
		if len(emps) == 0 {
			errOut("no active employees")
		}

		runID := newID()
		now := vnNowISO()
		_, err := exec("INSERT INTO payroll_runs (id,org_id,period,status,created_at) VALUES (?,?,?,?,?)",
			runID, org, period, "draft", now)
		if err != nil {
			errOut(err.Error())
		}

		var totalGross, totalDed, totalNet, totalEmployer int64

		for _, emp := range emps {
			base, _ := emp["base_salary"].(int64)
			insSalary, _ := emp["social_ins_salary"].(int64)
			if insSalary == 0 {
				insSalary = base
			}
			deps := 0
			if d, ok := emp["dependents"].(int64); ok {
				deps = int(d)
			}

			// Parse allowances JSON
			allowanceTotal := int64(0)
			if raw, ok := emp["allowances"].(string); ok && raw != "" && raw != "{}" {
				var allowMap map[string]int64
				if json.Unmarshal([]byte(raw), &allowMap) == nil {
					for _, v := range allowMap {
						allowanceTotal += v
					}
				}
			}

			gross := base + allowanceTotal

			// Employee insurance
			bhxhE := int64(float64(insSalary) * 0.08)
			bhytE := int64(float64(insSalary) * 0.015)
			bhtnE := int64(float64(insSalary) * 0.01)

			// PIT
			insurance := bhxhE + bhytE + bhtnE
			selfDed := int64(selfDeduction)
			depDed := int64(dependentDeduction) * int64(deps)
			taxable := gross - insurance - selfDed - depDed
			if taxable < 0 {
				taxable = 0
			}
			pit := calculatePITAmount(taxable)

			totalDedItem := insurance + pit
			netPay := gross - totalDedItem

			// Employer insurance
			bhxhR := int64(float64(insSalary) * 0.175)
			bhytR := int64(float64(insSalary) * 0.03)
			bhtnR := int64(float64(insSalary) * 0.01)

			itemID := newID()
			exec(`INSERT INTO payroll_items (id,payroll_id,employee_id,base_salary,allowances,gross_total,
				bhxh_employee,bhyt_employee,bhtn_employee,pit_amount,total_deductions,net_pay,
				bhxh_employer,bhyt_employer,bhtn_employer,created_at)
				VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				itemID, runID, emp["id"], base, allowanceTotal, gross,
				bhxhE, bhytE, bhtnE, pit, totalDedItem, netPay,
				bhxhR, bhytR, bhtnR, now)

			totalGross += gross
			totalDed += totalDedItem
			totalNet += netPay
			totalEmployer += bhxhR + bhytR + bhtnR
		}

		exec("UPDATE payroll_runs SET total_gross=?,total_deductions=?,total_net=?,total_employer_cost=? WHERE id=?",
			totalGross, totalDed, totalNet, totalGross+totalEmployer, runID)

		okOut(map[string]interface{}{
			"payroll_id":       runID,
			"period":           period,
			"employees":        len(emps),
			"total_gross":      totalGross,
			"total_deductions": totalDed,
			"total_net":        totalNet,
			"employer_cost":    totalGross + totalEmployer,
			"status":           "draft",
		})

	case "list":
		rows, _ := queryRows("SELECT id,period,status,total_gross,total_net,total_employer_cost FROM payroll_runs WHERE org_id=? ORDER BY period DESC LIMIT 24", org)
		okOut(map[string]interface{}{"payroll_runs": rows, "count": len(rows)})

	case "get":
		if len(args) < 2 {
			errOut("usage: payroll get <id_or_period>")
		}
		run, _ := queryOne("SELECT * FROM payroll_runs WHERE (id=? OR period=?) AND org_id=?", args[1], args[1], org)
		if run == nil {
			errOut("payroll not found")
		}
		items, _ := queryRows(`SELECT pi.*, e.full_name, e.department FROM payroll_items pi
			JOIN employees e ON pi.employee_id=e.id WHERE pi.payroll_id=? ORDER BY e.full_name`, run["id"])
		run["items"] = items
		okOut(map[string]interface{}{"payroll": run})

	case "approve":
		if len(args) < 2 {
			errOut("usage: payroll approve <id>")
		}
		exec("UPDATE payroll_runs SET status='approved' WHERE id=? AND org_id=?", args[1], org)
		okOut(map[string]interface{}{"approved": args[1]})

	default:
		errOut("unknown payroll command: " + args[0])
	}
}

func calculatePITAmount(taxable int64) int64 {
	if taxable <= 0 {
		return 0
	}
	remaining := taxable
	prevLimit := int64(0)
	total := int64(0)
	for _, b := range pitBrackets {
		if remaining <= 0 {
			break
		}
		bSize := b.Upper - prevLimit
		if b.Upper == math.MaxInt64 {
			bSize = remaining
		}
		if remaining < bSize {
			bSize = remaining
		}
		total += int64(float64(bSize) * b.Rate / 100)
		remaining -= bSize
		prevLimit = b.Upper
	}
	return total
}

func cmdLeave(args []string) {
	if len(args) == 0 {
		errOut("usage: leave request|list|approve|reject|balance")
	}
	org := defaultOrgID()
	switch args[0] {
	case "request":
		// leave request <employee_id> <type> <start> <end> <days> [reason]
		if len(args) < 6 {
			errOut("usage: leave request <emp_id> <annual|sick|personal|maternity|unpaid> <start> <end> <days> [reason]")
		}
		days, _ := strconv.ParseFloat(args[5], 64)
		reason := ""
		if len(args) > 6 {
			reason = args[6]
		}
		id := newID()
		_, err := exec(`INSERT INTO leave_requests (id,org_id,employee_id,leave_type,start_date,end_date,days,reason,status,created_at)
			VALUES (?,?,?,?,?,?,?,?,?,?)`, id, org, args[1], args[2], args[3], args[4], days, reason, "pending", vnNowISO())
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"id": id, "employee_id": args[1], "days": days, "status": "pending"})

	case "list":
		status := "pending"
		if len(args) > 1 {
			status = args[1]
		}
		q := "SELECT lr.*, e.full_name FROM leave_requests lr JOIN employees e ON lr.employee_id=e.id WHERE lr.org_id=?"
		qargs := []interface{}{org}
		if status != "all" {
			q += " AND lr.status=?"
			qargs = append(qargs, status)
		}
		q += " ORDER BY lr.created_at DESC LIMIT 50"
		rows, _ := queryRows(q, qargs...)
		okOut(map[string]interface{}{"leave_requests": rows, "count": len(rows)})

	case "approve":
		if len(args) < 2 {
			errOut("usage: leave approve <id>")
		}
		// Get the request to update employee leave balance
		req, _ := queryOne("SELECT employee_id,days FROM leave_requests WHERE id=? AND org_id=? AND status='pending'", args[1], org)
		if req == nil {
			errOut("leave request not found or already processed")
		}
		exec("UPDATE leave_requests SET status='approved', approved_at=? WHERE id=?", vnNowISO(), args[1])
		if days, ok := req["days"].(float64); ok {
			exec("UPDATE employees SET used_leave_days=used_leave_days+?, remaining_leave=remaining_leave-? WHERE id=?",
				days, days, req["employee_id"])
		}
		okOut(map[string]interface{}{"approved": args[1]})

	case "reject":
		if len(args) < 2 {
			errOut("usage: leave reject <id>")
		}
		exec("UPDATE leave_requests SET status='rejected' WHERE id=? AND org_id=?", args[1], org)
		okOut(map[string]interface{}{"rejected": args[1]})

	case "balance":
		rows, _ := queryRows("SELECT id,full_name,department,annual_leave_days,used_leave_days,remaining_leave FROM employees WHERE org_id=? AND status='active' ORDER BY full_name", org)
		okOut(map[string]interface{}{"balances": rows, "count": len(rows)})

	default:
		errOut("unknown leave command: " + args[0])
	}
}
