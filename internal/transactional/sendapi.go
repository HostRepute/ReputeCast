package transactional

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

var tagMessages = []string{"Transactional messages"}

type sendInput struct {
	api.WorkspaceInput
	IdempotencyKey string `header:"Idempotency-Key" maxLength:"255" doc:"A retry with the same key within 24 hours never sends twice"`
	Body           TransactionalSend
}

type sendOutput struct {
	Status int
	Body   TransactionalSendResult
}

type batchInput struct {
	api.WorkspaceInput
	IdempotencyKey string `header:"Idempotency-Key" maxLength:"255" doc:"A retry of the same batch within 24 hours never sends twice"`
	Body           TransactionalBatch
}

// RegisterSending adds the send and batch operations.
func RegisterSending(h huma.API, s *Service) {
	send := api.Workspace(authz.TransactionalSend)
	api.Post(h, api.Op{ID: "send-transactional-message", Path: Base + "/messages", Summary: "Send one message with a template's live version",
		Description: "Answers 202 with the message ID and status queued (scheduled with send_at). Errors are RFC 9457 problems whose detail says what to fix: " +
			"template_not_found, template_not_published, missing_variables (listed in missing), invalid_recipient, recipient_suppressed (with the reason and date), " +
			"sender_not_verified, attachments_too_large (naming the route's provider limit), rate_limited (with Retry-After), no_healthy_route. " +
			"Send an Idempotency-Key header: a retry with the same key within 24 hours gets the first answer and sends nothing. " +
			"A test-mode key delivers only to the workspace's members and its allowed test addresses.",
		Tags: tagMessages, Access: send, Status: http.StatusAccepted, MaxBodyBytes: 48 << 20},
		func(ctx context.Context, in *sendInput) (*sendOutput, error) {
			if err := refuseAgents(ctx); err != nil {
				return nil, err
			}
			status, raw, err := s.Send(ctx, api.WorkspaceFrom(ctx), in.Body, in.IdempotencyKey, "api")
			if err != nil {
				return nil, err
			}
			return sendAnswer(status, raw)
		})
	api.Post(h, api.Op{ID: "send-transactional-batch", Path: Base + "/messages/batch", Summary: "Send up to 100 messages, each with its own recipient and data",
		Description: "Each message gets its own answer in results, in the order sent: its ID and status, or the problem that stopped it (the same codes as one send).",
		Tags:        tagMessages, Access: send, MaxBodyBytes: 64 << 20},
		func(ctx context.Context, in *batchInput) (*api.Body[TransactionalBatchResult], error) {
			if err := refuseAgents(ctx); err != nil {
				return nil, err
			}
			raw, err := s.SendBatch(ctx, api.WorkspaceFrom(ctx), in.Body, in.IdempotencyKey)
			if err != nil {
				return nil, err
			}
			var out TransactionalBatchResult
			if err := json.Unmarshal(raw, &out); err != nil {
				return nil, err
			}
			return api.OK(out), nil
		})
}

// sendAnswer turns a send's (possibly remembered) answer into the response.
func sendAnswer(status int, raw json.RawMessage) (*sendOutput, error) {
	if status >= 400 {
		var p Problem
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		if p.Status == 0 {
			p.Status = status
		}
		return nil, &p
	}
	var res TransactionalSendResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}
	return &sendOutput{Status: http.StatusAccepted, Body: res}, nil
}
