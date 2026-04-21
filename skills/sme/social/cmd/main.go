// ocs-cli — {brand_name} Social CLI
// Deterministic helpers for Facebook content planning skill.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "buckets":
		cmdBuckets()
	case "voice":
		cmdVoice()
	case "formats":
		cmdFormats()
	case "next-slot":
		cmdNextSlot()
	case "draft":
		cmdDraft(os.Args[2:])
	case "get":
		cmdGet(os.Args[2:])
	case "update":
		cmdUpdate(os.Args[2:])
	case "list":
		cmdList(os.Args[2:])
	case "schedule":
		cmdSchedule(os.Args[2:])
	case "mark-posted":
		cmdMarkPosted(os.Args[2:])
	case "upcoming":
		cmdUpcoming(os.Args[2:])
	case "delete":
		cmdDelete(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `ocs-cli — {brand_name} Social (Facebook content planner)

  buckets              List 7 topic buckets (JSON, hardcoded — never invent new buckets)
  voice                Brand voice guide (tone, do/don't)
  formats              5 post format templates
  next-slot            Next free Mon/Thu 10am ICT slot

  draft <bucket> <title>       Create draft → returns id
  update <id> <field> <value>  Edit field: hook|body|cta|media_note|title|bucket
  get <id>                     Show post
  list [--status <s>] [--bucket <b>] [--limit <n>]
  schedule <id> <YYYY-MM-DDTHH:MM+0700>  Validates Mon/Thu 10am, no double-book
  mark-posted <id>             Set status=posted, record posted_at
  upcoming [--days N]          Scheduled posts in next N days (default 14)
  delete <id>                  Hard delete a post`)
}

// --- helpers ---

func errOut(msg string) {
	json.NewEncoder(os.Stdout).Encode(map[string]interface{}{"ok": false, "error": msg})
	os.Exit(1)
}

func okOut(extra map[string]interface{}) {
	out := map[string]interface{}{"ok": true}
	for k, v := range extra {
		out[k] = v
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	enc.Encode(out)
}

func ictLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil || loc == nil {
		return time.FixedZone("ICT", 7*3600)
	}
	return loc
}

func dataDir() string {
	home, _ := os.UserHomeDir()
	d := filepath.Join(home, ".openclaw", "workspace", "social-data")
	os.MkdirAll(d, 0o755)
	return d
}

func dataFile() string { return filepath.Join(dataDir(), "posts.json") }

func genID() string {
	b := make([]byte, 5)
	rand.Read(b)
	return "s_" + hex.EncodeToString(b)
}

// --- post struct & storage ---

type Post struct {
	ID           string `json:"id"`
	Bucket       string `json:"bucket"`
	Title        string `json:"title"`
	Hook         string `json:"hook,omitempty"`
	Body         string `json:"body,omitempty"`
	CTA          string `json:"cta,omitempty"`
	MediaNote    string `json:"media_note,omitempty"`
	Status       string `json:"status"`        // draft, scheduled, posted, archived
	ScheduledFor string `json:"scheduled_for,omitempty"`
	PostedAt     string `json:"posted_at,omitempty"`
	Platform     string `json:"platform"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type store struct {
	Posts []Post `json:"posts"`
}

func loadStore() *store {
	s := &store{Posts: []Post{}}
	f, err := os.ReadFile(dataFile())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(f, s)
	if s.Posts == nil {
		s.Posts = []Post{}
	}
	return s
}

func saveStore(s *store) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		errOut("failed to serialize store: " + err.Error())
	}
	tmp := dataFile() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		errOut("failed to write store: " + err.Error())
	}
	if err := os.Rename(tmp, dataFile()); err != nil {
		errOut("failed to rename store: " + err.Error())
	}
}

func (s *store) findByID(id string) *Post {
	for i := range s.Posts {
		if s.Posts[i].ID == id {
			return &s.Posts[i]
		}
	}
	return nil
}

// --- argv helpers ---

func popFlag(args []string, name string) (value string, rest []string) {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == name && i+1 < len(args) {
			value = args[i+1]
			i++
			continue
		}
		out = append(out, args[i])
	}
	return value, out
}

func nowICT() string { return time.Now().In(ictLoc()).Format(time.RFC3339) }

// --- slot math ---

// isValidSlot returns (ok, reason). A slot is valid when it is Mon or Thu 10:00 ICT.
func isValidSlot(t time.Time) (bool, string) {
	t = t.In(ictLoc())
	if t.Hour() != 10 || t.Minute() != 0 || t.Second() != 0 {
		return false, "slot must be exactly 10:00 ICT (Asia/Ho_Chi_Minh)"
	}
	wd := t.Weekday()
	if wd != time.Monday && wd != time.Thursday {
		return false, fmt.Sprintf("slot must be Monday or Thursday, got %s", wd)
	}
	return true, ""
}

func nextSlotAfter(t time.Time) time.Time {
	t = t.In(ictLoc())
	base := time.Date(t.Year(), t.Month(), t.Day(), 10, 0, 0, 0, ictLoc())
	for d := 0; d < 14; d++ {
		cand := base.AddDate(0, 0, d)
		wd := cand.Weekday()
		if (wd == time.Monday || wd == time.Thursday) && cand.After(t) {
			return cand
		}
	}
	return base.AddDate(0, 0, 14)
}

// --- commands ---

func cmdBuckets() {
	okOut(map[string]interface{}{
		"buckets": proposedBuckets,
		"rules": []string{
			"Only these 7 buckets exist. Never invent new buckets.",
			"Rotation: do not use the same bucket two posts in a row.",
			"Max 3 consecutive posts per bucket within a rolling 2-week window.",
		},
	})
}

func cmdVoice() {
	okOut(map[string]interface{}{
		"tone":          "casual builder voice, confident not braggy, practical first, Vietnamese natural with English tech terms",
		"do":            voiceDo,
		"dont":          voiceDont,
		"length_words":  "80-200 words (longer if code or case study)",
		"emoji_per_post": "max 2-3, placed at section landmarks (🎯 💡 🛠️)",
		"hashtag_rules": "max 3 hashtags, always include #{brand_name} or #Rockship, plus 1 tech tag",
	})
}

func cmdFormats() {
	okOut(map[string]interface{}{
		"formats": postFormats,
		"pick_by_bucket": map[string][]string{
			"tricks":     {"discovery", "listicle"},
			"innovation": {"discovery", "hot_take"},
			"team":       {"story", "showcase"},
			"cases":      {"story", "showcase"},
			"community":  {"showcase"},
			"insights":   {"hot_take", "listicle"},
			"tools":      {"discovery", "listicle"},
		},
	})
}

func cmdNextSlot() {
	s := loadStore()
	cand := nextSlotAfter(time.Now())
	for i := 0; i < 14; i++ {
		candStr := cand.Format(time.RFC3339)
		taken := false
		for _, p := range s.Posts {
			if p.Status == "scheduled" && p.ScheduledFor == candStr {
				taken = true
				break
			}
		}
		if !taken {
			weekday := cand.Weekday().String()
			okOut(map[string]interface{}{
				"slot":      candStr,
				"weekday":   weekday,
				"local":     cand.Format("2006-01-02 15:04 MST"),
				"note":      fmt.Sprintf("Next free %s 10:00 ICT.", weekday),
			})
			return
		}
		cand = nextSlotAfter(cand)
	}
	errOut("no free slot in the next 14 days — check scheduled queue")
}

func cmdDraft(args []string) {
	if len(args) < 2 {
		errOut("usage: draft <bucket> <title>")
	}
	bucket := args[0]
	title := strings.Join(args[1:], " ")
	if !isValidBucket(bucket) {
		errOut(fmt.Sprintf("invalid bucket %q — run `ocs-cli buckets` to see the 7 allowed names", bucket))
	}
	s := loadStore()
	now := nowICT()
	p := Post{
		ID:        genID(),
		Bucket:    bucket,
		Title:     title,
		Status:    "draft",
		Platform:  "facebook",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.Posts = append(s.Posts, p)
	saveStore(s)
	okOut(map[string]interface{}{
		"post": p,
		"next": "fill hook/body/cta via `ocs-cli update <id> <field> <value>`",
	})
}

func cmdGet(args []string) {
	if len(args) < 1 {
		errOut("usage: get <id>")
	}
	s := loadStore()
	p := s.findByID(args[0])
	if p == nil {
		errOut("post not found: " + args[0])
	}
	okOut(map[string]interface{}{"post": p})
}

func cmdUpdate(args []string) {
	if len(args) < 3 {
		errOut("usage: update <id> <field> <value>")
	}
	id := args[0]
	field := args[1]
	value := strings.Join(args[2:], " ")
	s := loadStore()
	p := s.findByID(id)
	if p == nil {
		errOut("post not found: " + id)
	}
	switch field {
	case "title":
		p.Title = value
	case "hook":
		p.Hook = value
	case "body":
		p.Body = value
	case "cta":
		p.CTA = value
	case "media_note":
		p.MediaNote = value
	case "bucket":
		if !isValidBucket(value) {
			errOut(fmt.Sprintf("invalid bucket %q", value))
		}
		p.Bucket = value
	default:
		errOut(fmt.Sprintf("unknown field %q — allowed: title, hook, body, cta, media_note, bucket", field))
	}
	p.UpdatedAt = nowICT()
	saveStore(s)
	okOut(map[string]interface{}{"post": p})
}

func cmdList(args []string) {
	status, args := popFlag(args, "--status")
	bucket, args := popFlag(args, "--bucket")
	limitStr, _ := popFlag(args, "--limit")
	limit := 50
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	s := loadStore()
	out := make([]Post, 0, len(s.Posts))
	for _, p := range s.Posts {
		if status != "" && p.Status != status {
			continue
		}
		if bucket != "" && p.Bucket != bucket {
			continue
		}
		out = append(out, p)
		if len(out) >= limit {
			break
		}
	}
	okOut(map[string]interface{}{"posts": out, "count": len(out)})
}

func cmdSchedule(args []string) {
	if len(args) < 2 {
		errOut("usage: schedule <id> <YYYY-MM-DDTHH:MM+0700>")
	}
	id := args[0]
	raw := args[1]
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		errOut("datetime must be RFC3339 with timezone, e.g. 2026-05-04T10:00:00+07:00")
	}
	if ok, reason := isValidSlot(t); !ok {
		errOut(reason)
	}
	s := loadStore()
	p := s.findByID(id)
	if p == nil {
		errOut("post not found: " + id)
	}
	if p.Status != "draft" && p.Status != "approved" {
		errOut(fmt.Sprintf("cannot schedule post in status %q — must be draft or approved", p.Status))
	}
	slot := t.Format(time.RFC3339)
	for _, other := range s.Posts {
		if other.ID == id {
			continue
		}
		if other.Status == "scheduled" && other.ScheduledFor == slot {
			errOut(fmt.Sprintf("slot %s already taken by post %s — pick a different slot", slot, other.ID))
		}
	}
	if p.Hook == "" || p.Body == "" {
		errOut("post missing hook or body — fill via `ocs-cli update` before scheduling")
	}
	p.Status = "scheduled"
	p.ScheduledFor = slot
	p.UpdatedAt = nowICT()
	saveStore(s)
	okOut(map[string]interface{}{
		"post":    p,
		"message": fmt.Sprintf("Scheduled for %s ICT. Remember: manual post.", t.In(ictLoc()).Format("Mon 02/01/2006 15:04")),
	})
}

func cmdMarkPosted(args []string) {
	if len(args) < 1 {
		errOut("usage: mark-posted <id>")
	}
	s := loadStore()
	p := s.findByID(args[0])
	if p == nil {
		errOut("post not found: " + args[0])
	}
	if p.Status != "scheduled" && p.Status != "approved" && p.Status != "draft" {
		errOut(fmt.Sprintf("cannot mark-posted from status %q", p.Status))
	}
	p.Status = "posted"
	p.PostedAt = nowICT()
	p.UpdatedAt = nowICT()
	saveStore(s)
	okOut(map[string]interface{}{"post": p})
}

func cmdUpcoming(args []string) {
	daysStr, _ := popFlag(args, "--days")
	days := 14
	if daysStr != "" {
		fmt.Sscanf(daysStr, "%d", &days)
	}
	s := loadStore()
	now := time.Now().In(ictLoc())
	until := now.AddDate(0, 0, days)
	out := make([]Post, 0)
	for _, p := range s.Posts {
		if p.Status != "scheduled" {
			continue
		}
		t, err := time.Parse(time.RFC3339, p.ScheduledFor)
		if err != nil {
			continue
		}
		if t.After(now) && t.Before(until) {
			out = append(out, p)
		}
	}
	okOut(map[string]interface{}{
		"posts": out,
		"count": len(out),
		"window": map[string]string{
			"from": now.Format(time.RFC3339),
			"to":   until.Format(time.RFC3339),
			"days": fmt.Sprintf("%d", days),
		},
	})
}

func cmdDelete(args []string) {
	if len(args) < 1 {
		errOut("usage: delete <id>")
	}
	id := args[0]
	s := loadStore()
	idx := -1
	for i, p := range s.Posts {
		if p.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		errOut("post not found: " + id)
	}
	s.Posts = append(s.Posts[:idx], s.Posts[idx+1:]...)
	saveStore(s)
	okOut(map[string]interface{}{"deleted_id": id})
}
