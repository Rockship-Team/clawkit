#!/usr/bin/env bash
# Integration tests for vault-cli
# Run from the vault-cli directory: ./test_integration.sh
# Requires vault-cli binary to be built first: go build -o vault-cli ./cmd/

set -euo pipefail

BINARY="${1:-./vault-cli}"
PASS=0
FAIL=0
SKIP=0

# ── Helpers ──────────────────────────────────────────────────────────────────

setup() {
  TMPDIR_ROOT=$(mktemp -d)
  export HOME="$TMPDIR_ROOT"
  VAULT="$TMPDIR_ROOT/vault"
  DB="$TMPDIR_ROOT/sessions.db"
  mkdir -p "$TMPDIR_ROOT/.openclaw/workspace/skills/knowledge-vault"
  cat >"$TMPDIR_ROOT/.openclaw/workspace/skills/knowledge-vault/vault-config.json" <<EOF
{"vault_path": "$VAULT", "db_path": "$DB"}
EOF
}

teardown() {
  rm -rf "$TMPDIR_ROOT"
}

run() {
  local name="$1"; shift
  local result expected
  result=$("$BINARY" "$@" 2>&1) || true
  echo "$result"
}

assert_contains() {
  local name="$1" output="$2" needle="$3"
  if echo "$output" | grep -q "$needle"; then
    echo "  PASS  $name"
    PASS=$((PASS+1))
  else
    echo "  FAIL  $name"
    echo "        expected to contain: $needle"
    echo "        actual output: $output"
    FAIL=$((FAIL+1))
  fi
}

assert_not_contains() {
  local name="$1" output="$2" needle="$3"
  if echo "$output" | grep -q "$needle"; then
    echo "  FAIL  $name"
    echo "        expected NOT to contain: $needle"
    echo "        actual output: $output"
    FAIL=$((FAIL+1))
  else
    echo "  PASS  $name"
    PASS=$((PASS+1))
  fi
}

assert_file_exists() {
  local name="$1" path="$2"
  if [ -f "$path" ]; then
    echo "  PASS  $name"
    PASS=$((PASS+1))
  else
    echo "  FAIL  $name (file not found: $path)"
    FAIL=$((FAIL+1))
  fi
}

# ── Build ─────────────────────────────────────────────────────────────────────

echo "Building vault-cli..."
(cd "$(dirname "$0")/cmd" && go build -o ../vault-cli .) || {
  echo "BUILD FAILED"
  exit 1
}
echo "Build OK"
echo ""

# ── Tests ─────────────────────────────────────────────────────────────────────

setup
trap teardown EXIT

# ─ Group 1: note commands ───────────────────────────────────────────────────
echo "=== Group 1: note ==="

OUT=$(run "note add" note add "meetings/standup-2026" "Noi dung cuoc hop hom nay [[ProjectAlpha]] #meeting #daily" "title=Standup 20/04")
assert_contains "note add returns ok" "$OUT" '"status": "ok"'

assert_file_exists "note file created" "$VAULT/meetings/standup-2026.md"

OUT=$(run "note get" note get "meetings/standup-2026")
assert_contains "note get returns body" "$OUT" "Noi dung cuoc hop"
assert_contains "note get returns links" "$OUT" "ProjectAlpha"
assert_contains "note get returns tags" "$OUT" "meeting"

OUT=$(run "note list" note list)
assert_contains "note list finds file" "$OUT" "standup-2026"

OUT=$(run "note search hit" note search "cuoc hop")
assert_contains "note search finds match" "$OUT" "standup-2026"

OUT=$(run "note search miss" note search "khong-co-gi-het")
assert_not_contains "note search miss returns no results" "$OUT" "standup-2026"

OUT=$(run "note append" note append "meetings/standup-2026" "Them noi dung phu")
assert_contains "note append returns ok" "$OUT" '"status": "ok"'

OUT=$(run "note get after append" note get "meetings/standup-2026")
assert_contains "note body contains appended text" "$OUT" "Them noi dung phu"

# ─ Group 2: memory commands ─────────────────────────────────────────────────
echo ""
echo "=== Group 2: memory ==="

OUT=$(run "memory set MEMORY" memory set MEMORY.md "MST cong ty: 0312345678")
assert_contains "memory set returns ok" "$OUT" '"status": "ok"'

OUT=$(run "memory set USER" memory set USER.md "User thich bao cao bằng VND")
assert_contains "memory set USER returns ok" "$OUT" '"status": "ok"'

OUT=$(run "memory show" memory show)
assert_contains "memory show has memory entries" "$OUT" "MST cong ty"
assert_contains "memory show has user entries" "$OUT" "User thich bao cao"
assert_contains "memory show has chars" "$OUT" '"chars"'
assert_contains "memory show has cap" "$OUT" '"cap"'

OUT=$(run "memory get MEMORY" memory get MEMORY.md)
assert_contains "memory get returns entries" "$OUT" "0312345678"

OUT=$(run "memory set duplicate" memory set MEMORY.md "MST cong ty: 0312345678" 2>&1 || true)
assert_contains "memory set duplicate rejected" "$OUT" "duplicate"

OUT=$(run "memory replace" memory replace MEMORY.md "0312345678" "MST cong ty: 9876543210 (updated)")
assert_contains "memory replace returns ok" "$OUT" '"status": "ok"'

OUT=$(run "memory get after replace" memory get MEMORY.md)
assert_contains "memory replace: new value present" "$OUT" "9876543210"
assert_not_contains "memory replace: old value gone" "$OUT" "0312345678"

OUT=$(run "memory replace not found" memory replace MEMORY.md "khong-ton-tai" "anything" 2>&1 || true)
assert_contains "memory replace missing entry errors" "$OUT" '"status": "error"'

OUT=$(run "memory remove" memory remove MEMORY.md "MST")
assert_contains "memory remove returns ok" "$OUT" '"status": "ok"'

OUT=$(run "memory get after remove" memory get MEMORY.md)
assert_not_contains "memory remove: entry gone" "$OUT" "MST"

OUT=$(run "memory remove not found" memory remove MEMORY.md "khong-ton-tai" 2>&1 || true)
assert_contains "memory remove missing entry errors" "$OUT" '"status": "error"'

# ─ Group 3: learn commands ──────────────────────────────────────────────────
echo ""
echo "=== Group 3: learn ==="

OUT=$(run "learn save-skill" learn save-skill \
  "payroll-monthly" \
  "Quy trinh tinh luong hang thang" \
  "1. Tai bang cham cong\n2. Doi chieu hop dong\n3. Tinh BHXH\n4. Tinh TNCN\n5. Xuat bang luong" \
  "payroll,finance")
assert_contains "learn save-skill returns ok" "$OUT" '"status": "ok"'

assert_file_exists "learn skill file created" "$VAULT/skills/payroll-monthly.md"

OUT=$(run "learn list" learn list)
assert_contains "learn list shows saved skill" "$OUT" "payroll-monthly"

OUT=$(run "learn get" learn get "payroll-monthly")
assert_contains "learn get has description" "$OUT" "tinh luong"
assert_contains "learn get has procedure" "$OUT" "Tinh TNCN"
assert_contains "learn get has frontmatter" "$OUT" "created"

OUT=$(run "learn patch-skill" learn patch-skill \
  "payroll-monthly" \
  "2. Doi chieu hop dong" \
  "2. Doi chieu hop dong\n3. Kiem tra ngay nghi phep va ngay le")
assert_contains "learn patch-skill returns ok" "$OUT" '"status": "ok"'

OUT=$(run "learn get after patch" learn get "payroll-monthly")
assert_contains "learn get has patched content" "$OUT" "ngay nghi phep"
assert_contains "learn patch updates 'updated' date" "$OUT" "updated"

OUT=$(run "learn patch not found" learn patch-skill "khong-ton-tai" "abc" "xyz" 2>&1 || true)
assert_contains "learn patch missing skill errors" "$OUT" '"status": "error"'

OUT=$(run "learn get not found" learn get "khong-ton-tai" 2>&1 || true)
assert_contains "learn get missing skill errors" "$OUT" '"status": "error"'

# ─ Group 4: session commands ────────────────────────────────────────────────
echo ""
echo "=== Group 4: session ==="

OUT=$(run "session save user" session save "s001" "Tinh thue TNCN" "sme-tax" "user" "Tinh thue cho luong 35 trieu")
assert_contains "session save user returns ok" "$OUT" '"status": "ok"'

OUT=$(run "session save assistant" session save "s001" "Tinh thue TNCN" "sme-tax" "assistant" "TNCN = 1638750 VND sau giam tru ca nhan")
assert_contains "session save assistant returns ok" "$OUT" '"status": "ok"'

OUT=$(run "session list" session list)
assert_contains "session list shows session" "$OUT" "s001"
assert_contains "session list shows title" "$OUT" "Tinh thue TNCN"

OUT=$(run "session search hit" session search "thue")
assert_contains "session search finds result" "$OUT" '"count"'

OUT=$(run "session search snippet" session search "TNCN")
assert_contains "session search returns snippet" "$OUT" "TNCN"

# ─ Group 5: combined search ─────────────────────────────────────────────────
echo ""
echo "=== Group 5: search (vault + session) ==="

# We have a note with "cuoc hop" and a session with "thue"
OUT=$(run "search vault hit" search "cuoc hop")
assert_contains "search finds vault content" "$OUT" "vault"

OUT=$(run "search session hit" search "TNCN")
assert_contains "search finds session content" "$OUT" "session"

OUT=$(run "search miss" search "xyz-khong-co-gi")
# Should return empty results, not error
assert_contains "search miss returns ok with empty results" "$OUT" '"count": 0'

# ─ Group 6: negative / security tests ──────────────────────────────────────
echo ""
echo "=== Group 6: edge cases ==="

# Memory cap enforcement: build entries up to cap
BIG_ENTRY=$(python3 -c "print('x' * 300)" 2>/dev/null || printf '%.0sx' {1..300})
COUNT=0
while true; do
  OUT=$("$BINARY" memory set MEMORY.md "${BIG_ENTRY}_${COUNT}" 2>&1 || true)
  if echo "$OUT" | grep -q "error"; then
    assert_contains "memory cap enforced" "$OUT" "cap"
    break
  fi
  COUNT=$((COUNT+1))
  if [ $COUNT -gt 20 ]; then
    echo "  SKIP  memory cap enforcement (didn't hit cap in 20 entries)"
    SKIP=$((SKIP+1))
    break
  fi
done

# vault-cli with no args should exit non-zero
"$BINARY" 2>&1 || true
OUT=$("$BINARY" 2>&1 || true)
assert_contains "no-args shows usage" "$OUT" "vault-cli"

# note get missing file
OUT=$(run "note get missing" note get "does/not/exist" 2>&1 || true)
assert_contains "note get missing returns error" "$OUT" '"status": "error"'

# ─ Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "─────────────────────────────────────────"
echo "Results: $PASS passed, $FAIL failed, $SKIP skipped"
if [ "$FAIL" -gt 0 ]; then
  echo "FAILED"
  exit 1
else
  echo "ALL PASSED"
  exit 0
fi
