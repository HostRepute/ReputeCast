package demo

import (
	"context"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/website"
)

// WebsiteCertOverlay answers the website's certificate checks for the
// demo's domains (acme.io and its subdomains, news.acme.io among them):
// a Let's Encrypt certificate that renews on time, without going to the
// network. Other domains go to next (nil: unknown).
func WebsiteCertOverlay(next website.CertChecker) website.CertChecker {
	return certOverlay{next: next}
}

type certOverlay struct{ next website.CertChecker }

// Check implements website.CertChecker.
func (o certOverlay) Check(ctx context.Context, domain string) website.CertInfo {
	d := strings.ToLower(strings.TrimSuffix(domain, "."))
	if d == "acme.io" || strings.HasSuffix(d, ".acme.io") {
		// Issued 1 Oct, 90 days: renews 30 days before it ends, on 30 Nov.
		issued := time.Date(2026, 10, 1, 4, 12, 0, 0, time.UTC)
		return website.CertInfo{Status: "valid", Issuer: "Let's Encrypt", NotAfter: issued.AddDate(0, 0, 90), CheckedAt: time.Now()}
	}
	if o.next == nil {
		return website.CertInfo{Status: "unknown", Detail: "not checked", CheckedAt: time.Now()}
	}
	return o.next.Check(ctx, domain)
}
