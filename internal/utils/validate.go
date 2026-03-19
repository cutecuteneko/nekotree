package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// Allow alphanumeric, underscores, and dashes for names/branches
var safeNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

// Allow alphanumeric, underscores, dashes, dots, and slashes for paths
var safePathRegex = regexp.MustCompile(`^[a-zA-Z0-9\-\._/]+$`)

// Sanitize handles basic strings (branches, container names)
func Sanitize(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("input is empty")
	}
	if !safeNameRegex.MatchString(trimmed) {
		return "", fmt.Errorf("input contains forbidden characters: %s", trimmed)
	}
	return trimmed, nil
}

// SanitizePath handles file/directory paths specifically
func SanitizePath(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("path is empty")
	}
	// Prevents directory traversal like ../../etc/passwd
	if strings.Contains(trimmed, "..") {
		return "", fmt.Errorf("path contains directory traversal: %s", trimmed)
	}
	if !safePathRegex.MatchString(trimmed) {
		return "", fmt.Errorf("path contains forbidden characters: %s", trimmed)
	}
	return trimmed, nil
}
