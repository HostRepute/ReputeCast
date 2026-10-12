package webhooks

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type hookInput struct {
	api.WorkspaceInput
	WebhookID string `path:"webhookId" format:"uuid"`
}

func (in *hookInput) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.WebhookID)
	if err != nil {
		return uuid.Nil, api.NotFound("Webhook")
	}
	return id, nil
}

type deliveryInput struct {
	api.WorkspaceInput
	DeliveryID string `path:"deliveryId" format:"uuid"`
}

type deliveriesInput struct {
	api.WorkspaceInput
	WebhookID string `query:"webhookId" doc:"Only this endpoint"`
	State     string `query:"state" enum:"pending,delivered,retrying,dead,held,failed"`
	Limit     int    `query:"limit" minimum:"1" maximum:"200" default:"50"`
}

// Register adds the webhooks operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Webhooks"}
	base := "/workspaces/{workspace}/webhooks"
	access := api.Workspace(authz.WebhooksManage)

	api.Get(h, api.Op{ID: "list-webhooks", Path: base, Summary: "List outgoing webhook endpoints with 24-hour delivery stats", Tags: tags, Access: access},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[WebhookList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "create-webhook", Path: base, Summary: "Add an endpoint; the signing secret is shown once", Tags: tags, Access: access, Status: 201},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body WebhookInput
		}) (*api.Body[WebhookCreated], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "update-webhook", Path: base + "/{webhookId}", Summary: "Change an endpoint's URL, name or events", Tags: tags, Access: access},
		func(ctx context.Context, in *struct {
			hookInput
			Body WebhookInput
		}) (*api.Body[WebhookView], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Update(ctx, api.WorkspaceFrom(ctx), id, in.Body)
			return api.OK(out), err
		})

	api.Delete(h, api.Op{ID: "delete-webhook", Path: base + "/{webhookId}", Summary: "Delete an endpoint and its delivery log", Tags: tags, Access: access},
		func(ctx context.Context, in *hookInput) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Delete(ctx, api.WorkspaceFrom(ctx), id)
		})

	api.Post(h, api.Op{ID: "rotate-webhook-secret", Path: base + "/{webhookId}/rotate-secret", Summary: "Replace the signing secret; the new one is shown once", Tags: tags, Access: api.WorkspaceHuman(authz.WebhooksManage)},
		func(ctx context.Context, in *hookInput) (*api.Body[SecretResult], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.RotateSecret(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "pause-webhook", Path: base + "/{webhookId}/pause", Summary: "Pause an endpoint; events keep queuing", Tags: tags, Access: access, Status: 204},
		func(ctx context.Context, in *hookInput) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.SetEnabled(ctx, api.WorkspaceFrom(ctx), id, false)
		})

	api.Post(h, api.Op{ID: "resume-webhook", Path: base + "/{webhookId}/resume", Summary: "Resume an endpoint and deliver what queued, in order", Tags: tags, Access: access, Status: 204},
		func(ctx context.Context, in *hookInput) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.SetEnabled(ctx, api.WorkspaceFrom(ctx), id, true)
		})

	api.Post(h, api.Op{ID: "test-webhook", Path: base + "/{webhookId}/test", Summary: "Send a webhook.test event now and return the result", Tags: tags, Access: access},
		func(ctx context.Context, in *hookInput) (*api.Body[TestResult], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Test(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "list-webhook-deliveries", Path: base + "/deliveries", Summary: "The delivery log, newest first", Tags: tags, Access: access},
		func(ctx context.Context, in *deliveriesInput) (*api.Body[DeliveryList], error) {
			var hook *uuid.UUID
			if in.WebhookID != "" {
				id, err := uuid.Parse(in.WebhookID)
				if err != nil {
					return nil, api.Invalid("query.webhookId", "Unknown endpoint.")
				}
				hook = &id
			}
			out, err := s.Deliveries(ctx, api.WorkspaceFrom(ctx), hook, in.State, in.Limit)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-webhook-delivery", Path: base + "/deliveries/{deliveryId}", Summary: "One delivery with its payload and the endpoint's response", Tags: tags, Access: access},
		func(ctx context.Context, in *deliveryInput) (*api.Body[DeliveryDetail], error) {
			id, err := uuid.Parse(in.DeliveryID)
			if err != nil {
				return nil, api.NotFound("Delivery")
			}
			out, err := s.Delivery(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "retry-webhook-delivery", Path: base + "/deliveries/{deliveryId}/retry", Summary: "Retry a failed delivery now, or replay one from the dead-letter queue", Tags: tags, Access: access},
		func(ctx context.Context, in *deliveryInput) (*api.Body[RetryResult], error) {
			id, err := uuid.Parse(in.DeliveryID)
			if err != nil {
				return nil, api.NotFound("Delivery")
			}
			out, err := s.Retry(ctx, api.WorkspaceFrom(ctx), &id, nil)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "retry-failed-webhooks", Path: base + "/retry-failed", Summary: "Retry every failed delivery (of one endpoint, or all)", Tags: tags, Access: access},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body struct {
				WebhookID string `json:"webhookId,omitempty"`
			}
		}) (*api.Body[RetryResult], error) {
			var hook *uuid.UUID
			if in.Body.WebhookID != "" {
				id, err := uuid.Parse(in.Body.WebhookID)
				if err != nil {
					return nil, api.Invalid("body.webhookId", "Unknown endpoint.")
				}
				hook = &id
			}
			out, err := s.Retry(ctx, api.WorkspaceFrom(ctx), nil, hook)
			return api.OK(out), err
		})
}
