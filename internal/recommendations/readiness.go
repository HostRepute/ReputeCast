package recommendations

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/hostrepute/reputecast/internal/store"
)

var recsMarker = regexp.MustCompile(`<!--\s*rc:recommendations\s*(\{[\s\S]*?\})?\s*-->`)

// BlockCheck is what the Recommendations block of an email shows, for
// Send Readiness and the composer's checks.
type BlockCheck struct {
	Present bool
	Shows   int
	Hidden  []BlockHiddenRec
}

// CheckBlock reads the Recommendations block of an email.
func (s *Service) CheckBlock(ctx context.Context, ws *store.Workspace, html string) (BlockCheck, error) {
	m := recsMarker.FindStringSubmatch(html)
	if m == nil {
		return BlockCheck{}, nil
	}
	var raw json.RawMessage
	if m[1] != "" {
		raw = json.RawMessage(m[1])
	}
	r, err := s.RenderBlock(ctx, ws, BlockRenderInput{Settings: raw})
	if err != nil {
		return BlockCheck{Present: true}, err
	}
	return BlockCheck{Present: true, Shows: r.Shows, Hidden: r.Hidden}, nil
}

// HiddenReason is why a recommendation doesn't show, as a note reads it.
func HiddenReason(state string) string {
	switch state {
	case "paused":
		return "paused"
	case "partner_paused":
		return "paused by the partner"
	case "invalid":
		return "partner link invalid"
	case "limit":
		return "daily limit reached"
	case "ended":
		return "partnership ended"
	case "not_in_emails":
		return "confirmation page only"
	}
	return state
}
