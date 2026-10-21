package demo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hostrepute/reputecast/internal/sms/twilio/twiliotest"
)

// The sample's texting (X21's SMS, agencies brief §4): Fabrikam Outdoors
// and Alpine Ski House text through Twilio (the in-process demo account)
// from a toll-free number for the US and Canada, which is where Usage's
// SMS segments come from. The sample has no phone numbers or SMS consent,
// so it has no text campaign: Fabrikam's weekend sale is its email. No
// text ever leaves: the account is the recorded demo one.

// nwSMSClients are the clients that text, with their toll-free number.
var nwSMSClients = []struct {
	key, brand, number, ref string
}{
	{"fabrikam", "Fabrikam:", "+18885550173", "PN0173c4b2a19e8d7f6a5b4c3d2e1f0a9b"},
	{"alpine", "Alpine Ski House:", "+18885550129", "PN0129a8b7c6d5e4f3a2b1c0d9e8f7a6b5"},
}

func (n *nwSeed) sms(ctx context.Context) error {
	for _, c := range nwSMSClients {
		if err := n.smsClient(ctx, c.key, c.brand, c.number, c.ref); err != nil {
			return err
		}
	}
	return nil
}

// smsClient connects Twilio with its toll-free number, verified, routed
// for the US and Canada, with the client's SMS rules and terms.
func (n *nwSeed) smsClient(ctx context.Context, key, brand, number, ref string) error {
	ws := n.ws[key]
	by := n.people[map[string]string{"fabrikam": "mei", "alpine": "ella"}[key]]
	connected := n.at(-120, 10, 12)
	id := ID()
	cfg, _ := json.Marshal(map[string]string{"account_sid": twiliotest.AccountSID, "api_key_sid": twiliotest.KeySID, "mode": "live"})
	credsJSON, _ := json.Marshal(map[string]string{"api_key_secret": twiliotest.KeySecret})
	creds, err := n.Box.Seal(credsJSON, "sms_provider:"+id.String()+":credentials")
	if err != nil {
		return err
	}
	secret, err := n.Box.Seal([]byte(TwilioDemoWebhookSecret), "sms_provider:"+id.String()+":webhook_secret")
	if err != nil {
		return err
	}
	health, _ := json.Marshal(map[string]any{"lastSuccessAt": n.Ago(time.Minute), "pricesReadAt": n.at(-1, 6, 0), "optOutsReadAt": n.Ago(4 * time.Minute)})
	if _, err := n.Tx.Exec(ctx, `INSERT INTO sms_providers (id, workspace_id, adapter, name, account_label, account_hint, config, credentials_enc, webhook_token,
			webhook_secret_enc, webhooks, status, status_since, health, last_receipt_at, last_checked_at, registrations_read_at, demo, connected_by, connected_at)
		VALUES ($1,$2,'twilio','Twilio',$3,'AC…3f9',$4,$5,$6,$7,'registered','healthy',$8,$9,$10,$11,$12,true,$13,$8)`,
		id, ws.ID, ws.Name, cfg, creds, smsToken("northwind-"+key), secret, connected, health, n.Ago(6*time.Minute), n.Ago(time.Minute), n.at(0, 9, 0), by.ID); err != nil {
		return err
	}
	if _, err := n.Tx.Exec(ctx, `INSERT INTO sms_setups (id, workspace_id, adapter, provider_id, step, status, state, created_by, created_at, updated_at)
		VALUES ($1,$2,'twilio',$3,'done','finished','{}',$4,$5,$6)`, ID(), ws.ID, id, by.ID, connected.Add(-5*time.Minute), connected.Add(20*time.Minute)); err != nil {
		return err
	}
	sender := ID()
	if _, err := n.Tx.Exec(ctx, `INSERT INTO sms_senders (id, workspace_id, provider_id, kind, value, countries, throughput, provider_ref, added_by, added_at)
		VALUES ($1,$2,$3,'toll_free',$4,$5,3,$6,$7,$8)`, sender, ws.ID, id, number, []string{"US", "CA"}, ref, by.ID, connected.Add(8*time.Minute)); err != nil {
		return err
	}
	details, _ := json.Marshal(map[string]string{"useCase": "Order updates and offers for customers who opted in", "volume": "Up to 50,000 texts",
		"optIn": "Checkout and signup forms · consent text v1 · unchecked by default", "submittedBy": "Mei Lin"})
	verified := n.at(-110, 12, 0)
	if _, err := n.Tx.Exec(ctx, `INSERT INTO sms_registrations (id, workspace_id, sender_id, kind, status, status_on, source, details, read_at, updated_at)
		VALUES ($1,$2,$3,'toll_free','verified',$4,'api',$5,$6,$7)`, ID(), ws.ID, sender, time.Date(verified.Year(), verified.Month(), verified.Day(), 0, 0, 0, 0, time.UTC),
		details, n.at(0, 9, 0), verified); err != nil {
		return err
	}
	if _, err := n.Tx.Exec(ctx, `INSERT INTO sms_routes (id, workspace_id, position, countries, sender_id, created_at, updated_at) VALUES ($1,$2,0,$3,$4,$5,$5)`,
		ID(), ws.ID, []string{"US", "CA"}, sender, connected.Add(30*time.Minute)); err != nil {
		return err
	}
	asOf := n.at(-1, 12, 0)
	for _, country := range []string{"US", "CA"} {
		if _, err := n.Tx.Exec(ctx, `INSERT INTO sms_prices (workspace_id, provider_id, country, price, source, as_of) VALUES ($1,$2,$3,0.0107,'api',$4)`,
			ws.ID, id, country, time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC)); err != nil {
			return err
		}
	}
	name := ws.Name
	if _, err := n.Tx.Exec(ctx, `INSERT INTO sms_settings (workspace_id, brand_prefix, confirmation_text, help_reply, opt_out_reply, opt_in_reply, updated_by, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, ws.ID, brand,
		brand+" You're signed up for "+name+" texts. Msg & data rates may apply. Reply HELP for help, STOP to cancel.",
		brand+" Help at "+nwSiteOf(key)+". Msg & data rates may apply. Reply STOP to cancel.",
		brand+" You're unsubscribed from "+name+" texts and won't get more. Reply START to resubscribe.",
		brand+" You're subscribed to "+name+" texts again. Reply HELP for help, STOP to cancel.", by.ID, connected.Add(40*time.Minute)); err != nil {
		return err
	}
	terms, _ := json.Marshal(map[string]string{
		"intro":     "These terms cover the texts you agree to when you tick the SMS box at checkout or on one of our forms. Email and texts are separate: one never signs you up for the other.",
		"program":   name + " texts: order updates and offers.",
		"frequency": "Up to 6 messages a month.",
		"cost":      "Message and data rates may apply.",
		"help":      "Text HELP to " + number + ".",
		"optOut":    "Text STOP at any time. We'll confirm once and send nothing more.",
		"senders":   number + " in the US and Canada.",
		"privacy":   "We never sell or share your number for marketing.",
		"carriers":  "Carriers aren't liable for delayed or undelivered messages.",
	})
	_, err = n.Tx.Exec(ctx, `INSERT INTO sms_terms_versions (workspace_id, version, content, published_by, published_at) VALUES ($1,1,$2,$3,$4)`,
		ws.ID, terms, by.ID, connected.Add(45*time.Minute))
	return err
}

// nwSiteOf is a client's web address, for its SMS help reply.
func nwSiteOf(key string) string {
	return map[string]string{"fabrikam": "fabrikamoutdoors.example/help", "alpine": "alpineskihouse.example/help"}[key]
}
