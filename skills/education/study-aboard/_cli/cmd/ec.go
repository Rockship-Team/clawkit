package main

import "fmt"

func cmdEC(args []string) {
	if len(args) == 0 {
		errOut("usage: ec add|list|update-tier")
	}
	switch args[0] {
	case "add":
		// ec add <student_id> <name> <role> [hours_per_week] [achievements]
		if len(args) < 4 {
			errOut("usage: ec add <student_id> <name> <role> [hours_per_week] [achievements]")
		}
		studentID := args[1]
		name := args[2]
		role := args[3]
		var hours interface{}
		if len(args) > 4 && args[4] != "-" {
			hours = args[4]
		}
		achievements := ""
		if len(args) > 5 {
			achievements = args[5]
		}
		ecID := newID()
		_, err := exec(
			`INSERT INTO extracurricular_activity (id,student_id,name,role,hours_per_week,achievements,created_at)
			 VALUES (?,?,?,?,?,?,?)`,
			ecID, studentID, name, role, hours, achievements, vnNowISO(),
		)
		if err != nil {
			errOut("insert failed: " + err.Error())
		}
		okOut(map[string]interface{}{"ec_id": ecID, "name": name, "role": role})

	case "list":
		// ec list <student_id>
		if len(args) < 2 {
			errOut("usage: ec list <student_id>")
		}
		rows, err := queryRows(
			`SELECT id,name,role,hours_per_week,achievements,tier,upgrade_notes,created_at
			 FROM extracurricular_activity WHERE student_id=? ORDER BY tier ASC NULLS LAST, created_at ASC`,
			args[1],
		)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"activities": rows, "count": len(rows)})

	case "update-tier":
		// ec update-tier <ec_id> <tier> [upgrade_notes]
		if len(args) < 3 {
			errOut("usage: ec update-tier <ec_id> <tier> [upgrade_notes]")
		}
		ecID := args[1]
		tier := args[2]
		notes := ""
		if len(args) > 3 {
			notes = args[3]
		}
		q := "UPDATE extracurricular_activity SET tier=?"
		qargs := []interface{}{tier}
		if notes != "" {
			q += ", upgrade_notes=?"
			qargs = append(qargs, notes)
		}
		q += " WHERE id=?"
		qargs = append(qargs, ecID)
		_, err := exec(fmt.Sprintf("%s", q), qargs...)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"updated": ecID, "tier": tier})

	default:
		errOut("unknown ec command: " + args[0])
	}
}
