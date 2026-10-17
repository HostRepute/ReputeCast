package smtprelay

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/acme/autocert"

	"github.com/hostrepute/reputecast/internal/store"
)

// certSource serves the relay's certificate.
type certSource interface {
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
}

// fileSource reads the certificate and key an operator manages, again when
// the files change.
type fileSource struct {
	certFile, keyFile string
	relay             *Relay

	mu      sync.Mutex
	cert    *tls.Certificate
	modTime time.Time
	checked time.Time
}

// GetCertificate implements certSource.
func (f *fileSource) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	if f.cert != nil && now.Sub(f.checked) < 30*time.Second {
		return f.cert, nil
	}
	f.checked = now
	st, err := os.Stat(f.certFile)
	if err != nil {
		if f.cert != nil {
			return f.cert, nil
		}
		f.relay.recordCert(nil, err)
		return nil, err
	}
	if f.cert != nil && st.ModTime().Equal(f.modTime) {
		return f.cert, nil
	}
	c, err := tls.LoadX509KeyPair(f.certFile, f.keyFile)
	if err != nil {
		f.relay.recordCert(nil, err)
		if f.cert != nil {
			return f.cert, nil
		}
		return nil, err
	}
	f.cert, f.modTime = &c, st.ModTime()
	f.relay.recordCert(&c, nil)
	return f.cert, nil
}

// acmeSource gets and renews a Let's Encrypt certificate for the relay's
// hostname. The account key and certificates are cached in Postgres,
// sealed, so every web node shares them and any of them answers the
// HTTP-01 challenge.
type acmeSource struct {
	m     *autocert.Manager
	relay *Relay
	mu    sync.Mutex
	last  *x509.Certificate
}

func newACMESource(r *Relay, hostname string) *acmeSource {
	a := &acmeSource{relay: r}
	a.m = &autocert.Manager{Prompt: autocert.AcceptTOS, HostPolicy: autocert.HostWhitelist(hostname), Cache: dbCache{r: r}}
	return a
}

// GetCertificate implements certSource. Mail clients don't always send
// SNI, so the hostname stands in for it.
func (a *acmeSource) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if hello.ServerName == "" {
		h := *hello
		for _, name := range a.hostnames() {
			h.ServerName = name
		}
		hello = &h
	}
	c, err := a.m.GetCertificate(hello)
	if err != nil {
		a.relay.recordCert(nil, err)
		return nil, err
	}
	if c.Leaf != nil {
		a.mu.Lock()
		changed := a.last == nil || !a.last.NotAfter.Equal(c.Leaf.NotAfter)
		a.last = c.Leaf
		a.mu.Unlock()
		if changed {
			a.relay.recordCert(c, nil)
		}
	}
	return c, nil
}

func (a *acmeSource) hostnames() []string {
	if cfg, ok := a.relay.Listening(); ok {
		return []string{cfg.Hostname}
	}
	return nil
}

// ACMEHandler answers Let's Encrypt's HTTP-01 challenge for the relay's
// hostname (/.well-known/acme-challenge/…); the web router mounts it.
func (r *Relay) ACMEHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		a := r.acme
		r.mu.Unlock()
		if a == nil {
			http.NotFound(w, req)
			return
		}
		a.m.HTTPHandler(http.NotFoundHandler()).ServeHTTP(w, req)
	})
}

// recordCert keeps the served certificate's expiry and issuer, or why
// getting one failed, for the instance settings and the status checks.
func (r *Relay) recordCert(c *tls.Certificate, failure error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p := store.RelaySetCertParams{}
	if c != nil {
		leaf := c.Leaf
		if leaf == nil && len(c.Certificate) > 0 {
			leaf, _ = x509.ParseCertificate(c.Certificate[0])
		}
		if leaf != nil {
			na := leaf.NotAfter
			p.CertNotAfter, p.CertIssuer = &na, issuerName(leaf)
		}
	} else if failure != nil {
		row, err := r.Q.RelaySettings(ctx)
		if err == nil {
			p.CertNotAfter, p.CertIssuer = row.CertNotAfter, row.CertIssuer
		}
		now := r.now()
		p.RenewError, p.RenewFailedAt = clip(failure.Error(), 500), &now
	}
	if err := r.Q.RelaySetCert(ctx, p); err != nil {
		r.log().Warn("record the relay's certificate", "err", err)
	}
}

func issuerName(c *x509.Certificate) string {
	if len(c.Issuer.Organization) > 0 {
		return c.Issuer.Organization[0]
	}
	return c.Issuer.CommonName
}

// dbCache is autocert's cache in smtp_relay_acme, sealed with the
// instance's secret key.
type dbCache struct{ r *Relay }

func acmeAD(key string) string { return "smtp_relay_acme:" + key }

// Get implements autocert.Cache.
func (c dbCache) Get(ctx context.Context, key string) ([]byte, error) {
	sealed, err := c.r.Q.RelayACMEGet(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, autocert.ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	return c.r.Box.Open(sealed, acmeAD(key))
}

// Put implements autocert.Cache.
func (c dbCache) Put(ctx context.Context, key string, data []byte) error {
	sealed, err := c.r.Box.Seal(data, acmeAD(key))
	if err != nil {
		return err
	}
	return c.r.Q.RelayACMEPut(ctx, store.RelayACMEPutParams{Key: key, Sealed: sealed})
}

// Delete implements autocert.Cache.
func (c dbCache) Delete(ctx context.Context, key string) error {
	return c.r.Q.RelayACMEDelete(ctx, key)
}
