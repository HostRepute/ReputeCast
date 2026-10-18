package llm

import (
	"strings"
	"time"
)

// Capabilities a provider card shows.
const (
	CapText             = "text"
	CapImageInput       = "image_input"
	CapStructuredOutput = "structured_output"
	CapOwnNetwork       = "own_network"
	CapDataRegion       = "data_region"
)

// ProviderInfo describes a provider in the catalog.
type ProviderInfo struct {
	Kind string
	Name string
	// NeedsKey is false for compatible servers (local servers often have none).
	NeedsKey bool
	// NeedsBaseURL: Azure's endpoint, a compatible server's address.
	NeedsBaseURL bool
	Capabilities []string
	// DefaultModel is offered first in the connect dialog.
	DefaultModel string
}

// Providers is the catalog, in order.
var Providers = []ProviderInfo{
	{Kind: Anthropic, Name: "Anthropic", NeedsKey: true, Capabilities: []string{CapText, CapImageInput, CapStructuredOutput}, DefaultModel: "claude-sonnet-5-5"},
	{Kind: OpenAI, Name: "OpenAI", NeedsKey: true, Capabilities: []string{CapText, CapImageInput, CapStructuredOutput}, DefaultModel: "gpt-5"},
	{Kind: Google, Name: "Google Gemini", NeedsKey: true, Capabilities: []string{CapText, CapImageInput, CapStructuredOutput}, DefaultModel: "gemini-2.5-flash"},
	{Kind: Azure, Name: "Azure OpenAI", NeedsKey: true, NeedsBaseURL: true, Capabilities: []string{CapText, CapImageInput, CapDataRegion}},
	{Kind: Compatible, Name: "OpenAI-compatible", NeedsBaseURL: true, Capabilities: []string{CapText, CapOwnNetwork}},
}

// Info returns a provider's catalog entry.
func Info(kind string) ProviderInfo {
	for _, p := range Providers {
		if p.Kind == kind {
			return p
		}
	}
	return ProviderInfo{Kind: kind, Name: kind}
}

// PriceListDate is the date of the price list below, the one ReputeCast's
// release ships with.
var PriceListDate = time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)

// KnownModel is a model of the price list: its name, its prices in dollars
// per million tokens, and whether it reads images.
type KnownModel struct {
	Provider   string
	ID         string
	Name       string
	Input      float64
	Output     float64
	ImageInput bool
}

// PriceList holds the prices pre-filled when a model is picked. People edit
// them when their contract differs; a model that isn't listed starts at 0.
var PriceList = []KnownModel{
	{Provider: Anthropic, ID: "claude-sonnet-5-5", Name: "Claude Sonnet 5.5", Input: 2, Output: 10, ImageInput: true},
	{Provider: Anthropic, ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5", Input: 1, Output: 5, ImageInput: true},
	{Provider: Anthropic, ID: "claude-sonnet-4-5", Name: "Claude Sonnet 4.5", Input: 3, Output: 15, ImageInput: true},
	{Provider: Anthropic, ID: "claude-opus-4-5", Name: "Claude Opus 4.5", Input: 5, Output: 25, ImageInput: true},
	{Provider: OpenAI, ID: "gpt-5", Name: "GPT-5", Input: 1.25, Output: 10, ImageInput: true},
	{Provider: OpenAI, ID: "gpt-5-mini", Name: "GPT-5 mini", Input: 0.25, Output: 2, ImageInput: true},
	{Provider: OpenAI, ID: "gpt-4.1", Name: "GPT-4.1", Input: 2, Output: 8, ImageInput: true},
	{Provider: OpenAI, ID: "gpt-4o-mini", Name: "GPT-4o mini", Input: 0.15, Output: 0.6, ImageInput: true},
	{Provider: Google, ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", Input: 1.25, Output: 10, ImageInput: true},
	{Provider: Google, ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", Input: 0.3, Output: 2.5, ImageInput: true},
	{Provider: Google, ID: "gemini-2.5-flash-lite", Name: "Gemini 2.5 Flash-Lite", Input: 0.1, Output: 0.4, ImageInput: true},
}

// Known returns the price list's entry for a model, matching dated
// snapshots ("claude-haiku-4-5-20251001") and Azure deployments named after
// their model.
func Known(provider, model string) (KnownModel, bool) {
	m := strings.ToLower(strings.TrimSpace(model))
	p := provider
	if p == Azure {
		p = OpenAI
	}
	var best KnownModel
	for _, k := range PriceList {
		if k.Provider != p {
			continue
		}
		if m == k.ID {
			return k, true
		}
		if strings.HasPrefix(m, k.ID+"-") && len(k.ID) > len(best.ID) {
			best = k
		}
	}
	return best, best.ID != ""
}

// ModelName is how people call a model: the price list's name, else the id.
func ModelName(provider, model string) string {
	if k, ok := Known(provider, model); ok && strings.EqualFold(k.ID, model) {
		return k.Name
	}
	return model
}

// visionHints are parts of the names of open models that read images.
var visionHints = []string{"vision", "llava", "-vl", "vl:", "gemma3", "pixtral", "minicpm-v", "moondream", "qwen2.5vl", "qwen3-vl", "llama4", "mistral-small3"}

// ImageInput reports whether a model reads images: the hosted providers'
// current models do; an open model does when its name says so.
func ImageInput(provider, model string) bool {
	if k, ok := Known(provider, model); ok {
		return k.ImageInput
	}
	if provider != Compatible {
		return true
	}
	m := strings.ToLower(model)
	for _, h := range visionHints {
		if strings.Contains(m, h) {
			return true
		}
	}
	return false
}
