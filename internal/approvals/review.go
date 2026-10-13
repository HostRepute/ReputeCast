package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
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
}

// ReviewAccess is what the viewer may do.
type ReviewAccess struct {
	CanEdit             bool     `json:"canEdit"`
	CanSend             bool     `json:"canSend"`
	CanApprove          bool     `json:"canApprove"`
	CanManageRouting    bool     `json:"canManageRouting"`
	CanSendTestToOthers bool     `json:"canSendTestToOthers"`
	IsOwner             bool     `json:"isOwner"`
	HasPassword         bool     `json:"hasPassword"`
	Email               string   `json:"email"`
	SendHelpers         []string `json:"sendHelpers"`
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
}

// CampaignReview is the Review & send screen.
type CampaignReview struct {
	Campaign   ReviewCampaign                `json:"campaign"`
	Run        *readiness.ReadinessRun       `json:"run,omitempty"`
	Stale      bool                          `json:"stale"`
	Running    *readiness.ReadinessProgress  `json:"running,omitempty"`
	Thresholds readiness.ReadinessThresholds `json:"thresholds"`
	Schedule   readiness.ReadinessSchedule   `json:"schedule"`
	TestSends  []ReviewTestSend              `json:"testSends"`
	Approval   ReviewApproval                `json:"approval"`
	Held       *ReviewHeld                   `json:"held,omitempty"`
	Scheduled  *ReviewScheduled              `json:"scheduled,omitempty"`
	Access     ReviewAccess                  `json:"access"`
	// LargeSend is the audience size from which sending asks to type the count.
	LargeSend int64 `json:"largeSend"`
	// Timezone is the workspace's.
	Timezone string `json:"timezone"`
}

func abSummary(raw json.RawMessage) *ReviewAB {
	var d struct {
		Kind          string            `json:"kind"`
		Variants      []json.RawMessage `json:"variants"`
		TestPercent   int               `json:"test_percent"`
		DurationHours int               `json:"duration_hours"`
	}
	if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &d) != nil || d.Kind == "" {
		return nil
	}
	word := map[string]string{"subject": "Subject test", "content": "Content test", "send_time": "Send-time test"}[d.Kind]
	return &ReviewAB{Kind: d.Kind, Variants: len(d.Variants), TestPercent: d.TestPercent, DurationHours: d.DurationHours,
		Label: fmt.Sprintf("%s · %d variants · %d%% sample", word, len(d.Variants), d.TestPercent)}
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
	out := CampaignReview{TestSends: []ReviewTestSend{}, LargeSend: LargeSend, Timezone: ws.Timezone}
	out.Campaign = ReviewCampaign{ID: c.ID.String(), Name: c.Name, Status: c.Status, Subject: c.Subject, Preheader: c.Preheader, From: fromLine(c.SenderName, c.SenderEmail),
		ReplyTo: c.ReplyTo, Version: int(c.Version), AutosavedAt: c.AutosavedAt, UpdatedAt: c.UpdatedAt, UpdatedBy: c.UpdatedByName, ScheduledAt: c.ScheduledAt,
		StartedAt: c.StartedAt, AB: abSummary(c.AbTest)}
	switch c.Status {
	case "draft", "approval", "approved", "scheduled":
		out.Campaign.Editable = true
	}
	out.Thresholds = readiness.ParseThresholds(c.Thresholds, ws.SendThresholds)
	out.Schedule = readiness.ScheduleView(c.Schedule, ws.Timezone)
	if out.Run, err = s.Readiness.Latest(ctx, c.ID); err != nil {
		return out, err
	}
	if out.Run != nil {
		out.Stale = out.Run.Fingerprint != s.Readiness.CurrentFingerprint(ctx, ws, c)
		if len(out.Run.Estimates.Audience.Lists) > 0 {
			out.Campaign.AudienceLabel = strings.Join(out.Run.Estimates.Audience.Lists, " + ")
		}
	}
	if out.Running, err = s.Readiness.Progress(ctx, c.ID); err != nil {
		return out, err
	}
	// Test sends from the audit log.
	if rows, err := s.Q.ReadinessTestSends(ctx, store.ReadinessTestSendsParams{WorkspaceID: &ws.ID, CampaignID: c.ID.String()}); err == nil {
		for _, r := range rows {
			var after struct {
				To       []string `json:"to"`
				Provider string   `json:"provider"`
			}
			var meta struct {
				Delivered map[string]string `json:"delivered"`
			}
			_ = json.Unmarshal(r.After, &after)
			_ = json.Unmarshal(r.Metadata, &meta)
			for _, to := range after.To {
				out.TestSends = append(out.TestSends, ReviewTestSend{To: to, At: r.OccurredAt, By: r.ActorName, Provider: after.Provider, Delivered: meta.Delivered[to]})
			}
		}
		if len(out.TestSends) > 4 {
			out.TestSends = out.TestSends[:4]
		}
	}
	// Who scheduled it.
	if c.Status == "scheduled" {
		if a, err := s.Q.ApprovalLastAudit(ctx, store.ApprovalLastAuditParams{WorkspaceID: &ws.ID, Action: "campaign.scheduled", TargetID: c.ID.String()}); err == nil {
			out.Scheduled = &ReviewScheduled{By: a.ActorName, At: a.OccurredAt}
		}
	}
	// A held send.
	if h, err := s.Q.ApprovalOpenHold(ctx, c.ID); err == nil {
		var d HoldDetail
		_ = json.Unmarshal(h.Detail, &d)
		if d.Blocks == nil {
			d.Blocks = []readiness.ReadinessCheck{}
		}
		out.Held = &ReviewHeld{Since: h.TrippedAt, Reason: d.Reason, Held: h.Held, Blocks: d.Blocks}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	// Access.
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	out.Access = ReviewAccess{CanEdit: p.Can(authz.CampaignsEdit), CanSend: p.Can(authz.CampaignsSend), CanApprove: p.Can(authz.CampaignsApprove),
		CanManageRouting: p.Can(authz.ProvidersManage), CanSendTestToOthers: p.Can(authz.CampaignsSend), IsOwner: p.RoleKey == authz.RoleOwner, Email: p.Email,
		SendHelpers: approverNames(ms, &p.UserID, authz.CampaignsSend)}
	if me := memberByID(ms, &p.UserID); me != nil {
		out.Access.HasPassword = me.HasPassword
	}
	out.Approval, err = s.reviewApproval(ctx, ws, p, c, out.Run, ms)
	return out, err
}

func (s *Service) reviewApproval(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.ReadinessCampaignRow, run *readiness.ReadinessRun, ms []member) (ReviewApproval, error) {
	policy := ParsePolicy(ws.ApprovalPolicy)
	out := ReviewApproval{Rules: []ApprovalRuleMatch{}, Approvers: []ApprovalPerson{}, Needed: policy.ApprovalsRequired, PolicyRules: policy.RuleLabels(),
		MedianWaitMinutes: s.medianWait(ctx, ws)}
	for _, a := range approversOf(ms) {
		out.Approvers = append(out.Approvers, s.person(a))
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
	// The open request, or the latest one.
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
			RemindedAt: last.RemindedAt, PreviousID: last.PreviousID, InvalidatedFrom: last.InvalidatedFrom}
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
	// Can the viewer ask for approval?
	switch {
	case !p.Can(authz.CampaignsEdit):
		out.RequestReason = "Asking for approval needs the campaigns:edit permission."
	case !out.Required:
		out.RequestReason = "No approval rule matches this send."
	case run == nil:
		out.RequestReason = "Run Send Readiness first."
	case run.Verdict == readiness.VerdictBlocked:
		out.RequestReason = "Fix the blocking issues before asking for approval."
	case req != nil && req.Status == "pending":
		out.RequestReason = "Approval is already requested."
	case req != nil && req.Status == "approved":
		out.RequestReason = "This version is approved."
	default:
		others := 0
		for _, a := range approversOf(ms) {
			if a.ID != p.UserID {
				others++
			}
		}
		if others < policy.ApprovalsRequired {
			out.RequestReason = "Nobody else holds campaigns:approve. An Owner can give it to another role."
		} else {
			out.CanRequest = true
		}
	}
	out.CanBypass = out.Required && p.RoleKey == authz.RoleOwner && policy.EmergencyBypass && p.IsHuman()
	return out, nil
}
