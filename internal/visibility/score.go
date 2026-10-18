package visibility

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/scoring"
	"github.com/hostrepute/reputecast/internal/store"
)

// The score and its band (contact scoring, audience-insights brief §3.1):
// "Score is at least / at most / between" and "Score band is", as the
// segment builder reads them. Live scores come from internal/scoring in
// bulk, once per count, send or preview. While the workspace has no
// scoring nobody has a score or a band: score conditions and "Score band
// is" never match, "Score band isn't" always does (as in segments).

func init() { scoring.RegisterUses(scoreUses) }

// Bands are the score bands in the picker's order.
var Bands = []string{scoring.BandHot, scoring.BandWarm, scoring.BandCold}

// scoreFact is a contact's score as conditions read it (Facts.Extra[KindScore]).
type scoreFact struct {
	// Has is false while the workspace has no scoring.
	Has   bool
	Score int
	Band  string
}

func scoreOf(f *Facts) scoreFact {
	s, _ := f.Extra[KindScore].(scoreFact)
	return s
}

// loadScores reads the contacts' live scores and bands. Both kinds use it;
// the second one finds them loaded.
func loadScores(ctx context.Context, q *store.Queries, wsID uuid.UUID, _ []VisibilityCondition, facts map[uuid.UUID]*Facts) error {
	ids := make([]uuid.UUID, 0, len(facts))
	for id, f := range facts {
		if _, done := f.Extra[KindScore]; done {
			return nil
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	bands, ok, err := scoring.Bands(ctx, q, wsID, ids)
	if err != nil {
		return err
	}
	for id, f := range facts {
		if f.Extra == nil {
			f.Extra = map[string]any{}
		}
		b := bands[id]
		f.Extra[KindScore] = scoreFact{Has: ok, Score: b.Score, Band: b.Band}
	}
	return nil
}

// scoreNumber reads a score of a condition: a whole number from 0 to 100.
func scoreNumber(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	return n, err == nil && n >= 0 && n <= 100
}

func matchScore(c VisibilityCondition, f *Facts) bool {
	s := scoreOf(f)
	a, ok := scoreNumber(c.Value)
	if !s.Has || !ok {
		return false
	}
	switch c.Op {
	case "at_most":
		return s.Score <= a
	case "between":
		b, ok := scoreNumber(c.To)
		return ok && s.Score >= a && s.Score <= b
	}
	return s.Score >= a
}

func matchScoreBand(c VisibilityCondition, f *Facts) bool {
	s := scoreOf(f)
	is := s.Has && s.Band == c.Value
	if c.Op == "is_not" {
		return !is
	}
	return is
}

func checkScore(c VisibilityCondition) error {
	a, ok := scoreNumber(c.Value)
	if !ok {
		return errors.New("the score needs a whole number from 0 to 100")
	}
	if c.Op != "between" {
		return nil
	}
	b, ok := scoreNumber(c.To)
	if !ok {
		return errors.New("enter both ends of the range, from 0 to 100")
	}
	if b < a {
		return errors.New("the range ends below where it starts")
	}
	return nil
}

func checkScoreBand(c VisibilityCondition) error {
	if !slices.Contains(Bands, c.Value) {
		return errors.New("pick Hot, Warm or Cold")
	}
	return nil
}

var (
	mBandHot  = msgs.M("band.hot")
	mBandWarm = msgs.M("band.warm")
	mBandCold = msgs.M("band.cold")

	mDescScore        = msgs.M("describe.score")
	mDescScoreBand    = msgs.M("describe.scoreBand")
	mDescNotScoreBand = msgs.M("describe.notScoreBand")
	mWhyScore         = msgs.M("why.score")
	mWhyNoScore       = msgs.M("why.noScore")
	mWhyScoreBand     = msgs.M("why.scoreBand")
	mWhyScoreBandNot  = msgs.M("why.scoreBandNot")
	mTraitScore       = msgs.M("trait.score")
	mTraitNoScore     = msgs.M("trait.noScore")
)

// bandName is a band's name in the reader's language ("Hot").
func bandName(l i18n.Lang, band string) string {
	switch band {
	case scoring.BandHot:
		return mBandHot.Text(l, nil)
	case scoring.BandWarm:
		return mBandWarm.Text(l, nil)
	case scoring.BandCold:
		return mBandCold.Text(l, nil)
	}
	return band
}

func describeScore(c VisibilityCondition, n *Names) string {
	return mDescScore.Text(n.Lang, i18n.Args{"op": c.Op, "value": strings.TrimSpace(c.Value), "to": strings.TrimSpace(c.To)})
}

func describeScoreBand(c VisibilityCondition, n *Names) string {
	if c.Op == "is_not" {
		return mDescNotScoreBand.Text(n.Lang, i18n.Args{"band": bandName(n.Lang, c.Value)})
	}
	return mDescScoreBand.Text(n.Lang, i18n.Args{"band": bandName(n.Lang, c.Value)})
}

func whyScore(_ VisibilityCondition, f *Facts, _ bool, n *Names) string {
	s := scoreOf(f)
	if !s.Has {
		return mWhyNoScore.Text(n.Lang, i18n.Args{"name": f.Name()})
	}
	return mWhyScore.Text(n.Lang, i18n.Args{"name": f.Name(), "score": s.Score})
}

func whyScoreBand(c VisibilityCondition, f *Facts, matched bool, n *Names) string {
	s := scoreOf(f)
	if !s.Has {
		return mWhyNoScore.Text(n.Lang, i18n.Args{"name": f.Name()})
	}
	actual := bandName(n.Lang, s.Band)
	if c.Op == "is" && !matched {
		return mWhyScoreBandNot.Text(n.Lang, i18n.Args{"name": f.Name(), "actual": actual, "band": bandName(n.Lang, c.Value)})
	}
	return mWhyScoreBand.Text(n.Lang, i18n.Args{"name": f.Name(), "band": actual})
}

// scoreTrait is what the preview says the rules read about the contact's
// score ("score 52 · Hot").
func scoreTrait(f *Facts, n *Names) string {
	s := scoreOf(f)
	if !s.Has {
		return mTraitNoScore.Text(n.Lang, nil)
	}
	return mTraitScore.Text(n.Lang, i18n.Args{"score": s.Score, "band": bandName(n.Lang, s.Band)})
}

// scoreUseCondition is a score condition as Scoring's Used by summarises
// it, in the segment builder's terms.
type scoreUseCondition struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value string `json:"value"`
	To    string `json:"to,omitempty"`
}

// scoreConditions are the conditions of a rule that read the score.
func scoreConditions(r VisibilityRule) []scoreUseCondition {
	if r.Show != ShowOnly && r.Show != ShowExcept {
		return nil
	}
	var out []scoreUseCondition
	for _, c := range r.Conditions {
		if c.Kind == KindScore || c.Kind == KindScoreBand {
			out = append(out, scoreUseCondition{Field: c.Kind, Op: c.Op, Value: c.Value, To: c.To})
		}
	}
	return out
}

// scoreUses lists, for Scoring's Used by, the conditional blocks of emails
// not sent yet whose rules read the score, with how many recipients of the
// email's last count see each (while that count is current).
func scoreUses(ctx context.Context, q *store.Queries, ws *store.Workspace) ([]scoring.ScoringUse, error) {
	rows, err := q.VisibilityUnsent(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := []scoring.ScoringUse{}
	for _, r := range rows {
		if !strings.Contains(r.Mjml, KindScore) {
			continue
		}
		e := Read(r.Mjml)
		var shown map[string]int64
		var total *int64
		counted := false
		for _, b := range e.Blocks {
			conds := scoreConditions(b.Rule)
			if len(conds) == 0 {
				continue
			}
			if !counted {
				counted = true
				if c, err := q.VisibilityGetCount(ctx, r.ID); err == nil && c.RulesHash == e.Hash() && c.AudienceHash == ParseAudience(r.Audience).hash() {
					_ = json.Unmarshal(c.Shown, &shown)
					t := c.Total
					total = &t
				}
			}
			raw, _ := json.Marshal(conds)
			path := make([]string, len(b.Path))
			for i, p := range b.Path {
				path[i] = strconv.Itoa(p)
			}
			u := scoring.ScoringUse{Kind: "block", ID: r.ID.String() + ":" + b.ID, Name: b.Name, Status: r.Status, Where: r.Name, Show: b.Rule.Show,
				Conditions: raw, Href: "/w/" + ws.Slug + "/campaigns/" + r.ID.String() + "/edit?select=" + strings.Join(path, "-") + "&panel=settings"}
			if total != nil {
				n := shown[b.ID]
				u.Contacts, u.Of = &n, total
			}
			out = append(out, u)
		}
	}
	return out, nil
}
