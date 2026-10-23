package authz

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The words of the permission catalogue are in messages/<locale>.json: each
// permission's label (perm.<key>) and description (perm.<key>.about), the
// groups of the role editor (group.<id>), the built-in, client and agency
// roles' descriptions (role.<key>.about) and the phrases that say what a
// missing permission stops ("Sending needs the campaigns:send permission").
// Defs, Groups and the roles keep the English; LabelIn, DescriptionIn,
// GroupIn and RoleDescriptionIn write them in a reader's language.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "authz")

var (
	mDenied     = msgs.M("denied")
	mDeniedAsk  = msgs.M("deniedAsk")
	mActionThis = msgs.M("action.this")
)

// g is a permission group's English name, from its id.
func g(id string) string { return msgs.M("group." + id).Source() }

// groupIDs maps the groups' English names (Def.Group) back to their ids.
var groupIDs = func() map[string]string {
	out := map[string]string{}
	for _, k := range msgs.Keys() {
		if len(k) > len("group.") && k[:len("group.")] == "group." {
			out[msgs.M(k).Source()] = k[len("group."):]
		}
	}
	return out
}()

// described fills in the permissions' English labels and descriptions.
func described(ds []Def) []Def {
	for i := range ds {
		ds[i].Label = msgs.M("perm." + string(ds[i].Key)).Source()
		ds[i].Description = msgs.M("perm." + string(ds[i].Key) + ".about").Source()
	}
	return ds
}

// roleAbout is a built-in, client or agency role's English description.
func roleAbout(key string) string { return msgs.M("role." + key + ".about").Source() }

// LabelIn is a permission's label in a language: "Send or schedule campaigns".
func LabelIn(l i18n.Lang, p Permission) string {
	if m, err := msgs.Lookup("perm." + string(p)); err == nil {
		return m.Text(l, nil)
	}
	return string(p)
}

// DescriptionIn is a permission's description in a language.
func DescriptionIn(l i18n.Lang, p Permission) string {
	if m, err := msgs.Lookup("perm." + string(p) + ".about"); err == nil {
		return m.Text(l, nil)
	}
	return ""
}

// GroupIn is a permission group's name (Def.Group, as Groups lists them)
// in a language: "Lists and segments".
func GroupIn(l i18n.Lang, group string) string {
	if id, ok := groupIDs[group]; ok {
		return msgs.M("group."+id).Text(l, nil)
	}
	return group
}

// RoleDescriptionIn is a built-in, client or agency role's description in
// a language, or "" for any other role.
func RoleDescriptionIn(l i18n.Lang, key string) string {
	if m, err := msgs.Lookup("role." + key + ".about"); err == nil {
		return m.Text(l, nil)
	}
	return ""
}

// DeniedText explains a missing permission in a language: "Sending needs
// the campaigns:send permission. Ask Marcus Chen or Linh Nguyen." Helpers
// are the names of members who hold it.
func DeniedText(l i18n.Lang, perm Permission, helpers []string) string {
	action := mActionThis
	if _, ok := Lookup(perm); ok {
		if m, err := msgs.Lookup("action." + string(perm)); err == nil {
			action = m
		}
	}
	args := i18n.Args{"action": action.Text(l, nil), "permission": string(perm)}
	if len(helpers) > 0 {
		args["helpers"] = i18n.OrList(l.Locale, helpers)
		return mDeniedAsk.Text(l, args)
	}
	return mDenied.Text(l, args)
}
