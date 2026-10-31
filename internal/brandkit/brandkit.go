// Package brandkit is the workspace's brand kit (review 2026-10, CO-11): the
// logo in light and dark versions, the colours with their contrast where
// they are used, the heading and body fonts with email-safe fallbacks, and
// the button style that new emails, templates and blocks start from. It can
// be filled in from the workspace's website (import.go). Emails already
// written keep their own styles.
//
// Synced blocks (synced.go) are the Header and Footer every email can hold,
// and any section saved as a named synced block (R4.A3): edit once, and the
// drafts and scheduled emails holding the block get the change; sent emails
// keep what they went out with.
package brandkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
)

// Campaigns is what synced blocks need from the campaign domain
// (internal/campaigns).
type Campaigns interface {
	CountSynced(ctx context.Context, ws *store.Workspace, kind string) (campaigns.SyncedUses, error)
	ApplySynced(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind, block, label string) (campaigns.SyncedUses, []uuid.UUID, error)
	UnlinkSynced(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind, label string) (campaigns.SyncedUses, []uuid.UUID, error)
}

// Service is the brand kit domain.
type Service struct {
	Pool      *pgxpool.Pool
	Q         *store.Queries
	Audit     *audit.Logger
	Campaigns Campaigns
	// Media stores the logos an import finds.
	Media *media.Service
	// Guard checks the addresses an import fetches.
	Guard netguard.Policy
	Now   func() time.Time

	clientOnce sync.Once
	httpClient *http.Client
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// BrandColors are the kit's colours, #RRGGBB.
type BrandColors struct {
	Button     string `json:"button" pattern:"^#[0-9A-Fa-f]{6}$" doc:"Button fill, with white text"`
	Text       string `json:"text" pattern:"^#[0-9A-Fa-f]{6}$" doc:"Paragraphs and headings"`
	Background string `json:"background" pattern:"^#[0-9A-Fa-f]{6}$" doc:"Behind the 600 px email column"`
	Link       string `json:"link" pattern:"^#[0-9A-Fa-f]{6}$" doc:"Links inside text, underlined"`
}

// Fallbacks are the email-safe font stacks a font can fall back to: Gmail
// and Outlook for Windows show these, whatever the web font.
var Fallbacks = []string{
	"Georgia, 'Times New Roman', serif",
	"'Times New Roman', serif",
	"Arial, Helvetica, sans-serif",
	"Helvetica, Arial, sans-serif",
	"Verdana, Geneva, sans-serif",
	"'Trebuchet MS', Arial, sans-serif",
	"Tahoma, Geneva, sans-serif",
}

// BrandFont is a heading or body font: a web font (Apple Mail and iOS Mail load
// it) and the email-safe fallback everyone else sees.
type BrandFont struct {
	Name     string `json:"name" maxLength:"60" pattern:"^[A-Za-z0-9 ]*$" doc:"The web font's family (Inter); empty uses the fallback alone"`
	URL      string `json:"url,omitempty" maxLength:"500" doc:"The web font's stylesheet (https)"`
	Fallback string `json:"fallback" maxLength:"60" doc:"One of the email-safe stacks: Georgia, 'Times New Roman', serif · 'Times New Roman', serif · Arial, Helvetica, sans-serif · Helvetica, Arial, sans-serif · Verdana, Geneva, sans-serif · 'Trebuchet MS', Arial, sans-serif · Tahoma, Geneva, sans-serif"`
}

// Stack is the font-family value emails use: the web font first, then the
// fallback.
func (f BrandFont) Stack() string {
	if strings.TrimSpace(f.Name) == "" {
		return f.Fallback
	}
	return "'" + strings.TrimSpace(f.Name) + "', " + f.Fallback
}

// BrandButtonStyle is how buttons look.
type BrandButtonStyle struct {
	Shape string `json:"shape" enum:"square,rounded,pill"`
	Size  int    `json:"size" enum:"14,15" doc:"Text size in px (bold, body font)"`
}

// Radius is the shape's border-radius.
func (b BrandButtonStyle) Radius() string {
	switch b.Shape {
	case "square":
		return "0px"
	case "pill":
		return "999px"
	}
	return "6px"
}

// BrandStyle is everything but the logo.
type BrandStyle struct {
	Colors  BrandColors      `json:"colors"`
	Heading BrandFont        `json:"heading"`
	Body    BrandFont        `json:"body"`
	Button  BrandButtonStyle `json:"button"`
}

// DefaultStyle is ReputeCast's own: the starters' and blocks' colours and
// fonts before a kit is saved.
func DefaultStyle() BrandStyle {
	return BrandStyle{
		Colors:  BrandColors{Button: "#0F766E", Text: "#1F2937", Background: "#F3F4F6", Link: "#0F766E"},
		Heading: BrandFont{Fallback: "Georgia, 'Times New Roman', serif"},
		Body:    BrandFont{Fallback: "Arial, Helvetica, sans-serif"},
		Button:  BrandButtonStyle{Shape: "rounded", Size: 14},
	}
}

// BrandLogo is a logo image from the media library.
type BrandLogo struct {
	MediaID  string `json:"mediaId"`
	URL      string `json:"url"`
	FileName string `json:"fileName"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// BrandKit is the workspace's kit.
type BrandKit struct {
	// Saved is false until the kit is first saved: the values are then
	// ReputeCast's defaults.
	Saved     bool       `json:"saved"`
	LogoLight *BrandLogo `json:"logoLight,omitempty" doc:"For light backgrounds"`
	LogoDark  *BrandLogo `json:"logoDark,omitempty" doc:"For dark mode (Apple Mail and Outlook for Mac switch to it)"`
	LogoAlt   string     `json:"logoAlt"`
	BrandStyle
	Website   string     `json:"website"`
	UpdatedBy string     `json:"updatedBy,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	// Synced are the synced Header and Footer, with the emails holding them.
	Synced []SyncedBlock `json:"synced"`
}

// BrandKitInput saves the kit.
type BrandKitInput struct {
	LogoLightID string `json:"logoLightId,omitempty" doc:"A media library image id; empty: no logo"`
	LogoDarkID  string `json:"logoDarkId,omitempty"`
	LogoAlt     string `json:"logoAlt" maxLength:"120"`
	BrandStyle
	Website string `json:"website,omitempty" maxLength:"200"`
}

var hexColor = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func (s *Service) validate(ctx context.Context, in *BrandKitInput) error {
	in.LogoAlt = strings.TrimSpace(in.LogoAlt)
	in.Website = strings.TrimSpace(in.Website)
	for field, c := range map[string]*string{"button": &in.Colors.Button, "text": &in.Colors.Text, "background": &in.Colors.Background, "link": &in.Colors.Link} {
		if !hexColor.MatchString(*c) {
			return api.Invalid("body.colors."+field, say(ctx, mBadColor))
		}
		*c = strings.ToUpper(*c)
	}
	for field, f := range map[string]*BrandFont{"heading": &in.Heading, "body": &in.Body} {
		f.Name = strings.Join(strings.Fields(f.Name), " ")
		if !slices.Contains(Fallbacks, f.Fallback) {
			return api.Invalid("body."+field+".fallback", say(ctx, mBadFallback))
		}
		f.URL = strings.TrimSpace(f.URL)
		if f.URL != "" {
			if f.Name == "" || !strings.HasPrefix(f.URL, "https://") || strings.ContainsAny(f.URL, `"'<> `) {
				return api.Invalid("body."+field+".url", say(ctx, mBadFontURL))
			}
		}
	}
	if in.Button.Shape == "" {
		in.Button.Shape = "rounded"
	}
	if in.Button.Size == 0 {
		in.Button.Size = 14
	}
	return nil
}

// logo reads a media library image of the workspace.
func (s *Service) logo(ctx context.Context, q *store.Queries, ws *store.Workspace, id *uuid.UUID) (*BrandLogo, error) {
	if id == nil {
		return nil, nil
	}
	a, err := q.GetMediaAsset(ctx, store.GetMediaAssetParams{WorkspaceID: ws.ID, ID: *id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &BrandLogo{MediaID: a.ID.String(), URL: media.URL(a.ID, a.FileName), FileName: a.FileName, Width: int(a.Width), Height: int(a.Height)}, nil
}

// Kit returns the workspace's kit, ReputeCast's defaults when none is saved.
func (s *Service) Kit(ctx context.Context, ws *store.Workspace) (BrandKit, error) {
	return s.kit(ctx, s.Q, ws)
}

func (s *Service) kit(ctx context.Context, q *store.Queries, ws *store.Workspace) (BrandKit, error) {
	k := BrandKit{BrandStyle: DefaultStyle(), Synced: []SyncedBlock{}}
	r, err := q.BrandKitGet(ctx, ws.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return k, err
	}
	if err == nil {
		k.Saved = true
		_ = json.Unmarshal(r.Style, &k.BrandStyle)
		k.LogoAlt, k.Website, k.UpdatedBy = r.LogoAlt, r.Website, r.UpdatedByName
		at := r.UpdatedAt
		k.UpdatedAt = &at
		if k.LogoLight, err = s.logo(ctx, q, ws, r.LogoLightID); err != nil {
			return k, err
		}
		if k.LogoDark, err = s.logo(ctx, q, ws, r.LogoDarkID); err != nil {
			return k, err
		}
	}
	return k, nil
}

// Get returns the kit with its synced blocks and the emails holding them.
func (s *Service) Get(ctx context.Context, ws *store.Workspace) (BrandKit, error) {
	k, err := s.kit(ctx, s.Q, ws)
	if err != nil {
		return k, err
	}
	k.Synced, err = s.syncedBlocks(ctx, ws)
	return k, err
}

func parseID(raw string) (*uuid.UUID, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, false
	}
	return &id, true
}

// Save stores the kit. A new light logo or alt text reaches the synced
// Header, and through it the drafts and scheduled emails holding it when
// the caller can edit campaigns.
func (s *Service) Save(ctx context.Context, ws *store.Workspace, p *authz.Principal, in BrandKitInput) (BrandKit, error) {
	var saved KitSaved
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if saved, err = s.SaveIn(ctx, q, ws, p, in); err != nil {
			return err
		}
		for _, e := range saved.Audit {
			if err := s.Audit.LogWith(ctx, q, e); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return BrandKit{}, err
	}
	// The Header's new logo reaches the emails holding it (after the kit is
	// saved), as a change to the synced block would: the kit needs
	// templates:edit, changing those emails campaigns:edit. Without it they
	// keep their Header until someone who can edit them changes it.
	if saved.Header != nil && p.Can(authz.CampaignsEdit) {
		if _, err := s.propagate(ctx, ws, p, *saved.Header); err != nil {
			return BrandKit{}, err
		}
	}
	return s.Get(ctx, ws)
}

// KitSaved is what saving the kit did.
type KitSaved struct {
	Before, After BrandKit
	// Header is the synced Header when the kit's logo changed it, for the
	// emails holding it once the kit's transaction commits.
	Header *SyncedBlock
	// Audit are the entries the save makes, for the caller to write.
	Audit []audit.Event
}

// SaveIn is Save's work in the caller's transaction (q): it checks and
// stores the kit, creates the synced Header and Footer on the first save,
// and puts a new light logo or alt text in the Header. It leaves the
// audit entries and the emails holding the Header to the caller (Save;
// internal/demo seeds the Acme kit through it).
func (s *Service) SaveIn(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, in BrandKitInput) (KitSaved, error) {
	var out KitSaved
	if err := s.validate(ctx, &in); err != nil {
		return out, err
	}
	light, ok := parseID(in.LogoLightID)
	if !ok {
		return out, api.Invalid("body.logoLightId", say(ctx, mBadLogo))
	}
	dark, ok := parseID(in.LogoDarkID)
	if !ok {
		return out, api.Invalid("body.logoDarkId", say(ctx, mBadLogo))
	}
	var err error
	if out.Before, err = s.kit(ctx, q, ws); err != nil {
		return out, err
	}
	for field, id := range map[string]*uuid.UUID{"logoLightId": light, "logoDarkId": dark} {
		if id == nil {
			continue
		}
		if l, err := s.logo(ctx, q, ws, id); err != nil {
			return out, err
		} else if l == nil {
			return out, api.Invalid("body."+field, say(ctx, mBadLogo))
		}
	}
	style, _ := json.Marshal(in.BrandStyle)
	if err := q.BrandKitUpsert(ctx, store.BrandKitUpsertParams{WorkspaceID: ws.ID, LogoLightID: light, LogoDarkID: dark, LogoAlt: in.LogoAlt, Style: style,
		Website: in.Website, UpdatedBy: actor(p), UpdatedAt: s.now()}); err != nil {
		return out, err
	}
	if out.After, err = s.kit(ctx, q, ws); err != nil {
		return out, err
	}
	// First save: the synced Header and Footer start from the kit.
	if err := s.ensureSynced(ctx, q, ws, p, out.After); err != nil {
		return out, err
	}
	if logoURL(out.Before.LogoLight) != logoURL(out.After.LogoLight) || out.Before.LogoAlt != out.After.LogoAlt {
		var e *audit.Event
		if out.Header, e, err = s.relogo(ctx, q, ws, p, out.After); err != nil {
			return out, err
		}
		if e != nil {
			out.Audit = append(out.Audit, *e)
		}
	}
	out.Audit = append(out.Audit, audit.Event{WorkspaceID: &ws.ID, Action: "brand_kit.save", TargetType: "brand_kit", TargetID: ws.ID.String(), TargetName: ws.Name,
		Before: kitAudit(out.Before), After: kitAudit(out.After)})
	return out, nil
}

func logoURL(l *BrandLogo) string {
	if l == nil {
		return ""
	}
	return l.URL
}

func kitAudit(k BrandKit) map[string]any {
	if !k.Saved {
		return map[string]any{"saved": false}
	}
	return map[string]any{"colors": k.Colors, "heading": k.Heading.Stack(), "body": k.Body.Stack(), "button": k.Button,
		"logoLight": logoURL(k.LogoLight), "logoDark": logoURL(k.LogoDark), "logoAlt": k.LogoAlt, "website": k.Website}
}

func actor(p *authz.Principal) *uuid.UUID {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// ── Contrast ────────────────────────────────────────────────────────────

func channel(c uint8) float64 {
	v := float64(c) / 255
	if v <= 0.03928 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// luminance of #RRGGBB (WCAG 2).
func luminance(hex string) float64 {
	n, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	if err != nil || len(hex) != 7 {
		return 0
	}
	r, g, b := uint8(n>>16), uint8(n>>8), uint8(n)
	return 0.2126*channel(r) + 0.7152*channel(g) + 0.0722*channel(b)
}

// Contrast is the WCAG contrast ratio of two colours, 1 to 21.
func Contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// round2 is a ratio with two decimals (4.47), never rounded up to a
// threshold it misses (4.499 reads 4.49, not 4.50).
func round2(v float64) float64 {
	r := math.Round(v*100) / 100
	for _, t := range []float64{3, 4.5, 7} {
		if v < t && r >= t {
			return math.Floor(v*100) / 100
		}
	}
	return r
}

// ── Applying the kit ────────────────────────────────────────────────────

var (
	buttonTag    = regexp.MustCompile(`<mj-button\b[^>]*>`)
	tagBg        = regexp.MustCompile(`(?i)background-color="#0F766E"`)
	tagRadius    = regexp.MustCompile(`border-radius="[^"]*"`)
	tagSize      = regexp.MustCompile(`font-size="1[34]px"`)
	tagPadding   = regexp.MustCompile(`inner-padding="[^"]*"`)
	linkColor    = regexp.MustCompile(`(?i)color:\s*#0F766E`)
	textColor    = regexp.MustCompile(`(?i)#1F2937|#0B1220`)
	bgColor      = regexp.MustCompile(`(?i)background-color="#F3F4F6"`)
	plexStack    = "'IBM Plex Sans', Helvetica, Arial, sans-serif"
	georgiaStack = "Georgia, 'Times New Roman', serif"
	starterLogo  = regexp.MustCompile(`(<mj-image\b[^>]*?)src="/media/starters/logo\.png"([^>]*?)alt="[^"]*"`)
)

// Apply puts the kit on content written with ReputeCast's default style (a
// starter template or a new block): its colours, fonts, button style and
// logo replace the defaults. Content written by people is left as it is;
// only what still has a default changes.
func Apply(src string, k BrandKit) string {
	if !k.Saved {
		return src
	}
	st := k.BrandStyle
	src = buttonTag.ReplaceAllStringFunc(src, func(tag string) string {
		tag = tagBg.ReplaceAllString(tag, `background-color="`+st.Colors.Button+`"`)
		tag = tagRadius.ReplaceAllString(tag, `border-radius="`+st.Button.Radius()+`"`)
		tag = tagSize.ReplaceAllString(tag, `font-size="`+strconv.Itoa(st.Button.Size)+`px"`)
		// 44 px tall, easy to tap (BrandKit · Button style).
		return tagPadding.ReplaceAllString(tag, `inner-padding="14px 22px"`)
	})
	src = linkColor.ReplaceAllString(src, "color:"+st.Colors.Link)
	src = textColor.ReplaceAllString(src, st.Colors.Text)
	src = bgColor.ReplaceAllString(src, `background-color="`+st.Colors.Background+`"`)
	src = strings.ReplaceAll(src, plexStack, st.Body.Stack())
	src = strings.ReplaceAll(src, georgiaStack, st.Body.Stack())
	src = headings(src, st.Heading.Stack())
	if k.LogoLight != nil {
		alt := escapeAttr(k.LogoAlt)
		src = starterLogo.ReplaceAllString(src, `${1}src="`+k.LogoLight.URL+`"${2}alt="`+alt+`"`)
	}
	return withFonts(src, st)
}

// headings gives the text blocks holding a heading the heading font.
func headings(src, stack string) string {
	if !strings.HasPrefix(strings.TrimSpace(src), "<mjml") {
		return mjml.UnwrapFragment(headings(mjml.WrapFragment(src), stack))
	}
	root, err := mjml.Parse(src)
	if err != nil {
		return src
	}
	var nodes []*mjml.Node
	root.Walk(func(n *mjml.Node, _ []int) {
		if n.Tag == "mj-text" && (strings.Contains(n.Inner, "<h1") || strings.Contains(n.Inner, "<h2")) {
			nodes = append(nodes, n)
		}
	})
	// From the end, so earlier offsets stay valid.
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Start > nodes[j].Start })
	for _, n := range nodes {
		src, _ = mjml.SetAttrRaw(src, n, "font-family", stack)
	}
	return src
}

// withFonts declares the web fonts in the head (mj-font).
func withFonts(src string, st BrandStyle) string {
	var fonts strings.Builder
	seen := map[string]bool{}
	for _, f := range []BrandFont{st.Heading, st.Body} {
		if f.Name == "" || f.URL == "" || seen[f.Name] {
			continue
		}
		seen[f.Name] = true
		fmt.Fprintf(&fonts, `<mj-font name="%s" href="%s"></mj-font>`, escapeAttr(f.Name), escapeAttr(f.URL))
	}
	if fonts.Len() == 0 {
		return src
	}
	if i := strings.Index(src, "<mj-head>"); i >= 0 {
		return src[:i+len("<mj-head>")] + fonts.String() + src[i+len("<mj-head>"):]
	}
	if i := strings.Index(src, "<mjml>"); i >= 0 {
		return src[:i+len("<mjml>")] + "<mj-head>" + fonts.String() + "</mj-head>" + src[i+len("<mjml>"):]
	}
	return src
}

func escapeAttr(s string) string {
	return strings.NewReplacer("&", "&amp;", `"`, "&quot;", "<", "&lt;", ">", "&gt;").Replace(s)
}
