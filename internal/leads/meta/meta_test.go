package meta

import (
	"testing"
	"time"
)

func TestParseWebhookReadsIDsAsStringsOrNumbers(t *testing.T) {
	body := []byte(`{"object":"page","entry":[{"id":153125381133,"time":1438292065,"changes":[
		{"field":"leadgen","value":{"leadgen_id":123123123123,"page_id":123123123,"form_id":"12312312312","ad_id":12312312312,"created_time":1440120384}},
		{"field":"feed","value":{"post_id":"1"}},
		{"field":"leadgen","value":{"leadgen_id":"9","form_id":"7","created_time":1440120385}}]}]}`)
	got, err := ParseWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("changes: %+v", got)
	}
	if got[0].LeadgenID != "123123123123" || got[0].PageID != "123123123" || got[0].FormID != "12312312312" || !got[0].CreatedTime.Equal(time.Unix(1440120384, 0)) {
		t.Fatalf("first: %+v", got[0])
	}
	// A change without page_id belongs to its entry's Page.
	if got[1].PageID != "153125381133" {
		t.Fatalf("second: %+v", got[1])
	}
	if _, err := ParseWebhook([]byte(`{"object":"whatsapp_business_account","entry":[]}`)); err != ErrNotLeads {
		t.Fatalf("not a Page webhook: %v", err)
	}
}

func TestTestLeads(t *testing.T) {
	cases := []struct {
		fields []Answer
		test   bool
	}{
		{[]Answer{{Name: "email", Values: []string{"test@meta.com"}}}, true},
		{[]Answer{{Name: "email", Values: []string{"sam@example.com"}}, {Name: "full_name", Values: []string{"<test lead: dummy data for full_name>"}}}, true},
		{[]Answer{{Name: "email", Values: []string{"sam@example.com"}}, {Name: "full_name", Values: []string{"Sam Rivera"}}}, false},
	}
	for _, c := range cases {
		if got := (Lead{Fields: c.fields}).IsTest(); got != c.test {
			t.Fatalf("%+v: %v", c.fields, got)
		}
	}
}

func TestProofAndSignature(t *testing.T) {
	body := []byte(`{"object":"page"}`)
	sig := Sign("secret", body)
	if !SignatureShape(sig) || !VerifySignature("secret", body, sig) || VerifySignature("other", body, sig) {
		t.Fatal("signature")
	}
	if Proof("secret", "token") == Proof("secret", "token2") {
		t.Fatal("proof depends on the token")
	}
}
