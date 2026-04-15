# SOL Finance Coach — Implementation Approach

## Current State

The sol-finance-coach skill has **all 13 CLI commands fully implemented** (1,863 LOC in `cmd/`) and a **working web crawler** (1,806 LOC in `tools/crawl/`). There are no stubs or placeholder code. The remaining work falls into 5 categories: data expansion, cron automation, messaging integration, OpenClaw platform integration, and testing.

---

## Phase 1: OpenClaw Platform Integration (Priority: HIGH)

**Goal:** Make the skill installable via `clawkit install sol-finance-coach`.

### 1.1 Create `schema.json`

The OpenClaw installer expects a `schema.json` defining the data model. For sol-finance-coach, this is the user state (not the static knowledge data).

```json
{
  "tables": {
    "profile": {
      "fields": [
        {"name": "income", "type": "integer", "role": "price"},
        {"name": "goal", "type": "text"},
        {"name": "risk_level", "type": "text"},
        {"name": "credit_cards", "type": "text"},
        {"name": "knowledge_level", "type": "text"},
        {"name": "daily_tips", "type": "text"},
        {"name": "name", "type": "text"},
        {"name": "monthly_fixed", "type": "integer", "role": "price"}
      ]
    },
    "transactions": {
      "fields": [
        {"name": "id", "type": "integer", "auto": "increment"},
        {"name": "place", "type": "text"},
        {"name": "amount", "type": "integer", "role": "price"},
        {"name": "category", "type": "text"},
        {"name": "note", "type": "text"},
        {"name": "date", "type": "text", "auto": "timestamp", "role": "timestamp"}
      ]
    }
  },
  "primary": "transactions",
  "timezone": "Asia/Ho_Chi_Minh"
}
```

**Decision:** sol-cli manages its own JSON files under `~/.openclaw/workspace/skills/sol-finance-coach/`. This is simpler than using cli.js + Supabase for now. The schema.json serves as documentation for the installer and potential future migration to Supabase.

### 1.2 Register in Skills Directory

1. Move `sol-finance-coach/` under an appropriate vertical (e.g., `skills/utilities/sol-finance-coach/`)
2. Ensure SKILL.md frontmatter has correct fields (name, description, metadata) and clawkit.json has version, setup_prompts, etc.
3. Run `make generate` to add to `registry.json`
4. Update `skills/skills.go` embed directive if needed

### 1.3 Build sol-cli Binary

The sol-cli binary needs to be cross-compiled and included in the skill package:

```bash
# In cmd/ directory
GOOS=darwin GOARCH=arm64 go build -o sol-cli-darwin-arm64
GOOS=darwin GOARCH=amd64 go build -o sol-cli-darwin-amd64
GOOS=linux GOARCH=amd64 go build -o sol-cli-linux-amd64
GOOS=windows GOARCH=amd64 go build -o sol-cli-windows-amd64.exe
```

**Approach:** Add a Makefile target or build script that produces platform binaries. The installer should select the correct binary at install time (similar to how clawkit itself handles cross-platform).

---

## Phase 2: Data Expansion (Priority: HIGH)

**Goal:** Meet PRD data volume targets for a credible MVP.

### 2.1 Tips: 40 → 200+

Current distribution by category (from `data/tips.json`):
- food, transport, shopping, bills, entertainment, general

**Approach:**
1. Run `tools/crawl/crawl all` to seed additional content from Vietnamese financial sites
2. Manually curate and add tips in batches by category
3. Add seasonal tips (Tet spending, back-to-school, 11.11/12.12 sales)
4. Target: 30-40 tips per category

### 2.2 Credit Cards: Expand via Crawler

The crawler already covers 12 major Vietnamese banks. Run it to populate a larger `credit-cards-crawled.json`, then manually review and merge quality entries into `credit-cards.json`.

### 2.3 Knowledge Base: Expand for Better LLM Context

Current `knowledge-base.md` is 152 lines. Expand to cover:
- Investment basics: stocks, bonds, funds, gold, savings (deeper)
- Tax basics for salaried employees
- Insurance fundamentals (health, life, vehicle)
- Vietnamese financial regulations relevant to individuals
- Common scam patterns and how to avoid them

**Approach:** This content goes into SKILL.md prompt context (not RAG). Keep it concise but comprehensive — the LLM uses it as reference when answering questions.

---

## Phase 3: Cron & Automation (Priority: MEDIUM)

**Goal:** Bot proactively engages users instead of only responding.

### 3.1 Daily Digest Cron

Create OpenClaw cron configuration:

```yaml
# cron/daily-digest.yaml
schedule: "30 7 * * *"  # 7:30 AM Vietnam time
command: |
  Run: skills/sol-finance-coach/sol-cli digest generate
  Format the output as a friendly morning message.
  Send to user.
```

**Implementation notes:**
- `digest generate` already aggregates from 5 sources — no new code needed
- The cron job is an OpenClaw platform feature (not sol-cli responsibility)
- Needs OpenClaw cron documentation to confirm exact YAML format

### 3.2 Weekly Spending Report

```yaml
# cron/weekly-spending-report.yaml
schedule: "0 20 * * 0"  # Sunday 20:00
command: |
  Run: skills/sol-finance-coach/sol-cli spend report week
  Render as ASCII chart with category breakdown.
  Include comparison to previous week if available.
```

### 3.3 Deal Alerts

```yaml
# cron/deal-alerts.yaml
schedule: "0 8,12,18 * * *"  # 8 AM, noon, 6 PM
command: |
  Run: skills/sol-finance-coach/sol-cli deals match
  If matches found, send top deal to user.
```

**Dependency:** Requires understanding OpenClaw's cron system. If cron is not yet supported, these become manual triggers or rely on the messaging platform's scheduling features.

---

## Phase 4: Messaging Integration (Priority: MEDIUM)

**Goal:** Users interact via Zalo/Telegram, not a terminal.

### 4.1 Telegram Bot (Recommended First)

Telegram is the easiest integration point:
- Well-documented Bot API
- Free, no approval process
- Supports inline keyboards for quiz choices, challenge check-ins
- Webhook or long-polling

**Approach:** OpenClaw handles the messaging layer. The skill only needs to ensure SKILL.md instructions work with OpenClaw's channel adapters. No code changes to sol-cli.

**Key interaction patterns to test:**
- Quick expense logging: "cafe 55k" → auto-parse and save
- Quiz with inline keyboard buttons (A/B/C/D)
- Challenge check-in with streak counter
- Daily digest as a formatted message

### 4.2 Zalo OA (Second Priority)

Requires Zalo OA account approval (business verification). More friction but larger Vietnamese user base.

### 4.3 WhatsApp (Deferred)

Requires Meta Business account + API access. Highest friction, defer to post-MVP.

---

## Phase 5: Testing (Priority: MEDIUM)

**Goal:** Confidence in correctness, especially financial calculations.

### 5.1 Unit Tests for `cmd/`

Priority test targets (by risk if wrong):

1. **simulator.go** — Financial math must be correct
   - Compound interest: known inputs → verify FV matches manual calculation
   - Loan amortization: verify monthly payment, total interest
   - Goal planner: verify monthly savings amounts across 3 scenarios

2. **store.go:parseAmount()** — Currency parsing edge cases
   - "55k" → 55000
   - "1.5tr" → 1500000
   - "55.000" → 55000
   - "1tr5" → 1500000
   - "0" → 0
   - Invalid input → error

3. **spending.go** — Date range filtering
   - "today" includes only today's transactions
   - "week" covers Monday-Sunday of current week
   - Category percentages sum to 100%

4. **gamification.go** — Badge logic
   - First challenge completion → savings_newbie badge
   - 5 challenges → challenge_master badge
   - 20 correct quizzes → finance_101 badge

### 5.2 Test Structure

```
cmd/
  simulator_test.go
  store_test.go
  spending_test.go
  gamification_test.go
```

Use Go standard `testing` package only (zero external deps rule). Test with temp directories to isolate user data.

### 5.3 Crawler Tests

Lower priority — crawlers are heuristic-based and will break when bank websites change. Instead of unit tests, add a `crawl validate` command that checks output format (valid JSON, required fields present, reasonable value ranges).

---

## Phase 6: Polish & Engagement (Priority: LOW)

### 6.1 Referral System

Add to `feedback` command:
```
sol-cli feedback referral generate  → unique referral code
sol-cli feedback referral redeem <code>  → track referral
```

### 6.2 NPS Tracking

Extend `feedback rate` to calculate NPS:
- Score 1-3: detractor
- Score 4: passive
- Score 5: promoter
- `feedback stats` → NPS score + response count

### 6.3 More Challenges

Expand `data/challenges.json` from 10 to 20+:
- "7 ngay khong dat tra sua"
- "30 ngay tiet kiem 100K/ngay"
- "No Spend Weekend"
- "Meal Prep Challenge" (1 week)
- "Walk/Bus Instead of Grab" (1 week)
- Monthly savings milestones

### 6.4 Streak System

The quiz already tracks streaks. Extend to a global interaction streak:
- `sol-cli streak check` → days of consecutive interaction
- Badge at 7, 30, 100 day streaks

---

## Architecture Decisions

### Why sol-cli (Go binary) instead of cli.js?

1. **Financial calculations** — Go's math is precise and testable in isolation
2. **Data parsing** — Vietnamese currency formats need robust parsing
3. **Offline-first** — All data is local JSON, no network dependency for core features
4. **Cross-platform** — Single binary, no Node.js runtime required for data ops
5. **Zero deps** — Matches clawkit's zero-external-dependency rule

### Why local JSON instead of Supabase?

1. **Privacy** — Financial data stays on user's machine
2. **Simplicity** — No account setup, no API keys for core features
3. **Offline** — Works without internet (except crawler and LLM)
4. **Migration path** — Can add Supabase later via schema.json db_target change

### Why no RAG?

The PRD mentions RAG for `financial-knowledge-base`, but the current approach embeds knowledge directly in SKILL.md prompt context. This is simpler and sufficient for the current knowledge base size (~200 lines). RAG becomes valuable when:
- Knowledge base exceeds LLM context window limits
- Content needs frequent updates without SKILL.md changes
- Multiple knowledge domains need selective retrieval

**Recommendation:** Defer RAG until knowledge base exceeds 500+ entries or response quality degrades.

---

## Sprint Plan (Remaining Work)

### Sprint A — Ship as OpenClaw Skill (1 day)
1. Create schema.json
2. Move to correct vertical directory
3. Cross-compile sol-cli binaries
4. Run `make generate`, verify `make check-generate` passes
5. Test `clawkit install sol-finance-coach` end-to-end

### Sprint B — Data & Content (1 day)
1. Run crawler, review and merge card/deal data
2. Expand tips.json to 100+ (batch by category)
3. Expand knowledge-base.md (tax, insurance, scams)
4. Add 10 more challenges

### Sprint C — Automation & Testing (1 day)
1. Write unit tests for simulator, parseAmount, spending
2. Create cron YAML configs (daily digest, weekly report, deal alerts)
3. Test with Telegram bot via OpenClaw channel adapter
4. End-to-end conversation testing (onboarding → spending → quiz → digest)

### Sprint D — Polish (0.5 day)
1. NPS tracking in feedback
2. Referral code generation
3. Global streak tracking
4. Final demo prep
