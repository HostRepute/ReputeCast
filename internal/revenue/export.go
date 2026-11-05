package revenue

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/csvexport"
	"github.com/hostrepute/reputecast/internal/store"
)

// exportMax caps an export of the conversions log.
const exportMax = 100_000

// ExportConversions writes the conversions log as CSV, newest first
// (review 2026-11 EA-2). Audited.
func (s *Service) ExportConversions(ctx context.Context, ws *store.Workspace) ([]byte, int, error) {
	var buf bytes.Buffer
	w := csvexport.NewWriter(&buf)
	_ = w.Write([]string{"received_at", "occurred_at", "name", "value", "currency", "matched_by", "credited_kind", "credited_to", "key", "test"})
	rows := 0
	q := ConversionsQuery{Limit: 200}
	for rows < exportMax {
		l, err := s.Conversions(ctx, ws, q)
		if err != nil {
			return nil, 0, err
		}
		for _, c := range l.Items {
			kind, to := "", ""
			if c.Credited != nil {
				kind, to = c.Credited.Kind, c.Credited.Name
			}
			_ = w.Write([]string{c.ReceivedAt.UTC().Format(time.RFC3339), c.OccurredAt.UTC().Format(time.RFC3339), c.Name,
				fmt.Sprintf("%d.%02d", c.ValueCents/100, abs(c.ValueCents%100)), strings.ToUpper(c.Currency), c.Matched, kind, to, c.KeyName, strconv.FormatBool(c.Test)})
			rows++
		}
		if l.Next == nil || len(l.Items) == 0 {
			break
		}
		q.Before = *l.Next
	}
	w.Flush()
	err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "conversion.exported", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
		Metadata: map[string]any{"rows": rows, "summary": fmt.Sprintf("Exported %d conversions as CSV", rows)}})
	return buf.Bytes(), rows, err
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
