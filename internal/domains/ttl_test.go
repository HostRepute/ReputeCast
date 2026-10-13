package domains

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

// TestLookupTTL asks a local DNS server: the lowest TTL of the record set,
// an error when there is no record of the type.
func TestLookupTTL(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &dns.Server{PacketConn: pc, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		q := r.Question[0]
		if q.Qtype == dns.TypeTXT && q.Name == "news.example.com." {
			for _, ttl := range []uint32{3600, 1800} {
				m.Answer = append(m.Answer, &dns.TXT{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: ttl}, Txt: []string{"v=spf1 -all"}})
			}
		}
		_ = w.WriteMsg(m)
	})}
	go func() { _ = srv.ActivateAndServe() }()
	defer func() { _ = srv.Shutdown() }()

	lookup := LookupTTL([]string{pc.LocalAddr().String()}, 2*time.Second)
	ctx := context.Background()
	got, err := lookup(ctx, "news.example.com", "TXT")
	if err != nil || got != 30*time.Minute {
		t.Fatalf("TTL: %v %v", got, err)
	}
	if _, err := lookup(ctx, "other.example.com", "TXT"); err == nil {
		t.Fatal("no record should be an error")
	}
	if _, err := lookup(ctx, "news.example.com", "NOPE"); err == nil {
		t.Fatal("unknown type should be an error")
	}
	s := &Service{TTL: lookup}
	if n := s.ttl(ctx, "news.example.com", "TXT"); n != 1800 {
		t.Fatalf("service TTL: %d", n)
	}
	if n := (&Service{}).ttl(ctx, "news.example.com", "TXT"); n != 0 {
		t.Fatalf("no lookup: %d", n)
	}
}
