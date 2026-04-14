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

//go:embed skill.md
var skillDefinition []byte

// SkillDefinition returns the lore skill markdown.
func SkillDefinition() ([]byte, error) {
	return skillDefinition, nil
}
