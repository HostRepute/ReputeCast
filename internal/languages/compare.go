package languages

import "time"

// LanguageCompare is the compare view: the source and one language version
// side by side, block by block, with the subject and preheader first.
type LanguageCompare struct {
	Source   string `json:"source"`
	Language string `json:"language"`
	// SourceVersion is the source's current version; TranslatedFrom the
	// source version the language version was translated from.
	SourceVersion  int          `json:"sourceVersion"`
	TranslatedFrom int          `json:"translatedFrom"`
	TranslatedBy   string       `json:"translatedBy"`
	TranslatedAt   *time.Time   `json:"translatedAt,omitempty"`
	Status         string       `json:"status" enum:"up_to_date,outdated,draft"`
	Blocks         int          `json:"blocks"`
	UpToDate       int          `json:"upToDate"`
	Attention      int          `json:"attention"`
	Translated     int          `json:"translated"`
	Version        int          `json:"version"`
	Subject        CompareText  `json:"subject"`
	Preheader      CompareText  `json:"preheader"`
	Rows           []CompareRow `json:"rows"`
}

// CompareText is the subject or preheader row.
type CompareText struct {
	Source  string `json:"source"`
	Version string `json:"version"`
	State   string `json:"state" enum:"up_to_date,changed"`
	// Before is the source text the version was translated from.
	Before string `json:"before"`
}

// CompareRow is one block of the compare view.
type CompareRow struct {
	// Number is the source block's position (1-based); 0 for a block removed
	// from the source or only in the language version.
	Number int    `json:"number"`
	Key    string `json:"key"`
	State  string `json:"state" enum:"up_to_date,changed,still,new,removed,only"`
	// Source is the source block now; nil when it was removed or never existed.
	Source *CompareBlock `json:"source,omitempty"`
	// Version is the language version's block; nil for a block new in the source.
	Version *CompareBlock `json:"version,omitempty"`
	// Before is the source text the version was translated from (the diff's
	// old side), from source version FromVersion.
	Before      string       `json:"before"`
	FromVersion int          `json:"fromVersion"`
	Changed     *BlockChange `json:"changed,omitempty"`
}

// CompareBlock is one side of a row.
type CompareBlock struct {
	Index    int            `json:"index"`
	Name     string         `json:"name"`
	Text     string         `json:"text"`
	Segments []BlockSegment `json:"segments"`
}

func compareBlock(b *Block, index int) *CompareBlock {
	if b == nil {
		return nil
	}
	return &CompareBlock{Index: index, Name: b.Name, Text: b.Text, Segments: b.Segments}
}

// compareView builds the compare view of a language version.
func compareView(st sourceState, src Source, v view) LanguageCompare {
	p := v.progress
	out := LanguageCompare{Source: st.Language, Language: v.row.Language, SourceVersion: st.Version, TranslatedFrom: int(v.row.TranslatedFrom), TranslatedBy: v.by,
		TranslatedAt: v.row.TranslatedAt, Status: p.Status, Blocks: p.Blocks, UpToDate: p.UpToDate, Attention: p.Attention, Translated: p.Translated,
		Version: int(v.row.Version), Rows: []CompareRow{}}
	out.Subject = CompareText{Source: src.Subject, Version: v.row.Subject, State: RowUpToDate}
	if v.links.Subject != nil {
		out.Subject.Before = v.links.Subject.Text
	}
	if p.SubjectChanged {
		out.Subject.State = RowChanged
	}
	out.Preheader = CompareText{Source: src.Preheader, Version: v.row.Preheader, State: RowUpToDate}
	if v.links.Preheader != nil {
		out.Preheader.Before = v.links.Preheader.Text
	}
	if p.PreheaderChanged {
		out.Preheader.State = RowChanged
	}
	for _, r := range v.rows {
		row := CompareRow{Number: r.Number, Key: r.Key, State: r.State, Before: r.Before, FromVersion: r.FromVersion, Changed: r.Changed}
		if r.Source != nil {
			row.Source = compareBlock(r.Source, r.Number-1)
		}
		if r.Version != nil {
			row.Version = compareBlock(r.Version, r.VersionIndex)
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}
