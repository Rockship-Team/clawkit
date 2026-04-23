#!/usr/bin/env bash
# Search for contact in COSMO CRM
# Usage: cosmo_search_contact.sh <query> [page_size]

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../.env" 2>/dev/null || true

: "${COSMO_API_KEY:?COSMO_API_KEY is not set}"
: "${COSMO_BASE_URL:?COSMO_BASE_URL is not set}"

QUERY="${1:?Usage: cosmo_search_contact.sh <query> [page_size]}"
PAGESIZE="${2:-25}"

curl -s -H "Authorization: Bearer $COSMO_API_KEY" \
  -H "Content-Type: application/json" \
  -X POST "$COSMO_BASE_URL/v2/contacts/search" \
  -d "{\"query\": \"$QUERY\", \"pageSize\": $PAGESIZE}" | jq -r '.data.list[] | "\(.entity.name) | \(.entity.company) | \(.entity.job_title)"'
