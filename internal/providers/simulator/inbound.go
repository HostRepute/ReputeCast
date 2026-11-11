package simulator

import (
	"context"
	"crypto/hmac"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/hostrepute/reputecast/internal/providers"
)

// The demo transport receives mail too: tests and the demo post a raw
// message to the inbound hook, signed with the route's secret. Nothing is
// registered anywhere, so registering and removing do nothing.

// Inbound request headers.
const (
	// InboundSignatureHeader carries SignInbound of the body.
	InboundSignatureHeader = "X-Simulator-Signature"
	// InboundRecipientsHeader lists the envelope recipients, comma
	// separated (optional).
	InboundRecipientsHeader = "X-Simulator-Recipients"
)

// InboundMX is the MX host a simulated route names.
const InboundMX = "inbound.simulator.invalid"

var (
	_ providers.InboundReceiver  = (*Provider)(nil)
	_ providers.InboundRegistrar = (*Provider)(nil)
)

// SignInbound returns the X-Simulator-Signature of an inbound body: the
// lowercase hex HMAC-SHA256 of the body keyed with the route's secret, as
// the simulator's webhooks are signed.
func SignInbound(secret string, body []byte) string { return Sign(secret, body) }

// InboundSetup implements providers.InboundReceiver.
func (p *Provider) InboundSetup(route providers.InboundRoute) providers.InboundSetup {
	return providers.InboundSetup{
		Automatic: true,
		MX:        []providers.DNSRecord{{Purpose: "inbound", Type: "MX", Name: route.Domain, Value: "10 " + InboundMX}},
	}
}

// VerifyInbound implements providers.InboundReceiver: the signature must be
// SignInbound of the body with the route's secret.
func (p *Provider) VerifyInbound(_ context.Context, route providers.InboundRoute, r *http.Request, body []byte) error {
	got, err := hex.DecodeString(strings.TrimSpace(r.Header.Get(InboundSignatureHeader)))
	if err != nil || len(got) == 0 || route.Secret == "" {
		return fmt.Errorf("%w: the request isn't signed with the route's secret", providers.ErrBadSignature)
	}
	want, _ := hex.DecodeString(SignInbound(route.Secret, body))
	if !hmac.Equal(got, want) {
		return fmt.Errorf("%w: the request isn't signed with the route's secret", providers.ErrBadSignature)
	}
	return nil
}

// ParseInbound implements providers.InboundReceiver: the body is the raw
// message, and X-Simulator-Recipients its envelope recipients.
func (p *Provider) ParseInbound(_ context.Context, _ providers.InboundRoute, r *http.Request, body []byte) ([]providers.InboundMessage, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("simulator inbound: the request has no message")
	}
	recipients := []string{}
	for _, a := range strings.Split(r.Header.Get(InboundRecipientsHeader), ",") {
		if a = strings.ToLower(strings.TrimSpace(a)); a != "" {
			recipients = append(recipients, a)
		}
	}
	return []providers.InboundMessage{{Raw: append([]byte(nil), body...), Recipients: recipients, ReceivedAt: p.now()}}, nil
}

// RegisterInbound implements providers.InboundRegistrar: there is nothing
// to point anywhere.
func (p *Provider) RegisterInbound(context.Context, providers.InboundRoute, string) error { return nil }

// UnregisterInbound implements providers.InboundRegistrar.
func (p *Provider) UnregisterInbound(context.Context, providers.InboundRoute, string) error {
	return nil
}
