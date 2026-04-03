package claude

import (
	"encoding/json"
	"strings"
	"testing"
)

// trimFences replicates the markdown fence stripping logic from client.go
func trimFences(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}
	return text
}

func strPtr(s string) *string {
	return &s
}

func TestStripMarkdownFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ExtractedTask
	}{
		{
			name:  "clean JSON array",
			input: `[{"email_id":"abc","title":"Pay bill","category":"Finance","due_date":null,"notes":"Electric bill"}]`,
			want: []ExtractedTask{
				{EmailID: "abc", Title: "Pay bill", Category: "Finance", Notes: "Electric bill"},
			},
		},
		{
			name:  "empty array",
			input: `[]`,
			want:  []ExtractedTask{},
		},
		{
			name:  "wrapped in json fences",
			input: "```json\n[{\"email_id\":\"abc\",\"title\":\"Pay bill\",\"category\":\"Finance\",\"due_date\":null,\"notes\":\"\"}]\n```",
			want: []ExtractedTask{
				{EmailID: "abc", Title: "Pay bill", Category: "Finance"},
			},
		},
		{
			name:  "wrapped in bare fences",
			input: "```\n[{\"email_id\":\"abc\",\"title\":\"Pay bill\",\"category\":\"Finance\",\"due_date\":null,\"notes\":\"\"}]\n```",
			want: []ExtractedTask{
				{EmailID: "abc", Title: "Pay bill", Category: "Finance"},
			},
		},
		{
			name:  "with due_date",
			input: `[{"email_id":"x1","title":"Dentist","category":"Health","due_date":"2025-03-20","notes":"Annual checkup"}]`,
			want: []ExtractedTask{
				{EmailID: "x1", Title: "Dentist", Category: "Health", DueDate: strPtr("2025-03-20"), Notes: "Annual checkup"},
			},
		},
		{
			name:  "multiple tasks",
			input: `[{"email_id":"a","title":"Task A","category":"Car","due_date":null,"notes":""},{"email_id":"b","title":"Task B","category":"Health","due_date":"2025-04-01","notes":"note"}]`,
			want: []ExtractedTask{
				{EmailID: "a", Title: "Task A", Category: "Car"},
				{EmailID: "b", Title: "Task B", Category: "Health", DueDate: strPtr("2025-04-01"), Notes: "note"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := trimFences(tt.input)

			var got []ExtractedTask
			if err := json.Unmarshal([]byte(text), &got); err != nil {
				t.Fatalf("failed to unmarshal: %v\ninput after strip: %q", err, text)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %d tasks, want %d", len(got), len(tt.want))
			}

			for i, g := range got {
				w := tt.want[i]
				if g.EmailID != w.EmailID {
					t.Errorf("[%d] EmailID = %q, want %q", i, g.EmailID, w.EmailID)
				}
				if g.Title != w.Title {
					t.Errorf("[%d] Title = %q, want %q", i, g.Title, w.Title)
				}
				if g.Category != w.Category {
					t.Errorf("[%d] Category = %q, want %q", i, g.Category, w.Category)
				}
				if (g.DueDate == nil) != (w.DueDate == nil) {
					t.Errorf("[%d] DueDate nil mismatch: got %v, want %v", i, g.DueDate, w.DueDate)
				} else if g.DueDate != nil && *g.DueDate != *w.DueDate {
					t.Errorf("[%d] DueDate = %q, want %q", i, *g.DueDate, *w.DueDate)
				}
				if g.Notes != w.Notes {
					t.Errorf("[%d] Notes = %q, want %q", i, g.Notes, w.Notes)
				}
			}
		})
	}
}

func TestMalformedJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "not JSON at all", input: "Here are the tasks I found:", wantErr: true},
		{name: "truncated JSON", input: `[{"email_id":"abc","title":"Pa`, wantErr: true},
		{name: "JSON object instead of array", input: `{"email_id":"abc","title":"Pay"}`, wantErr: true},
		{name: "empty string", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := trimFences(tt.input)
			var tasks []ExtractedTask
			err := json.Unmarshal([]byte(text), &tasks)
			if tt.wantErr && err == nil {
				t.Error("expected unmarshal error, got nil")
			}
		})
	}
}
