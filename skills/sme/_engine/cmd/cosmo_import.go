package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Import commands:
//   sme-cli cosmo import-txt <file> [--list-id UUID] [--source STRING]
//   sme-cli cosmo import-csv <file> [--list-id UUID] [--source STRING] [--format luma|generic]
//
// Text format: "Name — email" or "Name - email" per line.
// Luma CSV columns: api_id,name,first_name,last_name,email,phone_number,
//                   approval_status,...,ticket_name

type importedContact map[string]interface{}

type importOpts struct {
	ListID string
	Source string
	Format string
}

func parseImportFlags(args []string) (file string, opts importOpts, rest []string) {
	if len(args) == 0 {
		return "", opts, nil
	}
	file = args[0]
	opts.Format = "luma"
	opts.Source = "cli_import"
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--list-id":
			if i+1 < len(args) {
				opts.ListID = args[i+1]
				i++
			}
		case "--source":
			if i+1 < len(args) {
				opts.Source = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				opts.Format = args[i+1]
				i++
			}
		default:
			rest = append(rest, args[i])
		}
	}
	return
}

func cosmoImportTXT(args []string) {
	file, opts, _ := parseImportFlags(args)
	if file == "" {
		errOut("usage: cosmo import-txt <file> [--list-id UUID] [--source STRING]")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		errOut("read file: " + err.Error())
	}
	contacts, parseErrs := parseTXTContacts(string(raw), opts.Source)
	if len(contacts) == 0 {
		errOut(fmt.Sprintf("no valid contacts parsed (errors: %v)", parseErrs))
	}
	report := bulkImport(contacts, opts, parseErrs)
	okOut(report)
}

func cosmoImportCSV(args []string) {
	file, opts, _ := parseImportFlags(args)
	if file == "" {
		errOut("usage: cosmo import-csv <file> [--list-id UUID] [--source STRING] [--format luma|generic]")
	}
	f, err := os.Open(file)
	if err != nil {
		errOut("open file: " + err.Error())
	}
	defer f.Close()

	var contacts []importedContact
	var parseErrs []string
	switch opts.Format {
	case "luma", "":
		contacts, parseErrs = parseLumaCSV(f, opts.Source)
	case "generic":
		contacts, parseErrs = parseGenericCSV(f, opts.Source)
	default:
		errOut("unknown --format: " + opts.Format + " (supported: luma, generic)")
	}
	if len(contacts) == 0 {
		errOut(fmt.Sprintf("no valid contacts parsed (errors: %v)", parseErrs))
	}
	report := bulkImport(contacts, opts, parseErrs)
	okOut(report)
}

// parseTXTContacts handles "Name — email" or "Name - email" lines. Skips
// empty lines and lines starting with '#'.
func parseTXTContacts(text, source string) ([]importedContact, []string) {
	var out []importedContact
	var errs []string
	for ln, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, email, ok := splitTXTLine(line)
		if !ok {
			errs = append(errs, fmt.Sprintf("line %d: no delimiter (— or -)", ln+1))
			continue
		}
		if !strings.Contains(email, "@") {
			errs = append(errs, fmt.Sprintf("line %d: invalid email %q", ln+1, email))
			continue
		}
		out = append(out, importedContact{
			"name":   name,
			"email":  email,
			"source": source,
		})
	}
	return out, errs
}

func splitTXTLine(line string) (name, email string, ok bool) {
	for _, sep := range []string{"—", "–", " - ", " — "} {
		if i := strings.Index(line, sep); i >= 0 {
			return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+len(sep):]), true
		}
	}
	if i := strings.LastIndex(line, "-"); i > 0 {
		return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
	}
	return "", "", false
}

// parseLumaCSV reads Luma-style CSV and builds contact payloads mirroring
// scripts/parse_event_csv.py in bot-cosmo.
func parseLumaCSV(r io.Reader, source string) ([]importedContact, []string) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, []string{"read header: " + err.Error()}
	}
	idx := buildCSVIndex(header)
	var out []importedContact
	var errs []string
	row := 1
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		row++
		if err != nil {
			errs = append(errs, fmt.Sprintf("row %d: %v", row, err))
			continue
		}
		email := strings.TrimSpace(csvGet(rec, idx, "email"))
		if email == "" || !strings.Contains(email, "@") {
			continue
		}
		first := strings.TrimSpace(csvGet(rec, idx, "first_name"))
		last := strings.TrimSpace(csvGet(rec, idx, "last_name"))
		nameField := strings.TrimSpace(csvGet(rec, idx, "name"))

		var name string
		switch {
		case first != "" && last != "":
			name = first + " " + last
		case nameField != "":
			name = nameField
		case first != "":
			name = first
		case last != "":
			name = last
		default:
			name = "Unknown"
		}

		contact := importedContact{
			"name":   name,
			"email":  email,
			"source": source,
		}
		if phone := strings.TrimSpace(csvGet(rec, idx, "phone_number")); phone != "" {
			contact["phone"] = phone
		}
		custom := map[string]string{}
		if v := strings.TrimSpace(csvGet(rec, idx, "approval_status")); v != "" {
			custom["approval_status"] = v
		}
		if v := strings.TrimSpace(csvGet(rec, idx, "ticket_name")); v != "" {
			custom["ticket_type"] = v
		}
		if v := strings.TrimSpace(csvGet(rec, idx, "utm_source")); v != "" {
			custom["utm_source"] = v
		}
		contact["custom_fields"] = custom
		out = append(out, contact)
	}
	return out, errs
}

// parseGenericCSV maps header columns 1:1 into contact fields (lowercased,
// spaces replaced with underscores). email/name/phone flow to top-level,
// the rest go to custom_fields.
func parseGenericCSV(r io.Reader, source string) ([]importedContact, []string) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, []string{"read header: " + err.Error()}
	}
	normalized := make([]string, len(header))
	for i, h := range header {
		normalized[i] = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(h), " ", "_"))
	}
	var out []importedContact
	var errs []string
	topLevel := map[string]bool{"name": true, "email": true, "phone": true, "company": true, "job_title": true}
	row := 1
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		row++
		if err != nil {
			errs = append(errs, fmt.Sprintf("row %d: %v", row, err))
			continue
		}
		contact := importedContact{"source": source}
		custom := map[string]string{}
		for i, col := range normalized {
			if i >= len(rec) {
				break
			}
			val := strings.TrimSpace(rec[i])
			if val == "" {
				continue
			}
			if topLevel[col] {
				contact[col] = val
			} else {
				custom[col] = val
			}
		}
		email, _ := contact["email"].(string)
		if email == "" || !strings.Contains(email, "@") {
			continue
		}
		if _, ok := contact["name"]; !ok {
			contact["name"] = "Unknown"
		}
		contact["custom_fields"] = custom
		out = append(out, contact)
	}
	return out, errs
}

func buildCSVIndex(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		m[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return m
}

func csvGet(rec []string, idx map[string]int, col string) string {
	i, ok := idx[col]
	if !ok || i >= len(rec) {
		return ""
	}
	return rec[i]
}

// bulkImport POSTs contacts to /v1/contacts/bulk and optionally assigns
// created IDs to a contact list.
func bulkImport(contacts []importedContact, opts importOpts, parseErrs []string) map[string]interface{} {
	body, err := json.Marshal(contacts)
	if err != nil {
		errOut("encode contacts: " + err.Error())
	}
	raw, code, err := cosmoRequest("POST", "/v1/contacts/bulk", body)
	if err != nil {
		errOut("bulk POST: " + err.Error())
	}
	if code >= 400 {
		errOut(fmt.Sprintf("bulk POST failed (HTTP %d): %s", code, string(raw)))
	}
	var bulkResp struct {
		Status string                   `json:"status"`
		Data   []map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(raw, &bulkResp)

	createdIDs := make([]string, 0, len(bulkResp.Data))
	for _, item := range bulkResp.Data {
		if id, ok := item["id"].(string); ok && id != "" {
			createdIDs = append(createdIDs, id)
		}
	}

	report := map[string]interface{}{
		"total_parsed":  len(contacts),
		"created_count": len(createdIDs),
		"created_ids":   createdIDs,
		"parse_errors":  parseErrs,
	}

	if opts.ListID != "" && len(createdIDs) > 0 {
		listBody, _ := json.Marshal(map[string]interface{}{"contact_ids": createdIDs})
		listRaw, listCode, listErr := cosmoRequest("PATCH", "/v1/list-contacts/"+opts.ListID, listBody)
		report["list_id"] = opts.ListID
		if listErr != nil || listCode >= 400 {
			report["list_assign_error"] = fmt.Sprintf("failed (HTTP %d): %s — %v", listCode, string(listRaw), listErr)
		} else {
			report["list_assigned"] = true
		}
	}
	return report
}
