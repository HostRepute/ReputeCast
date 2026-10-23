package referrals

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/liquid"
)

// The Referral tokens of the composer's token menu read in the member's
// language; the catalog keeps their English.
func TestTokenLabelsInTheMembersLanguage(t *testing.T) {
	labels := func(l i18n.Lang) map[string]string {
		out := map[string]string{}
		for _, tok := range liquid.InLanguage(liquid.Catalog(nil), l) {
			if tok.Kind == "referral" {
				out[tok.Key] = tok.Label
			}
		}
		return out
	}
	en, vi := labels(i18n.Default), labels(i18n.For(i18n.Vietnamese))
	if len(en) != 9 || en["referral.link"] != "Referral link" || en["referral.to_next"] != "Referrals to the next reward" {
		t.Errorf("English: %v", en)
	}
	if len(vi) != 9 || vi["referral.link"] != "Liên kết giới thiệu" || vi["referral.rewards"] != "Phần thưởng đã nhận" {
		t.Errorf("Vietnamese: %v", vi)
	}
	for _, tok := range liquid.Catalog(nil) {
		if tok.Key == "referral.link" && tok.Label != "Referral link" {
			t.Errorf("catalog: %q", tok.Label)
		}
	}
}
