package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// ReadinessFixRequest is one fix to apply.
type ReadinessFixRequest struct {
	Action string            `json:"action" enum:"exclude_sunset,route_only,insert_footer,replace_link,use_subject,minify,mark_language_up_to_date,mark_language_translated,hold_language,release_language,accessibility_decorative,accessibility_color,accessibility_one_click,visibility_show_everyone,visibility_hide_block,visibility_remove_condition,poll_close_days,exclude_risky,verify_addresses,sponsor_remove_block,sponsor_refresh_block"`
	Params map[string]string `json:"params,omitempty"`
}

// ReadinessFixesInput applies one or more fixes ("Apply all 3 fixes").
type ReadinessFixesInput struct {
	Fixes []ReadinessFixRequest `json:"fixes" minItems:"1" maxItems:"10"`
}

// ReadinessFixResult says what changed.
type ReadinessFixResult struct {
	// Applied are the fix labels in the product voice ("Excluded 412 unengaged contacts").
	Applied []string `json:"applied"`
	// Version is the campaign version after content fixes.
	Version int `json:"version"`
}

// AfterChange is told when a fix or a send setting changed the campaign
// (the approvals domain re-checks open approvals).
type AfterChange func(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID)

// footerMarker marks the composer's footer section in the MJML.
const footerMarker = "<!-- rc:footer -->"

// footerMJML is the composer's footer block (blocks.ts footerBlock).
func footerMJML(address string) string {
	addr := address
	if strings.TrimSpace(addr) == "" {
		addr = "Your company · Street and number · City, postcode"
	}
	addr = html.EscapeString(addr)
	const plex = `'IBM Plex Sans', Helvetica, Arial, sans-serif`
	const teal = "#0F766E"
	return footerMarker + `<mj-section background-color="#F3F4F6" padding="18px 24px"><mj-column><mj-text align="center" font-family="` + plex +
		`" font-size="12px" line-height="1.6" color="#5B6472" padding="0px">` + addr + `<br><a href="{{ unsubscribe_url }}" style="color:` + teal +
		`;text-decoration:none">Unsubscribe</a> · <a href="{{ preferences_url }}" style="color:` + teal + `;text-decoration:none">Email preferences</a> · <a href="{{ view_in_browser_url }}" style="color:` +
		teal + `;text-decoration:none">View in browser</a></mj-text></mj-column></mj-section>`
}

// InsertFooter adds the footer block at the end of the MJML body, or puts
// the composer's footer block back as new when it lost its address or links.
func InsertFooter(src, address string) (string, bool) {
	if i := strings.Index(src, footerMarker); i >= 0 {
		rest := src[i+len(footerMarker):]
		start, end := strings.Index(rest, "<mj-section"), strings.Index(rest, "</mj-section>")
		if start >= 0 && end > start && strings.TrimSpace(rest[:start]) == "" {
			return src[:i] + footerMJML(address) + rest[end+len("</mj-section>"):], true
		}
	}
	i := strings.LastIndex(strings.ToLower(src), "</mj-body>")
	if i < 0 {
		return src, false
	}
	return src[:i] + footerMJML(address) + "\n  " + src[i:], true
}

// ReplaceLink points every link to from at to.
func ReplaceLink(src, from, to string) (string, int) {
	n := 0
	for _, f := range []string{from, html.EscapeString(from)} {
		for _, q := range []string{`"`, `'`} {
			old, repl := "href="+q+f+q, "href="+q+to+q
			n += strings.Count(src, old)
			src = strings.ReplaceAll(src, old, repl)
		}
	}
	return src, n
}

func validLink(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
}

// ApplyFixes applies one-click fixes to a campaign. Content fixes save a new
// version through the composer; audience and route fixes change the
// campaign. Each change is audited.
func (s *Service) ApplyFixes(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ReadinessFixesInput) (ReadinessFixResult, error) {
	out := ReadinessFixResult{Applied: []string{}}
	c, err := s.Campaign(ctx, ws, id)
	if err != nil {
		return out, err
	}
	if !editable(c.Status) {
		return out, api.Conflict("This campaign already started sending, so it can't change.")
	}
	// Content fixes are applied together as one new version.
	src, subject, content := c.Mjml, c.Subject, false
	labels := []string{}
	var a11y []accessibility.FixChange
	var a11yFixes []ReadinessFixRequest
	for i, fx := range in.Fixes {
		field := fmt.Sprintf("body.fixes[%d]", i)
		switch fx.Action {
		case FixExcludeSunset:
			label, err := s.excludeSunset(ctx, ws, p, c)
			if err != nil {
				return out, err
			}
			out.Applied = append(out.Applied, label)
		case FixExcludeRisky:
			label, err := s.excludeRisky(ctx, ws, p, c)
			if err != nil {
				return out, err
			}
			out.Applied = append(out.Applied, label)
		case FixVerifyAddresses:
			// The Verify addresses dialog starts the job (POST …/verification/jobs).
			return out, api.Invalid(field, "Verifying addresses starts from the Verify addresses dialog on Review & send.")
		case FixRouteOnly:
			label, err := s.routeOnly(ctx, ws, p, c, fx.Params["providerId"], field)
			if err != nil {
				return out, err
			}
			out.Applied = append(out.Applied, label)
		case FixInsertFooter:
			if strings.TrimSpace(src) == "" {
				return out, api.Invalid(field, "This campaign is written in HTML. Add the footer with the address and unsubscribe link in the editor.")
			}
			next, ok := InsertFooter(src, templates.WorkspaceAddress(ws))
			if !ok {
				return out, api.Invalid(field, "The MJML has no body to add the footer to. Open the campaign in the editor.")
			}
			src, content = next, true
			labels = append(labels, "footer block inserted")
			out.Applied = append(out.Applied, "Inserted the footer block")
		case FixReplaceLink:
			from, to := strings.TrimSpace(fx.Params["from"]), strings.TrimSpace(fx.Params["to"])
			if from == "" || !validLink(to) {
				return out, api.Invalid(field, "Give the link to replace and an http or https address to point it at.")
			}
			next, n := ReplaceLink(src, from, to)
			if n == 0 {
				return out, api.Invalid(field, "That link isn't in the email any more. Run the checks again.")
			}
			src, content = next, true
			labels = append(labels, "link replaced")
			out.Applied = append(out.Applied, "Replaced "+displayURL(from)+" with "+displayURL(to))
		case FixUseSubject:
			next := strings.TrimSpace(fx.Params["subject"])
			if next == "" || len(next) > 998 {
				return out, api.Invalid(field, "Write a subject of up to 998 characters.")
			}
			subject, content = next, true
			labels = append(labels, "subject")
			out.Applied = append(out.Applied, "Changed the subject to “"+next+"”")
		case FixMinify:
			var settings campaigns.ComposerContentSettings
			_ = json.Unmarshal(c.ContentSettings, &settings)
			if !settings.Minify {
				settings.Minify = true
				b, _ := json.Marshal(settings)
				if err := s.Q.ReadinessSetContentSettings(ctx, store.ReadinessSetContentSettingsParams{ContentSettings: b, WorkspaceID: ws.ID, ID: id}); err != nil {
					return out, err
				}
			}
			content = true
			labels = append(labels, "minified")
			out.Applied = append(out.Applied, "Minified the inline CSS")
		case FixMarkLanguageUpToDate, FixMarkLanguageTranslated, FixHoldLanguage, FixReleaseLanguage:
			label, err := s.languageFix(ctx, ws, p, id, fx.Action, fx.Params["language"], field)
			if err != nil {
				return out, err
			}
			out.Applied = append(out.Applied, label)
		case FixAccessibilityDecorative, FixAccessibilityColor, FixAccessibilityOneClick:
			if s.Accessibility == nil {
				return out, api.Conflict("Accessibility fixes aren't available in this process.")
			}
			next, change, err := s.Accessibility.FixMJML(ctx, ws, src, fx.Action, fx.Params)
			if err != nil {
				return out, api.Invalid(field, sentence(capitalize(err.Error())))
			}
			src, content = next, true
			labels = append(labels, change.Label)
			out.Applied = append(out.Applied, change.Applied)
			a11y = append(a11y, change)
			a11yFixes = append(a11yFixes, fx)
		case FixVisibilityShowEveryone, FixVisibilityHideBlock, FixVisibilityRemoveCondition, FixPollCloseDays:
			// Rules and polls are the source's: language versions follow it.
			if strings.TrimSpace(src) == "" {
				return out, api.Invalid(field, "This campaign is written in HTML. Change it in the editor.")
			}
			next, label, applied, err := conditionalFix(src, fx.Action, fx.Params)
			if err != nil {
				return out, api.Invalid(field, sentence(capitalize(err.Error())))
			}
			src, content = next, true
			labels = append(labels, label)
			out.Applied = append(out.Applied, applied)
		case FixSponsorRemove, FixSponsorRefresh:
			if s.Sponsor == nil {
				return out, api.Conflict("Sponsor block fixes aren't available in this process.")
			}
			if strings.TrimSpace(src) == "" {
				return out, api.Invalid(field, "This campaign is written in HTML. Change it in the editor.")
			}
			next, applied, err := s.Sponsor.FixSponsor(ctx, ws, id, src, fx.Action)
			if err != nil {
				return out, api.Invalid(field, sentence(capitalize(err.Error())))
			}
			src, content = next, true
			if fx.Action == FixSponsorRemove {
				labels = append(labels, "Sponsor block removed")
			} else {
				labels = append(labels, "Sponsor block updated")
			}
			out.Applied = append(out.Applied, applied)
		default:
			return out, api.Invalid(field+".action", "Unknown fix.")
		}
	}
	out.Version = int(c.Version)
	if content {
		if s.Campaigns == nil {
			return out, api.Conflict("Content fixes aren't available in this process.")
		}
		var ab *campaigns.ComposerABTest
		if detail, err := s.Campaigns.Detail(ctx, ws, p, id); err == nil {
			ab = detail.ABTest
		}
		saved, err := s.Campaigns.Save(ctx, ws, p, id, campaigns.ComposerSaveContent{BaseVersion: int(c.Version), Subject: subject, Preheader: c.Preheader, MJML: src,
			ABTest: ab, Label: strings.Join(labels, ", ") + " from Review & send", Source: "manual"})
		if err != nil {
			return out, err
		}
		out.Version = saved.Version
		if len(a11y) > 0 {
			// The language versions get the same fixes: they need no words.
			var a11yLabels []string
			for _, ch := range a11y {
				a11yLabels = append(a11yLabels, ch.Label)
			}
			versions, err := s.fixLanguageVersions(ctx, ws, p, id, a11yFixes, strings.Join(a11yLabels, ", ")+" from Review & send")
			if err != nil {
				return out, err
			}
			langs := append([]string{accessibility.EmailLanguage(ctx, s.Q, ws.ID, accessibility.OwnerCampaign, id, 0)}, versions...)
			if err := s.Accessibility.RecordFix(ctx, ws, p, id, c.Name, saved.ContentHash, langs, a11y); err != nil {
				return out, err
			}
		}
	}
	s.changed(ctx, ws, id)
	return out, nil
}

func (s *Service) changed(ctx context.Context, ws *store.Workspace, id uuid.UUID) {
	if s.AfterChange != nil {
		s.AfterChange(ctx, ws, id)
	}
}

func (s *Service) excludeSunset(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.ReadinessCampaignRow) (string, error) {
	var raw map[string]any
	if err := json.Unmarshal(c.Audience, &raw); err != nil || raw == nil {
		raw = map[string]any{}
	}
	def := ParseAudience(c.Audience)
	if def.Empty() {
		return "", api.Invalid("body.fixes", "Pick an audience first.")
	}
	if def.ExcludesSunset() {
		return "Unengaged contacts are already excluded", nil
	}
	tiers := append(slices.Clone(def.ExcludeTiers), "sunset")
	raw["exclude_tiers"] = tiers
	// Excluding the tier ends an override of the sunset policy.
	delete(raw, "include_sunset")
	next, _ := json.Marshal(raw)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReadinessSetAudience(ctx, store.ReadinessSetAudienceParams{Audience: next, AudienceCount: c.AudienceCount, UpdatedBy: &p.UserID, Now: s.now(), WorkspaceID: ws.ID, ID: c.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.update", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"audience": c.Audience}, After: map[string]any{"audience": json.RawMessage(next)},
			Metadata: map[string]any{"source": "send_readiness", "fix": FixExcludeSunset}})
	})
	if err != nil {
		return "", err
	}
	return "Excluded unengaged contacts (no open in 180 days)", nil
}

func (s *Service) routeOnly(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.ReadinessCampaignRow, provider, field string) (string, error) {
	if !p.Can(authz.CampaignsSend) && !p.Can(authz.ProvidersManage) {
		return "", api.Forbidden(authz.CampaignsSend, nil).WithMeta("reason", "Changing the route of a send needs campaigns:send.")
	}
	pid, err := uuid.Parse(provider)
	if err != nil {
		return "", api.Invalid(field+".params.providerId", "Pick a sending provider.")
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return "", err
	}
	var target *store.Provider
	for i := range provs {
		if provs[i].ID == pid {
			target = &provs[i]
		}
	}
	if target == nil {
		return "", api.Invalid(field+".params.providerId", "That provider isn't connected to this workspace.")
	}
	if !state.Available(*target) {
		return "", api.Invalid(field+".params.providerId", target.Name+" can't take mail right now. Pick a healthy provider.")
	}
	if !state.Caps(*target).BulkAllowed() {
		return "", api.Invalid(field+".params.providerId", target.Name+"'s terms don't allow bulk marketing.")
	}
	next, _ := json.Marshal(RouteOverride{Splits: []routing.Split{{ProviderID: pid, Weight: 100}}})
	short := state.Short(target.Name, target.Adapter)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReadinessSetRouteOverride(ctx, store.ReadinessSetRouteOverrideParams{RouteOverride: next, UpdatedBy: &p.UserID, Now: s.now(), WorkspaceID: ws.ID, ID: c.ID}); err != nil {
			return err
		}
		var before any
		if len(c.RouteOverride) > 0 {
			before = c.RouteOverride
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.route_override", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"route_override": before}, After: map[string]any{"route_override": json.RawMessage(next), "provider": short},
			Metadata: map[string]any{"source": "send_readiness", "fix": FixRouteOnly}})
	})
	if err != nil {
		return "", err
	}
	return "This send goes 100% via " + short, nil
}

// ClearRouteOverride puts the campaign back on the stream's routing.
func (s *Service) ClearRouteOverride(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	c, err := s.Campaign(ctx, ws, id)
	if err != nil {
		return err
	}
	if len(c.RouteOverride) == 0 || string(c.RouteOverride) == "null" {
		return nil
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReadinessSetRouteOverride(ctx, store.ReadinessSetRouteOverrideParams{RouteOverride: nil, UpdatedBy: &p.UserID, Now: s.now(), WorkspaceID: ws.ID, ID: c.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.route_override", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"route_override": c.RouteOverride}, After: map[string]any{"route_override": nil}})
	})
	if err == nil {
		s.changed(ctx, ws, id)
	}
	return err
}
