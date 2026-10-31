package mailgun

import (
	"context"
	"net/http"
	"slices"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
)

var _ providers.IPReporter = (*Provider)(nil)

// ipList is Mailgun's answer listing IPs: {"items": ["…"], "total_count": n},
// whole, without pages.
type ipList struct {
	Items []string `json:"items"`
}

// SendingIPs implements providers.IPReporter: the account's dedicated IPs
// (GET /v3/ips?dedicated=true) that are assigned to the connection's domain
// (GET /v3/domains/{domain}/ips), the IPs its mail goes out from. An
// account without dedicated IPs, or a domain on none of them, sends from
// Mailgun's shared IPs.
func (p *Provider) SendingIPs(ctx context.Context) (providers.SendingIPs, error) {
	none := providers.SendingIPs{IPs: []string{}}
	var dedicated ipList
	if err := p.api.Do(ctx, "GET /v3/ips", http.MethodGet, "/v3/ips?dedicated=true", nil, &dedicated); err != nil {
		return providers.SendingIPs{}, err
	}
	own, _ := providers.CleanIPs(dedicated.Items)
	if len(own) == 0 {
		return none, nil
	}
	var domain ipList
	if err := p.api.Do(ctx, "GET /v3/domains/{domain}/ips", http.MethodGet, apiclient.Path("/v3/domains/%s/ips", p.s.Domain), nil, &domain); err != nil {
		return providers.SendingIPs{}, err
	}
	assigned, _ := providers.CleanIPs(domain.Items)
	ips := []string{}
	for _, ip := range assigned {
		if slices.Contains(own, ip) {
			ips = append(ips, ip)
		}
	}
	return providers.SendingIPs{Dedicated: len(ips) > 0, IPs: ips}, nil
}
