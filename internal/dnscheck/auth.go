package dnscheck

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/emersion/go-msgauth/dmarc"
	"golang.org/x/net/publicsuffix"
)

// Status words shared by every check result.
const (
	StatusPass     = "pass"    // present, valid and doing its job
	StatusMissing  = "missing" // no record
	StatusInvalid  = "invalid" // a record exists but is broken
	StatusFail     = "fail"    // valid record that doesn't cover what it must
	StatusError    = "error"   // DNS didn't answer; try again
	StatusRevoked  = "revoked" // DKIM key with an empty p=
	StatusTooMany  = "too_many_lookups"
	StatusAligned  = "aligned"
	StatusMisalign = "misaligned"
)

// OrgDomain returns the organizational domain (RFC 7489 §3.2) of a name:
// "news.acme.io" → "acme.io", "mail.example.co.uk" → "example.co.uk".
func OrgDomain(name string) string {
	name = strings.ToLower(strings.TrimSuffix(name, "."))
	if d, err := publicsuffix.EffectiveTLDPlusOne(name); err == nil {
		return d
	}
	return name
}

// Aligned reports whether two domains align: identical in strict mode, the
// same organizational domain in relaxed mode.
func Aligned(a, b string, strict bool) bool {
	a, b = strings.ToLower(strings.TrimSuffix(a, ".")), strings.ToLower(strings.TrimSuffix(b, "."))
	if strict {
		return a == b
	}
	return OrgDomain(a) == OrgDomain(b)
}

func statusFor(err error) string {
	if IsNotFound(err) {
		return StatusMissing
	}
	return StatusError
}

// SPF is the result of checking a domain's SPF record.
type SPF struct {
	Domain string `json:"domain"`
	Status string `json:"status"`
	Record string `json:"record,omitempty"`
	// All is the final "all" mechanism with its qualifier: "-all", "~all"…
	All string `json:"all,omitempty"`
	// Lookups counts DNS-querying terms, including nested includes (max 10).
	Lookups  int      `json:"lookups"`
	Includes []string `json:"includes,omitempty"`
	IP4      []string `json:"ip4,omitempty"`
	IP6      []string `json:"ip6,omitempty"`
	Error    string   `json:"error,omitempty"`
}

// MaxSPFLookups is the RFC 7208 limit on DNS-querying terms.
const MaxSPFLookups = 10

// CheckSPF fetches and evaluates the SPF record of domain: exactly one
// "v=spf1" record, parseable terms and at most 10 DNS lookups.
func CheckSPF(ctx context.Context, r Resolver, domain string) SPF {
	out := SPF{Domain: domain}
	rec, err := spfRecord(ctx, r, domain)
	if err != nil {
		out.Status, out.Error = spfErrStatus(err), err.Error()
		return out
	}
	out.Record = rec
	seen := map[string]bool{strings.ToLower(domain): true}
	t, err := walkSPF(ctx, r, rec, seen, 0)
	out.Lookups, out.Includes, out.IP4, out.IP6, out.All = t.lookups, t.includes, t.ip4, t.ip6, t.all
	switch {
	case err != nil:
		out.Status, out.Error = StatusInvalid, err.Error()
	case t.lookups > MaxSPFLookups:
		out.Status, out.Error = StatusTooMany, fmt.Sprintf("%d DNS lookups (the limit is %d)", t.lookups, MaxSPFLookups)
	case out.All == "+all":
		out.Status, out.Error = StatusInvalid, "+all lets anyone send as this domain"
	default:
		out.Status = StatusPass
	}
	return out
}

// Covers reports whether the SPF result authorizes a provider: either the
// include domain (e.g. "amazonses.com") or the IP appears in the record or
// in anything it includes.
func (s SPF) Covers(include string, ip netip.Addr) bool {
	if include != "" {
		for _, inc := range s.Includes {
			if strings.EqualFold(inc, include) || strings.HasSuffix(strings.ToLower(inc), "."+strings.ToLower(include)) {
				return true
			}
		}
	}
	if ip.IsValid() {
		for _, c := range append(slices.Clone(s.IP4), s.IP6...) {
			if p, err := netip.ParsePrefix(c); err == nil && p.Contains(ip) {
				return true
			}
			if a, err := netip.ParseAddr(c); err == nil && a == ip {
				return true
			}
		}
	}
	return false
}

var errMultipleSPF = errors.New("more than one v=spf1 record (permerror)")

func spfErrStatus(err error) string {
	if errors.Is(err, errMultipleSPF) {
		return StatusInvalid
	}
	return statusFor(err)
}

func spfRecord(ctx context.Context, r Resolver, domain string) (string, error) {
	txts, err := r.TXT(ctx, domain)
	if err != nil {
		return "", err
	}
	var recs []string
	for _, t := range txts {
		lt := strings.ToLower(strings.TrimSpace(t))
		if lt == "v=spf1" || strings.HasPrefix(lt, "v=spf1 ") {
			recs = append(recs, strings.TrimSpace(t))
		}
	}
	switch len(recs) {
	case 0:
		return "", fmt.Errorf("%s: no v=spf1 record: %w", domain, ErrNoData)
	case 1:
		return recs[0], nil
	default:
		return "", errMultipleSPF
	}
}

type spfTally struct {
	lookups  int
	includes []string
	ip4, ip6 []string
	all      string
}

// walkSPF parses an SPF record and follows include/redirect terms to count
// lookups and collect what the record authorizes.
func walkSPF(ctx context.Context, r Resolver, rec string, seen map[string]bool, depth int) (spfTally, error) {
	var t spfTally
	if depth > MaxSPFLookups {
		return t, errors.New("include chain too deep")
	}
	terms := strings.Fields(rec)[1:]
	var redirect string
	for _, term := range terms {
		lt := strings.ToLower(term)
		q := ""
		if strings.ContainsAny(lt[:1], "+-~?") {
			q, lt = lt[:1], lt[1:]
		}
		switch {
		case lt == "all":
			if q == "" {
				q = "+"
			}
			t.all = q + "all"
		case strings.HasPrefix(lt, "include:"):
			inc := strings.TrimPrefix(lt, "include:")
			t.lookups++
			t.includes = append(t.includes, inc)
			if seen[inc] {
				continue
			}
			seen[inc] = true
			sub, err := spfRecord(ctx, r, inc)
			if err != nil {
				if IsNotFound(err) {
					return t, fmt.Errorf("include:%s has no SPF record", inc)
				}
				if errors.Is(err, errMultipleSPF) {
					return t, fmt.Errorf("include:%s has more than one SPF record", inc)
				}
				continue // temporary failure: count it, don't fail the record
			}
			st, err := walkSPF(ctx, r, sub, seen, depth+1)
			t.lookups += st.lookups
			t.includes = append(t.includes, st.includes...)
			t.ip4 = append(t.ip4, st.ip4...)
			t.ip6 = append(t.ip6, st.ip6...)
			if err != nil {
				return t, err
			}
		case lt == "a" || strings.HasPrefix(lt, "a:") || strings.HasPrefix(lt, "a/"),
			lt == "mx" || strings.HasPrefix(lt, "mx:") || strings.HasPrefix(lt, "mx/"),
			lt == "ptr" || strings.HasPrefix(lt, "ptr:"),
			strings.HasPrefix(lt, "exists:"):
			t.lookups++
		case strings.HasPrefix(lt, "ip4:"):
			t.ip4 = append(t.ip4, strings.TrimPrefix(lt, "ip4:"))
		case strings.HasPrefix(lt, "ip6:"):
			t.ip6 = append(t.ip6, strings.TrimPrefix(lt, "ip6:"))
		case strings.HasPrefix(lt, "redirect="):
			redirect = strings.TrimPrefix(lt, "redirect=")
		case strings.HasPrefix(lt, "exp=") || strings.Contains(lt, "="):
			// exp= and unknown modifiers are allowed and ignored.
		default:
			return t, fmt.Errorf("unknown term %q", term)
		}
	}
	if redirect != "" && t.all == "" {
		t.lookups++
		if !seen[redirect] {
			seen[redirect] = true
			sub, err := spfRecord(ctx, r, redirect)
			if err != nil {
				if IsNotFound(err) {
					return t, fmt.Errorf("redirect=%s has no SPF record", redirect)
				}
				return t, nil
			}
			st, err := walkSPF(ctx, r, sub, seen, depth+1)
			t.lookups += st.lookups
			t.includes = append(t.includes, st.includes...)
			t.ip4 = append(t.ip4, st.ip4...)
			t.ip6 = append(t.ip6, st.ip6...)
			t.all = st.all
			return t, err
		}
	}
	return t, nil
}

// DKIM is the result of checking one DKIM selector.
type DKIM struct {
	Selector string `json:"selector"`
	Domain   string `json:"domain"`
	Status   string `json:"status"`
	// Target is the CNAME the selector points to (provider-hosted keys).
	Target  string `json:"target,omitempty"`
	KeyType string `json:"keyType,omitempty"`
	// KeyBits is an estimate from the public key length (RSA only).
	KeyBits int    `json:"keyBits,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CheckDKIM looks up selector._domainkey.domain (following a CNAME to a
// provider-hosted key) and checks that it publishes a usable key.
func CheckDKIM(ctx context.Context, r Resolver, selector, domain string) DKIM {
	name := selector + "._domainkey." + domain
	out := DKIM{Selector: selector, Domain: domain}
	if target, err := r.CNAME(ctx, name); err == nil {
		out.Target = target
	}
	txts, err := r.TXT(ctx, name)
	if err != nil {
		out.Status, out.Error = statusFor(err), err.Error()
		return out
	}
	rec := strings.Join(txts, "")
	tags := parseTags(rec)
	if v, ok := tags["v"]; ok && v != "DKIM1" {
		out.Status, out.Error = StatusInvalid, "v= must be DKIM1"
		return out
	}
	p, ok := tags["p"]
	if !ok {
		out.Status, out.Error = StatusInvalid, "no p= public key"
		return out
	}
	if p == "" {
		out.Status, out.Error = StatusRevoked, "the key is revoked (empty p=)"
		return out
	}
	out.KeyType = "rsa"
	if k, ok := tags["k"]; ok && k != "" {
		out.KeyType = k
	}
	if out.KeyType == "rsa" {
		// Base64 of a SubjectPublicKeyInfo: ~392 chars for 2048-bit keys.
		n := len(strings.TrimRight(p, "="))
		switch {
		case n >= 700:
			out.KeyBits = 4096
		case n >= 360:
			out.KeyBits = 2048
		case n >= 200:
			out.KeyBits = 1024
		default:
			out.KeyBits = 512
		}
		if out.KeyBits < 1024 {
			out.Status, out.Error = StatusInvalid, "the RSA key is shorter than 1024 bits"
			return out
		}
	}
	out.Status = StatusPass
	return out
}

func parseTags(rec string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(rec, ";") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.Join(strings.Fields(v), "")
	}
	return out
}

// DMARC is the result of DMARC policy discovery for a domain.
type DMARC struct {
	Domain string `json:"domain"`
	// PolicyDomain is where the record was found: the domain itself or its
	// organizational domain (RFC 7489 §6.6.3).
	PolicyDomain string `json:"policyDomain,omitempty"`
	Status       string `json:"status"`
	Record       string `json:"record,omitempty"`
	// Policy applied to Domain: p= or, for a subdomain, sp= when present.
	Policy          string   `json:"policy,omitempty"`
	SubdomainPolicy string   `json:"subdomainPolicy,omitempty"`
	Percent         int      `json:"percent"`
	RUA             []string `json:"rua,omitempty"`
	RUF             []string `json:"ruf,omitempty"`
	StrictDKIM      bool     `json:"strictDkim"`
	StrictSPF       bool     `json:"strictSpf"`
	Error           string   `json:"error,omitempty"`
}

// CheckDMARC finds the DMARC policy for domain: _dmarc.<domain>, then
// _dmarc.<organizational domain>.
func CheckDMARC(ctx context.Context, r Resolver, domain string) DMARC {
	out := DMARC{Domain: domain, Percent: 100}
	candidates := []string{domain}
	if org := OrgDomain(domain); org != strings.ToLower(domain) {
		candidates = append(candidates, org)
	}
	var lastErr error
	for _, d := range candidates {
		txts, err := r.TXT(ctx, "_dmarc."+d)
		if err != nil {
			lastErr = err
			if !IsNotFound(err) {
				break
			}
			continue
		}
		var recs []string
		for _, t := range txts {
			if strings.HasPrefix(strings.TrimSpace(t), "v=DMARC1") {
				recs = append(recs, strings.TrimSpace(t))
			}
		}
		if len(recs) == 0 {
			lastErr = fmt.Errorf("_dmarc.%s: no v=DMARC1 record: %w", d, ErrNoData)
			continue
		}
		out.PolicyDomain, out.Record = d, recs[0]
		if len(recs) > 1 {
			out.Status, out.Error = StatusInvalid, "more than one DMARC record"
			return out
		}
		rec, err := dmarc.Parse(recs[0])
		if err != nil {
			out.Status, out.Error = StatusInvalid, strings.TrimPrefix(err.Error(), "dmarc: ")
			return out
		}
		out.Policy = string(rec.Policy)
		out.SubdomainPolicy = string(rec.SubdomainPolicy)
		if d != strings.ToLower(domain) && rec.SubdomainPolicy != "" {
			out.Policy = string(rec.SubdomainPolicy)
		}
		if rec.Percent != nil {
			out.Percent = *rec.Percent
		}
		out.RUA, out.RUF = rec.ReportURIAggregate, rec.ReportURIFailure
		out.StrictDKIM = rec.DKIMAlignment == dmarc.AlignmentStrict
		out.StrictSPF = rec.SPFAlignment == dmarc.AlignmentStrict
		out.Status = StatusPass
		return out
	}
	out.Status = statusFor(lastErr)
	if lastErr != nil {
		out.Error = lastErr.Error()
	}
	return out
}

// Enforced reports whether the policy keeps failing mail out of inboxes.
func (d DMARC) Enforced() bool {
	return d.Status == StatusPass && (d.Policy == "quarantine" || d.Policy == "reject")
}

// BIMI is the result of checking default._bimi.<domain>.
type BIMI struct {
	Domain string `json:"domain"`
	Status string `json:"status"`
	Logo   string `json:"logo,omitempty"`
	// Certificate is the VMC/CMC URL (a=); Gmail requires one.
	Certificate string `json:"certificate,omitempty"`
	Error       string `json:"error,omitempty"`
}

// CheckBIMI looks up the default BIMI record of domain.
func CheckBIMI(ctx context.Context, r Resolver, domain string) BIMI {
	out := BIMI{Domain: domain}
	txts, err := r.TXT(ctx, "default._bimi."+domain)
	if err != nil {
		out.Status, out.Error = statusFor(err), err.Error()
		return out
	}
	for _, t := range txts {
		tags := parseTags(t)
		if tags["v"] != "BIMI1" {
			continue
		}
		out.Logo, out.Certificate = tags["l"], tags["a"]
		if out.Logo == "" || !strings.HasPrefix(out.Logo, "https://") {
			out.Status, out.Error = StatusInvalid, "l= must be an https URL to an SVG logo"
			return out
		}
		out.Status = StatusPass
		return out
	}
	out.Status = StatusMissing
	return out
}

// MailFrom is the result of checking a custom MAIL FROM (return-path) domain
// against the From domain it sends for.
type MailFrom struct {
	Domain     string `json:"domain"`
	FromDomain string `json:"fromDomain"`
	Status     string `json:"status"` // aligned, misaligned, missing, error
	// Alignment under relaxed (default) DMARC SPF alignment.
	Aligned bool `json:"aligned"`
	// MX is true when bounces can be received (MX, or a CNAME to the provider).
	MX    bool   `json:"mx"`
	SPF   string `json:"spf"`
	Error string `json:"error,omitempty"`
}

// CheckMailFrom checks that the MAIL FROM domain aligns with the From domain,
// can receive bounces and publishes SPF (directly or through a CNAME).
func CheckMailFrom(ctx context.Context, r Resolver, mailFrom, fromDomain string, strict bool) MailFrom {
	out := MailFrom{Domain: mailFrom, FromDomain: fromDomain, Aligned: Aligned(mailFrom, fromDomain, strict)}
	_, mxErr := r.MX(ctx, mailFrom)
	out.MX = mxErr == nil
	if !out.MX {
		if _, err := r.CNAME(ctx, mailFrom); err == nil {
			out.MX = true // provider-managed return path (CNAME)
		}
	}
	spf := CheckSPF(ctx, r, mailFrom)
	out.SPF = spf.Status
	switch {
	case !out.MX && spf.Status == StatusMissing:
		out.Status = StatusMissing
		if mxErr != nil && !IsNotFound(mxErr) {
			out.Status, out.Error = StatusError, mxErr.Error()
		}
	case !out.Aligned:
		out.Status = StatusMisalign
	default:
		out.Status = StatusAligned
	}
	return out
}
