package growth

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Kinds later phases register (Grow's referral and partner sources) take a
// join before the built-in rules, and name their sources.
func TestRegisteredKinds(t *testing.T) {
	RegisterKind(Kind{Key: "test_partner", Order: 14})
	RegisterAttributor(func(contactSource, consentSource string) (string, string, bool) {
		if id, ok := strings.CutPrefix(consentSource, "test_partner:"); ok {
			return "test_partner", id, true
		}
		return "", "", false
	})
	joined := time.Now()
	src := "test_partner:abc"
	subscribed := "subscribed"
	s := &Service{}
	n := &names{forms: map[string]store.GrowthFormsOfRow{}, formsBySlug: map[string]store.GrowthFormsOfRow{}, extra: map[string]map[string]string{"test_partner": {"abc": "Partner · Ops Weekly"}}}
	a, ok := s.attribute(candidate{id: uuid.New(), createdAt: joined, joinedAt: &joined, action: &subscribed, evSource: &src}, n, nil, nil)
	if !ok || a.kind != "test_partner" || a.id != "abc" || !a.joinedAt.Equal(joined) {
		t.Fatalf("attribution: %+v %v", a, ok)
	}
	if d := n.describe(SourceKey(a.kind, a.id)); d.Name != "Partner · Ops Weekly" || EnglishLabel(d) != "Partner · Ops Weekly" || d.Order != 14 {
		t.Fatalf("described: %+v", d)
	}
	// A form signup still reads as the form.
	form := "form:newsletter"
	if a, ok := s.attribute(candidate{id: uuid.New(), createdAt: joined, joinedAt: &joined, action: &subscribed, evSource: &form}, n, nil, nil); !ok || a.kind != KindForm {
		t.Fatalf("form: %+v", a)
	}
}
