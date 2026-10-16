package accessibility

import (
	"errors"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// Additions says what Finalize set on an email's HTML.
type Additions struct {
	// Lang and Dir are the values on <html> and the email's wrapper.
	Lang string
	Dir  string
	// LangAdded is true when lang or dir was missing or different.
	LangAdded bool
	// LayoutTables counts the layout tables that carry role="presentation"
	// after finalizing; RoleAdded how many of them ReputeCast marked.
	LayoutTables int
	RoleAdded    int
	// DataTables counts the tables kept as data tables (header cells, or
	// the Table block).
	DataTables int
}

// DirLTR is the only direction for now: right-to-left languages aren't
// supported yet.
const DirLTR = "ltr"

type openTable struct {
	start     int
	role      string
	hasRole   bool
	dataTable bool
}

// Finalize adds what a machine can add safely to an email's HTML: lang and
// dir from the version's language, and role="presentation" on layout
// tables MJML didn't mark (tables inside Raw HTML blocks, for instance).
// Data tables keep their meaning: the Table block's table and any table
// with header cells. Only those attributes change; everything else in the
// HTML stays byte for byte, and running it again changes nothing.
func Finalize(src, lang string) (string, Additions) {
	add := Additions{Lang: strings.TrimSpace(lang), Dir: DirLTR}
	if add.Lang == "" {
		add.Lang = "en"
	}
	if strings.TrimSpace(src) == "" {
		return src, add
	}
	var edits []edit
	setLangDir := func(start int) {
		_, attrs, _, ok := openTag(src, start)
		if !ok {
			return
		}
		cur := map[string]string{}
		for _, a := range attrs {
			cur[a.name] = strings.TrimSpace(src[a.valStart:a.valEnd])
		}
		for _, kv := range [][2]string{{"lang", add.Lang}, {"dir", add.Dir}} {
			if strings.EqualFold(cur[kv[0]], kv[1]) {
				continue
			}
			if e, ok := setAttrEdit(src, start, kv[0], kv[1]); ok {
				edits = append(edits, e)
				add.LangAdded = true
			}
		}
	}

	z := html.NewTokenizer(strings.NewReader(src))
	off := 0
	var tables []openTable
	bodyDepth, depth := -1, 0
	wrapperDone, sawHTML := false, false
	bodyStart, candidate := -1, -1
	closeTable := func(t openTable) {
		switch {
		case t.hasRole && (t.role == "presentation" || t.role == "none"):
			add.LayoutTables++
		case t.hasRole:
			add.DataTables++
		case t.dataTable:
			add.DataTables++
		default:
			if e, ok := setAttrEdit(src, t.start, "role", "presentation"); ok {
				edits = append(edits, e)
				add.LayoutTables++
				add.RoleAdded++
			}
		}
	}
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if !errors.Is(z.Err(), io.EOF) {
				return src, add
			}
			break
		}
		raw := len(z.Raw())
		start := off
		off += raw
		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			tag := strings.ToLower(string(name))
			attrs := map[string]string{}
			for hasAttr {
				var k, v []byte
				k, v, hasAttr = z.TagAttr()
				attrs[strings.ToLower(string(k))] = string(v)
			}
			switch tag {
			case "html":
				sawHTML = true
				setLangDir(start)
			case "body":
				bodyDepth, bodyStart = depth, start
			case "div":
				// The email's wrapper (MJML's article div) keeps lang and dir
				// where clients drop <html>, as MJML 4.15 does. The hidden
				// preview text comes before it.
				if !wrapperDone && bodyDepth >= 0 && depth == bodyDepth+1 {
					switch {
					case attrs["role"] == "article" || attrs["aria-roledescription"] == "email":
						wrapperDone = true
						setLangDir(start)
					case candidate < 0 && !strings.Contains(strings.ReplaceAll(strings.ToLower(attrs["style"]), " ", ""), "display:none"):
						candidate = start
					}
				}
			case "table":
				role, hasRole := attrs["role"]
				style := strings.ToLower(strings.ReplaceAll(attrs["style"], " ", ""))
				tables = append(tables, openTable{start: start, role: strings.ToLower(strings.TrimSpace(role)), hasRole: hasRole,
					dataTable: strings.Contains(style, "table-layout:auto")})
			case "th", "caption", "thead":
				if len(tables) > 0 {
					tables[len(tables)-1].dataTable = true
				}
			}
			if tt == html.StartTagToken && !voidElements[tag] {
				depth++
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			if strings.ToLower(string(name)) == "table" && len(tables) > 0 {
				closeTable(tables[len(tables)-1])
				tables = tables[:len(tables)-1]
			}
			if depth > 0 {
				depth--
			}
		}
	}
	for i := len(tables) - 1; i >= 0; i-- {
		closeTable(tables[i])
	}
	switch {
	case wrapperDone:
	case candidate >= 0:
		setLangDir(candidate)
	case bodyStart >= 0:
		setLangDir(bodyStart)
	}
	if !sawHTML && bodyStart < 0 && !wrapperDone {
		// A fragment without <html> or <body>: nothing to carry lang.
		add.LangAdded = false
	}
	return applyEdits(src, edits), add
}
