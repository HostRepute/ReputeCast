package liquid

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// Blocks that fill in per reader at send time (Grow's Referral, Reward and
// Recommendations blocks) are stored as a marker comment with their
// settings, <!-- rc:referral {"layout":"progress",…} -->, inside an mj-raw.
// An expander registered for the block's name turns the marker into the
// HTML and Liquid that renders it from per-recipient variables (see
// RecipientData), so every place that renders an email for a reader
// renders the block: sends, previews and test sends, view online, link
// redirects. A marker nothing expands is left as it is (a comment).

var (
	blocksMu sync.RWMutex
	blocks   = map[string]func(settings []byte) string{}
	markerRe = regexp.MustCompile(`<!--\s*rc:([a-z][a-z-]*)\s*(\{[\s\S]*?\})?\s*-->`)
)

// RegisterBlock adds a block expander. Expanders are pure: the output
// depends on the settings only.
func RegisterBlock(name string, expand func(settings []byte) string) {
	blocksMu.Lock()
	defer blocksMu.Unlock()
	blocks[name] = expand
}

// HasBlocks reports whether src has a marker of a registered block.
func HasBlocks(src string) bool {
	if !strings.Contains(src, "rc:") {
		return false
	}
	blocksMu.RLock()
	defer blocksMu.RUnlock()
	for _, m := range markerRe.FindAllStringSubmatch(src, -1) {
		if _, ok := blocks[m[1]]; ok {
			return true
		}
	}
	return false
}

// ExpandBlocks replaces the markers of registered blocks with what renders
// them. It is idempotent: the output has no markers left to expand.
func ExpandBlocks(src string) string {
	if !strings.Contains(src, "<!--") || !strings.Contains(src, "rc:") {
		return src
	}
	blocksMu.RLock()
	defer blocksMu.RUnlock()
	return markerRe.ReplaceAllStringFunc(src, func(m string) string {
		sub := markerRe.FindStringSubmatch(m)
		fn, ok := blocks[sub[1]]
		if !ok {
			return m
		}
		return fn([]byte(sub[2]))
	})
}

// Recipient is one reader of a page of messages.
type Recipient struct {
	ContactID uuid.UUID
	MessageID uuid.UUID
	// EmailID is the automation email being sent (milestone emails read
	// their reward from it); uuid.Nil for campaigns.
	EmailID uuid.UUID
}

// RecipientData adds variables that depend on the reader, for a page of
// readers at once: Grow's referral.* (the reader's link and progress),
// reward.* (milestone emails) and recommendations.* (the recommendations
// page link). Roots are the variable names it provides; Uses says which
// a template needs.
type RecipientData interface {
	Roots() []string
	Vars(ctx context.Context, workspaceID uuid.UUID, recipients []Recipient, roots []string) (map[uuid.UUID]map[string]any, error)
}

var (
	liquidSpanRe = regexp.MustCompile(`\{\{[\s\S]*?\}\}|\{%[\s\S]*?%\}`)
	rootRes      sync.Map // root → *regexp.Regexp
)

func rootRe(root string) *regexp.Regexp {
	if v, ok := rootRes.Load(root); ok {
		return v.(*regexp.Regexp)
	}
	re := regexp.MustCompile(`(^|[^.\w])` + regexp.QuoteMeta(root) + `\b`)
	rootRes.Store(root, re)
	return re
}

// UsedRoots are the roots of d a template's Liquid uses (a marker of a
// block counts as using its roots). Text outside Liquid never does.
func UsedRoots(d RecipientData, src string) []string {
	if d == nil || src == "" {
		return nil
	}
	var spans []string
	if strings.Contains(src, "{{") || strings.Contains(src, "{%") {
		spans = liquidSpanRe.FindAllString(src, -1)
	}
	var out []string
	for _, r := range d.Roots() {
		used := strings.Contains(src, "rc:"+r)
		re := rootRe(r)
		for _, s := range spans {
			if used {
				break
			}
			used = re.MatchString(s)
		}
		if used {
			out = append(out, r)
		}
	}
	return out
}

// Merge adds a reader's per-recipient variables to vars.
func Merge(vars map[string]any, extra map[string]any) map[string]any {
	for k, v := range extra {
		vars[k] = v
	}
	return vars
}

// FillBatch is the per-recipient variables of every provider whose roots
// src uses, for a page of readers, by contact. A provider that fails adds
// nothing (its blocks hide).
func FillBatch(ctx context.Context, providers []RecipientData, workspaceID uuid.UUID, recipients []Recipient, src string) map[uuid.UUID]map[string]any {
	out := map[uuid.UUID]map[string]any{}
	if len(recipients) == 0 {
		return out
	}
	ctx = context.WithValue(ctx, sourceKey{}, src)
	for _, d := range providers {
		roots := UsedRoots(d, src)
		if len(roots) == 0 {
			continue
		}
		got, err := d.Vars(ctx, workspaceID, recipients, roots)
		if err != nil {
			continue
		}
		for id, v := range got {
			if out[id] == nil {
				out[id] = map[string]any{}
			}
			Merge(out[id], v)
		}
	}
	return out
}

// Fill adds one reader's per-recipient variables (FillBatch) to vars.
func Fill(ctx context.Context, providers []RecipientData, workspaceID uuid.UUID, r Recipient, src string, vars map[string]any) map[string]any {
	if len(providers) == 0 {
		return vars
	}
	return Merge(vars, FillBatch(ctx, providers, workspaceID, []Recipient{r}, src)[r.ContactID])
}

type sourceKey struct{}

// SourceFrom is the template a RecipientData provider is filling in
// (FillBatch passes it along), for providers whose variables depend on
// how blocks were set up (the Recommendations block's picks).
func SourceFrom(ctx context.Context) string {
	s, _ := ctx.Value(sourceKey{}).(string)
	return s
}
