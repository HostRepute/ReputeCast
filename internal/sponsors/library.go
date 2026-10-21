package sponsors

import (
	"context"
	"net/mail"
	"net/url"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// SponsorInput creates or changes a sponsor of the library.
type SponsorInput struct {
	Name         string `json:"name" minLength:"1" maxLength:"80"`
	Website      string `json:"website,omitempty" maxLength:"300"`
	LogoURL      string `json:"logoUrl,omitempty" maxLength:"1000"`
	ContactName  string `json:"contactName,omitempty" maxLength:"120"`
	ContactEmail string `json:"contactEmail,omitempty" maxLength:"254"`
	TermsDays    *int   `json:"termsDays,omitempty" minimum:"0" maximum:"120" doc:"Payments are due this many days after the issue (14 when absent)."`
	Notes        string `json:"notes,omitempty" maxLength:"2000"`
}

// initials are a name's first letters ("Northwind Analytics" → NA).
func initials(name string) string {
	var out []rune
	for _, w := range strings.Fields(name) {
		for _, r := range w {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				out = append(out, unicode.ToUpper(r))
				break
			}
		}
		if len(out) == 2 {
			break
		}
	}
	if len(out) == 1 {
		rs := []rune(strings.TrimSpace(name))
		for _, r := range rs[1:] {
			if unicode.IsLetter(r) {
				out = append(out, unicode.ToUpper(r))
				break
			}
		}
	}
	return string(out)
}

func refOf(id uuid.UUID, name, logo string) *SponsorRef {
	return &SponsorRef{ID: id.String(), Name: name, Initials: initials(name), LogoURL: logo}
}

// cleanSponsor checks a sponsor's fields: a name, a web address, an email.
func cleanSponsor(ctx context.Context, in SponsorInput) (SponsorInput, int32, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Website = strings.TrimSpace(in.Website)
	in.ContactName = strings.TrimSpace(in.ContactName)
	in.ContactEmail = strings.TrimSpace(strings.ToLower(in.ContactEmail))
	in.Notes = strings.TrimSpace(in.Notes)
	if in.Name == "" {
		return in, 0, invalid(ctx, "body.name", "problem.nameRequired", nil)
	}
	if in.Website != "" {
		w := in.Website
		if !strings.Contains(w, "://") {
			w = "https://" + w
		}
		u, err := url.Parse(w)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return in, 0, invalid(ctx, "body.website", "problem.website", nil)
		}
		in.Website = strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(w, "https://"), "http://"), "/")
	}
	if in.ContactEmail != "" {
		if a, err := mail.ParseAddress(in.ContactEmail); err != nil || a.Address != in.ContactEmail {
			return in, 0, invalid(ctx, "body.contactEmail", "problem.email", nil)
		}
	}
	terms := int32(14)
	if in.TermsDays != nil {
		terms = int32(*in.TermsDays)
	}
	return in, terms, nil
}

// CreateSponsor adds a sponsor to the library.
func (s *Service) CreateSponsor(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SponsorInput) (SponsorItem, error) {
	in, terms, err := cleanSponsor(ctx, in)
	if err != nil {
		return SponsorItem{}, err
	}
	if taken, _ := s.Q.SponsorNameTaken(ctx, store.SponsorNameTakenParams{WorkspaceID: ws.ID, Name: in.Name, ID: uuid.Nil}); taken {
		return SponsorItem{}, problem(ctx, 409, api.CodeConflict, "problem.nameTaken", map[string]any{"name": in.Name})
	}
	id := newID()
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorInsert(ctx, store.SponsorInsertParams{ID: id, WorkspaceID: ws.ID, Name: in.Name, Website: in.Website, LogoUrl: in.LogoURL,
			ContactName: in.ContactName, ContactEmail: in.ContactEmail, TermsDays: terms, Notes: in.Notes, CreatedBy: userID(p), At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.sponsor_created", TargetType: "sponsor", TargetID: id.String(),
			TargetName: in.Name, After: map[string]any{"website": in.Website, "contact": in.ContactName, "termsDays": terms}})
	})
	if err != nil {
		return SponsorItem{}, err
	}
	return s.sponsorItem(ctx, ws, id)
}

// UpdateSponsor changes a sponsor.
func (s *Service) UpdateSponsor(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SponsorInput) (SponsorItem, error) {
	old, err := s.Q.SponsorGet(ctx, store.SponsorGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return SponsorItem{}, api.MapError(err, "Sponsor")
	}
	in, terms, err := cleanSponsor(ctx, in)
	if err != nil {
		return SponsorItem{}, err
	}
	if taken, _ := s.Q.SponsorNameTaken(ctx, store.SponsorNameTakenParams{WorkspaceID: ws.ID, Name: in.Name, ID: id}); taken {
		return SponsorItem{}, problem(ctx, 409, api.CodeConflict, "problem.nameTaken", map[string]any{"name": in.Name})
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorUpdate(ctx, store.SponsorUpdateParams{ID: id, WorkspaceID: ws.ID, Name: in.Name, Website: in.Website, LogoUrl: in.LogoURL,
			ContactName: in.ContactName, ContactEmail: in.ContactEmail, TermsDays: terms, Notes: in.Notes, At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.sponsor_updated", TargetType: "sponsor", TargetID: id.String(),
			TargetName: in.Name,
			Before:     map[string]any{"name": old.Name, "website": old.Website, "contact": old.ContactName, "email": old.ContactEmail, "termsDays": old.TermsDays},
			After:      map[string]any{"name": in.Name, "website": in.Website, "contact": in.ContactName, "email": in.ContactEmail, "termsDays": terms}})
	})
	if err != nil {
		return SponsorItem{}, err
	}
	return s.sponsorItem(ctx, ws, id)
}

// DeleteSponsor removes a sponsor without upcoming bookings; past bookings
// keep its name.
func (s *Service) DeleteSponsor(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	old, err := s.Q.SponsorGet(ctx, store.SponsorGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "Sponsor")
	}
	var upcoming int
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM sponsor_bookings WHERE sponsor_id = $1 AND status IN ('hold','booked') AND issue_on >= $2`, id, s.today(ws)).Scan(&upcoming)
	if upcoming > 0 {
		return problem(ctx, 409, api.CodeConflict, "problem.sponsorHasBookings", map[string]any{"name": old.Name, "count": upcoming})
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorDelete(ctx, store.SponsorDeleteParams{ID: id, WorkspaceID: ws.ID, At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.sponsor_deleted", TargetType: "sponsor", TargetID: id.String(),
			TargetName: old.Name, Before: map[string]any{"website": old.Website, "contact": old.ContactName}})
	})
}

func (s *Service) sponsorItem(ctx context.Context, ws *store.Workspace, id uuid.UUID) (SponsorItem, error) {
	items, err := s.library(ctx, ws)
	if err != nil {
		return SponsorItem{}, err
	}
	for _, it := range items {
		if it.ID == id.String() {
			return it, nil
		}
	}
	return SponsorItem{}, api.NotFound("Sponsor")
}

// library lists the sponsors with their bookings, amounts and latest
// click rate.
func (s *Service) library(ctx context.Context, ws *store.Workspace) ([]SponsorItem, error) {
	rows, err := s.Q.SponsorList(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	bookings, err := s.Q.SponsorBookingsOfWorkspace(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	stats := map[uuid.UUID]store.SponsorReportStat{}
	ids := make([]uuid.UUID, 0, len(bookings))
	for _, b := range bookings {
		ids = append(ids, b.ID)
	}
	if len(ids) > 0 {
		st, err := s.Q.SponsorReportStatsOf(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, r := range st {
			stats[r.BookingID] = r
		}
	}
	out := make([]SponsorItem, 0, len(rows))
	for _, r := range rows {
		it := SponsorItem{ID: r.ID.String(), Name: r.Name, Initials: initials(r.Name), Website: r.Website, LogoURL: r.LogoUrl, ContactName: r.ContactName,
			ContactEmail: r.ContactEmail, TermsDays: int(r.TermsDays), Notes: r.Notes, Issues: []int{}, Hold: []int{}, Currency: "usd"}
		latest := -1
		for _, b := range bookings {
			if b.SponsorID == nil || *b.SponsorID != r.ID {
				continue
			}
			it.Currency = b.Currency
			switch b.Status {
			case "hold":
				it.Hold = append(it.Hold, int(b.IssueNumber))
				it.HoldCents += b.PriceCents
			case "booked", "sent":
				it.Issues = append(it.Issues, int(b.IssueNumber))
				it.BookedCents += b.PriceCents
			default:
				continue
			}
			if b.PaymentStatus == "paid" && b.PaidCents != nil {
				it.PaidCents += *b.PaidCents
			}
			if st, ok := stats[b.ID]; ok && st.Final && st.Delivered > 0 && int(b.IssueNumber) > latest {
				latest = int(b.IssueNumber)
				rate := float64(st.UniqueClickers) / float64(st.Delivered) * 100
				it.ClickRate, it.ClickRateIssue = &rate, latest
			}
		}
		out = append(out, it)
	}
	return out, nil
}
