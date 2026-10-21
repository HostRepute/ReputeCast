package readiness

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// An Upgrade block's offer in the Content group (Earn, design expansion
// X17): a block that still offers an offer that ended would promise a price
// the upgrade page no longer gives. Opening the draft puts the normal price
// back.

// EndedUpgradeOffer is an offer an email's Upgrade block carries after it
// ended.
type EndedUpgradeOffer struct {
	Code    string
	Name    string
	EndedOn *time.Time
	// Path is the block's element path from mj-body.
	Path []int
}

// UpgradeOfferChecker finds the ended offers of an email's Upgrade blocks;
// nil leaves the check out.
type UpgradeOfferChecker interface {
	EndedUpgradeOffers(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, src string) []EndedUpgradeOffer
}

// upgradeOffers adds a blocking check per Upgrade block whose offer ended.
func (b *builder) upgradeOffers() {
	for _, o := range b.f.UpgradeOffers {
		title := fmt.Sprintf("Upgrade block: %s has ended, and the block still offers it", o.Name)
		if o.EndedOn != nil {
			title = fmt.Sprintf("Upgrade block: %s ended on %s, and the block still offers it", o.Name, b.day(*o.EndedOn))
		}
		parts := make([]string, len(o.Path))
		for i, p := range o.Path {
			parts[i] = fmt.Sprint(p)
		}
		b.add(ReadinessCheck{Key: "content.upgrade_offer." + strings.Join(parts, "-"), Group: GroupContent, Status: Block, Label: "Upgrade block",
			Title:  title,
			Detail: "Readers would be promised a price the upgrade page no longer gives. Open the email in the composer: the block goes back to the normal price.",
			Fixes:  []ReadinessFix{link("Open in the composer", b.editor(o.Path, "settings"))}})
	}
}
