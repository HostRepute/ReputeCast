package transactional

// Digital products (design expansion phase 7): the delivery email. The
// products domain makes it, in the Essential "Purchases" category, when
// the first product is created. Its download_url is sensitive, so the
// message log never keeps the link.
func init() {
	starterList = append(starterList, TransactionalStarter{
		Key: "product-delivery", Name: "Product delivery", Description: "A digital product's download link, sent after a purchase", Kind: KindEssential,
		Subject: "Your download: {{ product_name }}", Preheader: "Your link works until {{ expires_on }}.",
		Variables: []TransactionalVariable{
			{Name: "first_name", Type: TypeText, Sample: "Sam", Default: "there"},
			{Name: "product_name", Type: TypeText, Required: true, Sample: "The Deliverability Field Guide"},
			{Name: "download_url", Type: TypeURL, Required: true, Sensitive: true, Sample: "https://news.example.com/d/sample-token"},
			{Name: "file", Type: TypeText, Required: true, Sample: "the PDF, 4.2 MB"},
			{Name: "expires_on", Type: TypeText, Required: true, Sample: "22 Oct 2026"},
			{Name: "downloads_allowed", Type: TypeNumber, Required: true, Sample: "5"},
			{Name: "order_number", Type: TypeText, Sample: "1311"},
			{Name: "amount", Type: TypeText, Sample: "$29.00"},
		},
		TestData: []TransactionalTestData{
			{Name: "A buyer", Data: map[string]any{"first_name": "Sam", "product_name": "The Deliverability Field Guide", "download_url": "https://news.example.com/d/sample-token",
				"file": "the PDF, 4.2 MB", "expires_on": "22 Oct 2026", "downloads_allowed": 5, "order_number": "1311", "amount": "$29.00"}},
			{Name: "No first name, several files", Data: map[string]any{"product_name": "Launch Kit", "download_url": "https://news.example.com/d/sample-token",
				"file": "the 3 files, 16.4 MB", "expires_on": "22 Oct 2026", "downloads_allowed": 5, "order_number": "1312", "amount": "$49.00"}},
		},
		heading: "Your download is ready",
		body:    []string{"Hi {{ first_name }},", "Thanks for buying {{ product_name }}. Your order number is {{ order_number }}."},
		button:  "Download {{ file }}", href: "{{ download_url }}",
		after: "This link works until {{ expires_on }}, for {{ downloads_allowed }} downloads. If it runs out, the download page sends you a new one.",
	})
}
