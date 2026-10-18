package verification

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// TrayJob is a verification job in the background-jobs tray: running ones
// with their progress, and ones that ended in the last two hours.
type TrayJob struct {
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

// TrayJobs lists the jobs for the tray, in the reader's language.
func (s *Service) TrayJobs(ctx context.Context, ws *store.Workspace) ([]TrayJob, error) {
	l := i18n.LangFrom(ctx)
	now := s.now()
	rows, err := s.Q.VerifJobList(ctx, store.VerifJobListParams{WorkspaceID: ws.ID, Since: now.Add(-2 * time.Hour), Lim: 20})
	if err != nil {
		return nil, err
	}
	out := []TrayJob{}
	for _, r := range rows {
		j := r.VerificationJob
		args := i18n.Args{"kind": j.Kind, "label": j.Label, "service": ServiceName(j.Service), "checked": int64(j.Checked), "total": int64(j.Total),
			"suppressed": int64(j.Suppressed), "risky": int64(j.Risky)}
		t := TrayJob{ID: j.ID.String(), Href: fmt.Sprintf("/w/%s/audience/verification/%s", ws.Slug, j.ID), StartedAt: j.StartedAt, FinishedAt: j.FinishedAt}
		switch j.Status {
		case "queued", "running":
			t.Status, t.Title = "running", mTrayRunning.Text(l, args)
			if j.Total > 0 {
				pct := int(math.Min(99, math.Floor(float64(j.Checked)*100/float64(j.Total))))
				t.Progress = &pct
				if j.StartedAt != nil && j.Checked > 0 && j.Total > j.Checked {
					left := time.Duration(float64(now.Sub(*j.StartedAt)) * float64(j.Total-j.Checked) / float64(j.Checked))
					if left < time.Minute {
						t.Eta = mTrayEtaSoon.Text(l, nil)
					} else {
						t.Eta = mTrayEta.Text(l, i18n.Args{"minutes": int64(math.Ceil(left.Minutes()))})
					}
				}
			}
		case "paused":
			t.Status, t.Title, t.Detail = "queued", mTrayRunning.Text(l, args), mTrayPaused.Text(l, args)
		case "finished":
			t.Status, t.Title, t.Detail = "done", mTrayDone.Text(l, args), mTrayDoneDetail.Text(l, args)
		case "stopped":
			t.Status, t.Title, t.Detail = "failed", mTrayStopped.Text(l, args), mTrayStoppedDetail.Text(l, args)
		case "cancelled":
			t.Status, t.Title, t.Detail = "canceled", mTrayCancelled.Text(l, args), mTrayStoppedDetail.Text(l, args)
		default:
			continue
		}
		out = append(out, t)
	}
	return out, nil
}
