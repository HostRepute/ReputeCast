package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

// anthropicVersion is the Messages API version header.
const anthropicVersion = "2023-06-01"

// anthropic is the Messages API (https://docs.anthropic.com/en/api/messages).
type anthropic struct{ cfg Config }

type anthropicContent struct {
	Type      string                `json:"type"`
	Text      string                `json:"text,omitempty"`
	Source    *anthropicImageSource `json:"source,omitempty"`
	ID        string                `json:"id,omitempty"`
	Name      string                `json:"name,omitempty"`
	Input     json.RawMessage       `json:"input,omitempty"`
	ToolUseID string                `json:"tool_use_id,omitempty"`
	Content   string                `json:"content,omitempty"`
	IsError   bool                  `json:"is_error,omitempty"`
}

type anthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
	Stream    bool               `json:"stream,omitempty"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicResponse struct {
	Model      string             `json:"model"`
	StopReason string             `json:"stop_reason"`
	Content    []anthropicContent `json:"content"`
	Usage      anthropicUsage     `json:"usage"`
}

type anthropicUsage struct {
	InputTokens   *int `json:"input_tokens"`
	OutputTokens  *int `json:"output_tokens"`
	CacheRead     *int `json:"cache_read_input_tokens"`
	CacheCreation *int `json:"cache_creation_input_tokens"`
}

func (a *anthropic) headers() http.Header {
	h := http.Header{}
	h.Set("x-api-key", a.cfg.APIKey)
	h.Set("anthropic-version", anthropicVersion)
	return h
}

func (a *anthropic) Generate(ctx context.Context, req Request) (Response, error) {
	body, err := a.request(req)
	if err != nil {
		return Response{}, err
	}
	raw, err := a.messages(body).do(ctx)
	if err != nil {
		return Response{}, err
	}
	var res anthropicResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return Response{}, &Error{Kind: ErrInvalid, Err: err, Message: "Anthropic's answer couldn't be read"}
	}
	input, inputKnown := tokenUsage(res.Usage.InputTokens, res.Usage.CacheRead, res.Usage.CacheCreation)
	output, outputKnown := tokenUsage(res.Usage.OutputTokens)
	out := Response{Model: res.Model, InputTokens: input, OutputTokens: output, UsageKnown: inputKnown && outputKnown, Message: Message{Role: "assistant", Parts: []Part{}}}
	var text strings.Builder
	for _, c := range res.Content {
		switch c.Type {
		case "text":
			text.WriteString(c.Text)
			out.Message.Parts = append(out.Message.Parts, Part{Text: c.Text})
		case "tool_use":
			out.Message.Parts = append(out.Message.Parts, Part{ToolCall: &ToolCall{ID: c.ID, Name: c.Name, Arguments: c.Input}})
		default:
			// Dropping an unknown block could lose a required signature or
			// hide a provider-hosted action. Do not execute a partial turn.
			return out, toolError()
		}
	}
	out.Text = text.String()
	anthropicStop(res.StopReason, &out)
	return checkedTools(req, out, false)
}

// request is the Messages API request of req.
func (a *anthropic) request(req Request) (anthropicRequest, error) {
	if err := validateTools(req, false); err != nil {
		return anthropicRequest{}, err
	}
	body := anthropicRequest{Model: a.cfg.Model, MaxTokens: maxTokens(req), System: req.System, Messages: []anthropicMessage{}}
	for _, tool := range req.Tools {
		body.Tools = append(body.Tools, anthropicTool{Name: tool.Name, Description: tool.Description, InputSchema: tool.Parameters})
	}
	system := req.System
	if req.JSON && !strings.Contains(strings.ToLower(system), "json") {
		system += "\n\nAnswer with one JSON object only."
	}
	body.System = system
	for _, m := range req.Messages {
		msg := anthropicMessage{Role: m.Role, Content: []anthropicContent{}}
		if m.Role == "tool" {
			msg.Role = "user"
		}
		for _, p := range m.Parts {
			if p.ToolCall != nil {
				msg.Content = append(msg.Content, anthropicContent{Type: "tool_use", ID: p.ToolCall.ID, Name: p.ToolCall.Name, Input: p.ToolCall.Arguments})
				continue
			}
			if p.ToolResult != nil {
				msg.Content = append(msg.Content, anthropicContent{Type: "tool_result", ToolUseID: p.ToolResult.ID, Content: string(p.ToolResult.Content), IsError: p.ToolResult.IsError})
				continue
			}
			if p.Image != nil {
				msg.Content = append(msg.Content, anthropicContent{Type: "image", Source: &anthropicImageSource{Type: "base64", MediaType: p.Image.MediaType, Data: base64.StdEncoding.EncodeToString(p.Image.Data)}})
				continue
			}
			msg.Content = append(msg.Content, anthropicContent{Type: "text", Text: p.Text})
		}
		body.Messages = append(body.Messages, msg)
	}
	return body, nil
}

func (a *anthropic) messages(body anthropicRequest) call {
	return call{cfg: a.cfg, method: http.MethodPost, url: a.cfg.base("https://api.anthropic.com") + "/v1/messages", header: a.headers(), body: body, classify: anthropicError}
}

// anthropicStop maps a stop reason.
func anthropicStop(reason string, out *Response) {
	switch reason {
	case "end_turn", "stop_sequence":
		out.StopReason = StopEnd
	case "tool_use":
		out.StopReason = StopTools
	case "max_tokens":
		out.StopReason = StopLength
	case "refusal":
		out.StopReason, out.Refused = StopRefusal, true
	default:
		out.StopReason = StopOther
	}
}

// Stream reads the Messages API's events
// (https://platform.claude.com/docs/en/build-with-claude/streaming):
// message_start (the model, the input tokens), content blocks with
// text_delta and input_json_delta, message_delta (the stop reason and the
// cumulative usage), message_stop, and error events.
func (a *anthropic) Stream(ctx context.Context, req Request, emit func(Delta) error) (Response, error) {
	body, err := a.request(req)
	if err != nil {
		return Response{}, err
	}
	body.Stream = true
	st := &anthropicStream{emit: emit, parts: []Part{}}
	err = a.messages(body).stream(ctx, st)
	out := st.response()
	if err != nil {
		return out, err
	}
	if st.unknown {
		return out, toolError()
	}
	return checkedTools(req, out, false)
}

type anthropicEvent struct {
	Type         string             `json:"type"`
	Message      *anthropicResponse `json:"message"`
	ContentBlock *anthropicContent  `json:"content_block"`
	Delta        struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
		StopReason  string `json:"stop_reason"`
	} `json:"delta"`
	Usage *anthropicUsage `json:"usage"`
}

type anthropicStream struct {
	emit   func(Delta) error
	model  string
	parts  []Part
	text   strings.Builder
	usage  anthropicUsage
	output bool // a message_delta reported the output tokens
	stop   string
	ended  bool
	// unknown is set by a block this adapter can't replay (as Generate).
	unknown bool
	// block is the open content block: "text", "tool_use" or another type.
	block string
	tool  ToolCall
	args  strings.Builder
	frags int
}

func (s *anthropicStream) complete() bool { return s.ended }

func (s *anthropicStream) event(ev sseEvent) error {
	var e anthropicEvent
	if err := decodeEvent("Anthropic", ev, &e); err != nil {
		return err
	}
	name := ev.name
	if name == "" {
		name = e.Type
	}
	switch name {
	case "message_start":
		if e.Message != nil {
			s.model = e.Message.Model
			s.usage.InputTokens, s.usage.CacheRead, s.usage.CacheCreation = e.Message.Usage.InputTokens, e.Message.Usage.CacheRead, e.Message.Usage.CacheCreation
		}
	case "content_block_start":
		if e.ContentBlock == nil {
			return &Error{Kind: ErrInvalid, Message: "Anthropic's answer couldn't be read"}
		}
		s.block = e.ContentBlock.Type
		switch s.block {
		case "text":
			s.parts = append(s.parts, Part{})
			return s.addText(e.ContentBlock.Text)
		case "tool_use":
			s.tool = ToolCall{ID: e.ContentBlock.ID, Name: e.ContentBlock.Name, Arguments: e.ContentBlock.Input}
			s.args.Reset()
			s.frags = 0
			return s.emit(Delta{ToolCallStart: &ToolCall{ID: s.tool.ID, Name: s.tool.Name}})
		default:
			s.unknown = true
		}
	case "content_block_delta":
		switch {
		case s.block == "text" && e.Delta.Type == "text_delta":
			return s.addText(e.Delta.Text)
		case s.block == "tool_use" && e.Delta.Type == "input_json_delta" && e.Delta.PartialJSON != "":
			s.args.WriteString(e.Delta.PartialJSON)
			s.frags++
			return s.emit(Delta{ToolArgs: e.Delta.PartialJSON})
		}
	case "content_block_stop":
		block := s.block
		s.block = ""
		if block == "tool_use" {
			// The input arrives as fragments; a call without input has
			// none, and its block's input ({}) is the whole of it.
			if s.frags > 0 {
				s.tool.Arguments = json.RawMessage(s.args.String())
			} else {
				if len(s.tool.Arguments) == 0 {
					s.tool.Arguments = json.RawMessage(`{}`)
				}
				if err := s.emit(Delta{ToolArgs: string(s.tool.Arguments)}); err != nil {
					return err
				}
			}
			call := s.tool
			s.parts = append(s.parts, Part{ToolCall: &call})
		}
	case "message_delta":
		if e.Delta.StopReason != "" {
			s.stop = e.Delta.StopReason
		}
		// The counts are cumulative: the last event's are the request's.
		if u := e.Usage; u != nil {
			if u.InputTokens != nil {
				s.usage.InputTokens = u.InputTokens
			}
			if u.CacheRead != nil {
				s.usage.CacheRead = u.CacheRead
			}
			if u.CacheCreation != nil {
				s.usage.CacheCreation = u.CacheCreation
			}
			if u.OutputTokens != nil {
				s.usage.OutputTokens, s.output = u.OutputTokens, true
			}
		}
	case "message_stop":
		s.ended = true
		return errStreamEnd
	case "error":
		return midStream(anthropicError(0, nil, ev.data))
	}
	// ping, and event types added later.
	return nil
}

func (s *anthropicStream) addText(t string) error {
	if t == "" {
		return nil
	}
	s.parts[len(s.parts)-1].Text += t
	s.text.WriteString(t)
	return s.emit(Delta{Text: t})
}

// response is the answer so far.
func (s *anthropicStream) response() Response {
	out := Response{Model: s.model, Text: s.text.String(), Message: Message{Role: "assistant", Parts: s.parts}}
	if s.output {
		input, inputKnown := tokenUsage(s.usage.InputTokens, s.usage.CacheRead, s.usage.CacheCreation)
		output, outputKnown := tokenUsage(s.usage.OutputTokens)
		out.InputTokens, out.OutputTokens, out.UsageKnown = input, output, inputKnown && outputKnown
	}
	if s.stop != "" {
		anthropicStop(s.stop, &out)
	}
	return out
}

func (a *anthropic) Models(ctx context.Context) ([]Model, error) {
	raw, err := call{cfg: a.cfg, method: http.MethodGet, url: a.cfg.base("https://api.anthropic.com") + "/v1/models?limit=100", header: a.headers(), classify: anthropicError}.do(ctx)
	if err != nil {
		return nil, err
	}
	var res struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, &Error{Kind: ErrInvalid, Err: err, Message: "Anthropic's model list couldn't be read"}
	}
	out := make([]Model, 0, len(res.Data))
	for _, m := range res.Data {
		out = append(out, Model{ID: m.ID, Name: m.DisplayName})
	}
	return out, nil
}

// anthropicError reads {"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}.
func anthropicError(status int, _ http.Header, body []byte) *Error {
	var e struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &e)
	kind := kindOf(status)
	switch e.Error.Type {
	case "authentication_error", "permission_error":
		kind = ErrAuth
	case "rate_limit_error":
		kind = ErrRateLimit
	case "overloaded_error":
		kind = ErrOverloaded
	}
	return &Error{Kind: kind, Status: status, Code: e.Error.Type, Message: e.Error.Message}
}
