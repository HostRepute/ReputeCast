package visibility

import (
	"context"

	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/scoring"
	"github.com/hostrepute/reputecast/internal/store"
)

// VisibilityCatalogKind is one condition of the picker. The app names
// groups, kinds and operators by their keys.
type VisibilityCatalogKind struct {
	Key   string     `json:"key"`
	Group string     `json:"group"`
	Ops   []Operator `json:"ops"`
}

// VisibilityOption is a value a condition can take, with how many contacts
// have it.
type VisibilityOption struct {
	Value    string `json:"value"`
	Label    string `json:"label"`
	Contacts *int64 `json:"contacts,omitempty"`
}

// VisibilityCatalogField is a custom field with its operators.
type VisibilityCatalogField struct {
	Key     string     `json:"key"`
	Label   string     `json:"label"`
	Type    string     `json:"type"`
	Ops     []Operator `json:"ops"`
	Options []string   `json:"options"`
}

// VisibilityCatalogScore is how the workspace's scores fall into bands.
type VisibilityCatalogScore struct {
	// Configured is false while the workspace has no scoring: nobody has a
	// score or a band yet.
	Configured bool `json:"configured"`
	WarmFrom   int  `json:"warmFrom"`
	HotFrom    int  `json:"hotFrom"`
}

// VisibilityCatalog is everything the condition picker offers.
type VisibilityCatalog struct {
	// Groups in the picker's order; kinds in each group's order.
	Groups    []string                 `json:"groups"`
	Kinds     []VisibilityCatalogKind  `json:"kinds"`
	Segments  []VisibilityOption       `json:"segments"`
	Lists     []VisibilityOption       `json:"lists"`
	Tags      []VisibilityOption       `json:"tags"`
	Fields    []VisibilityCatalogField `json:"fields"`
	Languages []VisibilityOption       `json:"languages"`
	Tiers     []VisibilityOption       `json:"tiers"`
	Bands     []VisibilityOption       `json:"bands"`
	Score     VisibilityCatalogScore   `json:"score"`
}

// Catalog returns the condition picker's kinds and values.
func (s *Service) Catalog(ctx context.Context, ws *store.Workspace) (VisibilityCatalog, error) {
	out := VisibilityCatalog{Groups: []string{}, Kinds: []VisibilityCatalogKind{}, Segments: []VisibilityOption{}, Lists: []VisibilityOption{}, Tags: []VisibilityOption{},
		Fields: []VisibilityCatalogField{}, Languages: []VisibilityOption{}, Tiers: []VisibilityOption{}, Bands: []VisibilityOption{}}
	for _, k := range Kinds() {
		found := false
		for _, g := range out.Groups {
			if g == k.Group {
				found = true
			}
		}
		if !found {
			out.Groups = append(out.Groups, k.Group)
		}
		ops := k.Ops
		if ops == nil {
			ops = []Operator{}
		}
		out.Kinds = append(out.Kinds, VisibilityCatalogKind{Key: k.Key, Group: k.Group, Ops: ops})
	}
	lists, err := s.Q.VisibilityLists(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		n := l.Members
		out.Lists = append(out.Lists, VisibilityOption{Value: l.ID.String(), Label: l.Name, Contacts: &n})
	}
	segs, err := s.Q.VisibilitySegments(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, sg := range segs {
		n := sg.Members
		out.Segments = append(out.Segments, VisibilityOption{Value: sg.ID.String(), Label: sg.Name, Contacts: &n})
	}
	tags, err := s.Q.VisibilityTags(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, t := range tags {
		n := t.Contacts
		out.Tags = append(out.Tags, VisibilityOption{Value: t.Tag, Label: t.Tag, Contacts: &n})
	}
	names, err := LoadNames(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	fields, err := s.Q.VisibilityFields(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, f := range fields {
		opts := names.Fields[f.Key].Options
		if opts == nil {
			opts = []string{}
		}
		out.Fields = append(out.Fields, VisibilityCatalogField{Key: f.Key, Label: f.Label, Type: f.Type, Ops: FieldOps(f.Type), Options: opts})
	}
	st, err := (&languages.Service{Q: s.Q}).Settings(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	for _, code := range st.Languages {
		out.Languages = append(out.Languages, VisibilityOption{Value: code, Label: languages.NameOf(code)})
	}
	l := lang(ctx)
	for _, t := range Tiers {
		out.Tiers = append(out.Tiers, VisibilityOption{Value: t, Label: tierName(l, t)})
	}
	bands, configured, err := scoring.AppliedBands(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	out.Score = VisibilityCatalogScore{Configured: configured, WarmFrom: bands.WarmFrom, HotFrom: bands.HotFrom}
	for _, b := range Bands {
		out.Bands = append(out.Bands, VisibilityOption{Value: b, Label: bandName(l, b)})
	}
	return out, nil
}
