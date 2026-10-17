package digests

import (
	"context"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
)

// DigestPath is the {workspace}/{digestId} of digest operations.
type DigestPath struct {
	api.WorkspaceInput
	DigestID string `path:"digestId" format:"uuid"`
}

func (in DigestPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.DigestID)
	if err != nil {
		return uuid.Nil, api.NotFound("Digest")
	}
	return id, nil
}

// IssuePath is the {workspace}/{digestId}/{issueId} of issue operations.
type IssuePath struct {
	DigestPath
	IssueID string `path:"issueId" format:"uuid"`
}

func (in IssuePath) ids() (uuid.UUID, uuid.UUID, error) {
	d, err := in.id()
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	i, err := uuid.Parse(in.IssueID)
	if err != nil {
		return uuid.Nil, uuid.Nil, api.NotFound("Issue")
	}
	return d, i, nil
}

type createInput struct {
	api.WorkspaceInput
	Body DigestCreateInput
}

type settingsInput struct {
	DigestPath
	Body DigestSettingsInput
}

type turnOnInput struct {
	DigestPath
	Body DigestTurnOnInput
}

type pauseInput struct {
	DigestPath
	Body DigestPauseInput
}

type findInput struct {
	api.WorkspaceInput
	Body DigestFindInput
}

type feedInput struct {
	DigestPath
	Limit int `query:"limit" minimum:"1" maximum:"50" default:"6"`
}

type historyInput struct {
	DigestPath
	Limit int `query:"limit" minimum:"1" maximum:"30" default:"5"`
}

type skipInput struct {
	IssuePath
	Body DigestSkipInput
}

type renderInput struct {
	DigestPath
	Body DigestRenderInput
}

type testInput struct {
	DigestPath
	Body DigestTestInput
}

type infosInput struct {
	api.WorkspaceInput
	Campaigns string `query:"campaigns" maxLength:"4000" doc:"Campaign ids, comma-separated (up to 100)"`
}

// DigestReadNow is a read of the feed started by a person.
type DigestReadNow struct {
	OK          bool             `json:"ok"`
	NotModified bool             `json:"notModified"`
	Items       int              `json:"items"`
	New         int              `json:"new"`
	Error       *DigestFeedError `json:"error,omitempty"`
}

// Register adds the digest operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Digests"}
	view, edit := api.Workspace(authz.CampaignsView), api.Workspace(authz.CampaignsEdit)
	base := "/workspaces/{workspace}/digests"
	one := base + "/{digestId}"
	api.Get(h, api.Op{ID: "list-digests", Path: base, Summary: "Recurring digests: status, schedule, next issue, the last issue's human opens and the owner",
		Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[DigestList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-digest", Path: base, Summary: "Create a recurring digest from a feed or a connected WordPress site",
		Description: "The digest starts off, with the Blog digest template, the workspace's default sender and its time zone. Without campaigns:send it starts in draft mode.",
		Tags:        tags, Access: edit, Status: 201},
		func(ctx context.Context, in *createInput) (*api.Body[DigestDetail], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "find-digest-feed", Path: base + "/find", Summary: "Find and read the feed of a feed or page address",
		Description: "Reads the address through the outbound guard (no private networks, three redirects at most, 5 MB and 15 seconds), and finds the feed a page links to.",
		Tags:        tags, Access: edit},
		func(ctx context.Context, in *findInput) (*api.Body[DigestFound], error) {
			out, err := s.Find(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-digest-sites", Path: base + "/sites", Summary: "Connected WordPress sites a digest can read, with their feeds",
		Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[DigestSites], error) {
			out, err := s.Sites(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-digest-issue-infos", Path: "/workspaces/{workspace}/digest-issues", Summary: "Which campaigns are digest issues, with their posts and what changed since the last approved issue",
		Tags: tags, Access: view},
		func(ctx context.Context, in *infosInput) (*api.Body[DigestIssueInfos], error) {
			ids := []uuid.UUID{}
			for _, raw := range strings.Split(in.Campaigns, ",") {
				if id, err := uuid.Parse(strings.TrimSpace(raw)); err == nil && len(ids) < 100 {
					ids = append(ids, id)
				}
			}
			out, err := s.IssueInfos(ctx, api.WorkspaceFrom(ctx), ids)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-digest", Path: one, Summary: "A digest: its feed, filters, schedule, audience, route, approvals, owner and next issue",
		Tags: tags, Access: view},
		func(ctx context.Context, in *DigestPath) (*api.Body[DigestDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Detail(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-digest", Path: one, Summary: "Save a digest's settings",
		Description: "Changes apply from the next preparation; an issue already prepared keeps its content. Switching to Send automatically while it is on needs campaigns:send.",
		Tags:        tags, Access: edit},
		func(ctx context.Context, in *settingsInput) (*api.Body[DigestDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Update(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-digest", Path: one, Summary: "Delete a digest that isn't on (its sent issues stay as campaigns)",
		Tags: tags, Access: api.WorkspaceHuman(authz.CampaignsEdit), Status: 204},
		func(ctx context.Context, in *DigestPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Delete(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "turn-on-digest", Path: one + "/turn-on", Summary: "Turn a digest on, or back on after a pause",
		Description: "Send automatically needs campaigns:send; Create a draft needs campaigns:edit. Whoever turns it on becomes its owner: issues go out on their authority. People only: API keys and agents can't turn a digest on.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.CampaignsEdit)},
		func(ctx context.Context, in *turnOnInput) (*api.Body[DigestDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.TurnOn(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "pause-digest", Path: one + "/pause", Summary: "Pause a digest: no new issues are prepared",
		Description: "An issue waiting to go out is skipped and its posts stay new; one that is sending pauses like any send.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsPause)},
		func(ctx context.Context, in *pauseInput) (*api.Body[DigestDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Pause(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "read-digest-feed", Path: one + "/read", Summary: "Read the digest's feed now",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *DigestPath) (*api.Body[DigestReadNow], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			ws := api.WorkspaceFrom(ctx)
			res, err := s.Read(ctx, ws, id, ViaPerson)
			if err != nil {
				return nil, err
			}
			out := DigestReadNow{OK: res.OK, NotModified: res.NotModified, Items: res.Items, New: res.New}
			if res.Err != nil {
				fe := toFeedError(res.Err, "")
				out.Error = &fe
			}
			return api.OK(out), nil
		})
	api.Get(h, api.Op{ID: "get-digest-feed", Path: one + "/feed", Summary: "The feed's newest items, each with what the digest does with it",
		Tags: tags, Access: view},
		func(ctx context.Context, in *feedInput) (*api.Body[DigestFeedPreview], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.FeedPreview(ctx, api.WorkspaceFrom(ctx), id, in.Limit)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-digest-issues", Path: one + "/issues", Summary: "A digest's issues: sent, skipped, held, waiting and paused, each with its numbers",
		Tags: tags, Access: view},
		func(ctx context.Context, in *DigestPath) (*api.Body[DigestIssues], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Issues(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-digest-history", Path: one + "/history", Summary: "What happened since the last issue: posts that went in or were left out, issues, and who turned it on",
		Tags: tags, Access: view},
		func(ctx context.Context, in *historyInput) (*api.Body[DigestHistory], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.History(ctx, api.WorkspaceFrom(ctx), id, in.Limit)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "skip-digest-issue", Path: one + "/issues/{issueId}/skip", Summary: "Skip an issue: nothing goes out and its posts stay new",
		Tags: tags, Access: api.WorkspaceHuman(authz.CampaignsPause)},
		func(ctx context.Context, in *skipInput) (*api.Body[DigestIssues], error) {
			d, i, err := in.ids()
			if err != nil {
				return nil, err
			}
			out, err := s.Skip(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), d, i, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "release-digest-issue", Path: one + "/issues/{issueId}/release", Summary: "Release a held issue once its block is fixed",
		Description: "Send Readiness runs again and must not be Blocked; the issue then waits for its approval, if one is needed, and goes out at its time.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.CampaignsSend)},
		func(ctx context.Context, in *IssuePath) (*api.Body[DigestIssues], error) {
			d, i, err := in.ids()
			if err != nil {
				return nil, err
			}
			out, err := s.Release(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), d, i)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "rebuild-digest-issue", Path: one + "/issues/{issueId}/rebuild", Summary: "Build a prepared issue again from the current template, and check it again",
		Tags: tags, Access: api.WorkspaceHuman(authz.CampaignsEdit)},
		func(ctx context.Context, in *IssuePath) (*api.Body[DigestIssues], error) {
			d, i, err := in.ids()
			if err != nil {
				return nil, err
			}
			out, err := s.Rebuild(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), d, i)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "render-digest", Path: one + "/render", Summary: "Draw the Feed items block and the subject with a preview's posts (the composer in digest mode)",
		Tags: tags, Access: view, MaxBodyBytes: 4 << 20},
		func(ctx context.Context, in *renderInput) (*api.Body[DigestRender], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Render(ctx, api.WorkspaceFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "preview-digest", Path: one + "/preview", Summary: "The next issue as a reader would get it now",
		Tags: tags, Access: view},
		func(ctx context.Context, in *DigestPath) (*api.Body[DigestPreview], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Preview(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-digest-issue", Path: one + "/check", Summary: "Run Send Readiness on the next issue as it would be built now",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *DigestPath) (*api.Body[DigestCheck], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Check(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "test-digest-issue", Path: one + "/test", Summary: "Send the next issue, as it would be built now, as a test",
		Description: "Goes through the System route. Without campaigns:send, test emails can only go to your own address.",
		Tags:        tags, Access: edit},
		func(ctx context.Context, in *testInput) (*api.Body[campaigns.ComposerTestSendResult], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SendTest(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
}
