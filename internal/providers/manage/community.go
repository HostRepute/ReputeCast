package manage

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CommunityKindRemote is the kind of community adapters that run in their
// own process and speak the adapter protocol (docs/adapters.md).
const CommunityKindRemote = "remote"

// CommunityRuntime runs installed community adapters of kind "remote"
// (internal/providers/remote implements it). The Adapters tab calls it so an
// install, update or removal takes effect at once; nil means this build
// can't run them, and their manifests are refused.
type CommunityRuntime interface {
	// ParseManifest validates a manifest of kind "remote" and returns its
	// registry key ("community:<id>") and the capabilities to list.
	ParseManifest(raw []byte) (key string, capabilities []string, err error)
	// Installed runs inside the transaction that installs such an adapter.
	Installed(ctx context.Context, tx pgx.Tx, adapterID uuid.UUID, raw []byte) error
	// Changed runs after an install, update or removal is committed.
	Changed(ctx context.Context)
}

// communityKey is the registry key of an installed community adapter:
// "community:<id>" for remote adapters, "community:<name>" for the others.
func (s *Service) communityKey(kind, name string, manifest []byte) (string, []string, bool) {
	if kind == CommunityKindRemote && s.Community != nil {
		if key, caps, err := s.Community.ParseManifest(manifest); err == nil {
			return key, caps, true
		}
	}
	return "community:" + name, nil, false
}

func (s *Service) communityChanged(ctx context.Context, kind string) {
	if kind == CommunityKindRemote && s.Community != nil {
		s.Community.Changed(ctx)
	}
}
