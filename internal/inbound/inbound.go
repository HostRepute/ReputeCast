package inbound

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/mail"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/domains"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/smtp/bouncebox"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service receives mail for workspaces and serves the replies inbox.
type Service struct {
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Audit   *audit.Logger
	Box     *secrets.Box
	Signer  *linktoken.Signer
	Factory *manage.Factory
	// Mail sends the setup's test message (System mail).
	Mail mailer.Sender
	Jobs func() *jobs.Client
	// Base is the instance's public URL (the hook's address).
	Base func() string
	DNS  dnscheck.Resolver
	Log  *slog.Logger
	Now  func() time.Time
	// Audience unsubscribes contacts (mailto: unsubscribes, the inbox's
	// Unsubscribe contact).
	Audience *audience.Service
	// MessageUnsubscribed counts an unsubscribe on an automation email's
	// step (optout.Deps).
	MessageUnsubscribed func(ctx context.Context, q *store.Queries, messageID uuid.UUID, at time.Time) error
	// EndRunsOnReply ends the flows a reply stops (automations), inside
	// the reply's transaction.
	EndRunsOnReply func(ctx context.Context, tx pgx.Tx, workspaceID, automationID, contactID uuid.UUID, at time.Time) (int, error)
	// StoreDMARC keeps a DMARC aggregate report that arrived by email
	// (deliverability), in tx: ErrNotAReport for a file that isn't one,
	// ErrNotOwned (with the report's domain) for a domain the workspace
	// doesn't send from.
	StoreDMARC func(ctx context.Context, tx pgx.Tx, ws *store.Workspace, data []byte) (domain string, err error)
	// ApplyReport applies a feedback-loop report's complaint as an event
	// of the provider that sent the message (ingest), in tx; it returns how
	// many addresses it newly suppressed.
	ApplyReport func(ctx context.Context, tx pgx.Tx, providerID uuid.UUID, source string, receivedAt time.Time, raw []byte, evs []providers.Event) (int, error)
	// OnSuppressed is told when a report suppressed addresses (the list's
	// health is recounted).
	OnSuppressed func(ctx context.Context, workspaceID uuid.UUID)
	// Guard dials mailbox routes' servers (hosts people enter).
	Guard netguard.Policy
	// OpenBox opens a mailbox (bouncebox.Open; tests replace it).
	OpenBox func(ctx context.Context, c bouncebox.Config) (bouncebox.Box, error)

	mu     sync.Mutex
	mx     map[string]mxEntry
	tested map[uuid.UUID]time.Time
}

type mxEntry struct {
	hosts []string
	err   error
	at    time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// Sealed fields of a route name the row and the field.
func secretAD(id uuid.UUID) string   { return "inbound_route:" + id.String() + ":secret" }
func tokenAD(id uuid.UUID) string    { return "inbound_route:" + id.String() + ":hook_token" }
func passwordAD(id uuid.UUID) string { return "inbound_route:" + id.String() + ":mailbox_password" }

// InboundRouteView is Settings › Senders' Receiving mail card.
type InboundRouteView struct {
	// Status is not_set_up until a route is saved, then pending (waiting
	// for the first message), receiving or failing.
	Status       string `json:"status" enum:"not_set_up,pending,receiving,failing"`
	StatusDetail string `json:"statusDetail"`
	Kind         string `json:"kind,omitempty" enum:"provider,mailbox"`
	// Provider is the connected provider a provider route receives
	// through.
	Provider *InboundProviderRef `json:"provider,omitempty"`
	Domain   string              `json:"domain"`
	// Mailbox describes a mailbox route (never its password).
	Mailbox *InboundMailboxView `json:"mailbox,omitempty"`
	Setup   InboundSetupView    `json:"setup"`
	// HookURL is where the provider posts what it receives, for manual
	// setups; only people who can manage domains see it (it carries the
	// route's token).
	HookURL        string             `json:"hookUrl,omitempty"`
	LastReceivedAt *time.Time         `json:"lastReceivedAt,omitempty"`
	LastTest       InboundTestView    `json:"lastTest"`
	Addresses      InboundAddressView `json:"addresses"`
	// Choices are the setup dialog's first step: each connected provider
	// and whether it can receive.
	Choices []InboundProviderChoice `json:"choices"`
	// Suggestion is a domain to receive on (reply.<first verified sending
	// domain>).
	Suggestion string `json:"suggestion"`
	// ReplyHandling is the Senders defaults' reply handling ("forward" or
	// "inbox").
	ReplyHandling string   `json:"replyHandling" enum:"forward,inbox"`
	CanManage     bool     `json:"canManage"`
	Helpers       []string `json:"helpers"`
}

// InboundProviderRef names a connected provider.
type InboundProviderRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Adapter string `json:"adapter"`
}

// InboundMailboxView is a mailbox route's settings, without the password.
type InboundMailboxView struct {
	Address     string     `json:"address"`
	Protocol    string     `json:"protocol" enum:"imap,pop3"`
	Host        string     `json:"host"`
	Port        int        `json:"port"`
	Security    string     `json:"security" enum:"tls,starttls"`
	Username    string     `json:"username"`
	PasswordSet bool       `json:"passwordSet"`
	LastPolled  *time.Time `json:"lastPolledAt,omitempty"`
}

// InboundSetupView is what setting the route up takes.
type InboundSetupView struct {
	Automatic bool                `json:"automatic"`
	MX        []InboundMXRecord   `json:"mx"`
	Steps     []string            `json:"steps"`
	Settings  []InboundSetupField `json:"settings"`
}

// InboundMXRecord is one MX record the route's domain needs, with what DNS
// says.
type InboundMXRecord struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Value  string `json:"value"`
	Status string `json:"status" enum:"found,missing,wrong_value,unknown"`
}

// InboundSetupField is a value a provider's route needs (an SNS topic ARN).
type InboundSetupField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Help     string `json:"help,omitempty"`
	Required bool   `json:"required"`
	Value    string `json:"value"`
}

// InboundTestView is the last test message.
type InboundTestView struct {
	SentAt     *time.Time `json:"sentAt,omitempty"`
	ReceivedAt *time.Time `json:"receivedAt,omitempty"`
}

// InboundAddressView are the fixed addresses on the route's domain.
type InboundAddressView struct {
	DMARC string `json:"dmarc"`
	FBL   string `json:"fbl"`
}

// InboundProviderChoice is a connected provider in the setup dialog.
type InboundProviderChoice struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Adapter string `json:"adapter"`
	// Inbound is how it receives: automatic, manual, or "" when it can't
	// (Reason says so).
	Inbound string `json:"inbound" enum:",automatic,manual"`
	Reason  string `json:"reason,omitempty"`
}

// InboundRouteInput saves the route.
type InboundRouteInput struct {
	Kind       string `json:"kind" enum:"provider,mailbox"`
	ProviderID string `json:"providerId,omitempty" format:"uuid"`
	// Domain is the provider route's domain (reply.news.acme.io).
	Domain   string               `json:"domain,omitempty" maxLength:"253"`
	Mailbox  *InboundMailboxInput `json:"mailbox,omitempty"`
	Settings map[string]string    `json:"settings,omitempty"`
}

// InboundMailboxInput is a mailbox route's settings. A blank password
// keeps the saved one while the server and user name stay the same.
type InboundMailboxInput struct {
	Address  string `json:"address" maxLength:"254"`
	Protocol string `json:"protocol" enum:"imap,pop3"`
	Host     string `json:"host" maxLength:"253"`
	Port     int    `json:"port,omitempty" minimum:"0" maximum:"65535"`
	Security string `json:"security" enum:"tls,starttls"`
	Username string `json:"username" maxLength:"254"`
	Password string `json:"password,omitempty" maxLength:"500"`
}

// InboundTestResult is the test message just sent.
type InboundTestResult struct {
	SentAt time.Time `json:"sentAt"`
	To     string    `json:"to"`
}

// mailboxSettings is inbound_routes.mailbox_settings.
type mailboxSettings struct {
	Address  string `json:"address"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	Security string `json:"security"`
	Username string `json:"username"`
}

// boxOf is where a route receives.
func boxOf(r store.InboundRoute) Box { return Box{Domain: r.Domain, Local: r.MailboxLocal} }

// Route returns the workspace's route, or pgx.ErrNoRows.
func (s *Service) Route(ctx context.Context, wsID uuid.UUID) (store.InboundRoute, error) {
	return s.Q.InboundRouteGet(ctx, wsID)
}

// adapterRoute is the route as its adapter sees it.
func (s *Service) adapterRoute(r store.InboundRoute) (providers.InboundRoute, error) {
	secret, err := s.Box.OpenString(r.SecretSealed, secretAD(r.ID))
	if err != nil {
		return providers.InboundRoute{}, err
	}
	token, err := s.Box.OpenString(r.HookTokenSealed, tokenAD(r.ID))
	if err != nil {
		return providers.InboundRoute{}, err
	}
	settings := map[string]string{}
	_ = json.Unmarshal(r.Settings, &settings)
	return providers.InboundRoute{ID: r.ID, Domain: r.Domain, Settings: settings, Secret: secret, URL: s.hookURL(token)}, nil
}

func (s *Service) hookURL(token string) string {
	base := ""
	if s.Base != nil {
		base = s.Base()
	}
	return strings.TrimRight(base, "/") + "/hooks/inbound/" + token
}

// receiver opens the route's provider as an inbound receiver.
func (s *Service) receiver(ctx context.Context, r store.InboundRoute) (providers.InboundReceiver, store.Provider, error) {
	if r.ProviderID == nil {
		return nil, store.Provider{}, errNoProvider
	}
	p, err := s.Q.ProvGetAny(ctx, *r.ProviderID)
	if err != nil {
		return nil, p, err
	}
	prov, err := s.Factory.Open(ctx, p)
	if err != nil {
		return nil, p, err
	}
	rc, ok := prov.(providers.InboundReceiver)
	if !ok {
		return nil, p, errUnsupported
	}
	return rc, p, nil
}

var (
	errUnsupported = errors.New("the provider doesn't receive mail")
	errNoProvider  = errors.New("the route's provider was removed")
)

// sendingDefaults is the part of workspaces.sending_defaults receiving reads.
type sendingDefaults struct {
	ReplyHandling string `json:"reply_handling"`
}

func replyHandling(raw json.RawMessage) string {
	var d sendingDefaults
	_ = json.Unmarshal(raw, &d)
	if d.ReplyHandling == "inbox" {
		return "inbox"
	}
	return "forward"
}

// View is the Receiving mail card for the caller.
func (s *Service) View(ctx context.Context, ws *store.Workspace) (InboundRouteView, error) {
	l := i18n.LangFrom(ctx)
	p := api.PrincipalFrom(ctx)
	can := p.Can(authz.DomainsManage)
	out := InboundRouteView{Status: "not_set_up", Setup: InboundSetupView{MX: []InboundMXRecord{}, Steps: []string{}, Settings: []InboundSetupField{}},
		Choices: []InboundProviderChoice{}, ReplyHandling: replyHandling(ws.SendingDefaults), CanManage: can, Helpers: []string{}}
	if !can {
		if h := access.Helpers(ctx, s.Q, ws.ID, authz.DomainsManage); h != nil {
			out.Helpers = h
		}
	}
	doms, err := s.Q.DomList(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, d := range doms {
		if verifiedDomain(d) {
			out.Suggestion = "reply." + d.Domain
			break
		}
	}
	provs, err := s.Q.ProvListAllActive(ctx)
	if err != nil {
		return out, err
	}
	for _, pr := range provs {
		if pr.WorkspaceID != ws.ID {
			continue
		}
		c := InboundProviderChoice{ID: pr.ID.String(), Name: pr.Name, Adapter: pr.Adapter}
		if d, ok := providers.Lookup(pr.Adapter); ok {
			c.Inbound = d.Capabilities.Inbound
		}
		if c.Inbound == "" {
			c.Reason = text(l, "choice.cantReceive", i18n.Args{"provider": pr.Name})
		}
		out.Choices = append(out.Choices, c)
	}
	r, err := s.Q.InboundRouteGet(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.Status, out.Kind, out.Domain, out.LastReceivedAt = r.Status, r.Kind, r.Domain, r.LastReceivedAt
	out.StatusDetail = detailIn(r.StatusDetail, l)
	out.LastTest = InboundTestView{SentAt: r.TestSentAt, ReceivedAt: r.TestReceivedAt}
	b := boxOf(r)
	out.Addresses = InboundAddressView{DMARC: b.Fixed(LocalDMARC), FBL: b.Fixed(LocalFBL)}
	if r.Kind == "mailbox" {
		var m mailboxSettings
		_ = json.Unmarshal(r.MailboxSettings, &m)
		out.Mailbox = &InboundMailboxView{Address: m.Address, Protocol: m.Protocol, Host: m.Host, Port: m.Port, Security: m.Security, Username: m.Username,
			PasswordSet: len(r.MailboxPasswordSealed) > 0, LastPolled: r.LastPolledAt}
		return out, nil
	}
	if r.ProviderID == nil {
		return out, nil
	}
	pr, err := s.Q.ProvGetAny(ctx, *r.ProviderID)
	if err != nil {
		return out, nil // the provider was removed: the card says so through its status
	}
	out.Provider = &InboundProviderRef{ID: pr.ID.String(), Name: pr.Name, Adapter: pr.Adapter}
	rc, _, err := s.receiver(ctx, r)
	if err != nil {
		return out, nil
	}
	ar, err := s.adapterRoute(r)
	if err != nil {
		return out, err
	}
	setup := rc.InboundSetup(ar)
	out.Setup.Automatic = setup.Automatic
	values := ar.Settings
	for _, f := range setup.Settings {
		f = f.In(l)
		out.Setup.Settings = append(out.Setup.Settings, InboundSetupField{Key: f.Key, Label: f.Label, Help: f.Help, Required: f.Required, Value: values[f.Key]})
	}
	if can {
		// The steps and the hook's address carry the route's token.
		for _, st := range setup.Steps {
			out.Setup.Steps = append(out.Setup.Steps, st.In(l))
		}
		if !setup.Automatic {
			out.HookURL = ar.URL
		}
	}
	out.Setup.MX = s.checkMX(ctx, r.Domain, setup.MX, r.Status != "receiving")
	return out, nil
}

func verifiedDomain(d store.SendingDomain) bool {
	return d.Status == "verified" || d.Status == "partial"
}

// checkMX looks the route's domain's MX records up (cached a minute) and
// says, for each record the provider needs, whether DNS has it.
func (s *Service) checkMX(ctx context.Context, domain string, want []providers.DNSRecord, look bool) []InboundMXRecord {
	out := make([]InboundMXRecord, 0, len(want))
	var hosts []string
	var lerr error
	if look {
		hosts, lerr = s.lookupMX(ctx, domain)
	}
	for _, w := range want {
		rec := InboundMXRecord{Type: "MX", Name: w.Name, Value: w.Value, Status: "unknown"}
		switch {
		case !look:
			rec.Status = "found"
		case lerr != nil && !errors.Is(lerr, dnscheck.ErrNotFound):
		case hasMX(hosts, w.Value):
			rec.Status = "found"
		case len(hosts) == 0:
			rec.Status = "missing"
		default:
			rec.Status = "wrong_value"
		}
		out = append(out, rec)
	}
	return out
}

func (s *Service) lookupMX(ctx context.Context, domain string) ([]string, error) {
	key := strings.ToLower(domain)
	s.mu.Lock()
	if e, ok := s.mx[key]; ok && s.now().Sub(e.at) < time.Minute {
		s.mu.Unlock()
		return e.hosts, e.err
	}
	s.mu.Unlock()
	if s.DNS == nil {
		return nil, errors.New("no resolver")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	mxs, err := s.DNS.MX(ctx, key)
	hosts := make([]string, 0, len(mxs))
	for _, m := range mxs {
		hosts = append(hosts, strings.ToLower(strings.TrimSuffix(m.Host, ".")))
	}
	s.mu.Lock()
	if s.mx == nil {
		s.mx = map[string]mxEntry{}
	}
	if len(s.mx) > 1000 {
		s.mx = map[string]mxEntry{}
	}
	s.mx[key] = mxEntry{hosts: hosts, err: err, at: s.now()}
	s.mu.Unlock()
	return hosts, err
}

// hasMX reports whether the MX hosts include the record's host ("10
// inbound.postmarkapp.com" or just the host).
func hasMX(hosts []string, value string) bool {
	f := strings.Fields(value)
	if len(f) == 0 {
		return false
	}
	want := strings.ToLower(strings.TrimSuffix(f[len(f)-1], "."))
	return slices.Contains(hosts, want)
}

// Save sets up or changes the workspace's route.
func (s *Service) Save(ctx context.Context, ws *store.Workspace, in InboundRouteInput) (InboundRouteView, error) {
	l := i18n.LangFrom(ctx)
	cur, err := s.Q.InboundRouteGet(ctx, ws.ID)
	exists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return InboundRouteView{}, err
	}
	id := newID()
	if exists {
		id = cur.ID
	}
	p := store.InboundRouteUpsertParams{ID: id, WorkspaceID: ws.ID, Kind: in.Kind, MailboxSettings: json.RawMessage(`{}`), Settings: json.RawMessage(`{}`), Now: s.now()}
	if pr := api.PrincipalFrom(ctx); pr != nil && pr.Type == authz.ActorUser {
		u := pr.UserID
		p.CreatedBy = &u
	}
	var rc providers.InboundReceiver
	var prov store.Provider
	switch in.Kind {
	case "provider":
		pid, err := uuid.Parse(in.ProviderID)
		if err != nil {
			return InboundRouteView{}, api.Invalid("body.providerId", say(ctx, "problem.pickProvider", nil))
		}
		prov, err = s.Q.ProvGetAny(ctx, pid)
		if err != nil || prov.WorkspaceID != ws.ID || prov.DisconnectedAt != nil {
			return InboundRouteView{}, api.Invalid("body.providerId", say(ctx, "problem.pickProvider", nil))
		}
		opened, err := s.Factory.Open(ctx, prov)
		if err != nil {
			return InboundRouteView{}, err
		}
		var ok bool
		if rc, ok = opened.(providers.InboundReceiver); !ok {
			return InboundRouteView{}, api.Err(409, "inbound.unsupported", say(ctx, "problem.unsupported", i18n.Args{"provider": prov.Name}))
		}
		domain, err := domains.NormalizeDomain(ctx, "body.domain", in.Domain)
		if err != nil {
			return InboundRouteView{}, err
		}
		if err := s.checkDomain(ctx, ws, domain); err != nil {
			return InboundRouteView{}, err
		}
		taken, err := s.Q.InboundRouteDomainTaken(ctx, store.InboundRouteDomainTakenParams{Domain: domain, WorkspaceID: ws.ID})
		if err != nil {
			return InboundRouteView{}, err
		}
		if taken {
			return InboundRouteView{}, api.Err(409, "inbound.domain_taken", say(ctx, "problem.domainTaken", i18n.Args{"domain": domain}))
		}
		p.ProviderID, p.Domain = &prov.ID, domain
		// The adapter's settings, only the keys it asks for.
		probe := providers.InboundRoute{ID: id, Domain: domain, Settings: map[string]string{}}
		settings := map[string]string{}
		for _, f := range rc.InboundSetup(probe).Settings {
			v := strings.TrimSpace(in.Settings[f.Key])
			if v == "" && f.Required {
				return InboundRouteView{}, api.Invalid("body.settings."+f.Key, say(ctx, "problem.settingRequired", i18n.Args{"label": f.In(l).Label}))
			}
			if len(v) > 500 {
				return InboundRouteView{}, api.Invalid("body.settings."+f.Key, say(ctx, "problem.settingLong", i18n.Args{"label": f.In(l).Label}))
			}
			if v != "" {
				settings[f.Key] = v
			}
		}
		p.Settings, _ = json.Marshal(settings)
		if err := s.checkMXFree(ctx, domain, rc.InboundSetup(probe).MX); err != nil {
			return InboundRouteView{}, err
		}
	case "mailbox":
		m := in.Mailbox
		if m == nil {
			return InboundRouteView{}, api.Invalid("body.mailbox", say(ctx, "problem.mailboxMissing", nil))
		}
		addr, err := mail.ParseAddress(strings.TrimSpace(m.Address))
		if err != nil || !strings.Contains(addr.Address, "@") {
			return InboundRouteView{}, api.Invalid("body.mailbox.address", say(ctx, "problem.mailboxAddress", nil))
		}
		local, domain, _ := strings.Cut(strings.ToLower(addr.Address), "@")
		if strings.Contains(local, "+") {
			return InboundRouteView{}, api.Invalid("body.mailbox.address", say(ctx, "problem.mailboxPlus", nil))
		}
		if len(local) > MaxMailboxLocal {
			return InboundRouteView{}, api.Invalid("body.mailbox.address", say(ctx, "problem.mailboxLocal", i18n.Args{"max": MaxMailboxLocal}))
		}
		host := strings.ToLower(strings.TrimSpace(m.Host))
		if host == "" || strings.ContainsAny(host, " /:@") {
			return InboundRouteView{}, api.Invalid("body.mailbox.host", say(ctx, "problem.mailboxHost", nil))
		}
		if strings.TrimSpace(m.Username) == "" {
			return InboundRouteView{}, api.Invalid("body.mailbox.username", say(ctx, "problem.mailboxUser", nil))
		}
		ms := mailboxSettings{Address: local + "@" + domain, Protocol: m.Protocol, Host: host, Port: m.Port, Security: m.Security, Username: strings.TrimSpace(m.Username)}
		p.MailboxSettings, _ = json.Marshal(ms)
		p.MailboxLocal, p.Domain = local, domain
		switch {
		case m.Password != "":
			if p.MailboxPasswordSealed, err = s.Box.SealString(m.Password, passwordAD(id)); err != nil {
				return InboundRouteView{}, err
			}
		case exists && cur.Kind == "mailbox" && len(cur.MailboxPasswordSealed) > 0:
			// The saved password goes only to the server and user it was
			// entered for (architecture.md: secrets follow their address).
			var old mailboxSettings
			_ = json.Unmarshal(cur.MailboxSettings, &old)
			if old.Host != ms.Host || old.Username != ms.Username {
				return InboundRouteView{}, api.Invalid("body.mailbox.password", say(ctx, "problem.passwordAgain", nil))
			}
			p.MailboxPasswordSealed = cur.MailboxPasswordSealed
		default:
			return InboundRouteView{}, api.Invalid("body.mailbox.password", say(ctx, "problem.passwordRequired", nil))
		}
	default:
		return InboundRouteView{}, api.Invalid("body.kind", say(ctx, "problem.pickKind", nil))
	}
	if exists {
		p.SecretSealed, p.HookTokenSealed, p.HookTokenHash = cur.SecretSealed, cur.HookTokenSealed, cur.HookTokenHash
	} else {
		secret, _ := authn.NewToken()
		if p.SecretSealed, err = s.Box.SealString(secret, secretAD(id)); err != nil {
			return InboundRouteView{}, err
		}
		token, hash := authn.NewToken()
		if p.HookTokenSealed, err = s.Box.SealString(token, tokenAD(id)); err != nil {
			return InboundRouteView{}, err
		}
		p.HookTokenHash = hash
	}
	var saved store.InboundRoute
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if saved, err = q.InboundRouteUpsert(ctx, p); err != nil {
			return err
		}
		var before map[string]any
		if exists {
			before = routeSnapshot(cur)
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "inbound.route_saved", TargetType: "inbound_route", TargetID: saved.ID.String(),
			TargetName: saved.Domain, Before: before, After: routeSnapshot(saved)})
	})
	if err != nil {
		return InboundRouteView{}, err
	}
	// The old provider's registration goes when the route moved.
	if exists && cur.Kind == "provider" && (saved.Kind != "provider" || !sameRef(cur.ProviderID, saved.ProviderID) || !strings.EqualFold(cur.Domain, saved.Domain)) {
		s.unregister(ctx, cur)
	}
	if rc != nil {
		s.register(ctx, saved, prov, rc)
	}
	return s.View(ctx, ws)
}

func sameRef(a, b *uuid.UUID) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

// routeSnapshot is a route as the audit log keeps it: never its secrets.
func routeSnapshot(r store.InboundRoute) map[string]any {
	out := map[string]any{"kind": r.Kind, "domain": r.Domain}
	if r.ProviderID != nil {
		out["provider_id"] = r.ProviderID.String()
	}
	if r.Kind == "mailbox" {
		var m mailboxSettings
		_ = json.Unmarshal(r.MailboxSettings, &m)
		out["mailbox"] = map[string]any{"address": m.Address, "protocol": m.Protocol, "host": m.Host, "port": m.Port, "security": m.Security, "username": m.Username}
	}
	var settings map[string]string
	if json.Unmarshal(r.Settings, &settings) == nil && len(settings) > 0 {
		out["settings"] = settings
	}
	return out
}

// checkDomain holds a provider route's domain to the workspace's verified
// sending domains: the domain or a subdomain of one.
func (s *Service) checkDomain(ctx context.Context, ws *store.Workspace, domain string) error {
	doms, err := s.Q.DomList(ctx, ws.ID)
	if err != nil {
		return err
	}
	for _, d := range doms {
		if verifiedDomain(d) && (domain == d.Domain || strings.HasSuffix(domain, "."+d.Domain)) {
			return nil
		}
	}
	return api.Invalid("body.domain", say(ctx, "problem.domainNotOwned", i18n.Args{"domain": domain}))
}

// checkMXFree refuses a domain that already receives mail elsewhere: its
// MX records would have to change, and that mail would stop.
func (s *Service) checkMXFree(ctx context.Context, domain string, want []providers.DNSRecord) error {
	hosts, err := s.lookupMX(ctx, domain)
	if err != nil || len(hosts) == 0 {
		return nil // no MX yet, or DNS can't tell: setup shows the records
	}
	for _, h := range hosts {
		ours := false
		for _, w := range want {
			if hasMX([]string{h}, w.Value) {
				ours = true
			}
		}
		if !ours {
			return api.Invalid("body.domain", say(ctx, "problem.domainReceives", i18n.Args{"domain": domain, "host": h}))
		}
	}
	return nil
}

// register points an automatic provider's receiving at the hook and keeps
// how it went; a demo provider never calls its API.
func (s *Service) register(ctx context.Context, r store.InboundRoute, p store.Provider, rc providers.InboundReceiver) {
	reg, ok := rc.(providers.InboundRegistrar)
	if !ok {
		return
	}
	now := s.now()
	detail := ""
	if !manage.IsDemo(p) {
		ar, err := s.adapterRoute(r)
		if err == nil {
			err = reg.RegisterInbound(ctx, ar, ar.URL)
		}
		if err != nil {
			s.Log.WarnContext(ctx, "register inbound", "route", r.ID, "err", err)
			detail = keepDetail(msgs.M("status.registerFailed").Keep(i18n.Args{"provider": p.Name, "error": providers.MessageIn(err, i18n.Default)}))
		}
	}
	at := &now
	if detail != "" {
		at = nil
	}
	if err := s.Q.InboundRouteRegistered(ctx, store.InboundRouteRegisteredParams{ID: r.ID, RegisteredAt: at, StatusDetail: detail, Now: now}); err != nil {
		s.Log.WarnContext(ctx, "save inbound registration", "route", r.ID, "err", err)
	}
}

// unregister removes what register added (best effort: the provider may
// already be gone).
func (s *Service) unregister(ctx context.Context, r store.InboundRoute) {
	if r.Kind != "provider" || r.ProviderID == nil {
		return
	}
	rc, p, err := s.receiver(ctx, r)
	if err != nil || manage.IsDemo(p) {
		return
	}
	reg, ok := rc.(providers.InboundRegistrar)
	if !ok {
		return
	}
	ar, err := s.adapterRoute(r)
	if err == nil {
		err = reg.UnregisterInbound(ctx, ar, ar.URL)
	}
	if err != nil {
		s.Log.WarnContext(ctx, "unregister inbound", "route", r.ID, "err", err)
	}
}

// Remove stops receiving: the route goes (replies stay), and reply
// handling goes back to forwarding.
func (s *Service) Remove(ctx context.Context, ws *store.Workspace) error {
	var gone store.InboundRoute
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if gone, err = q.InboundRouteDelete(ctx, ws.ID); err != nil {
			return api.MapError(err, "Receiving route")
		}
		if replyHandling(ws.SendingDefaults) == "inbox" {
			if _, err := tx.Exec(ctx, `UPDATE workspaces SET sending_defaults = jsonb_set(coalesce(sending_defaults, '{}'::jsonb), '{reply_handling}', '"forward"'), updated_at = $2 WHERE id = $1`,
				ws.ID, s.now()); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "inbound.route_removed", TargetType: "inbound_route", TargetID: gone.ID.String(),
			TargetName: gone.Domain, Before: routeSnapshot(gone)})
	})
	if err != nil {
		return err
	}
	s.unregister(ctx, gone)
	return nil
}

// testEvery is how often a workspace may send a test message.
const testEvery = 30 * time.Second

// SendTest sends a System-stream message to the route's test address.
func (s *Service) SendTest(ctx context.Context, ws *store.Workspace) (InboundTestResult, error) {
	r, err := s.Q.InboundRouteGet(ctx, ws.ID)
	if err != nil {
		return InboundTestResult{}, api.MapError(err, "Receiving route")
	}
	now := s.now()
	s.mu.Lock()
	if s.tested == nil {
		s.tested = map[uuid.UUID]time.Time{}
	}
	if last, ok := s.tested[ws.ID]; ok && now.Sub(last) < testEvery {
		s.mu.Unlock()
		wait := int((testEvery - now.Sub(last)).Seconds()) + 1
		return InboundTestResult{}, api.Err(409, api.CodeRateLimited, say(ctx, "problem.testWait", i18n.Args{"seconds": wait})).WithMeta("retryAfter", wait)
	}
	s.tested[ws.ID] = now
	s.mu.Unlock()
	to := boxOf(r).Address(s.Signer, LetterTest, ws.ID, newID())
	l := i18n.LangFrom(ctx)
	subject := text(l, "test.subject", nil)
	body := text(l, "test.body", i18n.Args{"domain": r.Domain})
	demo := false
	if r.ProviderID != nil {
		if p, err := s.Q.ProvGetAny(ctx, *r.ProviderID); err == nil {
			demo = manage.IsDemo(p)
		}
	}
	if !demo {
		if s.Mail == nil {
			return InboundTestResult{}, api.Conflict(say(ctx, "problem.noSystemMail", nil))
		}
		if _, err := s.Mail.SendSystem(ctx, mailer.Message{WorkspaceID: &ws.ID, To: to, Subject: subject, Text: body, Kind: "inbound_test", Lang: l}); err != nil {
			if errors.Is(err, mailer.ErrNoSystemProvider) {
				return InboundTestResult{}, api.Conflict(say(ctx, "problem.noSystemMail", nil))
			}
			return InboundTestResult{}, err
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.InboundRouteTestSent(ctx, store.InboundRouteTestSentParams{ID: r.ID, At: now}); err != nil {
			return err
		}
		if demo {
			// The demo's provider receives in process: the test arrives as
			// a real one would, through the job.
			if err := s.storeRaw(ctx, tx, r, "test", "message/rfc822", nil, testMessage(to, subject, body, now), true); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "inbound.test_sent", TargetType: "inbound_route", TargetID: r.ID.String(),
			TargetName: r.Domain, Metadata: map[string]any{"to": to}})
	})
	if err != nil {
		return InboundTestResult{}, err
	}
	return InboundTestResult{SentAt: now, To: to}, nil
}

// testMessage is the test message as a mail server would deliver it.
func testMessage(to, subject, body string, at time.Time) []byte {
	var b strings.Builder
	b.WriteString("From: ReputeCast <no-reply@reputecast.invalid>\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + mimeWord(subject) + "\r\n")
	b.WriteString("Date: " + at.UTC().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("Message-ID: <test-" + newID().String() + "@reputecast.invalid>\r\n")
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(body + "\r\n")
	return []byte(b.String())
}

// mimeWord encodes a header value as an RFC 2047 word when it isn't ASCII.
func mimeWord(s string) string { return mime.QEncoding.Encode("utf-8", s) }

// storeRaw keeps what arrived and queues its processing, in tx.
func (s *Service) storeRaw(ctx context.Context, tx pgx.Tx, r store.InboundRoute, source, contentType string, headers json.RawMessage, body []byte, ok bool) error {
	id := newID()
	if headers == nil {
		headers = json.RawMessage(`{}`)
	}
	if err := store.New(tx).InboundRawInsert(ctx, store.InboundRawInsertParams{ID: id, WorkspaceID: r.WorkspaceID, RouteID: r.ID, Source: source,
		ReceivedAt: s.now(), ContentType: contentType, Headers: headers, Body: body, SignatureOk: ok}); err != nil {
		return err
	}
	if !ok || s.Jobs == nil {
		return nil
	}
	if c := s.Jobs(); c != nil {
		if _, err := c.InsertTx(ctx, tx, ProcessArgs{RawID: id}, nil); err != nil {
			return err
		}
	}
	return nil
}
