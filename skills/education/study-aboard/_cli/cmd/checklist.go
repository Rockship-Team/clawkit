package main

import "fmt"

func cmdChecklist(args []string) {
	if len(args) == 0 {
		errOut("usage: checklist get|update|notes")
	}
	switch args[0] {
	case "get":
		// checklist get <student_id>
		if len(args) < 2 {
			errOut("usage: checklist get <student_id>")
		}
		row, err := queryOne("SELECT * FROM student_checklist WHERE student_id=?", args[1])
		if err != nil {
			errOut(err.Error())
		}
		if row == nil {
			// Auto-create empty checklist
			id := newID()
			exec(
				`INSERT INTO student_checklist (id,student_id) VALUES (?,?)`,
				id, args[1],
			)
			row, _ = queryOne("SELECT * FROM student_checklist WHERE student_id=?", args[1])
		}
		// Count done/total
		checkKeys := []string{
			"common_app_submitted", "sat_scores_sent", "toefl_scores_sent", "ielts_scores_sent",
			"css_profile_done", "fafsa_done", "rec_letter_1_submitted", "rec_letter_2_submitted",
			"rec_letter_3_submitted", "transcript_sent", "school_report_sent", "mid_year_report_sent",
		}
		done, total := 0, len(checkKeys)
		for _, k := range checkKeys {
			if v, ok := row[k]; ok {
				switch vt := v.(type) {
				case int64:
					if vt == 1 {
						done++
					}
				case float64:
					if vt == 1 {
						done++
					}
				}
			}
		}
		row["done"] = done
		row["total"] = total
		okOut(map[string]interface{}{"checklist": row})

	case "update":
		// checklist update <student_id> <item_key> <0|1>
		if len(args) < 4 {
			errOut("usage: checklist update <student_id> <item_key> <0|1>")
		}
		studentID, key, val := args[1], args[2], args[3]
		allowed := map[string]bool{
			"common_app_submitted": true, "sat_scores_sent": true, "toefl_scores_sent": true,
			"ielts_scores_sent": true, "css_profile_done": true, "fafsa_done": true,
			"rec_letter_1_submitted": true, "rec_letter_2_submitted": true, "rec_letter_3_submitted": true,
			"transcript_sent": true, "school_report_sent": true, "mid_year_report_sent": true,
		}
		if !allowed[key] {
			errOut("unknown checklist item: " + key)
		}
		// Upsert checklist row
		existing, _ := queryOne("SELECT id FROM student_checklist WHERE student_id=?", studentID)
		if existing == nil {
			id := newID()
			exec("INSERT INTO student_checklist (id,student_id) VALUES (?,?)", id, studentID)
		}
		q := fmt.Sprintf("UPDATE student_checklist SET %s=?, updated_at=? WHERE student_id=?", key)
		_, err := exec(q, val, vnNowISO(), studentID)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"student_id": studentID, "item": key, "done": val})

	case "notes":
		// checklist notes <student_id> <text>
		if len(args) < 3 {
			errOut("usage: checklist notes <student_id> <text>")
		}
		existing, _ := queryOne("SELECT id FROM student_checklist WHERE student_id=?", args[1])
		if existing == nil {
			id := newID()
			exec("INSERT INTO student_checklist (id,student_id) VALUES (?,?)", id, args[1])
		}
		_, err := exec(
			"UPDATE student_checklist SET notes=?, updated_at=? WHERE student_id=?",
			args[2], vnNowISO(), args[1],
		)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"student_id": args[1], "notes_updated": true})

	default:
		errOut("unknown checklist command: " + args[0])
	}
}
