package assistant

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
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

type messageInput struct {
	api.WorkspaceInput
	Thread uuid.UUID `path:"thread" format:"uuid" doc:"The thread's id"`
	Body   AIThreadMessageInput
}

type turnPath struct {
	api.WorkspaceInput
	Thread uuid.UUID `path:"thread" format:"uuid" doc:"The thread's id"`
	Turn   uuid.UUID `path:"turn" format:"uuid" doc:"The turn's id"`
}

type streamInput struct {
	api.WorkspaceInput
	Thread      uuid.UUID `path:"thread" format:"uuid" doc:"The thread's id"`
	LastEventID string    `header:"Last-Event-ID" doc:"Resume after this event id"`
	// A browser's EventSource sets Last-Event-ID itself when it reconnects;
	// a client that opens a new stream passes it here.
	LastEventIDQuery string `query:"lastEventId" doc:"Same as the Last-Event-ID header, for a new stream"`
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
	api.Get(h, api.Op{ID: "get-ai-thread", Path: base + "/threads/{thread}", Summary: "One of the caller's threads: its messages, attached resources, usage and latest turn",
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
	api.Post(h, api.Op{ID: "send-ai-thread-message", Path: base + "/threads/{thread}/messages", Summary: "Send a message: it starts a turn on the thread's model",
		Description: "The answer streams on the thread's events (stream-ai-thread). One turn answers at a time in a thread (409 while one does). A question about sender reputation, blocklists or placement gets the HostRepute card at once, without a model call. A retry with the same request key returns the thread. Audited as ai_assistant_turn.created, without the text.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *messageInput) (*api.Body[AIThread], error) {
			out, err := s.SendMessage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Thread, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "stop-ai-thread-turn", Path: base + "/threads/{thread}/turns/{turn}/stop", Summary: "Stop a turn",
		Description: "No model request starts for it any more; what a request already answering writes after now is dropped, and it is still accounted (the provider may charge for it). Stopping a turn that ended changes nothing. Works with the assistant switched off. Audited as ai_assistant_turn.cancelled.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *turnPath) (*api.Body[AIThread], error) {
			out, err := s.StopTurn(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Thread, in.Turn)
			return api.OK(out), err
		})
	sse.Register(h, huma.Operation{
		OperationID: "stream-ai-thread", Method: http.MethodGet, Path: api.Version + base + "/threads/{thread}/stream",
		Summary: "Server-sent events of a thread: its turns, their answers as they arrive, its messages and usage",
		Description: "Each event's id is its number in the thread; a client that reconnects with Last-Event-ID (or lastEventId) gets every event after it, once, in order. " +
			"Without a cursor, or when the events after it are gone (24 hours after their turn, or lost in a crash), a `snapshot` event carries the whole thread first. " +
			"A `heartbeat` event every 15 s says the connection is alive. A client that doesn't read is cut, and resumes from its last event. " +
			"A `problem` event ends a stream the caller may not read (the thread isn't theirs, they lost ai:use, the assistant is off): don't reconnect after it.",
		Tags: tags, Security: []map[string][]string{{"session": {}}},
		Metadata:    map[string]any{"reputecast.access": use},
		Extensions:  map[string]any{"x-access": string(api.AccessWorkspace), "x-permission": string(authz.AIUse)},
		Middlewares: huma.Middlewares{keepWriter},
	}, map[string]any{"turn": AIThreadTurnEvent{}, "delta": AIThreadDeltaEvent{}, "message": AIThreadMessageEvent{}, "usage": AIThreadUsageEvent{},
		"snapshot": AIThreadSnapshotEvent{}, "heartbeat": AIThreadHeartbeat{}, "problem": AIThreadStreamProblem{}},
		func(ctx context.Context, in *streamInput, send sse.Sender) {
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			if ws == nil || p == nil {
				return
			}
			cursor := in.LastEventID
			if cursor == "" {
				cursor = in.LastEventIDQuery
			}
			s.Stream(ctx, ws, p, in.Thread, cursor, send)
		})
	api.Delete(h, api.Op{ID: "detach-ai-thread-context", Path: base + "/threads/{thread}/contexts/{context}", Status: http.StatusOK, Summary: "Remove an attached resource from a thread",
		Description: "Audited as ai_assistant_context.remove.", Tags: tags, Access: use},
		func(ctx context.Context, in *contextPath) (*api.Body[AIThread], error) {
			out, err := s.DetachContext(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Thread, in.Context)
			return api.OK(out), err
		})
}
