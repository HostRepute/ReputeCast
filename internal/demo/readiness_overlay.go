package demo

import (
	"context"
	"hash/fnv"
	"net/url"
	"strings"

	"github.com/hostrepute/reputecast/internal/contentcheck"
)

// ReadinessDemoShortener is where the November update's shortened link leads.
const ReadinessDemoShortener = "https://acme.io/blog/november-2026-update?utm_source=newsletter"

// readinessDemoHosts are the sites the sample emails link to.
var readinessDemoHosts = map[string]bool{
	"acme.io": true, "github.com": true, "youtube.com": true, "www.youtube.com": true, "twitter.com": true, "x.com": true,
	"linkedin.com": true, "www.linkedin.com": true, "bit.ly": true, "go.acme-promo.link": true,
}

// ReadinessProberOverlay answers Send Readiness's link requests for the
// demo's links (acme.io and the sites the sample emails link to) without
// going to the network, so the demo reads like the boards offline; other
// links go to next.
func ReadinessProberOverlay(next contentcheck.Prober) contentcheck.Prober {
	return readinessProber{next: next}
}

type readinessProber struct{ next contentcheck.Prober }

// Probe implements contentcheck.Prober.
func (r readinessProber) Probe(ctx context.Context, urls []string) map[string]contentcheck.ProbeResult {
	out := map[string]contentcheck.ProbeResult{}
	rest := []string{}
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil {
			rest = append(rest, raw)
			continue
		}
		host := strings.ToLower(u.Hostname())
		switch {
		case host == "bit.ly":
			out[raw] = contentcheck.ProbeResult{Status: 200, Millis: 140, Final: ReadinessDemoShortener}
		case readinessDemoHosts[host] || strings.HasSuffix(host, ".acme.io"):
			out[raw] = contentcheck.ProbeResult{Status: 200, Millis: demoMillis(host + u.Path)}
		default:
			rest = append(rest, raw)
		}
	}
	if len(rest) > 0 && r.next != nil {
		for k, v := range r.next.Probe(ctx, rest) {
			out[k] = v
		}
	}
	return out
}

// demoMillis is a stable response time; the changelog is the slowest page.
func demoMillis(key string) int {
	if strings.Contains(key, "docs.acme.io/changelog") {
		return 840
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return 90 + int(h.Sum32()%480)
}
