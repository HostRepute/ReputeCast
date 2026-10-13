package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/store"
)

// W09 · Campaign reports and read-only report links.
func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		reports.Register(h, &reports.Service{
			Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, URLs: a.URLs,
			Suppress: reportSuppressor{a}, Limits: reports.NewLimiter(),
		})
	})
}

// reportSuppressor adds bounced addresses from a report to the shared
// suppression list, logs it and tells every provider through the
// suppression sync.
type reportSuppressor struct{ a *App }

func (r reportSuppressor) AddSuppressions(ctx context.Context, ws *store.Workspace, emails []string, note string) (int, int, error) {
	added, already := 0, 0
	var addedEmails []string
	err := db.Tx(ctx, r.a.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := time.Now()
		for _, e := range emails {
			e = strings.ToLower(strings.TrimSpace(e))
			if e == "" {
				continue
			}
			_, err := q.IngestAddSuppression(ctx, store.IngestAddSuppressionParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Email: e, Reason: "manual",
				Source: "manual", OriginalCode: note, CreatedAt: now})
			if errors.Is(err, pgx.ErrNoRows) {
				already++
				continue
			}
			if err != nil {
				return err
			}
			added++
			addedEmails = append(addedEmails, e)
		}
		if added == 0 {
			return nil
		}
		if _, err := tx.Exec(ctx, `UPDATE contacts SET status = 'suppressed', updated_at = now() WHERE workspace_id = $1 AND email = ANY ($2) AND status = 'active'`, ws.ID, addedEmails); err != nil {
			return err
		}
		name := strings.Join(addedEmails[:min(3, len(addedEmails))], ", ")
		return r.a.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "suppression.add", TargetType: "suppression", TargetName: name, Reason: note,
			Metadata: map[string]any{"count": added, "emails": addedEmails[:min(50, len(addedEmails))], "source": "report"}})
	})
	if err != nil {
		return 0, 0, err
	}
	if added > 0 && r.a.Jobs != nil {
		r.a.suppressionSync().Enqueue(ctx, ws.ID)
	}
	return added, already, nil
}
