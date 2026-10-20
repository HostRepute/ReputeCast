package app

import (
	"context"
	"sort"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		audience.Register(h, a.audience(), a.Secrets)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		audience.RegisterJobs(a.JobReg, a.audience(), a.Log)
	})
}

func (a *App) audience() *audience.Service {
	return &audience.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Segments: a.segments(), Mail: a.Mail, Links: a.Links, URLs: a.URLs,
		Answers: joinActivity(audienceAnswers(a), audienceStores(a)), SourceNames: audienceSourceNames(a)}
}

// audienceSourceNames names Grow's sources in the contacts list and the
// profile; Grow sets it from its wire file.
var audienceSourceNames = func(*App) audience.SourceNamer { return nil }

// audienceAnswers adds poll answers and survey responses to the contact
// timeline; the polls domain sets it from its wire file.
var audienceAnswers = func(*App) audience.ActivitySource { return nil }

// audienceStores adds store orders and abandoned checkouts; the commerce
// domain sets it from its wire file.
var audienceStores = func(*App) audience.ActivitySource { return nil }

// activitySources merges the timelines other domains keep, newest first.
type activitySources []audience.ActivitySource

// joinActivity is the sources that are set, as one.
func joinActivity(srcs ...audience.ActivitySource) audience.ActivitySource {
	var out activitySources
	for _, s := range srcs {
		if s != nil {
			out = append(out, s)
		}
	}
	switch len(out) {
	case 0:
		return nil
	case 1:
		return out[0]
	}
	return out
}

// ContactActivity implements audience.ActivitySource.
func (a activitySources) ContactActivity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, before *time.Time, beforeID string, limit int) ([]audience.ContactActivityItem, int64, error) {
	var items []audience.ContactActivityItem
	var total int64
	for _, s := range a {
		more, n, err := s.ContactActivity(ctx, ws, contactID, before, beforeID, limit)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, more...)
		total += n
	}
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].At.Equal(items[j].At) {
			return items[i].At.After(items[j].At)
		}
		return items[i].ID > items[j].ID
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, total, nil
}
