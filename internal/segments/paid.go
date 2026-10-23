package segments

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/paidaccess"
	"github.com/hostrepute/reputecast/internal/store"
)

// Paid subscription conditions (Earn, design expansion X16, brief §6):
// status, tier and above, period, member since, ended in the last N days
// and lifetime value. They read the segment views paid_members (one row per
// live subscription, complimentary grant or gift; test paid members never
// count) and paid_tiers (see views.go), so the generated SQL also runs as a
// SQL segment. Status, tier and period describe current paid access; ended
// means access ended and none is left.

// The paid subscription fields.
const (
	FieldPaidStatus = "paid_status"
	FieldPaidTier   = "paid_tier"
	FieldPaidPeriod = "paid_period"
	FieldPaidSince  = "paid_since"
	FieldPaidEnded  = "paid_ended"
	FieldPaidLTV    = "paid_ltv"
)

func init() {
	builtinFields[FieldPaidStatus] = fieldSpec{KindEnum, []string{"is", "is_not"}}
	builtinFields[FieldPaidTier] = fieldSpec{KindEnum, []string{"is", "is_not"}}
	builtinFields[FieldPaidPeriod] = fieldSpec{KindEnum, []string{"is", "is_not"}}
	builtinFields[FieldPaidSince] = fieldSpec{KindDate, []string{"before", "after"}}
	builtinFields[FieldPaidEnded] = fieldSpec{KindPeriod, []string{"in_last"}}
	builtinFields[FieldPaidLTV] = fieldSpec{KindNumber, scoreOps}
	enumValues[FieldPaidStatus] = [][2]string{{"active", ""}, {"trialing", ""}, {"past_due", ""}, {"complimentary", ""},
		{"gift", ""}, {"ended", ""}, {"none", ""}}
	enumValues[FieldPaidPeriod] = [][2]string{{"month", ""}, {"year", ""}}
	builtinOrder = append(builtinOrder, FieldPaidTier, FieldPaidStatus, FieldPaidPeriod, FieldPaidSince, FieldPaidEnded, FieldPaidLTV)
	extraValidate[FieldPaidTier] = validatePaidTier
	extraValidate[FieldPaidLTV] = validatePaidLTV
	extraCompile[FieldPaidStatus] = compilePaidStatus
	extraCompile[FieldPaidTier] = compilePaidTier
	extraCompile[FieldPaidPeriod] = compilePaidPeriod
	extraCompile[FieldPaidSince] = compilePaidSince
	extraCompile[FieldPaidEnded] = compilePaidEnded
	extraCompile[FieldPaidLTV] = compilePaidLTV
	RegisterFieldOptions(FieldPaidTier, paidTierOptions)
}

// paidTierOptions are the tiers, lowest first: "Insider and above", then
// the top tier alone ("Founding member").
func paidTierOptions(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]SegmentOption, error) {
	tiers, err := paidaccess.Tiers(ctx, q, wsID)
	if err != nil {
		return nil, err
	}
	l := i18n.LangFrom(ctx)
	out := make([]SegmentOption, 0, len(tiers))
	for _, t := range tiers {
		label := mTierAndAbove.Text(l, i18n.Args{"tier": t.Short})
		if t.Top {
			label = t.Short
		}
		out = append(out, SegmentOption{Value: t.ID.String(), Label: label})
	}
	return out, nil
}

func validatePaidTier(c SegmentCondition) error {
	if _, err := uuid.Parse(c.Value); err != nil {
		return says(mPickTier)
	}
	return nil
}

func ltvValue(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v, err == nil && v >= 0
}

func validatePaidLTV(c SegmentCondition) error {
	a, ok := ltvValue(c.Value)
	if !ok {
		return says(mLTV)
	}
	if c.Op == "between" {
		b, ok := ltvValue(c.To)
		if !ok {
			return says(mBothEnds)
		}
		if b < a {
			return says(mRangeSwapAm)
		}
	}
	return nil
}

// paidAccess is "the contact has paid access now" over paid_members p.
const paidAccess = "EXISTS (SELECT 1 FROM paid_members p WHERE p.contact_id = c.id AND p.access"

func compilePaidStatus(t *sqlText, c SegmentCondition) {
	if c.Op == "is_not" {
		t.raw("NOT (")
		defer t.raw(")")
	}
	switch c.Value {
	case "none":
		t.raw("NOT " + paidAccess + ")")
	case "ended":
		t.raw("(NOT " + paidAccess + ")" + indent + "AND EXISTS (SELECT 1 FROM paid_members p WHERE p.contact_id = c.id AND p.ended_at IS NOT NULL))")
	case "complimentary", "gift":
		t.raw(paidAccess + " AND p.kind = ")
		t.bind(str(c.Value))
		t.raw(")")
	default:
		t.raw(paidAccess + " AND p.status = ")
		t.bind(str(c.Value))
		t.raw(")")
	}
}

func compilePaidTier(t *sqlText, c SegmentCondition) {
	id, _ := uuid.Parse(c.Value)
	if c.Op == "is_not" {
		t.raw("NOT ")
	}
	t.raw(paidAccess + indent + "AND p.tier_rank >= (SELECT pt.rank FROM paid_tiers pt WHERE pt.id = ")
	t.bind(uid(id))
	t.raw("))")
}

func compilePaidPeriod(t *sqlText, c SegmentCondition) {
	if c.Op == "is_not" {
		t.raw("NOT ")
	}
	t.raw(paidAccess + " AND p.period = ")
	t.bind(str(c.Value))
	t.raw(")")
}

func compilePaidSince(t *sqlText, c SegmentCondition) {
	t.raw(paidAccess + " AND p.started_at::date ")
	if c.Op == "before" {
		t.raw("< ")
	} else {
		t.raw("> ")
	}
	v := strings.TrimSpace(c.Value)
	d, _ := time.Parse("2006-01-02", v)
	t.bind(arg{v: d, literal: quote(v)})
	t.raw("::date)")
}

func compilePaidEnded(t *sqlText, c SegmentCondition) {
	t.raw("(NOT " + paidAccess + ")" + indent + "AND EXISTS (SELECT 1 FROM paid_members p WHERE p.contact_id = c.id AND p.ended_at >= " + interval(c) + "))")
}

func compilePaidLTV(t *sqlText, c SegmentCondition) {
	ltv := "(SELECT coalesce(sum(p.paid), 0) FROM paid_members p WHERE p.contact_id = c.id)"
	a, _ := ltvValue(c.Value)
	switch c.Op {
	case "at_most":
		t.raw(ltv + " <= ")
		t.bind(num(a))
	case "between":
		b, _ := ltvValue(c.To)
		t.raw(ltv + " BETWEEN ")
		t.bind(num(a))
		t.raw(" AND ")
		t.bind(num(b))
	default:
		t.raw(ltv + " >= ")
		t.bind(num(a))
	}
}
