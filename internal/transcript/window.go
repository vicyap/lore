package transcript

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Entry represents a single JSONL transcript entry.
type Entry struct {
	Type    string  `json:"type"`
	Message Message `json:"message"`
}

// Message represents the message field of a transcript entry.
type Message struct {
	Content json.RawMessage `json:"content"`
}

// ContentBlock represents a block within an assistant message's content array.
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolInput holds the parsed input fields we care about from tool_use blocks.
type ToolInput struct {
	Command     string `json:"command"`
	FilePath    string `json:"file_path"`
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
}

// ParseJSONL reads a JSONL file and returns parsed entries.
func ParseJSONL(path string) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []Entry
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

// IsGitCommitToolCall checks if an entry is an assistant Bash tool call running git commit.
func IsGitCommitToolCall(entry Entry) bool {
	if entry.Type != "assistant" {
		return false
	}

	blocks, err := parseContentBlocks(entry.Message.Content)
	if err != nil {
		return false
	}

	for _, block := range blocks {
		if block.Type == "tool_use" && block.Name == "Bash" {
			var input ToolInput
			if err := json.Unmarshal(block.Input, &input); err == nil {
				if strings.HasPrefix(input.Command, "git commit") {
					return true
				}
			}
		}
	}
	return false
}

// ExtractWindow extracts the relevant transcript window for the most recent commit.
func ExtractWindow(entries []Entry, maxChars int) string {
	if len(entries) == 0 {
		return "(empty transcript)"
	}

	// Find all git commit tool call indices
	var commitIndices []int
	for idx, entry := range entries {
		if IsGitCommitToolCall(entry) {
			commitIndices = append(commitIndices, idx)
		}
	}

	var windowStart, windowEnd int
	switch {
	case len(commitIndices) == 0:
		windowStart = 0
		windowEnd = len(entries)
	case len(commitIndices) == 1:
		windowStart = 0
		windowEnd = commitIndices[0] + 1
	default:
		windowStart = commitIndices[len(commitIndices)-2] + 1
		windowEnd = commitIndices[len(commitIndices)-1] + 1
	}

	// Extract messages in the window
	var messages []string
	for _, entry := range entries[windowStart:windowEnd] {
		if text := ExtractMessageText(entry); text != "" {
			messages = append(messages, text)
		}
	}

	if len(messages) == 0 {
		return "(no readable messages in window)"
	}

	result := strings.Join(messages, "\n\n")

	// Truncate if too long (keep the end — most relevant context)
	if len(result) > maxChars {
		result = "...(truncated)...\n\n" + result[len(result)-maxChars:]
	}

	return result
}

// ExtractMessageText extracts human-readable text from a transcript entry.
func ExtractMessageText(entry Entry) string {
	switch entry.Type {
	case "user":
		return extractUserMessage(entry)
	case "assistant":
		return extractAssistantMessage(entry)
	default:
		return ""
	}
}

func extractUserMessage(entry Entry) string {
	// Try as plain string first
	var text string
	if err := json.Unmarshal(entry.Message.Content, &text); err == nil {
		return "**User:** " + text
	}

	// Try as content blocks
	blocks, err := parseContentBlocks(entry.Message.Content)
	if err != nil {
		return ""
	}

	var parts []string
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	if len(parts) > 0 {
		return "**User:** " + strings.Join(parts, " ")
	}
	return ""
}

func extractAssistantMessage(entry Entry) string {
	// Try as plain string first
	var text string
	if err := json.Unmarshal(entry.Message.Content, &text); err == nil {
		return "**Assistant:** " + text
	}

	// Try as content blocks
	blocks, err := parseContentBlocks(entry.Message.Content)
	if err != nil {
		return ""
	}

	var parts []string
	for _, block := range blocks {
		switch block.Type {
		case "text":
			if trimmed := strings.TrimSpace(block.Text); trimmed != "" {
				parts = append(parts, trimmed)
			}
		case "tool_use":
			parts = append(parts, formatToolUse(block))
		}
	}

	if len(parts) > 0 {
		return "**Assistant:** " + strings.Join(parts, " | ")
	}
	return ""
}

func formatToolUse(block ContentBlock) string {
	var input ToolInput
	_ = json.Unmarshal(block.Input, &input)

	switch block.Name {
	case "Bash":
		return fmt.Sprintf("[Tool: Bash] %s", input.Command)
	case "Edit", "Write", "Read":
		return fmt.Sprintf("[Tool: %s] %s", block.Name, input.FilePath)
	case "Grep", "Glob":
		return fmt.Sprintf("[Tool: %s] %s", block.Name, input.Pattern)
	case "Agent":
		return fmt.Sprintf("[Tool: Agent] %s", input.Description)
	default:
		return fmt.Sprintf("[Tool: %s]", block.Name)
	}
}

func parseContentBlocks(raw json.RawMessage) ([]ContentBlock, error) {
	if raw == nil {
		return nil, nil
	}
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}
