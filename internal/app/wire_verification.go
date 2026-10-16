package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/verification"
)

// Address verification services (design expansion, M24): the workspace's
// own ZeroBounce, Kickbox… account, its uses and their results.

func (a *App) verification() *verification.Service {
	guard := netguard.Policy{AllowPrivate: a.Cfg != nil && a.Cfg.Providers.AllowPrivateNetworks}
	return &verification.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log,
		Audience: verificationAudience{a: a},
		Imports:  a.imports(),
		Notify:   verificationNotifier{a: a},
		Jobs:     func() *jobs.Client { return a.Jobs },
		// The services are fixed public hosts; calls still go through the
		// address guard like every provider call.
		HTTP: guard.Client(75 * time.Second),
	}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		verification.Register(h, a.verification())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		verification.RegisterJobs(a.JobReg, a.verification())
	})
	// Send Readiness's Audience quality uses the address checks.
	readinessVerification = func(a *App) readiness.VerificationChecker { return readinessAddressChecks{s: a.verification()} }
}

// verificationAudience resolves a job's target through the audience domain.
type verificationAudience struct{ a *App }

func (v verificationAudience) SelectionIDs(ctx context.Context, ws *store.Workspace, sel verification.Selection) ([]uuid.UUID, error) {
	aud := v.a.audience()
	all := func(f audience.ContactFilter) ([]uuid.UUID, error) {
		return aud.SelectionIDs(ctx, ws, audience.Selection{AllMatching: true, Filter: f})
	}
	switch {
	case sel.ListID != nil:
		return all(audience.ContactFilter{List: sel.ListID.String()})
	case sel.SegmentID != nil:
		return all(audience.ContactFilter{Segment: sel.SegmentID.String()})
	case len(sel.Tiers) > 0:
		seen := map[uuid.UUID]bool{}
		out := []uuid.UUID{}
		for _, t := range sel.Tiers {
			ids, err := all(audience.ContactFilter{Tier: t})
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				if !seen[id] {
					seen[id] = true
					out = append(out, id)
				}
			}
		}
		return out, nil
	case len(sel.ContactIDs) > 0:
		ids := make([]string, len(sel.ContactIDs))
		for i, id := range sel.ContactIDs {
			ids[i] = id.String()
		}
		return aud.SelectionIDs(ctx, ws, audience.Selection{IDs: ids})
	case len(sel.Filter) > 0:
		var f audience.ContactFilter
		if err := json.Unmarshal(sel.Filter, &f); err != nil {
			return nil, err
		}
		return all(f)
	}
	return []uuid.UUID{}, nil
}

// readinessAddressChecks gives Send Readiness the audience's address checks.
type readinessAddressChecks struct{ s *verification.Service }

func (r readinessAddressChecks) ReadinessFacts(ctx context.Context, ws *store.Workspace, _ uuid.UUID, recipients []readiness.Recipient, suppressed []string) readiness.VerificationFacts {
	rs := make([]verification.Recipient, len(recipients))
	for i, x := range recipients {
		rs[i] = verification.Recipient{Email: x.Email, Tier: x.Tier}
	}
	f, err := r.s.AudienceChecks(ctx, ws, rs, suppressed)
	if err != nil {
		return readiness.VerificationFacts{}
	}
	return readiness.VerificationFacts{Connected: f.Connected, Ran: true, Service: f.Service, ServiceName: f.ServiceName, Price: f.Price, Balance: f.Balance,
		AtRisk: f.AtRisk, AtRiskUnchecked: f.AtRiskUnchecked, Sunset: f.Sunset, SunsetUnchecked: f.SunsetUnchecked,
		AtRiskCheckedOn: f.AtRiskCheckedOn, SunsetCheckedOn: f.SunsetCheckedOn,
		Risky: f.Risky, RiskyReasons: f.RiskyReasons, RiskyEmails: f.RiskyEmails, RiskyCheckedOn: f.RiskyCheckedOn,
		InvalidSuppressed: f.InvalidSuppressed, InvalidCheckedOn: f.InvalidCheckedOn}
}

func (r readinessAddressChecks) RiskySegment(ctx context.Context, ws *store.Workspace, campaign string, emails []string) (uuid.UUID, error) {
	return r.s.RiskySegment(ctx, ws, campaign, emails)
}

// verificationNotifier tells the person who started a job when it ends,
// and members with integrations:manage when credits run low, each in their
// own language.
type verificationNotifier struct{ a *App }

func (n verificationNotifier) JobEnded(ctx context.Context, ws *store.Workspace, j store.VerificationJob) {
	if j.StartedBy == nil {
		return
	}
	l := applang.ForUserID(ctx, n.a.Q, *j.StartedBy)
	name := verification.ServiceName(j.Service)
	label := j.Label
	if j.Kind == "tier" {
		label = j.Label + " tier"
	}
	args := i18n.Args{"label": label, "service": name, "checked": int64(j.Checked), "total": int64(j.Total),
		"suppressed": int64(j.Suppressed), "risky": int64(j.Risky)}
	nt := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindVerificationEnded, Severity: "ok",
		Link: fmt.Sprintf("/w/%s/audience/verification/%s", ws.Slug, j.ID), Data: map[string]any{"job": j.ID.String()}}
	switch {
	case j.Status == "finished":
		nt.Title, nt.Body = verification.MJobFinishedTitle.Text(l, args), verification.MJobFinishedSummary.Text(l, args)
	case j.StopReason == "no_credits":
		nt.Severity, nt.Title, nt.Body = "warn", verification.MJobStoppedTitle.Text(l, args), verification.MJobStoppedCredits.Text(l, args)
	case j.StopReason == "service_down":
		nt.Severity, nt.Title, nt.Body = "bad", verification.MJobStoppedTitle.Text(l, args), verification.MJobStoppedDown.Text(l, args)
	case j.StopReason == "key_rejected":
		nt.Severity, nt.Title, nt.Body = "bad", verification.MJobStoppedTitle.Text(l, args), verification.MJobStoppedKey.Text(l, args)
	default:
		return // cancelled by a person: they know
	}
	nt.Summary = nt.Body
	if err := notify.ToUser(ctx, n.a.Pool, *j.StartedBy, nt); err != nil && n.a.Log != nil {
		n.a.Log.WarnContext(ctx, "notify verification job", "err", err)
	}
}

func (n verificationNotifier) CreditsLow(ctx context.Context, ws *store.Workspace, svc store.VerificationService, needed int64, label string) {
	ids, err := notify.MembersWith(ctx, n.a.Pool, ws.ID, authz.IntegrationsManage)
	if err != nil {
		return
	}
	name := verification.ServiceName(svc.Service)
	var balance int64
	if svc.Balance != nil {
		balance = *svc.Balance
	}
	for _, id := range ids {
		l := applang.ForUserID(ctx, n.a.Q, id)
		args := i18n.Args{"service": name, "balance": balance, "needed": needed, "label": label}
		body := verification.MCreditsLowSummary.Text(l, args)
		if needed > 0 {
			body = verification.MCreditsLowNeeded.Text(l, args)
		}
		_ = notify.ToUser(ctx, n.a.Pool, id, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindVerificationCreditsLow, Severity: "warn",
			Title: verification.MCreditsLowTitle.Text(l, args), Body: body, Summary: body,
			Link: fmt.Sprintf("/w/%s/settings/integrations/verification?service=%s", ws.Slug, svc.Service)})
	}
}
