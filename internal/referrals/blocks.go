package referrals

import (
	"context"
	"encoding/json"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Referral block (brief §3.4) and the Reward block (milestone emails):
// custom composer blocks stored as markers,
//
//	<mj-raw><!-- rc:referral {"layout":"progress",…} --></mj-raw>
//	<mj-raw><!-- rc:reward {"button":"Download the Field Guide",…} --></mj-raw>
//
// that liquid.ExpandBlocks turns into email-safe HTML (tables, no images)
// rendered from the reader's referral.* and reward.* variables. A reader
// outside the program, a program that is off and the public archive get
// nothing: the block hides itself.

func init() {
	liquid.RegisterBlock("referral", func(raw []byte) string {
		b := parseReferralBlock(raw)
		return bindProgram(b.Program, ExpandReferral(b))
	})
	liquid.RegisterBlock("reward", func(raw []byte) string { return ExpandReward(parseRewardBlock(raw)) })
	// The Referral tokens of the composer's token menu: empty for readers
	// outside a program (the block hides itself; the checks say so).
	liquid.RegisterTokens(
		liquid.PersonalizationToken{Key: "referral.link", Label: "Referral link", Kind: "referral"},
		liquid.PersonalizationToken{Key: "referral.code", Label: "Referral code", Kind: "referral"},
		liquid.PersonalizationToken{Key: "referral.hub_url", Label: "Rewards page link", Kind: "referral"},
		liquid.PersonalizationToken{Key: "referral.count", Label: "Counted referrals", Kind: "referral"},
		liquid.PersonalizationToken{Key: "referral.pending", Label: "Referrals not counted yet", Kind: "referral"},
		liquid.PersonalizationToken{Key: "referral.next_milestone.count", Label: "Next milestone", Kind: "referral"},
		liquid.PersonalizationToken{Key: "referral.next_milestone.reward", Label: "Next reward", Kind: "referral"},
		liquid.PersonalizationToken{Key: "referral.to_next", Label: "Referrals to the next reward", Kind: "referral"},
		liquid.PersonalizationToken{Key: "referral.rewards", Label: "Rewards earned", Kind: "referral"},
	)
}

// ReferralBlockTexts are the block's headline for 0, 1 and several counted
// referrals, with referral tokens.
type ReferralBlockTexts struct {
	None string `json:"none" maxLength:"300"`
	One  string `json:"one" maxLength:"300"`
	Many string `json:"many" maxLength:"300"`
}

// ReferralBlock is the Referral block's settings.
type ReferralBlock struct {
	// Program binds the block to a program; without it, the reader's first.
	Program string `json:"program,omitempty" format:"uuid"`
	// Layout: a single line, a card with progress, or a card with the next
	// rewards.
	Layout   string             `json:"layout" enum:"line,progress,rewards"`
	Texts    ReferralBlockTexts `json:"texts"`
	Button   string             `json:"button" maxLength:"60"`
	Share    []string           `json:"share" maxItems:"4"`
	Progress bool               `json:"progress"`
}

// DefaultReferralBlock is a new Referral block.
func DefaultReferralBlock() ReferralBlock {
	return ReferralBlock{Layout: "progress", Button: "Share your link", Share: []string{ViaEmail, ViaX, ViaLinkedIn, ViaWhatsApp}, Progress: true,
		Texts: ReferralBlockTexts{
			None: `Know someone who'd like it? Share your link: {{ referral.to_next }} friend and you get {{ referral.next_milestone.reward }}.`,
			One:  `You've referred 1 friend. {{ referral.to_next }} more and you get {{ referral.next_milestone.reward }}.`,
			Many: `You've referred {{ referral.count }} friends. {{ referral.to_next }} more and you get {{ referral.next_milestone.reward }}.`,
		}}
}

func parseReferralBlock(raw []byte) ReferralBlock {
	b := DefaultReferralBlock()
	if len(raw) > 0 {
		var in ReferralBlock
		if json.Unmarshal(raw, &in) == nil {
			if _, err := uuid.Parse(in.Program); err == nil {
				b.Program = in.Program
			}
			if in.Layout == "line" || in.Layout == "progress" || in.Layout == "rewards" {
				b.Layout = in.Layout
			}
			if strings.TrimSpace(in.Texts.None) != "" {
				b.Texts.None = in.Texts.None
			}
			if strings.TrimSpace(in.Texts.One) != "" {
				b.Texts.One = in.Texts.One
			}
			if strings.TrimSpace(in.Texts.Many) != "" {
				b.Texts.Many = in.Texts.Many
			}
			if strings.TrimSpace(in.Button) != "" {
				b.Button = in.Button
			}
			if in.Share != nil {
				b.Share = in.Share
			}
			b.Progress = in.Progress
		}
	}
	return b
}

// bindProgram makes a block read one program's referral.* variables
// (referral_programs["<id>"]), and gives the template's referral back after.
func bindProgram(program, body string) string {
	if _, err := uuid.Parse(program); err != nil {
		return body
	}
	return `{% assign rc_referral_prev = referral %}{% assign referral = referral_programs["` + program + `"] %}` + body + `{% assign referral = rc_referral_prev %}`
}

// RewardBlock is the Reward block's settings (milestone emails).
type RewardBlock struct {
	Button      string `json:"button" maxLength:"80"`
	Image       bool   `json:"image"`
	Description bool   `json:"description"`
	Expiry      bool   `json:"expiry"`
	Hub         bool   `json:"hub"`
}

func parseRewardBlock(raw []byte) RewardBlock {
	b := RewardBlock{Button: "Get your reward", Image: true, Description: true, Expiry: true, Hub: true}
	if len(raw) > 0 {
		var in RewardBlock
		if json.Unmarshal(raw, &in) == nil {
			if strings.TrimSpace(in.Button) != "" {
				b.Button = in.Button
			}
			b.Image, b.Description, b.Expiry, b.Hub = in.Image, in.Description, in.Expiry, in.Hub
		}
	}
	return b
}

var liquidSpan = regexp.MustCompile(`\{\{[\s\S]*?\}\}|\{%[\s\S]*?%\}`)

// valueChip is how the composer's canvas marks a token's value in the
// block's text (ComposerGrowBlocks: "You've referred [2] friends").
const valueChip = `<span style="display:inline-block;padding:0 6px;border-radius:5px;background:#EDE9FE;color:#5B21B6;font-family:'IBM Plex Mono',monospace;font-size:.86em;font-weight:500;line-height:1.55;white-space:nowrap">`

// escTok escapes text for HTML and keeps its Liquid spans as they are; for
// the canvas (chips) it puts each {{ token }} in a chip.
func escTok(s string, chips bool) string {
	var b strings.Builder
	last := 0
	for _, m := range liquidSpan.FindAllStringIndex(s, -1) {
		b.WriteString(html.EscapeString(s[last:m[0]]))
		if chips && strings.HasPrefix(s[m[0]:m[1]], "{{") {
			b.WriteString(valueChip + s[m[0]:m[1]] + `</span>`)
		} else {
			b.WriteString(s[m[0]:m[1]])
		}
		last = m[1]
	}
	b.WriteString(html.EscapeString(s[last:]))
	return b.String()
}

const (
	font   = `font-family:'IBM Plex Sans',Arial,sans-serif;`
	accent = "#0F766E"
)

func shareLine(share []string) string {
	labels := map[string]string{ViaEmail: "Email", ViaX: "X", ViaLinkedIn: "LinkedIn", ViaWhatsApp: "WhatsApp"}
	var parts []string
	for _, ch := range []string{ViaEmail, ViaX, ViaLinkedIn, ViaWhatsApp} {
		for _, sh := range share {
			if sh == ch {
				parts = append(parts, `<a href="{{ referral.share.`+ch+` }}" style="color:`+accent+`;font-weight:600">`+labels[ch]+`</a>`)
			}
		}
	}
	hub := `<a href="{{ referral.hub_url }}" style="color:` + accent + `;font-weight:600">See your rewards</a>`
	if len(parts) == 0 {
		return `<p style="margin:0;` + font + `font-size:12.5px;color:#5B6472;line-height:1.7">` + hub + `</p>`
	}
	return `<p style="margin:0;` + font + `font-size:12.5px;color:#5B6472;line-height:1.7">Share: ` + strings.Join(parts, " &nbsp;·&nbsp; ") +
		` &nbsp;&nbsp;|&nbsp;&nbsp; ` + hub + `</p>`
}

// texts is the block's sentence for the reader's count: the text for none,
// one or several referrals, or, past the last milestone, a thank-you.
func texts(t ReferralBlockTexts, chips bool) string {
	return `{% if referral.next_milestone %}{% if referral.count == 0 %}` + escTok(t.None, chips) + `{% elsif referral.count == 1 %}` + escTok(t.One, chips) +
		`{% else %}` + escTok(t.Many, chips) + `{% endif %}{% else %}You've referred {{ referral.count }} {% if referral.count == 1 %}friend{% else %}friends{% endif %} and earned every reward. Thank you for sharing.{% endif %}`
}

func headline(t ReferralBlockTexts, size string, chips bool) string {
	return `<p style="margin:0 0 14px;` + font + `font-size:` + size + `;line-height:1.35;font-weight:700;color:#0B1220">` + texts(t, chips) + `</p>`
}

// ExpandReferral is the Referral block as Liquid.
func ExpandReferral(b ReferralBlock) string { return expandReferral(b, false) }

// expandReferral is the block as Liquid; chips marks the tokens' values for
// the composer's canvas.
func expandReferral(b ReferralBlock, chips bool) string {
	var out strings.Builder
	out.WriteString(`{% if referral.show %}`)
	out.WriteString(`<div class="rc-referral" style="padding:16px 24px">`)
	switch b.Layout {
	case "line":
		out.WriteString(`<p style="margin:0;` + font + `font-size:15px;line-height:1.6;color:#0B1220">`)
		out.WriteString(texts(b.Texts, chips))
		out.WriteString(` Your link: <a href="{{ referral.link }}" style="color:` + accent + `;font-weight:600">{{ referral.display }}</a></p>`)
	default:
		out.WriteString(`<div style="padding:20px;border:1px solid #E3E6EB;border-radius:8px;background:#F9FAFB">`)
		out.WriteString(`<p style="margin:0 0 6px;` + font + `font-size:11px;font-weight:700;letter-spacing:.06em;text-transform:uppercase;color:#5B6472">Share the {{ referral.list }}</p>`)
		out.WriteString(headline(b.Texts, "18px", chips))
		if b.Layout == "rewards" {
			out.WriteString(`{% if referral.upcoming.size > 0 %}<table role="presentation" style="width:100%;border-collapse:collapse;margin:0 0 14px">`)
			out.WriteString(`{% for m in referral.upcoming %}<tr><td style="padding:6px 0;` + font + `font-size:13px;color:#374151;border-top:1px solid #E3E6EB">` +
				`<b style="font-weight:600;color:#0B1220">{{ m.count }} {% if m.count == 1 %}referral{% else %}referrals{% endif %}</b> · {{ m.name }}</td>` +
				`<td style="padding:6px 0;` + font + `font-size:12px;color:#5B6472;text-align:right;border-top:1px solid #E3E6EB">{{ m.to_go }} to go</td></tr>{% endfor %}</table>{% endif %}`)
		}
		if b.Progress && b.Layout == "progress" {
			out.WriteString(`{% if referral.next_milestone %}<table role="presentation" style="width:100%;border-collapse:separate;border-spacing:4px 0;table-layout:fixed;margin:0"><tr>`)
			out.WriteString(`{% for cell in referral.cells %}<td style="height:8px;padding:0;background:{{ cell }};border-radius:4px;font-size:0;line-height:0">&nbsp;</td>{% endfor %}`)
			out.WriteString(`</tr></table>`)
			out.WriteString(`<table role="presentation" style="width:100%;margin:6px 0 16px;border-collapse:collapse"><tr>`)
			out.WriteString(`<td style="` + font + `font-size:12px;color:#374151;font-weight:600;padding:0">{{ referral.count }} of {{ referral.next_milestone.count }}</td>`)
			out.WriteString(`<td style="` + font + `font-size:12px;color:#5B6472;text-align:right;padding:0">{% if referral.soon > 0 %}{{ referral.soon }} counts soon{% endif %}</td>`)
			out.WriteString(`</tr></table>{% endif %}`)
		}
		out.WriteString(`<table role="presentation" style="width:100%;border-collapse:collapse;margin:0 0 12px"><tr>`)
		out.WriteString(`<td style="padding:0;vertical-align:middle"><span style="display:block;` + font + `font-size:11.5px;color:#5B6472;font-weight:600">Your link</span>`)
		out.WriteString(`<a href="{{ referral.link }}" style="` + font + `font-size:14px;font-weight:600;color:#0B1220;text-decoration:none">{{ referral.display }}</a></td>`)
		out.WriteString(`<td style="padding:0;text-align:right;vertical-align:middle"><a href="{{ referral.hub_url }}" style="display:inline-block;padding:9px 14px;border-radius:6px;background:` + accent + `;color:#fff;` + font + `font-size:13px;font-weight:600;white-space:nowrap;text-decoration:none">` + html.EscapeString(b.Button) + `</a></td>`)
		out.WriteString(`</tr></table>`)
		out.WriteString(shareLine(b.Share))
		out.WriteString(`</div>`)
	}
	out.WriteString(`</div>{% endif %}`)
	return out.String()
}

// ExpandReward is the Reward block as Liquid (milestone emails).
func ExpandReward(b RewardBlock) string {
	var out strings.Builder
	out.WriteString(`{% if reward.name %}<div class="rc-reward" style="padding:16px 24px">`)
	out.WriteString(`<table role="presentation" style="width:100%;border-collapse:separate;border-spacing:0;border:1px solid #E3E6EB;border-radius:8px"><tr>`)
	if b.Image {
		out.WriteString(`{% if reward.image_url %}<td style="padding:20px 0 20px 20px;width:96px;vertical-align:top"><img src="{{ reward.image_url }}" width="96" alt="{{ reward.name }}" style="display:block;width:96px;height:auto;border-radius:6px"></td>{% endif %}`)
	}
	out.WriteString(`<td style="padding:20px;vertical-align:top">`)
	out.WriteString(`<p style="margin:0 0 4px;` + font + `font-size:11px;font-weight:700;letter-spacing:.06em;text-transform:uppercase;color:#5B6472">Your reward</p>`)
	out.WriteString(`<p style="margin:0 0 4px;` + font + `font-size:19px;line-height:1.3;font-weight:700;color:#0B1220">{{ reward.name }}</p>`)
	if b.Description {
		out.WriteString(`{% if reward.description %}<p style="margin:0 0 14px;` + font + `font-size:14px;line-height:1.5;color:#374151">{{ reward.description }}</p>{% endif %}`)
	}
	out.WriteString(`{% if reward.code %}<p style="margin:0 0 14px;` + font + `font-size:13px;color:#374151">Your code: <b style="font-family:'IBM Plex Mono',monospace;font-size:15px;color:#0B1220">{{ reward.code }}</b></p>{% endif %}`)
	out.WriteString(`{% if reward.note %}<p style="margin:0 0 14px;` + font + `font-size:13px;color:#374151">{{ reward.note }}</p>{% endif %}`)
	out.WriteString(`{% if reward.url %}<a href="{{ reward.url }}" style="display:inline-block;padding:9px 14px;border-radius:6px;background:` + accent + `;color:#fff;` + font + `font-size:13px;font-weight:600;white-space:nowrap;text-decoration:none">` + html.EscapeString(b.Button) + `</a>{% endif %}`)
	if b.Expiry || b.Hub {
		out.WriteString(`<p style="margin:10px 0 0;` + font + `font-size:12.5px;color:#5B6472;line-height:1.5">`)
		if b.Expiry {
			out.WriteString(`{% if reward.download_url %}The link works for 7 days. {% elsif reward.expires_at %}Claim it by {{ reward.expires_at }}. {% endif %}`)
		}
		if b.Hub {
			out.WriteString(`You can get a fresh one from your <a href="{{ referral.hub_url }}" style="color:` + accent + `">rewards page</a>.`)
		}
		out.WriteString(`</p>`)
	}
	out.WriteString(`</td></tr></table></div>{% endif %}`)
	return out.String()
}

// --- Per-recipient variables ------------------------------------------------------

// Roots implements liquid.RecipientData.
func (s *Service) Roots() []string { return []string{"referral", "reward"} }

// Vars implements liquid.RecipientData: each reader's referral.* (their
// link and progress, when they're on a program's list while it's on) and,
// in milestone emails, reward.*.
func (s *Service) Vars(ctx context.Context, wsID uuid.UUID, recipients []liquid.Recipient, roots []string) (map[uuid.UUID]map[string]any, error) {
	out := map[uuid.UUID]map[string]any{}
	if len(recipients) == 0 || len(roots) == 0 {
		return out, nil
	}
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return out, err
	}
	pages, err := s.pages(ctx, &ws)
	if err != nil {
		return out, err
	}
	ids := make([]uuid.UUID, 0, len(recipients))
	for _, r := range recipients {
		if r.ContactID != uuid.Nil {
			ids = append(ids, r.ContactID)
		}
	}
	now := s.now()
	refs, err := s.referralVars(ctx, &ws, pages, ids, now)
	if err != nil {
		return out, err
	}
	for _, r := range recipients {
		v := map[string]any{"referral": map[string]any{"show": false}, "referral_programs": map[string]any{}}
		if rr, ok := refs[r.ContactID]; ok {
			v["referral"], v["referral_programs"] = rr.def, rr.byProgram
		}
		out[r.ContactID] = v
	}
	if containsRoot(roots, "reward") {
		for _, r := range recipients {
			if r.EmailID == uuid.Nil || r.ContactID == uuid.Nil {
				continue
			}
			rv, err := s.rewardVars(ctx, &ws, pages, r.EmailID, r.ContactID, now)
			if err != nil {
				return out, err
			}
			if rv != nil {
				out[r.ContactID]["reward"] = rv
			}
		}
	}
	return out, nil
}

func containsRoot(roots []string, r string) bool {
	for _, x := range roots {
		if x == r {
			return true
		}
	}
	return false
}

type programInfo struct {
	p          store.ReferralProgram
	list       string
	milestones []store.ReferralMilestone
}

// readerRefs are a reader's referral.* variables: their first program's
// (by creation) as referral, and every program's by id for Referral blocks
// bound to one.
type readerRefs struct {
	def       map[string]any
	byProgram map[string]any
}

func (s *Service) referralVars(ctx context.Context, ws *store.Workspace, pages interface {
	Join(string) string
	Referrals(string) string
}, contacts []uuid.UUID, now time.Time) (map[uuid.UUID]*readerRefs, error) {
	out := map[uuid.UUID]*readerRefs{}
	if len(contacts) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT rl.contact_id, rl.id, rl.code, rl.program_id
		FROM referral_links rl
		JOIN referral_programs p ON p.id = rl.program_id AND p.status = 'on' AND p.deleted_at IS NULL
		JOIN list_memberships lm ON lm.list_id = p.list_id AND lm.contact_id = rl.contact_id AND lm.status = 'subscribed'
		WHERE rl.contact_id = ANY ($1) AND rl.status = 'active' AND rl.workspace_id = $2
		ORDER BY rl.contact_id, p.created_at, p.id`, contacts, ws.ID)
	if err != nil {
		return nil, err
	}
	type linkRow struct {
		contact, link, program uuid.UUID
		code                   string
	}
	var links []linkRow
	var linkIDs []uuid.UUID
	for rows.Next() {
		var l linkRow
		if err := rows.Scan(&l.contact, &l.link, &l.code, &l.program); err != nil {
			rows.Close()
			return nil, err
		}
		links = append(links, l)
		linkIDs = append(linkIDs, l.link)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return out, nil
	}
	counts := map[uuid.UUID][4]int64{}
	crow, err := s.Pool.Query(ctx, `SELECT link_id, count(*) FILTER (WHERE status = 'counted'), count(*) FILTER (WHERE status = 'hold'),
		count(*) FILTER (WHERE status = 'review'), count(*) FILTER (WHERE status = 'awaiting')
		FROM referrals WHERE link_id = ANY ($1) GROUP BY 1`, linkIDs)
	if err != nil {
		return nil, err
	}
	for crow.Next() {
		var id uuid.UUID
		var c [4]int64
		if err := crow.Scan(&id, &c[0], &c[1], &c[2], &c[3]); err != nil {
			crow.Close()
			return nil, err
		}
		counts[id] = c
	}
	crow.Close()
	progs := map[uuid.UUID]*programInfo{}
	byLinkProgram := map[uuid.UUID]map[string]any{}
	for _, l := range links {
		pi := progs[l.program]
		if pi == nil {
			p, err := s.Q.ReferralProgramByID(ctx, l.program)
			if err != nil {
				return nil, err
			}
			ms, err := s.Q.ReferralMilestones(ctx, l.program)
			if err != nil {
				return nil, err
			}
			var list string
			_ = s.Pool.QueryRow(ctx, `SELECT name FROM lists WHERE id = $1`, p.ListID).Scan(&list)
			pi = &programInfo{p: p, list: list, milestones: ms}
			progs[l.program] = pi
		}
		c := counts[l.link]
		v := s.referralMap(pi, l.code, pages.Join(l.code), pages.Referrals(s.HubToken(ws.ID, l.contact, now)), c[0], c[1], c[1]+c[2]+c[3])
		byLinkProgram[l.link] = v
		rr := out[l.contact]
		if rr == nil {
			rr = &readerRefs{def: v, byProgram: map[string]any{}}
			out[l.contact] = rr
		}
		rr.byProgram[l.program.String()] = v
	}
	// Rewards earned, for referral.rewards (each program's own).
	rrows, err := s.Pool.Query(ctx, `SELECT link_id, reward_name, status FROM referral_rewards WHERE link_id = ANY ($1) ORDER BY milestone_count`, linkIDs)
	if err != nil {
		return nil, err
	}
	defer rrows.Close()
	for rrows.Next() {
		var link uuid.UUID
		var name, status string
		if err := rrows.Scan(&link, &name, &status); err != nil {
			return nil, err
		}
		if v, ok := byLinkProgram[link]; ok {
			list, _ := v["rewards"].([]map[string]any)
			v["rewards"] = append(list, map[string]any{"name": name, "status": status, "url": v["hub_url"]})
		}
	}
	return out, rrows.Err()
}

// referralMap is one reader's referral.* variables.
func (s *Service) referralMap(pi *programInfo, code, link, hub string, counted, soon, pending int64) map[string]any {
	display := strings.TrimPrefix(strings.TrimPrefix(link, "https://"), "http://")
	v := map[string]any{"show": true, "link": link, "code": code, "hub_url": hub, "display": display, "count": counted, "pending": pending,
		"soon": soon, "list": pi.list, "rewards": []map[string]any{}}
	share := map[string]any{}
	for _, sl := range ShareLinks(link, pi.p.ShareMessage, pi.list, []string{ViaEmail, ViaX, ViaLinkedIn, ViaWhatsApp}) {
		share[sl.Channel] = sl.URL
	}
	v["share"] = share
	var upcoming []map[string]any
	for _, m := range pi.milestones {
		if int64(m.Count) > counted {
			if _, ok := v["next_milestone"]; !ok {
				v["next_milestone"] = map[string]any{"count": m.Count, "reward": articled(m.RewardName), "name": m.RewardName}
				v["to_next"] = int64(m.Count) - counted
				cells := make([]string, 0, m.Count)
				for i := int64(1); i <= int64(m.Count); i++ {
					switch {
					case i <= counted:
						cells = append(cells, "#0F766E")
					case i <= counted+soon:
						cells = append(cells, "#99F6E4")
					default:
						cells = append(cells, "#E3E6EB")
					}
				}
				v["cells"] = cells
			}
			if len(upcoming) < 2 {
				upcoming = append(upcoming, map[string]any{"count": m.Count, "name": m.RewardName, "to_go": int64(m.Count) - counted})
			}
		}
	}
	if upcoming == nil {
		upcoming = []map[string]any{}
	}
	v["upcoming"] = upcoming
	return v
}

// rewardVars is reward.* for a milestone email: the reward the reader got
// at the email's milestone.
func (s *Service) rewardVars(ctx context.Context, ws *store.Workspace, pages interface{ Download(string) string }, emailID, contactID uuid.UUID, now time.Time) (map[string]any, error) {
	var mid uuid.UUID
	err := s.Pool.QueryRow(ctx, `SELECT id FROM referral_milestones WHERE email_id = $1`, emailID).Scan(&mid)
	if err != nil {
		return nil, nil //nolint:nilerr // not a milestone email
	}
	var rid uuid.UUID
	if err := s.Pool.QueryRow(ctx, `SELECT id FROM referral_rewards WHERE milestone_id = $1 AND contact_id = $2`, mid, contactID).Scan(&rid); err != nil {
		return nil, nil //nolint:nilerr // no reward (a test send)
	}
	rw, err := s.Q.ReferralRewardByID(ctx, rid)
	if err != nil {
		return nil, err
	}
	m, err := s.Q.ReferralMilestoneByID(ctx, mid)
	if err != nil {
		return nil, err
	}
	return s.rewardMap(ctx, ws, pages, m, rw, now), nil
}

func (s *Service) rewardMap(ctx context.Context, ws *store.Workspace, pages interface{ Download(string) string }, m store.ReferralMilestone, rw store.ReferralReward, now time.Time) map[string]any {
	v := map[string]any{"name": rw.RewardName, "description": m.RewardDescription, "note": rw.FulfillNote, "download_url": "", "code": "",
		"claim_url": rw.ClaimUrl, "expires_at": "", "url": "", "image_url": ""}
	if m.ImageID != nil {
		if a, err := s.Q.GetMediaAsset(ctx, store.GetMediaAssetParams{WorkspaceID: ws.ID, ID: *m.ImageID}); err == nil {
			v["image_url"] = s.mediaURL(a.ID, a.FileName)
		}
	}
	switch rw.RewardType {
	case RewardDownload:
		if m.FileID != nil {
			v["download_url"] = pages.Download(s.DownloadToken(ws.ID, rw.ID, rw.ContactID, now))
		} else {
			v["download_url"] = m.DownloadUrl
		}
		v["url"] = v["download_url"]
	case RewardCoupon:
		code, _ := s.rewardCode(ctx, rw)
		v["code"] = code
		v["url"] = rw.ClaimUrl
	default:
		code, u := s.rewardCode(ctx, rw)
		v["code"], v["url"] = code, u
	}
	if rw.ExpiresAt != nil {
		v["expires_at"] = rw.ExpiresAt.Format("2 Jan 2006")
	}
	return dropEmpty(v)
}

// dropEmpty leaves out a reward's empty fields: Liquid takes "" as true, so
// {% if reward.code %} would show an empty code. Printed, a missing field is
// still empty.
func dropEmpty(v map[string]any) map[string]any {
	for k, x := range v {
		if s, ok := x.(string); ok && s == "" {
			delete(v, k)
		}
	}
	return v
}

// SampleReferral is referral.* for a preview without a reader ("A reader
// with no referrals yet").
func (s *Service) SampleReferral(ctx context.Context, ws *store.Workspace, programID uuid.UUID, counted, soon int64) (map[string]any, error) {
	p, err := s.Q.ReferralProgramByID(ctx, programID)
	if err != nil {
		return nil, err
	}
	ms, err := s.Q.ReferralMilestones(ctx, programID)
	if err != nil {
		return nil, err
	}
	var list string
	_ = s.Pool.QueryRow(ctx, `SELECT name FROM lists WHERE id = $1`, p.ListID).Scan(&list)
	pages, err := s.pages(ctx, ws)
	if err != nil {
		return nil, err
	}
	return s.referralMap(&programInfo{p: p, list: list, milestones: ms}, "K7Q2MX", pages.Join("K7Q2MX"), pages.ReferralsRequest(), counted, soon, soon), nil
}

// mediaURL is a library image's absolute address (emails leave ReputeCast).
func (s *Service) mediaURL(id uuid.UUID, name string) string {
	base := ""
	if s.Base != nil {
		base = strings.TrimRight(s.Base(), "/")
	}
	return base + media.URL(id, name)
}
