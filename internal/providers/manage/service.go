package manage

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service is the sending-providers domain.
type Service struct {
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Audit   *audit.Logger
	Factory *Factory
	Jobs    func() *jobs.Client
	URLs    publicurl.Builder
	Sending func() sending.Starter
	Log     *slog.Logger
	Now     func() time.Time
	// CheckAddress overrides the webhook address check (tests); nil looks
	// the address up in DNS. ReputeCast never calls its own URL (PR-9).
	CheckAddress func(ctx context.Context, url string) (AddressProblem, bool)
	// Community runs installed community adapters; nil when the build has no
	// runtime for them.
	Community CommunityRuntime
	// HTTPMappings checks and previews generic HTTP adapter mappings; nil
	// when the build has no generic HTTP adapter.
	HTTPMappings HTTPMappings
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}
