package website

import (
	"context"
	"math"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/importer"
)

// ImportTrayJob is an import of posts in the background-jobs tray:
// running ones with their progress, and ones that ended in the last two
// hours.
type ImportTrayJob struct {
	ID         string
	Title      string
	Status     string // queued, running, done, failed, canceled
	Progress   *int
	Eta        string
	Detail     string
	Href       string
	StartedAt  *time.Time
	FinishedAt *time.Time
}

// ImportTray lists the imports of posts for the jobs tray, in the reader's
// language.
func (s *Service) ImportTray(ctx context.Context, ws *store.Workspace) ([]ImportTrayJob, error) {
	l := i18n.LangFrom(ctx)
	now := s.now()
	rows, err := s.Q.WebsiteImportTray(ctx, store.WebsiteImportTrayParams{WorkspaceID: ws.ID, EndedAfter: now.Add(-2 * time.Hour)})
	if err != nil {
		return nil, err
	}
	out := []ImportTrayJob{}
	for _, r := range rows {
		st := decodeStats(r.Stats)
		pr := st.Progress
		if pr == nil {
			pr = &WebsiteImportProgress{}
		}
		args := i18n.Args{"source": importer.Name(r.Source), "done": pr.Done, "total": pr.Total, "count": pr.Published + pr.Drafts + pr.Updated}
		created := r.CreatedAt
		j := ImportTrayJob{ID: r.ID.String(), Href: "/w/" + ws.Slug + "/forms/website/import?import=" + r.ID.String(), StartedAt: &created, FinishedAt: r.FinishedAt}
		switch r.Status {
		case ImportUploaded, ImportAnalysing:
			j.Status, j.Title = "running", msgs.M("import.tray.analysing").Text(l, args)
		case ImportReady:
			// Reading the export is done: the review waits.
			j.Status, j.Title, j.FinishedAt = "done", msgs.M("import.tray.ready").Text(l, args), &r.UpdatedAt
		case ImportImporting:
			j.Status, j.Title, j.Detail = "running", msgs.M("import.tray.running").Text(l, args), msgs.M("import.tray.runningDetail").Text(l, args)
			if pr.StartedAt != nil {
				j.StartedAt = pr.StartedAt
			}
			if pr.Total > 0 {
				pct := int(math.Min(99, math.Floor(float64(pr.Done)*100/float64(pr.Total))))
				j.Progress = &pct
				if pr.StartedAt != nil && pr.Done > 0 && pr.Total > pr.Done {
					left := time.Duration(float64(now.Sub(*pr.StartedAt)) * float64(pr.Total-pr.Done) / float64(pr.Done))
					j.Eta = etaText(l, left)
				}
			}
		case ImportDone:
			j.Status, j.Title, j.Detail = "done", msgs.M("import.tray.done").Text(l, args), msgs.M("import.tray.doneDetail").Text(l, nil)
		case ImportFailed:
			j.Status, j.Title = "failed", msgs.M("import.tray.failed").Text(l, args)
			if st.ErrorText != nil {
				j.Detail = st.ErrorText.In(l)
			}
		case ImportCanceled:
			j.Status, j.Title, j.Detail = "canceled", msgs.M("import.tray.canceled").Text(l, args), msgs.M("import.tray.canceledDetail").Text(l, nil)
		default:
			continue
		}
		out = append(out, j)
	}
	return out, nil
}

// etaText is the time a run has left: "about 3 minutes left".
func etaText(l i18n.Lang, left time.Duration) string {
	if left < time.Minute {
		return msgs.M("import.tray.etaSoon").Text(l, nil)
	}
	return msgs.M("import.tray.eta").Text(l, i18n.Args{"minutes": int64(math.Ceil(left.Minutes()))})
}
