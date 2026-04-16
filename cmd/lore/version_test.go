package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLatestRelease(t *testing.T) {
	tests := []struct {
		name     string
		location string
		want     string
		wantErr  bool
	}{
		{"standard tag", "https://github.com/vicyap/lore/releases/tag/v0.3.6", "0.3.6", false},
		{"no v prefix", "https://github.com/vicyap/lore/releases/tag/0.3.6", "0.3.6", false},
		{"missing /tag/", "https://github.com/vicyap/lore/releases", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Location", tc.location)
				writer.WriteHeader(http.StatusFound)
			}))
			defer srv.Close()

			got, err := latestRelease(t.Context(), srv.URL)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
