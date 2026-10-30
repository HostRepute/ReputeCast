package ai

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type settingsInput struct {
	api.WorkspaceInput
}

type activityInput struct {
	api.WorkspaceInput
	Member  string `query:"member" doc:"A member's id"`
	Feature string `query:"feature" enum:"subject_ideas,rewrite,translate,alt_text,accessibility_fix,segment_draft,report_summary"`
	Outcome string `query:"outcome" enum:"accepted,kept,rejected,refused,error,nothing_drafted,pending"`
	Cursor  string `query:"cursor" maxLength:"200"`
	Limit   int    `query:"limit" minimum:"1" maximum:"100" default:"50"`
}

type modelsInput struct {
	api.WorkspaceInput
	Body AIModelsInput
}

type testInput struct {
	api.WorkspaceInput
	Body AITestInput
}

type connectInput struct {
	api.WorkspaceInput
	Body AIConnectInput
}

type keyInput struct {
	api.WorkspaceInput
	Body AIKeyInput
}

// AIKeyReplaced is the settings after a new key passed its test.
type AIKeyReplaced struct {
	Settings AISettings   `json:"settings"`
	Test     AITestResult `json:"test"`
}

type modelInput struct {
	api.WorkspaceInput
	Body AIModelInput
}

type capInput struct {
	api.WorkspaceInput
	Body AICapInput
}

type pricesInput struct {
	api.WorkspaceInput
	Body AIPricesInput
}

// AIFeatureInput switches a feature.
type AIFeatureInput struct {
	On bool `json:"on"`
}

type featureInput struct {
	api.WorkspaceInput
	Feature string `path:"feature" enum:"subject_ideas,rewrite,translate,alt_text,accessibility_fix,segment_draft,report_summary"`
	Body    AIFeatureInput
}

type subjectInput struct {
	api.WorkspaceInput
	Body AISubjectIdeasInput
}

type rewriteInput struct {
	api.WorkspaceInput
	Body AIRewriteInput
}

type translateInput struct {
	api.WorkspaceInput
	Body AITranslateInput
}

type altInput struct {
	api.WorkspaceInput
	Body AIAltTextInput
}

type linkInput struct {
	api.WorkspaceInput
	Body AILinkTextInput
}

type segmentInput struct {
	api.WorkspaceInput
	Body AISegmentDraftInput
}

type summaryInput struct {
	api.WorkspaceInput
	Body AIReportSummaryInput
}

type askInput struct {
	api.WorkspaceInput
	Body AIReportQuestionInput
}

type outcomeInput struct {
	api.WorkspaceInput
	Request string `path:"request" format:"uuid"`
	Body    AIOutcomeInput
}

type campaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid"`
}

type keepInput struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid"`
	Body       AIKeepSummaryInput
}

type languagePath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid"`
	Language   string `path:"language" pattern:"^[a-z]{2,3}$"`
}

type acceptInput struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid"`
	Language   string `path:"language" pattern:"^[a-z]{2,3}$"`
	Body       AITranslationAcceptInput
}

func campaignID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, api.NotFound("Campaign")
	}
	return id, nil
}

// Register adds the assistant's operations. Its settings, usage and log
// need integrations:view; changing them integrations:manage ⚑ (people
// only). The suggestions need ai:use, which API keys and MCP agents can't
// hold: agents bring their own model.
func Register(h huma.API, s *Service) {
	tags := []string{"AI assistant"}
	base := "/workspaces/{workspace}/ai"
	view := api.Workspace(authz.IntegrationsView)
	manage := api.WorkspaceHuman(authz.IntegrationsManage)
	use := api.WorkspaceHuman(authz.AIUse)

	api.Get(h, api.Op{ID: "get-ai-settings", Path: base, Summary: "The AI assistant's settings: the model, the cap, this month's usage and the features",
		Description: "The key is write-only: only whether one is set, and when. Settings › Integrations › AI assistant.", Tags: tags, Access: view},
		func(ctx context.Context, in *settingsInput) (*api.Body[AISettings], error) {
			out, err := s.Settings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-ai-activity", Path: base + "/activity", Summary: "The AI activity log: each request of the last 90 days, never its content",
		Tags: tags, Access: view},
		func(ctx context.Context, in *activityInput) (*api.Body[AIActivityList], error) {
			out, err := s.Activity(ctx, api.WorkspaceFrom(ctx), AIActivityFilter{Member: in.Member, Feature: in.Feature, Outcome: in.Outcome, Cursor: in.Cursor, Limit: in.Limit})
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "list-ai-models", Path: base + "/models", Summary: "The models a provider offers the key, with ReputeCast's prices",
		Description: "Without a key it lists the price list's models. Nothing about the workspace is sent.", Tags: tags, Access: manage},
		func(ctx context.Context, in *modelsInput) (*api.Body[AIModelList], error) {
			out, err := s.Models(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "test-ai-connection", Path: base + "/test", Summary: "Test connection: a fixed prompt with no workspace content",
		Description: "Tests the model being set up, or the saved one when the body is empty. Shows the answer and the time it took.", Tags: tags, Access: manage},
		func(ctx context.Context, in *testInput) (*api.Body[AITestResult], error) {
			out, err := s.Test(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "connect-ai-model", Path: base + "/connection", Summary: "Connect the workspace's model (one per workspace), after the test passes",
		Description: "The key is sealed and never returned. Audited as ai.connection_created or ai.connection_updated.", Tags: tags, Access: manage},
		func(ctx context.Context, in *connectInput) (*api.Body[AISettings], error) {
			out, err := s.Connect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "disconnect-ai-model", Path: base + "/connection", Status: http.StatusOK, Summary: "Disconnect the model: the key is deleted",
		Description: "The model, the cap and the prices are kept for the next connection; the activity log, kept summaries and accepted text stay. Audited as ai.connection_removed.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *settingsInput) (*api.Body[AISettings], error) {
			out, err := s.Disconnect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "replace-ai-key", Path: base + "/key", Summary: "Replace the key, after it passes the test", Tags: tags, Access: manage},
		func(ctx context.Context, in *keyInput) (*api.Body[AIKeyReplaced], error) {
			out, test, err := s.ReplaceKey(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(AIKeyReplaced{Settings: out, Test: test}), err
		})
	api.Put(h, api.Op{ID: "change-ai-model", Path: base + "/model", Summary: "Change model: another model of the same provider", Tags: tags, Access: manage},
		func(ctx context.Context, in *modelInput) (*api.Body[AISettings], error) {
			out, err := s.ChangeModel(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-ai-cap", Path: base + "/cap", Summary: "Set the monthly cap: dollars, tokens or none", Tags: tags, Access: manage},
		func(ctx context.Context, in *capInput) (*api.Body[AISettings], error) {
			out, err := s.SetCap(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-ai-prices", Path: base + "/prices", Summary: "Set the prices per million tokens the cap is estimated with", Tags: tags, Access: manage},
		func(ctx context.Context, in *pricesInput) (*api.Body[AISettings], error) {
			out, err := s.SetPrices(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-ai-feature", Path: base + "/features/{feature}", Summary: "Turn a feature on or off", Tags: tags, Access: manage},
		func(ctx context.Context, in *featureInput) (*api.Body[AISettings], error) {
			out, err := s.SetFeature(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Feature, in.Body.On)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-ai-status", Path: base + "/status", Summary: "Whether the assistant is ready: the model, the features on, this month's use",
		Tags: tags, Access: use},
		func(ctx context.Context, in *settingsInput) (*api.Body[AIStatus], error) {
			out, err := s.Status(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "suggest-subject-ideas", Path: base + "/subject-ideas", MaxBodyBytes: 4 << 20, Summary: "Five subject lines, each with the subject checks",
		Description: "Sends the subject, the preheader, the email's text without images, its language and the subject rules. Never contact data: tokens go as written.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *subjectInput) (*api.Body[AISubjectIdeas], error) {
			out, err := s.SubjectIdeas(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "suggest-rewrite", Path: base + "/rewrite", Summary: "Shorten, friendlier, more formal, more direct, or with an instruction",
		Description: "Sends the selected text and the instruction. An instruction about sender reputation isn't sent: the answer points at HostRepute's assistant.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *rewriteInput) (*api.Body[AIRewrite], error) {
			out, err := s.Rewrite(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "suggest-translation", Path: base + "/translate", Summary: "Translate a campaign's source into one of the workspace's languages",
		Description: "Sends the source's text block by block, its subject and preheader, and the target language. Tokens and links the model changed are restored.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *translateInput) (*api.Body[AITranslation], error) {
			out, err := s.Translate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "suggest-alt-text", Path: base + "/alt-text", Summary: "Alt text for an image: the image at 1,024 px and the text around it",
		Tags: tags, Access: use},
		func(ctx context.Context, in *altInput) (*api.Body[AIAltText], error) {
			out, err := s.AltText(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "suggest-link-text", Path: base + "/link-text", Summary: "Link text that says where a link goes (an accessibility fix)",
		Tags: tags, Access: use},
		func(ctx context.Context, in *linkInput) (*api.Body[AILinkText], error) {
			out, err := s.LinkText(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "draft-segment", Path: base + "/segment-draft", Summary: "Describe the segment: draft conditions with the builder's own conditions",
		Description: "Sends the description and the condition schema: field names and types, list, segment and tag names, tiers. Never values from any contact. Needs segments:edit.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *segmentInput) (*api.Body[AISegmentDraft], error) {
			out, err := s.DraftSegment(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "summarize-report", Path: base + "/report-summary", Summary: "Summarize results: a plain-language summary of a report's numbers",
		Description: "Sends the report's aggregate numbers only (no addresses). Needs reports:view.", Tags: tags, Access: use},
		func(ctx context.Context, in *summaryInput) (*api.Body[AIReportSummary], error) {
			out, err := s.Summarize(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "ask-about-report", Path: base + "/report-question", Summary: "Ask about this report: an answer from the report's numbers, each one checked",
		Description: "Sends the report's aggregate numbers and the question only (no addresses) to the workspace's own model; the answer's numbers are filled in from the report. Needs reports:view.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *askInput) (*api.Body[AIReportAnswer], error) {
			out, err := s.Ask(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "record-ai-outcome", Path: base + "/requests/{request}/outcome", Summary: "Record what was done with a suggestion: accepted, rejected…",
		Tags: tags, Access: use},
		func(ctx context.Context, in *outcomeInput) (*api.Empty, error) {
			id, err := uuid.Parse(in.Request)
			if err != nil {
				return nil, api.NotFound("Suggestion")
			}
			return api.NoContent, s.RecordOutcome(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
		})

	cbase := "/workspaces/{workspace}/campaigns/{campaignId}"
	api.Get(h, api.Op{ID: "get-ai-report-summary", Path: cbase + "/ai-summary", Summary: "The summary kept on a campaign's report (never on read-only links)",
		Tags: tags, Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *campaignPath) (*api.Body[AIKeptSummary], error) {
			id, err := campaignID(in.CampaignID)
			if err != nil {
				return nil, err
			}
			out, err := s.KeptSummary(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "keep-ai-report-summary", Path: cbase + "/ai-summary", Summary: "Keep a summary at the top of the report", Tags: tags, Access: use},
		func(ctx context.Context, in *keepInput) (*api.Body[AIKeptSummary], error) {
			id, err := campaignID(in.CampaignID)
			if err != nil {
				return nil, err
			}
			out, err := s.KeepSummary(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-ai-report-summary", Path: cbase + "/ai-summary", Summary: "Take the kept summary off the report", Tags: tags, Access: use},
		func(ctx context.Context, in *campaignPath) (*api.Empty, error) {
			id, err := campaignID(in.CampaignID)
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.RemoveSummary(ctx, api.WorkspaceFrom(ctx), id)
		})
	api.Get(h, api.Op{ID: "list-ai-translations", Path: cbase + "/ai-translations", Summary: "A campaign's language versions translated by AI, and whether they were reviewed",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *campaignPath) (*api.Body[AITranslationMarks], error) {
			id, err := campaignID(in.CampaignID)
			if err != nil {
				return nil, err
			}
			out, err := s.TranslationMarks(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "accept-ai-translation", Path: cbase + "/languages/{language}/ai-translation", Summary: "Accept a translation, or some of its blocks, into the language version",
		Description: "Adds the version when it is missing. Accepting every block marks it translated, \"Translated by AI · not reviewed\"; otherwise it stays a Draft. A content edit, in the version's history.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *acceptInput) (*api.Body[AITranslationAccepted], error) {
			id, err := campaignID(in.CampaignID)
			if err != nil {
				return nil, err
			}
			out, err := s.AcceptTranslation(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Language, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "mark-ai-translation-reviewed", Path: cbase + "/languages/{language}/ai-reviewed", Summary: "Mark an AI translation reviewed",
		Tags: tags, Access: api.WorkspaceHuman(authz.CampaignsEdit)},
		func(ctx context.Context, in *languagePath) (*api.Empty, error) {
			id, err := campaignID(in.CampaignID)
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.MarkReviewed(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Language)
		})
}
