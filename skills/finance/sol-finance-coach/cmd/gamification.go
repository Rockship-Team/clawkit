package main

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"os"
	"strconv"
	"time"
)

// ChallengeTemplate is a predefined challenge from data/challenges.json.
type ChallengeTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Duration    int    `json:"duration_days"`
	EstSavings  string `json:"est_savings"`
	Category    string `json:"category"`
}

// ChallengeState tracks the user's active challenge progress.
type ChallengeState struct {
	ActiveID          string   `json:"active_id,omitempty"`
	StartedAt         string   `json:"started_at,omitempty"`
	Streak            int      `json:"streak"`
	Checkins          []string `json:"checkins"`
	Completed         []string `json:"completed"`
	Badges            []string `json:"badges"`
	TotalSaved        int64    `json:"total_saved"`
	InteractionStreak int      `json:"interaction_streak"`
	LastInteraction   string   `json:"last_interaction,omitempty"`
}

// QuizQuestion is a financial quiz question from data/quizzes.json.
type QuizQuestion struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	Choices     []string `json:"choices"`
	Answer      string   `json:"answer"`
	Explanation string   `json:"explanation"`
	Difficulty  string   `json:"difficulty"`
}

// QuizState tracks the user's quiz progress.
type QuizState struct {
	Answered   []QuizAnswer `json:"answered"`
	Score      int          `json:"score"`
	Streak     int          `json:"streak"`
	BestStreak int          `json:"best_streak"`
	WeeklyScore int         `json:"weekly_score"`
	WeekStart  string       `json:"week_start,omitempty"`
}

type QuizAnswer struct {
	ID      string `json:"id"`
	Correct bool   `json:"correct"`
	At      string `json:"at"`
}

func loadChallengeTemplates() []ChallengeTemplate {
	var ct []ChallengeTemplate
	readJSON(dataPath("challenges.json"), &ct)
	return ct
}

func loadChallengeState() ChallengeState {
	var cs ChallengeState
	if !readJSON(userPath("challenge_state.json"), &cs) {
		cs.Checkins = []string{}
		cs.Completed = []string{}
		cs.Badges = []string{}
	}
	return cs
}

func saveChallengeState(cs ChallengeState) error {
	return writeJSON(userPath("challenge_state.json"), cs)
}

func loadQuizQuestions() []QuizQuestion {
	var qq []QuizQuestion
	readJSON(dataPath("quizzes.json"), &qq)
	return qq
}

func loadQuizState() QuizState {
	var qs QuizState
	if !readJSON(userPath("quiz_state.json"), &qs) {
		qs.Answered = []QuizAnswer{}
	}
	return qs
}

func saveQuizState(qs QuizState) error {
	return writeJSON(userPath("quiz_state.json"), qs)
}

func cmdChallenge(args []string) {
	ensureInit()
	if len(args) == 0 {
		errOut("usage: challenge list|start|checkin|status")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		templates := loadChallengeTemplates()
		okOut(map[string]interface{}{"challenges": templates, "count": len(templates)})

	case "start":
		if len(args) < 2 {
			errOut("usage: challenge start <id>")
			os.Exit(1)
		}
		id := args[1]
		templates := loadChallengeTemplates()
		var found *ChallengeTemplate
		for i := range templates {
			if templates[i].ID == id {
				found = &templates[i]
				break
			}
		}
		if found == nil {
			errOut("challenge not found: " + id)
			os.Exit(1)
		}

		cs := loadChallengeState()
		if cs.ActiveID != "" {
			errOut("already in challenge: " + cs.ActiveID + ". Complete or abandon it first.")
			os.Exit(1)
		}

		cs.ActiveID = id
		cs.StartedAt = vnToday()
		cs.Streak = 0
		cs.Checkins = []string{}

		if err := saveChallengeState(cs); err != nil {
			errOut("failed to save: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{"started": found, "started_at": cs.StartedAt})

	case "checkin":
		cs := loadChallengeState()
		if cs.ActiveID == "" {
			errOut("no active challenge")
			os.Exit(1)
		}

		today := vnToday()
		// Check if already checked in today
		for _, d := range cs.Checkins {
			if d == today {
				errOut("already checked in today")
				os.Exit(1)
			}
		}

		cs.Checkins = append(cs.Checkins, today)
		cs.Streak = len(cs.Checkins)

		note := ""
		if len(args) > 1 {
			note = args[1]
		}

		// Check if challenge is complete
		templates := loadChallengeTemplates()
		completed := false
		for _, t := range templates {
			if t.ID == cs.ActiveID && cs.Streak >= t.Duration {
				completed = true
				cs.Completed = append(cs.Completed, cs.ActiveID)
				// Award badge for first completion
				if len(cs.Completed) == 1 {
					cs.Badges = addBadge(cs.Badges, "savings_newbie")
				}
				if len(cs.Completed) >= 5 {
					cs.Badges = addBadge(cs.Badges, "challenge_master")
				}
				cs.ActiveID = ""
				cs.StartedAt = ""
				break
			}
		}

		if err := saveChallengeState(cs); err != nil {
			errOut("failed to save: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{
			"checkin_day": cs.Streak,
			"date":        today,
			"note":        note,
			"completed":   completed,
			"badges":      cs.Badges,
		})

	case "abandon":
		cs := loadChallengeState()
		if cs.ActiveID == "" {
			errOut("no active challenge to abandon")
			os.Exit(1)
		}
		abandoned := cs.ActiveID
		cs.ActiveID = ""
		cs.StartedAt = ""
		cs.Streak = 0
		cs.Checkins = []string{}
		if err := saveChallengeState(cs); err != nil {
			errOut("failed to save: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{"abandoned": abandoned})

	case "status":
		cs := loadChallengeState()
		if cs.ActiveID == "" {
			okOut(map[string]interface{}{
				"active":    false,
				"completed": cs.Completed,
				"badges":    cs.Badges,
			})
			return
		}

		// Find template for duration info
		templates := loadChallengeTemplates()
		var tmpl *ChallengeTemplate
		for i := range templates {
			if templates[i].ID == cs.ActiveID {
				tmpl = &templates[i]
				break
			}
		}

		remaining := 0
		if tmpl != nil {
			remaining = tmpl.Duration - cs.Streak
		}

		okOut(map[string]interface{}{
			"active":       true,
			"challenge_id": cs.ActiveID,
			"challenge":    tmpl,
			"started_at":   cs.StartedAt,
			"streak":       cs.Streak,
			"remaining":    remaining,
			"checkins":     cs.Checkins,
			"completed":    cs.Completed,
			"badges":       cs.Badges,
		})

	default:
		errOut("unknown challenge command: " + args[0])
		os.Exit(1)
	}
}

func addBadge(badges []string, badge string) []string {
	for _, b := range badges {
		if b == badge {
			return badges
		}
	}
	return append(badges, badge)
}

func cmdQuiz(args []string) {
	ensureInit()
	if len(args) == 0 {
		errOut("usage: quiz random|answer|stats")
		os.Exit(1)
	}

	switch args[0] {
	case "random":
		questions := loadQuizQuestions()
		if len(questions) == 0 {
			errOut("no quiz questions found")
			os.Exit(1)
		}
		qs := loadQuizState()

		// Filter out already answered
		answered := map[string]bool{}
		for _, a := range qs.Answered {
			answered[a.ID] = true
		}
		var available []QuizQuestion
		for _, q := range questions {
			if !answered[q.ID] {
				available = append(available, q)
			}
		}
		if len(available) == 0 {
			// Reset — all answered
			available = questions
		}

		// Use seeded random for variety
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		q := available[r.Intn(len(available))]

		// Don't expose the answer in the output
		okOut(map[string]interface{}{
			"question": map[string]interface{}{
				"id":         q.ID,
				"text":       q.Text,
				"choices":    q.Choices,
				"difficulty": q.Difficulty,
			},
		})

	case "answer":
		if len(args) < 3 {
			errOut("usage: quiz answer <id> <choice>")
			os.Exit(1)
		}
		qID := args[1]
		choice := args[2]

		questions := loadQuizQuestions()
		var q *QuizQuestion
		for i := range questions {
			if questions[i].ID == qID {
				q = &questions[i]
				break
			}
		}
		if q == nil {
			errOut("question not found: " + qID)
			os.Exit(1)
		}

		correct := choice == q.Answer
		qs := loadQuizState()

		qs.Answered = append(qs.Answered, QuizAnswer{
			ID:      qID,
			Correct: correct,
			At:      vnNow().Format("2006-01-02T15:04:05-07:00"),
		})

		// Reset weekly score if new week
		today := vnToday()
		if qs.WeekStart == "" || today >= addDays(qs.WeekStart, 7) {
			qs.WeeklyScore = 0
			qs.WeekStart = today
		}

		if correct {
			qs.Score += 5
			qs.Streak++
			qs.WeeklyScore += 5
			if qs.Streak > qs.BestStreak {
				qs.BestStreak = qs.Streak
			}
		} else {
			qs.Streak = 0
		}

		// Check for badge
		cs := loadChallengeState()
		correctCount := 0
		for _, a := range qs.Answered {
			if a.Correct {
				correctCount++
			}
		}
		if correctCount >= 20 {
			cs.Badges = addBadge(cs.Badges, "finance_101")
			saveChallengeState(cs)
		}

		if err := saveQuizState(qs); err != nil {
			errOut("failed to save: " + err.Error())
			os.Exit(1)
		}

		okOut(map[string]interface{}{
			"correct":     correct,
			"answer":      q.Answer,
			"explanation": q.Explanation,
			"score":       qs.Score,
			"streak":      qs.Streak,
		})

	case "stats":
		qs := loadQuizState()
		total := len(qs.Answered)
		correct := 0
		for _, a := range qs.Answered {
			if a.Correct {
				correct++
			}
		}
		pct := 0
		if total > 0 {
			pct = correct * 100 / total
		}
		okOut(map[string]interface{}{
			"total_answered": total,
			"correct":        correct,
			"accuracy_pct":   pct,
			"score":          qs.Score,
			"streak":         qs.Streak,
			"best_streak":    qs.BestStreak,
			"weekly_score":   qs.WeeklyScore,
		})

	default:
		errOut("unknown quiz command: " + args[0])
		os.Exit(1)
	}
}

// deterministicIndex returns a stable index for a given date string.
func deterministicIndex(date string, max int) int {
	h := sha256.Sum256([]byte(date))
	return int(binary.BigEndian.Uint32(h[:4])) % max
}

// atoi is a helper that returns 0 on error.
func atoi(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
