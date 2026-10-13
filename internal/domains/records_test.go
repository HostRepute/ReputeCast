package domains

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/store"
)

func TestMergeSPF(t *testing.T) {
	cases := []struct {
		current string
		mechs   []string
		want    string
	}{
		{"", []string{"include:amazonses.com"}, "v=spf1 include:amazonses.com -all"},
		{"v=spf1 include:_spf.google.com ~all", []string{"include:amazonses.com"}, "v=spf1 include:_spf.google.com include:amazonses.com ~all"},
		{"v=spf1 include:amazonses.com -all", []string{"include:amazonses.com"}, "v=spf1 include:amazonses.com -all"},
		{"v=spf1 +include:amazonses.com -all", []string{"include:amazonses.com"}, "v=spf1 +include:amazonses.com -all"},
		{"v=spf1 ip4:203.0.113.25 redirect=_spf.acme.io", []string{"a:mta.acme.io"}, "v=spf1 ip4:203.0.113.25 a:mta.acme.io redirect=_spf.acme.io"},
		{"v=spf1 mx", []string{"include:spf.mtasv.net"}, "v=spf1 mx include:spf.mtasv.net -all"},
	}
	for _, c := range cases {
		if got := MergeSPF(c.current, c.mechs); got != c.want {
			t.Errorf("MergeSPF(%q, %v) = %q, want %q", c.current, c.mechs, got, c.want)
		}
	}
	if got := SPFMechanisms("v=spf1 include:amazonses.com ~all"); len(got) != 1 || got[0] != "include:amazonses.com" {
		t.Errorf("SPFMechanisms: %v", got)
	}
	if !SPFHas("v=spf1 INCLUDE:AmazonSES.com -all", "include:amazonses.com") {
		t.Error("SPFHas is case-insensitive")
	}
}

func TestRecordLabels(t *testing.T) {
	recs := []Record{
		{ProviderID: "a", Purpose: PurposeDKIM, Type: "CNAME", Name: "x._domainkey.news.acme.io"},
		{ProviderID: "a", Purpose: PurposeDKIM, Type: "CNAME", Name: "y._domainkey.news.acme.io"},
		{ProviderID: "a", Purpose: PurposeReturnPath, Type: "MX", Name: "bounce.news.acme.io"},
		{ProviderID: "a", Purpose: PurposeSPF, Type: "TXT", Name: "bounce.news.acme.io"},
		{ProviderID: "a", Purpose: PurposeSPF, Type: "TXT", Name: "news.acme.io"},
		{Purpose: PurposeDMARC, Type: "TXT", Name: "_dmarc.news.acme.io"},
	}
	labels, notes := recordLabels(recs, "news.acme.io", func(string) string { return "Easy DKIM, 2048-bit" })
	want := []string{"DKIM 1 of 2", "DKIM 2 of 2", "Return-path / MAIL FROM", "SPF for MAIL FROM", "SPF for the sending domain", "DMARC"}
	for i, w := range want {
		if labels[i] != w {
			t.Errorf("label %d = %q, want %q", i, labels[i], w)
		}
	}
	if notes[0] != "Easy DKIM, 2048-bit" || notes[1] != "" || notes[4] != "merged with your existing record" || notes[5] != "required by Gmail and Yahoo for bulk senders" {
		t.Errorf("notes: %q", notes)
	}
	if zl := zoneLine(Record{Type: "MX", Name: "bounce.news.acme.io", Value: "10 feedback-smtp.us-east-1.amazonses.com"}); zl != "bounce.news.acme.io. 3600 IN MX 10 feedback-smtp.us-east-1.amazonses.com." {
		t.Errorf("zone line: %s", zl)
	}
}

func TestCells(t *testing.T) {
	if c := dkimCell(3, 3, "", 0, 0); c.Text != "3 of 3 providers" || c.Tone != "ok" {
		t.Errorf("dkim all: %+v", c)
	}
	if c := dkimCell(1, 0, "CNAME", 2, 3); c.Text != "Pending CNAME 2 of 3" || c.Tone != "warn" {
		t.Errorf("dkim pending: %+v", c)
	}
	if c := spfCell("pass", "-all"); c.Text != "pass · -all" {
		t.Errorf("spf: %+v", c)
	}
	if c := dmarcCell("none"); c.Text != "p=none" || c.Tone != "warn" {
		t.Errorf("dmarc none: %+v", c)
	}
	if c := returnPathCell([]MailFromHost{{Host: "bounce.news.acme.io", Status: "aligned"}, {Host: "rp.news.acme.io", Status: "aligned"}}, "news.acme.io"); c.Text != "bounce. · rp." {
		t.Errorf("return path: %+v", c)
	}
	if c := returnPathCell(nil, "acme.io"); c.Text != "Default" {
		t.Errorf("default return path: %+v", c)
	}
}

func TestDNSHostName(t *testing.T) {
	if DNSHostName([]string{"dana.ns.cloudflare.com"}) != "Cloudflare" || DNSHostName([]string{"ns-12.awsdns-01.org"}) != "Amazon Route 53" || DNSHostName([]string{"ns1.example.net"}) != "" {
		t.Error("DNS host names")
	}
}

func TestNormalizeDomain(t *testing.T) {
	for in, want := range map[string]string{"News.Acme.io.": "news.acme.io", "https://acme.io": "acme.io", "hello@news.acme.io": "news.acme.io", "bücher.example": "xn--bcher-kva.example"} {
		got, err := NormalizeDomain("d", in)
		if err != nil || got != want {
			t.Errorf("NormalizeDomain(%q) = %q, %v", in, got, err)
		}
	}
	for _, bad := range []string{"localhost", "a b.com", "-x.com", ""} {
		if _, err := NormalizeDomain("d", bad); err == nil {
			t.Errorf("NormalizeDomain(%q) should fail", bad)
		}
	}
}

func TestVerifyRecords(t *testing.T) {
	z := dnscheck.NewZone()
	z.Add("news.example.com", "TXT", "v=spf1 include:_spf.google.com -all")
	z.Add("_dmarc.example.com", "TXT", "v=DMARC1; p=reject")
	z.Add("k1._domainkey.news.example.com", "CNAME", "k1.dkim.amazonses.com")
	s := &Service{DNS: z}
	ctx := context.Background()
	spf := s.verify(ctx, "news.example.com", Record{Purpose: PurposeSPF, Type: "TXT", Name: "news.example.com", Mechanisms: []string{"include:amazonses.com"}})
	if spf.Status != StatusWrong || spf.Value != "v=spf1 include:_spf.google.com include:amazonses.com -all" || spf.Found == "" {
		t.Errorf("domain SPF: %+v", spf)
	}
	dm := s.verify(ctx, "news.example.com", Record{Purpose: PurposeDMARC, Type: "TXT", Name: "_dmarc.news.example.com", Value: SuggestDMARC("")})
	if dm.Status != StatusVerified || dm.FoundName != "_dmarc.example.com" || !strings.Contains(dm.Found, "p=reject") {
		t.Errorf("inherited DMARC: %+v", dm)
	}
	dk := s.verify(ctx, "news.example.com", Record{Purpose: PurposeDKIM, Type: "CNAME", Name: "k1._domainkey.news.example.com", Value: "k1.dkim.amazonses.com"})
	if dk.Status != StatusVerified {
		t.Errorf("DKIM: %+v", dk)
	}
	missing := s.verify(ctx, "news.example.com", Record{Purpose: PurposeDKIM, Type: "CNAME", Name: "k2._domainkey.news.example.com", Value: "k2.dkim.amazonses.com"})
	if missing.Status != StatusMissing {
		t.Errorf("missing DKIM: %+v", missing)
	}
	if tip := dmarcTip("news.example.com", []Record{dm}, AuthDetail{DMARCPolicy: "reject", DMARCInherited: true}, nil); tip != "news.example.com follows `p=reject` from example.com's DMARC record." {
		t.Errorf("tip: %q", tip)
	}
	want := "start with `p=none` if other services also send as news.example.com; ReputeCast suggested `p=quarantine` because only SES is configured. Reports go to HostRepute and appear in Deliverability."
	if tip := dmarcTip("news.example.com", []Record{{Purpose: PurposeDMARC, Value: SuggestDMARC("dmarc@reports.hostrepute.com"), Status: StatusMissing}}, AuthDetail{},
		[]SenderProviderColumn{{Short: "SES"}}); tip != want {
		t.Errorf("tip: %q", tip)
	}
}

func TestCloudflareApply(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		body, _ := io.ReadAll(r.Body)
		calls = append(calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery+" "+string(body))
		if r.Header.Get("Authorization") != "Bearer good-token-0123456789" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":9109,"message":"Invalid access token"}]}`))
			return
		}
		switch {
		case r.URL.Path == "/zones" && r.URL.Query().Get("name") == "acme.io":
			_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"z1","name":"acme.io"}]}`))
		case r.URL.Path == "/zones":
			_, _ = w.Write([]byte(`{"success":true,"result":[]}`))
		case r.Method == http.MethodGet && r.URL.Query().Get("type") == "TXT" && r.URL.Query().Get("name") == "news.acme.io":
			_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"r9","type":"TXT","name":"news.acme.io","content":"\"v=spf1 include:_spf.google.com -all\""}]}`))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"success":true,"result":[]}`))
		default:
			_, _ = w.Write([]byte(`{"success":true,"result":{}}`))
		}
	}))
	defer srv.Close()
	cf := &Cloudflare{BaseURL: srv.URL}
	recs := []Record{
		{Purpose: PurposeDKIM, Type: "CNAME", Name: "k1._domainkey.news.acme.io", Value: "k1.dkim.amazonses.com", Status: StatusMissing},
		{Purpose: PurposeReturnPath, Type: "MX", Name: "bounce.news.acme.io", Value: "10 feedback-smtp.us-east-1.amazonses.com", Status: StatusMissing},
		{Purpose: PurposeSPF, Type: "TXT", Name: "news.acme.io", Value: "v=spf1 include:_spf.google.com include:amazonses.com -all", Status: StatusWrong},
		{Purpose: PurposeDMARC, Type: "TXT", Name: "_dmarc.news.acme.io", Value: SuggestDMARC(""), Status: StatusVerified},
	}
	res, err := cf.Apply(context.Background(), "good-token-0123456789", "news.acme.io", recs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Zone != "acme.io" || len(res.Created) != 2 || len(res.Updated) != 1 || len(res.Skipped) != 1 || len(res.Failed) != 0 {
		t.Fatalf("result: %+v\ncalls: %s", res, strings.Join(calls, "\n"))
	}
	joined := strings.Join(calls, "\n")
	if !strings.Contains(joined, `PATCH /zones/z1/dns_records/r9`) || !strings.Contains(joined, `"priority":10`) || !strings.Contains(joined, `"proxied":false`) {
		t.Errorf("calls: %s", joined)
	}
	if _, err := cf.Apply(context.Background(), "wrong-token-0123456789", "news.acme.io", recs); err != ErrCloudflareToken {
		t.Errorf("bad token: %v", err)
	}
}

func TestSendingDefaultsRoundTrip(t *testing.T) {
	d := decodeSendingDefaults(json.RawMessage(`{"reply_to":"a@b.c","throttles":{"outlook":5},"seed_list":{"gmail":"x@gmail.com"}}`))
	d.TrackingDomain = "go.news.acme.io"
	var back map[string]any
	_ = json.Unmarshal(d.encode(), &back)
	if back["tracking_domain"] != "go.news.acme.io" || back["reply_to"] != "a@b.c" || back["throttles"] == nil || back["seed_list"] == nil {
		t.Errorf("defaults: %v", back)
	}
	_ = store.Sender{}
}
