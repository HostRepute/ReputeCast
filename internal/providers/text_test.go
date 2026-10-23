package providers

import (
	"testing"
	"testing/fstest"

	"github.com/hostrepute/reputecast/internal/i18n"
)

func TestDescriptorTextsReadInTheReadersLanguage(t *testing.T) {
	c, err := i18n.Load(fstest.MapFS{
		"messages/en.json": {Data: []byte(`{"pricing": "$1 per 1,000", "field.api_key.label": "API key", "field.api_key.help": "Settings › API keys.",
			"terms.0.note": "Opted-in lists only.", "suppressionMode": "Two-way", "account": "{plan} plan"}`)},
		"messages/vi.json": {Data: []byte(`{"pricing": "1 US$ cho mỗi 1.000 email", "field.api_key.label": "Khóa API", "field.api_key.help": "Cài đặt › Khóa API.",
			"terms.0.note": "Chỉ danh sách đã đồng ý nhận.", "suppressionMode": "Hai chiều", "account": "Gói {plan}"}`)},
	}, "providers.texttest")
	if err != nil {
		t.Fatal(err)
	}
	// The English comes from the catalog where the descriptor leaves it
	// empty; a text the descriptor wrote keeps its English and gets the
	// message.
	d := Descriptor{Adapter: "texttest", Capabilities: Capabilities{BatchSize: 1}, SuppressionMode: "Two-way",
		Terms:  []Terms{{Bulk: BulkAllowed}},
		Fields: []Field{{Key: "api_key", Secret: true}, {Key: "region", Label: "Region"}}}.WithMessages(c)
	if d.Messages != c || d.Capabilities.Pricing != "$1 per 1,000" || d.Fields[0].Label != "API key" || d.Fields[0].Help != "Settings › API keys." ||
		d.Terms[0].Note != "Opted-in lists only." || d.SuppressionMode != "Two-way" || d.SuppressionModeText.IsZero() {
		t.Fatalf("WithMessages: %+v", d)
	}
	// A text without a message reads in its English in every language.
	if d.Fields[1].Label != "Region" || !d.Fields[1].LabelText.IsZero() {
		t.Fatalf("a text without a message: %+v", d.Fields[1])
	}

	vi := d.In(i18n.For(i18n.Vietnamese))
	if vi.Capabilities.Pricing != "1 US$ cho mỗi 1.000 email" || vi.Fields[0].Label != "Khóa API" || vi.Fields[0].Help != "Cài đặt › Khóa API." ||
		vi.Fields[1].Label != "Region" || vi.Terms[0].Note != "Chỉ danh sách đã đồng ý nhận." || vi.SuppressionMode != "Hai chiều" {
		t.Fatalf("in Vietnamese: %+v", vi)
	}
	// The registry's descriptor keeps its English: In makes a copy.
	if d.Fields[0].Label != "API key" || d.Terms[0].Note != "Opted-in lists only." {
		t.Fatalf("In changed the descriptor: %+v", d)
	}
	en := d.In(i18n.Default)
	if en.Capabilities.Pricing != "$1 per 1,000" || en.Fields[0].Label != "API key" {
		t.Fatalf("in English: %+v", en)
	}

	// Text kept with arguments; English readers get the English as it was
	// written, a text whose message is gone reads in English.
	kept := c.M("account").Keep(i18n.Args{"plan": "Growth"})
	if got := TextIn("Growth plan", kept, i18n.For(i18n.Vietnamese)); got != "Gói Growth" {
		t.Errorf("vi: %q", got)
	}
	if got := TextIn("Growth  plan (as written)", kept, i18n.Default); got != "Growth  plan (as written)" {
		t.Errorf("en: %q", got)
	}
	if got := TextIn("Growth plan", i18n.Text{ID: "providers.texttest.gone"}, i18n.For(i18n.Vietnamese)); got != "Growth plan" {
		t.Errorf("gone: %q", got)
	}
	if got := TextIn("Growth plan", i18n.Text{}, i18n.For(i18n.Vietnamese)); got != "Growth plan" {
		t.Errorf("none: %q", got)
	}

	texts := d.Texts()
	if len(texts) != 6 {
		t.Fatalf("Texts: %+v", texts)
	}
	for _, x := range texts {
		if x.What == "field.region.label" && !x.Text.IsZero() {
			t.Errorf("%+v", x)
		}
	}
}
