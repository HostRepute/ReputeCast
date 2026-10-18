package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// gemini is the Gemini API's generateContent
// (https://ai.google.dev/api/generate-content).
type gemini struct{ cfg Config }

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inline_data,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  map[string]any  `json:"generationConfig,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback *struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	ModelVersion string `json:"modelVersion"`
}

func (g *gemini) headers() http.Header {
	h := http.Header{}
	h.Set("x-goog-api-key", g.cfg.APIKey)
	return h
}

func (g *gemini) root() string { return g.cfg.base("https://generativelanguage.googleapis.com") }

func (g *gemini) Generate(ctx context.Context, req Request) (Response, error) {
	body := geminiRequest{Contents: []geminiContent{}, GenerationConfig: map[string]any{"maxOutputTokens": maxTokens(req)}}
	if req.System != "" {
		body.SystemInstruction = &geminiContent{Parts: []geminiPart{{Text: req.System}}}
	}
	for _, m := range req.Messages {
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		c := geminiContent{Role: role, Parts: []geminiPart{}}
		for _, p := range m.Parts {
			if p.Image != nil {
				c.Parts = append(c.Parts, geminiPart{InlineData: &geminiInlineData{MimeType: p.Image.MediaType, Data: base64.StdEncoding.EncodeToString(p.Image.Data)}})
				continue
			}
			c.Parts = append(c.Parts, geminiPart{Text: p.Text})
		}
		body.Contents = append(body.Contents, c)
	}
	if req.JSON {
		body.GenerationConfig["responseMimeType"] = "application/json"
	}
	model := strings.TrimPrefix(g.cfg.Model, "models/")
	raw, err := call{cfg: g.cfg, method: http.MethodPost, url: g.root() + "/v1beta/models/" + url.PathEscape(model) + ":generateContent", header: g.headers(), body: body, classify: geminiError}.do(ctx)
	if err != nil {
		return Response{}, err
	}
	var res geminiResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return Response{}, &Error{Kind: ErrInvalid, Err: err, Message: "Gemini's answer couldn't be read"}
	}
	out := Response{Model: res.ModelVersion, InputTokens: res.UsageMetadata.PromptTokenCount, OutputTokens: res.UsageMetadata.CandidatesTokenCount}
	if out.Model == "" {
		out.Model = model
	}
	if res.PromptFeedback != nil && res.PromptFeedback.BlockReason != "" && len(res.Candidates) == 0 {
		out.Refused, out.StopReason = true, StopRefusal
		return out, nil
	}
	if len(res.Candidates) == 0 {
		return Response{}, &Error{Kind: ErrInvalid, Message: "Gemini answered without a candidate"}
	}
	c := res.Candidates[0]
	var text strings.Builder
	for _, p := range c.Content.Parts {
		text.WriteString(p.Text)
	}
	out.Text = text.String()
	switch c.FinishReason {
	case "STOP":
		out.StopReason = StopEnd
	case "MAX_TOKENS":
		out.StopReason = StopLength
	case "SAFETY", "RECITATION", "PROHIBITED_CONTENT", "BLOCKLIST", "SPII":
		out.StopReason, out.Refused = StopRefusal, true
	default:
		out.StopReason = StopOther
	}
	return out, nil
}

func (g *gemini) Models(ctx context.Context) ([]Model, error) {
	raw, err := call{cfg: g.cfg, method: http.MethodGet, url: g.root() + "/v1beta/models?pageSize=200", header: g.headers(), classify: geminiError}.do(ctx)
	if err != nil {
		return nil, err
	}
	var res struct {
		Models []struct {
			Name                       string   `json:"name"`
			DisplayName                string   `json:"displayName"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, &Error{Kind: ErrInvalid, Err: err, Message: "Gemini's model list couldn't be read"}
	}
	out := []Model{}
	for _, m := range res.Models {
		gen := false
		for _, x := range m.SupportedGenerationMethods {
			if x == "generateContent" {
				gen = true
			}
		}
		if gen {
			out = append(out, Model{ID: strings.TrimPrefix(m.Name, "models/"), Name: m.DisplayName})
		}
	}
	return out, nil
}

// geminiError reads {"error":{"code":400,"message":"API key not valid…","status":"INVALID_ARGUMENT","details":[{"reason":"API_KEY_INVALID"}]}}.
func geminiError(status int, _ http.Header, body []byte) *Error {
	var e struct {
		Error struct {
			Message string `json:"message"`
			Status  string `json:"status"`
			Details []struct {
				Type       string `json:"@type"`
				Reason     string `json:"reason"`
				RetryDelay string `json:"retryDelay"`
			} `json:"details"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &e)
	kind := kindOf(status)
	out := &Error{Status: status, Code: e.Error.Status, Message: e.Error.Message}
	for _, d := range e.Error.Details {
		switch {
		case d.Reason == "API_KEY_INVALID" || d.Reason == "API_KEY_SERVICE_BLOCKED":
			kind = ErrAuth
		case d.RetryDelay != "":
			if dur, err := time.ParseDuration(d.RetryDelay); err == nil {
				out.RetryAfter = dur
			}
		}
	}
	switch e.Error.Status {
	case "UNAUTHENTICATED", "PERMISSION_DENIED":
		kind = ErrAuth
	case "RESOURCE_EXHAUSTED":
		kind = ErrRateLimit
	case "UNAVAILABLE":
		kind = ErrOverloaded
	}
	out.Kind = kind
	return out
}
