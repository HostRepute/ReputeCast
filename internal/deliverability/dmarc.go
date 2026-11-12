package deliverability

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/netip"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/hostrepute"
	"github.com/hostrepute/reputecast/internal/i18n"
	provstate "github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// ptrSuffixes recognise provider-operated sending IPs by their reverse DNS,
// for sources that aren't dedicated IPs of a connected provider.
var ptrSuffixes = map[string][]string{
	"ses":       {"amazonses.com"},
	"resend":    {"amazonses.com"},
	"postmark":  {"mtasv.net"},
	"mailgun":   {"mailgun.net", "mailgun.org"},
	"sendgrid":  {"sendgrid.net"},
	"sparkpost": {"sparkpostmail.com"},
	"mailjet":   {"mailjet.com"},
	"brevo":     {"sendinblue.com", "brevo.com"},
	"azureacs":  {"azurecomm.net"},
}

type dmarcRecord = hostrepute.DMARCRecord

// sourceKey groups records: a provider, or an unknown IP.
type sourceKey struct {
	provider uuid.UUID
	ip       string
}

// attribute names the provider that sent from ip: a dedicated IP of a
// connected provider, else a provider whose PTR domain matches.
func (st *state) attribute(ip, ptr string) (store.Provider, bool) {
	for _, p := range st.providers {
		for _, a := range p.Ips {
			if a == ip {
				return p, true
			}
		}
	}
	ptr = strings.ToLower(strings.TrimSuffix(ptr, "."))
	if ptr == "" {
		return store.Provider{}, false
	}
	for _, p := range st.providers {
		for _, suf := range ptrSuffixes[p.Adapter] {
			if ptr == suf || strings.HasSuffix(ptr, "."+suf) {
				return p, true
			}
		}
	}
	return store.Provider{}, false
}

// ownedDomain reports whether a From domain belongs to the workspace.
func (st *state) ownedDomain(d string) bool { return ownsDomain(st.cands, d) }

// ownsDomain reports whether a domain is one of the workspace's (its
// sending domains and their organizational domains), by organizational
// domain.
func ownsDomain(cands []hostrepute.Candidate, d string) bool {
	org := dnscheck.OrgDomain(d)
	for _, c := range cands {
		if c.Type == hostrepute.AssetDomain && dnscheck.OrgDomain(c.Asset) == org {
			return true
		}
	}
	return false
}

// viaRank orders where reports came from: the summary names the best
// source among the reports it counts.
var viaRank = map[string]int{"upload": 1, "inbound": 2, "hostrepute": 3}

type dmarcAgg struct {
	row      DeliverabilityDMARCSource
	spfPass  int64
	dkimPass int64
	spfRaw   map[string]int64
	dkimRaw  map[string]int64
	disp     map[string]int64
}

func dmarcPassed(r dmarcRecord) bool { return r.DKIM == "pass" || r.SPF == "pass" }

// dmarcSummary aggregates the last 7 days of reports per sending source.
func (st *state) dmarcSummary() DeliverabilityDMARCSummary {
	out := DeliverabilityDMARCSummary{Via: "none", Days: hostrepute.DMARCDays, Sources: []DeliverabilityDMARCSource{}}
	if st.conn != nil {
		out.Rua = st.conn.DmarcRua
	}
	out.Domain = st.mainDomain()
	if c, ok := st.checks[strings.ToLower(out.Domain)]; ok {
		if a := authSummary(c); a != nil {
			out.Policy = a.DMARC
		}
	}
	aggs := map[sourceKey]*dmarcAgg{}
	var order []sourceKey
	var pass int64
	for _, rep := range st.dmarc {
		if rep.Via == "hostrepute" && st.conn == nil {
			continue
		}
		var recs []dmarcRecord
		if json.Unmarshal(rep.Records, &recs) != nil {
			continue
		}
		counted := false
		for _, r := range recs {
			if r.HeaderFrom != "" && !st.ownedDomain(r.HeaderFrom) {
				continue
			}
			counted = true
			out.Messages += r.Count
			if dmarcPassed(r) {
				pass += r.Count
			}
			p, known := st.attribute(r.SourceIP, r.PTR)
			k := sourceKey{ip: r.SourceIP}
			if known {
				k = sourceKey{provider: p.ID}
				if p.IpMode == "dedicated" {
					k.ip = r.SourceIP
				}
			}
			a, ok := aggs[k]
			if !ok {
				a = &dmarcAgg{spfRaw: map[string]int64{}, dkimRaw: map[string]int64{}, disp: map[string]int64{}}
				a.row.Known = known
				a.row.IP = r.SourceIP
				if known {
					a.row.ProviderID, a.row.Adapter = p.ID.String(), p.Adapter
					a.row.Label = p.Name
					if p.IpMode == "dedicated" {
						a.row.Label = provstate.Short(p.Name, p.Adapter) + " · " + r.SourceIP
					} else {
						a.row.IP = ""
					}
				} else {
					a.row.Label = text(st.l, "label.unknownSource", i18n.Args{"ip": r.SourceIP})
				}
				aggs[k] = a
				order = append(order, k)
			}
			a.row.Messages += r.Count
			if r.SPF == "pass" {
				a.spfPass += r.Count
			} else {
				a.spfRaw[firstNonEmpty(r.SPFResult, "fail")] += r.Count
			}
			if r.DKIM == "pass" {
				a.dkimPass += r.Count
			} else {
				a.dkimRaw[firstNonEmpty(r.DKIMResult, "fail")] += r.Count
			}
			a.disp[firstNonEmpty(r.Disposition, "none")] += r.Count
		}
		if counted {
			out.Reports++
			if viaRank[rep.Via] > viaRank[out.Via] {
				out.Via = rep.Via
			}
		}
	}
	if out.Messages > 0 {
		v := round(float64(pass)/float64(out.Messages)*100, 1)
		out.PassRate = &v
	}
	for _, k := range order {
		a := aggs[k]
		a.row.SPF = verdictOf(a.spfPass, a.spfRaw, a.row.Messages)
		a.row.DKIM = verdictOf(a.dkimPass, a.dkimRaw, a.row.Messages)
		switch {
		case a.row.SPF == "aligned" || a.row.DKIM == "aligned":
			a.row.Result = "pass"
		case a.disp["reject"] >= a.row.Messages/2:
			a.row.Result = "rejected"
		case a.disp["quarantine"] >= a.row.Messages/2:
			a.row.Result = "quarantined"
		default:
			a.row.Result = "fail"
		}
		out.Sources = append(out.Sources, a.row)
	}
	// Known providers by volume first, then unknown sources by volume.
	sort.SliceStable(out.Sources, func(i, j int) bool {
		a, b := out.Sources[i], out.Sources[j]
		if a.Known != b.Known {
			return a.Known
		}
		return a.Messages > b.Messages
	})
	for i := range out.Sources {
		if !out.Sources[i].Known {
			u := out.Sources[i]
			out.Unknown = &u
			break
		}
	}
	return out
}

// verdictOf is "aligned" when most messages passed, else the most common
// raw result (fail, softfail, none…).
func verdictOf(passed int64, raw map[string]int64, total int64) string {
	if total == 0 || passed*2 >= total {
		return "aligned"
	}
	best, n := "fail", int64(-1)
	for k, v := range raw {
		if v > n || (v == n && k < best) {
			best, n = k, v
		}
	}
	return best
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// dmarcAligned is the DMARC pass rate (percent) of a domain's mail over the
// last 7 days; nil without reports.
func (st *state) dmarcAligned(domain string) *float64 {
	var total, pass int64
	org := dnscheck.OrgDomain(domain)
	for _, rep := range st.dmarc {
		if rep.Via == "hostrepute" && st.conn == nil {
			continue
		}
		var recs []dmarcRecord
		if json.Unmarshal(rep.Records, &recs) != nil {
			continue
		}
		for _, r := range recs {
			hf := strings.ToLower(r.HeaderFrom)
			// An organization domain covers its subdomains' mail.
			if hf != domain && (domain != org || dnscheck.OrgDomain(hf) != org) {
				continue
			}
			total += r.Count
			if dmarcPassed(r) {
				pass += r.Count
			}
		}
	}
	if total == 0 {
		return nil
	}
	v := round(float64(pass)/float64(total)*100, 1)
	return &v
}

// DeliverabilityDMARCSourceDetail is "Investigate source": everything the reports say
// about one sending IP.
type DeliverabilityDMARCSourceDetail struct {
	IP          string   `json:"ip"`
	PTR         string   `json:"ptr,omitempty"`
	Known       bool     `json:"known"`
	Provider    string   `json:"provider,omitempty"`
	Messages    int64    `json:"messages"`
	Reports     int      `json:"reports"`
	Reporters   []string `json:"reporters"`
	HeaderFrom  []string `json:"headerFrom"`
	SPFDomains  []string `json:"spfDomains"`
	DKIMDomains []string `json:"dkimDomains"`
	SPF         string   `json:"spf"`
	DKIM        string   `json:"dkim"`
	Disposition string   `json:"disposition"`
	// Policy is the published DMARC policy that applied (quarantine).
	Policy    string    `json:"policy,omitempty"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
}

// DMARCSource describes one sending IP from the last 7 days of reports.
func (s *Service) DMARCSource(ctx context.Context, ws *store.Workspace, ip string) (DeliverabilityDMARCSourceDetail, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return DeliverabilityDMARCSourceDetail{}, api.Invalid("path.ip", say(ctx, "problem.ip", nil))
	}
	ip = addr.String()
	st, err := s.load(ctx, ws)
	if err != nil {
		return DeliverabilityDMARCSourceDetail{}, err
	}
	out := DeliverabilityDMARCSourceDetail{IP: ip, Reporters: []string{}, HeaderFrom: []string{}, SPFDomains: []string{}, DKIMDomains: []string{}}
	add := func(list *[]string, v string) {
		if v != "" && !contains(*list, v) {
			*list = append(*list, v)
		}
	}
	var spfPass, dkimPass int64
	spfRaw, dkimRaw, disp := map[string]int64{}, map[string]int64{}, map[string]int64{}
	for _, rep := range st.dmarc {
		var recs []dmarcRecord
		if json.Unmarshal(rep.Records, &recs) != nil {
			continue
		}
		found := false
		for _, r := range recs {
			if r.SourceIP != ip {
				continue
			}
			found = true
			out.Messages += r.Count
			if r.PTR != "" {
				out.PTR = r.PTR
			}
			add(&out.HeaderFrom, r.HeaderFrom)
			add(&out.SPFDomains, r.SPFDomain)
			add(&out.DKIMDomains, r.DKIMDomain)
			if r.SPF == "pass" {
				spfPass += r.Count
			} else {
				spfRaw[firstNonEmpty(r.SPFResult, "fail")] += r.Count
			}
			if r.DKIM == "pass" {
				dkimPass += r.Count
			} else {
				dkimRaw[firstNonEmpty(r.DKIMResult, "fail")] += r.Count
			}
			disp[firstNonEmpty(r.Disposition, "none")] += r.Count
		}
		if !found {
			continue
		}
		out.Reports++
		add(&out.Reporters, rep.OrgName)
		if out.FirstSeen.IsZero() || rep.DateBegin.Before(out.FirstSeen) {
			out.FirstSeen = rep.DateBegin
		}
		if rep.DateEnd.After(out.LastSeen) {
			out.LastSeen = rep.DateEnd
		}
		var pol map[string]string
		if json.Unmarshal(rep.Policy, &pol) == nil && pol["p"] != "" {
			out.Policy = pol["p"]
		}
	}
	if out.Reports == 0 {
		return out, api.NotFound("DMARC source")
	}
	if p, ok := st.attribute(ip, out.PTR); ok {
		out.Known, out.Provider = true, p.Name
	}
	out.SPF = verdictOf(spfPass, spfRaw, out.Messages)
	out.DKIM = verdictOf(dkimPass, dkimRaw, out.Messages)
	best, n := "none", int64(-1)
	for k, v := range disp {
		if v > n {
			best, n = k, v
		}
	}
	out.Disposition = best
	return out, nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(x, v) {
			return true
		}
	}
	return false
}

// feedback is an aggregate report: RFC 7489 appendix C or RFC 9990, whose
// elements are in the urn:ietf:params:xml:ns:dmarc-2.0 namespace (the tags
// carry no namespace, so they match either).
type feedback struct {
	XMLName  xml.Name `xml:"feedback"`
	Metadata struct {
		OrgName   string `xml:"org_name"`
		ReportID  string `xml:"report_id"`
		DateRange struct {
			Begin int64 `xml:"begin"`
			End   int64 `xml:"end"`
		} `xml:"date_range"`
	} `xml:"report_metadata"`
	// Policy is the record receivers applied; RFC 9990 removed pct.
	Policy struct {
		Domain          string `xml:"domain"`
		ADKIM           string `xml:"adkim"`
		ASPF            string `xml:"aspf"`
		P               string `xml:"p"`
		SP              string `xml:"sp"`
		NP              string `xml:"np"`
		FO              string `xml:"fo"`
		Testing         string `xml:"testing"`
		DiscoveryMethod string `xml:"discovery_method"`
	} `xml:"policy_published"`
	Records []struct {
		Row struct {
			SourceIP  string `xml:"source_ip"`
			Count     int64  `xml:"count"`
			Evaluated struct {
				Disposition string `xml:"disposition"`
				DKIM        string `xml:"dkim"`
				SPF         string `xml:"spf"`
			} `xml:"policy_evaluated"`
		} `xml:"row"`
		Identifiers struct {
			HeaderFrom string `xml:"header_from"`
		} `xml:"identifiers"`
		Auth struct {
			DKIM []struct {
				Domain string `xml:"domain"`
				Result string `xml:"result"`
			} `xml:"dkim"`
			SPF []struct {
				Domain string `xml:"domain"`
				Result string `xml:"result"`
			} `xml:"spf"`
		} `xml:"auth_results"`
	} `xml:"record"`
}

// MaxReportBytes caps an uploaded report, compressed and decompressed.
const MaxReportBytes = 20 << 20

// ParseReport reads an aggregate report: XML, gzip'd XML or a zip with one
// XML file, as mailbox providers send them.
func ParseReport(data []byte) (hostrepute.DMARCReport, error) {
	switch {
	case bytes.HasPrefix(data, []byte{0x1f, 0x8b}):
		zr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return hostrepute.DMARCReport{}, err
		}
		if data, err = io.ReadAll(io.LimitReader(zr, MaxReportBytes+1)); err != nil {
			return hostrepute.DMARCReport{}, err
		}
	case bytes.HasPrefix(data, []byte("PK\x03\x04")):
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return hostrepute.DMARCReport{}, err
		}
		var found bool
		for _, f := range zr.File {
			if !strings.HasSuffix(strings.ToLower(f.Name), ".xml") {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return hostrepute.DMARCReport{}, err
			}
			data, err = io.ReadAll(io.LimitReader(rc, MaxReportBytes+1))
			_ = rc.Close()
			if err != nil {
				return hostrepute.DMARCReport{}, err
			}
			found = true
			break
		}
		if !found {
			return hostrepute.DMARCReport{}, &reportError{key: "problem.zipNoXML", english: "the zip file has no .xml report"}
		}
	}
	if len(data) > MaxReportBytes {
		return hostrepute.DMARCReport{}, &reportError{key: "problem.tooLarge", args: i18n.Args{"mb": MaxReportBytes >> 20}, english: "the report is larger than 20 MB"}
	}
	var fb feedback
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	if err := dec.Decode(&fb); err != nil {
		return hostrepute.DMARCReport{}, &reportError{key: "problem.notDMARC", args: i18n.Args{"error": err.Error()},
			english: "this isn't a DMARC aggregate report: " + err.Error(), cause: err}
	}
	if fb.Metadata.OrgName == "" || fb.Metadata.ReportID == "" || len(fb.Records) == 0 {
		return hostrepute.DMARCReport{}, &reportError{key: "problem.notDMARCEmpty", english: "this isn't a DMARC aggregate report: no report_metadata or records"}
	}
	// The policy as stored: the tags RFC 9990 reports, empty ones left out.
	pp := fb.Policy
	policy := map[string]string{}
	for _, kv := range [][2]string{{"domain", pp.Domain}, {"p", pp.P}, {"sp", pp.SP}, {"np", pp.NP}, {"adkim", pp.ADKIM}, {"aspf", pp.ASPF},
		{"fo", pp.FO}, {"testing", pp.Testing}, {"discovery_method", pp.DiscoveryMethod}} {
		if v := strings.TrimSpace(kv[1]); v != "" {
			policy[kv[0]] = v
		}
	}
	out := hostrepute.DMARCReport{
		Domain: strings.ToLower(fb.Policy.Domain), OrgName: fb.Metadata.OrgName, ReportID: fb.Metadata.ReportID,
		DateBegin: time.Unix(fb.Metadata.DateRange.Begin, 0).UTC(), DateEnd: time.Unix(fb.Metadata.DateRange.End, 0).UTC(),
		Policy: policy, Records: []hostrepute.DMARCRecord{},
	}
	for _, r := range fb.Records {
		disposition := strings.ToLower(r.Row.Evaluated.Disposition)
		if disposition == "pass" {
			// RFC 9990's "pass": no action, the mail passed DMARC under an
			// enforcing policy. Delivered, like "none".
			disposition = "none"
		}
		rec := hostrepute.DMARCRecord{
			SourceIP: strings.TrimSpace(r.Row.SourceIP), Count: r.Row.Count, Disposition: disposition,
			DKIM: strings.ToLower(r.Row.Evaluated.DKIM), SPF: strings.ToLower(r.Row.Evaluated.SPF), HeaderFrom: strings.ToLower(r.Identifiers.HeaderFrom),
			DKIMResult: "none", SPFResult: "none",
		}
		if len(r.Auth.DKIM) > 0 {
			rec.DKIMResult, rec.DKIMDomain = strings.ToLower(r.Auth.DKIM[0].Result), strings.ToLower(r.Auth.DKIM[0].Domain)
		}
		if len(r.Auth.SPF) > 0 {
			rec.SPFResult, rec.SPFDomain = strings.ToLower(r.Auth.SPF[0].Result), strings.ToLower(r.Auth.SPF[0].Domain)
		}
		out.Records = append(out.Records, rec)
	}
	if out.Domain == "" && len(out.Records) > 0 {
		out.Domain = out.Records[0].HeaderFrom
	}
	return out, nil
}

// reportError is why a file isn't a report ParseReport can read, in words
// for the person who uploaded it.
type reportError struct {
	key     string
	args    i18n.Args
	english string
	cause   error
}

func (e *reportError) Error() string { return e.english }
func (e *reportError) Unwrap() error { return e.cause }

// DeliverabilityDMARCUpload is the result of uploading a report.
type DeliverabilityDMARCUpload struct {
	OrgName  string    `json:"orgName"`
	ReportID string    `json:"reportId"`
	Domain   string    `json:"domain"`
	Messages int64     `json:"messages"`
	Records  int       `json:"records"`
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
}

// UploadReport stores an aggregate report pasted or uploaded by hand.
func (s *Service) UploadReport(ctx context.Context, ws *store.Workspace, data []byte) (DeliverabilityDMARCUpload, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return DeliverabilityDMARCUpload{}, api.Invalid("body", say(ctx, "problem.chooseReport", nil))
	}
	rep, err := ParseReport(data)
	var re *reportError
	switch {
	case errors.As(err, &re):
		return DeliverabilityDMARCUpload{}, api.Invalid("body", say(ctx, re.key, re.args))
	case err != nil:
		return DeliverabilityDMARCUpload{}, api.Invalid("body", strings.ToUpper(err.Error()[:1])+err.Error()[1:]+".")
	}
	out := DeliverabilityDMARCUpload{OrgName: rep.OrgName, ReportID: rep.ReportID, Domain: rep.Domain, Records: len(rep.Records), From: rep.DateBegin, To: rep.DateEnd}
	for _, r := range rep.Records {
		out.Messages += r.Count
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := s.StoreReport(ctx, tx, ws, rep, "upload"); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &ws.ID, Action: "dmarc_report.upload", TargetType: "dmarc_report",
			TargetID: rep.OrgName + "/" + rep.ReportID, TargetName: "DMARC report from " + rep.OrgName,
			After: map[string]any{"domain": rep.Domain, "records": len(rep.Records), "messages": out.Messages}})
	})
	if errors.Is(err, ErrNotOwnedDomain) {
		return DeliverabilityDMARCUpload{}, api.Invalid("body", say(ctx, "problem.notSendingDomain", i18n.Args{"domain": rep.Domain}))
	}
	return out, err
}

// ErrNotOwnedDomain is a report about a domain that isn't one of the
// workspace's sending domains (or their organizational domains).
var ErrNotOwnedDomain = errors.New("deliverability: the report is about a domain the workspace doesn't send from")

// StoreReport stores an aggregate report in tx, received by via ("upload",
// "inbound" for one that arrived at dmarc@ the receiving domain, M58): a
// report already stored (same reporter and report id) is replaced. A
// report about another domain is ErrNotOwnedDomain. The caller audits.
func (s *Service) StoreReport(ctx context.Context, tx pgx.Tx, ws *store.Workspace, rep hostrepute.DMARCReport, via string) error {
	q := store.New(tx)
	cands, err := hostrepute.Candidates(ctx, q, ws.ID)
	if err != nil {
		return err
	}
	if !ownsDomain(cands, rep.Domain) {
		return ErrNotOwnedDomain
	}
	policy, _ := json.Marshal(rep.Policy)
	records, _ := json.Marshal(rep.Records)
	return q.UpsertDeliverabilityDMARCReport(ctx, store.UpsertDeliverabilityDMARCReportParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Domain: rep.Domain, OrgName: rep.OrgName, ReportID: rep.ReportID,
		DateBegin: rep.DateBegin, DateEnd: rep.DateEnd, Policy: policy, Records: records, Via: via, ReceivedAt: s.now(),
	})
}

// dmarcCollect is where DMARC reports arrive while the workspace receives
// mail (M58): dmarc@ its receiving domain, when the last report came, and
// the external destination record each sending domain under another
// organizational domain needs, as the last check found it.
func (s *Service) dmarcCollect(ctx context.Context, st *state) (*DeliverabilityDMARCCollect, error) {
	if s.Inbound == nil {
		return nil, nil
	}
	receiving, domain, last, err := s.Inbound(ctx, st.ws)
	if err != nil || !receiving || domain == "" {
		return nil, err
	}
	out := &DeliverabilityDMARCCollect{Address: "dmarc@" + domain, LastReceivedAt: last, ExternalRecords: []DeliverabilityDMARCExternalRecord{}}
	if out.LastReceivedAt == nil {
		for _, rep := range st.dmarc {
			if rep.Via == "inbound" && (out.LastReceivedAt == nil || rep.ReceivedAt.After(*out.LastReceivedAt)) {
				at := rep.ReceivedAt
				out.LastReceivedAt = &at
			}
		}
	}
	for _, d := range st.domains {
		if !needsReportRecord(d.Domain, domain) {
			continue
		}
		rec := DeliverabilityDMARCExternalRecord{Domain: d.Domain, Name: reportRecordName(d.Domain, domain), Type: "TXT", Value: reportRecordValue, Status: "unknown"}
		if chk, ok := st.checks[strings.ToLower(d.Domain)]; ok {
			var r struct {
				Report *reportRecordCheck `json:"dmarcReport"`
			}
			if json.Unmarshal(chk.Result, &r) == nil && r.Report != nil && strings.EqualFold(r.Report.Name, rec.Name) {
				rec.Status = r.Report.Status
			}
		}
		out.ExternalRecords = append(out.ExternalRecords, rec)
	}
	return out, nil
}

// reportRecordValue is the external destination record's content: the
// receiving domain agrees to take the sending domain's reports.
const reportRecordValue = "v=DMARC1"

// needsReportRecord reports whether reports for a sending domain sent to
// dest need an external destination record: dest is under another
// organizational domain (RFC 7489 §7.1).
func needsReportRecord(domain, dest string) bool {
	return dnscheck.OrgDomain(domain) != dnscheck.OrgDomain(dest)
}

// reportRecordName is where the external destination record goes:
// "<sending domain>._report._dmarc.<receiving domain>".
func reportRecordName(domain, dest string) string {
	return strings.ToLower(domain) + "._report._dmarc." + strings.ToLower(dest)
}

// reportRecordCheck is the stored lookup of an external destination
// record (authChecks): found, missing, or unknown when DNS didn't answer.
type reportRecordCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// checkReportRecord looks the external destination record of domain's
// reports at dest up.
func (s *Service) checkReportRecord(ctx context.Context, domain, dest string) reportRecordCheck {
	out := reportRecordCheck{Name: reportRecordName(domain, dest), Status: "unknown"}
	txts, err := s.DNS.TXT(ctx, out.Name)
	switch {
	case err == nil && slices.ContainsFunc(txts, isReportRecord):
		out.Status = "found"
	case err == nil || dnscheck.IsNotFound(err):
		out.Status = "missing"
	}
	return out
}

// isReportRecord: a TXT record that starts with v=DMARC1.
func isReportRecord(txt string) bool {
	v, _, _ := strings.Cut(strings.TrimSpace(txt), ";")
	k, val, ok := strings.Cut(v, "=")
	return ok && strings.EqualFold(strings.TrimSpace(k), "v") && strings.EqualFold(strings.TrimSpace(val), "DMARC1")
}
