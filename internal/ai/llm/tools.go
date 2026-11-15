package llm

import (
	"bytes"
	"encoding/json"
	"regexp"
)

const maxToolCalls = 40

var toolName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]{0,63}$`)

func toolError() error {
	return &Error{Kind: ErrInvalid, Message: "The tool exchange is invalid"}
}

func jsonObject(raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	return len(raw) > 0 && len(raw) <= 200_000 && raw[0] == '{' && json.Valid(raw)
}

func validateTools(req Request, optionalID bool) error {
	if len(req.Tools) > maxToolCalls || (len(req.Tools) > 0 && req.JSON) {
		return toolError()
	}
	names := map[string]bool{}
	for _, t := range req.Tools {
		if !toolName.MatchString(t.Name) || names[t.Name] || t.Parameters["type"] != "object" {
			return toolError()
		}
		names[t.Name] = true
	}
	var pending []ToolCall
	for _, m := range req.Messages {
		if m.Role == "tool" {
			if len(pending) == 0 || len(m.Parts) != len(pending) {
				return toolError()
			}
		} else if len(pending) > 0 || (m.Role != "user" && m.Role != "assistant") {
			return toolError()
		}
		ids := map[string]bool{}
		for i, p := range m.Parts {
			if (p.Thought || p.ThoughtSignature != "") && (!optionalID || m.Role != "assistant") {
				return toolError()
			}
			if p.ToolResult != nil || m.Role == "tool" {
				r := p.ToolResult
				if r == nil || m.Role != "tool" || r.ID != pending[i].ID || r.Name != pending[i].Name || !jsonObject(r.Content) || p.ToolCall != nil || p.Image != nil || p.Text != "" {
					return toolError()
				}
			}
		}
		if m.Role == "tool" {
			pending = nil
			continue
		}
		for _, p := range m.Parts {
			if c := p.ToolCall; c != nil {
				if m.Role != "assistant" || !toolName.MatchString(c.Name) || !jsonObject(c.Arguments) || p.ToolResult != nil || p.Image != nil || p.Text != "" || p.Thought || len(c.ID) > 200 || (!optionalID && c.ID == "") || (c.ID != "" && ids[c.ID]) {
					return toolError()
				}
				ids[c.ID] = true
				pending = append(pending, *c)
			}
		}
		if len(pending) > maxToolCalls {
			return toolError()
		}
	}
	if len(pending) > 0 {
		return toolError()
	}
	return nil
}

// checkedTools preserves usage even when a paid response is unusable. Tool
// names are not an authorization decision here: the registry must record and
// deny attempted calls outside its allowlist, rather than silently dropping them.
func checkedTools(req Request, out Response, optionalID bool) (Response, error) {
	ids := map[string]bool{}
	n := 0
	for _, p := range out.Message.Parts {
		c := p.ToolCall
		if c == nil {
			continue
		}
		n++
		if !toolName.MatchString(c.Name) || !jsonObject(c.Arguments) || p.Text != "" || p.Image != nil || p.Thought || p.ToolResult != nil || len(c.ID) > 200 || (!optionalID && c.ID == "") || (c.ID != "" && ids[c.ID]) {
			return out, toolError()
		}
		ids[c.ID] = true
	}
	if n > maxToolCalls || (n > 0 && (len(req.Tools) == 0 || out.StopReason != StopTools || out.Refused)) || (n == 0 && out.StopReason == StopTools) {
		return out, toolError()
	}
	return out, nil
}

// Provider token counts fit int32. A missing required counter, negative value,
// or implausibly large sum cannot be treated as known zero-cost usage.
func tokenUsage(required *int, extras ...*int) (int, bool) {
	if required == nil {
		return 0, false
	}
	total := int64(*required)
	if total < 0 || total > 1_000_000_000 {
		return 0, false
	}
	for _, extra := range extras {
		if extra == nil {
			continue
		}
		if *extra < 0 || *extra > 1_000_000_000 {
			return 0, false
		}
		total += int64(*extra)
	}
	if total > 1_000_000_000 {
		return 0, false
	}
	return int(total), true
}
