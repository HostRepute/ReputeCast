package domains

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// AuthDetail is sending_domains.auth_detail: what the last authentication
// check found, beyond the one-word summary in auth_summary.
type AuthDetail struct {
	SPFStatus string `json:"spf_status,omitempty"`
	// SPFAll is the domain's final "all" term ("-all", "~all").
	SPFAll      string `json:"spf_all,omitempty"`
	SPFRecord   string `json:"spf_record,omitempty"`
	DMARCStatus string `json:"dmarc_status,omitempty"`
	DMARCPolicy string `json:"dmarc_policy,omitempty"`
	DMARCRecord string `json:"dmarc_record,omitempty"`
	// DMARCInherited is true when the policy comes from the organizational
	// domain (acme.io for news.acme.io).
	DMARCInherited bool   `json:"dmarc_inherited,omitempty"`
	BIMIStatus     string `json:"bimi_status,omitempty"`
	// MailFrom lists the custom MAIL FROM (return-path) hosts with their
	// alignment.
	MailFrom []MailFromHost `json:"mail_from,omitempty"`
}

// MailFromHost is one custom return-path domain.
type MailFromHost struct {
	Host   string `json:"host"`
	Status string `json:"status"` // aligned, misaligned, missing…
}

func decodeAuthDetail(raw json.RawMessage) AuthDetail {
	var d AuthDetail
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &d)
	}
	return d
}

func decodeSummary(raw json.RawMessage) dnscheck.AuthSummary {
	var s dnscheck.AuthSummary
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}

func cell(text, tone string) SenderDomainCell { return SenderDomainCell{Text: text, Tone: tone} }

// domainRow builds one row of the domains table from the stored records and
// the last authentication check.
func domainRow(d store.SendingDomain, provs []store.Provider) SendingDomainRow {
	recs := DecodeRecords(d.DnsRecords)
	sum := decodeSummary(d.AuthSummary)
	det := decodeAuthDetail(d.AuthDetail)
	row := SendingDomainRow{ID: d.ID.String(), Domain: d.Domain, Note: d.Note, Status: d.Status, UsedBy: []string{}, CheckedAt: d.LastCheckedAt}
	// Providers set up for the domain, in connection order.
	has := map[string]bool{}
	for _, r := range recs {
		if r.ProviderID != "" {
			has[r.ProviderID] = true
		}
	}
	dkimProviders := 0
	dkimVerified := 0
	var pendingType string
	pendingIndex, pendingTotal := 0, 0
	for _, p := range provs {
		id := p.ID.String()
		if !has[id] {
			continue
		}
		row.UsedBy = append(row.UsedBy, state.Short(p.Name, p.Adapter))
		var dk []Record
		for _, r := range recs {
			if r.ProviderID == id && r.Purpose == PurposeDKIM {
				dk = append(dk, r)
			}
		}
		if len(dk) == 0 {
			continue
		}
		dkimProviders++
		ok := true
		for i, r := range dk {
			if r.Status != StatusVerified {
				if ok {
					pendingType, pendingIndex, pendingTotal = r.Type, i+1, len(dk)
				}
				ok = false
			}
		}
		if ok {
			dkimVerified++
		}
	}
	row.DKIM = dkimCell(dkimProviders, dkimVerified, pendingType, pendingIndex, pendingTotal)
	// With the providers' own return paths, receivers check SPF on the
	// providers' domains, so the domain's own qualifier says nothing about
	// this mail.
	qualifier := det.SPFAll
	if len(det.MailFrom) == 0 {
		qualifier = ""
	}
	row.SPF = spfCell(sum.SPF, qualifier)
	row.DMARC = dmarcCell(sum.DMARC)
	row.ReturnPath = returnPathCell(det.MailFrom, d.Domain)
	row.Tracking = trackingCell(d.TrackingDomain, d.TrackingStatus)
	row.BIMI = bimiCell(sum.BIMI)
	if d.LastCheckedAt == nil {
		row.SPF, row.DMARC, row.BIMI = cell("Not checked", "neutral"), cell("Not checked", "neutral"), cell("Not checked", "neutral")
	}
	return row
}

// dkimCell: "3 of 3 providers", "Verified", "Pending CNAME 2 of 3".
func dkimCell(providers, verified int, pendingType string, pendingIndex, pendingTotal int) SenderDomainCell {
	switch {
	case providers == 0:
		return cell("Not set up", "neutral")
	case verified == providers && providers == 1:
		return cell("Verified", "ok")
	case verified == providers:
		return cell(fmt.Sprintf("%d of %d providers", verified, providers), "ok")
	case providers == 1 && pendingTotal > 1:
		return cell(fmt.Sprintf("Pending %s %d of %d", pendingType, pendingIndex, pendingTotal), "warn")
	case providers == 1:
		return cell("Pending", "warn")
	}
	return cell(fmt.Sprintf("%d of %d providers", verified, providers), "warn")
}

func spfCell(status, all string) SenderDomainCell {
	switch status {
	case dnscheck.StatusPass:
		if all == "-all" {
			return cell("pass · -all", "ok")
		}
		return cell("pass", "ok")
	case dnscheck.StatusMissing:
		return cell("Missing", "bad")
	case dnscheck.StatusTooMany:
		return cell("Too many lookups", "bad")
	case dnscheck.StatusFail:
		return cell("fail", "bad")
	case dnscheck.StatusInvalid:
		return cell("Invalid", "bad")
	case "":
		return cell("Not checked", "neutral")
	}
	return cell("Couldn't check", "neutral")
}

func dmarcCell(v string) SenderDomainCell {
	switch v {
	case "quarantine", "reject":
		return cell("p="+v, "ok")
	case "none":
		return cell("p=none", "warn")
	case dnscheck.StatusMissing:
		return cell("Missing", "bad")
	case dnscheck.StatusInvalid:
		return cell("Invalid", "bad")
	case "":
		return cell("Not checked", "neutral")
	}
	return cell("Couldn't check", "neutral")
}

// returnPathCell lists the custom MAIL FROM hosts by their first label
// ("bounce. · rp."), or "Default" when the providers use their own.
func returnPathCell(hosts []MailFromHost, domain string) SenderDomainCell {
	if len(hosts) == 0 {
		return cell("Default", "neutral")
	}
	parts := make([]string, 0, len(hosts))
	tone := "ok"
	for _, h := range hosts {
		label := h.Host
		if strings.HasSuffix(h.Host, "."+domain) {
			label = strings.TrimSuffix(h.Host, domain)
		}
		parts = append(parts, label)
		if h.Status != dnscheck.StatusAligned {
			tone = "warn"
		}
	}
	return cell(strings.Join(parts, " · "), tone)
}

func trackingCell(domain, status string) SenderDomainCell {
	switch {
	case domain == "":
		return cell("None", "neutral")
	case status == StatusVerified:
		return cell(domain, "ok")
	case status == StatusMissing || status == StatusWrong:
		return cell(domain+" · not pointed yet", "warn")
	}
	return cell(domain+" · pending", "warn")
}

func bimiCell(status string) SenderDomainCell {
	switch status {
	case dnscheck.StatusPass:
		return cell("Set", "ok")
	case dnscheck.StatusInvalid:
		return cell("Invalid", "warn")
	case "", dnscheck.StatusMissing:
		return cell("Not set", "neutral")
	}
	return cell("Not set", "neutral")
}
