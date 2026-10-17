package imports

// A source column mapped to the contact's language (design expansion M17,
// brief §3.4): values are normalized to BCP 47 ("Spanish", "es_MX" and
// "ES" become es and es-MX), and values that aren't languages are listed on
// Map fields and left empty.

import (
	"context"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/store"
)

// importLanguage is the code an imported value becomes, or "" when it
// isn't a language.
func importLanguage(v string) string {
	code, ok := languages.Normalize(v)
	if !ok {
		return ""
	}
	return code
}

// ImportLanguageValue is one value of a language column and its rows.
type ImportLanguageValue struct {
	Value string `json:"value"`
	// Code is what the value becomes ("es-MX"); empty for values that
	// aren't languages.
	Code string `json:"code"`
	Name string `json:"name"`
	Rows int64  `json:"rows"`
}

// ImportLanguageColumn says what importing a column into the contact's
// language does: the values that change when normalized, and the values
// that aren't languages, which are left empty.
type ImportLanguageColumn struct {
	// Normalized are values written differently from their code ("Spanish"
	// → es), most rows first; at most five.
	Normalized []ImportLanguageValue `json:"normalized"`
	// Unknown are values that aren't languages, most rows first; at most
	// five, UnknownValues counts them all and UnknownRows their rows.
	Unknown       []ImportLanguageValue `json:"unknown"`
	UnknownValues int                   `json:"unknownValues"`
	UnknownRows   int64                 `json:"unknownRows"`
	// Languages counts the rows per language, by English name, most first.
	Languages []ImportLanguageValue `json:"languages"`
}

// languageColumn reads every staged value of a source field.
func (s *Service) languageColumn(ctx context.Context, importID uuid.UUID, key string) (*ImportLanguageColumn, error) {
	rows, err := s.Pool.Query(ctx, `SELECT coalesce(btrim(data->>$2), '') AS value, count(*)::bigint FROM import_rows WHERE import_id = $1 GROUP BY 1`, importID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := &ImportLanguageColumn{Normalized: []ImportLanguageValue{}, Unknown: []ImportLanguageValue{}, Languages: []ImportLanguageValue{}}
	byLang := map[string]int64{}
	for rows.Next() {
		var v ImportLanguageValue
		if err := rows.Scan(&v.Value, &v.Rows); err != nil {
			return nil, err
		}
		if v.Value == "" {
			continue
		}
		v.Code = importLanguage(v.Value)
		if v.Code == "" {
			out.Unknown = append(out.Unknown, v)
			out.UnknownRows += v.Rows
			continue
		}
		v.Name = languages.NameOf(v.Code)
		byLang[languages.Base(v.Code)] += v.Rows
		if v.Code != v.Value {
			out.Normalized = append(out.Normalized, v)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for code, n := range byLang {
		out.Languages = append(out.Languages, ImportLanguageValue{Value: code, Code: code, Name: languages.NameOf(code), Rows: n})
	}
	byRows := func(xs []ImportLanguageValue) {
		sort.Slice(xs, func(i, j int) bool {
			if xs[i].Rows != xs[j].Rows {
				return xs[i].Rows > xs[j].Rows
			}
			return strings.ToLower(xs[i].Value) < strings.ToLower(xs[j].Value)
		})
	}
	byRows(out.Normalized)
	byRows(out.Unknown)
	byRows(out.Languages)
	out.UnknownValues = len(out.Unknown)
	if len(out.Normalized) > 5 {
		out.Normalized = out.Normalized[:5]
	}
	if len(out.Unknown) > 5 {
		out.Unknown = out.Unknown[:5]
	}
	return out, nil
}

// LanguageColumn reads a source field as a language column: what its
// values become and which aren't languages.
func (s *Service) LanguageColumn(ctx context.Context, ws *store.Workspace, id uuid.UUID, field string) (ImportLanguageColumn, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportLanguageColumn{}, err
	}
	known := false
	for _, f := range parseConnection(imp.Connection).Fields {
		known = known || f.Key == field
	}
	if !known {
		return ImportLanguageColumn{}, api.NotFound("Field")
	}
	out, err := s.languageColumn(ctx, imp.ID, field)
	if err != nil {
		return ImportLanguageColumn{}, err
	}
	return *out, nil
}

type languageColumnInput struct {
	ImportPathInput
	Field string `path:"field" maxLength:"200" doc:"The source field's key"`
}

// registerLanguageColumn adds the language column's check to Map fields.
func registerLanguageColumn(h huma.API, s *Service) {
	api.Get(h, api.Op{ID: "get-import-language-column", Path: "/workspaces/{workspace}/imports/{id}/fields/{field}/languages",
		Summary: "What a source field becomes as the contact's language, and the values that aren't languages",
		Tags:    []string{"Imports"}, Access: api.Workspace(authz.SubscribersImport)},
		func(ctx context.Context, in *languageColumnInput) (*api.Body[ImportLanguageColumn], error) {
			out, err := s.LanguageColumn(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID), in.Field)
			return api.OK(out), err
		})
}
