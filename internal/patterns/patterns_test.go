package patterns

import (
	"testing"
)

func TestGetMacro(t *testing.T) {
	tests := []struct {
		name     string
		exists   bool
		expected string
	}{
		{"semver", true, `^v\d+\.\d+\.\d+$`},
		{"semver_extended", true, `^v\d+\.\d+\.\d+(-[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*)?(\+[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*)?$`},
		{"uuid", true, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`},
		{"url", true, `https?://[^\s/$.?#].[^\s]*`},
		{"ipv4", true, `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`},
		{"slug", true, `^[a-z0-9]+(?:-[a-z0-9]+)*$`},
		{"docker_tag", true, `^[a-zA-Z0-9][a-zA-Z0-9._-]*$`},
		{"git_branch", true, `^[a-zA-Z0-9][a-zA-Z0-9._/-]*[a-zA-Z0-9]$`},
		{"nonexistent", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			macro, exists := GetMacro(tt.name)
			if exists != tt.exists {
				t.Errorf("GetMacro(%q) exists = %v, want %v", tt.name, exists, tt.exists)
			}
			if exists && macro.Pattern != tt.expected {
				t.Errorf("GetMacro(%q) pattern = %q, want %q", tt.name, macro.Pattern, tt.expected)
			}
		})
	}
}

func TestValidatePattern_Semver(t *testing.T) {
	tests := []struct {
		value     string
		wantError bool
	}{
		{"v1.0.0", false},
		{"v2.1.3", false},
		{"v10.20.30", false},
		{"1.0.0", true},        // missing 'v'
		{"v1.0", true},         // incomplete
		{"v1.0.0.0", true},     // too many parts
		{"version1.0.0", true}, // wrong prefix
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			err := ValidatePattern(tt.value, "semver")
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePattern(%q, \"semver\") error = %v, wantError %v", tt.value, err, tt.wantError)
			}
		})
	}
}

func TestValidatePattern_SemverExtended(t *testing.T) {
	tests := []struct {
		value     string
		wantError bool
	}{
		{"v1.0.0", false},
		{"v2.1.3-alpha", false},
		{"v1.0.0-beta.1", false},
		{"v2.0.0-RC2", false},
		{"v1.0.0+build.123", false},
		{"v2.0.1-alpha.1+build.456", false},
		{"1.0.0", true}, // missing 'v'
		{"v1.0", true},  // incomplete
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			err := ValidatePattern(tt.value, "semver_extended")
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePattern(%q, \"semver_extended\") error = %v, wantError %v", tt.value, err, tt.wantError)
			}
		})
	}
}

func TestValidatePattern_UUID(t *testing.T) {
	tests := []struct {
		value     string
		wantError bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", false},
		{"6ba7b810-9dad-11d1-80b4-00c04fd430c8", false},
		{"00000000-0000-0000-0000-000000000000", false},
		{"550e8400-e29b-41d4-a716-44665544000", true},   // too short
		{"550e8400-e29b-41d4-a716-4466554400000", true}, // too long
		{"550e8400e29b41d4a716446655440000", true},      // no hyphens
		{"not-a-uuid", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			err := ValidatePattern(tt.value, "uuid")
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePattern(%q, \"uuid\") error = %v, wantError %v", tt.value, err, tt.wantError)
			}
		})
	}
}

func TestValidatePattern_URL(t *testing.T) {
	tests := []struct {
		value     string
		wantError bool
	}{
		{"https://example.com", false},
		{"http://api.example.com", false},
		{"https://api.example.com/v1/users", false},
		{"https://subdomain.example.com:8080/path", false},
		{"ftp://example.com", true}, // not http/https
		{"example.com", true},       // no protocol
		{"not-a-url", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			err := ValidatePattern(tt.value, "url")
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePattern(%q, \"url\") error = %v, wantError %v", tt.value, err, tt.wantError)
			}
		})
	}
}

func TestValidatePattern_IPv4(t *testing.T) {
	tests := []struct {
		value     string
		wantError bool
	}{
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"255.255.255.255", false},
		{"0.0.0.0", false},
		{"256.1.1.1", true},     // out of range
		{"192.168.1", true},     // incomplete
		{"192.168.1.1.1", true}, // too many parts
		{"not.an.ip", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			err := ValidatePattern(tt.value, "ipv4")
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePattern(%q, \"ipv4\") error = %v, wantError %v", tt.value, err, tt.wantError)
			}
		})
	}
}

func TestValidatePattern_Slug(t *testing.T) {
	tests := []struct {
		value     string
		wantError bool
	}{
		{"my-project", false},
		{"hello-world-123", false},
		{"simple", false},
		{"a", false},
		{"My-Project", true}, // uppercase
		{"my_project", true}, // underscore
		{"my project", true}, // space
		{"-project", true},   // starts with hyphen
		{"project-", true},   // ends with hyphen
		{"", true},           // empty
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			err := ValidatePattern(tt.value, "slug")
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePattern(%q, \"slug\") error = %v, wantError %v", tt.value, err, tt.wantError)
			}
		})
	}
}

func TestValidatePattern_DockerTag(t *testing.T) {
	tests := []struct {
		value     string
		wantError bool
	}{
		{"latest", false},
		{"v1.0.0", false},
		{"my-app_v2.1", false},
		{"my-app", false},
		{"-invalid", true}, // starts with hyphen
		{"", true},         // empty
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			err := ValidatePattern(tt.value, "docker_tag")
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePattern(%q, \"docker_tag\") error = %v, wantError %v", tt.value, err, tt.wantError)
			}
		})
	}
}

func TestValidatePattern_GitBranch(t *testing.T) {
	tests := []struct {
		value     string
		wantError bool
	}{
		{"main", false},
		{"feature/new-feature", false},
		{"release/v1.0.0", false},
		{"hotfix-123", false},
		{"-invalid", true}, // starts with hyphen
		{"invalid-", true}, // ends with hyphen
		{"", true},         // empty
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			err := ValidatePattern(tt.value, "git_branch")
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePattern(%q, \"git_branch\") error = %v, wantError %v", tt.value, err, tt.wantError)
			}
		})
	}
}

func TestExpandMacro(t *testing.T) {
	tests := []struct {
		name      string
		wantError bool
		expected  string
	}{
		{"semver", false, `^v\d+\.\d+\.\d+$`},
		{"uuid", false, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`},
		{"nonexistent", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandMacro(tt.name)
			if (err != nil) != tt.wantError {
				t.Errorf("ExpandMacro(%q) error = %v, wantError %v", tt.name, err, tt.wantError)
			}
			if !tt.wantError && result != tt.expected {
				t.Errorf("ExpandMacro(%q) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}
