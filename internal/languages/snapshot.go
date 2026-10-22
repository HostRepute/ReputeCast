package languages

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// VersionSnapshot is what an approval covers of one language version.
type VersionSnapshot struct {
	Language string `json:"language"`
	Status   string `json:"status"`
	Held     bool   `json:"held,omitempty"`
	Version  int    `json:"version"`
	Hash     string `json:"hash"`
	Subject  string `json:"subject"`
	// Blocks fingerprint the version's blocks, to name what changed.
	Blocks []BlockSnapshot `json:"blocks"`
}

// BlockSnapshot is one block of a version at approval time.
type BlockSnapshot struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
	Raw  string `json:"raw"`
	// Ref says what the block is, to name it in any language; snapshots
	// taken before it was kept have none (their English Name is used).
	Ref *BlockRef `json:"ref,omitempty"`
}

// CampaignSnapshot returns what an approval covers of a campaign's
// language versions (nil without versions).
func (s *Service) CampaignSnapshot(ctx context.Context, q *store.Queries, campaignID uuid.UUID) ([]VersionSnapshot, error) {
	if q == nil {
		q = s.Q
	}
	rows, err := q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: OwnerCampaign, OwnerID: campaignID})
	if err != nil {
		return nil, err
	}
	var out []VersionSnapshot
	for _, r := range rows {
		v := VersionSnapshot{Language: r.Language, Status: r.Status, Held: r.Held, Version: int(r.Version), Hash: r.ContentHash, Subject: r.Subject, Blocks: []BlockSnapshot{}}
		for _, b := range Blocks(r.Mjml, r.Html) {
			ref := b.ref
			v.Blocks = append(v.Blocks, BlockSnapshot{Name: b.Name, Hash: b.Hash, Raw: b.Raw, Ref: &ref})
		}
		out = append(out, v)
	}
	return out, nil
}

// VersionContent is a language version's content at one of its versions.
type VersionContent struct {
	Subject   string
	Preheader string
	MJML      string
	HTML      string
}

// CampaignVersionAt returns a campaign's language version as it was at a
// saved version (the version an approval covers); 0, or a version no longer
// kept, gives the version as it is now.
func (s *Service) CampaignVersionAt(ctx context.Context, campaignID uuid.UUID, lang string, version int) (VersionContent, error) {
	t, err := s.Q.ContentTranslation(ctx, store.ContentTranslationParams{OwnerKind: OwnerCampaign, OwnerID: campaignID, Language: Base(lang)})
	if err != nil {
		return VersionContent{}, notFoundVersion(langOf(ctx), err, lang)
	}
	if version > 0 && version != int(t.Version) {
		if v, err := s.Q.ContentTranslationVersion(ctx, store.ContentTranslationVersionParams{TranslationID: t.ID, Version: int32(version)}); err == nil {
			return VersionContent{Subject: v.Subject, Preheader: v.Preheader, MJML: v.Mjml, HTML: v.Html}, nil
		}
	}
	return VersionContent{Subject: t.Subject, Preheader: t.Preheader, MJML: t.Mjml, HTML: t.Html}, nil
}

// VersionChange is a language version that changed after approval.
type VersionChange struct {
	Language string
	// Name is the language's name ("German").
	Name string
	// Before and After are "v3" and "v4 · intro changed".
	Before string
	After  string
	// What is the change in a few words ("intro changed", "added").
	What string
	// Detail is the change as data, to keep with an approval and write in
	// each reader's language later (VersionChangeDetail.Texts).
	Detail VersionChangeDetail
}

// VersionChangeDetail is what changed in a language version after
// approval, as data: an approval stores it and writes it in each reader's
// language when it's read (Texts).
type VersionChangeDetail struct {
	Language string `json:"language"`
	// Change is added, edited, status (marked translated or back to
	// draft), held (held or sent again) or removed.
	Change string `json:"change"`
	// Prev and Version are the version numbers before and after.
	Prev    int `json:"prev,omitempty"`
	Version int `json:"version,omitempty"`
	// PrevStatus and Status are the stored statuses (draft, translated);
	// PrevHeld and Held whether the version was held.
	PrevStatus string `json:"prevStatus,omitempty"`
	Status     string `json:"status,omitempty"`
	PrevHeld   bool   `json:"prevHeld,omitempty"`
	Held       bool   `json:"held,omitempty"`
	// What an edit changed: the subject, how many blocks (one of them named
	// by Block, or by its English BlockName for snapshots without refs),
	// and whether a block was removed.
	Subject   bool      `json:"subject,omitempty"`
	Blocks    int       `json:"blocks,omitempty"`
	Block     *BlockRef `json:"block,omitempty"`
	BlockName string    `json:"blockName,omitempty"`
	Removed   bool      `json:"removed,omitempty"`
}

// DiffVersions compares the versions an approval covered with the
// versions now: added, removed, edited (naming the blocks), marked
// translated or held. It writes in English; DiffVersionsIn writes for a
// reader, and each change's Detail writes it for later readers.
func DiffVersions(before, after []VersionSnapshot) []VersionChange {
	return DiffVersionsIn(i18n.Default, before, after)
}

// DiffVersionsIn is DiffVersions in a reader's language: the language
// names and what changed read in l.
func DiffVersionsIn(l i18n.Lang, before, after []VersionSnapshot) []VersionChange {
	details := diffVersions(before, after)
	out := make([]VersionChange, 0, len(details))
	for _, d := range details {
		name, b, a, what := d.Texts(l)
		out = append(out, VersionChange{Language: d.Language, Name: name, Before: b, After: a, What: what, Detail: d})
	}
	return out
}

func diffVersions(before, after []VersionSnapshot) []VersionChangeDetail {
	out := []VersionChangeDetail{}
	prev := map[string]VersionSnapshot{}
	for _, v := range before {
		prev[v.Language] = v
	}
	seen := map[string]bool{}
	for _, v := range after {
		seen[v.Language] = true
		p, ok := prev[v.Language]
		d := VersionChangeDetail{Language: v.Language, Prev: p.Version, Version: v.Version}
		switch {
		case !ok:
			d.Change, d.Prev = "added", 0
		case p.Hash != v.Hash:
			d.Change = "edited"
			changedBlocks(&d, p, v)
		case p.Status != v.Status:
			d.Change, d.PrevStatus, d.Status = "status", p.Status, v.Status
		case p.Held != v.Held:
			d.Change, d.PrevHeld, d.Held = "held", p.Held, v.Held
		default:
			continue
		}
		out = append(out, d)
	}
	for _, p := range before {
		if !seen[p.Language] {
			out = append(out, VersionChangeDetail{Language: p.Language, Change: "removed", Prev: p.Version})
		}
	}
	return out
}

// Texts writes the change for a reader: the language's name, the version
// before and after, and what changed.
func (d VersionChangeDetail) Texts(l i18n.Lang) (name, before, after, what string) {
	name = NameIn(l, d.Language)
	switch d.Change {
	case "added":
		return name, mDiffNone.Text(l, nil), mDiffAdded.Text(l, i18n.Args{"version": strconv.Itoa(d.Version)}), mChangeVersionAdded.Text(l, nil)
	case "edited":
		what = d.what(l)
		return name, fmt.Sprintf("v%d", d.Prev), mDiffChanged.Text(l, i18n.Args{"version": strconv.Itoa(d.Version), "what": what}), what
	case "status":
		m := mChangeMarkedTranslated
		if d.Status != storedTranslated {
			m = mChangeBackToDraft
		}
		return name, statusName(l, d.PrevStatus), statusName(l, d.Status), m.Text(l, nil)
	case "held":
		m := mChangeSentAgain
		if d.Held {
			m = mChangeHeld
		}
		return name, heldWord(l, d.PrevHeld), heldWord(l, d.Held), m.Text(l, nil)
	}
	return name, fmt.Sprintf("v%d", d.Prev), mDiffRemoved.Text(l, nil), mChangeVersionRemoved.Text(l, nil)
}

func statusName(l i18n.Lang, s string) string {
	status := "draft"
	if s == storedTranslated {
		status = "translated"
	}
	return mDiffStatus.Text(l, i18n.Args{"status": status})
}

func heldWord(l i18n.Lang, held bool) string {
	h := "no"
	if held {
		h = "yes"
	}
	return mDiffHeld.Text(l, i18n.Args{"held": h})
}

// changedBlocks records what changed between two snapshots of a version:
// the subject, the blocks whose source changed and a removed block.
func changedBlocks(d *VersionChangeDetail, before, after VersionSnapshot) {
	old := make([]AlignKey, len(before.Blocks))
	for i, b := range before.Blocks {
		old[i] = AlignKey{Raw: b.Raw, Hash: b.Hash}
	}
	cur := make([]AlignKey, len(after.Blocks))
	for i, b := range after.Blocks {
		cur[i] = AlignKey{Raw: b.Raw, Hash: b.Hash}
	}
	m := Align(old, cur)
	used := map[int]bool{}
	for i, j := range m {
		if j >= 0 {
			used[j] = true
		}
		if j < 0 || old[j].Raw != cur[i].Raw {
			if d.Blocks == 0 {
				d.Block, d.BlockName = after.Blocks[i].Ref, after.Blocks[i].Name
			}
			d.Blocks++
		}
	}
	for j := range old {
		if !used[j] && d.Blocks == 0 {
			d.Removed = true
		}
	}
	d.Subject = before.Subject != after.Subject
	if d.Blocks != 1 {
		d.Block, d.BlockName = nil, ""
	}
}

// what names what an edit changed, in l: "intro changed", "subject and 2
// blocks changed".
func (d VersionChangeDetail) what(l i18n.Lang) string {
	switch {
	case d.Blocks == 0 && d.Removed && d.Subject:
		return mChangeSubjectBlockRemoved.Text(l, nil)
	case d.Blocks == 0 && d.Removed:
		return mChangeBlockRemoved.Text(l, nil)
	case d.Blocks == 0 && d.Subject:
		return mChangeSubject.Text(l, nil)
	case d.Blocks == 0:
		return mChangeEdited.Text(l, nil)
	case d.Blocks == 1:
		block := shortBlock(d.BlockName)
		if l.Locale != i18n.English && d.Block != nil && d.Block.Kind != "" {
			block = d.Block.Short(l)
		}
		if d.Subject {
			return mChangeSubjectBlock.Text(l, i18n.Args{"block": block})
		}
		return mChangeBlock.Text(l, i18n.Args{"block": block})
	case d.Subject:
		return mChangeSubjectBlocks.Text(l, i18n.Args{"count": d.Blocks})
	}
	return mChangeBlocks.Text(l, i18n.Args{"count": d.Blocks})
}

// shortBlock is a block's English name in a sentence: "intro", "two cards".
func shortBlock(name string) string {
	if i := strings.Index(name, " · Heading"); i >= 0 {
		name = name[:i]
	}
	if strings.HasPrefix(name, "Text · ") {
		return strings.TrimPrefix(name, "Text · ") + " text"
	}
	if strings.HasPrefix(name, "Header") {
		return "header"
	}
	if name == "" {
		return "a block"
	}
	return strings.ToLower(name[:1]) + name[1:]
}
