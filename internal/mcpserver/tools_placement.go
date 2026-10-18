package mcpserver

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/placement"
	"github.com/hostrepute/reputecast/internal/store"
)

// PlacementReader reads the workspace's placement tests (X14). Agents can
// read results; running tests and managing seed mailboxes stay with people
// (placement:run and placement:manage can't be granted to agents).
type PlacementReader interface {
	Tests(ctx context.Context, ws *store.Workspace) (placement.PlacementTestList, error)
	Test(ctx context.Context, ws *store.Workspace, id uuid.UUID) (placement.PlacementTestDetail, error)
}

// PlacementRef names a placement test by id or name.
type PlacementRef struct {
	Test string `json:"test" jsonschema:"The test's id, or its name (Weekly Digest #142 · SES vs Postal)"`
}

func placementCounts(c placement.PlacementCounts) map[string]int {
	return map[string]int{"copies": c.Copies, "inbox": c.Inbox, "tab": c.Tab, "spam": c.Spam, "not_arrived": c.NotArrived,
		"not_checked": c.NotChecked, "waiting": c.Waiting}
}

func placementSummary(r placement.PlacementTestRow) map[string]any {
	ps := []map[string]any{}
	for _, p := range r.Providers {
		ps = append(ps, map[string]any{"provider": p.Name, "results": placementCounts(p.Counts), "median_seconds_to_arrive": medianSeconds(p.MedianMs)})
	}
	return map[string]any{"id": r.ID, "name": r.Name, "content": r.ContentName, "content_kind": r.ContentKind, "through": r.Through,
		"status": r.Status, "started_at": r.StartedAt, "started_by": r.StartedBy, "providers": ps}
}

func medianSeconds(ms *int64) any {
	if ms == nil {
		return nil
	}
	return float64(*ms) / 1000
}

// addPlacementTools adds list_placement_tests and get_placement_test.
func (s *Service) addPlacementTools(srv *mcp.Server) {
	tool(s, srv, "list_placement_tests", authz.DeliverabilityView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "placement.tests_read", targetType: "placement_test"}
		if s.Placement == nil {
			return map[string]any{"tests": []any{}, "message": "Placement tests aren't available on this server."}, e, nil
		}
		l, err := s.Placement.Tests(ctx, &c.Workspace)
		if err != nil {
			return nil, e, err
		}
		out := []map[string]any{}
		for _, t := range l.Tests {
			out = append(out, placementSummary(t))
		}
		return map[string]any{"tests": out, "seed_mailboxes": l.SeedsConnected,
			"note": "Each test is one copy of an email to each seed mailbox, started by a person. 10 seeds are a sample, not the audience."}, e, nil
	}, nil)
	tool(s, srv, "get_placement_test", authz.DeliverabilityView, func(ctx context.Context, c *agentCall, in PlacementRef) (any, entry, error) {
		e := entry{action: "placement.test_read", targetType: "placement_test", targetName: in.Test}
		if s.Placement == nil {
			return map[string]any{"message": "Placement tests aren't available on this server."}, e, nil
		}
		id, err := uuid.Parse(strings.TrimSpace(in.Test))
		if err != nil {
			l, err := s.Placement.Tests(ctx, &c.Workspace)
			if err != nil {
				return nil, e, err
			}
			for _, t := range l.Tests {
				if strings.EqualFold(t.Name, strings.TrimSpace(in.Test)) {
					id = uuid.MustParse(t.ID)
					break
				}
			}
			if id == uuid.Nil {
				return nil, e, errors.New("there is no placement test named " + in.Test)
			}
		}
		d, err := s.Placement.Test(ctx, &c.Workspace, id)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = d.ID, d.Name
		copies := []map[string]any{}
		for _, cp := range d.Copies {
			copies = append(copies, map[string]any{"seed": cp.SeedEmail, "mailbox_provider": cp.SeedProviderName, "provider": cp.ProviderName,
				"status": cp.Status, "folder": cp.Folder, "folder_name": cp.FolderName, "tab": cp.Tab, "late": cp.Late,
				"seconds_to_arrive": medianSeconds(cp.ArriveMs), "spf": cp.SPF, "dkim": cp.DKIM, "dmarc": cp.DMARC, "reason": cp.Reason})
		}
		guidance := []map[string]any{}
		for _, g := range d.Guidance {
			guidance = append(guidance, map[string]any{"provider": g.ProviderName, "spam": g.Spam, "copies": g.Copies, "likely_cause": g.Cause, "at": g.At})
		}
		out := placementSummary(d.PlacementTestRow)
		out["copies"], out["guidance"], out["from"] = copies, guidance, d.FromEmail
		out["note"] = "Opens and clicks of these copies are never counted. Guidance comes from ReputeCast's own rules, not from AI."
		return out, e, nil
	}, nil)
}
