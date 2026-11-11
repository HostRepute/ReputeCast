package leads

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/leads/meta"
	"github.com/hostrepute/reputecast/internal/store"
)

// LeadAdsSetup is what the connect dialog's first step shows: the
// permissions the token needs and whether Meta can reach this server.
type LeadAdsSetup struct {
	Permissions []string `json:"permissions"`
	URL         string   `json:"url" doc:"The instance's public URL"`
	Reachable   bool     `json:"reachable" doc:"Meta can reach it, so leads arrive within seconds; otherwise they're polled every 15 minutes"`
	// Connection is the workspace's live connection, when there is one:
	// one Meta connection per workspace.
	Connection *LeadAdsConnectionItem `json:"connection,omitempty"`
}

// Setup is the connect dialog's first step.
func (s *Service) Setup(ctx context.Context, ws *store.Workspace) (LeadAdsSetup, error) {
	url, ok := s.reachable(ctx)
	out := LeadAdsSetup{Permissions: slices.Clone(meta.Permissions), URL: url, Reachable: ok}
	c, err := s.Q.LeadConnectionLive(ctx, ws.ID)
	switch {
	case err == nil:
		out.Connection = &LeadAdsConnectionItem{ID: c.ID.String(), Name: c.Name, Status: c.Status, CreatedAt: c.CreatedAt}
	case !errors.Is(err, pgx.ErrNoRows):
		return out, err
	}
	return out, nil
}

// LeadAdsConnectInput is the connect dialog's second step.
type LeadAdsConnectInput struct {
	Name      string `json:"name" minLength:"1" maxLength:"80" example:"Acme Marketing"`
	AppID     string `json:"appId" minLength:"1" maxLength:"32" pattern:"^[0-9]+$" doc:"The Meta app's id, from App settings › Basic"`
	AppSecret string `json:"appSecret" minLength:"16" maxLength:"64" doc:"The app secret, from App settings › Basic (write-only)"`
	Token     string `json:"token" minLength:"20" maxLength:"1024" doc:"A system user's token for the app, with no expiry (write-only)"`
}

// metaProblem turns a Graph API refusal into a 422 on the field it's
// about, or a 502/503 when Meta didn't answer.
func metaProblem(ctx context.Context, err error, field string) error {
	l := i18n.LangFrom(ctx)
	switch {
	case meta.RateLimited(err):
		return api.Err(http.StatusServiceUnavailable, "meta_rate_limited", mMetaRateLimited.Text(l, nil))
	case meta.Expired(err):
		return api.Invalid("body.token", mTokenExpired.Text(l, nil))
	case field == "body.appSecret" && (meta.TokenRejected(err) || isAppError(err)):
		return api.Invalid(field, mAppRejected.Text(l, nil))
	case meta.TokenRejected(err):
		return api.Invalid("body.token", mTokenRejected.Text(l, nil))
	case meta.PermissionMissing(err):
		return api.Invalid("body.token", mTokenPermissions.Text(l, i18n.Args{"permissions": strings.Join(meta.Permissions, ", ")}))
	}
	if _, ok := meta.AsError(err); ok {
		return api.Err(http.StatusBadGateway, "meta_unavailable", mMetaUnavailable.Text(l, nil))
	}
	return err
}

// isAppError is Meta's answer to an app id and secret that don't belong
// together (101: invalid client id, 1: an unknown app).
func isAppError(err error) bool {
	e, ok := meta.AsError(err)
	return ok && (e.Code == 101 || e.Code == 1 || (e.Code == 100 && e.Subcode == 33))
}

// problemOf is the attention a Graph API refusal calls for ("" when it
// isn't the credentials' fault).
func problemOf(err error) string {
	switch {
	case meta.Expired(err):
		return ProblemTokenExpired
	case meta.TokenRejected(err):
		return ProblemTokenRejected
	case meta.PermissionMissing(err):
		return ProblemPermissionMissing
	}
	return ""
}

// checkCredentials checks the app id and secret together, then the token
// by listing its Pages.
func (s *Service) checkCredentials(ctx context.Context, creds meta.Credentials, demo bool) ([]meta.Page, error) {
	cl := s.clientFor(creds, demo)
	if _, err := cl.App(ctx); err != nil {
		return nil, metaProblem(ctx, err, "body.appSecret")
	}
	pages, err := cl.Pages(ctx)
	if err != nil {
		return nil, metaProblem(ctx, err, "body.token")
	}
	if len(pages) == 0 {
		return nil, api.Invalid("body.token", mNoPages.Text(i18n.LangFrom(ctx), nil))
	}
	return pages, nil
}

// Connect checks the credentials with Meta, keeps them sealed and
// subscribes the app to the Pages' leads at the connection's callback URL.
// Meta checks that address at once, so the connection is stored first; a
// subscription Meta refuses leaves the connection polling.
func (s *Service) Connect(ctx context.Context, ws *store.Workspace, p *authz.Principal, in LeadAdsConnectInput) (LeadAdsConnection, error) {
	l := i18n.LangFrom(ctx)
	if c, err := s.Q.LeadConnectionLive(ctx, ws.ID); err == nil {
		return LeadAdsConnection{}, api.Conflict(mConnectionExists.Text(l, i18n.Args{"name": c.Name})).WithMeta("connectionId", c.ID.String())
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return LeadAdsConnection{}, err
	}
	creds := meta.Credentials{AppID: strings.TrimSpace(in.AppID), AppSecret: strings.TrimSpace(in.AppSecret), Token: strings.TrimSpace(in.Token)}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return LeadAdsConnection{}, api.Invalid("body.name", mNameMissing.Text(l, nil))
	}
	pages, err := s.checkCredentials(ctx, creds, false)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	id := newID()
	raw, _ := json.Marshal(creds)
	sealed, err := s.Box.Seal(raw, credAD(id))
	if err != nil {
		return LeadAdsConnection{}, err
	}
	token, hash := authn.NewToken()
	cbSealed, err := s.Box.SealString(token, callbackAD(id))
	if err != nil {
		return LeadAdsConnection{}, err
	}
	vtSealed, err := s.Box.SealString(randomToken(), verifyAD(id))
	if err != nil {
		return LeadAdsConnection{}, err
	}
	var by *uuid.UUID
	if p != nil && p.UserID != uuid.Nil {
		by = new(p.UserID)
	}
	var c store.LeadConnection
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		c, err = q.LeadConnectionInsert(ctx, store.LeadConnectionInsertParams{ID: id, WorkspaceID: ws.ID, Name: name, AppID: creds.AppID,
			CredentialsSealed: sealed, CallbackTokenHash: hash, CallbackTokenSealed: cbSealed, VerifyTokenSealed: vtSealed, CreatedBy: by, At: s.now()})
		if isUnique(err) {
			return api.Conflict(mConnectionExists.Text(l, i18n.Args{"name": name}))
		}
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "lead_connection.created", TargetType: "lead_connection",
			TargetID: id.String(), TargetName: name, After: map[string]any{"name": name, "platform": Platform, "appId": creds.AppID, "pagesReached": len(pages)}})
	})
	if err != nil {
		return LeadAdsConnection{}, err
	}
	s.subscribeApp(ctx, c, s.clientFor(creds, false))
	c, err = s.Q.LeadConnectionAny(ctx, id)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	return s.view(ctx, ws, p, c)
}

// subscribeApp subscribes the app to Pages' leads at the connection's
// callback URL; Meta refusing it (its verification request failed, the
// app has no Webhooks product) leaves leads polled.
func (s *Service) subscribeApp(ctx context.Context, c store.LeadConnection, cl *meta.Client) {
	cb, err := s.callbackURL(c)
	if err != nil {
		return
	}
	vt, err := s.Box.OpenString(c.VerifyTokenSealed, verifyAD(c.ID))
	if err != nil {
		return
	}
	var at *time.Time
	if err := cl.SubscribeApp(ctx, cb, vt); err != nil {
		s.log().WarnContext(ctx, "lead ads: Meta refused the app's webhook subscription", "connection", c.ID, "err", err)
	} else {
		at = new(s.now())
	}
	if err := s.Q.LeadConnectionSubscribed(ctx, store.LeadConnectionSubscribedParams{ID: c.ID, At: at}); err != nil {
		s.log().WarnContext(ctx, "lead ads: keep the subscription", "connection", c.ID, "err", err)
	}
}

// LeadAdsPageChoices are the Pages the token reaches, for choosing.
type LeadAdsPageChoices struct {
	Items []LeadAdsPageChoice `json:"items"`
}

// LeadAdsPageChoice is one Page the token reaches.
type LeadAdsPageChoice struct {
	PageID string `json:"pageId"`
	Name   string `json:"name" example:"Acme Inc"`
	Chosen bool   `json:"chosen"`
	// Forms is the Page's lead forms (left out past the first 25 Pages).
	Forms *int `json:"forms,omitempty"`
}

// maxCountedPages bounds how many Pages' forms are counted for the
// choice.
const maxCountedPages = 25

// PageChoices lists the Pages the token reaches, the chosen ones marked.
func (s *Service) PageChoices(ctx context.Context, ws *store.Workspace, id uuid.UUID) (LeadAdsPageChoices, error) {
	c, err := s.live(ctx, s.Q, ws, id)
	if err != nil {
		return LeadAdsPageChoices{}, err
	}
	cl, err := s.client(c)
	if err != nil {
		return LeadAdsPageChoices{}, err
	}
	pages, err := cl.Pages(ctx)
	if err != nil {
		s.noteProblem(ctx, c, err)
		return LeadAdsPageChoices{}, metaProblem(ctx, err, "")
	}
	stored, err := s.Q.LeadPagesOfConnection(ctx, c.ID)
	if err != nil {
		return LeadAdsPageChoices{}, err
	}
	chosen := map[string]bool{}
	for _, sp := range stored {
		chosen[sp.PageID] = sp.Chosen
	}
	out := LeadAdsPageChoices{Items: make([]LeadAdsPageChoice, 0, len(pages))}
	for i, pg := range pages {
		ch := LeadAdsPageChoice{PageID: pg.ID, Name: pg.Name, Chosen: chosen[pg.ID]}
		if i < maxCountedPages {
			if forms, err := cl.Forms(ctx, pg); err == nil {
				ch.Forms = new(len(forms))
			}
		}
		out.Items = append(out.Items, ch)
	}
	return out, nil
}

// LeadAdsPagesInput chooses the Pages whose leads are collected.
type LeadAdsPagesInput struct {
	PageIDs []string `json:"pageIds" maxItems:"100" doc:"Meta's ids of the chosen Pages"`
}

// ChoosePages subscribes the app to the chosen Pages' leads and lists
// their forms; Pages left out are unsubscribed (their mappings stay).
func (s *Service) ChoosePages(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in LeadAdsPagesInput) (LeadAdsConnection, error) {
	l := i18n.LangFrom(ctx)
	c, err := s.live(ctx, s.Q, ws, id)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	cl, err := s.client(c)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	reached, err := cl.Pages(ctx)
	if err != nil {
		s.noteProblem(ctx, c, err)
		return LeadAdsConnection{}, metaProblem(ctx, err, "")
	}
	byID := map[string]meta.Page{}
	for _, pg := range reached {
		byID[pg.ID] = pg
	}
	want := map[string]bool{}
	for _, pid := range in.PageIDs {
		if _, ok := byID[pid]; !ok {
			return LeadAdsConnection{}, api.Invalid("body.pageIds", mPageUnknown.Text(l, nil))
		}
		want[pid] = true
	}
	stored, err := s.Q.LeadPagesOfConnection(ctx, c.ID)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	storedBy := map[string]store.LeadPage{}
	before := []string{}
	for _, sp := range stored {
		storedBy[sp.PageID] = sp
		if sp.Chosen {
			before = append(before, sp.Name)
		}
	}
	type change struct {
		row    uuid.UUID
		page   meta.Page
		chosen bool
		subAt  *time.Time
		sealed []byte
	}
	var changes []change
	for _, pid := range sortedKeys(want) {
		pg := byID[pid]
		row := newID()
		if sp, ok := storedBy[pid]; ok {
			row = sp.ID
		}
		if err := cl.Subscribe(ctx, pg); err != nil {
			s.noteProblem(ctx, c, err)
			if prob := metaProblem(ctx, err, ""); !errors.Is(prob, err) {
				return LeadAdsConnection{}, prob
			}
			return LeadAdsConnection{}, api.Err(http.StatusBadGateway, "meta_unavailable", mSubscribeFailed.Text(l, i18n.Args{"page": pg.Name}))
		}
		sealed, err := s.Box.SealString(pg.AccessToken, pageAD(row))
		if err != nil {
			return LeadAdsConnection{}, err
		}
		changes = append(changes, change{row: row, page: pg, chosen: true, subAt: new(s.now()), sealed: sealed})
	}
	for _, sp := range stored {
		if !sp.Chosen || want[sp.PageID] {
			continue
		}
		if pg, ok := byID[sp.PageID]; ok {
			if err := cl.Unsubscribe(ctx, pg); err != nil {
				s.log().WarnContext(ctx, "lead ads: unsubscribe a Page", "connection", c.ID, "page", sp.PageID, "err", err)
			}
		}
		changes = append(changes, change{row: sp.ID, page: meta.Page{ID: sp.PageID, Name: sp.Name}})
	}
	after := []string{}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := s.now()
		for _, ch := range changes {
			name := ch.page.Name
			if name == "" {
				name = storedBy[ch.page.ID].Name
			}
			if _, err := q.LeadPageUpsert(ctx, store.LeadPageUpsertParams{ID: ch.row, WorkspaceID: ws.ID, ConnectionID: c.ID, PageID: ch.page.ID, Name: name,
				PageTokenSealed: ch.sealed, Chosen: ch.chosen, SubscribedAt: ch.subAt, At: now}); err != nil {
				return err
			}
			if ch.chosen {
				after = append(after, name)
			}
		}
		slices.Sort(before)
		slices.Sort(after)
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "lead_connection.updated", TargetType: "lead_connection",
			TargetID: c.ID.String(), TargetName: c.Name, Before: map[string]any{"pages": before}, After: map[string]any{"pages": after}})
	})
	if err != nil {
		return LeadAdsConnection{}, err
	}
	if err := s.syncForms(ctx, c, cl); err != nil {
		s.log().WarnContext(ctx, "lead ads: list forms", "connection", c.ID, "err", err)
	}
	return s.Get(ctx, ws, p, c.ID)
}

// Refresh lists the chosen Pages' forms again (new forms, new questions).
func (s *Service) Refresh(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (LeadAdsConnection, error) {
	c, err := s.live(ctx, s.Q, ws, id)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	cl, err := s.client(c)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	if err := s.syncForms(ctx, c, cl); err != nil {
		s.noteProblem(ctx, c, err)
		return LeadAdsConnection{}, metaProblem(ctx, err, "")
	}
	return s.Get(ctx, ws, p, c.ID)
}

// syncForms stores the chosen Pages' lead forms as Meta lists them.
func (s *Service) syncForms(ctx context.Context, c store.LeadConnection, cl *meta.Client) error {
	pages, err := s.Q.LeadPagesOfConnection(ctx, c.ID)
	if err != nil {
		return err
	}
	for _, sp := range pages {
		if !sp.Chosen {
			continue
		}
		pg, err := s.pageOf(sp.ID, sp.PageID, sp.PageTokenSealed)
		if err != nil {
			continue
		}
		if err := s.syncPageForms(ctx, c.WorkspaceID, sp.ID, cl, pg); err != nil {
			return err
		}
	}
	return nil
}

// syncPageForms stores one Page's lead forms.
func (s *Service) syncPageForms(ctx context.Context, wsID, pageRow uuid.UUID, cl *meta.Client, pg meta.Page) error {
	forms, err := cl.Forms(ctx, pg)
	if err != nil {
		return err
	}
	now := s.now()
	for _, f := range forms {
		if _, err := s.Q.LeadFormUpsert(ctx, store.LeadFormUpsertParams{ID: newID(), WorkspaceID: wsID, PageID: pageRow, FormID: f.ID, Name: clip(f.Name, 200),
			MetaStatus: f.Status, Questions: mustJSON(nonNil(f.Questions)), Checkboxes: mustJSON(nonNil(f.Checkboxes)), Disclaimer: clip(f.Disclaimer, 4000), At: now}); err != nil {
			return err
		}
	}
	return nil
}

// LeadAdsCredentialsInput replaces a connection's name or credentials; a
// blank secret keeps the stored one.
type LeadAdsCredentialsInput struct {
	Name      string `json:"name,omitempty" maxLength:"80"`
	AppID     string `json:"appId,omitempty" maxLength:"32" pattern:"^[0-9]*$"`
	AppSecret string `json:"appSecret,omitempty" maxLength:"64"`
	Token     string `json:"token,omitempty" maxLength:"1024"`
}

// ReplaceCredentials checks and keeps new credentials, then subscribes the
// app and the chosen Pages again with them. A connection that needed
// attention is connected again.
func (s *Service) ReplaceCredentials(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in LeadAdsCredentialsInput) (LeadAdsConnection, error) {
	c, err := s.live(ctx, s.Q, ws, id)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	creds, err := s.credentials(c)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	changed := []string{}
	name := c.Name
	if v := strings.TrimSpace(in.Name); v != "" && v != c.Name {
		name = v
		changed = append(changed, "name")
	}
	appChanged := false
	if v := strings.TrimSpace(in.AppID); v != "" && v != creds.AppID {
		creds.AppID, appChanged = v, true
		changed = append(changed, "appId")
	}
	if v := strings.TrimSpace(in.AppSecret); v != "" {
		if len(v) < 16 {
			return LeadAdsConnection{}, api.Invalid("body.appSecret", mAppRejected.Text(i18n.LangFrom(ctx), nil))
		}
		creds.AppSecret = v
		changed = append(changed, "appSecret")
	}
	if v := strings.TrimSpace(in.Token); v != "" {
		creds.Token = v
		changed = append(changed, "token")
	}
	if appChanged && in.AppSecret == "" {
		return LeadAdsConnection{}, api.Invalid("body.appSecret", mAppSecretWithID.Text(i18n.LangFrom(ctx), nil))
	}
	pages, err := s.checkCredentials(ctx, creds, c.Demo)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	raw, _ := json.Marshal(creds)
	sealed, err := s.Box.Seal(raw, credAD(c.ID))
	if err != nil {
		return LeadAdsConnection{}, err
	}
	stored, err := s.Q.LeadPagesOfConnection(ctx, c.ID)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := s.now()
		if err := q.LeadConnectionSetCredentials(ctx, store.LeadConnectionSetCredentialsParams{ID: c.ID, Name: name, AppID: creds.AppID, CredentialsSealed: sealed, At: now}); err != nil {
			return err
		}
		// The chosen Pages' tokens come from the new token.
		for _, sp := range stored {
			if !sp.Chosen {
				continue
			}
			for _, pg := range pages {
				if pg.ID != sp.PageID {
					continue
				}
				ps, err := s.Box.SealString(pg.AccessToken, pageAD(sp.ID))
				if err != nil {
					return err
				}
				if _, err := q.LeadPageUpsert(ctx, store.LeadPageUpsertParams{ID: sp.ID, WorkspaceID: ws.ID, ConnectionID: c.ID, PageID: sp.PageID, Name: pg.Name,
					PageTokenSealed: ps, Chosen: true, SubscribedAt: sp.SubscribedAt, At: now}); err != nil {
					return err
				}
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "lead_connection.updated", TargetType: "lead_connection",
			TargetID: c.ID.String(), TargetName: name, Before: map[string]any{"name": c.Name, "appId": c.AppID}, After: map[string]any{"name": name, "appId": creds.AppID},
			Metadata: map[string]any{"replaced": changed}})
	})
	if err != nil {
		return LeadAdsConnection{}, err
	}
	c, err = s.Q.LeadConnectionAny(ctx, c.ID)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	cl := s.clientFor(creds, c.Demo)
	if appChanged || c.SubscribedAt == nil {
		s.subscribeApp(ctx, c, cl)
	}
	if err := s.resubscribePages(ctx, c, cl); err != nil {
		s.log().WarnContext(ctx, "lead ads: subscribe Pages again", "connection", c.ID, "err", err)
	}
	// Deliveries held while the token was refused can be applied now.
	s.requeueConnection(ctx, c.ID)
	return s.Get(ctx, ws, p, c.ID)
}

// resubscribePages installs the app on every chosen Page again.
func (s *Service) resubscribePages(ctx context.Context, c store.LeadConnection, cl *meta.Client) error {
	pages, err := s.Q.LeadPagesOfConnection(ctx, c.ID)
	if err != nil {
		return err
	}
	var errs []error
	for _, sp := range pages {
		if !sp.Chosen {
			continue
		}
		pg, err := s.pageOf(sp.ID, sp.PageID, sp.PageTokenSealed)
		if err != nil {
			continue
		}
		if err := cl.Subscribe(ctx, pg); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Disconnect stops collecting: the Pages are unsubscribed (as far as Meta
// answers), the credentials and Page tokens deleted and the callback URL
// stops working. Contacts, their consent and the leads' record stay.
func (s *Service) Disconnect(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (LeadAdsConnection, error) {
	c, err := s.live(ctx, s.Q, ws, id)
	if err != nil {
		return LeadAdsConnection{}, err
	}
	if cl, err := s.client(c); err == nil {
		if pages, err := s.Q.LeadPagesOfConnection(ctx, c.ID); err == nil {
			for _, sp := range pages {
				if !sp.Chosen {
					continue
				}
				if pg, err := s.pageOf(sp.ID, sp.PageID, sp.PageTokenSealed); err == nil {
					if err := cl.Unsubscribe(ctx, pg); err != nil {
						s.log().WarnContext(ctx, "lead ads: unsubscribe a Page", "connection", c.ID, "page", sp.PageID, "err", err)
					}
				}
			}
		}
	}
	var by *uuid.UUID
	if p != nil && p.UserID != uuid.Nil {
		by = new(p.UserID)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := s.now()
		if err := q.LeadConnectionDisconnect(ctx, store.LeadConnectionDisconnectParams{ID: c.ID, By: by, At: now}); err != nil {
			return err
		}
		if err := q.LeadPageWipeTokens(ctx, store.LeadPageWipeTokensParams{ConnectionID: c.ID, At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "lead_connection.disconnected", TargetType: "lead_connection",
			TargetID: c.ID.String(), TargetName: c.Name, Before: map[string]any{"status": c.Status}, After: map[string]any{"status": StatusDisconnected}})
	})
	if err != nil {
		return LeadAdsConnection{}, err
	}
	return s.Get(ctx, ws, p, c.ID)
}

func isUnique(err error) bool {
	var pe interface{ SQLState() string }
	return errors.As(err, &pe) && pe.SQLState() == "23505"
}
