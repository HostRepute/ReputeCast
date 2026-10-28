package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Address checks in Audience quality (design expansion, X14): invalid
// addresses a verification service found are already suppressed (a passing
// line counts them), risky ones warn from 2% of the audience, and inactive
// contacts (At risk and Sunset) nobody checked in 90 days warn from 10%.

// Fix actions of the address checks.
const (
	// FixExcludeRisky leaves the audience's risky addresses out of the send.
	FixExcludeRisky = "exclude_risky"
	// FixVerifyAddresses opens the Verify addresses dialog on Review & send;
	// the dialog starts the job, so the fixes endpoint never applies it.
	FixVerifyAddresses = "verify_addresses"
)

// Thresholds of the address checks.
const (
	VerifyInactiveShare = 10.0 // % of the audience At risk or Sunset and unchecked in 90 days
	VerifyRiskyShare    = 2.0  // % of the audience risky
)

// VerificationChecker gives the address checks' facts for a run (the
// verification domain, wired in internal/app) and makes the exclusion.
type VerificationChecker interface {
	ReadinessFacts(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, recipients []Recipient, suppressed []string) VerificationFacts
	// RiskySegment makes or refreshes a static segment of these risky
	// addresses' contacts, for the campaign to exclude.
	RiskySegment(ctx context.Context, ws *store.Workspace, campaign string, emails []string) (uuid.UUID, error)
}

// VerificationFacts are the audience's address-check results.
type VerificationFacts struct {
	// Connected is true when a verification service is connected; Ran
	// when the checker looked at the audience.
	Connected bool
	Ran       bool
	// The service a check would use, its price per check (dollars) and balance.
	Service     string
	ServiceName string
	Price       float64
	Balance     *int64
	// At risk and Sunset recipients, and those not checked in 90 days.
	AtRisk, AtRiskUnchecked int64
	Sunset, SunsetUnchecked int64
	// When the checked ones were last checked.
	AtRiskCheckedOn, SunsetCheckedOn *time.Time
	// Risky recipients by reason (catch_all, disposable, role…), their
	// addresses (for the exclusion) and when they were checked.
	Risky          int64
	RiskyReasons   map[string]int64
	RiskyEmails    []string
	RiskyCheckedOn *time.Time
	// InvalidSuppressed counts the audience's contacts a check found invalid,
	// already suppressed and left out at send time.
	InvalidSuppressed int64
	InvalidCheckedOn  *time.Time
}

// checkedDay is a check's day as the board writes it ("14 Oct").
func (b *builder) checkedDay(t *time.Time) map[string]any {
	if t == nil {
		return nil
	}
	return ArgDayMonth(t.In(b.loc))
}

// reasonWords names risky reasons the way the board lists them: "600
// catch-all, 200 disposable and 50 role addresses".
func reasonWords(reasons map[string]int64) Text {
	order := []string{"catch_all", "disposable", "role"}
	parts := []any{}
	for _, k := range order {
		if n := reasons[k]; n > 0 {
			parts = append(parts, tx("verification.reason."+k, A{"count": n}))
		}
	}
	other := []string{}
	for k, n := range reasons {
		if !slices.Contains(order, k) && n > 0 {
			other = append(other, k)
		}
	}
	slices.Sort(other)
	for _, k := range other {
		parts = append(parts, tx("verification.reasonOther", A{"count": reasons[k], "reason": strings.ReplaceAll(k, "_", " ")}))
	}
	if len(parts) == 0 {
		return nil
	}
	return tx("verification.reasons", A{"reasons": argCapital(ArgAnd(parts))})
}

func (b *builder) verification() {
	v := b.f.Verification
	a := b.f.Audience
	if !v.Ran || a.Empty || a.Final == 0 {
		return
	}
	final := float64(a.Final)
	unchecked := v.AtRiskUnchecked + v.SunsetUnchecked
	if unchecked > 0 && float64(unchecked)/final*100 >= VerifyInactiveShare {
		c := check{Key: "audience.verify_inactive", Group: GroupAudience, Status: Warn, Label: tx("label.addressChecks", nil)}
		switch {
		case v.AtRiskUnchecked > 0 && v.SunsetUnchecked > 0:
			c.Title = tx("verification.inactiveBoth", A{"count": unchecked})
		case v.AtRiskUnchecked > 0:
			c.Title = tx("verification.inactiveAtRisk", A{"count": v.AtRiskUnchecked})
		default:
			c.Title = tx("verification.inactiveSunset", A{"count": v.SunsetUnchecked})
		}
		c.FlatTitle = tx("verification.inactiveFlat", A{"count": unchecked})
		detail := tx("verification.inactiveDetail", nil)
		switch {
		case v.SunsetUnchecked == 0 && v.Sunset > 0 && v.SunsetCheckedOn != nil:
			detail = sentences(tx("verification.sunsetChecked", A{"count": v.Sunset, "day": b.checkedDay(v.SunsetCheckedOn)}), detail)
		case v.AtRiskUnchecked == 0 && v.AtRisk > 0 && v.AtRiskCheckedOn != nil:
			detail = sentences(tx("verification.atRiskChecked", A{"count": v.AtRisk, "day": b.checkedDay(v.AtRiskCheckedOn)}), detail)
		}
		c.Detail = detail
		c.Ack = tx("verification.inactiveAck", A{"count": unchecked})
		if v.Connected && v.Service != "" {
			params := map[string]string{"count": fmt.Sprint(unchecked), "service": v.Service, "serviceName": v.ServiceName,
				"cost": fmt.Sprintf("%.2f", float64(unchecked)*v.Price)}
			note := fmt.Sprintf("%s, about %s", v.ServiceName, money2(float64(unchecked)*v.Price))
			if v.Balance != nil {
				params["balance"] = fmt.Sprint(*v.Balance)
				note += fmt.Sprintf(" · balance %s credits", Count(*v.Balance))
			}
			params["note"] = note
			c.Fixes = []fix{apply(FixVerifyAddresses, tx("fix.verifyFirst", A{"count": unchecked}), "primary", params)}
		} else {
			c.Fixes = []fix{link(tx("fix.connectVerification", nil), b.href("/settings/integrations/verification"))}
		}
		b.add(c)
	}
	if v.Risky > 0 {
		share := float64(v.Risky) / final * 100
		c := check{Key: "audience.risky", Group: GroupAudience, Status: Pass, Label: tx("label.riskyAddresses", nil)}
		words := reasonWords(v.RiskyReasons)
		var on any = ""
		if v.RiskyCheckedOn != nil {
			on = tx("verification.byCheck", A{"day": b.checkedDay(v.RiskyCheckedOn)})
		}
		args := A{"reasons": words, "listed": yes(!words.Empty()), "on": on, "dated": yes(v.RiskyCheckedOn != nil)}
		c.Title = tx("verification.risky", A{"count": v.Risky, "share": ArgPercent1(share)})
		if share >= VerifyRiskyShare {
			c.Status = Warn
			c.FlatTitle = tx("verification.riskyFlat", A{"count": v.Risky})
			c.Detail = tx("verification.riskyDetail", args)
			c.Ack = tx("verification.riskyAck", A{"count": v.Risky})
			c.Fixes = []fix{
				apply(FixExcludeRisky, tx("fix.excludeRisky", A{"count": v.Risky}), "", map[string]string{"count": fmt.Sprint(v.Risky)}),
				link(tx("fix.seeThem", nil), b.href("/audience?check=risky")),
			}
		} else {
			c.Detail = tx("verification.riskyFew", A{"limit": ArgPercent(2), "on": on, "dated": yes(v.RiskyCheckedOn != nil)})
			c.Short = tx("verification.riskyShort", A{"count": v.Risky})
		}
		b.add(c)
	}
	if v.InvalidSuppressed > 0 {
		var on Text
		if v.InvalidCheckedOn != nil {
			on = tx("verification.afterTheCheck", A{"day": b.checkedDay(v.InvalidCheckedOn)})
		} else {
			on = tx("verification.afterACheck", nil)
		}
		b.add(check{Key: "audience.invalid", Group: GroupAudience, Status: Pass, Label: tx("label.invalidAddresses", nil),
			Title:  tx("verification.invalid", A{"count": v.InvalidSuppressed}),
			Detail: tx("verification.invalidDetail", A{"after": on, "base": a.Base}),
			Short:  tx("verification.invalidShort", A{"count": v.InvalidSuppressed})})
	}
}

// money2 writes dollars with cents ("$42.42").
func money2(v float64) string {
	cents := int64(math.Round(v * 100))
	return fmt.Sprintf("$%s.%02d", Count(cents/100), cents%100)
}

// excludeRisky leaves the audience's risky addresses out: a static segment
// of them joins the campaign's excluded segments.
func (s *Service) excludeRisky(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.ReadinessCampaignRow) (string, error) {
	if s.Verification == nil {
		return "", api.Conflict(say(ctx, "problem.noAddressChecks", nil))
	}
	aud, recipients, err := s.computeAudience(ctx, s.Q, ws.ID, ParseAudience(c.Audience), sunsetDays(ws.SunsetPolicy), s.now(), true)
	if err != nil {
		return "", err
	}
	vf := s.Verification.ReadinessFacts(ctx, ws, c.ID, recipients, aud.SuppressedEmails)
	if len(vf.RiskyEmails) == 0 {
		return say(ctx, "applied.noRisky", nil), nil
	}
	seg, err := s.Verification.RiskySegment(ctx, ws, c.Name, vf.RiskyEmails)
	if err != nil {
		return "", err
	}
	var raw map[string]any
	if err := json.Unmarshal(c.Audience, &raw); err != nil || raw == nil {
		raw = map[string]any{}
	}
	def := ParseAudience(c.Audience)
	if def.Empty() {
		return "", api.Invalid("body.fixes", say(ctx, "problem.pickAudience", nil))
	}
	if !slices.Contains(def.ExcludeSegments, seg.String()) {
		raw["exclude_segments"] = append(slices.Clone(def.ExcludeSegments), seg.String())
	}
	next, _ := json.Marshal(raw)
	n := len(vf.RiskyEmails)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReadinessSetAudience(ctx, store.ReadinessSetAudienceParams{Audience: next, AudienceCount: c.AudienceCount, UpdatedBy: &p.UserID, Now: s.now(), WorkspaceID: ws.ID, ID: c.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.update", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"audience": c.Audience}, After: map[string]any{"audience": json.RawMessage(next)},
			Metadata: map[string]any{"source": "send_readiness", "fix": FixExcludeRisky, "count": n}})
	})
	if err != nil {
		return "", err
	}
	return say(ctx, "applied.excludedRisky", A{"count": n}), nil
}
