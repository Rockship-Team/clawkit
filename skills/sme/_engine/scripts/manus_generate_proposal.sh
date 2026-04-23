#!/usr/bin/env bash
# ⛔ DO NOT MODIFY THIS SCRIPT. It is tested and working.
# All-in-one: Encode style template + create Manus task + poll for PDF
# Usage: scripts/manus_generate_proposal.sh <company_name> <outline_file>
# Example: scripts/manus_generate_proposal.sh "Heineken_Vietnam" /tmp/outline.md

set -euo pipefail

# Auto-load .env from skill root
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SKILL_DIR="$(dirname "$SCRIPT_DIR")"
if [ -f "$SKILL_DIR/.env" ]; then
  set -a
  source "$SKILL_DIR/.env"
  set +a
fi

API_KEY="${MANUS_API_KEY:-}"
BASE_URL="${MANUS_BASE_URL:-https://api.manus.ai}"
STYLE_TEMPLATE="$SKILL_DIR/assets/templates/style_template.pdf"
COMPANY="${1:?Usage: scripts/manus_generate_proposal.sh <company_name> <outline_file>}"
OUTLINE_FILE="${2:-}"

if [ -z "$API_KEY" ]; then
  echo "ERROR: MANUS_API_KEY not set" >&2
  exit 1
fi

if [ ! -f "$STYLE_TEMPLATE" ]; then
  echo "ERROR: Style template not found at $STYLE_TEMPLATE" >&2
  exit 1
fi

# Read outline from file or stdin
if [ -n "$OUTLINE_FILE" ] && [ -f "$OUTLINE_FILE" ]; then
  OUTLINE=$(cat "$OUTLINE_FILE")
elif [ -n "$OUTLINE_FILE" ]; then
  echo "ERROR: Outline file not found: $OUTLINE_FILE" >&2
  exit 1
else
  OUTLINE=$(cat)
fi

if [ -z "$OUTLINE" ]; then
  echo "ERROR: No outline provided." >&2
  exit 1
fi

echo "=== Step 1: Encoding style_template.pdf ==="
echo "File: $STYLE_TEMPLATE"

# Build JSON using Python for safe escaping
# ⛔ The prompt below is HARDCODED. Do not change it.
JSON_BODY=$(python3 - "$STYLE_TEMPLATE" "$OUTLINE_FILE" "$COMPANY" << 'PYEOF'
import json, base64, sys

style_path = sys.argv[1]
outline_path = sys.argv[2]
company = sys.argv[3]

with open(style_path, 'rb') as f:
    style_b64 = base64.b64encode(f.read()).decode()

with open(outline_path, 'r') as f:
    outline = f.read()

prompt = (
    "Dựa trên outline dưới đây, tạo 1 bản PDF proposal format đẹp như "
    "1 bài thuyết trình. Style giống y chang file style_template.pdf "
    "đính kèm nha — giữ đúng màu sắc, layout, font chữ, card design. "
    "Không dùng HTML convert PDF, làm slide-style cho đẹp.\n\n"
    "Outline:\n\n" + outline + "\n\n"
    "Output: 1 file PDF tên " + company + "_proposal.pdf"
)

task = {
    "prompt": prompt,
    "agentProfile": "manus-1.6",
    "attachments": [{
        "type": "file",
        "file_name": "style_template.pdf",
        "file_data": "data:application/pdf;base64," + style_b64
    }]
}

print(json.dumps(task, ensure_ascii=False))
PYEOF
)

echo "JSON built."

echo ""
echo "=== Step 2: Creating Manus task ==="

# Write JSON to temp file to avoid shell arg length limits (base64 can be large)
JSON_TMP=$(mktemp)
echo "$JSON_BODY" > "$JSON_TMP"

# ⛔ Auth header MUST be "API_KEY:" — NOT "Authorization: Bearer"
RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/tasks" \
  -H "API_KEY: $API_KEY" \
  -H "Content-Type: application/json" \
  -d @"$JSON_TMP")

rm -f "$JSON_TMP"

TASK_ID=$(echo "$RESPONSE" | jq -r '.task_id // .id // empty')

if [ -z "$TASK_ID" ]; then
  echo "ERROR: Could not create task. Response:" >&2
  echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE" >&2
  exit 1
fi

TASK_URL=$(echo "$RESPONSE" | jq -r '.task_url // empty')
echo "Task created!"
echo "  Task ID: $TASK_ID"
echo "  Task URL: $TASK_URL"

echo ""
echo "=== Step 3: Polling for completion ==="

MAX_RETRIES=30
RETRY_INTERVAL=10
RETRIES=0

while [ $RETRIES -lt $MAX_RETRIES ]; do
  # ⛔ Auth header MUST be "API_KEY:" — NOT "Authorization: Bearer"
  POLL_RESPONSE=$(curl -s -X GET "${BASE_URL}/v1/tasks/${TASK_ID}" \
    -H "API_KEY: $API_KEY")

  STATUS=$(echo "$POLL_RESPONSE" | jq -r '.status // "unknown"')

  if [ "$STATUS" = "completed" ]; then
    echo "Task completed!"
    echo ""

    PDF_URL=$(echo "$POLL_RESPONSE" | jq -r '
      .output[]? | .content[]? |
      select(.mimeType == "application/pdf" or (.fileName // "" | endswith(".pdf"))) |
      .fileUrl // empty' | head -1)

    if [ -n "$PDF_URL" ]; then
      echo "=== RESULT ==="
      echo "PDF URL: $PDF_URL"
      echo "Task URL: $TASK_URL"
    else
      echo "=== RESULT ==="
      echo "Task completed but no PDF URL found."
      echo "Check task manually: $TASK_URL"
      echo "Raw output:"
      echo "$POLL_RESPONSE" | jq '.output'
    fi
    exit 0
  fi

  if [ "$STATUS" = "failed" ]; then
    echo "ERROR: Task failed!" >&2
    echo "$POLL_RESPONSE" | jq . 2>/dev/null || echo "$POLL_RESPONSE" >&2
    exit 1
  fi

  RETRIES=$((RETRIES + 1))
  echo "  Status: $STATUS — waiting ${RETRY_INTERVAL}s ($RETRIES/$MAX_RETRIES)"
  sleep $RETRY_INTERVAL
done

echo "ERROR: Task timed out after $((MAX_RETRIES * RETRY_INTERVAL))s" >&2
echo "Check manually: $TASK_URL" >&2
exit 1
