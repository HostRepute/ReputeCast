package woocommerce

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/stores"
)

// Discount codes (M59 G-22): WooCommerce has no discount above its coupons,
// so each code is a coupon of its own (discount ids are ""), made 100 at a
// time with the coupons batch endpoint and confirmed at once. Each coupon
// works once, once per customer, alone (individual use). The key needs
// write access: a read-only key is refused as stores.ErrPermission.

// maxBatch is the batch endpoint's limit.
const maxBatch = 100

var _ stores.DiscountCreator = (*Client)(nil)

// MaxCodesPerCall implements stores.DiscountCreator.
func (c *Client) MaxCodesPerCall() int { return maxBatch }

// coupon is one code's coupon.
func coupon(spec stores.DiscountSpec, code string) (map[string]any, error) {
	in := map[string]any{"code": code, "description": spec.Title, "individual_use": true, "usage_limit": 1, "usage_limit_per_user": 1}
	switch spec.Kind {
	case stores.DiscountPercentage:
		if spec.Value < 1 || spec.Value > 100 {
			return nil, fmt.Errorf("woocommerce: a percentage of %d", spec.Value)
		}
		in["discount_type"], in["amount"] = "percent", strconv.FormatInt(spec.Value, 10)
	case stores.DiscountFixedAmount:
		if spec.Value < 1 {
			return nil, fmt.Errorf("woocommerce: an amount of %d", spec.Value)
		}
		in["discount_type"], in["amount"] = "fixed_cart", stores.FormatAmount(spec.Value, spec.Currency)
	case stores.DiscountFreeShipping:
		in["discount_type"], in["amount"], in["free_shipping"] = "fixed_cart", "0", true
	default:
		return nil, fmt.Errorf("woocommerce: unknown discount kind %q", spec.Kind)
	}
	if spec.MinimumCents > 0 {
		in["minimum_amount"] = stores.FormatAmount(spec.MinimumCents, spec.Currency)
	}
	if spec.EndsAt != nil {
		in["date_expires_gmt"] = spec.EndsAt.UTC().Format("2006-01-02T15:04:05")
	}
	if len(spec.CollectionIDs) > 0 && spec.Kind != stores.DiscountFreeShipping {
		cats := make([]int64, 0, len(spec.CollectionIDs))
		for _, id := range spec.CollectionIDs {
			n, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("woocommerce: category %q", id)
			}
			cats = append(cats, n)
		}
		in["product_categories"] = cats
	}
	return in, nil
}

// writeRefused turns a refused write into the contract's missing access:
// WooCommerce answers a read-only key 401 (or 403) on writes.
func writeRefused(err error) error {
	if errors.Is(err, stores.ErrUnauthorized) && !errors.Is(err, stores.ErrPermission) {
		return fmt.Errorf("%w: %w", stores.ErrPermission, err)
	}
	return err
}

type batchItem struct {
	ID    int64  `json:"id"`
	Code  string `json:"code"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// create makes one coupon per code; the answer lists them in the request's
// order.
func (c *Client) create(ctx context.Context, spec stores.DiscountSpec, codes []string) (stores.DiscountCodes, error) {
	out := stores.DiscountCodes{Done: true, IDs: map[string]string{}, Failed: []string{}}
	if len(codes) == 0 || len(codes) > maxBatch {
		return out, fmt.Errorf("woocommerce: %d coupons in one batch; 1 to %d", len(codes), maxBatch)
	}
	create := make([]map[string]any, len(codes))
	for i, code := range codes {
		in, err := coupon(spec, code)
		if err != nil {
			return out, err
		}
		create[i] = in
	}
	var res struct {
		Create []batchItem `json:"create"`
	}
	if _, err := c.call(ctx, http.MethodPost, "/coupons/batch", nil, map[string]any{"create": create}, &res); err != nil {
		return out, writeRefused(err)
	}
	for i, code := range codes {
		if i < len(res.Create) && res.Create[i].Error == nil && res.Create[i].ID > 0 {
			out.IDs[code] = strconv.FormatInt(res.Create[i].ID, 10)
		} else {
			out.Failed = append(out.Failed, code)
		}
	}
	return out, nil
}

// CreateDiscount implements stores.DiscountCreator: the first code's
// coupon (there is no discount above it).
func (c *Client) CreateDiscount(ctx context.Context, spec stores.DiscountSpec, firstCode string) (string, string, error) {
	got, err := c.create(ctx, spec, []string{firstCode})
	if err != nil {
		return "", "", err
	}
	id, ok := got.IDs[firstCode]
	if !ok {
		return "", "", fmt.Errorf("woocommerce: the coupon %s wasn't created (the code is taken)", firstCode)
	}
	return "", id, nil
}

// AddCodes implements stores.DiscountCreator: one batch, confirmed at once.
func (c *Client) AddCodes(ctx context.Context, _ string, spec stores.DiscountSpec, codes []string) (stores.DiscountCodes, error) {
	return c.create(ctx, spec, codes)
}

// CodesStatus implements stores.DiscountCreator: batches finish in their
// call, so there is nothing to follow.
func (c *Client) CodesStatus(context.Context, string, string, []string) (stores.DiscountCodes, error) {
	return stores.DiscountCodes{Done: true, IDs: map[string]string{}, Failed: []string{}}, nil
}

// RemoveCodes implements stores.DiscountCreator: the coupons (found by code
// when their id isn't known) are deleted for good, 100 a batch.
func (c *Client) RemoveCodes(ctx context.Context, _ string, codes []string, ids []string) error {
	var del []int64
	for i, code := range codes {
		id := ""
		if i < len(ids) {
			id = ids[i]
		}
		if id == "" {
			var found []struct {
				ID   int64  `json:"id"`
				Code string `json:"code"`
			}
			if _, err := c.call(ctx, http.MethodGet, "/coupons", url.Values{"code": {code}}, nil, &found); err != nil {
				return err
			}
			for _, f := range found {
				if strings.EqualFold(f.Code, code) {
					id = strconv.FormatInt(f.ID, 10)
				}
			}
		}
		if n, err := strconv.ParseInt(id, 10, 64); err == nil && n > 0 {
			del = append(del, n)
		}
	}
	for start := 0; start < len(del); start += maxBatch {
		var res struct {
			Delete []batchItem `json:"delete"`
		}
		if _, err := c.call(ctx, http.MethodPost, "/coupons/batch", nil, map[string]any{"delete": del[start:min(start+maxBatch, len(del))]}, &res); err != nil {
			return writeRefused(err)
		}
	}
	return nil
}

// DisableDiscount implements stores.DiscountCreator. There is no discount
// above the coupons: their codes stop working when RemoveCodes deletes them.
func (c *Client) DisableDiscount(_ context.Context, discountID string) error {
	if discountID != "" {
		return fmt.Errorf("woocommerce: %w: no discount %q above coupons", stores.ErrNotFound, discountID)
	}
	return nil
}
