package coupons

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is the codes contacts were given: erasing a contact clears who
// got a code and in which message; the code stays given, so it's never
// handed out again. The data export lists the contact's codes, opened.
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "coupons",
		Tables: []erasure.Table{
			{Name: "coupon_codes", Action: "clear", Columns: []string{"contact_id", "message_id"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b, `UPDATE coupon_codes SET contact_id = NULL, message_id = NULL WHERE contact_id = ANY ($1)`)
		},
		Exports: []erasure.Export{
			{Section: "discountCodes", SQL: `SELECT p.name AS pool, c.code_sealed AS code, 'coupon_code:' || p.id::text AS code_ad, c.assigned_at, c.redeemed_at
				FROM coupon_codes c JOIN coupon_pools p ON p.id = c.pool_id WHERE c.contact_id = $1 ORDER BY c.assigned_at, c.id`,
				Sealed: map[string]string{"code": "code_ad"}},
		},
	}
}
