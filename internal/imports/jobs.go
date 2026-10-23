package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// ImportJob is an import shown in the background-jobs tray: running ones
// with their progress, and ones that ended in the last two hours.
type ImportJob struct {
	ID   string `json:"id"`
	Kind string `json:"kind" enum:"import,verification"`
	// Title: "Importing 13,739 contacts from Mailchimp".
	Title  string `json:"title"`
	Status string `json:"status" enum:"queued,running,done,failed,canceled"`
	// Progress is 0–100 while running.
	Progress *int `json:"progress,omitempty"`
	// Eta: "about 1 min left".
	Eta string `json:"eta,omitempty"`
	// Detail is the result or the failure in words.
	Detail string `json:"detail,omitempty"`
	// Href is the app path of the import's progress or report.
	Href       string     `json:"href"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// JobSource adds another domain's jobs to the tray (address verification).
type JobSource func(ctx context.Context, ws *store.Workspace) ([]ImportJob, error)

// ImportJobList is the jobs tray's list.
type ImportJobList struct {
	Items []ImportJob `json:"items"`
}

// BackgroundJobs lists the imports for the jobs tray, in the request's
// language.
func (s *Service) BackgroundJobs(ctx context.Context, ws *store.Workspace) (ImportJobList, error) {
	now := s.now()
	rows, err := s.Q.ImportJobs(ctx, store.ImportJobsParams{WorkspaceID: ws.ID, EndedAfter: ptrTime(now.Add(-2 * time.Hour))})
	if err != nil {
		return ImportJobList{}, err
	}
	out := ImportJobList{Items: make([]ImportJob, 0, len(rows))}
	loc := location(ws)
	lang := i18n.LangFrom(ctx)
	for _, r := range rows {
		name := SourceName(r.Source)
		var p ImportProgress
		_ = json.Unmarshal(r.Progress, &p)
		var st ImportStats
		_ = json.Unmarshal(r.Stats, &st)
		j := ImportJob{ID: r.ID.String(), Kind: "import", StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
			Href: fmt.Sprintf("/w/%s/import/%s/progress", ws.Slug, r.ID)}
		total := p.Total
		if total == 0 {
			conn := parseConnection(r.Connection)
			for _, l := range conn.Lists {
				total += l.Subscribed + l.Unsubscribed + l.Cleaned + l.Pending + l.Complained
			}
		}
		switch r.Status {
		case "staging":
			j.Status, j.Title = "running", mJobReading.Text(lang, i18n.Args{"source": name})
		case "running", "paused":
			j.Status = "running"
			j.Title = mJobImporting.Text(lang, i18n.Args{"total": total, "source": name})
			if total > 0 {
				pct := int(math.Min(99, math.Floor(float64(p.Processed)*100/float64(total))))
				j.Progress = &pct
			}
			if r.Status == "paused" {
				j.Status = "queued"
				j.Detail = mJobPaused.Text(lang, i18n.Args{"row": p.Processed})
				if r.ResumeAt != nil {
					j.Detail = mJobPausedResumes.Text(lang, i18n.Args{"row": p.Processed, "time": clockIn(lang, r.ResumeAt.In(loc))})
				}
			} else if r.StartedAt != nil && p.Processed > 0 && total > p.Processed {
				j.Eta = eta(lang, now.Sub(*r.StartedAt), p.Processed, total)
			}
		case "done":
			j.Status, j.Title = "done", mJobDone.Text(lang, i18n.Args{"source": name})
			j.Detail = mJobDoneDetail.Text(lang, i18n.Args{"imported": st.Imported + st.Merged, "suppressed": st.Suppressed})
			j.Href = fmt.Sprintf("/w/%s/import/%s/report", ws.Slug, r.ID)
		case "failed":
			// The failure is the system's own words (a database error).
			j.Status, j.Title, j.Detail = "failed", mJobFailed.Text(lang, i18n.Args{"source": name}), r.Error
			j.Href = fmt.Sprintf("/w/%s/import/%s/report", ws.Slug, r.ID)
		case "cancelled":
			j.Status, j.Title = "canceled", mJobStopped.Text(lang, i18n.Args{"source": name})
			j.Detail = mJobStoppedDetail.Text(lang, i18n.Args{"processed": p.Processed, "total": total})
			j.Href = fmt.Sprintf("/w/%s/import/%s/report", ws.Slug, r.ID)
		default:
			continue
		}
		out.Items = append(out.Items, j)
	}
	for _, src := range s.JobSources {
		if more, err := src(ctx, ws); err == nil {
			out.Items = append(out.Items, more...)
		}
	}
	return out, nil
}

// eta estimates the time left from the pace so far: "about 3 minutes left".
func eta(l i18n.Lang, elapsed time.Duration, done, total int64) string {
	left := time.Duration(float64(elapsed) * float64(total-done) / float64(done))
	if left < time.Minute {
		return mEtaSoon.Text(l, nil)
	}
	return mEtaMinutes.Text(l, i18n.Args{"n": int(math.Ceil(left.Minutes()))})
}

func ptrTime(t time.Time) *time.Time { return &t }
