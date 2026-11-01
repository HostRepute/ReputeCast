package app

import (
	"bytes"
	"html/template"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/version"
)

// The API reference this instance serves at /api/reference: every
// operation of its own OpenAPI document, grouped by tag, with the
// permission it needs. It needs nothing from outside, so it always matches
// the version running (review 2026-11 IN-4).

type refOperation struct {
	Method, Path, Summary, Description, Permission, ID string
	Deprecated                                         bool
}

type refTag struct {
	Name string
	Ops  []refOperation
}

// referenceSource is the page; anchor names a tag's id.
const referenceSource = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>ReputeCast API reference · {{.Version}}</title>
<style>
:root{--bg:#f6f7f9;--card:#fff;--ink:#0b1220;--muted:#5b6472;--line:#e3e6eb;--accent:#0f766e;--get:#1e40af;--post:#0f766e;--put:#92400e;--patch:#6d28d9;--delete:#991b1b}
@media (prefers-color-scheme: dark){:root{--bg:#0b1220;--card:#111a2c;--ink:#e6eaf0;--muted:#9aa3b2;--line:#243049;--accent:#5eead4;--get:#93c5fd;--post:#5eead4;--put:#fcd34d;--patch:#c4b5fd;--delete:#fca5a5}}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font:15px/1.5 system-ui,-apple-system,"Segoe UI",sans-serif}
a{color:var(--accent)}header{padding:24px 16px;max-width:1100px;margin:0 auto}
h1{margin:0 0 4px;font-size:24px}p.lead{margin:0;color:var(--muted)}
.wrap{display:grid;grid-template-columns:220px minmax(0,1fr);gap:24px;max-width:1100px;margin:0 auto;padding:0 16px 48px}
nav{position:sticky;top:12px;align-self:start;max-height:calc(100vh - 24px);overflow:auto;font-size:13px}
nav a{display:block;padding:3px 0;text-decoration:none;color:var(--muted)}nav a:hover,nav a:focus{color:var(--ink)}
section{background:var(--card);border:1px solid var(--line);border-radius:10px;margin-bottom:16px}
section h2{margin:0;padding:12px 16px;font-size:16px;border-bottom:1px solid var(--line)}
.op{padding:10px 16px;border-top:1px solid var(--line)}.op:first-of-type{border-top:0}
.sig{display:flex;gap:10px;align-items:baseline;flex-wrap:wrap}
.m{font:600 12px ui-monospace,monospace;text-transform:uppercase;min-width:52px}
.m.get{color:var(--get)}.m.post{color:var(--post)}.m.put{color:var(--put)}.m.patch{color:var(--patch)}.m.delete{color:var(--delete)}
code{font:13px ui-monospace,"IBM Plex Mono",monospace;word-break:break-all}
.sum{margin:2px 0 0}.desc{margin:4px 0 0;color:var(--muted);font-size:14px}
.perm{font:12px ui-monospace,monospace;color:var(--muted);border:1px solid var(--line);border-radius:999px;padding:1px 8px}
@media (max-width:760px){.wrap{grid-template-columns:1fr;gap:12px}nav{position:sticky;top:0;z-index:1;display:flex;gap:16px;overflow-x:auto;white-space:nowrap;max-height:none;margin:0 -16px;padding:10px 16px;background:var(--bg);border-bottom:1px solid var(--line)}nav a{flex:none}}
.skip{position:absolute;left:-9999px}.skip:focus{left:16px;top:8px;background:var(--card);padding:6px 10px;border-radius:6px;border:1px solid var(--line)}
section{scroll-margin-top:56px}
</style>
</head>
<body>
<a class="skip" href="#ops">Skip to the operations</a>
<header>
<h1>ReputeCast API reference</h1>
<p class="lead">Version {{.Version}} · OpenAPI 3.1: <a href="/api/openapi.json">openapi.json</a> · <a href="/api/openapi.yaml">openapi.yaml</a>.
Errors are RFC 9457 problem details; each operation names the permission it needs.</p>
</header>
<div class="wrap">
<nav aria-label="Sections">{{range .Tags}}<a href="#{{.Name | anchor}}">{{.Name}}</a>{{end}}</nav>
<main id="ops">
{{range .Tags}}<section id="{{.Name | anchor}}" aria-labelledby="h-{{.Name | anchor}}"><h2 id="h-{{.Name | anchor}}">{{.Name}}</h2>
{{range .Ops}}<div class="op" id="{{.ID}}"><div class="sig"><span class="m {{.Method}}">{{.Method}}</span><code>{{.Path}}</code>{{if .Permission}}<span class="perm">{{.Permission}}</span>{{end}}{{if .Deprecated}}<span class="perm">deprecated</span>{{end}}</div>
<p class="sum">{{.Summary}}</p>{{if .Description}}<p class="desc">{{.Description}}</p>{{end}}</div>
{{end}}</section>
{{end}}
</main>
</div>
</body>
</html>
`

var referenceTmpl = template.Must(template.New("reference").Funcs(template.FuncMap{"anchor": anchor}).Parse(referenceSource))

// referenceHandler renders the reference once, from the API's document.
func referenceHandler(api huma.API) http.HandlerFunc {
	var once sync.Once
	var page []byte
	render := func() {
		tags := map[string]*refTag{}
		var order []string
		add := func(method string, op *huma.Operation) {
			if op == nil || op.Hidden {
				return
			}
			tag := "Other"
			if len(op.Tags) > 0 {
				tag = op.Tags[0]
			}
			t := tags[tag]
			if t == nil {
				t = &refTag{Name: tag}
				tags[tag] = t
				order = append(order, tag)
			}
			perm, _ := op.Extensions["x-permission"].(string)
			t.Ops = append(t.Ops, refOperation{Method: method, Path: op.Path, Summary: op.Summary, Description: op.Description, Permission: perm,
				ID: op.OperationID, Deprecated: op.Deprecated})
		}
		oa := api.OpenAPI()
		paths := make([]string, 0, len(oa.Paths))
		for p := range oa.Paths {
			paths = append(paths, p)
		}
		slices.Sort(paths)
		for _, p := range paths {
			it := oa.Paths[p]
			add("get", it.Get)
			add("post", it.Post)
			add("put", it.Put)
			add("patch", it.Patch)
			add("delete", it.Delete)
		}
		slices.Sort(order)
		out := struct {
			Version string
			Tags    []refTag
		}{Version: version.Get().Version}
		for _, name := range order {
			out.Tags = append(out.Tags, *tags[name])
		}
		var b bytes.Buffer
		if err := referenceTmpl.Execute(&b, out); err == nil {
			page = b.Bytes()
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		once.Do(render)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'")
		_, _ = w.Write(page)
	}
}

// anchor is a tag's id in the page ("Transactional messages" →
// "transactional-messages").
func anchor(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		}
		return '-'
	}, s)
}
