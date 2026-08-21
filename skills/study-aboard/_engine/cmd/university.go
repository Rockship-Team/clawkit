package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func cmdUniversity(args []string) {
	if len(args) == 0 {
		errOut("usage: university list|get|search|seed|match")
	}
	switch args[0] {
	case "list":
		// university list [country]
		q := `SELECT id,name,location_city,location_state,location_country,type,usnews_ranking,
		      acceptance_rate_overall,sat_25,sat_75,toefl_minimum,financial_aid_international,
		      ed_available,ea_available,avg_annual_cost_usd
		      FROM university_record`
		qargs := []interface{}{}
		if len(args) > 1 {
			q += " WHERE location_country=?"
			qargs = append(qargs, args[1])
		}
		q += " ORDER BY usnews_ranking ASC NULLS LAST LIMIT 200"
		rows, err := queryRows(q, qargs...)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"universities": rows, "count": len(rows)})

	case "get":
		// university get <id_or_name>
		if len(args) < 2 {
			errOut("usage: university get <id>")
		}
		row, err := queryOne("SELECT * FROM university_record WHERE id=?", args[1])
		if err != nil {
			errOut(err.Error())
		}
		if row == nil {
			// Try by name
			row, err = queryOne("SELECT * FROM university_record WHERE name LIKE ?", "%"+args[1]+"%")
			if err != nil {
				errOut(err.Error())
			}
		}
		if row == nil {
			errOut("university not found: " + args[1])
		}
		okOut(map[string]interface{}{"university": row})

	case "search":
		// university search <query> [country]
		if len(args) < 2 {
			errOut("usage: university search <query> [country]")
		}
		q := `SELECT id,name,location_city,location_country,type,usnews_ranking,
		      acceptance_rate_overall,sat_25,sat_75,toefl_minimum,financial_aid_international,avg_annual_cost_usd
		      FROM university_record WHERE name LIKE ?`
		qargs := []interface{}{"%" + args[1] + "%"}
		if len(args) > 2 {
			q += " AND location_country=?"
			qargs = append(qargs, args[2])
		}
		q += " ORDER BY usnews_ranking ASC NULLS LAST LIMIT 20"
		rows, err := queryRows(q, qargs...)
		if err != nil {
			errOut(err.Error())
		}
		okOut(map[string]interface{}{"results": rows, "count": len(rows)})

	case "seed":
		// university seed <json_file>
		if len(args) < 2 {
			errOut("usage: university seed <json_file>")
		}
		data, err := os.ReadFile(args[1])
		if err != nil {
			errOut("cannot read file: " + err.Error())
		}
		var records []map[string]interface{}
		if err = json.Unmarshal(data, &records); err != nil {
			errOut("invalid JSON: " + err.Error())
		}
		inserted, updated := 0, 0
		for _, r := range records {
			id, _ := r["id"].(string)
			if id == "" {
				id = newID()
			}
			name, _ := r["name"].(string)
			city, _ := r["location_city"].(string)
			country, _ := r["location_country"].(string)
			if country == "" {
				country = "US"
			}
			utype, _ := r["type"].(string)

			existing, _ := queryOne("SELECT id FROM university_record WHERE id=?", id)
			if existing != nil {
				exec(
					`UPDATE university_record SET name=?,location_city=?,location_country=?,type=?,updated_at=? WHERE id=?`,
					name, city, country, utype, vnNowISO(), id,
				)
				updated++
			} else {
				now := vnNowISO()
				exec(
					`INSERT INTO university_record
					(id,name,location_city,location_state,location_country,type,usnews_ranking,
					 acceptance_rate_overall,acceptance_rate_international,sat_25,sat_75,act_25,act_75,gpa_avg_admitted,
					 test_policy,toefl_minimum,ielts_minimum,strong_programs,application_platform,
					 ed_available,ea_available,ed_deadline,ea_deadline,rd_deadline,
					 financial_aid_international,avg_annual_cost_usd,essay_prompts,
					 admits_by_major,cycle_year,created_at,updated_at)
					VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
					id, name, city, strOrNil(r, "location_state"), country, utype,
					intOrNil(r, "usnews_ranking"), floatOrNil(r, "acceptance_rate_overall"),
					floatOrNil(r, "acceptance_rate_international"),
					intOrNil(r, "sat_25"), intOrNil(r, "sat_75"),
					intOrNil(r, "act_25"), intOrNil(r, "act_75"),
					floatOrNil(r, "gpa_avg_admitted"),
					strDefault(r, "test_policy", "optional"),
					intOrNil(r, "toefl_minimum"), floatOrNil(r, "ielts_minimum"),
					strOrNil(r, "strong_programs"),
					strDefault(r, "application_platform", "Common App"),
					boolInt(r, "ed_available"), boolInt(r, "ea_available"),
					strOrNil(r, "ed_deadline"), strOrNil(r, "ea_deadline"), strOrNil(r, "rd_deadline"),
					strDefault(r, "financial_aid_international", "none"),
					intOrNil(r, "avg_annual_cost_usd"), strOrNil(r, "essay_prompts"),
					boolInt(r, "admits_by_major"),
					intDefault(r, "cycle_year", 2026),
					now, now,
				)
				inserted++
			}
		}
		okOut(map[string]interface{}{"inserted": inserted, "updated": updated, "total": len(records)})

	case "match":
		// university match <student_id> [limit]
		if len(args) < 2 {
			errOut("usage: university match <student_id> [limit]")
		}
		limit := 12
		if len(args) > 2 {
			if n, err := strconv.Atoi(args[2]); err == nil {
				limit = n
			}
		}

		profile, err := queryOne("SELECT * FROM student_profile WHERE id=?", args[1])
		if err != nil || profile == nil {
			errOut("student not found: " + args[1])
		}
		activities, _ := queryRows("SELECT tier FROM extracurricular_activity WHERE student_id=?", args[1])

		// Determine country filter from student's target_countries
		filterCountries := []string{}
		if raw, ok := profile["target_countries"].(string); ok && raw != "" && raw != "[]" {
			var countries []string
			if json.Unmarshal([]byte(raw), &countries) == nil {
				for _, c := range countries {
					filterCountries = append(filterCountries, strings.ToUpper(c))
				}
			}
		}

		q := "SELECT * FROM university_record"
		qargs := []interface{}{}
		if len(filterCountries) > 0 {
			placeholders := make([]string, len(filterCountries))
			for i, c := range filterCountries {
				placeholders[i] = "?"
				qargs = append(qargs, c)
			}
			q += " WHERE location_country IN (" + strings.Join(placeholders, ",") + ")"
		}
		unis, _ := queryRows(q, qargs...)

		type result struct {
			uni      map[string]interface{}
			fitScore float64
			category string
		}
		var results []result
		for _, uni := range unis {
			fs := uniMatchScore(uni, profile, activities)
			ar, _ := uni["acceptance_rate_international"].(float64)
			if ar == 0 {
				ar, _ = uni["acceptance_rate_overall"].(float64)
			}
			cat := categorizeUniversity(fs, ar)
			results = append(results, result{uni, fs, cat})
		}

		// Sort by fit score desc
		for i := 0; i < len(results)-1; i++ {
			for j := i + 1; j < len(results); j++ {
				if results[j].fitScore > results[i].fitScore {
					results[i], results[j] = results[j], results[i]
				}
			}
		}

		// Balance Reach/Target/Safety
		reach, target, safety := []map[string]interface{}{}, []map[string]interface{}{}, []map[string]interface{}{}
		for _, r := range results {
			entry := map[string]interface{}{
				"id":                         r.uni["id"],
				"name":                       r.uni["name"],
				"location":                   fmt.Sprintf("%v, %v", r.uni["location_city"], r.uni["location_country"]),
				"category":                   r.category,
				"fit_score":                  r.fitScore,
				"acceptance_rate_intl":        r.uni["acceptance_rate_international"],
				"avg_annual_cost_usd":         r.uni["avg_annual_cost_usd"],
				"financial_aid_international": r.uni["financial_aid_international"],
				"strong_programs":             r.uni["strong_programs"],
				"ed_available":               r.uni["ed_available"],
				"ea_available":               r.uni["ea_available"],
				"rd_deadline":                r.uni["rd_deadline"],
			}
			switch r.category {
			case "reach":
				if len(reach) < 3 {
					reach = append(reach, entry)
				}
			case "target":
				if len(target) < 6 {
					target = append(target, entry)
				}
			case "safety":
				if len(safety) < 3 {
					safety = append(safety, entry)
				}
			}
		}
		final := append(append(reach, target...), safety...)
		if len(final) > limit {
			final = final[:limit]
		}

		okOut(map[string]interface{}{
			"student_id":                   args[1],
			"total_universities_evaluated": len(unis),
			"school_list":                  final,
			"summary": map[string]interface{}{
				"reach":  len(reach),
				"target": len(target),
				"safety": len(safety),
			},
		})

	default:
		errOut("unknown university command: " + args[0])
	}
}

// --- JSON seed helpers ---

func strOrNil(r map[string]interface{}, key string) interface{} {
	v, ok := r[key]
	if !ok || v == nil || fmt.Sprintf("%v", v) == "" {
		return nil
	}
	return fmt.Sprintf("%v", v)
}

func strDefault(r map[string]interface{}, key, def string) string {
	v, ok := r[key]
	if !ok || v == nil {
		return def
	}
	s := fmt.Sprintf("%v", v)
	if s == "" {
		return def
	}
	return s
}

func intOrNil(r map[string]interface{}, key string) interface{} {
	v, ok := r[key]
	if !ok || v == nil {
		return nil
	}
	switch vt := v.(type) {
	case float64:
		return int(vt)
	case int:
		return vt
	case string:
		i, err := strconv.Atoi(vt)
		if err != nil {
			return nil
		}
		return i
	}
	return nil
}

func intDefault(r map[string]interface{}, key string, def int) int {
	v := intOrNil(r, key)
	if v == nil {
		return def
	}
	return v.(int)
}

func floatOrNil(r map[string]interface{}, key string) interface{} {
	v, ok := r[key]
	if !ok || v == nil {
		return nil
	}
	switch vt := v.(type) {
	case float64:
		return vt
	case string:
		f, err := strconv.ParseFloat(vt, 64)
		if err != nil {
			return nil
		}
		return f
	}
	return nil
}

// ── University matching helpers ──────────────────────────────────────────────

func uniMatchScore(uni map[string]interface{}, profile map[string]interface{}, activities []map[string]interface{}) float64 {
	score := 50.0

	// SAT match
	sat := floatFromRow(profile, "sat_score")
	sat25, _ := uni["sat_25"].(int64)
	sat75, _ := uni["sat_75"].(int64)
	if sat > 0 && sat25 > 0 && sat75 > 0 {
		mid := float64(sat25+sat75) / 2
		diff := absF(sat-mid) / float64(sat75-sat25+1)
		adj := 20.0 - diff*20.0
		if adj < -20 {
			adj = -20
		}
		if adj > 20 {
			adj = 20
		}
		score += adj
	}

	// GPA match
	gpa := floatFromRow(profile, "gpa_value")
	gpaScale := floatFromRow(profile, "gpa_scale")
	gpaAdmitted := floatFromRow(uni, "gpa_avg_admitted")
	if gpa > 0 && gpaScale > 0 && gpaAdmitted > 0 {
		gpa4 := gpa / gpaScale * 4.0
		adj := (gpa4 - gpaAdmitted) * 10.0
		if adj < -15 {
			adj = -15
		}
		if adj > 15 {
			adj = 15
		}
		score += adj
	}

	// Budget match
	budget := floatFromRow(profile, "annual_budget_usd")
	cost := floatFromRow(uni, "avg_annual_cost_usd")
	if budget > 0 && cost > 0 {
		if cost <= budget {
			score += 10
		} else {
			over := (cost - budget) / 5000
			if over > 20 {
				over = 20
			}
			score -= over
		}
	}

	// Major match
	major := strings.ToLower(strFromRow(profile, "intended_major", ""))
	programs := strFromRow(uni, "strong_programs", "")
	if major != "" && programs != "" {
		if strings.Contains(strings.ToLower(programs), major) {
			score += 10
		}
	}

	// Country preference
	targetCountries := strFromRow(profile, "target_countries", "[]")
	uniCountry := strFromRow(uni, "location_country", "")
	if targetCountries != "[]" && targetCountries != "" {
		var countries []string
		if json.Unmarshal([]byte(targetCountries), &countries) == nil {
			for _, c := range countries {
				if strings.EqualFold(c, uniCountry) {
					score += 5
					break
				}
			}
		}
	} else {
		score += 5 // no preference = small bonus for all
	}

	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func categorizeUniversity(score float64, acceptanceRate float64) string {
	ar := acceptanceRate
	if ar == 0 {
		ar = 0.5
	}
	if score < 45 || ar < 0.15 {
		return "reach"
	}
	if score >= 65 && ar >= 0.35 {
		return "safety"
	}
	return "target"
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func boolInt(r map[string]interface{}, key string) int {
	v, ok := r[key]
	if !ok || v == nil {
		return 0
	}
	switch vt := v.(type) {
	case bool:
		if vt {
			return 1
		}
	case float64:
		if vt != 0 {
			return 1
		}
	case int:
		if vt != 0 {
			return 1
		}
	}
	return 0
}
