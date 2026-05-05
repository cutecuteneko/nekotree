package mergify_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot resolves the repository root relative to this test file's location.
// The file lives at internal/mergify/mergify_test.go, so the root is two levels up.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// readMergifyConfig reads .mergify.yml from the repository root.
func readMergifyConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(repoRoot(t), ".mergify.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read .mergify.yml: %v", err)
	}
	return string(data)
}

// TestMergifyConfigFileExists verifies the config file is present and non-empty.
func TestMergifyConfigFileExists(t *testing.T) {
	content := readMergifyConfig(t)
	if strings.TrimSpace(content) == "" {
		t.Error(".mergify.yml is empty")
	}
}

// TestDependabotRulePresent checks the auto-merge rule for Dependabot exists.
func TestDependabotRulePresent(t *testing.T) {
	content := readMergifyConfig(t)
	if !strings.Contains(content, "Auto-merge Dependabot dependency updates") {
		t.Error("expected Dependabot auto-merge rule to be present in .mergify.yml")
	}
}

// TestDependabotRuleContainsReviewAction verifies that the new `review:` action
// is present in the configuration (added by this PR).
func TestDependabotRuleContainsReviewAction(t *testing.T) {
	content := readMergifyConfig(t)
	if !strings.Contains(content, "review:") {
		t.Error("expected 'review:' action to be present in .mergify.yml")
	}
}

// TestDependabotReviewActionType verifies the review action uses APPROVE type.
func TestDependabotReviewActionType(t *testing.T) {
	content := readMergifyConfig(t)
	if !strings.Contains(content, "type: APPROVE") {
		t.Error("expected review action to have 'type: APPROVE' in .mergify.yml")
	}
}

// TestDependabotReviewActionMessage verifies the review action carries the exact
// approval message introduced by this PR.
func TestDependabotReviewActionMessage(t *testing.T) {
	const expectedMessage = `"Automatically approving Dependabot PR — CI passed."`
	content := readMergifyConfig(t)
	if !strings.Contains(content, expectedMessage) {
		t.Errorf("expected review message %q not found in .mergify.yml", expectedMessage)
	}
}

// TestReviewActionPrecedesMergeAction verifies that the `review:` action appears
// before the `merge:` action within the Dependabot rule block, ensuring Mergify
// evaluates them in the intended order.
func TestReviewActionPrecedesMergeAction(t *testing.T) {
	content := readMergifyConfig(t)

	reviewIdx := strings.Index(content, "review:")
	mergeIdx := strings.Index(content, "merge:")

	if reviewIdx == -1 {
		t.Fatal("'review:' action not found in .mergify.yml")
	}
	if mergeIdx == -1 {
		t.Fatal("'merge:' action not found in .mergify.yml")
	}
	if reviewIdx >= mergeIdx {
		t.Errorf("expected 'review:' (offset %d) to precede 'merge:' (offset %d) in .mergify.yml",
			reviewIdx, mergeIdx)
	}
}

// TestDependabotRuleConditionsIntact verifies that the pre-existing conditions
// of the Dependabot auto-merge rule were not altered by this PR.
func TestDependabotRuleConditionsIntact(t *testing.T) {
	content := readMergifyConfig(t)

	requiredConditions := []string{
		"author = dependabot[bot]",
		"check-success = build",
		'"-draft"',
		'"-conflict"',
	}

	for _, cond := range requiredConditions {
		if !strings.Contains(content, cond) {
			t.Errorf("expected condition %q to be present in .mergify.yml", cond)
		}
	}
}

// TestMergeMethodIsSquash verifies the merge method was not changed by this PR.
func TestMergeMethodIsSquash(t *testing.T) {
	content := readMergifyConfig(t)
	if !strings.Contains(content, "method: squash") {
		t.Error("expected 'method: squash' to be present in .mergify.yml")
	}
}

// TestAllRulesPresent is a regression test ensuring that all four original rules
// remain intact after this PR's changes.
func TestAllRulesPresent(t *testing.T) {
	content := readMergifyConfig(t)

	expectedRules := []string{
		"Auto-merge Dependabot dependency updates",
		"Label feature branches",
		"Label bug fix branches",
		"Label chore branches",
		"Close stale PRs",
	}

	for _, rule := range expectedRules {
		if !strings.Contains(content, rule) {
			t.Errorf("expected rule %q to be present in .mergify.yml", rule)
		}
	}
}

// TestReviewAndMergeActionsCoexist verifies both the review and merge actions
// are present together under the Dependabot rule's actions block, confirming
// neither was accidentally removed when the review action was added.
func TestReviewAndMergeActionsCoexist(t *testing.T) {
	content := readMergifyConfig(t)

	if !strings.Contains(content, "review:") {
		t.Error("'review:' action missing from .mergify.yml")
	}
	if !strings.Contains(content, "merge:") {
		t.Error("'merge:' action missing from .mergify.yml")
	}
}

// TestApproveMessageDoesNotContainTypo is a regression test guarding against
// accidental typos or whitespace changes in the approval message.
func TestApproveMessageDoesNotContainTypo(t *testing.T) {
	content := readMergifyConfig(t)

	// Ensure common misformulations are absent.
	badVariants := []string{
		`"Automatically approving dependabot PR`,       // wrong capitalisation
		`"Automatically approving Dependabot PR — CI"`, // truncated message
		"type: approve",                                 // wrong case for APPROVE
		"type: Approve",                                 // wrong case for APPROVE
	}

	for _, bad := range badVariants {
		if strings.Contains(content, bad) {
			t.Errorf("unexpected content found in .mergify.yml: %q", bad)
		}
	}
}
