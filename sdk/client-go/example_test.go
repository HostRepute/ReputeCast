package reputecast_test

import (
	"context"
	"fmt"
	"log"
	"os"

	reputecast "github.com/hostrepute/reputecast/sdk/client-go"
)

type user struct{ ID, Email, Name, FirstName string }

type resetToken struct{ ID, URL string }

// sendPasswordReset sends the published template password-reset; a retry
// with the same Idempotency-Key never sends twice.
func sendPasswordReset(ctx context.Context, u user, token resetToken) error {
	rc, err := reputecast.New("https://reputecast.acme.io", os.Getenv("REPUTECAST_KEY"))
	if err != nil {
		return err
	}
	key := fmt.Sprintf("pwreset-%s-%s", u.ID, token.ID)
	data := map[string]any{"first_name": u.FirstName, "reset_url": token.URL}
	res, err := rc.SendTransactionalMessageWithResponse(ctx, "acme-newsletter",
		&reputecast.SendTransactionalMessageParams{IdempotencyKey: &key},
		reputecast.TransactionalSend{
			Template: "password-reset",
			To:       reputecast.TransactionalAddress{Email: u.Email, Name: &u.Name},
			Data:     &data,
		})
	if err != nil {
		return err // an API error is a *reputecast.Error
	}
	log.Printf("message %s is %s", res.JSON202.Id, res.JSON202.Status)
	return nil
}

func ExampleNew() {
	u := user{ID: "u_42", Email: "sam.rivera@outlook.com", Name: "Sam Rivera", FirstName: "Sam"}
	if err := sendPasswordReset(context.Background(), u, resetToken{ID: "t_7", URL: "https://app.acme.io/reset/t_7"}); err != nil {
		log.Print(err)
	}
}
