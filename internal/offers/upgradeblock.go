package offers

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Upgrade block's offer (ComposerEarnBlocks): the composer writes the
// offer's line, its button and a link that carries the offer
// ({{ upgrade_url | with_query: 'offer', 'FALL26' }}) into the card, and
// keeps the code in the block's settings attribute. Once the offer ends the
// draft goes back to the normal price when it's opened, and Send Readiness
// holds an email whose block still offers it.

// UpgradeBlockAttr is the Upgrade block's settings attribute (JSON, the
// composer's: web/src/features/earn/composer/upgrade.ts).
const UpgradeBlockAttr = "rc-upgrade"

// EndedUpgradeOffer is an offer an email's Upgrade block carries after it
// ended.
type EndedUpgradeOffer struct {
	Code string
	Name string
	// EndedOn is the day it ended, when known.
	EndedOn *time.Time
	// Path is the block's element path from mj-body.
	Path []int
}

// EndedUpgradeOffers are the offers of an email's Upgrade blocks that no
// longer work: ended, past their date, at their limit or gone from the
// provider.
func (s *Service) EndedUpgradeOffers(ctx context.Context, ws *store.Workspace, src string) []EndedUpgradeOffer {
	if !strings.Contains(src, UpgradeBlockAttr) {
		return nil
	}
	root, err := mjml.Parse(src)
	if err != nil {
		return nil
	}
	body := mjml.Body(root)
	if body == nil {
		return nil
	}
	mode, _, _ := s.account(ctx, ws)
	var out []EndedUpgradeOffer
	body.Walk(func(n *mjml.Node, path []int) {
		raw, ok := n.Get(UpgradeBlockAttr)
		if !ok || len(path) == 0 {
			return
		}
		var cfg struct {
			Offer string `json:"offer"`
		}
		if json.Unmarshal([]byte(raw), &cfg) != nil || strings.TrimSpace(cfg.Offer) == "" {
			return
		}
		r, found := s.byCode(ctx, ws, cfg.Offer)
		if found && mode != "" && s.live(ctx, ws, r, mode) {
			return
		}
		e := EndedUpgradeOffer{Code: cfg.Offer, Name: cfg.Offer, Path: append([]int(nil), path...)}
		if found {
			e.Name = r.Name
			switch {
			case r.EndedAt != nil:
				e.EndedOn = r.EndedAt
			case r.EndsAt != nil && !r.EndsAt.After(s.now()):
				e.EndedOn = r.EndsAt
			}
		}
		out = append(out, e)
	})
	return out
}
