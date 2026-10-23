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
		args := A{"offer": o.Name, "dated": yes(o.EndedOn != nil)}
		if o.EndedOn != nil {
			args["day"] = b.day(*o.EndedOn)
		}
		parts := make([]string, len(o.Path))
		for i, p := range o.Path {
			parts[i] = fmt.Sprint(p)
		}
		b.add(check{Key: "content.upgrade_offer." + strings.Join(parts, "-"), Group: GroupContent, Status: Block, Label: tx("label.upgradeBlock", nil),
			Title:  tx("upgrade.ended", args),
			Detail: tx("upgrade.endedDetail", nil),
			Fixes:  []fix{link(tx("fix.openInTheComposer", nil), b.editor(o.Path, "settings"))}})
	}
}
