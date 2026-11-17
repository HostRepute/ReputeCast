package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

func base64Std(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

// openAI is the Chat Completions API, spoken by OpenAI
// (https://platform.openai.com/docs/api-reference/chat), Azure OpenAI (a
// deployment URL, the api-key header and api-version) and compatible
// servers such as Ollama, vLLM and LM Studio (their address with /v1, an
// optional bearer key).
type openAI struct{ cfg Config }

type openAIContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openAIImageURL `json:"image_url,omitempty"`
}

type openAIImageURL struct {
	URL string `json:"url"`
}

type openAIMessage struct {
	Role string `json:"role"`
	// Content is a string for system turns and text-only turns (what every
	// compatible server reads), a list of parts with images.
	Content    any              `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openAIRequest struct {
	Model               string               `json:"model,omitempty"`
	Messages            []openAIMessage      `json:"messages"`
	MaxTokens           int                  `json:"max_tokens,omitempty"`
	MaxCompletionTokens int                  `json:"max_completion_tokens,omitempty"`
	ResponseFormat      map[string]any       `json:"response_format,omitempty"`
	Tools               []openAITool         `json:"tools,omitempty"`
	Stream              bool                 `json:"stream,omitempty"`
	StreamOptions       *openAIStreamOptions `json:"stream_options,omitempty"`
}

type openAIResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content   *string          `json:"content"`
			Refusal   *string          `json:"refusal"`
			ToolCalls []openAIToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage openAIUsage `json:"usage"`
}

func (o *openAI) name() string {
	switch o.cfg.Kind {
	case Azure:
		return "Azure OpenAI"
	case Compatible:
		return "The server"
	}
	return "OpenAI"
}

func (o *openAI) headers() http.Header {
	h := http.Header{}
	switch {
	case o.cfg.Kind == Azure:
		h.Set("api-key", o.cfg.APIKey)
	case o.cfg.APIKey != "":
		h.Set("Authorization", "Bearer "+o.cfg.APIKey)
	}
	return h
}

func (o *openAI) apiVersion() string {
	if v := strings.TrimSpace(o.cfg.APIVersion); v != "" {
		return v
	}
	return DefaultAzureAPIVersion
}

// endpoint is the URL of a path ("/chat/completions", "/models").
func (o *openAI) endpoint(path string) string {
	switch o.cfg.Kind {
	case Azure:
		return o.cfg.base("") + "/openai/deployments/" + url.PathEscape(o.cfg.Deployment) + path + "?api-version=" + url.QueryEscape(o.apiVersion())
	case Compatible:
		return o.cfg.base("") + path
	}
	return o.cfg.base("https://api.openai.com/v1") + path
}

func (o *openAI) Generate(ctx context.Context, req Request) (Response, error) {
	body, err := o.request(req)
	if err != nil {
		return Response{}, err
	}
	raw, err := o.completions(body).do(ctx)
	if err != nil {
		return Response{}, err
	}
	var res openAIResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return Response{}, &Error{Kind: ErrInvalid, Err: err, Message: o.name() + "'s answer couldn't be read"}
	}
	input, inputKnown := tokenUsage(res.Usage.PromptTokens)
	output, outputKnown := tokenUsage(res.Usage.CompletionTokens)
	out := Response{Model: res.Model, InputTokens: input, OutputTokens: output, UsageKnown: inputKnown && outputKnown, Message: Message{Role: "assistant", Parts: []Part{}}}
	if len(res.Choices) != 1 {
		return out, &Error{Kind: ErrInvalid, Message: o.name() + " answered without exactly one choice"}
	}
	ch := res.Choices[0]
	if ch.Message.Content != nil {
		out.Text = *ch.Message.Content
		out.Message.Parts = append(out.Message.Parts, Part{Text: out.Text})
	}
	for _, c := range ch.Message.ToolCalls {
		if c.Type != "function" {
			return out, toolError()
		}
		out.Message.Parts = append(out.Message.Parts, Part{ToolCall: &ToolCall{ID: c.ID, Name: c.Function.Name, Arguments: json.RawMessage(c.Function.Arguments)}})
	}
	openAIStop(ch.FinishReason, &out)
	if ch.Message.Refusal != nil && strings.TrimSpace(*ch.Message.Refusal) != "" {
		out.Refused, out.StopReason, out.Text = true, StopRefusal, *ch.Message.Refusal
	}
	if out.Model == "" {
		out.Model = o.cfg.Model
	}
	return checkedTools(req, out, false)
}

// request is the Chat Completions request of req.
func (o *openAI) request(req Request) (openAIRequest, error) {
	if err := validateTools(req, false); err != nil {
		return openAIRequest{}, err
	}
	body := openAIRequest{Messages: []openAIMessage{}}
	for _, tool := range req.Tools {
		body.Tools = append(body.Tools, openAITool{Type: "function", Function: openAIFunction(tool)})
	}
	if o.cfg.Kind != Azure {
		body.Model = o.cfg.Model
	}
	// OpenAI's newer models take max_completion_tokens; Azure's GA API and
	// compatible servers read max_tokens.
	if o.cfg.Kind == OpenAI {
		body.MaxCompletionTokens = maxTokens(req)
	} else {
		body.MaxTokens = maxTokens(req)
	}
	if req.System != "" {
		body.Messages = append(body.Messages, openAIMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		if m.Role == "tool" {
			for _, p := range m.Parts {
				if p.ToolResult != nil {
					body.Messages = append(body.Messages, openAIMessage{Role: "tool", Content: string(p.ToolResult.Content), ToolCallID: p.ToolResult.ID})
				}
			}
			continue
		}
		calls := []openAIToolCall{}
		images := false
		for _, p := range m.Parts {
			if p.ToolCall != nil {
				c := openAIToolCall{ID: p.ToolCall.ID, Type: "function"}
				c.Function.Name, c.Function.Arguments = p.ToolCall.Name, string(p.ToolCall.Arguments)
				calls = append(calls, c)
			}
			if p.Image != nil {
				images = true
			}
		}
		if !images {
			var text strings.Builder
			for _, p := range m.Parts {
				text.WriteString(p.Text)
			}
			body.Messages = append(body.Messages, openAIMessage{Role: m.Role, Content: text.String(), ToolCalls: calls})
			continue
		}
		parts := []openAIContent{}
		for _, p := range m.Parts {
			if p.Image != nil {
				parts = append(parts, openAIContent{Type: "image_url", ImageURL: &openAIImageURL{URL: imageDataURL(p.Image)}})
				continue
			}
			parts = append(parts, openAIContent{Type: "text", Text: p.Text})
		}
		body.Messages = append(body.Messages, openAIMessage{Role: m.Role, Content: parts, ToolCalls: calls})
	}
	if req.JSON {
		if req.Schema != nil && o.cfg.Kind != Compatible {
			name := req.SchemaName
			if name == "" {
				name = "answer"
			}
			body.ResponseFormat = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": name, "schema": req.Schema}}
		} else {
			body.ResponseFormat = map[string]any{"type": "json_object"}
		}
	}
	return body, nil
}

func (o *openAI) completions(body openAIRequest) call {
	return call{cfg: o.cfg, method: http.MethodPost, url: o.endpoint("/chat/completions"), header: o.headers(), body: body, classify: openAIError}
}

// openAIStop maps a finish reason.
func openAIStop(reason string, out *Response) {
	switch reason {
	case "stop":
		out.StopReason = StopEnd
	case "tool_calls":
		out.StopReason = StopTools
	case "length":
		out.StopReason = StopLength
	case "content_filter":
		out.StopReason, out.Refused = StopRefusal, true
	default:
		out.StopReason = StopOther
	}
}

// Stream reads Chat Completions chunks
// (https://platform.openai.com/docs/api-reference/chat-streaming): the
// text in delta.content, refusals in delta.refusal, tool calls in
// delta.tool_calls by index, the finish reason, and, asked for with
// stream_options.include_usage, a last chunk with the usage and no
// choices, then data: [DONE]. A server that sends no usage chunk leaves
// the usage unknown. Azure answers the same, after a first chunk with its
// prompt filter results and no choices.
func (o *openAI) Stream(ctx context.Context, req Request, emit func(Delta) error) (Response, error) {
	body, err := o.request(req)
	if err != nil {
		return Response{}, err
	}
	body.Stream = true
	body.StreamOptions = &openAIStreamOptions{IncludeUsage: true}
	st := &openAIStream{name: o.name(), emit: emit, current: -1}
	err = o.completions(body).stream(ctx, st)
	out := st.response()
	if out.Model == "" {
		out.Model = o.cfg.Model
	}
	if err != nil {
		return out, err
	}
	if st.unknown {
		return out, toolError()
	}
	return checkedTools(req, out, false)
}

type openAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openAIUsage struct {
	PromptTokens     *int `json:"prompt_tokens"`
	CompletionTokens *int `json:"completion_tokens"`
}

type openAIChunk struct {
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content   *string `json:"content"`
			Refusal   *string `json:"refusal"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *openAIUsage    `json:"usage"`
	Error json.RawMessage `json:"error"`
}

type openAIStream struct {
	name    string
	emit    func(Delta) error
	model   string
	text    strings.Builder // what was emitted: content, then a refusal
	content strings.Builder
	refusal strings.Builder
	calls   []ToolCall
	args    []strings.Builder
	byIndex map[int]int
	current int // the index whose arguments are being emitted
	finish  string
	// usage is the last reported; final once it came with or after the
	// finish reason, or the stream ended with [DONE].
	usage   *openAIUsage
	final   bool
	done    bool
	unknown bool
}

func (s *openAIStream) complete() bool { return s.done || s.finish != "" }

func (s *openAIStream) event(ev sseEvent) error {
	if bytes.Equal(bytes.TrimSpace(ev.data), []byte("[DONE]")) {
		s.done, s.final = true, s.usage != nil
		return errStreamEnd
	}
	var c openAIChunk
	if err := decodeEvent(s.name, ev, &c); err != nil {
		return err
	}
	if len(c.Error) > 0 && !bytes.Equal(c.Error, []byte("null")) {
		return midStream(openAIError(0, nil, ev.data))
	}
	if s.model == "" {
		s.model = c.Model
	}
	if len(c.Choices) > 1 || (len(c.Choices) == 1 && c.Choices[0].Index != 0) {
		return &Error{Kind: ErrInvalid, Message: s.name + " answered without exactly one choice"}
	}
	for _, ch := range c.Choices {
		d := ch.Delta
		if d.Content != nil && *d.Content != "" {
			s.content.WriteString(*d.Content)
			if err := s.say(*d.Content); err != nil {
				return err
			}
		}
		if d.Refusal != nil && *d.Refusal != "" {
			s.refusal.WriteString(*d.Refusal)
			if err := s.say(*d.Refusal); err != nil {
				return err
			}
		}
		for _, tc := range d.ToolCalls {
			if tc.Type != "" && tc.Type != "function" {
				s.unknown = true
			}
			i, ok := s.byIndex[tc.Index]
			if !ok {
				if s.byIndex == nil {
					s.byIndex = map[int]int{}
				}
				i = len(s.calls)
				s.byIndex[tc.Index] = i
				s.calls = append(s.calls, ToolCall{ID: tc.ID, Name: tc.Function.Name})
				s.args = append(s.args, strings.Builder{})
				s.current = i
				if err := s.emit(Delta{ToolCallStart: &ToolCall{ID: tc.ID, Name: tc.Function.Name}}); err != nil {
					return err
				}
			}
			if tc.Function.Arguments == "" {
				continue
			}
			s.args[i].WriteString(tc.Function.Arguments)
			// Calls arrive one after another; a fragment of an earlier one
			// is kept in its call but not emitted as the current call's.
			if i == s.current {
				if err := s.emit(Delta{ToolArgs: tc.Function.Arguments}); err != nil {
					return err
				}
			}
		}
		if ch.FinishReason != nil && *ch.FinishReason != "" {
			s.finish = *ch.FinishReason
		}
	}
	if c.Usage != nil {
		s.usage, s.final = c.Usage, s.finish != ""
	}
	return nil
}

func (s *openAIStream) say(t string) error {
	s.text.WriteString(t)
	return s.emit(Delta{Text: t})
}

// response is the answer so far.
func (s *openAIStream) response() Response {
	out := Response{Model: s.model, Text: s.text.String(), Message: Message{Role: "assistant", Parts: []Part{}}}
	if s.final {
		input, inputKnown := tokenUsage(s.usage.PromptTokens)
		output, outputKnown := tokenUsage(s.usage.CompletionTokens)
		out.InputTokens, out.OutputTokens, out.UsageKnown = input, output, inputKnown && outputKnown
	}
	if s.content.Len() > 0 {
		out.Message.Parts = append(out.Message.Parts, Part{Text: s.content.String()})
	}
	for i, c := range s.calls {
		c.Arguments = json.RawMessage(s.args[i].String())
		out.Message.Parts = append(out.Message.Parts, Part{ToolCall: &c})
	}
	if s.finish != "" {
		openAIStop(s.finish, &out)
	}
	if strings.TrimSpace(s.refusal.String()) != "" {
		out.Refused, out.StopReason = true, StopRefusal
	}
	return out
}

func (o *openAI) Models(ctx context.Context) ([]Model, error) {
	if o.cfg.Kind == Azure {
		// A deployment is one model; Azure's model list is the resource's,
		// which the deployment URL can't read.
		return []Model{{ID: o.cfg.Deployment, Name: o.cfg.Deployment}}, nil
	}
	raw, err := call{cfg: o.cfg, method: http.MethodGet, url: o.endpoint("/models"), header: o.headers(), classify: openAIError}.do(ctx)
	if err != nil {
		return nil, err
	}
	var res struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, &Error{Kind: ErrInvalid, Err: err, Message: o.name() + "'s model list couldn't be read"}
	}
	out := make([]Model, 0, len(res.Data))
	for _, m := range res.Data {
		out = append(out, Model{ID: m.ID, Name: m.ID})
	}
	return out, nil
}

// openAIError reads {"error":{"message":"…","type":"invalid_request_error","code":"invalid_api_key"}}.
// Azure answers the same shape; some compatible servers answer
// {"error":"model not found"}.
func openAIError(status int, _ http.Header, body []byte) *Error {
	var e struct {
		Error json.RawMessage `json:"error"`
	}
	_ = json.Unmarshal(body, &e)
	var detail struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	}
	msg := ""
	if len(e.Error) > 0 && e.Error[0] == '{' {
		_ = json.Unmarshal(e.Error, &detail)
		msg = detail.Message
	} else if len(e.Error) > 0 {
		_ = json.Unmarshal(e.Error, &msg)
	}
	code := detail.Type
	if s, ok := detail.Code.(string); ok && s != "" {
		code = s
	}
	kind := kindOf(status)
	switch code {
	case "invalid_api_key", "invalid_authentication", "401":
		kind = ErrAuth
	case "rate_limit_exceeded", "insufficient_quota", "429":
		kind = ErrRateLimit
	}
	return &Error{Kind: kind, Status: status, Code: code, Message: msg}
}
