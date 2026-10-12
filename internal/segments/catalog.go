package segments

import (
	"context"
	"strings"

	"github.com/hostrepute/reputecast/internal/store"
)

// Option is a value a condition can take.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Operator is an operator a field supports.
type Operator struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// CatalogField is a field the builder can filter on.
type CatalogField struct {
	Key   string     `json:"key"`
	Label string     `json:"label"`
	Kind  string     `json:"kind" enum:"enum,list,campaign,text,number,date,boolean,period"`
	Ops   []Operator `json:"ops"`
	// Options are the values of enum, list and campaign fields.
	Options []Option `json:"options"`
}

// Catalog describes everything the builder can offer.
type Catalog struct {
	Fields []CatalogField `json:"fields"`
	// SQLEnabled is false when the database can't run SQL segments.
	SQLEnabled bool `json:"sqlEnabled"`
	// Views documents what SQL segments can read.
	Views []string `json:"views"`
}

// Catalog returns the fields, operators and values for the builder.
func (s *Service) Catalog(ctx context.Context, ws *store.Workspace) (Catalog, error) {
	lists, err := s.Q.ListWorkspaceLists(ctx, ws.ID)
	if err != nil {
		return Catalog{}, err
	}
	campaigns, err := s.Q.RecentSentCampaigns(ctx, ws.ID)
	if err != nil {
		return Catalog{}, err
	}
	sources, err := s.Q.ContactSources(ctx, ws.ID)
	if err != nil {
		return Catalog{}, err
	}
	custom, err := s.Q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return Catalog{}, err
	}
	out := Catalog{Fields: []CatalogField{}, SQLEnabled: s.SQL.Available(ctx), Views: ViewNames}
	ops := func(keys []string) []Operator {
		o := make([]Operator, 0, len(keys))
		for _, k := range keys {
			o = append(o, Operator{Key: k, Label: opLabels[k]})
		}
		return o
	}
	for _, key := range builtinOrder {
		spec := builtinFields[key]
		f := CatalogField{Key: key, Label: spec.Label, Kind: string(spec.Kind), Ops: ops(spec.Ops), Options: []Option{}}
		switch key {
		case "list":
			for _, l := range lists {
				f.Options = append(f.Options, Option{Value: l.ID.String(), Label: l.Name})
			}
		case "opened_campaign":
			for _, c := range campaigns {
				f.Options = append(f.Options, Option{Value: c.ID.String(), Label: c.Name})
			}
		case "source":
			for _, src := range sources {
				f.Options = append(f.Options, Option{Value: src.Source, Label: SourceLabel(src.Source)})
			}
		default:
			for _, v := range enumValues[key] {
				f.Options = append(f.Options, Option{Value: v[0], Label: v[1]})
			}
		}
		out.Fields = append(out.Fields, f)
	}
	for _, c := range custom {
		kind := KindText
		switch c.Type {
		case "number":
			kind = KindNumber
		case "date":
			kind = KindDate
		case "boolean":
			kind = KindBoolean
		}
		out.Fields = append(out.Fields, CatalogField{Key: "custom:" + c.Key, Label: "Custom field · " + c.Key, Kind: string(kind), Ops: ops(customOps(c.Type)), Options: []Option{}})
	}
	return out, nil
}

// SourceLabel turns a contact source key into words: "import:brevo" →
// "Brevo import", "form:docs" → "Signup form · docs", "form:popup" →
// "Popup form", "api" → "API".
func SourceLabel(src string) string {
	kind, detail, _ := strings.Cut(src, ":")
	switch kind {
	case "import":
		return importerName(detail) + " import"
	case "form":
		if detail == "popup" {
			return "Popup form"
		}
		if detail == "" {
			return "Signup form"
		}
		return "Signup form · " + detail
	case "api":
		return "API"
	case "manual":
		return "Added by the team"
	case "":
		return "Unknown"
	}
	return src
}

func importerName(k string) string {
	switch k {
	case "brevo":
		return "Brevo"
	case "mailchimp":
		return "Mailchimp"
	case "mailerlite":
		return "MailerLite"
	case "listmonk":
		return "listmonk"
	case "keila":
		return "Keila"
	case "csv":
		return "CSV"
	case "xlsx":
		return "XLSX"
	}
	if k == "" {
		return "File"
	}
	return strings.ToUpper(k[:1]) + k[1:]
}
