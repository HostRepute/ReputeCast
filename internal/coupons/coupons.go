// Package coupons is unique discount codes per recipient (M59 G-22): pools
// of codes uploaded or created in a connected Shopify or WooCommerce store,
// one code handed to each campaign message and automation email that shows
// the pool's code ({{ coupons.<slug>.code }}, or the Discount code block),
// store orders matched to the codes they used, and each pool's numbers.
//
// A code is handed out only when the sending engine sends (liquid.Allocating):
// one statement per pool and page, the same code again when a page is sent
// again, the contact's earlier code when the pool gives one per contact.
// Previews and test sends show the pool's prefix with SAMPLE1; view online
// and link redirects read the code the message was given.
//
// Codes are sealed at rest (secrets box, "coupon_code:<pool>") with a keyed
// hash of the upper-cased code for duplicates and order matching; lists
// never show them, and the CSV export (coupons:manage) is audited. Giving a
// code out isn't audited: the message holds it.
package coupons

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
)

// Sources of a pool's codes.
const (
	SourceUpload  = "upload"
	SourceShopify = "shopify"
	SourceWoo     = "woocommerce"
)

// Statuses of a pool.
const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

// What a store pool's discount takes off.
const (
	KindPercentage   = "percentage"
	KindFixedAmount  = "fixed_amount"
	KindFreeShipping = "free_shipping"
)

// Limits.
const (
	// MaxUploadBytes and MaxUploadCodes bound one upload of codes.
	MaxUploadBytes = 6 << 20
	MaxUploadCodes = 100_000
	// MaxGenerate is the most codes one request creates in a store.
	MaxGenerate = 50_000
)

// Root is the Liquid variable the pools' codes are under.
const Root = "coupons"

// Created codes are the pool's prefix and 8 characters without look-alikes
// (no 0, O, 1 or I): WELCOME-7KQ4M2XA.
const (
	codeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	codeLength   = 8
	// sampleSuffix follows the prefix in previews and test sends.
	sampleSuffix = "SAMPLE1"
)

// Service is the discount codes domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Log   *slog.Logger
	Now   func() time.Time
	// Jobs queues the jobs that create codes in stores and remove them
	// there; nil leaves them to the hourly job.
	Jobs func() *jobs.Client
	// Discounts opens a connected store's discount creator
	// (internal/commerce); it answers a problem saying why a store can't.
	Discounts func(ctx context.Context, ws *store.Workspace, storeID uuid.UUID) (stores.DiscountCreator, error)
	// Sources lists the workspace's stores and whether each can create
	// discount codes, from what ReputeCast knows (no call to the store).
	Sources func(ctx context.Context, ws *store.Workspace) ([]StoreSource, error)
}

// StoreSource is a connected store as a source of codes.
type StoreSource struct {
	StoreID                uuid.UUID
	Platform, Name, Domain string
	Currency               string
	Can                    bool
	Reason, Detail         string
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

func (s *Service) jobs() *jobs.Client {
	if s.Jobs == nil {
		return nil
	}
	return s.Jobs()
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// CouponPoolSettings are a pool's settings as stored (coupon_pools.settings) and
// shown: a store pool's discount, an upload pool's apply address.
type CouponPoolSettings struct {
	Kind          string   `json:"kind,omitempty" enum:"percentage,fixed_amount,free_shipping," doc:"Store pools: what the discount takes off"`
	Value         int64    `json:"value,omitempty" minimum:"0" doc:"Percent off (1–100) for percentage, cents off for fixed_amount"`
	MinimumCents  int64    `json:"minimumCents,omitempty" minimum:"0" doc:"Store pools: the smallest order the code works on, in cents"`
	CollectionIDs []string `json:"collectionIds,omitempty" maxItems:"50" doc:"Store pools: the collections the discount applies to; none is the whole store"`
	EndsDays      int      `json:"endsDays,omitempty" minimum:"0" maximum:"365" doc:"Store pools: the discount ends this many days after the pool was created; 0 never"`
	ApplyURL      string   `json:"applyUrl,omitempty" maxLength:"500" doc:"Upload pools: the address that applies a code, with {code} where it goes"`
}

func settingsOf(raw []byte) CouponPoolSettings {
	var s CouponPoolSettings
	_ = json.Unmarshal(raw, &s)
	if s.CollectionIDs == nil {
		s.CollectionIDs = []string{}
	}
	return s
}

// endsAt is when a store pool's discount ends (nil: never).
func endsAt(p store.CouponPoolsRow, set CouponPoolSettings) *time.Time {
	if set.EndsDays <= 0 {
		return nil
	}
	t := p.CreatedAt.AddDate(0, 0, set.EndsDays)
	return &t
}

// codeAD is the associated data a pool's codes are sealed with.
func codeAD(pool uuid.UUID) string { return "coupon_code:" + pool.String() }

// pendingAD is the associated data of a batch's codes under way.
func pendingAD(batch uuid.UUID) string { return "coupon_batch:" + batch.String() + ":pending" }

// CodeHash is the keyed hash of a code within a workspace, upper-cased: a
// pool keeps a code once, and a store order finds the code it used by it.
func CodeHash(ws uuid.UUID, code string) []byte {
	m := hmac.New(sha256.New, []byte("reputecast-coupon-codes:"+ws.String()))
	m.Write([]byte(strings.ToUpper(strings.TrimSpace(code))))
	return m.Sum(nil)
}

// newCodes are n codes with the prefix and codeLength random characters
// (crypto/rand; 32 characters, so each byte maps without bias).
func newCodes(prefix string, n int) []string {
	out := make([]string, 0, n)
	buf := make([]byte, codeLength*n)
	_, _ = rand.Read(buf)
	var b strings.Builder
	for i := range n {
		b.Reset()
		b.WriteString(prefix)
		for _, c := range buf[i*codeLength : (i+1)*codeLength] {
			b.WriteByte(codeAlphabet[c&31])
		}
		out = append(out, b.String())
	}
	return out
}

// sample is the code previews and test sends show.
func sample(prefix string) string { return prefix + sampleSuffix }

// actorUser is the person behind a request (a key's creator for keys).
func actorUser(p *authz.Principal) *uuid.UUID {
	if p == nil {
		return nil
	}
	id := p.UserID
	if id == uuid.Nil && p.Type == authz.ActorUser {
		id = p.ID
	}
	if id == uuid.Nil {
		return nil
	}
	return &id
}
