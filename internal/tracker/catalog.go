package tracker

import (
	"context"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// CatalogFacts are the Website tracking card's facts in Settings ›
// Integrations: the live sites, their events today and the last one, and
// the fact line in the reader's language ("2 sites · 1,284 events today").
type CatalogFacts struct {
	Sites       int64
	EventsToday int64
	LastEventAt *time.Time
	Line        string
}

// Catalog reads the card's facts.
func (s *Service) Catalog(ctx context.Context, ws *store.Workspace) (CatalogFacts, error) {
	r, err := s.Q.TrackerCatalogFacts(ctx, store.TrackerCatalogFactsParams{WorkspaceID: ws.ID, Since: startOfDay(ws, s.now())})
	if err != nil {
		return CatalogFacts{}, err
	}
	out := CatalogFacts{Sites: r.Sites, EventsToday: r.Events,
		Line: mHubFacts.Text(i18n.LangFrom(ctx), i18n.Args{"sites": r.Sites, "events": r.Events})}
	if r.LastEventAt.After(epoch) {
		at := r.LastEventAt
		out.LastEventAt = &at
	}
	return out, nil
}
