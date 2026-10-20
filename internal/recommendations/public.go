package recommendations

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
)

// PublicRec is a recommendation on a public page.
type PublicRec struct {
	ID          string
	Kind        string
	Name        string
	Description string
	Initials    string
	// LogoURL is the logo on this instance (never the partner's server).
	LogoURL string
	// Host is where Visit goes ("fabrikam.example").
	Host string
	// VisitURL counts the click and sends the reader to the link.
	VisitURL string
	// Outcome after Subscribe: accepted, passed, limit, unavailable.
	Outcome string
}

// PublicRecs are the recommendations a public page shows.
type PublicRecs struct {
	// Publisher names the workspace ("Acme"); ListName the reader's list,
	// which subscribing doesn't change.
	Publisher string
	ListName  string
	Items     []PublicRec
	// Token lets the page's Subscribe buttons act for the reader; Email is
	// the address a partner will write to.
	Token string
	Email string
}

// PageToken is the reader's signed link to the recommendations page.
func (s *Service) PageToken(wsID, contactID uuid.UUID, at time.Time) string {
	if s.Links == nil {
		return ""
	}
	return s.Links.Sign(linktoken.Claims{Kind: linktoken.Recommendations, WorkspaceID: wsID, ContactID: contactID, Link: linktoken.IssueDay(at)})
}

// ErrExpired: the personal link expired; the page asks for an address.
var ErrExpired = errors.New("recommendations: the link expired")

// Reader reads a recommendations page token: the reader, or ErrExpired.
func (s *Service) Reader(ws *store.Workspace, token string) (uuid.UUID, error) {
	if s.Links == nil || token == "" {
		return uuid.Nil, ErrExpired
	}
	c, err := s.Links.Verify(token, linktoken.Recommendations)
	if err != nil || c.WorkspaceID != ws.ID {
		return uuid.Nil, ErrExpired
	}
	if s.now().Sub(linktoken.IssuedOn(c.Link)) > PageValid+24*time.Hour {
		return uuid.Nil, ErrExpired
	}
	return c.ContactID, nil
}

// showing are the recommendations readers see now, in order: those for the
// confirmation page or for emails.
func (s *Service) showing(ctx context.Context, ws *store.Workspace, confirm bool) ([]store.Recommendation, error) {
	recs, err := s.Q.RecList(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	var out []store.Recommendation
	for _, r := range recs {
		if confirm && !r.ShowConfirm || !confirm && !r.ShowBlock {
			continue
		}
		var link *store.PartnerLink
		if r.PartnerLinkID != nil {
			if l, err := s.Q.PartnerLinkByID(ctx, *r.PartnerLinkID); err == nil {
				link = &l
			}
		}
		if _, hidden := visibility(r, link, now); hidden {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Service) publicRec(ctx context.Context, ws *store.Workspace, r store.Recommendation, visit func(string) string) PublicRec {
	p := PublicRec{ID: r.ID.String(), Kind: r.Kind, Name: r.Name, Description: r.Description, Initials: initials(r.Name)}
	if r.Kind == KindLink {
		p.Host = hostOf(r.Url, false)
		p.VisitURL = visit(r.ID.String())
	}
	if img := s.image(ctx, ws, r.LogoID); img != nil {
		p.LogoURL = strings.TrimRight(s.instance(), "/") + img.URL
		if s.instance() == "" {
			p.LogoURL = img.URL
		}
	}
	return p
}

// readerList is the list a reader's subscription stays on ("doesn't change
// your Weekly Digest"): the partner profile's list when they're on it,
// else their first.
func (s *Service) readerList(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) string {
	var name string
	_ = s.Pool.QueryRow(ctx, `SELECT l.name FROM list_memberships m JOIN lists l ON l.id = m.list_id AND l.deleted_at IS NULL
		LEFT JOIN recommendation_settings rs ON rs.workspace_id = l.workspace_id
		WHERE m.contact_id = $1 AND l.workspace_id = $2 AND m.status IN ('subscribed', 'pending')
		ORDER BY (l.id = rs.profile_list_id) DESC NULLS LAST, l.created_at LIMIT 1`, contactID, ws.ID).Scan(&name)
	return name
}

// ConfirmSection is "While you're here" on the opt-in confirmation page:
// up to 3 recommendations, counted as shown. Nil when it's off or nothing
// can show.
func (s *Service) ConfirmSection(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, count bool) (*PublicRecs, error) {
	set, err := s.settings(ctx, ws)
	if err != nil || !set.ConfirmPage {
		return nil, err
	}
	recs, err := s.showing(ctx, ws, true)
	if err != nil || len(recs) == 0 {
		return nil, err
	}
	if len(recs) > ConfirmPageMax {
		recs = recs[:ConfirmPageMax]
	}
	pages, err := s.pages(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := &PublicRecs{Publisher: s.publisher(ctx, ws), ListName: s.readerList(ctx, ws, contactID), Token: s.PageToken(ws.ID, contactID, s.now())}
	_ = s.Pool.QueryRow(ctx, `SELECT email FROM contacts WHERE id = $1 AND workspace_id = $2`, contactID, ws.ID).Scan(&out.Email)
	day := dayOf(s.now(), ws)
	for _, r := range recs {
		out.Items = append(out.Items, s.publicRec(ctx, ws, r, pages.RecommendsVisit))
		if count {
			if err := s.Q.RecDailyBump(ctx, store.RecDailyBumpParams{RecommendationID: r.ID, Day: day, ShownConfirm: 1}); err != nil {
				s.log().WarnContext(ctx, "recommendations: count shown", "err", err)
			}
		}
	}
	return out, nil
}

// Page is the hosted recommendations page: the recommendations the block
// shows, for a reader (contactID) or for an address they type (uuid.Nil).
func (s *Service) Page(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) (*PublicRecs, error) {
	recs, err := s.showing(ctx, ws, false)
	if err != nil {
		return nil, err
	}
	pages, err := s.pages(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := &PublicRecs{Publisher: s.publisher(ctx, ws), Items: []PublicRec{}}
	if contactID != uuid.Nil {
		out.Token = s.PageToken(ws.ID, contactID, s.now())
		out.ListName = s.readerList(ctx, ws, contactID)
		_ = s.Pool.QueryRow(ctx, `SELECT email FROM contacts WHERE id = $1 AND workspace_id = $2 AND erased_at IS NULL`, contactID, ws.ID).Scan(&out.Email)
	}
	if out.ListName == "" {
		set, _ := s.settings(ctx, ws)
		if p, err := s.profile(ctx, ws, set); err == nil {
			out.ListName = p.ListName
		}
	}
	for _, r := range recs {
		out.Items = append(out.Items, s.publicRec(ctx, ws, r, pages.RecommendsVisit))
	}
	return out, nil
}

// Visit counts a click on a link recommendation and returns where it goes.
func (s *Service) Visit(ctx context.Context, ws *store.Workspace, id uuid.UUID) (string, bool) {
	r, err := s.Q.RecGet(ctx, store.RecGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil || r.Kind != KindLink || r.Url == "" {
		return "", false
	}
	if err := s.Q.RecDailyBump(ctx, store.RecDailyBumpParams{RecommendationID: r.ID, Day: dayOf(s.now(), ws), Clicks: 1}); err != nil {
		s.log().WarnContext(ctx, "recommendations: count a click", "err", err)
	}
	return r.Url, true
}

// Subscribe is a reader's Subscribe on a public page: for the reader of a
// token, or an address they typed when their link expired.
func (s *Service) Subscribe(ctx context.Context, ws *store.Workspace, recID uuid.UUID, contactID uuid.UUID, email, origin string) (string, error) {
	if contactID != uuid.Nil {
		if err := s.Pool.QueryRow(ctx, `SELECT email FROM contacts WHERE id = $1 AND workspace_id = $2 AND erased_at IS NULL`, contactID, ws.ID).Scan(&email); err != nil {
			return "", ErrExpired
		}
	}
	return s.RequestSubscribe(ctx, ws, recID, email, origin)
}
