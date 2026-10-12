package dnscheck

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
)

var ctx = context.Background()

// acmeZone is the sample sending setup: SES with a custom MAIL FROM,
// Postmark with its own return path and Postal on a dedicated IP.
func acmeZone() *Zone {
	z := NewZone()
	z.Add("news.acme.io", "TXT", "v=spf1 include:amazonses.com include:spf.mtasv.net ip4:203.0.113.25 -all")
	z.Add("amazonses.com", "TXT", "v=spf1 ip4:199.255.192.0/22 ip4:54.240.0.0/18 -all")
	z.Add("spf.mtasv.net", "TXT", "v=spf1 ip4:50.31.156.96/27 -all")
	z.Add("bounce.news.acme.io", "MX", "10 feedback-smtp.us-east-1.amazonses.com")
	z.Add("bounce.news.acme.io", "TXT", "v=spf1 include:amazonses.com ~all")
	z.Add("pm-bounces.news.acme.io", "CNAME", "pm.mtasv.net")
	z.Add("pm.mtasv.net", "TXT", "v=spf1 include:spf.mtasv.net -all")
	z.Add("ses1._domainkey.news.acme.io", "CNAME", "ses1.dkim.amazonses.com")
	z.Add("ses1.dkim.amazonses.com", "TXT", "p="+strings.Repeat("A", 392))
	z.Add("pm._domainkey.news.acme.io", "TXT", "k=rsa; p="+strings.Repeat("B", 392))
	z.Add("_dmarc.acme.io", "TXT", "v=DMARC1; p=quarantine; sp=quarantine; rua=mailto:dmarc@reports.hostrepute.com")
	z.Add("203.0.113.25", "PTR", "mta.acme.io.")
	z.Add("mta.acme.io", "A", "203.0.113.25")
	return z
}

func TestSPF(t *testing.T) {
	z := acmeZone()
	s := CheckSPF(ctx, z, "news.acme.io")
	if s.Status != StatusPass || s.All != "-all" || s.Lookups != 2 {
		t.Fatalf("spf: %+v", s)
	}
	if !s.Covers("amazonses.com", netip.Addr{}) || !s.Covers("", netip.MustParseAddr("203.0.113.25")) || !s.Covers("", netip.MustParseAddr("54.240.8.30")) {
		t.Fatalf("covers: %+v", s)
	}
	if s.Covers("sendgrid.net", netip.MustParseAddr("198.51.100.77")) {
		t.Fatal("covers too much")
	}

	z.Add("two.example", "TXT", "v=spf1 -all").Add("two.example", "TXT", "v=spf1 ~all")
	if s := CheckSPF(ctx, z, "two.example"); s.Status != StatusInvalid {
		t.Fatalf("two records: %+v", s)
	}
	z.Add("plain.example", "TXT", "google-site-verification=abc")
	if s := CheckSPF(ctx, z, "plain.example"); s.Status != StatusMissing {
		t.Fatalf("no spf: %+v", s)
	}
	if s := CheckSPF(ctx, z, "nothing.example"); s.Status != StatusMissing {
		t.Fatalf("nxdomain: %+v", s)
	}
	z.Add("open.example", "TXT", "v=spf1 +all")
	if s := CheckSPF(ctx, z, "open.example"); s.Status != StatusInvalid {
		t.Fatalf("+all: %+v", s)
	}
	// Eleven includes break the 10-lookup limit.
	var b strings.Builder
	b.WriteString("v=spf1")
	for i := range 11 {
		name := "i" + string(rune('a'+i)) + ".example"
		z.Add(name, "TXT", "v=spf1 -all")
		b.WriteString(" include:" + name)
	}
	z.Add("many.example", "TXT", b.String()+" -all")
	if s := CheckSPF(ctx, z, "many.example"); s.Status != StatusTooMany || s.Lookups != 11 {
		t.Fatalf("too many: %+v", s)
	}
	z.Fail("flaky.example")
	if s := CheckSPF(ctx, z, "flaky.example"); s.Status != StatusError {
		t.Fatalf("servfail: %+v", s)
	}
}

func TestDKIM(t *testing.T) {
	z := acmeZone()
	if k := CheckDKIM(ctx, z, "ses1", "news.acme.io"); k.Status != StatusPass || k.Target != "ses1.dkim.amazonses.com" || k.KeyBits != 2048 {
		t.Fatalf("cname key: %+v", k)
	}
	if k := CheckDKIM(ctx, z, "pm", "news.acme.io"); k.Status != StatusPass || k.KeyType != "rsa" {
		t.Fatalf("txt key: %+v", k)
	}
	z.Add("old._domainkey.news.acme.io", "TXT", "v=DKIM1; p=")
	if k := CheckDKIM(ctx, z, "old", "news.acme.io"); k.Status != StatusRevoked {
		t.Fatalf("revoked: %+v", k)
	}
	if k := CheckDKIM(ctx, z, "nope", "news.acme.io"); k.Status != StatusMissing {
		t.Fatalf("missing: %+v", k)
	}
}

func TestDMARC(t *testing.T) {
	z := acmeZone()
	d := CheckDMARC(ctx, z, "news.acme.io")
	if d.Status != StatusPass || d.PolicyDomain != "acme.io" || d.Policy != "quarantine" || !d.Enforced() || len(d.RUA) != 1 {
		t.Fatalf("org fallback: %+v", d)
	}
	z.Add("_dmarc.example.org", "TXT", "v=DMARC1; p=reject; sp=none; pct=50")
	if d := CheckDMARC(ctx, z, "mail.example.org"); d.Policy != "none" || d.Percent != 50 {
		t.Fatalf("sp applies to subdomains: %+v", d)
	}
	if d := CheckDMARC(ctx, z, "example.org"); d.Policy != "reject" {
		t.Fatalf("p applies to the org domain: %+v", d)
	}
	z.Add("_dmarc.broken.example", "TXT", "v=DMARC1; p=sometimes")
	if d := CheckDMARC(ctx, z, "broken.example"); d.Status != StatusInvalid {
		t.Fatalf("invalid: %+v", d)
	}
	if d := CheckDMARC(ctx, z, "none.example"); d.Status != StatusMissing || d.Enforced() {
		t.Fatalf("missing: %+v", d)
	}
}

func TestOrgDomainAndAlignment(t *testing.T) {
	cases := map[string]string{"news.acme.io": "acme.io", "acme.io": "acme.io", "a.b.example.co.uk": "example.co.uk", "Mail.Example.COM.": "example.com"}
	for in, want := range cases {
		if got := OrgDomain(in); got != want {
			t.Errorf("OrgDomain(%q) = %q, want %q", in, got, want)
		}
	}
	if !Aligned("bounce.news.acme.io", "news.acme.io", false) || Aligned("bounce.news.acme.io", "news.acme.io", true) || Aligned("amazonses.com", "acme.io", false) {
		t.Fatal("alignment")
	}
}

func TestBIMIAndMailFrom(t *testing.T) {
	z := acmeZone()
	if b := CheckBIMI(ctx, z, "news.acme.io"); b.Status != StatusMissing {
		t.Fatalf("bimi: %+v", b)
	}
	z.Add("default._bimi.acme.io", "TXT", "v=BIMI1; l=https://acme.io/logo.svg; a=https://acme.io/vmc.pem")
	if b := CheckBIMI(ctx, z, "acme.io"); b.Status != StatusPass || b.Certificate == "" {
		t.Fatalf("bimi: %+v", b)
	}
	if m := CheckMailFrom(ctx, z, "bounce.news.acme.io", "news.acme.io", false); m.Status != StatusAligned || !m.MX || m.SPF != StatusPass {
		t.Fatalf("mail from: %+v", m)
	}
	if m := CheckMailFrom(ctx, z, "pm-bounces.news.acme.io", "news.acme.io", false); m.Status != StatusAligned || !m.MX {
		t.Fatalf("cname mail from: %+v", m)
	}
	z.Add("bounces.other.example", "MX", "10 mx.other.example")
	if m := CheckMailFrom(ctx, z, "bounces.other.example", "news.acme.io", false); m.Status != StatusMisalign {
		t.Fatalf("misaligned: %+v", m)
	}
}

func TestReverseDNS(t *testing.T) {
	z := acmeZone()
	if r := CheckReverseDNS(ctx, z, netip.MustParseAddr("203.0.113.25")); !r.Matches || r.PTR != "mta.acme.io" || r.Status != StatusPass {
		t.Fatalf("fcrdns: %+v", r)
	}
	z.Add("203.0.113.26", "PTR", "mta.acme.io")
	if r := CheckReverseDNS(ctx, z, netip.MustParseAddr("203.0.113.26")); r.Matches || r.Status != StatusFail {
		t.Fatalf("forward mismatch: %+v", r)
	}
	if r := CheckReverseDNS(ctx, z, netip.MustParseAddr("198.51.100.77")); r.Status != StatusMissing {
		t.Fatalf("no ptr: %+v", r)
	}
}

func TestCheckDomainSummary(t *testing.T) {
	z := acmeZone()
	reqs := []Requirement{
		{ProviderID: "ses", Purpose: "dkim", Type: "CNAME", Name: "ses1._domainkey.news.acme.io", Value: "ses1.dkim.amazonses.com"},
		{ProviderID: "ses", Purpose: "return_path", Type: "MX", Name: "bounce.news.acme.io", Value: "10 feedback-smtp.us-east-1.amazonses.com"},
		{ProviderID: "ses", Purpose: "spf", Type: "TXT", Name: "bounce.news.acme.io", Value: "v=spf1 include:amazonses.com ~all"},
		{ProviderID: "postmark", Purpose: "dkim", Type: "TXT", Name: "pm._domainkey.news.acme.io", Value: "k=rsa; p=" + strings.Repeat("B", 392)},
		{ProviderID: "postmark", Purpose: "return_path", Type: "CNAME", Name: "pm-bounces.news.acme.io", Value: "pm.mtasv.net"},
	}
	a := CheckDomain(ctx, z, "news.acme.io", reqs)
	want := AuthSummary{SPF: "pass", DKIM: "pass", DMARC: "quarantine", BIMI: "missing", ReturnPath: "aligned"}
	if a.Summary != want {
		t.Fatalf("summary %+v, want %+v", a.Summary, want)
	}
	if len(a.Records) != 5 || len(a.DKIM) != 2 || len(a.MailFrom) != 2 {
		t.Fatalf("details: %+v", a)
	}
	for _, rc := range a.Records {
		if rc.Status != RecordVerified {
			t.Fatalf("record %s: %+v", rc.Name, rc)
		}
	}
	// A DKIM record pointing somewhere else fails the summary.
	reqs = append(reqs, Requirement{Purpose: "dkim", Type: "CNAME", Name: "ses2._domainkey.news.acme.io", Value: "ses2.dkim.amazonses.com"})
	if a := CheckDomain(ctx, z, "news.acme.io", reqs); a.Summary.DKIM != StatusMissing {
		t.Fatalf("missing selector: %+v", a.Summary)
	}
}

func TestDNSBL(t *testing.T) {
	z := NewZone()
	zen, dbl, surbl := Builtin[0], Builtin[1], Builtin[2]
	z.Add("2.0.0.127.zen.spamhaus.org", "A", "127.0.0.2")
	z.Add("2.0.0.127.zen.spamhaus.org", "TXT", "https://check.spamhaus.org/sbl/query/SBL1")
	z.Add("listed.example.dbl.spamhaus.org", "A", "127.0.1.2")
	z.Add("4.0.0.127.zen.spamhaus.org", "A", "127.255.255.254")
	z.Add("blocked.example.multi.surbl.org", "A", "127.0.0.1")

	if l := Lookup(ctx, z, zen, "127.0.0.2"); !l.Listed || l.Codes[0] != "127.0.0.2" || l.Reason == "" || l.Err != nil {
		t.Fatalf("listed ip: %+v", l)
	}
	if l := Lookup(ctx, z, zen, "203.0.113.25"); l.Listed || l.Err != nil {
		t.Fatalf("clean ip: %+v", l)
	}
	if l := Lookup(ctx, z, dbl, "listed.example"); !l.Listed {
		t.Fatalf("listed domain: %+v", l)
	}
	if l := Lookup(ctx, z, zen, "127.0.0.4"); l.Listed || !errors.Is(l.Err, ErrRefused) {
		t.Fatalf("public resolver answer must not count as listed: %+v", l)
	}
	if l := Lookup(ctx, z, surbl, "blocked.example"); l.Listed || !errors.Is(l.Err, ErrRefused) {
		t.Fatalf("surbl refusal: %+v", l)
	}
	if n, _ := zen.QueryName("2001:db8::1"); !strings.HasSuffix(n, ".8.b.d.0.1.0.0.2.zen.spamhaus.org") {
		t.Fatalf("ipv6 query name: %s", n)
	}

	assets := []Asset{{Name: "news.acme.io", Kind: KindDomain}, {Name: "203.0.113.25", Kind: KindIP}}
	var calls atomic.Int32
	res := CheckAll(ctx, z, Builtin, assets, 4, time.Second, func(done, total int) {
		calls.Add(1)
		if total != 8 {
			t.Errorf("total %d", total)
		}
	})
	// 3 domain lists for the domain, 5 IP lists for the IP.
	if len(res) != 8 || calls.Load() != 8 {
		t.Fatalf("checked %d, progress %d", len(res), calls.Load())
	}
	for _, r := range res {
		if r.Listed || r.Err != nil {
			t.Fatalf("%s on %s: %+v", r.Asset, r.List.Name, r)
		}
	}
}

func TestOverlay(t *testing.T) {
	z := acmeZone()
	fallback := NewZone().Add("example.com", "TXT", "v=spf1 -all")
	o := Overlay(z, []string{"acme.io"}, fallback)
	if _, err := o.TXT(ctx, "unknown.acme.io"); !errors.Is(err, ErrNXDomain) {
		t.Fatalf("owned suffix answers from the zone: %v", err)
	}
	if txt, err := o.TXT(ctx, "example.com"); err != nil || txt[0] != "v=spf1 -all" {
		t.Fatalf("fallback: %v %v", txt, err)
	}
}

// TestClient runs the miekg/dns client against a local DNS server, covering
// answers, NXDOMAIN, NODATA, CNAME chains in answers and TCP on truncation.
func TestClient(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("no UDP: %v", err)
	}
	ln, err := net.Listen("tcp", pc.LocalAddr().String())
	if err != nil {
		pc.Close()
		t.Skipf("no TCP on the same port: %v", err)
	}
	h := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		q := r.Question[0]
		switch {
		case q.Name == "news.acme.io." && q.Qtype == dns.TypeTXT:
			m.Answer = append(m.Answer, &dns.TXT{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60}, Txt: []string{"v=spf1 include:amazonses.com", " -all"}})
		case q.Name == "big.example." && q.Qtype == dns.TypeTXT:
			if _, isUDP := w.RemoteAddr().(*net.UDPAddr); isUDP {
				m.Truncated = true
			} else {
				m.Answer = append(m.Answer, &dns.TXT{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60}, Txt: []string{"over tcp"}})
			}
		case q.Name == "sel._domainkey.news.acme.io.":
			m.Answer = append(m.Answer,
				&dns.CNAME{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: 60}, Target: "sel.dkim.amazonses.com."},
				&dns.TXT{Hdr: dns.RR_Header{Name: "sel.dkim.amazonses.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60}, Txt: []string{"p=KEY"}})
		case q.Name == "25.113.0.203.in-addr.arpa." && q.Qtype == dns.TypePTR:
			m.Answer = append(m.Answer, &dns.PTR{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 60}, Ptr: "mta.acme.io."})
		case q.Name == "news.acme.io.":
			// exists, but not with this type: NODATA
		default:
			m.Rcode = dns.RcodeNameError
		}
		_ = w.WriteMsg(m)
	})
	udp := &dns.Server{PacketConn: pc, Handler: h}
	tcp := &dns.Server{Listener: ln, Handler: h}
	go func() { _ = udp.ActivateAndServe() }()
	go func() { _ = tcp.ActivateAndServe() }()
	t.Cleanup(func() { _ = udp.Shutdown(); _ = tcp.Shutdown() })

	c, err := NewClient(Options{Servers: []string{pc.LocalAddr().String()}, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if txt, err := c.TXT(ctx, "news.acme.io"); err != nil || txt[0] != "v=spf1 include:amazonses.com -all" {
		t.Fatalf("txt: %v %v", txt, err)
	}
	if txt, err := c.TXT(ctx, "big.example"); err != nil || txt[0] != "over tcp" {
		t.Fatalf("tcp fallback: %v %v", txt, err)
	}
	if txt, err := c.TXT(ctx, "sel._domainkey.news.acme.io"); err != nil || txt[0] != "p=KEY" {
		t.Fatalf("cname chain: %v %v", txt, err)
	}
	if target, err := c.CNAME(ctx, "sel._domainkey.news.acme.io"); err != nil || target != "sel.dkim.amazonses.com" {
		t.Fatalf("cname: %v %v", target, err)
	}
	if _, err := c.MX(ctx, "news.acme.io"); !errors.Is(err, ErrNoData) {
		t.Fatalf("nodata: %v", err)
	}
	if _, err := c.A(ctx, "missing.example"); !errors.Is(err, ErrNXDomain) || !IsNotFound(err) {
		t.Fatalf("nxdomain: %v", err)
	}
	if ptr, err := c.PTR(ctx, netip.MustParseAddr("203.0.113.25")); err != nil || ptr[0] != "mta.acme.io" {
		t.Fatalf("ptr: %v %v", ptr, err)
	}
	// A server that doesn't answer is a temporary failure, not "not found".
	dead, _ := NewClient(Options{Servers: []string{"127.0.0.1:1"}, Timeout: 200 * time.Millisecond})
	if _, err := dead.TXT(ctx, "news.acme.io"); !errors.Is(err, ErrTemporary) || IsNotFound(err) {
		t.Fatalf("dead server: %v", err)
	}
}
