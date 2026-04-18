#!/usr/bin/env bash
# Enrich person data from Apollo.io
# Usage: apollo_enrich_person.sh <name> <company_name>

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../.env" 2>/dev/null || true

: "${APOLLO_IO_API_KEY:?APOLLO_IO_API_KEY is not set}"

NAME="${1:?Usage: apollo_enrich_person.sh <name> <company_name>}"
COMPANY_NAME="${2:?Usage: apollo_enrich_person.sh <name> <company_name>}"

curl -s "https://api.apollo.io/v1/people/enrich?first_name=$(echo "$NAME" | cut -d' ' -f1)&last_name=$(echo "$NAME" | cut -d' ' -f2-)&company_domain=${COMPANY_NAME}" \
  -H "Authorization: Bearer $APOLLO_IO_API_KEY"
