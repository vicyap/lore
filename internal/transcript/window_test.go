package transcript

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", name)
}

func TestParseJSONL(t *testing.T) {
	entries, err := ParseJSONL(testdataPath("transcript_simple.jsonl"))
	if err != nil {
		t.Fatalf("ParseJSONL: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}
	if entries[0].Type != "user" {
		t.Errorf("expected first entry type 'user', got %q", entries[0].Type)
	}
	if entries[3].Type != "assistant" {
		t.Errorf("expected last entry type 'assistant', got %q", entries[3].Type)
	}
}

func TestParseJSONL_NotFound(t *testing.T) {
	_, err := ParseJSONL("/nonexistent/path.jsonl")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestIsGitCommitToolCall(t *testing.T) {
	entries, err := ParseJSONL(testdataPath("transcript_simple.jsonl"))
	if err != nil {
		t.Fatalf("ParseJSONL: %v", err)
	}

	// Only the last entry (git commit) should match
	for idx, entry := range entries {
		got := IsGitCommitToolCall(entry)
		want := idx == 3
		if got != want {
			t.Errorf("entry %d: IsGitCommitToolCall = %v, want %v", idx, got, want)
		}
	}
}

func TestExtractWindow_SingleCommit(t *testing.T) {
	entries, err := ParseJSONL(testdataPath("transcript_simple.jsonl"))
	if err != nil {
		t.Fatalf("ParseJSONL: %v", err)
	}

	result := ExtractWindow(entries, 50000)

	// Should include all messages from start to the commit
	if result == "(empty transcript)" || result == "(no readable messages in window)" {
		t.Fatalf("unexpected empty result: %q", result)
	}

	// Should contain the user message
	assertContains(t, result, "**User:** Fix the login bug")

	// Should contain assistant deliberation
	assertContains(t, result, "nil pointer dereference")

	// Should contain the commit tool call
	assertContains(t, result, "[Tool: Bash] git commit")
}

func TestExtractWindow_MultiCommit(t *testing.T) {
	entries, err := ParseJSONL(testdataPath("transcript_multi.jsonl"))
	if err != nil {
		t.Fatalf("ParseJSONL: %v", err)
	}

	result := ExtractWindow(entries, 50000)

	// Should only contain the window between the 2nd and 3rd commit
	// (the last two commits)

	// Should NOT contain the first commit's content
	assertNotContains(t, result, "event type definitions")

	// Should contain the last window's content
	assertContains(t, result, "Slack notification")
	assertContains(t, result, "wire Slack notifications")
}

func TestExtractWindow_NoCommits(t *testing.T) {
	entries, err := ParseJSONL(testdataPath("transcript_empty.jsonl"))
	if err != nil {
		t.Fatalf("ParseJSONL: %v", err)
	}

	result := ExtractWindow(entries, 50000)

	// Should include the full transcript since there are no commits
	assertContains(t, result, "**User:** What does this project do?")
	assertContains(t, result, "Go web application")
}

func TestExtractWindow_Empty(t *testing.T) {
	result := ExtractWindow(nil, 50000)
	if result != "(empty transcript)" {
		t.Errorf("expected '(empty transcript)', got %q", result)
	}
}

func TestExtractWindow_Truncation(t *testing.T) {
	entries, err := ParseJSONL(testdataPath("transcript_large.jsonl"))
	if err != nil {
		t.Fatalf("ParseJSONL: %v", err)
	}

	// Use a very small max to force truncation
	result := ExtractWindow(entries, 100)

	assertContains(t, result, "...(truncated)...")

	// The end of the result should contain the most recent content
	// (the git commit tool call)
	if len(result) > 200 {
		t.Errorf("result too long after truncation: %d chars", len(result))
	}
}

func TestExtractMessageText_UserMessage(t *testing.T) {
	entries, err := ParseJSONL(testdataPath("transcript_simple.jsonl"))
	if err != nil {
		t.Fatalf("ParseJSONL: %v", err)
	}

	text := ExtractMessageText(entries[0])
	assertContains(t, text, "**User:**")
	assertContains(t, text, "Fix the login bug")
}

func TestExtractMessageText_ToolUse(t *testing.T) {
	entries, err := ParseJSONL(testdataPath("transcript_simple.jsonl"))
	if err != nil {
		t.Fatalf("ParseJSONL: %v", err)
	}

	// Entry 1 has a Read tool use
	text := ExtractMessageText(entries[1])
	assertContains(t, text, "[Tool: Read] src/auth/login.go")

	// Entry 2 has an Edit tool use
	text = ExtractMessageText(entries[2])
	assertContains(t, text, "[Tool: Edit] src/auth/login.go")
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !containsString(haystack, needle) {
		t.Errorf("expected result to contain %q, got:\n%s", needle, haystack)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if containsString(haystack, needle) {
		t.Errorf("expected result NOT to contain %q, got:\n%s", needle, haystack)
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || len(haystack) > 0 && containsSubstring(haystack, needle))
}

func containsSubstring(haystack, needle string) bool {
	for idx := 0; idx <= len(haystack)-len(needle); idx++ {
		if haystack[idx:idx+len(needle)] == needle {
			return true
		}
	}
	return false
}
