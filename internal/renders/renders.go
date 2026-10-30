// Package renders sends a campaign to the workspace's own rendering service,
// Litmus or Mailgun Inspect, for screenshots from real email clients
// (review 2026-10, CO-13). The composer's built-in previews are drawn in the
// browser from the compiled HTML and say so; this is the optional real thing.
//
// The API key is the workspace's own, sealed at rest (secrets.Box) and never
// returned. Calls go through the address guard (netguard) like every call to
// a service outside the instance. Content leaves the instance only on
// purpose: a person asks for each render, it is audited, and the email is
// personalized for the synthetic sample contact, never a real one.
package renders

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Services a workspace can connect.
const (
	Litmus         = "litmus"
	MailgunInspect = "mailgun_inspect"
)

// Render and per-client states.
const (
	StatusPending = "pending"
	StatusDone    = "done"
	StatusFailed  = "failed"
)

const (
	// pollEvery is how often a running render asks the service again,
	// however many people watch it.
	pollEvery = 4 * time.Second
	// giveUpAfter fails a render the service hasn't finished.
	giveUpAfter = 15 * time.Minute
	// listed is how many renders a campaign shows.
	listed = 5
)

var (
	endpointsMu sync.RWMutex
	// endpoints are the services' API bases (Mailgun by region). Tests
	// point them at fakes (export_test.go).
	endpoints = map[string]string{
		Litmus:                 "https://instant-api.litmus.com",
		MailgunInspect + "/us": "https://api.mailgun.net",
		MailgunInspect + "/eu": "https://api.eu.mailgun.net",
	}
)

func endpoint(service, region string) string {
	endpointsMu.RLock()
	defer endpointsMu.RUnlock()
	if service == MailgunInspect {
		if region == "" {
			region = "us"
		}
		return endpoints[service+"/"+region]
	}
	return endpoints[service]
}

// Email is what a service renders: a saved version of the campaign as the
// sample contact gets it.
type Email struct {
	Subject string
	HTML    string
	Version int
	// Name is the campaign's, for the audit log.
	Name string
}

// Content gives the email to render (internal/campaigns, wired in
// internal/app).
type Content interface {
	// SampleRender is the campaign's saved content (one A/B variant)
	// personalized for the synthetic sample contact, so no contact's data
	// leaves the instance.
	SampleRender(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, variant string) (Email, error)
	// CampaignName is the campaign's name, or a 404 problem when the
	// workspace has no such campaign.
	CampaignName(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (string, error)
}

// Service is the rendering service domain.
type Service struct {
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Audit   *audit.Logger
	Box     *secrets.Box
	Content Content
	// HTTP calls the services (a netguard client in the app).
	HTTP *http.Client
	Now  func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// RenderingService is the workspace's connection, without its key.
type RenderingService struct {
	Connected bool   `json:"connected"`
	Service   string `json:"service,omitempty" enum:"litmus,mailgun_inspect"`
	Region    string `json:"region,omitempty" enum:"us,eu" doc:"Mailgun Inspect: the account's region"`
	// KeyHint is the key's last four characters; the key itself is never returned.
	KeyHint     string         `json:"keyHint,omitempty"`
	Status      string         `json:"status,omitempty" enum:"connected,failing" doc:"failing: the last call was refused or failed (lastError says why)"`
	LastError   string         `json:"lastError,omitempty"`
	ConnectedBy string         `json:"connectedBy,omitempty"`
	ConnectedAt *time.Time     `json:"connectedAt,omitempty"`
	CheckedAt   *time.Time     `json:"checkedAt,omitempty"`
	Clients     []RenderClient `json:"clients" doc:"The email clients the account can render, as the service last listed them"`
}

// RenderClient is an email client a service renders in.
type RenderClient struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Default marks the clients picked when nobody chooses: Gmail, Outlook,
	// Apple Mail, iPhone, Android, Yahoo, when the account has them.
	Default bool `json:"default"`
}

// RenderingConnectInput connects (or replaces) the workspace's service.
type RenderingConnectInput struct {
	Service string `json:"service" enum:"litmus,mailgun_inspect"`
	APIKey  string `json:"apiKey" minLength:"8" maxLength:"256" doc:"Write-only: sealed at rest and never returned"`
	Region  string `json:"region,omitempty" enum:"us,eu" doc:"Mailgun Inspect: the account's region (us by default)"`
}

// CampaignRenderInput asks for a render of the saved campaign.
type CampaignRenderInput struct {
	Clients []string `json:"clients" minItems:"1" maxItems:"20" doc:"Client ids from the connection's clients; each is one preview on the account's plan"`
	Variant string   `json:"variant,omitempty" enum:"A,B,C" doc:"The A/B variant to render (A by default)"`
}

// Render is one request to the service and what it returned.
type Render struct {
	ID          string         `json:"id"`
	Version     int            `json:"version" doc:"The saved version rendered"`
	Variant     string         `json:"variant" enum:"A,B,C"`
	Service     string         `json:"service" enum:"litmus,mailgun_inspect"`
	Status      string         `json:"status" enum:"pending,done,failed"`
	Results     []RenderResult `json:"results"`
	Error       string         `json:"error,omitempty"`
	RequestedBy string         `json:"requestedBy"`
	RequestedAt time.Time      `json:"requestedAt"`
	CompletedAt *time.Time     `json:"completedAt,omitempty"`
}

// RenderResult is one client's screenshot.
type RenderResult struct {
	Client   string `json:"client"`
	Name     string `json:"name"`
	Status   string `json:"status" enum:"pending,done,failed"`
	ImageURL string `json:"imageUrl,omitempty" doc:"The full screenshot, hosted by the service"`
	ThumbURL string `json:"thumbUrl,omitempty"`
}

// RenderList is a campaign's latest renders, newest first.
type RenderList struct {
	Items []Render `json:"items"`
}

// credentials is the sealed connection secret.
type credentials struct {
	APIKey string `json:"apiKey"`
}

func credentialsAD(workspaceID uuid.UUID) string {
	return "rendering:" + workspaceID.String() + ":credentials"
}

func (s *Service) seal(ws uuid.UUID, key string) ([]byte, error) {
	b, err := json.Marshal(credentials{APIKey: key}) //nolint:gosec // G117: sealed on the next line, never stored or returned in clear
	if err != nil {
		return nil, err
	}
	return s.Box.Seal(b, credentialsAD(ws))
}

func (s *Service) open(ws uuid.UUID, sealed []byte) (string, error) {
	b, err := s.Box.Open(sealed, credentialsAD(ws))
	if err != nil {
		return "", err
	}
	var c credentials
	if err := json.Unmarshal(b, &c); err != nil {
		return "", err
	}
	return c.APIKey, nil
}

func keyHint(key string) string {
	if len(key) <= 4 {
		return ""
	}
	return key[len(key)-4:]
}

func view(r store.RenderingServiceGetRow) RenderingService {
	at := r.ConnectedAt
	v := RenderingService{Connected: true, Service: r.Service, Region: r.Region, KeyHint: r.KeyHint, Status: r.Status, LastError: r.LastError,
		ConnectedBy: r.ConnectedByName, ConnectedAt: &at, CheckedAt: r.CheckedAt, Clients: []RenderClient{}}
	_ = json.Unmarshal(r.Clients, &v.Clients)
	return v
}

// Get returns the workspace's connection (Connected false when none).
func (s *Service) Get(ctx context.Context, ws *store.Workspace) (RenderingService, error) {
	r, err := s.Q.RenderingServiceGet(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return RenderingService{Clients: []RenderClient{}}, nil
	}
	if err != nil {
		return RenderingService{}, err
	}
	return view(r), nil
}

func (s *Service) conn(service, region, key string) conn {
	return conn{http: s.HTTP, base: endpoint(service, region), service: service, key: key}
}

// Connect tests a key with the service, then keeps it (sealed).
func (s *Service) Connect(ctx context.Context, ws *store.Workspace, p *authz.Principal, in RenderingConnectInput) (RenderingService, error) {
	key := strings.TrimSpace(in.APIKey)
	region := ""
	if in.Service == MailgunInspect {
		region = in.Region
		if region == "" {
			region = "us"
		}
	}
	a, ok := adapters[in.Service]
	if !ok {
		return RenderingService{}, api.Invalid("body.service", say(ctx, mUnknownService))
	}
	clients, err := a.clients(ctx, s.conn(in.Service, region, key))
	if err != nil {
		return RenderingService{}, problem(ctx, in.Service, err, "body.apiKey")
	}
	markDefaults(clients)
	sealed, err := s.seal(ws.ID, key)
	if err != nil {
		return RenderingService{}, err
	}
	raw, _ := json.Marshal(clients)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RenderingServiceUpsert(ctx, store.RenderingServiceUpsertParams{WorkspaceID: ws.ID, Service: in.Service, Region: region, Credentials: sealed,
			KeyHint: keyHint(key), Clients: raw, ConnectedBy: actor(p), Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "rendering.connect", TargetType: "rendering_service", TargetID: ws.ID.String(),
			TargetName: serviceName(in.Service), After: map[string]any{"service": in.Service, "region": region, "keyHint": keyHint(key), "clients": len(clients)}})
	})
	if err != nil {
		return RenderingService{}, err
	}
	return s.Get(ctx, ws)
}

// Test asks the service again with the stored key and records the outcome.
func (s *Service) Test(ctx context.Context, ws *store.Workspace) (RenderingService, error) {
	r, err := s.Q.RenderingServiceGet(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return RenderingService{}, notConnected(ctx)
	}
	if err != nil {
		return RenderingService{}, err
	}
	key, err := s.open(ws.ID, r.Credentials)
	if err != nil {
		return RenderingService{}, err
	}
	clients, cerr := adapters[r.Service].clients(ctx, s.conn(r.Service, r.Region, key))
	status, lastError := "connected", ""
	if cerr != nil {
		status, lastError = "failing", describe(ctx, r.Service, cerr)
	}
	markDefaults(clients)
	raw, _ := json.Marshal(clients)
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RenderingServiceChecked(ctx, store.RenderingServiceCheckedParams{Status: status, LastError: lastError, Now: &now, Clients: raw, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "rendering.test", TargetType: "rendering_service", TargetID: ws.ID.String(),
			TargetName: serviceName(r.Service), After: map[string]any{"status": status}})
	})
	if err != nil {
		return RenderingService{}, err
	}
	return s.Get(ctx, ws)
}

// Disconnect forgets the service and its key. Renders already made stay.
func (s *Service) Disconnect(ctx context.Context, ws *store.Workspace) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.RenderingServiceGet(ctx, ws.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return notConnected(ctx)
		}
		if err != nil {
			return err
		}
		if _, err := q.RenderingServiceDelete(ctx, ws.ID); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "rendering.disconnect", TargetType: "rendering_service", TargetID: ws.ID.String(),
			TargetName: serviceName(r.Service), Before: map[string]any{"service": r.Service, "keyHint": r.KeyHint}})
	})
}

// failing records a refused or failing call on the connection.
func (s *Service) failing(ctx context.Context, ws *store.Workspace, service string, err error, clients json.RawMessage) {
	now := s.now()
	_ = s.Q.RenderingServiceChecked(ctx, store.RenderingServiceCheckedParams{Status: "failing", LastError: describe(ctx, service, err), Now: &now, Clients: clients, WorkspaceID: ws.ID})
}

// Start sends the campaign's saved content to the service for the clients.
func (s *Service) Start(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, in CampaignRenderInput) (Render, error) {
	r, err := s.Q.RenderingServiceGet(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Render{}, notConnected(ctx)
	}
	if err != nil {
		return Render{}, err
	}
	known := []RenderClient{}
	_ = json.Unmarshal(r.Clients, &known)
	names := map[string]string{}
	for _, c := range known {
		names[c.ID] = c.Name
	}
	var clients []string
	for _, id := range in.Clients {
		if _, ok := names[id]; !ok {
			return Render{}, api.Invalid("body.clients", say(ctx, mUnknownClient, i18n.Args{"client": id, "service": serviceName(r.Service)}))
		}
		if !slices.Contains(clients, id) {
			clients = append(clients, id)
		}
	}
	if pending, err := s.Q.CampaignRenderPending(ctx, store.CampaignRenderPendingParams{WorkspaceID: ws.ID, CampaignID: campaignID}); err == nil {
		return Render{}, api.Err(http.StatusConflict, "render_running", say(ctx, mStillRunning)).WithMeta("renderId", pending.String())
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Render{}, err
	}
	variant := in.Variant
	if variant == "" {
		variant = "A"
	}
	e, err := s.Content.SampleRender(ctx, ws, p, campaignID, variant)
	if err != nil {
		return Render{}, err
	}
	key, err := s.open(ws.ID, r.Credentials)
	if err != nil {
		return Render{}, err
	}
	id, err := adapters[r.Service].start(ctx, s.conn(r.Service, r.Region, key), e, clients)
	if err != nil {
		var ce *callError
		if errors.As(err, &ce) && ce.kind == refused {
			s.failing(ctx, ws, r.Service, err, r.Clients)
		}
		return Render{}, problem(ctx, r.Service, err, "")
	}
	results := make([]RenderResult, len(clients))
	for i, c := range clients {
		results[i] = RenderResult{Client: c, Name: names[c], Status: StatusPending}
	}
	raw, _ := json.Marshal(results)
	rid := uuid.Must(uuid.NewV7())
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.CampaignRenderInsert(ctx, store.CampaignRenderInsertParams{ID: rid, WorkspaceID: ws.ID, CampaignID: campaignID, Version: int32(e.Version), Variant: variant,
			Service: r.Service, ExternalID: id, Clients: clients, Results: raw, RequestedBy: actor(p), RequestedAt: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.render", TargetType: "campaign", TargetID: campaignID.String(),
			TargetName: e.Name, After: map[string]any{"service": r.Service, "clients": clients, "version": e.Version, "variant": variant, "renderId": rid.String()}})
	})
	if err != nil {
		return Render{}, err
	}
	name := ""
	if p != nil {
		name = p.Name
	}
	return Render{ID: rid.String(), Version: e.Version, Variant: variant, Service: r.Service, Status: StatusPending, Results: results, RequestedBy: name, RequestedAt: now}, nil
}

// List returns the campaign's latest renders, asking the service about
// running ones (at most every few seconds, whoever is watching).
func (s *Service) List(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (RenderList, error) {
	rows, err := s.Q.CampaignRendersList(ctx, store.CampaignRendersListParams{WorkspaceID: ws.ID, CampaignID: campaignID, Lim: listed})
	if err != nil {
		return RenderList{}, err
	}
	var key, region string
	var conn *store.RenderingServiceGetRow
	out := RenderList{Items: make([]Render, 0, len(rows))}
	for _, r := range rows {
		item := Render{ID: r.ID.String(), Version: int(r.Version), Variant: r.Variant, Service: r.Service, Status: r.Status, Error: r.Error,
			RequestedBy: r.RequestedByName, RequestedAt: r.RequestedAt, CompletedAt: r.CompletedAt, Results: []RenderResult{}}
		_ = json.Unmarshal(r.Results, &item.Results)
		now := s.now()
		if r.Status == StatusPending && (r.PolledAt == nil || now.Sub(*r.PolledAt) >= pollEvery) {
			if conn == nil {
				c, err := s.Q.RenderingServiceGet(ctx, ws.ID)
				if err == nil {
					conn = &c
					key, _ = s.open(ws.ID, c.Credentials)
					region = c.Region
				}
			}
			s.refresh(ctx, ws, conn, key, region, r, &item)
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

// refresh asks the service about a running render and stores the answer.
func (s *Service) refresh(ctx context.Context, ws *store.Workspace, conn *store.RenderingServiceGetRow, key, region string, r store.CampaignRendersListRow, item *Render) {
	now := s.now()
	switch {
	case now.Sub(r.RequestedAt) > giveUpAfter:
		item.Status, item.Error = StatusFailed, say(ctx, mGaveUp, i18n.Args{"service": serviceName(r.Service)})
		for i := range item.Results {
			if item.Results[i].Status == StatusPending {
				item.Results[i].Status = StatusFailed
			}
		}
	case conn == nil || conn.Service != r.Service || key == "":
		// Disconnected (or another service) since: nothing to ask.
		item.Status, item.Error = StatusFailed, say(ctx, mDisconnected, i18n.Args{"service": serviceName(r.Service)})
	default:
		got, err := adapters[r.Service].results(ctx, s.conn(r.Service, region, key), r.ExternalID, r.Clients)
		if err != nil {
			var ce *callError
			if errors.As(err, &ce) && ce.kind == refused {
				s.failing(ctx, ws, r.Service, err, conn.Clients)
			}
			// Try again on the next look; the render stays running.
			_ = s.Q.CampaignRenderUpdate(ctx, store.CampaignRenderUpdateParams{Status: StatusPending, Results: r.Results, Error: "", Now: &now, WorkspaceID: ws.ID, ID: r.ID})
			return
		}
		byClient := map[string]clientResult{}
		for _, g := range got {
			byClient[g.client] = g
		}
		done, failed := 0, 0
		for i := range item.Results {
			if g, ok := byClient[item.Results[i].Client]; ok {
				item.Results[i].Status, item.Results[i].ImageURL, item.Results[i].ThumbURL = g.status, safeURL(g.image), safeURL(g.thumb)
				if g.status == StatusDone && item.Results[i].ImageURL == "" {
					item.Results[i].Status = StatusFailed
				}
			}
			switch item.Results[i].Status {
			case StatusDone:
				done++
			case StatusFailed:
				failed++
			}
		}
		switch {
		case done+failed < len(item.Results):
			item.Status = StatusPending
		case done == 0:
			item.Status, item.Error = StatusFailed, say(ctx, mNoScreenshots, i18n.Args{"service": serviceName(r.Service)})
		default:
			item.Status = StatusDone
		}
	}
	raw, _ := json.Marshal(item.Results)
	_ = s.Q.CampaignRenderUpdate(ctx, store.CampaignRenderUpdateParams{Status: item.Status, Results: raw, Error: item.Error, Now: &now, WorkspaceID: ws.ID, ID: r.ID})
	if item.Status != StatusPending {
		item.CompletedAt = &now
	}
}

// safeURL keeps an https URL with a host, and nothing else: screenshots are
// shown as images and links.
func safeURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return ""
	}
	return u.String()
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// validID is an id that's safe in a URL path segment.
func validID(id string) bool { return idPattern.MatchString(id) }

// defaultPicks are the clients picked when nobody chooses, by name.
var defaultPicks = []*regexp.Regexp{
	regexp.MustCompile(`(?i)gmail`),
	regexp.MustCompile(`(?i)outlook.*(20\d\d|365|windows)|(20\d\d|365).*outlook`),
	regexp.MustCompile(`(?i)apple mail|mac ?os`),
	regexp.MustCompile(`(?i)iphone|ios`),
	regexp.MustCompile(`(?i)android|samsung`),
	regexp.MustCompile(`(?i)yahoo`),
}

// markDefaults picks one client per common kind.
func markDefaults(cs []RenderClient) {
	for _, re := range defaultPicks {
		for i := range cs {
			if !cs[i].Default && re.MatchString(cs[i].Name) {
				cs[i].Default = true
				break
			}
		}
	}
}

func actor(p *authz.Principal) *uuid.UUID {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

func serviceName(service string) string {
	switch service {
	case Litmus:
		return "Litmus"
	case MailgunInspect:
		return "Mailgun Inspect"
	}
	return service
}
