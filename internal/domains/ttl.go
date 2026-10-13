package domains

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// TTLFunc returns how long resolvers may cache the records of a type at a
// name ("Pending · old value still cached (TTL 3600)").
type TTLFunc func(ctx context.Context, name, rrtype string) (time.Duration, error)

// ttl looks up a record's TTL when the service has a lookup; 0 otherwise.
func (s *Service) ttl(ctx context.Context, name, rrtype string) uint32 {
	if s.TTL == nil {
		return 0
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	d, err := s.TTL(cctx, name, rrtype)
	if err != nil || d <= 0 {
		return 0
	}
	return uint32(d / time.Second)
}

// LookupTTL asks the resolvers (host or host:port; empty means the
// system's from /etc/resolv.conf) for the records of a type at a name and
// returns their lowest TTL: how long a resolver may still serve the value
// it has cached.
func LookupTTL(servers []string, timeout time.Duration) TTLFunc {
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	udp := &dns.Client{Net: "udp", Timeout: timeout, UDPSize: 1232}
	tcp := &dns.Client{Net: "tcp", Timeout: timeout}
	return func(ctx context.Context, name, rrtype string) (time.Duration, error) {
		qtype, ok := dns.StringToType[strings.ToUpper(rrtype)]
		if !ok {
			return 0, fmt.Errorf("ttl: unknown record type %q", rrtype)
		}
		list := servers
		if len(list) == 0 {
			cfg, err := dns.ClientConfigFromFile("/etc/resolv.conf")
			if err != nil {
				return 0, err
			}
			for _, s := range cfg.Servers {
				list = append(list, net.JoinHostPort(s, cfg.Port))
			}
		}
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(name), qtype)
		m.RecursionDesired = true
		m.SetEdns0(1232, false)
		lastErr := errors.New("ttl: no resolver answered")
		for _, server := range list {
			if _, _, err := net.SplitHostPort(server); err != nil {
				server = net.JoinHostPort(strings.Trim(server, "[]"), "53")
			}
			in, _, err := udp.ExchangeContext(ctx, m, server)
			if err == nil && in != nil && in.Truncated {
				in, _, err = tcp.ExchangeContext(ctx, m, server)
			}
			if err != nil {
				lastErr = err
				continue
			}
			var low uint32
			found := false
			for _, rr := range in.Answer {
				if rr.Header().Rrtype != qtype {
					continue
				}
				if !found || rr.Header().Ttl < low {
					low, found = rr.Header().Ttl, true
				}
			}
			if !found {
				return 0, fmt.Errorf("ttl: no %s record at %s", rrtype, name)
			}
			return time.Duration(low) * time.Second, nil
		}
		return 0, lastErr
	}
}
