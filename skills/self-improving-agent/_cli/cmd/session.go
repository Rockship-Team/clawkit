package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func cmdSession(args []string) {
	if len(args) < 1 {
		errOut("usage: vault-cli session <save|search|list> [args...]")
	}

	switch args[0] {
	case "save":
		sessionSave(args[1:])
	case "search":
		sessionSearch(args[1:])
	case "list":
		sessionList(args[1:])
	default:
		errOut(fmt.Sprintf("unknown session subcommand: %s", args[0]))
	}
}

// sessionSave upserts a session and inserts a message.
// Usage: session save <session_id> <title> <skill> <role> <content>
func sessionSave(args []string) {
	if len(args) < 5 {
		errOut("usage: vault-cli session save <session_id> <title> <skill> <role> <content>")
	}

	sessionID := args[0]
	title := args[1]
	skill := args[2]
	role := args[3]
	content := args[4]

	db := openSessionDB()
	defer db.Close()

	now := vnNowISO()

	// Upsert session — use skill in metadata
	metadata := fmt.Sprintf(`{"skill":%q}`, skill)
	_, err := db.Exec(`
		INSERT INTO sessions (id, title, started_at, metadata)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			metadata = excluded.metadata
	`, sessionID, title, now, metadata)
	if err != nil {
		errOut(fmt.Sprintf("cannot upsert session: %v", err))
	}

	// Insert message
	msgID := newID()
	_, err = db.Exec(`
		INSERT INTO messages (id, session_id, role, content, timestamp)
		VALUES (?, ?, ?, ?, ?)
	`, msgID, sessionID, role, content, now)
	if err != nil {
		errOut(fmt.Sprintf("cannot insert message: %v", err))
	}

	// Get message count
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM messages WHERE session_id = ?`, sessionID).Scan(&count)
	if err != nil {
		errOut(fmt.Sprintf("cannot count messages: %v", err))
	}

	jsonOut(map[string]interface{}{
		"status":        "ok",
		"session_id":    sessionID,
		"message_id":    msgID,
		"message_count": count,
	})
}

// sessionSearch performs FTS5 search with snippet(), joins to sessions for title/skill.
// Usage: session search <query> [limit]
func sessionSearch(args []string) {
	if len(args) < 1 {
		errOut("usage: vault-cli session search <query> [limit]")
	}

	query := args[0]
	limit := 20
	if len(args) > 1 {
		if n, err := strconv.Atoi(args[1]); err == nil && n > 0 {
			limit = n
		}
	}

	db := openSessionDB()
	defer db.Close()

	rows, err := db.Query(`
		SELECT
			m.session_id,
			s.title,
			s.metadata,
			m.role,
			snippet(messages_fts, 0, '>>>', '<<<', '...', 40) AS snippet
		FROM messages_fts
		JOIN messages m ON m.rowid = messages_fts.rowid
		JOIN sessions s ON s.id = m.session_id
		WHERE messages_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, limit)
	if err != nil {
		errOut(fmt.Sprintf("search failed: %v", err))
	}
	defer rows.Close()

	type searchResult struct {
		SessionID string `json:"session_id"`
		Title     string `json:"title"`
		Skill     string `json:"skill"`
		Role      string `json:"role"`
		Snippet   string `json:"snippet"`
	}

	var results []searchResult
	for rows.Next() {
		var r searchResult
		var metadata string
		if err := rows.Scan(&r.SessionID, &r.Title, &metadata, &r.Role, &r.Snippet); err != nil {
			continue
		}
		r.Skill = extractSkillFromMetadata(metadata)
		results = append(results, r)
	}

	jsonOut(map[string]interface{}{
		"status":  "ok",
		"query":   query,
		"count":   len(results),
		"results": results,
	})
}

// sessionList shows recent sessions ordered by created_at desc.
// Usage: session list [limit]
func sessionList(args []string) {
	limit := 20
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
			limit = n
		}
	}

	db := openSessionDB()
	defer db.Close()

	rows, err := db.Query(`
		SELECT
			s.id,
			s.title,
			s.started_at,
			s.metadata,
			(SELECT COUNT(*) FROM messages WHERE session_id = s.id) AS msg_count
		FROM sessions s
		ORDER BY s.started_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		errOut(fmt.Sprintf("cannot list sessions: %v", err))
	}
	defer rows.Close()

	type sessionSummary struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		StartedAt    string `json:"started_at"`
		Skill        string `json:"skill"`
		MessageCount int    `json:"message_count"`
	}

	var sessions []sessionSummary
	for rows.Next() {
		var s sessionSummary
		var metadata string
		if err := rows.Scan(&s.ID, &s.Title, &s.StartedAt, &metadata, &s.MessageCount); err != nil {
			continue
		}
		s.Skill = extractSkillFromMetadata(metadata)
		sessions = append(sessions, s)
	}

	jsonOut(map[string]interface{}{
		"status":   "ok",
		"count":    len(sessions),
		"sessions": sessions,
	})
}

// cmdSearch performs combined search across vault notes AND session messages.
func cmdSearch(args []string) {
	if len(args) < 1 {
		errOut("usage: vault-cli search <query>")
	}

	query := args[0]

	type result struct {
		Source  string `json:"source"`
		Path    string `json:"path,omitempty"`
		Snippet string `json:"snippet"`
		// Session fields
		SessionID string `json:"session_id,omitempty"`
		Title     string `json:"title,omitempty"`
		Role      string `json:"role,omitempty"`
	}

	var results []result

	// Search vault notes
	vault := mustVaultPath()
	notes, err := listNotes(vault)
	if err == nil {
		lowerQuery := strings.ToLower(query)
		for _, n := range notes {
			content, err := readNote(vault, n)
			if err != nil {
				continue
			}
			lower := strings.ToLower(content)
			idx := strings.Index(lower, lowerQuery)
			if idx < 0 {
				continue
			}
			start := idx - 50
			if start < 0 {
				start = 0
			}
			end := idx + len(query) + 100
			if end > len(content) {
				end = len(content)
			}
			results = append(results, result{
				Source:  "vault",
				Path:    n,
				Snippet: content[start:end],
			})
		}
	}

	// Search session messages via FTS5
	db := openSessionDB()
	defer db.Close()

	rows, err := db.Query(`
		SELECT
			m.session_id,
			s.title,
			m.role,
			snippet(messages_fts, 0, '>>>', '<<<', '...', 40) AS snippet
		FROM messages_fts
		JOIN messages m ON m.rowid = messages_fts.rowid
		JOIN sessions s ON s.id = m.session_id
		WHERE messages_fts MATCH ?
		ORDER BY rank
		LIMIT 20
	`, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r result
			r.Source = "session"
			if err := rows.Scan(&r.SessionID, &r.Title, &r.Role, &r.Snippet); err != nil {
				continue
			}
			results = append(results, r)
		}
	}

	jsonOut(map[string]interface{}{
		"status":  "ok",
		"query":   query,
		"count":   len(results),
		"results": results,
	})
}

// extractSkillFromMetadata parses the skill field from a JSON metadata string.
func extractSkillFromMetadata(metadata string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &m); err != nil {
		return ""
	}
	if skill, ok := m["skill"].(string); ok {
		return skill
	}
	return ""
}
