package segments

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Site tracker conditions (M62): "Viewed a product in the last N days",
// any product or one of a linked store's collections, and "Last seen on
// the site". The contacts view reads when the contact last viewed a
// product (tracker_product_views) and was last seen on a tracked site
// (tracker_visitors); a collection is checked against the store's
// products.

var mPickCollection = msgs.M("problem.pickCollection")

func init() {
	builtinFields["site_viewed_product"] = fieldSpec{KindPeriod, []string{"in_last", "not_in_last"}}
	builtinFields["site_last_seen"] = fieldSpec{KindPeriod, []string{"in_last", "not_in_last", "never"}}
	builtinOrder = append(builtinOrder, "site_viewed_product", "site_last_seen")
	extraValidate["site_viewed_product"] = validateSiteViewed
	extraValidate["site_last_seen"] = validatePeriodField
	extraCompile["site_viewed_product"] = compileSiteViewed
	extraCompile["site_last_seen"] = compileSiteLastSeen
	extraOptions["site_viewed_product"] = siteCollectionOptions
}

// trackerColumns add the site tracker's two moments to the contacts view.
const trackerColumns = `,
           (SELECT max(pv.last_viewed_at) FROM public.tracker_product_views pv WHERE pv.contact_id = c.id) AS site_viewed_at,
           (SELECT max(tv.last_seen) FROM public.tracker_visitors tv WHERE tv.contact_id = c.id) AS site_last_seen_at`

// validatePeriodField checks "in the last N days/weeks" as the built-in
// period fields do.
func validatePeriodField(c SegmentCondition) error {
	if c.Op == "never" {
		return nil
	}
	if c.Amount < 1 || c.Amount > 3650 {
		return sayField(mPeriodRange, c.Field, false, nil)
	}
	if c.Unit != "days" && c.Unit != "weeks" {
		return sayField(mPeriodUnit, c.Field, false, nil)
	}
	return nil
}

// collectionRef reads "collection:<store>:<id>"; "" is any product.
func collectionRef(v string) (uuid.UUID, string, bool) {
	kind, rest, ok := strings.Cut(strings.TrimSpace(v), ":")
	if !ok || kind != "collection" {
		return uuid.Nil, "", false
	}
	sid, id, ok := strings.Cut(rest, ":")
	st, err := uuid.Parse(sid)
	if !ok || err != nil || id == "" || len(id) > 64 {
		return uuid.Nil, "", false
	}
	return st, id, true
}

func validateSiteViewed(c SegmentCondition) error {
	if err := validatePeriodField(c); err != nil {
		return err
	}
	if v := strings.TrimSpace(c.Value); v != "" && v != "any" {
		if _, _, ok := collectionRef(v); !ok {
			return says(mPickCollection)
		}
	}
	return nil
}

func compileSiteViewed(t *sqlText, c SegmentCondition) {
	if c.Op == "not_in_last" {
		t.raw("NOT ")
	}
	st, col, ok := collectionRef(c.Value)
	if !ok {
		t.raw("coalesce(c.site_viewed_at >= " + interval(c) + ", false)")
		return
	}
	t.raw("EXISTS (SELECT 1 FROM public.tracker_product_views pv JOIN public.store_products p ON p.id = pv.product_id" + indent +
		"WHERE pv.contact_id = c.id AND pv.last_viewed_at >= " + interval(c) + " AND p.store_id = ")
	t.bind(uid(st))
	t.raw(" AND ")
	t.bind(str(col))
	t.raw(" = ANY (p.collections))")
}

func compileSiteLastSeen(t *sqlText, c SegmentCondition) {
	switch c.Op {
	case "never":
		t.raw("c.site_last_seen_at IS NULL")
	case "not_in_last":
		t.raw("(c.site_last_seen_at IS NULL OR c.site_last_seen_at < " + interval(c) + ")")
	default:
		t.raw("c.site_last_seen_at >= " + interval(c))
	}
}

// siteCollectionOptions are "Any product", then each connected store's
// collections.
func siteCollectionOptions(ctx context.Context, q *store.Queries, ws uuid.UUID) ([]SegmentOption, error) {
	l := i18n.LangFrom(ctx)
	out := []SegmentOption{{Value: "any", Label: mAnyProduct.Text(l, nil)}}
	rows, err := q.CommerceStores(ctx, ws)
	if err != nil {
		return nil, err
	}
	for _, st := range rows {
		if st.Status == "connecting" {
			continue
		}
		cols, err := q.CommerceCollections(ctx, st.ID)
		if err != nil {
			return nil, err
		}
		for _, col := range cols {
			out = append(out, SegmentOption{Value: "collection:" + st.ID.String() + ":" + col.ExternalID, Label: mCollection.Text(l, i18n.Args{"store": st.Name, "collection": col.Title})})
		}
	}
	return out, nil
}
