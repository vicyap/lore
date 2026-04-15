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
		"deadbeef1234",
		"v0.5.0",
	)

	// Should contain commit metadata
	if !strings.Contains(result, "abc123def012") {
		t.Error("expected commit hash in prompt")
	}
	if !strings.Contains(result, "fix nil pointer") {
		t.Error("expected commit subject in prompt")
	}

	// Should contain metadata fields
	if !strings.Contains(result, "version: v0.5.0") {
		t.Error("expected version in prompt")
	}
	if !strings.Contains(result, "session: sess-001") {
		t.Error("expected session ID in prompt")
	}
	if !strings.Contains(result, "transcript: deadbeef1234") {
		t.Error("expected transcript commit hash in prompt")
	}
	if !strings.Contains(result, "branch: main") {
		t.Error("expected branch name in prompt")
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
	sections := []string{"## Commit", "## Metadata", "## Diff", "## Transcript"}
	for _, section := range sections {
		if !strings.Contains(result, section) {
			t.Errorf("expected section %q in prompt", section)
		}
	}
}

func TestBuildPromptInput_EmptyFields(t *testing.T) {
	result := BuildPromptInput("", "", "detached", "", "", "(empty transcript)", "", "")

	// Should still produce valid structure
	if !strings.Contains(result, "## Commit") {
		t.Error("expected structure even with empty fields")
	}
	if !strings.Contains(result, "(empty transcript)") {
		t.Error("expected transcript placeholder")
	}
}
