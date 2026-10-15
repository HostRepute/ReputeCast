package languages

// Reasons a reader gets the version they get (messages.language_reason).
const (
	// ReasonMatch: the version in their language.
	ReasonMatch = "match"
	// ReasonNoLanguage: the contact has no language.
	ReasonNoLanguage = "no_language"
	// ReasonNotOffered: the workspace doesn't offer their language.
	ReasonNotOffered = "not_offered"
	// ReasonMissing: the email has no version in their language.
	ReasonMissing = "missing"
	// ReasonDraft: the version in their language is a draft.
	ReasonDraft = "draft"
	// ReasonHeld: the version is held ("Send English to them instead").
	ReasonHeld = "held"
	// ReasonRequested: the API asked for a language (transactional sends).
	ReasonRequested = "requested"
)

// Chooser picks the version each reader gets: the version in their
// language when it is ready (Up to date or Outdated), otherwise the
// fallback language's version when it is ready, otherwise the source.
type Chooser struct {
	// Source is the language the email is written in.
	Source string
	// Fallback is the workspace's fallback language.
	Fallback string
	offered  map[string]bool
	state    map[string]string // language → ready, draft or held
}

// Choice is the version one reader gets and why.
type Choice struct {
	// Language is the version's language.
	Language string
	Reason   string
	// Contact is the reader's base language ("" when they have none).
	Contact string
}

// Fallback reports whether the reader gets another language than theirs
// because their own has no ready version.
func (c Choice) Fallback() bool {
	return c.Reason == ReasonMissing || c.Reason == ReasonDraft || c.Reason == ReasonHeld
}

// VersionState is what a chooser needs to know about one language version.
type VersionState struct {
	Language string
	// Status is the stored status: draft or translated.
	Status string
	Held   bool
}

// NewChooser builds a chooser for an email: its source language, the
// workspace's settings and the email's language versions. Versions in a
// language the workspace no longer offers are never sent.
func NewChooser(source string, st Settings, versions []VersionState) Chooser {
	c := Chooser{Source: source, Fallback: st.Fallback, offered: map[string]bool{}, state: map[string]string{}}
	for _, l := range st.Languages {
		c.offered[l] = true
	}
	if c.Source == "" {
		c.Source = st.Default
	}
	for _, v := range versions {
		if v.Language == c.Source || !c.offered[v.Language] {
			continue
		}
		switch {
		case v.Status != storedTranslated:
			c.state[v.Language] = ReasonDraft
		case v.Held:
			c.state[v.Language] = ReasonHeld
		default:
			c.state[v.Language] = "ready"
		}
	}
	return c
}

// Ready reports whether a language's version is sent to its readers.
func (c Chooser) Ready(lang string) bool {
	return lang == c.Source || c.state[lang] == "ready"
}

// Pick returns the version a reader with this contact language gets.
func (c Chooser) Pick(contactLanguage string) Choice {
	base := Base(contactLanguage)
	switch {
	case base == "":
		return c.fallback(ReasonNoLanguage, base)
	case !c.offered[base]:
		return c.fallback(ReasonNotOffered, base)
	case c.Ready(base):
		return Choice{Language: base, Reason: ReasonMatch, Contact: base}
	}
	reason := ReasonMissing
	if r := c.state[base]; r == ReasonDraft || r == ReasonHeld {
		reason = r
	}
	return c.fallback(reason, base)
}

// Request returns the version for a language asked for explicitly (a
// transactional send's `language`): that version when it is ready,
// otherwise as Pick would choose.
func (c Chooser) Request(requested string) Choice {
	base := Base(requested)
	if base != "" && c.offered[base] && c.Ready(base) {
		return Choice{Language: base, Reason: ReasonMatch, Contact: base}
	}
	ch := c.Pick(requested)
	if base != "" {
		ch.Reason = ReasonRequested
	}
	return ch
}

func (c Chooser) fallback(reason, contact string) Choice {
	lang := c.Source
	if c.Ready(c.Fallback) {
		lang = c.Fallback
	}
	return Choice{Language: lang, Reason: reason, Contact: contact}
}
