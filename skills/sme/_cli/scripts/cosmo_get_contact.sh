#!/usr/bin/env bash
# Get contact details by ID
# Usage: cosmo_get_contact.sh <contact_id>

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../.env" 2>/dev/null || true

: "${COSMO_API_KEY:?COSMO_API_KEY is not set}"
: "${COSMO_BASE_URL:?COSMO_BASE_URL is not set}"

CONTACT_ID="${1:?Usage: cosmo_get_contact.sh <contact_id>}"

curl -s -H "Authorization: Bearer $COSMO_API_KEY" \
  -H "Content-Type: application/json" \
  -X GET "$COSMO_BASE_URL/v1/contacts/$CONTACT_ID"
