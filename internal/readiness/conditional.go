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

func init() {
	groupTitles[GroupConditional] = "Conditional content"
	groupPrefixes[GroupConditional] = "Conditional content"
}

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

const conditionalRunning = "every version: content, size and the conditions' segments, lists and fields"

func kbOf(n int) int { return int(math.Round(float64(n) / 1024)) }

func (b *builder) day(t time.Time) string { return t.In(b.loc).Format("Mon 2 Jan") }

// conditionalVersions reports whether the email has versions to check.
func (b *builder) conditionalVersions() bool {
	c := b.f.Conditional
	return c.Ran && c.Checked && c.Report.Counted && len(c.Report.Versions) > 1
}

// conditionalSize is the Content group's size check of an email with
// versions: each version is checked in Conditional content.
func (b *builder) conditionalSize(size ReadinessCheck) ReadinessCheck {
	r := b.f.Conditional.Report
	lo, hi := kbOf(r.MinSize), kbOf(r.MaxSize)
	size.Status, size.Ack, size.Fixes = Pass, "", nil
	size.Title = fmt.Sprintf("HTML size %d to %d KB across %d versions", lo, hi, len(r.Versions))
	largest := 0
	for _, v := range r.Versions {
		if v.Largest {
			largest = v.Number
		}
	}
	size.Detail = fmt.Sprintf("Conditional content checks each version against Gmail's 102 KB clipping limit; version %d is the largest.", largest)
	size.Short = fmt.Sprintf("%d to %d KB of 102 KB", lo, hi)
	return size
}

// blockNames joins the names of a version's blocks: "Beta corner and
// Start a free trial", "none of the conditional blocks".
func (b *builder) blockNames(v visibility.VisibilityVersion) string {
	names := []string{}
	for _, id := range v.Shown {
		for _, blk := range b.f.Conditional.Report.Blocks {
			if blk.ID == id {
				names = append(names, blk.Name)
			}
		}
	}
	if len(names) == 0 {
		return "none of the conditional blocks"
	}
	return JoinAnd(names)
}

func (b *builder) conditional() {
	cf := b.f.Conditional
	if !cf.Ran {
		return
	}
	r := cf.Report
	if !cf.Checked {
		b.add(ReadinessCheck{Key: "conditional.unchecked", Group: GroupConditional, Status: Unknown, Title: "Couldn't count the versions",
			Detail: firstNonEmpty(cf.Problem, "The other checks finished; the verdict waits for this one."), Label: "Versions", Note: "not checked"})
		return
	}
	if len(r.Blocks) == 0 && len(r.Invalid) == 0 {
		return
	}
	// Conditions that refer to something deleted.
	for _, blk := range r.Blocks {
		for _, br := range blk.Broken {
			what := map[string]string{"segment": "segment", "list": "list", "field": "field"}[br.Kind]
			detail := fmt.Sprintf("“%s” was deleted", br.Name)
			if br.DeletedBy != "" {
				detail += " by " + br.DeletedBy
			}
			if br.DeletedAt != nil {
				detail += " on " + b.day(*br.DeletedAt)
			}
			detail += ". A condition that refers to a deleted segment, field, list or tag blocks the send."
			params := map[string]string{"block": blk.ID}
			remove := map[string]string{"block": blk.ID, "condition": fmt.Sprint(br.Condition)}
			b.add(ReadinessCheck{Key: "conditional.broken." + blk.ID + "." + fmt.Sprint(br.Condition), Group: GroupConditional, Status: Block,
				Title: fmt.Sprintf("“%s” uses a deleted %s", blk.Name, what), Detail: detail, Label: "Broken condition",
				Fixes: []ReadinessFix{link("Pick another "+what, b.editor(blk.Path, "settings")), apply(FixVisibilityHideBlock, "Hide this block for everyone", "", params),
					apply(FixVisibilityRemoveCondition, "Remove the condition", "", remove)}})
		}
	}
	if len(r.Invalid) > 0 {
		b.add(ReadinessCheck{Key: "conditional.invalid", Group: GroupConditional, Status: Warn, Label: "Visibility rules",
			Title:  fmt.Sprintf("%d Visibility %s couldn't be read", len(r.Invalid), Plural(int64(len(r.Invalid)), "rule", "rules")),
			Detail: "Those blocks are shown to everyone. Open the email in the composer and set their visibility again.",
			Ack:    "I understand some blocks are shown to everyone.", Fixes: []ReadinessFix{link("Open in editor", b.editor(nil, ""))}})
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
		c := ReadinessCheck{Key: fmt.Sprintf("conditional.empty.%d", v.Number), Group: GroupConditional, Status: Block, Label: "Empty version",
			Title:  fmt.Sprintf("Version %d has nothing between the header and the footer", v.Number),
			Detail: fmt.Sprintf("%s %s would get only the header and the footer. Send Readiness blocks the send until every version has content.", Count(v.Recipients), Plural(v.Recipients, "recipient", "recipients")),
			Fixes:  []ReadinessFix{}}
		for _, id := range v.Hidden {
			for _, blk := range r.Blocks {
				if blk.ID == id && blk.Section && len(c.Fixes) < 2 {
					c.Fixes = append(c.Fixes, apply(FixVisibilityShowEveryone, fmt.Sprintf("Show “%s” to everyone", blk.Name), "", map[string]string{"block": blk.ID}))
				}
			}
		}
		c.Fixes = append(c.Fixes, link(fmt.Sprintf("Preview version %d", v.Number), b.versionHref(v.Number)))
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
		detail := fmt.Sprintf("%s · %s %s", capitalize(b.blockNames(v)), Count(v.Recipients), Plural(v.Recipients, "recipient", "recipients"))
		if len(others) > 0 {
			lo, hi := others[0], others[0]
			for _, k := range others {
				lo, hi = min(lo, k), max(hi, k)
			}
			detail += fmt.Sprintf(" · the other versions are %d to %d KB", lo, hi)
		}
		c := ReadinessCheck{Key: fmt.Sprintf("conditional.size.%d", v.Number), Group: GroupConditional, Status: Warn, Label: "Version size",
			Title:  fmt.Sprintf("Version %d is %d KB, over Gmail's 102 KB clipping size", v.Number, kbOf(v.SizeBytes)),
			Detail: detail + ". Gmail clips it and hides the rest, the footer's unsubscribe link included.",
			Ack:    fmt.Sprintf("I understand version %d is %d KB, over Gmail's clipping size.", v.Number, kbOf(v.SizeBytes))}
		for _, is := range issuesOf(b.f.Content.Report, "content.size") {
			for _, fx := range is.Fixes {
				if fx.Kind == "minify" {
					c.Fixes = append(c.Fixes, apply(FixMinify, fx.Label, "", nil))
				}
			}
		}
		if len(c.Fixes) == 0 && b.f.Content.InlineCSSBytes > 0 {
			c.Fixes = append(c.Fixes, apply(FixMinify, "Minify inline CSS", "", nil))
		}
		c.Fixes = append(c.Fixes, link(fmt.Sprintf("Preview version %d", v.Number), b.versionHref(v.Number)))
		b.add(c)
	}
	if !empty {
		smallest := r.Versions[0]
		for _, v := range r.Versions {
			if v.Smallest {
				smallest = v
			}
		}
		b.add(ReadinessCheck{Key: "conditional.content", Group: GroupConditional, Status: Pass, Label: "Every version has content",
			Title:  "Every version has content between the header and the footer",
			Detail: fmt.Sprintf("%d %s · the smallest is version %d, %d KB", len(r.Versions), Plural(int64(len(r.Versions)), "version", "versions"), smallest.Number, kbOf(smallest.SizeBytes)),
			Short:  "every version has content"})
	}
	if len(over) == 0 {
		b.add(ReadinessCheck{Key: "conditional.size", Group: GroupConditional, Status: Pass, Label: "Version size",
			Title:  "Every version is under Gmail's 102 KB clipping size",
			Detail: fmt.Sprintf("%d to %d KB", kbOf(r.MinSize), kbOf(r.MaxSize)), Short: fmt.Sprintf("%d to %d KB", kbOf(r.MinSize), kbOf(r.MaxSize))})
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
		b.add(ReadinessCheck{Key: "conditional.links", Group: GroupConditional, Status: Pass, Label: "Links",
			Title:  fmt.Sprintf("%d %s across all the versions", n, Plural(int64(n), "link responds", "links respond")),
			Detail: "Checked once each over the union of every version's links", Short: "links checked across the versions"})
	}
	if len(r.Versions) > visibility.MaxChecked {
		var covered int64
		for _, v := range r.Versions {
			if v.Checked {
				covered += v.Recipients
			}
		}
		b.add(ReadinessCheck{Key: "conditional.sampled", Group: GroupConditional, Status: Pass, Label: "Versions checked",
			Title:  fmt.Sprintf("%d of %d versions checked: the %d with the most recipients, plus the largest and the smallest email", r.Checked, len(r.Versions), visibility.MaxChecked),
			Detail: fmt.Sprintf("Together they cover %s of %s recipients (%s).", Count(covered), Count(r.Total), Pct1(float64(covered)/float64(max(r.Total, 1))*100)),
			Short:  fmt.Sprintf("%d of %d versions checked", r.Checked, len(r.Versions))})
	}
}

// versionHref opens a version's preview in the composer.
func (b *builder) versionHref(n int) string {
	return b.href(fmt.Sprintf("/campaigns/%s/edit?view=preview&version=%d", b.f.Campaign.ID, n))
}

func (b *builder) conditionalCaption() string {
	r := b.f.Conditional.Report
	s := fmt.Sprintf("%d conditional %s", len(r.Blocks), Plural(int64(len(r.Blocks)), "block", "blocks"))
	if r.Counted {
		s += fmt.Sprintf(" · %d %s", len(r.Versions), Plural(int64(len(r.Versions)), "version", "versions"))
		if r.CountedAt != nil {
			s += " · rendered " + b.when(*r.CountedAt)
		}
	}
	return s
}

// conditionalNote is what the verdict says about the versions: all were
// checked, or which ones beyond 50.
func (b *builder) conditionalNote() string {
	r := b.f.Conditional.Report
	if !r.Counted || len(r.Versions) == 0 {
		return ""
	}
	if len(r.Versions) > visibility.MaxChecked {
		return fmt.Sprintf("%d of the %d versions were rendered and checked: the %d with the most recipients, plus the largest and the smallest email.", r.Checked, len(r.Versions), visibility.MaxChecked)
	}
	if len(r.Versions) == 1 {
		return "The one version was rendered and checked."
	}
	return fmt.Sprintf("Every one of the %d versions was rendered and checked.", len(r.Versions))
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
			title := "A poll isn't complete"
			if q != "" {
				title = fmt.Sprintf("The poll “%s” isn't complete", q)
			}
			b.add(ReadinessCheck{Key: "content.poll." + p.ID + ".incomplete", Group: GroupContent, Status: Block, Label: "Poll",
				Title: title, Detail: strings.Join(p.Problems, " "), Fixes: []ReadinessFix{link("Show the poll", b.editor(p.Path, "settings"))}})
		}
		if p.Close.Mode == polls.CloseDate && p.ClosesAt != nil && !p.ClosesAt.After(start) {
			b.add(ReadinessCheck{Key: "content.poll." + p.ID + ".closes", Group: GroupContent, Status: Warn, Label: "Poll closing",
				Title:  fmt.Sprintf("The poll closes %s, before this campaign sends on %s", b.day(*p.ClosesAt), b.day(start)),
				Detail: fmt.Sprintf("Every answer would arrive after it closed, so the page would record none of them. Closing 7 days after sending keeps it open until %s.", b.day(start.Add(7*24*time.Hour))),
				Ack:    "I understand the poll closes before the campaign sends.",
				Fixes: []ReadinessFix{apply(FixPollCloseDays, "Close 7 days after sending", "", map[string]string{"poll": p.ID, "days": "7"}),
					link("Show the poll", b.editor(p.Path, "settings"))}})
		}
	}
	for _, sv := range pf.Surveys {
		switch {
		case sv.Status == polls.SurveyDraft:
			b.add(ReadinessCheck{Key: "content.survey." + sv.ID, Group: GroupContent, Status: Warn, Label: "Survey",
				Title: fmt.Sprintf("The survey “%s” isn't published", sv.Name), Detail: "Its link opens a page that doesn't take answers until the survey is published.",
				Ack: fmt.Sprintf("I understand the survey “%s” isn't published.", sv.Name), Fixes: []ReadinessFix{link("Open the survey", b.href("/forms/surveys/"+sv.ID))}})
		case sv.Status == polls.SurveyClosed || (sv.ClosesAt != nil && !sv.ClosesAt.After(start)):
			b.add(ReadinessCheck{Key: "content.survey." + sv.ID, Group: GroupContent, Status: Warn, Label: "Survey",
				Title: fmt.Sprintf("The survey “%s” closes before this campaign sends", sv.Name), Detail: "Its page would take no answers. Reopen it with a later closing date.",
				Ack: fmt.Sprintf("I understand the survey “%s” takes no answers.", sv.Name), Fixes: []ReadinessFix{link("Open the survey", b.href("/forms/surveys/"+sv.ID))}})
		}
	}
	if pf.Domain != "" && pf.DomainStatus != "verified" && len(pf.Polls)+len(pf.Surveys) > 0 {
		detail := pf.Domain + " isn't verified yet."
		if d := strings.TrimSpace(pf.DomainDetail); d != "" {
			detail = pf.Domain + ": " + strings.TrimSuffix(d, ".") + "."
		}
		instance := firstNonEmpty(pf.Instance, "the instance's address")
		b.add(ReadinessCheck{Key: "content.poll.domain", Group: GroupContent, Status: Warn, Label: "Answer page domain",
			Title:  "The publisher domain isn't verified, so answer links use " + instance,
			Detail: detail + " Answers still work, but the links point at the instance's own address, which readers may not recognise.",
			Ack:    "I understand answer links use " + instance + ".",
			Fixes:  []ReadinessFix{link("Verify "+pf.Domain, b.href("/forms/branding"))}})
	}
	if len(pf.Polls) > 0 {
		b.add(ReadinessCheck{Key: "content.poll.tracking", Group: GroupContent, Status: Pass, Label: "Poll links",
			Title:  "Poll links aren't tracked as clicks",
			Detail: "The report counts answers from the answer page, and shows the poll as one region of the click map.", Short: "poll links untracked"})
	}
}
