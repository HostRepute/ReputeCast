package sparkpost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/received"
)

// Receiving mail (M58). SparkPost receives mail for an inbound domain
// whose MX points at its relay hosts and posts each message, as a batch of
// relay messages with the raw message, to the relay webhook matching the
// domain (https://developers.sparkpost.com/api/inbound-domains/,
// https://developers.sparkpost.com/api/relay-webhooks/). ReputeCast adds
// both. Relay webhooks authenticate with a token sent as it is in a header
// (they offer no basic authentication): the route's secret.

// InboundTokenHeader carries a relay webhook's auth_token.
const InboundTokenHeader = "X-MessageSystems-Webhook-Token"

// Relay hosts per region.
var (
	usRelayMX = []string{"rx1.sparkpostmail.com", "rx2.sparkpostmail.com", "rx3.sparkpostmail.com"}
	euRelayMX = []string{"rx1.eu.sparkpostmail.com", "rx2.eu.sparkpostmail.com", "rx3.eu.sparkpostmail.com"}
)

var (
	_ providers.InboundReceiver  = (*Provider)(nil)
	_ providers.InboundRegistrar = (*Provider)(nil)
	_ providers.WebhookSecrets   = (*Provider)(nil)
)

// WebhookSecretHeaders implements providers.WebhookSecrets: a relay
// webhook's token travels as it is and is never stored.
func (p *Provider) WebhookSecretHeaders() []string { return []string{InboundTokenHeader} }

func (p *Provider) eu() bool {
	return strings.EqualFold(p.s.Region, "eu") || strings.HasPrefix(p.api.BaseURL, EUBaseURL)
}

// InboundSetup implements providers.InboundReceiver.
func (p *Provider) InboundSetup(route providers.InboundRoute) providers.InboundSetup {
	hosts := usRelayMX
	if p.eu() {
		hosts = euRelayMX
	}
	setup := providers.InboundSetup{Automatic: true, Steps: []i18n.Text{mInboundEnterprise.Keep(i18n.Args{"domain": route.Domain})}}
	for _, h := range hosts {
		setup.MX = append(setup.MX, providers.DNSRecord{Purpose: "inbound", Type: "MX", Name: route.Domain, Value: "10 " + h})
	}
	return setup
}

func relayName(route providers.InboundRoute) string { return "ReputeCast " + route.ID.String() }

type relayWebhook struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	Target string `json:"target"`
	Token  string `json:"auth_token,omitempty"`
	Match  struct {
		Protocol string `json:"protocol,omitempty"`
		Domain   string `json:"domain"`
	} `json:"match"`
}

func (p *Provider) relayWebhooks(ctx context.Context) ([]relayWebhook, error) {
	var out struct {
		Results []relayWebhook `json:"results"`
	}
	if err := p.api.Do(ctx, "GET /api/v1/relay-webhooks", http.MethodGet, "/api/v1/relay-webhooks", nil, &out); err != nil {
		return nil, classify(err)
	}
	return out.Results, nil
}

func status(err error, codes ...int) bool {
	var ce *providers.CallError
	if !errors.As(err, &ce) {
		return false
	}
	for _, c := range codes {
		if ce.Status == c {
			return true
		}
	}
	return false
}

// RegisterInbound implements providers.InboundRegistrar: the inbound domain
// (one already there is fine) and a relay webhook for it posting to the
// hook with the route's token. ReputeCast's relay webhook for the route is
// updated when it is there.
func (p *Provider) RegisterInbound(ctx context.Context, route providers.InboundRoute, hookURL string) error {
	domain := strings.ToLower(route.Domain)
	err := p.api.Do(ctx, "POST /api/v1/inbound-domains", http.MethodPost, "/api/v1/inbound-domains", map[string]string{"domain": domain}, nil)
	if err != nil && !status(err, http.StatusConflict) {
		return classify(err)
	}
	hooks, err := p.relayWebhooks(ctx)
	if err != nil {
		return err
	}
	w := relayWebhook{Name: relayName(route), Target: hookURL, Token: route.Secret}
	w.Match.Protocol, w.Match.Domain = "SMTP", domain
	for _, h := range hooks {
		if h.Name == relayName(route) {
			return classify(p.api.Do(ctx, "PUT /api/v1/relay-webhooks/{id}", http.MethodPut, apiclient.Path("/api/v1/relay-webhooks/%s", h.ID), w, nil))
		}
	}
	return classify(p.api.Do(ctx, "POST /api/v1/relay-webhooks", http.MethodPost, "/api/v1/relay-webhooks", w, nil))
}

// UnregisterInbound implements providers.InboundRegistrar: ReputeCast's
// relay webhook for the route, then the inbound domain unless another
// relay webhook still uses it (SparkPost refuses with 409). What is gone
// already counts as removed.
func (p *Provider) UnregisterInbound(ctx context.Context, route providers.InboundRoute, _ string) error {
	hooks, err := p.relayWebhooks(ctx)
	if err != nil {
		return err
	}
	for _, h := range hooks {
		if h.Name != relayName(route) {
			continue
		}
		if err := p.api.Do(ctx, "DELETE /api/v1/relay-webhooks/{id}", http.MethodDelete, apiclient.Path("/api/v1/relay-webhooks/%s", h.ID), nil, nil); err != nil && !status(err, http.StatusNotFound) {
			return classify(err)
		}
	}
	err = p.api.Do(ctx, "DELETE /api/v1/inbound-domains/{domain}", http.MethodDelete, apiclient.Path("/api/v1/inbound-domains/%s", strings.ToLower(route.Domain)), nil, nil)
	if err != nil && !status(err, http.StatusNotFound, http.StatusConflict) {
		return classify(err)
	}
	return nil
}

// VerifyInbound implements providers.InboundReceiver: the relay webhook's
// token is the route's secret.
func (p *Provider) VerifyInbound(_ context.Context, route providers.InboundRoute, r *http.Request, _ []byte) error {
	return received.CheckToken(r.Header.Get(InboundTokenHeader), route)
}

type relayBatch []struct {
	Msys struct {
		RelayMessage *struct {
			Content struct {
				EmailRFC822         string `json:"email_rfc822"`
				EmailRFC822IsBase64 bool   `json:"email_rfc822_is_base64"`
			} `json:"content"`
			RcptTo string `json:"rcpt_to"`
		} `json:"relay_message"`
	} `json:"msys"`
}

// ParseInbound implements providers.InboundReceiver: every relay message of
// the batch, its raw message and its envelope recipient.
func (p *Provider) ParseInbound(_ context.Context, _ providers.InboundRoute, _ *http.Request, body []byte) ([]providers.InboundMessage, error) {
	var batch relayBatch
	if err := json.Unmarshal(body, &batch); err != nil {
		return nil, fmt.Errorf("sparkpost relay batch: %w", err)
	}
	out := []providers.InboundMessage{}
	for _, item := range batch {
		rm := item.Msys.RelayMessage
		if rm == nil || rm.Content.EmailRFC822 == "" {
			continue
		}
		raw := []byte(rm.Content.EmailRFC822)
		if rm.Content.EmailRFC822IsBase64 {
			var err error
			if raw, err = received.Base64(rm.Content.EmailRFC822); err != nil {
				return nil, fmt.Errorf("sparkpost relay message: %w", err)
			}
		}
		m := providers.InboundMessage{Raw: raw, Recipients: []string{}}
		if rcpt := strings.ToLower(strings.TrimSpace(rm.RcptTo)); rcpt != "" {
			m.Recipients = append(m.Recipients, rcpt)
		}
		out = append(out, m)
	}
	return out, nil
}
