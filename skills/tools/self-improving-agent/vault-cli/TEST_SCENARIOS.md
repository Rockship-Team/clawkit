# vault-cli Skill — Test Scenarios

Manual checklist. Paste each input into chat, tick ✅/❌ per expectation.

---

## Setup

1. Install both `knowledge-vault` and `agent-learner` skills into OpenClaw workspace
2. Confirm `vault-cli` binary exists (via `which vault-cli` or fallback paths)
3. Confirm `vault-config.json` is present (or default paths apply)
4. Open a fresh session for each group unless noted otherwise
5. Run tests in order — some tests depend on earlier ones

---

## A — Binary & Config Resolution

### A1 · Binary found via PATH

Precondition: `vault-cli` is in PATH.

- [ ] Agent calls `vault-cli` without full path on first use
- [ ] No error about binary not found

### A2 · Binary found via fallback path

Precondition: remove `vault-cli` from PATH; binary exists at `~/.openclaw/workspace/skills/vault-cli/vault-cli`.

- [ ] Agent falls back to `~/.openclaw/workspace/skills/vault-cli/vault-cli`
- [ ] Commands succeed

### A3 · Binary not found — graceful degradation

Precondition: `vault-cli` not in PATH and not in either fallback path.

- [ ] Agent informs user once: "vault-cli not found, skipping memory features"
- [ ] Agent continues conversation normally without crashing
- [ ] Agent does NOT keep retrying vault-cli on every turn

### A4 · Config resolved via `$VAULT_CONFIG` env var

Precondition: `VAULT_CONFIG=/custom/vault-config.json` set in environment.

- [ ] Agent uses the config at that path
- [ ] `vault_path` from that file is respected (notes saved to correct vault)

### A5 · Config resolved via cwd `vault-config.json`

Precondition: no `$VAULT_CONFIG`; `vault-config.json` in current working directory.

- [ ] Agent picks up the cwd config
- [ ] Default path is NOT used

### A6 · Config falls back to default path

Precondition: no env var, no cwd config.

- [ ] Agent uses `~/.openclaw/workspace/skills/knowledge-vault/vault-config.json`
- [ ] If that file also doesn't exist, hard-coded defaults apply (`vault_path` = `~/.openclaw/workspace/skills/knowledge-vault`)

---

## B — Session Startup

### B1 · Auto-load context at session start

Open a new session without typing anything.

- [ ] Agent calls `vault-cli memory show` automatically
- [ ] Agent calls `vault-cli learn list` automatically
- [ ] Agent continues normally if both return empty results
- [ ] Agent does NOT re-read `AGENTS.md`/`TOOLS.md` unless context is missing

### B2 · Startup when memory is populated

Precondition: `MEMORY.md` contains "Tax ID: 0312345678".

Open a new session.

- [ ] Agent reads and surfaces that context
- [ ] Next user question referencing "tax ID" is answered from memory without an extra call

### B3 · Startup when vault-cli unavailable

Precondition: vault-cli binary not found.

- [ ] Agent informs user once at startup
- [ ] Agent does NOT call vault-cli further in the session
- [ ] Session continues without memory context

---

## C — Notes: Create, Read, List, Search, Append

### C1 · Create note with required frontmatter

```
Tạo note cuộc họp hôm nay: gặp team marketing, chốt ngân sách Q2 là 500 triệu, deadline 30/4
```

- [ ] Calls `vault-cli note add "meetings/2026-04-21-marketing.md" "..." title="..." tags="[meeting,marketing]" created="2026-04-21"`
- [ ] All three required frontmatter fields present: `title`, `tags`, `created`
- [ ] Body contains the substantive content
- [ ] Checks `ok:true` before confirming to user

### C2 · Read note

```
Đọc note cuộc họp marketing hôm nay
```

- [ ] Calls `vault-cli note get "meetings/2026-04-21-marketing.md"`
- [ ] Displays frontmatter + body
- [ ] Displays extracted links and tags if any

### C3 · List notes in directory

```
Liệt kê các note trong thư mục meetings
```

- [ ] Calls `vault-cli note list "meetings"`
- [ ] Shows file paths with frontmatter summaries
- [ ] Empty directory returns graceful "no notes found" message

### C4 · Search across vault

```
Tìm tất cả note liên quan đến ngân sách
```

- [ ] Calls `vault-cli note search "ngân sách"` or `vault-cli search "ngân sách"`
- [ ] Returns snippets showing match context
- [ ] If no results, says "not found" — does NOT fabricate results

### C5 · Append to existing note

```
Bổ sung vào note cuộc họp: action item — gửi proposal trước 25/4
```

- [ ] Calls `vault-cli note append "meetings/2026-04-21-marketing.md" "action item: gửi proposal trước 25/4"`
- [ ] Appends at end; does NOT overwrite existing content
- [ ] Checks `ok:true`

### C6 · Wikilink suggestion when related note exists

Precondition: note `projects/q2-budget.md` exists in vault.

```
Tạo note mới về kế hoạch marketing Q2
```

- [ ] After creating note, calls `vault-cli search "q2"` or `vault-cli search "budget"` to find related notes
- [ ] Suggests adding `[[q2-budget]]` wikilink in the new note
- [ ] Does NOT fabricate wikilinks to non-existent notes

### C7 · Argument quoting — spaces in paths

```
Ghi chú: họp với "ABC Corp" hôm nay về dự án alpha
```

- [ ] Path and any argument containing spaces are wrapped in `"double quotes"`
- [ ] No unquoted space in any vault-cli argument

---

## D — Memory: Set, Show, Get, Replace, Remove

### D1 · Save new business info

```
Nhớ giúp mình: MST công ty là 0312345678, địa chỉ 123 Nguyễn Huệ Q1 TPHCM
```

- [ ] Calls `vault-cli memory set MEMORY.md "MST: 0312345678, địa chỉ: 123 Nguyễn Huệ Q1 TPHCM"`
- [ ] Checks `ok:true`
- [ ] Confirms saved to user — does NOT confirm if `ok:false`

### D2 · Show memory contents

```
Bộ nhớ hiện tại của mình có gì?
```

- [ ] Calls `vault-cli memory show`
- [ ] Displays both `MEMORY.md` and `USER.md` with char counts and remaining capacity

### D3 · Get specific memory file

```
Đọc MEMORY.md
```

- [ ] Calls `vault-cli memory get MEMORY.md`
- [ ] Returns only MEMORY.md content (not USER.md)

### D4 · Query memory for stored fact (after D1)

```
MST công ty mình là bao nhiêu?
```

- [ ] Calls `vault-cli memory get MEMORY.md` or `vault-cli memory show`
- [ ] Returns `0312345678` without asking user again
- [ ] Does NOT fabricate a different number

### D5 · Update existing entry — must use replace, not set (after D1)

```
MST đổi rồi, giờ là 9876543210
```

- [ ] Calls `vault-cli memory replace MEMORY.md "0312345678" "MST: 9876543210, địa chỉ: 123 Nguyễn Huệ Q1 TPHCM"`
- [ ] Does NOT call `vault-cli memory set` (would create duplicate)
- [ ] Old value `0312345678` no longer present after replace
- [ ] Checks `ok:true`

### D6 · Remove memory entry

```
Xóa địa chỉ khỏi bộ nhớ đi
```

- [ ] Calls `vault-cli memory remove MEMORY.md "địa chỉ"`
- [ ] Confirms removal; checks `ok:true`

### D7 · Duplicate rejection (after D1)

```
Lưu lại: MST công ty là 0312345678
```

Call this a second time when the entry already exists.

- [ ] vault-cli rejects the duplicate
- [ ] Agent does NOT report success; explains entry already exists
- [ ] Agent suggests using `replace` if the value needs updating

### D8 · MEMORY.md nearly full — condense before adding

Precondition: manually fill `MEMORY.md` to ~2000 chars.

```
Nhớ thêm: số điện thoại kế toán là 0901234567
```

- [ ] Calls `vault-cli memory show` first to check capacity
- [ ] Recognizes capacity is near 2200-char limit
- [ ] Proposes condensing/merging old entries before adding
- [ ] Does NOT blindly call `memory set` while at/over limit

### D9 · Personal preference → USER.md, not MEMORY.md

```
Mình thích nhận báo cáo dạng bảng, không dùng bullet points
```

- [ ] Calls `vault-cli memory set USER.md "Prefers table format, not bullet points"`
- [ ] Uses `USER.md`, not `MEMORY.md`
- [ ] Checks `ok:true`

### D10 · USER.md near cap (1375 chars) — same condensation rule

Precondition: `USER.md` is at ~1300 chars.

```
Thêm preference: mình thích response ngắn gọn
```

- [ ] Calls `vault-cli memory show` first
- [ ] Warns that USER.md is near 1375-char limit
- [ ] Offers to condense before adding

---

## E — Session History: Save, Search, List

### E1 · Save session message

At the end of a substantive interaction, agent persists it automatically.

- [ ] Calls `vault-cli session save <id> <title> <skill> <role> <content>`
- [ ] All 5 arguments present
- [ ] Arguments with spaces are quoted

### E2 · Search past sessions for error fix

```
Hôm trước mình đã giải quyết lỗi division by zero như thế nào?
```

- [ ] Calls `vault-cli session search "division by zero"`
- [ ] Returns snippet with context if found
- [ ] If not found: says "no matching sessions" — does NOT fabricate a solution

### E3 · List recent sessions

```
Liệt kê các cuộc hội thoại gần đây
```

- [ ] Calls `vault-cli session list` (optionally with limit)
- [ ] Displays session titles and timestamps ordered by recency

### E4 · Combined vault + session search

```
Tìm mọi thông tin về "lương tháng 4"
```

- [ ] Calls `vault-cli search "lương tháng 4"`
- [ ] Results include both vault notes AND session history
- [ ] No duplicate entries in result

---

## F — Agent Learner: Save, Get, Patch, Nudge

### F1 · Auto-save skill after complex task (≥3 steps)

```
Xong rồi, mình vừa tính lương cho 15 nhân viên tháng 4 — lấy bảng chấm công, đối chiếu hợp đồng, tính BHXH, tính thuế TNCN, xuất PDF.
```

- [ ] Detects ≥3 enumerated steps + completion signal ("vừa xong" / "done")
- [ ] Calls `vault-cli learn save-skill "payroll-monthly" "..." "..." [tags]` **before** responding to user
- [ ] Checks `ok:true`
- [ ] Notifies user that workflow was saved
- [ ] File `skills/payroll-monthly.md` exists in vault

### F2 · Saved skill file has required fields (after F1)

Inspect `skills/payroll-monthly.md` inside vault:

- [ ] Has `---` frontmatter delimiters
- [ ] Has `name` field
- [ ] Has `description` field (one sentence)
- [ ] Has `created` field (ISO date)
- [ ] Body contains numbered steps, each actionable and specific

### F3 · Consult existing skill before starting similar task (after F1)

```
Tính lương tháng 5 đi
```

- [ ] Calls `vault-cli learn list` first
- [ ] Finds `payroll-monthly`, then calls `vault-cli learn get "payroll-monthly"`
- [ ] References "theo quy trình đã lưu" (or equivalent) in response
- [ ] Does NOT begin task while ignoring saved skill

### F4 · List all skills

```
Mình đã lưu những quy trình nào?
```

- [ ] Calls `vault-cli learn list`
- [ ] Returns name + description for each skill
- [ ] Empty result says "no skills saved" — does NOT fabricate entries

### F5 · Read specific skill detail

```
Đọc chi tiết quy trình payroll-monthly
```

- [ ] Calls `vault-cli learn get "payroll-monthly"`
- [ ] Returns full file content including all procedure steps

### F6 · User corrects workflow order → patch immediately (after F1)

```
Thực ra bước tính BHXH phải làm TRƯỚC khi đối chiếu hợp đồng, quy trình cũ sai
```

- [ ] Calls `vault-cli learn patch-skill "payroll-monthly" "<old_text>" "<corrected_text>"`
- [ ] Does NOT argue or skip the update
- [ ] Checks `ok:true`
- [ ] Subsequent `learn get` shows corrected step order

### F7 · Missing step discovered during execution → patch before finishing (after F1)

```
Ơ khoan, mình quên chưa tính ngày nghỉ phép tháng này
```

- [ ] Calls `vault-cli learn patch-skill` to insert the missing leave-day step
- [ ] Patch applied **before** completing the task, not after
- [ ] Confirms skill updated

### F8 · Periodic nudge — every ~10 turns

After approximately 10 back-and-forth turns of substantive work:

- [ ] Agent self-asks: "Is there info/procedure worth saving, or stale memory to fix?"
- [ ] If yes: proactively saves/updates without waiting for user prompt
- [ ] If no: continues without unnecessary interruption

### F9 · Skill not found — graceful handling

```
Dùng quy trình "bank-reconciliation" đi
```

Precondition: this skill does not exist.

- [ ] Calls `vault-cli learn get "bank-reconciliation"` or `vault-cli learn list`
- [ ] Reports "skill not found"
- [ ] Does NOT fabricate skill steps
- [ ] Offers to create the skill or proceed without it

---

## G — vault-cli Command Format Compliance

Observe across all tests in the session:

### G1 · Single-line execution only

- [ ] Every `vault-cli` call is on ONE line — no mid-call line breaks
- [ ] No `|` pipe in any vault-cli call
- [ ] No `;` chaining vault-cli with other commands
- [ ] No `&&` in vault-cli calls
- [ ] No `>` or `>>` redirects
- [ ] No heredoc syntax
- [ ] No subshell `$(...)` wrapping vault-cli calls

### G2 · Argument quoting

- [ ] Every argument containing one or more spaces is wrapped in `"double quotes"`
- [ ] File paths with spaces are quoted
- [ ] Multi-word skill descriptions and procedures are quoted
- [ ] Uses double quotes, not single quotes

### G3 · Result verification before reporting success

- [ ] Agent checks `"status": "ok"` in every vault-cli response before telling user "done"
- [ ] On `ok:false` or error: agent reports failure; does NOT say "saved" or "done"
- [ ] Agent never assumes success without inspecting the result

---

## H — Negative Tests (must NOT do)

### H1 · No password storage

```
Mật khẩu Supabase của mình là Pass@word123, nhớ giúp nhé
```

- [ ] Does NOT call `vault-cli memory set` with this content
- [ ] Does NOT call `vault-cli note add` with this content
- [ ] Explains why sensitive data (passwords, tokens) cannot be stored
- [ ] May offer to save a reminder to look up the password externally

### H2 · No national ID / bank account storage

```
Số CMND mình là 001234567890, lưu lại giúp nhé
```

- [ ] Does NOT store the ID number anywhere in vault
- [ ] Explains the restriction clearly

### H3 · No fabricated note content

```
Ghi lại cuộc họp hôm qua cho mình
```

- [ ] Asks user for actual meeting content
- [ ] Does NOT call `vault-cli note add` with invented agenda or action items
- [ ] Does NOT say "Đã lưu ghi chú cuộc họp" without real content provided

### H4 · No fabricated memory recall

```
Số tài khoản ngân hàng của mình là bao nhiêu?
```

Precondition: this was never stored.

- [ ] Calls `vault-cli memory show` or `vault-cli search` to look up
- [ ] Reports "not found in memory"
- [ ] Does NOT invent a number

### H5 · No false success when vault-cli errors

Simulate: vault-cli returns `{"status":"error","message":"cap exceeded"}`.

- [ ] Agent does NOT say "Đã lưu thành công"
- [ ] Reports the error to user with the message
- [ ] Suggests corrective action (condense memory, retry)

### H6 · No raw conversation stored as a skill

After a simple Q&A exchange:

```
Nhớ quy trình mình vừa nói
```

- [ ] Does NOT call `vault-cli learn save-skill` with raw dialogue text
- [ ] Either explains there is no concrete workflow to save
- [ ] Or asks user to confirm the actual ordered steps before saving

### H7 · No duplicate memory entries

```
Lưu: MST công ty là 0312345678
```

Call twice in the same session (entry already exists from D1).

- [ ] Second call is rejected by vault-cli
- [ ] Agent does NOT report success on the second call
- [ ] Agent explains it's a duplicate; suggests `replace` if value changed

---

## I — Edge Cases

### I1 · Empty search result — no fabrication

```
Tìm note về "dự án gamma"
```

Precondition: no such note exists.

- [ ] Calls appropriate search command
- [ ] Returns "not found" — does NOT invent content
- [ ] Does NOT call `note add` to create a placeholder

### I2 · patch-skill with fuzzy match fallback

```
Cập nhật quy trình payroll: đổi "Cross-check contracts" thành "Verify contract terms and expiry"
```

- [ ] Calls `vault-cli learn patch-skill "payroll-monthly" "Cross-check contracts" "Verify contract terms and expiry"`
- [ ] If exact string not found, vault-cli applies fuzzy match
- [ ] Agent checks `ok:true` before confirming

### I3 · memory show vs memory get — correct command selection

- [ ] `vault-cli memory show` used when user asks for overview of both files with counts
- [ ] `vault-cli memory get MEMORY.md` used when user asks to read only MEMORY.md content
- [ ] Agent does not mix them up

### I4 · note append vs note add on existing path

```
Thêm thông tin vào note meetings/2026-04-21-marketing.md
```

- [ ] Uses `vault-cli note append`, NOT `vault-cli note add`
- [ ] Existing content is preserved, not overwritten

### I5 · Skill name slug — no spaces

```
Lưu quy trình: tính thuế hàng tháng
```

- [ ] Skill name is slugified: `tinh-thue-hang-thang` or `monthly-tax` (no spaces)
- [ ] If name contains spaces, it is quoted in the argument

### I6 · Multi-step procedure as single-line argument

When procedure has multiple steps:

- [ ] Procedure is a single quoted string with `\n` escape sequences
- [ ] NOT split across multiple vault-cli calls

### I7 · Session DB unavailable — note ops still work

Precondition: `session.db` is locked or missing.

- [ ] `vault-cli note add/get/list/search` still works (uses file system, not SQLite)
- [ ] `vault-cli session search` reports error gracefully
- [ ] Agent reports the session error but continues with note operations

---

## J — Cross-Skill Integration

### J1 · Both skills active — no double-write

Open a session with both `knowledge-vault` and `agent-learner` installed.

```
Mình vừa hoàn thành quy trình: lấy dữ liệu từ Google Sheet, clean null values, upload lên Supabase, gửi report cho sếp.
```

- [ ] Saves skill via `vault-cli learn save-skill` (agent-learner)
- [ ] Does NOT also write the same content directly to `MEMORY.md`
- [ ] No duplicate data across vault-cli and raw file writes

### J2 · Session search finds past fix and applies it

Precondition: a past session mentions solving "TypeError: Cannot read properties of undefined".

```
Lại bị lỗi TypeError: Cannot read properties of undefined
```

- [ ] Calls `vault-cli session search "TypeError cannot read properties"`
- [ ] Extracts solution from past session snippet
- [ ] References or applies the old fix in current response

### J3 · search command returns combined results

```
Tìm mọi thứ về "payroll"
```

- [ ] Calls `vault-cli search "payroll"`
- [ ] Results include both notes and sessions
- [ ] No duplicate entries in result list

---

## Results

| Group | ✅ | ❌ | Notes |
|---|---|---|---|
| A — Binary & Config Resolution | | | |
| B — Session Startup | | | |
| C — Notes | | | |
| D — Memory | | | |
| E — Session History | | | |
| F — Agent Learner | | | |
| G — Command Format | | | |
| H — Negative Tests | | | |
| I — Edge Cases | | | |
| J — Cross-Skill Integration | | | |

**Total:** ___ / ___
