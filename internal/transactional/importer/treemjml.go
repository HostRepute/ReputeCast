package importer

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Notifuse's visual editor stores an email as a tree of MJML blocks:
// {"type":"mj-text","attributes":{"paddingTop":"10px"},"content":"<p>…</p>",
// "children":[…]}. Attribute keys are camelCase. TreeToMJML writes the tree
// out as an MJML document, so the template opens in ReputeCast's visual
// editor with the same sections.

type treeNode struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Attributes map[string]any    `json:"attributes"`
	Content    *string           `json:"content"`
	Children   []json.RawMessage `json:"children"`
}

// mjmlTags are the elements MJML 4 knows; mj-liquid is Notifuse's own.
var mjmlTags = map[string]bool{
	"mjml": true, "mj-head": true, "mj-body": true, "mj-wrapper": true, "mj-section": true, "mj-column": true,
	"mj-group": true, "mj-text": true, "mj-button": true, "mj-image": true, "mj-divider": true, "mj-spacer": true,
	"mj-social": true, "mj-social-element": true, "mj-raw": true, "mj-attributes": true, "mj-all": true,
	"mj-class": true, "mj-breakpoint": true, "mj-font": true, "mj-html-attributes": true, "mj-preview": true,
	"mj-style": true, "mj-title": true, "mj-table": true, "mj-hero": true, "mj-navbar": true,
	"mj-navbar-link": true, "mj-accordion": true, "mj-accordion-element": true, "mj-accordion-title": true,
	"mj-accordion-text": true, "mj-carousel": true, "mj-carousel-image": true, "mj-selector": true, "mj-html-attribute": true,
}

// ending tags whose content is text or HTML, not child elements.
var contentTags = map[string]bool{
	"mj-text": true, "mj-button": true, "mj-raw": true, "mj-title": true, "mj-preview": true, "mj-style": true,
	"mj-social-element": true, "mj-table": true, "mj-navbar-link": true, "mj-accordion-title": true,
	"mj-accordion-text": true, "mj-html-attribute": true,
}

// TreeToMJML converts a visual editor tree. Blocks MJML doesn't know are
// left out and named in the notes.
func TreeToMJML(raw json.RawMessage) (string, []Note, error) {
	var root treeNode
	if err := json.Unmarshal(raw, &root); err != nil {
		return "", nil, fmt.Errorf("the visual editor tree can't be read: %w", err)
	}
	if root.Type != "mjml" {
		return "", nil, fmt.Errorf("the visual editor tree doesn't start with an mjml block")
	}
	var b strings.Builder
	notes := []Note{}
	skipped := map[string]bool{}
	var walk func(n treeNode, depth int) error
	walk = func(n treeNode, depth int) error {
		tag := n.Type
		indent := strings.Repeat("  ", depth)
		if tag == "mj-liquid" {
			// Liquid code renders at send time inside raw HTML.
			tag = "mj-raw"
			notes = append(notes, Note{Kind: NoteChange, Field: "body", Message: "Notifuse's Liquid block became an `mj-raw` block with the same code."})
		}
		if !mjmlTags[tag] {
			skipped[n.Type] = true
			return nil
		}
		b.WriteString(indent + "<" + tag + attrs(n.Attributes) + ">")
		switch {
		case contentTags[tag]:
			if n.Content != nil {
				b.WriteString(*n.Content)
			}
		case len(n.Children) > 0:
			b.WriteString("\n")
			for _, c := range n.Children {
				var child treeNode
				if err := json.Unmarshal(c, &child); err != nil {
					return fmt.Errorf("a block of the visual editor tree can't be read: %w", err)
				}
				if err := walk(child, depth+1); err != nil {
					return err
				}
			}
			b.WriteString(indent)
		}
		b.WriteString("</" + tag + ">\n")
		return nil
	}
	if err := walk(root, 0); err != nil {
		return "", nil, err
	}
	if len(skipped) > 0 {
		names := make([]string, 0, len(skipped))
		for k := range skipped {
			names = append(names, "`"+k+"`")
		}
		sort.Strings(names)
		notes = append(notes, Note{Kind: NoteManual, Field: "body", Message: "these Notifuse blocks have no MJML equivalent and were left out: " + strings.Join(names, ", ") + "."})
	}
	return b.String(), dedupeNotes(notes), nil
}

// attrs writes camelCase attributes as MJML's kebab-case ones, in a stable
// order.
func attrs(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		var v string
		switch x := m[k].(type) {
		case nil:
			continue
		case string:
			v = x
		case bool:
			if !x {
				continue
			}
			v = "true"
		case float64:
			v = trimFloat(x)
		default:
			continue
		}
		if v == "" {
			continue
		}
		b.WriteString(" " + kebab(k) + `="` + attrEscape(v) + `"`)
	}
	return b.String()
}

func trimFloat(f float64) string {
	s := fmt.Sprintf("%g", f)
	return s
}

func kebab(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func attrEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", `"`, "&quot;", "<", "&lt;", ">", "&gt;").Replace(s)
}
