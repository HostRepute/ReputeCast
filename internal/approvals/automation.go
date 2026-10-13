package approvals

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// AutomationGate asks for approval before an automation version goes live
// when the workspace policy says so (automations.PublishGate). Once
// approved, the approvals domain publishes it with Service.CompletePublish.
type AutomationGate struct{ S *Service }

// Check implements automations.PublishGate, inside the publish transaction.
func (g AutomationGate) Check(ctx context.Context, tx pgx.Tx, req automations.PublishRequest) (automations.GateDecision, error) {
	q := store.New(tx)
	ws, err := q.GetWorkspace(ctx, req.WorkspaceID)
	if err != nil {
		return automations.GateDecision{}, err
	}
	policy := ParsePolicy(ws.ApprovalPolicy)
	if !policy.AutomationPublish {
		return automations.GateDecision{}, nil
	}
	ms, err := g.S.members(ctx, q, ws.ID)
	if err != nil {
		return automations.GateDecision{}, err
	}
	approvers := []member{}
	names := []string{}
	for _, a := range approversOf(ms) {
		if a.ID != req.By {
			approvers = append(approvers, a)
			names = append(names, a.Name)
		}
	}
	if len(approvers) == 0 {
		return automations.GateDecision{}, fmt.Errorf("publishing needs approval and nobody else holds campaigns:approve")
	}
	who := strings.Join(names, " or ")
	if len(names) > 2 {
		who = strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
	}
	auto := req.AutomationID
	var existing uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM approval_requests WHERE automation_id = $1 AND status = 'pending' LIMIT 1`, auto).Scan(&existing); err == nil {
		return automations.GateDecision{Required: true, ApprovalID: existing, Message: "Approval is already requested. Waiting for " + who + "."}, nil
	}
	changes := []string{}
	for _, it := range req.Changes {
		if it.Kind != "unchanged" {
			changes = append(changes, it.Kind+": "+it.Title)
		}
	}
	snap := Snapshot{Version: req.Version, Rules: []ApprovalRuleMatch{{Key: RuleAutomationPublish, Label: "Publishing an automation requires approval", Short: "automation publish", Badge: "Automation publish"}},
		Automation: &ApprovalAutomationSnapshot{Version: req.Version, Migration: req.Migration, Changes: changes}}
	if m := memberByID(ms, &req.By); m != nil {
		snap.AuthorRole = m.RoleName
	}
	sb, _ := json.Marshal(snap)
	id := uuid.Must(uuid.NewV7())
	by := req.By
	if err := q.ApprovalInsertRequest(ctx, store.ApprovalInsertRequestParams{ID: id, WorkspaceID: ws.ID, SubjectType: "automation", AutomationID: &auto, RequestedBy: &by,
		RequestedAt: g.S.now(), Status: "pending", MatchedRules: []string{RuleAutomationPublish}, ApprovalsRequired: int32(policy.ApprovalsRequired), Snapshot: sb}); err != nil {
		return automations.GateDecision{}, err
	}
	author := ""
	if m := memberByID(ms, &req.By); m != nil {
		author = m.Name
	}
	if err := emitRequested(ctx, tx, &ws, id, "automation", auto, req.Name, author, []string{RuleAutomationPublish}, 0); err != nil {
		return automations.GateDecision{}, err
	}
	n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalRequested, Severity: "warn", Title: "Approval requested",
		Summary: fmt.Sprintf("%s asked you to approve publishing “%s” v%d", author, req.Name, req.Version),
		Body:    fmt.Sprintf("%s asked to publish “%s” v%d · %d %s", author, req.Name, req.Version, len(changes), pluralWord(len(changes), "change", "changes")),
		Link:    "/w/" + ws.Slug + "/approvals/" + id.String(), Data: map[string]any{"request_id": id.String(), "automation_id": auto.String()}}
	for _, a := range approvers {
		if err := notify.ToUser(ctx, tx, a.ID, n); err != nil {
			return automations.GateDecision{}, err
		}
	}
	return automations.GateDecision{Required: true, ApprovalID: id, Message: "Waiting for approval from " + who + "."}, nil
}

func pluralWord(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// completePublish publishes an approved automation version.
func (s *Service) completePublish(ctx context.Context, ws *store.Workspace, by, id uuid.UUID, version int, migration string) {
	if s.Automations == nil {
		return
	}
	if migration == "" {
		migration = automations.MigrationFinish
	}
	if err := s.Automations.CompletePublish(ctx, ws, by, id, version, migration); err != nil {
		if s.Log != nil {
			s.Log.WarnContext(ctx, "publishing an approved automation failed", "automation", id, "err", err)
		}
		_ = notify.ToUser(ctx, s.Pool, by, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalDecided, Severity: "warn", Title: "Approved, but not published",
			Summary: "The automation was approved but couldn't be published: " + err.Error(),
			Body:    "The automation was approved but couldn't be published: " + err.Error() + " Publish it again to request a new approval.",
			Link:    "/w/" + ws.Slug + "/automations/" + id.String()})
	}
}
