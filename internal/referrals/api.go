package referrals

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// ProgramInput is a program's path. The input types are exported so Huma
// finds their parameters when an operation's input embeds them.
type ProgramInput struct {
	api.WorkspaceInput
	Program string `path:"program" format:"uuid"`
}

func (in ProgramInput) id() uuid.UUID { return uuid.MustParse(in.Program) }

// MilestoneInput is a milestone's path.
type MilestoneInput struct {
	api.WorkspaceInput
	Milestone string `path:"milestone" format:"uuid"`
}

func (in MilestoneInput) id() uuid.UUID { return uuid.MustParse(in.Milestone) }

// RewardInput is a reward's path.
type RewardInput struct {
	api.WorkspaceInput
	Reward string `path:"reward" format:"uuid"`
}

func (in RewardInput) id() uuid.UUID { return uuid.MustParse(in.Reward) }

// ReferrerInput is a referrer's path: their program and link.
type ReferrerInput struct {
	ProgramInput
	Link string `path:"link" format:"uuid"`
}

// ReferralFile is a CSV download.
type ReferralCSV struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

// ReferralLeaderboardInput turns the leaderboard on or off.
type ReferralLeaderboardInput struct {
	On bool `json:"on"`
}

// ReferralLinkPauseInput pauses a referrer's link, with a note.
type ReferralLinkPauseInput struct {
	Note string `json:"note,omitempty" maxLength:"500"`
}

// ReferralEmailOnInput turns a milestone's email on or off.
type ReferralEmailOnInput struct {
	On bool `json:"on"`
}

// ReferralWebhookTestResult is a webhook reward's test call.
type ReferralWebhookTestResult struct {
	OK     bool   `json:"ok"`
	Status int    `json:"status"`
	Code   string `json:"code"`
	URL    string `json:"url"`
	Error  string `json:"error"`
}

// ReferralRemoved says how many unused codes were removed.
type ReferralRemoved struct {
	Removed int64 `json:"removed"`
}

// Register adds the referral operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Grow"}
	base := "/workspaces/{workspace}/grow/referrals"
	view, edit, review := api.Workspace(authz.GrowView), api.WorkspaceHuman(authz.GrowEdit), api.WorkspaceHuman(authz.GrowReview)

	api.Get(h, api.Op{ID: "list-referral-programs", Path: base, Summary: "The referral programs and the lists a new one can use", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[ReferralProgramList], error) {
			out, err := s.Programs(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-referral-program", Path: base, Summary: "Start a referral program on a list (off, at the guide's second step)", Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body ReferralCreateInput
		}) (*api.Body[ReferralProgramDetail], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-referral-program", Path: base + "/{program}", Summary: "A referral program with its settings", Tags: tags, Access: view},
		func(ctx context.Context, in *ProgramInput) (*api.Body[ReferralProgramDetail], error) {
			out, err := s.Detail(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-referral-settings", Path: base + "/{program}/settings", Summary: "Change the counting rules, pages, leaderboard and milestone emails' sender",
		Description: "Changes apply going forward: counted referrals and sent rewards never change.", Tags: tags, Access: edit},
		func(ctx context.Context, in *struct {
			ProgramInput
			Body ReferralSettingsInput
		}) (*api.Body[ReferralProgramDetail], error) {
			out, err := s.UpdateSettings(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "referral-settings-impact", Path: base + "/{program}/settings/impact", Summary: "What a change of the counting rules does to referrals not decided yet",
		Tags: tags, Access: view},
		func(ctx context.Context, in *struct {
			ProgramInput
			Body ReferralCounting
		}) (*api.Body[ReferralSettingsImpact], error) {
			out, err := s.Impact(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-referral-leaderboard", Path: base + "/{program}/leaderboard", Summary: "Turn the public leaderboard on or off", Tags: tags, Access: edit},
		func(ctx context.Context, in *struct {
			ProgramInput
			Body ReferralLeaderboardInput
		}) (*api.Body[ReferralProgramDetail], error) {
			out, err := s.SetLeaderboard(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body.On)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-referral-setup", Path: base + "/{program}/setup", Summary: "Keep the set-up guide's step (and the list, until the program has been on)", Tags: tags, Access: edit},
		func(ctx context.Context, in *struct {
			ProgramInput
			Body ReferralSetupInput
		}) (*api.Body[ReferralProgramDetail], error) {
			out, err := s.SaveSetup(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-referral-checks", Path: base + "/{program}/checks", Summary: "The checks before a program turns on", Tags: tags, Access: view},
		func(ctx context.Context, in *ProgramInput) (*api.Body[ReferralChecks], error) {
			out, err := s.Checks(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "turn-on-referral-program", Path: base + "/{program}/on", Summary: "Turn a program on: every confirmed subscriber of its list gets a link", Tags: tags, Access: edit},
		func(ctx context.Context, in *ProgramInput) (*api.Body[ReferralTurnedOn], error) {
			out, err := s.TurnOn(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "turn-off-referral-program", Path: base + "/{program}/off", Summary: "Turn a program off: links lead to the normal subscribe page and Referral blocks hide",
		Description: "Earned rewards stay available on readers' hubs; counted referrals and sent rewards are kept.", Tags: tags, Access: edit},
		func(ctx context.Context, in *ProgramInput) (*api.Body[ReferralProgramDetail], error) {
			out, err := s.TurnOff(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-referral-program", Path: base + "/{program}", Summary: "Delete a program that is off: its links stop for good", Tags: tags, Access: edit},
		func(ctx context.Context, in *ProgramInput) (*api.Empty, error) {
			return api.NoContent, s.Delete(ctx, api.WorkspaceFrom(ctx), in.id())
		})
	api.Get(h, api.Op{ID: "get-referral-results", Path: base + "/{program}/results", Summary: "A program's results: the funnel, confirmed referrals over time, channels, top referrers, rewards and quality",
		Tags: tags, Access: view},
		func(ctx context.Context, in *struct {
			ProgramInput
			Period   string `query:"period" enum:"7d,30d,all" default:"all"`
			Grouping string `query:"grouping" enum:"day,week" default:"week"`
		}) (*api.Body[ReferralResults], error) {
			out, err := s.Results(ctx, api.WorkspaceFrom(ctx), in.id(), in.Period, in.Grouping)
			return api.OK(out), err
		})

	// Milestones and rewards.
	api.Get(h, api.Op{ID: "list-referral-milestones", Path: base + "/{program}/milestones", Summary: "The milestones with their rewards, health and emails, the files and the To fulfill queue",
		Tags: tags, Access: view},
		func(ctx context.Context, in *ProgramInput) (*api.Body[ReferralMilestoneList], error) {
			out, err := s.Milestones(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-referral-milestone", Path: base + "/{program}/milestones", Summary: "Add a milestone with its reward and a milestone email",
		Description: "A Stripe coupon reward also needs earn:edit and Stripe connected.", Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			ProgramInput
			Body ReferralMilestoneInput
		}) (*api.Body[ReferralMilestoneView], error) {
			out, err := s.CreateMilestone(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	mbase := "/workspaces/{workspace}/grow/milestones/{milestone}"
	api.Get(h, api.Op{ID: "get-referral-milestone", Path: mbase, Summary: "A milestone with its reward", Tags: tags, Access: view},
		func(ctx context.Context, in *MilestoneInput) (*api.Body[ReferralMilestoneView], error) {
			out, err := s.MilestoneView(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-referral-milestone", Path: mbase, Summary: "Change a milestone and its reward (rewards given from now on)", Tags: tags, Access: edit},
		func(ctx context.Context, in *struct {
			MilestoneInput
			Body ReferralMilestoneInput
		}) (*api.Body[ReferralMilestoneView], error) {
			out, err := s.UpdateMilestone(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-referral-milestone", Path: mbase, Summary: "Delete a milestone: rewards already sent stay with their referrers", Tags: tags, Access: edit},
		func(ctx context.Context, in *MilestoneInput) (*api.Empty, error) {
			return api.NoContent, s.DeleteMilestone(ctx, api.WorkspaceFrom(ctx), in.id())
		})
	api.Post(h, api.Op{ID: "upload-referral-codes", Path: mbase + "/codes", Summary: "Upload codes for a code reward (one per line); only their count is logged",
		Tags: tags, Access: edit, MaxBodyBytes: 6 << 20},
		func(ctx context.Context, in *struct {
			MilestoneInput
			Body ReferralCodesInput
		}) (*api.Body[ReferralCodesResult], error) {
			out, err := s.UploadCodes(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-referral-codes", Path: mbase + "/codes", Summary: "Remove the codes nobody has received yet", Tags: tags, Access: edit, Status: http.StatusOK},
		func(ctx context.Context, in *MilestoneInput) (*api.Body[ReferralRemoved], error) {
			n, err := s.RemoveCodes(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(ReferralRemoved{Removed: n}), err
		})
	api.Post(h, api.Op{ID: "test-referral-webhook", Path: mbase + "/webhook-test", Summary: "Send a test referral.milestone_reached to a webhook reward's endpoint",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *MilestoneInput) (*api.Body[ReferralWebhookTestResult], error) {
			out, err := s.TestWebhook(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-referral-milestone-email", Path: mbase + "/email", Summary: "A milestone email for the composer's milestone mode", Tags: tags, Access: view},
		func(ctx context.Context, in *MilestoneInput) (*api.Body[ReferralMilestoneEmailDetail], error) {
			out, err := s.MilestoneEmail(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "autosave-referral-milestone-email", Path: mbase + "/email", Summary: "Autosave a milestone email's draft (409 when saved elsewhere since)", Tags: tags, Access: edit,
		MaxBodyBytes: 8 << 20},
		func(ctx context.Context, in *struct {
			MilestoneInput
			Body ReferralEmailSave
		}) (*api.Body[ReferralEmailSaved], error) {
			out, err := s.SaveMilestoneEmail(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body, false)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "save-referral-milestone-email", Path: mbase + "/email/save", Summary: "Save a milestone email: the saved version is the one that goes out", Tags: tags, Access: edit,
		MaxBodyBytes: 8 << 20},
		func(ctx context.Context, in *struct {
			MilestoneInput
			Body ReferralEmailSave
		}) (*api.Body[ReferralEmailSaved], error) {
			out, err := s.SaveMilestoneEmail(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body, true)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-referral-milestone-email-on", Path: mbase + "/email/on", Summary: "Send this milestone's email, or none (an automation sends one)", Tags: tags, Access: edit},
		func(ctx context.Context, in *struct {
			MilestoneInput
			Body ReferralEmailOnInput
		}) (*api.Empty, error) {
			return api.NoContent, s.SetMilestoneEmailOn(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body.On)
		})
	api.Post(h, api.Op{ID: "preview-referral-milestone-email", Path: mbase + "/email/preview", Summary: "Render a milestone email as a reader gets it", Tags: tags, Access: view,
		MaxBodyBytes: 8 << 20},
		func(ctx context.Context, in *struct {
			MilestoneInput
			Body ReferralEmailPreviewInput
		}) (*api.Body[ReferralEmailRender], error) {
			out, err := s.PreviewMilestoneEmail(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "test-referral-milestone-email", Path: mbase + "/email/test", Summary: "Send a test of a milestone email through the System route", Tags: tags, Access: edit,
		MaxBodyBytes: 8 << 20},
		func(ctx context.Context, in *struct {
			MilestoneInput
			Body ReferralEmailTest
		}) (*api.Body[ReferralEmailTestResult], error) {
			out, err := s.TestMilestoneEmail(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "upload-referral-file", Path: base + "/{program}/files", Summary: "Upload a file for download rewards (PDF, ZIP or an image, up to 100 MB)",
		Tags: tags, Access: edit, MaxBodyBytes: MaxFileBytes + 1<<20, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			ProgramInput
			RawBody huma.MultipartFormFiles[struct {
				File huma.FormFile `form:"file" contentType:"application/pdf,application/zip,image/png,image/jpeg,image/gif,image/webp,application/octet-stream" required:"true"`
			}]
		}) (*api.Body[ReferralFile], error) {
			f := in.RawBody.Data().File
			if !f.IsSet {
				return nil, api.Invalid("body.file", "Choose a file to upload.")
			}
			out, err := s.UploadFile(ctx, api.WorkspaceFrom(ctx), in.id(), f.Filename, f)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-referral-file", Path: base + "/{program}/files/{file}", Summary: "Remove a file from the program's files", Tags: tags, Access: edit},
		func(ctx context.Context, in *struct {
			ProgramInput
			File string `path:"file" format:"uuid"`
		}) (*api.Empty, error) {
			return api.NoContent, s.RemoveFile(ctx, api.WorkspaceFrom(ctx), in.id(), uuid.MustParse(in.File))
		})

	// Referrers.
	api.Get(h, api.Op{ID: "list-referrers", Path: base + "/{program}/referrers", Summary: "Referrers by counted referrals, with search and filters", Tags: tags, Access: view},
		func(ctx context.Context, in *struct {
			ProgramInput
			Query     string `query:"q" maxLength:"200"`
			Milestone int    `query:"milestone" minimum:"-1" maximum:"100000" doc:"0 any, -1 none yet, n reached n"`
			Held      bool   `query:"held"`
			Paused    bool   `query:"paused"`
			Page      int    `query:"page" minimum:"1" default:"1"`
		}) (*api.Body[ReferralReferrerList], error) {
			out, err := s.Referrers(ctx, api.WorkspaceFrom(ctx), in.id(), ReferrerParams{Query: in.Query, Milestone: in.Milestone, Held: in.Held, Paused: in.Paused, Page: in.Page})
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-referrer", Path: base + "/{program}/referrers/{link}", Summary: "A referrer: their link, progress, referrals and rewards", Tags: tags, Access: view},
		func(ctx context.Context, in *ReferrerInput) (*api.Body[ReferralReferrerPanel], error) {
			out, err := s.Referrer(ctx, api.WorkspaceFrom(ctx), in.id(), uuid.MustParse(in.Link))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "pause-referral-link", Path: base + "/{program}/referrers/{link}/pause", Summary: "Pause a referrer's link: it leads to the normal subscribe page",
		Tags: tags, Access: review},
		func(ctx context.Context, in *struct {
			ReferrerInput
			Body ReferralLinkPauseInput
		}) (*api.Empty, error) {
			return api.NoContent, s.SetLinkStatus(ctx, api.WorkspaceFrom(ctx), in.id(), uuid.MustParse(in.Link), "paused", in.Body.Note)
		})
	api.Post(h, api.Op{ID: "resume-referral-link", Path: base + "/{program}/referrers/{link}/resume", Summary: "Resume a referrer's paused link", Tags: tags, Access: review},
		func(ctx context.Context, in *ReferrerInput) (*api.Empty, error) {
			return api.NoContent, s.SetLinkStatus(ctx, api.WorkspaceFrom(ctx), in.id(), uuid.MustParse(in.Link), "active", "")
		})
	api.Get(h, api.Op{ID: "export-referrers", Path: base + "/{program}/referrers.csv", Summary: "The referrers list as CSV (logged)", Tags: tags,
		Access: api.WorkspaceHuman(authz.SubscribersExport)},
		func(ctx context.Context, in *ProgramInput) (*ReferralCSV, error) {
			body, err := s.ExportReferrers(ctx, api.WorkspaceFrom(ctx), in.id())
			if err != nil {
				return nil, err
			}
			return &ReferralCSV{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="referrers.csv"`, CacheControl: "no-store", Body: body}, nil
		})
	api.Get(h, api.Op{ID: "export-referral-fulfillment", Path: base + "/{program}/fulfill.csv", Summary: "The To fulfill queue with shipping addresses as CSV (logged)", Tags: tags,
		Access: api.WorkspaceHuman(authz.SubscribersExport)},
		func(ctx context.Context, in *ProgramInput) (*ReferralCSV, error) {
			body, err := s.ExportToFulfill(ctx, api.WorkspaceFrom(ctx), in.id())
			if err != nil {
				return nil, err
			}
			return &ReferralCSV{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="to-fulfill.csv"`, CacheControl: "no-store", Body: body}, nil
		})

	// Rewards.
	rbase := "/workspaces/{workspace}/grow/rewards/{reward}"
	api.Post(h, api.Op{ID: "resend-referral-reward", Path: rbase + "/resend", Summary: "Send a reward's milestone email again", Tags: tags, Access: review},
		func(ctx context.Context, in *RewardInput) (*api.Empty, error) {
			return api.NoContent, s.ResendReward(ctx, api.WorkspaceFrom(ctx), in.id())
		})
	api.Post(h, api.Op{ID: "fulfill-referral-reward", Path: rbase + "/fulfill", Summary: "Mark a manual reward fulfilled, with a note the referrer sees", Tags: tags, Access: review},
		func(ctx context.Context, in *struct {
			RewardInput
			Body ReferralFulfillInput
		}) (*api.Empty, error) {
			return api.NoContent, s.FulfillReward(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
		})
	api.Post(h, api.Op{ID: "retry-referral-reward", Path: rbase + "/retry", Summary: "Try a failed webhook reward again", Tags: tags, Access: review},
		func(ctx context.Context, in *RewardInput) (*api.Empty, error) {
			return api.NoContent, s.RetryReward(ctx, api.WorkspaceFrom(ctx), in.id())
		})
	api.Post(h, api.Op{ID: "copy-referral-claim-link", Path: rbase + "/claim-link", Summary: "A coupon reward's claim link to copy (logged)", Tags: tags, Access: review},
		func(ctx context.Context, in *RewardInput) (*api.Body[ReferralClaimLink], error) {
			out, err := s.ClaimLink(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})

	// Review.
	api.Get(h, api.Op{ID: "get-referral-review", Path: base + "/{program}/review", Summary: "Held referrals grouped by referrer, with their evidence", Tags: tags, Access: view},
		func(ctx context.Context, in *struct {
			ProgramInput
			Signal   string `query:"signal" enum:"burst,shared_ip,numbered,"`
			Referrer string `query:"referrer" format:"uuid"`
		}) (*api.Body[ReferralReviewQueue], error) {
			var link *uuid.UUID
			if in.Referrer != "" {
				id := uuid.MustParse(in.Referrer)
				link = &id
			}
			out, err := s.Review(ctx, api.WorkspaceFrom(ctx), in.id(), in.Signal, link)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "preview-referral-decision", Path: base + "/{program}/review/preview", Summary: "What a decision on held referrals would do", Tags: tags, Access: view},
		func(ctx context.Context, in *struct {
			ProgramInput
			Body ReferralDecisionInput
		}) (*api.Body[ReferralDecisionPreview], error) {
			out, err := s.PreviewDecision(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "decide-referrals", Path: base + "/{program}/review/decide", Summary: "Count held referrals, don't count them, or don't count them and pause the link",
		Description: "People only: MCP agents can propose a decision (propose_referral_decision), never make one. Undo works for 10 seconds.", Tags: tags, Access: review},
		func(ctx context.Context, in *struct {
			ProgramInput
			Body ReferralDecisionInput
		}) (*api.Body[ReferralDecision], error) {
			out, err := s.Decide(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "undo-referral-decision", Path: base + "/{program}/review/decisions/{decision}/undo", Summary: "Undo a decision within 10 seconds", Tags: tags, Access: review},
		func(ctx context.Context, in *struct {
			ProgramInput
			Decision string `path:"decision" format:"uuid"`
		}) (*api.Empty, error) {
			return api.NoContent, s.UndoDecision(ctx, api.WorkspaceFrom(ctx), in.id(), uuid.MustParse(in.Decision))
		})

	// The composer's blocks.
	api.Post(h, api.Op{ID: "render-grow-block", Path: "/workspaces/{workspace}/grow/blocks/render", Summary: "Render a Grow block for the composer's canvas, as a reader or a sample",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body GrowBlockRenderInput
		}) (*api.Body[GrowBlockRender], error) {
			out, err := s.RenderBlock(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-grow-preview-contacts", Path: "/workspaces/{workspace}/grow/blocks/preview-contacts", Summary: "Readers to preview the Grow blocks as: referrers, and readers the block hides for",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Program string `query:"program" format:"uuid"`
		}) (*api.Body[GrowPreviewContacts], error) {
			var program *uuid.UUID
			if in.Program != "" {
				id := uuid.MustParse(in.Program)
				program = &id
			}
			out, err := s.PreviewContacts(ctx, api.WorkspaceFrom(ctx), program)
			return api.OK(out), err
		})
}

// RegisterContact adds the contact profile's Grow data; partners adds the
// partner requests to the timeline.
func RegisterContact(h huma.API, s *Service, partners func() PartnerEvents) {
	api.Get(h, api.Op{ID: "get-contact-referrals", Path: "/workspaces/{workspace}/contacts/{contact}/referrals",
		Summary:     "A contact's referral link and counts, who referred them, and Grow's timeline events",
		Description: "API keys with grow:view can read a contact's referral link and counts, so an app can show them to its users.",
		Tags:        []string{"Grow"}, Access: api.Workspace(authz.GrowView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Contact string `path:"contact" format:"uuid"`
		}) (*api.Body[ReferralContact], error) {
			var pe PartnerEvents
			if partners != nil {
				pe = partners()
			}
			out, err := s.Contact(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Contact), pe)
			return api.OK(out), err
		})
}
