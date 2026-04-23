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

func cosmoVectorSearch(args []string) {
	cosmoIntelligenceSearch("/v1/intelligence/vector-search", args, "usage: cosmo vector-search <query> [limit]")
}

func cosmoHybridSearch(args []string) {
	cosmoIntelligenceSearch("/v1/intelligence/hybrid-search", args, "usage: cosmo hybrid-search <query> [limit]")
}

func cosmoSearchInteractions(args []string) {
	cosmoIntelligenceSearch("/v1/intelligence/search-interactions", args, "usage: cosmo search-interactions <query> [limit]")
}
