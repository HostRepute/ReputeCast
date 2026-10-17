package segments

// The contact's language in the builder (design expansion M17, brief
// §3.4): "Language is Spanish" matches es and its regional codes (es-MX,
// es-ES); "Language is No language" matches contacts without one.

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/store"
)

// LanguageNone is the language condition's value for contacts with no
// language.
const LanguageNone = "none"

// languageOptions are the builder's choices: the workspace's languages by
// name, then "No language".
func languageOptions(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]SegmentOption, error) {
	st, err := (&languages.Service{Q: q}).Settings(ctx, q, wsID)
	if err != nil {
		return nil, err
	}
	out := make([]SegmentOption, 0, len(st.Languages)+1)
	for _, code := range st.Languages {
		out = append(out, SegmentOption{Value: code, Label: languages.NameOf(code)})
	}
	return append(out, SegmentOption{Value: LanguageNone, Label: "No language"}), nil
}

// validLanguage reports whether a condition's value is a language or "none".
func validLanguage(v string) bool {
	if v == LanguageNone {
		return true
	}
	_, ok := languages.Lookup(v)
	return ok && languages.Base(v) == v
}

// languageCondition renders a language condition over the segment views.
func languageCondition(t *sqlText, c SegmentCondition) {
	if c.Value == LanguageNone {
		if c.Op == "is_not" {
			t.raw("c.language <> ''")
		} else {
			t.raw("c.language = ''")
		}
		return
	}
	op := " = "
	if c.Op == "is_not" {
		op = " IS DISTINCT FROM "
	}
	t.raw("lower(split_part(replace(c.language, '_', '-'), '-', 1))" + op)
	t.bind(str(c.Value))
}
