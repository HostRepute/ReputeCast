package engine

import (
	"encoding/json"

	"github.com/google/uuid"
)

// parseID reads an id stored as text, ok false when there is none.
func parseID(s string) (uuid.UUID, bool) {
	id, err := uuid.Parse(s)
	return id, err == nil
}

func jsonUnmarshal(b []byte, v any) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, v)
}
