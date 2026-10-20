package automations

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Node types, as named in the step palette.
const (
	NodeTrigger = "trigger"
	NodeDelay   = "delay"
	NodeEmail   = "email"
	NodeBranch  = "branch"
	NodeFilter  = "filter"
	NodeSplit   = "split"
	NodeWebhook = "webhook"
	NodeList    = "list"
	NodeGoal    = "goal"
	NodeExit    = "exit"
)

// Trigger kinds.
const (
	TriggerJoinedList    = "joined_list"
	TriggerFormSubmitted = "form_submitted"
	TriggerTagAdded      = "tag_added"
	TriggerAPIEvent      = "api_event"
	TriggerDateField     = "date_field"
	// TriggerScoreCrosses starts when a contact's score rises to a
	// threshold or falls below it (contact scoring, see scoretrigger.go).
	TriggerScoreCrosses = "score_crosses"
	// TriggerMilestoneReached starts when a referrer reaches a referral
	// milestone (Grow), once per contact per milestone.
	TriggerMilestoneReached = "milestone_reached"
)

// Branch paths.
const (
	PathYes = "yes"
	PathNo  = "no"
)

// AutomationGraph is an automation as React Flow stores it: typed nodes with a
// position and data, connected by edges. Branch and split edges name the path
// they take ("yes"/"no", or the variant key).
type AutomationGraph struct {
	Nodes []AutomationNode `json:"nodes" maxItems:"200"`
	Edges []AutomationEdge `json:"edges" maxItems:"400"`
}

// AutomationNode is one step of a flow.
type AutomationNode struct {
	ID       string                 `json:"id" minLength:"1" maxLength:"40" pattern:"^[A-Za-z0-9_-]+$"`
	Type     string                 `json:"type" enum:"trigger,delay,email,branch,filter,split,webhook,list,goal,exit,sms"`
	Position AutomationNodePosition `json:"position"`
	Data     AutomationNodeData     `json:"data"`
}

// AutomationNodePosition is a node's top-left corner on the canvas.
type AutomationNodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// AutomationEdge connects one step to the next.
type AutomationEdge struct {
	ID     string             `json:"id" minLength:"1" maxLength:"80"`
	Source string             `json:"source" minLength:"1" maxLength:"40"`
	Target string             `json:"target" minLength:"1" maxLength:"40"`
	Data   AutomationEdgeData `json:"data"`
}

// AutomationEdgeData says which path out of a branch or split the edge is.
type AutomationEdgeData struct {
	// Path is "yes" or "no" after a branch, the variant key after a split.
	Path string `json:"path,omitempty" maxLength:"16"`
}

// AutomationNodeData holds a step's settings; which block is set depends on the type.
type AutomationNodeData struct {
	// Label is a short name used where the step is referenced (diffs, stats, goals).
	Label     string                   `json:"label,omitempty" maxLength:"80"`
	Trigger   *AutomationTrigger       `json:"trigger,omitempty"`
	Delay     *AutomationDuration      `json:"delay,omitempty"`
	Email     *AutomationEmailStep     `json:"email,omitempty"`
	Condition *AutomationCondition     `json:"condition,omitempty" doc:"Branch, filter and goal condition"`
	Variants  []AutomationSplitVariant `json:"variants,omitempty" maxItems:"5"`
	Webhook   *AutomationWebhook       `json:"webhook,omitempty"`
	Action    *AutomationListAction    `json:"action,omitempty"`
	// SMS is a "Send SMS" step (channels.go).
	SMS *AutomationSMSStep `json:"sms,omitempty"`
}

// AutomationTrigger says which contacts enter the flow.
type AutomationTrigger struct {
	Kind string `json:"kind" enum:"joined_list,form_submitted,tag_added,api_event,date_field,score_crosses,poll_answered,milestone_reached"`
	// Joined list: the list, whether to wait for double opt-in, and whether
	// contacts added by an import count.
	ListID         string `json:"listId,omitempty" maxLength:"40"`
	ConfirmedOnly  bool   `json:"confirmedOnly,omitempty"`
	IncludeImports bool   `json:"includeImports,omitempty"`
	FormID         string `json:"formId,omitempty" maxLength:"40"`
	Tag            string `json:"tag,omitempty" maxLength:"64"`
	Event          string `json:"event,omitempty" maxLength:"100"`
	// Date field: the contact field, days before (negative) or after the date,
	// whether it repeats every year, and the local time ("09:00").
	Field      string `json:"field,omitempty" maxLength:"64"`
	OffsetDays int    `json:"offsetDays,omitempty" minimum:"-365" maximum:"365"`
	Yearly     bool   `json:"yearly,omitempty"`
	At         string `json:"at,omitempty" maxLength:"5"`
	// Reentry: "once" (default) lets a contact through the flow one time;
	// "always" lets them enter again after they leave.
	Reentry string `json:"reentry,omitempty" maxLength:"8"`
	// Score crosses: the score rises to Threshold or more, or falls below
	// it, at most once per contact in OnceInDays days (0: no limit).
	Direction  string `json:"direction,omitempty" enum:"rises,falls"`
	Threshold  int    `json:"threshold,omitempty" minimum:"0" maximum:"100"`
	OnceInDays int    `json:"onceInDays,omitempty" minimum:"0" maximum:"3650"`
	// Answered a poll or survey (polls.go): the poll or survey ("poll:<id>",
	// "survey:<id>"), the answers that start it (keys, with is or is not),
	// and whether a changed answer starts it again.
	Poll     string `json:"poll,omitempty" maxLength:"60"`
	AnswerOp string `json:"answerOp,omitempty" enum:"any,is,is_not"`
	Answers  []int  `json:"answers,omitempty" maxItems:"11"`
	OnChange bool   `json:"onChange,omitempty"`
	// Reached a milestone (Grow): the referral program and its milestone,
	// or any of its milestones when the milestone is empty (any program's
	// when both are).
	ProgramID   string `json:"programId,omitempty" maxLength:"40"`
	MilestoneID string `json:"milestoneId,omitempty" maxLength:"40"`
}

// AutomationDuration is a delay or a time window.
type AutomationDuration struct {
	Amount int    `json:"amount" minimum:"0" maximum:"100000"`
	Unit   string `json:"unit" enum:"minutes,hours,days,weeks"`
}

// AutomationEmailStep is a "Send email" step.
type AutomationEmailStep struct {
	EmailID      string `json:"emailId,omitempty" maxLength:"40"`
	EmailVersion int    `json:"emailVersion,omitempty" minimum:"0"`
	Subject      string `json:"subject,omitempty" maxLength:"250"`
	// Route is empty for the Automations route, or a sending provider id to override it.
	Route      string                `json:"route,omitempty" maxLength:"40"`
	SendWindow *AutomationSendWindow `json:"sendWindow,omitempty"`
	// SkipRecentHours skips the email when the contact got any email in the last N hours (0 = off).
	SkipRecentHours int `json:"skipRecentHours,omitempty" minimum:"0" maximum:"720"`
	// SkipListID skips the email for members of this list.
	SkipListID string `json:"skipListId,omitempty" maxLength:"40"`
}

// AutomationSendWindow holds email until the recipient's local time is inside it.
type AutomationSendWindow struct {
	Enabled bool   `json:"enabled"`
	Start   string `json:"start" maxLength:"5" doc:"HH:MM, recipient-local"`
	End     string `json:"end" maxLength:"5"`
}

// AutomationCondition is what a branch asks, a filter keeps or a goal waits for.
type AutomationCondition struct {
	Kind string `json:"kind" enum:"opened,clicked,field,tag,list,event,sms_consent,sms_clicked,sms_replied"`
	// Opened / clicked: the email step (empty = any email in this flow), the
	// window and whether machine opens (Apple Mail Privacy Protection) count.
	Step        string              `json:"step,omitempty" maxLength:"40"`
	Within      *AutomationDuration `json:"within,omitempty"`
	HumanOnly   bool                `json:"humanOnly,omitempty"`
	URLContains string              `json:"urlContains,omitempty" maxLength:"200"`
	// Field: compare a contact field.
	Field string `json:"field,omitempty" maxLength:"64"`
	Op    string `json:"op,omitempty" maxLength:"16" doc:"equals, not_equals, contains, is_set, is_not_set, is_one_of (comma-separated values)"`
	Value string `json:"value,omitempty" maxLength:"200"`
	Tag   string `json:"tag,omitempty" maxLength:"64"`
	// List: member of this list (subscribed).
	ListID string `json:"listId,omitempty" maxLength:"40"`
	// Event: an API event arrived for the contact.
	Event string `json:"event,omitempty" maxLength:"100"`
}

// AutomationSplitVariant is one arm of an A/B split.
type AutomationSplitVariant struct {
	Key    string `json:"key" minLength:"1" maxLength:"4"`
	Weight int    `json:"weight" minimum:"0" maximum:"100"`
}

// AutomationWebhook is an outgoing HTTP call. The signing secret is stored
// separately and never returned; HasSecret says whether one is set.
type AutomationWebhook struct {
	Name      string `json:"name,omitempty" maxLength:"80"`
	URL       string `json:"url,omitempty" maxLength:"2000"`
	Method    string `json:"method,omitempty" maxLength:"6"`
	HasSecret bool   `json:"hasSecret,omitempty" readOnly:"true"`
	// Demo marks a step of the demo data: it is never called (its address is
	// fictional), until someone points it at another URL.
	Demo bool `json:"demo,omitempty" readOnly:"true"`
}

// AutomationListAction adds or removes a list or tag.
type AutomationListAction struct {
	Kind   string `json:"kind" enum:"add_list,remove_list,add_tag,remove_tag,unsubscribe_all"`
	ListID string `json:"listId,omitempty" maxLength:"40"`
	Tag    string `json:"tag,omitempty" maxLength:"64"`
}

// ParseGraph decodes a stored graph. A null or empty document is an empty graph.
func ParseGraph(raw []byte) (AutomationGraph, error) {
	g := AutomationGraph{Nodes: []AutomationNode{}, Edges: []AutomationEdge{}}
	if len(raw) == 0 || string(raw) == "null" {
		return g, nil
	}
	if err := json.Unmarshal(raw, &g); err != nil {
		return AutomationGraph{}, err
	}
	if g.Nodes == nil {
		g.Nodes = []AutomationNode{}
	}
	if g.Edges == nil {
		g.Edges = []AutomationEdge{}
	}
	return g, nil
}

// MustJSON encodes a graph for storage.
func (g AutomationGraph) MustJSON() json.RawMessage {
	b, err := json.Marshal(g)
	if err != nil {
		panic(err)
	}
	return b
}

// Node returns the node with id, or nil.
func (g AutomationGraph) Node(id string) *AutomationNode {
	for i := range g.Nodes {
		if g.Nodes[i].ID == id {
			return &g.Nodes[i]
		}
	}
	return nil
}

// Trigger returns the trigger node, or nil.
func (g AutomationGraph) Trigger() *AutomationNode {
	for i := range g.Nodes {
		if g.Nodes[i].Type == NodeTrigger {
			return &g.Nodes[i]
		}
	}
	return nil
}

// Out returns the edges leaving id in the order they were drawn.
func (g AutomationGraph) Out(id string) []AutomationEdge {
	var out []AutomationEdge
	for _, e := range g.Edges {
		if e.Source == id {
			out = append(out, e)
		}
	}
	return out
}

// In returns the edges entering id.
func (g AutomationGraph) In(id string) []AutomationEdge {
	var out []AutomationEdge
	for _, e := range g.Edges {
		if e.Target == id {
			out = append(out, e)
		}
	}
	return out
}

// Next returns the single next node of a non-branching step ("" at the end).
func (g AutomationGraph) Next(id string) string {
	for _, e := range g.Edges {
		if e.Source == id {
			return e.Target
		}
	}
	return ""
}

// NextOnPath returns the node reached through path out of a branch or split.
func (g AutomationGraph) NextOnPath(id, path string) string {
	for _, e := range g.Edges {
		if e.Source == id && e.Data.Path == path {
			return e.Target
		}
	}
	return ""
}

// Prev returns the first node that leads to id ("" when none).
func (g AutomationGraph) Prev(id string) string {
	for _, e := range g.Edges {
		if e.Target == id {
			return e.Source
		}
	}
	return ""
}

// Order returns node ids breadth-first from the trigger, then any node not
// reachable from it, so lists read top to bottom like the canvas.
func (g AutomationGraph) Order() []string {
	seen := map[string]bool{}
	var out []string
	var queue []string
	if t := g.Trigger(); t != nil {
		queue = append(queue, t.ID)
		seen[t.ID] = true
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		out = append(out, id)
		for _, e := range g.Out(id) {
			if !seen[e.Target] && g.Node(e.Target) != nil {
				seen[e.Target] = true
				queue = append(queue, e.Target)
			}
		}
	}
	for _, n := range g.Nodes {
		if !seen[n.ID] {
			out = append(out, n.ID)
		}
	}
	return out
}

// Depths returns each node's distance from the trigger (longest path), used
// to stagger the canvas entrance and to order steps.
func (g AutomationGraph) Depths() map[string]int {
	d := map[string]int{}
	order := g.topo()
	for _, id := range order {
		if _, ok := d[id]; !ok {
			d[id] = 0
		}
		for _, e := range g.Out(id) {
			if d[id]+1 > d[e.Target] {
				d[e.Target] = d[id] + 1
			}
		}
	}
	return d
}

// topo returns a topological order (Kahn); nodes on cycles are appended last.
func (g AutomationGraph) topo() []string {
	indeg := map[string]int{}
	for _, n := range g.Nodes {
		indeg[n.ID] += 0
	}
	for _, e := range g.Edges {
		if g.Node(e.Source) != nil && g.Node(e.Target) != nil {
			indeg[e.Target]++
		}
	}
	var queue, out []string
	for _, n := range g.Nodes {
		if indeg[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}
	done := map[string]bool{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		out = append(out, id)
		done[id] = true
		for _, e := range g.Out(id) {
			indeg[e.Target]--
			if indeg[e.Target] == 0 {
				queue = append(queue, e.Target)
			}
		}
	}
	for _, n := range g.Nodes {
		if !done[n.ID] {
			out = append(out, n.ID)
		}
	}
	return out
}

// HasCycle reports whether the graph loops back on itself.
func (g AutomationGraph) HasCycle() bool {
	const (
		white = iota
		grey
		black
	)
	color := map[string]int{}
	var visit func(string) bool
	visit = func(id string) bool {
		color[id] = grey
		for _, e := range g.Out(id) {
			switch color[e.Target] {
			case grey:
				return true
			case white:
				if visit(e.Target) {
					return true
				}
			}
		}
		color[id] = black
		return false
	}
	for _, n := range g.Nodes {
		if color[n.ID] == white && visit(n.ID) {
			return true
		}
	}
	return false
}

// Paths returns the paths a branch or split must connect.
func (n AutomationNode) Paths() []string {
	switch n.Type {
	case NodeBranch:
		return []string{PathYes, PathNo}
	case NodeSplit:
		out := make([]string, 0, len(n.Data.Variants))
		for _, v := range n.Data.Variants {
			out = append(out, v.Key)
		}
		return out
	}
	return nil
}

// Terminal reports whether a step ends the flow.
func (n AutomationNode) Terminal() bool { return n.Type == NodeGoal || n.Type == NodeExit }

func (d AutomationDuration) Duration() time.Duration {
	n := time.Duration(d.Amount)
	switch d.Unit {
	case "minutes":
		return n * time.Minute
	case "hours":
		return n * time.Hour
	case "weeks":
		return n * 7 * 24 * time.Hour
	default:
		return n * 24 * time.Hour
	}
}

// AutomationGraphIssue is a problem that blocks publishing (severity "error") or is
// worth a look ("warning").
type AutomationGraphIssue struct {
	NodeID   string `json:"nodeId,omitempty"`
	EdgeID   string `json:"edgeId,omitempty"`
	Code     string `json:"code"`
	Severity string `json:"severity" enum:"error,warning"`
	Message  string `json:"message"`
}

// Refs are the workspace objects steps may point to; Validate reports steps
// that reference something missing (a deleted list, form, email or field).
type Refs struct {
	Lists      map[string]string // id → name
	Forms      map[string]string // id → slug
	Emails     map[string]int    // id → latest version
	EmailNames map[string]string // id → name
	Fields     map[string]string // key → type
	Providers  map[string]string // id → name
	Polls      map[string]string // "poll:<id>" or "survey:<id>" → name
	// Milestones are Grow's referral milestones: "3 referrals · The Acme
	// Field Guide".
	Milestones map[string]string
}

var (
	reHHMM  = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
	reEvent = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:-]{0,99}$`)
	reTag   = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N} _.:/-]{0,63}$`)
)

// Validate checks the structure (one trigger, every step reachable, every
// branch path connected, no loops) and each step's settings. refs may be nil
// to skip reference checks.
func Validate(g AutomationGraph, refs *Refs) []AutomationGraphIssue {
	issues := []AutomationGraphIssue{}
	add := func(node, edge, code, sev, msg string) {
		issues = append(issues, AutomationGraphIssue{NodeID: node, EdgeID: edge, Code: code, Severity: sev, Message: msg})
	}
	if len(g.Nodes) == 0 {
		add("", "", "empty", "error", "The flow is empty. Drag a trigger onto the canvas to start.")
		return issues
	}
	ids := map[string]bool{}
	triggers := 0
	for _, n := range g.Nodes {
		if ids[n.ID] {
			add(n.ID, "", "duplicate_id", "error", fmt.Sprintf("Two steps share the id %s. Delete one and add it again.", quote(n.ID)))
		}
		ids[n.ID] = true
		if n.Type == NodeTrigger {
			triggers++
		}
	}
	switch {
	case triggers == 0:
		add("", "", "no_trigger", "error", "The flow has no trigger. Drag one from Triggers so contacts can enter.")
	case triggers > 1:
		add("", "", "many_triggers", "error", "A flow has one trigger. Delete the extra ones.")
	}
	edgeKeys := map[string]bool{}
	for _, e := range g.Edges {
		if !ids[e.Source] || !ids[e.Target] {
			add("", e.ID, "dangling_edge", "error", "A connection points to a step that no longer exists. Delete it.")
			continue
		}
		if e.Source == e.Target {
			add(e.Source, e.ID, "self_loop", "error", "A step can't lead to itself.")
		}
		key := e.Source + "→" + e.Target + "/" + e.Data.Path
		if edgeKeys[key] {
			add(e.Source, e.ID, "duplicate_edge", "error", "Two connections join the same steps. Delete one.")
		}
		edgeKeys[key] = true
	}
	if g.HasCycle() {
		add("", "", "cycle", "error", "The flow loops back to an earlier step. Flows run top to bottom; remove the connection that goes back up.")
	}
	if t := g.Trigger(); t != nil {
		if len(g.In(t.ID)) > 0 {
			add(t.ID, "", "trigger_input", "error", "Nothing can lead into the trigger. Remove the connection into it.")
		}
		reach := map[string]bool{}
		var walk func(string)
		walk = func(id string) {
			if reach[id] {
				return
			}
			reach[id] = true
			for _, e := range g.Out(id) {
				walk(e.Target)
			}
		}
		walk(t.ID)
		for _, n := range g.Nodes {
			if !reach[n.ID] {
				add(n.ID, "", "orphan", "error", fmt.Sprintf("%s isn't connected to the trigger, so no contact would reach it. Connect it or delete it.", capFirst(stepName(g, n, refs))))
			}
		}
	}
	for _, n := range g.Nodes {
		out := g.Out(n.ID)
		name := capFirst(stepName(g, n, refs))
		switch {
		case n.Terminal():
			if len(out) > 0 {
				add(n.ID, "", "terminal_output", "error", fmt.Sprintf("%s ends the flow, so it can't lead to another step.", name))
			}
		case n.Type == NodeBranch || n.Type == NodeSplit:
			have := map[string]int{}
			for _, e := range out {
				have[e.Data.Path]++
			}
			for _, p := range n.Paths() {
				if have[p] == 0 {
					add(n.ID, "", "missing_path", "error", fmt.Sprintf("%s has no %s path. Connect it to a step or an exit.", name, quote(pathLabel(n, p))))
				} else if have[p] > 1 {
					add(n.ID, "", "duplicate_path", "error", fmt.Sprintf("%s has two %s paths. Keep one.", name, quote(pathLabel(n, p))))
				}
			}
			for _, e := range out {
				if !slices.Contains(n.Paths(), e.Data.Path) {
					add(n.ID, e.ID, "unknown_path", "error", fmt.Sprintf("A connection out of %s isn't one of its paths. Delete it and connect again.", stepName(g, n, refs)))
				}
			}
		default:
			if len(out) > 1 {
				add(n.ID, "", "many_outputs", "error", fmt.Sprintf("%s leads to %d steps. Only branches and splits fork; keep one connection.", name, len(out)))
			}
		}
		for _, msg := range checkConfig(n, refs) {
			add(n.ID, "", msg.code, "error", msg.text)
		}
	}
	return issues
}

type configProblem struct{ code, text string }

func checkConfig(n AutomationNode, refs *Refs) []configProblem {
	var out []configProblem
	bad := func(code, format string, a ...any) { out = append(out, configProblem{code, fmt.Sprintf(format, a...)}) }
	listExists := func(id string) bool {
		if refs == nil {
			return true
		}
		_, ok := refs.Lists[id]
		return ok
	}
	d := n.Data
	switch n.Type {
	case NodeTrigger:
		t := d.Trigger
		if t == nil || t.Kind == "" {
			bad("trigger_kind", "Pick what starts the flow: a list join, a form, a tag, an API event, a date or a score.")
			break
		}
		switch t.Kind {
		case TriggerJoinedList:
			if t.ListID == "" {
				bad("missing_ref", "Pick the list contacts join.")
			} else if !listExists(t.ListID) {
				bad("missing_ref", "The trigger's list was deleted. Pick another list.")
			}
		case TriggerFormSubmitted:
			if t.FormID == "" {
				bad("missing_ref", "Pick the signup form.")
			} else if refs != nil {
				if _, ok := refs.Forms[t.FormID]; !ok {
					bad("missing_ref", "The trigger's form was deleted. Pick another form.")
				}
			}
		case TriggerTagAdded:
			if !reTag.MatchString(t.Tag) {
				bad("invalid_tag", "Name the tag that starts the flow (letters, numbers, spaces and - _ . : /).")
			}
		case TriggerAPIEvent:
			if !reEvent.MatchString(t.Event) {
				bad("invalid_event", "Name the API event, like workspace.created (letters, numbers and . _ : -).")
			}
		case TriggerPollAnswered:
			if t.Poll == "" {
				bad("missing_ref", "Pick the poll or survey.")
			} else if refs != nil && refs.Polls != nil {
				if _, ok := refs.Polls[t.Poll]; !ok {
					bad("missing_ref", "The trigger's poll or survey was deleted. Pick another one.")
				}
			}
		case TriggerDateField:
			if t.Field == "" {
				bad("missing_ref", "Pick the date field.")
			} else if refs != nil && t.Field != "created_at" {
				if typ, ok := refs.Fields[t.Field]; !ok {
					bad("missing_ref", "The trigger's date field was deleted. Pick another field.")
				} else if typ != "date" {
					bad("invalid_field", "The trigger needs a date field.")
				}
			}
			if t.At != "" && !reHHMM.MatchString(t.At) {
				bad("invalid_time", "Use a 24-hour time like 09:00.")
			}
		}
		if t.Kind == TriggerScoreCrosses {
			out = append(out, checkScoreTrigger(t)...)
		}
		if t.Kind == TriggerMilestoneReached && t.MilestoneID != "" && refs != nil && refs.Milestones != nil {
			if _, ok := refs.Milestones[t.MilestoneID]; !ok {
				bad("missing_ref", "The trigger's milestone was deleted. Pick another milestone.")
			}
		}
		if t.Reentry != "" && t.Reentry != "once" && t.Reentry != "always" {
			bad("invalid_reentry", "Re-entry is once or always.")
		}
	case NodeDelay:
		if d.Delay == nil || d.Delay.Amount <= 0 {
			bad("invalid_delay", "Set how long to wait.")
		} else if d.Delay.Duration() > 366*24*time.Hour {
			bad("invalid_delay", "Delays can be at most a year.")
		}
	case NodeEmail:
		e := d.Email
		if e == nil || e.EmailID == "" {
			bad("missing_email", "Pick the email to send.")
			break
		}
		if refs != nil {
			if _, ok := refs.Emails[e.EmailID]; !ok {
				bad("missing_ref", "This step's email was deleted. Pick another email.")
			}
			if e.Route != "" {
				if _, ok := refs.Providers[e.Route]; !ok {
					bad("missing_ref", "The sending provider this step overrides to was disconnected. Use the Automations route.")
				}
			}
			if e.SkipListID != "" && !listExists(e.SkipListID) {
				bad("missing_ref", "The \"Skip if\" list was deleted. Pick another list or clear the rule.")
			}
		}
		if strings.TrimSpace(e.Subject) == "" {
			bad("missing_subject", "Write a subject line.")
		}
		if w := e.SendWindow; w != nil && w.Enabled && (!reHHMM.MatchString(w.Start) || !reHHMM.MatchString(w.End) || w.Start == w.End) {
			bad("invalid_window", "Set the send window as two different times like 07:00 and 21:00.")
		}
	case NodeBranch, NodeFilter, NodeGoal:
		c := d.Condition
		if c == nil {
			if n.Type != NodeGoal {
				bad("missing_condition", "Set the condition.")
			}
			break
		}
		switch c.Kind {
		case "opened", "clicked":
			if n.Type == NodeBranch && (c.Within == nil || c.Within.Amount <= 0) {
				bad("invalid_window", "Set how long to wait for the open or click.")
			}
		case "field":
			if c.Field == "" {
				bad("missing_ref", "Pick the contact field to check.")
			} else if refs != nil && !isBuiltinField(c.Field) {
				if _, ok := refs.Fields[c.Field]; !ok {
					bad("missing_ref", "The condition's field was deleted. Pick another field.")
				}
			}
			if !slices.Contains([]string{"equals", "not_equals", "contains", "is_set", "is_not_set", "is_one_of"}, c.Op) {
				bad("invalid_op", "Pick how to compare the field.")
			}
		case "tag":
			if !reTag.MatchString(c.Tag) {
				bad("invalid_tag", "Name the tag to check.")
			}
		case "list":
			if c.ListID == "" {
				bad("missing_ref", "Pick the list to check.")
			} else if !listExists(c.ListID) {
				bad("missing_ref", "The condition's list was deleted. Pick another list.")
			}
		case "event":
			if !reEvent.MatchString(c.Event) {
				bad("invalid_event", "Name the API event, like teammate.invited.")
			}
		case CondSMSConsent:
			if c.ListID != "" && !listExists(c.ListID) {
				bad("missing_ref", "The condition's list was deleted. Pick another list.")
			}
		case CondSMSClicked, CondSMSReplied:
			if n.Type == NodeBranch && (c.Within == nil || c.Within.Amount <= 0) {
				bad("invalid_window", "Set how long to wait for the click or reply.")
			}
		}
	case NodeSplit:
		if len(d.Variants) < 2 {
			bad("invalid_split", "An A/B split needs at least two variants.")
			break
		}
		sum := 0
		keys := map[string]bool{}
		for _, v := range d.Variants {
			sum += v.Weight
			if keys[v.Key] {
				bad("invalid_split", "Each variant needs its own letter.")
			}
			keys[v.Key] = true
		}
		if sum != 100 {
			bad("invalid_split", "The split's weights add up to %d%%. Make them add up to 100%%.", sum)
		}
	case NodeWebhook:
		w := d.Webhook
		if w == nil || w.URL == "" {
			bad("missing_url", "Enter the URL to call.")
			break
		}
		if err := checkWebhookURL(w.URL); err != nil {
			bad("invalid_url", "%s", err.Error())
		}
		if w.Method != "" && w.Method != http.MethodPost && w.Method != http.MethodPut {
			bad("invalid_method", "Webhooks use POST or PUT.")
		}
	case NodeSMS:
		for _, p := range checkChannelNode(n, listExists) {
			bad(p.Code, "%s", p.Message)
		}
	case NodeList:
		a := d.Action
		if a == nil {
			bad("missing_action", "Pick what to do: add or remove a list or a tag.")
			break
		}
		switch a.Kind {
		case "add_list", "remove_list":
			if a.ListID == "" {
				bad("missing_ref", "Pick the list.")
			} else if !listExists(a.ListID) {
				bad("missing_ref", "This step's list was deleted. Pick another list.")
			}
		case "add_tag", "remove_tag":
			if !reTag.MatchString(a.Tag) {
				bad("invalid_tag", "Name the tag.")
			}
		}
	}
	return out
}

// checkWebhookURL rejects URLs a webhook may never call (the network check
// at call time also blocks names that resolve to private addresses).
func checkWebhookURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return fmt.Errorf("enter a full URL starting with https://")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("webhooks call http:// or https:// URLs")
	}
	if u.User != nil {
		return fmt.Errorf("put credentials in the signing secret, not in the URL")
	}
	if blockedHost(u.Hostname()) {
		return fmt.Errorf("webhooks can't call private, loopback or internal addresses")
	}
	return nil
}

func isBuiltinField(f string) bool {
	switch f {
	case "email", "first_name", "last_name", "country", "timezone", "source", "tier", "created_at":
		return true
	}
	return false
}

func pathLabel(n AutomationNode, p string) string {
	switch p {
	case PathYes:
		return "Yes"
	case PathNo:
		return "No"
	}
	_ = n
	return p
}

func capFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 'a' - 'A'
	}
	return string(r)
}

// Errors returns only blocking issues.
func Errors(issues []AutomationGraphIssue) []AutomationGraphIssue {
	out := []AutomationGraphIssue{}
	for _, i := range issues {
		if i.Severity == "error" {
			out = append(out, i)
		}
	}
	return out
}
