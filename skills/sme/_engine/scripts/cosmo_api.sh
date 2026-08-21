#!/usr/bin/env bash
# Universal COSMO API caller with auto token refresh
# Usage: cosmo_api.sh <method> <path> [json_body]
#
# Examples:
#   cosmo_api.sh GET /v1/events
#   cosmo_api.sh POST /v2/contacts/search '{"query":"acme"}'
#   cosmo_api.sh POST /v1/campaigns '{"name":"Q2 Outreach","playbook":"cold_outreach"}'
#   cosmo_api.sh PATCH /v1/contacts/UUID '{"business_stage":"QUALIFIED"}'
#   cosmo_api.sh DELETE /v1/contacts/UUID

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="$SCRIPT_DIR/../.env"
source "$ENV_FILE" 2>/dev/null || true

: "${COSMO_BASE_URL:?COSMO_BASE_URL is not set}"
: "${COSMO_AUTH_EMAIL:?COSMO_AUTH_EMAIL is not set}"

# --- Token refresh logic ---
is_token_expired() {
  local token="$1"
  # Personal API keys (p_api_key_*) never expire
  if [[ "$token" == p_api_key_* ]]; then
    return 1
  fi
  # Extract payload (second part of JWT)
  local payload
  payload=$(echo "$token" | cut -d. -f2)
  # Add base64 padding if needed
  local remainder=$(( ${#payload} % 4 ))
  if [ "$remainder" -eq 2 ]; then
    payload="${payload}=="
  elif [ "$remainder" -eq 3 ]; then
    payload="${payload}="
  fi
  # Decode and extract exp
  local exp
  exp=$(echo "$payload" | base64 -d 2>/dev/null | jq -r '.exp // 0' 2>/dev/null)
  if [ -z "$exp" ] || [ "$exp" = "0" ] || [ "$exp" = "null" ]; then
    return 0  # treat as expired if we can't parse
  fi
  local now
  now=$(date +%s)
  # Expired if less than 60s remaining (buffer)
  [ "$now" -ge $(( exp - 60 )) ]
}

refresh_token() {
  local response
  response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$COSMO_AUTH_EMAIL\"}" \
    "$COSMO_BASE_URL/v1/auth/login")

  local status
  status=$(echo "$response" | jq -r '.status // empty' 2>/dev/null)
  if [ "$status" != "success" ]; then
    echo "ERROR: Failed to refresh COSMO token. Response:" >&2
    echo "$response" | jq . >&2 2>/dev/null || echo "$response" >&2
    exit 1
  fi

  local new_token
  new_token=$(echo "$response" | jq -r '.data.token // empty' 2>/dev/null)
  if [ -z "$new_token" ]; then
    echo "ERROR: No token in login response" >&2
    exit 1
  fi

  # Update .env file with new token
  if [ -f "$ENV_FILE" ]; then
    sed -i "s|^COSMO_API_KEY=.*|COSMO_API_KEY=$new_token|" "$ENV_FILE"
  fi

  echo "$new_token"
}

# Check if we have a token and if it is still valid
if [ -z "${COSMO_API_KEY:-}" ] || is_token_expired "$COSMO_API_KEY"; then
  echo "Token expired or missing, refreshing..." >&2
  COSMO_API_KEY=$(refresh_token)
fi

# --- API call ---
METHOD="${1:?Usage: cosmo_api.sh <METHOD> <PATH> [JSON_BODY]}"
PATH_="${2:?Usage: cosmo_api.sh <METHOD> <PATH> [JSON_BODY]}"
BODY="${3:-}"

ARGS=(
  -s
  -H "Authorization: Bearer $COSMO_API_KEY"
  -H "Content-Type: application/json"
  -X "$METHOD"
)

if [ -n "$BODY" ]; then
  ARGS+=(-d "$BODY")
fi

RESPONSE=$(curl -w "\n%{http_code}" "${ARGS[@]}" "$COSMO_BASE_URL$PATH_")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY_RESPONSE=$(echo "$RESPONSE" | sed '$d')

# If 401, try refreshing token once and retry
if [ "$HTTP_CODE" = "401" ]; then
  echo "Got 401, refreshing token and retrying..." >&2
  COSMO_API_KEY=$(refresh_token)
  ARGS=(
    -s
    -H "Authorization: Bearer $COSMO_API_KEY"
    -H "Content-Type: application/json"
    -X "$METHOD"
  )
  if [ -n "$BODY" ]; then
    ARGS+=(-d "$BODY")
  fi
  curl "${ARGS[@]}" "$COSMO_BASE_URL$PATH_" | jq .
else
  echo "$BODY_RESPONSE" | jq .
fi
