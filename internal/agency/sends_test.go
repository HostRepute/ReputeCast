package agency_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// A campaign sent by email and by text at the same time is one send:
// the Overview counts its emails and its texts on one card, which opens on
// the email.
func TestSendByEmailAndTextIsOneSend(t *testing.T) {
	env, p := northwind(t)
	ctx := context.Background()
	const name = "Weekend gear sale, by email and text"
	o := getOverview(t, p["tomas"])
	fabrikam := workspaceID(t, o, "fabrikam-outdoors")
	var emailID uuid.UUID
	var at time.Time
	var audience []byte
	if err := env.App.Pool.QueryRow(ctx, `SELECT id, scheduled_at, audience FROM campaigns WHERE workspace_id = $1 AND name = $2`, fabrikam, name).
		Scan(&emailID, &at, &audience); err != nil {
		t.Fatal(err)
	}
	sms := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, editor_mode, mjml, html, plain_text, version,
			channel, content_hash, created_at, updated_at, scheduled_at, audience_count)
		VALUES ($1,$2,$3,'scheduled','','',$4,'html','','','',1,'sms','sms-test',now(),now(),$5,12480)`, sms, fabrikam, name, audience, at); err != nil {
		t.Fatal(err)
	}
	var view struct {
		Sends []struct {
			CampaignID string
			Name       string
			Channel    string
			Recipients int64
			Texts      int64
		}
		Counts struct{ Sends int }
	}
	p["tomas"].Do("GET", "/agency/overview", nil).MustStatus(t, 200).JSON(t, &view)
	if view.Counts.Sends != 4 {
		t.Fatalf("sends: %d", view.Counts.Sends)
	}
	found := false
	for _, s := range view.Sends {
		if s.Name != name {
			continue
		}
		if found {
			t.Fatal("the email and the text are listed apart")
		}
		found = true
		if s.Recipients != 141220 || s.Texts != 12480 || s.Channel != "email" || s.CampaignID != emailID.String() {
			t.Fatalf("one send by email and text: %+v", s)
		}
	}
	if !found {
		t.Fatalf("no %s in %+v", name, view.Sends)
	}
	// The client's next send in its row says the same.
	var pulse struct {
		NextSend struct {
			Name              string
			Recipients, Texts int64
		}
	}
	p["tomas"].Do("GET", "/agency/clients/"+fabrikam, nil).MustStatus(t, 200).JSON(t, &pulse)
	if pulse.NextSend.Name != name || pulse.NextSend.Recipients != 141220 || pulse.NextSend.Texts != 12480 {
		t.Fatalf("next send: %+v", pulse.NextSend)
	}
}
