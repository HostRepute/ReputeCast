package transactional

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Setup tab: the API (base URL, limits, errors), the keys that can
// send, and the SMTP relay as the instance runs it.

// TransactionalLimits are the API's limits.
type TransactionalLimits struct {
	MaxBatch int `json:"maxBatch"`
	// RatePerSecond is a key's default rate.
	RatePerSecond int `json:"ratePerSecond"`
	// RouteLimitBytes is the largest message the route's providers accept
	// (attachments included); RouteLimitProvider is the smallest's name.
	RouteLimitBytes    int64  `json:"routeLimitBytes"`
	RouteLimitProvider string `json:"routeLimitProvider,omitempty"`
	// RouteLimitOthers are the route's providers that accept more ("SES
	// allows 40 MB"), in route order.
	RouteLimitOthers []TransactionalProviderLimit `json:"routeLimitOthers"`
	IdempotencyHours int                          `json:"idempotencyHours"`
	ScheduleHours    int                          `json:"scheduleHours"`
	SMTPRecipients   int                          `json:"smtpRecipients"`
}

// TransactionalProviderLimit is the largest message one provider accepts.
type TransactionalProviderLimit struct {
	Provider string `json:"provider" example:"SES"`
	Bytes    int64  `json:"bytes"`
}

// TransactionalErrorCode is one row of the error table.
type TransactionalErrorCode struct {
	Code   string `json:"code"`
	Status int    `json:"status"`
}

// TransactionalRelayCheck is one status check of the relay.
type TransactionalRelayCheck struct {
	Key    string `json:"key" enum:"dns,starttls,tls,certificate"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// TransactionalRelay is the SMTP relay as the instance runs it.
type TransactionalRelay struct {
	Enabled      bool   `json:"enabled"`
	Hostname     string `json:"hostname,omitempty"`
	StartTLSPort int    `json:"starttlsPort"`
	TLSPort      int    `json:"tlsPort"`
	// State: off, ok, unreachable, certificate (expired or failing) or
	// unchecked.
	State            string                    `json:"state" enum:"off,ok,unreachable,certificate,unchecked"`
	UnreachableSince *time.Time                `json:"unreachableSince,omitempty"`
	CertNotAfter     *time.Time                `json:"certNotAfter,omitempty"`
	Checks           []TransactionalRelayCheck `json:"checks"`
	EnabledBy        string                    `json:"enabledBy,omitempty"`
	EnabledAt        *time.Time                `json:"enabledAt,omitempty"`
	// Admins are the instance admins who can turn it on.
	Admins []string `json:"admins"`
	// CanManage: the caller is an instance admin.
	CanManage bool `json:"canManage"`
}

// TransactionalSetup is the Setup tab.
type TransactionalSetup struct {
	// BaseURL is the API's transactional base; OpenAPIURL the instance's
	// OpenAPI document.
	BaseURL    string                    `json:"baseUrl"`
	OpenAPIURL string                    `json:"openapiUrl"`
	Limits     TransactionalLimits       `json:"limits"`
	Errors     []TransactionalErrorCode  `json:"errors"`
	Keys       []TransactionalSendingKey `json:"keys"`
	// Senders are the verified Transactional senders: the API's from and
	// the SMTP relay's allowed From addresses.
	Senders []TransactionalSenderOption `json:"senders"`
	Relay   TransactionalRelay          `json:"relay"`
	// CanManageKeys: creating keys needs apikeys:manage.
	CanManageKeys bool     `json:"canManageKeys"`
	KeyHelpers    []string `json:"keyHelpers"`
	// Headers are the relay's optional headers.
	Headers []string `json:"headers"`
}

var errorTable = []TransactionalErrorCode{
	{CodeTemplateNotFound, 422}, {CodeTemplateNotPublished, 422}, {CodeMissingVariables, 422}, {CodeInvalidRecipient, 422},
	{CodeRecipientSuppressed, 422}, {CodeSenderNotVerified, 422}, {CodeAttachmentsTooLarge, 422}, {CodeRateLimited, 429},
	{CodeNoHealthyRoute, 503}, {CodeIdempotencyReused, 409}, {CodeTemplateNotAllowed, 403},
}

// Setup returns the Setup tab.
func (s *Service) Setup(ctx context.Context, ws *store.Workspace, p *authz.Principal) (TransactionalSetup, error) {
	base := s.URLs.Base()
	out := TransactionalSetup{BaseURL: base + "/api/v1/workspaces/" + ws.Slug + "/transactional", OpenAPIURL: base + "/api/openapi.json",
		Limits: TransactionalLimits{MaxBatch: MaxBatch, RatePerSecond: 50, IdempotencyHours: int(IdempotencyTTL / time.Hour), ScheduleHours: int(MaxScheduleAge / time.Hour),
			SMTPRecipients: MaxRawRecipients, RouteLimitOthers: []TransactionalProviderLimit{}}, Errors: errorTable, Headers: []string{"X-ReputeCast-Tags", "X-ReputeCast-Category"}, KeyHelpers: []string{}}
	keys, err := s.sendingKeys(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Keys = keys
	senders, err := s.transactionalSenders(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	out.Senders = []TransactionalSenderOption{}
	for _, sn := range senders {
		if sn.Verified {
			out.Senders = append(out.Senders, sn)
		}
	}
	// The route's size limit: the smallest of its providers', and the ones
	// that accept more.
	if set, err := routing.Load(ctx, s.Q, ws.ID); err == nil {
		provs, err := s.Q.ProvListActive(ctx, ws.ID)
		if err != nil {
			return out, err
		}
		var route []store.Provider
		seen := map[string]bool{}
		for _, r := range set.StreamRules(routing.StreamTransactional) {
			ids := []string{}
			for _, sp := range r.Splits {
				ids = append(ids, sp.ProviderID.String())
			}
			if r.Fallback != nil {
				ids = append(ids, r.Fallback.String())
			}
			for _, id := range ids {
				for _, pr := range provs {
					if pr.ID.String() == id && !seen[id] {
						seen[id] = true
						route = append(route, pr)
					}
				}
			}
		}
		for _, pr := range route {
			if l := providerLimit(pr.Adapter); out.Limits.RouteLimitProvider == "" || l < out.Limits.RouteLimitBytes {
				out.Limits.RouteLimitBytes, out.Limits.RouteLimitProvider = l, state.Short(pr.Name, pr.Adapter)
			}
		}
		for _, pr := range route {
			if l := providerLimit(pr.Adapter); l > out.Limits.RouteLimitBytes {
				out.Limits.RouteLimitOthers = append(out.Limits.RouteLimitOthers, TransactionalProviderLimit{Provider: state.Short(pr.Name, pr.Adapter), Bytes: l})
			}
		}
	}
	if p != nil {
		out.CanManageKeys = p.Can(authz.APIKeysManage)
		out.Relay.CanManage = p.IsInstanceAdmin
	}
	if !out.CanManageKeys {
		out.KeyHelpers = s.helpers(ctx, ws.ID, authz.APIKeysManage, 3)
	}
	relay, err := s.relayInfo(ctx)
	if err != nil {
		return out, err
	}
	relay.CanManage = out.Relay.CanManage
	out.Relay = relay
	return out, nil
}

func (s *Service) relayInfo(ctx context.Context) (TransactionalRelay, error) {
	row, err := s.Q.RelaySettings(ctx)
	if err != nil {
		return TransactionalRelay{}, err
	}
	out := TransactionalRelay{Enabled: row.Enabled, Hostname: row.Hostname, StartTLSPort: int(row.StarttlsPort), TLSPort: int(row.TlsPort), State: "off",
		UnreachableSince: row.UnreachableSince, CertNotAfter: row.CertNotAfter, Checks: []TransactionalRelayCheck{}, EnabledBy: row.EnabledByName, EnabledAt: row.EnabledAt,
		Admins: []string{}}
	var st struct {
		CheckedAt *time.Time                `json:"checkedAt"`
		Checks    []TransactionalRelayCheck `json:"checks"`
	}
	_ = json.Unmarshal(row.Status, &st)
	if st.Checks != nil {
		out.Checks = st.Checks
	}
	if row.Enabled {
		out.State = "unchecked"
		if st.CheckedAt != nil {
			out.State = "ok"
			for _, c := range st.Checks {
				switch {
				case (c.Key == "starttls" || c.Key == "tls") && !c.OK:
					out.State = "unreachable"
				case c.Key == "certificate" && !c.OK && out.State == "ok":
					out.State = "certificate"
				}
			}
		}
		if row.CertNotAfter != nil && !row.CertNotAfter.After(s.now()) && out.State != "unreachable" {
			out.State = "certificate"
		}
	}
	admins, err := s.Q.InstanceAdminAdminNames(ctx)
	if err != nil {
		return out, err
	}
	for _, a := range admins {
		out.Admins = append(out.Admins, a.Name)
	}
	return out, nil
}
