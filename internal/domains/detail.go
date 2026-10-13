package domains

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// SendingDomainRecord is one row of a domain's records table.
type SendingDomainRecord struct {
	// Key identifies the row (provider, purpose, type and name).
	Key        string `json:"key"`
	ProviderID string `json:"providerId"`
	// Provider is the provider's name; empty for domain-wide records (DMARC).
	Provider   string     `json:"provider"`
	Purpose    string     `json:"purpose" enum:"dkim,return_path,spf,dmarc,verification"`
	Label      string     `json:"label"`
	Note       string     `json:"note"`
	Type       string     `json:"type"`
	Name       string     `json:"name"`
	Value      string     `json:"value"`
	Status     string     `json:"status" enum:"verified,pending,wrong_value,missing,error"`
	StatusText string     `json:"statusText"`
	Found      string     `json:"found,omitempty"`
	CheckedAt  *time.Time `json:"checkedAt,omitempty"`
	// ZoneLine is the record as a BIND zone-file line ("Copy as zone file").
	ZoneLine string `json:"zoneLine"`
}

// SendingDomainCounts summarize the records ("5 of 7 verified").
type SendingDomainCounts struct {
	Verified int `json:"verified"`
	Pending  int `json:"pending"`
	Missing  int `json:"missing"`
	Total    int `json:"total"`
}

// SendingDomainTracking is the custom link-tracking domain.
type SendingDomainTracking struct {
	Domain string `json:"domain"`
	// Suggested is the name ReputeCast proposes (go.<domain>).
	Suggested string `json:"suggested"`
	// Target is what the CNAME must point to (ReputeCast's public host).
	Target string `json:"target"`
	Status string `json:"status" enum:"none,pending,verified,wrong_value,missing,error"`
}

// SendingDomainDetail is a domain's records page (and the domain step).
type SendingDomainDetail struct {
	ID          string                 `json:"id"`
	Domain      string                 `json:"domain"`
	Note        string                 `json:"note"`
	Status      string                 `json:"status" enum:"pending,verified,partial,failed"`
	DNSHost     string                 `json:"dnsHost"`
	Nameservers []string               `json:"nameservers"`
	Cloudflare  bool                   `json:"cloudflare"`
	Records     []SendingDomainRecord  `json:"records"`
	Counts      SendingDomainCounts    `json:"counts"`
	CheckedAt   *time.Time             `json:"checkedAt,omitempty"`
	Providers   []SenderProviderColumn `json:"providers"`
	// DMARCTip explains the DMARC suggestion; `backticks` mark code.
	DMARCTip string                `json:"dmarcTip"`
	Tracking SendingDomainTracking `json:"tracking"`
	BIMI     string                `json:"bimi"`
	Senders  int                   `json:"senders"`
	// Notes say what to do in a provider's dashboard when ReputeCast can't
	// fetch its records.
	Notes     []ProviderSetupNote `json:"notes"`
	CanManage bool                `json:"canManage"`
}

func statusText(r Record) string {
	switch r.Status {
	case StatusVerified:
		return "Verified"
	case StatusWrong:
		if strings.EqualFold(r.Type, "TXT") {
			return "Pending · old value still cached"
		}
		return "Wrong value"
	case StatusMissing:
		return "Missing"
	case StatusError:
		return "Couldn't check"
	}
	return "Pending"
}

// Detail returns a domain's records, optionally only those one provider
// needs (plus the domain-wide ones), after a live check when refresh is set
// and the last one is older than 20 seconds.
func (s *Service) Detail(ctx context.Context, ws *store.Workspace, rawID, providerFilter string, refresh bool) (SendingDomainDetail, error) {
	id, err := parseID(rawID, "Sending domain")
	if err != nil {
		return SendingDomainDetail{}, err
	}
	d, err := s.Q.DomGet(ctx, store.DomGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return SendingDomainDetail{}, api.MapError(err, "Sending domain")
	}
	if refresh {
		if checked, err := s.CheckIfStale(ctx, d); err == nil {
			d = checked
		} else {
			s.log().WarnContext(ctx, "domain refresh", "domain", d.Domain, "err", err)
		}
	}
	return s.detailOf(ctx, ws, d, providerFilter)
}

// DetailOf builds the records page of a domain.
func (s *Service) detailOf(ctx context.Context, ws *store.Workspace, d store.SendingDomain, providerFilter string) (SendingDomainDetail, error) {
	provs, cols, err := s.providerColumns(ctx, ws.ID)
	if err != nil {
		return SendingDomainDetail{}, err
	}
	byID := map[string]store.Provider{}
	order := make([]string, 0, len(provs))
	for _, p := range provs {
		byID[p.ID.String()] = p
		order = append(order, p.ID.String())
	}
	recs := DecodeRecords(d.DnsRecords)
	kept := recs[:0:0]
	for _, r := range recs {
		if r.ProviderID != "" {
			if _, ok := byID[r.ProviderID]; !ok {
				continue // the provider was disconnected
			}
		}
		if providerFilter != "" && r.ProviderID != "" && r.ProviderID != providerFilter {
			continue
		}
		kept = append(kept, r)
	}
	SortRecords(kept, d.Domain, order)
	dkimNote := func(pid string) string {
		if p, ok := byID[pid]; ok && p.Adapter == "ses" {
			return "Easy DKIM, 2048-bit"
		}
		return ""
	}
	labels, notes := recordLabels(kept, d.Domain, dkimNote)
	out := SendingDomainDetail{ID: d.ID.String(), Domain: d.Domain, Note: d.Note, Status: d.Status, DNSHost: d.DnsHost, Nameservers: d.Nameservers,
		Cloudflare: d.DnsHost == "Cloudflare", Records: []SendingDomainRecord{}, CheckedAt: d.LastCheckedAt, Providers: []SenderProviderColumn{}, Notes: []ProviderSetupNote{}}
	if out.Nameservers == nil {
		out.Nameservers = []string{}
	}
	withRecords := map[string]bool{}
	for i, r := range kept {
		v := SendingDomainRecord{Key: r.key(), ProviderID: r.ProviderID, Purpose: r.Purpose, Label: labels[i], Note: notes[i], Type: r.Type, Name: r.Name, Value: r.Value,
			Status: r.Status, StatusText: statusText(r), Found: r.Found, CheckedAt: r.CheckedAt, ZoneLine: zoneLine(r)}
		if r.Purpose == PurposeDMARC && r.Status == StatusVerified && r.Found != "" {
			// Show what is published, where it is published.
			v.Value = r.Found
			if r.FoundName != "" {
				v.Name = r.FoundName
			}
			v.Found = ""
			v.ZoneLine = zoneLine(Record{Type: "TXT", Name: v.Name, Value: v.Value})
		}
		if p, ok := byID[r.ProviderID]; ok {
			v.Provider = p.Name
			withRecords[r.ProviderID] = true
		}
		out.Records = append(out.Records, v)
		out.Counts.Total++
		switch r.Status {
		case StatusVerified:
			out.Counts.Verified++
		case StatusMissing:
			out.Counts.Missing++
		default:
			out.Counts.Pending++
		}
	}
	for _, c := range cols {
		if withRecords[c.ID] {
			out.Providers = append(out.Providers, c)
		}
	}
	det := decodeAuthDetail(d.AuthDetail)
	sum := decodeSummary(d.AuthSummary)
	out.BIMI = sum.BIMI
	out.DMARCTip = dmarcTip(d.Domain, kept, det, out.Providers)
	target := ""
	if s.TrackingTarget != nil {
		target = s.TrackingTarget()
	}
	out.Tracking = SendingDomainTracking{Domain: d.TrackingDomain, Suggested: "go." + d.Domain, Target: target, Status: d.TrackingStatus}
	if out.Tracking.Status == "" {
		out.Tracking.Status = "none"
	}
	senders, err := s.Q.SenderListByDomain(ctx, d.ID)
	if err != nil {
		return SendingDomainDetail{}, err
	}
	out.Senders = len(senders)
	for _, p := range provs {
		if !withRecords[p.ID.String()] && (providerFilter == "" || providerFilter == p.ID.String()) {
			out.Notes = append(out.Notes, ProviderSetupNote{ProviderID: p.ID.String(), Provider: p.Name,
				Text: p.Name + " has no records for " + d.Domain + " yet. Fetch them to send as this domain through " + state.Short(p.Name, p.Adapter) + "."})
		}
	}
	if pr := api.PrincipalFrom(ctx); pr != nil {
		out.CanManage = pr.Can(authz.DomainsManage)
	}
	return out, nil
}

// dmarcTip explains the DMARC record: why ReputeCast suggests its policy,
// or which policy applies already.
func dmarcTip(domain string, recs []Record, det AuthDetail, provs []SenderProviderColumn) string {
	var dm *Record
	for i := range recs {
		if recs[i].Purpose == PurposeDMARC {
			dm = &recs[i]
		}
	}
	if dm == nil {
		return ""
	}
	reports := ""
	rec := dm.Value
	if dm.Status == StatusVerified && dm.Found != "" {
		rec = dm.Found
	}
	if strings.Contains(strings.ToLower(rec), "hostrepute") {
		reports = " Reports go to HostRepute and appear in Deliverability."
	}
	if dm.Status == StatusVerified {
		policy := det.DMARCPolicy
		if policy == "" {
			policy = tagValue(rec, "p")
		}
		if det.DMARCInherited {
			return domain + " follows `p=" + policy + "` from " + dnscheck.OrgDomain(domain) + "'s DMARC record." + reports
		}
		return domain + " publishes `p=" + policy + "`." + reports
	}
	names := make([]string, 0, len(provs))
	for _, p := range provs {
		names = append(names, p.Short)
	}
	verb := "is"
	if len(names) > 1 {
		verb = "are"
	}
	who := andList(names)
	if who == "" {
		who, verb = "no provider", "is"
	}
	return "start with `p=none` if other services also send as " + domain + "; ReputeCast suggested `p=" + tagValue(dm.Value, "p") + "` because only " + who + " " + verb + " configured." + reports
}

func tagValue(rec, tag string) string {
	for _, part := range strings.Split(rec, ";") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && strings.EqualFold(strings.TrimSpace(k), tag) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func andList(xs []string) string {
	switch len(xs) {
	case 0:
		return ""
	case 1:
		return xs[0]
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " and " + xs[len(xs)-1]
}

// DomainByName is the workspace's sending domain with that name, if any.
func (s *Service) DomainByName(ctx context.Context, wsID uuid.UUID, name string) (store.SendingDomain, error) {
	return s.Q.DomGetByName(ctx, store.DomGetByNameParams{WorkspaceID: wsID, Domain: strings.ToLower(name)})
}

// DetailFor builds the records page of a domain, for the connect flow.
func (s *Service) DetailFor(ctx context.Context, ws *store.Workspace, d store.SendingDomain, providerFilter string) (SendingDomainDetail, error) {
	return s.detailOf(ctx, ws, d, providerFilter)
}
