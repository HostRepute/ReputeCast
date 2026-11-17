package assistant

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type workspaceInput struct {
	api.WorkspaceInput
}

type threadsInput struct {
	api.WorkspaceInput
	Q      string `query:"q" maxLength:"120" doc:"Threads whose title has this"`
	Cursor string `query:"cursor" maxLength:"200"`
	Limit  int    `query:"limit" minimum:"1" maximum:"100" default:"50"`
}

type createThreadInput struct {
	api.WorkspaceInput
	Body AIThreadInput
}

type threadPath struct {
	api.WorkspaceInput
	Thread uuid.UUID `path:"thread" format:"uuid" doc:"The thread's id"`
}

type updateThreadInput struct {
	api.WorkspaceInput
	Thread uuid.UUID `path:"thread" format:"uuid" doc:"The thread's id"`
	Body   AIThreadUpdate
}

type attachInput struct {
	api.WorkspaceInput
	Thread uuid.UUID `path:"thread" format:"uuid" doc:"The thread's id"`
	Body   AIThreadContextInput
}

type contextPath struct {
	api.WorkspaceInput
	Thread  uuid.UUID `path:"thread" format:"uuid" doc:"The thread's id"`
	Context uuid.UUID `path:"context" format:"uuid" doc:"The attached resource's id"`
}

// Register adds the assistant's thread operations. They need ai:use and
// the assistant switched on, and are people's only: API keys and MCP agents
// can't use the assistant. A thread is its owner's: another person's,
// integration managers included, isn't found.
func Register(h huma.API, s *Service) {
	tags := []string{"AI assistant"}
	base := "/workspaces/{workspace}/ai/assistant"
	use := api.WorkspaceHuman(authz.AIUse)

	api.Get(h, api.Op{ID: "list-ai-assistant-models", Path: base + "/models", Summary: "The models a thread may use: the workspace's and the caller's own",
		Description: "Each says whether a thread can use it now, and why not.", Tags: tags, Access: use},
		func(ctx context.Context, in *workspaceInput) (*api.Body[ai.AIAssistantModelList], error) {
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			if _, _, err := s.actor(ctx, s.Q, ws, p, true); err != nil {
				return nil, err
			}
			out, err := s.AI.AssistantModels(ctx, ws, p)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-ai-threads", Path: base + "/threads", Summary: "The caller's assistant threads, the most recently active first",
		Tags: tags, Access: use},
		func(ctx context.Context, in *threadsInput) (*api.Body[AIThreadList], error) {
			out, err := s.Threads(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Q, in.Cursor, in.Limit)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-ai-thread", Path: base + "/threads", Summary: "Start an assistant thread",
		Description: "On the model given, or the caller's own pick for the assistant, or the workspace's; it needs a passed tool-calling check. A retry with the same request key returns the same thread. Audited as ai_assistant_thread.created.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *createThreadInput) (*api.Body[AIThread], error) {
			out, err := s.CreateThread(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-ai-thread", Path: base + "/threads/{thread}", Summary: "One of the caller's threads: its messages, attached resources, usage and the turn answering now",
		Tags: tags, Access: use},
		func(ctx context.Context, in *threadPath) (*api.Body[AIThread], error) {
			out, err := s.Thread(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Thread)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-ai-thread", Path: base + "/threads/{thread}", Summary: "Rename a thread, or pick another model for its next turns",
		Description: "Audited as ai_assistant_thread.updated, without the title.", Tags: tags, Access: use},
		func(ctx context.Context, in *updateThreadInput) (*api.Body[AIThread], error) {
			out, err := s.UpdateThread(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Thread, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-ai-thread", Path: base + "/threads/{thread}", Status: http.StatusNoContent, Summary: "Delete a thread and all its content at once",
		Description: "Its messages, attached resources and stream events go; the activity log keeps its usage. Works with the assistant switched off. Audited as ai_assistant_thread.deleted.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *threadPath) (*api.Empty, error) {
			return api.NoContent, s.DeleteThread(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Thread)
		})
	api.Post(h, api.Op{ID: "attach-ai-thread-context", Path: base + "/threads/{thread}/contexts", Summary: "Attach a resource to a thread, after the person reviewed sharing it",
		Description: "The caller must be able to read it; every later read checks again. A report's numbers are its fixed snapshot. Audited as ai_assistant_context.add.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *attachInput) (*api.Body[AIThread], error) {
			out, err := s.AttachContext(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Thread, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "detach-ai-thread-context", Path: base + "/threads/{thread}/contexts/{context}", Status: http.StatusOK, Summary: "Remove an attached resource from a thread",
		Description: "Audited as ai_assistant_context.remove.", Tags: tags, Access: use},
		func(ctx context.Context, in *contextPath) (*api.Body[AIThread], error) {
			out, err := s.DetachContext(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Thread, in.Context)
			return api.OK(out), err
		})
}
