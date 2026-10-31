package transactional

import (
	"errors"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// StarterDemo is a starter's content for the demo data: subject, preheader
// and MJML with the product name and the address filled in, compiled.
func StarterDemo(key, product, address string) (DemoTemplate, []TransactionalTestData, error) {
	st, ok := starterByKey(key)
	if !ok {
		return DemoTemplate{}, nil, errors.New("transactional: no starter " + key)
	}
	d := DemoTemplate{Subject: fillProduct(st.Subject, product), Preheader: fillProduct(st.Preheader, product), MJML: st.mjmlFor(i18n.Default, product, address),
		Variables: st.Variables}
	return d, st.TestData, d.Compile()
}

// Digital products (design expansion phase 7): the delivery email. The
// products domain makes it, in the Essential "Purchases" category, when
// the first product is created. Its download_url is sensitive, so the
// message log never keeps the link.
func init() {
	starterList = append(starterList, TransactionalStarter{
		Key: "product-delivery", Name: "Product delivery", Description: "A digital product's download link, sent after a purchase", Kind: KindEssential,
		Subject: "Your download: {{ product_name }}", Preheader: "Your link works until {{ expires_on }}.",
		Variables: []TransactionalVariable{
			{Name: "first_name", Type: TypeText, Sample: "Ana", Default: "there"},
			{Name: "product_name", Type: TypeText, Required: true, Sample: "The Deliverability Field Guide"},
			{Name: "download_url", Type: TypeURL, Required: true, Sensitive: true, Sample: "https://news.example.com/d/sample-token"},
			{Name: "file", Type: TypeText, Required: true, Sample: "the PDF, 4.2 MB"},
			{Name: "expires_on", Type: TypeText, Required: true, Sample: "22 Oct 2026"},
			{Name: "downloads_allowed", Type: TypeNumber, Required: true, Sample: "5"},
			{Name: "order_number", Type: TypeText, Required: true, Sample: "1311"},
			{Name: "amount", Type: TypeText, Required: true, Sample: "$29.00"},
			{Name: "account_url", Type: TypeText, Sample: "news.example.com/account"},
			{Name: "site", Type: TypeText, Sample: "news.example.com", Default: "us"},
		},
		TestData: []TransactionalTestData{
			{Name: "A buyer", Data: map[string]any{"first_name": "Ana", "product_name": "The Deliverability Field Guide", "download_url": "https://news.example.com/d/sample-token",
				"file": "the PDF, 4.2 MB", "expires_on": "22 Oct 2026", "downloads_allowed": 5, "order_number": "1311", "amount": "$29.00",
				"account_url": "news.example.com/account", "site": "news.example.com"}},
			{Name: "No first name, several files", Data: map[string]any{"product_name": "Newsletter Launch Kit", "download_url": "https://news.example.com/d/sample-token",
				"file": "the 3 files, 38 MB", "expires_on": "22 Oct 2026", "downloads_allowed": 5, "order_number": "1312", "amount": "$15.00", "site": "news.example.com"}},
		},
		heading: "Your download is ready",
		body:    []string{"Hi {{ first_name }},", "Thanks for buying {{ product_name }}."},
		button:  "Download {{ file }}", href: "{{ download_url }}",
		bodyAfter: []string{"This link works until {{ expires_on }}, for {{ downloads_allowed }} downloads.",
			"{% if account_url %}Need it again later? Sign in at {{ account_url }}.{% endif %}"},
		after: "Order {{ order_number }} · {{ amount }}. Stripe sends your receipt separately.",
		why:   "You get this email because you bought from {{ site }}. Purchase emails can't be turned off.",
	})
}

// The buyer's notice when a delayed payment (a bank debit) didn't clear
// after checkout (review 2026-11 EN-7): nothing was charged, and a link to
// try again. It is made with the delivery template, as a draft, and sent
// once it's published.
func init() {
	starterList = append(starterList, TransactionalStarter{
		Key: "product-payment-failed", Name: "Payment didn't go through", Description: "When a bank payment for a digital product fails after checkout", Kind: KindEssential,
		Subject: "Your payment for {{ product_name }} didn't go through", Preheader: "Nothing was charged. You can try again.",
		Variables: []TransactionalVariable{
			{Name: "first_name", Type: TypeText, Sample: "Ana", Default: "there"},
			{Name: "product_name", Type: TypeText, Required: true, Sample: "The Deliverability Field Guide"},
			{Name: "retry_url", Type: TypeURL, Required: true, Sample: "https://news.example.com/products/field-guide"},
			{Name: "site", Type: TypeText, Sample: "news.example.com", Default: "us"},
		},
		TestData: []TransactionalTestData{
			{Name: "A buyer", Data: map[string]any{"first_name": "Ana", "product_name": "The Deliverability Field Guide",
				"retry_url": "https://news.example.com/products/field-guide", "site": "news.example.com"}},
		},
		heading:   "Your payment didn't go through",
		body:      []string{"Hi {{ first_name }},", "Your bank didn't complete the payment for {{ product_name }}, so the order wasn't placed and nothing was charged."},
		button:    "Try again",
		href:      "{{ retry_url }}",
		bodyAfter: []string{"If you think this is a mistake, check with your bank or pay another way."},
		why:       "You get this email because you started a purchase at {{ site }}. Purchase emails can't be turned off.",
	})
}
