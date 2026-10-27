package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/placement"
	"github.com/hostrepute/reputecast/internal/sending/engine"
)

// canarySeeds sends a campaign's content to the workspace's seed inboxes
// as a placement test when its canary starts (review GS-10), and reads
// where the copies landed for the canary's seed gate.
type canarySeeds struct{ a *App }

// StartSeeds starts a placement test of the campaign through its route,
// to every connected seed inbox.
func (s canarySeeds) StartSeeds(ctx context.Context, ws, campaignID uuid.UUID) (uuid.UUID, int, error) {
	w, err := s.a.Q.GetWorkspace(ctx, ws)
	if err != nil {
		return uuid.Nil, 0, err
	}
	d, err := s.a.placement().Start(ctx, &w, placement.PlacementRunInput{Content: placement.PlacementContentRef{Kind: "campaign", ID: campaignID.String()},
		Through: "route", Origin: "review"})
	if err != nil {
		return uuid.Nil, 0, err
	}
	id, err := uuid.Parse(d.ID)
	if err != nil {
		return uuid.Nil, 0, err
	}
	return id, len(d.Seeds), nil
}

// SeedPlacement says where a canary's seed copies landed, per mailbox
// provider (Gmail with Google Workspace, Outlook with Microsoft 365, Yahoo
// with AOL; GMX and Web.de are Other).
func (s canarySeeds) SeedPlacement(ctx context.Context, ws, check uuid.UUID) (engine.SeedPlacement, error) {
	var out engine.SeedPlacement
	w, err := s.a.Q.GetWorkspace(ctx, ws)
	if err != nil {
		return out, err
	}
	d, err := s.a.placement().Test(ctx, &w, check)
	if err != nil {
		return out, err
	}
	by := map[string]*engine.SeedMailboxResult{}
	order := []string{}
	done := d.Status == "finished" || d.Status == "stopped"
	for _, c := range d.Copies {
		mb := seedMailbox(c.SeedProvider)
		r := by[mb]
		if r == nil {
			r = &engine.SeedMailboxResult{Mailbox: mb}
			by[mb] = r
			order = append(order, mb)
		}
		r.Seeds++
		switch {
		case c.Status == "found" && c.Folder == "spam":
			r.Spam++
			out.Spam++
		case c.Status == "found" && c.Folder == "tab" && c.Tab != "" && c.Tab != "Primary" && c.Tab != "Focused":
			r.Promotions++
			out.Promotions++
		case c.Status == "found":
			r.Inbox++
			out.Inbox++
		case done:
			// Never arrived: neither inbox nor spam.
			out.Missing++
		default:
			out.Pending++
		}
	}
	for _, k := range engine.SeedMailboxOrder(order) {
		out.ByMailbox = append(out.ByMailbox, *by[k])
	}
	return out, nil
}

// seedMailbox is the mailbox provider a seed inbox stands for.
func seedMailbox(provider string) string {
	switch provider {
	case "gmail", "google_workspace":
		return "gmail"
	case "outlook", "microsoft_365":
		return "outlook"
	case "yahoo", "aol":
		return "yahoo"
	case "icloud":
		return "icloud"
	}
	return "other"
}
