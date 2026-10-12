package deliverability

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// timelineDays and timelineItems bound the incident timeline.
const (
	timelineDays  = 45
	timelineItems = 8
)

// Incident kinds other domains record (see RecordIncident).
const (
	KindAutoPause       = "auto_pause"
	KindWarmupSlowdown  = "warmup_slowdown"
	KindWebhooksSilent  = "webhooks_silent"
	KindProviderOutage  = "provider_outage"
	KindBlocklistListed = "blocklist_listed"
	KindMigration       = "migration"
)

// Incident is an entry of the incident timeline recorded by another domain:
// the sending engine (auto-pauses), provider health (outages, silent
// webhooks), the warm-up planner (slowdowns) or the importer (migrations).
type Incident struct {
	Kind     string
	Severity string // info, ok, warn, bad
	Title    string
	Detail   string
	// CampaignID and ProviderID link the incident; an incident with a
	// provider but no campaign is linked to the campaign that was sending
	// through that provider at the time.
	CampaignID *uuid.UUID
	ProviderID *uuid.UUID
	Asset      string
	OccurredAt time.Time
	ResolvedAt *time.Time
}

// RecordIncident adds an incident to the timeline, inside the caller's
// transaction.
func RecordIncident(ctx context.Context, q *store.Queries, workspaceID uuid.UUID, in Incident) error {
	if in.OccurredAt.IsZero() {
		in.OccurredAt = time.Now()
	}
	if in.Severity == "" {
		in.Severity = "info"
	}
	_, err := q.InsertIncident(ctx, store.InsertIncidentParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: workspaceID, OccurredAt: in.OccurredAt, Kind: in.Kind, Severity: in.Severity,
		Title: in.Title, Detail: in.Detail, CampaignID: in.CampaignID, ProviderID: in.ProviderID, ResolvedAt: in.ResolvedAt,
		Source: "reputecast", Asset: in.Asset,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

// timeline reads the incident timeline with its links.
func (s *Service) timeline(ctx context.Context, st *state) (IncidentTimeline, error) {
	since := st.now.AddDate(0, 0, -timelineDays)
	rows, err := s.Q.ListDeliverabilityIncidents(ctx, store.ListDeliverabilityIncidentsParams{WorkspaceID: st.ws, Since: since, Lim: timelineItems})
	if err != nil {
		return IncidentTimeline{}, err
	}
	total, err := s.Q.CountDeliverabilityIncidents(ctx, store.CountDeliverabilityIncidentsParams{WorkspaceID: st.ws, Since: since})
	if err != nil {
		return IncidentTimeline{}, err
	}
	out := IncidentTimeline{Items: make([]IncidentItem, 0, len(rows)), Total: total}
	for _, r := range rows {
		it := IncidentItem{ID: r.ID.String(), OccurredAt: r.OccurredAt, Kind: r.Kind, Severity: r.Severity, Title: r.Title, Detail: r.Detail,
			Source: r.Source, Links: []IncidentLink{}}
		if r.Source == "hostrepute" {
			out.FromHostRepute = true
		}
		if r.Kind == KindBlocklistListed {
			it.Detail = st.withDelisting(it.Detail, r.Asset, r.Title)
		}
		it.Links = links(r)
		out.Items = append(out.Items, it)
	}
	return out, nil
}

// withDelisting appends the current delisting state of a listing to its
// incident ("Delisting requested 8 Oct, pending.").
func (st *state) withDelisting(detail, asset, title string) string {
	for _, b := range st.blocklists {
		if !b.Listed || !strings.EqualFold(b.Asset, asset) || b.DelistingRequestedAt == nil || b.Delisting == "" {
			continue
		}
		if !strings.Contains(strings.ToLower(title), strings.ToLower(shortSource(b.Source))) {
			continue
		}
		day := b.DelistingRequestedAt.In(st.loc()).Format("2 Jan")
		switch b.Delisting {
		case "requested", "pending":
			return strings.TrimSpace(detail + " Delisting requested " + day + ", pending.")
		case "denied":
			return strings.TrimSpace(detail + " Delisting requested " + day + ", denied.")
		case "delisted":
			return strings.TrimSpace(detail + " Delisted after a request on " + day + ".")
		}
	}
	return detail
}

// shortSource is how alerts name a source: "UCEPROTECT Level 3" → "UCEPROTECT L3".
func shortSource(s string) string {
	return strings.Replace(s, " Level ", " L", 1)
}

func (st *state) loc() *time.Location {
	if st.zone != nil {
		return st.zone
	}
	return time.UTC
}

// links are the screens an incident points to.
func links(r store.ListDeliverabilityIncidentsRow) []IncidentLink {
	out := []IncidentLink{}
	campaign := r.CampaignID != uuid.Nil
	switch r.Kind {
	case KindWarmupSlowdown:
		out = append(out, IncidentLink{Kind: "warmup", Label: "Warm-up plan"})
		if campaign {
			out = append(out, IncidentLink{Kind: "campaign", Label: r.CampaignName, ID: r.CampaignID.String()})
		}
	case KindAutoPause:
		if campaign {
			out = append(out, IncidentLink{Kind: "campaign_report", Label: "Report", ID: r.CampaignID.String()})
		}
	case KindWebhooksSilent, KindProviderOutage:
		out = append(out, IncidentLink{Kind: "provider_events", Label: "Details"})
	case KindBlocklistListed, "blocklist_delisted":
		if r.ProviderID != nil && r.ProviderName != "" {
			out = append(out, IncidentLink{Kind: "provider", Label: r.ProviderName, ID: r.ProviderID.String()})
		}
	case KindMigration, "reputation_change":
		out = append(out, IncidentLink{Kind: "import", Label: "Import summary"})
	default:
		if campaign {
			out = append(out, IncidentLink{Kind: "campaign", Label: r.CampaignName, ID: r.CampaignID.String()})
		}
	}
	return out
}
