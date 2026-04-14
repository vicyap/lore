// fakeclaude mimics the claude CLI's -p interface for hermetic integration tests.
//
// It accepts the same arguments as `claude -p --model X --system-prompt-file Y "prompt"`,
// reads stdin, and returns canned distillation output.
//
// Behavior is controlled via environment variables:
//   - FAKECLAUDE_EXIT_CODE: exit with this code (default 0)
//   - FAKECLAUDE_EMPTY: if "1", return empty output
//   - FAKECLAUDE_OUTPUT: override the canned output with this text
//   - FAKECLAUDE_LOG: write received args + stdin to this file path
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const cannedOutput = `## Intent
Automated test: fix the login handler nil pointer dereference.

## Constraints
- Must not break existing auth flow

## Rejected Alternatives
- Returning 404 instead of error -- inconsistent with API contract

## Directives
- Always nil-check user records before accessing fields

## Confidence
high

## Session
test-session | main`

func main() {
	// Read stdin completely (prevents broken pipe)
	stdinBytes, _ := io.ReadAll(os.Stdin)
	stdinContent := string(stdinBytes)

	// Log if requested
	if logPath := os.Getenv("FAKECLAUDE_LOG"); logPath != "" {
		var logContent strings.Builder
		logContent.WriteString("ARGS: " + strings.Join(os.Args[1:], " ") + "\n")
		logContent.WriteString("STDIN:\n" + stdinContent + "\n")
		os.WriteFile(logPath, []byte(logContent.String()), 0o644)
	}

	// Check for custom exit code
	if exitStr := os.Getenv("FAKECLAUDE_EXIT_CODE"); exitStr != "" {
		if code, err := strconv.Atoi(exitStr); err == nil && code != 0 {
			fmt.Fprintln(os.Stderr, "fakeclaude: simulated error")
			os.Exit(code)
		}
	}

	// Check for empty output mode
	if os.Getenv("FAKECLAUDE_EMPTY") == "1" {
		os.Exit(0)
	}

	// Check for custom output
	if custom := os.Getenv("FAKECLAUDE_OUTPUT"); custom != "" {
		fmt.Print(custom)
		os.Exit(0)
	}

	// Return canned output
	fmt.Print(cannedOutput)
}
