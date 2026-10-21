package approvals

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// Client sign-off (agencies; channels and agencies brief §3.10): in a
// client workspace of an agency, every send needs one approval from a
// Client approver. It covers campaigns on every channel and, when chosen,
// publishing automations. Approvals from the agency's staff don't count
// toward it: they're recorded, and the request keeps waiting for the
// client's own approver.

// RuleClientSignoff is the client sign-off rule.
const RuleClientSignoff = "client_signoff"

// ApprovalClientSignoff is the client sign-off rule of a client workspace.
type ApprovalClientSignoff struct {
	Enabled bool `json:"enabled"`
	// Automations: publishing an automation needs the client's sign-off too.
	Automations bool `json:"automations"`
}

// clientSignoffDoc is the rule as stored.
type clientSignoffDoc struct {
	Enabled     bool `json:"enabled"`
	Automations bool `json:"automations"`
}

func (d *clientSignoffDoc) policy() ApprovalClientSignoff {
	if d == nil {
		return ApprovalClientSignoff{}
	}
	return ApprovalClientSignoff(*d)
}

func (c ApprovalClientSignoff) doc() *clientSignoffDoc {
	if !c.Enabled && !c.Automations {
		return nil
	}
	d := clientSignoffDoc(c)
	return &d
}

// EnableClientSignoff turns a stored policy's client sign-off on (with
// publishing automations left as it was) or off, keeping the rest.
func EnableClientSignoff(raw json.RawMessage, on bool) (json.RawMessage, error) {
	p := ParsePolicy(raw)
	p.ClientSignoff.Enabled = on
	return p.toDB(raw), nil
}

func clientSignoffMatch() ApprovalRuleMatch {
	return ApprovalRuleMatch{Key: RuleClientSignoff, Short: "client sign-off", Badge: "Client sign-off",
		Label: "Client sign-off: one approval from a Client approver"}
}

// requiredFor is how many approvals a request needs: the policy's number,
// or one when client sign-off is the only rule that matched.
func requiredFor(p ApprovalPolicy, rules []ApprovalRuleMatch) int {
	if len(rules) == 1 && rules[0].Key == RuleClientSignoff {
		return 1
	}
	return p.ApprovalsRequired
}

// staffAmong returns which of these people are the agency's staff (while
// agency mode is on): their approvals don't count toward client sign-off.
func staffAmong(ctx context.Context, q *store.Queries, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := map[uuid.UUID]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.AgencyStaffIn(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, id := range rows {
		out[id] = true
	}
	return out, nil
}

// clientApprovers are the approvers whose approval counts toward client
// sign-off: people who hold campaigns:approve and aren't agency staff.
func clientApprovers(ctx context.Context, q *store.Queries, approvers []member) ([]member, error) {
	ids := make([]uuid.UUID, 0, len(approvers))
	for _, a := range approvers {
		ids = append(ids, a.ID)
	}
	staff, err := staffAmong(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	out := []member{}
	for _, a := range approvers {
		if !staff[a.ID] {
			out = append(out, a)
		}
	}
	return out, nil
}

// signoffMet reports whether a request's client sign-off is met once these
// people approved: true when the rule didn't match, or one of them isn't
// agency staff.
func signoffMet(ctx context.Context, q *store.Queries, rules []string, approved []uuid.UUID) (bool, error) {
	if !slices.Contains(rules, RuleClientSignoff) {
		return true, nil
	}
	staff, err := staffAmong(ctx, q, approved)
	if err != nil {
		return false, err
	}
	for _, id := range approved {
		if !staff[id] {
			return true, nil
		}
	}
	return false, nil
}

// ApprovalClientSignoffState is a request's client sign-off, for the
// review page: who can give it, whether it's met, and whose approvals
// didn't count.
type ApprovalClientSignoffState struct {
	Met bool `json:"met"`
	// Approvers are the client's people who can sign off.
	Approvers []string `json:"approvers"`
	// SignedOffBy is the client approver whose approval met it.
	SignedOffBy string `json:"signedOffBy,omitempty"`
	// StaffApprovals are agency staff who approved; they don't count.
	StaffApprovals []string `json:"staffApprovals"`
	// ViewerIsStaff: the person looking is agency staff, so their own
	// approval wouldn't count either.
	ViewerIsStaff bool `json:"viewerIsStaff"`
}

// signoffState describes a request's client sign-off for a viewer.
func (s *Service) signoffState(ctx context.Context, q *store.Queries, rules []string, decisions []store.ApprovalDecisionsRow, ms []member, requester *uuid.UUID, viewer uuid.UUID) (*ApprovalClientSignoffState, error) {
	if !slices.Contains(rules, RuleClientSignoff) {
		return nil, nil
	}
	ids := []uuid.UUID{viewer}
	for _, m := range ms {
		ids = append(ids, m.ID)
	}
	for _, d := range decisions {
		if d.ApproverID != nil {
			ids = append(ids, *d.ApproverID)
		}
	}
	staff, err := staffAmong(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	out := &ApprovalClientSignoffState{Approvers: []string{}, StaffApprovals: []string{}, ViewerIsStaff: staff[viewer]}
	for _, m := range approversOf(ms) {
		if !staff[m.ID] && (requester == nil || m.ID != *requester) {
			out.Approvers = append(out.Approvers, m.Name)
		}
	}
	for _, d := range decisions {
		if d.Decision != DecisionApprove || d.ApproverID == nil {
			continue
		}
		if staff[*d.ApproverID] {
			out.StaffApprovals = append(out.StaffApprovals, d.ApproverName)
		} else if !out.Met {
			out.Met, out.SignedOffBy = true, d.ApproverName
		}
	}
	return out, nil
}

// detailSignoff is a request's client sign-off for its review page.
func (s *Service) detailSignoff(ctx context.Context, rules []string, id uuid.UUID, ms []member, requester *uuid.UUID, viewer uuid.UUID) (*ApprovalClientSignoffState, error) {
	if !slices.Contains(rules, RuleClientSignoff) {
		return nil, nil
	}
	decisions, err := s.Q.ApprovalDecisions(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	return s.signoffState(ctx, s.Q, rules, decisions, ms, requester, viewer)
}

// needClientApprover refuses a request under client sign-off when nobody
// in the workspace can give it: no Client approver (or other approver who
// isn't agency staff) besides the requester.
func (s *Service) needClientApprover(ctx context.Context, ws *store.Workspace, rules []ApprovalRuleMatch, approvers []member) error {
	if !slices.ContainsFunc(rules, func(r ApprovalRuleMatch) bool { return r.Key == RuleClientSignoff }) {
		return nil
	}
	clients, err := clientApprovers(ctx, s.Q, approvers)
	if err != nil {
		return err
	}
	if len(clients) > 0 {
		return nil
	}
	return api.Err(http.StatusConflict, "no_client_approver", fmt.Sprintf("This send needs client sign-off, and nobody at %s can give it yet. Invite the client's approver in Settings › Team with the Client approver role.", ws.Name))
}

// ApprovalClientSignoffInfo is the client sign-off rule on Settings ›
// Approvals: the agency, and the client's people who can give it.
type ApprovalClientSignoffInfo struct {
	Agency string `json:"agency"`
	// Approvers are the client's people with campaigns:approve.
	Approvers []string `json:"approvers"`
	// Staff are the agency's people among the approvers; their approvals
	// don't count toward it.
	Staff []string `json:"staff"`
}

// signoffInfo is set in an agency's client workspaces.
func (s *Service) signoffInfo(ctx context.Context, ws *store.Workspace, ms []member) (*ApprovalClientSignoffInfo, error) {
	client, err := s.Q.AgencyIsClientWorkspace(ctx, ws.ID)
	if err != nil || !client {
		return nil, err
	}
	st, err := s.Q.AgencySettings(ctx)
	if err != nil {
		return nil, err
	}
	out := &ApprovalClientSignoffInfo{Agency: st.Name, Approvers: []string{}, Staff: []string{}}
	if out.Agency == "" {
		out.Agency = st.InstanceName
	}
	approvers := approversOf(ms)
	ids := make([]uuid.UUID, 0, len(approvers))
	for _, a := range approvers {
		ids = append(ids, a.ID)
	}
	staff, err := staffAmong(ctx, s.Q, ids)
	if err != nil {
		return nil, err
	}
	for _, a := range approvers {
		if staff[a.ID] {
			out.Staff = append(out.Staff, a.Name)
		} else {
			out.Approvers = append(out.Approvers, a.Name)
		}
	}
	return out, nil
}

// checkSignoff keeps client sign-off to client workspaces: elsewhere it
// can't be turned on.
func (s *Service) checkSignoff(ctx context.Context, ws *store.Workspace, in ApprovalPolicy) error {
	if !in.ClientSignoff.Enabled && !in.ClientSignoff.Automations {
		return nil
	}
	if before := ParsePolicy(ws.ApprovalPolicy); before.ClientSignoff == in.ClientSignoff {
		return nil
	}
	client, err := s.Q.AgencyIsClientWorkspace(ctx, ws.ID)
	if err != nil {
		return err
	}
	if !client {
		return api.Invalid("body.clientSignoff", "Client sign-off is for the client workspaces of an agency. Turn agency mode on in Instance admin › Instance & updates first.")
	}
	return nil
}
