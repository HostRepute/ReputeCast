package deliverability

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/hostrepute"
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
func (st *state) attribute(ip, ptr string) (store.HostReputeActiveProvidersRow, bool) {
	for _, p := range st.providers {
		for _, a := range p.Ips {
			if a == ip {
				return p, true
			}
		}
	}
	ptr = strings.ToLower(strings.TrimSuffix(ptr, "."))
	if ptr == "" {
		return store.HostReputeActiveProvidersRow{}, false
	}
	for _, p := range st.providers {
		for _, suf := range ptrSuffixes[p.Adapter] {
			if ptr == suf || strings.HasSuffix(ptr, "."+suf) {
				return p, true
			}
		}
	}
	return store.HostReputeActiveProvidersRow{}, false
}

// ownedDomain reports whether a From domain belongs to the workspace.
func (st *state) ownedDomain(d string) bool {
	org := dnscheck.OrgDomain(d)
	for _, c := range st.cands {
		if c.Type == hostrepute.AssetDomain && dnscheck.OrgDomain(c.Asset) == org {
			return true
		}
	}
	return false
}

type dmarcAgg struct {
	row      DMARCSourceRow
	spfPass  int64
	dkimPass int64
	spfRaw   map[string]int64
	dkimRaw  map[string]int64
	disp     map[string]int64
}

func dmarcPassed(r dmarcRecord) bool { return r.DKIM == "pass" || r.SPF == "pass" }

// dmarcSummary aggregates the last 7 days of reports per sending source.
func (st *state) dmarcSummary() DMARCSummary {
	out := DMARCSummary{Via: "none", Days: hostrepute.DMARCDays, Sources: []DMARCSourceRow{}}
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
						a.row.Label = campaigns.ShortName(p.Name, p.Adapter) + " · " + r.SourceIP
					} else {
						a.row.IP = ""
					}
				} else {
					a.row.Label = "Unknown · " + r.SourceIP
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
			if out.Via == "none" || rep.Via == "hostrepute" {
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
			if hf != domain && !(domain == org && dnscheck.OrgDomain(hf) == org) {
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

// DMARCSourceDetail is "Investigate source": everything the reports say
// about one sending IP.
type DMARCSourceDetail struct {
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
func (s *Service) DMARCSource(ctx context.Context, ws *store.Workspace, ip string) (DMARCSourceDetail, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return DMARCSourceDetail{}, api.Invalid("path.ip", "Give an IPv4 or IPv6 address.")
	}
	ip = addr.String()
	st, err := s.load(ctx, ws.ID, ws.Timezone)
	if err != nil {
		return DMARCSourceDetail{}, err
	}
	out := DMARCSourceDetail{IP: ip, Reporters: []string{}, HeaderFrom: []string{}, SPFDomains: []string{}, DKIMDomains: []string{}}
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

// feedback is an RFC 7489 appendix C aggregate report.
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
	Policy struct {
		Domain string `xml:"domain"`
		ADKIM  string `xml:"adkim"`
		ASPF   string `xml:"aspf"`
		P      string `xml:"p"`
		SP     string `xml:"sp"`
		Pct    string `xml:"pct"`
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
			rc.Close()
			if err != nil {
				return hostrepute.DMARCReport{}, err
			}
			found = true
			break
		}
		if !found {
			return hostrepute.DMARCReport{}, errors.New("the zip file has no .xml report")
		}
	}
	if len(data) > MaxReportBytes {
		return hostrepute.DMARCReport{}, errors.New("the report is larger than 20 MB")
	}
	var fb feedback
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	if err := dec.Decode(&fb); err != nil {
		return hostrepute.DMARCReport{}, fmt.Errorf("this isn't a DMARC aggregate report: %v", err)
	}
	if fb.Metadata.OrgName == "" || fb.Metadata.ReportID == "" || len(fb.Records) == 0 {
		return hostrepute.DMARCReport{}, errors.New("this isn't a DMARC aggregate report: no report_metadata or records")
	}
	out := hostrepute.DMARCReport{
		Domain: strings.ToLower(fb.Policy.Domain), OrgName: fb.Metadata.OrgName, ReportID: fb.Metadata.ReportID,
		DateBegin: time.Unix(fb.Metadata.DateRange.Begin, 0).UTC(), DateEnd: time.Unix(fb.Metadata.DateRange.End, 0).UTC(),
		Policy:  map[string]string{"domain": fb.Policy.Domain, "p": fb.Policy.P, "sp": fb.Policy.SP, "pct": fb.Policy.Pct, "adkim": fb.Policy.ADKIM, "aspf": fb.Policy.ASPF},
		Records: []hostrepute.DMARCRecord{},
	}
	for _, r := range fb.Records {
		rec := hostrepute.DMARCRecord{
			SourceIP: strings.TrimSpace(r.Row.SourceIP), Count: r.Row.Count, Disposition: strings.ToLower(r.Row.Evaluated.Disposition),
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

// DMARCUpload is the result of uploading a report.
type DMARCUpload struct {
	OrgName  string    `json:"orgName"`
	ReportID string    `json:"reportId"`
	Domain   string    `json:"domain"`
	Messages int64     `json:"messages"`
	Records  int       `json:"records"`
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
}

// UploadReport stores an aggregate report pasted or uploaded by hand.
func (s *Service) UploadReport(ctx context.Context, ws *store.Workspace, data []byte) (DMARCUpload, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return DMARCUpload{}, api.Invalid("body", "Choose a DMARC aggregate report: an .xml file, or the .xml.gz or .zip a mailbox provider sent.")
	}
	rep, err := ParseReport(data)
	if err != nil {
		return DMARCUpload{}, api.Invalid("body", strings.ToUpper(err.Error()[:1])+err.Error()[1:]+".")
	}
	st, err := s.load(ctx, ws.ID, ws.Timezone)
	if err != nil {
		return DMARCUpload{}, err
	}
	if !st.ownedDomain(rep.Domain) {
		return DMARCUpload{}, api.Invalid("body", "This report is for "+rep.Domain+", which isn't a sending domain of this workspace.")
	}
	out := DMARCUpload{OrgName: rep.OrgName, ReportID: rep.ReportID, Domain: rep.Domain, Records: len(rep.Records), From: rep.DateBegin, To: rep.DateEnd}
	for _, r := range rep.Records {
		out.Messages += r.Count
	}
	policy, _ := json.Marshal(rep.Policy)
	records, _ := json.Marshal(rep.Records)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.UpsertDeliverabilityDMARCReport(ctx, store.UpsertDeliverabilityDMARCReportParams{
			ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Domain: rep.Domain, OrgName: rep.OrgName, ReportID: rep.ReportID,
			DateBegin: rep.DateBegin, DateEnd: rep.DateEnd, Policy: policy, Records: records, Via: "upload", ReceivedAt: s.now(),
		}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "dmarc_report.upload", TargetType: "dmarc_report",
			TargetID: rep.OrgName + "/" + rep.ReportID, TargetName: "DMARC report from " + rep.OrgName,
			After: map[string]any{"domain": rep.Domain, "records": len(rep.Records), "messages": out.Messages}})
	})
	return out, err
}
