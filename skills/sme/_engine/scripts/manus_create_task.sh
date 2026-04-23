#!/usr/bin/env bash
# Create Manus AI task for PDF generation
# Usage: echo '<json>' | scripts/manus_create_task.sh

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

if [ -z "$API_KEY" ]; then
  echo "ERROR: MANUS_API_KEY not set" >&2
  exit 1
fi

# Read JSON from stdin
JSON_BODY=$(cat)

# Create task
RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/tasks" \
  -H "API_KEY: $API_KEY" \
  -H "Content-Type: application/json" \
  -d "$JSON_BODY")

# Extract task_id (API returns "task_id")
TASK_ID=$(echo "$RESPONSE" | jq -r '.task_id // .id // empty')

if [ -z "$TASK_ID" ]; then
  echo "ERROR: Could not create task. Response: $RESPONSE" >&2
  exit 1
fi

echo "$TASK_ID"
echo "$RESPONSE" | jq .
