// Package domains implements sending domains and sender identities (spec
// 7.13 Settings › Senders, and the domain step of 7.2): the DNS records each
// sending provider needs (through the provider's domain API when it has one),
// live verification with internal/dnscheck, the authentication summary per
// domain, verification per sender and sending provider, the default sender
// of each stream, custom link-tracking domains, adding records through the
// Cloudflare API, and a periodic re-verification job.
package domains

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/mail"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/idna"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service is the senders and domains domain.
type Service struct {
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Audit   *audit.Logger
	Factory *manage.Factory
	DNS     dnscheck.Resolver
	// NS returns the nameservers of a zone; nil skips the lookup.
	NS func(ctx context.Context, name string) ([]string, error)
	// TTL returns how long resolvers may cache a record set; nil skips it.
	TTL TTLFunc
	// TrackingTarget is the host custom tracking domains point to (the
	// host of the public URL).
	TrackingTarget func() string
	Cloudflare     *Cloudflare
	Jobs           func() *jobs.Client
	Log            *slog.Logger
	Now            func() time.Time

	checkMu   sync.Mutex
	lastCheck map[uuid.UUID]time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// Streams a sender can be the default for.
var streamOrder = []string{"broadcast", "automation", "system"}

// StreamName is the display name of a stream.
func StreamName(s string) string {
	switch s {
	case "broadcast":
		return "Broadcasts"
	case "automation":
		return "Automations"
	}
	return "System"
}

// SenderVerification is one sending-provider column of a sender row.
type SenderVerification struct {
	ProviderID string `json:"providerId"`
	Status     string `json:"status" enum:"verified,pending,failed,not_set_up,not_used"`
	// Text is the badge: "Verified", "Pending · acme.io DKIM", "Not set up", "Not used".
	Text string `json:"text"`
}

// SenderIdentity is one row of "Sender identities".
type SenderIdentity struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	ReplyTo  string `json:"replyTo"`
	Note     string `json:"note"`
	DomainID string `json:"domainId"`
	Domain   string `json:"domain"`
	// DefaultFor lists the streams this sender is the default From of.
	DefaultFor   []string             `json:"defaultFor"`
	Verification []SenderVerification `json:"verification"`
	// Pending is true while a provider that sends as this address still
	// waits for the domain's DNS records ("Verify").
	Pending bool `json:"pending"`
}

// SenderDomainCell is one status cell of the domains table.
type SenderDomainCell struct {
	Text string `json:"text"`
	Tone string `json:"tone" enum:"ok,warn,bad,neutral"`
}

// SendingDomainRow is one row of "Domains".
type SendingDomainRow struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
	Note   string `json:"note"`
	Status string `json:"status" enum:"pending,verified,partial,failed"`
	// UsedBy are the short names of the providers set up for the domain.
	UsedBy     []string         `json:"usedBy"`
	DKIM       SenderDomainCell `json:"dkim"`
	SPF        SenderDomainCell `json:"spf"`
	DMARC      SenderDomainCell `json:"dmarc"`
	ReturnPath SenderDomainCell `json:"returnPath"`
	Tracking   SenderDomainCell `json:"tracking"`
	BIMI       SenderDomainCell `json:"bimi"`
	Senders    int              `json:"senders"`
	CheckedAt  *time.Time       `json:"checkedAt,omitempty"`
}

// SenderProviderColumn is a connected sending provider.
type SenderProviderColumn struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Short   string `json:"short"`
	Adapter string `json:"adapter"`
}

// SenderTrackingOption is a tracking domain that can be the default.
type SenderTrackingOption struct {
	Domain   string `json:"domain"`
	Verified bool   `json:"verified"`
}

// SenderDefaults are the "Defaults" card of Settings › Senders.
type SenderDefaults struct {
	// DefaultSenderID is the Broadcasts default ("Default From for new campaigns").
	DefaultSenderID    string `json:"defaultSenderId"`
	AutomationSenderID string `json:"automationSenderId"`
	SystemSenderID     string `json:"systemSenderId"`
	// ReplyHandling: forward replies to ReplyTo, or keep them in a
	// ReputeCast inbox (not available in this version).
	ReplyHandling string `json:"replyHandling" enum:"forward,inbox"`
	ReplyTo       string `json:"replyTo"`
	FooterAddress string `json:"footerAddress"`
	// TrackingDomain is the default link tracking domain ("" = the
	// provider's default, i.e. ReputeCast's public URL).
	TrackingDomain  string                 `json:"trackingDomain"`
	TrackingOptions []SenderTrackingOption `json:"trackingOptions"`
	ReplyOptions    []string               `json:"replyOptions"`
}

// SendersOverview is Settings › Senders.
type SendersOverview struct {
	Senders   []SenderIdentity       `json:"senders"`
	Domains   []SendingDomainRow     `json:"domains"`
	Providers []SenderProviderColumn `json:"providers"`
	Defaults  SenderDefaults         `json:"defaults"`
	CanManage bool                   `json:"canManage"`
	// Helpers names who can manage domains and senders when the caller can't.
	Helpers []string `json:"helpers"`
}

// sendingDefaults is workspaces.sending_defaults (see migration 00550).
type sendingDefaults struct {
	ReplyHandling  string `json:"reply_handling,omitempty"`
	ReplyTo        string `json:"reply_to,omitempty"`
	TrackingDomain string `json:"tracking_domain,omitempty"`
	// The rest belongs to Settings › Providers (internal/connect) and is
	// kept as is.
	Rest map[string]json.RawMessage `json:"-"`
}

func decodeSendingDefaults(raw json.RawMessage) sendingDefaults {
	d := sendingDefaults{Rest: map[string]json.RawMessage{}}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &d.Rest)
		_ = json.Unmarshal(raw, &d)
	}
	return d
}

func (d sendingDefaults) encode() json.RawMessage {
	m := map[string]any{}
	for k, v := range d.Rest {
		m[k] = v
	}
	set := func(k, v string) {
		if v == "" {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	set("reply_handling", d.ReplyHandling)
	set("reply_to", d.ReplyTo)
	set("tracking_domain", d.TrackingDomain)
	b, _ := json.Marshal(m)
	return b
}

type workspaceDefaults struct {
	ids      map[string]*uuid.UUID // stream → sender
	sending  sendingDefaults
	branding map[string]any
}

func (s *Service) workspaceDefaults(ctx context.Context, q *store.Queries, wsID uuid.UUID) (workspaceDefaults, error) {
	row, err := q.SenderWorkspaceDefaults(ctx, wsID)
	if err != nil {
		return workspaceDefaults{}, err
	}
	out := workspaceDefaults{
		ids:      map[string]*uuid.UUID{"broadcast": row.DefaultSenderID, "automation": row.DefaultAutomationSenderID, "system": row.DefaultSystemSenderID},
		sending:  decodeSendingDefaults(row.SendingDefaults),
		branding: map[string]any{},
	}
	_ = json.Unmarshal(row.Branding, &out.branding)
	return out, nil
}

// providerColumns returns the workspace's connected providers in the order
// the senders table shows them (connection order).
func (s *Service) providerColumns(ctx context.Context, wsID uuid.UUID) ([]store.Provider, []SenderProviderColumn, error) {
	provs, err := s.Q.ProvListActive(ctx, wsID)
	if err != nil {
		return nil, nil, err
	}
	cols := make([]SenderProviderColumn, 0, len(provs))
	for _, p := range provs {
		cols = append(cols, SenderProviderColumn{ID: p.ID.String(), Name: p.Name, Short: state.Short(p.Name, p.Adapter), Adapter: p.Adapter})
	}
	return provs, cols, nil
}

// Overview returns Settings › Senders.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace) (SendersOverview, error) {
	provs, cols, err := s.providerColumns(ctx, ws.ID)
	if err != nil {
		return SendersOverview{}, err
	}
	doms, err := s.Q.DomList(ctx, ws.ID)
	if err != nil {
		return SendersOverview{}, err
	}
	senders, err := s.Q.SenderList(ctx, ws.ID)
	if err != nil {
		return SendersOverview{}, err
	}
	defs, err := s.workspaceDefaults(ctx, s.Q, ws.ID)
	if err != nil {
		return SendersOverview{}, err
	}
	out := SendersOverview{Senders: []SenderIdentity{}, Domains: []SendingDomainRow{}, Providers: cols}
	domByID := map[uuid.UUID]store.SendingDomain{}
	perDomain := map[uuid.UUID]int{}
	for _, d := range doms {
		domByID[d.ID] = d
	}
	for _, sn := range senders {
		perDomain[sn.DomainID]++
		out.Senders = append(out.Senders, s.senderView(sn.ID, sn.Name, sn.Email, sn.ReplyTo, sn.Note, sn.DomainID, sn.DomainName, sn.Verification, defs, provs, domByID[sn.DomainID]))
	}
	for _, d := range doms {
		row := domainRow(d, provs)
		row.Senders = perDomain[d.ID]
		out.Domains = append(out.Domains, row)
	}
	out.Defaults = defaultsView(defs, senders, doms)
	if p := api.PrincipalFrom(ctx); p != nil {
		out.CanManage = p.Can(authz.DomainsManage)
	}
	out.Helpers = s.helpers(ctx, ws.ID, out.CanManage)
	return out, nil
}

// helpers names who can manage domains and senders when the caller can't
// ("Ask Linh Nguyen or Marcus Chen").
func (s *Service) helpers(ctx context.Context, wsID uuid.UUID, can bool) []string {
	if can {
		return []string{}
	}
	if h := access.Helpers(ctx, s.Q, wsID, authz.DomainsManage); h != nil {
		return h
	}
	return []string{}
}

func defaultsView(defs workspaceDefaults, senders []store.SenderListRow, doms []store.SendingDomain) SenderDefaults {
	out := SenderDefaults{ReplyHandling: "forward", TrackingOptions: []SenderTrackingOption{}, ReplyOptions: []string{}}
	idStr := func(id *uuid.UUID) string {
		if id == nil {
			return ""
		}
		return id.String()
	}
	out.DefaultSenderID, out.AutomationSenderID, out.SystemSenderID = idStr(defs.ids["broadcast"]), idStr(defs.ids["automation"]), idStr(defs.ids["system"])
	if defs.sending.ReplyHandling != "" {
		out.ReplyHandling = defs.sending.ReplyHandling
	}
	for _, sn := range senders {
		if sn.ReplyTo != "" && !slices.Contains(out.ReplyOptions, sn.ReplyTo) {
			out.ReplyOptions = append(out.ReplyOptions, sn.ReplyTo)
		}
	}
	out.ReplyTo = defs.sending.ReplyTo
	if out.ReplyTo == "" {
		// The Broadcasts default's reply-to, else the first one used.
		for _, sn := range senders {
			if defs.ids["broadcast"] != nil && sn.ID == *defs.ids["broadcast"] && sn.ReplyTo != "" {
				out.ReplyTo = sn.ReplyTo
			}
		}
		if out.ReplyTo == "" && len(out.ReplyOptions) > 0 {
			out.ReplyTo = out.ReplyOptions[0]
		}
	}
	if out.ReplyTo != "" && !slices.Contains(out.ReplyOptions, out.ReplyTo) {
		out.ReplyOptions = append([]string{out.ReplyTo}, out.ReplyOptions...)
	}
	if v, ok := defs.branding["footer_address"].(string); ok {
		out.FooterAddress = v
	}
	out.TrackingDomain = defs.sending.TrackingDomain
	for _, d := range doms {
		if d.TrackingDomain != "" {
			out.TrackingOptions = append(out.TrackingOptions, SenderTrackingOption{Domain: d.TrackingDomain, Verified: d.TrackingStatus == StatusVerified})
		}
	}
	return out
}

// senderView builds a sender row.
func (s *Service) senderView(id uuid.UUID, name, email, replyTo, note string, domainID uuid.UUID, domain string, verification json.RawMessage,
	defs workspaceDefaults, provs []store.Provider, d store.SendingDomain) SenderIdentity {
	v := SenderIdentity{ID: id.String(), Name: name, Email: email, ReplyTo: replyTo, Note: note, DomainID: domainID.String(), Domain: domain,
		DefaultFor: []string{}, Verification: []SenderVerification{}}
	for _, st := range streamOrder {
		if p := defs.ids[st]; p != nil && *p == id {
			v.DefaultFor = append(v.DefaultFor, st)
		}
	}
	stored := decodeVerification(verification)
	recs := DecodeRecords(d.DnsRecords)
	for _, p := range provs {
		st := stored[p.ID.String()]
		if st != "not_used" {
			st = providerVerification(recs, p.ID.String())
		}
		sv := SenderVerification{ProviderID: p.ID.String(), Status: st, Text: verificationText(st, domain)}
		if st == "pending" || st == "failed" {
			v.Pending = true
		}
		v.Verification = append(v.Verification, sv)
	}
	return v
}

func decodeVerification(raw json.RawMessage) map[string]string {
	m := map[string]string{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &m)
	}
	return m
}

// providerVerification is a sender's status at one provider, from the DNS
// records that provider needs for the sender's domain.
func providerVerification(recs []Record, providerID string) string {
	n, verified, failed := 0, 0, 0
	for _, r := range recs {
		if r.ProviderID != providerID {
			continue
		}
		n++
		switch r.Status {
		case StatusVerified:
			verified++
		case StatusWrong:
			if r.ProviderStatus == "failed" {
				failed++
			}
		}
		if r.ProviderStatus == "failed" && r.Status != StatusVerified {
			failed++
		}
	}
	switch {
	case n == 0:
		return "not_set_up"
	case verified == n:
		return "verified"
	case failed > 0:
		return "failed"
	}
	return "pending"
}

func verificationText(status, domain string) string {
	switch status {
	case "verified":
		return "Verified"
	case "pending":
		return "Pending · " + domain + " DKIM"
	case "failed":
		return "Failed · " + domain + " DKIM"
	case "not_used":
		return "Not used"
	}
	return "Not set up"
}

// SenderInput creates or changes a sender identity.
type SenderInput struct {
	Name    string `json:"name" minLength:"1" maxLength:"120"`
	Email   string `json:"email,omitempty" maxLength:"254" doc:"Required when creating; the address can't change later"`
	ReplyTo string `json:"replyTo,omitempty" maxLength:"254"`
	Note    string `json:"note,omitempty" maxLength:"120" doc:"Shown when the sender is no stream's default, e.g. personal sends"`
	// DefaultFor makes the sender the default From of these streams.
	DefaultFor []string `json:"defaultFor,omitempty" enum:"broadcast,automation,system"`
	// NotUsedWith lists providers that must never send as this address.
	NotUsedWith []string `json:"notUsedWith,omitempty"`
}

func (s *Service) audit(ctx context.Context, q *store.Queries, ws *store.Workspace, action, targetType, id, name string, before, after any) error {
	wsID := ws.ID
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: action, TargetType: targetType, TargetID: id, TargetName: name, Before: before, After: after})
}

// normalizeEmail validates an address and returns it lowercased with its domain.
func normalizeEmail(field, raw string) (string, string, error) {
	a, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil || a.Name != "" || !strings.Contains(a.Address, "@") {
		return "", "", api.Invalid(field, "Enter an address like hello@news.acme.io.")
	}
	email := strings.ToLower(a.Address)
	domain := email[strings.LastIndexByte(email, '@')+1:]
	return email, domain, nil
}

// CreateSender adds a sender identity on a sending domain of the workspace.
func (s *Service) CreateSender(ctx context.Context, ws *store.Workspace, in SenderInput) (SenderIdentity, error) {
	email, domain, err := normalizeEmail("body.email", in.Email)
	if err != nil {
		return SenderIdentity{}, err
	}
	replyTo := ""
	if strings.TrimSpace(in.ReplyTo) != "" {
		if replyTo, _, err = normalizeEmail("body.replyTo", in.ReplyTo); err != nil {
			return SenderIdentity{}, err
		}
	}
	d, err := s.Q.DomGetByName(ctx, store.DomGetByNameParams{WorkspaceID: ws.ID, Domain: domain})
	if errors.Is(err, pgx.ErrNoRows) {
		return SenderIdentity{}, api.Invalid("body.email", domain+" isn't a sending domain of this workspace yet. Add the domain first, then the sender.").WithMeta("domain", domain)
	} else if err != nil {
		return SenderIdentity{}, err
	}
	if _, err := s.Q.SenderGetByEmail(ctx, store.SenderGetByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
		return SenderIdentity{}, api.Conflict(email + " is already a sender.")
	}
	ver := map[string]string{}
	for _, pid := range in.NotUsedWith {
		ver[pid] = "not_used"
	}
	verb, _ := json.Marshal(ver)
	id := newID()
	var by *uuid.UUID
	if p := api.PrincipalFrom(ctx); p != nil && p.UserID != uuid.Nil {
		u := p.UserID
		by = &u
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.SenderCreate(ctx, store.SenderCreateParams{ID: id, WorkspaceID: ws.ID, DomainID: d.ID, Name: strings.TrimSpace(in.Name), Email: email,
			ReplyTo: replyTo, Note: strings.TrimSpace(in.Note), Verification: verb, CreatedBy: by, CreatedAt: s.now()}); err != nil {
			return err
		}
		if err := s.setDefaults(ctx, q, ws.ID, id, in.DefaultFor, nil); err != nil {
			return err
		}
		return s.audit(ctx, q, ws, "sender.create", "sender", id.String(), email, nil, map[string]any{"name": in.Name, "email": email, "reply_to": replyTo, "default_for": in.DefaultFor})
	})
	if err != nil {
		return SenderIdentity{}, err
	}
	return s.Sender(ctx, ws, id)
}

// setDefaults makes sender the default of the given streams and, when
// previous is not nil, removes it from the streams it no longer defaults.
func (s *Service) setDefaults(ctx context.Context, q *store.Queries, wsID, sender uuid.UUID, streams []string, previous []string) error {
	if len(streams) == 0 && previous == nil {
		return nil
	}
	defs, err := s.workspaceDefaults(ctx, q, wsID)
	if err != nil {
		return err
	}
	for _, st := range streamOrder {
		id := defs.ids[st]
		switch {
		case slices.Contains(streams, st):
			id = &sender
		case previous != nil && id != nil && *id == sender:
			id = nil
		}
		defs.ids[st] = id
	}
	return q.SenderSetStreamDefaults(ctx, store.SenderSetStreamDefaultsParams{ID: wsID, DefaultSenderID: defs.ids["broadcast"],
		DefaultAutomationSenderID: defs.ids["automation"], DefaultSystemSenderID: defs.ids["system"], UpdatedAt: s.now()})
}

// Sender returns one sender row.
func (s *Service) Sender(ctx context.Context, ws *store.Workspace, id uuid.UUID) (SenderIdentity, error) {
	sn, err := s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return SenderIdentity{}, api.MapError(err, "Sender")
	}
	provs, _, err := s.providerColumns(ctx, ws.ID)
	if err != nil {
		return SenderIdentity{}, err
	}
	d, err := s.Q.DomGetAny(ctx, sn.DomainID)
	if err != nil {
		return SenderIdentity{}, err
	}
	defs, err := s.workspaceDefaults(ctx, s.Q, ws.ID)
	if err != nil {
		return SenderIdentity{}, err
	}
	return s.senderView(sn.ID, sn.Name, sn.Email, sn.ReplyTo, sn.Note, sn.DomainID, sn.DomainName, sn.Verification, defs, provs, d), nil
}

func parseID(raw, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, api.NotFound(what)
	}
	return id, nil
}

// UpdateSender changes a sender's name, reply-to, note, default streams and
// the providers that never send as it.
func (s *Service) UpdateSender(ctx context.Context, ws *store.Workspace, rawID string, in SenderInput) (SenderIdentity, error) {
	id, err := parseID(rawID, "Sender")
	if err != nil {
		return SenderIdentity{}, err
	}
	sn, err := s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return SenderIdentity{}, api.MapError(err, "Sender")
	}
	if in.Email != "" && !strings.EqualFold(strings.TrimSpace(in.Email), sn.Email) {
		return SenderIdentity{}, api.Invalid("body.email", "A sender's address can't change. Add a new sender instead.")
	}
	replyTo := ""
	if strings.TrimSpace(in.ReplyTo) != "" {
		if replyTo, _, err = normalizeEmail("body.replyTo", in.ReplyTo); err != nil {
			return SenderIdentity{}, err
		}
	}
	ver := decodeVerification(sn.Verification)
	for k, v := range ver {
		if v == "not_used" {
			delete(ver, k)
		}
	}
	for _, pid := range in.NotUsedWith {
		ver[pid] = "not_used"
	}
	verb, _ := json.Marshal(ver)
	defs, err := s.workspaceDefaults(ctx, s.Q, ws.ID)
	if err != nil {
		return SenderIdentity{}, err
	}
	before := map[string]any{"name": sn.Name, "reply_to": sn.ReplyTo, "note": sn.Note, "verification": sn.Verification, "default_for": defaultStreams(defs, id)}
	after := map[string]any{"name": strings.TrimSpace(in.Name), "reply_to": replyTo, "note": strings.TrimSpace(in.Note), "verification": json.RawMessage(verb), "default_for": in.DefaultFor}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SenderUpdate(ctx, store.SenderUpdateParams{ID: id, Name: strings.TrimSpace(in.Name), ReplyTo: replyTo, Note: strings.TrimSpace(in.Note), Verification: verb, UpdatedAt: s.now()}); err != nil {
			return err
		}
		streams := in.DefaultFor
		if streams == nil {
			streams = []string{}
		}
		if err := s.setDefaults(ctx, q, ws.ID, id, streams, defaultStreams(defs, id)); err != nil {
			return err
		}
		return s.audit(ctx, q, ws, "sender.update", "sender", id.String(), sn.Email, before, after)
	})
	if err != nil {
		return SenderIdentity{}, err
	}
	return s.Sender(ctx, ws, id)
}

func defaultStreams(defs workspaceDefaults, id uuid.UUID) []string {
	out := []string{}
	for _, st := range streamOrder {
		if p := defs.ids[st]; p != nil && *p == id {
			out = append(out, st)
		}
	}
	return out
}

// DeleteSender removes a sender. Unsent campaigns that used it lose their
// From and must pick another before sending.
func (s *Service) DeleteSender(ctx context.Context, ws *store.Workspace, rawID string) error {
	id, err := parseID(rawID, "Sender")
	if err != nil {
		return err
	}
	sn, err := s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return api.MapError(err, "Sender")
	}
	n, err := s.Q.SenderCampaignCount(ctx, &id)
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SenderDelete(ctx, id); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sender.delete", TargetType: "sender", TargetID: id.String(), TargetName: sn.Email,
			Before: map[string]any{"name": sn.Name, "email": sn.Email}, Metadata: map[string]any{"unsent_campaigns": n}})
	})
}

// SenderDefaultsInput saves the "Defaults" card.
type SenderDefaultsInput struct {
	DefaultSenderID string `json:"defaultSenderId,omitempty" doc:"Default From for new campaigns (the Broadcasts default)"`
	ReplyHandling   string `json:"replyHandling" enum:"forward,inbox"`
	ReplyTo         string `json:"replyTo,omitempty" maxLength:"254"`
	FooterAddress   string `json:"footerAddress" maxLength:"300"`
	TrackingDomain  string `json:"trackingDomain,omitempty" maxLength:"253" doc:"Empty for the provider default"`
}

// SaveDefaults saves the "Defaults" card.
func (s *Service) SaveDefaults(ctx context.Context, ws *store.Workspace, in SenderDefaultsInput) (SenderDefaults, error) {
	if in.ReplyHandling == "inbox" {
		return SenderDefaults{}, api.Invalid("body.replyHandling", "Keeping replies in a ReputeCast inbox isn't available yet. Forward them to an address you read.")
	}
	defs, err := s.workspaceDefaults(ctx, s.Q, ws.ID)
	if err != nil {
		return SenderDefaults{}, err
	}
	var broadcast *uuid.UUID
	if in.DefaultSenderID != "" {
		id, err := uuid.Parse(in.DefaultSenderID)
		if err != nil {
			return SenderDefaults{}, api.Invalid("body.defaultSenderId", "Pick a sender of this workspace.")
		}
		if _, err := s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return SenderDefaults{}, api.Invalid("body.defaultSenderId", "Pick a sender of this workspace.")
		}
		broadcast = &id
	}
	replyTo := ""
	if strings.TrimSpace(in.ReplyTo) != "" {
		if replyTo, _, err = normalizeEmail("body.replyTo", in.ReplyTo); err != nil {
			return SenderDefaults{}, err
		}
	}
	tracking := strings.ToLower(strings.TrimSpace(in.TrackingDomain))
	if tracking != "" {
		doms, err := s.Q.DomList(ctx, ws.ID)
		if err != nil {
			return SenderDefaults{}, err
		}
		ok := false
		for _, d := range doms {
			if d.TrackingDomain == tracking {
				ok = true
			}
		}
		if !ok {
			return SenderDefaults{}, api.Invalid("body.trackingDomain", "Add "+tracking+" as the tracking domain of a sending domain first.")
		}
	}
	before := map[string]any{"default_sender_id": defs.ids["broadcast"], "reply_handling": defs.sending.ReplyHandling, "reply_to": defs.sending.ReplyTo,
		"footer_address": defs.branding["footer_address"], "tracking_domain": defs.sending.TrackingDomain}
	next := defs.sending
	next.ReplyHandling, next.ReplyTo, next.TrackingDomain = in.ReplyHandling, replyTo, tracking
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SenderSetStreamDefaults(ctx, store.SenderSetStreamDefaultsParams{ID: ws.ID, DefaultSenderID: broadcast,
			DefaultAutomationSenderID: defs.ids["automation"], DefaultSystemSenderID: defs.ids["system"], UpdatedAt: s.now()}); err != nil {
			return err
		}
		if err := q.SenderSetSendingDefaults(ctx, store.SenderSetSendingDefaultsParams{ID: ws.ID, SendingDefaults: next.encode(), UpdatedAt: s.now()}); err != nil {
			return err
		}
		if err := q.SenderSetFooterAddress(ctx, store.SenderSetFooterAddressParams{ID: ws.ID, FooterAddress: strings.TrimSpace(in.FooterAddress), UpdatedAt: s.now()}); err != nil {
			return err
		}
		return s.audit(ctx, q, ws, "senders.defaults_update", "workspace", ws.ID.String(), ws.Name, before,
			map[string]any{"default_sender_id": broadcast, "reply_handling": in.ReplyHandling, "reply_to": replyTo, "footer_address": strings.TrimSpace(in.FooterAddress), "tracking_domain": tracking})
	})
	if err != nil {
		return SenderDefaults{}, err
	}
	ov, err := s.Overview(ctx, ws)
	return ov.Defaults, err
}

// NormalizeDomain lowercases a domain name, converts it to ASCII (IDNA) and
// checks it is a host name.
func NormalizeDomain(field, raw string) (string, error) {
	d := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(raw), "."))
	d = strings.TrimPrefix(strings.TrimPrefix(d, "https://"), "http://")
	if i := strings.IndexByte(d, '@'); i >= 0 {
		d = d[i+1:]
	}
	ascii, err := idna.Lookup.ToASCII(d)
	if err != nil || !strings.Contains(ascii, ".") || len(ascii) > 253 {
		return "", api.Invalid(field, "Enter a domain like news.acme.io.")
	}
	for _, label := range strings.Split(ascii, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", api.Invalid(field, "Enter a domain like news.acme.io.")
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
				return "", api.Invalid(field, "Enter a domain like news.acme.io.")
			}
		}
	}
	return ascii, nil
}
