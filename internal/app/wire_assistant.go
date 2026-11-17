package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/ai/assistant"
)

// assistant is the AI assistant's threads (B3), on the AI service's
// models, budget and projections.
func (a *App) assistant() *assistant.Service {
	return &assistant.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, AI: a.ai(), Log: a.Log}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		assistant.Register(h, a.assistant())
	})
}
