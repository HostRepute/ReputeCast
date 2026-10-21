package auditlog

import (
	"context"
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// AuditFilterInput is a path or query input shared by several operations.
type AuditFilterInput struct {
	api.WorkspaceInput
	Search    string    `query:"q" maxLength:"200" doc:"Target name, target id, request id or actor name"`
	ActorType string    `query:"actorType" enum:"user,api_key,mcp_agent,system"`
	Actor     string    `query:"actor" doc:"Actor id"`
	Action    string    `query:"action" maxLength:"80" doc:"Action group (routing) or action (routing.updated)"`
	Category  string    `query:"category" enum:"sending,routing,audience,team,settings,integrations,transactional,scoring,digests,polls,ai,earn,verification,placement,website,grow,sms" doc:"A group of actions, as in the action filter"`
	Result    string    `query:"result" enum:"allowed,denied,error"`
	Range     string    `query:"range" enum:"24h,7d,30d,90d,all" default:"7d"`
	Since     time.Time `query:"since" doc:"Overrides range"`
	Until     time.Time `query:"until"`
}

func (in *AuditFilterInput) filter() Filter {
	f := Filter{Search: in.Search, ActorType: in.ActorType, ActorID: in.Actor, Action: in.Action, Category: in.Category, Result: in.Result, Range: in.Range}
	if !in.Since.IsZero() {
		since := in.Since
		f.Since = &since
	}
	if !in.Until.IsZero() {
		until := in.Until
		f.Until = &until
	}
	return f
}

type listInput struct {
	AuditFilterInput
	Cursor string `query:"cursor" maxLength:"200"`
	Limit  int    `query:"limit" minimum:"1" maximum:"200" default:"50"`
}

type exportInput struct {
	AuditFilterInput
	Format string `query:"format" enum:"csv,json" default:"csv"`
}

// Register adds the audit log operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Audit log"}
	base := "/workspaces/{workspace}/audit"

	api.Get(h, api.Op{ID: "list-audit-events", Path: base, Summary: "List audit entries with filters, newest first", Tags: tags, Access: api.Workspace(authz.AuditView)},
		func(ctx context.Context, in *listInput) (*api.Body[AuditEventList], error) {
			f := in.filter()
			f.Cursor, f.Limit = in.Cursor, in.Limit
			out, err := s.List(ctx, api.WorkspaceFrom(ctx), f)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-audit-event", Path: base + "/events/{eventId}", Summary: "One audit entry with before/after values, IP, user agent and request id", Tags: tags, Access: api.Workspace(authz.AuditView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			EventID string `path:"eventId" format:"uuid"`
		}) (*api.Body[AuditEventDetail], error) {
			id, err := uuid.Parse(in.EventID)
			if err != nil {
				return nil, api.NotFound("Audit entry")
			}
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "export-audit-events", Path: base + "/export", Summary: "Export matching entries as CSV or JSON (streamed)", Tags: tags, Access: api.Workspace(authz.AuditView)},
		func(ctx context.Context, in *exportInput) (*huma.StreamResponse, error) {
			ws := api.WorkspaceFrom(ctx)
			if _, err := in.filter().params(ws.ID, time.Now()); err != nil {
				return nil, err
			}
			f := in.filter()
			format := in.Format
			return &huma.StreamResponse{Body: func(hctx huma.Context) {
				ext, ct := "csv", "text/csv; charset=utf-8"
				if format == "json" {
					ext, ct = "json", "application/json"
				}
				hctx.SetHeader("Content-Type", ct)
				hctx.SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="audit-%s-%s.%s"`, ws.Slug, time.Now().UTC().Format("2006-01-02"), ext))
				hctx.SetHeader("Cache-Control", "no-store")
				hctx.SetStatus(200)
				_ = s.Export(ctx, ws, f, format, hctx.BodyWriter())
			}}, nil
		})

	api.Get(h, api.Op{ID: "get-audit-settings", Path: base + "/settings", Summary: "Retention and the webhook stream of the audit log", Tags: tags, Access: api.Workspace(authz.AuditView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[AuditSettings], error) {
			out, err := s.Settings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "update-audit-settings", Path: base + "/settings", Summary: "Change retention or stream the log to a webhook", Tags: tags, Access: api.WorkspaceHuman(authz.WorkspaceManage)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body AuditSettingsInput
		}) (*api.Body[AuditSettingsResult], error) {
			out, err := s.UpdateSettings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
}
