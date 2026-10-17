package polls

import (
	"context"
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/permcheck"
	"github.com/hostrepute/reputecast/internal/store"
)

type PollCampaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid" doc:"Campaign id"`
}

func (in PollCampaignPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.CampaignID)
	if err != nil {
		return uuid.Nil, api.NotFound("Campaign")
	}
	return id, nil
}

type PollPath struct {
	api.WorkspaceInput
	PollID string `path:"pollId" format:"uuid" doc:"Poll id"`
}

func (in PollPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.PollID)
	if err != nil {
		return uuid.Nil, api.NotFound("Poll")
	}
	return id, nil
}

type commentsInput struct {
	PollPath
	Group  string `query:"group" enum:"all,promoters,passives,detractors" default:"all" doc:"Comments of 0–10 answers by group"`
	Search string `query:"search" maxLength:"200" doc:"Words in the comment (and the contact's name, with subscribers:view)"`
	Offset int    `query:"offset" minimum:"0" default:"0"`
	Limit  int    `query:"limit" minimum:"1" maximum:"100" default:"20"`
}

type SurveyPath struct {
	api.WorkspaceInput
	SurveyID string `path:"surveyId" format:"uuid" doc:"Survey id"`
}

func (in SurveyPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.SurveyID)
	if err != nil {
		return uuid.Nil, api.NotFound("Survey")
	}
	return id, nil
}

type createSurveyInput struct {
	api.WorkspaceInput
	Body SurveyCreate
}

type updateSurveyInput struct {
	SurveyPath
	Body SurveyUpdate
}

type reopenSurveyInput struct {
	SurveyPath
	Body SurveyReopen
}

type responsesInput struct {
	SurveyPath
	Offset int `query:"offset" minimum:"0" default:"0"`
	Limit  int `query:"limit" minimum:"1" maximum:"100" default:"20"`
}

type textAnswersInput struct {
	SurveyPath
	Key    string `path:"key" pattern:"^q[0-9]{1,3}$" doc:"The question's key"`
	Search string `query:"search" maxLength:"200"`
	Offset int    `query:"offset" minimum:"0" default:"0"`
	Limit  int    `query:"limit" minimum:"1" maximum:"100" default:"20"`
}

type PollContactPath struct {
	api.WorkspaceInput
	ContactID string `path:"contactId" format:"uuid" doc:"Contact id"`
}

// PollTestAnswersCleared says how many test answers were deleted.
type PollTestAnswersCleared struct {
	Deleted int64 `json:"deleted"`
}

// csvFile is a CSV download.
type csvFile struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

func csvOut(e Export) *csvFile {
	return &csvFile{ContentType: "text/csv; charset=utf-8", ContentDisposition: fmt.Sprintf(`attachment; filename="%s"`, e.Filename), CacheControl: "no-store", Body: e.Body}
}

// PollSource is a poll or a survey the automation trigger "Answered a poll
// or survey" can start from.
type PollSource struct {
	// Ref is "poll:<id>" or "survey:<id>", as the trigger stores it.
	Ref      string       `json:"ref"`
	Kind     string       `json:"kind" enum:"poll,survey"`
	Name     string       `json:"name"`
	Campaign string       `json:"campaign,omitempty"`
	PollKind string       `json:"pollKind,omitempty" enum:"choice,scale,emoji"`
	Status   string       `json:"status"`
	ClosesAt *time.Time   `json:"closesAt,omitempty"`
	Answers  []PollOption `json:"answers"`
}

// PollOption is one answer a trigger can filter on.
type PollOption struct {
	Key   int    `json:"key"`
	Label string `json:"label"`
}

// PollSourceList is the workspace's polls and surveys.
type PollSourceList struct {
	Items []PollSource `json:"items"`
}

// Sources lists the polls (of sent and scheduled campaigns and of drafts
// tested already) and the surveys of a workspace.
func (s *Service) Sources(ctx context.Context, ws *store.Workspace) (PollSourceList, error) {
	out := PollSourceList{Items: []PollSource{}}
	rows, err := s.Q.PollsOfWorkspace(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	now := s.now()
	for _, r := range rows {
		_, answers := configOf(r.BlockID, r.Kind, r.Question, r.Answers, []byte(`{}`))
		src := PollSource{Ref: "poll:" + r.ID.String(), Kind: "poll", Name: r.Question, Campaign: r.CampaignName, PollKind: r.Kind, Status: "open", ClosesAt: r.ClosesAt, Answers: []PollOption{}}
		if r.ClosedAt != nil || (r.ClosesAt != nil && !now.Before(*r.ClosesAt)) {
			src.Status = "closed"
		}
		for _, a := range answers {
			src.Answers = append(src.Answers, PollOption{Key: a.Key, Label: a.Label})
		}
		out.Items = append(out.Items, src)
	}
	surveys, err := s.Surveys(ctx, ws)
	if err != nil {
		return out, err
	}
	for _, sv := range surveys.Items {
		out.Items = append(out.Items, PollSource{Ref: "survey:" + sv.ID, Kind: "survey", Name: sv.Name, Status: sv.Status, ClosesAt: sv.ClosesAt, Answers: []PollOption{}})
	}
	return out, nil
}

// Register adds the poll and survey operations.
func Register(h huma.API, s *Service) {
	ptags := []string{"Polls"}
	stags := []string{"Surveys"}
	view, edit := api.Workspace(authz.ReportsView), api.Workspace(authz.CampaignsEdit)

	api.Get(h, api.Op{ID: "list-campaign-polls", Path: "/workspaces/{workspace}/campaigns/{campaignId}/polls",
		Summary:     "Results of a campaign's polls: answers by answer, NPS, breakdowns, the first 72 hours",
		Description: "Answers are recorded on the answer page, never on the click: automated visits, answers from people who aren't contacts and test answers are counted apart.",
		Tags:        ptags, Access: view},
		func(ctx context.Context, in *PollCampaignPath) (*api.Body[PollResultsList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CampaignPolls(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-poll-results", Path: "/workspaces/{workspace}/polls/{pollId}", Summary: "One poll's results", Tags: ptags, Access: view},
		func(ctx context.Context, in *PollPath) (*api.Body[PollResults], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.PollResultsOf(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-poll-comments", Path: "/workspaces/{workspace}/polls/{pollId}/comments",
		Summary: "A poll's comments, newest first", Description: "Contacts' names are included for people with subscribers:view.", Tags: ptags, Access: view},
		func(ctx context.Context, in *commentsInput) (*api.Body[PollCommentList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Comments(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, PollCommentsFilter{Group: in.Group, Search: in.Search, Offset: in.Offset, Limit: in.Limit})
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "close-poll", Path: "/workspaces/{workspace}/polls/{pollId}/close", Summary: "Close a poll now (logged)",
		Description: "Its answer page says when it closed and records nothing.", Tags: ptags, Access: edit},
		func(ctx context.Context, in *PollPath) (*api.Body[PollResults], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SetPollOpen(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, false)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "reopen-poll", Path: "/workspaces/{workspace}/polls/{pollId}/reopen", Summary: "Reopen a closed poll (logged)",
		Description: "Answers made after reopening count as usual; it stays open until someone closes it.", Tags: ptags, Access: edit},
		func(ctx context.Context, in *PollPath) (*api.Body[PollResults], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SetPollOpen(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, true)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-poll-results", Path: "/workspaces/{workspace}/polls/{pollId}/results.csv", Summary: "Download a poll's results as CSV, aggregate only (logged)",
		Tags: ptags, Access: view},
		func(ctx context.Context, in *PollPath) (*csvFile, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			e, err := s.ExportPollResults(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			if err != nil {
				return nil, err
			}
			return csvOut(e), nil
		})
	api.Get(h, api.Op{ID: "export-poll-answers", Path: "/workspaces/{workspace}/polls/{pollId}/answers.csv", Summary: "Download every answer with its contact as CSV (logged)",
		Tags: ptags, Access: api.Workspace(authz.SubscribersExport)},
		func(ctx context.Context, in *PollPath) (*csvFile, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			e, err := s.ExportPollAnswers(ctx, api.WorkspaceFrom(ctx), id)
			if err != nil {
				return nil, err
			}
			return csvOut(e), nil
		})
	api.Get(h, api.Op{ID: "list-poll-sources", Path: "/workspaces/{workspace}/poll-sources",
		Summary: "Polls and surveys an automation can start from", Tags: ptags, Access: api.Workspace(authz.AutomationsView)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[PollSourceList], error) {
			out, err := s.Sources(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-campaign-poll-test-answers", Path: "/workspaces/{workspace}/campaigns/{campaignId}/poll-test-answers",
		Summary: "Test answers to a campaign's polls, from test sends (never counted)", Tags: ptags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *PollCampaignPath) (*api.Body[PollTestAnswers], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.TestAnswers(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "clear-campaign-poll-test-answers", Path: "/workspaces/{workspace}/campaigns/{campaignId}/poll-test-answers",
		Summary: "Delete the test answers of a campaign's polls (logged)", Tags: ptags, Access: edit, Status: 200},
		func(ctx context.Context, in *PollCampaignPath) (*api.Body[PollTestAnswersCleared], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			n, err := s.ClearTestAnswers(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(PollTestAnswersCleared{Deleted: n}), err
		})
	api.Get(h, api.Op{ID: "contact-field-sources", Path: "/workspaces/{workspace}/contacts/{contactId}/field-sources",
		Summary: "The fields of a contact that a poll or a survey set, and when", Tags: ptags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *PollContactPath) (*api.Body[FieldSources], error) {
			id, err := uuid.Parse(in.ContactID)
			if err != nil {
				return nil, api.NotFound("Contact")
			}
			out, err := s.ContactFieldSources(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})

	// Surveys, built in Forms & pages › Forms.
	read, formsEdit := api.Workspace(authz.SubscribersView), api.Workspace(authz.FormsEdit)
	api.Get(h, api.Op{ID: "list-surveys", Path: "/workspaces/{workspace}/surveys", Summary: "Surveys, newest first, with their counts", Tags: stags, Access: read},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[SurveyList], error) {
			out, err := s.Surveys(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-survey", Path: "/workspaces/{workspace}/surveys", Summary: "Start a survey (a draft with no questions; logged)",
		Tags: stags, Access: formsEdit, Status: 201},
		func(ctx context.Context, in *createSurveyInput) (*api.Body[Survey], error) {
			out, err := s.CreateSurvey(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-survey", Path: "/workspaces/{workspace}/surveys/{surveyId}", Summary: "A survey: its questions, settings and counts", Tags: stags, Access: read},
		func(ctx context.Context, in *SurveyPath) (*api.Body[Survey], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			out, err := s.GetSurvey(ctx, ws, id)
			out.CanEdit, out.EditHelpers = p.Can(authz.FormsEdit), []string{}
			if err == nil && !out.CanEdit {
				out.EditHelpers = nonNil(permcheck.Helpers(ctx, s.Q, ws.ID, authz.FormsEdit))
			}
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-survey", Path: "/workspaces/{workspace}/surveys/{surveyId}",
		Summary:     "Change a survey's name, address, questions or settings (logged)",
		Description: "Edits to an open survey apply to new responses; answers already given stay as they were given.", Tags: stags, Access: formsEdit},
		func(ctx context.Context, in *updateSurveyInput) (*api.Body[Survey], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateSurvey(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "publish-survey", Path: "/workspaces/{workspace}/surveys/{surveyId}/publish", Summary: "Publish a survey: its page goes live (logged)",
		Description: "Every question must be complete. The page records answers only when someone presses Send.", Tags: stags, Access: formsEdit},
		func(ctx context.Context, in *SurveyPath) (*api.Body[Survey], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.PublishSurvey(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "close-survey", Path: "/workspaces/{workspace}/surveys/{surveyId}/close", Summary: "Close a survey: its page takes no answers (logged)",
		Tags: stags, Access: formsEdit},
		func(ctx context.Context, in *SurveyPath) (*api.Body[Survey], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SetSurveyOpen(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, false, SurveyReopen{})
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "reopen-survey", Path: "/workspaces/{workspace}/surveys/{surveyId}/reopen", Summary: "Reopen a closed survey, with a new closing date or none (logged)",
		Tags: stags, Access: formsEdit},
		func(ctx context.Context, in *reopenSurveyInput) (*api.Body[Survey], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SetSurveyOpen(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, true, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-survey", Path: "/workspaces/{workspace}/surveys/{surveyId}", Summary: "Delete a survey; its page stops answering (logged)",
		Tags: stags, Access: formsEdit},
		func(ctx context.Context, in *SurveyPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.DeleteSurvey(ctx, api.WorkspaceFrom(ctx), id)
		})
	api.Get(h, api.Op{ID: "get-survey-results", Path: "/workspaces/{workspace}/surveys/{surveyId}/results",
		Summary: "A survey's completion and per-question results", Tags: stags, Access: view},
		func(ctx context.Context, in *SurveyPath) (*api.Body[SurveyResults], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SurveyResultsOf(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-survey-responses", Path: "/workspaces/{workspace}/surveys/{surveyId}/responses",
		Summary: "Completed responses, newest first", Description: "Contacts are named for people with subscribers:view.", Tags: stags, Access: view},
		func(ctx context.Context, in *responsesInput) (*api.Body[SurveyResponseList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Responses(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Offset, in.Limit)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-survey-text-answers", Path: "/workspaces/{workspace}/surveys/{surveyId}/questions/{key}/answers",
		Summary: "A text question's answers, newest first", Tags: stags, Access: view},
		func(ctx context.Context, in *textAnswersInput) (*api.Body[SurveyTextAnswerList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.TextAnswers(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Key, in.Search, in.Offset, in.Limit)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-survey-results", Path: "/workspaces/{workspace}/surveys/{surveyId}/results.csv",
		Summary: "Download a survey's results as CSV, aggregate only (logged)", Tags: stags, Access: view},
		func(ctx context.Context, in *SurveyPath) (*csvFile, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			e, err := s.ExportSurveyResults(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			if err != nil {
				return nil, err
			}
			return csvOut(e), nil
		})
	api.Get(h, api.Op{ID: "export-survey-answers", Path: "/workspaces/{workspace}/surveys/{surveyId}/answers.csv",
		Summary: "Download every response with its contact as CSV (logged)", Tags: stags, Access: api.Workspace(authz.SubscribersExport)},
		func(ctx context.Context, in *SurveyPath) (*csvFile, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			e, err := s.ExportSurveyAnswers(ctx, api.WorkspaceFrom(ctx), id)
			if err != nil {
				return nil, err
			}
			return csvOut(e), nil
		})
}
