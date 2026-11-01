package gifts

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/store"
)

// exportMax caps an export; the gifts table pages 20 at a time.
const exportMax = 100_000

// Export writes the gifts a filter shows as CSV (review 2026-11 EA-2). They
// hold buyers' and recipients' addresses, so the export is audited.
func (s *Service) Export(ctx context.Context, ws *store.Workspace, filter, search string) ([]byte, int, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"bought_at", "buyer", "buyer_email", "in_person", "recipient", "recipient_email", "tier", "months", "amount", "currency", "status"})
	rows := 0
	for page := 1; rows < exportMax; page++ {
		l, err := s.List(ctx, ws, filter, search, page)
		if err != nil {
			return nil, 0, err
		}
		for _, g := range l.Items {
			_ = w.Write([]string{g.BoughtAt.UTC().Format(time.RFC3339), g.Buyer, g.BuyerEmail, strconv.FormatBool(g.InPerson), g.RecipientName, g.RecipientEmail,
				g.Tier, strconv.Itoa(g.Months), fmt.Sprintf("%d.%02d", g.AmountCents/100, g.AmountCents%100), strings.ToUpper(g.Currency), g.Status})
			rows++
		}
		if len(l.Items) == 0 || int64(page*l.PageSize) >= l.Total {
			break
		}
	}
	w.Flush()
	err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "gift.exported", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
		Metadata: map[string]any{"rows": rows, "filter": map[string]any{"status": filter, "q": search}, "summary": fmt.Sprintf("Exported %d gifts as CSV", rows)}})
	return buf.Bytes(), rows, err
}
