package sendgrid

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/hostrepute/reputecast/internal/providers"
)

var _ providers.IPReporter = (*Provider)(nil)

// SendingIPs implements providers.IPReporter with GET /v3/ips/assigned
// ("Retrieve all assigned IPs"), which returns the IPs assigned to the
// account whole, without pages. With an IP pool set, mail goes out through
// that pool's IPs only, so only those count. An account on SendGrid's
// shared IPs has none assigned. A key without access to IP Management is
// refused with what to change, so the IPs ReputeCast knows stay.
func (p *Provider) SendingIPs(ctx context.Context) (providers.SendingIPs, error) {
	var assigned []struct {
		IP    string   `json:"ip"`
		Pools []string `json:"pools"`
	}
	if err := p.api.Do(ctx, "GET /v3/ips/assigned", http.MethodGet, "/v3/ips/assigned", nil, &assigned); err != nil {
		if restricted(err) {
			var ce *providers.CallError
			if errors.As(err, &ce) {
				ce.Message, ce.Text = say(mIPsRestricted, nil)
			}
		}
		return providers.SendingIPs{}, err
	}
	ips := []string{}
	for _, a := range assigned {
		if p.s.IPPool == "" || slices.Contains(a.Pools, p.s.IPPool) {
			ips = append(ips, a.IP)
		}
	}
	ips, _ = providers.CleanIPs(ips)
	return providers.SendingIPs{Dedicated: len(ips) > 0, IPs: ips}, nil
}
