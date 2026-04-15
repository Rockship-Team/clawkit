package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSchemaMultiTable(t *testing.T) {
	dir := t.TempDir()
	schema := `{
		"tables": {
			"orders": {
				"fields": [
					{"name": "id", "type": "integer", "auto": "increment"},
					{"name": "contact_id", "type": "integer", "ref": "contacts"},
					{"name": "total", "type": "integer", "role": "price"}
				],
				"statuses": ["pending", "confirmed", "shipped"]
			},
			"contacts": {
				"fields": [
					{"name": "id", "type": "integer", "auto": "increment"},
					{"name": "name", "type": "text"}
				]
			}
		},
		"primary": "orders",
		"timezone": "Asia/Ho_Chi_Minh"
	}`
	os.WriteFile(filepath.Join(dir, "schema.json"), []byte(schema), 0644)

	s, err := loadSchema(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Tables) != 2 {
		t.Errorf("tables: got %d, want 2", len(s.Tables))
	}
	if s.Primary != "orders" {
		t.Errorf("primary: got %q, want %q", s.Primary, "orders")
	}
	if len(s.Tables["orders"].Fields) != 3 {
		t.Errorf("orders fields: got %d, want 3", len(s.Tables["orders"].Fields))
	}
	if s.Tables["orders"].Fields[1].Ref != "contacts" {
		t.Errorf("ref: got %q, want %q", s.Tables["orders"].Fields[1].Ref, "contacts")
	}
}

func TestLoadSchemaLegacySingleTable(t *testing.T) {
	dir := t.TempDir()
	schema := `{
		"table": "orders",
		"fields": [
			{"name": "id", "type": "integer", "auto": "increment"},
			{"name": "status", "type": "text", "default": "new", "role": "status"}
		],
		"statuses": ["new", "completed"]
	}`
	os.WriteFile(filepath.Join(dir, "schema.json"), []byte(schema), 0644)

	s, err := loadSchema(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Legacy format should be normalized to multi-table.
	if s.Primary != "orders" {
		t.Errorf("primary: got %q, want %q", s.Primary, "orders")
	}
	if len(s.Tables) != 1 {
		t.Errorf("tables: got %d, want 1", len(s.Tables))
	}
	if len(s.Tables["orders"].Fields) != 2 {
		t.Errorf("fields: got %d, want 2", len(s.Tables["orders"].Fields))
	}
}

func TestLoadSchemaNotPresent(t *testing.T) {
	dir := t.TempDir()
	s, err := loadSchema(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Error("expected nil schema when schema.json is absent")
	}
}

func TestValidateSchema(t *testing.T) {
	tests := []struct {
		name    string
		schema  Schema
		wantErr bool
	}{
		{
			name: "valid multi-table",
			schema: Schema{
				Primary: "orders",
				Tables: map[string]TableDef{
					"orders": {
						Fields: []SchemaField{
							{Name: "id", Type: "integer", Auto: "increment"},
							{Name: "total", Type: "integer"},
						},
						Statuses: []string{"new", "done"},
					},
					"contacts": {
						Fields: []SchemaField{
							{Name: "id", Type: "integer", Auto: "increment"},
							{Name: "name", Type: "text"},
						},
					},
				},
			},
		},
		{
			name:    "no tables",
			schema:  Schema{Primary: "x"},
			wantErr: true,
		},
		{
			name: "primary not found",
			schema: Schema{
				Primary: "missing",
				Tables: map[string]TableDef{
					"orders": {Fields: []SchemaField{{Name: "id", Type: "integer", Auto: "increment"}}},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid type in table",
			schema: Schema{
				Primary: "t",
				Tables: map[string]TableDef{
					"t": {Fields: []SchemaField{
						{Name: "id", Type: "integer", Auto: "increment"},
						{Name: "bad", Type: "boolean"},
					}},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate role in table",
			schema: Schema{
				Primary: "t",
				Tables: map[string]TableDef{
					"t": {Fields: []SchemaField{
						{Name: "id", Type: "integer", Auto: "increment"},
						{Name: "a", Type: "text", Role: "owner"},
						{Name: "b", Type: "text", Role: "owner"},
					}},
				},
			},
			wantErr: true,
		},
		{
			name: "no auto increment",
			schema: Schema{
				Primary: "t",
				Tables: map[string]TableDef{
					"t": {Fields: []SchemaField{{Name: "name", Type: "text"}}},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSchema(&tc.schema)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMergeSchemaExtendMultiTable(t *testing.T) {
	base := &Schema{
		Timezone: "UTC",
		Primary:  "orders",
		Tables: map[string]TableDef{
			"orders": {
				Fields: []SchemaField{
					{Name: "id", Type: "integer", Auto: "increment"},
					{Name: "total", Type: "integer"},
				},
			},
		},
	}

	profile := &Schema{
		Extend:   true,
		Timezone: "Asia/Ho_Chi_Minh",
		Tables: map[string]TableDef{
			"orders": {
				// Extend orders with new field.
				Fields: []SchemaField{
					{Name: "total", Type: "integer"}, // duplicate, skip
					{Name: "note", Type: "text"},     // new, add
				},
			},
			"contacts": {
				// Entirely new table.
				Fields: []SchemaField{
					{Name: "id", Type: "integer", Auto: "increment"},
					{Name: "name", Type: "text"},
				},
			},
		},
	}

	merged := mergeSchema(base, profile)

	if merged.Timezone != "Asia/Ho_Chi_Minh" {
		t.Errorf("timezone not overridden: got %q", merged.Timezone)
	}
	if len(merged.Tables) != 2 {
		t.Errorf("tables: got %d, want 2", len(merged.Tables))
	}
	// orders should have 3 fields (2 base + 1 new).
	if len(merged.Tables["orders"].Fields) != 3 {
		t.Errorf("orders fields: got %d, want 3", len(merged.Tables["orders"].Fields))
	}
	// contacts should be added.
	if _, ok := merged.Tables["contacts"]; !ok {
		t.Error("contacts table should be added from profile")
	}
}

func TestMergeSchemaReplace(t *testing.T) {
	base := &Schema{
		Primary: "old",
		Tables: map[string]TableDef{
			"old": {Fields: []SchemaField{{Name: "id", Type: "integer", Auto: "increment"}}},
		},
	}
	profile := &Schema{
		Primary: "new_table",
		Tables: map[string]TableDef{
			"new_table": {Fields: []SchemaField{
				{Name: "id", Type: "integer", Auto: "increment"},
				{Name: "title", Type: "text"},
			}},
		},
	}

	merged := mergeSchema(base, profile)
	if merged.Primary != "new_table" {
		t.Errorf("primary: got %q, want %q", merged.Primary, "new_table")
	}
	if len(merged.Tables) != 1 {
		t.Errorf("tables: got %d, want 1", len(merged.Tables))
	}
}

func TestInitLocalDBMultiTable(t *testing.T) {
	dir := t.TempDir()
	s := &Schema{
		Primary: "orders",
		Tables: map[string]TableDef{
			"orders":   {Fields: []SchemaField{{Name: "id", Type: "integer", Auto: "increment"}}},
			"contacts": {Fields: []SchemaField{{Name: "id", Type: "integer", Auto: "increment"}}},
		},
	}

	if err := initLocalDB(dir, s); err != nil {
		t.Fatal(err)
	}

	for _, tname := range []string{"orders", "contacts"} {
		data, err := os.ReadFile(filepath.Join(dir, tname+".json"))
		if err != nil {
			t.Fatalf("%s.json not created: %v", tname, err)
		}
		if string(data) != "[]\n" {
			t.Errorf("%s.json: got %q, want empty array", tname, string(data))
		}
	}
}

func TestInitLocalDBIdempotent(t *testing.T) {
	dir := t.TempDir()
	s := &Schema{
		Primary: "orders",
		Tables: map[string]TableDef{
			"orders": {Fields: []SchemaField{{Name: "id", Type: "integer", Auto: "increment"}}},
		},
	}

	existing := `[{"id":1}]`
	os.WriteFile(filepath.Join(dir, "orders.json"), []byte(existing), 0644)

	if err := initLocalDB(dir, s); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "orders.json"))
	if string(data) != existing {
		t.Errorf("existing data was overwritten: got %q", string(data))
	}
}

func TestApplySchemaOverlayExtend(t *testing.T) {
	dir := t.TempDir()

	base := Schema{
		Primary: "orders",
		Tables: map[string]TableDef{
			"orders": {Fields: []SchemaField{
				{Name: "id", Type: "integer", Auto: "increment"},
				{Name: "name", Type: "text"},
			}},
		},
	}
	baseData, _ := json.MarshalIndent(base, "", "  ")
	basePath := filepath.Join(dir, "schema.json")
	os.WriteFile(basePath, baseData, 0644)

	profile := Schema{
		Extend: true,
		Tables: map[string]TableDef{
			"orders": {Fields: []SchemaField{
				{Name: "phone", Type: "text"},
			}},
		},
	}
	profileData, _ := json.MarshalIndent(profile, "", "  ")
	profilePath := filepath.Join(dir, "profile-schema.json")
	os.WriteFile(profilePath, profileData, 0644)

	if err := applySchemaOverlay(basePath, profilePath); err != nil {
		t.Fatal(err)
	}

	result, _ := os.ReadFile(basePath)
	var merged Schema
	json.Unmarshal(result, &merged)
	merged.normalize()

	if len(merged.Tables["orders"].Fields) != 3 {
		t.Errorf("merged fields: got %d, want 3", len(merged.Tables["orders"].Fields))
	}
	if merged.Extend {
		t.Error("extend flag should be cleared in output")
	}
}
