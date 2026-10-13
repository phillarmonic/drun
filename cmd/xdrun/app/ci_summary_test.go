package app

import "testing"

func TestNormalizeCISummary(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty means default text", input: "", want: "text"},
		{name: "text", input: "text", want: "text"},
		{name: "json", input: "json", want: "json"},
		{name: "off", input: "off", want: "off"},
		{name: "mixed case is normalized", input: " JSON ", want: "json"},
		{name: "invalid fails", input: "pretty", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeCISummary(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeCISummary(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
