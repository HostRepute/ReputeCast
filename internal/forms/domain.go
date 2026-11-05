package forms

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// PublicDomainInput sets or clears the custom domain.
type PublicDomainInput struct {
	Domain string `json:"domain" maxLength:"253" example:"news.acme.io" doc:"Empty removes the custom domain"`
}

// domainToken is the value of the TXT record that proves a workspace owns a
// domain, for domains that can't point a CNAME at the instance (apexes).
func (s *Service) domainToken(ws *store.Workspace, domain string) string {
	sum := sha256.Sum256([]byte("reputecast-domain:" + ws.ID.String() + ":" + domain))
	return "reputecast-verify=" + hex.EncodeToString(sum[:12])
}

// Purposes of CustomHostClaims that the domain check tells apart.
const claimInstance = "instance"

// SetDomain stores the custom domain and checks its DNS right away. A host
// another workspace uses already (for its pages, short links, link tracking
// or push site) is refused: pointing at this server doesn't make it this
// workspace's.
func (s *Service) SetDomain(ctx context.Context, ws *store.Workspace, in PublicDomainInput) (PublicPagesSettings, error) {
	raw := strings.TrimSpace(in.Domain)
	var domain *string
	if raw != "" {
		host, ok := NormalizeOrigin(raw)
		if !ok || strings.Contains(host, ":") || !strings.Contains(host, ".") {
			return PublicPagesSettings{}, api.Invalid("body.domain", say(ctx, mDomainName, nil))
		}
		if host == instanceHost(s.URLs) {
			return PublicPagesSettings{}, api.Invalid("body.domain", say(ctx, mDomainOwn, nil))
		}
		domain = &host
	}
	set, err := s.settings(ctx, ws)
	if err != nil {
		return PublicPagesSettings{}, err
	}
	var before string
	if set.row.CustomDomain != nil {
		before = *set.row.CustomDomain
	}
	unchanged := (domain == nil && before == "") || (domain != nil && *domain == before)
	if !unchanged {
		status := "none"
		if domain != nil {
			status = "pending"
		}
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if domain != nil {
				if err := s.claimHost(ctx, q, ws, *domain); err != nil {
					return err
				}
			}
			if err := q.PublicSettingsSetDomain(ctx, store.PublicSettingsSetDomainParams{WorkspaceID: ws.ID, CustomDomain: domain, DomainStatus: status,
				UpdatedBy: actor(ctx), At: s.now()}); err != nil {
				return err
			}
			after := ""
			if domain != nil {
				after = *domain
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "public_pages.domain", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
				Before: map[string]any{"domain": before}, After: map[string]any{"domain": after}})
		})
		if err != nil {
			return PublicPagesSettings{}, err
		}
		s.forgetHosts()
	}
	if domain != nil {
		if _, err := s.checkDomain(ctx, ws, true); err != nil {
			s.log().WarnContext(ctx, "custom domain check", "domain", *domain, "err", err)
		}
	}
	return s.PublicPages(ctx, ws)
}

// claimHost refuses a host that isn't free for ws: the instance's own, or
// one another workspace uses. It locks the host for the rest of q's
// transaction first, so two workspaces saving it at once can't both have it.
func (s *Service) claimHost(ctx context.Context, q *store.Queries, ws *store.Workspace, host string) error {
	if err := q.CustomHostLock(ctx, host); err != nil {
		return err
	}
	claims, err := q.CustomHostClaims(ctx, store.CustomHostClaimsParams{Host: host, WorkspaceID: ws.ID})
	if err != nil {
		return err
	}
	for _, c := range claims {
		if c.Purpose == claimInstance {
			return api.Invalid("body.domain", say(ctx, mDomainOwn, nil))
		}
	}
	if len(claims) > 0 {
		return api.Conflict(say(ctx, mDomainTaken, i18n.Args{"domain": host}))
	}
	return nil
}

// CheckDomain runs the DNS check now ("Check DNS").
func (s *Service) CheckDomain(ctx context.Context, ws *store.Workspace) (PublicPagesSettings, error) {
	if _, err := s.checkDomain(ctx, ws, true); err != nil {
		return PublicPagesSettings{}, err
	}
	return s.PublicPages(ctx, ws)
}

// checkDomain verifies that the custom domain is this workspace's to use and
// points at this instance: a CNAME to the instance's host, the same
// addresses as the instance, or the TXT record _reputecast.<domain>. A
// verified domain that fails once keeps working until a second check fails
// too (DNS hiccups happen), unless it isn't this workspace's to use: that
// stops it at once.
func (s *Service) checkDomain(ctx context.Context, ws *store.Workspace, manual bool) (string, error) {
	set, err := s.settings(ctx, ws)
	if err != nil {
		return "", err
	}
	if set.row.CustomDomain == nil {
		return "none", nil
	}
	domain := *set.row.CustomDomain
	var held *time.Time
	if set.row.DomainStatus == "verified" {
		held = set.row.DomainVerifiedAt
	}
	found, err := s.verifyDomain(ctx, ws, domain, held)
	if err != nil {
		return "", err
	}
	ok, detail := found.ok, found.detail
	now := s.now()
	status := "verified"
	verifiedAt := set.row.DomainVerifiedAt
	if ok {
		if verifiedAt == nil || set.row.DomainStatus != "verified" {
			verifiedAt = &now
		}
	} else {
		status = "failed"
		if set.row.DomainStatus == "pending" || (set.row.DomainStatus == "none") {
			status = "pending"
		}
		if set.row.DomainStatus == "verified" && !failedBefore(set.row.DomainDetail) && !found.refused {
			status = "verified"
			detail = mDomainLastFailed.Keep(i18n.Args{"detail": detail})
		}
		if status != "verified" {
			verifiedAt = nil
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PublicSettingsSetDomain(ctx, store.PublicSettingsSetDomainParams{WorkspaceID: ws.ID, CustomDomain: set.row.CustomDomain, DomainStatus: status,
			DomainDetail: english(detail), DomainCheckedAt: &now, DomainVerifiedAt: verifiedAt, UpdatedBy: set.row.UpdatedBy, At: set.row.UpdatedAt,
			Texts: keptTexts("domain_detail", detail)}); err != nil {
			return err
		}
		if status == set.row.DomainStatus && !manual {
			return nil
		}
		ev := audit.Event{WorkspaceID: &ws.ID, Action: "public_pages.domain_check", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"status": set.row.DomainStatus}, After: map[string]any{"status": status, "domain": domain}}
		if !manual {
			ev.Actor = &audit.Actor{Type: "system", Name: "System"}
		}
		return s.Audit.LogWith(ctx, q, ev)
	})
	if status != set.row.DomainStatus {
		s.forgetHosts()
	}
	return status, err
}

// domainCheck is what a check of the custom domain found; the finding is
// kept as a message, so each member reads it in their language.
type domainCheck struct {
	ok     bool
	detail i18n.Text
	// refused is a domain this workspace may not use whatever its DNS
	// says: no second check is waited for.
	refused bool
}

// verifyDomain checks that the custom domain may be this workspace's, then
// its DNS. A CNAME or matching addresses only show that a host points at
// this server, not whose host it is, so they count only for a host that is
// plainly this workspace's. Another workspace's host (held first) and the
// hosts under the instance's own need the TXT record, whose value is this
// workspace's alone; the instance's own hosts are never a workspace's.
// held is since when the workspace has had the domain verified (nil while
// it hasn't): a use another workspace made later doesn't take it away.
func (s *Service) verifyDomain(ctx context.Context, ws *store.Workspace, domain string, held *time.Time) (domainCheck, error) {
	target := instanceHost(s.URLs)
	own := domainCheck{detail: mDomainIsOwn.Keep(i18n.Args{"domain": domain}), refused: true}
	if domain == target {
		return own, nil
	}
	claims, err := s.Q.CustomHostClaims(ctx, store.CustomHostClaimsParams{Host: domain, WorkspaceID: ws.ID})
	if err != nil {
		return domainCheck{}, err
	}
	// why says why only the TXT record counts, when it's the only proof.
	var why i18n.Text
	for _, c := range claims {
		switch {
		case c.Purpose == claimInstance:
			return own, nil
		case takenBefore(c, held):
			why = mDomainInUse.Keep(i18n.Args{"domain": domain})
		}
	}
	if why.IsZero() && target != "" && strings.HasSuffix(domain, "."+target) {
		why = mDomainUnderOwn.Keep(i18n.Args{"domain": domain, "target": target})
	}
	if s.DNS == nil {
		return domainCheck{detail: mDomainUnavailable.Keep(nil)}, nil
	}
	if why.IsZero() {
		ok, detail := s.verifyDomainDNS(ctx, ws, domain)
		return domainCheck{ok: ok, detail: detail}, nil
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	txts, err := s.DNS.TXT(cctx, "_reputecast."+domain)
	switch {
	case err == nil && slices.Contains(txts, s.domainToken(ws, domain)):
		return domainCheck{ok: true, detail: mDomainTXT.Keep(nil)}, nil
	case err != nil && !dnscheck.IsNotFound(err):
		// No answer isn't a refusal: a verified domain gets its second check.
		return domainCheck{detail: mDomainNoAnswer.Keep(i18n.Args{"domain": domain})}, nil
	}
	return domainCheck{detail: why, refused: true}, nil
}

// takenBefore reports whether another workspace's use of a host keeps this
// workspace from it: any use while this workspace hasn't verified the host
// (held is nil), and afterwards one proven before this workspace's or at a
// time nobody recorded. A use proven later, or never, came second.
func takenBefore(c store.CustomHostClaimsRow, held *time.Time) bool {
	if held == nil {
		return true
	}
	if !c.Verified {
		return false
	}
	return c.Since == nil || !c.Since.After(*held)
}

// verifyDomainDNS checks the domain's records.
func (s *Service) verifyDomainDNS(ctx context.Context, ws *store.Workspace, domain string) (bool, i18n.Text) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	target := instanceHost(s.URLs)
	cname, err := s.DNS.CNAME(ctx, domain)
	if err == nil && strings.EqualFold(strings.TrimSuffix(cname, "."), target) {
		return true, mDomainCNAME.Keep(i18n.Args{"target": target})
	}
	token := s.domainToken(ws, domain)
	if txts, err := s.DNS.TXT(ctx, "_reputecast."+domain); err == nil && slices.Contains(txts, token) {
		return true, mDomainTXT.Keep(nil)
	}
	// Apex domains can point A/AAAA records at the instance instead.
	if target != "" && target != "localhost" {
		mine, _ := s.addrs(ctx, target)
		theirs, _ := s.addrs(ctx, domain)
		for _, a := range theirs {
			if slices.Contains(mine, a) {
				return true, mDomainResolves.Keep(i18n.Args{"domain": domain, "address": a.String(), "target": target})
			}
		}
	}
	switch {
	case err == nil && cname != "":
		return false, mDomainWrongCNAME.Keep(i18n.Args{"domain": domain, "cname": strings.TrimSuffix(cname, "."), "target": target})
	case err != nil && !dnscheck.IsNotFound(err):
		return false, mDomainNoAnswer.Keep(i18n.Args{"domain": domain})
	}
	return false, mDomainNoCNAME.Keep(i18n.Args{"domain": domain, "target": target})
}

// failedBefore reports whether a stored finding already says that a check
// failed: its English starts as domain.lastFailed's does.
func failedBefore(stored string) bool {
	prefix, _, _ := strings.Cut(english(mDomainLastFailed.Keep(i18n.Args{"detail": ""})), ":")
	return strings.HasPrefix(stored, prefix)
}

func (s *Service) addrs(ctx context.Context, host string) ([]netip.Addr, error) {
	var out []netip.Addr
	a, err4 := s.DNS.A(ctx, host)
	out = append(out, a...)
	aaaa, err6 := s.DNS.AAAA(ctx, host)
	out = append(out, aaaa...)
	if len(out) == 0 {
		return nil, errors.Join(err4, err6)
	}
	return out, nil
}

type hostEntry struct {
	slug    string
	expires time.Time
}

// WorkspaceForHost returns the workspace slug whose verified custom domain
// is host ("" when none). Answers are cached for a minute.
func (s *Service) WorkspaceForHost(ctx context.Context, host string) string {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if i := strings.LastIndexByte(host, ':'); i > 0 && !strings.Contains(host[i:], "]") {
		host = host[:i]
	}
	if host == "" || host == instanceHost(s.URLs) {
		return ""
	}
	now := s.now()
	s.hostsMu.Lock()
	if s.hosts == nil {
		s.hosts = map[string]hostEntry{}
	}
	if e, ok := s.hosts[host]; ok && now.Before(e.expires) {
		s.hostsMu.Unlock()
		return e.slug
	}
	s.hostsMu.Unlock()
	slug := ""
	if row, err := s.Q.PublicSettingsByDomain(ctx, &host); err == nil {
		slug = row.Slug
	}
	s.hostsMu.Lock()
	if len(s.hosts) > 10_000 {
		s.hosts = map[string]hostEntry{}
	}
	s.hosts[host] = hostEntry{slug: slug, expires: now.Add(time.Minute)}
	s.hostsMu.Unlock()
	return slug
}

// HostVerified reports whether host is any workspace's verified custom
// domain (the "ask" endpoint of on-demand TLS in Caddy and similar proxies).
func (s *Service) HostVerified(ctx context.Context, host string) bool {
	return s.WorkspaceForHost(ctx, host) != ""
}

func (s *Service) forgetHosts() {
	s.hostsMu.Lock()
	s.hosts = nil
	s.hostsMu.Unlock()
}
