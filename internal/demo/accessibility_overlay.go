package demo

import (
	"context"
	"net/url"
	"strings"

	"github.com/hostrepute/reputecast/internal/accessibility"
)

// accessibilityTitles are the titles of the pages Weekly Digest #143's
// vague links point at (brief §4: "Click here" and the bare URL).
var accessibilityTitles = map[string]accessibility.PageTitle{
	"https://docs.acme.io/scheduled-rollouts": {Title: "Scheduled rollouts · Acme Docs", Lang: "en"},
	"https://docs.acme.io/migrate-v2":         {Title: "Migrating to the v2 API · Acme Docs", Lang: "en"},
}

// AccessibilityTitleOverlay answers title reads for the demo's links
// without going to the network: their hosts are real domains the demo has
// no business calling. Other pages go to next.
func AccessibilityTitleOverlay(next accessibility.TitleFetcher) accessibility.TitleFetcher {
	return titleOverlay{next: next}
}

type titleOverlay struct{ next accessibility.TitleFetcher }

// Fetch implements accessibility.TitleFetcher.
func (o titleOverlay) Fetch(ctx context.Context, raw string) (accessibility.PageTitle, int, error) {
	if t, ok := accessibilityTitles[raw]; ok {
		return t, 200, nil
	}
	if u, err := url.Parse(raw); err == nil {
		host := strings.ToLower(u.Hostname())
		if readinessDemoHosts[host] || strings.HasSuffix(host, ".acme.io") {
			return accessibility.PageTitle{}, 200, nil
		}
	}
	return o.next.Fetch(ctx, raw)
}
