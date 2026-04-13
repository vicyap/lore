#!/usr/bin/env python3
"""Extract the relevant transcript window for a commit.

Reads a Claude Code JSONL transcript and extracts messages between the
previous git commit tool call and the current one (or session start if
this is the first commit). Strips verbose tool outputs to keep the
distillation input manageable.

Usage:
    python3 extract_window.py <transcript_path> [--max-chars N]

Output:
    Condensed transcript on stdout, suitable for piping to claude CLI.
"""

from __future__ import annotations

import json
import sys
from pathlib import Path


def is_git_commit_tool_call(entry: dict) -> bool:
    """Check if a JSONL entry is a Bash tool call running git commit."""
    if entry.get("type") != "assistant":
        return False
    message = entry.get("message", {})
    content = message.get("content", [])
    if isinstance(content, str):
        return False
    for block in content:
        if block.get("type") == "tool_use" and block.get("name") == "Bash":
            command = block.get("input", {}).get("command", "")
            if command.startswith("git commit"):
                return True
    return False


def extract_message_text(entry: dict) -> str | None:
    """Extract human-readable text from a transcript entry."""
    entry_type = entry.get("type")

    if entry_type == "user":
        message = entry.get("message", {})
        content = message.get("content", "")
        if isinstance(content, str):
            return f"**User:** {content}"
        # Handle structured content (rare in user messages)
        parts = []
        for block in content:
            if isinstance(block, dict) and block.get("type") == "text":
                parts.append(block.get("text", ""))
        if parts:
            return f"**User:** {' '.join(parts)}"
        return None

    if entry_type == "assistant":
        message = entry.get("message", {})
        content = message.get("content", [])
        if isinstance(content, str):
            return f"**Assistant:** {content}"
        parts = []
        for block in content:
            if isinstance(block, dict):
                if block.get("type") == "text":
                    text = block.get("text", "")
                    if text.strip():
                        parts.append(text)
                elif block.get("type") == "tool_use":
                    name = block.get("name", "unknown")
                    tool_input = block.get("input", {})
                    # For Bash, include the command
                    if name == "Bash":
                        command = tool_input.get("command", "")
                        parts.append(f"[Tool: {name}] {command}")
                    # For Edit/Write, include just the file path
                    elif name in ("Edit", "Write"):
                        file_path = tool_input.get("file_path", "")
                        parts.append(f"[Tool: {name}] {file_path}")
                    # For Read, include file path
                    elif name == "Read":
                        file_path = tool_input.get("file_path", "")
                        parts.append(f"[Tool: {name}] {file_path}")
                    # For Grep/Glob, include pattern
                    elif name in ("Grep", "Glob"):
                        pattern = tool_input.get("pattern", "")
                        parts.append(f"[Tool: {name}] {pattern}")
                    # For Agent, include description
                    elif name == "Agent":
                        desc = tool_input.get("description", "")
                        parts.append(f"[Tool: {name}] {desc}")
                    else:
                        parts.append(f"[Tool: {name}]")
        if parts:
            return f"**Assistant:** {' | '.join(parts)}"
        return None

    return None


def extract_window(
    transcript_path: str, max_chars: int = 50000
) -> str:
    """Extract the transcript window for the most recent commit."""
    path = Path(transcript_path)
    if not path.exists():
        return "(transcript not found)"

    entries = []
    with open(path, encoding="utf-8") as file:
        for line in file:
            line = line.strip()
            if not line:
                continue
            try:
                entries.append(json.loads(line))
            except json.JSONDecodeError:
                continue

    if not entries:
        return "(empty transcript)"

    # Find all git commit tool calls
    commit_indices = [
        idx for idx, entry in enumerate(entries) if is_git_commit_tool_call(entry)
    ]

    if not commit_indices:
        # No commits found — use the whole transcript
        window_start = 0
        window_end = len(entries)
    elif len(commit_indices) == 1:
        # First commit in session — from start to commit
        window_start = 0
        window_end = commit_indices[0] + 1
    else:
        # Between the second-to-last commit and the last commit
        window_start = commit_indices[-2] + 1
        window_end = commit_indices[-1] + 1

    # Extract messages in the window
    messages = []
    for entry in entries[window_start:window_end]:
        text = extract_message_text(entry)
        if text:
            messages.append(text)

    if not messages:
        return "(no readable messages in window)"

    result = "\n\n".join(messages)

    # Truncate if too long (keep the end, which has the most relevant context)
    if len(result) > max_chars:
        result = "...(truncated)...\n\n" + result[-max_chars:]

    return result


def main() -> None:
    if len(sys.argv) < 2:
        print("Usage: extract_window.py <transcript_path> [--max-chars N]", file=sys.stderr)
        sys.exit(1)

    transcript_path = sys.argv[1]
    max_chars = 50000

    if "--max-chars" in sys.argv:
        idx = sys.argv.index("--max-chars")
        if idx + 1 < len(sys.argv):
            max_chars = int(sys.argv[idx + 1])

    print(extract_window(transcript_path, max_chars))


if __name__ == "__main__":
    main()
