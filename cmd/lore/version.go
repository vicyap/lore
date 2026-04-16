package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const latestReleaseURL = "https://github.com/vicyap/lore/releases/latest"

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and check for updates",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("lore %s\n", version)

			ctx, cancel := context.WithTimeout(cmd.Context(), 3*time.Second)
			defer cancel()
			latest, err := latestRelease(ctx, latestReleaseURL)
			if err != nil || latest == "" {
				return
			}

			switch {
			case version == "dev":
				fmt.Printf("Latest release: %s\n", latest)
			case version == latest:
				fmt.Println("You are on the latest release.")
			default:
				fmt.Printf("New version available: %s — run `lore update`\n", latest)
			}
		},
	}
}

// latestRelease returns the latest release tag (without the leading "v")
// by following the GitHub /releases/latest redirect. This avoids the
// api.github.com rate limit that "lore update" runs into.
func latestRelease(ctx context.Context, releasesURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, releasesURL, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	loc := resp.Header.Get("Location")
	_, tag, ok := strings.Cut(loc, "/tag/")
	if !ok {
		return "", fmt.Errorf("unexpected redirect location: %q", loc)
	}
	return strings.TrimPrefix(tag, "v"), nil
}
