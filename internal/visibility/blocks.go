package visibility

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/hostrepute/reputecast/internal/mjml"
)

// Block is a conditional block of an email: a section or a content element
// with a Visibility rule that leaves some contacts out.
type Block struct {
	ID string
	// Name is the block's name in the layer tree: its own (rc-name, "Beta
	// corner"), else what it is ("text", "2 columns").
	Name string
	Tag  string
	// Path is the element-index path from mj-body, as the composer and the
	// content checks locate elements.
	Path    []int
	Section bool
	Rule    VisibilityRule
	// Parent is the conditional block this one sits in, "" for none: a
	// block inside a hidden one is hidden with it.
	Parent string
}

// Email is what an email's MJML says about conditional content.
type Email struct {
	Blocks []Block
	// Sections are mj-body's elements in order, with the conditional block
	// each one is (empty when it isn't).
	Sections []EmailSection
	// Invalid are the rc-visible attributes that couldn't be read.
	Invalid []string
}

// EmailSection is one top-level element of the body.
type EmailSection struct {
	Index int
	Label string
	// Role is header, footer or content: a version is empty when it shows
	// no content section.
	Role string
	// Block is the id of the section's own conditional block, if any.
	Block string
}

// conditionalTags are the elements Visibility is offered on: sections and
// content blocks. Columns and groups aren't (hiding one would leave the
// others at their widths, and Outlook's tables need every cell).
var conditionalTags = map[string]bool{
	"mj-section": true, "mj-wrapper": true, "mj-hero": true,
	"mj-text": true, "mj-image": true, "mj-button": true, "mj-divider": true, "mj-spacer": true,
	"mj-social": true, "mj-navbar": true, "mj-raw": true, "mj-table": true,
}

// Offered reports whether Visibility can be set on an element of this tag.
func Offered(tag string) bool { return conditionalTags[tag] }

// Read finds an email's conditional blocks. MJML that doesn't parse has
// none (the compile reports it). The footer is always shown: a rule on it is
// ignored, as on a block of a kind Visibility isn't offered on.
func Read(src string) Email {
	out := Email{Blocks: []Block{}, Sections: []EmailSection{}, Invalid: []string{}}
	if !strings.Contains(src, "<mj-body") {
		return out
	}
	root, err := mjml.Parse(src)
	if err != nil {
		return out
	}
	body := mjml.Body(root)
	if body == nil {
		return out
	}
	total := len(body.Children)
	seen := map[string]bool{}
	for i, sec := range body.Children {
		label := mjml.SectionLabel(sec, i, total)
		role := "content"
		switch {
		case sec.Marker == "footer" || label == "footer":
			role = "footer"
		case i == 0 && (sec.Marker == "header-logo" || sec.Marker == "header" || label == "header"):
			role = "header"
		}
		es := EmailSection{Index: i, Label: label, Role: role}
		if role != "footer" {
			var walk func(n *mjml.Node, path []int, parent string)
			walk = func(n *mjml.Node, path []int, parent string) {
				inner := parent
				if conditionalTags[n.Tag] {
					if r, ok := attrRule(n); ok {
						switch {
						case !validRuleID(r.ID):
							out.Invalid = append(out.Invalid, n.Attr(mjml.AttrVisible))
						case r.Conditional() && !seen[r.ID]:
							seen[r.ID] = true
							name := strings.TrimSpace(n.Attr(mjml.AttrName))
							if name == "" {
								if len(path) == 1 {
									name = label
								} else {
									name = strings.TrimPrefix(n.Tag, "mj-")
								}
							}
							out.Blocks = append(out.Blocks, Block{ID: r.ID, Name: name, Tag: n.Tag, Path: append([]int(nil), path...), Section: len(path) == 1, Rule: r, Parent: parent})
							inner = r.ID
							if len(path) == 1 {
								es.Block = r.ID
							}
						}
					} else if raw, has := n.Get(mjml.AttrVisible); has && strings.TrimSpace(raw) != "" {
						out.Invalid = append(out.Invalid, raw)
					}
				}
				for j, c := range n.Children {
					walk(c, append(append([]int(nil), path...), j), inner)
				}
			}
			walk(sec, []int{i}, "")
		}
		out.Sections = append(out.Sections, es)
	}
	return out
}

func validRuleID(id string) bool {
	if id == "" || len(id) > 24 {
		return false
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// Block returns the block with an id.
func (e Email) Block(id string) (Block, bool) {
	for _, b := range e.Blocks {
		if b.ID == id {
			return b, true
		}
	}
	return Block{}, false
}

// Refs lists what the rules refer to, once each.
func (e Email) Refs() []Ref {
	out := []Ref{}
	seen := map[Ref]bool{}
	for _, b := range e.Blocks {
		for _, c := range b.Rule.Conditions {
			k, ok := Lookup(c.Kind)
			if !ok || k.Ref == nil {
				continue
			}
			r := k.Ref(c)
			if r.Kind == "" || r.ID == "" || seen[r] {
				continue
			}
			seen[r] = true
			out = append(out, r)
		}
	}
	return out
}

// Segments lists the segment ids the rules refer to.
func (e Email) Segments() []string {
	out := []string{}
	for _, r := range e.Refs() {
		if r.Kind == "segment" {
			out = append(out, r.ID)
		}
	}
	return out
}

// Hash fingerprints the rules (ids and conditions): a count is valid while
// the rules and the audience stay the same.
func (e Email) Hash() string {
	parts := make([]string, 0, len(e.Blocks))
	for _, b := range e.Blocks {
		parts = append(parts, b.ID+"="+b.Rule.Encode())
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:12])
}

// Empty reports whether a version shows nothing between the header and the
// footer: every content section is hidden (an email with no content
// sections at all isn't empty because of its rules).
func (e Email) Empty(hidden map[string]bool) bool {
	content, shown := 0, 0
	for _, s := range e.Sections {
		if s.Role != "content" {
			continue
		}
		content++
		if s.Block == "" || !hidden[s.Block] {
			shown++
		}
	}
	return content > 0 && shown == 0
}
