package app

import (
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/commerce"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/erasure"
	"github.com/hostrepute/reputecast/internal/gifts"
	"github.com/hostrepute/reputecast/internal/growth"
	"github.com/hostrepute/reputecast/internal/imports"
	"github.com/hostrepute/reputecast/internal/integrations"
	"github.com/hostrepute/reputecast/internal/offers"
	"github.com/hostrepute/reputecast/internal/polls"
	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/push"
	"github.com/hostrepute/reputecast/internal/recommendations"
	"github.com/hostrepute/reputecast/internal/referrals"
	"github.com/hostrepute/reputecast/internal/revenue"
	"github.com/hostrepute/reputecast/internal/scoring"
	"github.com/hostrepute/reputecast/internal/texting"
	"github.com/hostrepute/reputecast/internal/transactional"
	"github.com/hostrepute/reputecast/internal/verification"
	"github.com/hostrepute/reputecast/internal/website"
	"github.com/hostrepute/reputecast/internal/whatsapp"
)

// erasureSources is every package's personal data about contacts (review
// 2026-11 DP-1). TestErasureCoversPersonalData fails when a table with
// personal-data columns is in none of them and not exempted with a reason,
// so a new table can't be forgotten.
func erasureSources() []erasure.Source {
	return []erasure.Source{
		automations.Erasure(), commerce.Erasure(), earn.Erasure(), gifts.Erasure(), growth.Erasure(), imports.Erasure(),
		integrations.Erasure(), offers.Erasure(), polls.Erasure(), products.Erasure(), push.Erasure(), recommendations.Erasure(),
		referrals.Erasure(), revenue.Erasure(), scoring.Erasure(), texting.Erasure(), transactional.Erasure(), verification.Erasure(),
		website.Erasure(), whatsapp.Erasure(),
	}
}

// erasureRegistry is the registry erase and export walk, the audience's own
// source last.
func erasureRegistry() *erasure.Registry {
	return erasure.New(audience.CoreErasure(), erasureSources()...)
}
