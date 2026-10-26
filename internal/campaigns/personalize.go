package campaigns

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/paidaccess"
	"github.com/hostrepute/reputecast/internal/store"
)

// The personalization a preview shows (CO-5): what each field prints for
// the contact previewed, which fields aren't contact fields at all ("did
// you mean first_name?"), and how many of the audience have no value for
// each field they use.

// fieldSet is what an email's Liquid may print: the workspace's tokens
// (contact fields, custom fields, other packages' tokens) and the
// variables filled in per recipient.
type fieldSet struct {
	known map[string]bool
	// countable are the fields a contact can lack a value for (first and
	// last name, custom fields): the audience is counted for them.
	countable map[string]bool
	// names are the known fields "did you mean" picks from.
	names []string
}

func (s *Service) fieldSet(ctx context.Context, wsID uuid.UUID, digestTemplate bool) (fieldSet, error) {
	tokens, err := s.tokens(ctx, wsID)
	if err != nil {
		return fieldSet{}, err
	}
	f := fieldSet{known: map[string]bool{}, countable: map[string]bool{}}
	add := func(k string) {
		if !f.known[k] {
			f.known[k] = true
			f.names = append(f.names, k)
		}
	}
	for _, t := range tokens {
		add(t.Key)
		if (t.Kind == "contact" && t.Key != "email") || t.Kind == "custom" {
			f.countable[t.Key] = true
		}
	}
	for k := range systemLinks {
		add(k)
	}
	if digestTemplate {
		// A recurring digest's template uses the feed tokens; each issue prints them.
		add("issue")
		add("feed")
	}
	// Earn's personal links (the Upgrade block's button) are filled in per reader.
	add(paidaccess.UpgradeURL)
	add(paidaccess.ManageURL)
	sort.Strings(f.names)
	return f, nil
}

// fields lists the field keys, for the content checks.
func (f fieldSet) fields() []string { return f.names }

// unknown reports whether a variable the email prints isn't a field: it
// prints nothing for everyone, fallback or not.
func (f fieldSet) unknown(key string) bool {
	root, _, _ := strings.Cut(key, ".")
	if f.known[key] || f.known[root] || root == "contact" || root == "forloop" {
		return false
	}
	// Poll answer links and survey links are filled per recipient (internal/polls).
	return !strings.HasPrefix(root, "rc_poll_") && !strings.HasPrefix(root, "rc_survey_")
}

// suggest is the known field a mistyped key most likely meant
// ("frist_name" → "first_name"), or "" when none is close.
func (f fieldSet) suggest(key string) string {
	best, bestD := "", 0
	limit := 2
	if len(key) <= 4 {
		limit = 1
	}
	for _, n := range f.names {
		d := editDistance(strings.ToLower(key), strings.ToLower(n))
		if d <= limit && (best == "" || d < bestD) {
			best, bestD = n, d
		}
	}
	return best
}

// editDistance counts the single-character edits (insert, delete, replace,
// or swap two neighbours) between a and b.
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev2 := make([]int, len(rb)+1)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				cur[j] = min(cur[j], prev2[j-2]+1)
			}
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[len(rb)]
}

// missingCounts counts, for each countable field, the audience contacts
// with no value for it. Only the campaign's lists are counted (minus the
// excluded lists and suppressions); segmentsLeft says the audience also
// has segments, which aren't. counted is the number of contacts counted;
// ok is false when there is nothing to count.
func (s *Service) missingCounts(ctx context.Context, wsID uuid.UUID, audience ComposerAudience, keys []string) (missing map[string]int64, counted int64, segmentsLeft, ok bool, err error) {
	lists, exclude := uuids(audience.Lists), uuids(audience.ExcludeLists)
	if len(lists) == 0 || len(keys) == 0 {
		return nil, 0, false, false, nil
	}
	rows, err := s.Q.ComposerCountMissingFields(ctx, store.ComposerCountMissingFieldsParams{WorkspaceID: wsID, Lists: lists, ExcludeLists: exclude, Keys: keys})
	if err != nil {
		return nil, 0, false, false, err
	}
	missing = make(map[string]int64, len(keys))
	for _, k := range keys {
		missing[k] = 0
	}
	for _, r := range rows {
		missing[r.Key], counted = r.Missing, r.Counted
	}
	return missing, counted, len(audience.Segments) > 0, true, nil
}

func uuids(in []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(in))
	for _, s := range in {
		if u, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
			out = append(out, u)
		}
	}
	return out
}

// usesOf lists the distinct variables printed in the given sources.
func usesOf(srcs ...string) []liquid.Use {
	return liquid.Uses(strings.Join(srcs, "\n"))
}

// checkVariants checks each A/B variant other than A: a subject test's
// subject (and preheader), a content test's content. A subject variant
// keeps what A's report doesn't have (the content is A's); a content
// variant keeps all its findings, since fixing A's body doesn't fix its
// own. A subject variant without a subject says so (it saves, and sends
// A's subject until one is written).
func (s *Service) checkVariants(ctx context.Context, checker *contentcheck.Checker, base contentcheck.Input, a contentcheck.ContentCheckReport, t *ComposerABTest, minify bool) ([]ComposerVariantChecks, error) {
	out := []ComposerVariantChecks{}
	if t == nil || t.Kind == "send_time" {
		return out, nil
	}
	key := func(is contentcheck.ContentCheckIssue) string {
		return fmt.Sprint(is.ID, "|", is.Title, "|", is.URL, "|", is.Path)
	}
	inA := map[string]bool{}
	for _, is := range a.Issues {
		inA[key(is)] = true
	}
	for _, v := range t.Variants {
		if v.Key == "A" {
			continue
		}
		in := base
		// A variant's check never requests links (Send Readiness does that for the content sent).
		in.ProbeLinks = false
		if t.Kind == "subject" {
			in.Subject = v.Subject
		}
		if v.Preheader != "" {
			in.Preheader = v.Preheader
		}
		if t.Kind == "content" && strings.TrimSpace(v.MJML) != "" {
			in.MJML, in.HTML = v.MJML, v.HTML
			if minify && in.HTML != "" {
				in.HTML = contentcheck.Minify(in.HTML)
			}
		}
		r, err := checker.Check(ctx, in)
		if err != nil {
			return nil, err
		}
		vc := ComposerVariantChecks{Key: v.Key, Issues: []contentcheck.ContentCheckIssue{}}
		for _, is := range r.Issues {
			if is.Severity != contentcheck.Block && is.Severity != contentcheck.Warn {
				continue
			}
			if t.Kind != "content" && inA[key(is)] {
				continue
			}
			if is.ID == "subject.missing" {
				args := i18n.Args{"variant": v.Key}
				is.Title, is.TitleText = mProblemVariantSubject.Text(i18n.Default, args), mProblemVariantSubject.Keep(args)
			}
			if is.Severity == contentcheck.Block {
				vc.Blocks++
			} else {
				vc.Warnings++
			}
			vc.Issues = append(vc.Issues, is)
		}
		out = append(out, vc)
	}
	return out, nil
}
