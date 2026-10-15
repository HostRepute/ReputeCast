package languages

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// ReportLanguages is "Results per language" of a campaign sent in more
// than one language. It holds aggregate numbers only (the report is shared
// read-only).
type ReportLanguages struct {
	// Languages counts the languages sent; Versions the rows (readers who
	// got the fallback have their own).
	Languages int                 `json:"languages"`
	Versions  int                 `json:"versions"`
	Rows      []ReportLanguageRow `json:"rows"`
	// Sent is the headline's breakdown by version ("English 27,396 ·
	// Spanish 2,740 · German 1,468").
	Sent []ReportLanguageCount `json:"sent"`
	// Fallback compares the readers who got the fallback with the version's
	// own readers ("their human-open rate is 8.3 points below").
	Fallback *ReportLanguageFallback `json:"fallback,omitempty"`
	// ClickMaps are the versions the click map can show, the source first.
	ClickMaps []ReportLanguageOption `json:"clickMaps"`
}

// ReportLanguageRow is one version and who got it.
type ReportLanguageRow struct {
	Language string `json:"language"`
	Name     string `json:"name"`
	// Fallback marks readers who got this version because theirs wasn't
	// ready; Missing is their language.
	Fallback bool   `json:"fallback"`
	Reason   string `json:"reason,omitempty" enum:"missing,draft,held"`
	Missing  string `json:"missing,omitempty"`
	// Readers names who got it: their languages, "no language" and "other
	// languages" (Kinds holds the same as codes: a language code,
	// "no_language" or "not_offered").
	Readers []string `json:"readers"`
	Kinds   []string `json:"kinds"`
	Source  bool     `json:"source"`
	// TranslatedFrom is the source version the version was translated from.
	TranslatedFrom  int      `json:"translatedFrom,omitempty"`
	Sent            int64    `json:"sent"`
	Delivered       int64    `json:"delivered"`
	DeliveredRate   *float64 `json:"deliveredRate,omitempty"`
	HumanOpens      int64    `json:"humanOpens"`
	HumanOpenRate   *float64 `json:"humanOpenRate,omitempty"`
	Clicks          int64    `json:"clicks"`
	ClickRate       *float64 `json:"clickRate,omitempty"`
	Unsubscribes    int64    `json:"unsubscribes"`
	UnsubscribeRate *float64 `json:"unsubscribeRate,omitempty"`
	Complaints      int64    `json:"complaints"`
	ComplaintRate   *float64 `json:"complaintRate,omitempty"`
}

// ReportLanguageCount is a version's share of the send.
type ReportLanguageCount struct {
	Language string `json:"language"`
	Name     string `json:"name"`
	Count    int64  `json:"count"`
}

// ReportLanguageFallback is the note under the table.
type ReportLanguageFallback struct {
	// Readers are the languages whose readers got the fallback ("vi").
	Readers []string `json:"readers"`
	Got     string   `json:"got"`
	// Gap is how many points their human-open rate is below the version's
	// own readers' (negative when above).
	Gap *float64 `json:"gap,omitempty"`
	// Draft is the newest unsent campaign to the same audience, to add a
	// version in their language to.
	Draft *ReportLanguageDraft `json:"draft,omitempty"`
}

// ReportLanguageDraft names a campaign.
type ReportLanguageDraft struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ReportLanguageOption is one version of the click map's language picker.
type ReportLanguageOption struct {
	Language string `json:"language"`
	Name     string `json:"name"`
}

// sendSnapshot is what reports read of campaign_sends.content.
type sendSnapshot struct {
	Source    string `json:"source"`
	HTML      string `json:"html"`
	Languages []struct {
		Language       string `json:"language"`
		HTML           string `json:"html"`
		TranslatedFrom int    `json:"translated_from"`
	} `json:"languages"`
	Links map[string]int `json:"links"`
}

func rateOf(n, d int64) *float64 {
	if d <= 0 {
		return nil
	}
	v := math.Round(float64(n)/float64(d)*10000) / 100
	return &v
}

// CampaignResults returns a sent campaign's results per language, or nil
// when it went out in one language.
func (s *Service) CampaignResults(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, audience json.RawMessage) (*ReportLanguages, error) {
	rows, err := s.Q.LanguagesMessageResults(ctx, &campaignID)
	if err != nil {
		return nil, err
	}
	langs := map[string]bool{}
	for _, r := range rows {
		if r.Language != "" && r.Sent > 0 {
			langs[r.Language] = true
		}
	}
	if len(langs) < 2 {
		return nil, nil
	}
	var snap sendSnapshot
	if raw, err := s.Q.LanguagesCampaignSendContent(ctx, campaignID); err == nil {
		_ = json.Unmarshal(raw, &snap)
	}
	from := map[string]int{}
	for _, l := range snap.Languages {
		from[l.Language] = l.TranslatedFrom
	}
	type acc struct {
		row   ReportLanguageRow
		kinds map[string]int64
	}
	main := map[string]*acc{}
	fallback := map[string]*acc{}
	get := func(m map[string]*acc, lang string) *acc {
		a, ok := m[lang]
		if !ok {
			a = &acc{row: ReportLanguageRow{Language: lang, Name: NameOf(lang), Readers: []string{}, Kinds: []string{}}, kinds: map[string]int64{}}
			m[lang] = a
		}
		return a
	}
	for _, r := range rows {
		if r.Language == "" {
			continue
		}
		var a *acc
		kind := r.ContactLanguage
		switch r.LanguageReason {
		case ReasonMissing, ReasonDraft, ReasonHeld:
			a = get(fallback, r.Language)
			a.row.Fallback, a.row.Reason, a.row.Missing = true, r.LanguageReason, Base(r.ContactLanguage)
		default:
			a = get(main, r.Language)
			switch r.LanguageReason {
			case ReasonNoLanguage:
				kind = "no_language"
			case ReasonNotOffered:
				kind = "not_offered"
			}
		}
		if kind == "" {
			kind = r.Language
		}
		a.kinds[Base(kind)] += r.Sent
		a.row.Sent += r.Sent
		a.row.Delivered += r.Delivered
		a.row.HumanOpens += r.HumanOpens
		a.row.Clicks += r.Clicks
		a.row.Unsubscribes += r.Unsubscribes
		a.row.Complaints += r.Complaints
	}
	finish := func(a *acc) ReportLanguageRow {
		row := a.row
		row.Source = row.Language == snap.Source
		if !row.Source {
			row.TranslatedFrom = from[row.Language]
		}
		kinds := make([]string, 0, len(a.kinds))
		for k := range a.kinds {
			kinds = append(kinds, k)
		}
		// Their own language first, then the others by readers, "no
		// language" and "other languages" last.
		weight := func(k string) int {
			switch {
			case k == row.Language:
				return 0
			case k == "no_language":
				return 2
			case k == "not_offered":
				return 3
			}
			return 1
		}
		sort.Slice(kinds, func(i, j int) bool {
			if weight(kinds[i]) != weight(kinds[j]) {
				return weight(kinds[i]) < weight(kinds[j])
			}
			return a.kinds[kinds[i]] > a.kinds[kinds[j]]
		})
		for _, k := range kinds {
			row.Kinds = append(row.Kinds, k)
			switch k {
			case "no_language":
				row.Readers = append(row.Readers, "no language")
			case "not_offered":
				row.Readers = append(row.Readers, "other languages")
			default:
				row.Readers = append(row.Readers, NameOf(k))
			}
		}
		row.DeliveredRate = rateOf(row.Delivered, row.Sent)
		row.HumanOpenRate = rateOf(row.HumanOpens, row.Delivered)
		row.ClickRate = rateOf(row.Clicks, row.Delivered)
		row.UnsubscribeRate = rateOf(row.Unsubscribes, row.Delivered)
		row.ComplaintRate = rateOf(row.Complaints, row.Delivered)
		return row
	}
	out := &ReportLanguages{Rows: []ReportLanguageRow{}, Sent: []ReportLanguageCount{}, ClickMaps: []ReportLanguageOption{}}
	order := func(m map[string]*acc) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if (keys[i] == snap.Source) != (keys[j] == snap.Source) {
				return keys[i] == snap.Source
			}
			return m[keys[i]].row.Sent > m[keys[j]].row.Sent
		})
		return keys
	}
	sent := map[string]int64{}
	for _, k := range order(main) {
		row := finish(main[k])
		out.Rows = append(out.Rows, row)
		sent[k] += row.Sent
	}
	var fbRows []ReportLanguageRow
	for _, k := range order(fallback) {
		row := finish(fallback[k])
		fbRows = append(fbRows, row)
		out.Rows = append(out.Rows, row)
		sent[k] += row.Sent
	}
	for _, k := range order(main) {
		out.Sent = append(out.Sent, ReportLanguageCount{Language: k, Name: NameOf(k), Count: sent[k]})
		out.ClickMaps = append(out.ClickMaps, ReportLanguageOption{Language: k, Name: NameOf(k)})
	}
	out.Languages, out.Versions = len(langs), len(out.Rows)
	if len(fbRows) > 0 {
		fb := fbRows[0]
		note := &ReportLanguageFallback{Readers: []string{}, Got: fb.Language}
		for _, k := range fb.Kinds {
			if k != "no_language" && k != "not_offered" {
				note.Readers = append(note.Readers, k)
			}
		}
		if own, ok := main[fb.Language]; ok {
			o := finish(own)
			if o.HumanOpenRate != nil && fb.HumanOpenRate != nil {
				g := math.Round((*o.HumanOpenRate-*fb.HumanOpenRate)*10) / 10
				note.Gap = &g
			}
		}
		if len(audience) > 0 {
			d, err := s.Q.LanguagesLatestDraftFor(ctx, store.LanguagesLatestDraftForParams{WorkspaceID: ws.ID, ExceptID: campaignID, Audience: audience})
			if err == nil {
				note.Draft = &ReportLanguageDraft{ID: d.ID.String(), Name: d.Name}
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		}
		out.Fallback = note
	}
	return out, nil
}

// ClickMapVersion returns the HTML a language version was sent with and
// the range of its link positions in campaign_links (from, to); ok is false
// when the send didn't capture that version.
func (s *Service) ClickMapVersion(ctx context.Context, campaignID uuid.UUID, lang string) (html string, from, to int, ok bool) {
	raw, err := s.Q.LanguagesCampaignSendContent(ctx, campaignID)
	if err != nil {
		return "", 0, 0, false
	}
	var snap sendSnapshot
	if json.Unmarshal(raw, &snap) != nil {
		return "", 0, 0, false
	}
	start := 0
	switch {
	case lang == "" || lang == snap.Source:
		html = snap.HTML
	default:
		for _, l := range snap.Languages {
			if l.Language == lang && l.HTML != "" {
				html = l.HTML
			}
		}
		off, has := snap.Links[lang+"|"]
		if html == "" || !has {
			return "", 0, 0, false
		}
		start = off
	}
	// The range ends where the next language version's links start (the
	// source keeps its A/B variants' links).
	end := math.MaxInt32
	for k, off := range snap.Links {
		if off <= start || ((lang == "" || lang == snap.Source) && !strings.Contains(k, "|")) {
			continue
		}
		end = min(end, off)
	}
	return html, start, end, true
}
