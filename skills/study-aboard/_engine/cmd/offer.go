package main

import "fmt"

func cmdOffer(args []string) {
	if len(args) == 0 {
		errOut("usage: offer add|list|decide|compare")
	}
	switch args[0] {
	case "add":
		// offer add <student_id> <university_name> <ED|EA|RD> <accepted|rejected|waitlisted|deferred> [tuition] [scholarship]
		if len(args) < 5 {
			errOut("usage: offer add <student_id> <university_name> <decision_type> <accepted|rejected|waitlisted|deferred> [tuition_usd] [scholarship_usd]")
		}
		studentID := args[1]
		uniName := args[2]
		decisionType := args[3]
		result := args[4]

		var tuition, scholarship interface{}
		if len(args) > 5 && args[5] != "-" {
			tuition = args[5]
		}
		if len(args) > 6 && args[6] != "-" {
			scholarship = args[6]
		}

		offerID := newID()
		now := vnNowISO()
		_, err := exec(
			`INSERT INTO admission_offer
			 (id,student_id,university_name,decision_type,result,tuition_fees_usd,scholarship_usd,created_at,updated_at)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			offerID, studentID, uniName, decisionType, result, tuition, scholarship, now, now,
		)
		if err != nil {
			errOut("insert failed: " + err.Error())
		}
		okOut(map[string]interface{}{
			"offer_id":      offerID,
			"university":    uniName,
			"result":        result,
			"tuition":       tuition,
			"scholarship":   scholarship,
		})

	case "update":
		// offer update <offer_id> <field> <value>
		if len(args) < 4 {
			errOut("usage: offer update <offer_id> <field> <value>")
		}
		offerID, field, val := args[1], args[2], args[3]
		allowed := map[string]bool{
			"tuition_fees_usd": true, "room_board_usd": true, "other_fees_usd": true,
			"scholarship_usd": true, "grant_usd": true, "loan_offered_usd": true, "work_study_usd": true,
			"major": true, "program_name": true, "program_start_date": true,
			"offer_deadline": true, "deposit_required_usd": true,
			"program_strength": true, "location_fit": true, "campus_culture_fit": true, "career_outcome_fit": true,
			"notes": true,
		}
		if !allowed[field] {
			errOut("cannot update field: " + field)
		}
		q := fmt.Sprintf("UPDATE admission_offer SET %s=?, updated_at=? WHERE id=?", field)
		_, err := exec(q, val, vnNowISO(), offerID)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"updated": offerID, "field": field})

	case "decide":
		// offer decide <offer_id> <accept|decline|waitlisted>
		if len(args) < 3 {
			errOut("usage: offer decide <offer_id> <accept|decline|waitlisted>")
		}
		_, err := exec(
			"UPDATE admission_offer SET student_decision=?, updated_at=? WHERE id=?",
			args[2], vnNowISO(), args[1],
		)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"offer_id": args[1], "decision": args[2]})

	case "list":
		// offer list <student_id>
		if len(args) < 2 {
			errOut("usage: offer list <student_id>")
		}
		rows, err := queryRows(
			`SELECT id,university_name,decision_type,result,
			 tuition_fees_usd,room_board_usd,scholarship_usd,grant_usd,loan_offered_usd,
			 major,offer_deadline,deposit_required_usd,student_decision,
			 program_strength,location_fit,campus_culture_fit,career_outcome_fit
			 FROM admission_offer WHERE student_id=? ORDER BY result,created_at DESC`,
			args[1],
		)
		if err != nil {
			errOut(err.Error())
		}
		// Compute net cost for accepted offers
		for _, r := range rows {
			tuition, _ := r["tuition_fees_usd"].(int64)
			room, _ := r["room_board_usd"].(int64)
			other, _ := r["other_fees_usd"].(int64)
			scholarship, _ := r["scholarship_usd"].(int64)
			grant, _ := r["grant_usd"].(int64)
			if tuition > 0 {
				r["net_cost_usd"] = tuition + room + other - scholarship - grant
			}
		}
		okOut(map[string]interface{}{"offers": rows, "count": len(rows)})

	case "compare":
		// offer compare <student_id>
		if len(args) < 2 {
			errOut("usage: offer compare <student_id>")
		}
		rows, err := queryRows(
			`SELECT id,university_name,decision_type,result,
			 tuition_fees_usd,room_board_usd,other_fees_usd,scholarship_usd,grant_usd,
			 loan_offered_usd,work_study_usd,major,offer_deadline,deposit_required_usd,
			 program_strength,location_fit,campus_culture_fit,career_outcome_fit,notes,student_decision
			 FROM admission_offer
			 WHERE student_id=? AND result IN ('accepted','waitlisted')
			 ORDER BY result DESC, tuition_fees_usd ASC`,
			args[1],
		)
		if err != nil {
			errOut(err.Error())
		}
		for _, r := range rows {
			tuition, _ := r["tuition_fees_usd"].(int64)
			room, _ := r["room_board_usd"].(int64)
			other, _ := r["other_fees_usd"].(int64)
			scholarship, _ := r["scholarship_usd"].(int64)
			grant, _ := r["grant_usd"].(int64)
			if tuition > 0 {
				r["net_cost_usd"] = tuition + room + other - scholarship - grant
			}
		}
		profile, _ := queryOne("SELECT annual_budget_usd FROM student_profile WHERE id=?", args[1])
		okOut(map[string]interface{}{
			"accepted_offers": rows,
			"count":           len(rows),
			"budget":          profile,
		})

	default:
		errOut("unknown offer command: " + args[0])
	}
}
