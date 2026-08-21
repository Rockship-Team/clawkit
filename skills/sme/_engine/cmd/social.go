package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cmdSocial dispatches Facebook content-planner subcommands.
//
//	sme-cli social buckets
//	sme-cli social voice
//	sme-cli social formats
//	sme-cli social next-slot
//	sme-cli social draft <bucket> <title>
//	sme-cli social update <id> <field> <value>
//	sme-cli social get <id>
//	sme-cli social list [--status s] [--bucket b] [--limit n]
//	sme-cli social schedule <id> <YYYY-MM-DDTHH:MM+0700>
//	sme-cli social mark-posted <id>
//	sme-cli social upcoming [--days N]
//	sme-cli social delete <id>
func cmdSocial(args []string) {
	if len(args) == 0 {
		errOut("usage: social buckets|voice|formats|next-slot|draft|update|get|list|schedule|mark-posted|upcoming|delete")
		return
	}
	switch args[0] {
	case "buckets":
		socialBuckets()
	case "voice":
		socialVoice()
	case "formats":
		socialFormats()
	case "next-slot":
		socialNextSlot()
	case "draft":
		socialDraft(args[1:])
	case "get":
		socialGet(args[1:])
	case "update":
		socialUpdate(args[1:])
	case "list":
		socialList(args[1:])
	case "schedule":
		socialSchedule(args[1:])
	case "mark-posted":
		socialMarkPosted(args[1:])
	case "upcoming":
		socialUpcoming(args[1:])
	case "delete":
		socialDelete(args[1:])
	default:
		errOut("unknown social command: " + args[0])
	}
}

// --- brand + output ---

func socialBrandName() string {
	if v := strings.TrimSpace(os.Getenv("OCS_BRAND_NAME")); v != "" {
		return v
	}
	return "Rockship"
}

// socialOkOut writes okOut-style JSON but substitutes {brand_name} placeholder
// in the serialized payload. Needed because bucket/voice strings embed the
// brand placeholder for display.
func socialOkOut(extra map[string]interface{}) {
	out := map[string]interface{}{"ok": true}
	for k, v := range extra {
		out[k] = v
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	enc.Encode(out)
	os.Stdout.Write(bytes.ReplaceAll(buf.Bytes(), []byte("{brand_name}"), []byte(socialBrandName())))
}

// --- time / paths / id ---

func socialICTLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil || loc == nil {
		return time.FixedZone("ICT", 7*3600)
	}
	return loc
}

func socialDataDir() string {
	home, _ := os.UserHomeDir()
	d := filepath.Join(home, ".openclaw", "workspace", "social-data")
	os.MkdirAll(d, 0o755)
	return d
}

func socialDataFile() string { return filepath.Join(socialDataDir(), "posts.json") }

func socialNowICT() string { return time.Now().In(socialICTLoc()).Format(time.RFC3339) }

func socialGenID() string {
	b := make([]byte, 5)
	rand.Read(b)
	return "s_" + hex.EncodeToString(b)
}

// --- post + store ---

type SocialPost struct {
	ID           string `json:"id"`
	Bucket       string `json:"bucket"`
	Title        string `json:"title"`
	Hook         string `json:"hook,omitempty"`
	Body         string `json:"body,omitempty"`
	CTA          string `json:"cta,omitempty"`
	MediaNote    string `json:"media_note,omitempty"`
	Status       string `json:"status"`
	ScheduledFor string `json:"scheduled_for,omitempty"`
	PostedAt     string `json:"posted_at,omitempty"`
	Platform     string `json:"platform"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type socialStore struct {
	Posts []SocialPost `json:"posts"`
}

func socialLoadStore() *socialStore {
	s := &socialStore{Posts: []SocialPost{}}
	f, err := os.ReadFile(socialDataFile())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(f, s)
	if s.Posts == nil {
		s.Posts = []SocialPost{}
	}
	return s
}

func socialSaveStore(s *socialStore) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		errOut("failed to serialize social store: " + err.Error())
	}
	tmp := socialDataFile() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		errOut("failed to write social store: " + err.Error())
	}
	if err := os.Rename(tmp, socialDataFile()); err != nil {
		errOut("failed to rename social store: " + err.Error())
	}
}

func (s *socialStore) findByID(id string) *SocialPost {
	for i := range s.Posts {
		if s.Posts[i].ID == id {
			return &s.Posts[i]
		}
	}
	return nil
}

// --- argv + slot math ---

func socialPopFlag(args []string, name string) (value string, rest []string) {
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

func socialIsValidSlot(t time.Time) (bool, string) {
	t = t.In(socialICTLoc())
	if t.Hour() != 10 || t.Minute() != 0 || t.Second() != 0 {
		return false, "slot must be exactly 10:00 ICT (Asia/Ho_Chi_Minh)"
	}
	wd := t.Weekday()
	if wd != time.Monday && wd != time.Thursday {
		return false, fmt.Sprintf("slot must be Monday or Thursday, got %s", wd)
	}
	return true, ""
}

func socialNextSlotAfter(t time.Time) time.Time {
	t = t.In(socialICTLoc())
	base := time.Date(t.Year(), t.Month(), t.Day(), 10, 0, 0, 0, socialICTLoc())
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

func socialBuckets() {
	socialOkOut(map[string]interface{}{
		"buckets": socialProposedBuckets,
		"rules": []string{
			"Only these 7 buckets exist. Never invent new buckets.",
			"Rotation: do not use the same bucket two posts in a row.",
			"Max 3 consecutive posts per bucket within a rolling 2-week window.",
		},
	})
}

func socialVoice() {
	socialOkOut(map[string]interface{}{
		"tone":           "casual builder voice, confident not braggy, practical first, Vietnamese natural with English tech terms",
		"do":             socialVoiceDo,
		"dont":           socialVoiceDont,
		"length_words":   "80-200 words (longer if code or case study)",
		"emoji_per_post": "max 2-3, placed at section landmarks (🎯 💡 🛠️)",
		"hashtag_rules":  "max 3 hashtags, always include #{brand_name}, plus 1 tech tag",
	})
}

func socialFormats() {
	socialOkOut(map[string]interface{}{
		"formats": socialPostFormats,
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

func socialNextSlot() {
	s := socialLoadStore()
	cand := socialNextSlotAfter(time.Now())
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
			socialOkOut(map[string]interface{}{
				"slot":    candStr,
				"weekday": weekday,
				"local":   cand.Format("2006-01-02 15:04 MST"),
				"note":    fmt.Sprintf("Next free %s 10:00 ICT.", weekday),
			})
			return
		}
		cand = socialNextSlotAfter(cand)
	}
	errOut("no free slot in the next 14 days — check scheduled queue")
}

func socialDraft(args []string) {
	if len(args) < 2 {
		errOut("usage: social draft <bucket> <title>")
	}
	bucket := args[0]
	title := strings.Join(args[1:], " ")
	if !socialIsValidBucket(bucket) {
		errOut(fmt.Sprintf("invalid bucket %q — run `sme-cli social buckets` to see the 7 allowed names", bucket))
	}
	s := socialLoadStore()
	now := socialNowICT()
	p := SocialPost{
		ID:        socialGenID(),
		Bucket:    bucket,
		Title:     title,
		Status:    "draft",
		Platform:  "facebook",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.Posts = append(s.Posts, p)
	socialSaveStore(s)
	socialOkOut(map[string]interface{}{
		"post": p,
		"next": "fill hook/body/cta via `sme-cli social update <id> <field> <value>`",
	})
}

func socialGet(args []string) {
	if len(args) < 1 {
		errOut("usage: social get <id>")
	}
	s := socialLoadStore()
	p := s.findByID(args[0])
	if p == nil {
		errOut("post not found: " + args[0])
	}
	socialOkOut(map[string]interface{}{"post": p})
}

func socialUpdate(args []string) {
	if len(args) < 3 {
		errOut("usage: social update <id> <field> <value>")
	}
	id := args[0]
	field := args[1]
	value := strings.Join(args[2:], " ")
	s := socialLoadStore()
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
		if !socialIsValidBucket(value) {
			errOut(fmt.Sprintf("invalid bucket %q", value))
		}
		p.Bucket = value
	default:
		errOut(fmt.Sprintf("unknown field %q — allowed: title, hook, body, cta, media_note, bucket", field))
	}
	p.UpdatedAt = socialNowICT()
	socialSaveStore(s)
	socialOkOut(map[string]interface{}{"post": p})
}

func socialList(args []string) {
	status, args := socialPopFlag(args, "--status")
	bucket, args := socialPopFlag(args, "--bucket")
	limitStr, _ := socialPopFlag(args, "--limit")
	limit := 50
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	s := socialLoadStore()
	out := make([]SocialPost, 0, len(s.Posts))
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
	socialOkOut(map[string]interface{}{"posts": out, "count": len(out)})
}

func socialSchedule(args []string) {
	if len(args) < 2 {
		errOut("usage: social schedule <id> <YYYY-MM-DDTHH:MM+0700>")
	}
	id := args[0]
	raw := args[1]
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		errOut("datetime must be RFC3339 with timezone, e.g. 2026-05-04T10:00:00+07:00")
	}
	if ok, reason := socialIsValidSlot(t); !ok {
		errOut(reason)
	}
	s := socialLoadStore()
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
		errOut("post missing hook or body — fill via `sme-cli social update` before scheduling")
	}
	p.Status = "scheduled"
	p.ScheduledFor = slot
	p.UpdatedAt = socialNowICT()
	socialSaveStore(s)
	socialOkOut(map[string]interface{}{
		"post":    p,
		"message": fmt.Sprintf("Scheduled for %s ICT. Remember: manual post.", t.In(socialICTLoc()).Format("Mon 02/01/2006 15:04")),
	})
}

func socialMarkPosted(args []string) {
	if len(args) < 1 {
		errOut("usage: social mark-posted <id>")
	}
	s := socialLoadStore()
	p := s.findByID(args[0])
	if p == nil {
		errOut("post not found: " + args[0])
	}
	if p.Status != "scheduled" && p.Status != "approved" && p.Status != "draft" {
		errOut(fmt.Sprintf("cannot mark-posted from status %q", p.Status))
	}
	p.Status = "posted"
	p.PostedAt = socialNowICT()
	p.UpdatedAt = socialNowICT()
	socialSaveStore(s)
	socialOkOut(map[string]interface{}{"post": p})
}

func socialUpcoming(args []string) {
	daysStr, _ := socialPopFlag(args, "--days")
	days := 14
	if daysStr != "" {
		fmt.Sscanf(daysStr, "%d", &days)
	}
	s := socialLoadStore()
	now := time.Now().In(socialICTLoc())
	until := now.AddDate(0, 0, days)
	out := make([]SocialPost, 0)
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
	socialOkOut(map[string]interface{}{
		"posts": out,
		"count": len(out),
		"window": map[string]string{
			"from": now.Format(time.RFC3339),
			"to":   until.Format(time.RFC3339),
			"days": fmt.Sprintf("%d", days),
		},
	})
}

func socialDelete(args []string) {
	if len(args) < 1 {
		errOut("usage: social delete <id>")
	}
	id := args[0]
	s := socialLoadStore()
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
	socialSaveStore(s)
	socialOkOut(map[string]interface{}{"deleted_id": id})
}

// --- content catalog (hardcoded — single source of truth) ---

type socialBucket struct {
	Name          string   `json:"name"`
	DisplayName   string   `json:"display_name"`
	Goal          string   `json:"goal"`
	ExampleAngles []string `json:"example_angles"`
	FrequencyHint string   `json:"frequency_hint"`
}

var socialProposedBuckets = []socialBucket{
	{
		Name:        "tricks",
		DisplayName: "Tricks & Tutorials",
		Goal:        "Help readers build AI agents faster and better.",
		ExampleAngles: []string{
			"5 cach giam cost Claude API bang prompt caching",
			"Prompt em dung de debug skill trong 30s",
			"Split tool call song song de agent nhanh gap 3",
			"Test skill truoc khi ship — day la checklist",
		},
		FrequencyHint: "1-2 bai/tuan, rotate voi innovation",
	},
	{
		Name:        "innovation",
		DisplayName: "Innovation Drops",
		Goal:        "Update Claude / {brand_name} / ecosystem features moi.",
		ExampleAngles: []string{
			"Claude 4.7 co 1M context — 3 use case dang test",
			"{brand_name} feature X — demo 30s",
			"Subagent spec moi cua Anthropic",
			"MCP server moi — tich hop the nao",
		},
		FrequencyHint: "1 bai/tuan khi co feature drop, bo qua khi im ang",
	},
	{
		Name:        "team",
		DisplayName: "Behind the Build",
		Goal:        "Story nhom Rockship dang xay, dogfood experience.",
		ExampleAngles: []string{
			"Tuan nay team build self-improve agent",
			"Bug ngao nhat tuan: bot tu xoa skill chinh no",
			"Anh Son vua demo bot cho khach X",
			"Internal tools tuan nay",
		},
		FrequencyHint: "1-2 bai/2 tuan",
	},
	{
		Name:        "cases",
		DisplayName: "Real Use Cases",
		Goal:        "Real deployment, customer win, before/after numbers.",
		ExampleAngles: []string{
			"Khach tiet kiem 20h/tuan voi bot proposal",
			"Tu 0 den 50 seller onboard bang 1 bot",
			"Before/after: response time 4h to 2 phut",
			"Case: 1 bot thay 3 CS junior, ROI 6 thang",
		},
		FrequencyHint: "1 bai/2 tuan khi co case moi",
	},
	{
		Name:        "community",
		DisplayName: "Community Spotlight",
		Goal:        "Shoutout user, skill open-source, ecosystem contribution.",
		ExampleAngles: []string{
			"Anh X build skill gold shop — 3 thu thu vi",
			"Skill finance-tracker — 50 install tuan dau",
			"Top 3 skill bi fork nhieu nhat",
			"Builder spotlight: em A",
		},
		FrequencyHint: "1 bai/thang",
	},
	{
		Name:        "insights",
		DisplayName: "Industry Insights",
		Goal:        "Trend, market analysis, opinion piece.",
		ExampleAngles: []string{
			"AI agent vs RPA — khi nao dung cai gi",
			"Vietnamese SME ap dung AI cham hon vi 3 ly do",
			"2026 la nam cua multi-agent",
			"Flash model khong du cho enterprise — ly do",
		},
		FrequencyHint: "1 bai/2 tuan",
	},
	{
		Name:        "tools",
		DisplayName: "Tool Deep Dives",
		Goal:        "Deep dive 1 tool/feature cu the.",
		ExampleAngles: []string{
			"Prompt caching: 90% giam cost — cach tinh",
			"MCP vs function calling: tradeoff",
			"Sub-agent spawn strategy",
			"Cronjob + skill: daily standup bot 5 phut",
		},
		FrequencyHint: "1 bai/2 tuan",
	},
}

func socialIsValidBucket(name string) bool {
	for _, b := range socialProposedBuckets {
		if b.Name == name {
			return true
		}
	}
	return false
}

var socialVoiceDo = []string{
	"Mo bai bang moi hoc / surprising finding: 'Tuan truoc em thu X va ngo ngang vi...'",
	"Numbers/data cu the: 'giam 60% cost', 'tu 4h xuong 2 phut'",
	"Share mistakes + learning: 'bot em ngao 3 lan truoc khi fix dung'",
	"Ket thuc bang question hoac CTA",
	"Short paragraph — 2-4 cau roi xuong dong",
}

var socialVoiceDont = []string{
	"Buzzword rong: 'revolutionary', 'game-changing', 'paradigm shift'",
	"Hashtag spam — toi da 3 hashtag",
	"Tu khen 'we are the best' — de reader tu ket luan",
	"Emoji qua 2-3 lan/post",
	"Pure promo — moi bai phai co 1 value/insight",
	"'In this post, we will...' — cut straight to hook",
}

type socialPostFormat struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Hook        string   `json:"hook_pattern"`
	Body        []string `json:"body_beats"`
	CTA         string   `json:"cta_pattern"`
	MediaHint   string   `json:"media_hint"`
	BestFor     []string `json:"best_for_buckets"`
}

var socialPostFormats = []socialPostFormat{
	{
		Name:        "discovery",
		DisplayName: "The Discovery",
		Hook:        "'Tuan truoc minh thu [X] va...'",
		Body:        []string{"Van de (1 cau)", "Cach thu (2-3 cau, co the code)", "Ket qua (1-2 cau, co number)"},
		CTA:         "'Ai da thu chua? Comment de share setup.'",
		MediaHint:   "screenshot config hoac before/after metric",
		BestFor:     []string{"tricks", "innovation", "tools"},
	},
	{
		Name:        "story",
		DisplayName: "The Story",
		Hook:        "'Co 1 chuyen tuan nay lam team cuoi lan / debate'",
		Body:        []string{"Context (1 cau)", "Moment ngo ngang / critical (2-3 cau)", "Bai hoc (1-2 cau)"},
		CTA:         "'Share full setup neu ai muon / tag builder khac'",
		MediaHint:   "screenshot conversation hoac code snippet funny",
		BestFor:     []string{"team", "cases"},
	},
	{
		Name:        "listicle",
		DisplayName: "The Listicle",
		Hook:        "'3 thu em uoc biet som hon khi build [X]'",
		Body:        []string{"Item 1 (1-2 cau + tip)", "Item 2 (1-2 cau + tip)", "Item 3 (1-2 cau + tip)"},
		CTA:         "'Thu nao anh em da biet? Comment them.'",
		MediaHint:   "numbered list graphic",
		BestFor:     []string{"tricks", "insights", "tools"},
	},
	{
		Name:        "hot_take",
		DisplayName: "The Hot Take",
		Hook:        "Bold statement, co the controversial",
		Body:        []string{"Thesis (1 cau)", "Reasoning (3-4 cau voi data)", "Counter ack (1-2 cau)"},
		CTA:         "'Disagree? Comment.'",
		MediaHint:   "quote card hoac chart",
		BestFor:     []string{"insights"},
	},
	{
		Name:        "showcase",
		DisplayName: "The Showcase",
		Hook:        "Spotlight cu the vao 1 agent/skill/user",
		Body:        []string{"Gioi thieu (1-2 cau)", "Highlight (metric/feature, 2-3 cau)", "Link (1 cau)"},
		CTA:         "'Check thu — link o day.'",
		MediaHint:   "screenshot agent output hoac demo GIF",
		BestFor:     []string{"community", "team", "cases"},
	},
}
