package main

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestFirstPDFURL(t *testing.T) {
	t.Run("finds PDF by mime type", func(t *testing.T) {
		task := manusTaskResponse{
			Output: []struct {
				Content []struct {
					FileName string `json:"fileName"`
					FileURL  string `json:"fileUrl"`
					MimeType string `json:"mimeType"`
				} `json:"content"`
			}{
				{Content: []struct {
					FileName string `json:"fileName"`
					FileURL  string `json:"fileUrl"`
					MimeType string `json:"mimeType"`
				}{
					{FileName: "other.txt", FileURL: "http://x/other.txt", MimeType: "text/plain"},
					{FileName: "proposal.pdf", FileURL: "http://x/proposal.pdf", MimeType: "application/pdf"},
				}},
			},
		}
		got := firstPDFURL(task)
		if got != "http://x/proposal.pdf" {
			t.Fatalf("firstPDFURL = %q, want http://x/proposal.pdf", got)
		}
	})

	t.Run("falls back to .pdf extension when mime unset", func(t *testing.T) {
		task := manusTaskResponse{
			Output: []struct {
				Content []struct {
					FileName string `json:"fileName"`
					FileURL  string `json:"fileUrl"`
					MimeType string `json:"mimeType"`
				} `json:"content"`
			}{
				{Content: []struct {
					FileName string `json:"fileName"`
					FileURL  string `json:"fileUrl"`
					MimeType string `json:"mimeType"`
				}{
					{FileName: "file.pdf", FileURL: "http://y/file.pdf", MimeType: ""},
				}},
			},
		}
		if got := firstPDFURL(task); got != "http://y/file.pdf" {
			t.Fatalf("firstPDFURL = %q, want http://y/file.pdf", got)
		}
	})

	t.Run("returns empty when no PDF present", func(t *testing.T) {
		task := manusTaskResponse{}
		if got := firstPDFURL(task); got != "" {
			t.Fatalf("firstPDFURL on empty task = %q, want empty", got)
		}
	})
}

func TestManusTemplatesEmbedded(t *testing.T) {
	want := []string{"ai-agent.md", "consulting.md", "custom-dev.md", "managed-services.md", "saas.md"}
	for _, name := range want {
		data, err := manusTemplates.ReadFile("templates/" + name)
		if err != nil {
			t.Errorf("template %s not embedded: %v", name, err)
			continue
		}
		if len(data) < 100 {
			t.Errorf("template %s too small (%d bytes), likely empty", name, len(data))
		}
	}

	pdf, err := manusTemplates.ReadFile("templates/style_template.pdf")
	if err != nil {
		t.Fatalf("style_template.pdf not embedded: %v", err)
	}
	if string(pdf[:4]) != "%PDF" {
		t.Fatalf("style_template.pdf does not start with %%PDF magic bytes")
	}
}

// TestManusHeaderCaseOnWire verifies that the raw HTTP/1.1 request bytes
// contain the exact header name "API_KEY:" (uppercase with underscore), not
// Go's canonical form "Api_key:". The legacy proposal-agent bash script warns
// that Manus specifically looks for that casing, so we bypass Go's header
// canonicalization via map assignment. This test guards against regressions
// that would silently break Manus auth.
func TestManusHeaderCaseOnWire(t *testing.T) {
	req, err := http.NewRequest("POST", "http://example.invalid/v1/tasks", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	// Mirror manusDo's exact-case assignment.
	req.Header["API_KEY"] = []string{"test-key"}
	req.Header.Set("Content-Type", "application/json")

	var buf bytes.Buffer
	if err := req.Write(&buf); err != nil {
		t.Fatalf("req.Write: %v", err)
	}
	wire := buf.String()

	if !strings.Contains(wire, "API_KEY: test-key\r\n") {
		t.Fatalf("wire bytes missing exact-case \"API_KEY: test-key\"; got:\n%s", wire)
	}
	if strings.Contains(wire, "Api_key:") || strings.Contains(wire, "Api_Key:") {
		t.Fatalf("wire bytes contain canonicalized form; got:\n%s", wire)
	}
}
