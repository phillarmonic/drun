package app

import "testing"

func TestNormalizeRuntimeTaskMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty means no override", input: "", want: ""},
		{name: "ci", input: "ci", want: "ci"},
		{name: "normal", input: "normal", want: "normal"},
		{name: "mixed case is normalized", input: " NoRmAl ", want: "normal"},
		{name: "invalid mode fails", input: "verbose", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeRuntimeTaskMode(tt.input)
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
				t.Fatalf("normalizeRuntimeTaskMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
