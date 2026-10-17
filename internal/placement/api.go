package placement

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

type testInput struct {
	api.WorkspaceInput
	Test string `path:"test" format:"uuid"`
}

type copyInput struct {
	api.WorkspaceInput
	Test string `path:"test" format:"uuid"`
	Copy string `path:"copy" format:"uuid"`
}

type runInput struct {
	api.WorkspaceInput
	Body PlacementRunInput
}

type contentInput struct {
	api.WorkspaceInput
	Q string `query:"q" maxLength:"100" doc:"Words of the name"`
}

type seedInput struct {
	api.WorkspaceInput
	Seed string `path:"seed" format:"uuid"`
}

type addSeedInput struct {
	api.WorkspaceInput
	Body PlacementSeedInput
}

// Inputs list their path parameters themselves: Huma can't set the fields
// of an unexported embedded struct.
type reconnectInput struct {
	api.WorkspaceInput
	Seed string `path:"seed" format:"uuid"`
	Body PlacementReconnectInput
}

type oauthInput struct {
	api.WorkspaceInput
	Body PlacementOAuthInput
}

type settingsInput struct {
	api.WorkspaceInput
	Body PlacementSettingsInput
}

type callbackInput struct {
	Code             string `query:"code" maxLength:"4096"`
	State            string `query:"state" maxLength:"512"`
	Error            string `query:"error" maxLength:"200"`
	ErrorDescription string `query:"error_description" maxLength:"1000"`
}

type redirectOutput struct {
	Status       int
	Location     string `header:"Location"`
	CacheControl string `header:"Cache-Control"`
}

// PlacementContentOption is something a test can send.
type PlacementContentOption struct {
	Kind string    `json:"kind" enum:"campaign,template,digest,transactional"`
	ID   string    `json:"id"`
	Name string    `json:"name"`
	Note string    `json:"note" doc:"A campaign's status, template, a transactional template's key, or a digest's state (on, paused, off)"`
	At   time.Time `json:"at"`
}

// PlacementContentList is the dialog's content search.
type PlacementContentList struct {
	Items []PlacementContentOption `json:"items"`
}

func id(raw, what string) (uuid.UUID, error) {
	u, err := uuid.Parse(raw)
	if err != nil {
		return u, api.NotFound(what)
	}
	return u, nil
}

// SearchContent finds campaigns, templates and transactional templates to test.
func (s *Service) SearchContent(ctx context.Context, ws *store.Workspace, q string) (PlacementContentList, error) {
	out := PlacementContentList{Items: []PlacementContentOption{}}
	rows, err := s.Q.PlacementContentSearch(ctx, store.PlacementContentSearchParams{WorkspaceID: ws.ID, Q: q})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Items = append(out.Items, PlacementContentOption{Kind: r.Kind, ID: r.ID.String(), Name: r.Name, Note: r.Note, At: r.At})
	}
	return out, nil
}

// RunAgain starts a new test with an earlier test's content and choices,
// to every seed connected now.
func (s *Service) RunAgain(ctx context.Context, ws *store.Workspace, testID uuid.UUID) (PlacementTestDetail, error) {
	t, err := s.Q.PlacementTestByID(ctx, testID)
	if err != nil || t.WorkspaceID != ws.ID {
		return PlacementTestDetail{}, api.NotFound("Placement test")
	}
	in := PlacementRunInput{Content: PlacementContentRef{Kind: t.ContentKind}, Variant: t.Variant, Language: t.Language, TestData: t.Personalization,
		Through: t.Through, Origin: "placement"}
	if t.ContentID != nil {
		in.Content.ID = t.ContentID.String()
	}
	for _, p := range t.ProviderIds {
		in.Providers = append(in.Providers, p.String())
	}
	if t.SenderID != nil {
		in.SenderID = t.SenderID.String()
	}
	return s.Start(ctx, ws, in)
}

// Register adds the placement test operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Placement tests"}
	base := "/workspaces/{workspace}/placement"

	api.Get(h, api.Op{ID: "list-placement-tests", Path: base + "/tests", Summary: "Placement tests, newest first, with results per provider",
		Tags: tags, Access: api.Workspace(authz.DeliverabilityView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[PlacementTestList], error) {
			out, err := s.Tests(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-placement-test", Path: base + "/tests/{test}", Summary: "One placement test: its copies, the matrix and the guidance",
		Tags: tags, Access: api.Workspace(authz.DeliverabilityView)},
		func(ctx context.Context, in *testInput) (*api.Body[PlacementTestDetail], error) {
			tid, err := id(in.Test, "Placement test")
			if err != nil {
				return nil, err
			}
			out, err := s.Test(ctx, api.WorkspaceFrom(ctx), tid)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-placement-copy", Path: base + "/tests/{test}/copies/{copy}", Summary: "One copy: where it landed, its headers and the provider's events",
		Tags: tags, Access: api.Workspace(authz.DeliverabilityView)},
		func(ctx context.Context, in *copyInput) (*api.Body[PlacementCopyDetail], error) {
			tid, err := id(in.Test, "Placement test")
			if err != nil {
				return nil, err
			}
			cid, err := id(in.Copy, "Copy")
			if err != nil {
				return nil, err
			}
			out, err := s.CopyDetail(ctx, api.WorkspaceFrom(ctx), tid, cid)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "search-placement-content", Path: base + "/content", Summary: "Campaigns, templates and transactional templates a test can send",
		Tags: tags, Access: api.Workspace(authz.PlacementRun)},
		func(ctx context.Context, in *contentInput) (*api.Body[PlacementContentList], error) {
			out, err := s.SearchContent(ctx, api.WorkspaceFrom(ctx), in.Q)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "plan-placement-test", Path: base + "/plan", Summary: "What a test would send: providers, senders, seeds, copies and cost (nothing is sent)",
		Tags: tags, Access: api.Workspace(authz.PlacementRun)},
		func(ctx context.Context, in *runInput) (*api.Body[PlacementPlan], error) {
			out, err := s.Plan(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "start-placement-test", Path: base + "/tests", Summary: "Start a placement test: one copy to each seed (audited)",
		Description: "Answers 409 placement_not_ready with the problems (meta.problems) when the plan can't start. People only: agents can't hold placement:run.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.PlacementRun), Status: http.StatusCreated},
		func(ctx context.Context, in *runInput) (*api.Body[PlacementTestDetail], error) {
			out, err := s.Start(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "stop-placement-test", Path: base + "/tests/{test}/stop", Summary: "Stop waiting: copies not found yet are Not arrived (audited)",
		Tags: tags, Access: api.WorkspaceHuman(authz.PlacementRun)},
		func(ctx context.Context, in *testInput) (*api.Body[PlacementTestDetail], error) {
			tid, err := id(in.Test, "Placement test")
			if err != nil {
				return nil, err
			}
			out, err := s.Stop(ctx, api.WorkspaceFrom(ctx), tid)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "delete-placement-copies", Path: base + "/tests/{test}/delete-copies", Summary: "Delete a test's copies from the seeds now, by test token (audited)",
		Tags: tags, Access: api.WorkspaceHuman(authz.PlacementRun)},
		func(ctx context.Context, in *testInput) (*api.Body[PlacementTestDetail], error) {
			tid, err := id(in.Test, "Placement test")
			if err != nil {
				return nil, err
			}
			out, err := s.DeleteCopiesNow(ctx, api.WorkspaceFrom(ctx), tid)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "rerun-placement-test", Path: base + "/tests/{test}/run-again", Summary: "Run a test again with the same content and choices (audited)",
		Tags: tags, Access: api.WorkspaceHuman(authz.PlacementRun), Status: http.StatusCreated},
		func(ctx context.Context, in *testInput) (*api.Body[PlacementTestDetail], error) {
			tid, err := id(in.Test, "Placement test")
			if err != nil {
				return nil, err
			}
			out, err := s.RunAgain(ctx, api.WorkspaceFrom(ctx), tid)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "list-seed-mailboxes", Path: base + "/seeds", Summary: "Seed mailboxes (passwords and tokens are never returned)",
		Tags: tags, Access: api.Workspace(authz.DeliverabilityView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[PlacementSeedList], error) {
			out, err := s.Seeds(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-seed-mailbox", Path: base + "/seeds", Summary: "Add a seed mailbox with an IMAP password, after checking the sign-in (audited)",
		Description: "The password is sealed at once and never returned. Answers 422 with meta.kind refused or unreachable when the sign-in fails.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.PlacementManage), Status: http.StatusCreated},
		func(ctx context.Context, in *addSeedInput) (*api.Body[PlacementSeedCheck], error) {
			out, err := s.AddSeed(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "reconnect-seed-mailbox", Path: base + "/seeds/{seed}/reconnect", Summary: "Reconnect a seed with a new IMAP password (audited)",
		Tags: tags, Access: api.WorkspaceHuman(authz.PlacementManage)},
		func(ctx context.Context, in *reconnectInput) (*api.Body[PlacementSeedCheck], error) {
			sid, err := id(in.Seed, "Seed mailbox")
			if err != nil {
				return nil, err
			}
			out, err := s.Reconnect(ctx, api.WorkspaceFrom(ctx), sid, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-seed-mailbox", Path: base + "/seeds/{seed}", Summary: "Remove a seed and its password or tokens (audited)",
		Tags: tags, Access: api.WorkspaceHuman(authz.PlacementManage)},
		func(ctx context.Context, in *seedInput) (*api.Empty, error) {
			sid, err := id(in.Seed, "Seed mailbox")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.RemoveSeed(ctx, api.WorkspaceFrom(ctx), sid)
		})
	api.Post(h, api.Op{ID: "start-seed-oauth", Path: base + "/seeds/oauth", Summary: "Start Google's or Microsoft's consent for a seed (OAuth with PKCE)",
		Description: "Returns the consent URL. The browser comes back to " + CallbackPath + ", which adds or reconnects the seed.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.PlacementManage)},
		func(ctx context.Context, in *oauthInput) (*api.Body[PlacementOAuthStart], error) {
			out, err := s.StartOAuth(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-placement-settings", Path: base + "/settings", Summary: "When copies are deleted from the seeds (audited)",
		Tags: tags, Access: api.Workspace(authz.PlacementManage)},
		func(ctx context.Context, in *settingsInput) (*api.Body[PlacementSettingsInput], error) {
			out, err := s.SetSettings(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	// One callback per instance: the single-use state names the workspace
	// and the person, and the handler checks placement:manage there.
	api.Get(h, api.Op{ID: "seed-oauth-callback", Path: "/placement/oauth/callback", Summary: "OAuth return from Google or Microsoft for a seed (browser redirect)",
		Description: "Exchanges the code, checks the mailbox, seals the tokens and redirects to Seed mailboxes. Requires the signed-in person who started it, holding placement:manage in its workspace.",
		Tags:        tags, Access: api.Access{Kind: api.AccessPublic, Permission: authz.PlacementManage}},
		func(ctx context.Context, in *callbackInput) (*redirectOutput, error) {
			self := CallbackPath + "?" + url.Values{"code": {in.Code}, "state": {in.State}, "error": {in.Error}}.Encode()
			loc := s.OAuthCallback(ctx, api.UserFrom(ctx), self, in.State, in.Code, in.Error, in.ErrorDescription)
			return &redirectOutput{Status: http.StatusFound, Location: loc, CacheControl: "no-store"}, nil
		})
}

// RegisterJobs adds the workers and the periodic jobs.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &SendWorker{Svc: s})
	river.AddWorker(reg.Workers, &CheckWorker{Svc: s})
	river.AddWorker(reg.Workers, &DeleteWorker{Svc: s})
	reg.Every(30*time.Second, func() (river.JobArgs, *river.InsertOpts) { return CheckArgs{}, nil }, true)
	reg.Every(5*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return DeleteArgs{}, nil }, false)
}
