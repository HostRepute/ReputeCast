package dependents

import (
	"testing"

	"github.com/google/uuid"
)

func TestFieldMatchesEachKindOfReference(t *testing.T) {
	seg := uuid.MustParse("0192f000-0000-7000-8000-000000000001")
	idx := Index{items: []item{
		{kind: KindSegment, id: "s1", name: "Paid plans", text: `{"groups":[{"conditions":[{"field":"custom:plan","op":"eq"}]}]}`},
		{kind: KindSegment, id: "s2", name: "Planners", text: `{"groups":[{"conditions":[{"field":"custom:planner"}]}]}`},
		{kind: KindSegment, id: "s3", name: "SQL", sql: "SELECT id FROM contacts_view WHERE plan = 'team'"},
		{kind: KindForm, id: "f1", name: "Signup", text: `[{"key": "plan"}]`},
		{kind: KindCampaign, id: "c1", name: "November", text: `Hi {{ contact.plan | default: "friend" }}`},
		{kind: KindCampaign, id: "c2", name: "December", text: `Hi {{ contact.planet }}`},
		{kind: KindView, id: "v1", name: "Paying", text: `{} field:plan`},
		{kind: KindDigest, id: "d1", name: "Weekly", text: `{"segmentId":"` + seg.String() + `"}`},
	}}
	got := map[string]bool{}
	for _, u := range idx.Field("plan") {
		got[u.ID] = true
	}
	for _, id := range []string{"s1", "s3", "f1", "c1", "v1"} {
		if !got[id] {
			t.Errorf("%s uses plan but wasn't found", id)
		}
	}
	for _, id := range []string{"s2", "c2", "d1"} {
		if got[id] {
			t.Errorf("%s doesn't use plan but was found", id)
		}
	}
	uses := idx.Segment(seg)
	if len(uses) != 1 || uses[0].ID != "d1" || uses[0].Kind != KindDigest {
		t.Fatalf("segment uses: %+v", uses)
	}
	if out := idx.Field("nothing_uses_this"); out == nil || len(out) != 0 {
		t.Fatalf("no uses must be an empty list, got %#v", out)
	}
}
