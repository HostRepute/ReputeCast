package seedbox

import (
	"bufio"
	"bytes"
	"net/netip"
	"net/textproto"
	"regexp"
	"strings"
)

// TestHeader carries a copy's test token ("pt-7f3a91c2"); the token is also
// in the copy's Message-ID, which some providers replace with their own.
const TestHeader = "X-ReputeCast-Test"

// tokenPattern finds a test token in a header value.
var tokenPattern = regexp.MustCompile(`pt-[0-9a-f]{8,32}`)

// Header is a message header read from a seed.
type Header struct {
	textproto.MIMEHeader
	Raw []byte
}

// ParseHeader reads a header block (as IMAP's BODY[HEADER] returns it).
func ParseHeader(raw []byte) Header {
	r := textproto.NewReader(bufio.NewReader(bytes.NewReader(append(bytes.TrimRight(raw, "\r\n"), "\r\n\r\n"...))))
	h, _ := r.ReadMIMEHeader()
	if h == nil {
		h = textproto.MIMEHeader{}
	}
	return Header{MIMEHeader: h, Raw: raw}
}

// uuidPattern finds a copy's id in a Message-ID (<copy id@sender domain>,
// as the providers that keep ReputeCast's Message-ID write it).
var uuidPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// Tokens returns the test tokens the header carries, from X-ReputeCast-Test
// first, then the Message-ID (a token, or the copy's id).
func (h Header) Tokens() []string {
	var out []string
	seen := map[string]bool{}
	add := func(t string) {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, v := range append(h.Values(TestHeader), h.Values("Message-Id")...) {
		for _, t := range tokenPattern.FindAllString(strings.ToLower(v), -1) {
			add(t)
		}
	}
	for _, v := range h.Values("Message-Id") {
		for _, t := range uuidPattern.FindAllString(strings.ToLower(v), -1) {
			add(t)
		}
	}
	for _, v := range h.Values("X-Reputecast-Message") {
		for _, t := range uuidPattern.FindAllString(strings.ToLower(v), -1) {
			add(t)
		}
	}
	return out
}

// Check is one authentication result from the mailbox's
// Authentication-Results header.
type Check struct {
	// Result is the method's result as the header says it: pass, fail,
	// softfail, neutral, none, temperror, permerror, bestguesspass.
	Result string `json:"result"`
	// Domain is smtp.mailfrom's domain (SPF), header.d (DKIM) or header.from (DMARC).
	Domain string `json:"domain,omitempty"`
	// Selector is the DKIM selector (header.s, else the DKIM-Signature's s=).
	Selector string `json:"selector,omitempty"`
	// Detail is the header's own comment ("sender IP is 203.0.113.25").
	Detail string `json:"detail,omitempty"`
}

// Auth is what a copy's headers say about SPF, DKIM and DMARC.
type Auth struct {
	SPF   *Check `json:"spf,omitempty"`
	DKIM  *Check `json:"dkim,omitempty"`
	DMARC *Check `json:"dmarc,omitempty"`
	// Raw is the Authentication-Results header as received, unfolded.
	Raw string `json:"raw,omitempty"`
}

// AuthResults reads the topmost Authentication-Results header (the one the
// receiving mailbox added). Both RFC 8601's form ("mx.google.com; spf=pass…")
// and Microsoft's, which leaves out the authserv-id ("spf=pass (sender IP
// is …) smtp.mailfrom=…"), are read. A DKIM selector the header doesn't
// name comes from the DKIM-Signature of the signing domain.
func (h Header) AuthResults() Auth {
	vals := h.Values("Authentication-Results")
	if len(vals) == 0 {
		// Some servers only write the ARC-Authentication-Results set.
		vals = h.Values("Arc-Authentication-Results")
	}
	var a Auth
	if len(vals) == 0 {
		return a
	}
	raw := unfold(vals[0])
	a.Raw = raw
	for _, part := range splitResults(raw) {
		method, result, props, comment := parseResult(part)
		c := &Check{Result: result, Detail: comment}
		switch method {
		case "spf":
			if a.SPF != nil {
				continue
			}
			c.Domain = domainOf(firstNonEmpty(props["smtp.mailfrom"], props["smtp.helo"]))
			a.SPF = c
		case "dkim":
			// The first passing signature wins; otherwise the first one.
			if a.DKIM != nil && a.DKIM.Result == "pass" {
				continue
			}
			c.Domain = strings.TrimPrefix(firstNonEmpty(props["header.d"], domainOf(props["header.i"])), "@")
			c.Selector = props["header.s"]
			if a.DKIM == nil || result == "pass" {
				a.DKIM = c
			}
		case "dmarc":
			if a.DMARC != nil {
				continue
			}
			c.Domain = props["header.from"]
			a.DMARC = c
		}
	}
	if a.DKIM != nil && a.DKIM.Selector == "" {
		a.DKIM.Selector = h.dkimSelector(a.DKIM.Domain)
	}
	return a
}

// dkimSelector returns s= of the DKIM-Signature signed by domain (or the
// first one).
func (h Header) dkimSelector(domain string) string {
	first := ""
	for _, v := range h.Values("Dkim-Signature") {
		tags := map[string]string{}
		for _, kv := range strings.Split(unfold(v), ";") {
			k, val, ok := strings.Cut(strings.TrimSpace(kv), "=")
			if ok {
				tags[strings.TrimSpace(k)] = strings.Join(strings.Fields(val), "")
			}
		}
		if first == "" {
			first = tags["s"]
		}
		if domain != "" && strings.EqualFold(tags["d"], domain) {
			return tags["s"]
		}
	}
	return first
}

// Hop is the Received line of the server that handed a copy to the mailbox.
type Hop struct {
	// Host is the sending server's name ("mta.acme.io"), IP its address.
	Host string `json:"host,omitempty"`
	IP   string `json:"ip,omitempty"`
	// By is the receiving server ("AM4PEPF00027A6C.mail.protection.outlook.com").
	By string `json:"by,omitempty"`
	// Line is the header as received, unfolded.
	Line string `json:"line"`
}

var (
	receivedFrom = regexp.MustCompile(`(?i)^\s*from\s+(\S+)`)
	receivedBy   = regexp.MustCompile(`(?i)\sby\s+(\S+)`)
	bracketIP    = regexp.MustCompile(`[\[(]((?:\d{1,3}\.){3}\d{1,3}|[0-9a-fA-F:]+:[0-9a-fA-F:.]+)[\])]`)
	ipv6Prefix   = regexp.MustCompile(`^(?i)ipv6:`)
)

// HandOver returns the Received line in which the mailbox accepted the copy
// from the outside: the topmost one whose "from" clause names a public
// address. Internal hops (Gmail's "by 2002:a05:…", Outlook's protection
// servers) are skipped.
func (h Header) HandOver() *Hop {
	for _, v := range h.Values("Received") {
		line := unfold(v)
		m := receivedFrom.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		hop := &Hop{Host: strings.TrimSuffix(m[1], "."), Line: line}
		if b := receivedBy.FindStringSubmatch(line); b != nil {
			hop.By = strings.TrimSuffix(b[1], ";")
		}
		from := line
		if i := strings.Index(strings.ToLower(line), " by "); i > 0 {
			from = line[:i]
		}
		for _, ipm := range bracketIP.FindAllStringSubmatch(from, -1) {
			ip, err := netip.ParseAddr(ipv6Prefix.ReplaceAllString(ipm[1], ""))
			if err != nil || !public(ip) {
				continue
			}
			hop.IP = ip.String()
			return hop
		}
	}
	return nil
}

func public(ip netip.Addr) bool {
	return ip.IsValid() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified() &&
		!netip.MustParsePrefix("100.64.0.0/10").Contains(ip)
}

// unfold joins a folded header value onto one line.
func unfold(v string) string {
	return strings.Join(strings.Fields(v), " ")
}

// splitResults splits an Authentication-Results value on the semicolons
// between results, ignoring those inside comments and quotes.
func splitResults(v string) []string {
	var parts []string
	depth, quoted, start := 0, false, 0
	for i := 0; i < len(v); i++ {
		switch c := v[i]; {
		case c == '\\' && i+1 < len(v):
			i++
		case c == '"':
			quoted = !quoted
		case quoted:
		case c == '(':
			depth++
		case c == ')' && depth > 0:
			depth--
		case c == ';' && depth == 0:
			parts = append(parts, strings.TrimSpace(v[start:i]))
			start = i + 1
		}
	}
	return append(parts, strings.TrimSpace(v[start:]))
}

// parseResult reads "method=result (comment) ptype.prop=value …". Parts
// without "=" in the first word (the authserv-id) give an empty method.
func parseResult(part string) (method, result string, props map[string]string, comment string) {
	props = map[string]string{}
	// Take comments out, keeping the first as the detail.
	var b strings.Builder
	depth := 0
	var cm strings.Builder
	for _, r := range part {
		switch {
		case r == '(':
			depth++
			if depth == 1 {
				continue
			}
		case r == ')' && depth > 0:
			depth--
			if depth == 0 {
				if comment == "" {
					comment = strings.TrimSpace(cm.String())
				}
				cm.Reset()
				b.WriteByte(' ')
				continue
			}
		}
		if depth > 0 {
			cm.WriteRune(r)
			continue
		}
		b.WriteRune(r)
	}
	words := strings.Fields(b.String())
	if len(words) == 0 {
		return "", "", props, comment
	}
	m, res, ok := strings.Cut(words[0], "=")
	if !ok {
		return "", "", props, comment
	}
	method = strings.ToLower(strings.SplitN(m, "/", 2)[0])
	result = strings.ToLower(strings.Trim(res, `"`))
	for _, w := range words[1:] {
		k, v, ok := strings.Cut(w, "=")
		if !ok {
			continue
		}
		props[strings.ToLower(k)] = strings.Trim(v, `"`)
	}
	return method, result, props, comment
}

// domainOf returns the domain of an address or a bare domain.
func domainOf(v string) string {
	v = strings.Trim(strings.TrimSpace(v), "<>")
	if i := strings.LastIndex(v, "@"); i >= 0 {
		return strings.ToLower(v[i+1:])
	}
	return strings.ToLower(v)
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
