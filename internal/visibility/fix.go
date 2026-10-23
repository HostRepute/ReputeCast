package visibility

import (
	"errors"
	"strconv"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
)

// Fixes Send Readiness offers for conditional content.
const (
	// FixShowEveryone removes a block's rule ("Show the content to everyone").
	FixShowEveryone = "visibility_show_everyone"
	// FixHideBlock hides a block for everyone ("Hide this block for everyone").
	FixHideBlock = "visibility_hide_block"
	// FixRemoveCondition removes one condition of a block's rule ("Remove
	// the condition").
	FixRemoveCondition = "visibility_remove_condition"
)

// ErrNoBlock is a fix for a block the email doesn't have (any more).
var ErrNoBlock = errors.New("that block isn't in the email any more; run the checks again")

// findRule returns the element whose rule has the id.
func findRule(src, id string) (*mjml.Node, VisibilityRule, error) {
	root, err := mjml.Parse(src)
	if err != nil {
		return nil, VisibilityRule{}, err
	}
	body := mjml.Body(root)
	var found *mjml.Node
	var rule VisibilityRule
	body.Walk(func(n *mjml.Node, _ []int) {
		if found != nil {
			return
		}
		if r, ok := attrRule(n); ok && r.ID == id {
			found, rule = n, r
		}
	})
	if found == nil {
		return nil, VisibilityRule{}, ErrNoBlock
	}
	return found, rule, nil
}

// Fix applies a Visibility fix to an email's MJML and returns the new
// source with the label of the version it saves and the line Review & send
// shows, in English.
func Fix(src, action string, params map[string]string) (next, label, applied string, err error) {
	return FixIn(i18n.Default, src, action, params)
}

// FixIn is Fix with the line Review & send shows in a reader's language.
// The version's label stays in English: it joins the other fixes' labels
// in the version's note.
func FixIn(l i18n.Lang, src, action string, params map[string]string) (next, label, applied string, err error) {
	n, rule, err := findRule(src, params["block"])
	if err != nil {
		return src, "", "", err
	}
	name := n.Attr(mjml.AttrName)
	named := "no"
	if name != "" {
		named = "yes"
	}
	words := i18n.Args{"name": name, "named": named}
	var ok bool
	switch action {
	case FixShowEveryone:
		next, ok = mjml.RemoveAttr(src, n, mjml.AttrVisible)
		label, applied = "shown to everyone", mFixShowed.Text(l, words)
	case FixHideBlock:
		rule.Show = ShowNobody
		next, ok = mjml.SetAttr(src, n, mjml.AttrVisible, rule.Encode())
		label, applied = "hidden for everyone", mFixHid.Text(l, words)
	case FixRemoveCondition:
		i, perr := strconv.Atoi(params["condition"])
		if perr != nil || i < 0 || i >= len(rule.Conditions) {
			return src, "", "", errors.New("that condition isn't in the rule any more; run the checks again")
		}
		rule.Conditions = append(rule.Conditions[:i:i], rule.Conditions[i+1:]...)
		if len(rule.Conditions) == 0 {
			// Without a condition the block is shown to everyone.
			next, ok = mjml.RemoveAttr(src, n, mjml.AttrVisible)
		} else {
			next, ok = mjml.SetAttr(src, n, mjml.AttrVisible, rule.Encode())
		}
		label, applied = "condition removed", mFixRemoved.Text(l, words)
	default:
		return src, "", "", errors.New("unknown fix")
	}
	if !ok {
		return src, "", "", ErrNoBlock
	}
	return next, label, applied, nil
}
