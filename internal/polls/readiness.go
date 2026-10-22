package polls

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// FixCloseDays makes a poll close N days after sending (Send Readiness'
// "Close 7 days after sending").
const FixCloseDays = "poll_close_days"

// ReadinessPoll is a poll of an email as Send Readiness checks it.
type ReadinessPoll struct {
	ID       string
	Question string
	Path     []int
	Problems []string
	// ProblemTexts are the Problems kept as messages, for a run that keeps
	// its text for later readers.
	ProblemTexts []i18n.Text
	// Close says when it closes: Mode date with ClosesAt (the end of that
	// day), days with Days after sending, or manual.
	Close    PollClose
	ClosesAt *time.Time
}

// ReadinessSurvey is a survey an email links to.
type ReadinessSurvey struct {
	ID       string
	Name     string
	Status   string
	ClosesAt *time.Time
}

// ReadinessFacts are an email's polls and surveys and where their links go.
type ReadinessFacts struct {
	Ran     bool
	Polls   []ReadinessPoll
	Surveys []ReadinessSurvey
	// Domain is the custom domain set for the public pages; DomainStatus
	// its check ("verified", "pending"…) and DomainDetail what the check
	// found. Instance is the address links use until it is verified.
	Domain       string
	DomainStatus string
	DomainDetail string
	Instance     string
	// TestAnswers counts the campaign's test answers (never counted).
	TestAnswers int
}

// CampaignReadiness reads the facts of a campaign's content.
func (s *Service) CampaignReadiness(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, src string) ReadinessFacts {
	out := ReadinessFacts{Ran: true}
	tz := ws.Timezone
	for _, b := range Read(src) {
		c := b.Config
		rp := ReadinessPoll{ID: c.ID, Question: c.Question, Path: b.Path, Problems: c.Problems(), ProblemTexts: c.ProblemTexts(), Close: c.Close}
		if c.Close.Mode == CloseDate {
			rp.ClosesAt = closesAt(c.Close, nil, tz)
		}
		out.Polls = append(out.Polls, rp)
	}
	for _, n := range surveysIn(src) {
		if sv, err := s.Q.SurveyByNumber(ctx, n); err == nil && sv.WorkspaceID == ws.ID {
			out.Surveys = append(out.Surveys, ReadinessSurvey{ID: sv.ID.String(), Name: sv.Name, Status: sv.Status, ClosesAt: sv.ClosesAt})
		}
	}
	if len(out.Polls) == 0 && len(out.Surveys) == 0 {
		return ReadinessFacts{}
	}
	if s.Forms != nil {
		if set, err := s.Forms.LoadSettings(ctx, ws); err == nil {
			out.Domain, out.DomainStatus, out.DomainDetail = set.DomainState()
		}
	}
	if u, err := url.Parse(s.URLs.Base()); err == nil {
		out.Instance = u.Host
	}
	if rows, err := s.Q.PollTestAnswers(ctx, store.PollTestAnswersParams{WorkspaceID: ws.ID, CampaignID: &campaignID}); err == nil {
		out.TestAnswers = len(rows)
	}
	return out
}

// FixMJML applies a poll fix to an email's MJML.
func FixMJML(src, action string, params map[string]string) (next, label, applied string, err error) {
	if action != FixCloseDays {
		return src, "", "", errors.New("unknown fix")
	}
	days, err := strconv.Atoi(params["days"])
	if err != nil || days < 1 || days > 365 {
		days = 7
	}
	root, err := mjml.Parse(src)
	if err != nil {
		return src, "", "", err
	}
	var found *mjml.Node
	var cfg PollConfig
	mjml.Body(root).Walk(func(n *mjml.Node, _ []int) {
		if found != nil {
			return
		}
		if raw, ok := n.Get(mjml.AttrPoll); ok {
			if c, err := ParseConfig(raw); err == nil && c.ID == params["poll"] {
				found, cfg = n, c
			}
		}
	})
	if found == nil {
		return src, "", "", errors.New("that poll isn't in the email any more; run the checks again")
	}
	cfg.Close = PollClose{Mode: CloseDays, Days: days}
	next, ok := mjml.SetAttr(src, found, mjml.AttrPoll, cfg.Encode())
	if !ok {
		return src, "", "", errors.New("the poll's settings couldn't be changed; open it in the composer")
	}
	word := strconv.Itoa(days) + " days"
	if days == 1 {
		word = "1 day"
	}
	q := strings.TrimSpace(cfg.Question)
	return next, "poll closes " + word + " after sending", "The poll “" + q + "” now closes " + word + " after sending", nil
}
