package shopifytest

import (
	"slices"
	"strconv"
	"strings"
)

// Discount codes: the code discounts the app creates, their redeem codes
// added in bulk (a background job that answers "not done" for BulkPolls
// polls), deleted and deactivated. Mutations need write_discounts in
// Scopes, queries read_discounts, as Shopify's do.

// Discount is a code discount the app created.
type Discount struct {
	ID    string // numeric, as in gid://shopify/DiscountCodeNode/{ID}
	Kind  string // DiscountCodeBasic or DiscountCodeFreeShipping
	Title string
	// Input is the input the app sent (usage limit, value, items…).
	Input  map[string]any
	Active bool
	Codes  []*DiscountCode
}

// DiscountCode is one redeem code of a discount.
type DiscountCode struct {
	ID   string // numeric, as in gid://shopify/DiscountRedeemCode/{ID}
	Code string
}

type bulkCreation struct {
	id       string
	discount *Discount
	codes    []string
	polls    int
	done     bool
	results  []bulkResult
}

type bulkResult struct {
	code    string
	created *DiscountCode
	err     string
}

// BulkAdds lists the codes of every discountRedeemCodeBulkAdd call, in order.
func (f *Fake) BulkAdds() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([][]string, len(f.bulkAdds))
	for i, c := range f.bulkAdds {
		out[i] = slices.Clone(c)
	}
	return out
}

// Codes lists a discount's codes now.
func (f *Fake) Codes(discountID string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := f.discount(discountID)
	if d == nil {
		return nil
	}
	out := make([]string, len(d.Codes))
	for i, c := range d.Codes {
		out[i] = c.Code
	}
	return out
}

func (f *Fake) discount(id string) *Discount {
	id = numeric(id)
	for _, d := range f.Discounts {
		if d.ID == id {
			return d
		}
	}
	return nil
}

// taken reports whether a code is already one of the store's codes.
func (f *Fake) taken(code string) bool {
	for _, d := range f.Discounts {
		for _, c := range d.Codes {
			if strings.EqualFold(c.Code, code) {
				return true
			}
		}
	}
	return false
}

func (f *Fake) newID() string {
	id := strconv.Itoa(f.nextID)
	f.nextID++
	return id
}

func accessDenied(field string) []any {
	return []any{map[string]any{"message": "Access denied for " + field + " field. Required access: `write_discounts` access scope.",
		"extensions": map[string]any{"code": "ACCESS_DENIED"}}}
}

func discountError(field, code, msg string) []any {
	return []any{map[string]any{"field": []string{field}, "code": code, "message": msg}}
}

func codeNodes(codes []*DiscountCode) []any {
	out := []any{}
	for _, c := range codes {
		out = append(out, map[string]any{"id": gid("DiscountRedeemCode", c.ID), "code": c.Code})
	}
	return out
}

// discountAnswer answers the discount operations; ok is false for others.
func (f *Fake) discountAnswer(op string, vars map[string]any) (map[string]any, []any, bool) {
	write := slices.Contains(f.Scopes, "write_discounts")
	read := write || slices.Contains(f.Scopes, "read_discounts")
	switch op {
	case "DiscountBasicCreate", "DiscountFreeShippingCreate":
		field, kind := "discountCodeBasicCreate", "DiscountCodeBasic"
		if op == "DiscountFreeShippingCreate" {
			field, kind = "discountCodeFreeShippingCreate", "DiscountCodeFreeShipping"
		}
		if !write {
			return nil, accessDenied(field), true
		}
		in, _ := vars["input"].(map[string]any)
		code := str(in["code"])
		if code == "" || f.taken(code) {
			return map[string]any{field: map[string]any{"codeDiscountNode": nil,
				"userErrors": discountError("code", "TAKEN", "Code must be unique. Please try a different code.")}}, nil, true
		}
		d := &Discount{ID: f.newID(), Kind: kind, Title: str(in["title"]), Input: in, Active: true}
		d.Codes = []*DiscountCode{{ID: f.newID(), Code: code}}
		f.Discounts = append(f.Discounts, d)
		return map[string]any{field: map[string]any{"codeDiscountNode": map[string]any{"id": gid("DiscountCodeNode", d.ID),
			"codeDiscount": map[string]any{"codes": map[string]any{"nodes": codeNodes(d.Codes)}}}, "userErrors": []any{}}}, nil, true
	case "DiscountCodesAdd":
		const field = "discountRedeemCodeBulkAdd"
		if !write {
			return nil, accessDenied(field), true
		}
		d := f.discount(str(vars["discountId"]))
		if d == nil {
			return map[string]any{field: map[string]any{"bulkCreation": nil, "userErrors": discountError("discountId", "INVALID", "Discount does not exist")}}, nil, true
		}
		list, _ := vars["codes"].([]any)
		if len(list) > 250 {
			return map[string]any{field: map[string]any{"bulkCreation": nil, "userErrors": discountError("codes", "TOO_LONG", "Codes is too long (maximum is 250)")}}, nil, true
		}
		codes := make([]string, 0, len(list))
		for _, c := range list {
			m, _ := c.(map[string]any)
			codes = append(codes, str(m["code"]))
		}
		f.bulkAdds = append(f.bulkAdds, slices.Clone(codes))
		if f.bulks == nil {
			f.bulks = map[string]*bulkCreation{}
		}
		b := &bulkCreation{id: f.newID(), discount: d, codes: codes}
		f.bulks[b.id] = b
		return map[string]any{field: map[string]any{"bulkCreation": map[string]any{"id": gid("DiscountRedeemCodeBulkCreation", b.id)}, "userErrors": []any{}}}, nil, true
	case "DiscountCodesStatus":
		if !read {
			return nil, accessDenied("discountRedeemCodeBulkCreation"), true
		}
		b := f.bulks[numeric(str(vars["id"]))]
		if b == nil {
			return map[string]any{"discountRedeemCodeBulkCreation": nil}, nil, true
		}
		if !b.done && b.polls < f.BulkPolls {
			b.polls++
			return map[string]any{"discountRedeemCodeBulkCreation": map[string]any{"done": false,
				"codes": map[string]any{"nodes": []any{}, "pageInfo": map[string]any{"hasNextPage": false, "endCursor": nil}}}}, nil, true
		}
		if !b.done {
			// The job runs: a code the store has already, or twice in the
			// call, is refused.
			b.done = true
			for _, code := range b.codes {
				r := bulkResult{code: code}
				if code == "" || f.taken(code) {
					r.err = "Code must be unique. Please try a different code."
				} else {
					r.created = &DiscountCode{ID: f.newID(), Code: code}
					b.discount.Codes = append(b.discount.Codes, r.created)
				}
				b.results = append(b.results, r)
			}
		}
		nodes := []any{}
		for _, r := range b.results {
			n := map[string]any{"code": r.code, "errors": []any{}, "discountRedeemCode": nil}
			if r.created != nil {
				n["discountRedeemCode"] = map[string]any{"id": gid("DiscountRedeemCode", r.created.ID), "code": r.created.Code}
			} else {
				n["errors"] = discountError("code", "TAKEN", r.err)
			}
			nodes = append(nodes, n)
		}
		return map[string]any{"discountRedeemCodeBulkCreation": map[string]any{"done": true,
			"codes": map[string]any{"nodes": nodes, "pageInfo": map[string]any{"hasNextPage": false, "endCursor": nil}}}}, nil, true
	case "DiscountCodeLookup":
		if !read {
			return nil, accessDenied("codeDiscountNode"), true
		}
		d := f.discount(str(vars["id"]))
		if d == nil {
			return map[string]any{"codeDiscountNode": nil}, nil, true
		}
		q := strings.Trim(str(vars["query"]), `"`)
		var found []*DiscountCode
		for _, c := range d.Codes {
			if strings.EqualFold(c.Code, q) {
				found = append(found, c)
			}
		}
		return map[string]any{"codeDiscountNode": map[string]any{"codeDiscount": map[string]any{"codes": map[string]any{"nodes": codeNodes(found)}}}}, nil, true
	case "DiscountCodesDelete":
		const field = "discountCodeRedeemCodeBulkDelete"
		if !write {
			return nil, accessDenied(field), true
		}
		d := f.discount(str(vars["discountId"]))
		if d == nil {
			return map[string]any{field: map[string]any{"job": nil, "userErrors": discountError("discountId", "INVALID", "Discount does not exist")}}, nil, true
		}
		ids, _ := vars["ids"].([]any)
		gone := map[string]bool{}
		for _, id := range ids {
			gone[numeric(str(id))] = true
		}
		d.Codes = slices.DeleteFunc(d.Codes, func(c *DiscountCode) bool { return gone[c.ID] })
		return map[string]any{field: map[string]any{"job": map[string]any{"id": gid("Job", f.newID())}, "userErrors": []any{}}}, nil, true
	case "DiscountDeactivate":
		const field = "discountCodeDeactivate"
		if !write {
			return nil, accessDenied(field), true
		}
		d := f.discount(str(vars["id"]))
		if d == nil {
			return map[string]any{field: map[string]any{"codeDiscountNode": nil, "userErrors": discountError("id", "INVALID", "Discount does not exist")}}, nil, true
		}
		d.Active = false
		return map[string]any{field: map[string]any{"codeDiscountNode": map[string]any{"id": gid("DiscountCodeNode", d.ID)}, "userErrors": []any{}}}, nil, true
	}
	return nil, nil, false
}
