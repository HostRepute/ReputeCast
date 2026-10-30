package ses

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"

	"github.com/hostrepute/reputecast/internal/providers"
)

var _ providers.IPReporter = (*Provider)(nil)

// SendingIPs implements providers.IPReporter with GetDedicatedIps, every
// page of it: the dedicated IPs of the account in the connection's region,
// standard and managed pools alike. An account without any sends from
// SES's shared pool. Access that can't call it (a role or keys from before
// ReputeCast asked for ses:GetDedicatedIps) is an error that says what to
// allow, so the IPs ReputeCast knows stay.
func (p *Provider) SendingIPs(ctx context.Context) (providers.SendingIPs, error) {
	out := providers.SendingIPs{IPs: []string{}}
	pages := sesv2.NewGetDedicatedIpsPaginator(p.client, &sesv2.GetDedicatedIpsInput{})
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			ce := callError("GetDedicatedIps", err)
			// Keys AWS doesn't know answer UnrecognizedClientException; a
			// missing permission is AccessDenied.
			if ce.Code == "AccessDeniedException" || ce.Code == "AccessDenied" {
				ce.Message, ce.Text = say(mDedicatedIPsDenied, nil)
			}
			return providers.SendingIPs{}, ce
		}
		for _, ip := range page.DedicatedIps {
			if ip.Ip != nil && *ip.Ip != "" {
				out.IPs = append(out.IPs, *ip.Ip)
			}
		}
	}
	out.IPs, _ = providers.CleanIPs(out.IPs)
	out.Dedicated = len(out.IPs) > 0
	return out, nil
}
