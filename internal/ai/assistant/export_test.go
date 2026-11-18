package assistant

import "github.com/google/uuid"

// SetAfterAnswer runs f once the provider answered a turn, before its
// checkpoint.
func SetAfterAnswer(s *Service, f func(turn uuid.UUID)) { s.afterAnswer = f }

// SetClaimHook runs f in a claim's transaction: once its locks are taken
// ("locked"), and once its slot is ("slot").
func SetClaimHook(s *Service, f func(turn uuid.UUID, stage string)) { s.inClaim = f }
