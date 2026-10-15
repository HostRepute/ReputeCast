package demo

import (
	"context"
)

// Transactional email (design expansion phase 1, brief §4): Acme Cloud,
// Acme's SaaS product, sends its product mail through the Acme Newsletter
// workspace.
func init() { seedTransactional = seedTransactionalImpl }

func seedTransactionalImpl(context.Context, *Seeder) error {
	return nil
}
