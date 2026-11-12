package coupons

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/store"
)

// usedSlugs are the pools a template's Liquid names (coupons.<slug>.…),
// in order of first use. Blocks must be expanded first.
func usedSlugs(src string) []string {
	var out []string
	for _, p := range liquid.UsedPaths(src, Root) {
		slug, _, _ := strings.Cut(p, ".")
		if slugRe.MatchString(slug) && !contains(out, slug) {
			out = append(out, slug)
		}
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// codeURL is the address that applies a code: Shopify's /discount/<code>
// on the storefront, a WooCommerce store's address, an upload pool's apply
// address with the code in it ("" without one).
func codeURL(p store.CouponPoolsRow, code string) string {
	switch p.Source {
	case SourceShopify:
		if h := storeHost(p); h != "" {
			return "https://" + h + "/discount/" + url.PathEscape(code)
		}
	case SourceWoo:
		if h := storeHost(p); h != "" {
			return "https://" + h + "/"
		}
	default:
		if a := settingsOf(p.Settings).ApplyURL; a != "" {
			return strings.ReplaceAll(a, "{code}", url.PathEscape(code))
		}
	}
	return ""
}

// Roots implements liquid.RecipientData.
func (s *Service) Roots() []string { return []string{Root} }

// Vars implements liquid.RecipientData: coupons.<slug>.code and .url for
// every pool the template names. A real send (liquid.Allocating) hands
// each message its code, one statement per pool; a reader left without
// one gets liquid.ShortKey and the pool's low alert fires. Otherwise a
// message reads the code it was given, and a render without a message
// (preview, test send) shows the prefix and SAMPLE1.
func (s *Service) Vars(ctx context.Context, wsID uuid.UUID, recipients []liquid.Recipient, _ []string) (map[uuid.UUID]map[string]any, error) {
	out := map[uuid.UUID]map[string]any{}
	slugs := usedSlugs(liquid.SourceFrom(ctx))
	if len(slugs) == 0 || len(recipients) == 0 {
		return out, nil
	}
	allocating := liquid.Allocating(ctx)
	pools, err := s.pools(ctx, wsID, nil, slugs)
	if err != nil {
		if allocating {
			s.log().WarnContext(ctx, "coupons: read the pools", "err", err)
			for _, r := range recipients {
				short(out, r.ContactID, "unavailable", "")
			}
			return out, nil
		}
		return out, err
	}
	for _, p := range pools {
		codes, err := s.codesFor(ctx, p, recipients, allocating)
		if err != nil {
			s.log().WarnContext(ctx, "coupons: give out codes", "pool", p.ID, "err", err)
			if !allocating {
				continue
			}
			for _, r := range recipients {
				short(out, r.ContactID, "unavailable", "")
			}
			continue
		}
		ran := false
		for _, r := range recipients {
			if r.ContactID == uuid.Nil {
				continue
			}
			code, ok := codes(r)
			if !ok {
				if allocating && r.MessageID != uuid.Nil {
					short(out, r.ContactID, liquid.ShortNoCodes, "the discount code pool "+p.Name)
					ran = true
				}
				continue
			}
			v := map[string]any{"code": code}
			if u := codeURL(p, code); u != "" {
				v["url"] = u
			}
			reader(out, r.ContactID)[Root].(map[string]any)[p.Slug] = v
		}
		if ran {
			free, _ := s.Q.CouponPoolFree(ctx, p.ID)
			s.alertLow(ctx, p, free)
		}
	}
	return out, nil
}

// reader is a reader's variables, with their coupons map.
func reader(out map[uuid.UUID]map[string]any, contact uuid.UUID) map[string]any {
	v := out[contact]
	if v == nil {
		v = map[string]any{}
		out[contact] = v
	}
	if _, ok := v[Root]; !ok {
		v[Root] = map[string]any{}
	}
	return v
}

// short marks a reader a pool couldn't serve; no_codes outranks a pool
// that couldn't be read.
func short(out map[uuid.UUID]map[string]any, contact uuid.UUID, why, what string) {
	if contact == uuid.Nil {
		return
	}
	v := reader(out, contact)
	if v[liquid.ShortKey] == liquid.ShortNoCodes {
		return
	}
	v[liquid.ShortKey] = why
	if what != "" {
		v[liquid.ShortWhat] = what
	}
}

// codesFor gives each reader of a page the pool's code (see Vars).
func (s *Service) codesFor(ctx context.Context, p store.CouponPoolsRow, recipients []liquid.Recipient, allocating bool) (func(liquid.Recipient) (string, bool), error) {
	msgs := make([]uuid.UUID, 0, len(recipients))
	contacts := make([]uuid.UUID, 0, len(recipients))
	for _, r := range recipients {
		if r.MessageID != uuid.Nil && r.ContactID != uuid.Nil {
			msgs, contacts = append(msgs, r.MessageID), append(contacts, r.ContactID)
		}
	}
	byMsg := map[uuid.UUID]string{}
	byContact := map[uuid.UUID]string{}
	open := func(sealed []byte) (string, bool) {
		c, err := s.Box.OpenString(sealed, codeAD(p.ID))
		return c, err == nil
	}
	if len(msgs) > 0 {
		if allocating {
			// A code another send took between this statement's snapshot
			// and its lock is skipped rather than shared, so a page can come
			// back short while free codes remain: the messages left over
			// ask again once (the codes they already hold come back as they
			// are).
			for try := 0; try < 2 && len(byMsg) < len(msgs); try++ {
				rows, err := s.Q.CouponAllocate(ctx, store.CouponAllocateParams{MessageIds: msgs, ContactIds: contacts, PoolID: p.ID,
					OnePerContact: p.OnePerContact, Take: p.Status == StatusActive, Now: s.now()})
				if err != nil {
					return nil, err
				}
				for _, r := range rows {
					if c, ok := open(r.CodeSealed); ok {
						byMsg[r.MessageID] = c
					}
				}
				if p.Status != StatusActive {
					break
				}
			}
		} else {
			rows, err := s.Q.CouponCodesOfMessages(ctx, store.CouponCodesOfMessagesParams{PoolID: p.ID, MessageIds: msgs, ContactIds: contacts, OnePerContact: p.OnePerContact})
			if err != nil {
				return nil, err
			}
			want := map[uuid.UUID]bool{}
			for _, m := range msgs {
				want[m] = true
			}
			for _, r := range rows {
				c, ok := open(r.CodeSealed)
				if !ok {
					continue
				}
				if r.MessageID != nil && want[*r.MessageID] {
					byMsg[*r.MessageID] = c
				}
				if r.ContactID != nil {
					if _, seen := byContact[*r.ContactID]; !seen {
						byContact[*r.ContactID] = c
					}
				}
			}
		}
	}
	return func(r liquid.Recipient) (string, bool) {
		if r.MessageID == uuid.Nil {
			if allocating {
				return "", false
			}
			return sample(p.Prefix), true
		}
		if c, ok := byMsg[r.MessageID]; ok {
			return c, true
		}
		if !allocating && p.OnePerContact {
			c, ok := byContact[r.ContactID]
			return c, ok
		}
		return "", false
	}, nil
}

// ComposerTokens are the active pools' tokens for the composer's token
// menu, labelled in the member's language.
func (s *Service) ComposerTokens(ctx context.Context, ws *store.Workspace) []liquid.PersonalizationToken {
	rows, err := s.pools(ctx, ws.ID, nil, nil)
	if err != nil {
		s.log().WarnContext(ctx, "coupons: the composer's tokens", "err", err)
		return nil
	}
	l := i18n.LangFrom(ctx)
	var out []liquid.PersonalizationToken
	for _, p := range rows {
		if p.Status != StatusActive {
			continue
		}
		args := i18n.Args{"name": p.Name}
		out = append(out,
			liquid.PersonalizationToken{Key: Root + "." + p.Slug + ".code", Label: mTokenCode.Text(l, args), LabelText: mTokenCode.Keep(args), Kind: "coupon"},
			liquid.PersonalizationToken{Key: Root + "." + p.Slug + ".url", Label: mTokenURL.Text(l, args), LabelText: mTokenURL.Keep(args), Kind: "coupon"})
	}
	return out
}

// Redeemed implements commerce.Service.Redeemed: each code an order used
// redeems the first matching code of the workspace, once per order.
func (s *Service) Redeemed(ctx context.Context, q *store.Queries, wsID, orderID uuid.UUID, codes []string, at time.Time) error {
	hashes := make([][]byte, 0, len(codes))
	seen := map[string]bool{}
	for _, c := range codes {
		c = strings.TrimSpace(c)
		if c == "" || len(c) > 255 {
			continue
		}
		h := CodeHash(wsID, c)
		if !seen[string(h)] {
			seen[string(h)] = true
			hashes = append(hashes, h)
		}
	}
	if len(hashes) == 0 {
		return nil
	}
	_, err := q.CouponRedeem(ctx, store.CouponRedeemParams{WorkspaceID: wsID, Hashes: hashes, OrderID: orderID, At: at})
	return err
}
