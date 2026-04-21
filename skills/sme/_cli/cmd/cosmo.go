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
func cmdCosmo(args []string) {
	if len(args) == 0 {
		errOut("usage: cosmo api|search-contact|get-contact|create-contact|get-interactions|log-interaction|import-txt|import-csv|enrich|score-icp|score-relationship|meeting-brief|vector-search|hybrid-search|search-interactions")
		return
	}
	switch args[0] {
	case "api":
		cosmoAPI(args[1:])
	case "search-contact":
		cosmoSearchContact(args[1:])
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

func cosmoSearchContact(args []string) {
	if len(args) == 0 {
		errOut("usage: cosmo search-contact <query> [page_size]")
	}
	query := args[0]
	pageSize := 25
	if len(args) > 1 {
		fmt.Sscanf(args[1], "%d", &pageSize)
	}
	body, _ := json.Marshal(map[string]interface{}{
		"query":    query,
		"pageSize": pageSize,
	})
	raw, code, err := cosmoRequest("POST", "/v2/contacts/search", body)
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
