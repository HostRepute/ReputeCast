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
	Type   string                `json:"type"`
	Text   string                `json:"text,omitempty"`
	Source *anthropicImageSource `json:"source,omitempty"`
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
}

type anthropicResponse struct {
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (a *anthropic) headers() http.Header {
	h := http.Header{}
	h.Set("x-api-key", a.cfg.APIKey)
	h.Set("anthropic-version", anthropicVersion)
	return h
}

func (a *anthropic) Generate(ctx context.Context, req Request) (Response, error) {
	body := anthropicRequest{Model: a.cfg.Model, MaxTokens: maxTokens(req), System: req.System, Messages: []anthropicMessage{}}
	system := req.System
	if req.JSON && !strings.Contains(strings.ToLower(system), "json") {
		system += "\n\nAnswer with one JSON object only."
	}
	body.System = system
	for _, m := range req.Messages {
		msg := anthropicMessage{Role: m.Role, Content: []anthropicContent{}}
		for _, p := range m.Parts {
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
	var text strings.Builder
	for _, c := range res.Content {
		if c.Type == "text" {
			text.WriteString(c.Text)
		}
	}
	out := Response{Text: text.String(), Model: res.Model, InputTokens: res.Usage.InputTokens, OutputTokens: res.Usage.OutputTokens}
	switch res.StopReason {
	case "end_turn", "stop_sequence":
		out.StopReason = StopEnd
	case "max_tokens":
		out.StopReason = StopLength
	case "refusal":
		out.StopReason, out.Refused = StopRefusal, true
	default:
		out.StopReason = StopOther
	}
	return out, nil
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
