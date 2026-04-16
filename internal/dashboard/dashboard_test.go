package dashboard

import (
	"reflect"
	"testing"
)

func TestBuildJSONTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    JSONTableResponse
		wantErr bool
	}{
		{
			name:  "array of objects",
			input: `[{"id":1,"name":"A"},{"name":"B","age":2}]`,
			want: JSONTableResponse{
				Kind:    "array",
				Columns: []string{"id", "name", "age"},
				Rows: []map[string]any{
					{"id": float64(1), "name": "A", "age": nil},
					{"id": nil, "name": "B", "age": float64(2)},
				},
			},
		},
		{
			name:  "array of empty objects",
			input: `[{},{}]`,
			want: JSONTableResponse{
				Kind:    "array",
				Columns: []string{"value"},
				Rows: []map[string]any{
					{"value": map[string]any{}},
					{"value": map[string]any{}},
				},
			},
		},
		{
			name:  "object map",
			input: `{"b":2,"a":"x"}`,
			want: JSONTableResponse{
				Kind:    "object",
				Columns: []string{"key", "value"},
				Rows: []map[string]any{
					{"key": "a", "value": "x"},
					{"key": "b", "value": float64(2)},
				},
			},
		},
		{
			name:  "primitive value",
			input: `42`,
			want: JSONTableResponse{
				Kind:    "value",
				Columns: []string{"value"},
				Rows:    []map[string]any{{"value": float64(42)}},
			},
		},
		{
			name:    "invalid json",
			input:   `{`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := buildJSONTable([]byte(tc.input))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("buildJSONTable() error = %v", err)
			}
			if got.Kind != tc.want.Kind {
				t.Fatalf("Kind = %q, want %q", got.Kind, tc.want.Kind)
			}
			if !reflect.DeepEqual(got.Columns, tc.want.Columns) {
				t.Fatalf("Columns = %#v, want %#v", got.Columns, tc.want.Columns)
			}
			if !reflect.DeepEqual(got.Rows, tc.want.Rows) {
				t.Fatalf("Rows = %#v, want %#v", got.Rows, tc.want.Rows)
			}
		})
	}
}
