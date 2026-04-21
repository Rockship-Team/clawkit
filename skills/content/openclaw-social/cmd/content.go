package main

// Hardcoded buckets, formats, voice — single source of truth.
// SKILL.md must reference only these names.

type Bucket struct {
	Name           string   `json:"name"`
	DisplayName    string   `json:"display_name"`
	Goal           string   `json:"goal"`
	ExampleAngles  []string `json:"example_angles"`
	FrequencyHint  string   `json:"frequency_hint"`
}

var proposedBuckets = []Bucket{
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
		Goal:        "Update Claude/OpenClaw/ecosystem features moi.",
		ExampleAngles: []string{
			"Claude 4.7 co 1M context — 3 use case dang test",
			"OpenClaw feature X — demo 30s",
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

func isValidBucket(name string) bool {
	for _, b := range proposedBuckets {
		if b.Name == name {
			return true
		}
	}
	return false
}

var voiceDo = []string{
	"Mo bai bang moi hoc / surprising finding: 'Tuan truoc em thu X va ngo ngang vi...'",
	"Numbers/data cu the: 'giam 60% cost', 'tu 4h xuong 2 phut'",
	"Share mistakes + learning: 'bot em ngao 3 lan truoc khi fix dung'",
	"Ket thuc bang question hoac CTA",
	"Short paragraph — 2-4 cau roi xuong dong",
}

var voiceDont = []string{
	"Buzzword rong: 'revolutionary', 'game-changing', 'paradigm shift'",
	"Hashtag spam — toi da 3 hashtag",
	"Tu khen 'we are the best' — de reader tu ket luan",
	"Emoji qua 2-3 lan/post",
	"Pure promo — moi bai phai co 1 value/insight",
	"'In this post, we will...' — cut straight to hook",
}

type PostFormat struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Hook        string   `json:"hook_pattern"`
	Body        []string `json:"body_beats"`
	CTA         string   `json:"cta_pattern"`
	MediaHint   string   `json:"media_hint"`
	BestFor     []string `json:"best_for_buckets"`
}

var postFormats = []PostFormat{
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
