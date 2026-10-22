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
}

// DiffVersions compares the versions an approval covered with the
// versions now: added, removed, edited (naming the blocks), marked
// translated or held. It writes in English; DiffVersionsIn writes for a
// reader.
func DiffVersions(before, after []VersionSnapshot) []VersionChange {
	return DiffVersionsIn(i18n.Default, before, after)
}

// DiffVersionsIn is DiffVersions in a reader's language: the language
// names and what changed read in l.
func DiffVersionsIn(l i18n.Lang, before, after []VersionSnapshot) []VersionChange {
	out := []VersionChange{}
	prev := map[string]VersionSnapshot{}
	for _, v := range before {
		prev[v.Language] = v
	}
	seen := map[string]bool{}
	for _, v := range after {
		seen[v.Language] = true
		name := NameIn(l, v.Language)
		p, ok := prev[v.Language]
		switch {
		case !ok:
			out = append(out, VersionChange{Language: v.Language, Name: name, Before: mDiffNone.Text(l, nil),
				After: mDiffAdded.Text(l, i18n.Args{"version": strconv.Itoa(v.Version)}), What: mChangeVersionAdded.Text(l, nil)})
		case p.Hash != v.Hash:
			what := changedBlocks(l, p, v)
			out = append(out, VersionChange{Language: v.Language, Name: name, Before: fmt.Sprintf("v%d", p.Version),
				After: mDiffChanged.Text(l, i18n.Args{"version": strconv.Itoa(v.Version), "what": what}), What: what})
		case p.Status != v.Status:
			what := mChangeMarkedTranslated
			if v.Status != storedTranslated {
				what = mChangeBackToDraft
			}
			out = append(out, VersionChange{Language: v.Language, Name: name, Before: statusName(l, p.Status), After: statusName(l, v.Status), What: what.Text(l, nil)})
		case p.Held != v.Held:
			what := mChangeSentAgain
			if v.Held {
				what = mChangeHeld
			}
			out = append(out, VersionChange{Language: v.Language, Name: name, Before: heldWord(l, p.Held), After: heldWord(l, v.Held), What: what.Text(l, nil)})
		}
	}
	for _, p := range before {
		if !seen[p.Language] {
			out = append(out, VersionChange{Language: p.Language, Name: NameIn(l, p.Language), Before: fmt.Sprintf("v%d", p.Version),
				After: mDiffRemoved.Text(l, nil), What: mChangeVersionRemoved.Text(l, nil)})
		}
	}
	return out
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

// changedBlocks names what changed between two snapshots of a version:
// "intro changed", "subject and 2 blocks changed".
func changedBlocks(l i18n.Lang, before, after VersionSnapshot) string {
	old := make([]AlignKey, len(before.Blocks))
	for i, b := range before.Blocks {
		old[i] = AlignKey{Raw: b.Raw, Hash: b.Hash}
	}
	cur := make([]AlignKey, len(after.Blocks))
	for i, b := range after.Blocks {
		cur[i] = AlignKey{Raw: b.Raw, Hash: b.Hash}
	}
	m := Align(old, cur)
	changed := []BlockSnapshot{}
	used := map[int]bool{}
	for i, j := range m {
		if j >= 0 {
			used[j] = true
		}
		if j < 0 || old[j].Raw != cur[i].Raw {
			changed = append(changed, after.Blocks[i])
		}
	}
	removed := 0
	for j := range old {
		if !used[j] {
			removed++
		}
	}
	subject := before.Subject != after.Subject
	switch {
	case len(changed) == 0 && removed > 0 && subject:
		return mChangeSubjectBlockRemoved.Text(l, nil)
	case len(changed) == 0 && removed > 0:
		return mChangeBlockRemoved.Text(l, nil)
	case len(changed) == 0 && subject:
		return mChangeSubject.Text(l, nil)
	case len(changed) == 0:
		return mChangeEdited.Text(l, nil)
	case len(changed) == 1 && subject:
		return mChangeSubjectBlock.Text(l, i18n.Args{"block": shortName(l, changed[0])})
	case len(changed) == 1:
		return mChangeBlock.Text(l, i18n.Args{"block": shortName(l, changed[0])})
	case subject:
		return mChangeSubjectBlocks.Text(l, i18n.Args{"count": len(changed)})
	}
	return mChangeBlocks.Text(l, i18n.Args{"count": len(changed)})
}

// shortName is a block in a sentence, in l: "intro", "two cards".
func shortName(l i18n.Lang, b BlockSnapshot) string {
	if l.Locale != i18n.English && b.Ref != nil && b.Ref.Kind != "" {
		return b.Ref.Short(l)
	}
	return shortBlock(b.Name)
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
