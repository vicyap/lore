package prompts

import (
	_ "embed"
)

//go:embed distill.md
var distillPrompt []byte

// DistillPrompt returns the distillation system prompt.
func DistillPrompt() []byte {
	return distillPrompt
}

//go:embed lore-workflow.yml
var workflowTemplate []byte

// WorkflowTemplate returns the GitHub Actions workflow YAML.
func WorkflowTemplate() []byte {
	return workflowTemplate
}
