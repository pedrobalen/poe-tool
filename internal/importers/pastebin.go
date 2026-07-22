package importers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// pastebinImporter resolves pastebin.com links exported from Path of Building
// via the site's /raw/<id> endpoint.
type pastebinImporter struct {
	client *http.Client
}

func newPastebinImporter(client *http.Client) *pastebinImporter {
	return &pastebinImporter{client: client}
}

func (p *pastebinImporter) Name() string { return "pastebin" }

func (p *pastebinImporter) Supports(input string) bool {
	u, err := url.Parse(strings.TrimSpace(input))
	if err != nil {
		return false
	}

	return hostMatches(u.Host, "pastebin.com")
}

func (p *pastebinImporter) Import(ctx context.Context, input string) (Result, error) {
	u, err := url.Parse(strings.TrimSpace(input))
	if err != nil {
		return Result{}, fmt.Errorf("invalid URL: %w", err)
	}

	id := pastebinID(u.Path)
	if id == "" {
		return Result{}, errors.New("missing paste id in pastebin URL")
	}

	rawURL := fmt.Sprintf("https://pastebin.com/raw/%s", id)
	code, err := fetchText(ctx, p.client, rawURL)
	if err != nil {
		return Result{}, err
	}

	canonical := fmt.Sprintf("https://pastebin.com/%s", id)

	return Result{Source: p.Name(), Code: code, URL: canonical}, nil
}

// pastebinID extracts the paste id, tolerating both "/<id>" and "/raw/<id>".
func pastebinID(path string) string {
	seg := firstPathSegment(path)
	if seg == "raw" {
		return firstPathSegment(strings.TrimPrefix(path, "/raw"))
	}

	return seg
}
