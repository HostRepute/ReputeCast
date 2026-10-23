package scoring

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The recalculation's notification reads in each member's language; its
// English is the line it always was.
func TestRecalcNotificationInVietnamese(t *testing.T) {
	flows := []ScoringFlowImpact{
		{Name: "Sales-ready handoff", Crossed: 1108, Started: 0},
		{Name: "Nurture", Crossed: 0},
		{Name: "Win-back", Crossed: 1, Started: 1},
		{Name: "Upsell", Crossed: 42, Started: 40},
	}
	vi := i18n.For(i18n.Vietnamese)
	for _, start := range []bool{false, true} {
		for n := 0; n <= len(flows); n++ {
			rc := store.ScoringRecalc{StartAutomations: start}
			sum := ScoringRecalcSummary{Changed: 1312, Flows: flows[:n]}
			kept := recalcSummaryText(rc, sum)
			if kept.IsZero() {
				t.Fatalf("%d flows: no kept text", n)
			}
			if got, want := kept.In(i18n.Default), recalcSummaryLine(rc, sum); got != want {
				t.Errorf("English with %d flows:\ngot  %q\nwant %q", n, got, want)
			}
		}
	}
	rc := store.ScoringRecalc{}
	sum := ScoringRecalcSummary{Changed: 1, Flows: flows[:1]}
	if got := recalcSummaryText(rc, sum).In(i18n.Default); got != "1 contact changed band · Sales-ready handoff not started for 1,108" {
		t.Errorf("English: %q", got)
	}
	if got := recalcSummaryText(rc, sum).In(vi); got != "1 liên hệ đã đổi mức · Sales-ready handoff không bắt đầu cho 1.108 liên hệ" {
		t.Errorf("Vietnamese: %q", got)
	}
	more := append(append([]ScoringFlowImpact{}, flows...), ScoringFlowImpact{Name: "Fifth", Crossed: 3})
	if !recalcSummaryText(rc, ScoringRecalcSummary{Changed: 5, Flows: more}).IsZero() {
		t.Error("four automations crossed: the body keeps its English")
	}
	if got := msgs.M("notify.recalcTitle").Keep(nil).In(vi); got != "Đã tính lại điểm xong" {
		t.Errorf("title: %q", got)
	}
}
