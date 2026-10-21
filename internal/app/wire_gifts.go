package app

import (
	"context"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/gifts"
	"github.com/hostrepute/reputecast/internal/store"
)

// Gift subscriptions (design expansion phase 6, X17): gift options, the
// hosted gift and redeem pages, delivery and reminders, and Earn › Gifts.

var giftServices sync.Map // *App → *gifts.Service

func init() {
	// The gifts take Earn's one-time payment events.
	earnHooks = append(earnHooks, func(a *App, e *earn.Service) {
		v, _ := giftServices.LoadOrStore(a, &gifts.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Earn: e, Log: a.Log, Signup: giftSignup{a},
			Confirm: func(ctx context.Context, ws *store.Workspace, contact uuid.UUID) error {
				_, err := a.audience().SendConfirmations(ctx, ws, []uuid.UUID{contact})
				return err
			}})
		e.OneTime = v.(*gifts.Service)
	})
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		gifts.Register(h, giftService(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		gifts.RegisterJobs(a.JobReg, giftService(a))
	})
}

func giftService(a *App) *gifts.Service {
	earnService(a) // makes the gifts service with it
	v, _ := giftServices.Load(a)
	return v.(*gifts.Service)
}

// Gifts is the gifts domain (tests set its clock).
func (a *App) Gifts() *gifts.Service { return giftService(a) }

// giftSignup joins a gift's recipient to the tier's lists, with the
// redeem page's text as the consent evidence.
type giftSignup struct{ a *App }

// SourceGiftRedeem is the consent source of a redeemed gift.
const SourceGiftRedeem = "gift_redeem"

func (s giftSignup) Join(ctx context.Context, tx pgx.Tx, ws *store.Workspace, j gifts.Join) (uuid.UUID, bool, error) {
	res, err := s.a.audience().Signup(ctx, tx, ws, audience.SignupInput{Email: j.Email, FirstName: j.FirstName, LastName: j.LastName, Lists: j.Lists,
		Evidence: audience.PublicEvidence{Source: SourceGiftRedeem, IP: j.IP, UserAgent: j.UserAgent, ConsentText: j.ConsentText, Language: "en",
			Note: j.Note, At: j.At}})
	if err != nil {
		return uuid.Nil, false, err
	}
	if res.Blocked {
		return res.ContactID, false, nil
	}
	return res.ContactID, res.NeedsConfirmation(), nil
}
