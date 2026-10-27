package readiness

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// ReadinessCampaignPath is the {workspace}/{campaignId} of readiness operations.
type ReadinessCampaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid"`
}

// ID parses the campaign id (404 when it isn't one).
func (in ReadinessCampaignPath) ID() (uuid.UUID, error) {
	id, err := uuid.Parse(in.CampaignID)
	if err != nil {
		return uuid.Nil, api.NotFound("Campaign")
	}
	return id, nil
}

type runPathInput struct {
	ReadinessCampaignPath
	RunID string `path:"runId" format:"uuid"`
}

type fixesInput struct {
	ReadinessCampaignPath
	Body ReadinessFixesInput
}

type settingsInput struct {
	ReadinessCampaignPath
	Body ReadinessSendSettings
}

// ReadinessFixOutcome is the result of applying fixes and the run they
// started (in the background: poll …/readiness/progress).
type ReadinessFixOutcome struct {
	ReadinessFixResult
	Running *ReadinessProgress `json:"running,omitempty"`
}

// pollForReader is a poll in the request's language.
func pollForReader(ctx context.Context, p ReadinessPoll) ReadinessPoll {
	p.Running = ProgressForReader(ctx, p.Running)
	return p
}

// Register adds the readiness operations.
func Register(h huma.API, s *Service) {
	registerRules(h, s)
	tags := []string{"Review & send"}
	registerComposer(h, s, tags)
	api.Post(h, api.Op{ID: "run-send-readiness", Path: "/workspaces/{workspace}/campaigns/{campaignId}/readiness", Summary: "Run Send Readiness on a campaign",
		Description: "Queues a run of every check; a worker runs it in the background and stores the result. Answers 202 with the run waiting or in " +
			"progress; poll …/readiness/progress until running is gone and latest is the new run. Fails with 409 readiness_running while a run waits or runs.",
		Tags: tags, Access: api.Workspace(authz.CampaignsEdit), Status: 202},
		func(ctx context.Context, in *ReadinessCampaignPath) (*api.Body[ReadinessPoll], error) {
			id, err := in.ID()
			if err != nil {
				return nil, err
			}
			p := api.PrincipalFrom(ctx)
			if _, err := s.Start(ctx, api.WorkspaceFrom(ctx), id, RunOptions{Trigger: TriggerManual, By: &p.UserID}); err != nil {
				if errors.Is(err, ErrRunning) {
					return nil, runningProblem(ctx)
				}
				return nil, err
			}
			poll, err := s.Poll(ctx, id)
			return api.OK(pollForReader(ctx, poll)), err
		})
	api.Get(h, api.Op{ID: "get-send-readiness-progress", Path: "/workspaces/{workspace}/campaigns/{campaignId}/readiness/progress", Summary: "The Send Readiness run waiting or in progress",
		Description: "Cheap to poll: the run waiting or in progress with each group's progress, the latest stored run's id and verdict, and a run that failed.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *ReadinessCampaignPath) (*api.Body[ReadinessPoll], error) {
			id, err := in.ID()
			if err != nil {
				return nil, err
			}
			// One indexed read for the campaign's workspace: polling stays cheap.
			if ws, err := s.Q.ReadinessCampaignWorkspace(ctx, id); err != nil || ws != api.WorkspaceFrom(ctx).ID {
				return nil, api.NotFound("Campaign")
			}
			poll, err := s.Poll(ctx, id)
			return api.OK(pollForReader(ctx, poll)), err
		})
	api.Post(h, api.Op{ID: "accept-unknown-readiness", Path: "/workspaces/{workspace}/campaigns/{campaignId}/readiness/{runId}/accept-unknown",
		Summary:     "Continue with provider data only",
		Description: "Accepts the checks that couldn't run (HostRepute unreachable). The skipped checks are recorded in the audit log and re-tried during the send.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.CampaignsEdit)},
		func(ctx context.Context, in *runPathInput) (*api.Body[ReadinessRun], error) {
			id, err := in.ID()
			if err != nil {
				return nil, err
			}
			runID, err := uuid.Parse(in.RunID)
			if err != nil {
				return nil, api.NotFound("Readiness run")
			}
			run, err := s.AcceptUnknown(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, runID)
			return api.OK(run.In(i18n.LangFrom(ctx))), err
		})
	api.Post(h, api.Op{ID: "apply-readiness-fixes", Path: "/workspaces/{workspace}/campaigns/{campaignId}/readiness/fixes", Summary: "Apply one-click fixes, then run the checks again",
		Description: "Content fixes save a new version; audience and route fixes change the campaign. Changing route needs campaigns:send. Changes invalidate an approval.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.CampaignsEdit)},
		func(ctx context.Context, in *fixesInput) (*api.Body[ReadinessFixOutcome], error) {
			id, err := in.ID()
			if err != nil {
				return nil, err
			}
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			res, err := s.ApplyFixes(ctx, ws, p, id, in.Body)
			if err != nil {
				return nil, err
			}
			// The checks run again in the background; sending stays locked
			// until they finish.
			out := ReadinessFixOutcome{ReadinessFixResult: res}
			running, err := s.Start(ctx, ws, id, RunOptions{Trigger: TriggerManual, By: &p.UserID})
			switch {
			case err == nil:
				out.Running = ProgressForReader(ctx, running)
			case errors.Is(err, ErrRunning):
				// A run already waits or runs: it reads the campaign as fixed.
				if pr, err := s.Progress(ctx, id); err == nil {
					out.Running = ProgressForReader(ctx, pr)
				}
			default:
				return nil, err
			}
			return api.OK(out), nil
		})
	api.Patch(h, api.Op{ID: "update-send-settings", Path: "/workspaces/{workspace}/campaigns/{campaignId}/send-settings", Summary: "Change the auto-pause thresholds or the schedule",
		Description: "The schedule is part of what an approval covers: changing it invalidates the approval.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsEdit), Status: 204},
		func(ctx context.Context, in *settingsInput) (*api.Empty, error) {
			id, err := in.ID()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.UpdateSendSettings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
		})
}
