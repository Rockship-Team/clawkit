package main

import (
	"encoding/json"
	"fmt"
	"math"
)

func cmdStudent(args []string) {
	if len(args) == 0 {
		errOut("usage: student query|save|list|update|scorecard")
	}
	switch args[0] {
	case "query":
		// student query <channel> <channel_user_id>
		if len(args) < 3 {
			errOut("usage: student query <channel> <channel_user_id>")
		}
		row, err := queryOne(
			"SELECT * FROM student_profile WHERE channel=? AND channel_user_id=?",
			args[1], args[2],
		)
		if err != nil {
			errOut(err.Error())
		}
		if row == nil {
			okOut(map[string]interface{}{"found": false, "student": nil})
			return
		}
		okOut(map[string]interface{}{"found": true, "student": row})

	case "save":
		// student save <json>
		// JSON keys: channel, user_id, name, grade, school, curriculum, gpa, gpa_scale,
		//            sat, act, toefl, ielts, ap_scores, major, countries, budget,
		//            needs_aid (0|1), consent_student (0|1), consent_guardian (0|1)
		if len(args) < 2 {
			errOut(`usage: student save '{"channel":"telegram","user_id":"123","name":"...","grade":11,"school":"...","curriculum":"VN","gpa":8.5,"gpa_scale":10,"sat":1350,"toefl":95,"ielts":null,"ap_scores":"[]","major":"CS","countries":"[\"US\"]","budget":50000,"needs_aid":0,"consent_student":1,"consent_guardian":0}'`)
		}
		var p map[string]interface{}
		if err := json.Unmarshal([]byte(args[1]), &p); err != nil {
			errOut("invalid JSON: " + err.Error())
		}
		channel, _ := p["channel"].(string)
		userID, _ := p["user_id"].(string)
		if channel == "" || userID == "" {
			errOut("JSON must include channel and user_id")
		}

		// Check existing
		existing, _ := queryOne(
			"SELECT id FROM student_profile WHERE channel=? AND channel_user_id=?",
			channel, userID,
		)
		now := vnNowISO()
		var studentID string

		nullableStr := func(key string) interface{} {
			v, ok := p[key]
			if !ok || v == nil {
				return nil
			}
			s := fmt.Sprintf("%v", v)
			if s == "<nil>" || s == "" {
				return nil
			}
			return s
		}
		nullableFloat := func(key string) interface{} {
			v, ok := p[key]
			if !ok || v == nil {
				return nil
			}
			switch vt := v.(type) {
			case float64:
				if vt == 0 {
					return nil
				}
				return vt
			}
			return nil
		}
		intVal := func(key string) int {
			v, ok := p[key]
			if !ok || v == nil {
				return 0
			}
			if f, ok := v.(float64); ok {
				return int(f)
			}
			return 0
		}
		strVal := func(key, def string) string {
			v, ok := p[key]
			if !ok || v == nil {
				return def
			}
			return fmt.Sprintf("%v", v)
		}

		if existing != nil {
			studentID = fmt.Sprintf("%v", existing["id"])
			_, err := exec(`UPDATE student_profile SET
				display_name=?, grade_level=?, school_name=?, curriculum_type=?,
				gpa_value=?, gpa_scale=?,
				sat_score=?, act_score=?, toefl_score=?, ielts_score=?,
				ap_scores=?, intended_major=?, target_countries=?,
				annual_budget_usd=?, needs_financial_aid=?, dream_schools=?,
				consent_student=?, consent_guardian=?,
				onboarding_completed_at=COALESCE(onboarding_completed_at, ?),
				updated_at=?
				WHERE id=?`,
				strVal("name", ""), intVal("grade"), strVal("school", ""), strVal("curriculum", "VN"),
				nullableFloat("gpa"), nullableFloat("gpa_scale"),
				nullableFloat("sat"), nullableFloat("act"), nullableFloat("toefl"), nullableFloat("ielts"),
				strVal("ap_scores", "[]"), nullableStr("major"), strVal("countries", "[]"),
				nullableFloat("budget"), intVal("needs_aid"), nullableStr("dreams"),
				intVal("consent_student"), intVal("consent_guardian"),
				now, now,
				studentID,
			)
			if err != nil {
				errOut("update failed: " + err.Error())
			}
		} else {
			studentID = newID()
			_, err := exec(`INSERT INTO student_profile
				(id,channel,channel_user_id,display_name,grade_level,school_name,curriculum_type,
				 gpa_value,gpa_scale,sat_score,act_score,toefl_score,ielts_score,
				 ap_scores,intended_major,target_countries,annual_budget_usd,
				 needs_financial_aid,dream_schools,consent_student,consent_guardian,
				 onboarding_completed_at,created_at,updated_at)
				VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				studentID, channel, userID,
				strVal("name", ""), intVal("grade"), strVal("school", ""), strVal("curriculum", "VN"),
				nullableFloat("gpa"), nullableFloat("gpa_scale"),
				nullableFloat("sat"), nullableFloat("act"), nullableFloat("toefl"), nullableFloat("ielts"),
				strVal("ap_scores", "[]"), nullableStr("major"), strVal("countries", "[]"),
				nullableFloat("budget"), intVal("needs_aid"), nullableStr("dreams"),
				intVal("consent_student"), intVal("consent_guardian"),
				now, now, now,
			)
			if err != nil {
				errOut("insert failed: " + err.Error())
			}
		}
		okOut(map[string]interface{}{"student_id": studentID, "channel": channel, "user_id": userID})

	case "list":
		rows, err := queryRows(
			"SELECT id,channel,channel_user_id,display_name,grade_level,school_name,gpa_value,sat_score,toefl_score,intended_major,onboarding_completed_at FROM student_profile ORDER BY created_at DESC LIMIT 100",
		)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"students": rows, "count": len(rows)})

	case "update":
		// student update <student_id> <field> <value>
		if len(args) < 4 {
			errOut("usage: student update <student_id> <field> <value>")
		}
		id, field, val := args[1], args[2], args[3]
		allowed := map[string]bool{
			"display_name": true, "grade_level": true, "school_name": true,
			"curriculum_type": true, "gpa_value": true, "gpa_scale": true,
			"sat_score": true, "act_score": true, "toefl_score": true, "ielts_score": true,
			"ap_scores": true, "intended_major": true, "target_countries": true,
			"annual_budget_usd": true, "needs_financial_aid": true,
			"dream_schools": true, "onboarding_completed_at": true,
		}
		if !allowed[field] {
			errOut("cannot update field: " + field)
		}
		q := fmt.Sprintf("UPDATE student_profile SET %s=?, updated_at=? WHERE id=?", field)
		_, err := exec(q, val, vnNowISO(), id)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"updated": id, "field": field})

	case "scorecard":
		// student scorecard <student_id>
		if len(args) < 2 {
			errOut("usage: student scorecard <student_id>")
		}
		profile, err := queryOne("SELECT * FROM student_profile WHERE id=?", args[1])
		if err != nil || profile == nil {
			errOut("student not found: " + args[1])
		}
		activities, _ := queryRows(
			"SELECT tier FROM extracurricular_activity WHERE student_id=?", args[1],
		)

		curriculum := strFromRow(profile, "curriculum_type", "VN")
		gpa := floatFromRow(profile, "gpa_value")
		gpaScale := floatFromRow(profile, "gpa_scale")
		sat := intFromRow(profile, "sat_score")
		act := intFromRow(profile, "act_score")
		toefl := intFromRow(profile, "toefl_score")
		ielts := floatFromRow(profile, "ielts_score")
		grade := intFromRow(profile, "grade_level")
		major := strFromRow(profile, "intended_major", "")

		academics := scoreAcademics(gpa, gpaScale, curriculum)
		testScores := scoreTests(sat, act, toefl, ielts)
		ec := scoreEC(activities)
		essay := scoreEssayReadiness(grade, activities)
		overall := int(math.Round(float64(academics)*0.35 + float64(testScores)*0.30 + float64(ec)*0.25 + float64(essay)*0.10))

		nextActions := buildNextActions(academics, testScores, ec, essay, grade, sat, toefl, ielts, major)

		okOut(map[string]interface{}{
			"student_id": args[1],
			"scores": map[string]interface{}{
				"academics":        academics,
				"test_scores":      testScores,
				"extracurriculars": ec,
				"essay_readiness":  essay,
				"overall":          overall,
			},
			"commentary": map[string]interface{}{
				"academics":        commentaryAcademics(academics, gpa, gpaScale, curriculum),
				"test_scores":      commentaryTests(testScores, sat, act, toefl, ielts),
				"extracurriculars": commentaryEC(ec, activities),
			},
			"next_actions":   nextActions,
			"activity_count": len(activities),
		})

	default:
		errOut("unknown student command: " + args[0])
	}
}

// ── Scorecard helpers ────────────────────────────────────────────────────────

func scoreAcademics(gpa, gpaScale float64, curriculum string) int {
	if gpa == 0 || gpaScale == 0 {
		return 0
	}
	pct := gpa / gpaScale
	if curriculum == "IB" || curriculum == "AP" || curriculum == "A-Level" {
		pct = math.Min(pct*1.05, 1.0)
	}
	return int(math.Round(pct * 100))
}

func scoreTests(sat, act, toefl int, ielts float64) int {
	var scores []float64
	if sat > 0 {
		scores = append(scores, math.Min(float64(sat-400)/float64(1600-400), 1.0))
	}
	if act > 0 {
		scores = append(scores, math.Min(float64(act-1)/float64(36-1), 1.0))
	}
	if toefl > 0 {
		scores = append(scores, math.Min(float64(toefl)/120.0, 1.0))
	}
	if ielts > 0 {
		scores = append(scores, math.Min(ielts/9.0, 1.0))
	}
	if len(scores) == 0 {
		return 0
	}
	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	return int(math.Round(sum / float64(len(scores)) * 100))
}

func scoreEC(activities []map[string]interface{}) int {
	if len(activities) == 0 {
		return 0
	}
	tierWeights := map[int]int{1: 100, 2: 75, 3: 50, 4: 25}
	total := 0
	for _, a := range activities {
		tier := intFromRow(a, "tier")
		w, ok := tierWeights[tier]
		if !ok {
			w = 30
		}
		total += w
	}
	result := int(math.Round(float64(total) / float64(5*100) * 100))
	if result > 100 {
		return 100
	}
	return result
}

func scoreEssayReadiness(grade int, activities []map[string]interface{}) int {
	base := 10
	if grade >= 11 {
		base = 30
	}
	if len(activities) > 0 {
		base += 20
	}
	if base > 60 {
		return 60
	}
	return base
}

func commentaryAcademics(score int, gpa, gpaScale float64, curriculum string) string {
	if gpa == 0 {
		return "Chưa có GPA — hãy cập nhật để mình đánh giá chính xác hơn."
	}
	note := ""
	if curriculum != "VN" {
		note = fmt.Sprintf(" (chương trình %s)", curriculum)
	}
	gpaStr := fmt.Sprintf("%.1f/%.0f", gpa, gpaScale)
	switch {
	case score >= 85:
		return fmt.Sprintf("GPA %s%s rất mạnh — nằm trong top applicants tại hầu hết các trường Target và nhiều trường Reach.", gpaStr, note)
	case score >= 70:
		return fmt.Sprintf("GPA %s%s tốt — đủ competitive cho các trường Target. Với trường Reach, cần EC và essay mạnh hơn.", gpaStr, note)
	case score >= 55:
		return fmt.Sprintf("GPA %s%s ở mức trung bình — nên tập trung cải thiện GPA và xây dựng EC mạnh hơn.", gpaStr, note)
	default:
		return fmt.Sprintf("GPA %s%s cần cải thiện đáng kể. Ưu tiên ổn định học thuật trước khi focus vào SAT hay EC.", gpaStr, note)
	}
}

func commentaryTests(score, sat, act, toefl int, ielts float64) string {
	var parts []string
	if sat > 0 {
		parts = append(parts, fmt.Sprintf("SAT %d", sat))
	}
	if act > 0 {
		parts = append(parts, fmt.Sprintf("ACT %d", act))
	}
	if toefl > 0 {
		parts = append(parts, fmt.Sprintf("TOEFL %d", toefl))
	}
	if ielts > 0 {
		parts = append(parts, fmt.Sprintf("IELTS %.1f", ielts))
	}
	if len(parts) == 0 {
		return "Chưa có điểm thi nào — đây là ưu tiên #1. SAT/TOEFL ảnh hưởng trực tiếp đến competitive profile."
	}
	scoreStr := ""
	for i, p := range parts {
		if i > 0 {
			scoreStr += " + "
		}
		scoreStr += p
	}
	switch {
	case score >= 80:
		return scoreStr + " — điểm thi rất tốt, hỗ trợ mạnh cho hồ sơ."
	case score >= 60:
		return scoreStr + " — điểm thi ổn, nhưng vẫn còn room to improve."
	case score >= 40:
		return scoreStr + " — điểm thi cần cải thiện. Nên lên lộ trình ôn tập ngay."
	default:
		return scoreStr + " — điểm thi thấp hơn median của hầu hết trường Target. Đây là gap lớn nhất cần giải quyết."
	}
}

func commentaryEC(score int, activities []map[string]interface{}) string {
	count := len(activities)
	tier1, tier2 := 0, 0
	for _, a := range activities {
		switch intFromRow(a, "tier") {
		case 1:
			tier1++
		case 2:
			tier2++
		}
	}
	switch {
	case count == 0:
		return "Chưa có hoạt động ngoại khoá nào. EC là yếu tố quan trọng — hãy kể cho mình nghe các hoạt động của em."
	case tier1 >= 1:
		return fmt.Sprintf("%d hoạt động (%d Tier 1) — EC profile rất mạnh, ở top 10–15%% applicants.", count, tier1)
	case tier2 >= 2:
		return fmt.Sprintf("%d hoạt động (%d Tier 2) — EC tốt. Nâng 1 hoạt động lên Tier 1 sẽ tăng đáng kể competitive profile.", count, tier2)
	case score >= 50:
		return fmt.Sprintf("%d hoạt động ở Tier 2–3 — EC ổn nhưng thiếu điểm nhấn. Cần ít nhất 1 achievement cấp quốc gia.", count)
	default:
		return fmt.Sprintf("%d hoạt động chủ yếu Tier 3–4 — EC cần được nâng cấp. Mình có thể giúp em lên chiến lược cụ thể.", count)
	}
}

func buildNextActions(academics, testScores, ec, essay, grade, sat, toefl int, ielts float64, major string) []map[string]interface{} {
	type candidate struct {
		priority int
		action   map[string]interface{}
	}
	var cands []candidate

	if sat == 0 && toefl == 0 && ielts == 0 {
		cands = append(cands, candidate{10, map[string]interface{}{"action": "Đăng ký thi SAT và TOEFL/IELTS ngay", "timeline": "Trong vòng 2 tuần"}})
	} else if sat == 0 {
		cands = append(cands, candidate{9, map[string]interface{}{"action": "Đăng ký thi SAT — mình có thể tạo study plan ngay", "timeline": "Thi trước tháng 10 để kịp EA/ED"}})
	} else if testScores < 60 {
		cands = append(cands, candidate{8, map[string]interface{}{"action": fmt.Sprintf("Cải thiện SAT (hiện tại %d) — mục tiêu tăng thêm 100+ điểm", sat), "timeline": "Lên lộ trình 12–16 tuần"}})
	}
	if toefl == 0 && ielts == 0 {
		cands = append(cands, candidate{7, map[string]interface{}{"action": "Đăng ký thi TOEFL hoặc IELTS", "timeline": "Nên có điểm trước tháng 11"}})
	}
	if ec < 40 {
		cands = append(cands, candidate{6, map[string]interface{}{"action": "Bắt đầu ít nhất 1 hoạt động ngoại khoá có impact đo được", "timeline": "Hè này là thời điểm tốt nhất"}})
	} else if ec < 70 {
		cands = append(cands, candidate{5, map[string]interface{}{"action": "Nâng cấp EC hiện tại — chuyển từ thành viên sang leader hoặc founder", "timeline": "Trong học kỳ tới"}})
	}
	if grade >= 11 && essay < 40 {
		cands = append(cands, candidate{4, map[string]interface{}{"action": "Bắt đầu brainstorm Common App Personal Statement", "timeline": "Lý tưởng nhất là bắt đầu từ bây giờ"}})
	}
	if academics < 55 {
		cands = append(cands, candidate{9, map[string]interface{}{"action": "Ổn định GPA — cải thiện môn đang bị giảm điểm", "timeline": "Học kỳ này"}})
	}
	if major == "" {
		cands = append(cands, candidate{3, map[string]interface{}{"action": "Xác định ngành học mục tiêu để định hướng EC và essay", "timeline": "Càng sớm càng tốt"}})
	}

	// Sort by priority desc, take top 3
	for i := 0; i < len(cands)-1; i++ {
		for j := i + 1; j < len(cands); j++ {
			if cands[j].priority > cands[i].priority {
				cands[i], cands[j] = cands[j], cands[i]
			}
		}
	}
	result := []map[string]interface{}{}
	for i, c := range cands {
		if i >= 3 {
			break
		}
		result = append(result, c.action)
	}
	return result
}

// ── Row helpers ──────────────────────────────────────────────────────────────

func floatFromRow(row map[string]interface{}, key string) float64 {
	v, ok := row[key]
	if !ok || v == nil {
		return 0
	}
	switch vt := v.(type) {
	case float64:
		return vt
	case int64:
		return float64(vt)
	}
	return 0
}

func intFromRow(row map[string]interface{}, key string) int {
	v, ok := row[key]
	if !ok || v == nil {
		return 0
	}
	switch vt := v.(type) {
	case int64:
		return int(vt)
	case float64:
		return int(vt)
	}
	return 0
}

func strFromRow(row map[string]interface{}, key, def string) string {
	v, ok := row[key]
	if !ok || v == nil {
		return def
	}
	s := fmt.Sprintf("%v", v)
	if s == "" {
		return def
	}
	return s
}
