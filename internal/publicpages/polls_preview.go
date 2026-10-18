package publicpages

// Previews of the poll answer page and the survey page for the app (design
// expansion, X11): the composer's "Preview the answer page" and the survey
// builder's live preview. Inert, like every preview: nothing submits.

import (
	"context"
	"encoding/json"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/polls"
	"github.com/hostrepute/reputecast/internal/store"
)

// PollPagePreviewInput is a poll's settings (rc-poll) and its email's name.
type PollPagePreviewInput struct {
	Config   string `json:"config" maxLength:"20000" doc:"The Poll block's settings, as its rc-poll attribute holds them"`
	Campaign string `json:"campaign,omitempty" maxLength:"200"`
}

// SurveyPagePreviewInput is a survey as the builder has it (unsaved edits
// included).
type SurveyPagePreviewInput struct {
	Name        string                 `json:"name" maxLength:"200"`
	Description string                 `json:"description,omitempty" maxLength:"1000"`
	Slug        string                 `json:"slug,omitempty" maxLength:"60"`
	Questions   []polls.SurveyQuestion `json:"questions" maxItems:"20"`
	Settings    polls.SurveySettings   `json:"settings"`
}

type pollPreviewInput struct {
	api.WorkspaceInput
	Body PollPagePreviewInput
}

type surveyPreviewInput struct {
	api.WorkspaceInput
	Body SurveyPagePreviewInput
}

func registerPollPreviews(h huma.API, p *Handler) {
	tags := []string{"Public pages"}
	api.Post(h, api.Op{ID: "preview-poll-page", Path: "/workspaces/{workspace}/public-pages/poll-preview",
		Summary: "Preview a poll's answer page as a contact sees it after answering", Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *pollPreviewInput) (*api.Body[PublicPagePreview], error) {
			out, err := p.PollPreview(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "preview-survey-page", Path: "/workspaces/{workspace}/public-pages/survey-preview",
		Summary: "Preview a survey's page as a contact sees it", Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *surveyPreviewInput) (*api.Body[PublicPagePreview], error) {
			out, err := p.SurveyPreview(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
}

func (h *Handler) previewSite(ctx context.Context, ws *store.Workspace) (*site, error) {
	if err := h.Init(); err != nil {
		return nil, err
	}
	set, err := h.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return nil, err
	}
	return &site{ws: ws, set: set, links: forms.LinksOn(h.URLs.Base(), ws.Slug, false), canon: h.Forms.Links(ws, set), preview: true}, nil
}

// PollPreview renders an answer page for a poll's settings: Sam answered
// the first answer, as a contact.
func (h *Handler) PollPreview(ctx context.Context, ws *store.Workspace, in PollPagePreviewInput) (PublicPagePreview, error) {
	cfg, err := polls.ParseConfig(in.Config)
	if err != nil {
		return PublicPagePreview{}, api.Invalid("body.config", "The poll's settings couldn't be read. Reload the composer.")
	}
	s, err := h.previewSite(ctx, ws)
	if err != nil {
		return PublicPagePreview{}, err
	}
	answers := cfg.AnswerList()
	first := 0
	if len(answers) > 0 {
		first = answers[0].Key
		if cfg.Kind == polls.KindScale {
			first = 9
		}
	}
	name := in.Campaign
	if name == "" {
		name = s.set.Branding.PublisherName
	}
	page := &polls.AnswerPage{Config: cfg, Answers: answers, CampaignName: name, Answerer: polls.AnswererContact, Name: "Sam", Email: "sam@example.com",
		Picked: first, Answer: &first, Saved: true}
	if cfg.Close.Mode == polls.CloseDays && cfg.Close.Days > 0 {
		t := h.now().Add(time.Duration(cfg.Close.Days) * 24 * time.Hour)
		page.ClosesAt = &t
	}
	if cfg.Results {
		share := &polls.PollShares{Total: 0, Rows: []polls.PollShare{}}
		for _, a := range answers {
			share.Rows = append(share.Rows, polls.PollShare{Key: a.Key, Label: a.Label})
		}
		page.Results = share
	}
	b := h.pollView(s, "preview", page, true)
	v := h.view(s, cfg.Question+" · "+s.set.Branding.PublisherName, b)
	v.Preview = true
	out, err := h.html("poll", v)
	if err != nil {
		return PublicPagePreview{}, err
	}
	return PublicPagePreview{HTML: string(out), Contact: "sam@example.com"}, nil
}

// SurveyPreview renders a survey's page as the builder has it.
func (h *Handler) SurveyPreview(ctx context.Context, ws *store.Workspace, in SurveyPagePreviewInput) (PublicPagePreview, error) {
	s, err := h.previewSite(ctx, ws)
	if err != nil {
		return PublicPagePreview{}, err
	}
	qs := in.Questions
	if qs == nil {
		qs = []polls.SurveyQuestion{}
	}
	raw, _ := json.Marshal(qs)
	var normalized []polls.SurveyQuestion
	_ = json.Unmarshal(raw, &normalized)
	page := &polls.SurveyPage{Name: in.Name, Description: in.Description, Slug: in.Slug, Questions: normalized, Settings: in.Settings, Answerer: polls.AnswererContact,
		FirstName: "Priya", Email: "priya@example.com", Answers: map[string]any{}}
	b := h.surveyView(s, page, "")
	v := h.view(s, in.Name+" · "+s.set.Branding.PublisherName, b)
	v.Preview = true
	out, err := h.html("survey", v)
	if err != nil {
		return PublicPagePreview{}, err
	}
	url := ""
	if in.Slug != "" {
		url = s.canon.Survey(in.Slug)
	}
	return PublicPagePreview{HTML: string(out), URL: url, Contact: "priya@example.com"}, nil
}
