package llm

import (
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
	Content any `json:"content"`
}

type openAIRequest struct {
	Model               string          `json:"model,omitempty"`
	Messages            []openAIMessage `json:"messages"`
	MaxTokens           int             `json:"max_tokens,omitempty"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	ResponseFormat      map[string]any  `json:"response_format,omitempty"`
}

type openAIResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content *string `json:"content"`
			Refusal *string `json:"refusal"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
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
	body := openAIRequest{Messages: []openAIMessage{}}
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
		images := false
		for _, p := range m.Parts {
			if p.Image != nil {
				images = true
			}
		}
		if !images {
			var text strings.Builder
			for _, p := range m.Parts {
				text.WriteString(p.Text)
			}
			body.Messages = append(body.Messages, openAIMessage{Role: m.Role, Content: text.String()})
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
		body.Messages = append(body.Messages, openAIMessage{Role: m.Role, Content: parts})
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
	raw, err := call{cfg: o.cfg, method: http.MethodPost, url: o.endpoint("/chat/completions"), header: o.headers(), body: body, classify: openAIError}.do(ctx)
	if err != nil {
		return Response{}, err
	}
	var res openAIResponse
	if err := json.Unmarshal(raw, &res); err != nil || len(res.Choices) == 0 {
		return Response{}, &Error{Kind: ErrInvalid, Err: err, Message: o.name() + "'s answer couldn't be read"}
	}
	ch := res.Choices[0]
	out := Response{Model: res.Model, InputTokens: res.Usage.PromptTokens, OutputTokens: res.Usage.CompletionTokens}
	if ch.Message.Content != nil {
		out.Text = *ch.Message.Content
	}
	switch ch.FinishReason {
	case "stop":
		out.StopReason = StopEnd
	case "length":
		out.StopReason = StopLength
	case "content_filter":
		out.StopReason, out.Refused = StopRefusal, true
	default:
		out.StopReason = StopOther
	}
	if ch.Message.Refusal != nil && strings.TrimSpace(*ch.Message.Refusal) != "" {
		out.Refused, out.StopReason, out.Text = true, StopRefusal, *ch.Message.Refusal
	}
	if out.Model == "" {
		out.Model = o.cfg.Model
	}
	return out, nil
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
