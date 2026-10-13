package bounce

import "testing"

func TestSplit(t *testing.T) {
	cases := []struct{ in, code, rest string }{
		{"smtp; 550 5.1.1 <r.w@outlook.com>: Recipient address rejected: User unknown", "550 5.1.1", "<r.w@outlook.com>: Recipient address rejected: User unknown"},
		{"250 2.0.0 OK 1477944899 ly5si31746220wjb.95 - gsmtp", "250 2.0.0", "OK 1477944899 ly5si31746220wjb.95 - gsmtp"},
		{"421 4.7.0 [TSS04] Messages from 203.0.113.25 temporarily deferred", "421 4.7.0", "[TSS04] Messages from 203.0.113.25 temporarily deferred"},
		{"5.1.1", "5.1.1", ""},
		{"Mailbox full", "", "Mailbox full"},
		{"550-5.7.1 Our system has detected", "550 5.7.1", "Our system has detected"},
	}
	for _, c := range cases {
		code, rest := Split(c.in)
		if code != c.code || rest != c.rest {
			t.Errorf("Split(%q) = %q, %q; want %q, %q", c.in, code, rest, c.code, c.rest)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct{ status, diag, want string }{
		{"5.1.1", "Recipient address rejected: User unknown", MailboxUnknown},
		{"550 5.1.1", "", MailboxUnknown},
		{"452 4.2.2", "Mailbox full", MailboxFull},
		{"421 4.7.0", "[TSS04] Messages from 203.0.113.25 temporarily deferred", Reputation},
		{"554 5.7.1", "Service unavailable; Client host [203.0.113.25] blocked using Spamhaus", SpamBlock},
		{"550 5.7.26", "Unauthenticated email is not accepted due to the sending domain's DMARC policy", PolicyBlock},
		{"550 5.4.1", "Recipient address rejected: Access denied", PolicyBlock},
		{"451 4.3.0", "Temporary local problem", Temporary},
		{"", "something odd", Other},
		{"5.1.2", "Host not found", DomainInvalid},
	}
	for _, c := range cases {
		if got := Classify(c.status, c.diag); got != c.want {
			t.Errorf("Classify(%q, %q) = %s; want %s", c.status, c.diag, got, c.want)
		}
	}
	if !Permanent("550 5.1.1") || Permanent("421 4.7.0") || Permanent("unknown") {
		t.Fatal("Permanent")
	}
}
