package readiness

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// The publication website's web checks in the Content group (design
// expansion phase 7): a campaign that also publishes to the website gets
// its post's checks — the address is free, the cover has alt text, the
// tokens that show their fallback on the web and the blocks the web
// leaves out. A taken address blocks the send: the post couldn't go live
// with it.

// WebCheck is one web check of a campaign's post.
type WebCheck struct {
	Key    string // address, cover, tokens, hidden
	Status string // pass, warn, block
	Title  string
	Detail string
	// Href opens the composer's Web tab.
	Href string
}

// WebFacts are a campaign's post on the website, when it publishes with the send.
type WebFacts struct {
	Ran    bool
	Checks []WebCheck
}

// WebsiteChecker gives a campaign's web checks; nil leaves them out.
type WebsiteChecker interface {
	WebFacts(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) WebFacts
}

// web adds the web checks to the Content group.
func (b *builder) web() {
	if !b.f.Web.Ran {
		return
	}
	for _, c := range b.f.Web.Checks {
		status := Pass
		switch c.Status {
		case "warn", "info":
			status = Warn
			if c.Status == "info" {
				status = Pass
			}
		case "block":
			status = Block
		}
		ch := ReadinessCheck{Key: "content.web." + c.Key, Group: GroupContent, Status: status, Label: "Web", Title: c.Title, Detail: c.Detail, Short: c.Title}
		if c.Href != "" {
			ch.Fixes = []ReadinessFix{link("Open the Web tab", c.Href)}
		}
		if status == Warn {
			ch.Ack = "I understand: " + c.Title + "."
		}
		b.add(ch)
	}
}
