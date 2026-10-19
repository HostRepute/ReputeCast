package referrals

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"

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

// formatCount is "31,604".
func formatCount(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n < 0 {
		return s
	}
	var b []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, ',')
		}
		b = append(b, s[i])
	}
	return string(b)
}

// ReadinessText is the block's check as Send Readiness words it.
func (c BlockCheck) ReadinessText() (status, title, detail, ack string) {
	switch {
	case c.None:
		return "warn", "The Referral block has no referral program to show. It will be hidden.", "Set up a referral program in Grow, or remove the block.",
			"I understand the Referral block won't show."
	case c.Off:
		detail = "Remove the block, or turn the program on before sending."
		if c.OffAt != nil {
			at := c.OffAt.UTC()
			if loc, err := time.LoadLocation(c.Zone); err == nil && c.Zone != "" {
				at = c.OffAt.In(loc)
			}
			when := at.Format("2 Jan, 15:04")
			if c.OffBy != "" {
				detail = "Turned off on " + when + " by " + c.OffBy + ". " + detail
			} else {
				detail = "Turned off on " + when + ". " + detail
			}
		}
		return "warn", "The referral program is off. The Referral block will be hidden.", detail, "I understand the Referral block won't show."
	}
	title = "Referral block: shown to " + formatCount(c.Shown) + " of " + formatCount(c.Total) + " recipients"
	if c.Shown < c.Total {
		detail = "Hidden for the " + formatCount(c.Total-c.Shown) + " recipients who aren't on " + c.ListName + ". Each reader on it gets their own link and progress."
	} else {
		detail = "Everyone in this audience is on " + c.ListName + ". Each reader gets their own link and progress."
	}
	return "pass", title, detail, ""
}
