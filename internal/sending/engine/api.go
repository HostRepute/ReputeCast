package engine

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/sending"
)

type liveInput struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid" doc:"Campaign id"`
}

type liveStreamInput struct {
	api.WorkspaceInput
	CampaignID  string `path:"campaignId" format:"uuid" doc:"Campaign id"`
	LastEventID string `header:"Last-Event-ID" doc:"Resume after this event id (the snapshot version)"`
	// The browser can't set Last-Event-ID on a new EventSource, so a client
	// that reconnects by itself passes it in the query.
	LastEventIDQuery string `query:"lastEventId" doc:"Same as the Last-Event-ID header, for clients that reconnect by themselves"`
	// An EventSource can't set Accept-Language either: the app passes its
	// language here, or the snapshots read in the browser's.
	Lang string `query:"lang" doc:"The language the snapshots' words read in (en, vi); like Accept-Language, for clients that can't set headers"`
}

// LivePauseRequest pauses a running send.
type LivePauseRequest struct {
	Reason string `json:"reason,omitempty" maxLength:"500"`
}

// LiveResumeRequest resumes a paused send, or one mailbox provider or
// sending provider on hold.
type LiveResumeRequest struct {
	Mailbox    string `json:"mailbox,omitempty" enum:"gmail,outlook,yahoo,icloud,other"`
	ProviderID string `json:"providerId,omitempty" format:"uuid"`
	// WithChanges applies the incident's recommended changes (exclude the
	// group, lower the rate, hand the excluded contacts to the flow).
	WithChanges bool `json:"withChanges,omitempty"`
	// Limit raises the limit of the metric that tripped, for this send
	// only, in percent ("Resume with the threshold raised to 0.20%").
	Limit *float64 `json:"limit,omitempty" minimum:"0.01" maximum:"100"`
	// Rate slows the resumed mailbox provider or route down for the rest
	// of the send, in messages per second ("Resume at 2 /s").
	Rate *float64 `json:"rate,omitempty" minimum:"0.1" maximum:"10000"`
}

// LiveAcknowledgeRequest marks an alert as read (complaints after the send).
type LiveAcknowledgeRequest struct {
	IncidentID string `json:"incidentId" format:"uuid"`
}

// LiveRatesRequest changes a running or waiting send's rates per mailbox
// provider for the rest of the send, on every route (review GS-13).
type LiveRatesRequest struct {
	// Rates in messages per second by mailbox provider (gmail, outlook,
	// yahoo, icloud, other); 0 is the sending providers' full rate. A
	// mailbox provider left out keeps its rate.
	Rates map[string]float64 `json:"rates" minProperties:"1" maxProperties:"5"`
}

// LiveThresholdsRequest changes a running or waiting send's auto-pause
// thresholds, in percent; a field left out keeps its value.
type LiveThresholdsRequest struct {
	Bounce           *float64 `json:"bounce,omitempty" minimum:"0.1" maximum:"20"`
	Complaint        *float64 `json:"complaint,omitempty" minimum:"0.01" maximum:"5"`
	MailboxComplaint *float64 `json:"mailboxComplaint,omitempty" minimum:"0.01" maximum:"5"`
	Deferral         *float64 `json:"deferral,omitempty" minimum:"1" maximum:"100"`
	APIErrors        *float64 `json:"apiErrors,omitempty" minimum:"1" maximum:"100"`
	// Scope: pause only the affected route or mailbox provider, or the whole campaign.
	Scope string `json:"scope,omitempty" enum:"route,campaign"`
}

// LiveCancelRequest cancels the whole send, or one share of it.
type LiveCancelRequest struct {
	Mailbox    string `json:"mailbox,omitempty" enum:"gmail,outlook,yahoo,icloud,other"`
	ProviderID string `json:"providerId,omitempty" format:"uuid"`
}

// LiveMoveRequest moves a held share to another sending provider.
type LiveMoveRequest struct {
	FromProviderID string `json:"fromProviderId" format:"uuid"`
	ToProviderID   string `json:"toProviderId" format:"uuid"`
}

// LiveRetryRequest retries a held share on its provider.
type LiveRetryRequest struct {
	ProviderID string `json:"providerId" format:"uuid"`
}

// LiveWinnerRequest picks the A/B winner.
type LiveWinnerRequest struct {
	Variant string `json:"variant" enum:"A,B,C"`
}

// LiveInDoubtRequest decides on the messages in doubt: send them again or
// leave them out, all of them or one sending provider's.
type LiveInDoubtRequest struct {
	Action     string `json:"action" enum:"resend,leave_out"`
	ProviderID string `json:"providerId,omitempty" format:"uuid"`
}

// LiveActionResult is what an action changed, with the new snapshot.
type LiveActionResult struct {
	Moved     int64 `json:"moved"`
	Cancelled int64 `json:"cancelled"`
	Excluded  int64 `json:"excluded"`
	// Decided is how many messages in doubt were sent again or left out.
	Decided  int64        `json:"decided"`
	Snapshot LiveSnapshot `json:"snapshot"`
}

// body is an action's input: the campaign's path and a JSON body. Huma reads
// parameters only from exported fields, so the path fields are spelled out
// here rather than embedded from the unexported liveInput.
type body[T any] struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid" doc:"Campaign id"`
	Body       T
}

// Register adds the live monitor's operations.
func Register(h huma.API, e *Engine) {
	RegisterTraffic(h, e)
	tags := []string{"Live sending"}
	base := "/workspaces/{workspace}/campaigns/{campaignId}/live"
	api.Get(h, api.Op{ID: "get-live-send", Path: base, Summary: "The live monitor's snapshot of a send", Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *liveInput) (*api.Body[LiveSnapshot], error) {
			id, err := uuid.Parse(in.CampaignID)
			if err != nil {
				return nil, api.NotFound("Campaign")
			}
			ws := api.WorkspaceFrom(ctx)
			s, err := e.Snapshot(ctx, ws.ID, id)
			if err != nil {
				return nil, mapErr(ctx, err)
			}
			s.Helpers = &LiveHelpers{Pause: access.Helpers(ctx, e.Q, ws.ID, authz.CampaignsPause), Send: access.Helpers(ctx, e.Q, ws.ID, authz.CampaignsSend)}
			return api.OK(s.In(i18n.LangFrom(ctx))), nil
		})
	sse.Register(h, huma.Operation{
		OperationID: "stream-live-send", Method: http.MethodGet, Path: api.Version + base + "/stream",
		Summary:     "Server-sent events of a send: a snapshot event at once and on every change, heartbeats as comments",
		Description: "Each `snapshot` event carries the whole live state; its id is the snapshot version, so a reconnecting EventSource resumes with Last-Event-ID.",
		Tags:        tags, Security: []map[string][]string{{"session": {}}, {"bearer": {}}},
		Metadata:   map[string]any{"reputecast.access": api.Workspace(authz.CampaignsView)},
		Extensions: map[string]any{"x-access": string(api.AccessWorkspace), "x-permission": string(authz.CampaignsView)},
	}, map[string]any{"snapshot": LiveSnapshot{}}, func(ctx context.Context, in *liveStreamInput, send sse.Sender) {
		id, err := uuid.Parse(in.CampaignID)
		if err != nil {
			return
		}
		ws := api.WorkspaceFrom(ctx)
		if ws == nil {
			return
		}
		if _, err := e.Snapshot(ctx, ws.ID, id); err != nil {
			return
		}
		// One snapshot is computed for every watcher; each reads it in their
		// own language.
		lang := i18n.LangFrom(ctx)
		if in.Lang != "" {
			lang = api.RequestLang(in.Lang, api.UserFrom(ctx))
		}
		updates, cancel := e.Subscribe(ctx, ws.ID, id)
		defer cancel()
		resumed := in.LastEventID != "" || in.LastEventIDQuery != ""
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()
		_ = send(sse.Message{Retry: 3000, Comment: "connected"})
		for {
			select {
			case <-ctx.Done():
				return
			case s := <-updates:
				s = s.In(lang)
				s.Resumed = resumed
				resumed = false
				if err := send(sse.Message{ID: int(s.Version), Data: s}); err != nil {
					return
				}
			case <-heartbeat.C:
				if err := send.Comment("heartbeat " + strconv.FormatInt(time.Now().Unix(), 10)); err != nil {
					return
				}
			}
		}
	})
	api.Post(h, api.Op{ID: "pause-live-send", Path: base + "/pause", Summary: "Pause a running send", Tags: tags, Access: api.Workspace(authz.CampaignsPause)},
		func(ctx context.Context, in *body[LivePauseRequest]) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			if err := e.Pause(ctx, id, by, in.Body.Reason); err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{})
		})
	api.Post(h, api.Op{ID: "resume-live-send", Path: base + "/resume", Summary: "Resume a paused send, or a mailbox provider or sending provider on hold", Tags: tags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *body[LiveResumeRequest]) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			req := ResumeRequest{Mailbox: in.Body.Mailbox, WithChanges: in.Body.WithChanges, Limit: in.Body.Limit}
			if in.Body.Rate != nil && !in.Body.WithChanges {
				req.Changes = &ResumeChanges{Mailbox: in.Body.Mailbox, Rate: *in.Body.Rate}
			}
			if in.Body.ProviderID != "" {
				pid, err := uuid.Parse(in.Body.ProviderID)
				if err != nil {
					return nil, api.Invalid("body.providerId", say(ctx, "problem.providerId", nil))
				}
				req.ProviderID = &pid
			}
			res, err := e.ResumeScope(ctx, id, by, req)
			if err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{Excluded: res.Excluded})
		})
	api.Put(h, api.Op{ID: "set-live-thresholds", Path: base + "/thresholds", Summary: "Change a running or waiting send's auto-pause thresholds", Tags: tags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *body[LiveThresholdsRequest]) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			if err := e.SetThresholds(ctx, id, by, ThresholdChange(in.Body)); err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{})
		})
	api.Put(h, api.Op{ID: "set-live-rates", Path: base + "/rates", Summary: "Change a running or waiting send's rates per mailbox provider", Tags: tags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *body[LiveRatesRequest]) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			for _, k := range sortedKeys(in.Body.Rates) {
				if key, args := rateProblem(k, in.Body.Rates[k]); key != "" {
					return nil, api.Invalid("body.rates."+k, say(ctx, key, args))
				}
			}
			if err := e.SetRates(ctx, id, by, in.Body.Rates); err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{})
		})
	api.Post(h, api.Op{ID: "acknowledge-live-alert", Path: base + "/acknowledge", Summary: "Mark an alert of a send as read (complaints after the send)", Tags: tags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *body[LiveAcknowledgeRequest]) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			iid, err := uuid.Parse(in.Body.IncidentID)
			if err != nil {
				return nil, api.Invalid("body.incidentId", say(ctx, "problem.incidentId", nil))
			}
			if err := e.Acknowledge(ctx, id, by, iid); err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{})
		})
	api.Post(h, api.Op{ID: "cancel-live-send", Path: base + "/cancel", Summary: "Cancel a send, or one mailbox provider's or sending provider's share", Tags: tags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *body[LiveCancelRequest]) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			var pid *uuid.UUID
			if in.Body.ProviderID != "" {
				p, err := uuid.Parse(in.Body.ProviderID)
				if err != nil {
					return nil, api.Invalid("body.providerId", say(ctx, "problem.providerId", nil))
				}
				pid = &p
			}
			n, err := e.CancelShare(ctx, id, by, in.Body.Mailbox, pid)
			if err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{Cancelled: n})
		})
	api.Post(h, api.Op{ID: "move-live-share", Path: base + "/move", Summary: "Move what is left of a sending provider's share to another provider", Tags: tags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *body[LiveMoveRequest]) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			from, err1 := uuid.Parse(in.Body.FromProviderID)
			to, err2 := uuid.Parse(in.Body.ToProviderID)
			if err1 != nil || err2 != nil {
				return nil, api.Invalid("body", say(ctx, "problem.pickProviders", nil))
			}
			n, err := e.MoveShare(ctx, id, by, from, to)
			if err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{Moved: n})
		})
	api.Post(h, api.Op{ID: "retry-live-share", Path: base + "/retry", Summary: "Keep a held share on its provider and retry every minute", Tags: tags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *body[LiveRetryRequest]) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			pid, err := uuid.Parse(in.Body.ProviderID)
			if err != nil {
				return nil, api.Invalid("body.providerId", say(ctx, "problem.providerId", nil))
			}
			if err := e.RetryRoute(ctx, id, by, pid); err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{})
		})
	api.Post(h, api.Op{ID: "resolve-live-in-doubt", Path: base + "/in-doubt", Summary: "Send the messages in doubt again, or leave them out", Tags: tags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *body[LiveInDoubtRequest]) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			var pid *uuid.UUID
			if in.Body.ProviderID != "" {
				p, err := uuid.Parse(in.Body.ProviderID)
				if err != nil {
					return nil, api.Invalid("body.providerId", say(ctx, "problem.providerId", nil))
				}
				pid = &p
			}
			n, err := e.ResolveInDoubt(ctx, id, by, in.Body.Action == "resend", pid)
			if err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{Decided: n})
		})
	api.Post(h, api.Op{ID: "pick-live-ab-winner", Path: base + "/ab-winner", Summary: "End the A/B test with a chosen winner", Tags: tags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *body[LiveWinnerRequest]) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			if err := e.PickWinner(ctx, id, by, in.Body.Variant); err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{})
		})
}

func (e *Engine) result(ctx context.Context, id uuid.UUID, r LiveActionResult) (*api.Body[LiveActionResult], error) {
	s, err := e.Snapshot(ctx, api.WorkspaceFrom(ctx).ID, id)
	if err != nil {
		return nil, mapErr(ctx, err)
	}
	r.Snapshot = s.In(i18n.LangFrom(ctx))
	return api.OK(r), nil
}

// actionIDs checks the campaign belongs to the workspace and returns who
// acts; another workspace's campaign is a 404, as if it didn't exist.
func (e *Engine) actionIDs(ctx context.Context, raw string) (uuid.UUID, uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, uuid.Nil, api.NotFound("Campaign")
	}
	ws := api.WorkspaceFrom(ctx)
	owner, err := e.Q.SendingCampaignWorkspace(ctx, id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, err
	}
	if err != nil || ws == nil || owner != ws.ID {
		return uuid.Nil, uuid.Nil, api.NotFound("Campaign")
	}
	p := api.PrincipalFrom(ctx)
	by := uuid.Nil
	if p != nil {
		by = p.ID
	}
	return id, by, nil
}

// mapErr turns the engine's errors into problems in the request's language.
func mapErr(ctx context.Context, err error) error {
	var se *StateError
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNotFound):
		return api.NotFound("Campaign")
	case errors.As(err, &se):
		return api.Conflict(se.In(i18n.LangFrom(ctx)))
	case errors.Is(err, sending.ErrNotReady):
		var nr *sending.NotReadyError
		if errors.As(err, &nr) && nr.Reason != "" {
			// The blocking check's title is Send Readiness's own words.
			return api.Conflict(say(ctx, "problem.readinessBlocked", i18n.Args{"reason": strings.TrimSuffix(nr.Reason, ".")}))
		}
		return api.Conflict(say(ctx, "problem.readinessBlockedAny", nil))
	}
	return err
}
