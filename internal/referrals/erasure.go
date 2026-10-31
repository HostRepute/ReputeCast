package referrals

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is the contact's referral link, hub and rewards (with any postal
// address) and the network details of their own referral; the referrals
// they made, and their own, stay counted without an address (design
// expansion X15; review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "referrals",
		Tables: []erasure.Table{
			{Name: "referral_rewards", Action: "delete", Columns: []string{"address"}},
			{Name: "referral_links", Action: "delete"},
			{Name: "referral_referrer_ips", Action: "delete", Columns: []string{"ip"}},
			{Name: "referrals", Action: "clear", Columns: []string{"ip", "ip_network"},
				Kept: map[string]string{"contact_id": "the referrer's count and the program's totals; the contact is scrubbed"}},
			{Name: "referral_events", Action: "clear", Columns: []string{"detail"},
				Kept: map[string]string{"contact_id": "the program's activity counts; the contact is scrubbed"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`DELETE FROM referral_rewards WHERE contact_id = ANY ($1)`,
				`DELETE FROM referral_links WHERE contact_id = ANY ($1)`,
				`UPDATE referrals SET ip = NULL, ip_network = '' WHERE contact_id = ANY ($1)`,
				`UPDATE referral_events SET detail = '{}' WHERE contact_id = ANY ($1) AND detail <> '{}'`,
			)
		},
		Exports: []erasure.Export{
			{Section: "referralLink", SQL: `SELECT code, status, visits, signups, created_at FROM referral_links WHERE contact_id = $1`},
			{Section: "referralRewards", SQL: `SELECT reward_name, reward_type, status, reached_at, sent_at, address FROM referral_rewards WHERE contact_id = $1 ORDER BY reached_at`},
		},
	}
}
