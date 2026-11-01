package transactional

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

// TransactionalTemplateTestInput sends a template's draft (or a version)
// as a test: rendered with one of its test data sets, through the System
// route (review 2026-11 TX-5).
type TransactionalTemplateTestInput struct {
	TransactionalPreviewInput
	// SendTo is the test's address; without transactional:send only one's
	// own.
	SendTo string `json:"sendTo" minLength:"3" maxLength:"254"`
}

// TransactionalTemplateTestResult says where the test went.
type TransactionalTemplateTestResult struct {
	SentTo   string `json:"sentTo"`
	TestData string `json:"testData,omitempty"`
	Provider string `json:"provider"`
}

// SendTemplateTest renders the template like Preview and sends it with
// "[Test]" in the subject through the System route. Nothing goes to the
// message log or its numbers.
func (s *Service) SendTemplateTest(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string, in TransactionalTemplateTestInput) (TransactionalTemplateTestResult, error) {
	to, ok := validEmail(strings.TrimSpace(in.SendTo))
	if !ok {
		return TransactionalTemplateTestResult{}, api.Invalid("body.sendTo", say(ctx, "test.address", i18n.Args{"address": clip(in.SendTo, 80)}))
	}
	if p == nil || (!strings.EqualFold(to, p.Email) && !p.Can(authz.TransactionalSend)) {
		own := ""
		if p != nil {
			own = p.Email
		}
		return TransactionalTemplateTestResult{}, api.Err(http.StatusForbidden, api.CodePermissionDenied, say(ctx, "test.ownAddress", i18n.Args{"email": own}))
	}
	r, err := s.Preview(ctx, ws, ref, in.TransactionalPreviewInput)
	if err != nil {
		return TransactionalTemplateTestResult{}, err
	}
	if !r.OK {
		why := r.Problem
		if why == "" && len(r.Missing) > 0 {
			why = say(ctx, "test.missing", i18n.Args{"names": i18n.AndList(lang(ctx).Locale, r.Missing), "count": len(r.Missing)})
		}
		return TransactionalTemplateTestResult{}, problem(http.StatusUnprocessableEntity, CodeTemplateDoesntRun, say(ctx, "test.doesNotRender", i18n.Args{"problem": why}))
	}
	m := s.mailer()
	if m == nil || !m.SystemAvailable(ctx, &ws.ID) {
		return TransactionalTemplateTestResult{}, api.Err(http.StatusConflict, api.CodeProviderUnavailable, say(ctx, "test.noSystemRoute", nil))
	}
	provider, err := m.SendSystem(ctx, mailer.Message{WorkspaceID: &ws.ID, To: to, Subject: "[Test] " + r.Subject, HTML: r.HTML, Text: r.Text, Kind: "transactional_test"})
	if err != nil {
		if errors.Is(err, mailer.ErrNoSystemProvider) {
			return TransactionalTemplateTestResult{}, api.Err(http.StatusConflict, api.CodeProviderUnavailable, say(ctx, "test.noSystemRoute", nil))
		}
		return TransactionalTemplateTestResult{}, api.Err(http.StatusBadGateway, api.CodeProviderUnavailable, say(ctx, "test.refused", i18n.Args{"error": err.Error()}))
	}
	return TransactionalTemplateTestResult{SentTo: to, TestData: in.TestData, Provider: provider}, nil
}

// mailer is the System route's sender, nil when none is wired.
func (s *Service) mailer() mailer.Sender {
	if s.Mail == nil {
		return nil
	}
	return s.Mail()
}
