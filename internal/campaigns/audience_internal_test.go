package campaigns

import (
	"encoding/json"
	"slices"
	"testing"
)

// A composer save keeps what Review & send set on the audience: "Include
// them" (the unengaged, SR-11) and the sunset policy's override.
func TestSaveKeepsReviewAudienceOptions(t *testing.T) {
	prev := json.RawMessage(`{"lists":["a"],"include_unengaged":true,"include_sunset":true,"exclude_tiers":["at_risk"]}`)
	var got struct {
		Lists            []string `json:"lists"`
		IncludeUnengaged bool     `json:"include_unengaged"`
		IncludeSunset    bool     `json:"include_sunset"`
		ExcludeTiers     []string `json:"exclude_tiers"`
	}
	if err := json.Unmarshal(ComposerAudience{Lists: []string{"b"}}.toDB(prev), &got); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Lists, []string{"b"}) || !got.IncludeUnengaged || !got.IncludeSunset || !slices.Equal(got.ExcludeTiers, []string{"at_risk"}) {
		t.Fatalf("saved: %+v", got)
	}
}
