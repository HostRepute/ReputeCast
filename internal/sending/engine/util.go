package engine

import (
	"encoding/json"
)

func jsonUnmarshal(b []byte, v any) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, v)
}

// yesNo is a select argument of the messages ("yes" or "no").
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
