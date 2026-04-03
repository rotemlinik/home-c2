package gmail

import (
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{
			name: "shorter than max",
			s:    "hello",
			max:  10,
			want: "hello",
		},
		{
			name: "exactly max length",
			s:    "hello",
			max:  5,
			want: "hello",
		},
		{
			name: "longer than max",
			s:    "hello world",
			max:  5,
			want: "hello...",
		},
		{
			name: "empty string",
			s:    "",
			max:  10,
			want: "",
		},
		{
			name: "max 0",
			s:    "hello",
			max:  0,
			want: "...",
		},
		{
			name: "long body truncated at 500",
			s:    string(make([]byte, 600)),
			max:  500,
			want: string(make([]byte, 500)) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q (len=%d), want %q (len=%d)",
					tt.s, tt.max, got, len(got), tt.want, len(tt.want))
			}
		})
	}
}

func TestExtractBody(t *testing.T) {
	// extractBody depends on gmailv1.MessagePart which is from the google API.
	// We test it with nil input to verify no panic.
	t.Run("nil part returns empty", func(t *testing.T) {
		got := extractBody(nil)
		if got != "" {
			t.Errorf("extractBody(nil) = %q, want empty", got)
		}
	})
}
