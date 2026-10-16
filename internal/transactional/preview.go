package transactional

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// TransactionalAddress is a recipient: an email and an optional name.
type TransactionalAddress struct {
	Email string `json:"email" maxLength:"254" example:"sam.rivera@outlook.com"`
	Name  string `json:"name,omitempty" maxLength:"200" example:"Sam Rivera"`
}

// TransactionalPreviewInput renders a template in the editor: its draft
// (default), its live version or a published version, with a test data
// set or data of your own. Content fields preview unsaved changes.
type TransactionalPreviewInput struct {
	Version  string                `json:"version,omitempty" doc:"draft (default), live, or a version number" example:"draft"`
	TestData string                `json:"testData,omitempty" maxLength:"80" doc:"The name of a test data set"`
	Data     map[string]any        `json:"data,omitempty" doc:"Data to render with instead of a test data set"`
	To       *TransactionalAddress `json:"to,omitempty"`
	// Unsaved editor content (draft only).
	Subject   *string                  `json:"subject,omitempty" maxLength:"998"`
	Preheader *string                  `json:"preheader,omitempty" maxLength:"500"`
	MJML      *string                  `json:"mjml,omitempty" maxLength:"2000000"`
	HTML      *string                  `json:"html,omitempty" maxLength:"2500000"`
	PlainText *string                  `json:"plainText,omitempty" maxLength:"500000"`
	Variables *[]TransactionalVariable `json:"variables,omitempty" maxItems:"200"`
}

// TransactionalPreview is a rendered template.
type TransactionalPreview struct {
	Version   string `json:"version"`
	Subject   string `json:"subject"`
	Preheader string `json:"preheader"`
	HTML      string `json:"html"`
	Text      string `json:"text"`
	FromName  string `json:"fromName"`
	FromEmail string `json:"fromEmail"`
	ReplyTo   string `json:"replyTo"`
	SizeBytes int    `json:"sizeBytes"`
	// OK is false when the data lacks a required variable or Liquid fails;
	// Missing and Problem say why.
	OK       bool     `json:"ok"`
	Missing  []string `json:"missing"`
	Defaults []string `json:"defaults"`
	Problem  string   `json:"problem,omitempty"`
}

// Preview renders a template version for the editor (nothing is sent).
func (s *Service) Preview(ctx context.Context, ws *store.Workspace, ref string, in TransactionalPreviewInput) (TransactionalPreview, error) {
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return TransactionalPreview{}, err
	}
	t, err := s.Q.TxnTemplate(ctx, store.TxnTemplateParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return TransactionalPreview{}, mapTemplateErr(err)
	}
	version := strings.ToLower(strings.TrimSpace(in.Version))
	if version == "" {
		version = "draft"
	}
	var c content
	var testData []TransactionalTestData
	senderID, replyTo := t.SenderID, t.ReplyTo
	switch version {
	case "draft":
		c = content{Subject: t.Subject, Preheader: t.Preheader, EditorMode: t.EditorMode, MJML: t.Mjml, HTML: t.Html, Text: t.PlainText, Variables: parseVariables(t.Variables)}
		testData = parseTestData(t.TestData)
		if in.Subject != nil {
			c.Subject = *in.Subject
		}
		if in.Preheader != nil {
			c.Preheader = *in.Preheader
		}
		if in.MJML != nil {
			c.MJML = *in.MJML
			if in.HTML == nil {
				c.HTML = ""
			}
		}
		if in.HTML != nil {
			c.HTML = *in.HTML
		}
		if in.PlainText != nil {
			c.Text = *in.PlainText
		}
		if in.Variables != nil {
			c.Variables = *in.Variables
		}
	default:
		n := 0
		if version == "live" {
			if t.LiveVersion == nil {
				return TransactionalPreview{}, problem(http.StatusUnprocessableEntity, CodeTemplateNotPublished, fmt.Sprintf("%s has no published version yet.", t.Key))
			}
			n = int(*t.LiveVersion)
		} else if v, err := strconv.Atoi(strings.TrimPrefix(version, "v")); err == nil && v > 0 {
			n = v
		} else {
			return TransactionalPreview{}, api.Invalid("body.version", "Pick draft, live or a version number.")
		}
		v, err := s.Q.TxnVersion(ctx, store.TxnVersionParams{TemplateID: id, Version: int32(n)})
		if err != nil {
			return TransactionalPreview{}, api.MapError(err, "Version")
		}
		c = content{Subject: v.Subject, Preheader: v.Preheader, EditorMode: v.EditorMode, MJML: v.Mjml, HTML: v.Html, Text: v.PlainText, Variables: parseVariables(v.Variables)}
		testData = parseTestData(v.TestData)
		senderID, replyTo = v.SenderID, v.ReplyTo
		version = "v" + strconv.Itoa(n)
	}
	data := in.Data
	if data == nil {
		switch {
		case in.TestData != "":
			found := false
			for _, td := range testData {
				if strings.EqualFold(td.Name, in.TestData) {
					data, found = td.Data, true
					break
				}
			}
			if !found {
				return TransactionalPreview{}, api.Invalid("body.testData", fmt.Sprintf("There is no test data set called “%s”.", in.TestData))
			}
		case len(testData) > 0:
			data = testData[0].Data
		default:
			data = sampleData(c.Variables)
		}
	}
	to := recipient{Email: "recipient@example.com", Name: "Sam Rivera"}
	if in.To != nil && in.To.Email != "" {
		to = recipient{Email: in.To.Email, Name: in.To.Name}
	}
	out := TransactionalPreview{Version: version, Missing: missingRequired(c.Variables, data), Defaults: []string{}, ReplyTo: replyTo}
	if senderID != nil {
		if snd, err := s.Q.TxnSenderByID(ctx, store.TxnSenderByIDParams{WorkspaceID: ws.ID, ID: *senderID}); err == nil {
			out.FromName, out.FromEmail = snd.Name, snd.Email
			if out.ReplyTo == "" {
				out.ReplyTo = snd.ReplyTo
			}
		}
	}
	for _, v := range c.Variables {
		if val, ok := data[v.Name]; (!ok || val == nil) && !v.Required && v.Default != "" {
			out.Defaults = append(out.Defaults, v.Name)
		}
	}
	// A preview always renders: missing required variables print nothing,
	// and the answer says which ones a real call would need.
	r, rerr := render(c, data, to, map[string]string{"unsubscribe_url": "#unsubscribe", "preferences_url": "#preferences"})
	if rerr != nil {
		// A template that doesn't render is the preview's answer, not a failure.
		out.Problem = rerr.Error()
		return out, nil //nolint:nilerr // the preview's answer, not a failure
	}
	out.Subject, out.Preheader, out.HTML, out.Text, out.SizeBytes = r.Subject, r.Preheader, r.HTML, r.Text, len(r.HTML)
	out.OK = len(out.Missing) == 0
	if !out.OK {
		out.Problem = fmt.Sprintf("%s is required and missing. A real call shaped like this fails with missing_variables.", joinAnd(out.Missing))
	}
	return out, nil
}

// TransactionalRenderInput is the API's render: the live version with the
// data a call would send.
type TransactionalRenderInput struct {
	Data map[string]any        `json:"data,omitempty" doc:"The template's variables"`
	To   *TransactionalAddress `json:"to,omitempty" doc:"Fills to.email and to.name"`
	// Language picks a language version, as a send does.
	Language string `json:"language,omitempty" maxLength:"16" example:"es" doc:"The language version to render; the linked contact's language, then the fallback, when omitted"`
}

// TransactionalRendered is a rendered live version.
type TransactionalRendered struct {
	Template string `json:"template" example:"password-reset"`
	Version  int    `json:"version" example:"3"`
	Subject  string `json:"subject"`
	HTML     string `json:"html"`
	Text     string `json:"text"`
	// Language is the language version rendered.
	Language string `json:"language,omitempty" example:"es"`
}

// liveTemplate is a template's live version, as the API sends it.
type liveTemplate struct {
	ID           uuid.UUID
	Key          string
	Name         string
	Version      int
	CategoryID   *uuid.UUID
	CategoryName string
	CategoryKind string
	SenderID     *uuid.UUID
	ReplyTo      string
	TrackOpens   bool
	TrackClicks  bool
	Content      content
	UpdatedAt    time.Time
}

// live loads a template's live version by key, with the API's errors.
func (s *Service) live(ctx context.Context, q *store.Queries, wsID uuid.UUID, key string) (liveTemplate, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return liveTemplate{}, problem(http.StatusUnprocessableEntity, CodeTemplateNotFound, "Name the template to send in template, such as \"password-reset\".")
	}
	id, err := q.TxnTemplateIDByKey(ctx, store.TxnTemplateIDByKeyParams{WorkspaceID: wsID, Key: key})
	if err != nil {
		return liveTemplate{}, problem(http.StatusUnprocessableEntity, CodeTemplateNotFound, fmt.Sprintf("There is no template with the ID %s in this workspace. Check the ID on Transactional › Templates.", key))
	}
	t, err := q.TxnTemplate(ctx, store.TxnTemplateParams{WorkspaceID: wsID, ID: id})
	if err != nil {
		return liveTemplate{}, problem(http.StatusUnprocessableEntity, CodeTemplateNotFound, fmt.Sprintf("There is no template with the ID %s in this workspace. Check the ID on Transactional › Templates.", key))
	}
	if t.ArchivedAt != nil {
		return liveTemplate{}, problem(http.StatusUnprocessableEntity, CodeTemplateNotFound, fmt.Sprintf("%s is archived, so nothing is sent. Restore it on Transactional › Templates to send it again.", key))
	}
	if t.LiveVersion == nil {
		return liveTemplate{}, problem(http.StatusUnprocessableEntity, CodeTemplateNotPublished, fmt.Sprintf("%s has a draft but no published version, so there is nothing to send. Publish it first, then call again with the same idempotency key.", key))
	}
	v, err := q.TxnVersion(ctx, store.TxnVersionParams{TemplateID: id, Version: *t.LiveVersion})
	if err != nil {
		return liveTemplate{}, err
	}
	return liveTemplate{
		ID: id, Key: t.Key, Name: t.Name, Version: int(v.Version), CategoryID: t.CategoryID, CategoryName: t.CategoryName, CategoryKind: t.CategoryKind,
		SenderID: v.SenderID, ReplyTo: v.ReplyTo, TrackOpens: v.TrackOpens, TrackClicks: v.TrackClicks, UpdatedAt: t.UpdatedAt,
		Content: content{Subject: v.Subject, Preheader: v.Preheader, EditorMode: v.EditorMode, MJML: v.Mjml, HTML: v.Html, Text: v.PlainText, Variables: parseVariables(v.Variables)},
	}, nil
}

// missingProblem answers a call that lacks required variables.
func missingProblem(key string, version int, missing []string) *Problem {
	p := problem(http.StatusUnprocessableEntity, CodeMissingVariables,
		fmt.Sprintf("%s v%d needs %s. Add %s to data and call again.", key, version, joinAnd(missing), plural(len(missing), "it", "them")))
	p.Title = "Missing variables"
	p.Missing = missing
	return p
}

// Render renders a template's live version with a call's data, as the API
// would send it (nothing is sent or logged).
func (s *Service) Render(ctx context.Context, ws *store.Workspace, key string, in TransactionalRenderInput) (TransactionalRendered, error) {
	lt, err := s.live(ctx, s.Q, ws.ID, key)
	if err != nil {
		return TransactionalRendered{}, err
	}
	data := in.Data
	if data == nil {
		data = map[string]any{}
	}
	if missing := missingRequired(lt.Content.Variables, data); len(missing) > 0 {
		return TransactionalRendered{}, missingProblem(lt.Key, lt.Version, missing)
	}
	to := recipient{}
	if in.To != nil {
		to = recipient{Email: strings.ToLower(strings.TrimSpace(in.To.Email)), Name: in.To.Name}
	}
	cnt, lang, err := s.renderLanguage(ctx, ws, lt, in.Language, to.Email)
	if err != nil {
		return TransactionalRendered{}, err
	}
	r, err := render(cnt, data, to, map[string]string{"unsubscribe_url": "#unsubscribe", "preferences_url": "#preferences"})
	if err != nil {
		return TransactionalRendered{}, problem(http.StatusUnprocessableEntity, CodeTemplateDoesntRun, fmt.Sprintf("%s v%d doesn't render with this data: %s", lt.Key, lt.Version, err))
	}
	return TransactionalRendered{Template: lt.Key, Version: lt.Version, Subject: r.Subject, HTML: r.HTML, Text: r.Text, Language: lang}, nil
}
