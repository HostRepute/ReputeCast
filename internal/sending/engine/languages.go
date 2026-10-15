package engine

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/media"
)

// LanguageVersions supplies an email's language versions when a send
// starts (internal/languages).
type LanguageVersions interface {
	CampaignSendSet(ctx context.Context, workspaceID, campaignID uuid.UUID) (languages.SendSet, error)
}

// langContent is one language version captured at send start.
type langContent struct {
	Language       string      `json:"language"`
	Status         string      `json:"status"`
	Held           bool        `json:"held,omitempty"`
	Subject        string      `json:"subject,omitempty"`
	Preheader      string      `json:"preheader,omitempty"`
	HTML           string      `json:"html,omitempty"`
	Text           string      `json:"text,omitempty"`
	Variants       []abVariant `json:"variants,omitempty"`
	TranslatedFrom int         `json:"translated_from,omitempty"`
	Version        int         `json:"version,omitempty"`
}

// sendLanguages adds the campaign's language versions to its snapshot.
func (e *Engine) sendLanguages(ctx context.Context, wsID, campaignID uuid.UUID, c *sendContent) error {
	if e.Languages == nil {
		return nil
	}
	set, err := e.Languages.CampaignSendSet(ctx, wsID, campaignID)
	if err != nil {
		return err
	}
	base := e.URLs.Base()
	c.Source, c.DefaultLanguage, c.Fallback, c.Offered = set.Source, set.Default, set.Fallback, set.Offered
	for _, v := range set.Versions {
		l := langContent{Language: v.Language, Status: v.Status, Held: v.Held, Subject: v.Subject, Preheader: v.Preheader, Text: v.Text,
			TranslatedFrom: v.TranslatedFrom, Version: v.Version}
		if v.HTML != "" {
			l.HTML = media.Absolutize(v.HTML, base)
		}
		if ab := abTestOf(v.ABTest); ab != nil {
			for _, av := range ab.Variants {
				if av.HTML != "" {
					av.HTML = media.Absolutize(av.HTML, base)
				}
				l.Variants = append(l.Variants, av)
			}
		}
		c.Languages = append(c.Languages, l)
	}
	return nil
}

// chooser picks each recipient's version; ok is false for a send without
// content languages.
func (c sendContent) chooser() (languages.Chooser, bool) {
	if c.Source == "" {
		return languages.Chooser{}, false
	}
	states := make([]languages.VersionState, 0, len(c.Languages))
	for _, l := range c.Languages {
		states = append(states, languages.VersionState{Language: l.Language, Status: l.Status, Held: l.Held})
	}
	return languages.NewChooser(c.Source, languages.NewSettings(c.DefaultLanguage, c.Fallback, c.Offered), states), true
}

// translation returns a language version that is sent, or nil for the
// source (and for languages without a ready version).
func (c sendContent) translation(lang string) *langContent {
	if lang == "" || lang == c.Source {
		return nil
	}
	for i := range c.Languages {
		if c.Languages[i].Language == lang && c.Languages[i].HTML+c.Languages[i].Text != "" {
			return &c.Languages[i]
		}
	}
	return nil
}

// readyLanguages are the translations a send renders.
func (c sendContent) readyLanguages() []string {
	out := []string{}
	for _, l := range c.Languages {
		if c.translation(l.Language) != nil {
			out = append(out, l.Language)
		}
	}
	return out
}

// langKey is the render key of a language version's variant ("de|B");
// the source's keys are its variant keys.
func langKey(lang, variant string) string { return lang + "|" + variant }

func splitKey(key string) (lang, variant string) {
	if i := strings.IndexByte(key, '|'); i >= 0 {
		return key[:i], key[i+1:]
	}
	return "", key
}

// renderKey is the template a message renders: its language version's
// variant, or the source's.
func renderKey(c sendContent, lang, variant string) string {
	if c.translation(lang) != nil {
		return langKey(lang, variant)
	}
	return variant
}

// forLanguage is the content of a language version's variant. Variant
// fields the version doesn't set come from the version itself, never from
// the source (a reader never gets a mix).
func (l langContent) forVariant(variant string) Content {
	out := Content{Subject: l.Subject, Preheader: l.Preheader, HTML: l.HTML, Text: l.Text}
	if variant == "" || variant == "A" {
		return out
	}
	for _, v := range l.Variants {
		if v.Key != variant {
			continue
		}
		if v.Subject != "" {
			out.Subject = v.Subject
		}
		if v.Preheader != "" {
			out.Preheader = v.Preheader
		}
		if v.HTML != "" {
			out.HTML, out.Text = v.HTML, ""
		}
	}
	return out
}

// languageContentKey is the link-set key of a message: the language
// version's content variant, or the source's.
func languageContentKey(c sendContent, lang, variant string) string {
	l := c.translation(lang)
	if l == nil {
		return contentKey(c, variant)
	}
	for _, v := range l.Variants {
		if v.Key == variant && v.HTML != "" && v.Key != "A" {
			return langKey(lang, v.Key)
		}
	}
	return langKey(lang, "")
}

// languageVariantKeys lists each language version's keys with their own
// content (links are numbered per key).
func languageVariantKeys(c sendContent) []string {
	keys := []string{}
	for _, lang := range c.readyLanguages() {
		keys = append(keys, langKey(lang, ""))
		for _, v := range c.translation(lang).Variants {
			if v.HTML != "" && v.Key != "A" {
				keys = append(keys, langKey(lang, v.Key))
			}
		}
	}
	return keys
}

// messageLanguage is what the messages row records for one recipient.
type messageLanguage struct {
	Language, Contact, Reason string
}

func (c sendContent) pick(ch languages.Chooser, ok bool, contactLanguage string) messageLanguage {
	if !ok {
		return messageLanguage{}
	}
	p := ch.Pick(contactLanguage)
	return messageLanguage{Language: p.Language, Contact: p.Contact, Reason: p.Reason}
}
