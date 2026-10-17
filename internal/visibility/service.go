package visibility

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service counts versions, previews them and answers what uses what.
type Service struct {
	Pool     *pgxpool.Pool
	Q        *store.Queries
	Segments SegmentResolver
	Log      *slog.Logger
	Now      func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// VisibilityBroken is a condition that refers to something deleted.
type VisibilityBroken struct {
	// Condition is the condition's position in the rule (0-based).
	Condition int    `json:"condition"`
	Kind      string `json:"kind" enum:"segment,list,field"`
	// Name is what it was called ("Customers with mrr over $50").
	Name      string     `json:"name"`
	DeletedBy string     `json:"deletedBy,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// VisibilityBlockInfo is one conditional block of an email.
type VisibilityBlockInfo struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Tag     string         `json:"tag"`
	Path    []int          `json:"path"`
	Section bool           `json:"section"`
	Rule    VisibilityRule `json:"rule"`
	// Words says the rule as the badge does ("Beta Program list").
	Words string `json:"words"`
	// Shown is how many recipients see it; Versions the versions that
	// show it.
	Shown    int64              `json:"shown"`
	Versions []int              `json:"versions"`
	Broken   []VisibilityBroken `json:"broken"`
}

// VisibilityReport is an email's conditional content: its blocks and the
// versions its audience gets ("3 conditional blocks · 6 versions").
type VisibilityReport struct {
	Blocks   []VisibilityBlockInfo `json:"blocks"`
	Versions []VisibilityVersion   `json:"versions"`
	// Total is the audience's final recipients, as Send Readiness counts them.
	Total int64 `json:"total"`
	// Possible is the number of combinations the blocks allow; Missing how
	// many of them no recipient gets.
	Possible   int64      `json:"possible"`
	Missing    int64      `json:"missing"`
	CountedAt  *time.Time `json:"countedAt,omitempty"`
	DurationMs int        `json:"durationMs"`
	// Counted is false when the audience is empty (nothing to count).
	Counted bool `json:"counted"`
	// MinSize and MaxSize are the smallest and largest version, in bytes.
	MinSize int `json:"minSize"`
	MaxSize int `json:"maxSize"`
	// Checked is how many versions Send Readiness renders one by one.
	Checked int `json:"checked"`
	// Invalid are rules that couldn't be read.
	Invalid []string `json:"invalid"`
}

// countResult is what a count keeps (visibility_counts).
type countResult struct {
	Tally      Tally
	At         time.Time
	DurationMs int
}

type storedVersion struct {
	Hidden     string `json:"hidden"`
	Recipients int64  `json:"recipients"`
}

// tally counts an email's versions for a campaign's audience, from the
// stored count while the rules and the audience are what it counted.
func (s *Service) tally(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, audience json.RawMessage, e Email, n *Names, recount bool) (countResult, error) {
	a := ParseAudience(audience)
	rulesHash, audHash := e.Hash(), a.hash()
	if !recount {
		if c, err := s.Q.VisibilityGetCount(ctx, campaignID); err == nil && c.RulesHash == rulesHash && c.AudienceHash == audHash {
			var vs []storedVersion
			_ = json.Unmarshal(c.Versions, &vs)
			t := Tally{Total: c.Total, ByHidden: map[string]int64{}, Shown: map[string]int64{}}
			for _, v := range vs {
				t.ByHidden[v.Hidden] = v.Recipients
			}
			_ = json.Unmarshal(c.Shown, &t.Shown)
			return countResult{Tally: t, At: c.CountedAt, DurationMs: int(c.DurationMs)}, nil
		} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return countResult{}, err
		}
	}
	start := time.Now()
	ruleSegs, err := LoadSegments(ctx, s.Segments, ws.ID, e.Segments(), n)
	if err != nil {
		return countResult{}, err
	}
	recipients, err := FinalRecipients(ctx, s.Q, s.Segments, ws.ID, a, ruleSegs)
	if err != nil {
		return countResult{}, err
	}
	byID := make(map[uuid.UUID]*Facts, len(recipients))
	for _, f := range recipients {
		byID[f.ID] = f
	}
	if err := loadExtra(ctx, s.Q, ws.ID, byID, e); err != nil {
		return countResult{}, err
	}
	t := Count(e, recipients, n)
	res := countResult{Tally: t, At: s.now(), DurationMs: int(time.Since(start).Milliseconds())}
	vs := make([]storedVersion, 0, len(t.ByHidden))
	for k, v := range t.ByHidden {
		vs = append(vs, storedVersion{Hidden: k, Recipients: v})
	}
	sort.Slice(vs, func(i, j int) bool { return vs[i].Recipients > vs[j].Recipients })
	vb, _ := json.Marshal(vs)
	sb, _ := json.Marshal(t.Shown)
	if err := s.Q.VisibilitySaveCount(ctx, store.VisibilitySaveCountParams{CampaignID: campaignID, WorkspaceID: ws.ID, RulesHash: rulesHash, AudienceHash: audHash,
		Total: t.Total, Versions: vb, Shown: sb, Factors: []byte(`{}`), CountedAt: res.At, DurationMs: int32(res.DurationMs)}); err != nil {
		s.log().WarnContext(ctx, "save the visibility count", "campaign", campaignID, "err", err)
	}
	return res, nil
}

// ReportInput is the content to report on: the composer's unsaved MJML
// and the HTML it compiled. Empty fields read the saved content.
type ReportInput struct {
	MJML string `json:"mjml,omitempty" maxLength:"2000000"`
	HTML string `json:"html,omitempty" maxLength:"2500000"`
	// Recount counts the audience again even when the rules didn't change.
	Recount bool `json:"recount,omitempty"`
}

// Report describes an email's conditional blocks and counts its versions.
func (s *Service) Report(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, in ReportInput) (VisibilityReport, error) {
	c, err := s.Q.VisibilityCampaign(ctx, store.VisibilityCampaignParams{WorkspaceID: ws.ID, ID: campaignID})
	if err != nil {
		return VisibilityReport{}, api.MapError(err, "Campaign")
	}
	src, html := c.Mjml, c.Html
	if in.MJML != "" {
		src, html = in.MJML, in.HTML
	}
	return s.report(ctx, ws, campaignID, c.Audience, src, html, in.Recount)
}

func (s *Service) report(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, audience json.RawMessage, src, html string, recount bool) (VisibilityReport, error) {
	out := VisibilityReport{Blocks: []VisibilityBlockInfo{}, Versions: []VisibilityVersion{}, Invalid: []string{}}
	e := Read(src)
	out.Invalid = e.Invalid
	if len(e.Blocks) == 0 {
		return out, nil
	}
	n, err := LoadNames(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	if err := n.LoadDeletions(ctx, s.Q, ws.ID, e.Refs()); err != nil {
		return out, err
	}
	for _, b := range e.Blocks {
		out.Blocks = append(out.Blocks, s.blockInfo(b, n))
	}
	out.Possible = Possible(e)
	if ParseAudience(audience).Empty() {
		return out, nil
	}
	res, err := s.tally(ctx, ws, campaignID, audience, e, n, recount)
	if err != nil {
		return out, err
	}
	at := res.At
	out.Counted, out.CountedAt, out.DurationMs, out.Total = true, &at, res.DurationMs, res.Tally.Total
	out.Versions = Versions(e, res.Tally, html)
	out.Missing = max(out.Possible-int64(len(out.Versions)), 0)
	for i := range out.Blocks {
		out.Blocks[i].Shown = res.Tally.Shown[out.Blocks[i].ID]
		for _, v := range out.Versions {
			for _, id := range v.Shown {
				if id == out.Blocks[i].ID {
					out.Blocks[i].Versions = append(out.Blocks[i].Versions, v.Number)
				}
			}
		}
	}
	for i, v := range out.Versions {
		if i == 0 || v.SizeBytes < out.MinSize {
			out.MinSize = v.SizeBytes
		}
		if v.SizeBytes > out.MaxSize {
			out.MaxSize = v.SizeBytes
		}
		if v.Checked {
			out.Checked++
		}
	}
	return out, nil
}

func (s *Service) blockInfo(b Block, n *Names) VisibilityBlockInfo {
	info := VisibilityBlockInfo{ID: b.ID, Name: b.Name, Tag: b.Tag, Path: b.Path, Section: b.Section, Rule: b.Rule, Words: Describe(b.Rule, n),
		Versions: []int{}, Broken: []VisibilityBroken{}}
	for i, c := range b.Rule.Conditions {
		k, ok := Lookup(c.Kind)
		if !ok || k.Ref == nil {
			continue
		}
		r := k.Ref(c)
		if r.Kind == "" || r.Kind == "tag" || n.Exists(r) {
			continue
		}
		br := VisibilityBroken{Condition: i, Kind: r.Kind, Name: c.Label}
		if br.Name == "" {
			br.Name = r.ID
		}
		if d, ok := n.Deleted[r]; ok {
			if d.Name != "" {
				br.Name = d.Name
			}
			br.DeletedBy = d.By
			at := d.At
			br.DeletedAt = &at
		}
		info.Broken = append(info.Broken, br)
	}
	return info
}

// lang is the request's language.
func lang(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }
