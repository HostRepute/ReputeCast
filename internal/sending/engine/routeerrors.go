package engine

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
)

// Route-level answers (review GS-4, PR-7 engine part). A provider that
// throttles the account, an exhausted sending quota and a sender or domain
// the provider hasn't verified are answers about the route, not about one
// message: retrying each message until it fails would drop mail that the
// route could still send later, or elsewhere. Such a batch goes back to the
// queue without using up its attempts, and the route is held through the
// outage path, with its options (retry with a probe, move, cancel the
// share):
//
//   - quota: the provider says the account's quota is used up: held at once.
//   - identity: the provider refuses the sender or its domain: held at once.
//   - throttled: the provider keeps throttling for three minutes, the route
//     sending at half its rate meanwhile: held.
//   - api_errors: the share of failed calls over three minutes passes the
//     send's limit (thresholds.api_errors), with enough evidence: held.

// Evidence before the API error threshold trips: at least this many calls in
// the three-minute window and this many failures.
const (
	apiErrorMinCalls  = 20
	apiErrorMinEvents = 3
	apiErrorWindow    = 3 * time.Minute
)

// throttle slowdown: a throttled route sends at half its rate (down to a
// quarter) until it has been answered normally for a minute.
const (
	throttleFloor = 0.25
	throttleCalm  = time.Minute
)

// quotaWords are what providers say when the account's quota is used up
// (SES "Daily message quota exceeded", Resend "daily_quota_exceeded",
// SendGrid "Maximum credits exceeded", Brevo and Mailgun "sending limit").
var quotaWords = []string{"quota", "credits exceeded", "sending limit", "daily limit", "monthly limit", "message limit", "limit reached"}

// quotaExhausted reports whether a failed call or message says the
// account's sending quota is used up.
func quotaExhausted(err error) bool {
	var ce *providers.CallError
	if errors.As(err, &ce) {
		return quotaText(ce.Code + " " + ce.Message)
	}
	return err != nil && quotaText(err.Error())
}

func quotaText(s string) bool {
	m := strings.ToLower(s)
	if strings.Contains(m, "rate") && !strings.Contains(m, "quota") {
		return false // "Maximum sending rate exceeded" is throttling
	}
	for _, w := range quotaWords {
		if strings.Contains(m, w) {
			return true
		}
	}
	return false
}

// sendErrQuota is quotaExhausted for one message's answer.
func sendErrQuota(se *providers.SendError) bool {
	return se != nil && quotaText(se.Code+" "+se.Message)
}

// throttledErr reports whether a failed call is the provider pacing the
// account (not its quota).
func throttledErr(err error) bool {
	var ce *providers.CallError
	return errors.As(err, &ce) && ce.Throttled && !quotaExhausted(err)
}

// identityWords are what providers answer when the From address or its
// domain isn't verified with them: SES "Email address is not verified",
// Postmark "not a Sender Signature", SendGrid "does not match a verified
// Sender Identity", Resend "domain is not verified", Mailgun "domain not
// found", Brevo "sender ... not valid".
var identityWords = []string{
	"not verified", "unverified", "sender signature", "verified sender identity", "domain is not verified", "domain not verified",
	"domain not found", "unknown domain", "sender is not valid", "sender not valid", "from address is not allowed", "not authorized to send from",
}

// identityError reports whether a provider refused the sender or its
// domain rather than a recipient.
func identityError(err error) bool {
	if err == nil {
		return false
	}
	var ce *providers.CallError
	if errors.As(err, &ce) {
		return identityText(ce.Code + " " + ce.Message)
	}
	return identityText(err.Error())
}

func identityText(s string) bool {
	m := strings.ToLower(s)
	for _, w := range identityWords {
		if strings.Contains(m, w) {
			return true
		}
	}
	return false
}

// sendErrIdentity is identityError for one message's answer.
func sendErrIdentity(se *providers.SendError) bool {
	return se != nil && identityText(se.Code+" "+se.Message)
}

// routeHold is why a route must be held after a call, "" when it needn't.
func routeHold(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, providers.ErrUnauthorized):
		return "auth"
	case identityError(err):
		return "identity"
	case quotaExhausted(err):
		return "quota"
	}
	return ""
}

// throttled slows a route down after the provider throttled it.
func (r *runRoute) throttled(now time.Time) {
	if r.slow == 0 {
		r.slow = 1
	}
	r.slow = math.Max(throttleFloor, r.slow/2)
	r.calmFrom = time.Time{}
	r.cooldown = now.Add(backoffMin)
}

// answered speeds a slowed route up again once it has been answered
// normally for a while.
func (r *runRoute) answered(now time.Time) {
	if r.slow == 0 || r.slow >= 1 {
		return
	}
	if r.calmFrom.IsZero() {
		r.calmFrom = now
		return
	}
	if now.Sub(r.calmFrom) >= throttleCalm {
		r.slow = math.Min(1, r.slow*2)
		r.calmFrom = now
	}
}

// slowFactor is the share of its rate a route sends at now.
func (r *runRoute) slowFactor() float64 {
	if r.slow <= 0 || r.slow > 1 {
		return 1
	}
	return r.slow
}

// apiErrorTrip reports whether a route's share of failed calls over the last
// three minutes is over the send's limit, with enough evidence, and the
// rate.
func (e *Engine) apiErrorTrip(ctx context.Context, rs *runState, r *runRoute, now time.Time) (bool, float64, error) {
	limit := rs.thresholds.APIErrors
	if limit <= 0 {
		return false, 0, nil
	}
	w, err := e.Q.SendingCallWindow(ctx, store.SendingCallWindowParams{ProviderID: r.provider.ID, Since: now.Add(-apiErrorWindow), Until: now})
	if err != nil {
		return false, 0, err
	}
	if w.Calls < apiErrorMinCalls {
		return false, 0, nil
	}
	return over(w.Errors, w.Calls, limit, apiErrorMinEvents), rate(w.Errors, w.Calls), nil
}
