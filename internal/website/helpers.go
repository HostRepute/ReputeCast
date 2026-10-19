package website

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

func (s *Service) approvals() Approvals {
	if s.Approvals == nil {
		return nil
	}
	return s.Approvals()
}

func (s *Service) jobs() Inserter {
	if s.Jobs == nil {
		return nil
	}
	return s.Jobs()
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func lang(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }

// tr is a message of this package in the request's language.
func tr(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(lang(ctx), args)
}

func userID(p *authz.Principal) *uuid.UUID {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// monthOf is the first day of the calendar month (the meter's month).
func monthOf(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// lock reads the settings for an update in a transaction.
func (s *Service) lock(ctx context.Context, q *store.Queries, ws *store.Workspace) (*config, error) {
	if err := q.WebsiteSettingsEnsure(ctx, ws.ID); err != nil {
		return nil, err
	}
	row, err := q.WebsiteSettingsLock(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	return decodeConfig(row), nil
}

// save writes every editable setting of c.
func (s *Service) save(ctx context.Context, q *store.Queries, ws *store.Workspace, c *config, by *uuid.UUID) error {
	sections, _ := json.Marshal(c.Sections)
	header, _ := json.Marshal(c.Header)
	footer, _ := json.Marshal(c.Footer)
	lists, _ := json.Marshal(c.SignupLists)
	paywall, _ := json.Marshal(c.Paywall)
	r := c.row
	return q.WebsiteSettingsSave(ctx, store.WebsiteSettingsSaveParams{WorkspaceID: ws.ID,
		Title: r.Title, Tagline: r.Tagline, Description: r.Description, SocialImage: r.SocialImage, IndexSite: r.IndexSite,
		HomeLayout: r.HomeLayout, Featured: r.Featured, Sections: sections, AboutTitle: r.AboutTitle, AboutText: r.AboutText, AboutLink: r.AboutLink,
		HeaderLinks: header, Footer: footer,
		SigninOn: r.SigninOn, LinkMinutes: r.LinkMinutes, SessionDays: r.SessionDays, SignupOn: r.SignupOn, SignupLists: lists,
		DefaultVisibility: r.DefaultVisibility, CampaignVisibility: r.CampaignVisibility,
		MeterVisitorsOn: r.MeterVisitorsOn, MeterVisitors: r.MeterVisitors, MeterMembersOn: r.MeterMembersOn, MeterMembers: r.MeterMembers,
		Paywall: paywall, At: s.now(), UpdatedBy: by})
}
