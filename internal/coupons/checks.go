package coupons

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// IssuePool is the composer's check and Send Readiness's row about a
// Discount code block's pool.
const IssuePool = "content.coupon_pool"

// ComposerIssues implements campaigns.MoreChecks: a Warn for a pool the
// email names that doesn't exist or is archived, on the element naming it.
func (s *Service) ComposerIssues(ctx context.Context, ws *store.Workspace, _ uuid.UUID, src string) []contentcheck.ContentCheckIssue {
	slugs := usedSlugs(liquid.ExpandBlocks(src))
	if len(slugs) == 0 {
		return nil
	}
	rows, err := s.pools(ctx, ws.ID, nil, slugs)
	if err != nil {
		return nil
	}
	known := map[string]store.CouponPoolsRow{}
	for _, p := range rows {
		known[p.Slug] = p
	}
	paths := slugPaths(src)
	l := i18n.LangFrom(ctx)
	var out []contentcheck.ContentCheckIssue
	for _, slug := range slugs {
		p, ok := known[slug]
		var title, detail i18n.Text
		switch {
		case !ok:
			title, detail = mCheckUnknownTitle.Keep(i18n.Args{"slug": slug}), mCheckUnknownDetail.Keep(nil)
		case p.Status == StatusArchived:
			title, detail = mCheckArchivedTitle.Keep(i18n.Args{"name": p.Name}), mCheckArchivedText.Keep(nil)
		default:
			continue
		}
		is := contentcheck.ContentCheckIssue{ID: IssuePool, Severity: contentcheck.Warn, Group: "Content", Path: []int{}, Section: -1,
			Title: title.In(l), Detail: detail.In(l), Tag: mCheckTag.Text(l, nil),
			TitleText: title, DetailText: []i18n.Text{detail}, TagText: mCheckTag.Keep(nil),
			Fixes: []contentcheck.ContentCheckFix{{Kind: "open", Label: mCheckOpenPools.Text(l, nil), Value: "/w/" + ws.Slug + "/campaigns/codes"}}}
		if path, ok := paths[slug]; ok {
			is.Path, is.Section = path, path[0]
		}
		out = append(out, is)
	}
	return out
}

// slugPaths are where in the body each pool is first named (element
// indices from mj-body).
func slugPaths(src string) map[string][]int {
	out := map[string][]int{}
	root, err := mjml.Parse(src)
	if err != nil {
		return out
	}
	body := mjml.Body(root)
	if body == nil {
		return out
	}
	body.Walk(func(n *mjml.Node, path []int) {
		if len(path) == 0 || n.Inner == "" {
			return
		}
		for _, slug := range usedSlugs(liquid.ExpandBlocks(n.Inner)) {
			if _, seen := out[slug]; !seen {
				out[slug] = path
			}
		}
	})
	return out
}

// ReadinessRow is a pool's row in Send Readiness's Content group. Its
// texts are kept messages: Send Readiness writes them in each reader's
// language.
type ReadinessRow struct {
	Status                      string // pass or block
	Title, Detail, Label, Short i18n.Text
	Links                       []ReadinessLink
}

// ReadinessLink is a row's link: its label and the workspace path it opens.
type ReadinessLink struct {
	Label i18n.Text
	Href  string
}

// ReadinessRows are the rows of the pools a campaign's HTML shows codes
// from, for its recipients: Block when an upload pool has fewer free codes
// than recipients, when a store pool is short (with the batch creating
// them), when the pool is archived or its store isn't connected.
// at writes a moment as Send Readiness keeps it (in each reader's time zone).
func (s *Service) ReadinessRows(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, html string, recipients int64, at func(time.Time) any) []ReadinessRow {
	slugs := usedSlugs(liquid.ExpandBlocks(html))
	if len(slugs) == 0 {
		return nil
	}
	rows, err := s.pools(ctx, ws.ID, nil, slugs)
	if err != nil {
		return nil
	}
	label, shortText := mReadyLabel.Keep(nil), mReadyShort.Keep(nil)
	email := "/campaigns/" + campaignID.String() + "/edit"
	var out []ReadinessRow
	for _, p := range rows {
		page := "/campaigns/codes/" + p.ID.String()
		row := ReadinessRow{Status: "block", Label: label, Short: shortText}
		switch {
		case p.Status == StatusArchived:
			row.Title, row.Detail = mReadyArchived.Keep(i18n.Args{"name": p.Name}), mReadyArchivedFor.Keep(nil)
			row.Links = []ReadinessLink{{Label: mReadyOpenEmail.Keep(nil), Href: email}}
			out = append(out, row)
			continue
		case p.Source != SourceUpload && (p.StoreID == nil || p.StoreStatus != "connected"):
			row.Title, row.Detail = mReadyDisconnected.Keep(i18n.Args{"store": p.StoreName, "name": p.Name}), mReadyDisconnFor.Keep(nil)
			if p.StoreID != nil {
				row.Links = []ReadinessLink{{Label: mReadyOpenStore.Keep(nil), Href: "/settings/integrations/stores/" + p.StoreID.String()}}
			}
			out = append(out, row)
			continue
		}
		free, err := s.Q.CouponPoolFree(ctx, p.ID)
		if err != nil {
			continue
		}
		args := i18n.Args{"free": free, "recipients": recipients, "name": p.Name}
		if free >= recipients {
			row.Status, row.Title = "pass", mReadyEnough.Keep(args)
			out = append(out, row)
			continue
		}
		missing := recipients - free
		if p.Source == SourceUpload {
			row.Title, row.Detail = mReadyNotEnough.Keep(args), mReadyNotEnoughFor.Keep(i18n.Args{"name": p.Name, "missing": missing})
			row.Links = []ReadinessLink{{Label: mReadyAddCodes.Keep(nil), Href: page + "?add=1"}}
			out = append(out, row)
			continue
		}
		count := min(int64(math.Ceil(float64(missing)*1.05)), MaxGenerate)
		b, err := s.Q.CouponBatchRunning(ctx, p.ID)
		switch {
		case err == nil:
			// Codes are being created: no button, who started it.
			row.Title = mReadyCreating.Keep(i18n.Args{"missing": missing, "store": p.StoreName, "done": b.Created})
			if b.Reason == "refill" || b.CreatedBy == nil {
				row.Detail = mReadyCreatingRefill.Keep(i18n.Args{"time": at(b.CreatedAt), "store": p.StoreName})
			} else {
				who := ""
				if u, err := s.Q.GetUser(ctx, *b.CreatedBy); err == nil {
					who = u.Name
				}
				row.Detail = mReadyCreatingBy.Keep(i18n.Args{"time": at(b.CreatedAt), "who": who, "store": p.StoreName})
			}
		case errors.Is(err, pgx.ErrNoRows):
			row.Title = mReadyCreating.Keep(i18n.Args{"missing": missing, "store": p.StoreName, "done": 0})
			row.Detail = mReadyStoreShort.Keep(i18n.Args{"name": p.Name, "platform": platformName(p.Source), "free": free, "store": p.StoreName, "count": count})
			row.Links = []ReadinessLink{{Label: mReadyCreateCodes.Keep(nil), Href: page + "?create=" + strconv.FormatInt(count, 10)}}
		default:
			continue
		}
		out = append(out, row)
	}
	return out
}

// PublishChecks implements automations.EmailPublishChecker: a badge for a
// pool the flow's emails name that is under its low mark, archived or
// doesn't exist, in the member's language.
func (s *Service) PublishChecks(ctx context.Context, ws *store.Workspace, emails []automations.EmailContent) []automations.AutomationPublishCheck {
	var slugs []string
	for _, e := range emails {
		for _, slug := range usedSlugs(liquid.ExpandBlocks(e.Subject + "\n" + e.Preheader + "\n" + e.HTML + "\n" + e.MJML)) {
			if !contains(slugs, slug) {
				slugs = append(slugs, slug)
			}
		}
	}
	if len(slugs) == 0 {
		return nil
	}
	rows, err := s.pools(ctx, ws.ID, nil, slugs)
	if err != nil {
		return nil
	}
	l := i18n.LangFrom(ctx)
	known := map[string]store.CouponPoolsRow{}
	for _, p := range rows {
		known[p.Slug] = p
	}
	var out []automations.AutomationPublishCheck
	for _, slug := range slugs {
		p, ok := known[slug]
		switch {
		case !ok:
			out = append(out, automations.AutomationPublishCheck{Status: "warn", Label: mPublishUnknown.Text(l, i18n.Args{"slug": slug})})
		case p.Status == StatusArchived:
			out = append(out, automations.AutomationPublishCheck{Status: "warn", Label: mPublishArchived.Text(l, i18n.Args{"name": p.Name})})
		default:
			if free, err := s.Q.CouponPoolFree(ctx, p.ID); err == nil && free < int64(p.LowAt) {
				out = append(out, automations.AutomationPublishCheck{Status: "warn", Label: mPublishLow.Text(l, i18n.Args{"name": p.Name, "free": free, "low": p.LowAt})})
			}
		}
	}
	return out
}

// EmailHold implements automations.EmailHold: an automation email that
// shows a pool's code waits while the pool has none for the contact
// (no_codes), and the pool's low alert fires.
func (s *Service) EmailHold(ctx context.Context, wsID, emailID uuid.UUID, version int, contactID uuid.UUID) (string, error) {
	content, err := s.Q.CouponAutomationEmailContent(ctx, store.CouponAutomationEmailContentParams{EmailID: emailID, WorkspaceID: wsID, Version: int32(version)})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	slugs := usedSlugs(liquid.ExpandBlocks(content))
	if len(slugs) == 0 {
		return "", nil
	}
	rows, err := s.pools(ctx, wsID, nil, slugs)
	if err != nil {
		return "", err
	}
	for _, p := range rows {
		if p.OnePerContact {
			has, err := s.Q.CouponContactHasCode(ctx, store.CouponContactHasCodeParams{PoolID: p.ID, ContactID: &contactID})
			if err != nil {
				return "", err
			}
			if has {
				continue
			}
		}
		if p.Status == StatusActive {
			free, err := s.Q.CouponPoolHasFree(ctx, p.ID)
			if err != nil {
				return "", err
			}
			if free {
				continue
			}
			s.alertLow(ctx, p, 0)
		}
		return liquid.ShortNoCodes, nil
	}
	return "", nil
}
