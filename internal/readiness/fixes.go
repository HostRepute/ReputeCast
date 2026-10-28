package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// ReadinessFixRequest is one fix to apply.
type ReadinessFixRequest struct {
	Action string            `json:"action" enum:"rule_add_param,exclude_sunset,include_unengaged,route_only,insert_footer,replace_link,use_subject,minify,mark_language_up_to_date,mark_language_translated,hold_language,release_language,accessibility_decorative,accessibility_color,accessibility_one_click,visibility_show_everyone,visibility_hide_block,visibility_remove_condition,poll_close_days,exclude_risky,verify_addresses,sponsor_remove_block,sponsor_refresh_block"`
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

// footerMJML is the composer's footer block (blocks.ts footerBlock), its
// words in the language of the person adding it, as the composer writes
// them.
func footerMJML(l i18n.Lang, address string) string {
	addr := address
	if strings.TrimSpace(addr) == "" {
		addr = msgs.M("footer.address").Text(l, nil)
	}
	addr = html.EscapeString(addr)
	word := func(key string) string { return html.EscapeString(msgs.M(key).Text(l, nil)) }
	const plex = `'IBM Plex Sans', Helvetica, Arial, sans-serif`
	const teal = "#0F766E"
	return footerMarker + `<mj-section background-color="#F3F4F6" padding="18px 24px"><mj-column><mj-text align="center" font-family="` + plex +
		`" font-size="12px" line-height="1.6" color="#5B6472" padding="0px">` + addr + `<br><a href="{{ unsubscribe_url }}" style="color:` + teal +
		`;text-decoration:none">` + word("footer.unsubscribe") + `</a> · <a href="{{ preferences_url }}" style="color:` + teal + `;text-decoration:none">` +
		word("footer.preferences") + `</a> · <a href="{{ view_in_browser_url }}" style="color:` + teal + `;text-decoration:none">` + word("footer.viewInBrowser") +
		`</a></mj-text></mj-column></mj-section>`
}

// InsertFooter adds the footer block at the end of the MJML body, or puts
// the composer's footer block back as new when it lost its address or links.
func InsertFooter(src, address string) (string, bool) {
	return InsertFooterIn(i18n.Default, src, address)
}

// InsertFooterIn is InsertFooter with the block's words in a language.
func InsertFooterIn(l i18n.Lang, src, address string) (string, bool) {
	if i := strings.Index(src, footerMarker); i >= 0 {
		rest := src[i+len(footerMarker):]
		start, end := strings.Index(rest, "<mj-section"), strings.Index(rest, "</mj-section>")
		if start >= 0 && end > start && strings.TrimSpace(rest[:start]) == "" {
			return src[:i] + footerMJML(l, address) + rest[end+len("</mj-section>"):], true
		}
	}
	i := strings.LastIndex(strings.ToLower(src), "</mj-body>")
	if i < 0 {
		return src, false
	}
	return src[:i] + footerMJML(l, address) + "\n  " + src[i:], true
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

// cleanDestination refuses an address a link would be pointed at when it
// is on the URI blocklists, or when looking it up failed: a fix never swaps
// a listed link for another one it didn't check (R1.U3).
func (s *Service) cleanDestination(ctx context.Context, to, field string) error {
	if s.Links == nil {
		return nil
	}
	host := hostOf(to)
	domains := []string{host}
	if d := contentcheck.RegistrableDomain(host); d != host {
		domains = append(domains, d)
	}
	rep, err := s.Links.Check(ctx, domains)
	lists := contentcheck.URIListNames
	if err == nil && len(rep.Lists) > 0 {
		lists = rep.Lists
	}
	loc := i18n.LangFrom(ctx).Locale
	for _, d := range domains {
		if listed := rep.Listed[d]; len(listed) > 0 {
			return api.Invalid(field, say(ctx, "problem.destinationListed", A{"host": host, "lists": i18n.AndList(loc, listed)}))
		}
		if err != nil || rep.Unknown[d] != "" {
			return api.Invalid(field, say(ctx, "problem.destinationUnknown", A{"host": host, "lists": i18n.AndList(loc, lists)}))
		}
	}
	return nil
}

// HTMLOnly reports whether a campaign is written in HTML only (the HTML
// editor, an import): it has no MJML to edit or compile.
func HTMLOnly(mjmlSrc, html string) bool {
	return strings.TrimSpace(mjmlSrc) == "" && strings.TrimSpace(html) != ""
}

// mjmlFixes are the one-click fixes that edit MJML: a campaign written in
// HTML is fixed in the editor instead.
var mjmlFixes = map[string]bool{FixInsertFooter: true, FixAccessibilityDecorative: true, FixAccessibilityColor: true, FixAccessibilityOneClick: true,
	FixVisibilityShowEveryone: true, FixVisibilityHideBlock: true, FixVisibilityRemoveCondition: true, FixPollCloseDays: true,
	FixSponsorRemove: true, FixSponsorRefresh: true}

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
		return out, api.Conflict(say(ctx, "problem.alreadySending", nil))
	}
	// Content fixes are applied together as one new version. A campaign
	// written in HTML (no MJML) keeps its HTML: a fix edits the HTML, and a
	// fix that only knows MJML is refused (Assemble doesn't offer it).
	htmlOnly := HTMLOnly(c.Mjml, c.Html)
	src, htmlSrc, subject, content := c.Mjml, c.Html, c.Subject, false
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
		case FixIncludeUnengaged:
			label, err := s.includeUnengaged(ctx, ws, p, c)
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
			return out, api.Invalid(field, say(ctx, "problem.verifyFromDialog", nil))
		case FixRouteOnly:
			label, err := s.routeOnly(ctx, ws, p, c, fx.Params["providerId"], field)
			if err != nil {
				return out, err
			}
			out.Applied = append(out.Applied, label)
		case FixInsertFooter:
			if htmlOnly || strings.TrimSpace(src) == "" {
				return out, api.Invalid(field, say(ctx, "problem.footerInHtml", nil))
			}
			next, ok := InsertFooterIn(i18n.LangFrom(ctx), src, templates.WorkspaceAddress(ws))
			if !ok {
				return out, api.Invalid(field, say(ctx, "problem.noBody", nil))
			}
			src, content = next, true
			labels = append(labels, say(ctx, "version.footer", nil))
			out.Applied = append(out.Applied, say(ctx, "applied.footer", nil))
		case FixRuleAddParam:
			// A workspace rule's query parameter on the links missing it
			// (SR-10), in the MJML or, for an HTML campaign, the HTML.
			param, value := strings.TrimSpace(fx.Params["param"]), strings.TrimSpace(fx.Params["value"])
			if param == "" || value == "" {
				return out, api.Invalid(field, say(ctx, "problem.ruleParam", nil))
			}
			body := src
			if htmlOnly {
				body = htmlSrc
			}
			next, n := AddLinkParam(body, param, value)
			if n == 0 {
				return out, api.Invalid(field, say(ctx, "problem.paramThere", A{"param": param}))
			}
			if htmlOnly {
				htmlSrc = next
			} else {
				src = next
			}
			content = true
			labels = append(labels, say(ctx, "version.ruleParam", A{"param": param}))
			out.Applied = append(out.Applied, say(ctx, "applied.addedParam", A{"param": param, "value": value, "count": n}))
		case FixReplaceLink:
			from, to := strings.TrimSpace(fx.Params["from"]), strings.TrimSpace(fx.Params["to"])
			if from == "" || !validLink(to) {
				return out, api.Invalid(field, say(ctx, "problem.replaceLink", nil))
			}
			if err := s.cleanDestination(ctx, to, field+".params.to"); err != nil {
				return out, err
			}
			body := src
			if htmlOnly {
				body = htmlSrc
			}
			next, n := ReplaceLink(body, from, to)
			if n == 0 {
				return out, api.Invalid(field, say(ctx, "problem.linkGone", nil))
			}
			if htmlOnly {
				htmlSrc = next
			} else {
				src = next
			}
			content = true
			labels = append(labels, say(ctx, "version.link", nil))
			out.Applied = append(out.Applied, say(ctx, "applied.replaced", A{"from": displayURL(from), "to": displayURL(to)}))
		case FixUseSubject:
			next := strings.TrimSpace(fx.Params["subject"])
			if next == "" || len(next) > 998 {
				return out, api.Invalid(field, say(ctx, "problem.subjectLength", A{"max": 998}))
			}
			subject, content = next, true
			labels = append(labels, say(ctx, "version.subject", nil))
			out.Applied = append(out.Applied, say(ctx, "applied.subject", A{"subject": next}))
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
			labels = append(labels, say(ctx, "version.minified", nil))
			out.Applied = append(out.Applied, say(ctx, "applied.minified", nil))
		case FixMarkLanguageUpToDate, FixMarkLanguageTranslated, FixHoldLanguage, FixReleaseLanguage:
			label, err := s.languageFix(ctx, ws, p, id, fx.Action, fx.Params["language"], field)
			if err != nil {
				return out, err
			}
			out.Applied = append(out.Applied, label)
		case FixAccessibilityDecorative, FixAccessibilityColor, FixAccessibilityOneClick:
			if s.Accessibility == nil {
				return out, api.Conflict(say(ctx, "problem.noAccessibility", nil))
			}
			if htmlOnly {
				return out, api.Invalid(field, say(ctx, "problem.htmlEditor", nil))
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
				return out, api.Invalid(field, say(ctx, "problem.htmlEditor", nil))
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
				return out, api.Conflict(say(ctx, "problem.noSponsor", nil))
			}
			if strings.TrimSpace(src) == "" {
				return out, api.Invalid(field, say(ctx, "problem.htmlEditor", nil))
			}
			next, applied, err := s.Sponsor.FixSponsor(ctx, ws, id, src, fx.Action)
			if err != nil {
				return out, api.Invalid(field, sentence(capitalize(err.Error())))
			}
			src, content = next, true
			if fx.Action == FixSponsorRemove {
				labels = append(labels, say(ctx, "version.sponsorRemoved", nil))
			} else {
				labels = append(labels, say(ctx, "version.sponsorUpdated", nil))
			}
			out.Applied = append(out.Applied, applied)
		default:
			return out, api.Invalid(field+".action", say(ctx, "problem.unknownFix", nil))
		}
	}
	out.Version = int(c.Version)
	if content {
		if s.Campaigns == nil {
			return out, api.Conflict(say(ctx, "problem.noContentFixes", nil))
		}
		var ab *campaigns.ComposerABTest
		if detail, err := s.Campaigns.Detail(ctx, ws, p, id); err == nil {
			ab = detail.ABTest
		}
		// MJML is compiled again on save; HTML-only content is saved as it is
		// (minified when the campaign asks for it), never left empty.
		html := ""
		if htmlOnly {
			html = htmlSrc
		}
		saved, err := s.Campaigns.Save(ctx, ws, p, id, campaigns.ComposerSaveContent{BaseVersion: int(c.Version), Subject: subject, Preheader: c.Preheader, MJML: src, HTML: html,
			ABTest: ab, Label: say(ctx, "version.fromReview", A{"changes": strings.Join(labels, ", ")}), Source: "manual"})
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
			versions, err := s.fixLanguageVersions(ctx, ws, p, id, a11yFixes, say(ctx, "version.fromReview", A{"changes": strings.Join(a11yLabels, ", ")}))
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

// excludeSunset leaves the unengaged contacts out of the campaign again,
// and with them an override of the sunset policy.
func (s *Service) excludeSunset(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.ReadinessCampaignRow) (string, error) {
	def := ParseAudience(c.Audience)
	if def.Empty() {
		return "", api.Invalid("body.fixes", say(ctx, "problem.pickAudience", nil))
	}
	if def.LeavesOutUnengaged(sendDefaults(ctx, s.Q, ws.ID).ExcludeUnengaged) {
		return say(ctx, "applied.alreadyExcluded", nil), nil
	}
	err := s.setAudience(ctx, ws, p, c, FixExcludeSunset, func(raw map[string]any) {
		raw["include_unengaged"] = false
		for _, k := range []string{"includeUnengaged", "include_sunset", "includeSunset"} {
			delete(raw, k)
		}
	})
	if err != nil {
		return "", err
	}
	return say(ctx, "applied.excludedUnengaged", A{"days": 180}), nil
}

// includeUnengaged puts back the unengaged contacts a send leaves out
// ("Include them", SR-11); the check then warns about them.
func (s *Service) includeUnengaged(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.ReadinessCampaignRow) (string, error) {
	def := ParseAudience(c.Audience)
	if def.Empty() {
		return "", api.Invalid("body.fixes", say(ctx, "problem.pickAudience", nil))
	}
	if !def.LeavesOutUnengaged(sendDefaults(ctx, s.Q, ws.ID).ExcludeUnengaged) {
		return say(ctx, "applied.alreadyIncluded", nil), nil
	}
	err := s.setAudience(ctx, ws, p, c, FixIncludeUnengaged, func(raw map[string]any) {
		raw["include_unengaged"] = true
		delete(raw, "includeUnengaged")
		// An exclusion of the tier would keep them out.
		for _, k := range []string{"exclude_tiers", "excludeTiers"} {
			tiers, ok := raw[k].([]any)
			if !ok {
				continue
			}
			kept := []any{}
			for _, t := range tiers {
				if t != "sunset" {
					kept = append(kept, t)
				}
			}
			if len(kept) == 0 {
				delete(raw, k)
			} else {
				raw[k] = kept
			}
		}
	})
	if err != nil {
		return "", err
	}
	return say(ctx, "applied.includedUnengaged", nil), nil
}

// setAudience changes campaigns.audience with edit, on the JSON as stored
// (keys it doesn't know are kept), and audits the change as the fix.
func (s *Service) setAudience(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.ReadinessCampaignRow, fixKind string, edit func(raw map[string]any)) error {
	var raw map[string]any
	if err := json.Unmarshal(c.Audience, &raw); err != nil || raw == nil {
		raw = map[string]any{}
	}
	edit(raw)
	next, _ := json.Marshal(raw)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReadinessSetAudience(ctx, store.ReadinessSetAudienceParams{Audience: next, AudienceCount: c.AudienceCount, UpdatedBy: &p.UserID, Now: s.now(), WorkspaceID: ws.ID, ID: c.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.update", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"audience": c.Audience}, After: map[string]any{"audience": json.RawMessage(next)},
			Metadata: map[string]any{"source": "send_readiness", "fix": fixKind}})
	})
}

func (s *Service) routeOnly(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.ReadinessCampaignRow, provider, field string) (string, error) {
	if !p.Can(authz.CampaignsSend) && !p.Can(authz.ProvidersManage) {
		// Who can, by name, never a permission key (SR-13).
		helpers := access.Helpers(ctx, s.Q, ws.ID, authz.CampaignsSend)
		names := "none"
		if len(helpers) > 0 {
			names = i18n.OrList(i18n.LangFrom(ctx).Locale, helpers)
		}
		pr := api.Err(http.StatusForbidden, api.CodePermissionDenied, say(ctx, "problem.routeNeedsSend", A{"names": names}))
		pr.Permission, pr.Helpers = string(authz.CampaignsSend), helpers
		return "", pr
	}
	pid, err := uuid.Parse(provider)
	if err != nil {
		return "", api.Invalid(field+".params.providerId", say(ctx, "problem.pickProvider", nil))
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
		return "", api.Invalid(field+".params.providerId", say(ctx, "problem.providerNotConnected", nil))
	}
	if !state.Available(*target) {
		return "", api.Invalid(field+".params.providerId", say(ctx, "problem.providerUnavailable", A{"provider": target.Name}))
	}
	if !state.Caps(*target).BulkAllowed() {
		return "", api.Invalid(field+".params.providerId", say(ctx, "problem.providerTerms", A{"provider": target.Name}))
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
	return say(ctx, "applied.routeOnly", A{"share": i18n.LangFrom(ctx).Region.Percent(100, 0), "provider": short}), nil
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
