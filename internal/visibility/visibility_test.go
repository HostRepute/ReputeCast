package visibility

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/mjml"
)

func TestRegistry(t *testing.T) {
	var keys []string
	groups := map[string]string{}
	for _, k := range Kinds() {
		keys = append(keys, k.Key)
		groups[k.Key] = k.Group
	}
	// The built-in kinds come first; the test binary also links phases that
	// register their own (Earn's paid access).
	builtin := "segment,list,tag,field,language,tier,score,score_band"
	if got := strings.Join(keys, ","); got != builtin && !strings.HasPrefix(got, builtin+",") {
		t.Fatalf("kinds in the picker's order: %s", got)
	}
	registered := strings.Join(keys, ",")
	if groups["segment"] != GroupLists || groups["tag"] != GroupLists || groups["field"] != GroupContact || groups["language"] != GroupContact || groups["tier"] != GroupEngagement ||
		groups["score"] != GroupEngagement || groups["score_band"] != GroupEngagement {
		t.Fatalf("groups: %v", groups)
	}

	// A later phase registers its own kind (a paid tier); registering it
	// again replaces it without moving it.
	paid := Kind{Key: "paid_tier", Group: "paid", Ops: []Operator{{Key: "is"}},
		Match: func(c VisibilityCondition, f *Facts) bool { s, _ := f.Extra["paid_tier"].(string); return s == c.Value }}
	RegisterKind(paid)
	t.Cleanup(func() {
		registryMu.Lock()
		defer registryMu.Unlock()
		delete(registry, "paid_tier")
		order = slices.DeleteFunc(order, func(k string) bool { return k == "paid_tier" })
	})
	paid.Ops = []Operator{{Key: "is"}, {Key: "is_not"}}
	RegisterKind(paid)
	keys = keys[:0]
	for _, k := range Kinds() {
		keys = append(keys, k.Key)
	}
	if got := strings.Join(keys, ","); got != registered+",paid_tier" {
		t.Fatalf("after registering paid_tier twice: %s", got)
	}
	if k, ok := Lookup("paid_tier"); !ok || len(k.Ops) != 2 || k.Group != "paid" {
		t.Fatalf("paid_tier: %+v %v", k, ok)
	}
	// Its conditions decide with what it loads into Facts.Extra.
	e := Read(email(section("gold", rule("gold", ShowOnly, MatchAll, VisibilityCondition{Kind: "paid_tier", Op: "is", Value: "gold"}), "Gold members")))
	if d := Decide(e, &Facts{Extra: map[string]any{"paid_tier": "gold"}}, nil); d.Hidden["gold"] {
		t.Fatalf("a gold member should see the block")
	}
	if d := Decide(e, &Facts{Extra: map[string]any{"paid_tier": "free"}}, nil); !d.Hidden["gold"] {
		t.Fatalf("a free member shouldn't see the block")
	}

	// The field operators are the segment builder's, by type.
	opKeys := func(ops []Operator) string {
		var out []string
		for _, o := range ops {
			k := o.Key
			if o.NoValue {
				k += "*"
			}
			out = append(out, k)
		}
		return strings.Join(out, ",")
	}
	if got := opKeys(FieldOps("number")); got != "equals,not_equals,gt,lt,is_empty*,is_not_empty*" {
		t.Fatalf("number ops: %s", got)
	}
	if got := opKeys(FieldOps("boolean")); got != "is_true*,is_false*" {
		t.Fatalf("boolean ops: %s", got)
	}
	if got := opKeys(FieldOps("text")); got != "equals,not_equals,contains,is_empty*,is_not_empty*" {
		t.Fatalf("text ops: %s", got)
	}
}

// The score and its band read as the segment builder's: "Score band is"
// matches only contacts with that band, "isn't" everyone else (contacts
// without a score too), and without scoring nobody has a score.
func TestScoreConditions(t *testing.T) {
	src := email(
		section("demo", rule("demo", ShowOnly, MatchAll, VisibilityCondition{Kind: KindScoreBand, Op: "is", Value: "hot"}), "Book a demo"),
		section("nothot", rule("nothot", ShowOnly, MatchAll, VisibilityCondition{Kind: KindScoreBand, Op: "is_not", Value: "hot"}), "Read more"),
		section("least", rule("least", ShowOnly, MatchAll, VisibilityCondition{Kind: KindScore, Op: "at_least", Value: "50"}), "50 or more"),
		section("most", rule("most", ShowOnly, MatchAll, VisibilityCondition{Kind: KindScore, Op: "at_most", Value: "19"}), "19 or less"),
		section("range", rule("range", ShowExcept, MatchAll, VisibilityCondition{Kind: KindScore, Op: "between", Value: "20", To: "49"}), "Not warm"),
	)
	e := Read(src)
	hidden := func(s scoreFact) string {
		return Decide(e, &Facts{Extra: map[string]any{KindScore: s}}, nil).Hidden.String()
	}
	cases := []struct {
		s    scoreFact
		want string
	}{
		{scoreFact{Has: true, Score: 52, Band: "hot"}, "most,nothot"},
		{scoreFact{Has: true, Score: 50, Band: "hot"}, "most,nothot"},
		{scoreFact{Has: true, Score: 49, Band: "warm"}, "demo,least,most,range"},
		{scoreFact{Has: true, Score: 20, Band: "warm"}, "demo,least,most,range"},
		{scoreFact{Has: true, Score: 19, Band: "cold"}, "demo,least"},
		{scoreFact{Has: true, Score: 0, Band: "cold"}, "demo,least"},
		// No scoring: no score and no band.
		{scoreFact{}, "demo,least,most"},
	}
	for _, c := range cases {
		if got := hidden(c.s); got != c.want {
			t.Errorf("%+v: hidden %s, want %s", c.s, got, c.want)
		}
	}
	// Facts nothing was loaded into have no score either.
	if got := Decide(e, &Facts{}, nil).Hidden.String(); got != "demo,least,most" {
		t.Errorf("no facts: hidden %s", got)
	}

	n := &Names{}
	words := map[string]string{}
	for _, b := range e.Blocks {
		words[b.ID] = Describe(b.Rule, n)
	}
	if words["demo"] != "Score band is Hot" || words["nothot"] != "Score band isn’t Hot" || words["least"] != "Score is at least 50" ||
		words["most"] != "Score is at most 19" || words["range"] != "Except Score is between 20 and 49" {
		t.Fatalf("badges: %v", words)
	}
	sam := &Facts{FirstName: "Sam", LastName: "Rivera", Extra: map[string]any{KindScore: scoreFact{Has: true, Score: 41, Band: "warm"}}}
	demo, _ := e.Block("demo")
	if got := Why(demo.Rule, Decide(e, sam, nil).Matched["demo"], sam, n); got != "Sam Rivera’s score band is Warm, not Hot" {
		t.Fatalf("why: %s", got)
	}
	least, _ := e.Block("least")
	if got := Why(least.Rule, []bool{false}, sam, n); got != "Sam Rivera’s score is 41" {
		t.Fatalf("why: %s", got)
	}
	noScore := &Facts{Email: "kenji.w@icloud.com", Extra: map[string]any{KindScore: scoreFact{}}}
	if got := Why(demo.Rule, []bool{false}, noScore, n); got != "kenji.w@icloud.com has no score" {
		t.Fatalf("why without scoring: %s", got)
	}
	if got := strings.Join(traits(e, sam, n), " | "); got != "score 41 · Warm" {
		t.Fatalf("traits: %s", got)
	}
	if got := blockTrait(demo, sam, n); got != "score 41 · Warm" {
		t.Fatalf("block trait: %s", got)
	}

	// Values are checked: whole scores from 0 to 100, ranges in order, bands by key.
	for _, c := range []struct {
		cond VisibilityCondition
		ok   bool
	}{
		{VisibilityCondition{Kind: KindScore, Op: "at_least", Value: "50"}, true},
		{VisibilityCondition{Kind: KindScore, Op: "at_least", Value: "101"}, false},
		{VisibilityCondition{Kind: KindScore, Op: "at_most", Value: "4.5"}, false},
		{VisibilityCondition{Kind: KindScore, Op: "between", Value: "20", To: "49"}, true},
		{VisibilityCondition{Kind: KindScore, Op: "between", Value: "20"}, false},
		{VisibilityCondition{Kind: KindScore, Op: "between", Value: "49", To: "20"}, false},
		{VisibilityCondition{Kind: KindScore, Op: "is", Value: "50"}, false},
		{VisibilityCondition{Kind: KindScoreBand, Op: "is", Value: "hot"}, true},
		{VisibilityCondition{Kind: KindScoreBand, Op: "is_not", Value: "lukewarm"}, false},
	} {
		err := Validate(VisibilityRule{ID: "x1y2z3", Show: ShowOnly, Conditions: []VisibilityCondition{c.cond}}, n)
		if (err == nil) != c.ok {
			t.Errorf("%+v: %v", c.cond, err)
		}
	}
}

func TestRuleParseAndShows(t *testing.T) {
	r, err := ParseRule(`{"id":"ab12cd","show":"sometimes","conditions":null}`)
	if err != nil || r.Show != ShowEveryone || r.Match != MatchAll || r.Conditions == nil || r.Conditional() {
		t.Fatalf("an unknown show is everyone: %+v %v", r, err)
	}
	if _, err := ParseRule("  "); !errors.Is(err, ErrNoRule) {
		t.Fatalf("empty attribute: %v", err)
	}
	if _, err := ParseRule("{not json"); err == nil {
		t.Fatal("broken JSON should fail")
	}
	nobody := VisibilityRule{ID: "ab12cd", Show: ShowNobody}
	if !nobody.Conditional() || nobody.Encode() != `{"id":"ab12cd","show":"nobody","match":"all","conditions":[]}` {
		t.Fatalf("nobody: %s", nobody.Encode())
	}

	cases := []struct {
		show, match string
		matched     []bool
		want        bool
	}{
		{ShowEveryone, MatchAll, nil, true},
		{ShowNobody, MatchAll, []bool{true}, false},
		{ShowOnly, MatchAll, []bool{true, true}, true},
		{ShowOnly, MatchAll, []bool{true, false}, false},
		{ShowOnly, MatchAny, []bool{false, true}, true},
		{ShowOnly, MatchAny, []bool{false, false}, false},
		{ShowExcept, MatchAll, []bool{true, true}, false},
		{ShowExcept, MatchAll, []bool{true, false}, true},
		{ShowExcept, MatchAny, []bool{false, true}, false},
		{ShowExcept, MatchAny, []bool{false, false}, true},
	}
	for _, c := range cases {
		r := VisibilityRule{ID: "x1y2z3", Show: c.show, Match: c.match}
		for range c.matched {
			r.Conditions = append(r.Conditions, VisibilityCondition{Kind: KindTag, Op: "has", Value: "x"})
		}
		if got := r.Shows(c.matched); got != c.want {
			t.Errorf("%s %s %v: shows %v, want %v", c.show, c.match, c.matched, got, c.want)
		}
	}
	// Only or except without conditions leaves nobody out.
	if !(VisibilityRule{Show: ShowOnly}).Shows(nil) {
		t.Fatal("only with no conditions shows the block")
	}
}

// The test email: a header, the sections given, and a footer.
func email(sections ...string) string {
	return `<mjml><mj-body>
<!-- rc:header -->
<mj-section><mj-column><mj-text>Acme</mj-text></mj-column></mj-section>
` + strings.Join(sections, "\n") + `
<!-- rc:footer -->
<mj-section><mj-column><mj-text>Acme Inc · <a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>
</mj-body></mjml>`
}

func rule(id, show, match string, conds ...VisibilityCondition) VisibilityRule {
	return VisibilityRule{ID: id, Show: show, Match: match, Conditions: conds}
}

func section(id string, r VisibilityRule, text string) string {
	attr := ""
	if r.ID != "" {
		attr = ` rc-visible="` + strings.ReplaceAll(r.Encode(), `"`, "&quot;") + `"`
	}
	return `<mj-section rc-name="` + id + `"` + attr + `><mj-column><mj-text>` + text + `</mj-text></mj-column></mj-section>`
}

func TestConditionsAndCombinations(t *testing.T) {
	beta, seg := uuid.NewString(), uuid.NewString()
	src := email(
		section("missed", rule("missed", ShowOnly, MatchAll, VisibilityCondition{Kind: KindTier, Op: "is", Value: "at_risk"}), "We've missed you."),
		section("beta", rule("beta", ShowOnly, MatchAll, VisibilityCondition{Kind: KindList, Op: "includes", Value: beta}), "Beta corner"),
		section("trial", rule("trial", ShowOnly, MatchAll, VisibilityCondition{Kind: KindField, Op: "is_empty", Field: "plan"}), "Start a free trial"),
		section("spanish", rule("spanish", ShowOnly, MatchAll, VisibilityCondition{Kind: KindLanguage, Op: "is", Value: "es"}), "Hola"),
		section("vip", rule("vip", ShowExcept, MatchAny,
			VisibilityCondition{Kind: KindTag, Op: "has", Value: "vip"},
			VisibilityCondition{Kind: KindSegment, Op: "in", Value: seg}), "Upgrade"),
		section("both", rule("both", ShowOnly, MatchAll,
			VisibilityCondition{Kind: KindTier, Op: "is_not", Value: "sunset"},
			VisibilityCondition{Kind: KindField, Op: "gt", Field: "seats", Value: "10"}), "Teams"),
		section("never", rule("never", ShowNobody, MatchAll), "Hidden for everyone"),
		section("always", VisibilityRule{}, "Everyone sees this"),
	)
	e := Read(src)
	var ids []string
	for _, b := range e.Blocks {
		ids = append(ids, b.ID)
	}
	if got := strings.Join(ids, ","); got != "missed,beta,trial,spanish,vip,both,never" {
		t.Fatalf("conditional blocks: %s (an everyone rule isn't one)", got)
	}
	if b, _ := e.Block("beta"); !b.Section || b.Name != "beta" || len(b.Path) != 1 || b.Path[0] != 2 {
		t.Fatalf("beta block: %+v", b)
	}

	hidden := func(f *Facts) string {
		d := Decide(e, f, nil)
		return d.Hidden.String()
	}
	sam := &Facts{Tier: "active", Language: "en-US", Lists: map[string]bool{}, Attributes: map[string]any{"plan": "team", "seats": 25}, Tags: map[string]bool{}}
	if got := hidden(sam); got != "beta,missed,never,spanish,trial" {
		t.Fatalf("Sam (active, English, team plan, 25 seats): hidden %s", got)
	}
	lucia := &Facts{Tier: "at_risk", Language: "es-MX", Lists: map[string]bool{beta: true}, Attributes: map[string]any{}, Tags: map[string]bool{"vip": true}}
	if got := hidden(lucia); got != "both,never,vip" {
		t.Fatalf("Lucía (at risk, Spanish, beta, no plan, vip): hidden %s", got)
	}
	inSeg := &Facts{Tier: "sunset", Segments: map[string]bool{seg: true}, Attributes: map[string]any{"plan": "", "seats": "many"}}
	if got := hidden(inSeg); got != "beta,both,missed,never,spanish,vip" {
		t.Fatalf("in the segment, sunset, empty plan, seats not a number: hidden %s", got)
	}

	// A condition on something deleted never matches: "only" hides, "except" shows.
	n := &Names{Lists: map[string]string{}, Segments: map[string]string{}, Fields: map[string]NamedField{"plan": {Type: "text"}, "seats": {Type: "number"}}}
	if d := Decide(e, lucia, n); !d.Hidden["beta"] {
		t.Fatalf("a deleted list never matches: %v", d.Hidden)
	}
	if d := Decide(e, inSeg, n); d.Hidden["vip"] {
		t.Fatalf("except a deleted segment shows the block: %v", d.Hidden)
	}
}

func TestVersionsSizesAndEmpty(t *testing.T) {
	beta := uuid.NewString()
	long := strings.Repeat("A long paragraph of the beta corner. ", 120)
	src := email(
		section("intro", VisibilityRule{}, "This week"),
		section("beta", rule("beta", ShowOnly, MatchAll, VisibilityCondition{Kind: KindList, Op: "includes", Value: beta}), long),
		section("trial", rule("trial", ShowOnly, MatchAll, VisibilityCondition{Kind: KindField, Op: "is_empty", Field: "plan"}), "Start a free trial"),
	)
	res := mjml.Compile(src)
	if len(res.Errors) > 0 {
		t.Fatalf("compile: %+v", res.Errors)
	}
	if !strings.Contains(res.HTML, mjml.MarkerIf+"beta-->") || !strings.Contains(res.HTML, mjml.MarkerFi+"trial-->") {
		t.Fatal("the compiled HTML marks the conditional blocks")
	}
	e := Read(src)
	people := []*Facts{
		{Lists: map[string]bool{beta: true}, Attributes: map[string]any{}},              // beta, trial
		{Lists: map[string]bool{beta: true}, Attributes: map[string]any{"plan": "pro"}}, // beta
		{Lists: map[string]bool{}, Attributes: map[string]any{}},                        // trial
		{Lists: map[string]bool{}, Attributes: map[string]any{}},                        // trial
		{Lists: map[string]bool{}, Attributes: map[string]any{"plan": "team"}},          // neither
	}
	tally := Count(e, people, nil)
	if tally.Total != 5 || tally.Shown["beta"] != 2 || tally.Shown["trial"] != 3 {
		t.Fatalf("tally: %+v", tally)
	}
	vs := Versions(e, tally, res.HTML)
	if len(vs) != 4 {
		t.Fatalf("4 of the 4 possible combinations occur: %+v", vs)
	}
	got := map[string]VisibilityVersion{}
	for _, v := range vs {
		got[v.Shows] = v
	}
	none, trial, both := got[""], got["trial"], got["beta, trial"]
	if none.Recipients != 1 || trial.Recipients != 2 || both.Recipients != 1 || got["beta"].Recipients != 1 {
		t.Fatalf("recipients per version: %+v", vs)
	}
	if none.SizeBytes >= trial.SizeBytes || trial.SizeBytes >= both.SizeBytes || !both.Largest || !none.Smallest {
		t.Fatalf("sizes: none %d, trial %d, both %d", none.SizeBytes, trial.SizeBytes, both.SizeBytes)
	}
	if vs[0].Shows != "" || vs[0].Number != 1 {
		t.Fatalf("version 1 shows none of the blocks: %+v", vs[0])
	}
	for _, v := range vs {
		if v.Empty || v.OverClip || !v.Checked {
			t.Fatalf("no version is empty or over the clipping size here: %+v", v)
		}
		cut := Apply(res.HTML, ParseHidden(strings.Join(v.Hidden, ",")))
		if strings.Contains(cut, "rc:if") || (!slices.Contains(v.Shown, "beta") && strings.Contains(cut, "beta corner")) {
			t.Fatalf("version %d keeps what it hides or its markers", v.Number)
		}
	}

	// A version is empty when it hides every content section; the header and
	// the footer don't count, and a rule on the footer is ignored.
	empty := email(section("content", rule("content", ShowOnly, MatchAll, VisibilityCondition{Kind: KindList, Op: "includes", Value: beta}), "All the content"))
	ee := Read(empty)
	if !ee.Empty(Hidden{"content": true}) || ee.Empty(Hidden{}) {
		t.Fatal("hiding the only content section leaves an empty version")
	}
	vs = Versions(ee, Count(ee, []*Facts{{Lists: map[string]bool{}}}, nil), "")
	if len(vs) != 1 || !vs[0].Empty {
		t.Fatalf("the empty version is flagged: %+v", vs)
	}
	footer := strings.Replace(email(), "<!-- rc:footer -->\n<mj-section>", `<!-- rc:footer -->
<mj-section rc-visible="{&quot;id&quot;:&quot;foot01&quot;,&quot;show&quot;:&quot;nobody&quot;}">`, 1)
	if fe := Read(footer); len(fe.Blocks) != 0 {
		t.Fatalf("the footer is always shown: %+v", fe.Blocks)
	}

	// Over Gmail's clipping size.
	huge := email(section("big", rule("big", ShowOnly, MatchAll, VisibilityCondition{Kind: KindTag, Op: "has", Value: "x"}), strings.Repeat("Lorem ipsum dolor sit amet. ", 4200)))
	hr := mjml.Compile(huge)
	he := Read(huge)
	vs = Versions(he, Count(he, []*Facts{{Tags: map[string]bool{"x": true}}, {Tags: map[string]bool{}}}, nil), hr.HTML)
	var over int
	for _, v := range vs {
		if v.OverClip {
			over++
			if v.SizeBytes <= contentcheck.ClipBytes || !slices.Contains(v.Shown, "big") {
				t.Fatalf("over clip: %+v", v)
			}
		}
	}
	if over != 1 {
		t.Fatalf("one version is over the clipping size: %+v", vs)
	}
}
