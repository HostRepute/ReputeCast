package pagelabels

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

func TestAppLanguagesShipEveryLabel(t *testing.T) {
	for _, l := range []i18n.Locale{i18n.German, i18n.Spanish, i18n.French, i18n.Vietnamese} {
		if missing := catalog.Missing(l); len(missing) > 0 {
			t.Fatalf("%s.json is missing %d labels: %v", l, len(missing), missing)
		}
	}
	if Count() < 100 || len(All()) != Count() {
		t.Fatalf("Count() = %d, All() = %d", Count(), len(All()))
	}
	if !Shipped("vi-VN") || !Shipped("en") || !Shipped("es-MX") || !Shipped("fr") || !Shipped("de-AT") || Shipped("it") {
		t.Fatal("Shipped should hold for the app languages only")
	}
}

func TestShippedLanguageNeedsNothingWritten(t *testing.T) {
	vi := For("vi-VN", map[string]string{"form.button": "ignored"})
	if got := vi.Text("form.button", nil); got != "Đăng ký" {
		t.Fatalf("vi form.button = %q", got)
	}
	if vi.Missing() != nil || !vi.Complete() || vi.Code() != "vi" {
		t.Fatalf("vi: missing %v, complete %v, code %q", vi.Missing(), vi.Complete(), vi.Code())
	}
	en := English()
	if got := en.Text("form.closed.text", i18n.Args{"publisher": "Acme"}); got != "Acme isn't taking signups through this form right now." {
		t.Fatalf("en = %q", got)
	}
}

func TestWrittenLabelsFallBackToEnglish(t *testing.T) {
	it := For("it-IT", map[string]string{"form.button": "Iscrivimi", "list.and": "{items} e {last}", "form.lists": "  "})
	if got := it.Text("form.button", nil); got != "Iscrivimi" {
		t.Fatalf("it form.button = %q", got)
	}
	if got := it.Text("form.lists", nil); got != "What would you like?" {
		t.Fatalf("blank label should show English, got %q", got)
	}
	if got := it.Join([]string{"Weekly Digest", "Product Updates", "Beta Program"}); got != "Weekly Digest, Product Updates e Beta Program" {
		t.Fatalf("it join = %q", got)
	}
	missing := it.Missing()
	if len(missing) != Count()-2 || it.Complete() {
		t.Fatalf("missing %d of %d", len(missing), Count())
	}
	for _, k := range missing {
		if k == "form.button" || k == "list.and" {
			t.Fatalf("%s is written", k)
		}
	}
}

func TestHTMLEscapesWrittenText(t *testing.T) {
	it := For("it", map[string]string{"form.confirm.text": "Abbiamo inviato un link a <b>{email}</b> & {publisher}."})
	got := it.HTML("form.confirm.text", i18n.Args{"email": "a<b>@x.io", "publisher": "Acme"}, nil)
	if got != "Abbiamo inviato un link a <b>a&lt;b&gt;@x.io</b> &amp; Acme." {
		t.Fatalf("html = %q", got)
	}
	en := English().HTML("confirm.notSubscribed", i18n.Args{"offer": "release news"}, i18n.Tags{"link": func(s string) string { return `<a href="/p">` + s + `</a>` }})
	if en != `Not subscribed. <a href="/p">Add it</a> if you'd like release news too.` {
		t.Fatalf("en html = %q", en)
	}
}

func TestCheckKeepsToTheEnglishArguments(t *testing.T) {
	if err := Check("form.closed.text", "{publisher} no acepta suscripciones ahora."); err != nil {
		t.Fatal(err)
	}
	var ae *ArgError
	if err := Check("form.closed.text", "{name} no acepta suscripciones."); !errors.As(err, &ae) || ae.Name != "name" {
		t.Fatalf("unknown argument: %v", err)
	}
	if err := Check("form.button", "<b>Suscribirme</b>"); !errors.As(err, &ae) || ae.Name != "<b>" {
		t.Fatalf("unknown tag: %v", err)
	}
	if err := Check("form.closed.text", "{publisher no acepta"); err == nil {
		t.Fatal("an unclosed brace should fail")
	}
	if err := Check("nope", "x"); !errors.Is(err, ErrUnknownLabel) {
		t.Fatalf("unknown key: %v", err)
	}
	if got := strings.Join(Arguments("confirm.notSubscribed"), " "); got != "<link> {offer}" {
		t.Fatalf("arguments = %q", got)
	}
}

func TestDatesFollowTheLanguage(t *testing.T) {
	at := time.Date(2026, 10, 3, 9, 14, 0, 0, time.UTC)
	if got := English().Stamp(at); got != "3 Oct 2026, 09:14" {
		t.Fatalf("en stamp = %q", got)
	}
	es := For("es", nil)
	if got := es.Stamp(at); got != "3 oct 2026, 9:14" {
		t.Fatalf("es stamp = %q", got)
	}
	if got := es.FullDate(at); !strings.Contains(got, "3 de oct de 2026") {
		t.Fatalf("es full date = %q", got)
	}
	if got := For("de", nil).Count(48213); got != "48.213" {
		t.Fatalf("de count = %q", got)
	}
	if got := English().Count(31988); got != "31,988" {
		t.Fatalf("en count = %q", got)
	}
	if Dir("ar") != "rtl" || Dir("es") != "ltr" {
		t.Fatal("dir")
	}
}
