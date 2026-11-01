package transactional_test

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/mailer"
)

type testMail struct {
	mu   sync.Mutex
	msgs []mailer.Message
}

func (m *testMail) SendSystem(_ context.Context, msg mailer.Message) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return "test", nil
}

func (m *testMail) SystemAvailable(context.Context, *uuid.UUID) bool { return true }

// TestSendTemplateTest: the template editor's Send test renders the draft
// with a test data set and sends it through the System route; without
// transactional:send only to one's own address (review 2026-11 TX-5).
func TestSendTemplateTest(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	box := &testMail{}
	env.App.Mail = box
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	// A draft from the starter: the test sends the draft, unpublished.
	owner.Do("POST", base+"/templates", map[string]any{"name": "Password reset", "key": "password-reset", "starter": "password-reset"}).MustStatus(t, 201)
	var res struct{ SentTo, Provider string }
	owner.Do("POST", base+"/templates/password-reset/test", map[string]any{"sendTo": "qa@acme.io"}).MustStatus(t, 200).JSON(t, &res)
	if res.SentTo != "qa@acme.io" || res.Provider == "" || len(box.msgs) != 1 || !strings.HasPrefix(box.msgs[0].Subject, "[Test] ") ||
		!strings.Contains(box.msgs[0].HTML, "Sam") {
		t.Fatalf("test sent: %+v, %d emails", res, len(box.msgs))
	}
	var logged int
	if err := env.App.Pool.QueryRow(t.Context(), `SELECT count(*) FROM messages m JOIN transactional_messages x ON x.message_id = m.id WHERE m.email = 'qa@acme.io'`).Scan(&logged); err != nil || logged != 0 {
		t.Fatalf("a test reached the message log: %d %v", logged, err)
	}
	owner.Do("POST", base+"/templates/password-reset/test", map[string]any{"sendTo": "not an address"}).MustStatus(t, http.StatusUnprocessableEntity)
	owner.Do("POST", base+"/templates/password-reset/test", map[string]any{"sendTo": "qa@acme.io", "testData": "No such set"}).MustStatus(t, http.StatusUnprocessableEntity)
	owner.Do("POST", base+"/templates/nope/test", map[string]any{"sendTo": "qa@acme.io"}).MustStatus(t, http.StatusNotFound)

	editor := env.Client()
	editor.SignInDemo("sofia@acme.io")
	editor.Do("POST", base+"/templates/password-reset/test", map[string]any{"sendTo": "qa@acme.io"}).MustStatus(t, http.StatusForbidden)
	editor.Do("POST", base+"/templates/password-reset/test", map[string]any{"sendTo": "sofia@acme.io"}).MustStatus(t, 200)
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("POST", base+"/templates/password-reset/test", map[string]any{"sendTo": "aisha@acme.io"}).MustStatus(t, http.StatusForbidden)
}
