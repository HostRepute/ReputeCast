// Package visibility is conditional content (design expansion, M21): the
// Visibility rule of a section or block, who it shows the block to, and
// what that does to an email.
//
// A rule lives in the email's MJML, in the block's rc-visible attribute, so
// it is versioned with the content and kept by language versions. When the
// email is compiled, every conditional block is wrapped in two markers
// (mjml.PrepareCompile); a send decides once, when its audience is built,
// which blocks each recipient doesn't get, and cuts those out of the
// compiled HTML (Apply). No MJML is compiled per recipient.
//
// Conditions come from a registry (RegisterKind) so later phases add their own:
// the score and its band (phase 4) join the Engagement group, a paid tier
// (phase 6) gets a group of its own.
package visibility

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hostrepute/reputecast/internal/mjml"
)

// Who sees a block.
const (
	// ShowEveryone is the default: no rule.
	ShowEveryone = "everyone"
	// ShowOnly shows the block to contacts who match the conditions.
	ShowOnly = "only"
	// ShowExcept shows it to everyone except contacts who match.
	ShowExcept = "except"
	// ShowNobody hides it for everyone ("Hide this block for everyone").
	ShowNobody = "nobody"
)

// How conditions combine.
const (
	MatchAll = "all"
	MatchAny = "any"
)

// VisibilityRule is a block's Visibility setting, as stored in its
// rc-visible attribute.
type VisibilityRule struct {
	// ID names the block in compiled HTML markers and in each recipient's
	// hidden blocks. Language versions keep the source's ids.
	ID         string                `json:"id"`
	Show       string                `json:"show" enum:"everyone,only,except,nobody"`
	Match      string                `json:"match,omitempty" enum:"all,any"`
	Conditions []VisibilityCondition `json:"conditions"`
}

// VisibilityCondition is one row of the rule: what is checked, how, and
// against what ("List membership · includes · Beta Program").
type VisibilityCondition struct {
	// Kind is a registered condition: segment, list, tag, field, language,
	// tier, and what later phases register.
	Kind string `json:"kind"`
	Op   string `json:"op"`
	// Field is the custom field a field condition reads ("plan").
	Field string `json:"field,omitempty"`
	// Value is a segment or list id, a tag, a field value, a language code
	// or a tier key, depending on the kind.
	Value string `json:"value,omitempty"`
	// Label is the name of the segment or list when it was picked, so a
	// condition still names it after it was deleted.
	Label string `json:"label,omitempty"`
}

// ErrNoRule is returned for an element without a Visibility rule.
var ErrNoRule = errors.New("visibility: no rule")

// ParseRule reads an rc-visible attribute value.
func ParseRule(raw string) (VisibilityRule, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return VisibilityRule{}, ErrNoRule
	}
	var r VisibilityRule
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return VisibilityRule{}, fmt.Errorf("visibility: %w", err)
	}
	r.normalize()
	return r, nil
}

func (r *VisibilityRule) normalize() {
	switch r.Show {
	case ShowOnly, ShowExcept, ShowNobody:
	default:
		r.Show = ShowEveryone
	}
	if r.Match != MatchAny {
		r.Match = MatchAll
	}
	if r.Conditions == nil {
		r.Conditions = []VisibilityCondition{}
	}
}

// Encode writes the rule as the attribute value.
func (r VisibilityRule) Encode() string {
	r.normalize()
	b, _ := json.Marshal(r)
	return string(b)
}

// Conditional reports whether the rule can leave some contacts out.
func (r VisibilityRule) Conditional() bool {
	switch r.Show {
	case ShowNobody:
		return true
	case ShowOnly, ShowExcept:
		return len(r.Conditions) > 0
	}
	return false
}

// NewID returns a fresh block id: six characters of [a-z0-9].
func NewID() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

// Shows decides the rule for a contact: whether the block is shown, given
// whether each condition matched (in order).
func (r VisibilityRule) Shows(matched []bool) bool {
	switch r.Show {
	case ShowNobody:
		return false
	case ShowOnly, ShowExcept:
		if len(r.Conditions) == 0 {
			return true
		}
		hit := r.Match != MatchAny
		for _, m := range matched {
			if r.Match == MatchAny {
				if m {
					hit = true
					break
				}
			} else if !m {
				hit = false
				break
			}
		}
		if r.Show == ShowOnly {
			return hit
		}
		return !hit
	}
	return true
}

// attrRule reads the rule of an element, ok false without one.
func attrRule(n *mjml.Node) (VisibilityRule, bool) {
	raw, has := n.Get(mjml.AttrVisible)
	if !has {
		return VisibilityRule{}, false
	}
	r, err := ParseRule(raw)
	if err != nil {
		return VisibilityRule{}, false
	}
	return r, true
}
