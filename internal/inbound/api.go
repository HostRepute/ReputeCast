package inbound

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

func init() {
	webhooks.Events = append(webhooks.Events,
		webhooks.WebhookEventInfo{Name: EventReplied, Group: "Replies", Description: "A contact replied to an email (including auto-replies).",
			DescriptionText: msgs.M("webhook.email.replied").Keep(nil)},
		webhooks.WebhookEventInfo{Name: EventReceived, Group: "Replies", Description: "Mail arrived at the receiving domain that isn't a reply.",
			DescriptionText: msgs.M("webhook.email.received").Keep(nil)},
	)
}

type inboundRouteInput struct {
	api.WorkspaceInput
	Body InboundRouteInput
}

type replyListInput struct {
	api.WorkspaceInput
	View       string `query:"view" enum:"open,done,auto,spam,other,unsubscribes" default:"open"`
	Campaign   string `query:"campaign" doc:"Only replies to this campaign" format:"uuid"`
	Automation string `query:"automation" doc:"Only replies to this flow's emails" format:"uuid"`
	Q          string `query:"q" maxLength:"200" doc:"Search the sender and subject"`
	Cursor     string `query:"cursor"`
	Limit      int    `query:"limit" minimum:"1" maximum:"100" default:"50"`
}

// ReplyPath names a reply.
type ReplyPath struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid" doc:"Reply id"`
}

type replyStatusInput struct {
	api.WorkspaceInput
	Body ReplyStatusInput
}

func optionalID(raw string) *uuid.UUID {
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}

// Register adds the receiving and replies operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Receiving mail"}
	api.Get(h, api.Op{ID: "get-inbound-route", Path: "/workspaces/{workspace}/inbound", Summary: "Get how the workspace receives mail", Tags: tags,
		Access: api.Workspace(authz.ProvidersView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[InboundRouteView], error) {
			out, err := s.View(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-inbound-route", Path: "/workspaces/{workspace}/inbound", Summary: "Set up or change receiving mail", Tags: tags,
		Access: api.WorkspaceHuman(authz.DomainsManage)},
		func(ctx context.Context, in *inboundRouteInput) (*api.Body[InboundRouteView], error) {
			out, err := s.Save(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-inbound-route", Path: "/workspaces/{workspace}/inbound", Summary: "Stop receiving mail", Tags: tags,
		Access: api.WorkspaceHuman(authz.DomainsManage)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Empty, error) {
			return nil, s.Remove(ctx, api.WorkspaceFrom(ctx))
		})
	api.Post(h, api.Op{ID: "test-inbound-route", Path: "/workspaces/{workspace}/inbound/test", Summary: "Send a test message to the receiving domain", Tags: tags,
		Access: api.WorkspaceHuman(authz.DomainsManage)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[InboundTestResult], error) {
			out, err := s.SendTest(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})

	rtags := []string{"Replies"}
	api.Get(h, api.Op{ID: "list-replies", Path: "/workspaces/{workspace}/replies", Summary: "List replies and other received mail", Tags: rtags,
		Access: api.Workspace(authz.RepliesView)},
		func(ctx context.Context, in *replyListInput) (*api.Body[ReplyList], error) {
			out, err := s.Replies(ctx, api.WorkspaceFrom(ctx), ReplyListParams{View: in.View, CampaignID: optionalID(in.Campaign),
				AutomationID: optionalID(in.Automation), Query: in.Q, Cursor: in.Cursor, Limit: in.Limit})
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-reply", Path: "/workspaces/{workspace}/replies/{id}", Summary: "Get a reply", Tags: rtags,
		Access: api.Workspace(authz.RepliesView)},
		func(ctx context.Context, in *ReplyPath) (*api.Body[ReplyDetail], error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, api.NotFound("Reply")
			}
			out, err := s.Reply(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "set-replies-status", Path: "/workspaces/{workspace}/replies/status", Summary: "Mark replies done or open", Tags: rtags,
		Access: api.WorkspaceHuman(authz.RepliesManage)},
		func(ctx context.Context, in *replyStatusInput) (*api.Body[ReplyStatusResult], error) {
			out, err := s.SetStatus(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "unsubscribe-reply-contact", Path: "/workspaces/{workspace}/replies/{id}/unsubscribe-contact",
		Summary: "Unsubscribe the contact a reply came from", Tags: rtags, Access: api.WorkspaceHuman(authz.RepliesManage)},
		func(ctx context.Context, in *ReplyPath) (*api.Body[ReplyUnsubscribeResult], error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, api.NotFound("Reply")
			}
			out, err := s.UnsubscribeContact(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
}
