package automations

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/store"
)

// EmailLanguages is what flows ask of the content languages domain
// (internal/languages): the language versions of an email version, and
// carrying them to the email's next version when it's edited.
type EmailLanguages interface {
	AutomationEmailStates(ctx context.Context, q *store.Queries, ws *store.Workspace, emailID uuid.UUID, version int) ([]languages.AutomationEmailState, error)
	CopyForward(ctx context.Context, q *store.Queries, wsID, emailID uuid.UUID, from, to int) error
}

// copyLanguages carries an email's language versions to its next version.
func (s *Service) copyLanguages(ctx context.Context, q *store.Queries, wsID, id uuid.UUID, from, to int32) error {
	if s.Languages == nil || from <= 0 {
		return nil
	}
	return s.Languages.CopyForward(ctx, q, wsID, id, int(from), int(to))
}

// langStep is an email step of a flow version with its language versions.
type langStep struct {
	node   AutomationNode
	number int
	langs  map[string]languages.AutomationEmailState
}

func (s *Service) emailSteps(ctx context.Context, q *store.Queries, ws *store.Workspace, g AutomationGraph) []langStep {
	var out []langStep
	n := 0
	for _, id := range g.Order() {
		node := g.Node(id)
		if node == nil || node.Type != NodeEmail {
			continue
		}
		n++
		st := langStep{node: *node, number: n, langs: map[string]languages.AutomationEmailState{}}
		if e := node.Data.Email; e != nil && s.Languages != nil {
			if eid, err := uuid.Parse(e.EmailID); err == nil {
				states, _ := s.Languages.AutomationEmailStates(ctx, q, ws, eid, e.EmailVersion)
				for _, l := range states {
					st.langs[l.Language] = l
				}
			}
		}
		out = append(out, st)
	}
	return out
}

// steps names step numbers: "step 2", "steps 1 and 3", "steps 1 to 3".
func stepRange(nums []int) string {
	sort.Ints(nums)
	if len(nums) == 1 {
		return fmt.Sprintf("step %d", nums[0])
	}
	run := true
	for i := 1; i < len(nums); i++ {
		run = run && nums[i] == nums[i-1]+1
	}
	if run && len(nums) > 2 {
		return fmt.Sprintf("steps %d to %d", nums[0], nums[len(nums)-1])
	}
	words := make([]string, len(nums))
	for i, n := range nums {
		words[i] = fmt.Sprint(n)
	}
	return "steps " + strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

func langJoinAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// sameSource reports whether two versions of an email have the same
// content: a version made only to change its translations.
func sameSource(ctx context.Context, q *store.Queries, wsID uuid.UUID, emailID string, a, b int) bool {
	id, err := uuid.Parse(emailID)
	if err != nil {
		return false
	}
	va, err1 := q.GetAutomationEmailVersion(ctx, store.GetAutomationEmailVersionParams{WorkspaceID: wsID, ID: id, Version: int32(a)})
	vb, err2 := q.GetAutomationEmailVersion(ctx, store.GetAutomationEmailVersionParams{WorkspaceID: wsID, ID: id, Version: int32(b)})
	return err1 == nil && err2 == nil && va.Subject == vb.Subject && va.Preheader == vb.Preheader && va.Html == vb.Html
}

// languageDiff names the language versions a draft adds to its email steps
// ("Added · Spanish and German versions of steps 1 to 3"). A step whose
// email changed only in its translations isn't listed as changed.
func (s *Service) languageDiff(ctx context.Context, q *store.Queries, ws *store.Workspace, from, to AutomationGraph, items []AutomationDiffItem) []AutomationDiffItem {
	if s.Languages == nil {
		return items
	}
	before := map[string]langStep{}
	for _, st := range s.emailSteps(ctx, q, ws, from) {
		before[st.node.ID] = st
	}
	sent := func(l languages.AutomationEmailState) bool { return l.Status != languages.StatusDraft && !l.Held }
	added := map[string][]int{}
	translators := map[string]bool{}
	drop := map[string]bool{}
	var all []langStep
	for _, st := range s.emailSteps(ctx, q, ws, to) {
		all = append(all, st)
		was, ok := before[st.node.ID]
		for code, l := range st.langs {
			if !sent(l) {
				continue
			}
			if ok {
				if old, had := was.langs[code]; had && sent(old) {
					continue
				}
			}
			added[code] = append(added[code], st.number)
			if l.By != "" {
				translators[l.By] = true
			}
		}
		if ok && st.node.Data.Email != nil && was.node.Data.Email != nil && st.node.Data.Email.EmailVersion != was.node.Data.Email.EmailVersion {
			a, b := *was.node.Data.Email, *st.node.Data.Email
			a.EmailVersion = b.EmailVersion
			if reflect.DeepEqual(a, b) && sameSource(ctx, q, ws.ID, b.EmailID, was.node.Data.Email.EmailVersion, b.EmailVersion) {
				drop[st.node.ID] = true
			}
		}
	}
	if len(added) == 0 {
		return items
	}
	out := make([]AutomationDiffItem, 0, len(items)+1)
	for _, it := range items {
		if it.Kind == "changed" && len(it.NodeIDs) == 1 && drop[it.NodeIDs[0]] {
			continue
		}
		out = append(out, it)
	}
	// One line per set of steps: "Spanish and German versions of steps 1 to 3".
	bySteps := map[string][]string{}
	nodes := map[string][]string{}
	for code, nums := range added {
		k := stepRange(append([]int(nil), nums...))
		bySteps[k] = append(bySteps[k], languages.NameOf(code))
		for _, st := range all {
			for _, n := range nums {
				if st.number == n {
					nodes[k] = append(nodes[k], st.node.ID)
				}
			}
		}
	}
	keys := make([]string, 0, len(bySteps))
	for k := range bySteps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	names := make([]string, 0, len(translators))
	for n := range translators {
		names = append(names, n)
	}
	sort.Strings(names)
	for i, k := range keys {
		langs := bySteps[k]
		sort.Strings(langs)
		word := "versions"
		if len(langs) == 1 {
			word = "version"
		}
		item := AutomationDiffItem{Kind: "added", Title: fmt.Sprintf("%s %s of %s", langJoinAnd(langs), word, k), NodeIDs: langUniq(nodes[k])}
		if i == 0 && len(names) > 0 {
			item.Detail = fmt.Sprintf("Translated by %s.", langJoinAnd(names))
		}
		out = append([]AutomationDiffItem{item}, out...)
	}
	return out
}

func langUniq(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// languageChecks warn about outdated language versions ("Step 2's German
// version is outdated"); they never stop publishing.
func (s *Service) languageChecks(ctx context.Context, ws *store.Workspace, g AutomationGraph) []AutomationPublishCheck {
	if s.Languages == nil {
		return nil
	}
	var out []AutomationPublishCheck
	for _, st := range s.emailSteps(ctx, s.Q, ws, g) {
		codes := make([]string, 0, len(st.langs))
		for code := range st.langs {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		for _, code := range codes {
			if l := st.langs[code]; l.Status == languages.StatusOutdated && !l.Held {
				out = append(out, AutomationPublishCheck{Status: "warn", Label: fmt.Sprintf("Step %d’s %s version is outdated", st.number, l.Name)})
			}
		}
	}
	return out
}
