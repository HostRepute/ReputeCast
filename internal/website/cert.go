package website

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/netguard"
)

// The website's own certificate: the site is served at the custom domain,
// so the Website tab says whether readers get a valid certificate there
// ("Live · news.acme.io · certificate valid"). It is checked when the tab
// is read (at most hourly) and by the hourly domain job: a check of the
// site ReputeCast serves, not monitoring of other hosts.

// CertInfo is what a check of the site's certificate found.
type CertInfo struct {
	// Status is valid, failing or unknown (the check couldn't connect).
	Status   string
	Issuer   string
	NotAfter time.Time
	// Detail says why a certificate fails, in English from the TLS stack.
	Detail    string
	CheckedAt time.Time
}

// CertChecker checks the certificate a domain serves.
type CertChecker interface {
	Check(ctx context.Context, domain string) CertInfo
}

// TLSChecker connects to the domain on port 443 through the outbound guard.
type TLSChecker struct {
	Policy  netguard.Policy
	Timeout time.Duration
	Now     func() time.Time
}

// Check implements CertChecker.
func (c TLSChecker) Check(ctx context.Context, domain string) CertInfo {
	now := time.Now()
	if c.Now != nil {
		now = c.Now()
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	d := c.Policy.Dialer(timeout)
	dctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, err := (&tls.Dialer{NetDialer: d, Config: &tls.Config{ServerName: domain, MinVersion: tls.VersionTLS12}}).DialContext(dctx, "tcp", net.JoinHostPort(domain, "443"))
	if err != nil {
		var cv *tls.CertificateVerificationError
		if errors.As(err, &cv) || strings.Contains(err.Error(), "x509") || strings.Contains(err.Error(), "certificate") {
			return CertInfo{Status: "failing", Detail: err.Error(), CheckedAt: now}
		}
		return CertInfo{Status: "unknown", Detail: err.Error(), CheckedAt: now}
	}
	defer func() { _ = conn.Close() }()
	state := conn.(*tls.Conn).ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return CertInfo{Status: "failing", Detail: "no certificate", CheckedAt: now}
	}
	leaf := state.PeerCertificates[0]
	issuer := strings.Join(leaf.Issuer.Organization, ", ")
	if issuer == "" {
		issuer = leaf.Issuer.CommonName
	}
	return CertInfo{Status: "valid", Issuer: issuer, NotAfter: leaf.NotAfter, CheckedAt: now}
}

// certCache keeps each domain's last check for an hour.
type certCache struct {
	mu sync.Mutex
	m  map[string]CertInfo
}

// cert is the site's certificate, checked at most hourly.
func (s *Service) cert(ctx context.Context, domain string, force bool) CertInfo {
	if s.Certs == nil || domain == "" {
		return CertInfo{Status: "unknown"}
	}
	s.certs.mu.Lock()
	c, ok := s.certs.m[domain]
	s.certs.mu.Unlock()
	if ok && !force && s.now().Sub(c.CheckedAt) < time.Hour {
		return c
	}
	c = s.Certs().Check(ctx, domain)
	if c.CheckedAt.IsZero() {
		c.CheckedAt = s.now()
	}
	s.certs.mu.Lock()
	if s.certs.m == nil {
		s.certs.m = map[string]CertInfo{}
	}
	s.certs.m[domain] = c
	s.certs.mu.Unlock()
	return c
}
