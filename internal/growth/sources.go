package growth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() {
	segments.RegisterFieldOptions("growth_source", segmentOptions)
}

// GrowthSourceOption is a source the contacts filter offers, with how many
// contacts came from it.
type GrowthSourceOption struct {
	Source   GrowthSource `json:"source"`
	Contacts int64        `json:"contacts"`
}

// GrowthSourceList is every source contacts joined through.
type GrowthSourceList struct {
	Items []GrowthSourceOption `json:"items"`
}

// Sources lists the sources of the workspace's contacts, biggest first.
func (s *Service) Sources(ctx context.Context, ws *store.Workspace) (GrowthSourceList, error) {
	out := GrowthSourceList{Items: []GrowthSourceOption{}}
	n, err := s.names(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	rows, err := s.Q.GrowthSourceCounts(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		src := n.describe(SourceKey(r.SourceKind, r.SourceID))
		first := r.FirstJoin
		src.FirstJoin = &first
		out.Items = append(out.Items, GrowthSourceOption{Source: src, Contacts: r.Contacts})
	}
	return out, nil
}

// EnglishLabel names a source in English, for places that have no app
// text of their own (segment condition values, CSV exports).
func EnglishLabel(src GrowthSource) string {
	switch src.Kind {
	case KindForm:
		if src.Name != "" {
			return src.Name
		}
		return "Signup form"
	case KindSubscribePage:
		return "Subscribe page"
	case KindAPIKey:
		if src.Name != "" {
			return "API key “" + src.Name + "”"
		}
		return "API"
	case KindIntegration:
		if src.Name != "" {
			return src.Name
		}
		return "Integration"
	case KindPerson:
		return "Added by a person"
	case KindPreferenceCenter:
		return "Preference center"
	case KindImport:
		name := src.Name
		if name == "" {
			name = "File"
		}
		if src.ImportedAt != nil {
			return fmt.Sprintf("%s import (%s)", name, src.ImportedAt.Format("2 Jan"))
		}
		return name + " import"
	case KindCarried:
		if strings.HasPrefix(src.Carried, "import:") {
			return src.Name + " import"
		}
		if src.Name != "" {
			return src.Name
		}
		return src.Carried
	case "other":
		return "Other"
	}
	return "Unknown"
}

// segmentOptions lists the growth sources for the segment builder.
func segmentOptions(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]segments.SegmentOption, error) {
	ws, err := q.GetWorkspace(ctx, wsID)
	if err != nil {
		return nil, err
	}
	s := &Service{Q: q}
	list, err := s.Sources(ctx, &ws)
	if err != nil {
		return nil, err
	}
	out := make([]segments.SegmentOption, 0, len(list.Items))
	for _, it := range list.Items {
		out = append(out, segments.SegmentOption{Value: it.Source.Key, Label: EnglishLabel(it.Source)})
	}
	return out, nil
}

// ContactJoin is how a contact first joined, for the profile's join line:
// "Joined Mon 13 Oct · Newsletter signup · linkedin / social / flags-ga ·
// from docs.acme.io/flags".
type ContactJoin struct {
	Attributed  bool         `json:"attributed"`
	Source      GrowthSource `json:"source"`
	JoinedAt    *time.Time   `json:"joinedAt,omitempty"`
	UTMSource   string       `json:"utmSource"`
	UTMMedium   string       `json:"utmMedium"`
	UTMCampaign string       `json:"utmCampaign"`
	Referrer    string       `json:"referrer"`
	Page        string       `json:"page"`
	LeftAt      *time.Time   `json:"leftAt,omitempty"`
	LeftKind    string       `json:"leftKind,omitempty"`
}

// ContactJoin returns a contact's attribution.
func (s *Service) ContactJoin(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) (ContactJoin, error) {
	out := ContactJoin{Source: GrowthSource{Hosts: []string{}}}
	if p := api.PrincipalFrom(ctx); p != nil && len(p.Scope.Lists) > 0 {
		var ok bool
		if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM list_memberships WHERE contact_id = $1 AND list_id = ANY ($2))`, contactID, p.Scope.Lists).Scan(&ok); err != nil {
			return out, err
		}
		if !ok {
			return out, api.NotFound("Contact")
		}
	}
	g, err := s.Q.GrowthContactTouch(ctx, contactID)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if g.WorkspaceID != ws.ID {
		return out, api.NotFound("Contact")
	}
	n, err := s.names(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	out.Attributed = true
	out.Source = n.describe(SourceKey(g.SourceKind, g.SourceID))
	joined := g.JoinedAt
	out.JoinedAt = &joined
	out.UTMSource, out.UTMMedium, out.UTMCampaign, out.Referrer, out.Page = g.UtmSource, g.UtmMedium, g.UtmCampaign, g.Referrer, g.Page
	out.LeftAt = g.LeftAt
	if g.LeftKind != nil {
		out.LeftKind = *g.LeftKind
	}
	return out, nil
}
