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
	// IncludeUnengaged includes or leaves out the unengaged (the sunset
	// tier); unset, a send leaves them out (SR-11).
	IncludeUnengaged  *bool `json:"include_unengaged"`
	IncludeUnengaged2 *bool `json:"includeUnengaged"`
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
	if a.IncludeUnengaged == nil {
		a.IncludeUnengaged = a.IncludeUnengaged2
	}
	return a
}

// leavesOutUnengaged reports whether a send leaves out the unengaged (the
// sunset tier): as the campaign says, else by default (byDefault) unless
// it overrides the sunset policy, as Send Readiness counts them (SR-11).
func (a Audience) leavesOutUnengaged(byDefault bool) bool {
	switch {
	case slices.Contains(a.ExcludeTiers, "sunset"):
		return true
	case a.IncludeUnengaged != nil:
		return !*a.IncludeUnengaged
	}
	return byDefault && !a.IncludeSunset
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
	switch {
	case a.IncludeUnengaged == nil:
	case *a.IncludeUnengaged:
		parts = append(parts, "unengaged:in")
	default:
		parts = append(parts, "unengaged:out")
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
	return finalRecipients(ctx, q, segs, wsID, a, ruleSegments, nil)
}

// finalRecipients is FinalRecipients, for the contacts among only when
// among isn't nil: those of them still in the audience.
func finalRecipients(ctx context.Context, q *store.Queries, segs SegmentResolver, wsID uuid.UUID, a Audience, ruleSegments Segments, among []uuid.UUID) ([]*Facts, error) {
	if a.Empty() || among != nil && len(among) == 0 {
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
	var rows []store.VisibilityRecipientsRow
	if among == nil {
		rows, err = q.VisibilityRecipients(ctx, store.VisibilityRecipientsParams{WorkspaceID: wsID, Lists: uuids(a.Lists), Contacts: in, ExcludeLists: uuids(a.ExcludeLists), ExcludeContacts: out})
	} else {
		var some []store.VisibilityRecipientsAmongRow
		some, err = q.VisibilityRecipientsAmong(ctx, store.VisibilityRecipientsAmongParams{Among: among, WorkspaceID: wsID, Lists: uuids(a.Lists), Contacts: in,
			ExcludeLists: uuids(a.ExcludeLists), ExcludeContacts: out})
		rows = make([]store.VisibilityRecipientsRow, len(some))
		for i, r := range some {
			rows[i] = store.VisibilityRecipientsRow(r)
		}
	}
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
	byDefault := true
	if d, err := q.SendDefaults(ctx, wsID); err == nil {
		byDefault = d.ExcludeUnengaged
	}
	leaveOut := a.leavesOutUnengaged(byDefault)
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
		if slices.Contains(a.ExcludeTiers, r.Tier) || leaveOut && r.Tier == "sunset" {
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
	// Samples are the contacts to preview the versions as.
	Samples Samples
}

// Samples are, for each version (Hidden.String()) and each base language of
// an audience, its first recipient with a first name, else its first one:
// the contacts the composer previews the email as (PreviewContacts).
type Samples struct {
	Versions map[string]uuid.UUID `json:"versions"`
	// Languages go from the one most recipients read to the fewest.
	Languages []LanguageSample `json:"languages"`
}

// LanguageSample is the contact a base language ("es") is previewed as.
type LanguageSample struct {
	Language string    `json:"language"`
	ID       uuid.UUID `json:"id"`
}

// sampler picks Samples from recipients decided in audience order.
type sampler struct {
	versions, languages map[string]*Facts
	readers             map[string]int64
}

func newSampler() *sampler {
	return &sampler{versions: map[string]*Facts{}, languages: map[string]*Facts{}, readers: map[string]int64{}}
}

// add is a recipient who gets the version key.
func (s *sampler) add(key string, f *Facts) {
	if cur, ok := s.versions[key]; !ok || cur.FirstName == "" && f.FirstName != "" {
		s.versions[key] = f
	}
	l := baseLanguage(f.Language)
	if l == "" {
		return
	}
	s.readers[l]++
	if cur, ok := s.languages[l]; !ok || cur.FirstName == "" && f.FirstName != "" {
		s.languages[l] = f
	}
}

func (s *sampler) samples() Samples {
	out := Samples{Versions: make(map[string]uuid.UUID, len(s.versions)), Languages: make([]LanguageSample, 0, len(s.languages))}
	for k, f := range s.versions {
		out.Versions[k] = f.ID
	}
	for l, f := range s.languages {
		out.Languages = append(out.Languages, LanguageSample{Language: l, ID: f.ID})
	}
	sort.Slice(out.Languages, func(i, j int) bool {
		a, b := out.Languages[i].Language, out.Languages[j].Language
		if s.readers[a] != s.readers[b] {
			return s.readers[a] > s.readers[b]
		}
		return a < b
	})
	return out
}

// Count decides every recipient and tallies the versions.
func Count(e Email, recipients []*Facts, n *Names) Tally {
	t := Tally{ByHidden: map[string]int64{}, Shown: map[string]int64{}}
	for _, b := range e.Blocks {
		t.Shown[b.ID] = 0
	}
	sm := newSampler()
	for _, f := range recipients {
		d := Decide(e, f, n)
		key := d.Hidden.String()
		t.Total++
		t.ByHidden[key]++
		for _, b := range e.Blocks {
			if !d.Hidden[b.ID] {
				t.Shown[b.ID]++
			}
		}
		sm.add(key, f)
	}
	t.Samples = sm.samples()
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
