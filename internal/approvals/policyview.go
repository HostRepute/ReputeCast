package approvals

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// ApprovalPolicyMatches says what each rule matches today.
type ApprovalPolicyMatches struct {
	// Warming is "Postal (mta.acme.io) is warming up, day 9 of 21." or empty.
	Warming string `json:"warming"`
	// LacksSend names the members who can edit but not send.
	LacksSend []string `json:"lacksSend"`
	// LacksSendRoles names their roles ("Content Writers").
	LacksSendRoles []string `json:"lacksSendRoles"`
	// ApprovalLists names the lists marked "Requires approval".
	ApprovalLists []string `json:"approvalLists"`
}

// ApprovalPendingItem is a pending request on the policy screen.
type ApprovalPendingItem struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Author      ApprovalPerson `json:"author"`
	Audience    int64          `json:"audience"`
	RequestedAt time.Time      `json:"requestedAt"`
}

// ApprovalPolicyView is Settings › Approvals.
type ApprovalPolicyView struct {
	Policy    ApprovalPolicy        `json:"policy"`
	Approvers []ApprovalPerson      `json:"approvers"`
	Matches   ApprovalPolicyMatches `json:"matches"`
	// BypassCount is how often the emergency bypass was used in 90 days.
	BypassCount int64                 `json:"bypassCount"`
	Pending     []ApprovalPendingItem `json:"pending"`
	CanManage   bool                  `json:"canManage"`
	Reason      string                `json:"reason,omitempty"`
	// ClientSignoff describes the client sign-off rule; set in an agency's
	// client workspaces, where the rule is offered (clientsignoff.go).
	ClientSignoff *ApprovalClientSignoffInfo `json:"clientSignoff,omitempty"`
}

// PolicyView returns the approval policy with what it matches today.
func (s *Service) PolicyView(ctx context.Context, ws *store.Workspace, p *authz.Principal) (ApprovalPolicyView, error) {
	out := ApprovalPolicyView{Policy: ParsePolicy(ws.ApprovalPolicy), Approvers: []ApprovalPerson{}, Pending: []ApprovalPendingItem{},
		Matches: ApprovalPolicyMatches{LacksSend: []string{}, LacksSendRoles: []string{}, ApprovalLists: []string{}}, CanManage: p.Can(authz.WorkspaceManage)}
	if !out.CanManage {
		out.Reason = authz.DeniedMessage(authz.WorkspaceManage, access.Helpers(ctx, s.Q, ws.ID, authz.WorkspaceManage))
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	if out.ClientSignoff, err = s.signoffInfo(ctx, ws, ms); err != nil {
		return out, err
	}
	roles := map[string]bool{}
	for _, m := range ms {
		if m.Perms.Has(authz.CampaignsApprove) {
			out.Approvers = append(out.Approvers, s.person(m))
		}
		if m.Perms.Has(authz.CampaignsEdit) && !m.Perms.Has(authz.CampaignsSend) {
			out.Matches.LacksSend = append(out.Matches.LacksSend, m.Name)
			if !roles[m.RoleName] {
				roles[m.RoleName] = true
				out.Matches.LacksSendRoles = append(out.Matches.LacksSendRoles, m.RoleName)
			}
		}
	}
	// Providers warming up or connected recently.
	if provs, err := s.Q.ProvListActive(ctx, ws.ID); err == nil {
		loc := state.Location(ws.Timezone)
		plans, _ := s.Q.WarmupListPlans(ctx, ws.ID)
		parts := []string{}
		for _, pr := range provs {
			label := state.Short(pr.Name, pr.Adapter)
			if pr.Label != "" && pr.Adapter != "ses" {
				host := pr.Label
				if i := strings.LastIndex(host, "· "); i >= 0 {
					host = strings.TrimSpace(host[i+len("· "):])
				}
				label += " (" + host + ")"
			}
			warming := false
			for _, w := range plans {
				if w.ProviderID == pr.ID && w.Status != "done" {
					day := min(max(state.WarmupDay(w.StartDate, s.now(), loc), 1), int(w.Days))
					parts = append(parts, fmt.Sprintf("%s is warming up, day %d of %d", label, day, w.Days))
					warming = true
					break
				}
			}
			if !warming && s.now().Sub(pr.ConnectedAt) < time.Duration(out.Policy.NewProviderDays)*24*time.Hour {
				parts = append(parts, fmt.Sprintf("%s was connected %d days ago", label, int(s.now().Sub(pr.ConnectedAt).Hours()/24)))
			}
		}
		if len(parts) > 0 {
			out.Matches.Warming = strings.Join(parts, "; ") + "."
		}
	}
	if lists, err := s.Q.ApprovalAudienceLists(ctx, ws.ID); err == nil {
		for _, l := range lists {
			if l.RequiresApproval {
				out.Matches.ApprovalLists = append(out.Matches.ApprovalLists, l.Name)
			}
		}
	}
	if out.BypassCount, err = s.Q.ApprovalBypassCount(ctx, store.ApprovalBypassCountParams{WorkspaceID: ws.ID, Since: s.now().AddDate(0, 0, -90)}); err != nil {
		return out, err
	}
	pending, err := s.Q.ApprovalPendingForWorkspace(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range pending {
		out.Pending = append(out.Pending, ApprovalPendingItem{ID: r.ID.String(), Name: r.SubjectName, Audience: r.AudienceCount, RequestedAt: r.RequestedAt,
			Author: ApprovalPerson{Name: r.RequestedByName, Email: r.RequestedByEmail, Initials: Initials(r.RequestedByName), Color: r.RequestedByColor}})
	}
	return out, nil
}

// SavePolicy stores the approval policy.
func (s *Service) SavePolicy(ctx context.Context, ws *store.Workspace, p *authz.Principal, in ApprovalPolicy) (ApprovalPolicyView, error) {
	if in.AudienceMin.Enabled && in.AudienceMin.N < 1 {
		return ApprovalPolicyView{}, api.Invalid("body.audienceMin.n", "Enter a number of recipients, like 10,000.")
	}
	if in.ApprovalsRequired < 1 || in.ApprovalsRequired > 2 {
		return ApprovalPolicyView{}, api.Invalid("body.approvalsRequired", "Require 1 or 2 approvals.")
	}
	if in.NewProviderDays <= 0 {
		in.NewProviderDays = 14
	}
	if in.AudienceMin.N <= 0 {
		in.AudienceMin.N = 10000
	}
	if in.ChannelCost.Amount <= 0 {
		in.ChannelCost.Amount = ParsePolicy(ws.ApprovalPolicy).ChannelCost.Amount
	}
	if err := s.checkSignoff(ctx, ws, in); err != nil {
		return ApprovalPolicyView{}, err
	}
	before := ParsePolicy(ws.ApprovalPolicy)
	if before == in {
		return s.PolicyView(ctx, ws, p)
	}
	raw := in.toDB(ws.ApprovalPolicy)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		next, err := q.UpdateWorkspaceApprovalPolicy(ctx, store.UpdateWorkspaceApprovalPolicyParams{ApprovalPolicy: raw, ID: ws.ID})
		if err != nil {
			return err
		}
		*ws = next
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "approval_policy.updated", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: before, After: in, Metadata: map[string]any{"summary": strings.Join(in.RuleLabels(), " · ")}})
	})
	if err != nil {
		return ApprovalPolicyView{}, err
	}
	return s.PolicyView(ctx, ws, p)
}
