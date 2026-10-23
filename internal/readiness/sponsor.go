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
	open := link(tx("fix.openBooking", nil), booking)
	label := tx("label.sponsorBlock", nil)
	issue := A{"issue": fmt.Sprint(sf.Issue), "numbered": yes(sf.Issue > 0)}
	sponsor := A{"sponsor": sf.Sponsor, "endsInS": yes(strings.HasSuffix(strings.ToLower(strings.TrimSpace(sf.Sponsor)), "s"))}
	switch {
	case sf.State == sponsorWaiting:
		detail := tx("sponsor.blocksIssue", nil)
		if sf.CreativeDue != nil {
			detail = sentences(tx("sponsor.due", A{"day": b.day(*sf.CreativeDue)}), detail)
		}
		b.add(check{Key: "content.sponsor.creative", Group: GroupContent, Status: Block, Label: label,
			Title: tx("sponsor.missing", sponsor), Detail: detail,
			Fixes: []fix{open, apply(FixSponsorRemove, tx("fix.removeBlock", nil), "danger", nil)}})
	case sf.Stale && sf.State == sponsorReady:
		b.add(check{Key: "content.sponsor.stale", Group: GroupContent, Status: Block, Label: label,
			Title:  tx("sponsor.changed", sponsor),
			Detail: tx("sponsor.changedDetail", nil),
			Fixes:  []fix{apply(FixSponsorRefresh, tx("fix.updateBlock", nil), "primary", nil), open}})
	case sf.Stale:
		b.add(check{Key: "content.sponsor.stale", Group: GroupContent, Status: Block, Label: label,
			Title:  tx("sponsor.noLonger", issue),
			Detail: tx("sponsor.noLongerDetail", nil),
			Fixes:  []fix{apply(FixSponsorRefresh, tx("fix.updateBlock", nil), "primary", nil), apply(FixSponsorRemove, tx("fix.removeBlock", nil), "danger", nil)}})
	case sf.State == sponsorNone:
		b.add(check{Key: "content.sponsor.none", Group: GroupContent, Status: Pass, Label: label, Title: tx("sponsor.leftOut", issue), Short: tx("sponsor.leftOutShort", nil)})
	}
	if sf.State == sponsorNone {
		return
	}
	if sf.Labelled {
		b.add(check{Key: "content.sponsor.label", Group: GroupContent, Status: Pass, Label: tx("label.sponsoredLabel", nil),
			Title: tx("sponsor.labelled", nil), Short: tx("sponsor.labelledShort", nil)})
		return
	}
	b.add(check{Key: "content.sponsor.label", Group: GroupContent, Status: Block, Label: tx("label.sponsoredLabel", nil),
		Title:  tx("sponsor.unlabelled", nil),
		Detail: tx("sponsor.unlabelledDetail", nil),
		Fixes:  []fix{apply(FixSponsorRefresh, tx("fix.updateBlock", nil), "primary", nil)}})
}
