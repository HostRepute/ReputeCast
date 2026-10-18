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
// issue go out), nothing without a booking. The label can't be removed.
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
	if out.booking == nil {
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

// blockHash names what fills a block: its state, label, layout and the
// creative.
func blockHash(b SponsorBlock) string {
	raw, _ := json.Marshal([]any{b.State, b.Label, b.Layout, b.Creative, b.Link, b.Sponsor})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:6])
}

const (
	serif = `Georgia, 'Times New Roman', serif`
	sans  = `'IBM Plex Sans', Helvetica, Arial, sans-serif`
)

func esc(s string) string { return html.EscapeString(s) }

// blockMJML is a Sponsor block's content: the creative in its layout, the
// waiting placeholder, or an empty section that renders nothing.
func (s *Service) blockMJML(ctx context.Context, ws *store.Workspace, b SponsorBlock) string {
	attrs := fmt.Sprintf(`%s="%s" %s="%s" %s="%s"`, BlockAttr, esc(firstNonEmpty(b.SlotID, "1")), LayoutAttr, b.Layout, LabelAttr, b.LabelMode)
	label := strings.ToUpper(b.Label)
	if b.LabelMode == "name" && b.Sponsor != "" && b.State != BlockReady {
		label = strings.ToUpper(text(ctx, "block.placeholderLabel", map[string]any{"sponsor": b.Sponsor}))
	}
	labelText := fmt.Sprintf(`<mj-text font-family="%s" font-size="11px" font-weight="700" letter-spacing="0.06em" color="#5B6472" padding="0px 0px 6px">%s</mj-text>`, sans, esc(label))
	switch b.State {
	case BlockNone:
		return fmt.Sprintf(`<mj-section %s padding="0px"><mj-column><mj-raw><!-- rc:sponsor none --></mj-raw></mj-column></mj-section>`, attrs)
	case BlockWaiting:
		due := ""
		if b.CreativeDue != nil {
			due = b.CreativeDue.In(location(ws)).Format("Mon 2 Jan")
		}
		note := text(ctx, "block.waiting", map[string]any{"due": due})
		box := fmt.Sprintf(`<mj-column border="1px dashed #C9CED6" background-color="#F8FAFC" padding="14px 16px">%s<mj-text font-family="%s" font-size="13px" line-height="1.5" color="#374151" padding="0px">%s</mj-text></mj-column>`,
			labelText, sans, esc(note))
		return fmt.Sprintf(`<mj-section %s background-color="#ffffff" padding="12px 24px">%s</mj-section>`, attrs, box)
	}
	c := b.Creative
	link := esc(b.Link)
	head := ""
	if c.Headline != "" {
		head = fmt.Sprintf(`<mj-text font-family="%s" font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px"><h3 style="margin:0;font-size:17px;font-weight:700">%s</h3></mj-text>`,
			serif, esc(c.Headline))
	}
	body := ""
	if c.Text != "" {
		body = fmt.Sprintf(`<mj-text font-family="%s" font-size="14px" line-height="1.55" color="#374151" padding="0px 0px 12px">%s</mj-text>`, serif, esc(c.Text))
	}
	button := ""
	if c.Button != "" && b.Link != "" {
		button = fmt.Sprintf(`<mj-button href="%s" align="left" background-color="#0B1220" color="#ffffff" font-family="%s" font-size="13px" font-weight="600" border-radius="6px" inner-padding="9px 14px" padding="0px">%s</mj-button>`,
			link, sans, esc(c.Button))
	}
	alt := esc(b.Sponsor)
	logo := ""
	switch {
	case c.Logo != "":
		logo = fmt.Sprintf(`<mj-image src="%s" alt="%s" width="44px" href="%s" align="left" padding="0px"></mj-image>`, esc(c.Logo), alt, link)
	default:
		logo = fmt.Sprintf(`<mj-text align="center" font-family="%s" font-size="13px" font-weight="700" color="#374151" container-background-color="#F3F4F6" padding="12px 0px">%s</mj-text>`,
			sans, esc(b.Initials))
	}
	open := fmt.Sprintf(`<mj-section %s background-color="#ffffff" border="1px solid #E3E6EB" border-radius="8px" padding="16px 16px">`, attrs)
	switch b.Layout {
	case "image-top":
		img := ""
		if c.Image != "" {
			img = fmt.Sprintf(`<mj-image src="%s" alt="%s" href="%s" padding="0px 0px 12px"></mj-image>`, esc(c.Image), alt, link)
		}
		return open + `<mj-column>` + img + labelText + head + body + button + `</mj-column></mj-section>`
	case "text":
		return open + `<mj-column>` + labelText + head + body + button + `</mj-column></mj-section>`
	}
	return open + `<mj-column width="15%" vertical-align="top" padding="0px 12px 0px 0px">` + logo + `</mj-column><mj-column width="85%">` + labelText + head + body + button +
		`</mj-column></mj-section>`
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
		next := strings.Replace(b.MJML, BlockAttr+`="`, HashAttr+`="`+b.Hash+`" `+BlockAttr+`="`, 1)
		return src[:n.Start] + next + src[n.End:], text(ctx, "fix.refreshed", nil), nil
	}
	return src, "", errors.New("unknown fix")
}
