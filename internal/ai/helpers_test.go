package ai_test

import (
	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/apptest"
)

// aiService is a bare service on the test app's database, for the jobs.
func aiService(env *apptest.Env) *ai.Service {
	return &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets}
}
