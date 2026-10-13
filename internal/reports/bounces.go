package reports

import (
	"sort"
	"strings"

	"github.com/hostrepute/reputecast/internal/providers/bounce"
	provstate "github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// bounceType is one row of "Bounces by type": a group of the shared bounce
// taxonomy (internal/providers/bounce) as people read it.
type bounceType struct {
	Key     string
	Label   string
	Tone    string
	Classes []string
	// Suppress says what happens to the addresses: auto (suppressed when
	// they bounce), retry (soft, retried next send) or manual (a person
	// decides, after a review for unclassified ones).
	Suppress string
}

// bounceTypes are the report's bounce types, in display order.
var bounceTypes = []bounceType{
	{Key: "unknown_user", Label: "Hard · unknown user", Tone: "bad", Classes: []string{bounce.MailboxUnknown}, Suppress: "auto"},
	{Key: "domain_invalid", Label: "Hard · domain not found", Tone: "bad", Classes: []string{bounce.DomainInvalid}, Suppress: "auto"},
	{Key: "mailbox_full", Label: "Soft · mailbox full", Tone: "warn", Classes: []string{bounce.MailboxFull}, Suppress: "retry"},
	{Key: "deferral_expired", Label: "Soft · deferral expired", Tone: "warn", Classes: []string{bounce.Temporary}, Suppress: "retry"},
	{Key: "blocked", Label: "Blocked · policy or reputation", Tone: "bad", Classes: []string{bounce.PolicyBlock, bounce.SpamBlock, bounce.Reputation}, Suppress: "manual"},
	{Key: "other", Label: "Other", Tone: "neutral", Classes: []string{bounce.Other, ""}, Suppress: "review"},
}

func bounceTypeOf(class string) bounceType {
	for _, t := range bounceTypes {
		for _, c := range t.Classes {
			if c == class {
				return t
			}
		}
	}
	return bounceTypes[len(bounceTypes)-1]
}

// BounceTypeByKey returns a bounce type by its key.
func bounceTypeByKey(key string) (bounceType, bool) {
	for _, t := range bounceTypes {
		if t.Key == key {
			return t, true
		}
	}
	return bounceType{}, false
}

// bounceTable groups the campaign's bounce events into the taxonomy, with
// each provider's original codes, most frequent first.
func bounceTable(rows []store.ReportBounceCodesRow) ReportBounceTable {
	type acc struct {
		row        ReportBounceRow
		recipients int64
		suppressed int64
		codes      map[string]*ReportCodeCount
	}
	byKey := map[string]*acc{}
	out := ReportBounceTable{Rows: []ReportBounceRow{}}
	// Codes are listed provider by provider, the provider with the most
	// bounces first, so every row reads in the same order.
	perProvider := map[string]int64{}
	for _, r := range rows {
		perProvider[provstate.Short(r.ProviderName, r.Adapter)] += r.N
	}
	for _, r := range rows {
		t := bounceTypeOf(r.BounceClass)
		a := byKey[t.Key]
		if a == nil {
			a = &acc{row: ReportBounceRow{Type: t.Key, Label: t.Label, Tone: t.Tone, Codes: []ReportCodeCount{}}, codes: map[string]*ReportCodeCount{}}
			byKey[t.Key] = a
		}
		a.row.Count += r.N
		a.recipients += r.Recipients
		a.suppressed += r.Suppressed
		short := provstate.Short(r.ProviderName, r.Adapter)
		k := short + "\x00" + r.OriginalType + "\x00" + r.OriginalCode
		c := a.codes[k]
		if c == nil {
			c = &ReportCodeCount{Provider: short, Type: strings.TrimSpace(r.OriginalType), Code: strings.TrimSpace(r.OriginalCode)}
			a.codes[k] = c
		}
		c.Count += r.N
		out.Total += r.N
	}
	for _, t := range bounceTypes {
		a := byKey[t.Key]
		if a == nil {
			continue
		}
		for _, c := range a.codes {
			a.row.Codes = append(a.row.Codes, *c)
		}
		sort.SliceStable(a.row.Codes, func(i, j int) bool {
			ci, cj := a.row.Codes[i], a.row.Codes[j]
			if ci.Provider != cj.Provider {
				if perProvider[ci.Provider] != perProvider[cj.Provider] {
					return perProvider[ci.Provider] > perProvider[cj.Provider]
				}
				return ci.Provider < cj.Provider
			}
			if ci.Count != cj.Count {
				return ci.Count > cj.Count
			}
			return ci.CodeText() < cj.CodeText()
		})
		a.row.Unsuppressed = max(a.recipients-a.suppressed, 0)
		a.row.Suppression = suppressionState(t, a.recipients, a.row.Unsuppressed)
		out.Rows = append(out.Rows, a.row)
	}
	return out
}

// suppressionState says what the Suppression column shows for a type.
func suppressionState(t bounceType, recipients, unsuppressed int64) string {
	switch t.Suppress {
	case "retry":
		return "retry"
	case "auto":
		if unsuppressed == 0 {
			return "auto"
		}
		return "add"
	case "manual":
		switch {
		case unsuppressed > 0:
			return "add"
		case recipients > 0:
			return "suppressed"
		}
		return "review"
	}
	if recipients > 0 && unsuppressed == 0 {
		return "suppressed"
	}
	return "review"
}

// CodeText is "SES Permanent/General 550 5.1.1".
func (c ReportCodeCount) CodeText() string {
	parts := []string{c.Provider}
	if c.Type != "" {
		parts = append(parts, c.Type)
	}
	if c.Code != "" {
		parts = append(parts, c.Code)
	}
	return strings.Join(parts, " ")
}
