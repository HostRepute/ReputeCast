package readiness

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Grow's rows in the Content group (design expansion phase 5, brief §6):
// "Referral block: shown to 31,604 of 31,604 recipients", or a warning
// when the referral program is off; "Recommendations block: 2
// recommendations show", with what is hidden and why.

// GrowRow is one of Grow's Content rows.
type GrowRow struct {
	Key    string
	Status string // pass or warn
	Title  string
	Detail string
	Label  string
	Short  string
	Ack    string
	// Link opens Grow ("Referral settings", "Open recommendations").
	LinkLabel, LinkHref string
}

// GrowChecker reads what Grow's blocks do in a campaign: the campaign's
// HTML, its recipients per list and how many there are.
type GrowChecker interface {
	ReadinessRows(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, html string, byList map[uuid.UUID]int64, recipients int64) []GrowRow
}

// grow adds Grow's rows to the Content group.
func (b *builder) grow() {
	for _, r := range b.f.Grow {
		c := ReadinessCheck{Key: r.Key, Group: GroupContent, Status: r.Status, Title: r.Title, Detail: r.Detail, Label: r.Label, Short: r.Short, Ack: r.Ack,
			Fixes: []ReadinessFix{}}
		if r.LinkHref != "" {
			c.Fixes = append(c.Fixes, link(r.LinkLabel, b.href(r.LinkHref)))
		}
		b.add(c)
	}
}
