package stores

import "testing"

func TestAmounts(t *testing.T) {
	cases := []struct {
		in, cur string
		want    int64
	}{
		{"58.00", "USD", 5800}, {"58.0", "USD", 5800}, {"58", "usd", 5800}, {"9.999", "USD", 1000}, {"0.005", "USD", 1},
		{"-18.00", "USD", -1800}, {"1200", "JPY", 1200}, {"1.234", "KWD", 1234}, {"", "USD", 0}, {"abc", "USD", 0},
	}
	for _, c := range cases {
		if got := ParseAmount(c.in, c.cur); got != c.want {
			t.Errorf("ParseAmount(%q, %s) = %d, want %d", c.in, c.cur, got, c.want)
		}
	}
	if FormatAmount(5800, "USD") != "58.00" || FormatAmount(5, "USD") != "0.05" || FormatAmount(1200, "JPY") != "1200" || FormatAmount(-1800, "USD") != "-18.00" {
		t.Fatal("FormatAmount")
	}
}

func TestLanding(t *testing.T) {
	utm, token := ParseLanding("/products/hoodie?utm_source=acme-newsletter&utm_medium=email&utm_campaign=october-product-update&rc_click=abc123#reviews")
	if utm.Source != "acme-newsletter" || utm.Medium != "email" || utm.Campaign != "october-product-update" || token != "abc123" {
		t.Fatalf("%+v %q", utm, token)
	}
	// The phase 7 brief's shorter name is read too.
	if _, token := ParseLanding("https://shop.acme.io/?rc=xyz"); token != "xyz" {
		t.Fatalf("rc: %q", token)
	}
	if got := StripLanding("/products/hoodie?utm_source=a&rc_click=abc123"); got != "/products/hoodie?utm_source=a" {
		t.Fatalf("StripLanding: %q", got)
	}
	if got := StripLanding("https://shop.acme.io/?rc=xyz"); got != "https://shop.acme.io/" {
		t.Fatalf("StripLanding full: %q", got)
	}
	if NormalizeHost(" https://Shop.Acme.io/admin?x=1 ") != "shop.acme.io" || NormalizeHost("acme-merch.myshopify.com.") != "acme-merch.myshopify.com" {
		t.Fatal("NormalizeHost")
	}
}
