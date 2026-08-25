package main

import (
	"encoding/json"
	"fmt"
)

// Convenience aliases for AI / intelligence endpoints in COSMO.
//
//	sme-cli cosmo enrich <contact_id>
//	sme-cli cosmo score-icp <contact_id>
//	sme-cli cosmo score-relationship <contact_id>
//	sme-cli cosmo meeting-brief <contact_id>
//	sme-cli cosmo vector-search <query> [limit]
//	sme-cli cosmo hybrid-search <query> [limit]
//	sme-cli cosmo search-interactions <query> [limit]

func cosmoEnrich(args []string) {
	if len(args) == 0 {
		errOut("usage: cosmo enrich <contact_id>")
	}
	raw, code, err := cosmoRequest("POST", "/v1/contacts/"+args[0]+"/enrich", nil)
	if err != nil {
		errOut(err.Error())
	}
	rawJSONPassthrough(raw, code)
}

// cosmoCalculateScores is the data-returning form of `cosmo score-icp`
// (POST /v1/contacts/{id}/calculate-scores) — extracted so sme-intelligence
// can read the actual score value instead of only printing raw JSON. The CLI
// command below is unchanged: same endpoint, same output, same exit
// behavior on error.
func cosmoCalculateScores(contactID string) (map[string]interface{}, int, error) {
	raw, code, err := cosmoRequest("POST", "/v1/contacts/"+contactID+"/calculate-scores", nil)
	if err != nil || code >= 400 {
		return nil, code, err
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, code, err
	}
	return resp, code, nil
}

func cosmoScoreICP(args []string) {
	if len(args) == 0 {
		errOut("usage: cosmo score-icp <contact_id>")
	}
	raw, code, err := cosmoRequest("POST", "/v1/contacts/"+args[0]+"/calculate-scores", nil)
	if err != nil {
		errOut(err.Error())
	}
	rawJSONPassthrough(raw, code)
}

// cosmoRelationshipScoreData is the data-returning form of
// `cosmo score-relationship` (POST /v1/contacts/{id}/relationship-score) —
// same reuse rationale as cosmoCalculateScores above.
func cosmoRelationshipScoreData(contactID string) (map[string]interface{}, int, error) {
	raw, code, err := cosmoRequest("POST", "/v1/contacts/"+contactID+"/relationship-score", nil)
	if err != nil || code >= 400 {
		return nil, code, err
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, code, err
	}
	return resp, code, nil
}

func cosmoScoreRelationship(args []string) {
	if len(args) == 0 {
		errOut("usage: cosmo score-relationship <contact_id>")
	}
	raw, code, err := cosmoRequest("POST", "/v1/contacts/"+args[0]+"/relationship-score", nil)
	if err != nil {
		errOut(err.Error())
	}
	rawJSONPassthrough(raw, code)
}

func cosmoMeetingBrief(args []string) {
	if len(args) == 0 {
		errOut("usage: cosmo meeting-brief <contact_id>")
	}
	raw, code, err := cosmoRequest("POST", "/v1/contacts/"+args[0]+"/generate-meeting-brief", nil)
	if err != nil {
		errOut(err.Error())
	}
	rawJSONPassthrough(raw, code)
}

func cosmoIntelligenceSearch(endpoint string, args []string, usage string) {
	if len(args) == 0 {
		errOut(usage)
	}
	query := args[0]
	limit := 10
	if len(args) > 1 {
		fmt.Sscanf(args[1], "%d", &limit)
	}
	body, err := json.Marshal(map[string]interface{}{"query": query, "limit": limit})
	if err != nil {
		errOut("encode body: " + err.Error())
	}
	raw, code, err := cosmoRequest("POST", endpoint, body)
	if err != nil {
		errOut(err.Error())
	}
	rawJSONPassthrough(raw, code)
}

// Real routes (cosmo-backend cmd/server/routes_v1.go): the intelligence
// group is mounted at /v1/intelligence, and vector-search endpoints live
// under /vector-search/<kind> — NOT the bare /vector-search, /hybrid-search,
// /search-interactions this file called before this fix (all 404 live,
// confirmed during the Phase 2B COSMO audit). NOTE: even with the correct
// path, this endpoint currently 500s live with an upstream 401 from OpenAI
// ("Incorrect API key provided") — a cosmo-backend-side embedding-key
// misconfiguration, NOT a client bug. Intelligence must not depend on this
// until that's fixed backend-side; use Apollo + cosmoContactsSearch instead.
func cosmoVectorSearch(args []string) {
	cosmoIntelligenceSearch("/v1/intelligence/vector-search/contacts", args, "usage: cosmo vector-search <query> [limit]")
}

func cosmoHybridSearch(args []string) {
	cosmoIntelligenceSearch("/v1/intelligence/vector-search/hybrid", args, "usage: cosmo hybrid-search <query> [limit]")
}

func cosmoSearchInteractions(args []string) {
	cosmoIntelligenceSearch("/v1/intelligence/vector-search/interactions", args, "usage: cosmo search-interactions <query> [limit]")
}
