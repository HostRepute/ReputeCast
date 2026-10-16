package languages

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// ReadinessFacts is what Send Readiness checks about an email's languages.
type ReadinessFacts struct {
	// Applies is false for a workspace with one language and an email
	// without versions: the Languages group isn't shown.
	Applies       bool
	Source        string
	SourceVersion int
	Fallback      string
	// Languages are the workspace's languages (and versions in removed
	// ones), the source first.
	Languages []ReadinessLanguage
	// NoLanguage and NotOffered count the recipients with no language and
	// with a language the workspace doesn't offer; NotOfferedCodes are those
	// codes.
	NoLanguage      int64
	NotOffered      int64
	NotOfferedCodes []string
	// OthersGet is the version readers with no language (or one the
	// workspace doesn't offer) get.
	OthersGet string
	// Recipients is the audience's total.
	Recipients int64
	// Versions are the versions that are sent besides the source, with their
	// content (Send Readiness checks every version).
	Versions []ReadinessVersion
}

// ReadinessLanguage is one language's state for a send.
type ReadinessLanguage struct {
	Language string
	Name     string
	Status   string // source, up_to_date, outdated, draft, missing
	Held     bool
	Offered  bool
	// Readers are the recipients with this language; Gets is the version
	// they get.
	Readers int64
	Gets    string
	// Attention counts outdated blocks; Changed names the source blocks
	// that changed since translation.
	Attention      int
	Changed        []string
	SubjectChanged bool
	TranslatedFrom int
	Blocks         int
	Translated     int
	PageCopy       string // live, incomplete, not_set_up
}

// ReadinessVersion is the content of a version that is sent.
type ReadinessVersion struct {
	Language  string
	Subject   string
	Preheader string
	MJML      string
	HTML      string
}

// CampaignReadiness returns the language facts of a campaign for Send
// Readiness. recipients counts the final audience per contact language code.
func (s *Service) CampaignReadiness(ctx context.Context, ws *store.Workspace, id uuid.UUID, recipients map[string]int64) (ReadinessFacts, error) {
	var out ReadinessFacts
	var st sourceState
	var views []view
	var settings Settings
	var rows []store.LanguagesListRow
	err := s.campaignTx(ctx, ws, id, false, func(q *store.Queries, o Owner, _ Source, _ store.LanguagesCampaignSourceRow, state sourceState, set Settings) error {
		var err error
		if views, err = s.versions(ctx, q, o, state); err != nil {
			return err
		}
		rows, err = q.LanguagesList(ctx, ws.ID)
		st, settings = state, set
		return err
	})
	if err != nil {
		return out, err
	}
	pageCopy := map[string]string{}
	for _, r := range rows {
		pageCopy[r.Code] = r.PageCopyStatus
	}
	out = readinessFacts(st, views, settings, recipients, pageCopy)
	return out, nil
}

func readinessFacts(st sourceState, views []view, settings Settings, recipients map[string]int64, pageCopy map[string]string) ReadinessFacts {
	total := int64(0)
	for _, n := range recipients {
		total += n
	}
	ov := overview(st, views, settings, recipients, &total)
	out := ReadinessFacts{Source: st.Language, SourceVersion: st.Version, Fallback: settings.Fallback, Recipients: total, NotOfferedCodes: []string{},
		Applies: len(settings.Languages) > 1 || len(views) > 0}
	byLang := map[string]view{}
	states := []VersionState{}
	for _, v := range views {
		byLang[v.row.Language] = v
		states = append(states, VersionState{Language: v.row.Language, Status: v.row.Status, Held: v.row.Held})
	}
	out.OthersGet = NewChooser(st.Language, settings, states).Pick("").Language
	others := map[string]int64{}
	for code, n := range recipients {
		b := Base(code)
		switch {
		case b == "":
			out.NoLanguage += n
		case !settings.Offers(b):
			out.NotOffered += n
			norm, ok := Normalize(code)
			if !ok {
				norm = code
			}
			others[norm] += n
		}
	}
	for code := range others {
		out.NotOfferedCodes = append(out.NotOfferedCodes, code)
	}
	sort.Slice(out.NotOfferedCodes, func(i, j int) bool {
		a, b := others[out.NotOfferedCodes[i]], others[out.NotOfferedCodes[j]]
		if a != b {
			return a > b
		}
		return out.NotOfferedCodes[i] < out.NotOfferedCodes[j]
	})
	for _, e := range ov.Versions {
		l := ReadinessLanguage{Language: e.Language, Name: e.Name, Status: e.Status, Held: e.Held, Offered: e.Offered, Gets: e.Gets, Attention: e.Attention,
			Changed: e.ChangedBlocks, SubjectChanged: e.SubjectChanged, TranslatedFrom: e.TranslatedFrom, Blocks: e.Blocks, Translated: e.Translated,
			PageCopy: pageCopy[e.Language]}
		if e.Readers != nil {
			l.Readers = *e.Readers - e.NoLanguage - e.NotOffered
		}
		out.Languages = append(out.Languages, l)
		if !e.Offered {
			continue
		}
		if e.Status == StatusSource {
			continue
		}
		if v, ok := byLang[e.Language]; ok && v.row.Status == storedTranslated && !v.row.Held {
			out.Versions = append(out.Versions, ReadinessVersion{Language: e.Language, Subject: v.row.Subject, Preheader: v.row.Preheader, MJML: v.row.Mjml, HTML: v.row.Html})
		}
	}
	return out
}
