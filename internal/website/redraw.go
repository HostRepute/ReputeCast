package website

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/render"
)

// RedrawPollPosts draws again the published posts whose web version still
// shows a poll's answers: drawn before a poll stayed in the inbox, those
// answers led to /0, /1 … on the site. Each is drawn from the content it
// was published with, so its version, hash and publication stay; once
// drawn, its notes list the poll, and it isn't picked again. It returns how
// many posts it drew.
func (s *Service) RedrawPollPosts(ctx context.Context) (int, error) {
	rows, err := s.Q.WebsitePostsWithPollAnswers(ctx)
	if err != nil {
		return 0, err
	}
	workspaces := map[uuid.UUID]*store.Workspace{}
	n := 0
	for _, r := range rows {
		ws, ok := workspaces[r.WorkspaceID]
		if !ok {
			w, err := s.Q.GetWorkspace(ctx, r.WorkspaceID)
			if err != nil {
				return n, err
			}
			ws = &w
			workspaces[r.WorkspaceID] = ws
		}
		p := store.WebsitePost{ID: r.ID, WorkspaceID: r.WorkspaceID, Title: r.Title, CoverUrl: r.CoverUrl, Slug: r.Slug}
		res, err := render.Render(r.PublishedMjml, s.renderOptions(ctx, ws, p))
		if err != nil {
			// It was drawn once; a content that no longer parses keeps it.
			s.log().WarnContext(ctx, "redraw post", "post", r.ID, "err", err)
			continue
		}
		notes, _ := json.Marshal(renderNotes{Hidden: res.Hidden, Fallbacks: res.Fallbacks})
		paid, _ := json.Marshal(res.Paid)
		if err := s.Q.WebsitePostRedrawn(ctx, store.WebsitePostRedrawnParams{ID: r.ID, BodyHtml: res.Body, PreviewHtml: res.Preview,
			HasMarker: res.HasMarker, MarkerAfter: int32(res.MarkerAfter), BodyText: res.Text, PreviewText: res.PreviewText,
			Words: int32(res.Words), RenderNotes: notes, PaidBlocks: paid}); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// RedrawArgs is the daily job (and one at start) that draws again the
// posts RedrawPollPosts finds.
type RedrawArgs struct{}

// Kind implements river.JobArgs.
func (RedrawArgs) Kind() string { return "website.redraw" }

// InsertOpts runs on the maintenance queue, once at a time.
func (RedrawArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 23 * time.Hour}}
}

type redrawWorker struct {
	river.WorkerDefaults[RedrawArgs]
	s *Service
}

func (w *redrawWorker) Work(ctx context.Context, _ *river.Job[RedrawArgs]) error {
	n, err := w.s.RedrawPollPosts(ctx)
	if n > 0 {
		w.s.log().InfoContext(ctx, "posts drawn again without their poll's answers", "posts", n)
	}
	return err
}

func (w *redrawWorker) Timeout(*river.Job[RedrawArgs]) time.Duration { return 10 * time.Minute }
