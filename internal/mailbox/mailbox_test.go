package mailbox

import "testing"

func TestFromHosting(t *testing.T) {
	for _, c := range []struct {
		domain, hosting string
		want            Class
	}{
		{"gmail.com", "", Class{Provider: Gmail}},
		{"hotmail.co.uk", "", Class{Provider: Outlook}},
		{"acme.io", "Google Workspace", Class{Provider: Gmail, Detail: "Google Workspace"}},
		{"northwind.com", "Microsoft 365", Class{Provider: Outlook, Detail: "Microsoft 365"}},
		{"shop.example", "Outlook.com", Class{Provider: Outlook, Detail: "Outlook.com"}},
		{"bakery.example", "Yahoo Mail", Class{Provider: Yahoo, Detail: "Yahoo Mail"}},
		{"family.example", "iCloud Mail", Class{Provider: ICloud, Detail: "iCloud Mail"}},
		{"berg.de", "Fastmail", Class{Provider: Other, Detail: "Fastmail"}},
		{"berg.de", "", Class{Provider: Other}},
		// A consumer domain keeps its provider whatever the MX says.
		{"outlook.com", "Google Workspace", Class{Provider: Outlook}},
	} {
		if got := FromHosting(c.domain, c.hosting); got != c.want {
			t.Errorf("FromHosting(%q, %q) = %+v, want %+v", c.domain, c.hosting, got, c.want)
		}
	}
	if !(Class{Provider: Gmail, Detail: "Google Workspace"}).Hosted() || (Class{Provider: Gmail}).Hosted() || (Class{Provider: Other, Detail: "Fastmail"}).Hosted() {
		t.Fatal("Hosted")
	}
}

func TestReportsComplaints(t *testing.T) {
	for p, want := range map[string]bool{Gmail: false, ICloud: false, Outlook: true, Yahoo: true, Other: true} {
		if got := ReportsComplaints(p); got != want {
			t.Errorf("ReportsComplaints(%s) = %v, want %v", p, got, want)
		}
	}
}
