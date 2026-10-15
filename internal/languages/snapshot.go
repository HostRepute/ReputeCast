package languages

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

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
			v.Blocks = append(v.Blocks, BlockSnapshot{Name: b.Name, Hash: b.Hash, Raw: b.Raw})
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
		return VersionContent{}, notFoundVersion(err, lang)
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
// translated or held.
func DiffVersions(before, after []VersionSnapshot) []VersionChange {
	out := []VersionChange{}
	prev := map[string]VersionSnapshot{}
	for _, v := range before {
		prev[v.Language] = v
	}
	seen := map[string]bool{}
	for _, v := range after {
		seen[v.Language] = true
		name := NameOf(v.Language)
		p, ok := prev[v.Language]
		switch {
		case !ok:
			out = append(out, VersionChange{Language: v.Language, Name: name, Before: "none", After: fmt.Sprintf("v%d · added", v.Version), What: "version added"})
		case p.Hash != v.Hash:
			what := changedBlocks(p, v)
			out = append(out, VersionChange{Language: v.Language, Name: name, Before: fmt.Sprintf("v%d", p.Version), After: fmt.Sprintf("v%d · %s", v.Version, what), What: what})
		case p.Status != v.Status:
			what := "marked translated"
			if v.Status != storedTranslated {
				what = "back to draft"
			}
			out = append(out, VersionChange{Language: v.Language, Name: name, Before: statusName(p.Status), After: statusName(v.Status), What: what})
		case p.Held != v.Held:
			what := "sent again"
			if v.Held {
				what = "held"
			}
			out = append(out, VersionChange{Language: v.Language, Name: name, Before: heldWord(p.Held), After: heldWord(v.Held), What: what})
		}
	}
	for _, p := range before {
		if !seen[p.Language] {
			out = append(out, VersionChange{Language: p.Language, Name: NameOf(p.Language), Before: fmt.Sprintf("v%d", p.Version), After: "removed", What: "version removed"})
		}
	}
	return out
}

func statusName(s string) string {
	if s == storedTranslated {
		return "translated"
	}
	return "draft"
}

func heldWord(held bool) string {
	if held {
		return "held"
	}
	return "sent"
}

// changedBlocks names what changed between two snapshots of a version:
// "intro changed", "subject and 2 blocks changed".
func changedBlocks(before, after VersionSnapshot) string {
	old := make([]AlignKey, len(before.Blocks))
	for i, b := range before.Blocks {
		old[i] = AlignKey{Raw: b.Raw, Hash: b.Hash}
	}
	cur := make([]AlignKey, len(after.Blocks))
	for i, b := range after.Blocks {
		cur[i] = AlignKey{Raw: b.Raw, Hash: b.Hash}
	}
	m := Align(old, cur)
	names := []string{}
	used := map[int]bool{}
	for i, j := range m {
		if j >= 0 {
			used[j] = true
		}
		if j < 0 || old[j].Raw != cur[i].Raw {
			names = append(names, shortBlock(after.Blocks[i].Name))
		}
	}
	removed := 0
	for j := range old {
		if !used[j] {
			removed++
		}
	}
	parts := []string{}
	if before.Subject != after.Subject {
		parts = append(parts, "subject")
	}
	switch {
	case len(names) == 1:
		parts = append(parts, names[0])
	case len(names) > 1:
		parts = append(parts, fmt.Sprintf("%d blocks", len(names)))
	}
	if removed > 0 && len(names) == 0 {
		parts = append(parts, "a block removed")
		return strings.Join(parts, " and ")
	}
	if len(parts) == 0 {
		return "edited"
	}
	return strings.Join(parts, " and ") + " changed"
}

// shortBlock is a block's name in a sentence: "intro", "two cards".
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
