package sso

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// SSODiscoveryInput tests discovery, optionally for an issuer not saved yet.
type SSODiscoveryInput struct {
	IssuerURL string `json:"issuerUrl,omitempty" maxLength:"300"`
}

// TestDiscovery fetches the issuer's discovery document and JWKS, stores
// the result (and the issuer when it was passed) and returns the provider.
func (s *Service) TestDiscovery(ctx context.Context, id uuid.UUID, in SSODiscoveryInput) (SSOProviderDetail, error) {
	p, err := s.Q.SSOProvider(ctx, id)
	if err != nil {
		return SSOProviderDetail{}, api.MapError(err, "SSO provider")
	}
	issuer := strings.TrimSpace(in.IssuerURL)
	if issuer == "" {
		issuer = p.IssuerUrl
	}
	if issuer == "" {
		return SSOProviderDetail{}, api.Invalid("body.issuerUrl", "Enter the issuer URL first, like "+PresetFor(p.Preset).IssuerPattern+".")
	}
	d := s.discover(ctx, issuer)
	if p.Status == "enabled" && !d.OK {
		// A live provider keeps its issuer: a typo here mustn't lock people out.
		issuer = p.IssuerUrl
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SSOSetDiscovery(ctx, store.SSOSetDiscoveryParams{ID: id, Discovery: mustJSON(d), IssuerUrl: issuer}); err != nil {
			return err
		}
		ev := audit.Event{Action: "sso.provider.discovery_tested", TargetType: "sso_provider", TargetID: id.String(), TargetName: p.Name,
			Metadata: map[string]any{"url": d.URL, "status": d.Status, "ms": d.Millis, "ok": d.OK, "summary": discoverySummary(d, issuer != p.IssuerUrl)}}
		if issuer != p.IssuerUrl {
			ev.Before, ev.After = map[string]any{"issuer": p.IssuerUrl}, map[string]any{"issuer": issuer}
		}
		return s.Audit.LogWith(ctx, q, ev)
	})
	if err != nil {
		return SSOProviderDetail{}, err
	}
	s.forget(issuer)
	fresh, err := s.Q.SSOProvider(ctx, id)
	if err != nil {
		return SSOProviderDetail{}, err
	}
	out, err := s.detail(ctx, fresh)
	if err != nil {
		return SSOProviderDetail{}, err
	}
	if !d.OK && p.Status == "enabled" && out.Discovery != nil {
		// Show the failed attempt, but the saved issuer stays the working one.
		out.Discovery.URL = d.URL
	}
	return out, nil
}

// discoverySummary describes a discovery test for the audit log:
// "discovery OK · 4 endpoints · 3 signing keys", or why it failed.
func discoverySummary(d discovery, issuerChanged bool) string {
	parts := []string{}
	if issuerChanged {
		parts = append(parts, "issuer changed")
	}
	if d.OK {
		n := 0
		for _, e := range []string{d.AuthURL, d.TokenURL, d.UserInfo, d.JWKSURL} {
			if e != "" {
				n++
			}
		}
		parts = append(parts, "discovery OK", fmt.Sprintf("%d endpoints", n), fmt.Sprintf("%d signing %s", d.Keys, plural(d.Keys, "key", "keys")))
	} else {
		parts = append(parts, "discovery failed: "+strings.TrimSuffix(d.Error, "."))
	}
	return strings.Join(parts, " · ")
}

// SSOTestStart is where the test sign-in popup goes.
type SSOTestStart struct {
	AuthorizeURL string `json:"authorizeUrl"`
}

// SSOTestInput names the workspace the admin returns to after the test.
type SSOTestInput struct {
	Workspace string `json:"workspace" pattern:"^[a-z0-9][a-z0-9-]{1,62}$"`
}

// StartTest begins a test sign-in: the admin signs in at the IdP in a
// popup; the callback stores the claims and signs nobody in.
func (s *Service) StartTest(ctx context.Context, id uuid.UUID, in SSOTestInput) (SSOTestStart, *http.Cookie, error) {
	admin, err := s.admin(ctx)
	if err != nil {
		return SSOTestStart{}, nil, err
	}
	p, err := s.Q.SSOProvider(ctx, id)
	if err != nil {
		return SSOTestStart{}, nil, api.MapError(err, "SSO provider")
	}
	if b := blockers(p); len(b) > 0 {
		return SSOTestStart{}, nil, api.Err(http.StatusConflict, "sso_incomplete", "Before a test sign-in: "+strings.Join(b, " "))
	}
	var out SSOTestStart
	var cookie *http.Cookie
	var ferr *flowError
	r := s.redirectToIdP(ctx, p, flowState{ProviderID: p.ID.String(), Mode: modeTest, Workspace: in.Workspace}, &admin.ID, func(fe *flowError) Redirect {
		ferr = fe
		return Redirect{}
	})
	if ferr != nil {
		return SSOTestStart{}, nil, api.Err(http.StatusBadGateway, "sso_unreachable", ferr.Detail)
	}
	out.AuthorizeURL = r.Location
	if len(r.Cookies) > 0 {
		c := r.Cookies[0]
		cookie = &c
	}
	return out, cookie, nil
}

// finishTest stores the result of a test sign-in and returns the popup to
// the provider's page.
func (s *Service) finishTest(ctx context.Context, p store.SsoProvider, ch store.AuthChallenge, fs flowState, query url.Values) Redirect {
	back := "/"
	if fs.Workspace != "" {
		back = "/w/" + fs.Workspace + "/admin/sso/" + p.ID.String() + "?test=done"
	}
	by := ""
	if ch.UserID != nil {
		if u, err := s.Q.GetUser(ctx, *ch.UserID); err == nil {
			by = displayName(u)
		}
	}
	run := testRun{At: s.now(), By: by}
	if e := query.Get("error"); e != "" {
		run.Error = p.Name + " returned " + e
		if d := query.Get("error_description"); d != "" {
			run.Error += ": " + firstLine(d)
		}
	} else if prov, conf, err := s.oauthConfig(ctx, p); err != nil {
		run.Error = asFlowError(err).Detail
	} else if claims, err := s.exchange(ctx, prov, conf, query.Get("code"), fs.Verifier, fs.Nonce, false); err != nil {
		run.Error = asFlowError(err).Detail
	} else {
		run.Claims = claims
	}
	meta := map[string]any{"result": "claims returned"}
	if run.Error != "" {
		meta["result"] = run.Error
	} else {
		id := extract(p, run.Claims)
		meta["email"], meta["groups"] = id.Email, len(id.Groups)
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SSOSetLastTest(ctx, store.SSOSetLastTestParams{ID: p.ID, LastTest: mustJSON(run)}); err != nil {
			return err
		}
		ev := audit.Event{Action: "sso.provider.tested", TargetType: "sso_provider", TargetID: p.ID.String(), TargetName: p.Name, Metadata: meta}
		if ch.UserID != nil {
			if u, err := q.GetUser(ctx, *ch.UserID); err == nil {
				ev.Actor = actorOf(u)
			}
		}
		if run.Error != "" {
			ev.Result = audit.Failed
		}
		return s.Audit.LogWith(ctx, q, ev)
	})
	if err != nil {
		s.Log.ErrorContext(ctx, "store test sign-in", "provider", p.Slug, "err", err)
	}
	return Redirect{Location: back}
}

// Enable makes a provider live: its button appears on the sign-in page.
func (s *Service) Enable(ctx context.Context, id uuid.UUID) (SSOProviderDetail, error) {
	return s.setStatus(ctx, id, "enabled")
}

// Disable turns a provider off; its enforcement ends with it.
func (s *Service) Disable(ctx context.Context, id uuid.UUID) (SSOProviderDetail, error) {
	return s.setStatus(ctx, id, "disabled")
}

func (s *Service) setStatus(ctx context.Context, id uuid.UUID, status string) (SSOProviderDetail, error) {
	before, err := s.Q.SSOProvider(ctx, id)
	if err != nil {
		return SSOProviderDetail{}, api.MapError(err, "SSO provider")
	}
	if status == "enabled" {
		if b := blockers(before); len(b) > 0 {
			return SSOProviderDetail{}, api.Err(http.StatusConflict, "sso_incomplete", "Before enabling "+before.Name+": "+strings.Join(b, " "))
		}
	}
	if before.Status == status {
		return s.detail(ctx, before)
	}
	var after store.SsoProvider
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		after, err = q.SSOSetStatus(ctx, store.SSOSetStatusParams{ID: id, Status: status})
		if err != nil {
			return err
		}
		action := "sso.provider.enabled"
		meta := map[string]any{}
		if status != "enabled" {
			action = "sso.provider.disabled"
			if len(before.EnforceDomains) > 0 {
				meta["enforcement_ended"] = before.EnforceDomains
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: action, TargetType: "sso_provider", TargetID: id.String(), TargetName: before.Name,
			Before: map[string]any{"status": before.Status, "enforced": before.EnforceDomains}, After: map[string]any{"status": after.Status, "enforced": after.EnforceDomains}, Metadata: meta})
	})
	if err != nil {
		return SSOProviderDetail{}, err
	}
	if s.OnPolicyChange != nil {
		s.OnPolicyChange()
	}
	return s.detail(ctx, after)
}

// Delete removes a provider that isn't live. The identities linked to it
// go with it; people keep their accounts.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	p, err := s.Q.SSOProvider(ctx, id)
	if err != nil {
		return api.MapError(err, "SSO provider")
	}
	if p.Status == "enabled" {
		return api.Conflict("Disable " + p.Name + " before deleting it.")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.SSODeleteProvider(ctx, id); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "sso.provider.deleted", TargetType: "sso_provider", TargetID: id.String(), TargetName: p.Name,
			Before: providerAuditView(p)})
	})
}

// SSOEnforcementMember is one person at the enforced domain.
type SSOEnforcementMember struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AvatarColor string `json:"avatarColor"`
	// Today and After describe how they sign in now and once enforced.
	Today      string `json:"today"`
	After      string `json:"after"`
	BreakGlass bool   `json:"breakGlass"`
	// Unlinked: they don't sign in with the provider yet.
	Unlinked bool `json:"unlinked"`
}

// SSOEnforcementPreview is what turning enforcement on would change.
type SSOEnforcementPreview struct {
	Provider   string                 `json:"provider"`
	Domains    []string               `json:"domains"`
	Members    []SSOEnforcementMember `json:"members"`
	BreakGlass []SSOBreakGlassOwner   `json:"breakGlass"`
	// Conflict names another enabled provider that already enforces a domain.
	Conflict string `json:"conflict,omitempty"`
	Enforced bool   `json:"enforced"`
}

// EnforcementPreview lists who signs in how before and after enforcement.
func (s *Service) EnforcementPreview(ctx context.Context, id uuid.UUID, domains []string) (SSOEnforcementPreview, error) {
	p, err := s.Q.SSOProvider(ctx, id)
	if err != nil {
		return SSOEnforcementPreview{}, api.MapError(err, "SSO provider")
	}
	ds, err := normalizeDomains(domains)
	if err != nil {
		return SSOEnforcementPreview{}, api.Invalid("query.domain", err.Error())
	}
	if len(ds) == 0 {
		ds = nonNil(p.AllowedDomains)
	}
	out := SSOEnforcementPreview{Provider: p.Name, Domains: ds, Members: []SSOEnforcementMember{}, BreakGlass: []SSOBreakGlassOwner{}, Enforced: len(p.EnforceDomains) > 0}
	if len(ds) == 0 {
		return out, nil
	}
	if other, err := s.Q.SSODomainEnforcedElsewhere(ctx, store.SSODomainEnforcedElsewhereParams{ID: id, Domains: ds}); err == nil && other != "" {
		out.Conflict = other
	}
	r, err := s.loadRefs(ctx)
	if err != nil {
		return out, err
	}
	out.BreakGlass = r.breakGlass(ds)
	owners := map[string]bool{}
	for _, o := range out.BreakGlass {
		owners[o.Email] = true
	}
	rows, err := s.Q.SSODomainUsers(ctx, store.SSODomainUsersParams{ProviderID: &p.ID, Domains: ds})
	if err != nil {
		return out, err
	}
	sso := p.Name + " SSO"
	for _, u := range rows {
		today := []string{}
		if u.Linked {
			today = append(today, sso)
		}
		if u.HasPassword {
			today = append(today, "Password")
		}
		if u.Passkeys > 0 {
			today = append(today, "passkey")
		}
		for _, k := range u.Social {
			today = append(today, socialName(k))
		}
		m := SSOEnforcementMember{ID: u.ID.String(), Name: u.Name, Email: u.Email, AvatarColor: u.AvatarColor, Today: strings.Join(today, " · "),
			BreakGlass: owners[u.Email], Unlinked: !u.Linked}
		if len(today) == 0 {
			m.Today = "–"
		}
		onlySSO := u.Linked && !u.HasPassword && u.Passkeys == 0 && len(u.Social) == 0
		switch {
		case m.BreakGlass && u.HasPassword:
			m.After = p.Name + ", plus break-glass password"
		case m.BreakGlass && u.Passkeys > 0:
			m.After = p.Name + ", plus break-glass passkey"
		case onlySSO:
			m.After = "Unchanged"
		default:
			m.After = p.Name + " only"
		}
		out.Members = append(out.Members, m)
	}
	// People who only use SSO first, then those whose sign-in changes, as in
	// the confirmation dialog.
	slices.SortStableFunc(out.Members, func(a, b SSOEnforcementMember) int {
		rank := func(m SSOEnforcementMember) int {
			switch {
			case m.BreakGlass:
				return 2
			case m.After == "Unchanged":
				return 0
			}
			return 1
		}
		return rank(a) - rank(b)
	})
	return out, nil
}

func socialName(k string) string {
	switch k {
	case "google":
		return "Google"
	case "microsoft":
		return "Microsoft"
	case "github":
		return "GitHub"
	}
	return k
}

// SSOEnforcementInput turns domain enforcement on or off.
type SSOEnforcementInput struct {
	// Domains to enforce; empty turns enforcement off.
	Domains []string `json:"domains" maxItems:"20"`
	// Confirm acknowledges who keeps password access (the dialog's list).
	Confirm bool `json:"confirm"`
}

// SetEnforcement turns password, passkey and social sign-in off for the
// domains, except for break-glass Owners, who are marked so that their
// password sign-ins keep working (and are logged as break-glass).
func (s *Service) SetEnforcement(ctx context.Context, id uuid.UUID, in SSOEnforcementInput) (SSOProviderDetail, error) {
	p, err := s.Q.SSOProvider(ctx, id)
	if err != nil {
		return SSOProviderDetail{}, api.MapError(err, "SSO provider")
	}
	ds, err := normalizeDomains(in.Domains)
	if err != nil {
		return SSOProviderDetail{}, api.Invalid("body.domains", err.Error())
	}
	if len(ds) > 0 {
		if p.Status != "enabled" {
			return SSOProviderDetail{}, api.Conflict("Enable " + p.Name + " before enforcing it for a domain.")
		}
		if !in.Confirm {
			return SSOProviderDetail{}, api.Invalid("body.confirm", "Confirm who keeps password access before turning enforcement on.")
		}
		for _, d := range ds {
			if len(p.AllowedDomains) > 0 && !slices.Contains(p.AllowedDomains, d) {
				return SSOProviderDetail{}, api.Invalid("body.domains", d+" isn't one of "+p.Name+"'s allowed domains.")
			}
		}
		if other, err := s.Q.SSODomainEnforcedElsewhere(ctx, store.SSODomainEnforcedElsewhereParams{ID: id, Domains: ds}); err == nil && other != "" {
			return SSOProviderDetail{}, api.Conflict(other + " already enforces one of these domains. Turn it off there first.")
		}
	}
	r, err := s.loadRefs(ctx)
	if err != nil {
		return SSOProviderDetail{}, err
	}
	keep := r.breakGlass(ds)
	var ids []uuid.UUID
	names := []string{}
	for _, o := range r.owners {
		if slices.ContainsFunc(keep, func(k SSOBreakGlassOwner) bool { return k.Email == o.Email }) {
			ids = append(ids, o.ID)
			names = append(names, o.Name)
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SSOSetEnforcement(ctx, store.SSOSetEnforcementParams{ID: id, EnforceDomains: ds}); err != nil {
			return err
		}
		if len(ids) > 0 {
			if _, err := q.SSOMarkBreakGlass(ctx, ids); err != nil {
				return err
			}
		}
		action := "sso.enforcement.enabled"
		summary := "password sign-in off for " + strings.Join(ds, ", ") + " · break-glass: " + strings.Join(names, ", ")
		if len(ds) == 0 {
			action, summary = "sso.enforcement.disabled", "password sign-in allowed again for "+strings.Join(p.EnforceDomains, ", ")
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: action, TargetType: "sso_provider", TargetID: id.String(), TargetName: p.Name,
			Before: map[string]any{"enforced": p.EnforceDomains}, After: map[string]any{"enforced": ds},
			Metadata: map[string]any{"summary": summary, "break_glass": names}})
	})
	if err != nil {
		return SSOProviderDetail{}, err
	}
	if s.OnPolicyChange != nil {
		s.OnPolicyChange()
	}
	fresh, err := s.Q.SSOProvider(ctx, id)
	if err != nil {
		return SSOProviderDetail{}, err
	}
	return s.detail(ctx, fresh)
}
