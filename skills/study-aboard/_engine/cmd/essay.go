package main

func cmdEssay(args []string) {
	if len(args) == 0 {
		errOut("usage: essay submit|list|get")
	}
	switch args[0] {
	case "submit":
		// essay submit <student_id> <essay_type> <word_count> <content>
		// essay_type: personal_statement | supplemental | scholarship
		if len(args) < 5 {
			errOut("usage: essay submit <student_id> <essay_type> <word_count> <content>")
		}
		studentID := args[1]
		essayType := args[2]
		wordCount := args[3]
		content := args[4]

		// Find next version for same student+type
		row, _ := queryOne(
			"SELECT MAX(version) as v FROM essay_draft WHERE student_id=? AND essay_type=?",
			studentID, essayType,
		)
		version := 1
		if row != nil {
			if v, ok := row["v"].(int64); ok {
				version = int(v) + 1
			}
		}

		draftID := newID()
		_, err := exec(
			`INSERT INTO essay_draft (id,student_id,essay_type,version,content,word_count,created_at)
			 VALUES (?,?,?,?,?,?,?)`,
			draftID, studentID, essayType, version, content, wordCount, vnNowISO(),
		)
		if err != nil {
			errOut("insert failed: " + err.Error())
		}
		okOut(map[string]interface{}{
			"draft_id":   draftID,
			"essay_type": essayType,
			"version":    version,
			"word_count": wordCount,
		})

	case "save-scores":
		// essay save-scores <draft_id> <authenticity> <structure> <specificity> <voice> <so_what> <grammar> [feedback_json]
		if len(args) < 8 {
			errOut("usage: essay save-scores <draft_id> <authenticity> <structure> <specificity> <voice> <so_what> <grammar> [feedback_json]")
		}
		draftID := args[1]
		feedback := ""
		if len(args) > 8 {
			feedback = args[8]
		}
		_, err := exec(
			`UPDATE essay_draft
			 SET score_authenticity=?,score_structure=?,score_specificity=?,
			     score_voice=?,score_so_what=?,score_grammar=?,feedback_json=?
			 WHERE id=?`,
			args[2], args[3], args[4], args[5], args[6], args[7], feedback, draftID,
		)
		if err != nil {
			errOut(err.Error())
		}
		row, _ := queryOne(
			`SELECT score_authenticity,score_structure,score_specificity,score_voice,score_so_what,score_grammar
			 FROM essay_draft WHERE id=?`, draftID,
		)
		okOut(map[string]interface{}{"draft_id": draftID, "scores": row})

	case "list":
		// essay list <student_id>
		if len(args) < 2 {
			errOut("usage: essay list <student_id>")
		}
		rows, err := queryRows(
			`SELECT id,essay_type,version,word_count,
			 score_authenticity,score_structure,score_specificity,score_voice,score_so_what,score_grammar,
			 created_at FROM essay_draft WHERE student_id=? ORDER BY essay_type,version DESC`,
			args[1],
		)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"drafts": rows, "count": len(rows)})

	case "get":
		// essay get <draft_id>
		if len(args) < 2 {
			errOut("usage: essay get <draft_id>")
		}
		row, err := queryOne("SELECT * FROM essay_draft WHERE id=?", args[1])
		if err != nil {
			errOut(err.Error())
		}
		if row == nil {
			errOut("draft not found: " + args[1])
		}
		okOut(map[string]interface{}{"draft": row})

	default:
		errOut("unknown essay command: " + args[0])
	}
}
