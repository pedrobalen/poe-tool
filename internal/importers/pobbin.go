package importers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// pobbinImporter resolves https://pobb.in/<id> links via the site's raw
// endpoint, which returns the PoB export code directly.
type pobbinImporter struct {
	client *http.Client
}

func newPobbinImporter(client *http.Client) *pobbinImporter {
	return &pobbinImporter{client: client}
}

func (p *pobbinImporter) Name() string { return "pobbin" }

func (p *pobbinImporter) Supports(input string) bool {
	u, err := url.Parse(strings.TrimSpace(input))
	if err != nil {
		return false
	}

	return hostMatches(u.Host, "pobb.in")
}

func (p *pobbinImporter) Import(ctx context.Context, input string) (Result, error) {
	u, err := url.Parse(strings.TrimSpace(input))
	if err != nil {
		return Result{}, fmt.Errorf("invalid URL: %w", err)
	}

	id := firstPathSegment(u.Path)
	if id == "" {
		return Result{}, errors.New("missing build id in pobb.in URL")
	}

	rawURL := fmt.Sprintf("https://pobb.in/%s/raw", id)
	code, err := fetchText(ctx, p.client, rawURL)
	if err != nil {
		return Result{}, err
	}

	canonical := fmt.Sprintf("https://pobb.in/%s", id)

	return Result{Source: p.Name(), Code: code, URL: canonical}, nil
}
