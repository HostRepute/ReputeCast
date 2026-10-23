package referrals

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// BlockCheck is what the Referral block does in an email, for Send
// Readiness and the composer's checks.
type BlockCheck struct {
	// Present: the email has the block.
	Present bool
	// Program is the program it reads; Off when it's off, None when the
	// workspace has none.
	ProgramID string
	ListName  string
	Off       bool
	None      bool
	// OffAt and OffBy say when and by whom it was turned off; Zone is the
	// workspace's timezone for the time.
	OffAt *time.Time
	OffBy string
	Zone  string
	// Shown recipients are on the program's list, of Total.
	Shown, Total int64
}

var referralMarker = regexp.MustCompile(`<!--\s*rc:referral\s*(\{[\s\S]*?\})?\s*-->`)

// CheckBlock reads the Referral block of an email: its program and how
// many of the recipients it shows to (byList: recipients per list).
func (s *Service) CheckBlock(ctx context.Context, ws *store.Workspace, html string, byList map[uuid.UUID]int64, total int64) (BlockCheck, error) {
	out := BlockCheck{Total: total}
	m := referralMarker.FindStringSubmatch(html)
	if m == nil {
		return out, nil
	}
	out.Present = true
	bound := ""
	if m[1] != "" {
		var b ReferralBlock
		if json.Unmarshal([]byte(m[1]), &b) == nil {
			bound = b.Program
		}
	}
	pr, err := s.blockProgram(ctx, ws, bound)
	if err != nil {
		return out, err
	}
	if pr == nil {
		out.None = true
		return out, nil
	}
	out.ProgramID, out.ListName = pr.ReferralProgram.ID.String(), pr.ListName
	out.Off = pr.ReferralProgram.Status != "on"
	out.OffAt, out.OffBy, out.Zone = pr.ReferralProgram.StatusAt, pr.StatusByName, ws.Timezone
	out.Shown = byList[pr.ReferralProgram.ListID]
	return out, nil
}

// TokenCheck is a referral token typed outside the Referral block, and how
// many recipients it would print nothing for (ComposerGrowBlocks: "a check:
// referral.link is empty for 16,225 recipients who aren't on Weekly
// Digest").
type TokenCheck struct {
	// Token is the first one without a fallback ("referral.link"); empty
	// when there is none.
	Token string
	// Empty recipients aren't on the program's list (or the program is off,
	// Off; or there is none, None), of Total.
	Empty, Total int64
	ListName     string
	Off, None    bool
}

var (
	// blockMarkers are the blocks' markers: their tokens are the block's own.
	blockMarkers = regexp.MustCompile(`<!--\s*rc:[a-z][a-z-]*[\s\S]*?-->`)
	// referralToken is {{ referral.… }}, with its filters.
	referralToken = regexp.MustCompile(`\{\{-?\s*(referral\.[a-z_][a-z0-9_.]*)\s*(\|[^}]*)?-?\}\}`)
)

// CheckTokens finds a referral token outside the Referral block and the
// recipients it prints nothing for: those who aren't on the program's list,
// or everyone while the program is off. A token with a fallback
// (| default: "…") is fine.
func (s *Service) CheckTokens(ctx context.Context, ws *store.Workspace, html string, byList map[uuid.UUID]int64, total int64) (TokenCheck, error) {
	out := TokenCheck{Total: total}
	if !strings.Contains(html, "referral.") {
		return out, nil
	}
	for _, m := range referralToken.FindAllStringSubmatch(blockMarkers.ReplaceAllString(html, ""), -1) {
		if !strings.Contains(m[2], "default") {
			out.Token = m[1]
			break
		}
	}
	if out.Token == "" {
		return out, nil
	}
	pr, err := s.blockProgram(ctx, ws, "")
	if err != nil {
		return out, err
	}
	switch {
	case pr == nil:
		out.None, out.Empty = true, total
	case pr.ReferralProgram.Status != "on":
		out.Off, out.Empty, out.ListName = true, total, pr.ListName
	default:
		out.ListName = pr.ListName
		out.Empty = max(total-byList[pr.ReferralProgram.ListID], 0)
	}
	return out, nil
}

// ReadinessRow is a Referral row of Send Readiness. Its texts are kept
// messages: Send Readiness writes them in each reader's language.
type ReadinessRow struct {
	Status                           string // pass or warn
	Title, Detail, Ack, Label, Short i18n.Text
	Links                            []ReadinessLink
}

// ReadinessLink is a link of a row: its label and the workspace path it opens.
type ReadinessLink struct {
	Label i18n.Text
	Href  string
}

func readinessText(key string, args i18n.Args) i18n.Text {
	return msgs.M("readiness." + key).Keep(args)
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// ReadinessRow is the token check's row; ok is false when there's nothing
// to say.
func (c TokenCheck) ReadinessRow(campaignID string) (row ReadinessRow, ok bool) {
	if c.Token == "" || c.Empty == 0 {
		return ReadinessRow{}, false
	}
	row = ReadinessRow{Status: "warn", Label: readinessText("tokensLabel", nil), Short: readinessText("tokensShort", nil),
		Detail: readinessText("tokenDetail", i18n.Args{"fallback": "{{ " + c.Token + ` | default: "…" }}`}), Ack: readinessText("tokenAck", nil),
		Links: []ReadinessLink{{Label: readinessText("openComposer", nil), Href: "/campaigns/" + campaignID + "/edit"}}}
	switch {
	case c.None:
		row.Title = readinessText("tokenNone", i18n.Args{"token": c.Token})
	case c.Off:
		row.Title = readinessText("tokenOff", i18n.Args{"token": c.Token})
	default:
		row.Title = readinessText("tokenEmpty", i18n.Args{"token": c.Token, "count": c.Empty, "list": c.ListName})
	}
	return row, true
}

// ReadinessRow is the block's row: who it shows to, or why it won't.
// when writes the time the program was turned off in each reader's region
// ("2 Oct, 14:05": readiness.ArgDayMonthTime).
func (c BlockCheck) ReadinessRow(campaignID string, when func(time.Time) any) ReadinessRow {
	row := ReadinessRow{Status: "pass", Label: readinessText("label", nil), Short: readinessText("short", nil)}
	switch {
	case c.None:
		row.Status = "warn"
		row.Title, row.Detail, row.Ack = readinessText("none", nil), readinessText("noneDetail", nil), readinessText("hiddenAck", nil)
		row.Links = []ReadinessLink{{Label: readinessText("setUp", nil), Href: "/grow/referrals"}}
	case c.Off:
		row.Status = "warn"
		row.Title, row.Ack = readinessText("off", nil), readinessText("hiddenAck", nil)
		row.Detail = readinessText("offDetail", nil)
		if c.OffAt != nil {
			at := c.OffAt.UTC()
			if loc, err := time.LoadLocation(c.Zone); err == nil && c.Zone != "" {
				at = c.OffAt.In(loc)
			}
			row.Detail = readinessText("offDetailAt", i18n.Args{"when": when(at), "name": c.OffBy, "named": yesNo(c.OffBy != "")})
		}
		row.Links = []ReadinessLink{{Label: readinessText("removeBlock", nil), Href: "/campaigns/" + campaignID + "/edit"},
			{Label: readinessText("settings", nil), Href: "/grow/referrals/settings?program=" + c.ProgramID}}
	default:
		row.Title = readinessText("shown", i18n.Args{"shown": c.Shown, "total": c.Total})
		if c.Shown < c.Total {
			row.Detail = readinessText("hiddenFor", i18n.Args{"count": c.Total - c.Shown, "list": c.ListName})
		} else {
			row.Detail = readinessText("everyone", i18n.Args{"list": c.ListName})
		}
	}
	return row
}
