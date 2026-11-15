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
	Usage      struct {
		InputTokens   *int `json:"input_tokens"`
		OutputTokens  *int `json:"output_tokens"`
		CacheRead     *int `json:"cache_read_input_tokens"`
		CacheCreation *int `json:"cache_creation_input_tokens"`
	} `json:"usage"`
}

func (a *anthropic) headers() http.Header {
	h := http.Header{}
	h.Set("x-api-key", a.cfg.APIKey)
	h.Set("anthropic-version", anthropicVersion)
	return h
}

func (a *anthropic) Generate(ctx context.Context, req Request) (Response, error) {
	if err := validateTools(req, false); err != nil {
		return Response{}, err
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
	raw, err := call{cfg: a.cfg, method: http.MethodPost, url: a.cfg.base("https://api.anthropic.com") + "/v1/messages", header: a.headers(), body: body, classify: anthropicError}.do(ctx)
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
	switch res.StopReason {
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
	return checkedTools(req, out, false)
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
