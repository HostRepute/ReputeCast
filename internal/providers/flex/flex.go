// Package flex holds setting types that accept what a connection form may
// store. providers.Field has no type, so a number such as a plan's monthly
// allowance can arrive as a JSON number or as a string; adapters decode it
// either way instead of failing to open.
package flex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Int is a whole number written as 10000, "10000" or "10,000"; null and ""
// are 0.
type Int int64

// UnmarshalJSON implements json.Unmarshaler.
func (n *Int) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if bytes.Equal(b, []byte("null")) {
		*n = 0
		return nil
	}
	s := string(b)
	if len(b) > 0 && b[0] == '"' {
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		s = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(s), ",", ""), "_", "")
		if s == "" {
			*n = 0
			return nil
		}
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		f, ferr := strconv.ParseFloat(s, 64)
		if ferr != nil || f != float64(int64(f)) {
			return fmt.Errorf("%q is not a whole number", s)
		}
		v = int64(f)
	}
	*n = Int(v)
	return nil
}
