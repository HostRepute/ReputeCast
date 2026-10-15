// Package integrations is the integrations hub (design expansion, phase 2,
// M16): the catalog of Settings › Integrations, connections to Zapier, Make,
// n8n and WordPress sites with one key each, the REST hooks those apps
// subscribe with, the actions they call, the 7-day call log, managed
// webhooks, health, and the WordPress site API.
//
// An integration key is an API key with integration_id set. It works only
// for the operations of the triggers and actions turned on, even where its
// permissions would allow more: internal/api asks Key for that list on every
// call (see api.Integrations) and logs the call through Record.
package integrations

import (
	"slices"

	"github.com/hostrepute/reputecast/internal/authz"
)

// Kinds of connections.
const (
	KindZapier    = "zapier"
	KindMake      = "make"
	KindN8n       = "n8n"
	KindWordPress = "wordpress"
)

// Operation IDs the integrations call. Those of other packages are listed
// by name; an unknown ID in a list is harmless.
const (
	OpMe            = "get-integration"
	OpSubscribe     = "subscribe-integration-hook"
	OpUnsubscribe   = "unsubscribe-integration-hook"
	OpSample        = "get-integration-trigger-sample"
	OpLists         = "list-integration-lists"
	OpFields        = "list-integration-fields"
	OpUpsert        = "upsert-contact"
	OpAddToList     = "add-contact-to-list"
	OpTag           = "tag-contact"
	OpEvent         = "send-automation-event" // internal/automations
	OpTemplates     = "list-integration-templates"
	OpSendMessage   = "send-transactional-message" // internal/transactional (M15)
	OpWPReport      = "report-wordpress-site"
	OpWPForms       = "list-wordpress-forms"
	OpWPDraft       = "create-wordpress-draft"
	transactionalOp = OpSendMessage
)

// Item is a trigger or an action an integration can be given.
type Item struct {
	Key  string
	Kind string // trigger | action
	// Event is the webhook event of a trigger.
	Event string
	// Permissions the key needs for it; ⚑ ones start off when connecting.
	Permissions []authz.Permission
	// Operations it calls.
	Operations []string
	// New marks items added in this release ("New" badge).
	New bool
}

// Sensitive reports whether the item needs a ⚑ permission.
func (it Item) Sensitive() bool {
	for _, p := range it.Permissions {
		if authz.IsSensitive(p) {
			return true
		}
	}
	return false
}

// Item kinds.
const (
	ItemTrigger = "trigger"
	ItemAction  = "action"
)

var hookOps = []string{OpSubscribe, OpUnsubscribe, OpSample}

// platformItems are the triggers and actions of Zapier, Make and n8n
// (brief 3.2), in the order of the boards.
var platformItems = []Item{
	{Key: "contact.subscribed", Kind: ItemTrigger, Event: "contact.subscribed", Permissions: []authz.Permission{authz.SubscribersView}, Operations: hookOps},
	{Key: "contact.unsubscribed", Kind: ItemTrigger, Event: "contact.unsubscribed", Permissions: []authz.Permission{authz.SubscribersView}, Operations: hookOps},
	{Key: "contact.updated", Kind: ItemTrigger, Event: "contact.updated", Permissions: []authz.Permission{authz.SubscribersView}, Operations: hookOps},
	{Key: "form.submitted", Kind: ItemTrigger, Event: "form.submitted", Permissions: []authz.Permission{authz.SubscribersView}, Operations: hookOps, New: true},
	{Key: "campaign.completed", Kind: ItemTrigger, Event: "campaign.completed", Permissions: []authz.Permission{authz.CampaignsView}, Operations: hookOps},
	{Key: "message.bounced", Kind: ItemTrigger, Event: "message.bounced", Permissions: []authz.Permission{authz.TransactionalView}, Operations: hookOps},
	{Key: "upsert_contact", Kind: ItemAction, Permissions: []authz.Permission{authz.SubscribersCreate, authz.SubscribersEdit}, Operations: []string{OpUpsert, OpLists, OpFields}},
	{Key: "add_to_list", Kind: ItemAction, Permissions: []authz.Permission{authz.SubscribersEdit}, Operations: []string{OpAddToList, OpLists}},
	{Key: "add_tag", Kind: ItemAction, Permissions: []authz.Permission{authz.SubscribersEdit}, Operations: []string{OpTag}},
	{Key: "record_event", Kind: ItemAction, Permissions: []authz.Permission{authz.SubscribersEdit}, Operations: []string{OpEvent}},
	{Key: "send_transactional", Kind: ItemAction, Permissions: []authz.Permission{authz.TransactionalSend}, Operations: []string{transactionalOp, OpTemplates}},
}

// wordpressItems are what a WordPress site's key can do (brief 3.3).
var wordpressItems = []Item{
	{Key: "list_forms", Kind: ItemAction, Permissions: []authz.Permission{authz.SubscribersView}, Operations: []string{OpWPForms}},
	{Key: "create_drafts", Kind: ItemAction, Permissions: []authz.Permission{authz.CampaignsEdit}, Operations: []string{OpWPDraft}},
}

// alwaysOps are what every key of a kind may call, whatever is turned on.
var alwaysOps = map[string][]string{
	KindZapier:    {OpMe},
	KindMake:      {OpMe},
	KindN8n:       {OpMe},
	KindWordPress: {OpMe, OpWPReport},
}

// Items returns the triggers and actions of a kind.
func Items(kind string) []Item {
	if kind == KindWordPress {
		return wordpressItems
	}
	if IsPlatform(kind) {
		return platformItems
	}
	return nil
}

// ItemByKey returns one item of a kind.
func ItemByKey(kind, key string) (Item, bool) {
	for _, it := range Items(kind) {
		if it.Key == key {
			return it, true
		}
	}
	return Item{}, false
}

// IsPlatform reports whether kind is an automation platform.
func IsPlatform(kind string) bool {
	return kind == KindZapier || kind == KindMake || kind == KindN8n
}

// Valid reports whether kind can be connected.
func Valid(kind string) bool { return IsPlatform(kind) || kind == KindWordPress }

// Name is how people call a kind.
func Name(kind string) string {
	switch kind {
	case KindZapier:
		return "Zapier"
	case KindMake:
		return "Make"
	case KindN8n:
		return "n8n"
	case KindWordPress:
		return "WordPress"
	}
	return kind
}

// Operations returns the operation IDs a key with these items may call.
func Operations(kind string, items []string) map[string]bool {
	ops := map[string]bool{}
	for _, op := range alwaysOps[kind] {
		ops[op] = true
	}
	for _, it := range Items(kind) {
		if slices.Contains(items, it.Key) {
			for _, op := range it.Operations {
				ops[op] = true
			}
		}
	}
	return ops
}

// Permissions returns the permissions a key with these items holds, in the
// order of the catalog: the items' permissions, then webhooks:manage for
// the endpoints it creates when a trigger is on.
func Permissions(kind string, items []string) []authz.Permission {
	var out []authz.Permission
	add := func(p authz.Permission) {
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	triggers := false
	for _, it := range Items(kind) {
		if !slices.Contains(items, it.Key) {
			continue
		}
		if it.Kind == ItemTrigger {
			triggers = true
		}
		for _, p := range it.Permissions {
			add(p)
		}
	}
	// subscribers:edit includes subscribers:view: keep the key's list short.
	set := authz.NewSet(out...)
	short := out[:0:0]
	for _, p := range out {
		implied := false
		for _, q := range out {
			if q != p && authz.NewSet(q).Has(p) && set.Has(q) {
				implied = true
			}
		}
		if !implied {
			short = append(short, p)
		}
	}
	if triggers {
		short = append(short, authz.WebhooksManage)
	}
	return short
}

// Needs returns every permission an item needs from the person who turns it
// on: its own, plus webhooks:manage for triggers.
func Needs(it Item) []authz.Permission {
	out := slices.Clone(it.Permissions)
	if it.Kind == ItemTrigger {
		out = append(out, authz.WebhooksManage)
	}
	return out
}

// DefaultItems are what starts on when connecting: everything except items
// that need a ⚑ permission.
func DefaultItems(kind string) []string {
	out := []string{}
	for _, it := range Items(kind) {
		if !it.Sensitive() {
			out = append(out, it.Key)
		}
	}
	return out
}

// Group is a group of the catalog, in display order.
type Group struct {
	Key string
	// Kinds of entries the group lists. A group with nothing in it isn't
	// shown; later phases add AI models, verification, payments and commerce.
	Entries []string
}

// Groups of the catalog (brief 3.1). Phase 2 fills the first four.
var Groups = []Group{
	{Key: "automation", Entries: []string{KindZapier, KindMake, KindN8n}},
	{Key: "websites", Entries: []string{KindWordPress}},
	{Key: "deliverability", Entries: []string{"hostrepute"}},
	{Key: "developer", Entries: []string{"api", "webhooks", "mcp"}},
	{Key: "ai"},
	{Key: "verification"},
	{Key: "payments"},
	{Key: "commerce"},
}
