package cadence

import (
	"slices"
	"testing"
)

func TestFrequencies(t *testing.T) {
	for _, c := range []struct {
		cadence, digest string
		want            []string
	}{
		{"", "", []string{"all"}},
		{"regular", "", []string{"all"}},
		{"announcement", "", nil},
		{"announcement", "weekly", nil},
		{"", "weekly", []string{"all", "weekly"}},
		{"", "monthly", []string{"all", "monthly"}},
		{"", "daily", []string{"all"}},
		{"", "items", []string{"all"}},
	} {
		if got := Frequencies(c.cadence, c.digest); !slices.Equal(got, c.want) {
			t.Errorf("Frequencies(%q, %q) = %v, want %v", c.cadence, c.digest, got, c.want)
		}
	}
}
