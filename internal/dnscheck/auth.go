package dnscheck

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/netip"
	"net/url"
	"slices"
	"strings"

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

// DMARCRecord is a DMARC Policy Record (RFC 9989 §4.7) with its defaults
// filled in: SubdomainPolicy is p= without an sp= tag, NonexistentPolicy is
// the subdomain policy without an np= tag, and PSD is "u" without a valid
// psd= tag.
type DMARCRecord struct {
	Policy, SubdomainPolicy, NonexistentPolicy, PSD string
	TestMode, StrictDKIM, StrictSPF                 bool
	RUA, RUF                                        []string
}

// ParseDMARC reads a DMARC Policy Record. ok is false when txt isn't one
// (its first tag isn't v=DMARC1) or when receivers apply no DMARC with it
// (§4.10.1): p= missing or invalid, or sp= or np= invalid, and no valid
// rua= URI. With a valid rua= URI such a record reads as p=none. Tags RFC
// 9989 doesn't define (pct=, ri=, rf=, anything unknown) are ignored, and
// other invalid values fall back to their defaults. Tag names and the
// values of p, sp, np, adkim, aspf, t and psd are case-insensitive (ABNF
// strings); the alignment and test-mode fields are filled even when ok is
// false.
func ParseDMARC(txt string) (DMARCRecord, bool) {
	if !isDMARC(txt) {
		return DMARCRecord{}, false
	}
	tags := dmarcTags(txt)
	rec := DMARCRecord{PSD: "u", RUA: dmarcURIs(tags["rua"]), RUF: dmarcURIs(tags["ruf"]),
		TestMode: strings.EqualFold(tags["t"], "y"), StrictDKIM: strings.EqualFold(tags["adkim"], "s"), StrictSPF: strings.EqualFold(tags["aspf"], "s")}
	if psd := strings.ToLower(tags["psd"]); psd == "y" || psd == "n" {
		rec.PSD = psd
	}
	valid := true
	policy := func(tag string) string {
		v, set := tags[tag]
		switch v = strings.ToLower(v); v {
		case "none", "quarantine", "reject":
			return v
		}
		if set || tag == "p" {
			valid = false
		}
		return ""
	}
	p, sp, np := policy("p"), policy("sp"), policy("np")
	if !valid {
		if !slices.ContainsFunc(rec.RUA, validDMARCURI) {
			return rec, false
		}
		p, sp, np = "none", "", ""
	}
	rec.Policy = p
	rec.SubdomainPolicy = cmp.Or(sp, p)
	rec.NonexistentPolicy = cmp.Or(np, rec.SubdomainPolicy)
	return rec, true
}

// isDMARC reports whether a TXT record's first tag is v=DMARC1; receivers
// discard every other record at a _dmarc name.
func isDMARC(txt string) bool {
	first, _, _ := strings.Cut(txt, ";")
	k, v, ok := strings.Cut(first, "=")
	return ok && strings.EqualFold(strings.TrimSpace(k), "v") && strings.TrimSpace(v) == "DMARC1"
}

// dmarcTags is parseTags with lower-case tag names; of two spellings of a
// name, the lower-case one wins.
func dmarcTags(txt string) map[string]string {
	tags := parseTags(txt)
	out := make(map[string]string, len(tags))
	for _, k := range slices.Sorted(maps.Keys(tags)) {
		out[strings.ToLower(k)] = tags[k]
	}
	return out
}

func dmarcURIs(list string) []string {
	var out []string
	for _, u := range strings.Split(list, ",") {
		if u = strings.TrimSpace(u); u != "" {
			out = append(out, u)
		}
	}
	return out
}

// validDMARCURI reports whether a rua= entry is a syntactically valid URI,
// ignoring the obsolete "!size" suffix.
func validDMARCURI(s string) bool {
	s, _, _ = strings.Cut(s, "!")
	u, err := url.Parse(s)
	return err == nil && u.Scheme != "" && (u.Opaque != "" || u.Host != "")
}

// DMARC is the result of DMARC policy discovery for a domain.
type DMARC struct {
	Domain string `json:"domain"`
	// PolicyDomain is where the record was found: the domain itself or the
	// parent the DNS Tree Walk found it at (RFC 9989 §4.10.1).
	PolicyDomain string `json:"policyDomain,omitempty"`
	Status       string `json:"status"`
	Record       string `json:"record,omitempty"`
	// Policy applied to Domain: p= of its own record; of a parent's, np=
	// when Domain doesn't exist (NXDOMAIN), else sp=, both defaulting to p=.
	Policy            string `json:"policy,omitempty"`
	SubdomainPolicy   string `json:"subdomainPolicy,omitempty"`
	NonexistentPolicy string `json:"nonexistentPolicy,omitempty"`
	// TestMode is t=y: receivers apply the policy one level lower.
	TestMode bool `json:"testMode,omitempty"`
	// PSD is the record's psd= flag, y or n; empty for the default u.
	PSD        string   `json:"psd,omitempty"`
	RUA        []string `json:"rua,omitempty"`
	RUF        []string `json:"ruf,omitempty"`
	StrictDKIM bool     `json:"strictDkim"`
	StrictSPF  bool     `json:"strictSpf"`
	Error      string   `json:"error,omitempty"`
}

// dmarcWalkQueries bounds the DNS Tree Walk (RFC 9989 §4.10).
const dmarcWalkQueries = 8

// dmarcWalk lists the names policy discovery asks for: name itself, then
// its parents down to the top-level label, a name of more than 8 labels
// jumping to its last 7 first, so 8 names at most.
func dmarcWalk(name string) []string {
	out := make([]string, 0, dmarcWalkQueries)
	out = append(out, name)
	rest := name
	for n := strings.Count(name, "."); n >= 1; n-- {
		_, rest, _ = strings.Cut(rest, ".")
		if n < dmarcWalkQueries {
			out = append(out, rest)
		}
	}
	return out
}

// dmarcCandidates are the v=DMARC1 records among a _dmarc name's TXT
// records, and the first one that only looks like it ("v=DMARC1 p=reject",
// "v=dmarc1"), which receivers discard.
func dmarcCandidates(txts []string) (recs []string, malformed string) {
	for _, t := range txts {
		t = strings.TrimSpace(t)
		switch {
		case isDMARC(t):
			recs = append(recs, t)
		case malformed == "" && len(t) >= 7 && strings.EqualFold(t[:7], "v=DMARC"):
			malformed = t
		}
	}
	return recs, malformed
}

// CheckDMARC finds the DMARC policy for domain by RFC 9989 policy discovery
// (§4.10.1): _dmarc.<domain>, then the DNS Tree Walk up its parents, at
// most 8 queries, to the first usable record or one with psd=y or psd=n.
// Receivers discard several records at one name and keep walking; at the
// domain itself several records, or one receivers can't use, are invalid:
// its owner has to fix them. A DNS error stops the walk: the policy is
// unknown, not missing.
func CheckDMARC(ctx context.Context, r Resolver, domain string) DMARC {
	out := DMARC{Domain: domain}
	invalid := func(at, record, why string) DMARC {
		out.PolicyDomain, out.Record, out.Status, out.Error = at, record, StatusInvalid, why
		return out
	}
	name := strings.ToLower(strings.TrimSuffix(domain, "."))
	var lastErr error
	for i, target := range dmarcWalk(name) {
		txts, err := r.TXT(ctx, "_dmarc."+target)
		if err != nil {
			if !IsNotFound(err) {
				out.Status, out.Error = StatusError, err.Error()
				return out
			}
			lastErr = err
			continue
		}
		recs, malformed := dmarcCandidates(txts)
		own := i == 0
		switch {
		case own && len(recs) > 1:
			return invalid(target, recs[0], "more than one DMARC record")
		case own && len(recs) == 0 && malformed != "":
			return invalid(target, malformed, "the record doesn't start with v=DMARC1")
		case len(recs) != 1:
			lastErr = fmt.Errorf("_dmarc.%s: no single v=DMARC1 record: %w", target, ErrNoData)
			continue
		}
		rec, ok := ParseDMARC(recs[0])
		if !ok {
			const why = "no valid p= policy and no rua= address"
			if own {
				return invalid(target, recs[0], why)
			}
			lastErr = fmt.Errorf("_dmarc.%s: %s: %w", target, why, ErrNoData)
			if rec.PSD != "u" {
				break
			}
			continue
		}
		out.PolicyDomain, out.Record = target, recs[0]
		out.Policy = rec.Policy
		if !own {
			out.Policy = rec.SubdomainPolicy
			// np= applies to an author domain that doesn't exist; the lookup
			// is made only when np= would change the answer.
			if rec.NonexistentPolicy != rec.SubdomainPolicy {
				if _, err := r.TXT(ctx, name); errors.Is(err, ErrNXDomain) {
					out.Policy = rec.NonexistentPolicy
				}
			}
		}
		out.SubdomainPolicy, out.NonexistentPolicy, out.TestMode = rec.SubdomainPolicy, rec.NonexistentPolicy, rec.TestMode
		if rec.PSD != "u" {
			out.PSD = rec.PSD
		}
		out.RUA, out.RUF, out.StrictDKIM, out.StrictSPF = rec.RUA, rec.RUF, rec.StrictDKIM, rec.StrictSPF
		out.Status = StatusPass
		return out
	}
	out.Status = statusFor(lastErr)
	if lastErr != nil {
		out.Error = lastErr.Error()
	}
	return out
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
