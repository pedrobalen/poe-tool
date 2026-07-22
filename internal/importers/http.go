package importers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// maxResponseBytes caps how much a source is allowed to return. PoB codes are
// small; a larger body indicates the wrong endpoint or a hostile server.
const maxResponseBytes = 8 << 20 // 8 MiB

// fetchText performs a GET and returns the trimmed body, validating the status
// code and that the content is non-empty.
func fetchText(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("User-Agent", "poe-build-overlay")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("source returned HTTP %d for %q", resp.StatusCode, url)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("reading response from %q: %w", url, err)
	}

	text := strings.TrimSpace(string(body))
	if text == "" {
		return "", errors.New("source returned an empty response")
	}

	return text, nil
}
