package leads

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// LeadAdsConnectionList is the workspace's Meta connections, the live one
// first.
type LeadAdsConnectionList struct {
	Items []LeadAdsConnectionItem `json:"items"`
}

// LeadAdsConnectionItem is one connection in the list.
type LeadAdsConnectionItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name" example:"Acme Marketing"`
	Status    string    `json:"status" enum:"connected,attention,disconnected"`
	CreatedAt time.Time `json:"createdAt"`
}

// LeadAdsConnection is Settings › Integrations › Meta Lead Ads.
type LeadAdsConnection struct {
	ID       string `json:"id"`
	Name     string `json:"name" example:"Acme Marketing"`
	Platform string `json:"platform" enum:"meta"`
	// AppID is the Meta app's id (not a secret).
	AppID  string `json:"appId" example:"481516234200"`
	Status string `json:"status" enum:"connected,attention,disconnected"`
	// Problem says why it needs attention: Meta refused the token (or it
	// expired), the app id and secret, or a permission.
	Problem string `json:"problem,omitempty" enum:"token_rejected,token_expired,app_rejected,permission_missing,"`
	// Receiving says how leads arrive: Meta's webhook, or polling every 15
	// minutes and why.
	Receiving LeadAdsReceiving `json:"receiving"`
	// RateLimitedUntil: Meta asked to slow down; calls resume then.
	RateLimitedUntil *time.Time    `json:"rateLimitedUntil,omitempty"`
	CheckedAt        *time.Time    `json:"checkedAt,omitempty" doc:"When the token was last checked"`
	CreatedAt        time.Time     `json:"createdAt"`
	CreatedBy        string        `json:"createdBy"`
	DisconnectedAt   *time.Time    `json:"disconnectedAt,omitempty"`
	Pages            []LeadAdsPage `json:"pages" doc:"The chosen Pages and their lead forms"`
	Leads30d         int64         `json:"leads30d" doc:"Leads in the last 30 days, Meta's test leads left out"`
	// Webhook is the callback URL and verify token, for setting the
	// subscription by hand in Meta's app dashboard: only for people who
	// can manage integrations.
	Webhook *LeadAdsWebhook `json:"webhook,omitempty"`
	// CanManage: the caller holds integrations:manage; Helpers are who
	// hold it (named for everyone, up to two).
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
}

// LeadAdsReceiving is how leads arrive.
type LeadAdsReceiving struct {
	Mode string `json:"mode" enum:"webhook,polling"`
	// Reason is why leads are polled: Meta refused the app's webhook
	// subscription, no delivery for 24 hours while forms are active, or
	// the instance isn't reachable from the internet.
	Reason       string     `json:"reason,omitempty" enum:"not_subscribed,quiet,unreachable,"`
	URL          string     `json:"url" doc:"The instance's public URL"`
	LastEventAt  *time.Time `json:"lastEventAt,omitempty" doc:"Meta's last delivery"`
	LastLeadAt   *time.Time `json:"lastLeadAt,omitempty"`
	LastPolledAt *time.Time `json:"lastPolledAt,omitempty"`
	SubscribedAt *time.Time `json:"subscribedAt,omitempty" doc:"When Meta accepted the app's webhook subscription"`
}

// LeadAdsWebhook is the callback Meta posts leads to.
type LeadAdsWebhook struct {
	CallbackURL string `json:"callbackUrl"`
	VerifyToken string `json:"verifyToken"`
}

// LeadAdsPage is a chosen Facebook Page.
type LeadAdsPage struct {
	ID           string        `json:"id"`
	PageID       string        `json:"pageId" doc:"Meta's id of the Page"`
	Name         string        `json:"name" example:"Acme Inc"`
	SubscribedAt *time.Time    `json:"subscribedAt,omitempty"`
	Forms        []LeadAdsForm `json:"forms"`
}

// LeadAdsForm is a lead form and how it's collected.
type LeadAdsForm struct {
	ID     string `json:"id"`
	FormID string `json:"formId" doc:"Meta's id of the form"`
	Name   string `json:"name" example:"Spring webinar signup"`
	PageID string `json:"pageId"`
	Page   string `json:"page" example:"Acme Inc"`
	// MetaStatus is the form's status in Meta: ACTIVE, ARCHIVED, DELETED
	// or DRAFT.
	MetaStatus string `json:"metaStatus"`
	// Status: active or paused; Mapped: it has been mapped, so it collects.
	Status      string           `json:"status" enum:"active,paused"`
	Mapped      bool             `json:"mapped"`
	Leads30d    int64            `json:"leads30d"`
	Lists       []LeadAdsListRef `json:"lists"`
	Tags        []string         `json:"tags"`
	ConsentMode string           `json:"consentMode" enum:"double,single"`
	Questions   int              `json:"questions"`
	// Waiting counts leads that arrived while the form wasn't mapped or
	// was paused (not stored: Import past leads brings them in).
	Waiting    int32         `json:"waiting"`
	Import     LeadAdsImport `json:"import"`
	LastLeadAt *time.Time    `json:"lastLeadAt,omitempty"`
}

// LeadAdsImport is a form's Import past leads.
type LeadAdsImport struct {
	State string     `json:"state" enum:"running,done,failed,"`
	Count int32      `json:"count" doc:"Leads read so far, or in the finished import"`
	At    *time.Time `json:"at,omitempty" doc:"When the last import finished"`
}

// LeadAdsListRef is a list a form's leads join.
type LeadAdsListRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LeadAdsActivity is a connection's recent leads.
type LeadAdsActivity struct {
	Items []LeadAdsActivityItem `json:"items"`
}

// LeadAdsActivityItem is one lead: when, which form, what happened and the
// contact. Answers aren't shown.
type LeadAdsActivityItem struct {
	ID       string    `json:"id"`
	At       time.Time `json:"at" doc:"When ReputeCast recorded it"`
	LeadAt   time.Time `json:"leadAt" doc:"When the person sent the form"`
	FormID   string    `json:"formId"`
	FormName string    `json:"formName"`
	Page     string    `json:"page"`
	Outcome  string    `json:"outcome" enum:"created,updated,pending,blocked,invalid,declined,test"`
	Source   string    `json:"source" enum:"webhook,poll,import"`
	// Contact is set for people who may see contacts; Hidden says there
	// is one they may not see ("a contact").
	Contact *LeadAdsContactRef `json:"contact,omitempty"`
	Hidden  bool               `json:"hidden"`
}

// LeadAdsContactRef names a lead's contact.
type LeadAdsContactRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Erased bool   `json:"erased"`
}

var errNoCredentials = errors.New("leads: the connection has no credentials")

// List is the workspace's connections.
func (s *Service) List(ctx context.Context, ws *store.Workspace) (LeadAdsConnectionList, error) {
	rows, err := s.Q.LeadConnectionsOfWorkspace(ctx, ws.ID)
	if err != nil {
		return LeadAdsConnectionList{}, err
	}
	out := LeadAdsConnectionList{Items: make([]LeadAdsConnectionItem, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, LeadAdsConnectionItem{ID: r.ID.String(), Name: r.Name, Status: r.Status, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

// connection loads one of the workspace's connections (404 otherwise).
func (s *Service) connection(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID) (store.LeadConnection, error) {
	c, err := q.LeadConnectionByID(ctx, store.LeadConnectionByIDParams{WorkspaceID: ws.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return c, api.Err(404, api.CodeNotFound, mConnectionNotFound.Text(i18n.LangFrom(ctx), nil))
	}
	return c, err
}

// live is a connection that isn't disconnected (409 otherwise).
func (s *Service) live(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID) (store.LeadConnection, error) {
	c, err := s.connection(ctx, q, ws, id)
	if err == nil && c.Status == StatusDisconnected {
		return c, api.Conflict(mConnectionGone.Text(i18n.LangFrom(ctx), i18n.Args{"name": c.Name}))
	}
	return c, err
}

// Get is a connection's page.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (LeadAdsConnection, error) {
	c, err := s.connection(ctx, s.Q, ws, id)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	return s.view(ctx, ws, p, c)
}

// view assembles a connection's page.
func (s *Service) view(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.LeadConnection) (LeadAdsConnection, error) {
	now := s.now()
	url, _ := s.reachable(ctx)
	out := LeadAdsConnection{ID: c.ID.String(), Name: c.Name, Platform: c.Platform, AppID: c.AppID, Status: c.Status, CheckedAt: c.CheckedAt,
		CreatedAt: c.CreatedAt, DisconnectedAt: c.DisconnectedAt, Pages: []LeadAdsPage{}, Helpers: []string{},
		Receiving: LeadAdsReceiving{Mode: "webhook", URL: url, LastEventAt: c.LastEventAt, SubscribedAt: c.SubscribedAt}}
	if c.Status == StatusAttention {
		out.Problem = c.StatusDetail
	}
	if c.PollingReason != "" && c.Status != StatusDisconnected {
		out.Receiving.Mode, out.Receiving.Reason = "polling", c.PollingReason
	}
	if c.RateLimitedUntil != nil && c.RateLimitedUntil.After(now) {
		out.RateLimitedUntil = c.RateLimitedUntil
	}
	if name, err := s.Q.LeadConnectionCreatorName(ctx, c.ID); err == nil {
		out.CreatedBy = name
	}
	// Helpers are named for everyone: "Who can see it" says who maps forms
	// and connects Meta, and a read-only member asks one of them.
	out.CanManage = p != nil && p.Can(authz.IntegrationsManage)
	out.Helpers = nonNil(access.Helpers(ctx, s.Q, ws.ID, authz.IntegrationsManage))
	if out.CanManage && c.Status != StatusDisconnected && len(c.CallbackTokenSealed) > 0 {
		cb, err := s.callbackURL(c)
		if err == nil {
			vt, err := s.Box.OpenString(c.VerifyTokenSealed, verifyAD(c.ID))
			if err == nil {
				out.Webhook = &LeadAdsWebhook{CallbackURL: cb, VerifyToken: vt}
			}
		}
	}
	lists, err := s.listNames(ctx, ws)
	if err != nil {
		return out, err
	}
	forms, err := s.Q.LeadFormsOfConnection(ctx, store.LeadFormsOfConnectionParams{ConnectionID: c.ID, Since: now.Add(-30 * 24 * time.Hour)})
	if err != nil {
		return out, err
	}
	pages, err := s.Q.LeadPagesOfConnection(ctx, c.ID)
	if err != nil {
		return out, err
	}
	byPage := map[uuid.UUID]int{}
	for _, pg := range pages {
		if !pg.Chosen {
			continue
		}
		byPage[pg.ID] = len(out.Pages)
		out.Pages = append(out.Pages, LeadAdsPage{ID: pg.ID.String(), PageID: pg.PageID, Name: pg.Name, SubscribedAt: pg.SubscribedAt, Forms: []LeadAdsForm{}})
	}
	for _, f := range forms {
		i, ok := byPage[f.PageID]
		if !ok || f.MetaStatus == "DELETED" {
			continue
		}
		out.Leads30d += f.Leads30d
		if f.LastLeadAt != nil && (out.Receiving.LastLeadAt == nil || f.LastLeadAt.After(*out.Receiving.LastLeadAt)) {
			out.Receiving.LastLeadAt = f.LastLeadAt
		}
		if f.LastPolledAt != nil && (out.Receiving.LastPolledAt == nil || f.LastPolledAt.After(*out.Receiving.LastPolledAt)) {
			out.Receiving.LastPolledAt = f.LastPolledAt
		}
		out.Pages[i].Forms = append(out.Pages[i].Forms, formView(formRow(f), f.PageName, f.Leads30d, lists))
	}
	return out, nil
}

// formRow is the columns every form query shares.
type formFields struct {
	ID, PageID                                    uuid.UUID
	FormID, Name, MetaStatus, Status, ConsentMode string
	MappedAt, ImportedAt, LastLeadAt              *time.Time
	Questions                                     json.RawMessage
	ListIDs                                       []uuid.UUID
	Tags                                          []string
	Waiting, ImportCount                          int32
	ImportState                                   string
}

func formRow(f store.LeadFormsOfConnectionRow) formFields {
	return formFields{ID: f.ID, PageID: f.PageID, FormID: f.FormID, Name: f.Name, MetaStatus: f.MetaStatus, Status: f.Status, ConsentMode: f.ConsentMode,
		MappedAt: f.MappedAt, ImportedAt: f.ImportedAt, LastLeadAt: f.LastLeadAt, Questions: f.Questions, ListIDs: f.ListIds, Tags: f.Tags,
		Waiting: f.Waiting, ImportCount: f.ImportCount, ImportState: f.ImportState}
}

func formByIDRow(f store.LeadFormByIDRow) formFields {
	return formFields{ID: f.ID, PageID: f.PageID, FormID: f.FormID, Name: f.Name, MetaStatus: f.MetaStatus, Status: f.Status, ConsentMode: f.ConsentMode,
		MappedAt: f.MappedAt, ImportedAt: f.ImportedAt, LastLeadAt: f.LastLeadAt, Questions: f.Questions, ListIDs: f.ListIds, Tags: f.Tags,
		Waiting: f.Waiting, ImportCount: f.ImportCount, ImportState: f.ImportState}
}

func formView(f formFields, page string, leads30d int64, lists map[uuid.UUID]string) LeadAdsForm {
	v := LeadAdsForm{ID: f.ID.String(), FormID: f.FormID, Name: f.Name, PageID: f.PageID.String(), Page: page, MetaStatus: f.MetaStatus, Status: f.Status,
		Mapped: f.MappedAt != nil, Leads30d: leads30d, Lists: []LeadAdsListRef{}, Tags: nonNil(f.Tags), ConsentMode: f.ConsentMode, Waiting: f.Waiting,
		Import: LeadAdsImport{State: f.ImportState, Count: f.ImportCount, At: f.ImportedAt}, LastLeadAt: f.LastLeadAt}
	var qs []json.RawMessage
	_ = json.Unmarshal(f.Questions, &qs)
	v.Questions = len(qs)
	for _, id := range f.ListIDs {
		if n, ok := lists[id]; ok {
			v.Lists = append(v.Lists, LeadAdsListRef{ID: id.String(), Name: n})
		}
	}
	return v
}

// listNames are the workspace's lists by id.
func (s *Service) listNames(ctx context.Context, ws *store.Workspace) (map[uuid.UUID]string, error) {
	rows, err := s.Q.LeadListsOfWorkspace(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]string, len(rows))
	for _, r := range rows {
		out[r.ID] = r.Name
	}
	return out, nil
}

// Activity is a connection's last 50 leads. Contacts are named only for
// people who may see them.
func (s *Service) Activity(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (LeadAdsActivity, error) {
	c, err := s.connection(ctx, s.Q, ws, id)
	if err != nil {
		return LeadAdsActivity{}, err
	}
	rows, err := s.Q.LeadEntriesRecent(ctx, store.LeadEntriesRecentParams{ConnectionID: c.ID, Lim: 50})
	if err != nil {
		return LeadAdsActivity{}, err
	}
	see := p != nil && p.Can(authz.SubscribersView)
	out := LeadAdsActivity{Items: make([]LeadAdsActivityItem, 0, len(rows))}
	for _, r := range rows {
		it := LeadAdsActivityItem{ID: r.ID.String(), At: r.CreatedAt, LeadAt: r.LeadAt, FormID: r.FormID.String(), FormName: r.FormName, Page: r.PageName,
			Outcome: r.Outcome, Source: r.Source}
		if r.ContactID != nil {
			if see {
				it.Contact = &LeadAdsContactRef{ID: r.ContactID.String(), Name: r.ContactName, Email: r.ContactEmail, Erased: r.ContactErased}
				if r.ContactErased {
					it.Contact.Name, it.Contact.Email = "", ""
				}
			} else {
				it.Hidden = true
			}
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}

func nonNil[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}

// sortedKeys returns a map's keys in order.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
