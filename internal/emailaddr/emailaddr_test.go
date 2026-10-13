package emailaddr

import "testing"

func TestCheck(t *testing.T) {
	cases := []struct {
		in, norm, problem string
	}{
		{"Sam.Rivera@Outlook.com ", "sam.rivera@outlook.com", ""},
		{"<priya.n@gmail.com>", "priya.n@gmail.com", ""},
		{"o'brien+news@acme.io", "o'brien+news@acme.io", ""},
		{"sam.rivera@", "sam.rivera@", ProblemNoDomain},
		{"sam@localhost", "sam@localhost", ProblemNoDomain},
		{"priya..n@gmail.com", "priya..n@gmail.com", ProblemDoubleDot},
		{"priya@gmail..com", "priya@gmail..com", ProblemDoubleDot},
		{`"hannah berg"`, `"hannah berg"`, ProblemNotAnAddress},
		{"hannah berg", "hannah berg", ProblemNotAnAddress},
		{"", "", ProblemNotAnAddress},
		{"@acme.io", "@acme.io", ProblemNotAnAddress},
		{"a@b@c.io", "a@b@c.io", ProblemNotAnAddress},
		{".dot@acme.io", ".dot@acme.io", ProblemNotAnAddress},
		{"x@-acme.io", "x@-acme.io", ProblemNotAnAddress},
		{"x@acme.c", "x@acme.c", ProblemNotAnAddress},
		{"x@acme.123", "x@acme.123", ProblemNotAnAddress},
		{"jörg@müller.de", "jörg@müller.de", ""},
	}
	for _, c := range cases {
		n, p := Check(c.in)
		if n != c.norm || p != c.problem {
			t.Errorf("Check(%q) = %q, %q; want %q, %q", c.in, n, p, c.norm, c.problem)
		}
	}
}

func TestClassify(t *testing.T) {
	if !IsRole("info@northwind-labs.com") || !IsRole("Sales+eu@acme.io") || IsRole("sam.rivera@outlook.com") {
		t.Fatal("role detection")
	}
	for domain, want := range map[string]string{
		"gmail.com": Gmail, "googlemail.com": Gmail, "hotmail.com": Outlook, "hotmail.co.uk": Outlook, "live.com.au": Outlook,
		"outlook.fr": Outlook, "yahoo.co.jp": Yahoo, "aol.com": Yahoo, "icloud.com": ICloud, "me.com": ICloud,
		"bergconsulting.de": Other, "live.example.com": Other, "fastmail.com": Other,
	} {
		if got := MailboxProvider(domain); got != want {
			t.Errorf("MailboxProvider(%s) = %s, want %s", domain, got, want)
		}
	}
	if HostingFromMX("aspmx.l.google.com.") != "Google Workspace" || HostingFromMX("in1-smtp.messagingengine.com") != "Fastmail" || HostingFromMX("mx.example.net") != "" {
		t.Fatal("hosting from MX")
	}
	if !IsDisposable("Mailinator.com") || IsDisposable("gmail.com") {
		t.Fatal("disposable")
	}
	if !TrapLike("spamtrap.ny@example.org") || TrapLike("sam@acme.io") {
		t.Fatal("trap-like")
	}
}

func TestTypoFix(t *testing.T) {
	for in, want := range map[string]string{
		"gmial.com": "gmail.com", "outlok.com": "outlook.com", "yaho.com": "yahoo.com", "gmail.con": "gmail.com",
		"gmail.co": "gmail.com", "hotmial.com": "hotmail.com", "iclod.com": "icloud.com", "gmaill.com": "gmail.com",
	} {
		got, ok := TypoFix(in)
		if !ok || got != want {
			t.Errorf("TypoFix(%s) = %s, %v; want %s", in, got, ok, want)
		}
	}
	for _, in := range []string{"gmail.com", "acme.io", "mail.com", "live.com", "gmx.de", "ymail.com"} {
		if got, ok := TypoFix(in); ok {
			t.Errorf("TypoFix(%s) = %s, want no fix", in, got)
		}
	}
}

func TestMask(t *testing.T) {
	for in, want := range map[string]string{
		"r.walker@outlook.com": "r.w···@outlook.com",
		"jo@gmail.com":         "j···@gmail.com",
		"not-an-address":       "not-an-address",
	} {
		if got := Mask(in); got != want {
			t.Errorf("Mask(%s) = %s, want %s", in, got, want)
		}
	}
}
