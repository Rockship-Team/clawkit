package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"regexp"
	"strings"
	"time"
)

var uuidRegexp = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func looksLikeUUID(s string) bool { return uuidRegexp.MatchString(s) }

// cmdProposal dispatches proposal subcommands.
//
//	sme-cli proposal pricing
//	sme-cli proposal generate <company> <contact_id> <tier> <outline_file>
func cmdProposal(args []string) {
	if len(args) == 0 {
		errOut("usage: proposal pricing|generate")
		return
	}
	switch args[0] {
	case "pricing":
		proposalPricing()
	case "generate":
		proposalGenerate(args[1:])
	default:
		errOut("unknown proposal command: " + args[0])
	}
}

type proposalTier struct {
	Name                  string   `json:"name"`
	PriceVND              int64    `json:"price_vnd"`
	AgentsMax             int      `json:"agents_max,omitempty"`
	AgentsUnlimited       bool     `json:"agents_unlimited,omitempty"`
	TransactionsPerMonth  int      `json:"transactions_per_month,omitempty"`
	TransactionsUnlimited bool     `json:"transactions_unlimited,omitempty"`
	Features              []string `json:"features"`
	BestFor               string   `json:"best_for"`
}

var proposalTiers = []proposalTier{
	{
		Name:                 "Starter",
		PriceVND:             15_000_000,
		AgentsMax:            1,
		TransactionsPerMonth: 5000,
		Features: []string{
			"1 AI Agent",
			"Up to 5,000 transactions/month",
			"Basic analytics",
			"Email support",
			"Template-based responses",
		},
		BestFor: "Small businesses, testing AI automation",
	},
	{
		Name:                 "Pro",
		PriceVND:             400_000_000,
		AgentsMax:            3,
		TransactionsPerMonth: 50000,
		Features: []string{
			"3 AI Agents (multi-channel)",
			"Up to 50,000 transactions/month",
			"Advanced analytics + reporting",
			"Priority support (24h)",
			"Custom training on company data",
			"CRM integration",
		},
		BestFor: "Mid-size companies with multi-channel needs",
	},
	{
		Name:                  "Enterprise",
		PriceVND:              800_000_000,
		AgentsUnlimited:       true,
		TransactionsUnlimited: true,
		Features: []string{
			"Unlimited AI Agents",
			"Unlimited conversations",
			"Custom integrations",
			"Dedicated account manager",
			"On-premise option",
			"SLA 99.9%",
			"Custom model training",
		},
		BestFor: "Large enterprises with complex requirements",
	},
}

var proposalAddOns = []string{
	"Dedicated Account Manager (beyond Enterprise default)",
	"On-premise deployment with hardened networking",
	"Custom SLA upgrade (99.95%+ uptime)",
	"Additional fine-tuning modules on customer data",
	"Extended BI / analytics integration",
}

func proposalPricing() {
	okOut(map[string]interface{}{
		"tiers":   proposalTiers,
		"add_ons": proposalAddOns,
		"rules": []string{
			"Only three tiers exist: Starter, Pro, Enterprise. Never invent new tiers (no 'Enterprise Plus', no 'Premium', no 'Custom').",
			"If client budget > Enterprise (800M VND/year), recommend Enterprise and flag add-ons for BD to quote separately.",
			"Never modify or round the listed prices.",
			"Discount policy: 2-year 10%, 3-year 15%, referral 5%, startup 20%.",
		},
	})
}

func validProposalTier(name string) (string, bool) {
	for _, t := range proposalTiers {
		if strings.EqualFold(t.Name, name) {
			return t.Name, true
		}
	}
	return "", false
}

func proposalGenerate(args []string) {
	if len(args) < 4 {
		errOut("usage: proposal generate <company> <contact_id> <tier> <outline_file>")
	}
	company := args[0]
	contactID := args[1]
	tierIn := args[2]
	outlinePath := args[3]

	tier, ok := validProposalTier(tierIn)
	if !ok {
		errOut(fmt.Sprintf("invalid tier %q — only Starter, Pro, Enterprise are allowed. Do not invent tiers.", tierIn))
	}

	if strings.TrimSpace(contactID) == "" {
		errOut("contact_id is required — run `sme-cli cosmo search-contact <company>` first. If no match, create via `sme-cli cosmo create-contact`, then pass the returned id.")
	}
	if !looksLikeUUID(contactID) {
		errOut(fmt.Sprintf("contact_id %q must be a UUID from COSMO CRM — Step 1 CRM check is mandatory. Run `sme-cli cosmo search-contact <company>` first.", contactID))
	}

	info, err := os.Stat(outlinePath)
	if err != nil {
		errOut("outline file not found: " + outlinePath)
	}
	if info.Size() < 200 {
		errOut(fmt.Sprintf("outline file too small (%d bytes) — provide a real approved outline, not a placeholder.", info.Size()))
	}

	self, err := os.Executable()
	if err != nil || self == "" {
		self = "sme-cli"
	}

	// Step 1 — submit task to Manus, get task_id quickly.
	submitOut, err := runSMECmd(self, "manus", "submit-proposal", company, outlinePath)
	if err != nil {
		errOut("manus submit failed: " + err.Error())
	}
	taskID, _ := submitOut["task_id"].(string)
	if taskID == "" {
		errOut(fmt.Sprintf("manus did not return task_id: %v", submitOut["error"]))
	}

	// Step 2 — short poll loop (~90s total: 9 retries x 10s).
	// Avoids bot tool-execution timeout. If not done, return pending + task_id
	// so the caller can poll via `sme-cli manus get-task <task_id>`.
	for i := 0; i < 9; i++ {
		time.Sleep(10 * time.Second)
		pollOut, err := runSMECmd(self, "manus", "get-task", taskID)
		if err != nil {
			continue
		}
		status, _ := pollOut["status"].(string)
		switch status {
		case "completed":
			pdfURL := pickPDFURL(pollOut)
			okOut(map[string]interface{}{
				"proposal": map[string]interface{}{
					"company":      company,
					"contact_id":   contactID,
					"tier":         tier,
					"outline_path": outlinePath,
				},
				"pdf_url":     pdfURL,
				"task_id":     taskID,
				"status":      "completed",
				"next_action": fmt.Sprintf("sme-cli cosmo log-interaction %s proposal_sent && sme-cli cosmo api PATCH /v1/contacts/%s '{\"business_stage\":\"PROPOSAL\"}'", contactID, contactID),
			})
			return
		case "failed":
			errOut("manus marked the task as failed — adjust the outline and retry")
		}
	}

	// Still running. Return task_id so caller can poll later without creating a duplicate task.
	okOut(map[string]interface{}{
		"proposal": map[string]interface{}{
			"company":      company,
			"contact_id":   contactID,
			"tier":         tier,
			"outline_path": outlinePath,
		},
		"status":   "pending",
		"task_id":  taskID,
		"poll_cmd": "sme-cli manus get-task " + taskID,
		"message":  fmt.Sprintf("Manus dang xu ly. KHONG tao task moi — check lai sau 2-3 phut bang `sme-cli manus get-task %s`.", taskID),
	})
}

func runSMECmd(self string, args ...string) (map[string]interface{}, error) {
	cmd := osexec.Command(self, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	var out map[string]interface{}
	if jerr := json.Unmarshal(stdout.Bytes(), &out); jerr != nil {
		return nil, fmt.Errorf("non-JSON from %s: %s%s", strings.Join(args, " "), stderr.String(), stdout.String())
	}
	if runErr != nil && out["ok"] != true {
		if errMsg, ok := out["error"].(string); ok {
			return out, fmt.Errorf("%s", errMsg)
		}
		return out, runErr
	}
	return out, nil
}

func pickPDFURL(pollOut map[string]interface{}) string {
	output, ok := pollOut["output"].([]interface{})
	if !ok {
		return ""
	}
	for _, o := range output {
		om, ok := o.(map[string]interface{})
		if !ok {
			continue
		}
		content, ok := om["content"].([]interface{})
		if !ok {
			continue
		}
		for _, c := range content {
			cm, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := cm["fileName"].(string)
			if name == "style_template.pdf" {
				continue
			}
			mime, _ := cm["mimeType"].(string)
			url, _ := cm["fileUrl"].(string)
			if url != "" && (mime == "application/pdf" || strings.HasSuffix(name, ".pdf")) {
				return url
			}
		}
	}
	return ""
}
