#!/usr/bin/env bash
# Log interaction to COSMO CRM
# Usage: cosmo_log_interaction.sh <contact_id> <type>

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../.env" 2>/dev/null || true

: "${COSMO_API_KEY:?COSMO_API_KEY is not set}"
: "${COSMO_BASE_URL:?COSMO_BASE_URL is not set}"

CONTACT_ID="${1:?Usage: cosmo_log_interaction.sh <contact_id> <type>}"
INTERACTION_TYPE="${2:?Usage: cosmo_log_interaction.sh <contact_id> <type>}"

curl -s -H "Authorization: Bearer $COSMO_API_KEY" \
  -H "Content-Type: application/json" \
  -X POST "$COSMO_BASE_URL/v1/contacts/$CONTACT_ID/interactions" \
  -d "{\"type\": \"$INTERACTION_TYPE\", \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\", \"created_by\": \"system\"}"
