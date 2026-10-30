package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// Why Request approval is off, in people's words rather than permission keys.
var (
	mReasonNeedsEditWho       = msgs.M("reason.needsEditWho")
	mReasonNobodyElseApproves = msgs.M("reason.nobodyElseApproves")
)

// ReviewAB summarises an A/B test ("Subject test · 2 variants · 20% sample").
type ReviewAB struct {
	Kind          string `json:"kind" enum:"subject,content,send_time"`
	Variants      int    `json:"variants"`
	TestPercent   int    `json:"testPercent"`
	DurationHours int    `json:"durationHours"`
	Label         string `json:"label"`
}

// ReviewCampaign is the campaign as Review & send shows it.
type ReviewCampaign struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Status        string     `json:"status" enum:"draft,approval,approved,scheduled,sending,paused,sent,cancelled,failed"`
	Subject       string     `json:"subject"`
	Preheader     string     `json:"preheader"`
	From          string     `json:"from"`
	ReplyTo       string     `json:"replyTo"`
	Version       int        `json:"version"`
	AutosavedAt   *time.Time `json:"autosavedAt,omitempty"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	UpdatedBy     string     `json:"updatedBy"`
	AudienceLabel string     `json:"audienceLabel"`
	ScheduledAt   *time.Time `json:"scheduledAt,omitempty"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	AB            *ReviewAB  `json:"ab,omitempty"`
	// Editable is false once the send started.
	Editable bool `json:"editable"`
}

// ReviewTestSend is a recent test email.
type ReviewTestSend struct {
	To       string    `json:"to"`
	At       time.Time `json:"at"`
	By       string    `json:"by"`
	Provider string    `json:"provider"`
	// Delivered names the mailbox provider that accepted it, when known ("Gmail").
	Delivered string `json:"delivered,omitempty"`
	// Languages are the language versions sent ("en", "es", "de").
	Languages []string `json:"languages,omitempty"`
}

// ReviewScheduled is who scheduled the campaign and when.
type ReviewScheduled struct {
	By string    `json:"by"`
	At time.Time `json:"at"`
}

// ReviewHeld is a scheduled send the send-time recheck held.
type ReviewHeld struct {
	Since  time.Time                  `json:"since"`
	Reason string                     `json:"reason"`
	Held   int64                      `json:"held"`
	Blocks []readiness.ReadinessCheck `json:"blocks"`
	// Holds are why the send-time recheck held the send (SR-3): blocks,
	// warnings or unknown checks nobody confirmed, the audience changed by
	// more than 10%, the content, sender or route edited after scheduling.
	Holds []ReviewHoldReason `json:"holds"`
	// Reviewed is the count the person confirmed when they scheduled it;
	// Rechecked the send-time recheck's (the count to confirm again).
	Reviewed  *ReviewCount `json:"reviewed,omitempty"`
	Rechecked *ReviewCount `json:"rechecked,omitempty"`
	// ExpiresAt is when the held send goes back to Draft if nobody
	// reviews it (readiness.HeldExpiry after its time).
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// ReviewHoldReason is one reason a send is held, in the reader's language.
type ReviewHoldReason struct {
	Kind string `json:"kind" enum:"blocked,new_warning,unknown,audience_change,edited"`
	Text string `json:"text"`
}

// ReviewCount is a recipient count and when it was counted.
type ReviewCount struct {
	At         time.Time `json:"at"`
	Recipients int64     `json:"recipients"`
}

// ReviewAccess is what the viewer may do.
type ReviewAccess struct {
	CanEdit             bool   `json:"canEdit"`
	CanSend             bool   `json:"canSend"`
	CanApprove          bool   `json:"canApprove"`
	CanManageRouting    bool   `json:"canManageRouting"`
	CanSendTestToOthers bool   `json:"canSendTestToOthers"`
	IsOwner             bool   `json:"isOwner"`
	HasPassword         bool   `json:"hasPassword"`
	Email               string `json:"email"`
	// SendHelpers and EditHelpers name up to two other members who can
	// send or change campaigns: the screen says who can, never a
	// permission key (X-1).
	SendHelpers []string `json:"sendHelpers"`
	EditHelpers []string `json:"editHelpers"`
}

// ReviewApproval is the approval side of Review & send.
type ReviewApproval struct {
	// Required is true when a rule matches this send by the viewer.
	Required  bool                `json:"required"`
	Rules     []ApprovalRuleMatch `json:"rules"`
	Needed    int                 `json:"needed"`
	Approvers []ApprovalPerson    `json:"approvers"`
	// Request is the open request, or the latest one when none is open.
	Request *ApprovalRequestView `json:"request,omitempty"`
	// Valid is true when an approved request covers the campaign as it is.
	Valid             bool     `json:"valid"`
	CanRequest        bool     `json:"canRequest"`
	RequestReason     string   `json:"requestReason,omitempty"`
	CanBypass         bool     `json:"canBypass"`
	MedianWaitMinutes *int     `json:"medianWaitMinutes,omitempty"`
	PolicyRules       []string `json:"policyRules"`
	// EmergencyBypass says the policy allows the Owners' emergency send;
	// Owners names the Owners other than the reader, who can use it
	// (the More menu: "Owners only. Ask Linh Nguyen.").
	EmergencyBypass bool     `json:"emergencyBypass"`
	Owners          []string `json:"owners"`
}

// CampaignReview is the Review & send screen.
type CampaignReview struct {
	Campaign ReviewCampaign               `json:"campaign"`
	Run      *readiness.ReadinessRun      `json:"run,omitempty"`
	Stale    bool                         `json:"stale"`
	Running  *readiness.ReadinessProgress `json:"running,omitempty"`
	// Failed is the last run when it stopped on an error.
	Failed *readiness.ReadinessRunFailure `json:"failed,omitempty"`
	// Fixes are the latest one-click fixes applied together, while Undo can
	// still put the campaign back as it was before them (R1.U3).
	Fixes *readiness.ReadinessFixBatch `json:"fixes,omitempty"`
	// SendNowWithin is how many minutes old the run may be for a send that
	// starts now (it is that send's check); older, the checks run again.
	SendNowWithin int                           `json:"sendNowWithin"`
	Thresholds    readiness.ReadinessThresholds `json:"thresholds"`
	// ThresholdsOwn is true when the campaign has thresholds of its own
	// instead of the workspace's defaults.
	ThresholdsOwn bool                        `json:"thresholdsOwn"`
	Schedule      readiness.ReadinessSchedule `json:"schedule"`
	TestSends     []ReviewTestSend            `json:"testSends"`
	Approval      ReviewApproval              `json:"approval"`
	Held          *ReviewHeld                 `json:"held,omitempty"`
	Scheduled     *ReviewScheduled            `json:"scheduled,omitempty"`
	Access        ReviewAccess                `json:"access"`
	// Typed says whether the sender types the recipient count to confirm,
	// and why: only unusual sends ask for it (SR-13).
	Typed ReviewTyped `json:"typed"`
	// PauseSlack is true when a webhook posts campaign.paused to Slack, so a
	// held send is announced there as well as by email.
	PauseSlack bool `json:"pauseSlack"`
	// Timezone is the workspace's.
	Timezone string `json:"timezone"`
	// Canary is the send's canary phase (C1): the choice, and what the
	// sending engine would do on its own.
	Canary ReviewCanary `json:"canary"`
}

// ReviewTyped is why a send asks its sender to type the recipient count
// ("31,192 is 2.1× your usual 15,000 for this list."); Needed is false for
// a routine send, confirmed with checkboxes.
type ReviewTyped struct {
	Needed  bool     `json:"needed"`
	Reasons []string `json:"reasons"`
}

// ReviewCanary is the canary phase of a send (C1, GS-10): the choice
// confirmed (auto when none was), and the sending engine's plan when it
// can tell: whether it starts with a canary on its own (Auto), why, and how
// many recipients the canary holds.
type ReviewCanary struct {
	Choice  string `json:"choice" enum:"auto,on,off"`
	Planned bool   `json:"planned"`
	Auto    bool   `json:"auto"`
	// Reasons are why the engine would run one by itself, with the numbers
	// behind them (the live monitor words them the same way).
	Reasons     []string `json:"reasons" enum:"new_domain,new_ip,never_mailed,over_usual"`
	Size        int64    `json:"size"`
	Seeds       int      `json:"seeds"`
	Domain      string   `json:"domain,omitempty"`
	NeverMailed int64    `json:"neverMailed"`
	Usual       int64    `json:"usual"`
	Audience    int64    `json:"audience"`
	// NewIPs are the dedicated sending IPs that never sent for the
	// workspace (new_ip, review GS-10).
	NewIPs []sending.CanaryIP `json:"newIps"`
}

func abSummary(l i18n.Lang, raw json.RawMessage) *ReviewAB {
	var d struct {
		Kind          string            `json:"kind"`
		Variants      []json.RawMessage `json:"variants"`
		TestPercent   int               `json:"test_percent"`
		DurationHours int               `json:"duration_hours"`
	}
	if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &d) != nil || d.Kind == "" {
		return nil
	}
	return &ReviewAB{Kind: d.Kind, Variants: len(d.Variants), TestPercent: d.TestPercent, DurationHours: d.DurationHours,
		Label: mABLabel.Text(l, i18n.Args{"kind": d.Kind, "variants": len(d.Variants), "sample": l.Region.Percent(float64(d.TestPercent), 0)})}
}

// Review builds the Review & send screen of a campaign. It re-checks the open
// approval first, so the screen never shows a stale approval.
func (s *Service) Review(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID) (CampaignReview, error) {
	if _, err := s.Revalidate(ctx, ws, campaignID); err != nil {
		return CampaignReview{}, err
	}
	c, err := s.Readiness.Campaign(ctx, ws, campaignID)
	if err != nil {
		return CampaignReview{}, err
	}
	out := CampaignReview{TestSends: []ReviewTestSend{}, Typed: ReviewTyped{Reasons: []string{}}, Timezone: ws.Timezone,
		Canary: ReviewCanary{Choice: sending.CanaryAuto, Reasons: []string{}, NewIPs: []sending.CanaryIP{}}}
	if conf, err := s.Readiness.Confirmed(ctx, campaignID); err == nil && conf != nil && conf.Canary != "" {
		out.Canary.Choice = conf.Canary
	}
	if cp := s.canaryPlanner(); cp != nil {
		if plan, err := cp.CanaryPlanFor(ctx, campaignID); err == nil {
			out.Canary.Planned, out.Canary.Auto, out.Canary.Size, out.Canary.Seeds = true, plan.Auto, plan.Size, plan.Seeds
			out.Canary.Domain, out.Canary.NeverMailed, out.Canary.Usual, out.Canary.Audience = plan.Domain, plan.NeverMailed, plan.Usual, plan.Audience
			if plan.Reasons != nil {
				out.Canary.Reasons = plan.Reasons
			}
			if plan.NewIPs != nil {
				out.Canary.NewIPs = plan.NewIPs
			}
		}
	}
	out.Campaign = ReviewCampaign{ID: c.ID.String(), Name: c.Name, Status: c.Status, Subject: c.Subject, Preheader: c.Preheader, From: fromLine(c.SenderName, c.SenderEmail),
		ReplyTo: c.ReplyTo, Version: int(c.Version), AutosavedAt: c.AutosavedAt, UpdatedAt: c.UpdatedAt, UpdatedBy: c.UpdatedByName, ScheduledAt: c.ScheduledAt,
		StartedAt: c.StartedAt, AB: abSummary(langOf(ctx), c.AbTest)}
	switch c.Status {
	case "draft", "approval", "approved", "scheduled":
		out.Campaign.Editable = true
	}
	out.Thresholds = readiness.ParseThresholds(c.Thresholds, ws.SendThresholds)
	// "Workspace defaults" or this send's own thresholds (SR-13).
	out.ThresholdsOwn = out.Thresholds != readiness.ParseThresholds(nil, ws.SendThresholds)
	out.Schedule = readiness.ScheduleView(c.Schedule, ws.Timezone)
	if out.Run, err = s.Readiness.Latest(ctx, c.ID); err != nil {
		return out, err
	}
	// Warnings dismissed in the composer, as they stand now (CO-14).
	if err = s.Readiness.Dismissed(ctx, ws, c.ID, out.Run); err != nil {
		return out, err
	}
	if out.Run != nil {
		// Only an unusual send asks for the count to be typed (SR-13).
		if reasons := s.typedReasons(ctx, c, *out.Run); len(reasons) > 0 {
			out.Typed = ReviewTyped{Needed: true, Reasons: reasons}
		}
		out.Stale = out.Run.Fingerprint != s.Readiness.CurrentFingerprint(ctx, ws, c)
		if len(out.Run.Estimates.Audience.Lists) > 0 {
			out.Campaign.AudienceLabel = strings.Join(out.Run.Estimates.Audience.Lists, " + ")
		}
	}
	if out.Running, out.Failed, err = s.Readiness.State(ctx, c.ID); err != nil {
		return out, err
	}
	if out.Fixes, err = s.Readiness.UndoableFixes(ctx, c); err != nil {
		return out, err
	}
	out.SendNowWithin = int(SendNowWithin / time.Minute)
	// Test sends from the audit log.
	if rows, err := s.Q.ReadinessTestSends(ctx, store.ReadinessTestSendsParams{WorkspaceID: &ws.ID, CampaignID: c.ID.String()}); err == nil {
		for _, r := range rows {
			var after struct {
				To        []string `json:"to"`
				Provider  string   `json:"provider"`
				Languages []string `json:"languages"`
			}
			var meta struct {
				Delivered map[string]string `json:"delivered"`
			}
			_ = json.Unmarshal(r.After, &after)
			_ = json.Unmarshal(r.Metadata, &meta)
			for _, to := range after.To {
				out.TestSends = append(out.TestSends, ReviewTestSend{To: to, At: r.OccurredAt, By: r.ActorName, Provider: after.Provider, Delivered: meta.Delivered[to],
					Languages: after.Languages})
			}
		}
		if len(out.TestSends) > 4 {
			out.TestSends = out.TestSends[:4]
		}
	}
	if slack, err := s.Q.ApprovalPauseAlertsSlack(ctx, ws.ID); err == nil {
		out.PauseSlack = slack
	}
	if c.Status == "scheduled" {
		if a, err := s.Q.ApprovalLastAudit(ctx, store.ApprovalLastAuditParams{WorkspaceID: &ws.ID, Action: "campaign.scheduled", TargetID: c.ID.String()}); err == nil {
			out.Scheduled = &ReviewScheduled{By: a.ActorName, At: a.OccurredAt}
		}
	}
	if h, err := s.Q.ApprovalOpenHold(ctx, c.ID); err == nil {
		out.Held = s.heldView(ctx, h)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	out.Access = ReviewAccess{CanEdit: p.Can(authz.CampaignsEdit), CanSend: p.Can(authz.CampaignsSend), CanApprove: p.Can(authz.CampaignsApprove),
		CanManageRouting: p.Can(authz.ProvidersManage), CanSendTestToOthers: p.Can(authz.CampaignsSend), IsOwner: p.RoleKey == authz.RoleOwner, Email: p.Email,
		SendHelpers: approverNames(ms, &p.UserID, authz.CampaignsSend), EditHelpers: approverNames(ms, &p.UserID, authz.CampaignsEdit)}
	if me := memberByID(ms, &p.UserID); me != nil {
		out.Access.HasPassword = me.HasPassword
	}
	out.Approval, err = s.reviewApproval(ctx, ws, p, c, out.Run, ms)
	// The checks read in the reader's language.
	out.Run = readiness.ForReader(ctx, out.Run)
	out.Running = readiness.ProgressForReader(ctx, out.Running)
	return out, err
}

func (s *Service) reviewApproval(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.ReadinessCampaignRow, run *readiness.ReadinessRun, ms []member) (ReviewApproval, error) {
	policy := ParsePolicy(ws.ApprovalPolicy)
	out := ReviewApproval{Rules: []ApprovalRuleMatch{}, Approvers: []ApprovalPerson{}, Needed: policy.ApprovalsRequired, PolicyRules: policy.RuleLabelsIn(langOf(ctx)),
		MedianWaitMinutes: s.medianWait(ctx, ws), EmergencyBypass: policy.EmergencyBypass, Owners: []string{}}
	for _, m := range ms {
		if m.RoleKey == authz.RoleOwner && m.ID != p.UserID {
			out.Owners = append(out.Owners, m.Name)
		}
	}
	for _, a := range approversOf(ms) {
		out.Approvers = append(out.Approvers, s.person(langOf(ctx), a))
	}
	me := memberByID(ms, &p.UserID)
	if me != nil && run != nil {
		facts, err := s.ruleFacts(ctx, ws, c, run, *me)
		if err != nil {
			return out, err
		}
		out.Rules = policy.Evaluate(facts)
		out.Required = len(out.Rules) > 0
	}
	var req *store.ApprovalRequest
	if open, err := s.open(ctx, c.ID); err != nil {
		return out, err
	} else if open != nil {
		req = open
	} else if last, err := s.Q.ApprovalLatestForCampaign(ctx, &c.ID); err == nil {
		r := store.ApprovalRequest{ID: last.ID, WorkspaceID: last.WorkspaceID, SubjectType: last.SubjectType, CampaignID: last.CampaignID, AutomationID: last.AutomationID,
			RequestedBy: last.RequestedBy, RequestedAt: last.RequestedAt, Note: last.Note, Status: last.Status, MatchedRules: last.MatchedRules, ApprovalsRequired: last.ApprovalsRequired,
			Snapshot: last.Snapshot, InvalidatedAt: last.InvalidatedAt, InvalidatedReason: last.InvalidatedReason, Changes: last.Changes, BypassReason: last.BypassReason,
			DecidedAt: last.DecidedAt, CampaignVersion: last.CampaignVersion, ReadinessRunID: last.ReadinessRunID, DecidedBy: last.DecidedBy, InvalidatedBy: last.InvalidatedBy,
			RemindedAt: last.RemindedAt, PreviousID: last.PreviousID, InvalidatedFrom: last.InvalidatedFrom, InvalidatedReasonText: last.InvalidatedReasonText}
		req = &r
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if req != nil {
		v, err := s.requestView(ctx, ws, *req, ms)
		if err != nil {
			return out, err
		}
		out.Request = &v
		out.Valid = req.Status == "approved"
		if req.Status == "pending" || req.Status == "approved" {
			// The open request's rules are what the screen shows.
			if snap := ParseSnapshot(req.Snapshot); len(snap.Rules) > 0 {
				out.Rules = snap.Rules
				out.Required = true
			}
			out.Needed = int(req.ApprovalsRequired)
		}
	}
	switch {
	case !p.Can(authz.CampaignsEdit):
		// Who can ask instead, by name (X-1).
		names := "none"
		if who := approverNames(ms, &p.UserID, authz.CampaignsEdit); len(who) > 0 {
			names = i18n.OrList(langOf(ctx).Locale, who)
		}
		out.RequestReason = say(ctx, mReasonNeedsEditWho, i18n.Args{"names": names})
	case !out.Required:
		out.RequestReason = say(ctx, mReasonNoRule, nil)
	case run == nil:
		out.RequestReason = say(ctx, mReasonRunFirst, nil)
	case run.Verdict == readiness.VerdictBlocked:
		out.RequestReason = say(ctx, mReasonFixBlocks, nil)
	case req != nil && req.Status == "pending":
		out.RequestReason = say(ctx, mReasonRequested, nil)
	case req != nil && req.Status == "approved":
		out.RequestReason = say(ctx, mReasonApproved, nil)
	default:
		others := 0
		for _, a := range approversOf(ms) {
			if a.ID != p.UserID {
				others++
			}
		}
		if others < policy.ApprovalsRequired {
			out.RequestReason = say(ctx, mReasonNobodyElseApproves, nil)
		} else {
			out.CanRequest = true
		}
	}
	out.CanBypass = out.Required && p.RoleKey == authz.RoleOwner && policy.EmergencyBypass && p.IsHuman()
	out.Rules = localRules(langOf(ctx), out.Rules)
	return out, nil
}
