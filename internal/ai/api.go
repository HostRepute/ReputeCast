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

type budgetReconcileInput struct {
	api.WorkspaceInput
	ID   uuid.UUID `path:"id"`
	Body AIBudgetReconcileInput
}

type activityInput struct {
	api.WorkspaceInput
	Member  string `query:"member" doc:"A member's id"`
	Feature string `query:"feature" enum:"subject_ideas,rewrite,translate,alt_text,accessibility_fix,segment_draft,report_summary,brand_voice,draft,agent,connection_test,assistant"`
	Outcome string `query:"outcome" enum:"accepted,kept,rejected,refused,error,nothing_drafted,pending,answered"`
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
	Feature string `path:"feature" enum:"subject_ideas,rewrite,translate,alt_text,accessibility_fix,segment_draft,report_summary,brand_voice,draft"`
	Body    AIFeatureInput
}

type learnVoiceInput struct {
	api.WorkspaceInput
	Body AILearnVoiceInput
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

type accountPath struct {
	api.WorkspaceInput
	Account uuid.UUID `path:"account" format:"uuid" doc:"The account's id"`
}

type addAccountInput struct {
	api.WorkspaceInput
	Body AIAccountInput
}

type accountKeyInput struct {
	api.WorkspaceInput
	Account uuid.UUID `path:"account" format:"uuid" doc:"The account's id"`
	Body    AIKeyInput
}

type addModelInput struct {
	api.WorkspaceInput
	Account uuid.UUID `path:"account" format:"uuid" doc:"The account's id"`
	Body    AIAccountModelInput
}

type modelPath struct {
	api.WorkspaceInput
	Account uuid.UUID `path:"account" format:"uuid" doc:"The account's id"`
	Model   uuid.UUID `path:"model" format:"uuid" doc:"The model's id"`
}

type modelPricesInput struct {
	api.WorkspaceInput
	Account uuid.UUID `path:"account" format:"uuid" doc:"The account's id"`
	Model   uuid.UUID `path:"model" format:"uuid" doc:"The model's id"`
	Body    AIPricesInput
}

type featureModelInput struct {
	api.WorkspaceInput
	Feature string `path:"feature" enum:"assistant,agent,subject_ideas,rewrite,translate,alt_text,accessibility_fix,segment_draft,report_summary,brand_voice,draft"`
	Body    AIFeatureModelInput
}

type featurePath struct {
	api.WorkspaceInput
	Feature string `path:"feature" enum:"assistant,agent,subject_ideas,rewrite,translate,alt_text,accessibility_fix,segment_draft,report_summary,brand_voice,draft"`
}

type assistantSettingsInput struct {
	api.WorkspaceInput
	Body AIAssistantSettings
}

type personalKeysInput struct {
	api.WorkspaceInput
	Body AIPersonalKeysInput
}

// AIPersonalKeyReplaced is the personal settings after a new key passed
// its test.
type AIPersonalKeyReplaced struct {
	Personal AIPersonalSettings `json:"personal"`
	Test     AITestResult       `json:"test"`
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
	registerDrafts(h, s)
	registerAutomationDrafts(h, s)
	registerAgent(h, s)
	tags := []string{"AI assistant"}
	base := "/workspaces/{workspace}/ai"
	view := api.Workspace(authz.IntegrationsView)
	manage := api.WorkspaceHuman(authz.IntegrationsManage)
	use := api.WorkspaceHuman(authz.AIUse)

	api.Get(h, api.Op{ID: "list-ai-budget-holds", Path: base + "/budget-holds", Summary: "List reserved AI usage awaiting a response or reconciliation", Tags: tags, Access: manage},
		func(ctx context.Context, in *settingsInput) (*api.Body[AIBudgetHolds], error) {
			out, err := s.BudgetHolds(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "reconcile-ai-budget", Path: base + "/budget-holds/{id}/reconcile", Summary: "Record provider-verified usage without retrying the request", Tags: tags, Access: manage},
		func(ctx context.Context, in *budgetReconcileInput) (*api.Empty, error) {
			return api.NoContent, s.ReconcileBudget(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
		})
	api.Post(h, api.Op{ID: "learn-brand-voice", Path: base + "/brand-voice", Summary: "Suggest a voice from up to ten selected sent campaigns", Tags: tags, Access: use},
		func(ctx context.Context, in *learnVoiceInput) (*api.Body[AILearnedVoice], error) {
			out, err := s.LearnVoice(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
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
	api.Put(h, api.Op{ID: "connect-ai-model", Path: base + "/connection", Summary: "Connect the workspace's model (its first account's first model), after the test passes",
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

	// Model accounts, their models and the model each feature uses. The
	// operations above act on the first account and its first model.
	api.Post(h, api.Op{ID: "add-ai-account", Path: base + "/accounts", Summary: "Add a model account with its first model, after the test passes",
		Description: "The key is sealed and never returned. A workspace's first model becomes every feature's model. Audited as ai_account.add.", Tags: tags, Access: manage},
		func(ctx context.Context, in *addAccountInput) (*api.Body[AISettings], error) {
			out, err := s.AddAccount(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-ai-account", Path: base + "/accounts/{account}", Status: http.StatusOK, Summary: "Remove an account, its key and its models",
		Description: "Refused while a feature uses one of its models. The activity log keeps their requests. Audited as ai_account.remove.", Tags: tags, Access: manage},
		func(ctx context.Context, in *accountPath) (*api.Body[AISettings], error) {
			out, err := s.RemoveAccount(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "replace-ai-account-key", Path: base + "/accounts/{account}/key", Summary: "Replace an account's key, after it passes the test with the account's first model",
		Description: "Audited as ai_account.key_replaced.", Tags: tags, Access: manage},
		func(ctx context.Context, in *accountKeyInput) (*api.Body[AIKeyReplaced], error) {
			out, test, err := s.ReplaceAccountKey(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account, in.Body)
			return api.OK(AIKeyReplaced{Settings: out, Test: test}), err
		})
	api.Post(h, api.Op{ID: "add-ai-account-model", Path: base + "/accounts/{account}/models", Summary: "Add a model to an account, with its prices",
		Description: "Prices follow the price list for a known model unless given. Audited as ai_model.add.", Tags: tags, Access: manage},
		func(ctx context.Context, in *addModelInput) (*api.Body[AISettings], error) {
			out, err := s.AddModel(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-ai-model-prices", Path: base + "/accounts/{account}/models/{model}/prices", Summary: "Set a model's prices per million tokens",
		Description: "Audited as ai_model.prices_updated.", Tags: tags, Access: manage},
		func(ctx context.Context, in *modelPricesInput) (*api.Body[AISettings], error) {
			out, err := s.SetModelPrices(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account, in.Model, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-ai-model", Path: base + "/accounts/{account}/models/{model}", Status: http.StatusOK, Summary: "Remove a model from an account",
		Description: "Refused while a feature uses it, and for an account's only model. Audited as ai_model.remove.", Tags: tags, Access: manage},
		func(ctx context.Context, in *modelPath) (*api.Body[AISettings], error) {
			out, err := s.RemoveModel(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account, in.Model)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "test-ai-model-tools", Path: base + "/accounts/{account}/models/{model}/test-tools", Summary: "Test a model's tool calling with fixed data, using the monthly AI budget",
		Description: "The agent and the assistant use only a model that passed.", Tags: tags, Access: manage},
		func(ctx context.Context, in *modelPath) (*api.Body[AIAgentCapability], error) {
			out, err := s.TestModelTools(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account, in.Model)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-ai-feature-model", Path: base + "/features/{feature}/model", Summary: "Pick the model a feature uses",
		Description: "A model of a connected account whose key works; alt text needs one that reads images, the agent and the assistant one that passed the tool-calling check. Requests already prepared with the previous model aren't sent; agent runs keep the model they began with. A personal key's model can't be one. Audited as ai_feature.model_changed.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *featureModelInput) (*api.Body[AISettings], error) {
			out, err := s.SetFeatureModel(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Feature, in.Body)
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "set-ai-assistant-settings", Path: base + "/assistant/settings", Summary: "Turn the assistant on or off and set its limits per thread and its turns at once",
		Description: "Limits apply to threads started afterwards. The revision is the one read: a change since then is refused. Audited as ai_assistant.settings_updated.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *assistantSettingsInput) (*api.Body[AISettings], error) {
			out, err := s.SetAssistantSettings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})

	// Personal keys (assistant brief D4): the switch and each person's cap
	// are managers'; a person's own accounts are theirs alone, with ai:use.
	api.Put(h, api.Op{ID: "set-ai-personal-keys", Path: base + "/personal-keys", Summary: "Turn personal keys on or off and set each person's monthly cap",
		Description: "Turning them off keeps the accounts; their requests are refused until they are on again. Audited as ai_personal_keys.enabled, ai_personal_keys.disabled or ai_personal_keys.cap_updated.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *personalKeysInput) (*api.Body[AISettings], error) {
			out, err := s.SetPersonalKeys(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-ai-member-account", Path: base + "/personal-keys/accounts/{account}", Status: http.StatusOK, Summary: "Remove a member's personal account, its key and its models",
		Description: "For offboarding. The member's picks of its models go with it; the activity log keeps their requests. Audited as ai_account.remove.", Tags: tags, Access: manage},
		func(ctx context.Context, in *accountPath) (*api.Body[AISettings], error) {
			out, err := s.RemoveMemberAccount(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-ai-personal-budget-holds", Path: base + "/personal/budget-holds", Summary: "List one's own usage reserved on personal keys, awaiting a response or reconciliation",
		Description: "Only the caller's own holds; they count against the caller's personal monthly cap.", Tags: tags, Access: use},
		func(ctx context.Context, in *settingsInput) (*api.Body[AIBudgetHolds], error) {
			out, err := s.PersonalBudgetHolds(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "reconcile-ai-personal-budget", Path: base + "/personal/budget-holds/{id}/reconcile", Summary: "Record provider-verified usage of one's own hold without retrying the request",
		Description: "Settles the hold in the caller's own budget. Another person's hold isn't found. Audited as ai.usage_reconciled.", Tags: tags, Access: use},
		func(ctx context.Context, in *budgetReconcileInput) (*api.Empty, error) {
			return api.NoContent, s.ReconcilePersonalBudget(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
		})
	api.Get(h, api.Op{ID: "get-ai-personal", Path: base + "/personal", Summary: "The caller's personal keys: their own accounts, models, picks and use",
		Description: "The key is write-only: only whether one is set, and when.", Tags: tags, Access: use},
		func(ctx context.Context, in *settingsInput) (*api.Body[AIPersonalSettings], error) {
			out, err := s.Personal(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-ai-personal-account", Path: base + "/personal/accounts", Summary: "Add one's own model account with its first model, after the test passes",
		Description: "Needs personal keys on. The key is sealed and never returned; only its owner's requests use it, in their own monthly cap. Audited as ai_account.add.", Tags: tags, Access: use},
		func(ctx context.Context, in *addAccountInput) (*api.Body[AIPersonalSettings], error) {
			out, err := s.AddPersonalAccount(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-ai-personal-account", Path: base + "/personal/accounts/{account}", Status: http.StatusOK, Summary: "Remove one's own account, its key and its models",
		Description: "Refused while one of its models is picked for a feature. Audited as ai_account.remove.", Tags: tags, Access: use},
		func(ctx context.Context, in *accountPath) (*api.Body[AIPersonalSettings], error) {
			out, err := s.RemovePersonalAccount(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "replace-ai-personal-key", Path: base + "/personal/accounts/{account}/key", Summary: "Replace one's own account's key, after it passes the test",
		Description: "Also adds the key again to an account whose key was removed. Audited as ai_account.key_replaced.", Tags: tags, Access: use},
		func(ctx context.Context, in *accountKeyInput) (*api.Body[AIPersonalKeyReplaced], error) {
			out, test, err := s.ReplacePersonalKey(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account, in.Body)
			return api.OK(AIPersonalKeyReplaced{Personal: out, Test: test}), err
		})
	api.Post(h, api.Op{ID: "add-ai-personal-model", Path: base + "/personal/accounts/{account}/models", Summary: "Add a model to one's own account, with its prices",
		Description: "Audited as ai_model.add.", Tags: tags, Access: use},
		func(ctx context.Context, in *addModelInput) (*api.Body[AIPersonalSettings], error) {
			out, err := s.AddPersonalModel(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-ai-personal-model-prices", Path: base + "/personal/accounts/{account}/models/{model}/prices", Summary: "Set one's own model's prices per million tokens",
		Description: "Audited as ai_model.prices_updated.", Tags: tags, Access: use},
		func(ctx context.Context, in *modelPricesInput) (*api.Body[AIPersonalSettings], error) {
			out, err := s.SetPersonalModelPrices(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account, in.Model, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-ai-personal-model", Path: base + "/personal/accounts/{account}/models/{model}", Status: http.StatusOK, Summary: "Remove a model from one's own account",
		Description: "Refused while it is picked for a feature, and for an account's only model. Audited as ai_model.remove.", Tags: tags, Access: use},
		func(ctx context.Context, in *modelPath) (*api.Body[AIPersonalSettings], error) {
			out, err := s.RemovePersonalModel(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account, in.Model)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "test-ai-personal-model", Path: base + "/personal/accounts/{account}/models/{model}/test", Summary: "Test one's own model: a fixed prompt with no workspace content",
		Description: "In the caller's own monthly cap.", Tags: tags, Access: use},
		func(ctx context.Context, in *modelPath) (*api.Body[AITestResult], error) {
			out, err := s.TestPersonalModel(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account, in.Model)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "test-ai-personal-model-tools", Path: base + "/personal/accounts/{account}/models/{model}/test-tools", Summary: "Test one's own model's tool calling with fixed data",
		Description: "In the caller's own monthly cap. The agent and the assistant use a personal model only once it passed.", Tags: tags, Access: use},
		func(ctx context.Context, in *modelPath) (*api.Body[AIAgentCapability], error) {
			out, err := s.TestPersonalModelTools(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Account, in.Model)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-ai-personal-feature-model", Path: base + "/personal/features/{feature}/model", Summary: "Pick one of one's own models for one's own requests of a feature",
		Description: "The rules of the workspace's models apply. Requests already prepared with another model aren't sent; agent runs keep the model they began with. Audited as ai_feature.model_changed.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *featureModelInput) (*api.Body[AIPersonalSettings], error) {
			out, err := s.SetPersonalFeatureModel(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Feature, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "clear-ai-personal-feature-model", Path: base + "/personal/features/{feature}/model", Status: http.StatusOK, Summary: "Go back to the workspace's model for one's own requests of a feature",
		Description: "Audited as ai_feature.model_changed.", Tags: tags, Access: use},
		func(ctx context.Context, in *featurePath) (*api.Body[AIPersonalSettings], error) {
			out, err := s.ClearPersonalFeatureModel(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Feature)
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
