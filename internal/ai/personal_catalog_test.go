package ai_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/llm"
)

// A member adds a personal key from /ai/personal alone: it lists every
// provider with the models and prices of the price list, as /ai does for
// people who may see the workspace's integrations.
func TestAIPersonalCatalog(t *testing.T) {
	first := newModelServer(t)
	p := newPersonal(t, first, nil)
	p.enable(5)
	var own ai.AIPersonalSettings
	p.mc.Do(http.MethodGet, p.base+"/personal", nil).MustStatus(t, 200).JSON(t, &own)
	var all ai.AISettings
	p.c.Do(http.MethodGet, p.base, nil).MustStatus(t, 200).JSON(t, &all)
	if len(own.Catalog) != len(llm.Providers) || !reflect.DeepEqual(own.Catalog, all.Catalog) || !own.PriceListDate.Equal(llm.PriceListDate) {
		t.Fatalf("the personal catalog: %d providers, price list of %s", len(own.Catalog), own.PriceListDate)
	}
	priced := 0
	for _, e := range own.Catalog {
		for _, m := range e.Models {
			if m.Known && m.Input > 0 {
				priced++
			}
		}
	}
	if priced == 0 {
		t.Fatal("the personal catalog has no priced models")
	}
}
