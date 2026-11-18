package assistant

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mcpclient"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// Tools of remote MCP servers (assistant brief D2, §3.6; B5). A thread
// offers the approved tools of the servers its owner picked, after
// ReputeCast's own tools and within the same 40. A read-only tool runs
// without asking; a call of a writing tool stops the turn ("waiting") until
// its owner allows or denies that exact call, and runs at most once. What
// a server answers reaches the model inside <untrusted_content> tags, as
// data: nothing in it is followed.

// Servers is what the assistant uses of internal/mcpclient, each as the
// owner is now. *mcpclient.Service is one.
type Servers interface {
	ThreadTools(ctx context.Context, q *store.Queries, ws *store.Workspace, owner *authz.Principal, thread uuid.UUID) ([]mcpclient.ThreadTool, error)
	Current(ctx context.Context, q *store.Queries, ws *store.Workspace, owner *authz.Principal, toolID uuid.UUID, hash []byte) (store.McpTool, store.McpServer, error)
	Call(ctx context.Context, ws *store.Workspace, owner *authz.Principal, thread uuid.UUID, callID string, toolID uuid.UUID, hash []byte, args json.RawMessage) (mcpclient.Result, error)
	Usable(ctx context.Context, ws *store.Workspace, p *authz.Principal) (mcpclient.RemoteMCPServerList, error)
}

var _ Servers = (*mcpclient.Service)(nil)

const kindConfirmation = "confirmation"

// AIThreadConfirmation is a call of a writing MCP tool waiting for the
// thread's owner: the server, the tool and the exact arguments it will be
// sent with.
type AIThreadConfirmation struct {
	ID         uuid.UUID  `json:"id"`
	TurnID     uuid.UUID  `json:"turnId"`
	ServerID   *uuid.UUID `json:"serverId,omitempty"`
	ServerName string     `json:"serverName"`
	Tool       string     `json:"tool" doc:"The tool's name on its server"`
	ModelName  string     `json:"modelName" doc:"The tool's name for the model: mcp__<server>__<tool>"`
	// Arguments are exactly what the call sends once allowed.
	Arguments json.RawMessage `json:"arguments"`
	// State: expired after a day without the owner's decision (the turn
	// ended, nothing was sent).
	State     string     `json:"state" enum:"pending,allowed,denied,calling,done,expired"`
	Decision  string     `json:"decision,omitempty" enum:"allow,deny"`
	CreatedAt time.Time  `json:"createdAt"`
	DecidedAt *time.Time `json:"decidedAt,omitempty"`
}

// AIThreadConfirmationEvent: a call waits for the owner, or was decided.
type AIThreadConfirmationEvent struct {
	Confirmation AIThreadConfirmation `json:"confirmation"`
}

// AIThreadConfirmationDecision is the owner's decision on one call.
type AIThreadConfirmationDecision struct {
	Decision string `json:"decision" enum:"allow,deny" doc:"allow runs this call once; deny tells the model it was refused"`
}

// AIThreadServersInput is the MCP servers a thread may use.
type AIThreadServersInput struct {
	ServerIDs []uuid.UUID `json:"serverIds" maxItems:"20" doc:"Servers from list-ai-assistant-mcp-servers; empty: none"`
}

func confirmationAD(id uuid.UUID, part string) string {
	return "ai_assistant_confirmation:" + id.String() + ":" + part
}

// round is a response's tool calls and the results of those that ran, to
// go on after a confirmation.
type round struct {
	Calls   []llm.ToolCall    `json:"calls"`
	Results []llm.Part        `json:"results"`
	Rows    []AIThreadToolRow `json:"rows"`
}

// resume is a turn going on after its owner decided on a call.
type resume struct {
	conf  store.AiAssistantConfirmation
	round round
}

// What of a server's tool definition reaches the model: its description,
// at most 1,000 characters; its input schema, at most 8 KB.
const (
	maxToolDescription = 1000
	maxToolSchema      = 8 << 10
)

// mcpSpecs are the MCP tools a thread may offer, at most room of them.
func (s *Service) mcpSpecs(ctx context.Context, q *store.Queries, ws *store.Workspace, owner *authz.Principal, thread uuid.UUID, room int) ([]ToolSpec, error) {
	if s.MCP == nil || thread == uuid.Nil || room <= 0 {
		return nil, nil
	}
	list, err := s.MCP.ThreadTools(ctx, q, ws, owner, thread)
	if err != nil {
		return nil, err
	}
	out := []ToolSpec{}
	for i := range list {
		t := list[i]
		desc := "From the MCP server " + tagSafe(t.ServerName) + ", outside ReputeCast"
		if t.Writes {
			desc += "; each call waits for the person to allow it"
		}
		desc += ". " + defang(clip(t.Description, maxToolDescription))
		params := t.InputSchema
		if params["type"] != "object" {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		// A schema is the server's text too: one too large to read isn't
		// offered.
		if raw, err := json.Marshal(params); err != nil || len(raw) > maxToolSchema {
			continue
		}
		out = append(out, ToolSpec{Name: t.ModelName, Description: desc, Parameters: params, Permission: authz.AIUse, Confirm: t.Writes, mcp: &t})
		if len(out) == room {
			break
		}
	}
	return out, nil
}

// runMCP calls an MCP tool as the owner, at the hash it was offered (or
// confirmed) with.
func (s *Service) runMCP(ctx context.Context, c *toolCall, spec ToolSpec, hash []byte) (toolResult, json.RawMessage) {
	t := spec.mcp
	source := t.ServerName + " · " + t.Name
	res := toolResult{label: source}
	r, err := s.MCP.Call(ctx, c.ws, c.owner, c.thread.ID, c.callID, t.ID, hash, c.args)
	if err != nil {
		res.code = problemCode(err)
		return res, envelope(spec.Name, res)
	}
	if r.IsError {
		res.code = "tool_error"
	}
	return res, untrusted(source, r.Text, !r.IsError)
}

// untrusted is an MCP result as the model reads it: inside delimiters
// naming its source, which its text can't close (nor open another
// reserved tag) and its source can't leave.
func untrusted(source, text string, ok bool) json.RawMessage {
	data := "<untrusted_content source=\"" + strings.ReplaceAll(tagSafe(source), `"`, "'") + "\">\n" + defang(text) + "\n</untrusted_content>"
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(map[string]any{"ok": ok, "source": source, "untrusted": true, "data": data})
	return bytes.TrimSpace(b.Bytes())
}

// confirmationTTL is how long a call waits for its owner before it expires.
const confirmationTTL = 24 * time.Hour

// pause stops a turn at a call of a writing tool: the call waits, sealed,
// with the results of the response's earlier calls; the slot goes; the
// owner is told, in the thread and in their notifications (the server and
// the tool, never the arguments). The thread and the turn are locked.
func (s *Service) pause(ctx context.Context, q *store.Queries, run *turnRun, turn store.AiAssistantTurn, rd round, index int, spec ToolSpec) error {
	id := uuid.Must(uuid.NewV7())
	call := rd.Calls[index]
	args := call.Arguments
	if len(bytes.TrimSpace(args)) == 0 {
		args = json.RawMessage("{}")
	}
	argsSealed, err := s.Box.Seal(args, confirmationAD(id, "arguments"))
	if err != nil {
		return err
	}
	raw, err := json.Marshal(rd)
	if err != nil {
		return err
	}
	done, err := s.Box.Seal(raw, confirmationAD(id, "done"))
	if err != nil {
		return err
	}
	digest := mcpclient.CallDigest(call.ID, spec.mcp.ID, args)
	toolID, serverID := spec.mcp.ID, spec.mcp.ServerID
	if err := q.AIAssistantConfirmationInsert(ctx, store.AIAssistantConfirmationInsertParams{ID: id, ThreadID: run.thread.ID, TurnID: turn.ID, StepID: run.step,
		CallIndex: int32(index), CallID: call.ID, ServerID: &serverID, ToolID: &toolID, ServerName: spec.mcp.ServerName, ToolName: spec.mcp.Name, //nolint:gosec // G115: a response's calls are few
		ModelName: spec.Name, ToolHash: spec.mcp.Hash, ArgumentsSealed: argsSealed, ArgumentsDigest: digest, DoneSealed: done, Now: s.now()}); err != nil {
		return err
	}
	if err := q.AIAssistantSlotRelease(ctx, store.AIAssistantSlotReleaseParams{TurnID: turn.ID, LeaseID: run.lease}); err != nil {
		return err
	}
	turn.State = "waiting"
	if err := q.AIAssistantTurnSave(ctx, store.AIAssistantTurnSaveParams{ID: turn.ID, State: turn.State, Reason: "", Version: turn.Version, StartedAt: turn.StartedAt}); err != nil {
		return err
	}
	if err := s.turnEvent(ctx, q, turn); err != nil {
		return err
	}
	c, err := q.AIAssistantConfirmationLock(ctx, store.AIAssistantConfirmationLockParams{ThreadID: run.thread.ID, ID: id})
	if err != nil {
		return err
	}
	if err := s.confirmationEvent(ctx, q, c); err != nil {
		return err
	}
	return s.notifyConfirmation(ctx, q, run.thread, c)
}

// notifyConfirmation tells a thread's owner a call waits for them, as
// their notification preferences allow: the server and the tool only.
func (s *Service) notifyConfirmation(ctx context.Context, q *store.Queries, t store.AiAssistantThread, c store.AiAssistantConfirmation) error {
	ws, err := q.GetWorkspace(ctx, t.WorkspaceID)
	if err != nil {
		return err
	}
	args := i18n.Args{"server": c.ServerName, "tool": c.ToolName}
	return notify.ToUserWith(ctx, q, t.UserID, notify.Notification{WorkspaceID: &t.WorkspaceID, Kind: notify.KindAIAssistantConfirmation, Severity: "warn",
		TitleText: msgs.M("notify.confirmationTitle").Keep(nil), BodyText: msgs.M("notify.confirmationBody").Keep(args),
		Link: "/w/" + ws.Slug + "/assistant/" + t.ID.String(),
		Data: map[string]any{"thread_id": t.ID.String(), "turn_id": c.TurnID.String(), "confirmation_id": c.ID.String(), "server": c.ServerName, "tool": c.ToolName}})
}

// noticesGone removes the notice of a call waiting in a thread: it was
// decided, stopped or expired, or the thread went.
func noticesGone(ctx context.Context, q *store.Queries, t store.AiAssistantThread) error {
	return q.AIAssistantConfirmationNoticesRemove(ctx, store.AIAssistantConfirmationNoticesRemoveParams{UserID: t.UserID, ThreadID: t.ID.String()})
}

// expireConfirmations ends the turns whose call waited a day for its
// owner: the call is never sent, the turn fails as confirmation_expired
// (its slot went when it paused, and its requests were accounted), the
// notice goes. It reports how many expired.
func (s *Service) expireConfirmations(ctx context.Context, now time.Time) (int, error) {
	rows, err := s.Q.AIAssistantConfirmationsExpiring(ctx, now.Add(-confirmationTTL))
	if err != nil {
		return 0, err
	}
	n := 0
	for _, r := range rows {
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			t, err := q.AIAssistantThreadLockByID(ctx, store.AIAssistantThreadLockByIDParams{WorkspaceID: r.WorkspaceID, ID: r.ThreadID})
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			c, err := q.AIAssistantConfirmationLock(ctx, store.AIAssistantConfirmationLockParams{ThreadID: t.ID, ID: r.ID})
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			if c.State != "pending" || !c.CreatedAt.Before(now.Add(-confirmationTTL)) {
				return nil
			}
			turn, err := q.AIAssistantTurnLock(ctx, store.AIAssistantTurnLockParams{ThreadID: t.ID, ID: c.TurnID})
			if err != nil {
				return err
			}
			if err := q.AIAssistantConfirmationSet(ctx, store.AIAssistantConfirmationSetParams{ID: c.ID, State: "expired"}); err != nil {
				return err
			}
			c.State = "expired"
			if err := s.confirmationEvent(ctx, q, c); err != nil {
				return err
			}
			if err := noticesGone(ctx, q, t); err != nil {
				return err
			}
			if turn.State == "waiting" {
				if err := q.AIAssistantSlotDrop(ctx, turn.ID); err != nil {
					return err
				}
				if err := s.endTurn(ctx, q, &turn, "failed", "confirmation_expired"); err != nil {
					return err
				}
			}
			n++
			meta := map[string]any{"turn_id": turn.ID, "confirmation_id": c.ID, "tool": c.ModelName, "arguments_digest": hex.EncodeToString(c.ArgumentsDigest)}
			if c.ServerID != nil {
				meta["server_id"] = *c.ServerID
			}
			ev := threadEvent(t.WorkspaceID, t.ID, "ai_assistant_confirmation.expired", meta)
			ev.Actor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}
			return s.Audit.LogWith(ctx, q, ev)
		})
		if err != nil {
			return n, fmt.Errorf("assistant: expire a confirmation: %w", err)
		}
	}
	return n, nil
}

func (s *Service) confirmationView(c store.AiAssistantConfirmation) (AIThreadConfirmation, error) {
	args, err := s.Box.Open(c.ArgumentsSealed, confirmationAD(c.ID, "arguments"))
	if err != nil {
		return AIThreadConfirmation{}, err
	}
	v := AIThreadConfirmation{ID: c.ID, TurnID: c.TurnID, ServerID: c.ServerID, ServerName: c.ServerName, Tool: c.ToolName, ModelName: c.ModelName,
		Arguments: args, State: c.State, CreatedAt: c.CreatedAt, DecidedAt: c.DecidedAt}
	if c.Decision != nil {
		v.Decision = *c.Decision
	}
	return v, nil
}

func (s *Service) confirmationEvent(ctx context.Context, q *store.Queries, c store.AiAssistantConfirmation) error {
	v, err := s.confirmationView(c)
	if err != nil {
		return err
	}
	return s.appendEvent(ctx, q, c.ThreadID, &c.TurnID, kindConfirmation, AIThreadConfirmationEvent{Confirmation: v})
}

// openRound is a waiting call's round.
func (s *Service) openRound(c store.AiAssistantConfirmation) (round, error) {
	var rd round
	raw, err := s.Box.Open(c.DoneSealed, confirmationAD(c.ID, "done"))
	if err != nil {
		return rd, err
	}
	if err := json.Unmarshal(raw, &rd); err != nil {
		return rd, err
	}
	if int(c.CallIndex) >= len(rd.Calls) || len(rd.Results) != int(c.CallIndex) {
		return rd, errors.New("assistant: a confirmation's round doesn't match its call")
	}
	if rd.Results == nil {
		rd.Results = []llm.Part{}
	}
	if rd.Rows == nil {
		rd.Rows = []AIThreadToolRow{}
	}
	return rd, nil
}

// DecideConfirmation is the owner's decision on a call waiting in one of
// their threads: allow runs it once, deny tells the model it was refused;
// the turn goes on either way. Sending the same decision again changes
// nothing; another decision, or one on a call whose turn ended, is a
// conflict. Allowing needs the tool as it was offered (not changed,
// blocked or removed) and a server the owner may still use.
func (s *Service) DecideConfirmation(ctx context.Context, ws *store.Workspace, p *authz.Principal, thread, id uuid.UUID, in AIThreadConfirmationDecision) (AIThread, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, _, err := s.actor(ctx, q, ws, p, in.Decision == "allow")
		if err != nil {
			return err
		}
		th, err := lockThread(ctx, q, ws.ID, fresh.UserID, thread)
		if err != nil {
			return err
		}
		c, err := q.AIAssistantConfirmationLock(ctx, store.AIAssistantConfirmationLockParams{ThreadID: thread, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Confirmation")
		}
		if err != nil {
			return err
		}
		if c.Decision != nil {
			if *c.Decision == in.Decision {
				return nil
			}
			return problem(ctx, http.StatusConflict, codeConfirmationClosed, mConfirmationClosed, nil)
		}
		turn, err := q.AIAssistantTurnLock(ctx, store.AIAssistantTurnLockParams{ThreadID: thread, ID: c.TurnID})
		if err != nil {
			return err
		}
		if c.State != "pending" || turn.State != "waiting" {
			return problem(ctx, http.StatusConflict, codeConfirmationClosed, mConfirmationClosed, nil)
		}
		if in.Decision == "allow" {
			if s.MCP == nil {
				return errors.New("assistant: no MCP client")
			}
			tool := uuid.Nil
			if c.ToolID != nil {
				tool = *c.ToolID
			}
			if _, _, err := s.MCP.Current(ctx, q, ws, fresh, tool, c.ToolHash); err != nil {
				return err
			}
		}
		state, action := "allowed", "ai_assistant_confirmation.approved"
		if in.Decision == "deny" {
			state, action = "denied", "ai_assistant_confirmation.rejected"
		}
		now := s.now()
		decision := in.Decision
		if err := q.AIAssistantConfirmationDecide(ctx, store.AIAssistantConfirmationDecideParams{ID: c.ID, State: state, Decision: &decision, DecidedBy: &fresh.UserID, Now: now}); err != nil {
			return err
		}
		c.State, c.Decision, c.DecidedAt = state, &decision, &now
		turn.Version++
		if err := q.AIAssistantTurnRequeue(ctx, store.AIAssistantTurnRequeueParams{ID: turn.ID, Version: turn.Version, Now: now}); err != nil {
			return err
		}
		turn.State = "queued"
		if err := s.confirmationEvent(ctx, q, c); err != nil {
			return err
		}
		if err := noticesGone(ctx, q, th); err != nil {
			return err
		}
		if err := s.turnEvent(ctx, q, turn); err != nil {
			return err
		}
		if err := s.touch(ctx, q, th); err != nil {
			return err
		}
		if err := s.enqueue(ctx, tx, ws.ID, turn); err != nil {
			return err
		}
		meta := map[string]any{"turn_id": turn.ID, "confirmation_id": c.ID, "tool": c.ModelName, "arguments_digest": hex.EncodeToString(c.ArgumentsDigest)}
		if c.ServerID != nil {
			meta["server_id"] = *c.ServerID
		}
		if c.ToolID != nil {
			meta["tool_id"] = *c.ToolID
		}
		return s.Audit.LogWith(ctx, q, threadEvent(ws.ID, thread, action, meta))
	})
	if err != nil {
		return AIThread{}, err
	}
	return s.Thread(ctx, ws, p, thread)
}

// SetThreadServers picks the MCP servers one of the person's threads may
// use: servers they may use now. Audited as ai_assistant_thread.updated.
func (s *Service) SetThreadServers(ctx context.Context, ws *store.Workspace, p *authz.Principal, thread uuid.UUID, in AIThreadServersInput) (AIThread, error) {
	ids := slices.Clone(in.ServerIDs)
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	ids = slices.Compact(ids)
	if len(ids) > 0 {
		if s.MCP == nil {
			return AIThread{}, api.NotFound("MCP server")
		}
		usable, err := s.MCP.Usable(ctx, ws, p)
		if err != nil {
			return AIThread{}, err
		}
		for _, id := range ids {
			if !slices.ContainsFunc(usable.Servers, func(v mcpclient.RemoteMCPServer) bool { return v.ID == id }) {
				return AIThread{}, api.NotFound("MCP server")
			}
		}
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, _, err := s.actor(ctx, q, ws, p, true)
		if err != nil {
			return err
		}
		if _, err := lockThread(ctx, q, ws.ID, fresh.UserID, thread); err != nil {
			return err
		}
		if err := q.MCPThreadServersClear(ctx, thread); err != nil {
			return err
		}
		for _, id := range ids {
			if err := q.MCPThreadServerAdd(ctx, store.MCPThreadServerAddParams{ThreadID: thread, ServerID: id}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, threadEvent(ws.ID, thread, "ai_assistant_thread.updated", map[string]any{"changed": []string{"mcp_servers"}, "mcp_servers": ids}))
	})
	if err != nil {
		return AIThread{}, err
	}
	return s.Thread(ctx, ws, p, thread)
}
