package main

import "fmt"

func cmdVisa(args []string) {
	if len(args) == 0 {
		errOut("usage: visa get|update")
	}
	switch args[0] {
	case "get":
		// visa get <student_id>
		if len(args) < 2 {
			errOut("usage: visa get <student_id>")
		}
		row, err := queryOne("SELECT * FROM visa_checklist WHERE student_id=?", args[1])
		if err != nil {
			errOut(err.Error())
		}
		if row == nil {
			// Auto-create empty checklist
			id := newID()
			exec("INSERT INTO visa_checklist (id,student_id) VALUES (?,?)", id, args[1])
			row, _ = queryOne("SELECT * FROM visa_checklist WHERE student_id=?", args[1])
		}
		okOut(map[string]interface{}{"visa_checklist": row})

	case "update":
		// visa update <student_id> <item_key> <0|1|text>
		if len(args) < 4 {
			errOut("usage: visa update <student_id> <item_key> <value>")
		}
		studentID, key, val := args[1], args[2], args[3]
		allowed := map[string]bool{
			"school_name": true, "program_start_date": true,
			"i20_received": true, "sevis_fee_paid": true, "sevis_id": true,
			"ds160_completed": true, "visa_appointment_booked": true, "visa_appointment_date": true,
			"visa_interview_done": true, "visa_approved": true, "visa_approved_date": true,
			"financial_proof_ready": true, "bank_statement_ready": true, "sponsor_letter_ready": true,
			"housing_arranged": true, "housing_type": true, "housing_address": true,
			"health_insurance_arranged": true, "vaccinations_done": true, "medical_records_ready": true,
			"flight_booked": true, "flight_date": true,
			"orientation_registered": true, "orientation_date": true,
			"course_registration_done": true, "notes": true,
		}
		if !allowed[key] {
			errOut("unknown visa field: " + key)
		}
		// Upsert
		existing, _ := queryOne("SELECT id FROM visa_checklist WHERE student_id=?", studentID)
		if existing == nil {
			id := newID()
			exec("INSERT INTO visa_checklist (id,student_id) VALUES (?,?)", id, studentID)
		}
		q := fmt.Sprintf("UPDATE visa_checklist SET %s=?, updated_at=? WHERE student_id=?", key)
		_, err := exec(q, val, vnNowISO(), studentID)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"student_id": studentID, "field": key, "value": val})

	default:
		errOut("unknown visa command: " + args[0])
	}
}
