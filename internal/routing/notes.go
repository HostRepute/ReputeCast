package routing

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// A message's route note (messages.route_note: why a campaign recipient
// went to its provider) and why a transactional message failed
// (messages.error, an attempt's error) are stored in English by the sending
// engine, from the messages below. NoteIn and DeliveryErrorIn write them in
// a reader's language for the screens that show them: a contact's activity
// timeline and the transactional log. What they don't recognize, such as
// a provider's own error, reads as stored.

// RouteNote is a route note in English, as the sending engine stores it.
// key is campaignRoute, byRule, split, splitWarmup, instead or failover.
func RouteNote(key string, args i18n.Args) string { return text(i18n.Default, "note."+key, args) }

// DeliveryError is why a message couldn't be delivered through its route,
// in English as stored. key is down, downNoOther, providersRemoved,
// contentGone, unreadable or allFailed.
func DeliveryError(key string, args i18n.Args) string { return text(i18n.Default, "error."+key, args) }

var (
	reNoteRoute    = regexp.MustCompile(`^Campaign route: (.+) only$`)
	reNoteInstead  = regexp.MustCompile(`^(.+) instead of (.+): (over today's warm-up cap|warm-up paused|not sending to (.+) contacts on warm-up day (\d+))$`)
	reNoteFailover = regexp.MustCompile(`^(.+) was down; Broadcasts failed over to (.+)$`)
	reNoteSplit    = regexp.MustCompile(`^(.+) share of the (\d+(?:/\d+)*) split(?:, warm-up day (\d+))?$`)
	reNoteRule     = regexp.MustCompile(`^(.+) by rule: (.+)$`)

	reErrDownNoOther = regexp.MustCompile(`^(.+) was down and no other provider in the route could take the message\.$`)
	reErrDown        = regexp.MustCompile(`^(.+) is down$`)
)

// NoteIn writes a stored route note in a reader's language.
func NoteIn(l i18n.Lang, english string) string {
	if english == "" || l.Locale == "" || l.Locale == i18n.English {
		return english
	}
	if m := reNoteRoute.FindStringSubmatch(english); m != nil {
		return text(l, "note.campaignRoute", i18n.Args{"provider": m[1]})
	}
	if m := reNoteInstead.FindStringSubmatch(english); m != nil {
		args := i18n.Args{"alt": m[1], "provider": m[2], "why": "tier", "tier": strings.ReplaceAll(m[4], " ", "_"), "day": atoi(m[5])}
		switch m[3] {
		case "over today's warm-up cap":
			args["why"] = "cap"
		case "warm-up paused":
			args["why"] = "paused"
		}
		return text(l, "note.instead", args)
	}
	if m := reNoteFailover.FindStringSubmatch(english); m != nil {
		return text(l, "note.failover", i18n.Args{"provider": m[1], "fallback": m[2]})
	}
	if m := reNoteSplit.FindStringSubmatch(english); m != nil {
		if m[3] != "" {
			return text(l, "note.splitWarmup", i18n.Args{"provider": m[1], "split": m[2], "day": atoi(m[3])})
		}
		return text(l, "note.split", i18n.Args{"provider": m[1], "split": m[2]})
	}
	if m := reNoteRule.FindStringSubmatch(english); m != nil {
		return text(l, "note.byRule", i18n.Args{"provider": m[1], "rule": m[2]})
	}
	return english
}

// DeliveryErrorIn writes why a message couldn't be delivered, stored in
// English, in a reader's language.
func DeliveryErrorIn(l i18n.Lang, english string) string {
	if english == "" || l.Locale == "" || l.Locale == i18n.English {
		return english
	}
	for _, key := range []string{"providersRemoved", "contentGone"} {
		if english == DeliveryError(key, nil) {
			return text(l, "error."+key, nil)
		}
	}
	for _, key := range []string{"unreadable", "allFailed"} {
		prefix, _, _ := strings.Cut(DeliveryError(key, i18n.Args{"error": "\x00"}), "\x00")
		if rest, ok := strings.CutPrefix(english, prefix); ok {
			return text(l, "error."+key, i18n.Args{"error": rest})
		}
	}
	if m := reErrDownNoOther.FindStringSubmatch(english); m != nil {
		return text(l, "error.downNoOther", i18n.Args{"provider": m[1]})
	}
	if m := reErrDown.FindStringSubmatch(english); m != nil {
		return text(l, "error.down", i18n.Args{"provider": m[1]})
	}
	return english
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
