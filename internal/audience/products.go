package audience

import "github.com/google/uuid"

// Digital products in the contacts table (design expansion phase 7, X19):
// the filter "Bought a product". A buyer is a contact with a live order
// that wasn't fully refunded; test orders never count.

// ProductsFilter narrows contacts by what they bought.
type ProductsFilter struct {
	// Bought keeps the buyers: "any" for any product, or a product's id.
	Bought string `json:"bought,omitempty" maxLength:"40" doc:"Buyers of a product (its id), or of any product (any)"`
}

// ProductsQuery is the same filter as query parameters.
type ProductsQuery struct {
	Bought string `query:"bought" maxLength:"40" doc:"Only contacts who bought a product: any, or the product's id"`
}

func (q ProductsQuery) filter() ProductsFilter { return ProductsFilter(q) }

// boughtSQL is a live order of contact c that wasn't fully refunded.
const boughtSQL = `EXISTS (SELECT 1 FROM product_orders po WHERE po.contact_id = c.id AND po.mode = 'live' AND po.status <> 'refunded'`

// productsFilter appends the products conditions of f.
func productsFilter(out *filterSQL, f ProductsFilter) {
	switch f.Bought {
	case "":
	case "any":
		out.add(boughtSQL + ")")
	default:
		id, err := uuid.Parse(f.Bought)
		if err != nil {
			// Not a product: nobody bought it.
			out.add("false")
			return
		}
		out.add(boughtSQL+" AND po.product_id = ?)", id)
	}
}
