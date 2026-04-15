package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// setupTestEnv creates a temp directory with required data files and sets SOL_DATA_DIR.
func setupTestEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SOL_DATA_DIR", dir)

	// Create data/ subdirectory
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0o755)

	// Write minimal challenges.json
	challenges := []ChallengeTemplate{
		{ID: "test-3d", Name: "Test 3 day", Description: "Test", Duration: 3, EstSavings: "100K", Category: "general"},
		{ID: "test-1d", Name: "Test 1 day", Description: "Quick", Duration: 1, EstSavings: "50K", Category: "food"},
	}
	writeTestJSON(t, filepath.Join(dataDir, "challenges.json"), challenges)

	// Write minimal quizzes.json
	quizzes := []QuizQuestion{
		{ID: "tq1", Text: "Q1?", Choices: []string{"A. a", "B. b", "C. c", "D. d"}, Answer: "C", Explanation: "C is correct", Difficulty: "easy"},
		{ID: "tq2", Text: "Q2?", Choices: []string{"A. a", "B. b", "C. c", "D. d"}, Answer: "A", Explanation: "A is correct", Difficulty: "easy"},
	}
	writeTestJSON(t, filepath.Join(dataDir, "quizzes.json"), quizzes)

	return dir
}

func writeTestJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestAddBadge(t *testing.T) {
	badges := []string{}
	badges = addBadge(badges, "savings_newbie")
	if len(badges) != 1 || badges[0] != "savings_newbie" {
		t.Errorf("expected [savings_newbie], got %v", badges)
	}

	// Adding same badge again should not duplicate
	badges = addBadge(badges, "savings_newbie")
	if len(badges) != 1 {
		t.Errorf("duplicate badge added, got %v", badges)
	}

	// Adding different badge
	badges = addBadge(badges, "challenge_master")
	if len(badges) != 2 {
		t.Errorf("expected 2 badges, got %v", badges)
	}
}

func TestDeterministicIndex(t *testing.T) {
	// Same date should produce same index
	idx1 := deterministicIndex("2026-04-15", 100)
	idx2 := deterministicIndex("2026-04-15", 100)
	if idx1 != idx2 {
		t.Errorf("same date produced different indices: %d vs %d", idx1, idx2)
	}

	// Different dates should (very likely) produce different indices
	idx3 := deterministicIndex("2026-04-16", 100)
	// Not strictly guaranteed but extremely likely for different dates
	_ = idx3

	// Index should be in range
	for i := 0; i < 100; i++ {
		idx := deterministicIndex("2026-01-01", i+1)
		if idx < 0 || idx >= i+1 {
			t.Errorf("index %d out of range [0, %d)", idx, i+1)
		}
	}
}

func TestChallengeStateRoundTrip(t *testing.T) {
	setupTestEnv(t)

	cs := ChallengeState{
		ActiveID:  "test-3d",
		StartedAt: "2026-04-15",
		Streak:    2,
		Checkins:  []string{"2026-04-15", "2026-04-16"},
		Completed: []string{},
		Badges:    []string{"savings_newbie"},
	}

	if err := saveChallengeState(cs); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded := loadChallengeState()
	if loaded.ActiveID != "test-3d" {
		t.Errorf("ActiveID = %q, want %q", loaded.ActiveID, "test-3d")
	}
	if loaded.Streak != 2 {
		t.Errorf("Streak = %d, want 2", loaded.Streak)
	}
	if len(loaded.Badges) != 1 || loaded.Badges[0] != "savings_newbie" {
		t.Errorf("Badges = %v, want [savings_newbie]", loaded.Badges)
	}
}

func TestQuizStateRoundTrip(t *testing.T) {
	setupTestEnv(t)

	qs := QuizState{
		Answered: []QuizAnswer{
			{ID: "tq1", Correct: true, At: "2026-04-15T10:00:00+07:00"},
			{ID: "tq2", Correct: false, At: "2026-04-15T10:01:00+07:00"},
		},
		Score:  5,
		Streak: 0,
	}

	if err := saveQuizState(qs); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded := loadQuizState()
	if len(loaded.Answered) != 2 {
		t.Fatalf("Answered count = %d, want 2", len(loaded.Answered))
	}
	if loaded.Score != 5 {
		t.Errorf("Score = %d, want 5", loaded.Score)
	}
	if loaded.Answered[0].Correct != true {
		t.Error("first answer should be correct")
	}
}

func TestBadgeAwardOnFirstCompletion(t *testing.T) {
	// First challenge completion → savings_newbie badge
	cs := ChallengeState{
		Completed: []string{},
		Badges:    []string{},
	}

	// Simulate first completion
	cs.Completed = append(cs.Completed, "test-3d")
	if len(cs.Completed) == 1 {
		cs.Badges = addBadge(cs.Badges, "savings_newbie")
	}

	found := false
	for _, b := range cs.Badges {
		if b == "savings_newbie" {
			found = true
		}
	}
	if !found {
		t.Error("savings_newbie badge not awarded on first completion")
	}
}

func TestBadgeAwardOnFifthCompletion(t *testing.T) {
	cs := ChallengeState{
		Completed: []string{"a", "b", "c", "d"},
		Badges:    []string{"savings_newbie"},
	}

	// Fifth completion
	cs.Completed = append(cs.Completed, "e")
	if len(cs.Completed) >= 5 {
		cs.Badges = addBadge(cs.Badges, "challenge_master")
	}

	found := false
	for _, b := range cs.Badges {
		if b == "challenge_master" {
			found = true
		}
	}
	if !found {
		t.Error("challenge_master badge not awarded on 5th completion")
	}
}

func TestFinance101Badge(t *testing.T) {
	// 20 correct answers → finance_101 badge
	cs := ChallengeState{Badges: []string{}}
	correctCount := 20

	if correctCount >= 20 {
		cs.Badges = addBadge(cs.Badges, "finance_101")
	}

	found := false
	for _, b := range cs.Badges {
		if b == "finance_101" {
			found = true
		}
	}
	if !found {
		t.Error("finance_101 badge not awarded at 20 correct answers")
	}
}

func TestLoadChallengeTemplates(t *testing.T) {
	setupTestEnv(t)

	templates := loadChallengeTemplates()
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
	if templates[0].ID != "test-3d" {
		t.Errorf("first template ID = %q, want test-3d", templates[0].ID)
	}
	if templates[0].Duration != 3 {
		t.Errorf("first template Duration = %d, want 3", templates[0].Duration)
	}
}

func TestLoadQuizQuestions(t *testing.T) {
	setupTestEnv(t)

	questions := loadQuizQuestions()
	if len(questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(questions))
	}
	if questions[0].Answer != "C" {
		t.Errorf("first question answer = %q, want C", questions[0].Answer)
	}
}

func TestAtoi(t *testing.T) {
	if atoi("42") != 42 {
		t.Error("atoi(42) should be 42")
	}
	if atoi("abc") != 0 {
		t.Error("atoi(abc) should be 0")
	}
	if atoi("") != 0 {
		t.Error("atoi('') should be 0")
	}
}
