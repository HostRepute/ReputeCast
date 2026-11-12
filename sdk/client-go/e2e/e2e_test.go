// Package e2e checks the client against a running ReputeCast instance with
// the Acme demo; internal/sdktest runs it against a fresh one. It is
// skipped unless REPUTECAST_URL and REPUTECAST_API_KEY (subscribers:view,
// subscribers:create, subscribers:edit and transactional:send, without
// team:view) are set.
package e2e

import (
	"context"
	"errors"
	"os"
	"testing"

	reputecast "github.com/hostrepute/reputecast/sdk/client-go"
)

func TestAgainstAnInstance(t *testing.T) {
	baseURL, apiKey := os.Getenv("REPUTECAST_URL"), os.Getenv("REPUTECAST_API_KEY")
	if baseURL == "" || apiKey == "" {
		t.Skip("set REPUTECAST_URL and REPUTECAST_API_KEY to run against an instance")
	}
	workspace := os.Getenv("REPUTECAST_WORKSPACE")
	if workspace == "" {
		workspace = "acme-newsletter"
	}
	ctx := context.Background()
	rc, err := reputecast.New(baseURL, apiKey)
	if err != nil {
		t.Fatal(err)
	}

	// Lists.
	lists, err := rc.ListListsWithResponse(ctx, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists.JSON200.Items) == 0 {
		t.Fatal("the demo has lists")
	}
	digest := lists.JSON200.Items[0]
	for _, l := range lists.JSON200.Items {
		if l.Name == "Weekly Digest" {
			digest = l
		}
	}

	// Upsert a contact, twice: created, then updated.
	email, first, last := "sdk-e2e-go@example.com", "Gopher", "Go"
	consent := reputecast.ContactUpsertInputConsentSingle
	body := reputecast.ContactUpsertInput{Email: email, FirstName: &first, Lists: &[]string{digest.Id}, Tags: &[]string{"sdk-e2e"}, Consent: &consent}
	created, err := rc.UpsertContactWithResponse(ctx, workspace, body)
	if err != nil {
		t.Fatal(err)
	}
	if !created.JSON200.Created {
		t.Fatalf("upsert: %s", created.Body)
	}
	body.LastName = &last
	updated, err := rc.UpsertContactWithResponse(ctx, workspace, body)
	if err != nil {
		t.Fatal(err)
	}
	if updated.JSON200.Created || updated.JSON200.ContactId != created.JSON200.ContactId {
		t.Fatalf("upsert again: %s", updated.Body)
	}

	// Page through contacts with the cursor: three pages of 25.
	seen := map[string]bool{}
	pageSize := int64(25)
	params := &reputecast.ListContactsParams{PageSize: &pageSize}
	for range 3 {
		page, err := rc.ListContactsWithResponse(ctx, workspace, params)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range page.JSON200.Items {
			if seen[c.Id] {
				t.Fatalf("contact %s came twice", c.Id)
			}
			seen[c.Id] = true
		}
		if page.JSON200.NextCursor == nil || *page.JSON200.NextCursor == "" {
			t.Fatal("there is another page")
		}
		params.Cursor = page.JSON200.NextCursor
	}
	if len(seen) != 75 {
		t.Fatalf("paged through %d contacts", len(seen))
	}
	found, err := rc.ListContactsWithResponse(ctx, workspace, &reputecast.ListContactsParams{Q: &email})
	if err != nil {
		t.Fatal(err)
	}
	if len(found.JSON200.Items) == 0 || found.JSON200.Items[0].Id != created.JSON200.ContactId {
		t.Fatalf("search: %s", found.Body)
	}

	// A transactional message, with an Idempotency-Key: a repeat sends nothing new.
	key, name := "pwreset-u_42-t_7", "Sam Rivera"
	data := map[string]any{"first_name": "Sam", "reset_url": "https://app.acme.io/reset/t_7", "expires_in": "30 minutes"}
	send := func() *reputecast.TransactionalSendResult {
		t.Helper()
		res, err := rc.SendTransactionalMessageWithResponse(ctx, workspace, &reputecast.SendTransactionalMessageParams{IdempotencyKey: &key},
			reputecast.TransactionalSend{Template: "password-reset", To: reputecast.TransactionalAddress{Email: "sam.rivera@outlook.com", Name: &name}, Data: &data})
		if err != nil {
			t.Fatal(err)
		}
		return res.JSON202
	}
	msg := send()
	if msg.Id == "" || msg.Status == "" {
		t.Fatalf("message: %+v", msg)
	}
	if again := send(); again.Id != msg.Id {
		t.Fatalf("the repeat sent %s, not %s", again.Id, msg.Id)
	}

	// A missing permission is an *Error with the problem's fields.
	_, err = rc.ListMembersWithResponse(ctx, workspace, nil)
	var apiErr *reputecast.Error
	if !errors.As(err, &apiErr) || apiErr.Status != 403 || apiErr.Code != "permission_denied" || apiErr.Permission != "team:view" || apiErr.Detail == "" {
		t.Fatalf("members: %v", err)
	}

	t.Logf("ok go: %d lists, %d contacts paged, message %s, members answered 403 permission_denied", len(lists.JSON200.Items), len(seen), msg.Status)
}
