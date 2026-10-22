package azureacs

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

var (
	_ providers.Verifier          = (*Provider)(nil)
	_ providers.WebhookVerifier   = (*Provider)(nil)
	_ providers.WebhookHandshaker = (*Provider)(nil)
	_ providers.TrackingManager   = (*Provider)(nil)
	_ providers.StatusPage        = (*Provider)(nil)
)

// Verify implements providers.Verifier without side effects: looking up an
// operation that doesn't exist answers 404 with valid credentials and 401
// or 403 otherwise. The sending limits are the ones Azure granted, entered
// at connect time.
func (p *Provider) Verify(ctx context.Context) (providers.Quota, error) {
	_, err := p.request(ctx, "GET /emails/operations/{id}", http.MethodGet, "/emails/operations/"+uuid.NewString()+"?api-version="+APIVersion, nil, nil)
	var ce *providers.CallError
	if err != nil && (!errors.As(err, &ce) || ce.Status != http.StatusNotFound) {
		return providers.Quota{}, err
	}
	// "Communication Services · reachable · 30 per minute · 100 per hour"
	var q providers.Quota
	account := i18n.Args{"minute": "no", "hour": "no"}
	if p.s.PerMinute > 0 {
		q.MaxRate = float64(p.s.PerMinute) / 60
		account["minute"], account["perMinute"] = "yes", strconv.FormatInt(int64(p.s.PerMinute), 10)
	}
	if p.s.PerHour > 0 {
		account["hour"], account["perHour"] = "yes", strconv.FormatInt(int64(p.s.PerHour), 10)
	}
	q.Account, q.AccountText = say(mAccount, account)
	return q, nil
}

// Tracking implements providers.TrackingManager. Engagement tracking is a
// domain setting in Azure, but ReputeCast sends every message with
// userEngagementTrackingDisabled, so ACS never tracks its mail.
func (p *Provider) Tracking(context.Context) (providers.Tracking, error) {
	var t providers.Tracking
	t.Scope, t.ScopeText = say(mTrackingScope, nil)
	t.Detail, t.DetailText = say(mTrackingDetail, nil)
	return t, nil
}

// DisableTracking implements providers.TrackingManager: nothing to change.
func (p *Provider) DisableTracking(context.Context) error { return nil }

// StatusPageURL implements providers.StatusPage.
func (p *Provider) StatusPageURL() (string, string) {
	return "Azure status", "https://azure.status.microsoft/en-us/status"
}
