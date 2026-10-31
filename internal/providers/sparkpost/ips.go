package sparkpost

import (
	"context"
	"net/http"
	"strings"

	"github.com/hostrepute/reputecast/internal/providers"
)

var _ providers.IPReporter = (*Provider)(nil)

// defaultPool is the IP pool a transmission without options.ip_pool goes
// out through.
const defaultPool = "default"

// SendingIPs implements providers.IPReporter with GET /api/v1/sending-ips,
// the account's dedicated sending IPs with their IP pool, returned whole,
// without pages. Transmissions go out through the connection's IP pool (or
// the default pool), so only its IPs count; an account without dedicated
// IPs has none and sends from SparkPost's shared IPs.
func (p *Provider) SendingIPs(ctx context.Context) (providers.SendingIPs, error) {
	var out struct {
		Results []struct {
			ExternalIP string `json:"external_ip"`
			IPPool     string `json:"ip_pool"`
		} `json:"results"`
	}
	if err := p.api.Do(ctx, "GET /api/v1/sending-ips", http.MethodGet, "/api/v1/sending-ips", nil, &out); err != nil {
		return providers.SendingIPs{}, classify(err)
	}
	pool := strings.TrimSpace(p.s.IPPool)
	if pool == "" {
		pool = defaultPool
	}
	ips := []string{}
	for _, r := range out.Results {
		if r.IPPool == pool {
			ips = append(ips, r.ExternalIP)
		}
	}
	ips, _ = providers.CleanIPs(ips)
	return providers.SendingIPs{Dedicated: len(ips) > 0, IPs: ips}, nil
}
