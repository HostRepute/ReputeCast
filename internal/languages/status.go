package languages

import (
	"strconv"
	"time"
)

// Translation statuses of a language version.
const (
	StatusSource   = "source"
	StatusUpToDate = "up_to_date"
	StatusOutdated = "outdated"
	StatusDraft    = "draft"
	StatusMissing  = "missing"
)

// Stored statuses of content_translations.
const (
	storedDraft      = "draft"
	storedTranslated = "translated"
)

// SourceBlock is a block of the source with its key: what the source looks
// like now, and when it last changed.
type SourceBlock struct {
	Key  string `json:"key"`
	Raw  string `json:"raw"`
	Hash string `json:"hash"`
	Text string `json:"text"`
	// Changed is the source save that last changed what is translated.
	Changed *BlockChange `json:"changed,omitempty"`
}

// BlockChange says which source version changed a block, who and when.
type BlockChange struct {
	Version int       `json:"version"`
	By      string    `json:"by"`
	At      time.Time `json:"at"`
}

// Links are what a language version was translated from: per block, the
// source block, its fingerprint and text then, and the source version.
type Links struct {
	Subject   *Link       `json:"subject,omitempty"`
	Preheader *Link       `json:"preheader,omitempty"`
	Blocks    []BlockLink `json:"blocks"`
}

// Link is the subject or preheader a version was translated from.
type Link struct {
	Hash    string `json:"hash"`
	Text    string `json:"text"`
	Version int    `json:"version"`
}

// BlockLink ties a block of a language version to its source block.
type BlockLink struct {
	// Src is the source block's key; empty for a block only this version has.
	Src     string `json:"src,omitempty"`
	Hash    string `json:"hash,omitempty"`
	Text    string `json:"text,omitempty"`
	Version int    `json:"version,omitempty"`
	// Own and OwnRaw fingerprint the version's own block, to keep the link
	// when the version is saved again.
	Own    string `json:"own"`
	OwnRaw string `json:"ownRaw"`
}

// Row states of the compare view.
const (
	RowUpToDate = "up_to_date"
	RowChanged  = "changed"
	RowStill    = "still"
	RowNew      = "new"
	RowRemoved  = "removed"
	RowOnly     = "only"
)

// Row is one row of the compare view: a source block and the version's
// block made from it.
type Row struct {
	// Number is the source block's position (1-based); 0 for a block the
	// source no longer has or never had.
	Number  int
	Key     string
	Source  *Block
	Version *Block
	// VersionIndex is the version block's position, -1 when it has none.
	VersionIndex int
	State        string
	// Before is the source text the version was translated from (the diff's
	// old side), and FromVersion the source version then.
	Before      string
	FromVersion int
	Changed     *BlockChange
}

// Progress counts a version's blocks against the source.
type Progress struct {
	Status string
	// Attention counts what needs a look: blocks changed in, new in or
	// removed from the source, and a changed subject or preheader.
	Attention int
	// Blocks is the number of source blocks; UpToDate of them are.
	Blocks   int
	UpToDate int
	// Changed counts blocks changed in the source since translation.
	Changed int
	// Still counts blocks of a draft that still say what the source says;
	// Translated the others.
	Still            int
	Translated       int
	SubjectChanged   bool
	PreheaderChanged bool
}

// Compare lines a version up with its source block by block and computes
// its status. source and keys describe the current source (same length);
// version are the version's blocks and links what they were translated from.
func Compare(source []Block, keys []SourceBlock, version []Block, links Links, draft bool, subjectHash, preheaderHash string) ([]Row, Progress) {
	byKey := map[string]int{}
	for i, k := range keys {
		byKey[k.Key] = i
	}
	pairedSource := make([]bool, len(source))
	pairs := make([]int, len(version)) // version block → source index, -1
	for i := range pairs {
		pairs[i] = -1
		if i >= len(links.Blocks) {
			continue
		}
		if si, ok := byKey[links.Blocks[i].Src]; ok && !pairedSource[si] {
			pairs[i] = si
			pairedSource[si] = true
		}
	}
	rows := make([]Row, 0, len(source)+len(version))
	p := Progress{Blocks: len(source)}
	emitVersion := func(vi int) {
		l := BlockLink{}
		if vi < len(links.Blocks) {
			l = links.Blocks[vi]
		}
		r := Row{VersionIndex: vi, Version: &version[vi], Before: l.Text, FromVersion: l.Version, Key: l.Src, State: RowOnly}
		if l.Src != "" {
			r.State = RowRemoved
		}
		p.Attention += boolInt(r.State == RowRemoved)
		rows = append(rows, r)
	}
	nextVersion := 0
	for si := range source {
		// Version blocks before this source block's partner that have no
		// partner of their own come first.
		vi := -1
		for i, s := range pairs {
			if s == si {
				vi = i
				break
			}
		}
		if vi >= 0 {
			for ; nextVersion < vi; nextVersion++ {
				if pairs[nextVersion] < 0 {
					emitVersion(nextVersion)
				}
			}
			nextVersion = max(nextVersion, vi+1)
		}
		r := Row{Number: si + 1, Key: keys[si].Key, Source: &source[si], VersionIndex: vi, Changed: keys[si].Changed}
		switch {
		case vi < 0:
			r.State = RowNew
			p.Attention++
		default:
			l := links.Blocks[vi]
			r.Version, r.Before, r.FromVersion = &version[vi], l.Text, l.Version
			switch {
			case l.Hash != source[si].Hash && !draft:
				r.State = RowChanged
				p.Changed++
				p.Attention++
			case version[vi].Hash == source[si].Hash && source[si].Hash != emptyHash && draft:
				r.State = RowStill
				p.Still++
			case l.Hash != source[si].Hash && draft:
				// A draft translated from an older source: show what changed.
				r.State = RowChanged
				p.Changed++
			default:
				r.State = RowUpToDate
				p.UpToDate++
			}
		}
		rows = append(rows, r)
	}
	for ; nextVersion < len(version); nextVersion++ {
		if pairs[nextVersion] < 0 {
			emitVersion(nextVersion)
		}
	}
	if links.Subject != nil && subjectHash != "" && links.Subject.Hash != subjectHash {
		p.SubjectChanged = true
		p.Attention++
	}
	if links.Preheader != nil && preheaderHash != "" && links.Preheader.Hash != preheaderHash {
		p.PreheaderChanged = true
		p.Attention++
	}
	p.Translated = p.Blocks - p.Still
	switch {
	case draft:
		p.Status = StatusDraft
	case p.Attention > 0:
		p.Status = StatusOutdated
	default:
		p.Status = StatusUpToDate
	}
	return rows, p
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// SyncSource brings stored source blocks up to the current source: blocks
// keep their keys through edits and moves, new blocks get new keys, and a
// block whose translatable text changed records the change.
func SyncSource(stored []SourceBlock, current []Block, nextKey int, change BlockChange) ([]SourceBlock, int, []string) {
	before := make([]AlignKey, len(stored))
	for i, b := range stored {
		before[i] = AlignKey{Raw: b.Raw, Hash: b.Hash}
	}
	m := Align(before, keysOf(current))
	out := make([]SourceBlock, len(current))
	changed := []string{}
	for i, b := range current {
		sb := SourceBlock{Raw: b.Raw, Hash: b.Hash, Text: b.Text}
		if j := m[i]; j >= 0 {
			sb.Key, sb.Changed = stored[j].Key, stored[j].Changed
			if stored[j].Hash != b.Hash {
				c := change
				sb.Changed = &c
				changed = append(changed, sb.Key)
			}
		} else {
			sb.Key = "b" + strconv.Itoa(nextKey)
			nextKey++
			c := change
			sb.Changed = &c
		}
		out[i] = sb
	}
	return out, nextKey, changed
}

// FreshLinks links every block of a version made from the source as it is
// now: a new version copied from the source, or one marked up to date.
func FreshLinks(source []Block, keys []SourceBlock, version []Block, subject, preheader string, sourceVersion int) Links {
	l := Links{Blocks: make([]BlockLink, len(version))}
	m := Align(keysOf(source), keysOf(version))
	used := map[int]bool{}
	for i, vb := range version {
		bl := BlockLink{Own: vb.Hash, OwnRaw: vb.Raw}
		j := m[i]
		if j < 0 && i < len(source) && !used[i] {
			j = i // same position: the layout was copied
		}
		if j >= 0 && j < len(keys) && !used[j] {
			used[j] = true
			bl.Src, bl.Hash, bl.Text, bl.Version = keys[j].Key, source[j].Hash, source[j].Text, sourceVersion
		}
		l.Blocks[i] = bl
	}
	l.Subject = &Link{Hash: HashText(subject), Text: subject, Version: sourceVersion}
	l.Preheader = &Link{Hash: HashText(preheader), Text: preheader, Version: sourceVersion}
	return l
}

// Relink carries a version's links over to its new content after it was
// edited: blocks keep their source while they are edited or moved, and new
// blocks are the version's own.
func Relink(old Links, version []Block) Links {
	before := make([]AlignKey, len(old.Blocks))
	for i, b := range old.Blocks {
		before[i] = AlignKey{Raw: b.OwnRaw, Hash: b.Own}
	}
	m := Align(before, keysOf(version))
	out := Links{Subject: old.Subject, Preheader: old.Preheader, Blocks: make([]BlockLink, len(version))}
	for i, vb := range version {
		bl := BlockLink{Own: vb.Hash, OwnRaw: vb.Raw}
		if j := m[i]; j >= 0 {
			o := old.Blocks[j]
			bl.Src, bl.Hash, bl.Text, bl.Version = o.Src, o.Hash, o.Text, o.Version
		}
		out.Blocks[i] = bl
	}
	return out
}

// MarkUpToDate points links at the current source: every block when keys
// is empty, otherwise the blocks with those source keys. The subject and
// preheader are refreshed with every block.
func MarkUpToDate(l Links, source []Block, sourceKeys []SourceBlock, subject, preheader string, sourceVersion int, only []string) Links {
	idx := map[string]int{}
	for i, k := range sourceKeys {
		idx[k.Key] = i
	}
	want := map[string]bool{}
	for _, k := range only {
		want[k] = true
	}
	out := Links{Subject: l.Subject, Preheader: l.Preheader, Blocks: make([]BlockLink, len(l.Blocks))}
	for i, b := range l.Blocks {
		if j, ok := idx[b.Src]; ok && (len(want) == 0 || want[b.Src]) {
			b.Hash, b.Text, b.Version = source[j].Hash, source[j].Text, sourceVersion
		}
		out.Blocks[i] = b
	}
	if len(want) == 0 || want["subject"] {
		out.Subject = &Link{Hash: HashText(subject), Text: subject, Version: sourceVersion}
	}
	if len(want) == 0 || want["preheader"] {
		out.Preheader = &Link{Hash: HashText(preheader), Text: preheader, Version: sourceVersion}
	}
	return out
}
