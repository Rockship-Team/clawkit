package main

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed templates/*.md templates/*.pdf
var manusTemplates embed.FS

const (
	manusDefaultBaseURL = "https://api.manus.ai"
	manusPollInterval   = 10 * time.Second
	manusMaxRetries     = 90
	manusProposalPrompt = "Dựa trên outline dưới đây, tạo 1 bản PDF proposal format đẹp như " +
		"1 bài thuyết trình. Style giống y chang file style_template.pdf " +
		"đính kèm nha — giữ đúng màu sắc, layout, font chữ, card design. " +
		"Không dùng HTML convert PDF, làm slide-style cho đẹp."
)

// cmdManus dispatches Manus AI subcommands.
//
//	sme-cli manus create-task               (JSON body on stdin)
//	sme-cli manus get-task <task_id>
//	sme-cli manus generate-proposal <company> <outline_file|->
//	sme-cli manus template <name>           (print embedded proposal template)
//	sme-cli manus list-templates
func cmdManus(args []string) {
	if len(args) == 0 {
		errOut("usage: manus create-task|get-task|generate-proposal|template|list-templates")
		return
	}
	switch args[0] {
	case "create-task":
		manusCreateTask()
	case "get-task":
		manusGetTask(args[1:])
	case "generate-proposal":
		manusGenerateProposal(args[1:])
	case "submit-proposal":
		manusSubmitProposal(args[1:])
	case "template":
		manusPrintTemplate(args[1:])
	case "list-templates":
		manusListTemplates()
	default:
		errOut("unknown manus command: " + args[0])
	}
}

func manusCreds() (apiKey, baseURL string) {
	c := loadConnections()
	if c.Manus.APIKey == "" {
		errOut("manus.api_key not set — run: sme-cli config set manus.api_key <key>")
	}
	base := c.Manus.BaseURL
	if base == "" {
		base = manusDefaultBaseURL
	}
	return c.Manus.APIKey, strings.TrimRight(base, "/")
}

// manusPost / manusGet use Manus's non-standard "API_KEY:" header
// (NOT "Authorization: Bearer").
func manusDo(method, url string, body []byte) ([]byte, int, error) {
	apiKey, _ := manusCreds()

	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, 0, err
	}
	// Manus requires the exact header "API_KEY" (not canonicalized). Assigning
	// via the map bypasses Go's automatic MIME header canonicalization that
	// would otherwise rewrite it to "Api_Key".
	req.Header["API_KEY"] = []string{apiKey}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, err
}

func manusCreateTask() {
	body, err := io.ReadAll(os.Stdin)
	if err != nil || len(bytes.TrimSpace(body)) == 0 {
		errOut("manus create-task: JSON body required on stdin")
	}
	_, base := manusCreds()
	raw, code, err := manusDo("POST", base+"/v1/tasks", body)
	if err != nil {
		errOut("failed to create task — check your connection")
	}
	rawJSONPassthrough(raw, code)
}

func manusGetTask(args []string) {
	if len(args) == 0 {
		errOut("usage: manus get-task <task_id>")
	}
	_, base := manusCreds()
	taskID := args[0]

	// Match bash manus_get_task.sh: poll until completed / failed / timeout.
	for i := 0; i < manusMaxRetries; i++ {
		raw, code, err := manusDo("GET", base+"/v1/tasks/"+taskID, nil)
		if err != nil {
			errOut("failed to check task status — check your connection")
		}
		if code >= 400 {
			errOut("failed to check task status — the server rejected the request")
		}
		var task manusTaskResponse
		if err := json.Unmarshal(raw, &task); err != nil {
			errOut("received an unexpected response while checking task status")
		}

		switch task.Status {
		case "completed":
			pdfURL := strings.TrimSpace(firstPDFURL(task))
			if pdfURL == "" {
				errOut("the task completed but no PDF URL could be determined")
			}
			okOut(map[string]interface{}{
				"task_id":  taskID,
				"task_url": task.TaskURL,
				"pdf_url":  pdfURL,
				"status":   task.Status,
			})
			return
		case "failed":
			errOut("the task encountered an error and could not be completed")
		}
		time.Sleep(manusPollInterval)
	}
	errOut(fmt.Sprintf("task timed out — generation took longer than %d minutes", manusMaxRetries*int(manusPollInterval.Seconds())/60))
}

type manusTaskResponse struct {
	TaskID  string `json:"task_id"`
	ID      string `json:"id"`
	TaskURL string `json:"task_url"`
	Status  string `json:"status"`
	Output  []struct {
		Content []struct {
			FileName string `json:"fileName"`
			FileURL  string `json:"fileUrl"`
			MimeType string `json:"mimeType"`
		} `json:"content"`
	} `json:"output"`
}

// buildProposalTask constructs the Manus task payload (prompt + style PDF
// attachment) from an outline file path or "-" for stdin.
func buildProposalTask(company, outlinePath string) map[string]interface{} {
	var outline []byte
	var err error
	if outlinePath == "-" {
		outline, err = io.ReadAll(os.Stdin)
	} else {
		outline, err = os.ReadFile(outlinePath)
	}
	if err != nil || len(bytes.TrimSpace(outline)) == 0 {
		errOut("outline is empty")
	}

	styleBytes, err := manusTemplates.ReadFile("templates/style_template.pdf")
	if err != nil {
		errOut("a required template file is missing — please reinstall the skill")
	}
	styleB64 := base64.StdEncoding.EncodeToString(styleBytes)

	prompt := fmt.Sprintf("%s\n\nOutline:\n\n%s\n\nOutput: 1 file PDF tên %s_proposal.pdf",
		manusProposalPrompt, string(outline), company)

	return map[string]interface{}{
		"prompt":       prompt,
		"agentProfile": "manus-1.6",
		"attachments": []map[string]interface{}{
			{
				"type":      "file",
				"file_name": "style_template.pdf",
				"file_data": "data:application/pdf;base64," + styleB64,
			},
		},
	}
}

// manusSubmitProposal creates the Manus task and returns task_id immediately —
// no polling. Callers (e.g. sme-cli proposal generate) poll via get-task on
// their own schedule to stay within bot tool execution timeouts.
func manusSubmitProposal(args []string) {
	if len(args) < 2 {
		errOut("usage: manus submit-proposal <company_name> <outline_file|->")
	}
	task := buildProposalTask(args[0], args[1])
	body, err := json.Marshal(task)
	if err != nil {
		errOut("failed to prepare the proposal request")
	}
	_, base := manusCreds()
	raw, code, err := manusDo("POST", base+"/v1/tasks", body)
	if err != nil {
		errOut("failed to submit the proposal task — check your connection")
	}
	if code >= 400 {
		errOut("the server rejected the proposal request — verify your API key")
	}
	var created manusTaskResponse
	if err := json.Unmarshal(raw, &created); err != nil {
		errOut("received an unexpected response after creating the task")
	}
	taskID := created.TaskID
	if taskID == "" {
		taskID = created.ID
	}
	if taskID == "" {
		errOut("the server did not return a valid task reference")
	}
	okOut(map[string]interface{}{
		"task_id":  taskID,
		"task_url": created.TaskURL,
		"status":   "submitted",
		"poll_cmd": "sme-cli manus get-task " + taskID,
	})
}

func manusGenerateProposal(args []string) {
	if len(args) < 2 {
		errOut("usage: manus generate-proposal <company_name> <outline_file|->")
	}
	task := buildProposalTask(args[0], args[1])
	body, err := json.Marshal(task)
	if err != nil {
		errOut("failed to prepare the proposal request")
	}

	_, base := manusCreds()
	raw, code, err := manusDo("POST", base+"/v1/tasks", body)
	if err != nil {
		errOut("failed to submit the proposal task — check your connection")
	}
	if code >= 400 {
		errOut("the server rejected the proposal request — verify your API key")
	}

	var created manusTaskResponse
	if err := json.Unmarshal(raw, &created); err != nil {
		errOut("received an unexpected response after creating the task")
	}
	taskID := created.TaskID
	if taskID == "" {
		taskID = created.ID
	}
	if taskID == "" {
		errOut("the server did not return a valid task reference")
	}

	for i := 0; i < manusMaxRetries; i++ {
		pollRaw, pollCode, err := manusDo("GET", base+"/v1/tasks/"+taskID, nil)
		if err != nil {
			errOut("failed to check proposal progress — check your connection")
		}
		if pollCode >= 400 {
			errOut("the server returned an error while checking proposal progress")
		}
		var task manusTaskResponse
		if err := json.Unmarshal(pollRaw, &task); err != nil {
			errOut("received an unexpected response while checking proposal status")
		}

		switch task.Status {
		case "completed":
			pdfURL := strings.TrimSpace(firstPDFURL(task))
			if pdfURL == "" {
				errOut("the proposal task completed but no PDF URL could be determined")
			}
			okOut(map[string]interface{}{
				"task_id":  taskID,
				"task_url": task.TaskURL,
				"pdf_url":  pdfURL,
				"status":   task.Status,
			})
			return
		case "failed":
			errOut("the proposal task failed — please try again or adjust the outline")
		}
		time.Sleep(manusPollInterval)
	}

	// Polling window exhausted but task is still running on Manus. Return
	// success with status=pending and the task_id so the caller can resume
	// polling via `sme-cli manus get-task <task_id>` instead of submitting
	// a new task and burning credits on a duplicate generation.
	okOut(map[string]interface{}{
		"task_id":  taskID,
		"status":   "pending",
		"message":  fmt.Sprintf("Manus dang xu ly (qua %d phut). KHONG tao task moi — dung 'sme-cli manus get-task %s' de check lai sau 1-2 phut.", manusMaxRetries*int(manusPollInterval.Seconds())/60, taskID),
		"poll_cmd": "sme-cli manus get-task " + taskID,
	})
}

func firstPDFURL(t manusTaskResponse) string {
	for _, o := range t.Output {
		for _, c := range o.Content {
			// Skip the style reference PDF that we attached as input — Manus
			// echoes it back in the task output alongside the generated file.
			if c.FileName == "style_template.pdf" {
				continue
			}
			if c.MimeType == "application/pdf" || strings.HasSuffix(c.FileName, ".pdf") {
				return c.FileURL
			}
		}
	}
	return ""
}

func manusPrintTemplate(args []string) {
	if len(args) == 0 {
		errOut("usage: manus template <name>   (ai-agent|consulting|custom-dev|managed-services|saas)")
	}
	name := args[0]
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}
	data, err := manusTemplates.ReadFile("templates/" + name)
	if err != nil {
		errOut("template not found: " + args[0])
	}
	os.Stdout.Write(data)
}

func manusListTemplates() {
	entries, err := manusTemplates.ReadDir("templates")
	if err != nil {
		errOut(err.Error())
	}
	names := []string{}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			names = append(names, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	okOut(map[string]interface{}{"templates": names})
}
