package visibility

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/store"
)

// Names resolves what rules refer to, for words and broken conditions.
type Names struct {
	Lang     i18n.Lang
	Lists    map[string]string
	Segments map[string]string
	Fields   map[string]NamedField
	// Deleted are the segments, lists and fields the workspace deleted,
	// by Ref (who and when, from the audit log). LoadDeletions fills it.
	Deleted map[Ref]Deletion
}

// NamedField is a custom field.
type NamedField struct {
	Label   string
	Type    string
	Options []string
}

// Deletion is who deleted a segment, list or field, and when.
type Deletion struct {
	Name string
	By   string
	At   time.Time
}

func (n *Names) fieldType(key string) string {
	if n == nil {
		return "text"
	}
	if f, ok := n.Fields[key]; ok {
		return f.Type
	}
	return "text"
}

// LoadNames reads the workspace's lists, segments and fields.
func LoadNames(ctx context.Context, q *store.Queries, wsID uuid.UUID) (*Names, error) {
	n := &Names{Lang: i18n.LangFrom(ctx), Lists: map[string]string{}, Segments: map[string]string{}, Fields: map[string]NamedField{}, Deleted: map[Ref]Deletion{}}
	lists, err := q.VisibilityLists(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for _, l := range lists {
		n.Lists[l.ID.String()] = l.Name
	}
	segs, err := q.VisibilitySegments(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for _, s := range segs {
		n.Segments[s.ID.String()] = s.Name
	}
	fields, err := q.VisibilityFields(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for _, f := range fields {
		var opts []string
		var raw []any
		if json.Unmarshal(f.Options, &raw) == nil {
			for _, o := range raw {
				switch v := o.(type) {
				case string:
					opts = append(opts, v)
				case map[string]any:
					if s, ok := v["value"].(string); ok {
						opts = append(opts, s)
					}
				}
			}
		}
		n.Fields[f.Key] = NamedField{Label: f.Label, Type: f.Type, Options: opts}
	}
	return n, nil
}

// LoadDeletions finds who deleted what the rules refer to and is gone.
func (n *Names) LoadDeletions(ctx context.Context, q *store.Queries, wsID uuid.UUID, refs []Ref) error {
	ids, keys := []string{}, []string{}
	for _, r := range refs {
		if n.Exists(r) {
			continue
		}
		if r.Kind == "field" {
			keys = append(keys, r.ID)
		} else if r.ID != "" {
			ids = append(ids, r.ID)
		}
	}
	if len(ids)+len(keys) == 0 {
		return nil
	}
	rows, err := q.VisibilityDeletions(ctx, store.VisibilityDeletionsParams{WorkspaceID: &wsID, Ids: ids, Names: keys})
	if err != nil {
		return err
	}
	for _, r := range rows {
		switch r.Action {
		case "segment.delete":
			n.Deleted[Ref{Kind: "segment", ID: r.TargetID}] = Deletion{Name: r.TargetName, By: r.ActorName, At: r.OccurredAt}
		case "list.delete":
			n.Deleted[Ref{Kind: "list", ID: r.TargetID}] = Deletion{Name: r.TargetName, By: r.ActorName, At: r.OccurredAt}
		case "field.delete":
			n.Deleted[Ref{Kind: "field", ID: r.TargetName}] = Deletion{Name: r.TargetName, By: r.ActorName, At: r.OccurredAt}
		}
	}
	return nil
}

// Exists reports whether what a condition refers to is still there. Tags
// always exist: a tag nobody has yet simply matches nobody.
func (n *Names) Exists(r Ref) bool {
	switch r.Kind {
	case "segment":
		_, ok := n.Segments[r.ID]
		return ok
	case "list":
		_, ok := n.Lists[r.ID]
		return ok
	case "field":
		_, ok := n.Fields[r.ID]
		return ok
	}
	return true
}

func (n *Names) list(id string, fallback string) string {
	if v, ok := n.Lists[id]; ok {
		return v
	}
	if d, ok := n.Deleted[Ref{Kind: "list", ID: id}]; ok && d.Name != "" {
		return d.Name
	}
	return fallback
}

func (n *Names) segment(id string, fallback string) string {
	if v, ok := n.Segments[id]; ok {
		return v
	}
	if d, ok := n.Deleted[Ref{Kind: "segment", ID: id}]; ok && d.Name != "" {
		return d.Name
	}
	return fallback
}

// tierName is a tier in words: "At risk".
func tierName(l i18n.Lang, key string) string {
	switch key {
	case "active":
		return mTierActive.Text(l, nil)
	case "cooling":
		return mTierCooling.Text(l, nil)
	case "at_risk":
		return mTierAtRisk.Text(l, nil)
	case "sunset":
		return mTierSunset.Text(l, nil)
	}
	return strings.ReplaceAll(key, "_", " ")
}

// languageName is a content language's name in the reader's language:
// "Spanish", "Tiếng Tây Ban Nha" (CLDR's names, as the browser writes them
// on the web). Languages a dictionary doesn't name keep their English name.
func languageName(l i18n.Lang, code string) string {
	if code == "" {
		return ""
	}
	base := baseLanguage(code)
	if d := languageNames(l.Locale); d != nil {
		if n := d.Name(language.Make(base)); n != "" {
			return n
		}
	}
	return languages.NameOf(base)
}

// languageNames is the dictionary of language names in an app language;
// nil for English, which uses the content languages' own catalog.
func languageNames(l i18n.Locale) display.Namer {
	if l == i18n.Vietnamese {
		return display.Vietnamese.Languages()
	}
	return nil
}
