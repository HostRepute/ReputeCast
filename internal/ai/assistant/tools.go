package assistant

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/mcpclient"
	"github.com/hostrepute/reputecast/internal/store"
)

// The assistant's tools (assistant brief §3.5). The model sees only the
// tools its owner may use now, with the features they need switched on, in
// this fixed order (a stable request prefix); every call is authorized
// again when it runs, as the owner. Results are projections, never raw
// domain rows, and reach the model as data inside an envelope that names
// their source.

// ToolSpec is one tool.
type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]any
	// Permission is what the owner needs; Feature the AI feature whose
	// switch must be on ("" for none); Needs the kind of attached resource
	// it works on ("" for none).
	Permission authz.Permission
	Feature    string
	Needs      string
	// Confirm: the call waits for the person to confirm that exact call
	// (the turn is "waiting"): MCP tools that write outside ReputeCast; no
	// tool of ReputeCast's own needs it.
	Confirm bool
	run     func(ctx context.Context, s *Service, c *toolCall) toolResult
	// mcp is an MCP server's tool (run is nil).
	mcp *mcpclient.ThreadTool
}

// maxTools bounds the tools of a request (ReputeCast's own come first).
const maxTools = 40

// maxToolResult bounds what a call returns to the model.
const maxToolResult = 128 << 10

// toolCall is a call as it runs: the owner as they are now.
type toolCall struct {
	ws       *store.Workspace
	owner    *authz.Principal
	thread   store.AiAssistantThread
	contexts []store.AiAssistantContext
	callID   string
	args     json.RawMessage
}

// toolResult is what a call gives back: the model's content and the row
// the person sees.
type toolResult struct {
	data  any
	code  string // failed: why
	label string
	card  *AIThreadCard
}

func failed(code string) toolResult { return toolResult{code: code} }

// AIThreadToolRow is a tool call as the thread shows it: what was read or
// called, how long it took and, for a handoff, its card.
type AIThreadToolRow struct {
	CallID string        `json:"callId"`
	Name   string        `json:"name"`
	OK     bool          `json:"ok"`
	Code   string        `json:"code,omitempty"`
	Ms     int64         `json:"ms"`
	Label  string        `json:"label,omitempty"`
	Card   *AIThreadCard `json:"card,omitempty"`
}

// AIThreadCard is a handoff into an existing screen: it only navigates.
type AIThreadCard struct {
	Kind       string     `json:"kind" enum:"link,proposal,agent_task"`
	Title      string     `json:"title"`
	Path       string     `json:"path"`
	ProposalID *uuid.UUID `json:"proposalId,omitempty"`
	// Brief is a drafted agent task: the person starts the run and confirms
	// sharing in the agent's screen.
	Brief string `json:"brief,omitempty"`
}

func object(props map[string]any, required ...string) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

var handleParam = str("The attached resource's handle, as listed under <attached>")

// tools are every tool, in the order requests list them.
var tools = []ToolSpec{
	{Name: "search_help", Description: "Search ReputeCast's own help for how something works. Returns up to three articles; no workspace data.",
		Parameters: object(map[string]any{"query": str("What to look for, in a few words")}, "query"), Permission: authz.AIUse, run: runSearchHelp},
	{Name: "describe_screen", Description: "Say what a screen of ReputeCast is for and what it offers. Without a screen, the one the person is on.",
		Parameters: object(map[string]any{"screen": str("A screen key, such as campaign.report")}), Permission: authz.AIUse, run: runDescribeScreen},
	{Name: "suggest_link", Description: "Give the person a link to a screen of ReputeCast they may open. It only navigates. Screens that show one resource need its id.",
		Parameters: object(map[string]any{"screen": str("A screen key, such as campaign.report"), "id": str("The resource's id, for screens of one resource")}, "screen"),
		Permission: authz.AIUse, run: runSuggestLink},
	{Name: "read_attached", Description: "Read a resource the person attached. A campaign comes as its subject, preheader and visible text, with links, images and Liquid tokens as opaque handles (keep them as they are); a report, as its title (read_report has its numbers).",
		Parameters: object(map[string]any{"handle": handleParam}, "handle"), Permission: authz.AIUse, Needs: "any", run: runReadAttached},
	{Name: "read_report", Description: "Read an attached report's fixed totals, as they were when it was attached. Counts round down to tens; small numbers are withheld, not zero. No recipients, links or diagnosis of delivery.",
		Parameters: object(map[string]any{"handle": handleParam}, "handle"), Permission: authz.ReportsView, Needs: "report", run: runReadReport},
	{Name: "propose_edit", Description: "Propose new content for an attached draft campaign, for the person to review in the composer: its subject, preheader and body as one MJML document (mj-section, mj-column, mj-text, mj-button, mj-image), at the revision read_attached gave. Use only the handles read_attached gave, as they are; the brand header and footer are kept. Nothing changes until the person accepts it.",
		Parameters: object(map[string]any{"handle": handleParam, "baseRevision": map[string]any{"type": "integer", "description": "The revision read_attached returned"},
			"subject": str("The subject"), "preheader": str("The preheader"), "source": str("The body as one MJML document")}, "handle", "baseRevision", "subject", "preheader", "source"),
		Permission: authz.CampaignsEdit, Feature: ai.FeatureDraft, Needs: "campaign", run: runProposeEdit},
	{Name: "draft_agent_task", Description: "Draft a task for the AI agent, for the person to review and start in the agent's screen. It starts nothing.",
		Parameters: object(map[string]any{"title": str("A short name for the task"), "brief": str("What the agent should do, in the person's words")}, "title", "brief"),
		Permission: authz.AIUse, Feature: ai.FeatureAgent, run: runDraftAgentTask},
	{Name: "subject_ideas", Description: "Suggest subject lines for an attached campaign, with their checks. Uses its own model request.",
		Parameters: object(map[string]any{"handle": handleParam}, "handle"), Permission: authz.CampaignsEdit, Feature: ai.FeatureSubjectIdeas, Needs: "campaign", run: runSubjectIdeas},
	{Name: "rewrite", Description: "Rewrite a passage of an attached campaign: shorten, friendlier, formal, direct, or by an instruction. Uses its own model request; nothing changes in the campaign.",
		Parameters: object(map[string]any{"handle": handleParam, "text": str("The passage"), "action": map[string]any{"type": "string", "enum": []string{"shorten", "friendlier", "formal", "direct", "instruction"}},
			"instruction": str("What to change, for the instruction action")}, "handle", "text", "action"),
		Permission: authz.CampaignsEdit, Feature: ai.FeatureRewrite, Needs: "campaign", run: runRewrite},
}

// offered is the tools the owner may use in a thread now: ReputeCast's
// own first, then the MCP tools of the servers the thread may use.
func (s *Service) offered(ctx context.Context, q *store.Queries, ws *store.Workspace, owner *authz.Principal, thread uuid.UUID, contexts []store.AiAssistantContext) ([]ToolSpec, error) {
	out := []ToolSpec{}
	for _, t := range tools {
		if !owner.Can(t.Permission) || t.Confirm {
			continue
		}
		switch t.Needs {
		case "":
		case "any":
			if len(contexts) == 0 {
				continue
			}
		default:
			if !slices.ContainsFunc(contexts, func(c store.AiAssistantContext) bool { return c.Kind == t.Needs }) {
				continue
			}
		}
		if t.Feature != "" {
			on, err := s.AI.AssistantFeatureOn(ctx, q, ws.ID, t.Feature)
			if err != nil {
				return nil, err
			}
			if !on {
				continue
			}
		}
		out = append(out, t)
		if len(out) == maxTools {
			break
		}
	}
	more, err := s.mcpSpecs(ctx, q, ws, owner, thread, maxTools-len(out))
	if err != nil {
		return nil, err
	}
	return append(out, more...), nil
}

func llmTools(specs []ToolSpec) []llm.Tool {
	out := make([]llm.Tool, 0, len(specs))
	for _, t := range specs {
		out = append(out, llm.Tool{Name: t.Name, Description: t.Description, Parameters: t.Parameters})
	}
	return out
}

// runTool runs a call as the owner is now: a tool they may no longer use,
// or one they weren't offered, fails without running. A tool that needs
// a confirmation runs only as the call its owner allowed (confirmed).
func (s *Service) runTool(ctx context.Context, q *store.Queries, c *toolCall, name string, confirmed *store.AiAssistantConfirmation) (toolResult, json.RawMessage) {
	result := failed("not_available")
	var content json.RawMessage
	offered, err := s.offered(ctx, q, c.ws, c.owner, c.thread.ID, c.contexts)
	if err != nil {
		result = failed("unavailable")
	} else if i := slices.IndexFunc(offered, func(t ToolSpec) bool { return t.Name == name }); i >= 0 {
		t := offered[i]
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		switch {
		case t.mcp == nil:
			result = t.run(cctx, s, c)
		case !t.Confirm:
			result, content = s.runMCP(cctx, c, t, t.mcp.Hash)
		case confirmed != nil && confirmed.ToolID != nil && *confirmed.ToolID == t.mcp.ID:
			result, content = s.runMCP(cctx, c, t, confirmed.ToolHash)
		default:
			result = failed("confirmation_required")
		}
		cancel()
	}
	if content == nil {
		content = envelope(name, result)
	}
	if len(content) > maxToolResult {
		result = failed("too_large")
		content = envelope(name, result)
	}
	return result, content
}

// envelope is a result as the model reads it: data from a named source,
// never instructions.
func envelope(name string, r toolResult) json.RawMessage {
	var v any = map[string]any{"ok": false, "code": r.code}
	if r.code == "" {
		v = map[string]any{"ok": true, "source": name, "data": r.data}
	}
	raw, _ := json.Marshal(v)
	return raw
}

// problemCode is a call's failure code for a problem: its code without
// "ai_", or not_found / forbidden; an operational error is unavailable.
func problemCode(err error) string {
	var p *api.Problem
	switch {
	case !errors.As(err, &p):
		return "unavailable"
	case strings.HasPrefix(p.Code, "mcp_"):
		return p.Code
	case p.Status == 404:
		return "not_found"
	case p.Status == 403:
		return "forbidden"
	}
	if code, ok := strings.CutPrefix(p.Code, "ai_"); ok {
		return code
	}
	return "refused"
}

func decode[T any](c *toolCall) (T, bool) {
	var v T
	return v, json.Unmarshal(c.args, &v) == nil
}

// attachment is the attached resource a handle names.
func (c *toolCall) attachment(handle string) (store.AiAssistantContext, bool) {
	id, err := uuid.Parse(strings.TrimSpace(handle))
	if err != nil {
		return store.AiAssistantContext{}, false
	}
	i := slices.IndexFunc(c.contexts, func(x store.AiAssistantContext) bool { return x.ID == id })
	if i < 0 {
		return store.AiAssistantContext{}, false
	}
	return c.contexts[i], true
}

//go:embed help/*.md
var helpFS embed.FS

type helpArticle struct {
	ID, Title, Text string
	words           []string
}

var help = loadHelp()

func loadHelp() []helpArticle {
	entries, _ := helpFS.ReadDir("help")
	out := []helpArticle{}
	for _, e := range entries {
		raw, _ := helpFS.ReadFile("help/" + e.Name())
		lines := strings.SplitN(string(raw), "\n", 3)
		if len(lines) < 3 {
			continue
		}
		a := helpArticle{ID: strings.TrimSuffix(e.Name(), ".md"), Title: strings.TrimPrefix(lines[0], "# "), Text: strings.TrimSpace(lines[2])}
		a.words = words(lines[0] + " " + lines[1] + " " + lines[2])
		out = append(out, a)
	}
	return out
}

var wordRe = regexp.MustCompile(`[\pL\pN]+`)

func words(s string) []string { return wordRe.FindAllString(strings.ToLower(s), -1) }

// HelpArticle is a help article as search_help returns it.
type HelpArticle struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// searchHelp ranks the articles by the query's words they hold.
func searchHelp(query string) []HelpArticle {
	type hit struct {
		a     helpArticle
		score int
	}
	hits := []hit{}
	for _, a := range help {
		score := 0
		for _, w := range words(query) {
			if len(w) < 3 {
				continue
			}
			for _, x := range a.words {
				if x == w || (len(w) > 4 && strings.HasPrefix(x, w)) {
					score++
				}
			}
		}
		if score > 0 {
			hits = append(hits, hit{a, score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	out := []HelpArticle{}
	for _, h := range hits[:min(len(hits), 3)] {
		out = append(out, HelpArticle{ID: h.a.ID, Title: h.a.Title, Text: h.a.Text})
	}
	return out
}

func runSearchHelp(_ context.Context, _ *Service, c *toolCall) toolResult {
	in, ok := decode[struct {
		Query string `json:"query"`
	}](c)
	if !ok || strings.TrimSpace(in.Query) == "" {
		return failed("invalid_arguments")
	}
	return toolResult{data: map[string]any{"articles": searchHelp(in.Query)}, label: clip(in.Query, 80)}
}

func clip(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
