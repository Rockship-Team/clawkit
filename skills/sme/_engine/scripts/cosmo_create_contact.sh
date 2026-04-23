#!/usr/bin/env bash
# Create new contact in COSMO CRM
# Usage: echo '{"name": "John", "company": "ABC"}' | cosmo_create_contact.sh

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../.env" 2>/dev/null || true

: "${COSMO_API_KEY:?COSMO_API_KEY is not set}"
: "${COSMO_BASE_URL:?COSMO_BASE_URL is not set}"

curl -s -H "Authorization: Bearer $COSMO_API_KEY" \
  -H "Content-Type: application/json" \
  -X POST "$COSMO_BASE_URL/v1/contacts" \
  -d @-
