package domains

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// ProviderSetupNote explains, for a provider whose records ReputeCast can't
// fetch, what the person does in the provider's own dashboard.
type ProviderSetupNote struct {
	ProviderID string `json:"providerId"`
	Provider   string `json:"provider"`
	Text       string `json:"text"`
}

// providerRecords asks a provider which records a domain needs. Providers
// with a domain API (SES, Postmark with an account token) add the domain
// and return their records; for the others ReputeCast only knows what the
// sending domain's SPF record must include. Demo providers never call
// their API: their records are derived from the domain name.
func (s *Service) providerRecords(ctx context.Context, domain string, p store.Provider) ([]Record, string, error) {
	var recs []Record
	note := ""
	switch {
	case manage.IsDemo(p):
		recs = demoProviderRecords(domain, p)
	default:
		if s.Factory == nil {
			return nil, "", errors.New("domains: no adapter factory")
		}
		adapter, err := s.Factory.Open(ctx, p)
		if err != nil {
			return nil, "", err
		}
		dm, ok := adapter.(providers.DomainManager)
		if ok {
			cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			fresh, err := dm.AddDomain(cctx, domain)
			cancel()
			switch {
			case errors.Is(err, providers.ErrNotSupported):
				ok = false
			case err != nil:
				return nil, "", api.Err(http.StatusBadGateway, api.CodeProviderUnavailable, p.Name+" didn't add "+domain+": "+err.Error())
			}
			for _, f := range fresh {
				recs = append(recs, Record{Purpose: f.Purpose, Type: strings.ToUpper(f.Type), Name: strings.ToLower(strings.TrimSuffix(f.Name, ".")), Value: f.Value,
					Status: StatusPending, ProviderStatus: providerWord(f.Status), Source: "provider"})
			}
		}
		if !ok {
			note = "Add " + domain + " in " + p.Name + " and publish the DKIM record it shows there. ReputeCast checks SPF and DMARC here and the rest on the first send."
		}
	}
	// The sending domain's SPF must authorize the provider too (merged into
	// any record that's already there).
	mechs := []string{}
	for _, r := range recs {
		if r.Purpose == PurposeSPF {
			for _, m := range SPFMechanisms(r.Value) {
				if !contains(mechs, m) {
					mechs = append(mechs, m)
				}
			}
		}
	}
	if len(mechs) == 0 {
		if h := providerHost(p); h != "" {
			mechs = append(mechs, "a:"+h)
		}
	}
	if len(mechs) > 0 {
		recs = append(recs, Record{Purpose: PurposeSPF, Type: "TXT", Name: domain, Mechanisms: mechs, Value: MergeSPF("", mechs), Status: StatusPending, Source: "reputecast"})
	}
	return recs, note, nil
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// providerHost is the server a self-hosted provider sends from (SMTP host,
// Postal server), for its SPF "a:" mechanism.
func providerHost(p store.Provider) string {
	var c struct {
		Host      string `json:"host"`
		ServerURL string `json:"server_url"`
	}
	_ = json.Unmarshal(p.Config, &c)
	if c.Host != "" {
		return strings.ToLower(c.Host)
	}
	if u, err := url.Parse(c.ServerURL); err == nil && u.Hostname() != "" {
		return strings.ToLower(u.Hostname())
	}
	return ""
}

// demoProviderRecords are the records a demo provider "needs": the same
// shapes as the real adapters, with tokens derived from the names.
func demoProviderRecords(domain string, p store.Provider) []Record {
	token := func(n int) string {
		sum := sha256.Sum256([]byte(p.ID.String() + "|" + domain + "|" + string(rune('a'+n))))
		return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])[:32])
	}
	rec := func(purpose, typ, name, value string) Record {
		return Record{Purpose: purpose, Type: typ, Name: name, Value: value, Status: StatusPending, Source: "provider"}
	}
	switch p.Adapter {
	case "ses":
		var region struct {
			Region string `json:"region"`
		}
		_ = json.Unmarshal(p.Config, &region)
		if region.Region == "" {
			region.Region = "us-east-1"
		}
		out := []Record{}
		for i := range 3 {
			t := token(i)
			out = append(out, rec(PurposeDKIM, "CNAME", t+"._domainkey."+domain, t+".dkim.amazonses.com"))
		}
		return append(out,
			rec(PurposeReturnPath, "MX", "bounce."+domain, "10 feedback-smtp."+region.Region+".amazonses.com"),
			rec(PurposeSPF, "TXT", "bounce."+domain, "v=spf1 include:amazonses.com ~all"))
	case "postmark":
		return []Record{
			rec(PurposeDKIM, "TXT", time.Now().UTC().Format("20060102")+"pm._domainkey."+domain, "k=rsa; p="+demoKey),
			rec(PurposeReturnPath, "CNAME", "pm-bounces."+domain, "pm.mtasv.net"),
		}
	case "postal":
		host := providerHost(p)
		return []Record{
			rec(PurposeDKIM, "TXT", "postal-"+token(0)[:6]+"._domainkey."+domain, "v=DKIM1; t=s; h=sha256; p="+demoKey),
			rec(PurposeReturnPath, "CNAME", "rp."+domain, "rp."+host),
		}
	}
	t := token(0)
	return []Record{rec(PurposeDKIM, "CNAME", t[:12]+"._domainkey."+domain, t[:12]+".dkim."+p.Adapter+".example")}
}

// demoKey is a sample RSA public key (2048-bit length) for demo DKIM records.
const demoKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwH9Uo4wVvKcV3B8eF4o3w5pD7fJ2cR1kQ9aL0mN8bT6yU5sZ3xG7hJ1vP4qW2eR9tY6uI3oP0aS5dF8gH2jK4lZ6xC1vB7nM9qE3wR5tY8uI0oP2aS4dF6gH8jK1lZ3xC5vB7nM9qE1wR3tY5uI7oP9aS2dF4gH6jK8lZ0xC2vB4nM6qE8wR0tY2uI4oP6aS8dF0gH2jK4lZ6xC8vB0nM2qE4wR6tY8uI0oP2aS4dF6gH8jK1lZ3xC5vB7nM9qE1wR3tY5uI7oP9aS2dF4gH6jK8lZ0xC2vB4nM6qE8wR0tY2uI4oP6aS8dF0gH2jK4lZ6xC8vB0nM2qE4wR6tY8uIDAQAB"

// AttachProvider stores the records a provider needs for a domain (from
// its domain API when it has one), adds the domain-wide DMARC record when
// missing and checks everything live. It returns a note when the person has
// to add the domain in the provider's dashboard themselves.
func (s *Service) AttachProvider(ctx context.Context, ws *store.Workspace, domainID, providerID uuid.UUID) (string, error) {
	d, err := s.Q.DomGet(ctx, store.DomGetParams{WorkspaceID: ws.ID, ID: domainID})
	if err != nil {
		return "", api.MapError(err, "Sending domain")
	}
	p, err := s.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: ws.ID, ID: providerID})
	if err != nil {
		return "", api.MapError(err, "Provider")
	}
	fresh, note, err := s.providerRecords(ctx, d.Domain, p)
	if err != nil {
		return "", err
	}
	rua, _ := s.Q.DomDMARCReports(ctx, ws.ID)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.DomGetAny(ctx, d.ID)
		if err != nil {
			return err
		}
		recs := ReplaceProviderRecords(DecodeRecords(cur.DnsRecords), p.ID.String(), fresh)
		recs = ensureDMARC(recs, d.Domain, rua)
		if err := q.DomUpdateRecords(ctx, store.DomUpdateRecordsParams{ID: d.ID, DnsRecords: EncodeRecords(recs), UpdatedAt: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "domain.provider_attach", TargetType: "sending_domain", TargetID: d.ID.String(), TargetName: d.Domain,
			After: map[string]any{"provider": p.Name, "records": len(fresh)}})
	})
	if err != nil {
		return "", err
	}
	cur, err := s.Q.DomGetAny(ctx, d.ID)
	if err != nil {
		return "", err
	}
	if _, err := s.Check(ctx, cur); err != nil {
		s.log().WarnContext(ctx, "check after attaching a provider", "domain", d.Domain, "err", err)
	}
	return note, nil
}

// DomainInput adds a sending domain.
type DomainInput struct {
	Domain string `json:"domain" minLength:"3" maxLength:"253" doc:"The domain senders use, e.g. news.acme.io"`
	Note   string `json:"note,omitempty" maxLength:"120" doc:"What it is for, e.g. primary sending domain"`
}

// CreateDomain adds a sending domain and asks every connected provider for
// the records it needs.
func (s *Service) CreateDomain(ctx context.Context, ws *store.Workspace, in DomainInput) (store.SendingDomain, []ProviderSetupNote, error) {
	name, err := NormalizeDomain("body.domain", in.Domain)
	if err != nil {
		return store.SendingDomain{}, nil, err
	}
	if _, err := s.Q.DomGetByName(ctx, store.DomGetByNameParams{WorkspaceID: ws.ID, Domain: name}); err == nil {
		return store.SendingDomain{}, nil, api.Conflict(name + " is already a sending domain of this workspace.")
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return store.SendingDomain{}, nil, err
	}
	recs := []Record{}
	notes := []ProviderSetupNote{}
	for _, p := range provs {
		fresh, note, err := s.providerRecords(ctx, name, p)
		if err != nil {
			// One provider failing doesn't stop the domain: its records can
			// be fetched again from the domain page.
			notes = append(notes, ProviderSetupNote{ProviderID: p.ID.String(), Provider: p.Name, Text: err.Error()})
			continue
		}
		if note != "" {
			notes = append(notes, ProviderSetupNote{ProviderID: p.ID.String(), Provider: p.Name, Text: note})
		}
		recs = ReplaceProviderRecords(recs, p.ID.String(), fresh)
	}
	rua, _ := s.Q.DomDMARCReports(ctx, ws.ID)
	recs = ensureDMARC(recs, name, rua)
	id := newID()
	var by *uuid.UUID
	if pr := api.PrincipalFrom(ctx); pr != nil && pr.UserID != uuid.Nil {
		u := pr.UserID
		by = &u
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.DomCreate(ctx, store.DomCreateParams{ID: id, WorkspaceID: ws.ID, Domain: name, Note: strings.TrimSpace(in.Note), DnsRecords: EncodeRecords(recs),
			CreatedBy: by, CreatedAt: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "domain.create", TargetType: "sending_domain", TargetID: id.String(), TargetName: name,
			After: map[string]any{"domain": name, "note": strings.TrimSpace(in.Note), "records": len(recs)}})
	})
	if err != nil {
		return store.SendingDomain{}, nil, err
	}
	d, err := s.Q.DomGetAny(ctx, id)
	if err != nil {
		return store.SendingDomain{}, nil, err
	}
	if checked, err := s.Check(ctx, d); err == nil {
		d = checked
	} else {
		s.log().WarnContext(ctx, "first check of a new domain", "domain", name, "err", err)
	}
	return d, notes, nil
}

// EnsureDomain returns the workspace's sending domain with that name,
// creating it (with domains:manage) when it doesn't exist yet.
func (s *Service) EnsureDomain(ctx context.Context, ws *store.Workspace, name string) (store.SendingDomain, error) {
	name, err := NormalizeDomain("body.domain", name)
	if err != nil {
		return store.SendingDomain{}, err
	}
	d, err := s.Q.DomGetByName(ctx, store.DomGetByNameParams{WorkspaceID: ws.ID, Domain: name})
	if err == nil {
		return d, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return store.SendingDomain{}, err
	}
	if p := api.PrincipalFrom(ctx); p == nil || !p.Can(authz.DomainsManage) {
		return store.SendingDomain{}, api.Forbidden(authz.DomainsManage, nil)
	}
	d, _, err = s.CreateDomain(ctx, ws, DomainInput{Domain: name})
	return d, err
}

// UpdateDomainInput changes a domain's note or tracking domain.
type UpdateDomainInput struct {
	Note *string `json:"note,omitempty" maxLength:"120"`
	// TrackingDomain is a subdomain that CNAMEs to ReputeCast so links and
	// the open pixel use the sender's domain ("" removes it).
	TrackingDomain *string `json:"trackingDomain,omitempty" maxLength:"253"`
}

// UpdateDomain changes a domain's note ("rename") or tracking domain.
func (s *Service) UpdateDomain(ctx context.Context, ws *store.Workspace, rawID string, in UpdateDomainInput) (store.SendingDomain, error) {
	id, err := parseID(rawID, "Sending domain")
	if err != nil {
		return store.SendingDomain{}, err
	}
	d, err := s.Q.DomGet(ctx, store.DomGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return store.SendingDomain{}, api.MapError(err, "Sending domain")
	}
	before := map[string]any{"note": d.Note, "tracking_domain": d.TrackingDomain}
	note, tracking := d.Note, d.TrackingDomain
	if in.Note != nil {
		note = strings.TrimSpace(*in.Note)
	}
	if in.TrackingDomain != nil {
		tracking = ""
		if t := strings.TrimSpace(*in.TrackingDomain); t != "" {
			if tracking, err = NormalizeDomain("body.trackingDomain", t); err != nil {
				return store.SendingDomain{}, err
			}
			if !strings.HasSuffix(tracking, "."+d.Domain) && tracking != d.Domain {
				return store.SendingDomain{}, api.Invalid("body.trackingDomain", "Use a subdomain of "+d.Domain+", for example go."+d.Domain+".")
			}
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.DomUpdateNote(ctx, store.DomUpdateNoteParams{ID: d.ID, Note: note, UpdatedAt: s.now()}); err != nil {
			return err
		}
		if tracking != d.TrackingDomain {
			status := ""
			if tracking != "" {
				status = StatusPending
			}
			if err := q.DomUpdateTracking(ctx, store.DomUpdateTrackingParams{ID: d.ID, TrackingDomain: tracking, TrackingStatus: status, UpdatedAt: s.now()}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "domain.update", TargetType: "sending_domain", TargetID: d.ID.String(), TargetName: d.Domain,
			Before: before, After: map[string]any{"note": note, "tracking_domain": tracking}})
	})
	if err != nil {
		return store.SendingDomain{}, err
	}
	d, err = s.Q.DomGetAny(ctx, d.ID)
	if err != nil {
		return store.SendingDomain{}, err
	}
	if tracking != "" && tracking != before["tracking_domain"] {
		if checked, err := s.Check(ctx, d); err == nil {
			d = checked
		}
	}
	return d, nil
}

// DeleteDomain removes a sending domain that no sender uses any more.
func (s *Service) DeleteDomain(ctx context.Context, ws *store.Workspace, rawID string) error {
	id, err := parseID(rawID, "Sending domain")
	if err != nil {
		return err
	}
	d, err := s.Q.DomGet(ctx, store.DomGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return api.MapError(err, "Sending domain")
	}
	senders, err := s.Q.SenderListByDomain(ctx, d.ID)
	if err != nil {
		return err
	}
	if len(senders) > 0 {
		n := len(senders)
		word := "senders use"
		if n == 1 {
			word = "sender uses"
		}
		return api.Conflict(strconv.Itoa(n) + " " + word + " " + d.Domain + ". Delete them first, so no campaign loses its From address.")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.DomDelete(ctx, d.ID); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "domain.delete", TargetType: "sending_domain", TargetID: d.ID.String(), TargetName: d.Domain,
			Before: map[string]any{"domain": d.Domain, "status": d.Status}})
	})
}
