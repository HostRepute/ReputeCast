package approvals

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// Notifications keep their title, summary and body as catalog messages
// (notify.Notification's TitleText, SummaryText and BodyText), so each
// member reads them in their own language; the English stored beside them
// is what it always was. A message's arguments are plain values, so a list
// of rules inside one is written in the language of the member it goes to.

var (
	mNotifyRequestedTitle      = msgs.M("notify.requestedTitle")
	mNotifyRequestedSummary    = msgs.M("notify.requestedSummary")
	mNotifyRequestedBody       = msgs.M("notify.requestedBody")
	mNotifyAutomationSummary   = msgs.M("notify.automationSummary")
	mNotifyAutomationBody      = msgs.M("notify.automationBody")
	mNotifyPublishSummary      = msgs.M("notify.publishSummary")
	mNotifyPageBody            = msgs.M("notify.pageBody")
	mNotifyPostBody            = msgs.M("notify.postBody")
	mNotifyTemplateSummary     = msgs.M("notify.templateSummary")
	mNotifyTemplateBody        = msgs.M("notify.templateBody")
	mNotifyDigestSummary       = msgs.M("notify.digestSummary")
	mNotifyDigestBody          = msgs.M("notify.digestBody")
	mNotifyReminderTitle       = msgs.M("notify.reminderTitle")
	mNotifyReminderSummary     = msgs.M("notify.reminderSummary")
	mNotifyReminderBody        = msgs.M("notify.reminderBody")
	mNotifyDecidedTitle        = msgs.M("notify.decidedTitle")
	mNotifyDecidedSummary      = msgs.M("notify.decidedSummary")
	mNotifyDecidedBody         = msgs.M("notify.decidedBody")
	mNotifyInvalidatedTitle    = msgs.M("notify.invalidatedTitle")
	mNotifyInvalidatedSummary  = msgs.M("notify.invalidatedSummary")
	mNotifyInvalidatedBody     = msgs.M("notify.invalidatedBody")
	mNotifyChangedSummary      = msgs.M("notify.changedSummary")
	mNotifyPageChangedBody     = msgs.M("notify.pageChangedBody")
	mNotifyPostChangedBody     = msgs.M("notify.postChangedBody")
	mNotifyBypassTitle         = msgs.M("notify.bypassTitle")
	mNotifyBypassSummary       = msgs.M("notify.bypassSummary")
	mNotifyBypassBody          = msgs.M("notify.bypassBody")
	mNotifyHeldTitle           = msgs.M("notify.heldTitle")
	mNotifyHeldSummary         = msgs.M("notify.heldSummary")
	mNotifyHeldBody            = msgs.M("notify.heldBody")
	mNotifyPublishFailedTitle  = msgs.M("notify.publishFailedTitle")
	mNotifyPublishFailedSumary = msgs.M("notify.publishFailedSummary")
	mNotifyPublishFailedBody   = msgs.M("notify.publishFailedBody")
)

// readerLang is the language a member reads in when no request says: their
// own choice, else English.
func readerLang(ctx context.Context, q *store.Queries, userID uuid.UUID) i18n.Lang {
	u, err := q.GetUser(ctx, userID)
	if err != nil {
		return i18n.Default
	}
	return api.RequestLang("", &u)
}

// decisionNoteArgs are the arguments of the decision notification's kept
// summary and body: who decided what about which subject, and what comes
// next (the send's time, a post or page going live, a template to publish),
// the time written in the requester's language.
func decisionNoteArgs(ctx context.Context, q *store.Queries, ws *store.Workspace, r store.ApprovalRequest, status, approver, name, comment string) i18n.Args {
	args := i18n.Args{"approver": approver, "status": status, "name": name, "extra": "none", "hasComment": "no", "comment": comment, "next": "none"}
	if comment != "" {
		args["hasComment"] = "yes"
	}
	if status != "approved" {
		return args
	}
	snap := ParseSnapshot(r.Snapshot)
	switch {
	case r.TransactionalTemplateID != nil:
		args["extra"], args["next"] = "publish", "template"
		if snap.Transactional != nil {
			args["extra"], args["version"] = "publishVersion", snap.Transactional.Version
		}
	case r.WebsitePostID != nil && snap.Website != nil:
		args["extra"], args["next"] = "live", "post"
		if snap.Website.When == website.WhenAt {
			args["extra"] = "at"
		}
	case r.WebsitePageID != nil && snap.Page != nil:
		args["extra"], args["next"] = "live", "page"
		if snap.Page.When == "at" {
			args["extra"] = "at"
		}
	case r.SubjectType == "campaign":
		loc := state.Location(ws.Timezone)
		if ScheduleText(snap.Schedule, loc) != "Send now" && r.RequestedBy != nil {
			args["extra"], args["when"] = "scheduled", scheduleTextIn(readerLang(ctx, q, *r.RequestedBy), snap.Schedule, loc)
		}
	}
	return args
}

// yesNo is a select argument: "yes" or "no".
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// ruleList names matched rules inside a notification, in a language:
// "audience ≥ 10,000, warming-up provider".
func ruleList(l i18n.Lang, rules []ApprovalRuleMatch) string {
	local := localRules(l, rules)
	parts := make([]string, len(local))
	for i, r := range local {
		parts[i] = r.Short
	}
	return strings.Join(parts, ", ")
}
