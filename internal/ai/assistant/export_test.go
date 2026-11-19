package assistant

import "github.com/google/uuid"

// SetAfterAnswer runs f once the provider answered a turn, before its
// checkpoint.
func SetAfterAnswer(s *Service, f func(turn uuid.UUID)) { s.afterAnswer = f }
