package utils

import (
	"strings"
	"testing"
)

func TestBuildName(t *testing.T) {
	if got := BuildName("myrepo", "feature-login"); got != "nekotree-myrepo-feature-login" {
		t.Errorf("BuildName returned %q", got)
	}
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid branch", "feature-login", false},
		{"Valid with underscore", "feature_login", false},
		{"Valid with dot", "v1.2.3", false},        // dot allowed in safeNameRegex
		{"Valid uppercase", "FeatureLogin", false},
		{"Command injection", "branch; rm -rf /", true},
		{"Empty string", "", true},
		{"Whitespace only", "   ", true},
		{"Shell metacharacters", "my$(whoami)", true},
		{"Slash in name", "feature/login", true},
		{"Space in name", "feature login", true},
		{"Whitespace trimmed valid", "  feature-login  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Sanitize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Sanitize(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			// When no error, result should be trimmed and non-empty
			if err == nil && strings.TrimSpace(result) == "" {
				t.Errorf("Sanitize(%q) returned empty result with no error", tt.input)
			}
		})
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid relative path", "build/docs/index.md", false},
		{"Valid absolute path", "/tmp/nekotree-test-123", false},
		{"Valid with dots in filename", "/tmp/nekotree.test", false},
		{"Directory traversal", "../../etc/passwd", true},
		{"Embedded traversal", "/tmp/../../etc", true},
		{"Pipe character", "path/with/|/pipe", true},
		{"Semicolon", "path;inject", true},
		{"Empty string", "", true},
		{"Whitespace only", "   ", true},
		{"Whitespace trimmed valid", "  /tmp/valid  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && strings.TrimSpace(result) == "" {
				t.Errorf("SanitizePath(%q) returned empty result with no error", tt.input)
			}
		})
	}
}
