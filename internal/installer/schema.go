package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rockship-co/clawkit/internal/config"
	"github.com/rockship-co/clawkit/internal/ui"
)

// Database target constants.
const (
	DBTargetLocal    = "local"
	DBTargetSupabase = "supabase"
	DBTargetAPI      = "api"
)

// SchemaField defines a single field in a table.
type SchemaField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`           // "text" | "integer"
	Auto     string `json:"auto,omitempty"` // "increment" | "timestamp"
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required,omitempty"`
	Role     string `json:"role,omitempty"` // "owner" | "status" | "price" | "timestamp"
	Ref      string `json:"ref,omitempty"`  // reference to another table name
}

// TableDef defines a single table's structure.
type TableDef struct {
	Fields   []SchemaField `json:"fields"`
	Statuses []string      `json:"statuses,omitempty"`
}

// Schema defines the database structure for a skill.
// Supports two formats:
//   - Legacy single-table: {"table":"orders", "fields":[...], "statuses":[...]}
//   - Multi-table:         {"tables":{"orders":{...}, "contacts":{...}}, "primary":"orders"}
type Schema struct {
	// Multi-table format.
	Tables  map[string]TableDef `json:"tables,omitempty"`
	Primary string              `json:"primary,omitempty"`

	// Legacy single-table format (converted to Tables on load).
	Table    string        `json:"table,omitempty"`
	Fields   []SchemaField `json:"fields,omitempty"`
	Statuses []string      `json:"statuses,omitempty"`

	// Shared config.
	Timezone  string `json:"timezone,omitempty"`
	ImagesDir string `json:"images_dir,omitempty"`
	DBTarget  string `json:"db_target,omitempty"` // "local" | "supabase" | "api"
	Extend    bool   `json:"extend,omitempty"`
}

// normalize converts legacy single-table format to multi-table.
// After this call, Tables and Primary are always set.
func (s *Schema) normalize() {
	if len(s.Tables) > 0 {
		if s.Primary == "" {
			// Pick first table as primary if not set.
			for name := range s.Tables {
				s.Primary = name
				break
			}
		}
		return
	}
	// Legacy: convert "table" + "fields" to Tables map.
	if s.Table != "" && len(s.Fields) > 0 {
		s.Tables = map[string]TableDef{
			s.Table: {Fields: s.Fields, Statuses: s.Statuses},
		}
		s.Primary = s.Table
	}
}

// TableNames returns all table names in definition order is not guaranteed (map),
// but primary is always accessible via s.Primary.
func (s *Schema) TableNames() []string {
	names := make([]string, 0, len(s.Tables))
	for name := range s.Tables {
		names = append(names, name)
	}
	return names
}

// loadSchema reads schema.json from skillDir. Returns nil, nil if not present.
func loadSchema(skillDir string) (*Schema, error) {
	data, err := os.ReadFile(filepath.Join(skillDir, "schema.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read schema.json: %w", err)
	}
	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse schema.json: %w", err)
	}
	s.normalize()
	return &s, nil
}

var validTypes = map[string]bool{"text": true, "integer": true}
var validRoles = map[string]bool{"owner": true, "status": true, "price": true, "timestamp": true}
var validAuto = map[string]bool{"increment": true, "timestamp": true, "": true}

// validateSchema checks that the schema is well-formed.
func validateSchema(s *Schema) error {
	if len(s.Tables) == 0 {
		return fmt.Errorf("schema: at least one table is required")
	}
	if s.Primary == "" {
		return fmt.Errorf("schema: primary table name is required")
	}
	if _, ok := s.Tables[s.Primary]; !ok {
		return fmt.Errorf("schema: primary table '%s' not found in tables", s.Primary)
	}

	for tname, tdef := range s.Tables {
		if err := validateTable(tname, &tdef); err != nil {
			return err
		}
	}
	return nil
}

func validateTable(tname string, t *TableDef) error {
	if len(t.Fields) == 0 {
		return fmt.Errorf("schema: table '%s' has no fields", tname)
	}
	if len(t.Statuses) > 0 && len(t.Statuses) < 2 {
		return fmt.Errorf("schema: table '%s' statuses must have at least 2 entries", tname)
	}

	seenRoles := map[string]bool{}
	hasAutoID := false

	for _, f := range t.Fields {
		if f.Name == "" {
			return fmt.Errorf("schema: table '%s' has a field with empty name", tname)
		}
		if !validTypes[f.Type] {
			return fmt.Errorf("schema: %s.%s has invalid type '%s'", tname, f.Name, f.Type)
		}
		if !validAuto[f.Auto] {
			return fmt.Errorf("schema: %s.%s has invalid auto '%s'", tname, f.Name, f.Auto)
		}
		if f.Auto == "increment" {
			hasAutoID = true
		}
		if f.Role != "" {
			if !validRoles[f.Role] {
				return fmt.Errorf("schema: %s.%s has invalid role '%s'", tname, f.Name, f.Role)
			}
			if seenRoles[f.Role] {
				return fmt.Errorf("schema: %s has duplicate role '%s' (field '%s')", tname, f.Role, f.Name)
			}
			seenRoles[f.Role] = true
		}
	}

	if !hasAutoID {
		return fmt.Errorf("schema: table '%s' must have at least one field with auto: increment", tname)
	}
	return nil
}

// mergeSchema merges a profile schema onto a base schema.
// If profile.Extend is true, new tables/fields are added and scalars are overridden.
// Otherwise the profile schema replaces the base entirely.
func mergeSchema(base, profile *Schema) *Schema {
	if !profile.Extend {
		return profile
	}

	merged := *base
	if profile.Timezone != "" {
		merged.Timezone = profile.Timezone
	}
	if profile.ImagesDir != "" {
		merged.ImagesDir = profile.ImagesDir
	}
	if profile.DBTarget != "" {
		merged.DBTarget = profile.DBTarget
	}
	if profile.Primary != "" {
		merged.Primary = profile.Primary
	}

	// Merge tables: add new tables, extend existing tables with new fields.
	if merged.Tables == nil {
		merged.Tables = make(map[string]TableDef)
	}
	for tname, ptable := range profile.Tables {
		if existing, ok := merged.Tables[tname]; ok {
			// Extend existing table: append new fields.
			fieldSet := map[string]bool{}
			for _, f := range existing.Fields {
				fieldSet[f.Name] = true
			}
			for _, f := range ptable.Fields {
				if !fieldSet[f.Name] {
					existing.Fields = append(existing.Fields, f)
				}
			}
			if len(ptable.Statuses) > 0 {
				existing.Statuses = ptable.Statuses
			}
			merged.Tables[tname] = existing
		} else {
			// New table from profile.
			merged.Tables[tname] = ptable
		}
	}

	return &merged
}

// applySchemaOverlay handles schema.json overlay during profile application.
// If the profile schema has extend: true, it merges with the base. Otherwise it replaces.
func applySchemaOverlay(basePath, profilePath string) error {
	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		return fmt.Errorf("read profile schema: %w", err)
	}
	var profile Schema
	if err := json.Unmarshal(profileData, &profile); err != nil {
		return fmt.Errorf("parse profile schema: %w", err)
	}
	profile.normalize()

	if !profile.Extend || !fileExists(basePath) {
		return copyFile(profilePath, basePath)
	}

	// Extend mode: load base, merge, write.
	baseData, err := os.ReadFile(basePath)
	if err != nil {
		return fmt.Errorf("read base schema: %w", err)
	}
	var base Schema
	if err := json.Unmarshal(baseData, &base); err != nil {
		return fmt.Errorf("parse base schema: %w", err)
	}
	base.normalize()

	merged := mergeSchema(&base, &profile)
	merged.Extend = false

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal merged schema: %w", err)
	}
	return os.WriteFile(basePath, out, 0644)
}

// initLocalDB creates empty JSON array files for each table in local storage.
func initLocalDB(skillDir string, s *Schema) error {
	for tname := range s.Tables {
		dbPath := filepath.Join(skillDir, tname+".json")
		if fileExists(dbPath) {
			continue
		}
		if err := os.WriteFile(dbPath, []byte("[]\n"), 0644); err != nil {
			return fmt.Errorf("create %s.json: %w", tname, err)
		}
	}
	return nil
}

// initSchema orchestrates database initialization from schema.json.
func initSchema(skillDir string, cfg *config.SkillConfig, profileValues map[string]string) error {
	s, err := loadSchema(skillDir)
	if err != nil {
		return err
	}
	if s == nil {
		return nil
	}

	if err := validateSchema(s); err != nil {
		return err
	}

	// Determine db_target: profile.yaml > schema.json > "local"
	dbTarget := DBTargetLocal
	if s.DBTarget != "" {
		dbTarget = s.DBTarget
	}
	if profileValues != nil && profileValues["db_target"] != "" {
		dbTarget = profileValues["db_target"]
	}
	dbTarget = strings.TrimSpace(dbTarget)

	cfg.DBTarget = dbTarget
	if cfg.Tokens == nil {
		cfg.Tokens = make(map[string]string)
	}

	switch dbTarget {
	case DBTargetLocal:
		if err := initLocalDB(skillDir, s); err != nil {
			return fmt.Errorf("init local DB: %w", err)
		}
	case DBTargetSupabase:
		if err := collectSupabaseCredentials(cfg); err != nil {
			return err
		}
	case DBTargetAPI:
		if err := collectAPICredentials(cfg); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown db_target: %s (valid: %s, %s, %s)", dbTarget, DBTargetLocal, DBTargetSupabase, DBTargetAPI)
	}

	return nil
}

// collectSupabaseCredentials prompts for Supabase URL and API key.
func collectSupabaseCredentials(cfg *config.SkillConfig) error {
	fmt.Println()
	ui.Info("Supabase database configuration")
	fmt.Println("  Create your tables in Supabase first, then provide the connection details.")
	fmt.Println()

	dbURL := ui.PromptInput("Supabase project URL (e.g. https://xyz.supabase.co)")
	if dbURL == "" {
		return fmt.Errorf("Supabase URL is required")
	}
	dbKey := ui.PromptInput("Supabase anon/service key")
	if dbKey == "" {
		return fmt.Errorf("Supabase API key is required")
	}

	cfg.Tokens["db_url"] = strings.TrimSpace(dbURL)
	cfg.Tokens["db_key"] = strings.TrimSpace(dbKey)
	return nil
}

// collectAPICredentials prompts for a generic REST API endpoint and optional auth.
func collectAPICredentials(cfg *config.SkillConfig) error {
	fmt.Println()
	ui.Info("REST API database configuration")
	fmt.Println("  Provide the base URL for your API.")
	fmt.Println("  cli.js will call <base_url>/<table> for each table.")
	fmt.Println()

	dbURL := ui.PromptInput("API base URL (e.g. https://my-server.com/api)")
	if dbURL == "" {
		return fmt.Errorf("API URL is required")
	}
	cfg.Tokens["db_url"] = strings.TrimSpace(dbURL)

	authHeader := ui.PromptInput("Authorization header (e.g. Bearer xxx) — leave empty to skip")
	if authHeader != "" {
		hdrs := map[string]string{"Authorization": strings.TrimSpace(authHeader)}
		hdrJSON, _ := json.Marshal(hdrs)
		cfg.Tokens["db_headers"] = string(hdrJSON)
	}
	return nil
}
