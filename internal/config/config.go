package config

import (
	"os"
	"strconv"
)

const (
	DefaultModel             = "sonnet"
	DefaultMaxDiffChars      = 20000
	DefaultMaxTranscriptChars = 50000
	DefaultNotesRef          = "lore"
	DefaultBranch            = "lore/transcripts"
)

type Config struct {
	Model             string
	MaxDiffChars      int
	MaxTranscriptChars int
	NotesRef          string
	Branch            string
	Debug             bool
}

func Load() Config {
	return Config{
		Model:             envOrDefault("LORE_MODEL", DefaultModel),
		MaxDiffChars:      envIntOrDefault("LORE_MAX_DIFF_CHARS", DefaultMaxDiffChars),
		MaxTranscriptChars: envIntOrDefault("LORE_MAX_TRANSCRIPT_CHARS", DefaultMaxTranscriptChars),
		NotesRef:          DefaultNotesRef,
		Branch:            DefaultBranch,
		Debug:             os.Getenv("LORE_DEBUG") == "1",
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
