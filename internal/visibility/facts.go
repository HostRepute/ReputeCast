package visibility

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// SegmentResolver lists a segment's contacts (the audience domain's
// segments service).
type SegmentResolver interface {
	ContactIDs(ctx context.Context, workspaceID, segmentID uuid.UUID) ([]uuid.UUID, error)
}

// Segments is the members of the segments rules refer to, read once per
// count, send or preview.
type Segments map[string]map[uuid.UUID]bool

// LoadSegments resolves the segments an email's rules refer to. A segment
// that no longer exists has no members (its condition is broken).
func LoadSegments(ctx context.Context, r SegmentResolver, wsID uuid.UUID, ids []string, n *Names) (Segments, error) {
	out := Segments{}
	for _, id := range ids {
		out[id] = map[uuid.UUID]bool{}
		if r == nil || (n != nil && !n.Exists(Ref{Kind: "segment", ID: id})) {
			continue
		}
		sid, err := uuid.Parse(id)
		if err != nil {
			continue
		}
		members, err := r.ContactIDs(ctx, wsID, sid)
		if err != nil {
			return nil, fmt.Errorf("segment %s: %w", id, err)
		}
		for _, m := range members {
			out[id][m] = true
		}
	}
	return out, nil
}

// newFacts builds a contact's facts from a row of the bulk reads.
func newFacts(id uuid.UUID, email, first, last, tier, lang string, tags []string, attrs json.RawMessage, lists []uuid.UUID, segs Segments) *Facts {
	f := &Facts{ID: id, Email: email, FirstName: first, LastName: last, Tier: tier, Language: lang,
		Lists: make(map[string]bool, len(lists)), Segments: map[string]bool{}, Tags: make(map[string]bool, len(tags)), Attributes: map[string]any{}, Extra: map[string]any{}}
	for _, l := range lists {
		f.Lists[l.String()] = true
	}
	for _, t := range tags {
		f.Tags[t] = true
	}
	if len(attrs) > 0 {
		_ = json.Unmarshal(attrs, &f.Attributes)
	}
	for sid, members := range segs {
		if members[id] {
			f.Segments[sid] = true
		}
	}
	return f
}

// LoadFacts reads what conditions need about some contacts, including what
// registered kinds load for themselves.
func LoadFacts(ctx context.Context, q *store.Queries, wsID uuid.UUID, ids []uuid.UUID, segs Segments, e Email) (map[uuid.UUID]*Facts, error) {
	out := make(map[uuid.UUID]*Facts, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.VisibilityContactFacts(ctx, store.VisibilityContactFactsParams{WorkspaceID: wsID, Ids: ids})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = newFacts(r.ID, r.Email, r.FirstName, r.LastName, r.Tier, r.Language, r.Tags, r.Attributes, r.ListIds, segs)
	}
	return out, loadExtra(ctx, q, wsID, out, e)
}

// loadExtra lets registered kinds with a loader read their facts.
func loadExtra(ctx context.Context, q *store.Queries, wsID uuid.UUID, facts map[uuid.UUID]*Facts, e Email) error {
	byKind := map[string][]VisibilityCondition{}
	for _, b := range e.Blocks {
		for _, c := range b.Rule.Conditions {
			byKind[c.Kind] = append(byKind[c.Kind], c)
		}
	}
	for kind, conds := range byKind {
		k, ok := Lookup(kind)
		if !ok || k.Load == nil {
			continue
		}
		if err := k.Load(ctx, q, wsID, conds, facts); err != nil {
			return fmt.Errorf("visibility %s facts: %w", kind, err)
		}
	}
	return nil
}

// Decision is what a contact gets: the blocks left out, and for each block
// whether each condition matched (for "Show why").
type Decision struct {
	Hidden  Hidden
	Matched map[string][]bool
}

// Decide evaluates every conditional block for a contact. A condition of an
// unknown kind (a phase's code not loaded) or referring to something
// deleted never matches.
func Decide(e Email, f *Facts, n *Names) Decision {
	d := Decision{Hidden: Hidden{}, Matched: map[string][]bool{}}
	for _, b := range e.Blocks {
		matched := make([]bool, len(b.Rule.Conditions))
		for i, c := range b.Rule.Conditions {
			k, ok := Lookup(c.Kind)
			if !ok || k.Match == nil {
				continue
			}
			if n != nil && k.Ref != nil {
				if r := k.Ref(c); r.Kind != "" && r.Kind != "tag" && !n.Exists(r) {
					continue
				}
			}
			matched[i] = k.Match(c, f)
		}
		d.Matched[b.ID] = matched
		// A block inside a hidden one goes with it, so contacts who get the
		// same email count as one version.
		if !b.Rule.Shows(matched) || (b.Parent != "" && d.Hidden[b.Parent]) {
			d.Hidden[b.ID] = true
		}
	}
	return d
}
