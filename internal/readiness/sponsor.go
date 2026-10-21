package readiness

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// The Sponsor block in the Content group (Earn, design expansion X17, brief
// §3.12): the issue can't go out while the sponsor's creative is waiting,
// the block must hold what the booking has now, and sponsored content is
// always labelled. A block without a booking is left out of the email.

// Fix actions of the Sponsor block.
const (
	FixSponsorRemove  = "sponsor_remove_block"
	FixSponsorRefresh = "sponsor_refresh_block"
)

// Sponsor block states, as internal/sponsors names them.
const (
	sponsorReady   = "ready"
	sponsorWaiting = "waiting"
	sponsorNone    = "none"
)

// SponsorFacts are what Send Readiness checks of an email's Sponsor block.
type SponsorFacts struct {
	HasBlock bool
	// State is ready (the creative), waiting (a placeholder) or none (no
	// booking: the block is left out).
	State   string
	Sponsor string
	Issue   int
	// BookingID and CreativeDue name the booking to open.
	BookingID   string
	CreativeDue *time.Time
	// Stale: the creative or the booking changed since the email was saved.
	Stale bool
	// Labelled: the block carries the "Sponsored" label.
	Labelled bool
}

// SponsorChecker gives the facts of an email's Sponsor block and applies
// its fixes; nil leaves the checks out.
type SponsorChecker interface {
	SponsorFacts(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, src string) SponsorFacts
	FixSponsor(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, src, action string) (next, applied string, err error)
}

// possessive is a sponsor's name as an owner: "Northwind Analytics’",
// "Contoso Cloud’s".
func possessive(name string) string {
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(name)), "s") {
		return name + "’"
	}
	return name + "’s"
}

// sponsor adds the Sponsor block's checks to the Content group.
func (b *builder) sponsor() {
	sf := b.f.Sponsor
	if !sf.HasBlock {
		return
	}
	booking := b.href("/earn/sponsors")
	if sf.BookingID != "" {
		booking += "?booking=" + sf.BookingID
	}
	open := link("Open booking", booking)
	issue := fmt.Sprintf("#%d", sf.Issue)
	switch {
	case sf.State == sponsorWaiting:
		detail := "Review & send blocks the issue until it’s ready."
		if sf.CreativeDue != nil {
			detail = fmt.Sprintf("Due %s. %s", b.day(*sf.CreativeDue), detail)
		}
		b.add(ReadinessCheck{Key: "content.sponsor.creative", Group: GroupContent, Status: Block, Label: "Sponsor block",
			Title: fmt.Sprintf("Sponsor block: %s creative is missing", possessive(sf.Sponsor)), Detail: detail,
			Fixes: []ReadinessFix{open, apply(FixSponsorRemove, "Remove the block", "danger", nil)}})
	case sf.Stale && sf.State == sponsorReady:
		b.add(ReadinessCheck{Key: "content.sponsor.stale", Group: GroupContent, Status: Block, Label: "Sponsor block",
			Title:  fmt.Sprintf("Sponsor block: %s creative changed since this email was saved", possessive(sf.Sponsor)),
			Detail: "Update the block so the issue goes out with the creative as the booking has it now.",
			Fixes:  []ReadinessFix{apply(FixSponsorRefresh, "Update the block", "primary", nil), open}})
	case sf.Stale:
		title := "Sponsor block: no sponsor is booked for this email any more"
		if sf.Issue > 0 {
			title = fmt.Sprintf("Sponsor block: no sponsor is booked for %s any more", issue)
		}
		b.add(ReadinessCheck{Key: "content.sponsor.stale", Group: GroupContent, Status: Block, Label: "Sponsor block",
			Title:  title,
			Detail: "The block still holds an earlier booking's creative. Update it and the email leaves the block out.",
			Fixes:  []ReadinessFix{apply(FixSponsorRefresh, "Update the block", "primary", nil), apply(FixSponsorRemove, "Remove the block", "danger", nil)}})
	case sf.State == sponsorNone:
		title := "Sponsor block: no sponsor is booked for this email, so the block is left out"
		if sf.Issue > 0 {
			title = fmt.Sprintf("Sponsor block: no sponsor is booked for %s, so the block is left out", issue)
		}
		b.add(ReadinessCheck{Key: "content.sponsor.none", Group: GroupContent, Status: Pass, Label: "Sponsor block", Title: title, Short: "Sponsor block left out"})
	}
	if sf.State == sponsorNone {
		return
	}
	if sf.Labelled {
		b.add(ReadinessCheck{Key: "content.sponsor.label", Group: GroupContent, Status: Pass, Label: "Sponsored label",
			Title: "Sponsored content is labelled", Short: "sponsored content labelled"})
		return
	}
	b.add(ReadinessCheck{Key: "content.sponsor.label", Group: GroupContent, Status: Block, Label: "Sponsored label",
		Title:  "Sponsored content isn’t labelled",
		Detail: "The Sponsor block always says “Sponsored”. Update the block to put the label back.",
		Fixes:  []ReadinessFix{apply(FixSponsorRefresh, "Update the block", "primary", nil)}})
}
