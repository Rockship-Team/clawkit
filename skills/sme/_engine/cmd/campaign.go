package main

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// cmdCampaign dispatches sme-campaign's execution layer — the smallest real
// layer supported by COSMO's actual Campaign API (audited against
// cosmo-backend source during Phase 2C, see campaign/SKILL.md and
// GTM_ARCHITECTURE.md for the full capability matrix). This does NOT
// duplicate ICP/research (sme-intelligence), CRM segmentation (sme-crm),
// content generation (sme-marketing, prompt-level per marketing/SKILL.md
// section C — there is no Go copywriting engine here on purpose), or
// LinkedIn execution (sme-outreach, read-only CDP client — no send
// capability exists anywhere in this codebase, confirmed during audit).
//
//	sme-cli campaign create --name X --playbook P [--segment LIST_CONTACT_ID] [--channel email|linkedin]
//	sme-cli campaign list [--status draft|scheduled|active|paused|ended]
//	sme-cli campaign view <campaign_id>
//	sme-cli campaign stats <campaign_id>          # alias of view — COSMO's GetByID already returns sent/reply/reply_rate
//	sme-cli campaign add-template <campaign_id> --type T --subject S --content C [--send-after N] [--position P]
//	sme-cli campaign templates <campaign_id>
//	sme-cli campaign activate <campaign_id>       # APPROVAL-gated at the SKILL.md/orchestrator layer, not here
//	sme-cli campaign pause <campaign_id>
//	sme-cli campaign delete <campaign_id>
func cmdCampaign(args []string) {
	if len(args) == 0 {
		errOut("usage: campaign create|list|view|stats|add-template|templates|activate|pause|delete")
		return
	}
	switch args[0] {
	case "create":
		campaignCreate(args[1:])
	case "list":
		campaignList(args[1:])
	case "view", "stats":
		campaignView(args[1:])
	case "add-template":
		campaignAddTemplate(args[1:])
	case "templates":
		campaignTemplates(args[1:])
	case "activate":
		campaignActivate(args[1:])
	case "pause":
		campaignPause(args[1:])
	case "delete":
		campaignDelete(args[1:])
	default:
		errOut("unknown campaign command: " + args[0])
	}
}

// --- create -----------------------------------------------------------

// campaignCreate always creates in COSMO's own default status: draft
// (domain.Campaign's gorm default, and Create() only flips to active if
// the caller explicitly passes status=active — this code path never does).
// Creating a campaign MUST NOT send anything: draft has no list/agent/
// template requirement, so nothing here can trigger COSMO's
// worker.TypeExecuteCampaign enqueue (that only fires on a transition INTO
// active, see campaignActivate).
func campaignCreate(args []string) {
	var name, playbook, segment, channel string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			i++
			name = args[i]
		case "--playbook":
			i++
			playbook = args[i]
		case "--segment":
			i++
			segment = args[i]
		case "--channel":
			i++
			channel = args[i]
		}
	}
	if playbook == "" {
		errOut("usage: campaign create --name X --playbook P [--segment LIST_CONTACT_ID] [--channel email|linkedin]")
		return
	}
	if channel != "" && channel != "email" && channel != "linkedin" {
		errOut("--channel phải là email hoặc linkedin")
		return
	}

	body := map[string]interface{}{"playbook": playbook}
	if name != "" {
		body["name"] = name
	}
	if segment != "" {
		body["list_contact_id"] = segment
	}
	// Best-effort: attach the org's email-sending agent so it's already set
	// once the user is ready to activate an email-channel campaign. Never
	// fatal if none found — a campaign with no agent_id simply can't
	// transition to active yet (COSMO's own validateStatusTransition already
	// enforces this), which is exactly the safe default.
	if agentID, _, err := cosmoDefaultAgent(); err == nil && agentID != "" {
		body["agent_id"] = agentID
	}

	raw, code, err := cosmoRequest("POST", "/v1/campaigns", mustJSON(body))
	if err != nil {
		errOut(err.Error())
		return
	}
	if code >= 400 {
		errOut(fmt.Sprintf("HTTP %d: %s", code, string(raw)))
		return
	}
	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		errOut(err.Error())
		return
	}
	campaignID := fmt.Sprint(resp.Data["id"])

	if channel != "" {
		if _, code, err := cosmoRequest("PATCH", "/v1/campaigns/"+campaignID+"/client-metadata",
			mustJSON(map[string]interface{}{"client": map[string]interface{}{"channel": channel}})); err != nil || code >= 400 {
			// Non-fatal: campaign exists as draft either way; channel marker
			// just controls activate's routing (email vs linkedin) below.
		}
	}

	okOut(map[string]interface{}{
		"campaign_id": campaignID,
		"status":      resp.Data["status"],
		"channel":     firstNonEmpty(channel, "email"),
		"message":     "Campaign tạo ở trạng thái draft — KHÔNG gửi gì. Cần add-template + activate (có approval) để bắt đầu.",
	})
}

// --- list / view --------------------------------------------------------

func campaignList(args []string) {
	status := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--status" && i+1 < len(args) {
			i++
			status = args[i]
		}
	}
	path := "/v1/campaigns?limit=25"
	if status != "" {
		path += "&status=" + status
	}
	raw, code, err := cosmoRequest("GET", path, nil)
	if err != nil {
		errOut(err.Error())
		return
	}
	rawJSONPassthrough(raw, code)
}

func campaignView(args []string) {
	if len(args) == 0 {
		errOut("usage: campaign view <campaign_id>")
		return
	}
	raw, code, err := cosmoRequest("GET", "/v1/campaigns/"+args[0], nil)
	if err != nil {
		errOut(err.Error())
		return
	}
	rawJSONPassthrough(raw, code)
}

// --- templates ----------------------------------------------------------

// campaignAddTemplate writes ONE cadence step's already-drafted content
// (sourced from sme-marketing's rules, applied by the agent — never
// generated here) via POST /v1/template. This is the AI-independent path:
// COSMO's own /v1/campaigns/{id}/generate and the v2/v3 template-generation
// endpoints all require an OpenAI client that is NOT configured on this
// backend right now (confirmed during the Phase 2B COSMO audit — same
// upstream key issue as /v1/intelligence/vector-search). Using the plain
// CRUD template endpoint avoids that dependency entirely and matches "no
// copywriting engine inside campaign.go" — content is a required argument,
// never generated here.
func campaignAddTemplate(args []string) {
	if len(args) == 0 {
		errOut("usage: campaign add-template <campaign_id> --type T --subject S --content C [--send-after N] [--position P]")
		return
	}
	campaignID := args[0]
	typ, subject, content := "", "", ""
	sendAfter := -1
	position := -1.0
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--type":
			i++
			typ = args[i]
		case "--subject":
			i++
			subject = args[i]
		case "--content":
			i++
			content = args[i]
		case "--send-after":
			i++
			n, _ := strconv.Atoi(args[i])
			sendAfter = n
		case "--position":
			i++
			p, _ := strconv.ParseFloat(args[i], 64)
			position = p
		}
	}
	if subject == "" || content == "" {
		errOut("usage: campaign add-template <campaign_id> --type T --subject S --content C [--send-after N] [--position P]")
		return
	}
	if typ == "" {
		typ = "outreach"
	}
	body := map[string]interface{}{
		"campaign_id": campaignID,
		"type":        typ,
		"category":    "Outreach",
		"subject":     subject,
		"content":     content,
	}
	if sendAfter >= 0 {
		body["send_after"] = sendAfter
	}
	if position >= 0 {
		body["position"] = position
	}
	raw, code, err := cosmoRequest("POST", "/v1/template", mustJSON(body))
	if err != nil {
		errOut(err.Error())
		return
	}
	rawJSONPassthrough(raw, code)
}

func campaignTemplates(args []string) {
	if len(args) == 0 {
		errOut("usage: campaign templates <campaign_id>")
		return
	}
	raw, code, err := cosmoRequest("GET", "/v1/template?campaign_id="+args[0]+"&limit=50", nil)
	if err != nil {
		errOut(err.Error())
		return
	}
	rawJSONPassthrough(raw, code)
}

// --- activate / pause / delete -------------------------------------------

// campaignActivate is the ONLY command in this file that can trigger a
// real external send, and only for channel=email — and only after this
// function independently verifies the sending agent's live Google-auth
// status (COSMO's own PATCH validation checks agent_id is non-nil, but NOT
// whether its Google grant is still valid — a stale/invalid grant would
// otherwise fail silently deep in the worker). This is the Approval Policy
// checkpoint's execution point; the actual approval GATE (asking the human
// to confirm) lives in campaign/SKILL.md and orchestrator's approval-policy
// reference, not in this CLI — matching how sme-proposal's send approval
// works today.
//
// channel=linkedin NEVER reaches COSMO's PATCH status=active at all: COSMO
// has no channel concept — agent_id is always interpreted as "the email
// identity that sends this" by the worker. Forcing a LinkedIn campaign
// through that path would either be rejected (no agent) or, worse, silently
// trigger a real EMAIL send for what was meant to be a LinkedIn cadence.
// Instead this marks readiness in cmetadata.client only (no COSMO status
// change), and the human executes LinkedIn messages manually, tracked via
// sme-outreach (which has no send capability by design — read-only CDP).
func campaignActivate(args []string) {
	if len(args) == 0 {
		errOut("usage: campaign activate <campaign_id>")
		return
	}
	campaignID := args[0]

	view, code, err := cosmoGetCampaign(campaignID)
	if err != nil || code >= 400 {
		errOut(fmt.Sprintf("không lấy được campaign: %v (HTTP %d)", err, code))
		return
	}
	channel := campaignChannel(view)

	if channel == "linkedin" {
		_, code, err := cosmoRequest("PATCH", "/v1/campaigns/"+campaignID+"/client-metadata",
			mustJSON(map[string]interface{}{"client": map[string]interface{}{"linkedin_status": "ready_for_manual_send"}}))
		if err != nil || code >= 400 {
			errOut(fmt.Sprintf("không đánh dấu ready được: %v (HTTP %d)", err, code))
			return
		}
		okOut(map[string]interface{}{
			"campaign_id": campaignID,
			"channel":     "linkedin",
			"status":      "ready_for_manual_send",
			"message":     "Channel LinkedIn KHÔNG có auto-send (sme-outreach chỉ đọc, không gửi). Campaign đánh dấu READY — thực thi thủ công qua LinkedIn, log lại qua `sme-cli outreach log-event`.",
		})
		return
	}

	// channel == "email" (default when unset)
	agentID, agentStatus, err := cosmoDefaultAgent()
	if err != nil {
		errOut("không kiểm tra được trạng thái Google auth: " + err.Error())
		return
	}
	if agentID == "" {
		errOut("Không có agent (email identity) nào để gửi — cần kết nối Google trước khi activate campaign kênh email.")
		return
	}
	if agentStatus != "active" {
		errOut(fmt.Sprintf("KHÔNG activate — Google auth của agent đang ở trạng thái %q (không phải 'active'). Cần reconnect Google trong COSMO trước khi gửi email thật. Campaign vẫn ở draft, không có gì bị gửi.", agentStatus))
		return
	}

	raw, code, err := cosmoRequest("PATCH", "/v1/campaigns/"+campaignID,
		mustJSON(map[string]interface{}{"status": "active", "agent_id": agentID}))
	if err != nil {
		errOut(err.Error())
		return
	}
	if code >= 400 {
		errOut(fmt.Sprintf("HTTP %d: %s", code, string(raw)))
		return
	}
	okOut(map[string]interface{}{
		"campaign_id": campaignID,
		"channel":     "email",
		"status":      "active",
		"message":     "Đã activate — COSMO worker sẽ bắt đầu gửi email theo template/cadence đã lưu.",
	})
}

func campaignPause(args []string) {
	if len(args) == 0 {
		errOut("usage: campaign pause <campaign_id>")
		return
	}
	raw, code, err := cosmoRequest("PATCH", "/v1/campaigns/"+args[0], mustJSON(map[string]interface{}{"status": "paused"}))
	if err != nil {
		errOut(err.Error())
		return
	}
	if code >= 400 {
		errOut(fmt.Sprintf("HTTP %d: %s", code, string(raw)))
		return
	}
	okOut(map[string]interface{}{"campaign_id": args[0], "status": "paused"})
}

func campaignDelete(args []string) {
	if len(args) == 0 {
		errOut("usage: campaign delete <campaign_id>")
		return
	}
	raw, code, err := cosmoRequest("DELETE", "/v1/campaigns/"+args[0], nil)
	if err != nil {
		errOut(err.Error())
		return
	}
	rawJSONPassthrough(raw, code)
}

// --- helpers --------------------------------------------------------------

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

// cosmoGetCampaign returns the parsed GET /v1/campaigns/{id} response body
// as a generic map — used by campaignActivate to read cmetadata.client.channel
// without needing a second typed schema.
func cosmoGetCampaign(campaignID string) (map[string]interface{}, int, error) {
	raw, code, err := cosmoRequest("GET", "/v1/campaigns/"+campaignID, nil)
	if err != nil || code >= 400 {
		return nil, code, err
	}
	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, code, err
	}
	return resp.Data, code, nil
}

// campaignChannel reads the channel marker set at create time
// (cmetadata.client.channel). Defaults to "email" — COSMO's own campaign
// model has no channel concept and is implicitly email-only, so an unset
// marker must resolve to the ONLY channel COSMO's activation path actually
// supports, never silently to "linkedin" (which would skip the real
// Google-auth check below).
func campaignChannel(view map[string]interface{}) string {
	entity, _ := view["entity"].(map[string]interface{})
	if entity == nil {
		entity = view
	}
	cmeta, _ := entity["cmetadata"].(map[string]interface{})
	if cmeta == nil {
		return "email"
	}
	client, _ := cmeta["client"].(map[string]interface{})
	if client == nil {
		return "email"
	}
	if ch, ok := client["channel"].(string); ok && ch != "" {
		return ch
	}
	return "email"
}

// cosmoDefaultAgent resolves this org's single email-sending identity and
// its LIVE Google-auth status (domain.AgentStatus — "active",
// "invalid Google grant", "insufficient scopes", "needs sync setup",
// "inactive"). Returns ("", "", nil) if no agent exists yet. Multiple
// agents are NOT disambiguated here — campaignActivate only needs to know
// whether sending is currently possible at all, not which of several
// identities to pick; if that becomes a real need, resolve explicitly via
// `cosmo api POST /v1/agents/search` instead of guessing.
func cosmoDefaultAgent() (agentID, status string, err error) {
	raw, code, err := cosmoRequest("POST", "/v1/agents/search", mustJSON(map[string]interface{}{}))
	if err != nil {
		return "", "", err
	}
	if code >= 400 {
		return "", "", fmt.Errorf("HTTP %d: %s", code, string(raw))
	}
	var resp struct {
		Data struct {
			List []struct {
				Entity struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"entity"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", "", err
	}
	if len(resp.Data.List) == 0 {
		return "", "", nil
	}
	return resp.Data.List[0].Entity.ID, resp.Data.List[0].Entity.Status, nil
}
