// Package audience implements the Audience screens (spec 5.6, 7.9):
// contacts with saved views and bulk actions, contact profiles with consent
// history and an activity timeline, lists and custom fields, engagement
// tiers and the sunset policy, list health, audited CSV export and the
// shared suppression list.
package audience

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/erasure"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

type Service struct {
	Pool     *pgxpool.Pool
	Q        *store.Queries
	Audit    *audit.Logger
	Segments *segments.Service
	// Mail sends opt-in confirmations through the System stream.
	Mail  mailer.Sender
	Links *linktoken.Signer
	URLs  publicurl.Builder
	Now   func() time.Time
	// Answers adds poll answers and survey responses to the timeline (nil:
	// none).
	Answers ActivitySource
	// SourceNames names sources a label can't (Grow: "Referral · Sam
	// Rivera", "Partner · Ops Weekly"); nil: none.
	SourceNames SourceNamer
	// Erasure is every package's personal data about contacts, for erasing
	// and exporting (internal/app fills it; nil: the audience's own only).
	Erasure *erasure.Registry
}

// SourceNamer names contacts' sources: by contact, for the contacts whose
// source it knows. sources[i] is contacts[i]'s.
type SourceNamer func(ctx context.Context, ws uuid.UUID, contacts []uuid.UUID, sources []string) map[uuid.UUID]string

// nameSources puts SourceNames' names on the labels.
func (s *Service) nameSources(ctx context.Context, ws uuid.UUID, contacts []uuid.UUID, sources []string) map[uuid.UUID]string {
	if s.SourceNames == nil || len(contacts) == 0 {
		return nil
	}
	return s.SourceNames(ctx, ws, contacts, sources)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// AudienceTierCounts are contacts per engagement tier.
type AudienceTierCounts struct {
	Active  int64 `json:"active"`
	Cooling int64 `json:"cooling"`
	AtRisk  int64 `json:"atRisk"`
	Sunset  int64 `json:"sunset"`
}

// AudienceListSummary is a list with its size.
type AudienceListSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DoubleOptIn bool   `json:"doubleOptIn"`
	// Subscribed members (the list size); pending members wait for opt-in.
	Subscribed int64     `json:"subscribed"`
	Pending    int64     `json:"pending"`
	CreatedAt  time.Time `json:"createdAt"`
	// Origin is what fills the list: "store" for a connected store's
	// customers who accepted marketing (X20), empty otherwise.
	Origin string `json:"origin,omitempty"`
}

// AudienceSummary feeds the Audience tab counts and the filter selects.
type AudienceSummary struct {
	Contacts        int64                 `json:"contacts"`
	Segments        int64                 `json:"segments"`
	Suppressions    int64                 `json:"suppressions"`
	Tiers           AudienceTierCounts    `json:"tiers"`
	TiersComputedAt *time.Time            `json:"tiersComputedAt,omitempty"`
	Lists           []AudienceListSummary `json:"lists"`
	// PaidTiers are the Earn tiers of the Paid status filter, lowest first;
	// empty without tiers or without earn:view (no Paid filter or column).
	PaidTiers []AudiencePaidTier `json:"paidTiers"`
	// Stripe: the workspace keeps paid members in Stripe (connected now or
	// before), so erasing contacts reminds that Stripe keeps its records.
	Stripe bool `json:"stripe"`
}

// Summary returns the counts shown in the Audience header.
func (s *Service) Summary(ctx context.Context, ws *store.Workspace) (AudienceSummary, error) {
	c, err := s.Q.AudienceCounts(ctx, ws.ID)
	if err != nil {
		return AudienceSummary{}, err
	}
	out := AudienceSummary{Contacts: c.Contacts, Tiers: AudienceTierCounts{c.Active, c.Cooling, c.AtRisk, c.Sunset}, Lists: []AudienceListSummary{}}
	if out.Segments, err = s.Q.CountWorkspaceSegments(ctx, ws.ID); err != nil {
		return out, err
	}
	if out.Suppressions, err = s.Q.CountWorkspaceSuppressions(ctx, ws.ID); err != nil {
		return out, err
	}
	if st, err := s.Q.GetAudienceState(ctx, ws.ID); err == nil {
		out.TiersComputedAt = st.TiersComputedAt
	}
	lists, err := s.Q.ListListsWithCounts(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		out.Lists = append(out.Lists, AudienceListSummary{ID: l.ID.String(), Name: l.Name, Description: l.Description, DoubleOptIn: l.DoubleOptIn, Subscribed: l.Subscribed, Pending: l.Pending,
			CreatedAt: l.CreatedAt, Origin: l.Origin})
	}
	if out.PaidTiers, err = s.paidTiers(ctx, s.Q, ws.ID); err != nil {
		return out, err
	}
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM payment_connections WHERE workspace_id = $1 AND status <> 'setup')`, ws.ID).Scan(&out.Stripe); err != nil {
		return out, err
	}
	return out, nil
}

// actorID is the signed-in person, nil for API keys and agents.
func actorID(ctx context.Context) *uuid.UUID {
	if u := api.UserFrom(ctx); u != nil {
		id := u.ID
		return &id
	}
	return nil
}

// workspaceLocation is the workspace timezone (UTC when unknown).
func workspaceLocation(ws *store.Workspace) *time.Location {
	if loc, err := time.LoadLocation(ws.Timezone); err == nil {
		return loc
	}
	return time.UTC
}
