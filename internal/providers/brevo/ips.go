package brevo

import (
	"context"
	"net/http"

	"github.com/hostrepute/reputecast/internal/providers"
)

var _ providers.IPReporter = (*Provider)(nil)

// SendingIPs implements providers.IPReporter with GET /v3/senders/ips, the
// account's dedicated IPs, returned whole, without pages. An IP Brevo lists
// as inactive carries no mail, so it doesn't count; an account without
// dedicated IPs sends from Brevo's shared IPs.
func (p *Provider) SendingIPs(ctx context.Context) (providers.SendingIPs, error) {
	var out struct {
		IPs []struct {
			IP     string `json:"ip"`
			Active bool   `json:"active"`
		} `json:"ips"`
	}
	if err := p.api.Do(ctx, "GET /v3/senders/ips", http.MethodGet, "/v3/senders/ips", nil, &out); err != nil {
		return providers.SendingIPs{}, classify(err)
	}
	ips := []string{}
	for _, x := range out.IPs {
		if x.Active {
			ips = append(ips, x.IP)
		}
	}
	ips, _ = providers.CleanIPs(ips)
	return providers.SendingIPs{Dedicated: len(ips) > 0, IPs: ips}, nil
}
