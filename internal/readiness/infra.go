package readiness

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/deliverability"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// The sending infrastructure around a send (SR-7): the reverse DNS of the
// dedicated IPs carrying it, the route's fallback (C11), how each provider
// passes DMARC alignment and carries one-click unsubscribe, and Gmail
// Postmaster's spam rate.

// reverseDNS looks up the forward-confirmed reverse DNS of the dedicated IPs
// carrying the send. Deliverability's stored check answers when DNS doesn't.
func (s *Service) reverseDNS(ctx context.Context, f Facts, ov *deliverability.DeliverabilityOverview) []ReverseDNSFacts {
	stored := map[string]*deliverability.DeliverabilityReverseDNS{}
	if ov != nil {
		for _, a := range ov.Assets {
			if a.Type == "ip" && a.ReverseDNS != nil {
				stored[a.Asset] = a.ReverseDNS
			}
		}
	}
	out := []ReverseDNSFacts{}
	for _, sh := range carrying(f.Route.Shares) {
		for _, ip := range sh.IPs {
			st := stored[ip]
			r := ReverseDNSFacts{IP: ip, Provider: sh.Short}
			if st != nil {
				r.PTR, r.Status = st.PTR, st.Status
			}
			// Deliverability checks the IPs on its own schedule: what it
			// found passing stands; anything else is looked up again, and a
			// lookup that fails keeps what it found.
			addr, err := netip.ParseAddr(ip)
			if (st == nil || st.Status != dnscheck.StatusPass) && err == nil && s.DNS != nil {
				live := dnscheck.CheckReverseDNS(ctx, s.DNS, addr)
				if live.Status != dnscheck.StatusError || st == nil || st.Status == "" || st.Status == dnscheck.StatusError {
					r.PTR, r.Status = live.PTR, live.Status
					if live.Status == dnscheck.StatusError {
						r.Trouble = dnsTrouble(live.Error)
					}
				}
			}
			if r.Status == "" {
				continue
			}
			if r.Status == dnscheck.StatusError && r.Trouble == "" {
				r.Trouble = TroubleOther
			}
			out = append(out, r)
		}
	}
	return out
}

// vetFallback checks the route's fallback provider with state.Vet, for the
// whole send moving there (C11): what failover and the outage card ask.
func (s *Service) vetFallback(ctx context.Context, g *gathered, id uuid.UUID, need int64, now time.Time, loc *time.Location) *FallbackFacts {
	p, ok := g.byID[id]
	if !ok {
		var err error
		if p, err = s.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: g.ws.ID, ID: id}); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) && s.Log != nil {
				s.Log.WarnContext(ctx, "readiness fallback provider", "provider", id, "err", err)
			}
			return &FallbackFacts{ProviderID: id.String(), Failed: state.VetAvailable, Text: tx("fallback.gone", nil)}
		}
	}
	var sender map[string]string
	if g.c.SenderID != nil {
		if sd, err := s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: g.ws.ID, ID: *g.c.SenderID}); err == nil {
			sender = state.SenderVerification(sd.Verification)
		}
	}
	v := state.Vet(p, state.Load{Broadcast: true, Messages: need, Sender: sender, Warmup: warmup.Today(ctx, s.Q, p.ID, now, loc)})
	out := &FallbackFacts{ProviderID: p.ID.String(), Name: p.Name}
	if c, failed := v.Failed(); failed {
		out.Failed, out.Text = c.Key, Kept(c.Keep(p.Name))
	}
	return out
}

// routeFallback is the fallback of the Broadcasts route the campaign uses:
// none when the campaign overrides the route.
func routeFallback(set routing.Set, override *RouteOverride) *uuid.UUID {
	if override != nil {
		return nil
	}
	if def, ok := set.Default("broadcast"); ok {
		return def.Fallback
	}
	return nil
}

// postmaster reads Gmail Postmaster Tools' last week for the sending
// domain, or its parent when Google reports only that one.
func (s *Service) postmaster(ctx context.Context, wsID uuid.UUID, domain string, now time.Time) *PostmasterFacts {
	domain = strings.ToLower(domain)
	if domain == "" {
		return nil
	}
	org := dnscheck.OrgDomain(domain)
	rows, err := s.Q.ReadinessPostmasterDays(ctx, store.ReadinessPostmasterDaysParams{WorkspaceID: wsID, Domains: []string{domain, org}, Since: now.AddDate(0, 0, -7)})
	if err != nil || len(rows) == 0 {
		return nil
	}
	pick := domain
	if !hasPostmasterDomain(rows, domain) {
		pick = org
	}
	var out *PostmasterFacts
	for _, r := range rows { // newest first
		if !strings.EqualFold(r.Domain, pick) {
			continue
		}
		if out == nil {
			out = &PostmasterFacts{Domain: pick, Day: r.Day}
		}
		if r.SpamRate != nil {
			v := *r.SpamRate * 100
			out.SpamRate = &v
			if !r.Day.Equal(out.Day) {
				day := r.Day
				out.SpamDay = &day
			}
			break
		}
	}
	return out
}

func hasPostmasterDomain(rows []store.ReadinessPostmasterDaysRow, domain string) bool {
	for _, r := range rows {
		if strings.EqualFold(r.Domain, domain) {
			return true
		}
	}
	return false
}

// reverseDNSRows adds a row for each dedicated IP whose reverse DNS is
// missing or doesn't resolve back (Block: Gmail refuses such mail with
// 550 5.7.25), or couldn't be looked up (Unknown). A passing IP is named in
// its blocklist row.
func (b *builder) reverseDNSRows() {
	for _, r := range b.f.Reputation.ReverseDNS {
		key := "reputation.rdns." + r.IP
		label := tx("label.reverseDNS", nil)
		providers := link(tx("fix.openProviders", nil), b.href("/providers"))
		switch r.Status {
		case dnscheck.StatusPass:
		case dnscheck.StatusMissing, dnscheck.StatusFail:
			c := check{Key: key, Group: GroupReputation, Status: Block, Label: label,
				Title:  tx("rdns.bad", A{"provider": r.Provider, "ip": r.IP, "missing": yes(r.Status == dnscheck.StatusMissing), "ptr": r.PTR}),
				Detail: tx("rdns.badDetail", nil), Fixes: []fix{providers}}
			for _, s := range b.availableShares() {
				if s.Short == r.Provider {
					if o := b.otherAvailable(s.ProviderID); o != nil {
						c.Fixes = append([]fix{routeOnlyFix(o, tx("fix.sendViaOnly", A{"provider": o.Short}), "")}, c.Fixes...)
					}
				}
			}
			b.add(c)
		default:
			note := tx("note.dnsError", nil)
			b.add(check{Key: key, Group: GroupReputation, Status: Unknown, Label: label, Note: note,
				Title:  tx("rdns.unknown", A{"provider": r.Provider, "ip": r.IP}),
				Detail: tx("auth.lookupFailedDetail", A{"trouble": firstNonEmpty(r.Trouble, TroubleOther)}), Fixes: []fix{providers}})
		}
	}
}

// reverseDNSOf is the passing reverse DNS name of a dedicated IP ("" when
// it didn't pass or wasn't looked up).
func (b *builder) reverseDNSOf(ip string) string {
	for _, r := range b.f.Reputation.ReverseDNS {
		if r.IP == ip && r.Status == dnscheck.StatusPass {
			return r.PTR
		}
	}
	return ""
}

// fallbackRow warns when the route's fallback couldn't take the send: if
// a provider fails, the send holds for a decision instead of moving there.
func (b *builder) fallbackRow() {
	fb := b.f.Route.Fallback
	if fb == nil || fb.Failed == "" {
		return
	}
	name := firstNonEmpty(fb.Name, tx("fallback.unnamed", nil).en())
	b.add(check{Key: "routing.fallback", Group: GroupRouting, Status: Warn, Label: tx("label.fallback", nil),
		Title:  tx("fallback.cant", A{"provider": name}),
		Detail: sentences(fb.Text, tx("fallback.cantDetail", A{"policy": b.f.Route.Policy})),
		Ack:    tx("fallback.ack", A{"provider": name}),
		Fixes:  []fix{link(tx("fix.editRoute", nil), b.href("/providers/routing"))}})
}

// dmarcMode is a DMARC record's alignment mode for adkim or aspf: s
// (strict) or r (relaxed, the default).
func dmarcMode(record, tag string) string {
	for _, part := range strings.Split(record, ";") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && strings.EqualFold(strings.TrimSpace(k), tag) && strings.EqualFold(strings.TrimSpace(v), "s") {
			return "s"
		}
	}
	return "r"
}

// unaligned are the providers carrying the send whose mail fails DMARC
// alignment: no verified DKIM key for the From domain, and no return path
// under it (exactly it, under strict SPF alignment) with SPF passing.
// Providers whose records couldn't be looked up aren't judged.
func (b *builder) unaligned(domain string) []string {
	a := b.f.Auth
	strictSPF := dmarcMode(a.DMARCRecord, "aspf") == "s"
	out := []string{}
	for _, s := range b.shares() {
		dkim, judged := false, false
		for _, d := range a.DKIM {
			if d.Short == s.Short {
				if d.Unknown {
					judged = false
					break
				}
				judged = true
				dkim = dkim || d.OK
			}
		}
		if !judged || dkim {
			continue
		}
		spf := false
		for _, p := range a.SPF {
			if p.Short == s.Short && p.OK {
				spf = true
			}
		}
		for _, m := range a.MailFrom {
			if m.Short != s.Short || !m.OK || !spf {
				continue
			}
			d := strings.ToLower(m.Domain)
			if d == domain || !strictSPF && dnscheck.OrgDomain(d) == dnscheck.OrgDomain(domain) {
				dkim = true
			}
		}
		if !dkim {
			out = append(out, s.Short)
		}
	}
	return out
}
