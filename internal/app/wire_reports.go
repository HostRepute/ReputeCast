package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/store"
)

// Campaign reports and read-only report links.
func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		reports.Register(h, &reports.Service{
			Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, URLs: a.URLs,
			Suppress: reportSuppressor{a}, Limits: reports.NewLimiter(),
			// Results per language for campaigns sent in several languages.
			Languages: &languages.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
		})
	})
}

// reportSuppressor adds bounced addresses from a report to the shared
// suppression list through the audience domain (which marks the contacts
// suppressed, records their consent history and writes the audit entry),
// then has the suppression sync tell every provider.
type reportSuppressor struct{ a *App }

// maxNote is the longest note the suppression list keeps.
const maxNote = 300

func (r reportSuppressor) AddSuppressions(ctx context.Context, ws *store.Workspace, emails []string, note string, text i18n.Text) (int, int, error) {
	if n := []rune(note); len(n) > maxNote {
		note = string(n[:maxNote-1]) + "…"
	}
	res, err := r.a.audience().AddSuppressionsKept(ctx, ws, emails, note, text)
	if err != nil {
		return 0, 0, err
	}
	if res.Added > 0 && r.a.Jobs != nil {
		r.a.suppressionSync().Enqueue(ctx, ws.ID)
	}
	return res.Added, res.Already, nil
}
