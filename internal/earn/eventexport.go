package earn

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// ExportEvents writes the payments event log a filter shows as CSV (review
// 2026-11 EA-2): what the log's table shows, not the raw payloads. It holds
// members' addresses, so it is audited.
func (s *Service) ExportEvents(ctx context.Context, ws *store.Workspace, p *authz.Principal, f EarnEventFilter) ([]byte, int, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"received_at", "occurred_at", "type", "event_id", "mode", "source", "status", "outcome", "amount", "currency", "member", "member_email"})
	rows := 0
	f.PageSize = 100
	for f.Page = 1; rows < 100_000; f.Page++ {
		l, err := s.EventLog(ctx, ws, p, f)
		if err != nil {
			return nil, 0, err
		}
		for _, e := range l.Items {
			amount := ""
			if e.AmountCents != nil {
				amount = fmt.Sprintf("%d.%02d", *e.AmountCents/100, absCents(*e.AmountCents%100))
			}
			_ = w.Write([]string{e.ReceivedAt.UTC().Format(time.RFC3339), e.OccurredAt.UTC().Format(time.RFC3339), e.Type, e.EventID, e.Mode, e.Source,
				e.Status, e.Outcome, amount, strings.ToUpper(e.Currency), e.MemberName, e.MemberEmail})
			rows++
		}
		if len(l.Items) == 0 || int64(f.Page*l.PageSize) >= l.Total {
			break
		}
	}
	w.Flush()
	err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "payment_events.exported", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
		Metadata: map[string]any{"rows": rows, "filter": map[string]any{"group": f.Group, "range": f.Range, "q": f.Q, "test": f.Test},
			"summary": fmt.Sprintf("Exported %d payment events as CSV", rows)}})
	return buf.Bytes(), rows, err
}

func absCents(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
