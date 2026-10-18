// Package gate decides what a reader of the publication website may read
// (design expansion phase 7, brief §3.3): a post's or a page's visibility
// (Public, Members, Paid for every tier or chosen tiers), the meter (N free
// posts per calendar month for visitors and for free members) and "Never
// free". It is pure: the website package reads the reader, the rules and
// the meter's state and records the reads; the pages of later phases call
// Page (pages aren't metered).
//
// Gating happens on the server. A reader without access gets the preview
// (the post down to its free preview marker, or its title, cover and
// excerpt) and the paywall card: the rest is never sent, so it can't be
// shown by removing a style, and feeds, the sitemap and search see what an
// anonymous visitor sees.
package gate

import (
	"slices"

	"github.com/google/uuid"
)

// Visibility levels.
const (
	Public  = "public"
	Members = "members"
	Paid    = "paid"
)

// ValidLevel reports whether level is a visibility.
func ValidLevel(level string) bool { return level == Public || level == Members || level == Paid }

// Visibility is who can read a post or a page.
type Visibility struct {
	Level string
	// Tiers are the paid tiers that may read a Paid post; empty means every
	// paid tier.
	Tiers []uuid.UUID
	// NeverFree leaves the post out of the meter ("Never free").
	NeverFree bool
}

// Tier is a paid tier. A higher rank includes the paid content of lower
// ranks ("Founding member includes everything in Insider").
type Tier struct {
	ID uuid.UUID
	// Name is the tier's name ("Weekly Digest Insider"); Label the quiet
	// label on posts ("Insider").
	Name  string
	Label string
	Rank  int
}

// Member is a signed-in reader.
type Member struct {
	ContactID uuid.UUID
	Email     string
	Name      string
	// Tiers are the paid tiers the member holds now (trialing, active, past
	// due with access, complimentary or a gift).
	Tiers []Tier
}

// Paid reports whether the member holds any paid tier.
func (m *Member) Paid() bool { return m != nil && len(m.Tiers) > 0 }

// Reader is who is reading: a signed-in member, or a visitor.
type Reader struct {
	Member *Member
	// Browser is a visitor's meter key: the hash of the first-party meter
	// cookie ("" until the browser has one).
	Browser string
}

// Key is the reader's meter key: "c:<contact>" for a member, "b:<hash>" for
// a visitor ("" for a visitor without a meter cookie yet).
func (r Reader) Key() string {
	if r.Member != nil {
		return "c:" + r.Member.ContactID.String()
	}
	if r.Browser == "" {
		return ""
	}
	return "b:" + r.Browser
}

// Rules are the website's access settings.
type Rules struct {
	// SigninOn is "Members can sign in". Off, members-only and paid posts
	// can't be read on the web by anyone.
	SigninOn bool
	// The meter: visitors who aren't signed in can read Visitors
	// members-only posts a month; free members can read Members paid posts
	// a month.
	VisitorsOn bool
	Visitors   int
	MembersOn  bool
	Members    int
	// Tiers are every paid tier, by id, for comparing ranks.
	Tiers map[uuid.UUID]Tier
}

// Meter rules.
const (
	RuleVisitor = "visitor"
	RuleMember  = "member"
)

// MeterState is a reader's month under one rule: how many posts were
// counted, and whether this post is one of them (reading it again is free).
type MeterState struct {
	Used int
	Read bool
}

// Access is what the reader gets.
type Access string

const (
	// Full is the whole post.
	Full Access = "full"
	// Preview is the post down to its marker (or its title, cover and
	// excerpt), then the paywall card.
	Preview Access = "preview"
)

// Reason says why.
type Reason string

const (
	ReasonPublic Reason = "public"
	// ReasonMember: a members-only post read by a member.
	ReasonMember Reason = "member"
	// ReasonTier: a paid post read by a member holding a tier that can read it.
	ReasonTier Reason = "tier"
	// ReasonMetered: a free read the meter counts now (the first time this month).
	ReasonMetered Reason = "metered"
	// ReasonReread: a post the meter already counted this month; free again.
	ReasonReread Reason = "reread"
	// ReasonSignin: a visitor on a members-only post without a meter read left
	// (or with the meter off or the post never free): sign in or sign up.
	ReasonSignin Reason = "signin"
	// ReasonSignupMeter: a visitor who used up the visitor meter: "Sign up
	// free to keep reading".
	ReasonSignupMeter Reason = "signup_meter"
	// ReasonUpgrade: a paid post for a visitor or a member without the tier.
	ReasonUpgrade Reason = "upgrade"
	// ReasonUpgradeMeter: a free member who used up the member meter.
	ReasonUpgradeMeter Reason = "upgrade_meter"
	// ReasonSigninOff: sign-in is turned off; members-only and paid posts
	// can't be read on the web.
	ReasonSigninOff Reason = "signin_off"
)

// Decision is what a reader gets of one post.
type Decision struct {
	Access Access
	Reason Reason
	// Rule, Used and Limit describe the meter when it applies: "This is 1
	// of 2 members-only posts you can read this month without signing in".
	Rule  string
	Used  int
	Limit int
	// Count asks the caller to record the read (the first read this month).
	Count bool
	// Tier is the member's tier that opened a paid post (the quiet label).
	Tier *Tier
	// FreeRow says the paywall card may offer a free sign-up: the meter
	// lets free members read paid posts, or the visitor meter is on.
	FreeRow bool
}

// Readable reports whether the reader gets the whole post.
func (d Decision) Readable() bool { return d.Access == Full }

// Decide decides a post for a reader. m is the reader's meter state under
// the rule that applies (MeterRule).
func Decide(v Visibility, r Reader, rules Rules, m MeterState) Decision {
	switch v.Level {
	case Members:
		if !rules.SigninOn {
			return Decision{Access: Preview, Reason: ReasonSigninOff}
		}
		if r.Member != nil {
			return Decision{Access: Full, Reason: ReasonMember}
		}
		if rules.VisitorsOn && rules.Visitors > 0 && !v.NeverFree {
			d := Decision{Rule: RuleVisitor, Limit: rules.Visitors}
			switch {
			case m.Read:
				d.Access, d.Reason, d.Used = Full, ReasonReread, m.Used
			case m.Used < rules.Visitors && r.Key() != "":
				d.Access, d.Reason, d.Used, d.Count = Full, ReasonMetered, m.Used+1, true
			default:
				d.Access, d.Reason, d.Used = Preview, ReasonSignupMeter, min(m.Used, rules.Visitors)
			}
			return d
		}
		return Decision{Access: Preview, Reason: ReasonSignin}
	case Paid:
		if !rules.SigninOn {
			return Decision{Access: Preview, Reason: ReasonSigninOff}
		}
		if r.Member != nil {
			if t := Qualifies(v, r.Member.Tiers, rules.Tiers); t != nil {
				return Decision{Access: Full, Reason: ReasonTier, Tier: t}
			}
			if rules.MembersOn && rules.Members > 0 && !v.NeverFree {
				d := Decision{Rule: RuleMember, Limit: rules.Members}
				switch {
				case m.Read:
					d.Access, d.Reason, d.Used = Full, ReasonReread, m.Used
				case m.Used < rules.Members:
					d.Access, d.Reason, d.Used, d.Count = Full, ReasonMetered, m.Used+1, true
				default:
					d.Access, d.Reason, d.Used = Preview, ReasonUpgradeMeter, min(m.Used, rules.Members)
				}
				return d
			}
			return Decision{Access: Preview, Reason: ReasonUpgrade}
		}
		// A visitor: the tiers, sign in, and a free sign-up where the
		// member meter would give them a free paid post.
		return Decision{Access: Preview, Reason: ReasonUpgrade, FreeRow: rules.MembersOn && rules.Members > 0 && !v.NeverFree}
	}
	return Decision{Access: Full, Reason: ReasonPublic}
}

// MeterRule is the meter rule that applies to a reader on a post ("" for
// none: a public post, a member on a members-only post, a paid member).
func MeterRule(v Visibility, r Reader, rules Rules) string {
	if v.NeverFree || !rules.SigninOn {
		return ""
	}
	switch v.Level {
	case Members:
		if r.Member == nil && rules.VisitorsOn {
			return RuleVisitor
		}
	case Paid:
		if r.Member != nil && rules.MembersOn && Qualifies(v, r.Member.Tiers, rules.Tiers) == nil {
			return RuleMember
		}
	}
	return ""
}

// Page decides a page (pages aren't metered): Public for everyone, Members
// for any member, Paid for a member holding a tier that can read it.
func Page(v Visibility, r Reader, rules Rules) Decision {
	switch v.Level {
	case Members:
		if !rules.SigninOn {
			return Decision{Access: Preview, Reason: ReasonSigninOff}
		}
		if r.Member != nil {
			return Decision{Access: Full, Reason: ReasonMember}
		}
		return Decision{Access: Preview, Reason: ReasonSignin}
	case Paid:
		if !rules.SigninOn {
			return Decision{Access: Preview, Reason: ReasonSigninOff}
		}
		if r.Member != nil {
			if t := Qualifies(v, r.Member.Tiers, rules.Tiers); t != nil {
				return Decision{Access: Full, Reason: ReasonTier, Tier: t}
			}
		}
		return Decision{Access: Preview, Reason: ReasonUpgrade}
	}
	return Decision{Access: Full, Reason: ReasonPublic}
}

// Qualifies returns the held tier that can read a Paid post, or nil. Any
// tier reads a post for every tier; for chosen tiers, a chosen one or a
// tier ranked above one of them (a higher tier includes a lower tier's
// paid content).
func Qualifies(v Visibility, held []Tier, all map[uuid.UUID]Tier) *Tier {
	if len(held) == 0 {
		return nil
	}
	best := func(ok func(Tier) bool) *Tier {
		var pick *Tier
		for i := range held {
			t := held[i]
			if !ok(t) {
				continue
			}
			if pick == nil || t.Rank > pick.Rank {
				tt := t
				pick = &tt
			}
		}
		return pick
	}
	if len(v.Tiers) == 0 {
		return best(func(Tier) bool { return true })
	}
	return best(func(t Tier) bool {
		if slices.Contains(v.Tiers, t.ID) {
			return true
		}
		for _, id := range v.Tiers {
			if c, ok := all[id]; ok && c.Rank < t.Rank {
				return true
			}
		}
		return false
	})
}

// Readers are the tiers that can read a Paid post, from every tier: the
// paywall card offers only these.
func Readers(v Visibility, all []Tier) []Tier {
	if len(v.Tiers) == 0 {
		return slices.Clone(all)
	}
	byID := map[uuid.UUID]Tier{}
	for _, t := range all {
		byID[t.ID] = t
	}
	out := []Tier{}
	for _, t := range all {
		if Qualifies(v, []Tier{t}, byID) != nil {
			out = append(out, t)
		}
	}
	return out
}
