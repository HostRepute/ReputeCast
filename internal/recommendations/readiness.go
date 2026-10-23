package recommendations

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/hostrepute/reputecast/internal/i18n"
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

// HiddenReason is why a recommendation doesn't show, as a note reads it
// ("paused by the partner"): a kept message, which Send Readiness writes in
// each reader's language.
func HiddenReason(state string) i18n.Text {
	switch state {
	case "paused", "partner_paused", "invalid", "limit", "ended", "not_in_emails":
		return msgs.M("readiness.hidden." + state).Keep(nil)
	}
	return msgs.M("readiness.hidden.other").Keep(i18n.Args{"state": state})
}

// ReadinessRow is the Recommendations block's row of Send Readiness. Its
// texts are kept messages: Send Readiness writes them in each reader's
// language. Detail is a sentence per hidden recommendation.
type ReadinessRow struct {
	Status                   string // pass or warn
	Title, Ack, Label, Short i18n.Text
	Detail                   []i18n.Text
	Link                     i18n.Text
}

// ReadinessRow is what the block shows, and what it hides and why.
func (c BlockCheck) ReadinessRow() ReadinessRow {
	row := ReadinessRow{Status: "pass", Label: msgs.M("readiness.label").Keep(nil), Short: msgs.M("readiness.short").Keep(nil), Link: msgs.M("readiness.open").Keep(nil)}
	if c.Shows == 0 {
		row.Status = "warn"
		row.Title = msgs.M("readiness.none").Keep(nil)
		row.Ack = msgs.M("readiness.noneAck").Keep(nil)
	} else {
		row.Title = msgs.M("readiness.shows").Keep(i18n.Args{"count": c.Shows})
	}
	for _, h := range c.Hidden {
		if h.State == "not_in_emails" {
			continue
		}
		row.Detail = append(row.Detail, msgs.M("readiness.hiddenNote").Keep(i18n.Args{"name": h.Name, "reason": HiddenReason(h.State)}))
	}
	return row
}
