package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/countdown"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() { seedCountdown = seedCountdownImpl }

// CountdownCampaign is the draft with the Countdown and Discount code blocks
// (ComposerTimedBlocks).
const CountdownCampaign = "Black Friday Early Access"

// CountdownSettings are the demo countdown's settings: it ends 16 days
// after the seed's day at 23:59 New York time (the boards: Fri 27 Nov),
// so the demo never shows an ended countdown.
func CountdownSettings(seeded time.Time) countdown.CountdownSettings {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	d := seeded.In(loc).AddDate(0, 0, 16)
	return countdown.CountdownSettings{EndsAt: d.Format("2006-01-02") + "T23:59", TimeZone: "America/New_York", Units: "dhms", Language: "en", Size: "l",
		Background: "#0F172A", Digits: "#FFFFFF", Labels: "#CBD5E1", Ended: countdown.CountdownEnded{Mode: "zeros"}}
}

func seedCountdownImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	set, endsAt, err := countdown.Normalize(ctx, CountdownSettings(s.Now))
	if err != nil {
		return err
	}
	hash, canon := countdown.Hash(set)
	countdownID := ID()
	if _, err := s.Q.CountdownInsert(ctx, store.CountdownInsertParams{ID: countdownID, WorkspaceID: ws.ID, Hash: hash, Settings: canon, EndsAt: endsAt,
		CreatedBy: &linh.ID, CreatedAt: s.Ago(50 * time.Minute)}); err != nil {
		return fmt.Errorf("countdown: %w", err)
	}
	links, err := demoLinks(s.Cfg)
	if err != nil {
		return err
	}
	src := publicurl.New(s.Cfg.HTTP.PublicURL).Countdown(links.Sign(linktoken.Claims{Kind: countdown.Kind, WorkspaceID: ws.ID, MessageID: countdownID}))
	const subject = "Black Friday early access: 15% off, subscribers first"
	const preheader = "Shop the sale before everyone else, until the countdown runs out."
	body := countdownMJML(subject, preheader, string(canon), src, countdown.Alt(set, endsAt))
	res := mjml.Compile(body)
	if len(res.Errors) > 0 {
		return fmt.Errorf("compile the countdown demo: %s", res.Errors[0].Message)
	}
	id := ID()
	created, saved := s.Ago(55*time.Minute), s.Ago(40*time.Minute)
	audience, _ := json.Marshal(map[string]any{"lists": []any{s.Lists["Weekly Digest"]}})
	count := int64(31192)
	plain := contentcheck.PlainText(res.HTML)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, audience_count, mjml, html, plain_text, content_hash, version,
		created_by, updated_by, created_at, updated_at, autosaved_at)
		VALUES ($1,$2,$3,'draft',$4,$5,$6,$7,$8,$9,$10,$11,1,$12,$12,$13,$14,$14)`,
		id, ws.ID, CountdownCampaign, subject, preheader, audience, count, body, res.HTML, plain, campaigns.ContentHash(subject, preheader, body, res.HTML, nil),
		linh.ID, created, saved); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_versions (campaign_id, version, subject, preheader, mjml, html, plain_text, saved_by, saved_at, label, source, size_bytes)
		VALUES ($1,1,$2,$3,$4,$5,$6,$7,$8,'created','create',$9)`, id, subject, preheader, body, res.HTML, plain, linh.ID, saved, len(res.HTML)); err != nil {
		return err
	}
	s.Campaigns[CountdownCampaign] = id
	return nil
}

// countdownMJML is the Black Friday email: the offer, the Countdown block,
// and the Discount code block of the Welcome 15% pool.
func countdownMJML(subject, preheader, settings, src, alt string) string {
	return `<mjml>
  <mj-head>
    <mj-title>` + composerXMLText(subject) + `</mj-title>
    <mj-preview>` + composerXMLText(preheader) + `</mj-preview>
    <mj-attributes>
      <mj-all font-family="` + composerPlexFont + `"></mj-all>
      <mj-text color="#1F2937" padding="0px"></mj-text>
    </mj-attributes>
  </mj-head>
  <mj-body width="600px" background-color="#F3F4F6">
    <mj-section background-color="#0F172A" padding="32px 24px 12px">
      <mj-column>
        <mj-text align="center" font-size="12px" letter-spacing="0.08em" color="#CBD5E1">ACME MERCH · BLACK FRIDAY</mj-text>
        <mj-text align="center" font-size="28px" font-weight="600" line-height="1.25" color="#FFFFFF" padding="10px 0px 0px">Early access starts now</mj-text>
        <mj-text align="center" font-size="15px" line-height="1.6" color="#CBD5E1" padding="10px 0px 0px">Subscribers shop the sale before everyone else, until the countdown below runs out.</mj-text>
      </mj-column>
    </mj-section>
    <mj-section background-color="#0F172A" padding="0px 0px 24px">
      <mj-column>
        <mj-image rc-name="Countdown" rc-countdown='` + settings + `' src="` + src + `" alt="` + composerXMLText(alt) + `" width="600px" padding="0px"></mj-image>
      </mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="28px 24px">
      <mj-column>
        <mj-text font-size="15px" line-height="1.6">Here is 15% off your order. The code works once, until the countdown ends.</mj-text>
        <mj-raw><!-- rc:coupon {"pool":"welcome15","label":"Your code","button":"Shop now","buttonUrl":true} --></mj-raw>
      </mj-column>
    </mj-section>
` + composerDigestFooter + `
  </mj-body>
</mjml>`
}

// demoLinks signs the demo's links with the instance's key, as the server
// will verify them: the configured key, or the key file the seed command
// created with the secrets box. Without either (a test seeding the demo on
// its own) the links get a key of their own, and no file is written.
func demoLinks(cfg *config.Config) (*linktoken.Signer, error) {
	key := cfg.Secrets.Key
	if key == "" && cfg.Secrets.KeyFile != "" {
		if raw, err := os.ReadFile(cfg.Secrets.KeyFile); err == nil {
			key = strings.TrimSpace(string(raw))
		}
	}
	if key == "" {
		key = secrets.GenerateKey()
	}
	cur, err := config.DecodeKey(key)
	if err != nil {
		return nil, err
	}
	var prev [][]byte
	for _, p := range cfg.Secrets.PreviousKeys {
		k, err := config.DecodeKey(p)
		if err != nil {
			return nil, err
		}
		prev = append(prev, k)
	}
	return linktoken.NewSigner(cur, prev...)
}
