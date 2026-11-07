package team_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The invitations a member sent end when the member is deactivated or
// removed: nobody vouches for the people they invited any more.
func TestLeavingMembersTakeTheirInvitations(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var l memberList
	linh.Do("GET", ws+"/members", nil).MustStatus(t, 200).JSON(t, &l)
	marcus := env.Client()
	marcus.SignInDemo("marcus@acme.io")
	sendInvites := func(emails ...string) []string {
		t.Helper()
		var res struct{ Invitations []struct{ Link string } }
		marcus.Do("POST", ws+"/invitations", map[string]any{"emails": emails, "roleId": roleID(t, l, "editor")}).MustStatus(t, 201).JSON(t, &res)
		tokens := []string{}
		for _, i := range res.Invitations {
			tokens = append(tokens, i.Link[strings.LastIndex(i.Link, "/")+1:])
		}
		return tokens
	}
	revoked := func(tokens []string) {
		t.Helper()
		for _, tok := range tokens {
			var v struct{ Status string }
			env.Client().Do("GET", "/invitations/"+tok, nil).MustStatus(t, 200).JSON(t, &v)
			if v.Status != "revoked" {
				t.Fatalf("invitation %s is %s", tok, v.Status)
			}
		}
	}
	m := byEmail(t, l, "marcus@acme.io")

	before := auditCount(t, env, "team.invite_revoked")
	tokens := sendInvites("new.one@example.com", "new.two@example.com")
	linh.Do("POST", ws+"/members/"+m.ID+"/deactivate", map[string]any{}).MustStatus(t, 200)
	revoked(tokens)
	if n := auditCount(t, env, "team.invite_revoked") - before; n != 2 {
		t.Fatalf("%d revocations audited", n)
	}
	var reason string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT reason FROM audit_events WHERE action = 'team.invite_revoked' LIMIT 1`).Scan(&reason); err != nil || !strings.Contains(reason, "Marcus Chen") {
		t.Fatalf("reason %q %v", reason, err)
	}
	var list struct{ Items []struct{ Email string } }
	linh.Do("GET", ws+"/invitations", nil).MustStatus(t, 200).JSON(t, &list)
	for _, i := range list.Items {
		if strings.HasPrefix(i.Email, "new.") {
			t.Fatalf("still pending: %+v", list.Items)
		}
	}

	linh.Do("POST", ws+"/members/"+m.ID+"/reactivate", nil).MustStatus(t, 204)
	marcus.SignInDemo("marcus@acme.io")
	tokens = sendInvites("new.three@example.com")
	linh.Do("DELETE", ws+"/members/"+m.ID, nil).MustStatus(t, 204)
	revoked(tokens)
	if n := auditCount(t, env, "team.invite_revoked") - before; n != 3 {
		t.Fatalf("%d revocations audited", n)
	}
}
