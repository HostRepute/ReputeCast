package audience

import (
	"errors"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/api"
)

func TestNormalizeLanguage(t *testing.T) {
	for in, want := range map[string]string{
		"Spanish": "es", "es_MX": "es-MX", "ES": "es", "es-mx": "es-MX", " de-at ": "de-AT", "Español": "es",
		"pt-br": "pt-BR", "zh-hant-tw": "zh-Hant-TW", "": "", "Tiếng Việt": "vi", "eng": "en",
	} {
		got, err := NormalizeLanguage(in, "body.language")
		if err != nil || got != want {
			t.Errorf("NormalizeLanguage(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for in, closest := range map[string]string{"Spanosh": "es", "Spanglish!": "es", "Germen": "de", "Portugees": "pt", "Klingon": "", "xx": ""} {
		_, err := NormalizeLanguage(in, "body.language")
		var p *api.Problem
		if !errors.As(err, &p) || p.Status != 422 {
			t.Fatalf("%q: want a 422, got %v", in, err)
		}
		if got, _ := p.Meta["closest"].(string); got != closest {
			t.Errorf("%q: closest %q, want %q (%s)", in, got, closest, p.Detail)
		}
		if closest != "" && !strings.Contains(p.Detail, "Did you mean") {
			t.Errorf("%q: the sentence names the closest match: %s", in, p.Detail)
		}
		if len(p.Errors) != 1 || p.Errors[0].Location != "body.language" {
			t.Errorf("%q: the error points at the field: %+v", in, p.Errors)
		}
	}
}

func TestBrowserLanguage(t *testing.T) {
	for header, want := range map[string]string{
		"es-MX,es;q=0.9,en;q=0.8":  "es-MX",
		"en;q=0.5, de":             "de",
		"*":                        "",
		"":                         "",
		"x-pig-latin, fr-CA":       "fr-CA",
		"vi-VN;q=0.9, en-US;q=0.9": "vi-VN",
	} {
		if got := BrowserLanguage(header); got != want {
			t.Errorf("BrowserLanguage(%q) = %q, want %q", header, got, want)
		}
	}
	if got := BrowserLanguages("de-AT;q=0.4, es, en;q=0.8"); strings.Join(got, ",") != "es,en,de-AT" {
		t.Errorf("most preferred first: %v", got)
	}
}
