package store_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// knownGaps are tables production code reads that only the demo or tests
// write today, each with the review item that gives it a production writer.
// Remove an entry when its writer lands; never add one without an ID.
var knownGaps = map[string]string{}

var (
	createTable = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-z_][a-z0-9_]*)`)
	writeSQL    = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bINSERT\s+INTO\s+([a-z_][a-z0-9_]*)`),
		regexp.MustCompile(`(?i)\bUPDATE\s+([a-z_][a-z0-9_]*)\s+(?:AS\s+[a-z_]+\s+|[a-z]\s+)?SET\b`),
		regexp.MustCompile(`(?i)\bDELETE\s+FROM\s+([a-z_][a-z0-9_]*)`),
		regexp.MustCompile(`pgx\.Identifier\{"([a-z_][a-z0-9_]*)"\}`),
	}
	// Readers use the upper-case keywords the repository's SQL is written
	// in, so English text in messages ("from the provider") isn't SQL.
	readSQL = []*regexp.Regexp{
		regexp.MustCompile(`\bFROM\s+([a-z_][a-z0-9_]*)`),
		regexp.MustCompile(`\bJOIN\s+([a-z_][a-z0-9_]*)`),
	}
)

// TestQueriedTablesHaveProductionWriters fails when production code reads a
// table that only demo seeds or tests write: a feature tested on the demo
// would then pass while it never works on a real install (link-domain
// blocklists did exactly that).
func TestQueriedTablesHaveProductionWriters(t *testing.T) {
	root := filepath.Join("..", "..")
	tables := map[string]bool{}
	writers := map[string]bool{}
	readers := map[string][]string{}

	scan := func(path string, fn func(path, text string)) {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		fn(path, string(b))
	}
	// Migrations define the tables and may fill reference tables.
	migrations, err := filepath.Glob(filepath.Join(root, "migrations", "*.sql"))
	if err != nil || len(migrations) == 0 {
		t.Fatalf("no migrations found: %v", err)
	}
	for _, m := range migrations {
		scan(m, func(_, text string) {
			for _, mm := range createTable.FindAllStringSubmatch(text, -1) {
				tables[strings.ToLower(mm[1])] = true
			}
			collect(text, writeSQL, func(name string) { writers[name] = true })
		})
	}
	production := func(path string) bool {
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		switch {
		case strings.HasSuffix(rel, "_test.go"),
			strings.HasPrefix(rel, "internal/demo/"),
			strings.HasPrefix(rel, "internal/apptest/"),
			strings.HasPrefix(rel, "internal/testdb/"),
			// sqlc output repeats internal/store/queries, read below.
			strings.HasPrefix(rel, "internal/store/") && strings.HasSuffix(rel, ".go"):
			return false
		}
		return strings.HasSuffix(rel, ".go") || (strings.HasPrefix(rel, "internal/store/queries/") && strings.HasSuffix(rel, ".sql"))
	}
	// Raw SQL in Go counts as written; a sqlc query counts only when
	// production Go code calls it (an INSERT nobody calls writes nothing).
	var goCode strings.Builder
	var queryFiles []string
	err = filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !production(path) {
			return err
		}
		if strings.HasSuffix(path, ".sql") {
			queryFiles = append(queryFiles, path)
			return nil
		}
		scan(path, func(path, text string) {
			goCode.WriteString(text)
			collect(text, writeSQL, func(name string) { writers[name] = true })
			collect(text, readSQL, func(name string) { readers[name] = append(readers[name], path) })
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	code := goCode.String()
	for _, path := range queryFiles {
		scan(path, func(path, text string) {
			for _, q := range namedQueries(text) {
				if !strings.Contains(code, "."+q.name+"(") {
					continue
				}
				collect(q.sql, writeSQL, func(name string) { writers[name] = true })
				collect(q.sql, readSQL, func(name string) { readers[name] = append(readers[name], path+" ("+q.name+")") })
			}
		})
	}

	var missing []string
	for name, paths := range readers {
		if !tables[name] || writers[name] {
			continue
		}
		if _, ok := knownGaps[name]; ok {
			continue
		}
		slices.Sort(paths)
		missing = append(missing, name+" (read in "+filepath.ToSlash(paths[0])+")")
	}
	slices.Sort(missing)
	for _, m := range missing {
		t.Errorf("table %s has no production writer: only the demo or tests write it", m)
	}
	for name, why := range knownGaps {
		if writers[name] {
			t.Errorf("table %s has a production writer now (%s): remove it from knownGaps", name, why)
		}
	}
}

func collect(text string, res []*regexp.Regexp, add func(string)) {
	for _, re := range res {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			add(strings.ToLower(m[1]))
		}
	}
}

type namedQuery struct{ name, sql string }

var queryName = regexp.MustCompile(`(?m)^--\s*name:\s*([A-Za-z0-9_]+)\s*:`)

// namedQueries splits a sqlc query file into its queries.
func namedQueries(text string) []namedQuery {
	idx := queryName.FindAllStringSubmatchIndex(text, -1)
	out := make([]namedQuery, 0, len(idx))
	for i, m := range idx {
		end := len(text)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		out = append(out, namedQuery{name: text[m[2]:m[3]], sql: text[m[1]:end]})
	}
	return out
}
