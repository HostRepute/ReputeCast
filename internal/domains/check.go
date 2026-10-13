package domains

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// CheckEvery is how often the job re-verifies every sending domain
// ("DNS checked every 30 minutes").
const CheckEvery = 30 * time.Minute

// refreshGap is the shortest time between two checks of one domain from
// the domain step's live polling ("Checking every 30 s").
const refreshGap = 20 * time.Second

// Check runs a live DNS check of a sending domain: every required record,
// the provider's own verdict where its API reports one, the authentication
// summary, the tracking domain and the nameservers; then it updates the
// verification of the domain's senders. Nothing here calls the API of a
// demo provider.
func (s *Service) Check(ctx context.Context, d store.SendingDomain) (store.SendingDomain, error) {
	now := s.now()
	recs := DecodeRecords(d.DnsRecords)
	provs, err := s.Q.ProvListActive(ctx, d.WorkspaceID)
	if err != nil {
		return d, err
	}
	byID := map[string]store.Provider{}
	for _, p := range provs {
		byID[p.ID.String()] = p
	}
	// The provider's own view (SES "SUCCESS", Postmark "verified").
	asked := map[string]bool{}
	for _, r := range recs {
		p, ok := byID[r.ProviderID]
		if !ok || asked[r.ProviderID] || manage.IsDemo(p) || s.Factory == nil {
			continue
		}
		asked[r.ProviderID] = true
		adapter, err := s.Factory.Open(ctx, p)
		if err != nil {
			continue
		}
		dm, ok := adapter.(providers.DomainManager)
		if !ok {
			continue
		}
		cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		fresh, err := dm.DomainStatus(cctx, d.Domain)
		cancel()
		if err != nil {
			s.log().WarnContext(ctx, "domain status from provider", "domain", d.Domain, "provider", p.ID, "err", err)
			continue
		}
		recs = mergeProviderStatus(recs, r.ProviderID, fresh)
	}
	for i := range recs {
		recs[i] = s.verify(ctx, d.Domain, recs[i])
		recs[i].CheckedAt = &now
	}
	reqs := make([]dnscheck.Requirement, 0, len(recs))
	for _, r := range recs {
		if r.Purpose == PurposeSPF && r.isDomainSPF(d.Domain) {
			continue // checked against the merged record above
		}
		reqs = append(reqs, dnscheck.Requirement{ProviderID: r.ProviderID, Purpose: r.Purpose, Type: r.Type, Name: r.Name, Value: r.Value})
	}
	auth := dnscheck.CheckDomain(ctx, s.DNS, d.Domain, reqs)
	det := AuthDetail{SPFStatus: auth.SPF.Status, SPFAll: auth.SPF.All, SPFRecord: auth.SPF.Record, DMARCStatus: auth.DMARC.Status,
		DMARCPolicy: auth.DMARC.Policy, DMARCRecord: auth.DMARC.Record, BIMIStatus: auth.BIMI.Status,
		DMARCInherited: auth.DMARC.PolicyDomain != "" && auth.DMARC.PolicyDomain != d.Domain}
	for _, mf := range auth.MailFrom {
		det.MailFrom = append(det.MailFrom, MailFromHost{Host: mf.Domain, Status: mf.Status})
	}
	sum := auth.Summary
	// A domain-SPF requirement that isn't met means the sending domain's SPF
	// doesn't authorize every provider yet.
	for _, r := range recs {
		if r.isDomainSPF(d.Domain) && r.Status != StatusVerified && sum.SPF == dnscheck.StatusPass {
			sum.SPF = dnscheck.StatusFail
		}
	}
	trackingStatus := ""
	if d.TrackingDomain != "" {
		trackingStatus = s.verifyTracking(ctx, d.TrackingDomain)
	}
	ns, host := s.nameservers(ctx, d.Domain)
	sumJSON, _ := json.Marshal(sum)
	detJSON, _ := json.Marshal(det)
	if err := s.Q.DomSaveCheck(ctx, store.DomSaveCheckParams{ID: d.ID, DnsRecords: EncodeRecords(recs), AuthSummary: sumJSON, AuthDetail: detJSON,
		Status: overallStatus(recs), TrackingStatus: trackingStatus, DnsHost: host, Nameservers: ns, CheckedAt: &now}); err != nil {
		return d, err
	}
	if err := s.refreshSenders(ctx, d.ID, recs, provs); err != nil {
		return d, err
	}
	s.markChecked(d.ID, now)
	return s.Q.DomGetAny(ctx, d.ID)
}

// CheckIfStale checks a domain unless it was checked moments ago (the
// domain step polls every 30 seconds).
func (s *Service) CheckIfStale(ctx context.Context, d store.SendingDomain) (store.SendingDomain, error) {
	if d.LastCheckedAt != nil && s.now().Sub(*d.LastCheckedAt) < refreshGap {
		return d, nil
	}
	s.checkMu.Lock()
	last := s.lastCheck[d.ID]
	s.checkMu.Unlock()
	if s.now().Sub(last) < refreshGap {
		return d, nil
	}
	s.markChecked(d.ID, s.now())
	return s.Check(ctx, d)
}

func (s *Service) markChecked(id uuid.UUID, at time.Time) {
	s.checkMu.Lock()
	defer s.checkMu.Unlock()
	if s.lastCheck == nil {
		s.lastCheck = map[uuid.UUID]time.Time{}
	}
	s.lastCheck[id] = at
	if len(s.lastCheck) > 1000 {
		for k, v := range s.lastCheck {
			if at.Sub(v) > time.Hour {
				delete(s.lastCheck, k)
			}
		}
	}
}

// mergeProviderStatus records a provider's verdict on its records and adds
// records it reports that weren't stored yet (rotated DKIM tokens).
func mergeProviderStatus(recs []Record, providerID string, fresh []providers.DNSRecord) []Record {
	idx := map[string]int{}
	for i, r := range recs {
		if r.ProviderID == providerID {
			idx[strings.ToUpper(r.Type)+"|"+strings.ToLower(r.Name)+"|"+r.Purpose] = i
		}
	}
	for _, f := range fresh {
		k := strings.ToUpper(f.Type) + "|" + strings.ToLower(strings.TrimSuffix(f.Name, ".")) + "|" + f.Purpose
		if i, ok := idx[k]; ok {
			recs[i].ProviderStatus = providerWord(f.Status)
			if f.Value != "" && f.Value != recs[i].Value && recs[i].Source == "provider" {
				recs[i].Value, recs[i].Status = f.Value, StatusPending
			}
			continue
		}
		recs = append(recs, Record{ProviderID: providerID, Purpose: f.Purpose, Type: strings.ToUpper(f.Type), Name: strings.ToLower(strings.TrimSuffix(f.Name, ".")),
			Value: f.Value, Status: StatusPending, ProviderStatus: providerWord(f.Status), Source: "provider"})
	}
	return recs
}

func providerWord(s string) string {
	switch s {
	case "verified":
		return "verified"
	case "wrong_value", "failed":
		return "failed"
	}
	return "pending"
}

// verify checks one record against DNS.
func (s *Service) verify(ctx context.Context, domain string, r Record) Record {
	r.Found, r.FoundName = "", ""
	switch {
	case r.Purpose == PurposeDMARC:
		dm := dnscheck.CheckDMARC(ctx, s.DNS, domain)
		switch dm.Status {
		case dnscheck.StatusPass:
			// Keep the suggestion in Value; Found is what is published,
			// possibly inherited from the organizational domain.
			r.Status, r.Found, r.FoundName = StatusVerified, dm.Record, "_dmarc."+dm.PolicyDomain
		case dnscheck.StatusMissing:
			r.Status = StatusMissing
		case dnscheck.StatusInvalid:
			r.Status, r.Found = StatusWrong, dm.Record
		default:
			r.Status = StatusError
		}
	case r.isDomainSPF(domain) && len(r.Mechanisms) > 0:
		spf := dnscheck.CheckSPF(ctx, s.DNS, domain)
		switch spf.Status {
		case dnscheck.StatusMissing:
			r.Status = StatusMissing
			r.Value = MergeSPF("", r.Mechanisms)
		case dnscheck.StatusError:
			r.Status = StatusError
		default:
			all := true
			for _, m := range r.Mechanisms {
				if !SPFHas(spf.Record, m) {
					all = false
				}
			}
			r.Value = MergeSPF(spf.Record, r.Mechanisms)
			if all {
				r.Status = StatusVerified
			} else {
				r.Status, r.Found = StatusWrong, spf.Record
			}
		}
	default:
		rc := dnscheck.VerifyRecord(ctx, s.DNS, dnscheck.Requirement{Purpose: r.Purpose, Type: r.Type, Name: r.Name, Value: r.Value})
		r.Status, r.Found = rc.Status, rc.Found
		if r.Status == dnscheck.RecordError {
			r.Status = StatusError
		}
	}
	return r
}

// verifyTracking checks a custom tracking domain's CNAME.
func (s *Service) verifyTracking(ctx context.Context, tracking string) string {
	target := ""
	if s.TrackingTarget != nil {
		target = s.TrackingTarget()
	}
	got, err := s.DNS.CNAME(ctx, tracking)
	switch {
	case err != nil && dnscheck.IsNotFound(err):
		return StatusMissing
	case err != nil:
		return StatusError
	case target == "" || strings.EqualFold(strings.TrimSuffix(got, "."), target):
		return StatusVerified
	}
	return StatusWrong
}

func overallStatus(recs []Record) string {
	if len(recs) == 0 {
		return "pending"
	}
	verified := 0
	for _, r := range recs {
		if r.Status == StatusVerified {
			verified++
		}
	}
	switch {
	case verified == len(recs):
		return "verified"
	case verified > 0:
		return "partial"
	}
	return "pending"
}

// refreshSenders recomputes each sender's verification per provider from
// the domain's records, keeping the providers marked "not used".
func (s *Service) refreshSenders(ctx context.Context, domainID uuid.UUID, recs []Record, provs []store.Provider) error {
	senders, err := s.Q.SenderListByDomain(ctx, domainID)
	if err != nil {
		return err
	}
	for _, sn := range senders {
		cur := decodeVerification(sn.Verification)
		next := map[string]string{}
		for _, p := range provs {
			id := p.ID.String()
			if cur[id] == "not_used" {
				next[id] = "not_used"
				continue
			}
			next[id] = providerVerification(recs, id)
		}
		if maps.Equal(cur, next) {
			continue
		}
		b, _ := json.Marshal(next)
		if err := s.Q.SenderSetVerification(ctx, store.SenderSetVerificationParams{ID: sn.ID, Verification: b}); err != nil {
			return err
		}
	}
	return nil
}

// nameservers looks up who hosts the domain's zone (walking up to the
// registered domain) and names the DNS provider.
func (s *Service) nameservers(ctx context.Context, domain string) ([]string, string) {
	if s.NS == nil {
		return []string{}, ""
	}
	name := domain
	for i := 0; i < 4 && strings.Contains(name, "."); i++ {
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		ns, err := s.NS(cctx, name)
		cancel()
		if err == nil && len(ns) > 0 {
			for j := range ns {
				ns[j] = strings.ToLower(strings.TrimSuffix(ns[j], "."))
			}
			slices.Sort(ns)
			return ns, DNSHostName(ns)
		}
		if name == dnscheck.OrgDomain(name) {
			break
		}
		name = name[strings.IndexByte(name, '.')+1:]
	}
	return []string{}, ""
}

// DNSHostName names the DNS provider of a set of nameservers.
func DNSHostName(ns []string) string {
	hosts := []struct{ suffix, name string }{
		{"ns.cloudflare.com", "Cloudflare"}, {"awsdns", "Amazon Route 53"}, {"googledomains.com", "Google Cloud DNS"},
		{"domaincontrol.com", "GoDaddy"}, {"registrar-servers.com", "Namecheap"}, {"digitalocean.com", "DigitalOcean"},
		{"linode.com", "Akamai (Linode)"}, {"hetzner.com", "Hetzner"}, {"hetzner.de", "Hetzner"}, {"ovh.net", "OVHcloud"},
		{"gandi.net", "Gandi"}, {"dnsimple.com", "DNSimple"}, {"vercel-dns.com", "Vercel"}, {"nsone.net", "NS1"},
		{"azure-dns", "Azure DNS"}, {"porkbun.com", "Porkbun"}, {"name.com", "Name.com"}, {"netlify", "Netlify"},
		{"hover.com", "Hover"}, {"ultradns", "UltraDNS"}, {"dynect.net", "Dyn"}, {"bunny.net", "bunny.net"},
	}
	for _, n := range ns {
		for _, h := range hosts {
			if strings.Contains(n, h.suffix) {
				return h.name
			}
		}
	}
	return ""
}
