package campaigns

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/store"
)

// Polls and conditional content in previews and test sends (design
// expansion, X11). The polls and visibility domains plug in through
// Extras, so this package needs neither.

// Extras fill in a test email's poll links and cut it to one contact's
// version of conditional content.
type Extras interface {
	// TestLinks are the poll and survey links of a test email to one
	// address: its answers are that address's test answers, never counted.
	TestLinks(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, email string) (map[string]string, error)
	// TestVersion is the email a contact gets ("" contact: every block).
	TestVersion(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, contactID, html string) string
}

// pollVar reports whether a Liquid variable is a poll's answer link base
// ({{ rc_poll_a1b2c3 }}) or a survey's link ({{ rc_survey_12 }}): filled in
// per recipient like the unsubscribe link.
func pollVar(key string) bool {
	return strings.HasPrefix(key, "rc_poll_") || strings.HasPrefix(key, "rc_survey_")
}

// previewPollLinks are a preview's poll and survey links: they go nowhere,
// so nothing is recorded from the composer.
func previewPollLinks(src string, links map[string]string) {
	for _, u := range liquid.Uses(src) {
		switch {
		case strings.HasPrefix(u.Key, "rc_poll_"):
			links[u.Key] = "#poll-" + strings.TrimPrefix(u.Key, "rc_poll_")
		case strings.HasPrefix(u.Key, "rc_survey_"):
			links[u.Key] = "#survey-" + strings.TrimPrefix(u.Key, "rc_survey_")
		}
	}
}

// testPollLinks points a rendered test email's poll and survey links at
// the test address's own links.
func testPollLinks(html string, links map[string]string) string {
	for k, v := range links {
		switch {
		case strings.HasPrefix(k, "rc_poll_"):
			html = strings.ReplaceAll(html, `href="#poll-`+strings.TrimPrefix(k, "rc_poll_")+`/`, `href="`+v+`/`)
		case strings.HasPrefix(k, "rc_survey_"):
			html = strings.ReplaceAll(html, `href="#survey-`+strings.TrimPrefix(k, "rc_survey_")+`"`, `href="`+v+`"`)
		}
	}
	return html
}
