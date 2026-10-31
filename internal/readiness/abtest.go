package readiness

import (
	"context"
	"encoding/json"
	"math"
	"net/url"
	"slices"
	"strings"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/store"
)

// A/B variants (SR-7): the campaign is variant A; variants B and C of a
// subject test have their own subject and preheader, those of a content
// test their own body too. Each variant is checked like the campaign, and a
// variant that fails where A passes gets its own row.

// abTest is the stored shape of campaigns.ab_test that the checks need.
type abTest struct {
	Kind     string      `json:"kind"`
	Variants []abVariant `json:"variants"`
}

type abVariant struct {
	Key       string `json:"key"`
	Subject   string `json:"subject,omitempty"`
	Preheader string `json:"preheader,omitempty"`
	MJML      string `json:"mjml,omitempty"`
	HTML      string `json:"html,omitempty"`
}

// parseABTest reads campaigns.ab_test (nil without a test).
func parseABTest(raw json.RawMessage) *abTest {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var t abTest
	if json.Unmarshal(raw, &t) != nil || t.Kind == "" {
		return nil
	}
	return &t
}

// others are the variants besides A that send something A doesn't: a
// send-time test sends A's content at other times.
func (t *abTest) others() []abVariant {
	if t == nil || t.Kind == "send_time" {
		return nil
	}
	out := []abVariant{}
	for _, v := range t.Variants {
		if v.Key != "" && v.Key != "A" {
			out = append(out, v)
		}
	}
	return out
}

// variantRequest is the content a variant sends: its own subject and
// preheader (A's when it has none) and, in a content test, its own body.
func variantRequest(c store.ReadinessCampaignRow, kind string, v abVariant, probe bool) (campaigns.ComposerChecksRequest, bool) {
	req := campaigns.ComposerChecksRequest{Subject: firstNonEmpty(v.Subject, c.Subject), Preheader: firstNonEmpty(v.Preheader, c.Preheader),
		MJML: c.Mjml, HTML: c.Html, ProbeLinks: probe}
	own := kind == "content" && (strings.TrimSpace(v.MJML) != "" || strings.TrimSpace(v.HTML) != "")
	if own {
		req.MJML, req.HTML = v.MJML, v.HTML
	}
	return req, own
}

// variants runs the content checks of variants B and C. probe also
// requests their links, which only a content test's variants have of
// their own.
func (s *Service) variants(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow, t *abTest, probe bool) []VariantReport {
	out := []VariantReport{}
	if s.Campaigns == nil {
		return out
	}
	for _, v := range t.others() {
		req, own := variantRequest(c, t.Kind, v, probe)
		req.ProbeLinks = probe && own
		chk, err := s.Campaigns.Checks(ctx, ws, c.ID, req)
		if err != nil {
			if s.Log != nil {
				s.Log.WarnContext(ctx, "readiness variant check failed", "campaign", c.ID, "variant", v.Key, "err", err)
			}
			continue
		}
		out = append(out, VariantReport{Key: v.Key, Subject: req.Subject, Preheader: req.Preheader, Content: own, Report: chk.ContentCheckReport})
	}
	return out
}

// variantEditor opens a variant in the composer.
func (b *builder) variantEditor(key string) string {
	q := url.Values{}
	q.Set("variant", key)
	q.Set("panel", "deliverability")
	return b.href("/campaigns/"+b.f.Campaign.ID+"/edit") + "?" + q.Encode()
}

// variantRows adds a row for each check a variant fails where variant A
// passes: its subject, preheader or body trips spam heuristics, its body
// is near Gmail's clipping limit or mostly images, its personalization
// doesn't render, or its footer lost the unsubscribe link or address.
func (b *builder) variantRows() {
	ct := b.f.Content
	if !ct.Checked {
		return
	}
	a := ct.Report
	for _, v := range ct.Variants {
		name := v.Key
		open := link(tx("fix.openVariant", A{"variant": name}), b.variantEditor(name))
		label := tx("label.variant", A{"variant": name})
		r := v.Report
		hs := append(append([]string{}, r.SubjectHeuristics...), r.PreheaderHeuristics...)
		if v.Content {
			hs = append(hs, r.BodyHeuristics...)
		}
		switch hs = dedupe(hs); {
		case strings.TrimSpace(v.Subject) == "":
			b.add(check{Key: "content.spam.variant." + name, Group: GroupContent, Status: Block, Label: label,
				Title: tx("variants.subjectEmpty", A{"variant": name}), Detail: tx("content.subjectEmptyDetail", nil), Fixes: []fix{open}})
		case len(hs) > 0:
			b.add(check{Key: "content.spam.variant." + name, Group: GroupContent, Status: Warn, Label: label,
				Title:  tx("variants.spam", A{"variant": name, "subject": v.Subject, "count": len(hs)}),
				Detail: heuristicsText(hs), Ack: tx("variants.spamAck", A{"variant": name}), Fixes: []fix{open}})
		}
		if syntax := issuesOf(r, "personalization.syntax"); len(syntax) > 0 && len(issuesOf(a, "personalization.syntax")) == 0 {
			b.add(check{Key: "content.personalization.variant." + name, Group: GroupContent, Status: Block, Label: label,
				Title: tx("variants.personalization", A{"variant": name}), Detail: issueTitle(syntax[0]), Fixes: []fix{open}})
		}
		if !v.Content {
			continue
		}
		if r.SizeBytes >= contentcheck.NearClipBytes && a.SizeBytes < contentcheck.NearClipBytes {
			kb := int(math.Round(float64(r.SizeBytes) / 1024))
			b.add(check{Key: "content.size.variant." + name, Group: GroupContent, Status: Warn, Label: label,
				Title:  tx("variants.size", A{"variant": name, "kb": kb, "clip": 102}),
				Detail: tx("languages.versionSizeDetail", nil), Ack: tx("variants.sizeAck", A{"variant": name, "kb": kb}), Fixes: []fix{open}})
		}
		if r.ImageShare >= contentcheck.ImageShareLimit && a.ImageShare < contentcheck.ImageShareLimit {
			b.add(check{Key: "content.image_ratio.variant." + name, Group: GroupContent, Status: Warn, Label: label,
				Title: tx("variants.ratio", A{"variant": name, "share": ArgPercent(math.Round(r.ImageShare * 100)), "limit": ArgPercent(contentcheck.ImageShareLimit * 100)}),
				Ack:   tx("variants.ratioAck", A{"variant": name}), Fixes: []fix{open}})
		}
		if len(issuesOf(r, "compliance.footer")) > 0 && len(issuesOf(a, "compliance.footer")) == 0 {
			b.add(check{Key: "compliance.footer.variant." + name, Group: GroupCompliance, Status: Block, Label: label,
				Title: tx("variants.footer", A{"variant": name}), Detail: tx("compliance.footerDetail", nil), Fixes: []fix{open}})
		}
	}
}

// dedupe keeps the first of each item.
func dedupe(items []string) []string {
	out := []string{}
	for _, s := range items {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}
