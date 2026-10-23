package ai

import (
	"context"
	"encoding/json"

	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

// schemaField is a builder field as the model sees it: its key, label,
// kind, operators and options. No contact's value is in it.
type schemaField struct {
	Key     string         `json:"key"`
	Label   string         `json:"label"`
	Kind    string         `json:"kind"`
	Ops     []string       `json:"ops"`
	Options []schemaOption `json:"options,omitempty"`
}

type schemaOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// segmentSchema is the condition schema a draft is written with.
type segmentSchema struct {
	Fields []schemaField `json:"fields"`
	Tags   []string      `json:"tags"`
	// Segments are the saved segments' names (for "like segment X").
	Segments []string `json:"segments"`
}

// segmentTask is a segment draft's request, as data.
type segmentTask struct {
	Description string
	Current     segments.SegmentDefinition
	Schema      segmentSchema
}

// AISegmentDraftInput describes a segment.
type AISegmentDraftInput struct {
	Description string `json:"description" minLength:"1" maxLength:"500"`
	// Current is the builder's conditions as they are.
	Current segments.SegmentDefinition `json:"current"`
	// SegmentID names the segment being edited (empty for a new one).
	SegmentID string `json:"segmentId,omitempty"`
}

// AISegmentDraftCondition is a condition of the draft, marked.
type AISegmentDraftCondition struct {
	segments.SegmentCondition
	// Mark: added by the draft, removed by it (struck through), or kept.
	Mark string `json:"mark" enum:"added,removed,kept"`
}

// AISegmentDraftGroup is a group of the draft.
type AISegmentDraftGroup struct {
	Match      string                    `json:"match" enum:"all,any"`
	Conditions []AISegmentDraftCondition `json:"conditions"`
}

// AISegmentDraft is a draft as a diff on the builder, with its live count.
type AISegmentDraft struct {
	RequestID string `json:"requestId,omitempty"`
	// Groups show the draft over the current conditions; Definition is the
	// draft alone, what Accept puts in the builder.
	Groups     []AISegmentDraftGroup      `json:"groups"`
	Definition segments.SegmentDefinition `json:"definition"`
	Added      int                        `json:"added"`
	Removed    int                        `json:"removed"`
	// Note is the model's note: what it couldn't express.
	Note string `json:"note,omitempty"`
	// Nothing is set when nothing could be drafted, with why.
	Nothing string                       `json:"nothing,omitempty"`
	Count   *segments.SegmentCountResult `json:"count,omitempty"`
	Sent    AISent                       `json:"sent"`
	Refusal string                       `json:"refusal,omitempty"`
}

// schemaOf builds the condition schema of a workspace.
func (s *Service) schemaOf(ctx context.Context, ws *store.Workspace) (segmentSchema, segments.SegmentCatalog, error) {
	cat, err := s.Segments.Catalog(ctx, ws)
	if err != nil {
		return segmentSchema{}, cat, err
	}
	out := segmentSchema{Fields: []schemaField{}, Tags: []string{}, Segments: []string{}}
	for _, f := range cat.Fields {
		sf := schemaField{Key: f.Key, Label: f.Label, Kind: f.Kind, Ops: []string{}}
		for _, o := range f.Ops {
			sf.Ops = append(sf.Ops, o.Key)
		}
		for _, o := range f.Options {
			sf.Options = append(sf.Options, schemaOption{Value: o.Value, Label: o.Label})
		}
		out.Fields = append(out.Fields, sf)
	}
	if tags, err := s.Q.AITagNames(ctx, ws.ID); err == nil {
		out.Tags = tags
	}
	if list, err := s.Segments.List(ctx, ws); err == nil {
		for _, it := range list.Items {
			out.Segments = append(out.Segments, it.Name)
		}
	}
	return out, cat, nil
}

// fieldOf finds a catalog field.
func fieldOf(cat segments.SegmentCatalog, key string) (segments.SegmentCatalogField, bool) {
	for _, f := range cat.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return segments.SegmentCatalogField{}, false
}

// conform keeps the conditions the builder can show: known fields,
// operators and options; it reports how many it dropped.
func conform(cat segments.SegmentCatalog, d segments.SegmentDefinition) (segments.SegmentDefinition, int) {
	out := segments.SegmentDefinition{Groups: []segments.SegmentGroup{}}
	dropped := 0
	for _, g := range d.Groups {
		ng := segments.SegmentGroup{Match: g.Match, Conditions: []segments.SegmentCondition{}}
		if ng.Match != "any" {
			ng.Match = "all"
		}
		for _, c := range g.Conditions {
			f, ok := fieldOf(cat, c.Field)
			if !ok || !slices.ContainsFunc(f.Ops, func(o segments.SegmentOperator) bool { return o.Key == c.Op }) {
				dropped++
				continue
			}
			switch f.Kind {
			case "period":
				if c.Op != "never" {
					if c.Amount <= 0 {
						c.Amount = 30
					}
					if c.Unit != "weeks" {
						c.Unit = "days"
					}
				}
				c.Value = ""
			case "enum", "list", "campaign":
				// Values are the options' values; a label stands in for one.
				found := false
				for _, o := range f.Options {
					if o.Value == c.Value || strings.EqualFold(o.Label, c.Value) {
						c.Value, found = o.Value, true
						break
					}
				}
				if !found {
					dropped++
					continue
				}
				c.Amount, c.Unit = 0, ""
			default:
				c.Amount, c.Unit = 0, ""
			}
			ng.Conditions = append(ng.Conditions, c)
		}
		if len(ng.Conditions) > 0 {
			out.Groups = append(out.Groups, ng)
		}
	}
	return out, dropped
}

func sameCondition(a, b segments.SegmentCondition) bool {
	return a.Field == b.Field && a.Op == b.Op && a.Value == b.Value && a.Amount == b.Amount && a.Unit == b.Unit
}

// diff marks the draft's conditions against the current ones, group by
// group: the draft's rows in its order, each removed row just before the
// first added row on its field (what replaces it), the others first.
func diff(current, draft segments.SegmentDefinition) ([]AISegmentDraftGroup, int, int) {
	out := []AISegmentDraftGroup{}
	added, removed := 0, 0
	for i := 0; i < max(len(current.Groups), len(draft.Groups)); i++ {
		var cur, nxt []segments.SegmentCondition
		g := AISegmentDraftGroup{Match: "all", Conditions: []AISegmentDraftCondition{}}
		if i < len(current.Groups) {
			cur, g.Match = current.Groups[i].Conditions, current.Groups[i].Match
		}
		if i < len(draft.Groups) {
			nxt, g.Match = draft.Groups[i].Conditions, draft.Groups[i].Match
		}
		kept := map[int]bool{}  // draft rows that were already there
		stays := map[int]bool{} // current rows the draft keeps
		for ci, c := range cur {
			for di, d := range nxt {
				if !kept[di] && sameCondition(c, d) {
					kept[di], stays[ci] = true, true
					break
				}
			}
		}
		rows := []AISegmentDraftCondition{}
		for di, d := range nxt {
			mark := "kept"
			if !kept[di] {
				mark = "added"
				added++
			}
			rows = append(rows, AISegmentDraftCondition{SegmentCondition: d, Mark: mark})
		}
		claimed := map[int]bool{}
		lead := 0
		for ci, c := range cur {
			if stays[ci] {
				continue
			}
			removed++
			row := AISegmentDraftCondition{SegmentCondition: c, Mark: "removed"}
			at, replaces := -1, false
			for ri, r := range rows {
				if r.Mark == "added" && r.Field == c.Field && !claimed[ri] {
					at, replaces = ri, true
					break
				}
			}
			if at < 0 {
				at = lead
				lead++
			}
			rows = slices.Insert(rows, at, row)
			// Indexes after the insert moved by one.
			next := map[int]bool{}
			for k := range claimed {
				if k >= at {
					next[k+1] = true
				} else {
					next[k] = true
				}
			}
			if replaces {
				next[at+1] = true
			}
			claimed = next
		}
		g.Conditions = rows
		if len(rows) > 0 {
			out = append(out, g)
		}
	}
	return out, added, removed
}

// DraftSegment asks for the conditions a description means. Only the
// builder's own conditions are drafted (never the SQL editor), and the
// draft is counted like the builder counts.
func (s *Service) DraftSegment(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AISegmentDraftInput) (AISegmentDraft, error) {
	if !p.Can(authz.SegmentsEdit) {
		return AISegmentDraft{}, api.Forbidden(authz.SegmentsEdit, nil)
	}
	desc := strings.TrimSpace(in.Description)
	if desc == "" {
		return AISegmentDraft{}, invalid(ctx, "body.description", mDescriptionNeeded, nil)
	}
	label := newSegmentLabel
	var target *uuid.UUID
	if id, err := uuid.Parse(in.SegmentID); err == nil {
		if name, err := s.Q.AISegmentName(ctx, store.AISegmentNameParams{ID: id, WorkspaceID: ws.ID}); err == nil {
			label, target = name, &id
		}
	}
	schema, cat, err := s.schemaOf(ctx, ws)
	if err != nil {
		return AISegmentDraft{}, err
	}
	current, _ := conform(cat, in.Current)
	t := segmentTask{Description: desc, Current: current, Schema: schema}
	res, err := s.run(ctx, ws, p, task{feature: FeatureSegment, targetKind: "segment", targetID: target, targetLabel: label, req: segmentPrompt(t)})
	if err != nil {
		return AISegmentDraft{}, err
	}
	// "Your description and the condition schema, N characters": both count,
	// and the conditions the draft would replace.
	sent := chars(desc)
	if b, err := json.Marshal(t.Schema); err == nil {
		sent += chars(string(b))
	}
	if len(current.Groups) > 0 {
		if b, err := json.Marshal(current); err == nil {
			sent += chars(string(b))
		}
	}
	out := AISegmentDraft{RequestID: res.RequestID.String(), Groups: []AISegmentDraftGroup{}, Definition: segments.SegmentDefinition{Groups: []segments.SegmentGroup{}},
		Sent: res.sent(sent), Refusal: res.Refusal}
	if res.Refusal != "" {
		return out, nil
	}
	var a struct {
		Groups  []segments.SegmentGroup `json:"groups"`
		Note    string                  `json:"note"`
		Nothing string                  `json:"nothing"`
	}
	_ = json.Unmarshal(res.JSON, &a)
	draft, dropped := conform(cat, segments.SegmentDefinition{Groups: a.Groups})
	out.Note = strings.TrimSpace(a.Note)
	if strings.TrimSpace(a.Nothing) != "" || len(draft.Groups) == 0 {
		out.Nothing = strings.TrimSpace(a.Nothing)
		if out.Nothing == "" {
			out.Nothing = out.Note
		}
		out.Note = ""
		s.finish(ctx, ws, p, res.RequestID, OutcomeNothingDrafted, "")
		return out, nil
	}
	if dropped > 0 && out.Note == "" {
		out.Note = text(ctx, mDroppedConditions, map[string]any{"count": dropped})
	}
	out.Definition = draft
	out.Groups, out.Added, out.Removed = diff(current, draft)
	if c, err := s.Segments.Preview(ctx, ws, segments.SegmentInput{Mode: "builder", Definition: draft}); err == nil {
		out.Count = &c
	}
	return out, nil
}
