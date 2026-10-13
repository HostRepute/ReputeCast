package forms

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
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

// SetDomain stores the custom domain and checks its DNS right away.
func (s *Service) SetDomain(ctx context.Context, ws *store.Workspace, in PublicDomainInput) (PublicPagesSettings, error) {
	raw := strings.TrimSpace(in.Domain)
	var domain *string
	if raw != "" {
		host, ok := NormalizeOrigin(raw)
		if !ok || strings.Contains(host, ":") || !strings.Contains(host, ".") {
			return PublicPagesSettings{}, api.Invalid("body.domain", "Use a domain name like news.acme.io, without https:// or a path.")
		}
		if host == instanceHost(s.URLs) {
			return PublicPagesSettings{}, api.Invalid("body.domain", "That's this ReputeCast's own address. Use a domain of your newsletter, like news.acme.io.")
		}
		taken, err := s.Q.PublicSettingsDomainTaken(ctx, store.PublicSettingsDomainTakenParams{Domain: &host, WorkspaceID: ws.ID})
		if err != nil {
			return PublicPagesSettings{}, err
		}
		if taken {
			return PublicPagesSettings{}, api.Conflict(host + " already serves another workspace's pages. Remove it there first.")
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

// CheckDomain runs the DNS check now ("Check DNS").
func (s *Service) CheckDomain(ctx context.Context, ws *store.Workspace) (PublicPagesSettings, error) {
	if _, err := s.checkDomain(ctx, ws, true); err != nil {
		return PublicPagesSettings{}, err
	}
	return s.PublicPages(ctx, ws)
}

// checkDomain verifies that the custom domain points at this instance: a
// CNAME to the instance's host, the same addresses as the instance, or the
// TXT record _reputecast.<domain>. A verified domain that fails once keeps
// working until a second check fails too (DNS hiccups happen).
func (s *Service) checkDomain(ctx context.Context, ws *store.Workspace, manual bool) (string, error) {
	set, err := s.settings(ctx, ws)
	if err != nil {
		return "", err
	}
	if set.row.CustomDomain == nil {
		return "none", nil
	}
	domain := *set.row.CustomDomain
	ok, detail := s.verifyDomainDNS(ctx, ws, domain)
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
		// A verified domain survives one failed check.
		if set.row.DomainStatus == "verified" && !strings.HasPrefix(set.row.DomainDetail, "Last check failed") {
			status = "verified"
			detail = "Last check failed: " + detail + " Checking again soon."
		}
		if status != "verified" {
			verifiedAt = nil
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PublicSettingsSetDomain(ctx, store.PublicSettingsSetDomainParams{WorkspaceID: ws.ID, CustomDomain: set.row.CustomDomain, DomainStatus: status,
			DomainDetail: detail, DomainCheckedAt: &now, DomainVerifiedAt: verifiedAt, UpdatedBy: set.row.UpdatedBy, At: set.row.UpdatedAt}); err != nil {
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

func (s *Service) verifyDomainDNS(ctx context.Context, ws *store.Workspace, domain string) (bool, string) {
	if s.DNS == nil {
		return false, "DNS checks aren't available on this server."
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	target := instanceHost(s.URLs)
	cname, err := s.DNS.CNAME(ctx, domain)
	if err == nil && strings.EqualFold(strings.TrimSuffix(cname, "."), target) {
		return true, fmt.Sprintf("CNAME points at %s.", target)
	}
	token := s.domainToken(ws, domain)
	if txts, err := s.DNS.TXT(ctx, "_reputecast."+domain); err == nil && slices.Contains(txts, token) {
		return true, "TXT record found."
	}
	// Apex domains can point A/AAAA records at the instance instead.
	if target != "" && target != "localhost" {
		mine, _ := s.addrs(ctx, target)
		theirs, _ := s.addrs(ctx, domain)
		for _, a := range theirs {
			if slices.Contains(mine, a) {
				return true, fmt.Sprintf("%s resolves to %s like %s.", domain, a, target)
			}
		}
	}
	switch {
	case err == nil && cname != "":
		return false, fmt.Sprintf("%s is a CNAME for %s, not %s. Point it at %s.", domain, strings.TrimSuffix(cname, "."), target, target)
	case err != nil && !dnscheck.IsNotFound(err):
		return false, "DNS didn't answer for " + domain + ". We'll check again in a few minutes."
	}
	return false, fmt.Sprintf("No CNAME from %s to %s yet. DNS changes can take up to an hour to show.", domain, target)
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

// --- host lookup -----------------------------------------------------------------

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
