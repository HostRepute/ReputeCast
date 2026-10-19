package gate

import (
	"testing"

	"github.com/google/uuid"
)

var (
	insider  = Tier{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Name: "Weekly Digest Insider", Label: "Insider", Rank: 1}
	founding = Tier{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), Name: "Founding member", Label: "Founding member", Rank: 2}
	all      = map[uuid.UUID]Tier{insider.ID: insider, founding.ID: founding}
)

func rules() Rules {
	return Rules{SigninOn: true, VisitorsOn: true, Visitors: 2, MembersOn: true, Members: 1, Tiers: all}
}

func visitor() Reader { return Reader{Browser: "b1"} }

func member(tiers ...Tier) Reader {
	return Reader{Member: &Member{ContactID: uuid.New(), Email: "sam.rivera@outlook.com", Tiers: tiers}}
}

func TestPublicPostsAreReadByEveryone(t *testing.T) {
	d := Decide(Visibility{Level: Public}, Reader{}, rules(), MeterState{})
	if !d.Readable() || d.Reason != ReasonPublic || d.Count {
		t.Fatalf("public: %+v", d)
	}
}

func TestMembersOnlyPosts(t *testing.T) {
	v := Visibility{Level: Members}
	if d := Decide(v, member(), rules(), MeterState{}); !d.Readable() || d.Reason != ReasonMember {
		t.Fatalf("member: %+v", d)
	}
	// A visitor's first members-only post this month opens and counts.
	d := Decide(v, visitor(), rules(), MeterState{Used: 0})
	if !d.Readable() || d.Reason != ReasonMetered || !d.Count || d.Used != 1 || d.Limit != 2 || d.Rule != RuleVisitor {
		t.Fatalf("first metered read: %+v", d)
	}
	// Reading it again is free and isn't counted twice.
	d = Decide(v, visitor(), rules(), MeterState{Used: 2, Read: true})
	if !d.Readable() || d.Reason != ReasonReread || d.Count {
		t.Fatalf("reread: %+v", d)
	}
	// After two posts: the preview and "Sign up free to keep reading".
	d = Decide(v, visitor(), rules(), MeterState{Used: 2})
	if d.Readable() || d.Reason != ReasonSignupMeter || d.Used != 2 {
		t.Fatalf("meter used up: %+v", d)
	}
	// Never free leaves the post out of the meter.
	d = Decide(Visibility{Level: Members, NeverFree: true}, visitor(), rules(), MeterState{})
	if d.Readable() || d.Reason != ReasonSignin {
		t.Fatalf("never free: %+v", d)
	}
	// With the visitor meter off, visitors sign in.
	r := rules()
	r.VisitorsOn = false
	if d := Decide(v, visitor(), r, MeterState{}); d.Readable() || d.Reason != ReasonSignin {
		t.Fatalf("meter off: %+v", d)
	}
	// A visitor without a meter cookie can't be counted: no free read.
	if d := Decide(v, Reader{}, rules(), MeterState{}); d.Readable() {
		t.Fatalf("no meter key: %+v", d)
	}
}

func TestPaidPosts(t *testing.T) {
	every := Visibility{Level: Paid}
	// A paid member reads the whole post, with their tier's label.
	d := Decide(every, member(insider), rules(), MeterState{})
	if !d.Readable() || d.Reason != ReasonTier || d.Tier == nil || d.Tier.Label != "Insider" {
		t.Fatalf("paid member: %+v", d)
	}
	// A free member's monthly paid post.
	d = Decide(every, member(), rules(), MeterState{})
	if !d.Readable() || d.Reason != ReasonMetered || !d.Count || d.Rule != RuleMember || d.Limit != 1 {
		t.Fatalf("free member metered: %+v", d)
	}
	// Used up: the preview and the tiers.
	d = Decide(every, member(), rules(), MeterState{Used: 1})
	if d.Readable() || d.Reason != ReasonUpgradeMeter {
		t.Fatalf("free member used up: %+v", d)
	}
	// A visitor never reads a paid post; the free row shows while the
	// member meter is on.
	d = Decide(every, visitor(), rules(), MeterState{})
	if d.Readable() || d.Reason != ReasonUpgrade || !d.FreeRow {
		t.Fatalf("visitor on paid: %+v", d)
	}
	// Never free: no meter for members, no free row for visitors.
	nf := Visibility{Level: Paid, NeverFree: true, Tiers: []uuid.UUID{founding.ID}}
	if d := Decide(nf, member(), rules(), MeterState{}); d.Readable() || d.Reason != ReasonUpgrade {
		t.Fatalf("never free member: %+v", d)
	}
	if d := Decide(nf, visitor(), rules(), MeterState{}); d.FreeRow {
		t.Fatalf("never free visitor gets a free row: %+v", d)
	}
}

func TestChosenTiersAndRanks(t *testing.T) {
	foundingOnly := Visibility{Level: Paid, Tiers: []uuid.UUID{founding.ID}}
	if d := Decide(foundingOnly, member(insider), rules(), MeterState{Used: 1}); d.Readable() {
		t.Fatalf("an Insider reads a Founding member post: %+v", d)
	}
	if d := Decide(foundingOnly, member(founding), rules(), MeterState{}); !d.Readable() {
		t.Fatalf("a Founding member can't read their post: %+v", d)
	}
	// A higher tier includes a lower tier's paid content.
	insiderOnly := Visibility{Level: Paid, Tiers: []uuid.UUID{insider.ID}}
	if d := Decide(insiderOnly, member(founding), rules(), MeterState{}); !d.Readable() || d.Tier.ID != founding.ID {
		t.Fatalf("a Founding member can't read an Insider post: %+v", d)
	}
	got := Readers(insiderOnly, []Tier{insider, founding})
	if len(got) != 2 {
		t.Fatalf("readers of an Insider post: %+v", got)
	}
	got = Readers(foundingOnly, []Tier{insider, founding})
	if len(got) != 1 || got[0].ID != founding.ID {
		t.Fatalf("readers of a Founding member post: %+v", got)
	}
}

func TestSigninOffClosesGatedPosts(t *testing.T) {
	r := rules()
	r.SigninOn = false
	for _, v := range []Visibility{{Level: Members}, {Level: Paid}} {
		if d := Decide(v, member(insider), r, MeterState{}); d.Readable() || d.Reason != ReasonSigninOff {
			t.Fatalf("%s with sign-in off: %+v", v.Level, d)
		}
	}
	if d := Decide(Visibility{Level: Public}, Reader{}, r, MeterState{}); !d.Readable() {
		t.Fatal("public post closed with sign-in off")
	}
}

func TestMeterRule(t *testing.T) {
	if got := MeterRule(Visibility{Level: Members}, visitor(), rules()); got != RuleVisitor {
		t.Fatalf("visitor on members-only: %q", got)
	}
	if got := MeterRule(Visibility{Level: Members}, member(), rules()); got != "" {
		t.Fatalf("member on members-only: %q", got)
	}
	if got := MeterRule(Visibility{Level: Paid}, member(), rules()); got != RuleMember {
		t.Fatalf("free member on paid: %q", got)
	}
	if got := MeterRule(Visibility{Level: Paid}, member(insider), rules()); got != "" {
		t.Fatalf("paid member on paid: %q", got)
	}
	if got := MeterRule(Visibility{Level: Paid, NeverFree: true}, member(), rules()); got != "" {
		t.Fatalf("never free: %q", got)
	}
}

func TestPagesAreNotMetered(t *testing.T) {
	if d := Page(Visibility{Level: Members}, visitor(), rules()); d.Readable() {
		t.Fatalf("visitor on a members page: %+v", d)
	}
	if d := Page(Visibility{Level: Paid}, member(), rules()); d.Readable() {
		t.Fatalf("free member on a paid page: %+v", d)
	}
	if d := Page(Visibility{Level: Paid}, member(insider), rules()); !d.Readable() {
		t.Fatalf("paid member on a paid page: %+v", d)
	}
}
