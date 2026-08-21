package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- extractLinks ---

func TestExtractLinksNone(t *testing.T) {
	links := extractLinks("No links here, just plain text.")
	if len(links) != 0 {
		t.Errorf("got %v, want empty", links)
	}
}

func TestExtractLinksBasic(t *testing.T) {
	links := extractLinks("See [[ProjectAlpha]] and [[Meeting Notes]].")
	if len(links) != 2 {
		t.Fatalf("got %d links, want 2: %v", len(links), links)
	}
	if links[0] != "ProjectAlpha" {
		t.Errorf("links[0] = %q, want %q", links[0], "ProjectAlpha")
	}
	if links[1] != "Meeting Notes" {
		t.Errorf("links[1] = %q, want %q", links[1], "Meeting Notes")
	}
}

func TestExtractLinksWithAlias(t *testing.T) {
	// [[Target|Display]] — should return Target
	links := extractLinks("Click [[RealNote|nice label]] here.")
	if len(links) != 1 || links[0] != "RealNote" {
		t.Errorf("got %v, want [RealNote]", links)
	}
}

func TestExtractLinksDuplicate(t *testing.T) {
	// extractLinks does NOT deduplicate — returns all occurrences
	links := extractLinks("[[Alpha]] and [[Alpha]] again.")
	if len(links) != 2 {
		t.Errorf("got %d, want 2", len(links))
	}
}

// --- extractTags ---

func TestExtractTagsNone(t *testing.T) {
	tags := extractTags("No tags here.")
	if len(tags) != 0 {
		t.Errorf("got %v, want empty", tags)
	}
}

func TestExtractTagsBasic(t *testing.T) {
	tags := extractTags("This is #finance and #payroll work.")
	if len(tags) != 2 {
		t.Fatalf("got %d tags, want 2: %v", len(tags), tags)
	}
	if tags[0] != "finance" {
		t.Errorf("tags[0] = %q, want %q", tags[0], "finance")
	}
	if tags[1] != "payroll" {
		t.Errorf("tags[1] = %q, want %q", tags[1], "payroll")
	}
}

func TestExtractTagsSlash(t *testing.T) {
	// Tags can have / for nested categories
	tags := extractTags("Tagged as #project/active here.")
	if len(tags) == 0 || tags[0] != "project/active" {
		t.Errorf("got %v, want [project/active]", tags)
	}
}

func TestExtractTagsAtLineStart(t *testing.T) {
	tags := extractTags("#meeting\nsome content\n#daily")
	if len(tags) != 2 {
		t.Errorf("got %d tags, want 2: %v", len(tags), tags)
	}
}

// --- readNote / writeNote ---

func TestWriteAndReadNote(t *testing.T) {
	vault := t.TempDir()
	content := "---\ntitle: Test\n---\nHello world\n"

	if err := writeNote(vault, "notes/hello.md", content); err != nil {
		t.Fatalf("writeNote failed: %v", err)
	}

	got, err := readNote(vault, "notes/hello.md")
	if err != nil {
		t.Fatalf("readNote failed: %v", err)
	}
	if got != content {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestWriteNoteCreatesParentDirs(t *testing.T) {
	vault := t.TempDir()
	if err := writeNote(vault, "deep/nested/dir/note.md", "content"); err != nil {
		t.Fatalf("writeNote should create dirs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vault, "deep/nested/dir/note.md")); err != nil {
		t.Errorf("file not found after write: %v", err)
	}
}

func TestReadNoteNotFound(t *testing.T) {
	vault := t.TempDir()
	_, err := readNote(vault, "nonexistent.md")
	if err == nil {
		t.Error("expected error for missing note, got nil")
	}
}

// --- listNotes ---

func TestListNotesEmpty(t *testing.T) {
	vault := t.TempDir()
	notes, err := listNotes(vault)
	if err != nil {
		t.Fatalf("listNotes failed: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("got %v, want empty", notes)
	}
}

func TestListNotesBasic(t *testing.T) {
	vault := t.TempDir()
	writeNote(vault, "a.md", "content a")
	writeNote(vault, "sub/b.md", "content b")
	writeNote(vault, "sub/c.md", "content c")

	notes, err := listNotes(vault)
	if err != nil {
		t.Fatalf("listNotes failed: %v", err)
	}
	if len(notes) != 3 {
		t.Fatalf("got %d notes, want 3: %v", len(notes), notes)
	}
}

func TestListNotesSkipsObsidian(t *testing.T) {
	vault := t.TempDir()
	writeNote(vault, "real.md", "content")
	// Create .obsidian dir with a .md file — should be skipped
	obsidianDir := filepath.Join(vault, ".obsidian")
	os.MkdirAll(obsidianDir, 0755)
	os.WriteFile(filepath.Join(obsidianDir, "config.md"), []byte("ignore me"), 0644)

	notes, err := listNotes(vault)
	if err != nil {
		t.Fatalf("listNotes failed: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1 (should skip .obsidian): %v", len(notes), notes)
	}
}

func TestListNotesSkipsNonMarkdown(t *testing.T) {
	vault := t.TempDir()
	writeNote(vault, "real.md", "content")
	os.WriteFile(filepath.Join(vault, "image.png"), []byte("binary"), 0644)
	os.WriteFile(filepath.Join(vault, "data.json"), []byte("{}"), 0644)

	notes, err := listNotes(vault)
	if err != nil {
		t.Fatalf("listNotes failed: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("got %d notes, want 1 (only .md): %v", len(notes), notes)
	}
}

// --- vnToday ---

func TestVnTodayFormat(t *testing.T) {
	today := vnToday()
	// Must be YYYY-MM-DD
	parts := strings.Split(today, "-")
	if len(parts) != 3 {
		t.Fatalf("vnToday = %q: expected YYYY-MM-DD format", today)
	}
	if len(parts[0]) != 4 {
		t.Errorf("year = %q: expected 4 digits", parts[0])
	}
	if len(parts[1]) != 2 {
		t.Errorf("month = %q: expected 2 digits", parts[1])
	}
	if len(parts[2]) != 2 {
		t.Errorf("day = %q: expected 2 digits", parts[2])
	}
}

func TestVnNowISO(t *testing.T) {
	iso := vnNowISO()
	// Must contain T and +07:00
	if !strings.Contains(iso, "T") {
		t.Errorf("vnNowISO = %q: missing T separator", iso)
	}
	if !strings.Contains(iso, "+07:00") && !strings.Contains(iso, "Z") {
		t.Errorf("vnNowISO = %q: missing timezone", iso)
	}
}

// --- newID ---

func TestNewIDFormat(t *testing.T) {
	id := newID()
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Fatalf("newID = %q: expected UUID v4 format (5 parts)", id)
	}
	lengths := []int{8, 4, 4, 4, 12}
	for i, p := range parts {
		if len(p) != lengths[i] {
			t.Errorf("part[%d] = %q: expected %d hex chars", i, p, lengths[i])
		}
	}
}

func TestNewIDUnique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := newID()
		if ids[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}
