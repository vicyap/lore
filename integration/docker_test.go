//go:build realclaude

package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDocker_RealClaude(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping real claude test")
	}

	// Build Docker image
	projectRoot, _ := os.Getwd()
	projectRoot = strings.TrimSuffix(projectRoot, "/integration")

	buildCmd := exec.Command("docker", "build",
		"-f", "integration/Dockerfile",
		"-t", "lore-e2e-test",
		".")
	buildCmd.Dir = projectRoot
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("docker build failed: %v", err)
	}

	// Run container
	runCmd := exec.Command("docker", "run",
		"--rm",
		"-e", "ANTHROPIC_API_KEY="+apiKey,
		"lore-e2e-test",
	)
	runCmd.Dir = projectRoot
	out, err := runCmd.CombinedOutput()
	t.Logf("Docker output:\n%s", out)
	if err != nil {
		t.Fatalf("docker run failed: %v\nOutput: %s", err, out)
	}

	// Verify output contains expected markers
	output := string(out)
	if !strings.Contains(output, "## Intent") {
		t.Error("expected '## Intent' in docker output")
	}
	if !strings.Contains(output, "## Confidence") {
		t.Error("expected '## Confidence' in docker output")
	}
}
