package automations

import (
	"bytes"
	"encoding/json"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// AutomationDiffItem is one line of a version diff: what was added, changed or removed,
// and a last line listing what stayed the same.
type AutomationDiffItem struct {
	Kind    string   `json:"kind" enum:"added,changed,removed,unchanged"`
	Title   string   `json:"title"`
	Detail  string   `json:"detail,omitempty"`
	NodeIDs []string `json:"nodeIds"`
	// TitleText and DetailText are the title and detail as texts, for
	// readers in other languages (TextIn writes them).
	TitleText  i18n.Text `json:"-"`
	DetailText i18n.Text `json:"-"`
}

// TextIn writes one of the package's texts (a diff line's TitleText) in a
// reader's language.
func TextIn(l i18n.Lang, t i18n.Text) string { return say(l, t) }

// WebhookFailure is what the diff says about a removed webhook that was failing.
type WebhookFailure struct {
	URL      string
	Status   int
	Since    time.Time
	Failures int64
}

// DiffContext carries the facts the diff mentions besides the two graphs.
type DiffContext struct {
	FromRefs  *Refs
	ToRefs    *Refs
	ToVersion int
	Loc       *time.Location
	// Failures of webhook steps, by node id.
	Failures map[string]WebhookFailure
	// Check runs content checks on an email version: "" when it passes,
	// otherwise the first problem ("no unsubscribe link").
	Check func(emailID string, version int) string
	// Lang is the language the lines are written in (English when unset).
	Lang i18n.Lang
	// checkText is Check with the problem as a text (the service's checks).
	checkText func(emailID string, version int) i18n.Text
}

var (
	mDiffHeadTrigger   = msg("diff.headTrigger")
	mDiffHeadSplit     = msg("diff.headSplit")
	mDiffHeadDelay     = msg("diff.headDelay")
	mDiffHeadEmail     = msg("diff.headEmail")
	mDiffHeadWebhook   = msg("diff.headWebhook")
	mDiffHeadBranch    = msg("diff.headBranch")
	mDiffHeadGoal      = msg("diff.headGoal")
	mDiffHeadExit      = msg("diff.headExit")
	mDiffAfter         = msg("diff.after")
	mDiffAfterSplit    = msg("diff.afterSplit")
	mDiffSplit         = msg("diff.split")
	mDiffEmailRef      = msg("diff.emailRef")
	mDiffNewEmails     = msg("diff.newEmails")
	mDiffAlsoAdds      = msg("diff.alsoAdds")
	mDiffChecksFail    = msg("diff.checksFail")
	mDiffCheckFailure  = msg("diff.checkFailure")
	mDiffChecksPass    = msg("diff.checksPass")
	mDiffWebhookFail   = msg("diff.webhookFailing")
	mDiffRemovedEmail  = msg("diff.removedEmail")
	mDiffDelayBefore   = msg("diff.delayBefore")
	mDiffDelayAfter    = msg("diff.delayAfter")
	mDiffDelay         = msg("diff.delay")
	mDiffDelayDetail   = msg("diff.delayDetail")
	mDiffEmailNowSends = msg("diff.emailNowSends")
	mDiffEmailContent  = msg("diff.emailContent")
	mDiffEmailSubject  = msg("diff.emailSubject")
	mDiffEmailRoute    = msg("diff.emailRoute")
	mDiffRouteAuto     = msg("diff.routeAutomations")
	mDiffRouteOnly     = msg("diff.routeOnly")
	mDiffStepRenamed   = msg("diff.stepRenamed")
	mDiffSendingRules  = msg("diff.sendingRules")
	mDiffEmailDetail   = msg("diff.emailDetail")
	mDiffTrigger       = msg("diff.trigger")
	mDiffTriggerDetail = msg("diff.triggerDetail")
	mDiffBranch        = msg("diff.branch")
	mDiffBranchDetail  = msg("diff.branchDetail")
	mDiffSplitWeights  = msg("diff.splitWeights")
	mDiffWebhookURL    = msg("diff.webhookUrl")
	mDiffWebhook       = msg("diff.webhook")
	mDiffGoal          = msg("diff.goal")
	mDiffGoalRenamed   = msg("diff.goalRenamed")
	mDiffStepTitle     = msg("diff.stepTitle")
	mDiffStepChanged   = msg("diff.stepChanged")
	mDiffRewire        = msg("diff.rewire")
	mDiffPathGoesTo    = msg("diff.pathGoesTo")
	mDiffContactsGoTo  = msg("diff.contactsGoTo")
	mDiffFlowEnds      = msg("diff.flowEnds")
	mDiffEndOfFlow     = msg("diff.endOfFlow")
	mRefQuoted         = msg("ref.quoted")

	mUnchangedTrigger = msg("unchanged.trigger")
	mUnchangedEmail   = msg("unchanged.email")
	mUnchangedBranch  = msg("unchanged.branch")
	mUnchangedSplit   = msg("unchanged.split")
	mUnchangedFilter  = msg("unchanged.filter")
	mUnchangedWebhook = msg("unchanged.webhook")
	mUnchangedPath    = msg("unchanged.path")
	mUnchangedGoal    = msg("unchanged.goal")
	mUnchangedExit    = msg("unchanged.exit")
)

// Changes counts the diff items that are changes (not the unchanged line).
func Changes(items []AutomationDiffItem) int {
	n := 0
	for _, it := range items {
		if it.Kind != "unchanged" {
			n++
		}
	}
	return n
}

// diffIn writes a diff's lines in a language.
func diffIn(l i18n.Lang, items []AutomationDiffItem) []AutomationDiffItem {
	out := make([]AutomationDiffItem, len(items))
	for i, it := range items {
		it.Title, it.Detail = say(l, it.TitleText), say(l, it.DetailText)
		out[i] = it
	}
	return out
}

// Diff compares two versions of a flow, step by step: steps keep their ids
// across versions, so a step is changed when its settings differ, added or
// removed when its id appears or disappears, and rewired when it now leads
// somewhere else. Positions on the canvas don't count as changes.
func Diff(from, to AutomationGraph, dc DiffContext) []AutomationDiffItem {
	if dc.ToRefs == nil {
		dc.ToRefs = dc.FromRefs
	}
	if dc.FromRefs == nil {
		dc.FromRefs = dc.ToRefs
	}
	if dc.Loc == nil {
		dc.Loc = time.UTC
	}
	inFrom := map[string]AutomationNode{}
	for _, n := range from.Nodes {
		inFrom[n.ID] = n
	}
	inTo := map[string]AutomationNode{}
	for _, n := range to.Nodes {
		inTo[n.ID] = n
	}
	added, removed := map[string]bool{}, map[string]bool{}
	for id := range inTo {
		if _, ok := inFrom[id]; !ok {
			added[id] = true
		}
	}
	for id := range inFrom {
		if _, ok := inTo[id]; !ok {
			removed[id] = true
		}
	}
	var items []AutomationDiffItem
	changedNodes := map[string]bool{}

	for _, cl := range clusters(to, added) {
		items = append(items, describeCluster(to, cl, "added", dc, dc.ToRefs))
	}
	for _, id := range to.Order() {
		a, okA := inFrom[id]
		b, okB := inTo[id]
		if !okA || !okB {
			continue
		}
		if !sameStep(a, b) {
			changedNodes[id] = true
			items = append(items, describeChange(from, to, a, b, dc))
		}
	}
	// Rewired steps: compare where each kept step leads, looking through
	// added and removed steps.
	for _, id := range to.Order() {
		if _, ok := inFrom[id]; !ok {
			continue
		}
		before := successors(from, id, removed)
		after := successors(to, id, added)
		if slices.Equal(before, after) {
			continue
		}
		n := inTo[id]
		changedNodes[id] = true
		items = append(items, describeRewire(to, n, after, dc))
	}
	for _, cl := range clusters(from, removed) {
		items = append(items, describeCluster(from, cl, "removed", dc, dc.FromRefs))
	}
	if u, ok := unchangedLine(to, added, changedNodes, dc.ToRefs); ok {
		items = append(items, u)
	}
	if items == nil {
		items = []AutomationDiffItem{}
	}
	return diffIn(dc.Lang, items)
}

// sameStep compares type and settings, ignoring the canvas position and
// read-only fields.
func sameStep(a, b AutomationNode) bool {
	if a.Type != b.Type {
		return false
	}
	return bytes.Equal(normData(a.Data), normData(b.Data))
}

func normData(d AutomationNodeData) []byte {
	if d.Webhook != nil {
		w := *d.Webhook
		w.HasSecret = false
		w.Demo = false
		d.Webhook = &w
	}
	b, _ := json.Marshal(d)
	return b
}

// successors lists "path→target" for each edge out of id, following edges
// through skipped steps until a kept step (or the end of the flow).
func successors(g AutomationGraph, id string, skip map[string]bool) []string {
	var out []string
	seen := map[string]bool{}
	var follow func(path, node string)
	follow = func(path, node string) {
		if !skip[node] {
			out = append(out, path+"→"+node)
			return
		}
		if seen[node] {
			return
		}
		seen[node] = true
		next := g.Out(node)
		if len(next) == 0 {
			out = append(out, path+"→")
		}
		for _, e := range next {
			follow(path, e.Target)
		}
	}
	for _, e := range g.Out(id) {
		follow(e.Data.Path, e.Target)
	}
	sort.Strings(out)
	return slices.Compact(out)
}

// clusters groups the given steps into connected components, each ordered
// from the step closest to the trigger.
func clusters(g AutomationGraph, set map[string]bool) [][]string {
	depth := g.Depths()
	parent := map[string]string{}
	var find func(string) string
	find = func(x string) string {
		if parent[x] == "" || parent[x] == x {
			parent[x] = x
			return x
		}
		parent[x] = find(parent[x])
		return parent[x]
	}
	for id := range set {
		find(id)
	}
	for _, e := range g.Edges {
		if set[e.Source] && set[e.Target] {
			parent[find(e.Source)] = find(e.Target)
		}
	}
	groups := map[string][]string{}
	for id := range set {
		r := find(id)
		groups[r] = append(groups[r], id)
	}
	var out [][]string
	for _, ids := range groups {
		sort.Slice(ids, func(i, j int) bool {
			if depth[ids[i]] != depth[ids[j]] {
				return depth[ids[i]] < depth[ids[j]]
			}
			return ids[i] < ids[j]
		})
		out = append(out, ids)
	}
	sort.Slice(out, func(i, j int) bool {
		if depth[out[i][0]] != depth[out[j][0]] {
			return depth[out[i][0]] < depth[out[j][0]]
		}
		return out[i][0] < out[j][0]
	})
	return out
}

// quotedRef names a step for a diff line: "the trigger" or "Step 2" in quotes.
func quotedRef(g AutomationGraph, id string, refs *Refs) i18n.Text {
	n := g.Node(id)
	if n == nil {
		return txt(mRefRemovedStep)
	}
	if n.Type == NodeTrigger {
		return txt(mRefTrigger)
	}
	return txt(mRefQuoted, "step", nodeRef(g, *n, refs))
}

// headline describes a step on its own ("A/B split", `Webhook "notify-slack"`).
func headline(g AutomationGraph, n AutomationNode, refs *Refs) i18n.Text {
	switch n.Type {
	case NodeTrigger:
		return txt(mDiffHeadTrigger, "trigger", triggerLabel(n.Data.Trigger, refs))
	case NodeSplit:
		return txt(mDiffHeadSplit)
	case NodeDelay:
		return txt(mDiffHeadDelay, "duration", durationText(n.Data.Delay))
	case NodeEmail:
		return txt(mDiffHeadEmail, "step", nodeRef(g, n, refs))
	case NodeWebhook:
		return txt(mDiffHeadWebhook, "step", nodeTitle(g, n, refs))
	case NodeBranch:
		return txt(mDiffHeadBranch, "step", nodeTitle(g, n, refs))
	case NodeGoal:
		return txt(mDiffHeadGoal, "step", nodeTitle(g, n, refs))
	case NodeExit:
		return txt(mDiffHeadExit)
	}
	return nodeTitle(g, n, refs)
}

// emailItemRef is how a new email reads in a list: `3A "Set up your first flag"`.
func emailItemRef(g AutomationGraph, n AutomationNode, refs *Refs) i18n.Text {
	nm := emailNodeName(n, refs)
	ref := nodeRef(g, n, refs)
	if n.Data.Label == "" && nm != "" {
		// A variant after a split takes its letter: Step 3 + A → 3A.
		for _, e := range g.In(n.ID) {
			if src := g.Node(e.Source); src != nil && src.Type == NodeSplit && e.Data.Path != "" {
				ref = name(strings.TrimPrefix(say(english, ref)+e.Data.Path, "Step "))
				break
			}
		}
	}
	if q := emailQuoted(nm); q != "" && q != say(english, ref) {
		return txt(mDiffEmailRef, "step", ref, "subject", q)
	}
	return txt(mRefQuoted, "step", ref)
}

// dayMonth is a day without its year in the reader's region format, in the
// workspace's zone: "29 Sep".
func dayMonth(t time.Time, loc *time.Location) i18n.Text {
	return i18n.Text{Args: i18n.Args{"dayMonth": t.In(loc)}}
}

// clock is a time of day in the reader's region format: "09:02".
func clock(t time.Time, loc *time.Location) i18n.Text {
	return i18n.Text{Args: i18n.Args{"clock": t.In(loc)}}
}

func describeCluster(g AutomationGraph, ids []string, kind string, dc DiffContext, refs *Refs) AutomationDiffItem {
	root := g.Node(ids[0])
	item := AutomationDiffItem{Kind: kind, NodeIDs: ids}
	title := headline(g, *root, refs)
	prev := ""
	if root.Type != NodeTrigger {
		prev = g.Prev(root.ID)
	}
	switch {
	case prev != "" && root.Type == NodeSplit:
		title = txt(mDiffAfterSplit, "step", title, "prev", quotedRef(g, prev, refs), "weights", splitWeights(*root))
	case prev != "":
		title = txt(mDiffAfter, "step", title, "prev", quotedRef(g, prev, refs))
	case root.Type == NodeSplit:
		title = txt(mDiffSplit, "step", title, "weights", splitWeights(*root))
	}
	item.TitleText = title

	var details []i18n.Text
	var emails []AutomationNode
	var others []i18n.Text
	for _, id := range ids {
		n := g.Node(id)
		switch {
		case n.Type == NodeEmail:
			emails = append(emails, *n)
		case id != root.ID:
			others = append(others, lowercase(typeName(n.Type)))
		}
	}
	if kind == "added" {
		listed := emails
		if root.Type == NodeEmail && len(emails) > 0 {
			listed = emails[1:]
		}
		if len(listed) > 0 {
			refsList := make([]i18n.Text, 0, len(listed))
			for _, e := range listed {
				refsList = append(refsList, emailItemRef(g, e, refs))
			}
			details = append(details, txt(mDiffNewEmails, "count", len(listed), "emails", andList(refsList)))
		}
		if len(others) > 0 {
			details = append(details, txt(mDiffAlsoAdds, "steps", andList(others)))
		}
		if s := checkSentence(g, emails, dc, refs); !emptyText(s) {
			details = append(details, s)
		}
	} else {
		for _, id := range ids {
			n := g.Node(id)
			if n.Type == NodeWebhook {
				if f, ok := dc.Failures[id]; ok && f.Failures > 0 {
					url := f.URL
					if url == "" && n.Data.Webhook != nil {
						url = n.Data.Webhook.URL
					}
					hasStatus := "no"
					if f.Status > 0 {
						hasStatus = "yes"
					}
					details = append(details, txt(mDiffWebhookFail, "url", shortURL(url), "hasStatus", hasStatus, "status", itoaPlain(f.Status),
						"date", dayMonth(f.Since, dc.Loc), "count", f.Failures))
				}
			}
		}
		if len(emails) > 0 && len(details) == 0 {
			details = append(details, txt(mDiffRemovedEmail))
		}
	}
	item.DetailText = joined(" ", details)
	return item
}

// itoaPlain writes a code or a version as it is, without grouping: 404.
func itoaPlain(n int) string { return strconv.Itoa(n) }

// checkSentence reports Send Readiness content checks of new emails.
func checkSentence(g AutomationGraph, emails []AutomationNode, dc DiffContext, refs *Refs) i18n.Text {
	if (dc.Check == nil && dc.checkText == nil) || len(emails) == 0 {
		return i18n.Text{}
	}
	var failing []i18n.Text
	for _, e := range emails {
		cfg := e.Data.Email
		if cfg == nil {
			continue
		}
		var problem i18n.Text
		if dc.checkText != nil {
			problem = dc.checkText(cfg.EmailID, cfg.EmailVersion)
		} else if p := dc.Check(cfg.EmailID, cfg.EmailVersion); p != "" {
			problem = name(p)
		}
		if !emptyText(problem) {
			failing = append(failing, txt(mDiffCheckFailure, "step", nodeRef(g, e, refs), "problem", problem))
		}
	}
	if len(failing) > 0 {
		return txt(mDiffChecksFail, "failures", joined("; ", failing))
	}
	return txt(mDiffChecksPass, "count", len(emails))
}

func describeChange(from, to AutomationGraph, a, b AutomationNode, dc DiffContext) AutomationDiffItem {
	item := AutomationDiffItem{Kind: "changed", NodeIDs: []string{b.ID}}
	refsA, refsB := dc.FromRefs, dc.ToRefs
	ref := quotedRef(to, b.ID, refsB)
	switch b.Type {
	case NodeDelay:
		was, now := durationText(a.Data.Delay), durationText(b.Data.Delay)
		if next := to.Next(b.ID); next != "" {
			item.TitleText = txt(mDiffDelayBefore, "step", quotedRef(to, next, refsB), "from", was, "to", now)
		} else if prev := to.Prev(b.ID); prev != "" {
			item.TitleText = txt(mDiffDelayAfter, "step", quotedRef(to, prev, refsB), "from", was, "to", now)
		} else {
			item.TitleText = txt(mDiffDelay, "from", was, "to", now)
		}
		item.DetailText = txt(mDiffDelayDetail, "version", itoaPlain(dc.ToVersion))
	case NodeEmail:
		ea, eb := a.Data.Email, b.Data.Email
		if ea == nil {
			ea = &AutomationEmailStep{}
		}
		if eb == nil {
			eb = &AutomationEmailStep{}
		}
		switch {
		case ea.EmailID != eb.EmailID:
			item.TitleText = txt(mDiffEmailNowSends, "step", ref, "email", nodeTitle(to, b, refsB))
		case ea.EmailVersion != eb.EmailVersion:
			item.TitleText = txt(mDiffEmailContent, "step", ref, "from", itoaPlain(ea.EmailVersion), "to", itoaPlain(eb.EmailVersion))
		case ea.Subject != eb.Subject:
			item.TitleText = txt(mDiffEmailSubject, "step", ref, "from", ea.Subject, "to", eb.Subject)
		case ea.Route != eb.Route:
			route := txt(mDiffRouteAuto)
			if refsB != nil && refsB.Providers[eb.Route] != "" {
				route = txt(mDiffRouteOnly, "provider", refsB.Providers[eb.Route])
			}
			item.TitleText = txt(mDiffEmailRoute, "step", ref, "route", route)
		case a.Data.Label != b.Data.Label:
			item.TitleText = txt(mDiffStepRenamed, "from", nodeRef(from, a, refsA), "to", nodeRef(to, b, refsB))
		default:
			item.TitleText = txt(mDiffSendingRules, "step", ref)
		}
		item.DetailText = txt(mDiffEmailDetail)
	case NodeTrigger:
		item.TitleText = txt(mDiffTrigger, "from", triggerLabel(a.Data.Trigger, refsA), "to", triggerLabel(b.Data.Trigger, refsB))
		item.DetailText = txt(mDiffTriggerDetail)
	case NodeBranch:
		item.TitleText = txt(mDiffBranch, "from", nodeTitle(from, a, refsA), "to", nodeTitle(to, b, refsB))
		item.DetailText = txt(mDiffBranchDetail, "version", itoaPlain(dc.ToVersion))
	case NodeSplit:
		item.TitleText = txt(mDiffSplitWeights, "from", splitWeights(a), "to", splitWeights(b))
	case NodeWebhook:
		wa, wb := a.Data.Webhook, b.Data.Webhook
		switch {
		case wa != nil && wb != nil && wa.URL != wb.URL:
			item.TitleText = txt(mDiffWebhookURL, "step", nodeTitle(to, b, refsB), "url", shortURL(wb.URL))
		default:
			item.TitleText = txt(mDiffWebhook, "step", nodeTitle(to, b, refsB))
		}
	case NodeGoal:
		item.TitleText = txt(mDiffGoal, "from", nodeTitle(from, a, refsA), "to", nodeTitle(to, b, refsB))
		if a.Data.Label != b.Data.Label && NodeTitle(from, a, refsA) == NodeTitle(to, b, refsB) {
			item.TitleText = txt(mDiffGoalRenamed, "from", goalLabel(a, refsA), "to", goalLabel(b, refsB))
		}
	default:
		if NodeTitle(from, a, refsA) != NodeTitle(to, b, refsB) {
			item.TitleText = txt(mDiffStepTitle, "type", typeName(b.Type), "from", nodeTitle(from, a, refsA), "to", nodeTitle(to, b, refsB))
		} else {
			item.TitleText = txt(mDiffStepChanged, "type", typeName(b.Type), "step", ref)
		}
	}
	return item
}

func describeRewire(to AutomationGraph, n AutomationNode, after []string, dc DiffContext) AutomationDiffItem {
	refs := dc.ToRefs
	item := AutomationDiffItem{Kind: "changed", NodeIDs: []string{n.ID}}
	var parts []i18n.Text
	for _, s := range after {
		path, target, _ := strings.Cut(s, "→")
		dest := txt(mDiffEndOfFlow)
		if target != "" {
			dest = quotedRef(to, target, refs)
		}
		if path != "" {
			parts = append(parts, txt(mDiffPathGoesTo, "path", pathLabel(n, path), "dest", dest))
		} else {
			parts = append(parts, txt(mDiffContactsGoTo, "dest", dest))
		}
	}
	if len(parts) == 0 {
		parts = append(parts, txt(mDiffFlowEnds))
	}
	item.TitleText = txt(mDiffRewire, "step", quotedRef(to, n.ID, refs), "changes", andList(parts))
	return item
}

// unchangedLine lists the steps that stayed the same: the trigger and the
// steps before the first fork, every branch path with no change at all (as
// `"No" path`), then goals and exits. Delays and steps on changed paths are
// left out; they're covered by the lines above. It's false when nothing
// stayed the same.
func unchangedLine(g AutomationGraph, added, changed map[string]bool, refs *Refs) (AutomationDiffItem, bool) {
	item := AutomationDiffItem{Kind: "unchanged", NodeIDs: []string{}}
	untouched := func(id string) bool { return !added[id] && !changed[id] }
	var parts []i18n.Text
	var ids []string
	var terminals []AutomationNode
	listed := map[string]bool{}
	visitPath := func(start string) ([]string, bool) {
		var members []string
		clean := true
		seen := map[string]bool{}
		var walk func(string)
		walk = func(id string) {
			if id == "" || seen[id] {
				return
			}
			n := g.Node(id)
			if n == nil {
				return
			}
			// A merge point belongs to whatever comes after the fork.
			if len(g.In(id)) > 1 {
				return
			}
			seen[id] = true
			if !untouched(id) {
				clean = false
			}
			if !n.Terminal() {
				members = append(members, id)
			}
			for _, e := range g.Out(id) {
				walk(e.Target)
			}
		}
		walk(start)
		return members, clean
	}
	spine := func(id string) {
		for id != "" {
			n := g.Node(id)
			if n == nil || listed[id] {
				return
			}
			listed[id] = true
			if n.Terminal() {
				terminals = append(terminals, *n)
				return
			}
			if untouched(id) && n.Type != NodeDelay {
				parts = append(parts, unchangedName(g, *n, refs))
				ids = append(ids, id)
			}
			if n.Type == NodeBranch || n.Type == NodeSplit {
				for _, p := range n.Paths() {
					start := g.NextOnPath(id, p)
					members, clean := visitPath(start)
					if clean && len(members) > 0 {
						parts = append(parts, txt(mUnchangedPath, "path", pathLabel(*n, p)))
						ids = append(ids, members...)
					}
					for _, m := range members {
						listed[m] = true
					}
				}
				return
			}
			id = g.Next(id)
		}
	}
	if t := g.Trigger(); t != nil {
		spine(t.ID)
	}
	for _, id := range g.Order() {
		n := g.Node(id)
		if n.Terminal() && !slices.ContainsFunc(terminals, func(x AutomationNode) bool { return x.ID == id }) {
			terminals = append(terminals, *n)
		}
	}
	sort.SliceStable(terminals, func(i, j int) bool { return terminals[i].Type == NodeGoal && terminals[j].Type != NodeGoal })
	hasGoal, hasExit := false, false
	for _, t := range terminals {
		if !untouched(t.ID) {
			continue
		}
		switch {
		case t.Type == NodeGoal && !hasGoal:
			parts = append(parts, txt(mUnchangedGoal))
			hasGoal = true
		case t.Type != NodeGoal && !hasExit:
			parts = append(parts, txt(mUnchangedExit))
			hasExit = true
		}
		ids = append(ids, t.ID)
	}
	if len(parts) == 0 {
		return item, false
	}
	item.TitleText = upper(andList(parts))
	item.NodeIDs = ids
	return item, true
}

func unchangedName(g AutomationGraph, n AutomationNode, refs *Refs) i18n.Text {
	switch n.Type {
	case NodeTrigger:
		return txt(mUnchangedTrigger)
	case NodeEmail:
		if n.Data.Label != "" {
			return name(n.Data.Label)
		}
		if nm := emailNodeName(n, refs); nm != "" {
			return lower(name(EmailShortRef(nm)))
		}
		return txt(mUnchangedEmail)
	case NodeBranch:
		return txt(mUnchangedBranch)
	case NodeSplit:
		return txt(mUnchangedSplit)
	case NodeWebhook:
		return txt(mUnchangedWebhook, "step", nodeTitle(g, n, refs))
	case NodeList:
		return lower(nodeTitle(g, n, refs))
	case NodeFilter:
		return txt(mUnchangedFilter)
	}
	return lowercase(typeName(n.Type))
}
