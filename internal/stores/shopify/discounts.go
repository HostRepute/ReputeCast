package shopify

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/stores"
)

// Discount codes (M59 G-22): one code discount (amount off or free
// shipping) whose every code works once and once per customer (Shopify
// counts usageLimit per redeem code), codes added 250 at a time in
// Shopify's background job and followed until it's done. They need the
// optional scopes write_discounts and read_discounts.

// DiscountScopes are the scopes "Allow discount codes" asks for.
var DiscountScopes = []string{"write_discounts", "read_discounts"}

// maxBulkCodes is discountRedeemCodeBulkAdd's limit.
const maxBulkCodes = 250

var _ stores.DiscountCreator = (*Client)(nil)

// discountCreate is a create mutation of a code discount, returning the
// discount and its first code.
func discountCreate(op, field, input, arg, kind string) string {
	return `mutation ` + op + `($input: ` + input + `!) {
  ` + field + `(` + arg + `: $input) {
    codeDiscountNode { id codeDiscount { ... on ` + kind + ` { codes(first: 1) { nodes { id code } } } } }
    userErrors { field code message }
  }
}`
}

var (
	mDiscountBasic        = discountCreate("DiscountBasicCreate", "discountCodeBasicCreate", "DiscountCodeBasicInput", "basicCodeDiscount", "DiscountCodeBasic")
	mDiscountFreeShipping = discountCreate("DiscountFreeShippingCreate", "discountCodeFreeShippingCreate", "DiscountCodeFreeShippingInput", "freeShippingCodeDiscount", "DiscountCodeFreeShipping")
)

const (
	mDiscountCodesAdd = `mutation DiscountCodesAdd($discountId: ID!, $codes: [DiscountRedeemCodeInput!]!) {
  discountRedeemCodeBulkAdd(discountId: $discountId, codes: $codes) { bulkCreation { id } userErrors { field code message } }
}`

	qDiscountCodesStatus = `query DiscountCodesStatus($id: ID!, $after: String) {
  discountRedeemCodeBulkCreation(id: $id) {
    done
    codes(first: 250, after: $after) {
      nodes { code errors { field code message } discountRedeemCode { id code } }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

	qDiscountCodeLookup = `query DiscountCodeLookup($id: ID!, $query: String!) {
  codeDiscountNode(id: $id) {
    codeDiscount {
      ... on DiscountCodeBasic { codes(first: 5, query: $query) { nodes { id code } } }
      ... on DiscountCodeFreeShipping { codes(first: 5, query: $query) { nodes { id code } } }
    }
  }
}`

	mDiscountCodesDelete = `mutation DiscountCodesDelete($discountId: ID!, $ids: [ID!]) {
  discountCodeRedeemCodeBulkDelete(discountId: $discountId, ids: $ids) { job { id } userErrors { field code message } }
}`

	mDiscountDeactivate = `mutation DiscountDeactivate($id: ID!) {
  discountCodeDeactivate(id: $id) { codeDiscountNode { id } userErrors { field code message } }
}`
)

type redeemCode struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

type discountNode struct {
	ID           string `json:"id"`
	CodeDiscount struct {
		Codes struct {
			Nodes []redeemCode `json:"nodes"`
		} `json:"codes"`
	} `json:"codeDiscount"`
}

type discountPayload struct {
	CodeDiscountNode *discountNode `json:"codeDiscountNode"`
	UserErrors       []userError   `json:"userErrors"`
}

// MaxCodesPerCall implements stores.DiscountCreator.
func (c *Client) MaxCodesPerCall() int { return maxBulkCodes }

// discountInput is the discount's input: every code works once, for one
// customer, from now until spec.EndsAt.
func (c *Client) discountInput(spec stores.DiscountSpec, code string) (map[string]any, error) {
	in := map[string]any{
		"title": spec.Title, "code": code, "startsAt": c.now().UTC().Format(time.RFC3339),
		"context": map[string]any{"all": "ALL"}, "usageLimit": 1, "appliesOncePerCustomer": true,
	}
	if spec.EndsAt != nil {
		in["endsAt"] = spec.EndsAt.UTC().Format(time.RFC3339)
	}
	if spec.MinimumCents > 0 {
		in["minimumRequirement"] = map[string]any{"subtotal": map[string]any{"greaterThanOrEqualToSubtotal": stores.FormatAmount(spec.MinimumCents, spec.Currency)}}
	}
	switch spec.Kind {
	case stores.DiscountFreeShipping:
		in["destination"] = map[string]any{"all": true}
		return in, nil
	case stores.DiscountPercentage:
		if spec.Value < 1 || spec.Value > 100 {
			return nil, fmt.Errorf("shopify: a percentage of %d", spec.Value)
		}
		in["customerGets"] = map[string]any{"value": map[string]any{"percentage": float64(spec.Value) / 100}, "items": items(spec.CollectionIDs)}
	case stores.DiscountFixedAmount:
		if spec.Value < 1 {
			return nil, fmt.Errorf("shopify: an amount of %d", spec.Value)
		}
		in["customerGets"] = map[string]any{"value": map[string]any{"discountAmount": map[string]any{"amount": stores.FormatAmount(spec.Value, spec.Currency), "appliesOnEachItem": false}},
			"items": items(spec.CollectionIDs)}
	default:
		return nil, fmt.Errorf("shopify: unknown discount kind %q", spec.Kind)
	}
	return in, nil
}

// items are the products a discount applies to: its collections, or all.
func items(collections []string) map[string]any {
	if len(collections) == 0 {
		return map[string]any{"all": true}
	}
	ids := make([]string, len(collections))
	for i, id := range collections {
		ids[i] = gid("Collection", id)
	}
	return map[string]any{"collections": map[string]any{"add": ids}}
}

// CreateDiscount implements stores.DiscountCreator: discountCodeBasicCreate,
// or discountCodeFreeShippingCreate for free shipping.
func (c *Client) CreateDiscount(ctx context.Context, spec stores.DiscountSpec, firstCode string) (string, string, error) {
	in, err := c.discountInput(spec, firstCode)
	if err != nil {
		return "", "", err
	}
	op, query, field := "DiscountBasicCreate", mDiscountBasic, "discountCodeBasicCreate"
	if spec.Kind == stores.DiscountFreeShipping {
		op, query, field = "DiscountFreeShippingCreate", mDiscountFreeShipping, "discountCodeFreeShippingCreate"
	}
	var d map[string]discountPayload
	if err := c.gql(ctx, op, query, map[string]any{"input": in}, &d); err != nil {
		return "", "", err
	}
	p := d[field]
	if err := userErrors(field, p.UserErrors); err != nil {
		return "", "", err
	}
	if p.CodeDiscountNode == nil {
		return "", "", fmt.Errorf("shopify: %s returned no discount", field)
	}
	codeID := ""
	for _, n := range p.CodeDiscountNode.CodeDiscount.Codes.Nodes {
		if strings.EqualFold(n.Code, firstCode) {
			codeID = n.ID
		}
	}
	return p.CodeDiscountNode.ID, codeID, nil
}

// AddCodes implements stores.DiscountCreator: discountRedeemCodeBulkAdd,
// which adds the codes in the background (CodesStatus follows it).
func (c *Client) AddCodes(ctx context.Context, discountID string, _ stores.DiscountSpec, codes []string) (stores.DiscountCodes, error) {
	if len(codes) == 0 || len(codes) > maxBulkCodes {
		return stores.DiscountCodes{}, fmt.Errorf("shopify: %d codes in one call; 1 to %d", len(codes), maxBulkCodes)
	}
	in := make([]map[string]string, len(codes))
	for i, code := range codes {
		in[i] = map[string]string{"code": code}
	}
	var d struct {
		DiscountRedeemCodeBulkAdd struct {
			BulkCreation *struct {
				ID string `json:"id"`
			} `json:"bulkCreation"`
			UserErrors []userError `json:"userErrors"`
		} `json:"discountRedeemCodeBulkAdd"`
	}
	if err := c.gql(ctx, "DiscountCodesAdd", mDiscountCodesAdd, map[string]any{"discountId": gid("DiscountCodeNode", discountID), "codes": in}, &d); err != nil {
		return stores.DiscountCodes{}, err
	}
	p := d.DiscountRedeemCodeBulkAdd
	if err := userErrors("discountRedeemCodeBulkAdd", p.UserErrors); err != nil {
		return stores.DiscountCodes{}, err
	}
	if p.BulkCreation == nil {
		return stores.DiscountCodes{}, errors.New("shopify: discountRedeemCodeBulkAdd returned no job")
	}
	return stores.DiscountCodes{JobID: p.BulkCreation.ID, IDs: map[string]string{}, Failed: []string{}}, nil
}

// CodesStatus implements stores.DiscountCreator: the bulk creation's state
// and, once it's done, each code's id or why it was refused.
func (c *Client) CodesStatus(ctx context.Context, _, jobID string, codes []string) (stores.DiscountCodes, error) {
	out := stores.DiscountCodes{JobID: jobID, IDs: map[string]string{}, Failed: []string{}}
	asked := make(map[string]string, len(codes))
	for _, code := range codes {
		asked[strings.ToUpper(code)] = code
	}
	after := ""
	for {
		var d struct {
			Bulk *struct {
				Done  bool `json:"done"`
				Codes struct {
					Nodes []struct {
						Code    string      `json:"code"`
						Errors  []userError `json:"errors"`
						Created *redeemCode `json:"discountRedeemCode"`
					} `json:"nodes"`
					PageInfo pageInfo `json:"pageInfo"`
				} `json:"codes"`
			} `json:"discountRedeemCodeBulkCreation"`
		}
		vars := map[string]any{"id": gid("DiscountRedeemCodeBulkCreation", jobID)}
		if after != "" {
			vars["after"] = after
		}
		if err := c.gql(ctx, "DiscountCodesStatus", qDiscountCodesStatus, vars, &d); err != nil {
			return out, err
		}
		if d.Bulk == nil {
			return out, fmt.Errorf("%w: bulk creation %s", stores.ErrNotFound, jobID)
		}
		if !d.Bulk.Done {
			return out, nil
		}
		for _, n := range d.Bulk.Codes.Nodes {
			code := n.Code
			if v, ok := asked[strings.ToUpper(code)]; ok {
				code = v
			}
			if n.Created != nil && len(n.Errors) == 0 {
				out.IDs[code] = n.Created.ID
			} else {
				out.Failed = append(out.Failed, code)
			}
		}
		if after = d.Bulk.Codes.PageInfo.next(); after == "" {
			break
		}
	}
	out.Done = true
	// A code the job doesn't list wasn't created.
	for _, code := range codes {
		if _, ok := out.IDs[code]; !ok && !slices.Contains(out.Failed, code) {
			out.Failed = append(out.Failed, code)
		}
	}
	return out, nil
}

// RemoveCodes implements stores.DiscountCreator: the codes' ids (looked up
// for codes without one) go to discountCodeRedeemCodeBulkDelete, 250 at a
// time. Codes Shopify doesn't have are skipped.
func (c *Client) RemoveCodes(ctx context.Context, discountID string, codes []string, ids []string) error {
	var del []string
	for i, code := range codes {
		id := ""
		if i < len(ids) {
			id = ids[i]
		}
		if id == "" {
			found, err := c.codeID(ctx, discountID, code)
			if err != nil {
				return err
			}
			id = found
		}
		if id != "" {
			del = append(del, gid("DiscountRedeemCode", id))
		}
	}
	for start := 0; start < len(del); start += maxBulkCodes {
		part := del[start:min(start+maxBulkCodes, len(del))]
		var d struct {
			Delete struct {
				UserErrors []userError `json:"userErrors"`
			} `json:"discountCodeRedeemCodeBulkDelete"`
		}
		if err := c.gql(ctx, "DiscountCodesDelete", mDiscountCodesDelete, map[string]any{"discountId": gid("DiscountCodeNode", discountID), "ids": part}, &d); err != nil {
			return err
		}
		if err := userErrors("discountCodeRedeemCodeBulkDelete", d.Delete.UserErrors); err != nil {
			return err
		}
	}
	return nil
}

// codeID finds a code's id on a discount ("" when it has no such code).
func (c *Client) codeID(ctx context.Context, discountID, code string) (string, error) {
	var d struct {
		Node *struct {
			CodeDiscount struct {
				Codes struct {
					Nodes []redeemCode `json:"nodes"`
				} `json:"codes"`
			} `json:"codeDiscount"`
		} `json:"codeDiscountNode"`
	}
	if err := c.gql(ctx, "DiscountCodeLookup", qDiscountCodeLookup, map[string]any{"id": gid("DiscountCodeNode", discountID), "query": strconv.Quote(code)}, &d); err != nil {
		return "", err
	}
	if d.Node == nil {
		return "", nil
	}
	for _, n := range d.Node.CodeDiscount.Codes.Nodes {
		if strings.EqualFold(n.Code, code) {
			return n.ID, nil
		}
	}
	return "", nil
}

// DisableDiscount implements stores.DiscountCreator: discountCodeDeactivate
// (the discount stays in Shopify, its codes stop working).
func (c *Client) DisableDiscount(ctx context.Context, discountID string) error {
	var d struct {
		Deactivate struct {
			UserErrors []userError `json:"userErrors"`
		} `json:"discountCodeDeactivate"`
	}
	if err := c.gql(ctx, "DiscountDeactivate", mDiscountDeactivate, map[string]any{"id": gid("DiscountCodeNode", discountID)}, &d); err != nil {
		return err
	}
	return userErrors("discountCodeDeactivate", d.Deactivate.UserErrors)
}
