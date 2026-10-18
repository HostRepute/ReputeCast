package recommendations

import (
	"context"
	"encoding/json"
	"html"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Recommendations block (brief §3.8, ComposerGrowBlocks frame 3): a
// card per recommendation with Subscribe (partners) or Visit (links). Its
// buttons open the hosted recommendations page, where the reader confirms
// with a button: link scanners open links but don't press buttons. Stored
// as <mj-raw><!-- rc:recommendations {…} --></mj-raw>; hidden when nothing
// can show.

func init() {
	liquid.RegisterBlock("recommendations", func(raw []byte) string { return ExpandRecommendations(parseBlock(raw)) })
}

// RecommendationsBlock is the block's settings.
type RecommendationsBlock struct {
	// Which: all that can show, or only the picked ones (IDs).
	Which   string   `json:"which" enum:"all,pick"`
	IDs     []string `json:"ids" maxItems:"20"`
	Layout  string   `json:"layout" enum:"stacked,columns,compact"`
	Heading string   `json:"heading" maxLength:"120"`
	Intro   string   `json:"intro" maxLength:"300"`
}

// DefaultBlock is a new Recommendations block.
func DefaultBlock() RecommendationsBlock {
	return RecommendationsBlock{Which: "all", IDs: []string{}, Layout: "stacked", Heading: "Newsletters we recommend", Intro: "Picked by the team."}
}

func parseBlock(raw []byte) RecommendationsBlock {
	b := DefaultBlock()
	if len(raw) == 0 {
		return b
	}
	var in RecommendationsBlock
	if json.Unmarshal(raw, &in) != nil {
		return b
	}
	if in.Which == "pick" {
		b.Which = "pick"
		for _, id := range in.IDs {
			if _, err := uuid.Parse(id); err == nil {
				b.IDs = append(b.IDs, id)
			}
		}
	}
	if in.Layout == "columns" || in.Layout == "compact" {
		b.Layout = in.Layout
	}
	if strings.TrimSpace(in.Heading) != "" {
		b.Heading = in.Heading
	}
	b.Intro = in.Intro
	return b
}

const (
	font   = `font-family:'IBM Plex Sans',Arial,sans-serif;`
	accent = "#0F766E"
)

// picksRe finds the blocks' picks in an expanded template.
var picksRe = regexp.MustCompile(`rc-recs:(all|[0-9a-f,\-]+)`)

// ExpandRecommendations is the block as Liquid reading recommendations.*.
func ExpandRecommendations(b RecommendationsBlock) string {
	var out strings.Builder
	pick := "all"
	if b.Which == "pick" {
		pick = strings.Join(b.IDs, ",")
		if pick == "" {
			pick = "none"
		}
	}
	// The picks, for the shown counts (Vars reads them back).
	out.WriteString(`{% comment %}rc-recs:` + pick + `{% endcomment %}`)
	out.WriteString(`{% assign rc_rec_ids = "` + pick + `" | split: "," %}{% assign rc_rec_n = 0 %}`)
	out.WriteString(`{% for r in recommendations.items %}{% if rc_rec_ids contains "all" or rc_rec_ids contains r.id %}{% assign rc_rec_n = rc_rec_n | plus: 1 %}{% endif %}{% endfor %}`)
	out.WriteString(`{% if rc_rec_n > 0 %}<div class="rc-recommendations" style="padding:16px 24px">`)
	out.WriteString(`<p style="margin:0 0 4px;` + font + `font-size:17px;font-weight:700;color:#0B1220">` + html.EscapeString(b.Heading) + `</p>`)
	if strings.TrimSpace(b.Intro) != "" {
		out.WriteString(`<p style="margin:0 0 12px;` + font + `font-size:13px;color:#5B6472">` + html.EscapeString(b.Intro) + `</p>`)
	}
	cols := b.Layout == "columns"
	if cols {
		out.WriteString(`<table role="presentation" style="width:100%;border-collapse:separate;border-spacing:8px 0;table-layout:fixed"><tr>`)
	}
	out.WriteString(`{% assign rc_rec_i = 0 %}{% for r in recommendations.items %}{% if rc_rec_ids contains "all" or rc_rec_ids contains r.id %}`)
	if cols {
		out.WriteString(`{% assign rc_rec_m = rc_rec_i | modulo: 2 %}{% if rc_rec_i > 0 and rc_rec_m == 0 %}</tr><tr>{% endif %}<td style="vertical-align:top;padding:0 0 8px">`)
	}
	switch b.Layout {
	case "compact":
		out.WriteString(`<p style="margin:0 0 8px;` + font + `font-size:14px;line-height:1.5;color:#0B1220"><b>{{ r.name }}</b> · {{ r.description }} ` +
			`<a href="{{ r.url }}" style="color:` + accent + `;font-weight:600">{{ r.button }}</a></p>`)
	default:
		out.WriteString(`<table role="presentation" style="width:100%;border-collapse:separate;border:1px solid #E3E6EB;border-radius:8px;margin:0 0 8px"><tr>`)
		out.WriteString(`<td style="width:44px;padding:14px 0 14px 14px;vertical-align:top">{% if r.logo_url %}<img src="{{ r.logo_url }}" width="36" height="36" alt="" style="display:block;width:36px;height:36px;border-radius:8px">` +
			`{% else %}<div style="width:36px;height:36px;border-radius:8px;background:#F0FDFA;color:` + accent + `;` + font + `font-size:13px;font-weight:700;line-height:36px;text-align:center">{{ r.initials }}</div>{% endif %}</td>`)
		out.WriteString(`<td style="padding:14px;vertical-align:top"><p style="margin:0;` + font + `font-size:14px;font-weight:700;color:#0B1220">{{ r.name }}</p>` +
			`<p style="margin:2px 0 10px;` + font + `font-size:13px;line-height:1.45;color:#374151">{{ r.description }}</p>` +
			`<a href="{{ r.url }}" style="display:inline-block;padding:7px 12px;border-radius:6px;background:` + accent + `;color:#fff;` + font + `font-size:13px;font-weight:600">{{ r.button }}</a></td>`)
		out.WriteString(`</tr></table>`)
	}
	if cols {
		out.WriteString(`</td>`)
	}
	out.WriteString(`{% assign rc_rec_i = rc_rec_i | plus: 1 %}{% endif %}{% endfor %}`)
	if cols {
		out.WriteString(`</tr></table>`)
	}
	out.WriteString(`<p style="margin:4px 0 0;` + font + `font-size:12px;color:#5B6472">Buttons open a page where you confirm. Nothing is sent until you press Subscribe there.</p>`)
	out.WriteString(`</div>{% endif %}`)
	return out.String()
}

// Roots implements liquid.RecipientData.
func (s *Service) Roots() []string { return []string{"recommendations"} }

// Vars implements liquid.RecipientData: recommendations.items, the
// recommendations the block can show now, each linking to the reader's
// recommendations page. A send records which ones each message showed:
// they count as shown once the message is delivered.
func (s *Service) Vars(ctx context.Context, wsID uuid.UUID, recipients []liquid.Recipient, _ []string) (map[uuid.UUID]map[string]any, error) {
	out := map[uuid.UUID]map[string]any{}
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return out, err
	}
	recs, err := s.showing(ctx, &ws, false)
	if err != nil {
		return out, err
	}
	pages, err := s.pages(ctx, &ws)
	if err != nil {
		return out, err
	}
	picked := map[string]bool{}
	all := false
	for _, m := range picksRe.FindAllStringSubmatch(liquid.SourceFrom(ctx), -1) {
		if m[1] == "all" {
			all = true
			continue
		}
		for _, id := range strings.Split(m[1], ",") {
			picked[id] = true
		}
	}
	now := s.now()
	type item struct {
		id   uuid.UUID
		data map[string]any
	}
	items := make([]item, 0, len(recs))
	for _, r := range recs {
		pr := s.publicRec(ctx, &ws, r, pages.RecommendsVisit)
		button := "Visit"
		if r.Kind == KindPartner {
			button = "Subscribe"
		}
		items = append(items, item{id: r.ID, data: map[string]any{"id": r.ID.String(), "name": r.Name, "description": r.Description, "kind": r.Kind,
			"initials": pr.Initials, "logo_url": pr.LogoURL, "host": pr.Host, "button": button}})
	}
	for _, rc := range recipients {
		page := pages.Recommends("")
		if rc.ContactID != uuid.Nil {
			page = pages.Recommends(s.PageToken(ws.ID, rc.ContactID, now))
		}
		list := make([]map[string]any, 0, len(items))
		var shown []uuid.UUID
		for _, it := range items {
			d := make(map[string]any, len(it.data)+1)
			for k, v := range it.data {
				d[k] = v
			}
			d["url"] = page + "#r-" + it.id.String()
			list = append(list, d)
			if all || picked[it.id.String()] {
				shown = append(shown, it.id)
			}
		}
		out[rc.ContactID] = map[string]any{"recommendations": map[string]any{"show": len(list) > 0, "url": page, "items": list}}
		if rc.MessageID != uuid.Nil && len(shown) > 0 {
			if _, err := s.Pool.Exec(ctx, `INSERT INTO recommendation_message_shown (message_id, workspace_id, recommendation_ids, created_at)
				VALUES ($1, $2, $3, $4) ON CONFLICT (message_id) DO NOTHING`, rc.MessageID, ws.ID, shown, now); err != nil {
				s.log().WarnContext(ctx, "recommendations: record shown", "err", err)
			}
		}
	}
	return out, nil
}

// MessageEvent counts a delivered message's recommendations as shown in
// emails (an ingest message hook).
func (s *Service) MessageEvent(ctx context.Context, tx pgx.Tx, _ store.Provider, messageID uuid.UUID, _ string, ev providers.Event) error {
	if ev.Type != providers.EventDelivered {
		return nil
	}
	var ws uuid.UUID
	var ids []uuid.UUID
	err := tx.QueryRow(ctx, `DELETE FROM recommendation_message_shown WHERE message_id = $1 RETURNING workspace_id, recommendation_ids`, messageID).Scan(&ws, &ids)
	if err != nil {
		return nil //nolint:nilerr // no recommendations in this message
	}
	w, err := store.New(tx).GetWorkspace(ctx, ws)
	if err != nil {
		return nil //nolint:nilerr // the workspace is gone
	}
	day := dayOf(s.now(), &w)
	q := store.New(tx)
	for _, id := range ids {
		if err := q.RecDailyBump(ctx, store.RecDailyBumpParams{RecommendationID: id, Day: day, ShownEmail: 1}); err != nil {
			return err
		}
	}
	return nil
}

// BlockRender is the Recommendations block for the composer's canvas.
type BlockRender struct {
	HTML string `json:"html"`
	// Shows is how many recommendations it shows; Hidden lists the
	// picked or showing ones that can't show now, with why.
	Shows  int              `json:"shows"`
	Hidden []BlockHiddenRec `json:"hidden"`
	// Settings are the block's effective settings, for the inspector.
	Settings RecommendationsBlock `json:"settings"`
}

// BlockHiddenRec is a recommendation the block can't show now.
type BlockHiddenRec struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Initials string `json:"initials"`
	// State: paused, partner_paused, invalid, limit, ended, or not_in_emails
	// (it shows on the confirmation page only).
	State    string `json:"state"`
	PausedBy string `json:"pausedBy"`
}

// BlockRenderInput renders the block for the canvas.
type BlockRenderInput struct {
	Settings json.RawMessage `json:"settings,omitempty"`
}

// RenderBlock renders the block for the canvas with what can show now.
func (s *Service) RenderBlock(ctx context.Context, ws *store.Workspace, in BlockRenderInput) (BlockRender, error) {
	b := parseBlock(in.Settings)
	src := ExpandRecommendations(b)
	vars, err := s.Vars(ctx, ws.ID, []liquid.Recipient{{}}, nil)
	if err != nil {
		return BlockRender{}, err
	}
	v := liquid.Merge(liquid.Vars("", "", "", nil, nil), vars[uuid.Nil])
	out := BlockRender{Hidden: []BlockHiddenRec{}, Settings: b}
	if items, ok := v["recommendations"].(map[string]any)["items"].([]map[string]any); ok {
		for _, it := range items {
			if b.Which == "all" || containsStr(b.IDs, it["id"].(string)) {
				out.Shows++
			}
		}
	}
	if out.HTML, err = liquid.Render(src, v); err != nil {
		return out, err
	}
	items, err := s.items(ctx, ws, s.now())
	if err != nil {
		return out, err
	}
	for _, it := range items {
		if b.Which == "pick" && !containsStr(b.IDs, it.ID) {
			continue
		}
		state := ""
		switch {
		case it.Hidden:
			state = it.State
		case !it.ShowBlock:
			state = "not_in_emails"
		}
		if state != "" {
			out.Hidden = append(out.Hidden, BlockHiddenRec{ID: it.ID, Name: it.Name, Initials: it.Initials, State: state, PausedBy: it.PausedBy})
		}
	}
	return out, nil
}

func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
