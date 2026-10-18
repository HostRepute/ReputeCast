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
func (b *builder) checkedDay(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.In(b.loc).Format("2 Jan")
}

// reasonWords names risky reasons the way the board lists them.
func reasonWords(reasons map[string]int64) string {
	order := []string{"catch_all", "disposable", "role"}
	words := map[string]string{"catch_all": "catch-all", "disposable": "disposable", "role": "role"}
	parts := []string{}
	for _, k := range order {
		if n := reasons[k]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s %s", Count(n), words[k]))
		}
	}
	other := []string{}
	for k, n := range reasons {
		if !slices.Contains(order, k) && n > 0 {
			other = append(other, fmt.Sprintf("%s %s", Count(n), strings.ReplaceAll(k, "_", " ")))
		}
	}
	slices.Sort(other)
	parts = append(parts, other...)
	if len(parts) == 0 {
		return ""
	}
	return JoinAnd(parts) + " addresses"
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
		c := ReadinessCheck{Key: "audience.verify_inactive", Group: GroupAudience, Status: Warn, Label: "Address checks"}
		switch {
		case v.AtRiskUnchecked > 0 && v.SunsetUnchecked > 0:
			c.Title = fmt.Sprintf("%s At risk and Sunset contacts haven't been checked in the last 90 days", Count(unchecked))
		case v.AtRiskUnchecked > 0:
			c.Title = fmt.Sprintf("%s At risk contacts haven't been checked in the last 90 days", Count(v.AtRiskUnchecked))
		default:
			c.Title = fmt.Sprintf("%s Sunset contacts haven't been checked in the last 90 days", Count(v.SunsetUnchecked))
		}
		c.FlatTitle = fmt.Sprintf("Audience: %s inactive contacts not checked in 90 days", Count(unchecked))
		detail := "Old, quiet addresses are where spam traps and dead mailboxes hide."
		switch {
		case v.SunsetUnchecked == 0 && v.Sunset > 0 && v.SunsetCheckedOn != nil:
			detail = fmt.Sprintf("The %s Sunset contacts were checked on %s. %s", Count(v.Sunset), b.checkedDay(v.SunsetCheckedOn), detail)
		case v.AtRiskUnchecked == 0 && v.AtRisk > 0 && v.AtRiskCheckedOn != nil:
			detail = fmt.Sprintf("The %s At risk contacts were checked on %s. %s", Count(v.AtRisk), b.checkedDay(v.AtRiskCheckedOn), detail)
		}
		c.Detail = detail
		c.Ack = fmt.Sprintf("I understand %s inactive contacts haven't been checked.", Count(unchecked))
		if v.Connected && v.Service != "" {
			params := map[string]string{"count": fmt.Sprint(unchecked), "service": v.Service, "serviceName": v.ServiceName,
				"cost": fmt.Sprintf("%.2f", float64(unchecked)*v.Price)}
			note := fmt.Sprintf("%s, about %s", v.ServiceName, money2(float64(unchecked)*v.Price))
			if v.Balance != nil {
				params["balance"] = fmt.Sprint(*v.Balance)
				note += fmt.Sprintf(" · balance %s credits", Count(*v.Balance))
			}
			params["note"] = note
			c.Fixes = []ReadinessFix{apply(FixVerifyAddresses, fmt.Sprintf("Verify %s addresses first", Count(unchecked)), "primary", params)}
		} else {
			c.Fixes = []ReadinessFix{link("Connect a verification service", b.href("/settings/integrations/verification"))}
		}
		b.add(c)
	}
	if v.Risky > 0 {
		share := float64(v.Risky) / final * 100
		c := ReadinessCheck{Key: "audience.risky", Group: GroupAudience, Status: Pass, Label: "Risky addresses"}
		words := reasonWords(v.RiskyReasons)
		on := ""
		if v.RiskyCheckedOn != nil {
			on = " by the " + b.checkedDay(v.RiskyCheckedOn) + " check"
		}
		if share >= VerifyRiskyShare {
			c.Status = Warn
			c.Title = fmt.Sprintf("%s risky addresses, %s of the audience", Count(v.Risky), Pct1(share))
			c.FlatTitle = fmt.Sprintf("Audience: %s risky addresses", Count(v.Risky))
			if words != "" {
				c.Detail = fmt.Sprintf("%s, tagged%s. Sending to them is your call.", upperFirst(words), on)
			} else {
				c.Detail = fmt.Sprintf("Tagged%s. Sending to them is your call.", on)
			}
			c.Ack = fmt.Sprintf("I understand %s risky addresses are included.", Count(v.Risky))
			c.Fixes = []ReadinessFix{
				apply(FixExcludeRisky, fmt.Sprintf("Exclude %s risky addresses", Count(v.Risky)), "", map[string]string{"count": fmt.Sprint(v.Risky)}),
				link("See them", b.href("/audience?check=risky")),
			}
		} else {
			c.Title = fmt.Sprintf("%s risky addresses, %s of the audience", Count(v.Risky), Pct1(share))
			c.Detail = fmt.Sprintf("Under 2%%, tagged%s.", on)
			c.Short = fmt.Sprintf("%s risky", Count(v.Risky))
		}
		b.add(c)
	}
	if v.InvalidSuppressed > 0 {
		on := "after a check"
		if v.InvalidCheckedOn != nil {
			on = "after the " + b.checkedDay(v.InvalidCheckedOn) + " check"
		}
		b.add(ReadinessCheck{Key: "audience.invalid", Group: GroupAudience, Status: Pass, Label: "Invalid addresses",
			Title:  fmt.Sprintf("%s invalid %s already suppressed", Count(v.InvalidSuppressed), Plural(v.InvalidSuppressed, "address is", "addresses are")),
			Detail: fmt.Sprintf("Suppressed for all mail %s; they are left out of the %s at send time", on, Count(a.Base)),
			Short:  fmt.Sprintf("%s invalid suppressed", Count(v.InvalidSuppressed))})
	}
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
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
		return "", api.Conflict("Address checks aren't available.")
	}
	aud, recipients, err := s.computeAudience(ctx, s.Q, ws.ID, ParseAudience(c.Audience), sunsetDays(ws.SunsetPolicy), s.now())
	if err != nil {
		return "", err
	}
	vf := s.Verification.ReadinessFacts(ctx, ws, c.ID, recipients, aud.SuppressedEmails)
	if len(vf.RiskyEmails) == 0 {
		return "No risky addresses to exclude", nil
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
		return "", api.Invalid("body.fixes", "Pick an audience first.")
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
	return fmt.Sprintf("Excluded %s risky addresses", Count(int64(n))), nil
}
