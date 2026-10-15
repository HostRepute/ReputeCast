package importer

import (
	"regexp"
	"slices"
	"strings"
	"time"
)

// pathUse is what a template does with one value. Paths are normalized from
// the root of the data: "user.profile.name", and loops add "[]" to their
// collection ("items[].price").
type pathUse struct {
	path    string
	order   int
	printed bool
	// bare: printed outside any condition or loop, so a call without it
	// renders a hole: the variable is required.
	bare      bool
	cond      bool
	iterated  bool
	inURL     bool
	date      bool
	number    bool
	hasDef    bool
	def       string
	firstLine int
}

// usage collects the paths of every field of a template (subject,
// preheader, body and text share one set of variables).
type usage struct {
	paths map[string]*pathUse
	n     int
	// locals are names a template assigns itself (Liquid assign/capture):
	// never variables.
	locals map[string]bool
}

func newUsage() *usage { return &usage{paths: map[string]*pathUse{}, locals: map[string]bool{}} }

func (u *usage) get(path string) *pathUse {
	p, ok := u.paths[path]
	if !ok {
		u.n++
		p = &pathUse{path: path, order: u.n}
		u.paths[path] = p
	}
	return p
}

// touch records path and every parent: "items[].price" makes items a list
// whose items are objects.
func (u *usage) touch(path string, line int, f func(*pathUse)) {
	if path == "" {
		return
	}
	p := u.get(path)
	if p.firstLine == 0 {
		p.firstLine = line
	}
	f(p)
	// Parents exist too, in the order they were first seen.
	for parent := parentPath(path); parent != ""; parent = parentPath(parent) {
		pp := u.get(parent)
		if pp.firstLine == 0 {
			pp.firstLine = line
		}
	}
}

// parentPath is "items[]" for "items[].price", "items" for "items[]" and
// "user" for "user.name".
func parentPath(p string) string {
	if strings.HasSuffix(p, "[]") {
		return strings.TrimSuffix(p, "[]")
	}
	if i := strings.LastIndex(p, "."); i > 0 {
		return p[:i]
	}
	return ""
}

func rootOf(p string) string {
	if i := strings.IndexAny(p, ".["); i > 0 {
		return p[:i]
	}
	return p
}

// children returns the direct child paths of p, in order of first use.
func (u *usage) children(p string) []*pathUse {
	out := []*pathUse{}
	for _, c := range u.paths {
		if parentPath(c.path) == p {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b *pathUse) int { return a.order - b.order })
	return out
}

func (u *usage) has(p string) bool { _, ok := u.paths[p]; return ok }

// typeOf infers the type of a path from how it is used. Weak signals (a
// name that sounds like a date) never change the type: a call that sends
// text where a number was declared would fail.
func (u *usage) typeOf(p string) string {
	pu := u.paths[p]
	if pu == nil {
		return TypeText
	}
	switch {
	case pu.iterated || u.has(p+"[]"):
		return TypeList
	case len(u.children(p)) > 0:
		return TypeObject
	case pu.date:
		return TypeDate
	case pu.number:
		return TypeNumber
	case pu.inURL && looksLikeURL(lastSegment(p)):
		return TypeURL
	case pu.cond && !pu.printed:
		return TypeBool
	}
	return TypeText
}

func lastSegment(p string) string {
	p = strings.TrimSuffix(p, "[]")
	if i := strings.LastIndex(p, "."); i >= 0 {
		return p[i+1:]
	}
	return p
}

var urlName = regexp.MustCompile(`(?i)(url|uri|link|href)$`)

func looksLikeURL(name string) bool { return urlName.MatchString(name) }

var sensitiveName = regexp.MustCompile(`(?i)(password|passcode|secret|token|otp|reset_?url|reset_?link|magic_?link|login_?(url|link)|sign_?in_?(url|link)|verification_?code|one_?time_?code|^code$|_code$)`)

// declare returns the root variables, in order of first use, with their
// type, whether a call must send them, a sample and the template's default.
func (u *usage) declare(sample map[string]any) []Variable {
	roots := []*pathUse{}
	seen := map[string]bool{}
	all := make([]*pathUse, 0, len(u.paths))
	for _, p := range u.paths {
		all = append(all, p)
	}
	slices.SortFunc(all, func(a, b *pathUse) int { return a.order - b.order })
	for _, p := range all {
		r := rootOf(p.path)
		if seen[r] || u.locals[r] || systemVars[r] || r == "forloop" {
			continue
		}
		seen[r] = true
		roots = append(roots, u.get(r))
	}
	out := make([]Variable, 0, len(roots))
	for _, r := range roots {
		v := Variable{Name: r.path, Type: u.typeOf(r.path), Default: r.def}
		v.Required = !r.hasDef && u.requiredTree(r.path)
		if sample != nil {
			if s, ok := sample[r.path]; ok {
				v.Sample = s
			}
		}
		if v.Sample == nil {
			v.Sample = u.sampleFor(r.path, v.Type)
		}
		v.Sensitive = v.Type != TypeList && v.Type != TypeObject && v.Type != TypeBool && sensitiveName.MatchString(r.path)
		out = append(out, v)
	}
	return out
}

// requiredTree: a root is required when it, or something under it, is
// printed or looped over outside any condition.
func (u *usage) requiredTree(root string) bool {
	for _, p := range u.paths {
		if (p.path == root || strings.HasPrefix(p.path, root+".") || strings.HasPrefix(p.path, root+"[")) && p.bare {
			return true
		}
	}
	return false
}

// sampleFor builds a value that renders: lists get one item with their
// fields, objects their fields.
func (u *usage) sampleFor(p, typ string) any {
	switch typ {
	case TypeList:
		item := p + "[]"
		if kids := u.children(item); len(kids) > 0 {
			obj := map[string]any{}
			for _, k := range kids {
				obj[lastSegment(k.path)] = u.sampleFor(k.path, u.typeOf(k.path))
			}
			return []any{obj}
		}
		return []any{humanize(lastSegment(p))}
	case TypeObject:
		obj := map[string]any{}
		for _, k := range u.children(p) {
			obj[lastSegment(k.path)] = u.sampleFor(k.path, u.typeOf(k.path))
		}
		return obj
	case TypeBool:
		return true
	case TypeNumber:
		return 1
	case TypeDate:
		return time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC).Format("2006-01-02")
	case TypeURL:
		return "https://example.com/" + strings.ReplaceAll(strings.ToLower(lastSegment(p)), "_", "-")
	}
	if pu := u.paths[p]; pu != nil && pu.hasDef {
		return pu.def
	}
	return humanize(lastSegment(p))
}

// humanize turns "first_name" or "firstName" into "First name".
func humanize(s string) string {
	var b strings.Builder
	for i, r := range s {
		switch {
		case r == '_' || r == '-':
			b.WriteByte(' ')
		case i > 0 && r >= 'A' && r <= 'Z':
			b.WriteByte(' ')
			b.WriteRune(r + ('a' - 'A'))
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "Sample"
	}
	return strings.ToUpper(out[:1]) + out[1:]
}

// sampleData is the first test data set: the source's own test data when it
// has one, filled in with samples for anything it lacks.
func sampleData(vars []Variable, src map[string]any) map[string]any {
	out := map[string]any{}
	for _, v := range vars {
		if s, ok := src[v.Name]; ok {
			out[v.Name] = s
			continue
		}
		if v.Sample != nil {
			out[v.Name] = v.Sample
		}
	}
	return out
}

// systemVars are filled in by ReputeCast at send time; templates use them
// without declaring them.
var systemVars = map[string]bool{
	"unsubscribe_url":     true,
	"preferences_url":     true,
	"view_in_browser_url": true,
}
