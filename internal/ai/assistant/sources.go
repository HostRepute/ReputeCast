package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
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

// An answer quotes what was in the request that produced it, so it is
// shown and sent again only while its owner may still read all of that:
// each assistant message (and summary) keeps the sources of its request's
// content: the tools' results, the summary and the earlier answers it
// carried, and the attached resources whose titles it listed. Partial
// answers saved by Stop or the sweep (which don't have the request) count
// every source the thread's tools and summaries read before them and what
// is attached; messages saved before they kept sources count what the
// thread read before them: fail-closed.

// unknownSource is what a result with content and no recorded source came
// from: never readable.
var unknownSource = toolSource{Kind: "unknown"}

// withheldAnswer is a withheld assistant message as the model reads it
// again.
const withheldAnswer = "An earlier answer here used something the person can no longer read, so it is left out."

// history is a thread's messages as their owner may read them now.
type history struct {
	// calls are the tool messages' withheld calls, by message index; held
	// the withheld assistant messages and summaries.
	calls map[int]map[string]bool
	held  map[int]bool
	// sources are what each message's content came from; prior is every
	// source the thread's messages read, and readable whether the owner
	// may read all of it and what is attached (what an answer in flight
	// may have used: Service.history adds the attachments).
	sources  [][]toolSource
	prior    []toolSource
	readable bool
}

// toolSources are a tool message's sources: a result with content and no
// recorded source counts as unknownSource.
func toolSources(c messageContent) []toolSource {
	out := slices.Clone(c.Sources)
	for _, p := range c.Results {
		tr := p.ToolResult
		if tr == nil || !sourced(tr.Name) || tr.IsError {
			continue
		}
		if !slices.ContainsFunc(c.Sources, func(s toolSource) bool { return s.CallID == tr.ID }) {
			out = append(out, unknownSource)
		}
	}
	return out
}

// merge adds srcs to a set of sources, without their calls' ids.
func merge(set []toolSource, srcs []toolSource) []toolSource {
	for _, s := range srcs {
		s.CallID = ""
		if !slices.Contains(set, s) {
			set = append(set, s)
		}
	}
	return set
}

// priorSources is every source a thread's messages read: what an answer
// saved without its request (Stop, the sweep) may have used.
func priorSources(messages []store.AiAssistantMessage, contents []messageContent) []toolSource {
	out := []toolSource{}
	for i, m := range messages {
		c := contents[i]
		switch m.Role {
		case "tool":
			out = merge(out, toolSources(c))
		case "assistant", "summary":
			out = merge(out, c.Sources)
		}
	}
	return out
}

// history decides which of a thread's results, answers and summaries its
// owner may read now, reading each source once.
func (r *readable) history(ctx context.Context, messages []store.AiAssistantMessage, contents []messageContent) (*history, error) {
	calls, err := r.withheld(ctx, messages, contents)
	if err != nil {
		return nil, err
	}
	h := &history{calls: calls, held: map[int]bool{}, sources: make([][]toolSource, len(messages)), prior: []toolSource{}}
	for i, m := range messages {
		c := contents[i]
		var srcs []toolSource
		switch m.Role {
		case "tool":
			srcs = toolSources(c)
		case "assistant", "summary":
			srcs = c.Sources
			if !c.SourcesKnown {
				srcs = merge(slices.Clone(h.prior), c.Sources)
			}
			ok, err := r.all(ctx, srcs)
			if err != nil {
				return nil, err
			}
			h.held[i] = !ok
		default:
			continue
		}
		h.sources[i] = srcs
		h.prior = merge(h.prior, srcs)
	}
	if h.readable, err = r.all(ctx, h.prior); err != nil {
		return nil, err
	}
	return h, nil
}

// threadSources is every source a thread's messages read so far, and what
// is attached (its requests list their titles).
func (s *Service) threadSources(ctx context.Context, q *store.Queries, thread uuid.UUID) ([]toolSource, error) {
	messages, err := q.AIAssistantMessages(ctx, thread)
	if err != nil {
		return nil, err
	}
	contents := make([]messageContent, len(messages))
	for i, m := range messages {
		if contents[i], err = s.messageContent(m); err != nil {
			return nil, err
		}
	}
	contexts, err := q.AIAssistantContexts(ctx, thread)
	if err != nil {
		return nil, err
	}
	return merge(priorSources(messages, contents), attachedSources(contexts)), nil
}

// attachedSources are a thread's attached resources as sources.
func attachedSources(contexts []store.AiAssistantContext) []toolSource {
	out := make([]toolSource, 0, len(contexts))
	for _, c := range contexts {
		out = append(out, *resourceSource(c))
	}
	return out
}
