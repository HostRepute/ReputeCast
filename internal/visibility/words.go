package visibility

import (
	"embed"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What rules say in words (badges, summaries, "Show why"), in the reader's
// app language, from messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "visibility")

var (
	mTierActive  = msgs.M("tier.active")
	mTierCooling = msgs.M("tier.cooling")
	mTierAtRisk  = msgs.M("tier.atRisk")
	mTierSunset  = msgs.M("tier.sunset")

	mDescList          = msgs.M("describe.list")
	mDescNotList       = msgs.M("describe.notList")
	mDescSegment       = msgs.M("describe.segment")
	mDescNotSegment    = msgs.M("describe.notSegment")
	mDescTag           = msgs.M("describe.tag")
	mDescNotTag        = msgs.M("describe.notTag")
	mDescField         = msgs.M("describe.field")
	mDescLanguage      = msgs.M("describe.language")
	mDescNotLanguage   = msgs.M("describe.notLanguage")
	mDescTier          = msgs.M("describe.tier")
	mDescNotTier       = msgs.M("describe.notTier")
	mDescDeletedSeg    = msgs.M("describe.deletedSegment")
	mDescDeletedList   = msgs.M("describe.deletedList")
	mDescDeletedField  = msgs.M("describe.deletedField")
	mDescMoreAll       = msgs.M("describe.moreAll")
	mDescMoreAny       = msgs.M("describe.moreAny")
	mDescExcept        = msgs.M("describe.except")
	mDescNobody        = msgs.M("describe.nobody")
	mWhyOnList         = msgs.M("why.onList")
	mWhyNotOnList      = msgs.M("why.notOnList")
	mWhyInSegment      = msgs.M("why.inSegment")
	mWhyNotInSegment   = msgs.M("why.notInSegment")
	mWhyHasTag         = msgs.M("why.hasTag")
	mWhyNoTag          = msgs.M("why.noTag")
	mWhyField          = msgs.M("why.field")
	mWhyNoField        = msgs.M("why.noField")
	mWhyLanguage       = msgs.M("why.language")
	mWhyNoLanguage     = msgs.M("why.noLanguage")
	mWhyTierNot        = msgs.M("why.tierNot")
	mWhyTier           = msgs.M("why.tier")
	mWhyNobody         = msgs.M("why.nobody")
	mWhyHidden         = msgs.M("why.hidden")
	mWhyEmpty          = msgs.M("why.empty")
	mProblemNoContact  = msgs.M("problem.noContact")
	mTraitField        = msgs.M("trait.field")
	mTraitNoField      = msgs.M("trait.noField")
	mTraitOnList       = msgs.M("trait.onList")
	mTraitNotOnList    = msgs.M("trait.notOnList")
	mTraitInSegment    = msgs.M("trait.inSegment")
	mTraitNotInSegment = msgs.M("trait.notInSegment")
	mTraitTag          = msgs.M("trait.tag")
	mTraitNoTag        = msgs.M("trait.noTag")
)

func describeList(c VisibilityCondition, n *Names) string {
	if !n.Exists(Ref{Kind: "list", ID: c.Value}) {
		return mDescDeletedList.Text(n.Lang, nil)
	}
	name := n.list(c.Value, c.Label)
	if c.Op == "excludes" {
		return mDescNotList.Text(n.Lang, i18n.Args{"list": name})
	}
	return mDescList.Text(n.Lang, i18n.Args{"list": name})
}

func describeSegment(c VisibilityCondition, n *Names) string {
	if !n.Exists(Ref{Kind: "segment", ID: c.Value}) {
		return mDescDeletedSeg.Text(n.Lang, nil)
	}
	name := n.segment(c.Value, c.Label)
	if c.Op == "not_in" {
		return mDescNotSegment.Text(n.Lang, i18n.Args{"segment": name})
	}
	return mDescSegment.Text(n.Lang, i18n.Args{"segment": name})
}

func describeTag(c VisibilityCondition, n *Names) string {
	if c.Op == "has_not" {
		return mDescNotTag.Text(n.Lang, i18n.Args{"tag": c.Value})
	}
	return mDescTag.Text(n.Lang, i18n.Args{"tag": c.Value})
}

func describeField(c VisibilityCondition, n *Names) string {
	if !n.Exists(Ref{Kind: "field", ID: c.Field}) {
		return mDescDeletedField.Text(n.Lang, nil)
	}
	return mDescField.Text(n.Lang, i18n.Args{"field": c.Field, "op": c.Op, "value": c.Value})
}

func describeLanguage(c VisibilityCondition, n *Names) string {
	name := languageName(c.Value)
	if c.Op == "is_not" {
		return mDescNotLanguage.Text(n.Lang, i18n.Args{"language": name})
	}
	return mDescLanguage.Text(n.Lang, i18n.Args{"language": name})
}

func describeTier(c VisibilityCondition, n *Names) string {
	name := tierName(n.Lang, c.Value)
	if c.Op == "is_not" {
		return mDescNotTier.Text(n.Lang, i18n.Args{"tier": name})
	}
	return mDescTier.Text(n.Lang, i18n.Args{"tier": name})
}

func whyList(c VisibilityCondition, f *Facts, _ bool, n *Names) string {
	name := n.list(c.Value, c.Label)
	if f.Lists[c.Value] {
		return mWhyOnList.Text(n.Lang, i18n.Args{"name": f.Name(), "list": name})
	}
	return mWhyNotOnList.Text(n.Lang, i18n.Args{"name": f.Name(), "list": name})
}

func whySegment(c VisibilityCondition, f *Facts, _ bool, n *Names) string {
	name := n.segment(c.Value, c.Label)
	if f.Segments[c.Value] {
		return mWhyInSegment.Text(n.Lang, i18n.Args{"name": f.Name(), "segment": name})
	}
	return mWhyNotInSegment.Text(n.Lang, i18n.Args{"name": f.Name(), "segment": name})
}

func whyTag(c VisibilityCondition, f *Facts, _ bool, n *Names) string {
	if f.Tags[c.Value] {
		return mWhyHasTag.Text(n.Lang, i18n.Args{"name": f.Name(), "tag": c.Value})
	}
	return mWhyNoTag.Text(n.Lang, i18n.Args{"name": f.Name(), "tag": c.Value})
}

func whyField(c VisibilityCondition, f *Facts, _ bool, n *Names) string {
	v := fieldText(f.Attributes[c.Field])
	if v == "" {
		return mWhyNoField.Text(n.Lang, i18n.Args{"name": f.Name(), "field": c.Field})
	}
	return mWhyField.Text(n.Lang, i18n.Args{"name": f.Name(), "field": c.Field, "value": v})
}

func whyLanguage(_ VisibilityCondition, f *Facts, _ bool, n *Names) string {
	if strings.TrimSpace(f.Language) == "" {
		return mWhyNoLanguage.Text(n.Lang, i18n.Args{"name": f.Name()})
	}
	return mWhyLanguage.Text(n.Lang, i18n.Args{"name": f.Name(), "language": languageName(f.Language)})
}

func whyTier(c VisibilityCondition, f *Facts, matched bool, n *Names) string {
	actual := tierName(n.Lang, f.Tier)
	if c.Op == "is" && !matched {
		return mWhyTierNot.Text(n.Lang, i18n.Args{"name": f.Name(), "actual": actual, "tier": tierName(n.Lang, c.Value)})
	}
	return mWhyTier.Text(n.Lang, i18n.Args{"name": f.Name(), "tier": actual})
}

// Describe says a rule in words: the badge on the block ("Beta Program
// list", "plan is empty", "Except At risk tier and 1 more").
func Describe(r VisibilityRule, n *Names) string {
	if r.Show == ShowNobody {
		return mDescNobody.Text(n.Lang, nil)
	}
	if len(r.Conditions) == 0 {
		return ""
	}
	first := describeCondition(r.Conditions[0], n)
	words := first
	if rest := len(r.Conditions) - 1; rest > 0 {
		m := mDescMoreAll
		if r.Match == MatchAny {
			m = mDescMoreAny
		}
		words = m.Text(n.Lang, i18n.Args{"first": first, "count": rest})
	}
	if r.Show == ShowExcept {
		return mDescExcept.Text(n.Lang, i18n.Args{"words": words})
	}
	return words
}

func describeCondition(c VisibilityCondition, n *Names) string {
	k, ok := Lookup(c.Kind)
	if !ok || k.Describe == nil {
		return c.Kind
	}
	return k.Describe(c, n)
}

// Why says why a contact doesn't get a block ("Sam Rivera isn't on Beta
// Program"): the condition that decided it.
func Why(r VisibilityRule, matched []bool, f *Facts, n *Names) string {
	if r.Show == ShowNobody {
		return mWhyNobody.Text(n.Lang, nil)
	}
	pick := -1
	for i, m := range matched {
		// Only: a condition that didn't match left it out (with ALL), or
		// the first one (with ANY, none matched). Except: one that matched.
		if (r.Show == ShowOnly && !m) || (r.Show == ShowExcept && m) {
			pick = i
			break
		}
	}
	if pick < 0 {
		pick = 0
	}
	if pick >= len(r.Conditions) {
		return ""
	}
	c := r.Conditions[pick]
	k, ok := Lookup(c.Kind)
	if !ok || k.Why == nil {
		return ""
	}
	return k.Why(c, f, matched[pick], n)
}
