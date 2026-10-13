package instance

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// InstanceSettings is Instance admin › Instance & updates › Instance.
type InstanceSettings struct {
	Name            string `json:"name"`
	PublicURL       string `json:"publicUrl"`
	DefaultTimezone string `json:"defaultTimezone"`
	// LogoSVG is the instance logo (PNG uploads are wrapped in an SVG);
	// empty means the ReputeCast mark.
	LogoSVG string `json:"logoSvg,omitempty"`
	// URLCheck is the last public URL check.
	URLCheck *accounts.URLCheck `json:"urlCheck,omitempty"`
	// ConfigPublicURL is the URL the server was started with
	// (http.public_url). A saved PublicURL replaces it for every link and
	// webhook; the command line and the HTTPS-only cookie setting still read
	// it.
	ConfigPublicURL string `json:"configPublicUrl,omitempty"`
	// Webhooks is set on a save that moved the public URL.
	Webhooks *PublicURLWebhooks `json:"webhooks,omitempty"`
}

// PublicURLWebhooks is what a new public URL did to provider webhooks.
type PublicURLWebhooks struct {
	Reregistering int                   `json:"reregistering" doc:"Providers whose webhook ReputeCast moves to the new URL, in the background"`
	Manual        []PublicURLManualHook `json:"manual" doc:"Providers whose webhook is set up in the provider itself"`
}

// PublicURLManualHook is a provider whose webhook has to be changed by hand.
type PublicURLManualHook struct {
	Workspace string `json:"workspace"`
	Provider  string `json:"provider"`
	URL       string `json:"url" doc:"The provider's webhook address on the new public URL"`
}

// Settings returns the instance settings.
func (s *Service) Settings(ctx context.Context) (InstanceSettings, error) {
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return InstanceSettings{}, err
	}
	out := InstanceSettings{Name: st.Name, PublicURL: st.PublicUrl, DefaultTimezone: st.DefaultTimezone}
	if s.Cfg != nil {
		out.ConfigPublicURL = s.Cfg.HTTP.PublicURL
	}
	if st.LogoSvg != nil {
		out.LogoSVG = *st.LogoSvg
	}
	if len(st.PublicUrlCheck) > 0 && string(st.PublicUrlCheck) != "null" {
		var c accounts.URLCheck
		if json.Unmarshal(st.PublicUrlCheck, &c) == nil {
			if st.PublicUrlCheckedAt != nil {
				c.CheckedAt = *st.PublicUrlCheckedAt
			}
			out.URLCheck = &c
		}
	}
	return out, nil
}

// InstanceSettingsInput saves the instance basics.
type InstanceSettingsInput struct {
	Name            string `json:"name" minLength:"1" maxLength:"80"`
	PublicURL       string `json:"publicUrl" maxLength:"300"`
	DefaultTimezone string `json:"defaultTimezone" maxLength:"64"`
}

// SaveSettings stores the name, public URL and default timezone.
func (s *Service) SaveSettings(ctx context.Context, in InstanceSettingsInput) (InstanceSettings, error) {
	before, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return InstanceSettings{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return InstanceSettings{}, api.Invalid("body.name", "Name the instance, like Acme ReputeCast.")
	}
	pub, err := normalizePublicURL(in.PublicURL)
	if err != nil {
		return InstanceSettings{}, err
	}
	tz := strings.TrimSpace(in.DefaultTimezone)
	if _, err := time.LoadLocation(tz); err != nil || tz == "" {
		return InstanceSettings{}, api.Invalid("body.defaultTimezone", "Use a timezone like America/New_York.")
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		// A new URL also drops the last URL check, which describes the old one.
		after, err := q.InstanceAdminSaveSettings(ctx, store.InstanceAdminSaveSettingsParams{Name: name, PublicUrl: pub, DefaultTimezone: tz})
		if err != nil {
			return err
		}
		var changes []string
		if before.Name != after.Name {
			changes = append(changes, "name "+after.Name)
		}
		if before.PublicUrl != after.PublicUrl {
			changes = append(changes, "public URL "+after.PublicUrl+" (was "+before.PublicUrl+")")
		}
		if before.DefaultTimezone != after.DefaultTimezone {
			changes = append(changes, "timezone "+after.DefaultTimezone)
		}
		if len(changes) == 0 {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "instance.settings_updated", TargetType: "instance", TargetName: after.Name,
			Before:   map[string]any{"name": before.Name, "public_url": before.PublicUrl, "default_timezone": before.DefaultTimezone},
			After:    map[string]any{"name": after.Name, "public_url": after.PublicUrl, "default_timezone": after.DefaultTimezone},
			Metadata: map[string]any{"summary": strings.Join(changes, " · ")}})
	})
	if err != nil {
		return InstanceSettings{}, err
	}
	out, err := s.Settings(ctx)
	if err != nil || before.PublicUrl == pub || s.OnPublicURL == nil {
		return out, err
	}
	// The setting is saved; moving the webhooks follows from it.
	moved, err := s.OnPublicURL(ctx, before.PublicUrl, pub)
	if err != nil {
		s.Log.WarnContext(ctx, "move provider webhooks to the new public URL", "err", err)
	}
	out.Webhooks = moved
	return out, nil
}

func normalizePublicURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.RawQuery != "" || u.Fragment != "" {
		return "", api.Invalid("body.publicUrl", "Use the full address people open, like https://reputecast.acme.io.")
	}
	return strings.TrimRight(u.Scheme+"://"+u.Host+u.EscapedPath(), "/"), nil
}

// InstanceURLCheckInput checks a URL (the saved one when empty).
type InstanceURLCheckInput struct {
	URL string `json:"url,omitempty" maxLength:"300"`
}

// CheckURL checks the public URL's DNS, TLS and reachability and keeps
// the result for the saved URL.
func (s *Service) CheckURL(ctx context.Context, in InstanceURLCheckInput) (InstanceSettings, error) {
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return InstanceSettings{}, err
	}
	target := s.publicURL(st)
	if strings.TrimSpace(in.URL) != "" {
		if target, err = normalizePublicURL(in.URL); err != nil {
			return InstanceSettings{}, api.Invalid("body.url", "Use the full address people open, like https://reputecast.acme.io.")
		}
	}
	if target == "" {
		return InstanceSettings{}, api.Invalid("body.url", "Enter the public URL first.")
	}
	res, err := s.Accounts.CheckPublicURL(ctx, target)
	if err != nil {
		return InstanceSettings{}, err
	}
	out, err := s.Settings(ctx)
	if err != nil {
		return out, err
	}
	if target == strings.TrimRight(st.PublicUrl, "/") || st.PublicUrl == "" {
		if err := s.Q.InstanceAdminSetURLCheck(ctx, mustJSON(res)); err != nil {
			return out, err
		}
	}
	out.URLCheck = &res
	return out, nil
}

// InstanceLogoInput uploads a logo: an SVG document or a PNG, as a data URL
// or base64.
type InstanceLogoInput struct {
	Data string `json:"data" minLength:"10" maxLength:"1400000"`
}

const (
	maxSVG = 100 << 10
	maxPNG = 512 << 10
)

// SetLogo stores the logo shown on the sign-in page and in team emails.
func (s *Service) SetLogo(ctx context.Context, in InstanceLogoInput) (InstanceSettings, error) {
	svg, err := logoSVG(in.Data)
	if err != nil {
		return InstanceSettings{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.InstanceAdminSetLogo(ctx, &svg); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "instance.logo_updated", TargetType: "instance", TargetName: "Logo", Metadata: map[string]any{"bytes": len(svg)}})
	})
	if err != nil {
		return InstanceSettings{}, err
	}
	return s.Settings(ctx)
}

// RemoveLogo goes back to the ReputeCast mark.
func (s *Service) RemoveLogo(ctx context.Context) (InstanceSettings, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.InstanceAdminSetLogo(ctx, nil); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "instance.logo_removed", TargetType: "instance", TargetName: "Logo", Metadata: map[string]any{"summary": "uses the ReputeCast mark"}})
	})
	if err != nil {
		return InstanceSettings{}, err
	}
	return s.Settings(ctx)
}

// logoSVG validates an upload and returns the SVG to store. PNGs are
// re-encoded (metadata dropped) and wrapped in a square SVG.
func logoSVG(data string) (string, error) {
	invalid := func(msg string) error { return api.Invalid("body.data", msg) }
	data = strings.TrimSpace(data)
	var raw []byte
	switch {
	case strings.HasPrefix(data, "data:image/svg+xml;base64,"), strings.HasPrefix(data, "data:image/png;base64,"):
		b, err := base64.StdEncoding.DecodeString(data[strings.Index(data, ",")+1:])
		if err != nil {
			return "", invalid("The file couldn't be read. Upload an SVG or PNG.")
		}
		raw = b
	case strings.HasPrefix(data, "<"):
		raw = []byte(data)
	default:
		b, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return "", invalid("The file couldn't be read. Upload an SVG or PNG.")
		}
		raw = b
	}
	if bytes.HasPrefix(raw, []byte("\x89PNG")) {
		if len(raw) > maxPNG {
			return "", invalid("The PNG is larger than 512 KB. Use a smaller, square image.")
		}
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			return "", invalid("That PNG couldn't be decoded.")
		}
		b := img.Bounds()
		if b.Dx() > 1024 || b.Dy() > 1024 || b.Dx() < 16 || b.Dy() < 16 {
			return "", invalid("Use a square PNG between 16 and 1024 pixels.")
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return "", err
		}
		return pngInSVG(buf.Bytes(), b), nil
	}
	if len(raw) > maxSVG {
		return "", invalid("The SVG is larger than 100 KB. Simplify it or upload a PNG.")
	}
	if err := checkSVG(raw); err != nil {
		return "", invalid(err.Error())
	}
	return string(raw), nil
}

func pngInSVG(p []byte, b image.Rectangle) string {
	side := max(b.Dx(), b.Dy())
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d"><image width="%d" height="%d" x="%d" y="%d" href="data:image/png;base64,%s"/></svg>`,
		side, side, side, side, b.Dx(), b.Dy(), (side-b.Dx())/2, (side-b.Dy())/2, base64.StdEncoding.EncodeToString(p))
}

var (
	svgEvent   = regexp.MustCompile(`(?i)\son[a-z]+\s*=`)
	svgBadHref = regexp.MustCompile(`(?i)(?:xlink:)?href\s*=\s*["'](?:\s*)(?:javascript:|data:(?:text|application)|https?:|//)`)
	svgCSSURL  = regexp.MustCompile(`(?i)(?:url\(\s*["']?\s*(?:https?:|//|javascript:)|@import)`)
)

// checkSVG accepts a well-formed SVG without scripts, event handlers,
// embedded HTML or external references. The logo is shown through <img>,
// which runs no scripts anyway; this keeps the stored document clean for
// emails and downloads too.
func checkSVG(raw []byte) error {
	lower := strings.ToLower(string(raw))
	for _, bad := range []struct{ token, what string }{
		{"<script", "a script"}, {"<foreignobject", "embedded HTML"}, {"<!entity", "an XML entity"}, {"<iframe", "an iframe"},
		{"<embed", "embedded content"}, {"<object", "embedded content"}, {"javascript:", "a javascript: link"},
	} {
		if strings.Contains(lower, bad.token) {
			return fmt.Errorf("The SVG contains %s, which isn't allowed in a logo. Export it again as a plain image.", bad.what) //nolint:staticcheck // ST1005: shown to people as a sentence
		}
	}
	if svgEvent.MatchString(string(raw)) || svgBadHref.MatchString(string(raw)) || svgCSSURL.MatchString(string(raw)) {
		return errors.New("The SVG contains event handlers or external links, which aren't allowed in a logo. Export it again as a plain image.") //nolint:staticcheck // ST1005: shown to people as a sentence
	}
	d := xml.NewDecoder(bytes.NewReader(raw))
	root := ""
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return errors.New("The SVG isn't well-formed XML. Export it again from your design tool.") //nolint:staticcheck // ST1005: shown to people as a sentence
		}
		if se, ok := tok.(xml.StartElement); ok && root == "" {
			root = se.Name.Local
		}
	}
	if root != "svg" {
		return errors.New("The file isn't an SVG image. Upload an SVG or a square PNG.") //nolint:staticcheck // ST1005: shown to people as a sentence
	}
	return nil
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
