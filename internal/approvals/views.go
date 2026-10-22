package approvals

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

// ApprovalDecisionView is one approver's decision.
type ApprovalDecisionView struct {
	Approver ApprovalPerson `json:"approver"`
	Decision string         `json:"decision" enum:"approve,request_changes,reject"`
	Comment  string         `json:"comment"`
	At       time.Time      `json:"at"`
}

// ApprovalSnapshotSummary is the gist of what a request covers.
type ApprovalSnapshotSummary struct {
	Version       int    `json:"version"`
	Subject       string `json:"subject"`
	AudienceCount int64  `json:"audienceCount"`
	// RouteLabel is "SES 90 / Postal 10"; RouteText "SES 43,121 · Postal 4,791".
	RouteLabel   string     `json:"routeLabel"`
	RouteText    string     `json:"routeText"`
	SplitLabel   string     `json:"splitLabel"`
	ScheduleText string     `json:"scheduleText"`
	ScheduleMode string     `json:"scheduleMode" enum:"now,at,best_time"`
	ScheduleAt   *time.Time `json:"scheduleAt,omitempty"`
	Timezone     string     `json:"timezone,omitempty"`
}

// ApprovalAuditLine is the audit entry behind a decision ("campaign.approved
// · Marcus Chen · 10:42 · from 198.51.100.40").
type ApprovalAuditLine struct {
	Action string    `json:"action"`
	Actor  string    `json:"actor"`
	At     time.Time `json:"at"`
	IP     string    `json:"ip,omitempty"`
}

// ApprovalRequestView is a request as Review & send and the approval page show it.
type ApprovalRequestView struct {
	ID          string                 `json:"id"`
	Status      string                 `json:"status" enum:"pending,approved,changes_requested,rejected,invalidated,bypassed,sent,cancelled"`
	SubjectType string                 `json:"subjectType" enum:"campaign,automation,transactional_template,website_post,website_page"`
	RequestedAt time.Time              `json:"requestedAt"`
	RequestedBy ApprovalPerson         `json:"requestedBy"`
	Note        string                 `json:"note"`
	Version     int                    `json:"version"`
	Given       int                    `json:"given"`
	Needed      int                    `json:"needed"`
	Decisions   []ApprovalDecisionView `json:"decisions"`
	DecidedBy   string                 `json:"decidedBy,omitempty"`
	DecidedAt   *time.Time             `json:"decidedAt,omitempty"`
	// Comment is the latest decision's comment.
	Comment       string     `json:"comment,omitempty"`
	InvalidatedAt *time.Time `json:"invalidatedAt,omitempty"`
	// InvalidatedReason says what changed ("subject changed", "audience
	// moved +14%"); screens add "after approval" when one was given.
	// InvalidatedFields is what it says changed, for a screen that writes its
	// own sentence around it ("edited the subject and a link"); empty when
	// the reason isn't about edited fields.
	InvalidatedReason string                  `json:"invalidatedReason,omitempty"`
	InvalidatedFields string                  `json:"invalidatedFields,omitempty"`
	InvalidatedBy     string                  `json:"invalidatedBy,omitempty"`
	Changes           []ApprovalChange        `json:"changes"`
	Snapshot          ApprovalSnapshotSummary `json:"snapshot"`
	Rules             []ApprovalRuleMatch     `json:"rules"`
	Audit             *ApprovalAuditLine      `json:"audit,omitempty"`
	BypassReason      string                  `json:"bypassReason,omitempty"`
	FollowsID         string                  `json:"followsId,omitempty"`
}

func routeText(l i18n.Lang, shares []SnapshotShare) string {
	parts := []string{}
	for _, s := range shares {
		if s.Recipients > 0 {
			parts = append(parts, s.Short+" "+count(l, s.Recipients))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += " · " + p
	}
	return out
}

func routeLabel(l i18n.Lang, shares []SnapshotShare) string {
	out := ""
	for i, s := range shares {
		if s.Recipients == 0 && len(shares) > 1 {
			continue
		}
		if i > 0 && out != "" {
			out += " / "
		}
		out += s.Short + " " + decimal(l, s.Percent)
	}
	return out
}

func summaryOf(l i18n.Lang, snap Snapshot, wsTZ string) ApprovalSnapshotSummary {
	sv := readiness.ScheduleView(snap.Schedule, wsTZ)
	return ApprovalSnapshotSummary{Version: snap.Version, Subject: snap.Subject, AudienceCount: snap.AudienceCount, RouteLabel: routeLabel(l, snap.Route), RouteText: routeText(l, snap.Route),
		SplitLabel: snap.SplitLabel, ScheduleText: scheduleTextIn(l, snap.Schedule, state.Location(wsTZ)), ScheduleMode: sv.Mode, ScheduleAt: sv.At, Timezone: sv.Timezone}
}

// requestView builds the view of a request.
func (s *Service) requestView(ctx context.Context, ws *store.Workspace, r store.ApprovalRequest, ms []member) (ApprovalRequestView, error) {
	snap := ParseSnapshot(r.Snapshot)
	l := langOf(ctx)
	v := ApprovalRequestView{ID: r.ID.String(), Status: r.Status, SubjectType: r.SubjectType, RequestedAt: r.RequestedAt, Note: r.Note, Version: snap.Version,
		Needed: int(r.ApprovalsRequired), DecidedAt: r.DecidedAt, InvalidatedAt: r.InvalidatedAt, InvalidatedReason: reasonFor(l, r.InvalidatedReason, r.InvalidatedReasonText),
		InvalidatedFields: fieldsFor(l, r.InvalidatedReason, r.InvalidatedReasonText), Changes: []ApprovalChange{},
		Decisions: []ApprovalDecisionView{}, Snapshot: summaryOf(langOf(ctx), snap, ws.Timezone), Rules: localRules(langOf(ctx), snap.Rules), BypassReason: r.BypassReason}
	if r.CampaignVersion != nil {
		v.Version = int(*r.CampaignVersion)
	}
	if r.PreviousID != nil {
		v.FollowsID = r.PreviousID.String()
	}
	if m := memberByID(ms, r.RequestedBy); m != nil {
		v.RequestedBy = s.person(langOf(ctx), *m)
	} else if r.RequestedBy != nil {
		if u, err := s.Q.GetUser(ctx, *r.RequestedBy); err == nil {
			v.RequestedBy = ApprovalPerson{ID: u.ID.String(), Name: u.Name, Email: u.Email, Initials: Initials(u.Name), Color: u.AvatarColor}
		}
	}
	if m := memberByID(ms, r.DecidedBy); m != nil {
		v.DecidedBy = m.Name
	}
	if m := memberByID(ms, r.InvalidatedBy); m != nil {
		v.InvalidatedBy = m.Name
	}
	v.Changes = changesFor(l, r.Changes)
	decisions, err := s.Q.ApprovalDecisions(ctx, []uuid.UUID{r.ID})
	if err != nil {
		return v, err
	}
	for _, d := range decisions {
		dv := ApprovalDecisionView{Decision: d.Decision, Comment: d.Comment, At: d.CreatedAt, Approver: ApprovalPerson{Name: d.ApproverName, Initials: Initials(d.ApproverName), Color: d.ApproverColor}}
		if d.ApproverID != nil {
			dv.Approver.ID = d.ApproverID.String()
		}
		v.Decisions = append(v.Decisions, dv)
		if d.Decision == DecisionApprove {
			v.Given++
		}
		if d.Comment != "" {
			v.Comment = d.Comment
		}
	}
	if r.Status == "approved" || r.Status == "invalidated" {
		action := "campaign.approved"
		target := ""
		if r.CampaignID != nil {
			target = r.CampaignID.String()
		}
		if a, err := s.Q.ApprovalLastAudit(ctx, store.ApprovalLastAuditParams{WorkspaceID: &ws.ID, Action: action, TargetID: target}); err == nil {
			line := &ApprovalAuditLine{Action: action, Actor: a.ActorName, At: a.OccurredAt}
			if a.Ip != nil && a.Ip.IsValid() {
				line.IP = a.Ip.String()
			}
			v.Audit = line
		}
	}
	return v, nil
}

// medianWait is the median minutes to a first decision over the last 30 days.
func (s *Service) medianWait(ctx context.Context, ws *store.Workspace) *int {
	rows, err := s.Q.ApprovalDecisionWaits(ctx, store.ApprovalDecisionWaitsParams{WorkspaceID: ws.ID, Since: s.now().AddDate(0, 0, -30)})
	if err != nil || len(rows) == 0 {
		return nil
	}
	vals := make([]int64, 0, len(rows))
	for _, r := range rows {
		if r >= 0 {
			vals = append(vals, r)
		}
	}
	if len(vals) == 0 {
		return nil
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
	m := vals[len(vals)/2]
	if len(vals)%2 == 0 {
		m = (vals[len(vals)/2-1] + vals[len(vals)/2]) / 2
	}
	minutes := int(m / 60)
	return &minutes
}
