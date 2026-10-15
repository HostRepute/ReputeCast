package transactional

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type logInput struct {
	api.WorkspaceInput
	Status   string `query:"status" enum:"all,queued,sent,delivered,deferred,bounced,complained,rejected,failed,test" default:"all"`
	Template string `query:"template" maxLength:"64" doc:"A template's ID, or raw for mail that arrived by SMTP"`
	Provider string `query:"provider" maxLength:"36" doc:"A sending provider's ID"`
	Key      string `query:"key" maxLength:"36" doc:"An API key's ID"`
	Range    string `query:"range" enum:"1h,24h,7d,30d,custom" default:"24h"`
	From     string `query:"from" maxLength:"10" doc:"First day of a custom range (YYYY-MM-DD, the workspace's time zone), within the last 90 days"`
	To       string `query:"to" maxLength:"10" doc:"Last day of a custom range"`
	Query    string `query:"q" maxLength:"254" doc:"Recipient, message ID, idempotency key, tag or metadata value"`
	Before   string `query:"before" maxLength:"200" doc:"next of the previous page: older messages"`
	After    string `query:"after" maxLength:"200" doc:"newest of a page: only newer messages (live updates)"`
	Limit    int    `query:"limit" minimum:"1" maximum:"100" default:"25"`
}

func (in logInput) filter() LogFilter {
	return LogFilter{Status: in.Status, Template: in.Template, Provider: in.Provider, Key: in.Key, Range: in.Range, From: in.From, To: in.To,
		Query: in.Query, Before: in.Before, After: in.After, Limit: in.Limit}
}

// MessagePath names a message.
type MessagePath struct {
	api.WorkspaceInput
	MessageID string `path:"messageId" maxLength:"36" doc:"The message ID the send API answered"`
}

type overviewInput struct {
	api.WorkspaceInput
	Period      string `query:"period" enum:"24h,7d,30d" default:"7d"`
	IncludeTest bool   `query:"includeTest" doc:"Count messages sent with test-mode keys"`
}

type alertPath struct {
	api.WorkspaceInput
	AlertID string `path:"alertId" maxLength:"36"`
}

type contactActivityInput struct {
	api.WorkspaceInput
	ContactID string `path:"contactId" maxLength:"36"`
}

type exportOutput struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

type resendOutput struct {
	Status int
	Body   TransactionalSendResult
}

var tagLog = []string{"Transactional messages"}

// refuseAgentsContent keeps message content from MCP agents: they read the
// log without it.
func refuseAgentsContent(ctx context.Context) error {
	if p := api.PrincipalFrom(ctx); p != nil && p.Type == authz.ActorMCPAgent {
		return problem(http.StatusForbidden, api.CodePermissionDenied, "MCP agents read the message log without message content.")
	}
	return nil
}

// RegisterMessages adds the message log operations.
func RegisterMessages(h huma.API, s *Service) {
	view, content := api.Workspace(authz.TransactionalView), api.Workspace(authz.TransactionalContent)
	api.Get(h, api.Op{ID: "list-transactional-messages", Path: Base + "/messages", Summary: "The message log, newest first, with its numbers per status",
		Description: "Messages are kept 90 days; their rendered content as long as the content retention says. Test messages show with status test.",
		Tags:        tagLog, Access: view},
		func(ctx context.Context, in *logInput) (*api.Body[TransactionalLog], error) {
			out, err := s.MessageLog(ctx, api.WorkspaceFrom(ctx), in.filter())
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-transactional-messages", Path: Base + "/messages/export", Summary: "Export the filtered log as CSV",
		Description: "Holds recipient addresses, so it needs transactional:content. Up to 50,000 rows; every export is audited.", Tags: tagLog, Access: content},
		func(ctx context.Context, in *logInput) (*exportOutput, error) {
			if err := refuseAgentsContent(ctx); err != nil {
				return nil, err
			}
			ws := api.WorkspaceFrom(ctx)
			b, _, err := s.Export(ctx, ws, in.filter())
			if err != nil {
				return nil, err
			}
			name := "transactional-messages-" + ws.Slug + "-" + s.now().UTC().Format("2006-01-02") + ".csv"
			return &exportOutput{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="` + name + `"`, CacheControl: "no-store", Body: b}, nil
		})
	api.Get(h, api.Op{ID: "get-transactional-message", Path: Base + "/messages/{messageId}", Summary: "One message: summary, timeline, request and attempts",
		Description: "Sensitive values show as ••••••. The rendered content is a separate call that needs transactional:content.", Tags: tagLog, Access: view},
		func(ctx context.Context, in *MessagePath) (*api.Body[TransactionalMessageDetail], error) {
			id, err := messageID(in.MessageID)
			if err != nil {
				return nil, err
			}
			out, err := s.Message(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-transactional-message-content", Path: Base + "/messages/{messageId}/content", Summary: "A message's rendered HTML, text and headers",
		Description: "Content can hold personal data: it needs transactional:content and every view is audited. Sensitive values are masked. Answers 410 content_removed after the content retention.",
		Tags:        tagLog, Access: content},
		func(ctx context.Context, in *MessagePath) (*api.Body[TransactionalContent], error) {
			if err := refuseAgentsContent(ctx); err != nil {
				return nil, err
			}
			id, err := messageID(in.MessageID)
			if err != nil {
				return nil, err
			}
			out, err := s.Content(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "resend-transactional-message", Path: Base + "/messages/{messageId}/resend", Summary: "Send a message again with the same data and a new message ID",
		Description: "People only. Unavailable when a sensitive variable or attachments weren't stored (resend_unavailable). Audited.",
		Tags:        tagLog, Access: api.WorkspaceHuman(authz.TransactionalSend), Status: http.StatusAccepted},
		func(ctx context.Context, in *MessagePath) (*resendOutput, error) {
			id, err := messageID(in.MessageID)
			if err != nil {
				return nil, err
			}
			out, err := s.Resend(ctx, api.WorkspaceFrom(ctx), id)
			if err != nil {
				return nil, err
			}
			return &resendOutput{Status: http.StatusAccepted, Body: out}, nil
		})
	api.Get(h, api.Op{ID: "get-transactional-overview", Path: Base + "/overview", Summary: "The period's numbers, the route, alerts, incidents and the keys that can send",
		Description: "The Templates tab's header and side cards, and the Overview's Transactional card.", Tags: tagLog, Access: view},
		func(ctx context.Context, in *overviewInput) (*api.Body[TransactionalOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Period, in.IncludeTest)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "mark-transactional-alert-seen", Path: Base + "/alerts/{alertId}/seen", Summary: "Mark a delivery alert as seen (it stays open until the numbers recover)",
		Tags: tagLog, Access: view},
		func(ctx context.Context, in *alertPath) (*api.Empty, error) {
			id, err := uuid.Parse(in.AlertID)
			if err != nil {
				return nil, api.NotFound("Alert")
			}
			return api.NoContent, s.MarkAlertSeen(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Get(h, api.Op{ID: "get-contact-transactional-activity", Path: Base + "/contacts/{contactId}", Summary: "A contact's transactional messages and category opt-outs",
		Description: "For the contact profile's activity timeline. Content stays behind transactional:content.", Tags: tagLog, Access: view},
		func(ctx context.Context, in *contactActivityInput) (*api.Body[TransactionalContactActivity], error) {
			id, err := uuid.Parse(in.ContactID)
			if err != nil {
				return nil, api.NotFound("Contact")
			}
			out, err := s.ContactActivity(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
}
