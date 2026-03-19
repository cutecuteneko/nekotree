package utils

import "testing"

func TestSanitize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid Branch", "feature-login", false},
		{"Command Injection", "branch; rm -rf /", true},
		{"Empty String", "", true},
		{"Shell Metacharacters", "my$(whoami)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Sanitize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Sanitize() error = %v, wantErr %v", err, tt.wantErr)
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
		{"Valid Path", "build/docs/index.md", false},
		{"Temp Path", "/tmp/nekotree-test-123", false},
		{"Directory Traversal", "../../etc/passwd", true},
		{"Forbidden Chars", "path/with/|/pipe", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SanitizePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
