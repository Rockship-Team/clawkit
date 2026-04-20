package main

import (
	"testing"
)

func TestParseBasicFrontmatter(t *testing.T) {
	content := "---\ntitle: My Note\ntags: journal\n---\nHello world\n"
	meta, body := parseFrontmatter(content)
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
	if meta["title"] != "My Note" {
		t.Errorf("title = %q, want %q", meta["title"], "My Note")
	}
	if meta["tags"] != "journal" {
		t.Errorf("tags = %q, want %q", meta["tags"], "journal")
	}
	if body != "Hello world\n" {
		t.Errorf("body = %q, want %q", body, "Hello world\n")
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	content := "Just some text\nwith no frontmatter.\n"
	meta, body := parseFrontmatter(content)
	if meta != nil {
		t.Errorf("expected nil meta, got %v", meta)
	}
	if body != content {
		t.Errorf("body = %q, want %q", body, content)
	}
}

func TestParseEmptyFrontmatter(t *testing.T) {
	content := "---\n---\nBody here\n"
	meta, body := parseFrontmatter(content)
	if meta == nil {
		t.Fatal("expected non-nil meta for empty frontmatter")
	}
	if len(meta) != 0 {
		t.Errorf("expected empty meta, got %v", meta)
	}
	if body != "Body here\n" {
		t.Errorf("body = %q, want %q", body, "Body here\n")
	}
}

func TestParseQuotedValues(t *testing.T) {
	content := "---\ntitle: \"My Quoted Title\"\nauthor: \"Jane Doe\"\n---\nContent\n"
	meta, body := parseFrontmatter(content)
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
	if meta["title"] != "My Quoted Title" {
		t.Errorf("title = %q, want %q", meta["title"], "My Quoted Title")
	}
	if meta["author"] != "Jane Doe" {
		t.Errorf("author = %q, want %q", meta["author"], "Jane Doe")
	}
	if body != "Content\n" {
		t.Errorf("body = %q, want %q", body, "Content\n")
	}
}

func TestBuildNote(t *testing.T) {
	meta := map[string]string{
		"title": "Test",
		"tags":  "demo",
	}
	body := "Hello\n"
	result := buildNote(meta, body)
	expected := "---\ntags: demo\ntitle: Test\n---\nHello\n"
	if result != expected {
		t.Errorf("buildNote = %q, want %q", result, expected)
	}
}

func TestBuildNoteNilMeta(t *testing.T) {
	body := "Just body\n"
	result := buildNote(nil, body)
	if result != body {
		t.Errorf("buildNote(nil, body) = %q, want %q", result, body)
	}
}

func TestRoundTrip(t *testing.T) {
	meta := map[string]string{
		"title":  "Round Trip",
		"status": "active",
		"tags":   "test",
	}
	body := "This is the body.\n"

	note := buildNote(meta, body)
	parsedMeta, parsedBody := parseFrontmatter(note)

	if parsedMeta == nil {
		t.Fatal("expected non-nil meta after round-trip")
	}
	for k, v := range meta {
		if parsedMeta[k] != v {
			t.Errorf("round-trip meta[%q] = %q, want %q", k, parsedMeta[k], v)
		}
	}
	if len(parsedMeta) != len(meta) {
		t.Errorf("round-trip meta has %d keys, want %d", len(parsedMeta), len(meta))
	}
	if parsedBody != body {
		t.Errorf("round-trip body = %q, want %q", parsedBody, body)
	}
}

func TestUpdateFrontmatterField(t *testing.T) {
	content := "---\ntitle: Original\nstatus: draft\n---\nBody text\n"
	updated := updateFrontmatterField(content, "status", "published")

	meta, body := parseFrontmatter(updated)
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
	if meta["status"] != "published" {
		t.Errorf("status = %q, want %q", meta["status"], "published")
	}
	if meta["title"] != "Original" {
		t.Errorf("title = %q, want %q", meta["title"], "Original")
	}
	if body != "Body text\n" {
		t.Errorf("body = %q, want %q", body, "Body text\n")
	}
}

func TestUpdateFrontmatterFieldAddsNew(t *testing.T) {
	content := "---\ntitle: Hello\n---\nBody\n"
	updated := updateFrontmatterField(content, "tags", "new")

	meta, _ := parseFrontmatter(updated)
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
	if meta["tags"] != "new" {
		t.Errorf("tags = %q, want %q", meta["tags"], "new")
	}
	if meta["title"] != "Hello" {
		t.Errorf("title = %q, want %q", meta["title"], "Hello")
	}
}

func TestUpdateFrontmatterFieldNoExisting(t *testing.T) {
	content := "Just plain text\n"
	updated := updateFrontmatterField(content, "title", "New Title")

	meta, body := parseFrontmatter(updated)
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
	if meta["title"] != "New Title" {
		t.Errorf("title = %q, want %q", meta["title"], "New Title")
	}
	if body != "Just plain text\n" {
		t.Errorf("body = %q, want %q", body, "Just plain text\n")
	}
}
