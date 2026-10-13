package dnscheck

import (
	"context"
	"slices"
	"strings"
)

// Requirement is a DNS record a sending provider needs for a domain (the
// records its DomainManager returns, stored on the sending domain).
type Requirement struct {
	ProviderID string `json:"providerId,omitempty"`
	// Purpose is dkim, spf, return_path, dmarc or verification.
	Purpose string `json:"purpose"`
	Type    string `json:"type"` // CNAME, TXT, MX
	Name    string `json:"name"`
	Value   string `json:"value"`
}

// Record check statuses (the live verification of one required record).
const (
	RecordVerified   = "verified"
	RecordWrongValue = "wrong_value"
	RecordMissing    = "missing"
	RecordError      = "error"
)

// RecordCheck is the live status of one required record.
type RecordCheck struct {
	Requirement
	Status string `json:"status"`
	// Found is what DNS returned when it differs from the requirement.
	Found string `json:"found,omitempty"`
}

// VerifyRecord checks one required record against DNS: a CNAME must point to
// the value, a TXT record must contain it, an MX must name the host.
func VerifyRecord(ctx context.Context, r Resolver, req Requirement) RecordCheck {
	out := RecordCheck{Requirement: req}
	want := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(req.Value), "."))
	switch strings.ToUpper(req.Type) {
	case "CNAME":
		got, err := r.CNAME(ctx, req.Name)
		if err != nil {
			out.Status = recordErr(err)
			return out
		}
		if strings.EqualFold(got, want) {
			out.Status = RecordVerified
		} else {
			out.Status, out.Found = RecordWrongValue, got
		}
	case "TXT":
		txts, err := r.TXT(ctx, req.Name)
		if err != nil {
			out.Status = recordErr(err)
			return out
		}
		norm := func(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), "")) }
		for _, t := range txts {
			if norm(t) == norm(want) || strings.Contains(norm(t), norm(want)) {
				out.Status = RecordVerified
				return out
			}
		}
		out.Status = RecordWrongValue
		if len(txts) > 0 {
			out.Found = txts[0]
		}
	case "MX":
		mxs, err := r.MX(ctx, req.Name)
		if err != nil {
			out.Status = recordErr(err)
			return out
		}
		host := want
		if f := strings.Fields(want); len(f) == 2 {
			host = f[1]
		}
		for _, mx := range mxs {
			if strings.EqualFold(mx.Host, host) {
				out.Status = RecordVerified
				return out
			}
		}
		out.Status = RecordWrongValue
		if len(mxs) > 0 {
			out.Found = mxs[0].Host
		}
	default:
		out.Status = RecordError
	}
	return out
}

func recordErr(err error) string {
	if IsNotFound(err) {
		return RecordMissing
	}
	return RecordError
}

// AuthSummary is the one-line verdict per mechanism for a sending domain, as
// shown on Deliverability ("pass · pass · p=quarantine") and stored on the
// sending domain.
type AuthSummary struct {
	SPF  string `json:"spf"`  // pass, missing, invalid, fail, too_many_lookups, error
	DKIM string `json:"dkim"` // pass, missing, invalid, revoked, fail, unknown, error
	// DMARC is the policy (none, quarantine, reject) or missing / invalid.
	DMARC      string `json:"dmarc"`
	BIMI       string `json:"bimi"`        // pass, missing, invalid
	ReturnPath string `json:"return_path"` // aligned, misaligned, missing, default
}

// DomainAuth is the full authentication check of one sending domain.
type DomainAuth struct {
	Domain   string        `json:"domain"`
	SPF      SPF           `json:"spf"`
	DKIM     []DKIM        `json:"dkim"`
	DMARC    DMARC         `json:"dmarc"`
	BIMI     BIMI          `json:"bimi"`
	MailFrom []MailFrom    `json:"mailFrom"`
	Records  []RecordCheck `json:"records"`
	Summary  AuthSummary   `json:"summary"`
}

// CheckDomain runs every authentication check for a sending domain against
// the records its providers require.
func CheckDomain(ctx context.Context, r Resolver, domain string, reqs []Requirement) DomainAuth {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	out := DomainAuth{Domain: domain, DKIM: []DKIM{}, MailFrom: []MailFrom{}, Records: []RecordCheck{}}
	out.SPF = CheckSPF(ctx, r, domain)
	out.DMARC = CheckDMARC(ctx, r, domain)
	out.BIMI = CheckBIMI(ctx, r, domain)

	var mailFroms []string
	spfOK, dkimSeen, dkimOK := true, false, true
	worstDKIM := ""
	for _, req := range reqs {
		rc := VerifyRecord(ctx, r, req)
		out.Records = append(out.Records, rc)
		switch req.Purpose {
		case "dkim":
			dkimSeen = true
			if sel, d, ok := strings.Cut(strings.ToLower(req.Name), "._domainkey."); ok {
				k := CheckDKIM(ctx, r, sel, d)
				out.DKIM = append(out.DKIM, k)
				if k.Status != StatusPass {
					dkimOK = false
					if worstDKIM == "" || worstDKIM == StatusError {
						worstDKIM = k.Status
					}
				}
			} else if rc.Status != RecordVerified {
				dkimOK = false
				worstDKIM = StatusFail
			}
		case "spf":
			if rc.Status != RecordVerified {
				spfOK = false
			}
		case "return_path":
			if !slices.Contains(mailFroms, strings.ToLower(req.Name)) {
				mailFroms = append(mailFroms, strings.ToLower(req.Name))
			}
		}
	}
	for _, mf := range mailFroms {
		out.MailFrom = append(out.MailFrom, CheckMailFrom(ctx, r, mf, domain, out.DMARC.StrictSPF))
	}

	s := AuthSummary{}
	// SPF: the domain's own record must be valid, and every provider's SPF
	// requirement (usually on its MAIL FROM domain) must be in place.
	s.SPF = out.SPF.Status
	if s.SPF == StatusMissing && len(out.MailFrom) > 0 {
		// Mail only uses the MAIL FROM domains: their SPF is what DMARC checks.
		s.SPF = StatusPass
		for _, mf := range out.MailFrom {
			if mf.SPF != StatusPass {
				s.SPF = mf.SPF
			}
		}
	}
	if s.SPF == StatusPass && !spfOK {
		s.SPF = StatusFail
	}
	switch {
	case !dkimSeen:
		s.DKIM = "unknown"
	case dkimOK:
		s.DKIM = StatusPass
	default:
		s.DKIM = worstDKIM
	}
	if out.DMARC.Status == StatusPass {
		s.DMARC = out.DMARC.Policy
	} else {
		s.DMARC = out.DMARC.Status
	}
	s.BIMI = out.BIMI.Status
	switch {
	case len(out.MailFrom) == 0:
		s.ReturnPath = "default"
	default:
		s.ReturnPath = StatusAligned
		for _, mf := range out.MailFrom {
			if mf.Status != StatusAligned {
				s.ReturnPath = mf.Status
			}
		}
	}
	out.Summary = s
	return out
}
