package automations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

// DiffItem is one line of a version diff: what was added, changed or removed,
// and a last line listing what stayed the same.
type DiffItem struct {
	Kind    string   `json:"kind" enum:"added,changed,removed,unchanged"`
	Title   string   `json:"title"`
	Detail  string   `json:"detail,omitempty"`
	NodeIDs []string `json:"nodeIds"`
}

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
}

// Changes counts the diff items that are changes (not the unchanged line).
func Changes(items []DiffItem) int {
	n := 0
	for _, it := range items {
		if it.Kind != "unchanged" {
			n++
		}
	}
	return n
}

// Diff compares two versions of a flow, step by step: steps keep their ids
// across versions, so a step is changed when its settings differ, added or
// removed when its id appears or disappears, and rewired when it now leads
// somewhere else. Positions on the canvas don't count as changes.
func Diff(from, to FlowGraph, dc DiffContext) []DiffItem {
	if dc.ToRefs == nil {
		dc.ToRefs = dc.FromRefs
	}
	if dc.FromRefs == nil {
		dc.FromRefs = dc.ToRefs
	}
	if dc.Loc == nil {
		dc.Loc = time.UTC
	}
	inFrom := map[string]FlowNode{}
	for _, n := range from.Nodes {
		inFrom[n.ID] = n
	}
	inTo := map[string]FlowNode{}
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
	var items []DiffItem
	changedNodes := map[string]bool{}

	// Added steps, grouped into connected clusters.
	for _, cl := range clusters(to, added) {
		items = append(items, describeCluster(to, cl, "added", dc, dc.ToRefs))
	}
	// Changed steps, in canvas order.
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
	// Removed steps.
	for _, cl := range clusters(from, removed) {
		items = append(items, describeCluster(from, cl, "removed", dc, dc.FromRefs))
	}
	if u := unchangedLine(to, added, changedNodes, dc.ToRefs); u.Title != "" {
		items = append(items, u)
	}
	if items == nil {
		items = []DiffItem{}
	}
	return items
}

// sameStep compares type and settings, ignoring the canvas position and
// read-only fields.
func sameStep(a, b FlowNode) bool {
	if a.Type != b.Type {
		return false
	}
	return bytes.Equal(normData(a.Data), normData(b.Data))
}

func normData(d FlowNodeData) []byte {
	if d.Webhook != nil {
		w := *d.Webhook
		w.HasSecret = false
		d.Webhook = &w
	}
	b, _ := json.Marshal(d)
	return b
}

// successors lists "path→target" for each edge out of id, following edges
// through skipped steps until a kept step (or the end of the flow).
func successors(g FlowGraph, id string, skip map[string]bool) []string {
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
func clusters(g FlowGraph, set map[string]bool) [][]string {
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
func quotedRef(g FlowGraph, id string, refs *Refs) string {
	n := g.Node(id)
	if n == nil {
		return "a removed step"
	}
	if n.Type == NodeTrigger {
		return "the trigger"
	}
	return `"` + NodeRef(g, *n, refs) + `"`
}

func weights(n FlowNode) string {
	parts := make([]string, 0, len(n.Data.Variants))
	for _, v := range n.Data.Variants {
		parts = append(parts, fmt.Sprint(v.Weight))
	}
	return strings.Join(parts, " / ")
}

// headline describes a step on its own ("A/B split", `Webhook "notify-slack"`).
func headline(g FlowGraph, n FlowNode, refs *Refs) string {
	switch n.Type {
	case NodeTrigger:
		return "Trigger: " + TriggerLabel(n.Data.Trigger, refs)
	case NodeSplit:
		return "A/B split"
	case NodeDelay:
		return "Delay of " + DurationText(n.Data.Delay)
	case NodeEmail:
		return fmt.Sprintf("Email %s", quote(NodeRef(g, n, refs)))
	case NodeWebhook:
		return fmt.Sprintf("Webhook %s", quote(NodeTitle(g, n, refs)))
	case NodeBranch:
		return fmt.Sprintf("Branch %s", quote(NodeTitle(g, n, refs)))
	case NodeGoal:
		return fmt.Sprintf("Goal %s", quote(NodeTitle(g, n, refs)))
	case NodeExit:
		return "Exit"
	}
	return NodeTitle(g, n, refs)
}

// emailItemRef is how a new email reads in a list: `3A "Set up your first flag"`.
func emailItemRef(g FlowGraph, n FlowNode, refs *Refs) string {
	name := emailNodeName(n, refs)
	ref := NodeRef(g, n, refs)
	if n.Data.Label == "" {
		// A variant after a split takes its letter: Step 3 + A → 3A.
		for _, e := range g.In(n.ID) {
			if src := g.Node(e.Source); src != nil && src.Type == NodeSplit && e.Data.Path != "" {
				ref += e.Data.Path
				if strings.HasPrefix(ref, "Step ") {
					ref = strings.TrimPrefix(ref, "Step ")
				}
				break
			}
		}
	}
	if q := emailQuoted(name); q != "" && q != ref {
		return fmt.Sprintf("%s %s", ref, quote(q))
	}
	return fmt.Sprintf("%s", quote(ref))
}

var numberWords = []string{"No", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine"}

func countWord(n int) string {
	if n >= 0 && n < len(numberWords) {
		return numberWords[n]
	}
	return fmt.Sprint(n)
}

func andJoin(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

func formatCount(n int64) string {
	s := fmt.Sprint(n)
	if n < 0 {
		return "-" + formatCount(-n)
	}
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func dayMonth(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("2 Jan")
}

func describeCluster(g FlowGraph, ids []string, kind string, dc DiffContext, refs *Refs) DiffItem {
	root := g.Node(ids[0])
	item := DiffItem{Kind: kind, NodeIDs: ids}
	title := headline(g, *root, refs)
	if root.Type != NodeTrigger {
		if prev := g.Prev(root.ID); prev != "" {
			title += " after " + quotedRef(g, prev, refs)
		}
	}
	if root.Type == NodeSplit {
		title += ", " + weights(*root)
	}
	item.Title = title

	var details []string
	var emails []FlowNode
	var others []string
	for _, id := range ids {
		n := g.Node(id)
		switch {
		case n.Type == NodeEmail:
			emails = append(emails, *n)
		case id != root.ID:
			others = append(others, strings.ToLower(TypeName(n.Type)))
		}
	}
	if kind == "added" {
		listed := emails
		if root.Type == NodeEmail && len(emails) > 0 {
			listed = emails[1:]
		}
		if len(listed) > 0 {
			refsList := make([]string, 0, len(listed))
			for _, e := range listed {
				refsList = append(refsList, emailItemRef(g, e, refs))
			}
			noun := "new email step"
			if len(listed) > 1 {
				noun += "s"
			}
			details = append(details, fmt.Sprintf("%s %s: %s.", countWord(len(listed)), noun, andJoin(refsList)))
		}
		if len(others) > 0 {
			details = append(details, "Also adds "+andJoin(others)+".")
		}
		if s := checkSentence(g, emails, dc, refs); s != "" {
			details = append(details, s)
		}
	} else {
		for _, id := range ids {
			n := g.Node(id)
			if n.Type == NodeWebhook {
				if f, ok := dc.Failures[id]; ok && f.Failures > 0 {
					status := "errors"
					if f.Status > 0 {
						status = fmt.Sprint(f.Status)
					}
					url := f.URL
					if url == "" && n.Data.Webhook != nil {
						url = n.Data.Webhook.URL
					}
					details = append(details, fmt.Sprintf("Endpoint %s was returning %s since %s. %s failed calls are in the webhook log.",
						shortURL(url), status, dayMonth(f.Since, dc.Loc), formatCount(f.Failures)))
				}
			}
		}
		if len(emails) > 0 && len(details) == 0 {
			details = append(details, "Contacts who haven't reached it yet won't get it.")
		}
	}
	item.Detail = strings.Join(details, " ")
	return item
}

// checkSentence reports Send Readiness content checks of new emails.
func checkSentence(g FlowGraph, emails []FlowNode, dc DiffContext, refs *Refs) string {
	if dc.Check == nil || len(emails) == 0 {
		return ""
	}
	var failing []string
	for _, e := range emails {
		cfg := e.Data.Email
		if cfg == nil {
			continue
		}
		if problem := dc.Check(cfg.EmailID, cfg.EmailVersion); problem != "" {
			failing = append(failing, fmt.Sprintf("%s: %s", quote(NodeRef(g, e, refs)), problem))
		}
	}
	if len(failing) > 0 {
		return "Fails Send Readiness content checks: " + strings.Join(failing, "; ") + "."
	}
	switch len(emails) {
	case 1:
		return "It passes Send Readiness content checks."
	case 2:
		return "Both pass Send Readiness content checks."
	}
	return fmt.Sprintf("All %d pass Send Readiness content checks.", len(emails))
}

func describeChange(from, to FlowGraph, a, b FlowNode, dc DiffContext) DiffItem {
	item := DiffItem{Kind: "changed", NodeIDs: []string{b.ID}}
	refsA, refsB := dc.FromRefs, dc.ToRefs
	ref := quotedRef(to, b.ID, refsB)
	switch b.Type {
	case NodeDelay:
		target := ""
		if next := to.Next(b.ID); next != "" {
			target = "before " + quotedRef(to, next, refsB)
		} else if prev := to.Prev(b.ID); prev != "" {
			target = "after " + quotedRef(to, prev, refsB)
		}
		item.Title = strings.TrimSpace(fmt.Sprintf("Delay %s: %s → %s", target, DurationText(a.Data.Delay), DurationText(b.Data.Delay)))
		item.Detail = fmt.Sprintf("Contacts already waiting keep their original timer unless you move them to v%d below.", dc.ToVersion)
	case NodeEmail:
		ea, eb := a.Data.Email, b.Data.Email
		if ea == nil {
			ea = &EmailStepConfig{}
		}
		if eb == nil {
			eb = &EmailStepConfig{}
		}
		switch {
		case ea.EmailID != eb.EmailID:
			item.Title = fmt.Sprintf("%s now sends %s", ref, quote(NodeTitle(to, b, refsB)))
		case ea.EmailVersion != eb.EmailVersion:
			item.Title = fmt.Sprintf("New content for %s: v%d → v%d", ref, ea.EmailVersion, eb.EmailVersion)
		case ea.Subject != eb.Subject:
			item.Title = fmt.Sprintf("Subject of %s: %s → %s", ref, quote(ea.Subject), quote(eb.Subject))
		case ea.Route != eb.Route:
			route := "the Automations route"
			if refsB != nil && refsB.Providers[eb.Route] != "" {
				route = refsB.Providers[eb.Route] + " only"
			}
			item.Title = fmt.Sprintf("%s now sends through %s", ref, route)
		case a.Data.Label != b.Data.Label:
			item.Title = fmt.Sprintf("Step renamed: %s → %s", quote(NodeRef(from, a, refsA)), quote(NodeRef(to, b, refsB)))
		default:
			item.Title = fmt.Sprintf("Sending rules of %s changed", ref)
		}
		item.Detail = "Contacts who already got this email won't get it again."
	case NodeTrigger:
		item.Title = fmt.Sprintf("Trigger: %s → %s", TriggerLabel(a.Data.Trigger, refsA), TriggerLabel(b.Data.Trigger, refsB))
		item.Detail = "Contacts already in the flow stay; new entrants come from the new trigger."
	case NodeBranch:
		item.Title = fmt.Sprintf("Branch: %s → %s", quote(NodeTitle(from, a, refsA)), quote(NodeTitle(to, b, refsB)))
		item.Detail = fmt.Sprintf("Contacts waiting at the branch are checked against the new condition if you move them to v%d.", dc.ToVersion)
	case NodeSplit:
		item.Title = fmt.Sprintf("A/B split: %s → %s", weights(a), weights(b))
	case NodeWebhook:
		wa, wb := a.Data.Webhook, b.Data.Webhook
		switch {
		case wa != nil && wb != nil && wa.URL != wb.URL:
			item.Title = fmt.Sprintf("Webhook %s now calls %s", quote(NodeTitle(to, b, refsB)), shortURL(wb.URL))
		default:
			item.Title = fmt.Sprintf("Webhook %s changed", quote(NodeTitle(to, b, refsB)))
		}
	case NodeGoal:
		item.Title = fmt.Sprintf("Goal: %s → %s", quote(NodeTitle(from, a, refsA)), quote(NodeTitle(to, b, refsB)))
		if a.Data.Label != b.Data.Label && NodeTitle(from, a, refsA) == NodeTitle(to, b, refsB) {
			item.Title = fmt.Sprintf("Goal renamed: %s → %s", quote(GoalLabel(a, refsA)), quote(GoalLabel(b, refsB)))
		}
	default:
		before, after := NodeTitle(from, a, refsA), NodeTitle(to, b, refsB)
		if before != after {
			item.Title = fmt.Sprintf("%s: %s → %s", TypeName(b.Type), quote(before), quote(after))
		} else {
			item.Title = fmt.Sprintf("%s %s changed", TypeName(b.Type), ref)
		}
	}
	return item
}

func describeRewire(to FlowGraph, n FlowNode, after []string, dc DiffContext) DiffItem {
	refs := dc.ToRefs
	item := DiffItem{Kind: "changed", NodeIDs: []string{n.ID}}
	var parts []string
	for _, s := range after {
		path, target, _ := strings.Cut(s, "→")
		dest := "the end of the flow"
		if target != "" {
			dest = quotedRef(to, target, refs)
		}
		if path != "" {
			parts = append(parts, fmt.Sprintf("the %s path goes to %s", quote(pathLabel(n, path)), dest))
		} else {
			parts = append(parts, "contacts go to "+dest)
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "the flow now ends")
	}
	item.Title = fmt.Sprintf("After %s, %s", quotedRef(to, n.ID, refs), andJoin(parts))
	return item
}

// unchangedLine lists the steps that stayed the same: the trigger and the
// steps before the first fork, every branch path with no change at all (as
// `"No" path`), then goals and exits. Delays and steps on changed paths are
// left out; they're covered by the lines above.
func unchangedLine(g FlowGraph, added, changed map[string]bool, refs *Refs) DiffItem {
	item := DiffItem{Kind: "unchanged", NodeIDs: []string{}}
	untouched := func(id string) bool { return !added[id] && !changed[id] }
	var parts []string
	var ids []string
	var terminals []FlowNode
	listed := map[string]bool{}
	var visitPath func(start string) (members []string, clean bool)
	visitPath = func(start string) ([]string, bool) {
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
	var spine func(id string)
	spine = func(id string) {
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
						parts = append(parts, fmt.Sprintf("%s path", quote(pathLabel(*n, p))))
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
		if n.Terminal() && !slices.ContainsFunc(terminals, func(x FlowNode) bool { return x.ID == id }) {
			terminals = append(terminals, *n)
		}
	}
	sort.SliceStable(terminals, func(i, j int) bool { return terminals[i].Type == NodeGoal && terminals[j].Type != NodeGoal })
	for _, t := range terminals {
		if !untouched(t.ID) {
			continue
		}
		name := "exit"
		if t.Type == NodeGoal {
			name = "goal"
		}
		if !slices.Contains(parts, name) {
			parts = append(parts, name)
		}
		ids = append(ids, t.ID)
	}
	if len(parts) > 0 {
		parts[0] = capFirst(parts[0])
	}
	item.Title = andJoin(parts)
	item.NodeIDs = ids
	return item
}

func unchangedName(g FlowGraph, n FlowNode, refs *Refs) string {
	switch n.Type {
	case NodeTrigger:
		return "trigger"
	case NodeEmail:
		if n.Data.Label != "" {
			return n.Data.Label
		}
		if name := emailNodeName(n, refs); name != "" {
			return lowerFirst(EmailShortRef(name))
		}
		return "email"
	case NodeBranch:
		return "branch"
	case NodeSplit:
		return "A/B split"
	case NodeWebhook:
		return fmt.Sprintf("webhook %s", quote(NodeTitle(g, n, refs)))
	case NodeList:
		return lowerFirst(NodeTitle(g, n, refs))
	case NodeFilter:
		return "filter"
	}
	return strings.ToLower(TypeName(n.Type))
}
