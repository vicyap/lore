package distill

import (
	"strings"
	"testing"
)

func TestBuildPromptInput(t *testing.T) {
	result := BuildPromptInput(
		"abc123def012",
		"fix nil pointer in login handler",
		"main",
		"sess-001",
		"diff --git a/login.go\n-old\n+new",
		"**User:** Fix the login bug",
	)

	// Should contain commit metadata
	if !strings.Contains(result, "abc123def012") {
		t.Error("expected commit hash in prompt")
	}
	if !strings.Contains(result, "fix nil pointer") {
		t.Error("expected commit subject in prompt")
	}
	if !strings.Contains(result, "Branch: main") {
		t.Error("expected branch name in prompt")
	}
	if !strings.Contains(result, "Session: sess-001") {
		t.Error("expected session ID in prompt")
	}

	// Should contain diff
	if !strings.Contains(result, "diff --git") {
		t.Error("expected diff content in prompt")
	}

	// Should contain transcript
	if !strings.Contains(result, "Fix the login bug") {
		t.Error("expected transcript content in prompt")
	}

	// Should have correct structure
	sections := []string{"## Commit", "## Diff", "## Transcript"}
	for _, section := range sections {
		if !strings.Contains(result, section) {
			t.Errorf("expected section %q in prompt", section)
		}
	}
}

func TestBuildPromptInput_EmptyFields(t *testing.T) {
	result := BuildPromptInput("", "", "detached", "", "", "(empty transcript)")

	// Should still produce valid structure
	if !strings.Contains(result, "## Commit") {
		t.Error("expected structure even with empty fields")
	}
	if !strings.Contains(result, "(empty transcript)") {
		t.Error("expected transcript placeholder")
	}
}
