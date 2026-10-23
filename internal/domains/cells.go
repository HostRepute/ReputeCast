package domains

import (
	"encoding/json"
	"strings"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/i18n"
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

// cellIn is a cell whose text is one of the package's messages.
func cellIn(l i18n.Lang, key, tone string, args i18n.Args) SenderDomainCell {
	return cell(text(l, key, args), tone)
}

// domainRow builds one row of the domains table from the stored records and
// the last authentication check, in a reader's language.
func domainRow(l i18n.Lang, d store.SendingDomain, provs []store.Provider) SendingDomainRow {
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
	row.DKIM = dkimCell(l, dkimProviders, dkimVerified, pendingType, pendingIndex, pendingTotal)
	// With the providers' own return paths, receivers check SPF on the
	// providers' domains, so the domain's own qualifier says nothing about
	// this mail.
	qualifier := det.SPFAll
	if len(det.MailFrom) == 0 {
		qualifier = ""
	}
	row.SPF = spfCell(l, sum.SPF, qualifier)
	row.DMARC = dmarcCell(l, sum.DMARC)
	row.ReturnPath = returnPathCell(l, det.MailFrom, d.Domain)
	row.Tracking = trackingCell(l, d.TrackingDomain, d.TrackingStatus)
	if d.Transactional && d.TrackingDomain == "" {
		// Transactional templates track nothing by default (SettingsSenders).
		row.Tracking = cellIn(l, "cell.offNoTracking", "neutral", nil)
	}
	row.BIMI = bimiCell(l, sum.BIMI)
	if d.LastCheckedAt == nil {
		notChecked := cellIn(l, "cell.notChecked", "neutral", nil)
		row.SPF, row.DMARC, row.BIMI = notChecked, notChecked, notChecked
	}
	return row
}

// dkimCell: "3 of 3 providers", "Verified", "Pending CNAME 2 of 3".
func dkimCell(l i18n.Lang, providers, verified int, pendingType string, pendingIndex, pendingTotal int) SenderDomainCell {
	switch {
	case providers == 0:
		return cellIn(l, "cell.notSetUp", "neutral", nil)
	case verified == providers && providers == 1:
		return cellIn(l, "cell.verified", "ok", nil)
	case verified == providers:
		return cellIn(l, "cell.providers", "ok", i18n.Args{"verified": verified, "total": providers})
	case providers == 1 && pendingTotal > 1:
		return cellIn(l, "cell.pendingRecord", "warn", i18n.Args{"type": pendingType, "index": pendingIndex, "total": pendingTotal})
	case providers == 1:
		return cellIn(l, "cell.pending", "warn", nil)
	}
	return cellIn(l, "cell.providers", "warn", i18n.Args{"verified": verified, "total": providers})
}

// spfCell is the SPF cell: the result as DNS says it ("pass · -all"), or
// what went wrong.
func spfCell(l i18n.Lang, status, all string) SenderDomainCell {
	switch status {
	case dnscheck.StatusPass:
		if all == "-all" {
			return cell("pass · -all", "ok")
		}
		return cell("pass", "ok")
	case dnscheck.StatusMissing:
		return cellIn(l, "cell.missing", "bad", nil)
	case dnscheck.StatusTooMany:
		return cellIn(l, "cell.tooManyLookups", "bad", nil)
	case dnscheck.StatusFail:
		return cell("fail", "bad")
	case dnscheck.StatusInvalid:
		return cellIn(l, "cell.invalid", "bad", nil)
	case "":
		return cellIn(l, "cell.notChecked", "neutral", nil)
	}
	return cellIn(l, "cell.couldntCheck", "neutral", nil)
}

// dmarcCell is the DMARC cell: the policy as DNS writes it ("p=quarantine").
func dmarcCell(l i18n.Lang, v string) SenderDomainCell {
	switch v {
	case "quarantine", "reject":
		return cell("p="+v, "ok")
	case "none":
		return cell("p=none", "warn")
	case dnscheck.StatusMissing:
		return cellIn(l, "cell.missing", "bad", nil)
	case dnscheck.StatusInvalid:
		return cellIn(l, "cell.invalid", "bad", nil)
	case "":
		return cellIn(l, "cell.notChecked", "neutral", nil)
	}
	return cellIn(l, "cell.couldntCheck", "neutral", nil)
}

// returnPathCell lists the custom MAIL FROM hosts by their first label
// ("bounce. · rp."), or "Default" when the providers use their own.
func returnPathCell(l i18n.Lang, hosts []MailFromHost, domain string) SenderDomainCell {
	if len(hosts) == 0 {
		return cellIn(l, "cell.default", "neutral", nil)
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

func trackingCell(l i18n.Lang, domain, status string) SenderDomainCell {
	switch {
	case domain == "":
		return cellIn(l, "cell.none", "neutral", nil)
	case status == StatusVerified:
		return cell(domain, "ok")
	case status == StatusMissing || status == StatusWrong:
		return cellIn(l, "cell.trackingNotPointed", "warn", i18n.Args{"domain": domain})
	}
	return cellIn(l, "cell.trackingPending", "warn", i18n.Args{"domain": domain})
}

func bimiCell(l i18n.Lang, status string) SenderDomainCell {
	switch status {
	case dnscheck.StatusPass:
		return cellIn(l, "cell.set", "ok", nil)
	case dnscheck.StatusInvalid:
		return cellIn(l, "cell.invalid", "warn", nil)
	}
	return cellIn(l, "cell.notSet", "neutral", nil)
}
