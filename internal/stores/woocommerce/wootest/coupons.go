package wootest

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

// Coupon is a coupon on the fake site, as the key created it.
type Coupon struct {
	ID    int64
	Code  string         // lower-cased, as WooCommerce keeps codes
	Input map[string]any // what the key sent (discount_type, usage limits…)
}

// Batches lists the codes of every coupons batch that created coupons.
func (f *Fake) Batches() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([][]string, len(f.batches))
	for i, b := range f.batches {
		out[i] = slices.Clone(b)
	}
	return out
}

// CouponCodes lists the coupons' codes.
func (f *Fake) CouponCodes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.Coupons))
	for i, c := range f.Coupons {
		out[i] = c.Code
	}
	return out
}

// coupons answers GET /coupons?code= and POST /coupons/batch (create up to
// 100, delete by id). A read-only key can't write. f.mu is held.
func (f *Fake) coupons(w http.ResponseWriter, r *http.Request, path string) {
	if path == "/coupons" && r.Method == http.MethodGet {
		code := strings.ToLower(r.URL.Query().Get("code"))
		list := []any{}
		for _, c := range f.Coupons {
			if code == "" || c.Code == code {
				list = append(list, map[string]any{"id": c.ID, "code": c.Code})
			}
		}
		writeJSON(w, http.StatusOK, list)
		return
	}
	if path != "/coupons/batch" || r.Method != http.MethodPost {
		errJSON(w, http.StatusNotFound, "rest_no_route", "No route was found matching the URL and request method.")
		return
	}
	if f.ReadOnly {
		errJSON(w, http.StatusUnauthorized, "woocommerce_rest_authentication_error", "The API key provided does not have write permissions.")
		return
	}
	var in struct {
		Create []map[string]any `json:"create"`
		Delete []int64          `json:"delete"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	if len(in.Create)+len(in.Delete) > 100 {
		errJSON(w, http.StatusRequestEntityTooLarge, "woocommerce_rest_request_entity_too_large", "Unable to accept more than 100 items for this request.")
		return
	}
	out := map[string]any{}
	if len(in.Create) > 0 {
		created := []any{}
		var codes []string
		for _, c := range in.Create {
			code, _ := c["code"].(string)
			codes = append(codes, code)
			code = strings.ToLower(code)
			if code == "" || slices.ContainsFunc(f.Coupons, func(x *Coupon) bool { return x.Code == code }) {
				created = append(created, map[string]any{"id": 0, "error": map[string]any{"code": "woocommerce_rest_coupon_code_already_exists",
					"message": "The coupon code already exists", "data": map[string]any{"status": 400}}})
				continue
			}
			f.nextID++
			f.Coupons = append(f.Coupons, &Coupon{ID: f.nextID, Code: code, Input: c})
			created = append(created, map[string]any{"id": f.nextID, "code": code})
		}
		f.batches = append(f.batches, codes)
		out["create"] = created
	}
	if len(in.Delete) > 0 {
		deleted := []any{}
		for _, id := range in.Delete {
			before := len(f.Coupons)
			f.Coupons = slices.DeleteFunc(f.Coupons, func(c *Coupon) bool { return c.ID == id })
			if len(f.Coupons) == before {
				deleted = append(deleted, map[string]any{"id": id, "error": map[string]any{"code": "woocommerce_rest_shop_coupon_invalid_id", "message": "Invalid ID.",
					"data": map[string]any{"status": 404}}})
				continue
			}
			deleted = append(deleted, map[string]any{"id": id})
		}
		out["delete"] = deleted
	}
	writeJSON(w, http.StatusOK, out)
}
