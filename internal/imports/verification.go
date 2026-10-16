package imports

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// The import preview's optional check with a verification service (design
// expansion, X14): the new addresses go to the service, its answers join
// the preview's groups (invalid ones are skipped, risky ones imported and
// tagged), and the import never needs it.

// NewAddresses returns the addresses the import would add as new contacts:
// the preview's rows to import and its risky rows (with typo fixes), each
// once; and the import's name as the use reads it (Mailchimp “Acme
// Customers”).
func (s *Service) NewAddresses(ctx context.Context, ws *store.Workspace, id uuid.UUID) (string, []string, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return "", nil, err
	}
	if imp.StagedAt == nil {
		return "", nil, api.Conflict("The rows aren't read from the source yet. Wait for the field mapping to load, then try again.")
	}
	custom, err := s.customFields(ctx, ws.ID)
	if err != nil {
		return "", nil, err
	}
	fm := s.effectiveMap(imp, custom)
	opts := parseOptions(imp.Options)
	cm := parseConsentMap(imp.ConsentMap, opts)
	now := s.now()
	seen := map[string]bool{}
	out := []string{}
	var after int32
	for {
		b, err := s.loadBatch(ctx, ws.ID, imp, fm, after, 5000)
		if err != nil {
			return "", nil, err
		}
		if len(b.rows) == 0 {
			break
		}
		for _, r := range b.rows {
			after = r.RowNo
			v := classify(r, b, cm, now)
			if v.Class != classImport && !isRisky(v.Class) {
				continue
			}
			e := strings.ToLower(strings.TrimSpace(r.Email))
			if v.Fixed != "" {
				e = strings.ToLower(v.Fixed)
			}
			if e == "" || seen[e] {
				continue
			}
			seen[e] = true
			out = append(out, e)
		}
	}
	return importLabel(imp), out, nil
}

// importLabel names an import by its platform and lists: Mailchimp “Acme
// Customers”.
func importLabel(imp store.Import) string {
	name := SourceName(imp.Source)
	conn := parseConnection(imp.Connection)
	opts := parseOptions(imp.Options)
	names := []string{}
	for _, c := range opts.Lists {
		for _, l := range conn.Lists {
			if l.ID == c.SourceID && l.Name != "" {
				names = append(names, "“"+l.Name+"”")
			}
		}
	}
	if len(names) == 0 {
		return name
	}
	return name + " " + strings.Join(names, ", ")
}
