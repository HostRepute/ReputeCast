package readiness

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/polls"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/visibility"
)

// The Conditional content group and the poll checks of the Content group
// (design expansion, X11): every version of an email with Visibility rules
// rendered and checked (nothing between the header and the footer, over
// Gmail's clipping size, a condition that refers to something deleted),
// and a poll that closes before the send or whose links can't use the
// publisher's domain.

// GroupConditional is the Conditional content group's key.
const GroupConditional = "conditional"

// Fix actions of conditional content and polls.
const (
	FixVisibilityShowEveryone    = visibility.FixShowEveryone
	FixVisibilityHideBlock       = visibility.FixHideBlock
	FixVisibilityRemoveCondition = visibility.FixRemoveCondition
	FixPollCloseDays             = polls.FixCloseDays
)

// ConditionalFacts are an email's conditional blocks and the versions its
// audience gets.
type ConditionalFacts struct {
	Ran bool
	// Checked is false when the versions couldn't be counted; Problem says why.
	Checked bool
	Problem string
	Report  visibility.VisibilityReport
}

// PollFacts are an email's polls, the surveys it links to and where their
// links go.
type PollFacts = polls.ReadinessFacts

// ConditionalChecker gives the facts of the Conditional content group and
// of the poll checks; nil leaves them out.
type ConditionalChecker interface {
	ConditionalFacts(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) ConditionalFacts
	PollFacts(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, src string) PollFacts
}

// conditionalFix applies a fix of conditional content or polls to the MJML.
func conditionalFix(src, action string, params map[string]string) (next, label, applied string, err error) {
	if action == FixPollCloseDays {
		return polls.FixMJML(src, action, params)
	}
	return visibility.Fix(src, action, params)
}

func kbOf(n int) int { return int(math.Round(float64(n) / 1024)) }

// conditionalVersions reports whether the email has versions to check.
func (b *builder) conditionalVersions() bool {
	c := b.f.Conditional
	return c.Ran && c.Checked && c.Report.Counted && len(c.Report.Versions) > 1
}

// conditionalSize is the Content group's size check of an email with
// versions: each version is checked in Conditional content.
func (b *builder) conditionalSize(size check) check {
	r := b.f.Conditional.Report
	lo, hi := kbOf(r.MinSize), kbOf(r.MaxSize)
	size.Status, size.Ack, size.Fixes = Pass, nil, nil
	size.Title = tx("conditional.sizeAcross", A{"min": lo, "max": hi, "count": len(r.Versions)})
	largest := 0
	for _, v := range r.Versions {
		if v.Largest {
			largest = v.Number
		}
	}
	size.Detail = tx("conditional.sizeAcrossDetail", A{"clip": 102, "largest": largest})
	size.Short = tx("conditional.sizeAcrossShort", A{"min": lo, "max": hi, "clip": 102})
	return size
}

// blockNames names the blocks a version shows: "Beta corner and Start a
// free trial", "none of the conditional blocks".
func (b *builder) blockNames(v visibility.VisibilityVersion) Text {
	names := []string{}
	for _, id := range v.Shown {
		for _, blk := range b.f.Conditional.Report.Blocks {
			if blk.ID == id {
				names = append(names, blk.Name)
			}
		}
	}
	return tx("conditional.shows", A{"count": len(names), "blocks": argCapital(ArgAnd(names))})
}

func (b *builder) conditional() {
	cf := b.f.Conditional
	if !cf.Ran {
		return
	}
	r := cf.Report
	if !cf.Checked {
		detail := RawText(cf.Problem)
		if detail.Empty() {
			detail = tx("check.waits", nil)
		}
		b.add(check{Key: "conditional.unchecked", Group: GroupConditional, Status: Unknown, Title: tx("conditional.unchecked", nil),
			Detail: detail, Label: tx("label.versions", nil), Note: tx("note.notChecked", nil)})
		return
	}
	if len(r.Blocks) == 0 && len(r.Invalid) == 0 {
		return
	}
	// Conditions that refer to something deleted.
	for _, blk := range r.Blocks {
		for _, br := range blk.Broken {
			kind := br.Kind
			deleted := A{"name": br.Name, "by": br.DeletedBy, "named": yes(br.DeletedBy != ""), "dated": yes(br.DeletedAt != nil)}
			if br.DeletedAt != nil {
				deleted["day"] = b.day(*br.DeletedAt)
			}
			params := map[string]string{"block": blk.ID}
			remove := map[string]string{"block": blk.ID, "condition": fmt.Sprint(br.Condition)}
			b.add(check{Key: "conditional.broken." + blk.ID + "." + fmt.Sprint(br.Condition), Group: GroupConditional, Status: Block,
				Title:  tx("conditional.broken", A{"block": blk.Name, "kind": kind}),
				Detail: sentences(tx("conditional.deleted", deleted), tx("conditional.deletedBlocks", nil)), Label: tx("label.brokenCondition", nil),
				Fixes: []fix{link(tx("fix.pickAnother", A{"kind": kind}), b.editor(blk.Path, "settings")), apply(FixVisibilityHideBlock, tx("fix.hideBlock", nil), "", params),
					apply(FixVisibilityRemoveCondition, tx("fix.removeCondition", nil), "", remove)}})
		}
	}
	if len(r.Invalid) > 0 {
		b.add(check{Key: "conditional.invalid", Group: GroupConditional, Status: Warn, Label: tx("label.visibilityRules", nil),
			Title:  tx("conditional.invalid", A{"count": len(r.Invalid)}),
			Detail: tx("conditional.invalidDetail", nil),
			Ack:    tx("conditional.invalidAck", nil), Fixes: []fix{link(tx("fix.openInEditor", nil), b.editor(nil, ""))}})
	}
	if !r.Counted || len(r.Versions) == 0 {
		return
	}
	// Versions with nothing between the header and the footer.
	empty := false
	for _, v := range r.Versions {
		if !v.Empty {
			continue
		}
		empty = true
		c := check{Key: fmt.Sprintf("conditional.empty.%d", v.Number), Group: GroupConditional, Status: Block, Label: tx("label.emptyVersion", nil),
			Title:  tx("conditional.empty", A{"number": v.Number}),
			Detail: tx("conditional.emptyDetail", A{"count": v.Recipients}),
			Fixes:  []fix{}}
		for _, id := range v.Hidden {
			for _, blk := range r.Blocks {
				if blk.ID == id && blk.Section && len(c.Fixes) < 2 {
					c.Fixes = append(c.Fixes, apply(FixVisibilityShowEveryone, tx("fix.showEveryone", A{"block": blk.Name}), "", map[string]string{"block": blk.ID}))
				}
			}
		}
		c.Fixes = append(c.Fixes, link(tx("fix.previewVersion", A{"number": v.Number}), b.versionHref(v.Number)))
		b.add(c)
	}
	// Versions over Gmail's clipping size: as severe as the size check.
	var over []visibility.VisibilityVersion
	for _, v := range r.Versions {
		if v.OverClip && v.Checked {
			over = append(over, v)
		}
	}
	for _, v := range over {
		others := []int{}
		for _, o := range r.Versions {
			if o.Number != v.Number && !o.OverClip {
				others = append(others, kbOf(o.SizeBytes))
			}
		}
		args := A{"shows": b.blockNames(v), "count": v.Recipients, "others": yes(len(others) > 0)}
		if len(others) > 0 {
			lo, hi := others[0], others[0]
			for _, k := range others {
				lo, hi = min(lo, k), max(hi, k)
			}
			args["min"], args["max"] = lo, hi
		}
		c := check{Key: fmt.Sprintf("conditional.size.%d", v.Number), Group: GroupConditional, Status: Warn, Label: tx("label.versionSize", nil),
			Title:  tx("conditional.over", A{"number": v.Number, "kb": kbOf(v.SizeBytes), "clip": 102}),
			Detail: tx("conditional.overDetail", args),
			Ack:    tx("conditional.overAck", A{"number": v.Number, "kb": kbOf(v.SizeBytes)})}
		for _, is := range issuesOf(b.f.Content.Report, "content.size") {
			for _, fx := range is.Fixes {
				if fx.Kind == "minify" {
					c.Fixes = append(c.Fixes, apply(FixMinify, minifyLabel(fx.Label), "", nil))
				}
			}
		}
		if len(c.Fixes) == 0 && b.f.Content.InlineCSSBytes > 0 {
			c.Fixes = append(c.Fixes, apply(FixMinify, tx("fix.minify", nil), "", nil))
		}
		c.Fixes = append(c.Fixes, link(tx("fix.previewVersion", A{"number": v.Number}), b.versionHref(v.Number)))
		b.add(c)
	}
	if !empty {
		smallest := r.Versions[0]
		for _, v := range r.Versions {
			if v.Smallest {
				smallest = v
			}
		}
		b.add(check{Key: "conditional.content", Group: GroupConditional, Status: Pass, Label: tx("label.everyVersionContent", nil),
			Title:  tx("conditional.content", nil),
			Detail: tx("conditional.contentDetail", A{"count": len(r.Versions), "number": smallest.Number, "kb": kbOf(smallest.SizeBytes)}),
			Short:  tx("conditional.contentShort", nil)})
	}
	if len(over) == 0 {
		sizes := A{"min": kbOf(r.MinSize), "max": kbOf(r.MaxSize), "clip": 102}
		b.add(check{Key: "conditional.size", Group: GroupConditional, Status: Pass, Label: tx("label.versionSize", nil),
			Title:  tx("conditional.under", sizes),
			Detail: tx("conditional.sizes", sizes), Short: tx("conditional.sizes", sizes)})
	}
	// Links are checked once each over the union of every version's links,
	// in the Links group; this row says so when they all respond.
	linksOK := true
	for _, c := range b.checks {
		if c.Group == GroupLinks && c.Status != Pass {
			linksOK = false
		}
	}
	if n := len(b.f.Content.Report.LinkList); b.f.Content.Checked && linksOK && n > 0 {
		b.add(check{Key: "conditional.links", Group: GroupConditional, Status: Pass, Label: tx("label.links", nil),
			Title:  tx("conditional.links", A{"count": n}),
			Detail: tx("conditional.linksDetail", nil), Short: tx("conditional.linksShort", nil)})
	}
	if len(r.Versions) > visibility.MaxChecked {
		var covered int64
		for _, v := range r.Versions {
			if v.Checked {
				covered += v.Recipients
			}
		}
		b.add(check{Key: "conditional.sampled", Group: GroupConditional, Status: Pass, Label: tx("label.versionsChecked", nil),
			Title:  tx("conditional.sampled", A{"checked": r.Checked, "count": len(r.Versions), "top": visibility.MaxChecked}),
			Detail: tx("conditional.sampledDetail", A{"covered": covered, "total": r.Total, "share": ArgPercent1(float64(covered) / float64(max(r.Total, 1)) * 100)}),
			Short:  tx("conditional.sampledShort", A{"checked": r.Checked, "count": len(r.Versions)})})
	}
}

// versionHref opens a version's preview in the composer.
func (b *builder) versionHref(n int) string {
	return b.href(fmt.Sprintf("/campaigns/%s/edit?view=preview&version=%d", b.f.Campaign.ID, n))
}

func (b *builder) conditionalCaption() Text {
	r := b.f.Conditional.Report
	args := A{"blocks": len(r.Blocks), "counted": yes(r.Counted), "versions": len(r.Versions), "rendered": yes(r.Counted && r.CountedAt != nil)}
	if r.CountedAt != nil {
		args["when"] = b.when(*r.CountedAt)
	}
	return tx("conditional.caption", args)
}

// conditionalNote is what the verdict says about the versions: all were
// checked, or which ones beyond 50.
func (b *builder) conditionalNote() Text {
	r := b.f.Conditional.Report
	if !r.Counted || len(r.Versions) == 0 {
		return nil
	}
	if len(r.Versions) > visibility.MaxChecked {
		return tx("conditional.noteSampled", A{"checked": r.Checked, "count": len(r.Versions), "top": visibility.MaxChecked})
	}
	return tx("conditional.noteAll", A{"count": len(r.Versions)})
}

// polls adds the poll checks to the Content group.
func (b *builder) polls() {
	pf := b.f.Polls
	if !pf.Ran || (len(pf.Polls) == 0 && len(pf.Surveys) == 0) {
		return
	}
	start := b.f.Campaign.StartAt
	for _, p := range pf.Polls {
		q := strings.TrimSpace(p.Question)
		if len(p.Problems) > 0 {
			parts := make([]Text, 0, len(p.Problems))
			for _, pr := range p.Problems {
				// The poll's own words for what's missing.
				parts = append(parts, RawText(pr))
			}
			b.add(check{Key: "content.poll." + p.ID + ".incomplete", Group: GroupContent, Status: Block, Label: tx("label.poll", nil),
				Title: tx("polls.incomplete", A{"question": q, "asked": yes(q != "")}), Detail: joined(" ", parts),
				Fixes: []fix{link(tx("fix.showPoll", nil), b.editor(p.Path, "settings"))}})
		}
		if p.Close.Mode == polls.CloseDate && p.ClosesAt != nil && !p.ClosesAt.After(start) {
			b.add(check{Key: "content.poll." + p.ID + ".closes", Group: GroupContent, Status: Warn, Label: tx("label.pollClosing", nil),
				Title:  tx("polls.closes", A{"closes": b.day(*p.ClosesAt), "sends": b.day(start)}),
				Detail: tx("polls.closesDetail", A{"days": 7, "until": b.day(start.Add(7 * 24 * time.Hour))}),
				Ack:    tx("polls.closesAck", nil),
				Fixes: []fix{apply(FixPollCloseDays, tx("fix.closeAfter", A{"days": 7}), "", map[string]string{"poll": p.ID, "days": "7"}),
					link(tx("fix.showPoll", nil), b.editor(p.Path, "settings"))}})
		}
	}
	for _, sv := range pf.Surveys {
		open := link(tx("fix.openSurvey", nil), b.href("/forms/surveys/"+sv.ID))
		switch {
		case sv.Status == polls.SurveyDraft:
			b.add(check{Key: "content.survey." + sv.ID, Group: GroupContent, Status: Warn, Label: tx("label.survey", nil),
				Title: tx("polls.surveyDraft", A{"name": sv.Name}), Detail: tx("polls.surveyDraftDetail", nil),
				Ack: tx("polls.surveyDraftAck", A{"name": sv.Name}), Fixes: []fix{open}})
		case sv.Status == polls.SurveyClosed || (sv.ClosesAt != nil && !sv.ClosesAt.After(start)):
			b.add(check{Key: "content.survey." + sv.ID, Group: GroupContent, Status: Warn, Label: tx("label.survey", nil),
				Title: tx("polls.surveyCloses", A{"name": sv.Name}), Detail: tx("polls.surveyClosesDetail", nil),
				Ack: tx("polls.surveyClosesAck", A{"name": sv.Name}), Fixes: []fix{open}})
		}
	}
	if pf.Domain != "" && pf.DomainStatus != "verified" && len(pf.Polls)+len(pf.Surveys) > 0 {
		detail := tx("polls.domainUnverified", A{"domain": pf.Domain})
		if d := strings.TrimSpace(pf.DomainDetail); d != "" {
			// The domain check's own words.
			detail = tx("polls.domainDetail", A{"domain": pf.Domain, "detail": strings.TrimSuffix(d, ".")})
		}
		var instance any = pf.Instance
		if pf.Instance == "" {
			instance = tx("polls.instanceAddress", nil)
		}
		b.add(check{Key: "content.poll.domain", Group: GroupContent, Status: Warn, Label: tx("label.answerDomain", nil),
			Title:  tx("polls.domain", A{"instance": instance}),
			Detail: sentences(detail, tx("polls.domainStillWorks", nil)),
			Ack:    tx("polls.domainAck", A{"instance": instance}),
			Fixes:  []fix{link(tx("fix.verify", A{"domain": pf.Domain}), b.href("/forms/branding"))}})
	}
	if len(pf.Polls) > 0 {
		b.add(check{Key: "content.poll.tracking", Group: GroupContent, Status: Pass, Label: tx("label.pollLinks", nil),
			Title:  tx("polls.untracked", nil),
			Detail: tx("polls.untrackedDetail", nil), Short: tx("polls.untrackedShort", nil)})
	}
}
