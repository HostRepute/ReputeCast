package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/hostrepute/reputecast/internal/store"
)

// ImportJob is an import shown in the background-jobs tray: running ones
// with their progress, and ones that ended in the last two hours.
type ImportJob struct {
	ID   string `json:"id"`
	Kind string `json:"kind" enum:"import"`
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

// ImportJobList is the jobs tray's list.
type ImportJobList struct {
	Items []ImportJob `json:"items"`
}

// BackgroundJobs lists the imports for the jobs tray.
func (s *Service) BackgroundJobs(ctx context.Context, ws *store.Workspace) (ImportJobList, error) {
	now := s.now()
	rows, err := s.Q.ImportJobs(ctx, store.ImportJobsParams{WorkspaceID: ws.ID, EndedAfter: ptrTime(now.Add(-2 * time.Hour))})
	if err != nil {
		return ImportJobList{}, err
	}
	out := ImportJobList{Items: make([]ImportJob, 0, len(rows))}
	loc := location(ws)
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
			j.Status, j.Title = "running", fmt.Sprintf("Reading contacts from %s", name)
		case "running", "paused":
			j.Status = "running"
			j.Title = fmt.Sprintf("Importing %s contacts from %s", thousands(int(total)), name)
			if total > 0 {
				pct := int(math.Min(99, math.Floor(float64(p.Processed)*100/float64(total))))
				j.Progress = &pct
			}
			if r.Status == "paused" {
				j.Status = "queued"
				j.Detail = fmt.Sprintf("Paused at row %s", thousands(int(p.Processed)))
				if r.ResumeAt != nil {
					j.Detail += fmt.Sprintf("; resumes at %s", r.ResumeAt.In(loc).Format("15:04"))
				}
			} else if r.StartedAt != nil && p.Processed > 0 && total > p.Processed {
				j.Eta = eta(now.Sub(*r.StartedAt), p.Processed, total)
			}
		case "done":
			j.Status, j.Title = "done", fmt.Sprintf("Imported from %s", name)
			j.Detail = fmt.Sprintf("%s contacts imported · %s suppressed", thousands(int(st.Imported+st.Merged)), thousands(int(st.Suppressed)))
			j.Href = fmt.Sprintf("/w/%s/import/%s/report", ws.Slug, r.ID)
		case "failed":
			j.Status, j.Title, j.Detail = "failed", fmt.Sprintf("Import from %s failed", name), r.Error
			j.Href = fmt.Sprintf("/w/%s/import/%s/report", ws.Slug, r.ID)
		case "cancelled":
			j.Status, j.Title = "canceled", fmt.Sprintf("Import from %s stopped", name)
			j.Detail = fmt.Sprintf("%s of %s rows were processed before it stopped", thousands(int(p.Processed)), thousands(int(total)))
			j.Href = fmt.Sprintf("/w/%s/import/%s/report", ws.Slug, r.ID)
		default:
			continue
		}
		out.Items = append(out.Items, j)
	}
	return out, nil
}

// eta estimates the time left from the pace so far: "about 3 min left".
func eta(elapsed time.Duration, done, total int64) string {
	left := time.Duration(float64(elapsed) * float64(total-done) / float64(done))
	switch m := int(math.Ceil(left.Minutes())); {
	case left < time.Minute:
		return "less than a minute left"
	case m == 1:
		return "about 1 minute left"
	default:
		return fmt.Sprintf("about %d minutes left", m)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
