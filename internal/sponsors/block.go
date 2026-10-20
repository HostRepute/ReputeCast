package sponsors

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Sponsor block is an mj-section with ReputeCast's own attributes; the
// compile step removes them (internal/mjml.PrepareCompile):
//
//	rc-sponsor         the block (its value is the slot's id, or "1")
//	rc-sponsor-layout  logo-left (default), image-top or text
//	rc-sponsor-label   plain ("Sponsored") or name ("Sponsored by …")
//	rc-sponsor-hash    what the block was filled with, to tell a creative
//	                   that changed since the email was saved
//
// Its content comes from the issue's booking: the creative once it's
// ready, a placeholder while it's waiting (Send Readiness won't let the
// issue go out), and without a booking a note the composer shows and the
// email leaves out (rc-omit, internal/mjml.PrepareCompile). The label can't
// be removed.
const (
	BlockAttr  = "rc-sponsor"
	LayoutAttr = "rc-sponsor-layout"
	LabelAttr  = "rc-sponsor-label"
	HashAttr   = "rc-sponsor-hash"
)

// Block states.
const (
	BlockReady   = "ready"
	BlockWaiting = "waiting"
	BlockNone    = "none"
)

// SponsorBlock is what the composer's Sponsor block shows for a campaign.
type SponsorBlock struct {
	// State: ready (the creative), waiting (a placeholder, the issue can't
	// go out with it), none (no booking: the block is hidden).
	State     string     `json:"state" enum:"ready,waiting,none"`
	Issue     int        `json:"issue,omitempty"`
	IssueOn   *time.Time `json:"issueOn,omitempty"`
	SlotID    string     `json:"slotId,omitempty"`
	SlotName  string     `json:"slotName,omitempty"`
	ListName  string     `json:"listName,omitempty"`
	BookingID string     `json:"bookingId,omitempty"`
	Booking   string     `json:"bookingStatus,omitempty" enum:"hold,booked,sent,"`
	Sponsor   string     `json:"sponsor,omitempty"`
	Initials  string     `json:"initials,omitempty"`
	// CreativeDue is when a waiting creative is due.
	CreativeDue *time.Time      `json:"creativeDue,omitempty"`
	Creative    SponsorCreative `json:"creative"`
	// Link is the creative's link with its UTM parameters, as sent.
	Link string `json:"link"`
	// PriceCents is shown only to people who can see Earn (earn:view).
	PriceCents *int64 `json:"priceCents,omitempty"`
	Currency   string `json:"currency,omitempty"`
	Label      string `json:"label"`
	Layout     string `json:"layout" enum:"logo-left,image-top,text"`
	LabelMode  string `json:"labelMode" enum:"plain,name"`
	// MJML is the block's content for its state, and Hash what fills it.
	MJML string `json:"mjml"`
	Hash string `json:"hash"`
	// CanSeePrice: the person holds earn:view; Helpers can help when not.
	CanSeePrice bool     `json:"canSeePrice"`
	Helpers     []string `json:"helpers"`
}

// blockBooking finds a campaign's booking: linked to it, or of its list on
// its date. The issue is known from the date even without a booking.
type blockBooking struct {
	booking *store.SponsorBooking
	sponsor *store.Sponsor
	slot    *store.SponsorSlotsRow
	issue   int
	issueOn time.Time
}

func (s *Service) campaignBooking(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (blockBooking, error) {
	var out blockBooking
	cp, err := s.Q.SponsorCampaignBrief(ctx, store.SponsorCampaignBriefParams{ID: campaignID, WorkspaceID: ws.ID})
	if err != nil {
		return out, api.MapError(err, "Campaign")
	}
	slots, err := s.Q.SponsorSlots(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	slotOfID := func(id uuid.UUID) *store.SponsorSlotsRow {
		for i := range slots {
			if slots[i].ID == id {
				return &slots[i]
			}
		}
		return nil
	}
	if b, err := s.Q.SponsorBookingOfCampaign(ctx, &campaignID); err == nil {
		bk := store.SponsorBooking{ID: b.ID, WorkspaceID: b.WorkspaceID, SlotID: b.SlotID, IssueNumber: b.IssueNumber, IssueOn: b.IssueOn, SponsorID: b.SponsorID,
			CampaignID: b.CampaignID, PriceCents: b.PriceCents, Currency: b.Currency, Status: b.Status, Creative: b.Creative, CreativeStatus: b.CreativeStatus,
			CreativeDue: b.CreativeDue, PaymentStatus: b.PaymentStatus}
		out.booking, out.slot, out.issue, out.issueOn = &bk, slotOfID(b.SlotID), int(b.IssueNumber), b.IssueOn
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	dated := true
	if out.booking == nil && (cp.StartedAt == nil && cp.ScheduledAt == nil) {
		// Without a date, a booking that was canceled still names the
		// issue, and the issue's new booking is the block's.
		if last, err := s.Q.SponsorLastIssueOfCampaign(ctx, &campaignID); err == nil {
			dated = false
			out.slot, out.issue, out.issueOn = slotOfID(last.SlotID), int(last.IssueNumber), last.IssueOn
			if b, err := s.Q.SponsorBookingOfIssue(ctx, store.SponsorBookingOfIssueParams{SlotID: last.SlotID, IssueNumber: last.IssueNumber}); err == nil &&
				(b.CampaignID == nil || *b.CampaignID == campaignID) {
				bk := b
				out.booking = &bk
			}
		}
	}
	if out.booking == nil && dated {
		at := firstTime(cp.StartedAt, cp.ScheduledAt)
		var lists []string
		var aud struct {
			Lists []string `json:"lists"`
		}
		_ = json.Unmarshal(cp.Audience, &aud)
		lists = aud.Lists
		for i := range slots {
			sl := &slots[i]
			if !contains(lists, sl.ListID.String()) {
				continue
			}
			if out.slot == nil {
				out.slot = sl
			}
			if at == nil {
				continue
			}
			n, ok := NumberOn(slotRow(*sl), localDay(*at, ws))
			if !ok {
				continue
			}
			out.slot, out.issue, out.issueOn = sl, n, IssueOn(slotRow(*sl), n)
			if b, err := s.Q.SponsorBookingOfIssue(ctx, store.SponsorBookingOfIssueParams{SlotID: sl.ID, IssueNumber: int32(n)}); err == nil {
				if b.CampaignID == nil || *b.CampaignID == campaignID {
					bk := b
					out.booking = &bk
				}
			}
			break
		}
	}
	if out.booking != nil && out.booking.SponsorID != nil {
		if sp, err := s.Q.SponsorGet(ctx, store.SponsorGetParams{ID: *out.booking.SponsorID, WorkspaceID: ws.ID}); err == nil {
			out.sponsor = &sp
		}
	}
	return out, nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// CampaignBlock is the Sponsor block of a campaign for the composer.
func (s *Service) CampaignBlock(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, layout, labelMode string) (SponsorBlock, error) {
	bb, err := s.campaignBooking(ctx, ws, campaignID)
	if err != nil {
		return SponsorBlock{}, err
	}
	out := s.block(ctx, ws, bb, layout, labelMode)
	out.CanSeePrice = p.Can(authz.EarnView)
	out.Helpers = []string{}
	if out.CanSeePrice && bb.booking != nil {
		pc := bb.booking.PriceCents
		out.PriceCents, out.Currency = &pc, bb.booking.Currency
	}
	if !out.CanSeePrice {
		out.Helpers = s.helpers(ctx, ws.ID, authz.EarnView, p.UserID, 2)
	}
	return out, nil
}

func (s *Service) block(ctx context.Context, ws *store.Workspace, bb blockBooking, layout, labelMode string) SponsorBlock {
	if layout != "image-top" && layout != "text" {
		layout = "logo-left"
	}
	if labelMode != "plain" {
		labelMode = "name"
	}
	out := SponsorBlock{State: BlockNone, Layout: layout, LabelMode: labelMode, Label: text(ctx, "block.label", nil)}
	if bb.slot != nil {
		out.SlotID, out.SlotName, out.ListName = bb.slot.ID.String(), bb.slot.Name, bb.slot.ListName
	}
	if bb.issue > 0 {
		out.Issue = bb.issue
		t := noonOn(bb.issueOn, ws)
		out.IssueOn = &t
	}
	if b := bb.booking; b != nil && (b.Status == "booked" || b.Status == "sent") {
		out.BookingID, out.Booking = b.ID.String(), b.Status
		out.Creative = parseCreative(b.Creative)
		out.Link = WithUTM(out.Creative.Link, out.Creative.UTM)
		out.CreativeDue = noonPtr(b.CreativeDue, ws)
		if bb.sponsor != nil {
			out.Sponsor, out.Initials = bb.sponsor.Name, initials(bb.sponsor.Name)
			if out.Creative.Logo == "" {
				out.Creative.Logo = bb.sponsor.LogoUrl
			}
		}
		if labelMode == "name" && out.Sponsor != "" {
			out.Label = text(ctx, "block.labelBy", map[string]any{"sponsor": out.Sponsor})
		}
		out.State = BlockWaiting
		if b.CreativeStatus == "ready" {
			out.State = BlockReady
		}
	} else if b != nil && b.Status == "hold" {
		out.BookingID, out.Booking = b.ID.String(), b.Status
	}
	out.MJML = s.blockMJML(ctx, ws, out)
	out.Hash = blockHash(out)
	return out
}

// blockHash names what fills a block: its state, issue, label, layout and
// the creative.
func blockHash(b SponsorBlock) string {
	raw, _ := json.Marshal([]any{b.State, b.Issue, b.Label, b.Layout, b.Creative, b.Link, b.Sponsor})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:6])
}

const (
	serif = `Georgia, 'Times New Roman', serif`
	sans  = `'IBM Plex Sans', Helvetica, Arial, sans-serif`
)

func esc(s string) string { return html.EscapeString(s) }

// The card's colours and type, as the ComposerEarnBlocks board draws them.
const (
	cardLabel  = `font-family:'IBM Plex Sans',Helvetica,Arial,sans-serif;font-size:12px;font-weight:700;letter-spacing:0.06em;text-transform:uppercase;color:#5B6472;`
	cardNote   = `font-family:'IBM Plex Sans',Helvetica,Arial,sans-serif;font-size:13.5px;line-height:1.5;color:#374151;margin:0;`
	cardSquare = `width:44px;height:44px;line-height:44px;border-radius:8px;background-color:#F3F4F6;border:1px solid #E3E6EB;text-align:center;` +
		`font-family:'IBM Plex Sans',Helvetica,Arial,sans-serif;font-size:13px;font-weight:700;color:#374151;`
)

// blockMJML is a Sponsor block's content: one card, so it looks the same on
// the canvas and in every mail app. Ready, the creative in its layout;
// waiting, a placeholder (Send Readiness won't let the issue go out); with
// no booking, a note the composer shows and the email leaves out (rc-omit).
// The label is 12 px, the accessibility policy's smallest text.
func (s *Service) blockMJML(ctx context.Context, ws *store.Workspace, b SponsorBlock) string {
	attrs := fmt.Sprintf(`%s="%s" %s="%s" %s="%s"`, BlockAttr, esc(firstNonEmpty(b.SlotID, "1")), LayoutAttr, b.Layout, LabelAttr, b.LabelMode)
	label := b.Label
	if b.LabelMode == "name" && b.Sponsor != "" && b.State != BlockReady {
		label = text(ctx, "block.placeholderLabel", map[string]any{"sponsor": b.Sponsor})
	}
	labelDiv := fmt.Sprintf(`<div style="%spadding-bottom:6px;">%s</div>`, cardLabel, esc(label))
	section := func(extra, card string) string {
		return fmt.Sprintf(`<mj-section %s%s background-color="#ffffff" padding="12px 24px"><mj-column><mj-text padding="0px">%s</mj-text></mj-column></mj-section>`, attrs, extra, card)
	}
	square := func() string {
		return fmt.Sprintf(`<div style="%s" aria-hidden="true">%s</div>`, cardSquare, esc(b.Initials))
	}
	// A card: a table, the side column (the logo or initials) and the main one.
	card := func(border, background, side, main string) string {
		out := fmt.Sprintf(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="border:%s;border-radius:8px;background-color:%s;border-collapse:separate;"><tr>`,
			border, background)
		if side != "" {
			out += `<td width="44" valign="top" style="width:44px;padding:16px 0px 16px 18px;">` + side + `</td>`
			out += `<td valign="top" style="padding:16px 18px 16px 14px;">` + main + `</td>`
		} else {
			out += `<td valign="top" style="padding:16px 18px;">` + main + `</td>`
		}
		return out + `</tr></table>`
	}
	switch b.State {
	case BlockNone:
		// The composer shows why the block is empty; the email leaves it out.
		note := text(ctx, "block.noneNoIssue", nil)
		switch {
		case b.Issue > 0:
			note = text(ctx, "block.none", map[string]any{"issue": b.Issue})
		case b.SlotID != "":
			note = text(ctx, "block.noDate", nil)
		}
		main := labelDiv + fmt.Sprintf(`<p style="%s">%s</p>`, cardNote, esc(note))
		return section(fmt.Sprintf(` %s="1"`, mjml.AttrOmit), card("1px dashed #D4D8DF", "#F9FAFB", "", main))
	case BlockWaiting:
		due := ""
		if b.CreativeDue != nil {
			due = b.CreativeDue.In(location(ws)).Format("Mon 2 Jan")
		}
		main := labelDiv + fmt.Sprintf(`<p style="%s">%s</p>`, cardNote, esc(text(ctx, "block.waiting", map[string]any{"due": due})))
		return section("", card("1px dashed #C9CED6", "#F9FAFB", square(), main))
	}
	c := b.Creative
	link := esc(b.Link)
	main := labelDiv
	if c.Headline != "" {
		main += fmt.Sprintf(`<p style="margin:0;padding-bottom:6px;font-family:%s;font-size:17px;font-weight:700;line-height:1.3;color:#0B1220;">%s</p>`, serif, esc(c.Headline))
	}
	if c.Text != "" {
		main += fmt.Sprintf(`<p style="margin:0;padding-bottom:12px;font-family:%s;font-size:14px;line-height:1.55;color:#374151;">%s</p>`, serif, esc(c.Text))
	}
	if c.Button != "" && b.Link != "" {
		main += fmt.Sprintf(`<a href="%s" style="display:inline-block;background-color:#0B1220;color:#ffffff;font-family:%s;font-size:13px;font-weight:600;line-height:1;`+
			`padding:9px 14px;border-radius:6px;text-decoration:none;">%s</a>`, link, sans, esc(c.Button))
	}
	alt := esc(b.Sponsor)
	switch b.Layout {
	case "image-top":
		if c.Image != "" {
			img := fmt.Sprintf(`<a href="%s"><img src="%s" alt="%s" width="100%%" style="display:block;width:100%%;height:auto;border-radius:6px;"></a>`, link, esc(c.Image), alt)
			main = `<div style="padding-bottom:12px;">` + img + `</div>` + main
		}
		return section("", card("1px solid #E3E6EB", "#ffffff", "", main))
	case "text":
		return section("", card("1px solid #E3E6EB", "#ffffff", "", main))
	}
	side := square()
	if c.Logo != "" {
		side = fmt.Sprintf(`<a href="%s"><img src="%s" alt="%s" width="44" height="44" style="display:block;width:44px;height:44px;border-radius:8px;border:1px solid #E3E6EB;"></a>`,
			link, esc(c.Logo), alt)
	}
	return section("", card("1px solid #E3E6EB", "#ffffff", side, main))
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// SponsorFacts is what Send Readiness checks of a campaign's Sponsor block.
type SponsorFacts struct {
	HasBlock bool
	State    string
	Sponsor  string
	Issue    int
	// BookingID and CreativeDue name the booking to open.
	BookingID   string
	CreativeDue *time.Time
	// Stale: the creative or the booking changed since the email was saved.
	Stale bool
	// Labelled: the block carries the "Sponsored" label.
	Labelled bool
}

// blockNodes are the Sponsor blocks of an email.
func blockNodes(src string) []*mjml.Node {
	if !strings.Contains(src, BlockAttr) {
		return nil
	}
	root, err := mjml.Parse(src)
	if err != nil {
		return nil
	}
	var out []*mjml.Node
	root.Walk(func(n *mjml.Node, _ []int) {
		if _, ok := n.Get(BlockAttr); ok {
			out = append(out, n)
		}
	})
	return out
}

// ReadinessFacts reads a campaign's Sponsor block for Send Readiness.
func (s *Service) ReadinessFacts(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, src string) SponsorFacts {
	nodes := blockNodes(src)
	if len(nodes) == 0 {
		return SponsorFacts{}
	}
	n := nodes[0]
	bb, err := s.campaignBooking(ctx, ws, campaignID)
	if err != nil {
		s.log().WarnContext(ctx, "sponsor block facts", "campaign", campaignID, "err", err)
		return SponsorFacts{HasBlock: true, State: BlockNone}
	}
	b := s.block(ctx, ws, bb, n.Attr(LayoutAttr), n.Attr(LabelAttr))
	out := SponsorFacts{HasBlock: true, State: b.State, Sponsor: b.Sponsor, Issue: b.Issue, BookingID: b.BookingID, CreativeDue: b.CreativeDue,
		Stale: n.Attr(HashAttr) != b.Hash, Labelled: strings.Contains(strings.ToUpper(src[n.Start:n.End]), strings.ToUpper(text(ctx, "block.label", nil)))}
	if b.State == BlockNone {
		out.Labelled = true
	}
	return out
}

// blockPath is a block's element-index path from mj-body, as the composer's
// checks locate elements, and its top-level section.
func blockPath(src string, n *mjml.Node) ([]int, int) {
	root, err := mjml.Parse(src)
	if err != nil {
		return []int{}, -1
	}
	body := mjml.Body(root)
	if body == nil {
		return []int{}, -1
	}
	var found []int
	body.Walk(func(x *mjml.Node, path []int) {
		if found == nil && len(path) > 0 && x.Start == n.Start && x.End == n.End {
			found = path
		}
	})
	if len(found) == 0 {
		return []int{}, -1
	}
	return found, found[0]
}

// ends is "s" when a name ends in an s, for possessives in messages
// ("Northwind Analytics’ creative").
func ends(name string) string {
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(name)), "s") {
		return "s"
	}
	return "other"
}

// ComposerIssues are the Sponsor block's findings for the composer's checks:
// a creative still waiting blocks the issue, as in Review & send, with the
// booking to open and the block to remove.
func (s *Service) ComposerIssues(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, src string) []contentcheck.ContentCheckIssue {
	nodes := blockNodes(src)
	if len(nodes) == 0 {
		return nil
	}
	bb, err := s.campaignBooking(ctx, ws, campaignID)
	if err != nil {
		return nil
	}
	n := nodes[0]
	b := s.block(ctx, ws, bb, n.Attr(LayoutAttr), n.Attr(LabelAttr))
	if b.State != BlockWaiting {
		return nil
	}
	path, section := blockPath(src, n)
	detail := text(ctx, "check.missingNoDue", nil)
	if b.CreativeDue != nil {
		detail = text(ctx, "check.missingDetail", map[string]any{"due": b.CreativeDue.In(location(ws)).Format("Mon 2 Jan")})
	}
	fixes := []contentcheck.ContentCheckFix{{Kind: "open", Label: text(ctx, "check.openBooking", nil), Value: "/w/" + ws.Slug + "/earn/sponsors?booking=" + b.BookingID}}
	if section >= 0 {
		fixes = append(fixes, contentcheck.ContentCheckFix{Kind: "remove_section", Label: text(ctx, "check.remove", nil), Path: []int{section}})
	}
	return []contentcheck.ContentCheckIssue{{ID: "content.sponsor_creative", Severity: contentcheck.Block, Group: "Content",
		Title: text(ctx, "check.missing", map[string]any{"sponsor": b.Sponsor, "ends": ends(b.Sponsor)}), Detail: detail, Tag: text(ctx, "check.tag", nil),
		Path: path, Section: section, Fixes: fixes}}
}

// Fix actions of the Sponsor block in Review & send.
const (
	FixRemoveBlock  = "sponsor_remove_block"
	FixRefreshBlock = "sponsor_refresh_block"
)

// FixMJML applies a Sponsor block fix to an email's MJML: remove the block,
// or fill it again from the booking.
func (s *Service) FixMJML(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, src, action string) (string, string, error) {
	nodes := blockNodes(src)
	if len(nodes) == 0 {
		return src, "", errors.New("the email has no Sponsor block any more")
	}
	n := nodes[0]
	switch action {
	case FixRemoveBlock:
		start, end := n.Start, n.End
		// The block's marker comment goes with it.
		if i := strings.LastIndex(src[:start], "<!-- rc:sponsor -->"); i >= 0 && strings.TrimSpace(src[i+len("<!-- rc:sponsor -->"):start]) == "" {
			start = i
		}
		return src[:start] + src[end:], text(ctx, "fix.removed", nil), nil
	case FixRefreshBlock:
		bb, err := s.campaignBooking(ctx, ws, campaignID)
		if err != nil {
			return src, "", err
		}
		b := s.block(ctx, ws, bb, n.Attr(LayoutAttr), n.Attr(LabelAttr))
		// The writer's own settings stay: the block's name, Visibility and classes.
		own := ""
		for _, a := range []string{mjml.AttrName, mjml.AttrVisible, "css-class"} {
			if v, ok := n.Get(a); ok && v != "" {
				own += fmt.Sprintf(`%s="%s" `, a, esc(v))
			}
		}
		next := strings.Replace(b.MJML, BlockAttr+`="`, own+HashAttr+`="`+b.Hash+`" `+BlockAttr+`="`, 1)
		return src[:n.Start] + next + src[n.End:], text(ctx, "fix.refreshed", nil), nil
	}
	return src, "", errors.New("unknown fix")
}
