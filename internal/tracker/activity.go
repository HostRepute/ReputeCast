package tracker

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

var _ audience.ActivitySource = (*Service)(nil)

// ContactActivity implements audience.ActivitySource: a line per day (in
// the workspace's time zone) and site the contact was on it, "Viewed 3
// products on shop.acme.io" or "Visited shop.acme.io", newest first.
func (s *Service) ContactActivity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, before *time.Time, beforeID string, limit int) ([]audience.ContactActivityItem, int64, error) {
	tz := ws.Timezone
	if _, err := time.LoadLocation(tz); err != nil || tz == "" {
		tz = "UTC"
	}
	total, err := s.Q.TrackerContactDayCount(ctx, store.TrackerContactDayCountParams{ContactID: &contactID, WorkspaceID: ws.ID, Tz: tz})
	if err != nil || total == 0 {
		return []audience.ContactActivityItem{}, 0, err
	}
	bid, _ := uuid.Parse(beforeID)
	rows, err := s.Q.TrackerContactDays(ctx, store.TrackerContactDaysParams{ContactID: &contactID, WorkspaceID: ws.ID, Tz: tz, Before: before, BeforeID: bid,
		Lim: int32(max(limit, 1))})
	if err != nil {
		return nil, 0, err
	}
	l := i18n.LangFrom(ctx)
	out := make([]audience.ContactActivityItem, 0, len(rows))
	for _, r := range rows {
		host := r.Origin
		if u, err := url.Parse(r.Origin); err == nil && u.Host != "" {
			host = u.Host
		}
		it := audience.ContactActivityItem{ID: r.ID.String(), Category: "engagement", Type: "site_visit", At: r.At, URL: host}
		if r.Products > 0 {
			it.Subject = mActivityViewed.Text(l, i18n.Args{"count": r.Products, "host": host})
		} else {
			it.Subject = mActivityVisited.Text(l, i18n.Args{"host": host})
		}
		// The products' names (three at most), then the add-to-carts.
		detail := strings.Join(r.Titles, ", ")
		if r.Carts > 0 {
			if detail != "" {
				detail += " · "
			}
			detail += mActivityCarts.Text(l, i18n.Args{"count": r.Carts})
		}
		it.Detail = detail
		it.SourceLabel = mActivitySource.Text(l, nil)
		out = append(out, it)
	}
	return out, total, nil
}
