package manage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/publicurl"
)

// A new public URL moves every provider webhook (spec 5.5): providers whose
// adapter registers webhooks get them re-registered by MoveWebhooksArgs;
// the others are set up in the provider itself, so the admin is told where
// to paste the new address.

// WebhookMove is what a new public URL means for provider webhooks.
type WebhookMove struct {
	// Reregistering counts the providers whose webhook ReputeCast moves.
	Reregistering int
	// Manual lists the providers whose webhook has to be changed by hand.
	Manual []ManualWebhook
}

// ManualWebhook is a provider whose webhook is set up in the provider itself.
type ManualWebhook struct {
	Workspace string
	Provider  string
	URL       string
}

// PublicURLMoved sorts the connected providers of every workspace for a new
// public URL and queues the re-registration (demo providers have no real
// webhook and are left out).
func (s *Service) PublicURLMoved(ctx context.Context, publicURL string) (WebhookMove, error) {
	provs, err := s.Q.ProvListAllActive(ctx)
	if err != nil {
		return WebhookMove{}, err
	}
	urls := publicurl.New(publicURL)
	out := WebhookMove{Manual: []ManualWebhook{}}
	var ids []uuid.UUID
	names := map[uuid.UUID]string{}
	for _, p := range provs {
		if IsDemo(p) {
			continue
		}
		if adapter, err := s.Factory.Open(ctx, p); err == nil {
			if _, ok := adapter.(providers.WebhookRegistrar); ok {
				ids = append(ids, p.ID)
				continue
			}
		}
		ws, ok := names[p.WorkspaceID]
		if !ok {
			if w, err := s.Q.GetWorkspace(ctx, p.WorkspaceID); err == nil {
				ws = w.Name
			}
			names[p.WorkspaceID] = ws
		}
		out.Manual = append(out.Manual, ManualWebhook{Workspace: ws, Provider: p.Name, URL: WebhookURL(urls, p)})
	}
	out.Reregistering = len(ids)
	if len(ids) > 0 {
		c := s.jobs()
		if c == nil {
			return out, errors.New("move webhooks: no job client")
		}
		if _, err := c.Insert(ctx, MoveWebhooksArgs{ProviderIDs: ids}, nil); err != nil {
			return out, err
		}
	}
	return out, nil
}

// MoveWebhooksArgs re-registers providers' webhooks at the public URL.
type MoveWebhooksArgs struct {
	ProviderIDs []uuid.UUID `json:"provider_ids"`
}

// Kind implements river.JobArgs.
func (MoveWebhooksArgs) Kind() string { return "providers.move_webhooks" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (MoveWebhooksArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 8}
}

// MoveWebhooksWorker runs MoveWebhooksArgs. It reads the public URL when it
// runs, so a job that waited behind a newer change still registers the
// newest address; registering is idempotent, so a retry repeats it all.
type MoveWebhooksWorker struct {
	river.WorkerDefaults[MoveWebhooksArgs]
	S *Service
	// PublicURL reads the instance's public URL from the database.
	PublicURL func(ctx context.Context) string
}

// Work implements river.Worker.
func (w *MoveWebhooksWorker) Work(ctx context.Context, job *river.Job[MoveWebhooksArgs]) error {
	base := w.S.URLs.Base()
	if w.PublicURL != nil {
		base = w.PublicURL(ctx)
	}
	urls := publicurl.New(base)
	var failed []error
	for _, id := range job.Args.ProviderIDs {
		p, err := w.S.Q.ProvGetAny(ctx, id)
		if err != nil || p.DisconnectedAt != nil {
			continue
		}
		ws, err := w.S.Q.GetWorkspace(ctx, p.WorkspaceID)
		if err != nil {
			continue
		}
		adapter, err := w.S.Factory.Open(ctx, p)
		if err != nil {
			failed = append(failed, fmt.Errorf("%s: %w", p.Name, err))
			continue
		}
		wr, ok := adapter.(providers.WebhookRegistrar)
		if !ok {
			continue
		}
		url := WebhookURL(urls, p)
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err = wr.RegisterWebhook(cctx, url)
		cancel()
		if err != nil {
			w.S.log().WarnContext(ctx, "move webhook to the new public URL", "provider", p.ID, "err", err)
			failed = append(failed, fmt.Errorf("%s: %w", p.Name, err))
			continue
		}
		if err := w.S.audit(ctx, w.S.Q, &ws, p, "provider.webhook_register", nil, nil, map[string]any{"url": url, "reason": "public URL changed"}); err != nil {
			return err
		}
	}
	return errors.Join(failed...)
}
