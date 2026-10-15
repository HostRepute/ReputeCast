package languages

import (
	"sort"
	"time"
)

// EmailLanguages is an email's language versions: what the composer's
// language switcher lists.
type EmailLanguages struct {
	// Source is the language the email is written in; SourceVersion its
	// current content version.
	Source        string `json:"source"`
	SourceVersion int    `json:"sourceVersion"`
	// CanChangeSource: the source language can change while the email has
	// only one version.
	CanChangeSource bool   `json:"canChangeSource"`
	Default         string `json:"default"`
	Fallback        string `json:"fallback"`
	// Blocks counts the source's blocks.
	Blocks   int             `json:"blocks"`
	Versions []EmailLanguage `json:"versions"`
	// Readers counts the recipients, when the audience is known.
	Readers *int64 `json:"readers,omitempty"`
	// SourceChange is the latest source save that changed blocks: the
	// toast after a source edit offers to keep the versions up to date.
	SourceChange *LanguageSourceChange `json:"sourceChange,omitempty"`
	CanEdit      bool                  `json:"canEdit"`
	EditHelpers  []string              `json:"editHelpers"`
	// CanManageWorkspace is workspace:manage (Add a workspace language).
	CanManageWorkspace bool     `json:"canManageWorkspace"`
	WorkspaceHelpers   []string `json:"workspaceHelpers"`
}

// LanguageSourceChange is a source save that made versions outdated.
type LanguageSourceChange struct {
	Version int       `json:"version"`
	By      string    `json:"by"`
	At      time.Time `json:"at"`
	// Blocks are the keys of the blocks it changed; Outdated the languages
	// it made outdated.
	Blocks   []string `json:"blocks"`
	Outdated []string `json:"outdated"`
}

// EmailLanguage is one language of the workspace for an email: its version
// and status, or Missing.
type EmailLanguage struct {
	Language string `json:"language"`
	Name     string `json:"name"`
	Native   string `json:"native"`
	Status   string `json:"status" enum:"source,up_to_date,outdated,draft,missing"`
	// Offered is false for a version in a language the workspace removed:
	// it is kept but not sent.
	Offered bool `json:"offered"`
	// Held: its readers get the fallback instead ("Send English to them").
	Held bool `json:"held"`
	// Attention counts what is outdated: blocks changed in, new in or
	// removed from the source, and a changed subject or preheader.
	Attention        int  `json:"attention"`
	Changed          int  `json:"changed"`
	Blocks           int  `json:"blocks"`
	UpToDate         int  `json:"upToDate"`
	Translated       int  `json:"translated"`
	Still            int  `json:"still"`
	SubjectChanged   bool `json:"subjectChanged"`
	PreheaderChanged bool `json:"preheaderChanged"`
	// Readers are the recipients whose language this is, plus, on the
	// version the others get, those with no language (NoLanguage) or a
	// language the workspace doesn't offer (NotOffered).
	Readers    *int64 `json:"readers,omitempty"`
	NoLanguage int64  `json:"noLanguage"`
	NotOffered int64  `json:"notOffered"`
	// Gets is the version its readers get: itself, or the fallback.
	Gets           string     `json:"gets"`
	TranslatedBy   string     `json:"translatedBy"`
	TranslatedAt   *time.Time `json:"translatedAt,omitempty"`
	TranslatedFrom int        `json:"translatedFrom"`
	Version        int        `json:"version"`
	UpdatedBy      string     `json:"updatedBy"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty"`
	Subject        string     `json:"subject"`
	Preheader      string     `json:"preheader"`
	// ChangedBlocks names the source blocks changed since translation, and
	// LastChange is the latest of those changes.
	ChangedBlocks []string     `json:"changedBlocks"`
	LastChange    *BlockChange `json:"lastChange,omitempty"`
	// StillBlocks are the indexes of a draft's blocks that still say what
	// the source says; ChangedIndexes the version's blocks changed in the
	// source (both for the canvas tags).
	StillIndexes   []int `json:"stillIndexes"`
	ChangedIndexes []int `json:"changedIndexes"`
}

// overview builds the switcher's list: every workspace language (and any
// version in a removed language), the source first.
func overview(st sourceState, views []view, settings Settings, readers map[string]int64, total *int64) EmailLanguages {
	out := EmailLanguages{Source: st.Language, SourceVersion: st.Version, Default: settings.Default, Fallback: settings.Fallback, Blocks: len(st.Blocks),
		Versions: []EmailLanguage{}, Readers: total, EditHelpers: []string{}, WorkspaceHelpers: []string{}}
	out.CanChangeSource = len(views) == 0
	vs := map[string]view{}
	states := []VersionState{}
	for _, v := range views {
		vs[v.row.Language] = v
		states = append(states, VersionState{Language: v.row.Language, Status: v.row.Status, Held: v.row.Held})
	}
	ch := NewChooser(st.Language, settings, states)
	// Who gets what: per contact base language.
	var noLang, notOffered int64
	byBase := map[string]int64{}
	for code, n := range readers {
		b := Base(code)
		switch {
		case b == "":
			noLang += n
		case !settings.Offers(b):
			notOffered += n
		default:
			byBase[b] += n
		}
	}
	others := ch.Pick("").Language // where readers without a language go
	langs := append([]string{}, settings.Languages...)
	for _, v := range views {
		if !settings.Offers(v.row.Language) {
			langs = append(langs, v.row.Language)
		}
	}
	// The source first, then the workspace's order.
	sort.SliceStable(langs, func(i, j int) bool { return langs[i] == st.Language && langs[j] != st.Language })
	if len(langs) == 0 || langs[0] != st.Language {
		langs = append([]string{st.Language}, langs...)
	}
	seen := map[string]bool{}
	for _, lang := range langs {
		if seen[lang] {
			continue
		}
		seen[lang] = true
		l, _ := Lookup(lang)
		e := EmailLanguage{Language: lang, Name: l.Name, Native: l.Native, Offered: settings.Offers(lang), ChangedBlocks: []string{}, StillIndexes: []int{}, ChangedIndexes: []int{}}
		if e.Name == "" {
			e.Name = lang
		}
		choice := ch.Pick(lang)
		e.Gets = choice.Language
		switch {
		case lang == st.Language:
			e.Status, e.Blocks, e.UpToDate = StatusSource, len(st.Blocks), len(st.Blocks)
			e.Gets = lang
		default:
			v, ok := vs[lang]
			if !ok {
				e.Status = StatusMissing
				break
			}
			p := v.progress
			e.Status, e.Held = p.Status, v.row.Held
			e.Attention, e.Changed, e.Blocks, e.UpToDate, e.Translated, e.Still = p.Attention, p.Changed, p.Blocks, p.UpToDate, p.Translated, p.Still
			e.SubjectChanged, e.PreheaderChanged = p.SubjectChanged, p.PreheaderChanged
			e.TranslatedBy, e.TranslatedAt, e.TranslatedFrom = v.by, v.row.TranslatedAt, int(v.row.TranslatedFrom)
			e.Version, e.UpdatedBy = int(v.row.Version), v.updater
			at := v.row.UpdatedAt
			e.UpdatedAt = &at
			e.Subject, e.Preheader = v.row.Subject, v.row.Preheader
			for _, r := range v.rows {
				switch r.State {
				case RowChanged, RowNew, RowRemoved:
					name := ""
					if r.Source != nil {
						name = r.Source.Name
					} else if r.Version != nil {
						name = r.Version.Name
					}
					if r.State == RowChanged || r.State == RowNew {
						e.ChangedBlocks = append(e.ChangedBlocks, name)
					}
					if r.Changed != nil && (e.LastChange == nil || r.Changed.At.After(e.LastChange.At)) {
						c := *r.Changed
						e.LastChange = &c
					}
					if r.State == RowChanged && r.VersionIndex >= 0 {
						e.ChangedIndexes = append(e.ChangedIndexes, r.VersionIndex)
					}
				case RowStill:
					e.StillIndexes = append(e.StillIndexes, r.VersionIndex)
				}
			}
		}
		if total != nil {
			n := byBase[lang]
			if e.Gets == others && lang == others {
				e.NoLanguage, e.NotOffered = noLang, notOffered
				n += noLang + notOffered
			}
			e.Readers = &n
		}
		out.Versions = append(out.Versions, e)
	}
	// The latest source save that changed blocks, and what it made outdated.
	var latest *BlockChange
	for _, k := range st.Keys {
		if k.Changed != nil && k.Changed.Version == st.Version && (latest == nil || k.Changed.At.After(latest.At)) {
			c := *k.Changed
			latest = &c
		}
	}
	if latest != nil && len(views) > 0 {
		sc := &LanguageSourceChange{Version: latest.Version, By: latest.By, At: latest.At, Blocks: []string{}, Outdated: []string{}}
		for _, k := range st.Keys {
			if k.Changed != nil && k.Changed.Version == latest.Version {
				sc.Blocks = append(sc.Blocks, k.Key)
			}
		}
		for _, v := range views {
			for _, r := range v.rows {
				if (r.State == RowChanged || r.State == RowNew) && contains(sc.Blocks, r.Key) && v.row.Status == storedTranslated {
					sc.Outdated = append(sc.Outdated, v.row.Language)
					break
				}
			}
		}
		if len(sc.Outdated) > 0 {
			out.SourceChange = sc
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
