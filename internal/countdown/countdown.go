// Package countdown is the Countdown block of emails (M59 G-21): a timer
// to a moment in a time zone, drawn as an animated GIF each time the email
// is opened, so every reader sees the time left when they look.
//
//	POST /workspaces/{workspace}/countdowns   the block's settings → its image
//	GET  /t/cd/{token}.gif                    the image (HEAD too)
//
// Each distinct set of settings is one email_countdowns row per workspace,
// found again by the SHA-256 of its canonical JSON, so the composer gets
// the same address back for the same settings. The image's token is signed
// (internal/linktoken, Kind) and names the workspace and the row. Images
// are drawn from a per-settings atlas of pre-drawn digits and labels, kept
// briefly per second, and drawn at most four at a time.
package countdown

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/customhost"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/pages"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/store"
)

// Kind is the image token's kind; its MessageID names the countdown.
const Kind linktoken.Kind = 230

// MinContrast is the digits' least contrast on the background (WCAG AA for
// text).
const MinContrast = 4.5

// MaxEndText is the longest message shown once the countdown ends.
const MaxEndText = 60

// CountdownSettings are a Countdown block's settings, as the composer keeps
// them on the block (rc-countdown) and sends them here.
type CountdownSettings struct {
	// EndsAt is the local date and time it ends ("2026-11-28T23:59") in
	// TimeZone (IANA, "America/New_York").
	EndsAt   string `json:"endsAt" maxLength:"16" doc:"The local date and time the countdown ends, YYYY-MM-DDTHH:MM" example:"2026-11-28T23:59"`
	TimeZone string `json:"timeZone" maxLength:"64" doc:"IANA time zone of endsAt" example:"America/New_York"`
	// Units: days, hours, minutes and seconds; hours, minutes and seconds;
	// days, hours and minutes.
	Units    string `json:"units" enum:"dhms,hms,dhm"`
	Language string `json:"language" enum:"en,de,es,fr,vi" doc:"The labels' and the alt text's language: the language version being edited"`
	// Size: s 360×96, m 480×120, l 600×150 pixels.
	Size       string         `json:"size" enum:"s,m,l"`
	Background string         `json:"background" maxLength:"7" doc:"#RRGGBB" example:"#0F172A"`
	Digits     string         `json:"digits" maxLength:"7" doc:"#RRGGBB, at least 4.5:1 on the background" example:"#FFFFFF"`
	Labels     string         `json:"labels" maxLength:"7" doc:"#RRGGBB" example:"#CBD5E1"`
	Ended      CountdownEnded `json:"ended"`
}

// CountdownEnded is what the timer shows once it ends.
type CountdownEnded struct {
	Mode string `json:"mode" enum:"zeros,text"`
	Text string `json:"text,omitempty" doc:"The message shown when mode is text, up to 60 characters"`
}

// CountdownImage is a countdown's image for the block.
type CountdownImage struct {
	ID     string    `json:"id" format:"uuid"`
	URL    string    `json:"url" doc:"The animated GIF, drawn when the email is opened"`
	EndsAt time.Time `json:"endsAt"`
	Alt    string    `json:"alt" example:"Offer ends Fri 28 Nov at 23:59 (EST)"`
	Width  int       `json:"width"`
	Height int       `json:"height"`
}

// Service creates and serves countdowns.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Links *linktoken.Signer
	URLs  publicurl.Builder
	// Hosts says whose host a request came to (nil: every host carries
	// every workspace's images).
	Hosts *customhost.Hosts
	Log   *slog.Logger
	Now   func() time.Time

	once  sync.Once
	cache *caches
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

var (
	localRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T([01]\d|2[0-3]):[0-5]\d$`)
	colorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

// Normalize checks settings and returns them as they're kept (colours in
// capitals, no message unless the end shows one) with the moment they end.
// Problems are 422s in the request's language.
func Normalize(ctx context.Context, in CountdownSettings) (CountdownSettings, time.Time, error) {
	out := in
	tz := strings.TrimSpace(in.TimeZone)
	if tz == "" || tz == "Local" {
		return out, time.Time{}, api.Invalid("body.timeZone", say(ctx, "problem.timeZone", i18n.Args{"zone": tz}))
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return out, time.Time{}, api.Invalid("body.timeZone", say(ctx, "problem.timeZone", i18n.Args{"zone": tz}))
	}
	out.TimeZone = tz
	endsAt, ok := time.Time{}, false
	if localRe.MatchString(in.EndsAt) {
		endsAt, ok = pages.Local(in.EndsAt, tz)
	}
	if !ok || endsAt.Year() < 2000 {
		return out, time.Time{}, api.Invalid("body.endsAt", say(ctx, "problem.endsAt", nil))
	}
	// The API checks the enums; settings read from a block's attribute may
	// hold anything.
	if _, ok := unitSets[in.Units]; !ok {
		return out, time.Time{}, fmt.Errorf("countdown: unknown units %q", in.Units)
	}
	if _, ok := sizes[in.Size]; !ok {
		return out, time.Time{}, fmt.Errorf("countdown: unknown size %q", in.Size)
	}
	if l, ok := i18n.ParseLocale(in.Language); !ok || l == i18n.Pseudo {
		out.Language = string(i18n.English)
	}
	for _, c := range []struct {
		field string
		v     *string
	}{{"body.background", &out.Background}, {"body.digits", &out.Digits}, {"body.labels", &out.Labels}} {
		v := strings.TrimSpace(*c.v)
		if !colorRe.MatchString(v) {
			return out, time.Time{}, api.Invalid(c.field, say(ctx, "problem.color", nil))
		}
		*c.v = strings.ToUpper(v)
	}
	if r := contrast(out.Digits, out.Background); r < MinContrast {
		shown := accessibility.RoundDown(r)
		return out, time.Time{}, api.Invalid("body.digits", say(ctx, "problem.contrast", i18n.Args{"ratio": shown})).WithMeta("ratio", shown)
	}
	switch in.Ended.Mode {
	case "text":
		t := strings.Join(strings.Fields(in.Ended.Text), " ")
		if t == "" || utf8.RuneCountInString(t) > MaxEndText {
			return out, time.Time{}, api.Invalid("body.ended.text", say(ctx, "problem.text", nil))
		}
		if missing := missingGlyphs(t); len(missing) > 0 {
			return out, time.Time{}, api.Invalid("body.ended.text", say(ctx, "problem.textGlyphs", i18n.Args{"chars": string(missing)}))
		}
		out.Ended = CountdownEnded{Mode: "text", Text: t}
	default:
		out.Ended = CountdownEnded{Mode: "zeros"}
	}
	return out, endsAt, nil
}

// contrast is the WCAG contrast ratio of two #RRGGBB colours.
func contrast(a, b string) float64 {
	ca, cb := hexColor(a), hexColor(b)
	return accessibility.Ratio(accessibility.Color{R: ca.R, G: ca.G, B: ca.B, A: 1}, accessibility.Color{R: cb.R, G: cb.G, B: cb.B, A: 1})
}

// Hash is the key of normalized settings: the SHA-256 of their canonical
// JSON (fields in a fixed order).
func Hash(s CountdownSettings) ([]byte, []byte) {
	canon, _ := json.Marshal(s)
	sum := sha256.Sum256(canon)
	return sum[:], canon
}

// Create returns the image of a block's settings: the workspace's
// countdown with the same settings, or a new one (audited).
func (s *Service) Create(ctx context.Context, ws *store.Workspace, actor *uuid.UUID, in CountdownSettings) (CountdownImage, error) {
	set, endsAt, err := Normalize(ctx, in)
	if err != nil {
		return CountdownImage{}, err
	}
	hash, canon := Hash(set)
	var id uuid.UUID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		got, err := q.CountdownInsert(ctx, store.CountdownInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Hash: hash, Settings: canon,
			EndsAt: endsAt, CreatedBy: actor, CreatedAt: s.now()})
		if errors.Is(err, pgx.ErrNoRows) {
			row, err := q.CountdownByHash(ctx, store.CountdownByHashParams{WorkspaceID: ws.ID, Hash: hash})
			id = row.ID
			return err
		}
		if err != nil {
			return err
		}
		id = got
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "countdown.created", TargetType: "countdown", TargetID: id.String(),
			TargetName: fmt.Sprintf("%s %s", strings.Replace(set.EndsAt, "T", " ", 1), set.TimeZone), After: set})
	})
	if err != nil {
		return CountdownImage{}, err
	}
	return s.image(ws.ID, id, set, endsAt), nil
}

// Token is the image token of a workspace's countdown.
func (s *Service) Token(workspace, id uuid.UUID) string {
	return s.Links.Sign(linktoken.Claims{Kind: Kind, WorkspaceID: workspace, MessageID: id})
}

func (s *Service) image(workspace, id uuid.UUID, set CountdownSettings, endsAt time.Time) CountdownImage {
	sz := sizes[set.Size]
	return CountdownImage{ID: id.String(), URL: s.URLs.Countdown(s.Token(workspace, id)), EndsAt: endsAt, Alt: Alt(set, endsAt), Width: sz.X, Height: sz.Y}
}

// Alt is the image's alt text in the block's language: "Offer ends Fri 28
// Nov at 23:59 (EST)", in the countdown's time zone.
func Alt(set CountdownSettings, endsAt time.Time) string {
	l := langOf(set.Language)
	at := endsAt
	if loc, err := time.LoadLocation(set.TimeZone); err == nil {
		at = endsAt.In(loc)
	}
	return msgs.M("alt").Text(l, i18n.Args{"date": l.Region.ShortDate(at), "time": l.Region.Time(at, ""), "zone": at.Format("MST")})
}
