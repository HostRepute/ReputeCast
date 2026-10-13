package dsn

import (
	"errors"
	"os"
	"testing"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseDSN(t *testing.T) {
	rep, err := Parse(read(t, "postfix-hard.eml"))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Kind != KindDSN || rep.ReportingMTA != "mail.acme.io" || rep.Mailbox != "bounces@news.acme.io" {
		t.Fatalf("report: %+v", rep)
	}
	if len(rep.Recipients) != 1 {
		t.Fatalf("recipients: %+v", rep.Recipients)
	}
	r := rep.Recipients[0]
	if r.FinalRecipient != "t.lee@example.org" || r.Action != "failed" || r.Status != "5.1.1" || r.RemoteMTA != "mx.example.org" {
		t.Fatalf("recipient: %+v", r)
	}
	if r.DiagnosticCode != "smtp; 550 5.1.1 <t.lee@example.org>: Recipient address rejected: User unknown in virtual mailbox table" {
		t.Fatalf("diagnostic: %q", r.DiagnosticCode)
	}
	if rep.Original.Get("X-Reputecast-Message") != "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e4f" {
		t.Fatalf("original headers: %v", rep.Original)
	}
}

func TestParseARF(t *testing.T) {
	rep, err := Parse(read(t, "arf.eml"))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Kind != KindARF || rep.FeedbackType != "abuse" || rep.ARFRecipient != "m.g@aol.com" || rep.UserAgent != "Yahoo!-Mail-Feedback/2.0" {
		t.Fatalf("arf: %+v", rep)
	}
	if rep.Original.Get("X-Reputecast-Message") != "0199d5a8-2b6e-7c41-9a0e-000000000001" {
		t.Fatalf("original: %v", rep.Original)
	}
}

func TestParseHeuristic(t *testing.T) {
	rep, err := Parse(read(t, "exim-delayed.eml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Recipients) == 0 || rep.Recipients[0].FinalRecipient != "b.h@yahoo.com" {
		t.Fatalf("recipients: %+v", rep.Recipients)
	}
	r := rep.Recipients[0]
	if r.Status != "4.2.2" || r.Action != "delayed" {
		t.Fatalf("heuristic recipient: %+v", r)
	}
	if rep.Original.Get("X-Reputecast-Message") != "0199d5a8-2b6e-7c41-9a0e-000000000002" {
		t.Fatalf("original: %v", rep.Original)
	}
}

func TestNotAReport(t *testing.T) {
	raw := []byte("From: a@b.c\r\nTo: bounces@news.acme.io\r\nSubject: Out of office\r\n\r\nI'm away until Monday.\r\n")
	if _, err := Parse(raw); !errors.Is(err, ErrNotAReport) {
		t.Fatalf("want ErrNotAReport, got %v", err)
	}
}
