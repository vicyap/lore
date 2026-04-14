#!/usr/bin/env bash
set -euo pipefail

echo "=== lore E2E test with real Claude CLI ==="

# Create test repo
REPO=$(mktemp -d)
cd "$REPO"
git init
git config user.email "e2e@test.com"
git config user.name "E2E Test"

# Initial commit
cat > main.go << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("hello")
}
EOF
git add main.go
git commit -m "initial commit"

# Init lore (non-interactive)
echo "n" | lore init
echo "  lore init: OK"

# Make a code change
cat > main.go << 'EOF'
package main

import "fmt"

func main() {
    user := getUser()
    if user == nil {
        fmt.Println("user not found")
        return
    }
    fmt.Println("hello", user.Name)
}

type User struct {
    Name string
}

func getUser() *User {
    return &User{Name: "world"}
}
EOF
git add main.go
git commit -m "add nil check for user lookup"

# Create a minimal transcript fixture
TRANSCRIPT=$(mktemp --suffix=.jsonl)
cat > "$TRANSCRIPT" << 'JSONL'
{"type":"user","message":{"content":"Fix the nil pointer dereference when looking up users"},"uuid":"u1","timestamp":"2026-04-14T10:00:00Z","sessionId":"e2e-sess"}
{"type":"assistant","message":{"content":[{"type":"text","text":"I see the issue - getUser can return nil but main doesn't check for it. Let me add a nil check."},{"type":"tool_use","name":"Edit","id":"t1","input":{"file_path":"main.go"}}]},"uuid":"a1","timestamp":"2026-04-14T10:00:05Z","sessionId":"e2e-sess"}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","id":"t2","input":{"command":"git commit -am \"add nil check for user lookup\""}}]},"uuid":"a2","timestamp":"2026-04-14T10:00:10Z","sessionId":"e2e-sess"}
JSONL

# Run hook
COMMIT_HASH=$(git rev-parse HEAD)
echo "{\"session_id\":\"e2e-sess\",\"transcript_path\":\"$TRANSCRIPT\",\"cwd\":\"$REPO\",\"tool_input\":{\"command\":\"git commit -am \\\"add nil check\\\"\"}}" | lore hook
echo "  lore hook: OK"

# Verify note
echo ""
echo "=== Decision note ==="
lore show "$COMMIT_HASH"
echo ""

# Check for expected sections
NOTE=$(git notes --ref=lore show HEAD 2>/dev/null || true)
if echo "$NOTE" | grep -q "## Intent"; then
    echo "  ## Intent: FOUND"
else
    echo "  ## Intent: MISSING"
    exit 1
fi

if echo "$NOTE" | grep -q "## Confidence"; then
    echo "  ## Confidence: FOUND"
else
    echo "  ## Confidence: MISSING"
    exit 1
fi

echo ""
echo "=== E2E test PASSED ==="

# Cleanup
rm -rf "$REPO" "$TRANSCRIPT"
