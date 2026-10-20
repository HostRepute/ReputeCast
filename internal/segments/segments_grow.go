package segments

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Grow's conditions (design expansion phase 5, brief §3.7): "Referrals
// counted" at least, at most or between; a "Referral milestone" reached or
// not; and "Joined through" a referral or a partner, any or one. The
// contacts view gives each contact their counted referrals and the
// milestones they reached; how they joined is the growth source.
//
// This file's init runs after scoring.go's, so the fields can follow the
// growth fields in the builder.

func init() {
	opLabels["reached"] = "reached"
	opLabels["not_reached"] = "not reached"
	builtinFields["joined_through"] = fieldSpec{"Joined through", KindEnum, []string{"is", "is_not"}}
	builtinFields["referrals_counted"] = fieldSpec{"Referrals counted", KindNumber, scoreOps}
	builtinFields["referral_milestone"] = fieldSpec{"Referral milestone", KindEnum, []string{"reached", "not_reached"}}
	builtinOrder = insertAfter(builtinOrder, "referring_domain", "joined_through", "referrals_counted", "referral_milestone")
	extraValidate["joined_through"] = validateJoinedThrough
	extraValidate["referrals_counted"] = validateReferralsCounted
	extraValidate["referral_milestone"] = validateMilestone
	extraCompile["joined_through"] = compileJoinedThrough
	extraCompile["referrals_counted"] = compileReferralsCounted
	extraCompile["referral_milestone"] = compileMilestone
	extraOptions["joined_through"] = joinedThroughOptions
	extraOptions["referral_milestone"] = milestoneOptions
}

// growColumns add Grow's numbers to the contacts view: counted referrals as
// a referrer, and the milestones reached (one reward row each). Scalar
// subqueries, so a query that doesn't read them doesn't pay for them.
const growColumns = `,
           (SELECT count(*) FROM public.referrals r WHERE r.referrer_id = c.id AND r.status = 'counted') AS referrals_counted,
           ARRAY(SELECT rw.milestone_id FROM public.referral_rewards rw WHERE rw.contact_id = c.id AND rw.milestone_id IS NOT NULL) AS referral_milestones`

func referralCount(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	return n, err == nil && n >= 0 && n <= 100000
}

func validateReferralsCounted(c SegmentCondition) error {
	a, ok := referralCount(c.Value)
	if !ok {
		return problemf("Enter a number of referrals, 0 or more.")
	}
	if c.Op == "between" {
		b, ok := referralCount(c.To)
		if !ok {
			return problemf("Enter both ends of the range.")
		}
		if b < a {
			return problemf("The range ends below where it starts. Swap the two numbers.")
		}
	}
	return nil
}

func compileReferralsCounted(t *sqlText, c SegmentCondition) {
	a, _ := referralCount(c.Value)
	switch c.Op {
	case "at_most":
		t.raw("c.referrals_counted <= ")
		t.bind(num(float64(a)))
	case "between":
		b, _ := referralCount(c.To)
		t.raw("c.referrals_counted BETWEEN ")
		t.bind(num(float64(a)))
		t.raw(" AND ")
		t.bind(num(float64(b)))
	default:
		t.raw("c.referrals_counted >= ")
		t.bind(num(float64(a)))
	}
}

func validateMilestone(c SegmentCondition) error {
	if _, err := uuid.Parse(strings.TrimSpace(c.Value)); err != nil {
		return problemf("Pick a milestone.")
	}
	return nil
}

func compileMilestone(t *sqlText, c SegmentCondition) {
	if c.Op == "not_reached" {
		t.raw("NOT (")
	} else {
		t.raw("(")
	}
	t.bind(str(strings.TrimSpace(c.Value)))
	t.raw("::uuid = ANY (c.referral_milestones))")
}

// joinedValues: "referral", "partner" (any partner) or "partner:<link id>".
func validateJoinedThrough(c SegmentCondition) error {
	v := strings.TrimSpace(c.Value)
	if v == "referral" || v == "partner" {
		return nil
	}
	if id, ok := strings.CutPrefix(v, "partner:"); ok {
		if _, err := uuid.Parse(id); err == nil {
			return nil
		}
	}
	return problemf("Pick a referral or a partner.")
}

func compileJoinedThrough(t *sqlText, c SegmentCondition) {
	v := strings.TrimSpace(c.Value)
	if c.Op == "is_not" {
		t.raw("NOT ")
	}
	t.raw("coalesce(")
	switch v {
	case "referral", "partner":
		// The growth source is "referral" or "partner:<link id>".
		t.raw("c.growth_source = ")
		t.bind(str(v))
		t.raw(" OR c.growth_source LIKE ")
		t.bind(str(v + ":%"))
	default:
		t.raw("c.growth_source = ")
		t.bind(str(v))
	}
	t.raw(", false)")
}

func joinedThroughOptions(ctx context.Context, q *store.Queries, ws uuid.UUID) ([]SegmentOption, error) {
	out := []SegmentOption{{Value: "referral", Label: "A referral"}, {Value: "partner", Label: "A partner (any)"}}
	partners, err := q.PartnerLinkNames(ctx, ws)
	if err != nil {
		return nil, err
	}
	for _, p := range partners {
		out = append(out, SegmentOption{Value: "partner:" + p.ID.String(), Label: "Partner · " + p.RemoteName})
	}
	return out, nil
}

func milestoneOptions(ctx context.Context, q *store.Queries, ws uuid.UUID) ([]SegmentOption, error) {
	rows, err := q.AutomationMilestones(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := []SegmentOption{}
	for _, m := range rows {
		unit := "referrals"
		if m.Count == 1 {
			unit = "referral"
		}
		out = append(out, SegmentOption{Value: m.ID.String(), Label: fmt.Sprintf("%s · %d %s · %s", m.ListName, m.Count, unit, m.RewardName)})
	}
	return out, nil
}
