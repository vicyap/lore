package config

import "os"

const (
	DefaultModel    = "opus"
	DefaultNotesRef = "lore"
	DefaultBranch   = "lore/transcripts"

	// Internal truncation limits — not user-configurable.
	// Generous given 1M-token context windows on Claude Opus 4.6 and
	// Sonnet 4.6 as of April 15, 2026.
	MaxDiffChars       = 200000
	MaxTranscriptChars = 500000
)

type Config struct {
	Model    string
	NotesRef string
	Branch   string
	Debug    bool
}

func Load() Config {
	return Config{
		Model:    envOrDefault("LORE_MODEL", DefaultModel),
		NotesRef: DefaultNotesRef,
		Branch:   DefaultBranch,
		Debug:    os.Getenv("LORE_DEBUG") == "1",
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
