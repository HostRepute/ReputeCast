package visibility

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/paidaccess"
	"github.com/hostrepute/reputecast/internal/store"
)

// MaxChecked is how many versions Send Readiness renders and checks one by
// one; beyond it, the most common ones plus the largest and the smallest.
const MaxChecked = 50

// Audience is campaigns.audience as the composer stores it.
type Audience struct {
	Lists           []string `json:"lists"`
	Segments        []string `json:"segments"`
	ExcludeLists    []string `json:"excludeLists"`
	ExcludeLists2   []string `json:"exclude_lists"`
	ExcludeSegments []string `json:"excludeSegments"`
	ExcludeSegs2    []string `json:"exclude_segments"`
	ExcludeTiers    []string `json:"exclude_tiers"`
	ExcludeTiers2   []string `json:"excludeTiers"`
	IncludeSunset   bool     `json:"include_sunset"`
	IncludeSunset2  bool     `json:"includeSunset"`
	// Paid is who on the lists gets a paid-only campaign (Earn).
	Paid paidaccess.Audience `json:"-"`
}

// ParseAudience reads campaigns.audience (both key styles).
func ParseAudience(raw json.RawMessage) Audience {
	var a Audience
	_ = json.Unmarshal(raw, &a)
	a.Paid = paidaccess.Parse(raw)
	a.ExcludeLists = append(a.ExcludeLists, a.ExcludeLists2...)
	a.ExcludeSegments = append(a.ExcludeSegments, a.ExcludeSegs2...)
	a.ExcludeTiers = append(a.ExcludeTiers, a.ExcludeTiers2...)
	a.IncludeSunset = a.IncludeSunset || a.IncludeSunset2
	return a
}

// Empty reports whether the audience selects nobody.
func (a Audience) Empty() bool { return len(a.Lists) == 0 && len(a.Segments) == 0 }

// CountKeys are the hashes a count is kept under (visibility_counts): the
// email's rules and the campaign's audience. The demo seeds the sample's
// count with them.
func CountKeys(src string, audience json.RawMessage) (rules, aud string) {
	return Read(src).Hash(), ParseAudience(audience).hash()
}

func (a Audience) hash() string {
	parts := []string{strings.Join(sorted(a.Lists), ","), strings.Join(sorted(a.Segments), ","), strings.Join(sorted(a.ExcludeLists), ","),
		strings.Join(sorted(a.ExcludeSegments), ","), strings.Join(sorted(a.ExcludeTiers), ",")}
	if a.IncludeSunset {
		parts = append(parts, "sunset")
	}
	if a.Paid.Set() {
		parts = append(parts, "paid:"+a.Paid.Who+":"+a.Paid.Tier)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:12])
}

func sorted(s []string) []string {
	out := append([]string{}, s...)
	sort.Strings(out)
	return out
}

func uuids(in []string) []uuid.UUID {
	out := []uuid.UUID{}
	for _, s := range in {
		if u, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
			out = append(out, u)
		}
	}
	return out
}

// FinalRecipients reads an audience's final recipients the way Send
// Readiness counts them (suppressed, role and duplicate addresses and
// contacts the sunset policy stopped left out), with what conditions read.
func FinalRecipients(ctx context.Context, q *store.Queries, segs SegmentResolver, wsID uuid.UUID, a Audience, ruleSegments Segments) ([]*Facts, error) {
	if a.Empty() {
		return []*Facts{}, nil
	}
	members := func(ids []string) ([]uuid.UUID, error) {
		out := []uuid.UUID{}
		if segs == nil {
			return out, nil
		}
		for _, id := range uuids(ids) {
			m, err := segs.ContactIDs(ctx, wsID, id)
			if err != nil {
				return nil, err
			}
			out = append(out, m...)
		}
		return out, nil
	}
	in, err := members(a.Segments)
	if err != nil {
		return nil, err
	}
	out, err := members(a.ExcludeSegments)
	if err != nil {
		return nil, err
	}
	rows, err := q.VisibilityRecipients(ctx, store.VisibilityRecipientsParams{WorkspaceID: wsID, Lists: uuids(a.Lists), Contacts: in, ExcludeLists: uuids(a.ExcludeLists), ExcludeContacts: out})
	if err != nil {
		return nil, err
	}
	// A paid-only campaign goes to the paid members on its lists, or to
	// the free subscribers.
	keep, err := paidaccess.Load(ctx, q, wsID, a.Paid)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(rows))
	excludeSunset := slices.Contains(a.ExcludeTiers, "sunset")
	res := make([]*Facts, 0, len(rows))
	for _, r := range rows {
		if !keep.Keeps(r.ID) {
			continue
		}
		if r.Suppressed || r.RoleAddress || emailaddr.IsRole(r.Email) {
			continue
		}
		k := mailboxKey(r.Email)
		if seen[k] {
			continue
		}
		seen[k] = true
		if r.BroadcastsStopped && (!a.IncludeSunset || excludeSunset) {
			continue
		}
		if slices.Contains(a.ExcludeTiers, r.Tier) {
			continue
		}
		res = append(res, newFacts(r.ID, r.Email, r.FirstName, r.LastName, r.Tier, r.Language, r.Tags, r.Attributes, r.ListIds, ruleSegments))
	}
	return res, nil
}

// mailboxKey folds addresses that reach the same mailbox, as Send Readiness
// counts duplicates: plus-tags anywhere, dots in Gmail addresses.
func mailboxKey(email string) string {
	e := strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndexByte(e, '@')
	if at < 0 {
		return e
	}
	local, domain := e[:at], e[at+1:]
	if i := strings.IndexByte(local, '+'); i > 0 {
		local = local[:i]
	}
	if domain == "googlemail.com" {
		domain = "gmail.com"
	}
	if domain == "gmail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}
	return local + "@" + domain
}

// Tally is a count of an audience by version: how many recipients get each
// set of hidden blocks.
type Tally struct {
	Total int64
	// ByHidden counts recipients per hidden set (Hidden.String()).
	ByHidden map[string]int64
	// Shown counts, for each block, the recipients who see it.
	Shown map[string]int64
}

// Count decides every recipient and tallies the versions.
func Count(e Email, recipients []*Facts, n *Names) Tally {
	t := Tally{ByHidden: map[string]int64{}, Shown: map[string]int64{}}
	for _, b := range e.Blocks {
		t.Shown[b.ID] = 0
	}
	for _, f := range recipients {
		d := Decide(e, f, n)
		t.Total++
		t.ByHidden[d.Hidden.String()]++
		for _, b := range e.Blocks {
			if !d.Hidden[b.ID] {
				t.Shown[b.ID]++
			}
		}
	}
	return t
}

// VisibilityVersion is one combination of conditional blocks that some of
// the audience gets.
type VisibilityVersion struct {
	Number int `json:"number"`
	// Hidden are the ids of the blocks it leaves out; Shown the ones it
	// shows, in the email's order.
	Hidden []string `json:"hidden"`
	Shown  []string `json:"shown"`
	// Shows names what it shows: "Beta corner, Start a free trial", or ""
	// for none of the conditional blocks.
	Shows      string `json:"shows"`
	Recipients int64  `json:"recipients"`
	SizeBytes  int    `json:"sizeBytes"`
	// OverClip is true over Gmail's clipping size (102 KB).
	OverClip bool `json:"overClip"`
	// Empty is true when nothing is left between the header and the footer.
	Empty bool `json:"empty"`
	// Largest and Smallest mark the extremes by size.
	Largest  bool `json:"largest,omitempty"`
	Smallest bool `json:"smallest,omitempty"`
	// Checked is false for versions Send Readiness doesn't render one by one
	// (beyond the 50 with the most recipients, except the extremes).
	Checked bool `json:"checked"`
}

// key is the version's hidden set as Hidden.String() writes it (Hidden
// lists the blocks in the email's order).
func (v VisibilityVersion) key() string {
	h := make(Hidden, len(v.Hidden))
	for _, id := range v.Hidden {
		h[id] = true
	}
	return h.String()
}

// orderVersions sorts versions: by their mask when few, the blocks seen by fewer
// recipients counting more (none first, then each rarer block's versions
// together); by recipients when there are more than MaxChecked.
func orderVersions(e Email, t Tally, list []VisibilityVersion) {
	if len(list) > MaxChecked {
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Recipients != list[j].Recipients {
				return list[i].Recipients > list[j].Recipients
			}
			return strings.Join(list[i].Hidden, ",") < strings.Join(list[j].Hidden, ",")
		})
		return
	}
	blocks := append([]Block(nil), e.Blocks...)
	sort.SliceStable(blocks, func(i, j int) bool { return t.Shown[blocks[i].ID] > t.Shown[blocks[j].ID] })
	bit := map[string]uint{}
	for i, b := range blocks {
		bit[b.ID] = uint(i)
	}
	mask := func(v VisibilityVersion) uint64 {
		var m uint64
		for _, id := range v.Shown {
			if b, ok := bit[id]; ok && b < 64 {
				m |= 1 << b
			}
		}
		return m
	}
	sort.SliceStable(list, func(i, j int) bool {
		mi, mj := mask(list[i]), mask(list[j])
		if mi != mj {
			return mi < mj
		}
		return list[i].Recipients > list[j].Recipients
	})
}

// Versions turns a tally into the versions table: what each shows, how many
// get it and its size (of the given HTML, the source as compiled).
func Versions(e Email, t Tally, html string) []VisibilityVersion {
	out := make([]VisibilityVersion, 0, len(t.ByHidden))
	for key, n := range t.ByHidden {
		hidden := ParseHidden(key)
		v := VisibilityVersion{Hidden: []string{}, Shown: []string{}, Recipients: n}
		names := []string{}
		for _, b := range e.Blocks {
			if hidden[b.ID] {
				v.Hidden = append(v.Hidden, b.ID)
				continue
			}
			v.Shown = append(v.Shown, b.ID)
			// A block shown inside another counts as part of it.
			if b.Parent == "" || !slices.Contains(v.Shown, b.Parent) {
				names = append(names, b.Name)
			}
		}
		v.Shows = strings.Join(names, ", ")
		if html != "" {
			v.SizeBytes = Size(html, hidden)
			v.OverClip = v.SizeBytes > contentcheck.ClipBytes
		}
		v.Empty = e.Empty(hidden)
		out = append(out, v)
	}
	orderVersions(e, t, out)
	largest, smallest := -1, -1
	for i := range out {
		out[i].Number = i + 1
		if largest < 0 || out[i].SizeBytes > out[largest].SizeBytes {
			largest = i
		}
		if smallest < 0 || out[i].SizeBytes < out[smallest].SizeBytes {
			smallest = i
		}
	}
	if largest >= 0 && len(out) > 1 {
		out[largest].Largest = true
		out[smallest].Smallest = true
	}
	// Send Readiness checks the 50 with the most recipients, plus the
	// largest and the smallest.
	byRecipients := make([]int, len(out))
	for i := range out {
		byRecipients[i] = i
	}
	sort.SliceStable(byRecipients, func(a, b int) bool { return out[byRecipients[a]].Recipients > out[byRecipients[b]].Recipients })
	for rank, i := range byRecipients {
		if rank < MaxChecked || out[i].Largest || out[i].Smallest {
			out[i].Checked = true
		}
	}
	return out
}

// Possible is how many combinations the blocks allow (2^n, capped).
func Possible(e Email) int64 {
	n := len(e.Blocks)
	if n >= 62 {
		return 1 << 62
	}
	return int64(1) << uint(n)
}
