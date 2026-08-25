package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// cmdCosmo dispatches COSMO CRM subcommands.
//
//	sme-cli cosmo api <METHOD> <PATH> [JSON_BODY]
//	sme-cli cosmo search-contact <query> [page_size]
//	sme-cli cosmo get-contact <contact_id>
//	sme-cli cosmo create-contact              (JSON on stdin)
//	sme-cli cosmo get-interactions <contact_id> [limit]
//	sme-cli cosmo log-interaction <contact_id> <type>
//	sme-cli cosmo import-txt <file> [--list-id UUID] [--source STRING]
//	sme-cli cosmo import-csv <file> [--list-id UUID] [--source STRING] [--format luma|generic]
//	sme-cli cosmo enrich <contact_id>
//	sme-cli cosmo score-icp <contact_id>
//	sme-cli cosmo score-relationship <contact_id>
//	sme-cli cosmo meeting-brief <contact_id>
//	sme-cli cosmo vector-search <query> [limit]
//	sme-cli cosmo hybrid-search <query> [limit]
//	sme-cli cosmo search-interactions <query> [limit]
//	sme-cli cosmo daily-plan [--mode morning|evening|all] [--max-pages N]
func cmdCosmo(args []string) {
	if len(args) == 0 {
		errOut("usage: cosmo api|search-contact|find-by-email|get-contact|create-contact|get-interactions|log-interaction|import-txt|import-csv|enrich|score-icp|score-relationship|meeting-brief|vector-search|hybrid-search|search-interactions|daily-plan")
		return
	}
	switch args[0] {
	case "api":
		cosmoAPI(args[1:])
	case "search-contact":
		cosmoSearchContact(args[1:])
	case "find-by-email":
		cosmoFindByEmailCmd(args[1:])
	case "get-contact":
		cosmoGetContact(args[1:])
	case "create-contact":
		cosmoCreateContact()
	case "get-interactions":
		cosmoGetInteractions(args[1:])
	case "log-interaction":
		cosmoLogInteraction(args[1:])
	case "import-txt":
		cosmoImportTXT(args[1:])
	case "import-csv":
		cosmoImportCSV(args[1:])
	case "enrich":
		cosmoEnrich(args[1:])
	case "score-icp":
		cosmoScoreICP(args[1:])
	case "score-relationship":
		cosmoScoreRelationship(args[1:])
	case "meeting-brief":
		cosmoMeetingBrief(args[1:])
	case "vector-search":
		cosmoVectorSearch(args[1:])
	case "hybrid-search":
		cosmoHybridSearch(args[1:])
	case "search-interactions":
		cosmoSearchInteractions(args[1:])
	case "daily-plan":
		cosmoDailyPlan(args[1:])
	default:
		errOut("unknown cosmo command: " + args[0])
	}
}

// cosmoRequest calls the COSMO API, auto-refreshing the JWT on first 401.
func cosmoRequest(method, apiPath string, body []byte) ([]byte, int, error) {
	c := loadConnections()
	if c.COSMO.BaseURL == "" {
		return nil, 0, fmt.Errorf("cosmo.base_url not set — run: sme-cli config set cosmo.base_url <url>")
	}
	if c.COSMO.APIKey == "" || cosmoTokenExpired(c.COSMO.APIKey) {
		token, err := cosmoRefreshToken(c)
		if err != nil {
			return nil, 0, err
		}
		c.COSMO.APIKey = token
	}

	resp, code, err := cosmoDoRequest(c.COSMO.BaseURL, apiPath, method, c.COSMO.APIKey, body)
	if err != nil {
		return nil, 0, err
	}
	if code != http.StatusUnauthorized {
		return resp, code, nil
	}

	// Refresh once and retry.
	token, err := cosmoRefreshToken(c)
	if err != nil {
		return nil, 0, err
	}
	c.COSMO.APIKey = token
	return cosmoDoRequest(c.COSMO.BaseURL, apiPath, method, c.COSMO.APIKey, body)
}

func cosmoDoRequest(baseURL, apiPath, method, token string, body []byte) ([]byte, int, error) {
	url := strings.TrimRight(baseURL, "/") + apiPath
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	out, err := io.ReadAll(resp.Body)
	return out, resp.StatusCode, err
}

// cosmoTokenExpired returns true if the JWT has <60s remaining.
// Personal API keys (prefix "p_api_key_") never expire.
func cosmoTokenExpired(token string) bool {
	if strings.HasPrefix(token, "p_api_key_") {
		return false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return true
	}
	payload := parts[1]
	// Pad base64.
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	raw, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		raw, err = base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return true
		}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil || claims.Exp == 0 {
		return true
	}
	return time.Now().Unix() >= claims.Exp-60
}

// cosmoRefreshToken exchanges the auth_email for a fresh bearer token and
// persists it back to connections.json.
func cosmoRefreshToken(c Connections) (string, error) {
	if c.COSMO.AuthEmail == "" {
		return "", fmt.Errorf("cosmo.auth_email not set — run: sme-cli config set cosmo.auth_email <email>")
	}
	body, err := json.Marshal(map[string]string{"email": c.COSMO.AuthEmail})
	if err != nil {
		return "", fmt.Errorf("encode COSMO login payload: %w", err)
	}
	url := strings.TrimRight(c.COSMO.BaseURL, "/") + "/v1/auth/login"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build COSMO login request (check cosmo.base_url): %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not connect to COSMO — check your internet connection")
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read COSMO login response: %w", err)
	}
	var payload struct {
		Status string `json:"status"`
		Data   struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("received an unexpected response from COSMO login")
	}
	if payload.Status != "success" || payload.Data.Token == "" {
		return "", fmt.Errorf("COSMO login failed — verify your auth_email is correct")
	}
	c.COSMO.APIKey = payload.Data.Token
	if err := saveConnections(c); err != nil {
		return "", fmt.Errorf("save refreshed token: %w", err)
	}
	return payload.Data.Token, nil
}

// rawJSONPassthrough prints the response body as parsed JSON (or raw text
// if it isn't JSON). Used by commands that return COSMO payloads verbatim.
func rawJSONPassthrough(raw []byte, code int) {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		fmt.Println(string(raw))
		if code >= 400 {
			os.Exit(1)
		}
		return
	}
	jsonOut(v)
	if code >= 400 {
		os.Exit(1)
	}
}

func cosmoAPI(args []string) {
	if len(args) < 2 {
		errOut("usage: cosmo api <METHOD> <PATH> [JSON_BODY]")
	}
	method, path := args[0], args[1]
	var body []byte
	if len(args) > 2 {
		body = []byte(args[2])
	}
	raw, code, err := cosmoRequest(method, path, body)
	if err != nil {
		errOut(err.Error())
	}
	rawJSONPassthrough(raw, code)
}

// cosmoContactsSearch is the ONE place that calls POST /v2/contacts/search
// with the request shape the backend actually implements: a
// {"filter": {...}} body (v2schema.ContactSearchRequest.Filter) — NOT a
// free-text "query"/"pageSize" body. The handler silently ignores unknown
// body fields, so a {"query": "...", "pageSize": N} body (the shape every
// call site in this file used before this fix) always falls through to
// "no filter" and returns the first page regardless of what was asked for.
// filter may be nil for "no filter, list all" (the actual behavior every
// call site was unknowingly relying on before this fix).
//
// Pagination is backend-side query-string only (parsePagination reads
// offset/limit/page_index from the query string, never the JSON body).
// pageIndex is 0-INDEXED, never a raw offset: cosmo-backend's parsePagination
// silently reinterprets a bare ?offset=N (N>0) as "N whole pages", computing
// real_offset = N*limit — so passing a true byte offset (e.g. offset=25 to
// skip one page of 25) actually skips 625 records and silently returns
// nothing. ?page_index=N sidesteps that entirely: the backend computes
// offset = N*limit itself, exactly matching normal "page N of size limit"
// pagination. Root-caused against cosmo-backend's
// internal/handler/v2/contact/search.go, contact_repository.go, and
// helpers.go (parsePagination) during the Phase 2B COSMO audit — all client-
// side bugs, not backend/API contract limitations, so fixing them here is
// safe and backward compatible.
func cosmoContactsSearch(filter map[string]interface{}, limit, pageIndex int) ([]byte, int, error) {
	body := []byte(`{}`)
	if len(filter) > 0 {
		b, err := json.Marshal(map[string]interface{}{"filter": filter})
		if err != nil {
			return nil, 0, err
		}
		body = b
	}
	path := fmt.Sprintf("/v2/contacts/search?limit=%d&page_index=%d", limit, pageIndex)
	return cosmoRequest("POST", path, body)
}

// contactTextFilter builds an ILIKE-based OR filter across name/company
// approximating free-text search — COSMO has no true full-text query param
// on this endpoint, only per-field filters (see cosmoContactsSearch).
//
// Deliberately does NOT include "email"/"phone": despite the backend's own
// isTextSearchField() listing them as text-search fields, domain.Contact has
// no email/phone COLUMN at all (see cosmo-backend/internal/domain/contact/
// contact.go) — those values live only inside the profile JSONB blob. Any
// filter naming "email"/"phone" directly builds `WHERE email ILIKE ...`
// against a nonexistent column and 500s ("Failed to get contacts"),
// confirmed live during the Phase 2B COSMO audit. Email lookups must go
// through contactExactFilter, which redirects to "profile.email".
func contactTextFilter(query string) map[string]interface{} {
	if query == "" {
		return nil
	}
	like := "%" + query + "%"
	return map[string]interface{}{
		"$or": []interface{}{
			map[string]interface{}{"name": map[string]interface{}{"$ilike": like}},
			map[string]interface{}{"company": map[string]interface{}{"$ilike": like}},
		},
	}
}

// contactExactFilter builds a single-field exact-match filter. Passes a
// PLAIN value (never wrapped in $ilike) because:
//   - "id" is a UUID column — Postgres rejects ILIKE against uuid (confirmed
//     live: 500 "Failed to get contacts"). A plain value on "id" isn't in
//     the backend's text-search/enum-exact field lists, so it falls through
//     to a raw "=" comparison, which is the exact match we want.
//   - "email"/"phone" aren't real columns (see contactTextFilter) — the key
//     is redirected to "profile.<field>", which cosmo-backend's
//     NormalizeContactFilter special-cases into an exact `profile->>'x' = 'v'`
//     JSONB match. This only works with a plain string value, and only at
//     the top level of the filter (not nested inside $or/$and).
func contactExactFilter(field, value string) map[string]interface{} {
	key := field
	switch field {
	case "email", "phone":
		key = "profile." + field
	}
	return map[string]interface{}{key: value}
}

func cosmoSearchContact(args []string) {
	if len(args) == 0 {
		errOut("usage: cosmo search-contact <query> [page_size]")
	}
	query := args[0]
	pageSize := 25
	if len(args) > 1 {
		fmt.Sscanf(args[1], "%d", &pageSize)
	}
	raw, code, err := cosmoContactsSearch(contactTextFilter(query), pageSize, 0)
	if err != nil {
		errOut(err.Error())
	}
	rawJSONPassthrough(raw, code)
}

func cosmoGetContact(args []string) {
	if len(args) == 0 {
		errOut("usage: cosmo get-contact <contact_id>")
	}
	raw, code, err := cosmoRequest("GET", "/v1/contacts/"+args[0], nil)
	if err != nil {
		errOut(err.Error())
	}
	rawJSONPassthrough(raw, code)
}

func cosmoCreateContact() {
	body, err := io.ReadAll(os.Stdin)
	if err != nil || len(bytes.TrimSpace(body)) == 0 {
		errOut("cosmo create-contact: JSON body required on stdin")
	}
	raw, code, err := cosmoRequest("POST", "/v1/contacts", body)
	if err != nil {
		errOut(err.Error())
	}
	rawJSONPassthrough(raw, code)
}

func cosmoGetInteractions(args []string) {
	if len(args) == 0 {
		errOut("usage: cosmo get-interactions <contact_id> [limit]")
	}
	limit := 50
	if len(args) > 1 {
		fmt.Sscanf(args[1], "%d", &limit)
	}
	path := fmt.Sprintf("/v1/contacts/%s/interactions?limit=%d", args[0], limit)
	raw, code, err := cosmoRequest("GET", path, nil)
	if err != nil {
		errOut(err.Error())
	}
	rawJSONPassthrough(raw, code)
}

func cosmoLogInteraction(args []string) {
	if len(args) < 2 {
		errOut("usage: cosmo log-interaction <contact_id> <type>")
	}
	body, _ := json.Marshal(map[string]string{
		"type":       args[1],
		"timestamp":  time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"created_by": "system",
	})
	raw, code, err := cosmoRequest("POST", "/v1/contacts/"+args[0]+"/interactions", body)
	if err != nil {
		errOut(err.Error())
	}
	rawJSONPassthrough(raw, code)
}

// cosmoUpsertContactForEvent creates or updates a contact in COSMO with
// `source` and `source_id` set to the event, plus custom_fields carrying
// event title/type/role. This is the gateway sme-events uses to push
// registrants into CRM without bypassing crm's role as the sole COSMO
// caller.
//
// Idempotency: first search by email; if a matching contact exists, patch
// it with the new source + custom_fields instead of creating a duplicate.
//
// role = "registered" | "paid" | "checked_in" (shows up in source string
// so you can segment by querying `source = event_<type>_<role>`).
//
// COSMO's contact model accepts `source` (string), `source_id` (UUID) and
// `custom_fields` (map). `tags` is a server-managed map and silently
// drops array values, so we don't rely on it. Attendee lookups should go
// through local event_registrations (source of truth) rather than CRM
// search.
func cosmoUpsertContactForEvent(name, email, eventID, eventTitle, eventTypeID, role string) (string, error) {
	if email == "" {
		return "", fmt.Errorf("email required")
	}
	if role == "" {
		role = "registered"
	}
	if eventTypeID == "" {
		eventTypeID = "event"
	}
	sourceStr := fmt.Sprintf("event_%s_%s", eventTypeID, role)
	custom := map[string]string{
		"event_id":    eventID,
		"event_title": eventTitle,
		"event_type":  eventTypeID,
		"event_role":  role,
	}

	existingID := cosmoFindContactByEmail(email)
	if existingID != "" {
		patchBody, _ := json.Marshal(map[string]interface{}{
			"source":        sourceStr,
			"source_id":     eventID,
			"custom_fields": custom,
		})
		cosmoRequest("PATCH", "/v1/contacts/"+existingID, patchBody)
		return existingID, nil
	}

	payload := map[string]interface{}{
		"name":          name,
		"email":         email,
		"source":        sourceStr,
		"source_id":     eventID,
		"custom_fields": custom,
	}
	body, _ := json.Marshal(payload)
	raw, code, err := cosmoRequest("POST", "/v1/contacts", body)
	if err != nil {
		return "", err
	}
	if code >= 400 {
		return "", fmt.Errorf("create contact HTTP %d: %s", code, string(raw))
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", err
	}
	if resp.Data.ID != "" {
		return resp.Data.ID, nil
	}
	return resp.ID, nil
}

// cosmoFindContactByEmail searches COSMO for a contact matching the given
// email. Returns contact id or empty string. Silent on errors — caller
// will fall through to create.
func cosmoFindContactByEmail(email string) string {
	raw, code, err := cosmoContactsSearch(contactExactFilter("email", email), 5, 0)
	if err != nil || code >= 400 {
		return ""
	}
	var resp struct {
		Data struct {
			List []struct {
				Entity struct {
					ID    string `json:"id"`
					Email string `json:"email"`
				} `json:"entity"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return ""
	}
	emailLower := strings.ToLower(strings.TrimSpace(email))
	for _, it := range resp.Data.List {
		if strings.ToLower(it.Entity.Email) == emailLower {
			return it.Entity.ID
		}
	}
	return ""
}

// cosmoFindByEmailCmd is a CLI-exposed wrapper around the same email lookup
// cosmoFindContactByEmail already does — added during Phase 3A's PIPELINE_WATCH
// dry-run: reminder/SKILL.md's documented step ("search trong COSMO contacts
// qua sme-cli cosmo search-contact") does not actually work for matching a
// Gmail sender address, because cosmo search-contact deliberately only
// filters name/company (email/phone aren't real columns — see cosmo.go's
// contactTextFilter, Phase 2B). Without this command, PIPELINE_WATCH would
// silently never match any real reply to a contact. Returns richer context
// (name/company/business_stage/next_step) than the internal helper so the
// caller doesn't need a second round-trip.
func cosmoFindByEmailCmd(args []string) {
	if len(args) == 0 {
		errOut("usage: cosmo find-by-email <email>")
		return
	}
	email := args[0]
	raw, code, err := cosmoContactsSearch(contactExactFilter("email", email), 5, 0)
	if err != nil {
		errOut(err.Error())
		return
	}
	if code >= 400 {
		errOut(fmt.Sprintf("HTTP %d: %s", code, string(raw)))
		return
	}
	var resp struct {
		Data struct {
			List []struct {
				Entity map[string]interface{} `json:"entity"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		errOut(err.Error())
		return
	}
	emailLower := strings.ToLower(strings.TrimSpace(email))
	for _, item := range resp.Data.List {
		e := item.Entity
		if em, _ := e["email"].(string); strings.ToLower(em) == emailLower {
			okOut(map[string]interface{}{
				"found":          true,
				"id":             e["id"],
				"name":           e["name"],
				"company":        e["company"],
				"business_stage": e["business_stage"],
				"next_step":      e["next_step"],
			})
			return
		}
	}
	okOut(map[string]interface{}{"found": false, "email": email})
}
