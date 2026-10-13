package ses

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
)

// StatusFeedURL is the public AWS status RSS feed for SES in a region. It is
// read only while a provider is down, to say whether AWS reports an incident.
var StatusFeedURL = "https://status.aws.amazon.com/rss/ses-%s.rss"

type rss struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			PubDate     string `xml:"pubDate"`
			Description string `xml:"description"`
			Link        string `xml:"link"`
		} `xml:"item"`
	} `xml:"channel"`
}

// ServiceStatus implements providers.StatusReporter.
func (p *Provider) ServiceStatus(ctx context.Context) (providers.ServiceStatus, error) {
	st := providers.ServiceStatus{URL: "https://health.aws.amazon.com/health/status"}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(StatusFeedURL, p.s.Region), nil)
	if err != nil {
		return st, err
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return st, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return st, fmt.Errorf("AWS status feed: HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return st, err
	}
	var feed rss
	if err := xml.Unmarshal(raw, &feed); err != nil {
		return st, err
	}
	if len(feed.Channel.Items) == 0 {
		return st, nil
	}
	item := feed.Channel.Items[0]
	t, _ := time.Parse(time.RFC1123Z, item.PubDate)
	if t.IsZero() {
		t, _ = time.Parse(time.RFC1123, item.PubDate)
	}
	st.Updated = t
	title := strings.ToLower(item.Title)
	if time.Since(t) < 24*time.Hour && !strings.Contains(title, "[resolved]") && !strings.Contains(title, "operating normally") {
		st.Incident = true
		st.Summary = "AWS reports an SES incident in " + p.s.Region + "."
	}
	return st, nil
}
