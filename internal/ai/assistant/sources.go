package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// A thread keeps what its tools read, sealed, and sends it to the model
// again in later requests and in summaries. Each result keeps where its
// content came from: the resource (an attached campaign or report, or an
// MCP server) and the permission reading it needs. Before it is sent
// again, its owner must still be able to read that, as they are now (a
// permission taken away, a resource deleted, a server they may no longer
// use): otherwise the model reads only that it is no longer available,
// and a summary made from it isn't sent at all (assistant brief §3.7).

// toolSource is where a call's result came from.
type toolSource struct {
	CallID     string           `json:"callId"`
	Kind       string           `json:"kind"` // campaign, report or mcp_server
	ID         uuid.UUID        `json:"id"`
	Permission authz.Permission `json:"permission"`
}

// resourceSource is an attached resource as a result's source.
func resourceSource(a store.AiAssistantContext) *toolSource {
	permission := authz.CampaignsView
	if a.Kind == "report" {
		permission = authz.ReportsView
	}
	return &toolSource{Kind: a.Kind, ID: a.TargetID, Permission: permission}
}

// sourced are the tools whose results carry a resource's content (an
// MCP server's text, a campaign's, a report's): a result of theirs with
// content and without a source is never sent again. Their failures
// without a source (not run: not available, denied, past the ceiling)
// carry a code only, as do the other tools' results (ReputeCast's help,
// screens and cards).
func sourced(name string) bool {
	switch name {
	case "read_attached", "read_report", "propose_edit", "subject_ideas", "rewrite":
		return true
	}
	return strings.HasPrefix(name, "mcp__")
}

// unavailable is a result as the model reads it once its source can't
// be read.
var unavailable = json.RawMessage(`{"ok":false,"code":"no_longer_available"}`)

// noSummary replaces a summary made from what the owner can't read now.
const noSummary = "An earlier part of this conversation used something the person can no longer read, so it is left out."

// readable decides, as a thread's owner is now, which saved results and
// summaries may be sent again. It reads each source once per request.
type readable struct {
	s     *Service
	q     *store.Queries
	ws    *store.Workspace
	owner *authz.Principal
	known map[toolSource]bool
}

func (s *Service) readable(q *store.Queries, ws *store.Workspace, owner *authz.Principal) *readable {
	return &readable{s: s, q: q, ws: ws, owner: owner, known: map[toolSource]bool{}}
}

// source reports whether the owner may read a source now.
func (r *readable) source(ctx context.Context, src toolSource) (bool, error) {
	src.CallID = ""
	if ok, seen := r.known[src]; seen {
		return ok, nil
	}
	ok, err := r.check(ctx, src)
	if err != nil {
		return false, err
	}
	r.known[src] = ok
	return ok, nil
}

func (r *readable) check(ctx context.Context, src toolSource) (bool, error) {
	if src.Permission == "" || !r.owner.Can(src.Permission) {
		return false, nil
	}
	switch src.Kind {
	case "campaign", "report":
		_, err := r.s.AI.AssistantAttachment(ctx, r.q, r.ws, r.owner, src.Kind, src.ID, false)
		var p *api.Problem
		if errors.As(err, &p) {
			return false, nil
		}
		return err == nil, err
	case "mcp_server":
		if r.s.MCP == nil {
			return false, nil
		}
		return r.s.MCP.MayRead(ctx, r.q, r.ws, r.owner, src.ID)
	}
	return false, nil
}

// all reports whether every source may be read now.
func (r *readable) all(ctx context.Context, sources []toolSource) (bool, error) {
	for _, src := range sources {
		if ok, err := r.source(ctx, src); err != nil || !ok {
			return false, err
		}
	}
	return true, nil
}

// withheld are, for each tool message (by its index), the calls whose
// results may not be sent again: their source can't be read now, or a
// result that needs one has none.
func (r *readable) withheld(ctx context.Context, messages []store.AiAssistantMessage, contents []messageContent) (map[int]map[string]bool, error) {
	out := map[int]map[string]bool{}
	for i, m := range messages {
		if m.Role != "tool" {
			continue
		}
		c := contents[i]
		for _, p := range c.Results {
			tr := p.ToolResult
			if tr == nil {
				continue
			}
			// Without a source, only what carries no content may go.
			ok := !sourced(tr.Name) || tr.IsError
			for _, src := range c.Sources {
				if src.CallID == tr.ID {
					var err error
					if ok, err = r.source(ctx, src); err != nil {
						return nil, err
					}
					break
				}
			}
			if !ok {
				if out[i] == nil {
					out[i] = map[string]bool{}
				}
				out[i][tr.ID] = true
			}
		}
	}
	return out, nil
}

// sourcesOf are a message's sources for the calls not withheld.
func sourcesOf(c messageContent, withheld map[string]bool) []toolSource {
	out := []toolSource{}
	for _, src := range c.Sources {
		if !withheld[src.CallID] {
			out = append(out, src)
		}
	}
	return out
}
