#!/usr/bin/env bash
# Poll Manus AI task until complete
# Usage: scripts/manus_get_task.sh <task_id>

# Auto-load .env from skill root
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SKILL_DIR="$(dirname "$SCRIPT_DIR")"
if [ -f "$SKILL_DIR/.env" ]; then
  set -a
  source "$SKILL_DIR/.env"
  set +a
fi

TASK_ID="$1"
API_KEY="${MANUS_API_KEY:-}"
BASE_URL="${MANUS_BASE_URL:-https://api.manus.ai}"
MAX_RETRIES=30
RETRY_INTERVAL=10

if [ -z "$TASK_ID" ]; then
  echo "Usage: scripts/manus_get_task.sh <task_id>" >&2
  exit 1
fi

if [ -z "$API_KEY" ]; then
  echo "ERROR: MANUS_API_KEY not set" >&2
  exit 1
fi

RETRIES=0
while [ $RETRIES -lt $MAX_RETRIES ]; do
  # Get task status
  RESPONSE=$(curl -s -X GET "${BASE_URL}/v1/tasks/$TASK_ID" \
    -H "API_KEY: $API_KEY" \
    -H "Content-Type: application/json")

  STATUS=$(echo "$RESPONSE" | jq -r '.status // "unknown"')

  echo "Status: $STATUS"

  if [ "$STATUS" = "completed" ]; then
    echo "Task completed!"
    echo "$RESPONSE" | jq .
    
    # Extract PDF URL if available
    PDF_URL=$(echo "$RESPONSE" | jq -r '.output[]?.content[]?.fileUrl // empty')
    if [ -n "$PDF_URL" ]; then
      echo ""
      echo "📄 PDF Download URL: $PDF_URL"
    fi
    
    exit 0
  fi

  if [ "$STATUS" = "failed" ]; then
    echo "ERROR: Task failed!" >&2
    echo "$RESPONSE" | jq .
    exit 1
  fi

  # Wait before retry
  echo "Waiting ${RETRY_INTERVAL}s... ($((RETRIES + 1))/$MAX_RETRIES)"
  sleep $RETRY_INTERVAL
  RETRIES=$((RETRIES + 1))
done

echo "ERROR: Task timed out after ${MAX_RETRIES} retries" >&2
exit 1
