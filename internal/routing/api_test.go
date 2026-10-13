package routing_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

const ws = "/workspaces/acme-newsletter"

type streamIn struct {
	Stream     string           `json:"stream"`
	Splits     []map[string]any `json:"splits"`
	Fallback   string           `json:"fallback,omitempty"`
	Failover   string           `json:"failover"`
	Conditions []map[string]any `json:"conditions"`
}

type config struct {
	Streams []struct {
		Stream string
		Splits []struct {
			ProviderID string
			Weight     int
		}
		Fallback   string
		Failover   string
		Conditions []struct {
			Field      string
			Values     []string
			ProviderID string
			Note       string
		}
		Volume *struct {
			Count    int64
			Per      string
			Smallest int64
		}
	}
	Providers []struct {
		ID, Name, Adapter string
		BulkAllowed       bool
		WarmingUp         bool
	}
	SavedBy string
}

// input turns the saved routing back into a save body.
func input(c config) []streamIn {
	out := []streamIn{}
	for _, s := range c.Streams {
		in := streamIn{Stream: s.Stream, Fallback: s.Fallback, Failover: s.Failover, Splits: []map[string]any{}, Conditions: []map[string]any{}}
		for _, sp := range s.Splits {
			in.Splits = append(in.Splits, map[string]any{"providerId": sp.ProviderID, "weight": sp.Weight})
		}
		for _, cd := range s.Conditions {
			in.Conditions = append(in.Conditions, map[string]any{"field": cd.Field, "values": cd.Values, "providerId": cd.ProviderID, "note": cd.Note})
		}
		out = append(out, in)
	}
	return out
}

func TestRoutingSaveAndBulkTerms(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var c config
	owner.Do("GET", ws+"/routing", nil).MustStatus(t, 200).JSON(t, &c)
	if len(c.Streams) != 3 || c.Streams[0].Stream != "broadcast" || len(c.Streams[0].Splits) != 2 || len(c.Streams[0].Conditions) != 2 {
		t.Fatalf("routing %+v", c.Streams)
	}
	if v := c.Streams[0].Volume; v == nil || v.Per != "week" || v.Count == 0 || v.Smallest == 0 {
		t.Fatalf("broadcast volume %+v", v)
	}
	ids := map[string]string{}
	for _, p := range c.Providers {
		ids[p.Adapter] = p.ID
	}

	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("PUT", ws+"/routing", map[string]any{"streams": input(c)}).MustStatus(t, 403)

	// Weights must add up to 100.
	bad := input(c)
	bad[0].Splits[1]["weight"] = 20
	owner.Do("PUT", ws+"/routing", map[string]any{"streams": bad}).MustStatus(t, 422)

	// Broadcasts falling back to Postmark's transactional stream breaks its terms.
	bulk := input(c)
	bulk[0].Fallback = ids["postmark"]
	p := owner.Do("PUT", ws+"/routing", map[string]any{"streams": bulk}).MustStatus(t, 409).Problem(t)
	if p["code"] != "bulk_terms" {
		t.Fatalf("problem %+v", p)
	}
	owner.Do("PUT", ws+"/routing", map[string]any{"streams": bulk, "acknowledgeBulkTerms": true}).MustStatus(t, 200)

	// A plain change: Postal's share to 15%.
	next := input(c)
	next[0].Splits[0]["weight"] = 85
	next[0].Splits[1]["weight"] = 15
	var saved config
	owner.Do("PUT", ws+"/routing", map[string]any{"streams": next}).MustStatus(t, 200).JSON(t, &saved)
	if saved.Streams[0].Splits[1].Weight != 15 || saved.SavedBy != "Linh Nguyen" {
		t.Fatalf("saved %+v by %q", saved.Streams[0].Splits, saved.SavedBy)
	}
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = 'routing.update'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("routing.update audit = %d %v", n, err)
	}
}

func TestRouteTester(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("aisha@acme.io") // viewers may test routes
	type result struct {
		Provider, MailboxProvider string
		DKIMSelector              string `json:"dkimSelector"`
		Steps                     []struct {
			Kind, Field, Value string
			Bucket             int
		}
		Fallback struct{ Policy string }
		Cost     struct {
			Basis   string
			Per1000 *float64
		}
	}
	var r result
	owner.Do("POST", ws+"/routing/test", map[string]any{"stream": "broadcast", "email": "sam.rivera@outlook.com", "senderDomain": "news.acme.io"}).MustStatus(t, 200).JSON(t, &r)
	if r.Provider != "Amazon SES" || r.MailboxProvider != "outlook" || r.Fallback.Policy != "pause_and_ask" || r.Cost.Basis != "metered" {
		t.Fatalf("outlook route %+v", r)
	}
	if len(r.Steps) < 3 || r.Steps[1].Kind != "condition" || r.Steps[1].Field != "mailbox_provider" || r.Steps[1].Value != "Outlook" {
		t.Fatalf("steps %+v", r.Steps)
	}
	owner.Do("POST", ws+"/routing/test", map[string]any{"stream": "broadcast", "email": "pat@gmail.com"}).MustStatus(t, 200).JSON(t, &r)
	if r.Steps[1].Kind != "split" || r.Steps[1].Bucket < 0 || r.Steps[1].Bucket > 99 {
		t.Fatalf("gmail steps %+v", r.Steps)
	}
	owner.Do("POST", ws+"/routing/test", map[string]any{"stream": "system", "email": "pat@gmail.com"}).MustStatus(t, 200).JSON(t, &r)
	if r.Provider != "Postmark" || r.Fallback.Policy != "automatic" {
		t.Fatalf("system route %+v", r)
	}
	owner.Do("POST", ws+"/routing/test", map[string]any{"stream": "broadcast", "email": "not-an-address"}).MustStatus(t, 422)
}
