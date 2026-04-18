#!/usr/bin/env bash
# Search people in Apollo.io by company
# Usage: apollo_search_people.sh <company_name> [seniorities]

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../.env" 2>/dev/null || true

: "${APOLLO_IO_API_KEY:?APOLLO_IO_API_KEY is not set}"

COMPANY_NAME="${1:?Usage: apollo_search_people.sh <company_name> [seniorities]}"
SENIORITIES="${2:-c_suite,vp}"

curl -s "https://api.apollo.io/v1/people/search?q=${COMPANY_NAME}&seniorities=${SENIORITIES}" \
  -H "Authorization: Bearer $APOLLO_IO_API_KEY"
