package domains

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Record is one DNS record a sending domain needs, stored in
// sending_domains.dns_records. ProviderID names the sending provider that
// needs it; it is empty for records every provider relies on (DMARC).
//
// The JSON shape is shared: Deliverability (dnscheck requirements) reads
// provider_id, purpose, type, name and value, and the route tester reads the
// DKIM selectors. Fields are only ever added.
type Record struct {
	ProviderID string `json:"provider_id,omitempty"`
	// Purpose is dkim, return_path, spf, dmarc or verification.
	Purpose string `json:"purpose"`
	Type    string `json:"type"` // CNAME, TXT, MX
	Name    string `json:"name"`
	Value   string `json:"value"`
	// Status is the live DNS check: pending (not checked yet), verified,
	// wrong_value, missing or error.
	Status string `json:"status,omitempty"`
	// Found is what DNS answered when it differs from Value (for DMARC:
	// the published policy, at FoundName when inherited).
	Found     string `json:"found,omitempty"`
	FoundName string `json:"found_name,omitempty"`
	// ProviderStatus is the provider's own verdict when its API reports one.
	ProviderStatus string `json:"provider_status,omitempty"`
	// Source says who asked for the record: "provider" (its domain API) or
	// "reputecast" (DMARC, the merged SPF record, manual-DNS providers).
	Source string `json:"source,omitempty"`
	// Mechanisms are the SPF terms a provider needs on the sending domain
	// itself (purpose spf, Name = the domain): Value is the domain's current
	// SPF record with them merged in.
	Mechanisms []string   `json:"mechanisms,omitempty"`
	CheckedAt  *time.Time `json:"checked_at,omitempty"`
}

// Record purposes.
const (
	PurposeDKIM         = "dkim"
	PurposeReturnPath   = "return_path"
	PurposeSPF          = "spf"
	PurposeDMARC        = "dmarc"
	PurposeVerification = "verification"
)

// Record statuses.
const (
	StatusVerified = "verified"
	StatusPending  = "pending"
	StatusWrong    = "wrong_value"
	StatusMissing  = "missing"
	StatusError    = "error"
)

// DecodeRecords reads sending_domains.dns_records.
func DecodeRecords(raw json.RawMessage) []Record {
	out := []Record{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	for i := range out {
		out[i].Type = strings.ToUpper(out[i].Type)
		out[i].Name = strings.ToLower(strings.TrimSuffix(out[i].Name, "."))
		if out[i].Status == "" {
			out[i].Status = StatusPending
		}
	}
	return out
}

// EncodeRecords writes sending_domains.dns_records.
func EncodeRecords(recs []Record) json.RawMessage {
	if recs == nil {
		recs = []Record{}
	}
	b, _ := json.Marshal(recs)
	return b
}

func (r Record) key() string {
	return strings.Join([]string{r.ProviderID, r.Purpose, strings.ToUpper(r.Type), strings.ToLower(r.Name)}, "|")
}

// isDomainSPF reports whether r is the SPF record of the sending domain
// itself (as opposed to the SPF record of a MAIL FROM subdomain).
func (r Record) isDomainSPF(domain string) bool {
	return r.Purpose == PurposeSPF && strings.EqualFold(r.Name, domain)
}

// rank orders records as the domain step lists them: DKIM, return-path,
// SPF for MAIL FROM, SPF for the sending domain, DMARC, verification.
func (r Record) rank(domain string) int {
	switch r.Purpose {
	case PurposeDKIM:
		return 0
	case PurposeReturnPath:
		return 1
	case PurposeSPF:
		if r.isDomainSPF(domain) {
			return 3
		}
		return 2
	case PurposeDMARC:
		return 4
	}
	return 5
}

// SortRecords orders records by provider (in the given order, domain-wide
// records last) and then by purpose.
func SortRecords(recs []Record, domain string, providerOrder []string) {
	pos := func(id string) int {
		if id == "" {
			return len(providerOrder) + 1
		}
		if i := slices.Index(providerOrder, id); i >= 0 {
			return i
		}
		return len(providerOrder)
	}
	slices.SortStableFunc(recs, func(a, b Record) int {
		if pa, pb := pos(a.ProviderID), pos(b.ProviderID); pa != pb {
			return pa - pb
		}
		return a.rank(domain) - b.rank(domain)
	})
}

// ReplaceProviderRecords swaps a provider's records for fresh ones, keeping
// the last check of records that didn't change.
func ReplaceProviderRecords(recs []Record, providerID string, fresh []Record) []Record {
	old := map[string]Record{}
	out := make([]Record, 0, len(recs)+len(fresh))
	for _, r := range recs {
		if r.ProviderID == providerID {
			old[r.key()] = r
			continue
		}
		out = append(out, r)
	}
	for _, f := range fresh {
		f.ProviderID = providerID
		if o, ok := old[f.key()]; ok && o.Value == f.Value {
			f.Status, f.Found, f.CheckedAt = o.Status, o.Found, o.CheckedAt
		}
		if f.Status == "" {
			f.Status = StatusPending
		}
		out = append(out, f)
	}
	return out
}

// RemoveProviderRecords drops a provider's records (it was disconnected).
func RemoveProviderRecords(recs []Record, providerID string) []Record {
	out := recs[:0:0]
	for _, r := range recs {
		if r.ProviderID != providerID {
			out = append(out, r)
		}
	}
	return out
}

// ensureDMARC adds the domain-wide DMARC record when it is missing.
func ensureDMARC(recs []Record, domain, rua string) []Record {
	for _, r := range recs {
		if r.Purpose == PurposeDMARC {
			return recs
		}
	}
	return append(recs, Record{Purpose: PurposeDMARC, Type: "TXT", Name: "_dmarc." + domain, Value: SuggestDMARC(rua), Status: StatusPending, Source: "reputecast"})
}

// SuggestDMARC is the policy ReputeCast proposes for a domain without one:
// quarantine, strict DKIM alignment, relaxed SPF alignment, aggregate
// reports to HostRepute when it is connected.
func SuggestDMARC(rua string) string {
	v := "v=DMARC1; p=quarantine"
	if rua != "" {
		if !strings.HasPrefix(rua, "mailto:") {
			rua = "mailto:" + rua
		}
		v += "; rua=" + rua
	}
	return v + "; adkim=s; aspf=r"
}

// spfTerms splits an SPF record into its terms (without "v=spf1").
func spfTerms(record string) []string {
	f := strings.Fields(strings.TrimSpace(record))
	if len(f) > 0 && strings.EqualFold(f[0], "v=spf1") {
		f = f[1:]
	}
	return f
}

// SPFMechanisms returns the mechanisms of an SPF record that authorize
// senders (include, a, mx, ip4, ip6, exists), without qualifiers.
func SPFMechanisms(record string) []string {
	out := []string{}
	for _, t := range spfTerms(record) {
		t = strings.TrimLeft(strings.ToLower(t), "+")
		name := t
		if i := strings.IndexAny(t, ":/"); i >= 0 {
			name = t[:i]
		}
		switch name {
		case "include", "a", "mx", "ip4", "ip6", "exists":
			out = append(out, t)
		}
	}
	return out
}

// SPFHas reports whether an SPF record contains a mechanism.
func SPFHas(record, mech string) bool {
	mech = strings.TrimLeft(strings.ToLower(mech), "+")
	for _, t := range spfTerms(record) {
		if strings.TrimLeft(strings.ToLower(t), "+") == mech {
			return true
		}
	}
	return false
}

// MergeSPF adds the mechanisms a provider needs to the domain's current SPF
// record, before its "all" (or redirect) term, keeping everything else.
// Without a current record it proposes "v=spf1 <mechanisms> -all".
func MergeSPF(current string, mechs []string) string {
	current = strings.TrimSpace(current)
	if current == "" || !strings.HasPrefix(strings.ToLower(current), "v=spf1") {
		return strings.TrimSpace("v=spf1 " + strings.Join(mechs, " ") + " -all")
	}
	terms := spfTerms(current)
	missing := []string{}
	for _, m := range mechs {
		if !SPFHas(current, m) && !slices.Contains(missing, m) {
			missing = append(missing, m)
		}
	}
	if len(missing) == 0 {
		return current
	}
	at := len(terms)
	for i, t := range terms {
		lt := strings.ToLower(t)
		if strings.HasSuffix(lt, "all") && len(lt) <= 4 || strings.HasPrefix(lt, "redirect=") || strings.HasPrefix(lt, "exp=") {
			at = i
			break
		}
	}
	out := append([]string{"v=spf1"}, terms[:at]...)
	out = append(out, missing...)
	out = append(out, terms[at:]...)
	if at == len(terms) {
		out = append(out, "-all")
	}
	return strings.Join(out, " ")
}

// recordLabels names each record as the domain step does: "DKIM 1 of 3",
// "Return-path / MAIL FROM", "SPF for MAIL FROM", "SPF for the sending
// domain", "DMARC", with a short note under some of them.
func recordLabels(recs []Record, domain string, dkimNote func(providerID string) string) (labels, notes []string) {
	labels, notes = make([]string, len(recs)), make([]string, len(recs))
	dkimTotal := map[string]int{}
	for _, r := range recs {
		if r.Purpose == PurposeDKIM {
			dkimTotal[r.ProviderID]++
		}
	}
	dkimSeen := map[string]int{}
	returnSeen := map[string]bool{}
	for i, r := range recs {
		switch r.Purpose {
		case PurposeDKIM:
			dkimSeen[r.ProviderID]++
			n, total := dkimSeen[r.ProviderID], dkimTotal[r.ProviderID]
			labels[i] = "DKIM"
			if total > 1 {
				labels[i] = fmt.Sprintf("DKIM %d of %d", n, total)
			}
			if n == 1 && dkimNote != nil {
				notes[i] = dkimNote(r.ProviderID)
			}
		case PurposeReturnPath:
			labels[i] = "Return-path / MAIL FROM"
			if !returnSeen[r.ProviderID] {
				notes[i] = "so bounces and SPF align with your domain"
			}
			returnSeen[r.ProviderID] = true
		case PurposeSPF:
			if r.isDomainSPF(domain) {
				labels[i], notes[i] = "SPF for the sending domain", "merged with your existing record"
			} else {
				labels[i] = "SPF for MAIL FROM"
			}
		case PurposeDMARC:
			labels[i], notes[i] = "DMARC", "required by Gmail and Yahoo for bulk senders"
		case PurposeVerification:
			labels[i] = "Domain verification"
		default:
			labels[i] = r.Purpose
		}
	}
	return labels, notes
}

// zoneLine renders a record as a BIND zone-file line ("Copy as zone file").
func zoneLine(r Record) string {
	name := r.Name + "."
	switch strings.ToUpper(r.Type) {
	case "TXT":
		return fmt.Sprintf("%s 3600 IN TXT %s", name, quoteTXT(r.Value))
	case "CNAME":
		return fmt.Sprintf("%s 3600 IN CNAME %s.", name, strings.TrimSuffix(r.Value, "."))
	case "MX":
		v := r.Value
		if f := strings.Fields(v); len(f) == 2 {
			v = f[0] + " " + strings.TrimSuffix(f[1], ".") + "."
		}
		return fmt.Sprintf("%s 3600 IN MX %s", name, v)
	}
	return fmt.Sprintf("%s 3600 IN %s %s", name, strings.ToUpper(r.Type), r.Value)
}

// quoteTXT splits long TXT values into 255-byte quoted strings.
func quoteTXT(v string) string {
	parts := []string{}
	for len(v) > 255 {
		parts = append(parts, `"`+strings.ReplaceAll(v[:255], `"`, `\"`)+`"`)
		v = v[255:]
	}
	parts = append(parts, `"`+strings.ReplaceAll(v, `"`, `\"`)+`"`)
	return strings.Join(parts, " ")
}
