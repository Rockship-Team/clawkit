#!/usr/bin/env bash
# Search company in Apollo.io
# Usage: apollo_search_company.sh <company_name>

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../.env" 2>/dev/null || true

: "${APOLLO_IO_API_KEY:?APOLLO_IO_API_KEY is not set}"

COMPANY_NAME="${1:?Usage: apollo_search_company.sh <company_name>}"

curl -s "https://api.apollo.io/v1/companies/follow?q=${COMPANY_NAME}" \
  -H "Authorization: Bearer $APOLLO_IO_API_KEY"
