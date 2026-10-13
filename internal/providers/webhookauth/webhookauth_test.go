package webhookauth_test

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

// The test vector from Svix's documentation
// (https://docs.svix.com/receiving/verifying-payloads/how-manual).
const (
	svixSecret  = "whsec_plJ3nmyCDGBKInavdOK15jsl"
	svixID      = "msg_loFOjxBNrRLzqYUf"
	svixTS      = "1731705121"
	svixPayload = `{"event_type":"ping","data":{"success":true}}`
	svixSig     = "v1,rAvfW3dJ/X/qxhsaXPOyyCGmRKsaKWcsNccKXlIktD0="
)

func TestStandardMatchesSvixVector(t *testing.T) {
	got, err := webhookauth.SignStandard(svixSecret, svixID, svixTS, []byte(svixPayload))
	if err != nil || got != svixSig {
		t.Fatalf("signature: %q, %v", got, err)
	}
	at := time.Unix(1731705121, 0).Add(time.Minute)
	for _, prefix := range []string{"svix-", "webhook-"} {
		h := http.Header{}
		h.Set(prefix+"id", svixID)
		h.Set(prefix+"timestamp", svixTS)
		h.Set(prefix+"signature", "v1,c29tZS1vbGQtc2VjcmV0 "+svixSig)
		if err := webhookauth.Standard(svixSecret, h, []byte(svixPayload), at); err != nil {
			t.Fatalf("%s headers: %v", prefix, err)
		}
	}
	h := http.Header{"Svix-Id": {svixID}, "Svix-Timestamp": {svixTS}, "Svix-Signature": {svixSig}}
	if err := webhookauth.Standard(svixSecret, h, []byte(svixPayload+" "), at); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("a changed body: %v", err)
	}
	if err := webhookauth.Standard(svixSecret, h, []byte(svixPayload), at.Add(time.Hour)); !errors.Is(err, webhookauth.ErrStale) {
		t.Fatalf("a replay an hour later: %v", err)
	}
	if err := webhookauth.Standard("not-a-secret!", h, []byte(svixPayload), at); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("a malformed secret: %v", err)
	}
	if err := webhookauth.Standard(svixSecret, http.Header{}, []byte(svixPayload), at); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("no headers: %v", err)
	}
}

func TestFresh(t *testing.T) {
	now := time.Unix(1_760_000_000, 0)
	for ts, ok := range map[string]bool{
		strconv.FormatInt(now.Unix(), 10):                     true,
		strconv.FormatInt(now.Add(4*time.Minute).Unix(), 10):  true,
		strconv.FormatInt(now.Add(-6*time.Minute).Unix(), 10): false,
		strconv.FormatInt(now.UnixMilli(), 10):                true,
		"not a number":                                        false,
		"NaN":                                                 false,
	} {
		if err := webhookauth.Fresh(ts, now); (err == nil) != ok {
			t.Fatalf("Fresh(%q) = %v", ts, err)
		}
	}
}

func TestSharedSecrets(t *testing.T) {
	if webhookauth.Equal("", "") || webhookauth.Equal("x", "") {
		t.Fatal("an empty expected secret never matches")
	}
	if !webhookauth.Equal("s3cret", "s3cret") || webhookauth.Equal("s3cret", "s3cre") {
		t.Fatal("equal")
	}
	r, _ := http.NewRequest(http.MethodPost, "https://example.com/", nil)
	if err := webhookauth.Basic(r, "reputecast", "pw"); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatal("no basic auth")
	}
	r.SetBasicAuth("reputecast", "pw")
	if err := webhookauth.Basic(r, "reputecast", "pw"); err != nil {
		t.Fatal(err)
	}
	if err := webhookauth.Basic(r, "reputecast", ""); err == nil {
		t.Fatal("no configured password refuses everything")
	}
	for _, enc := range []string{"hex", "base64"} {
		if _, err := webhookauth.Decode("zz", enc); err == nil && enc == "hex" {
			t.Fatal("bad hex")
		}
	}
	if b, err := webhookauth.Decode("aGk", "base64"); err != nil || string(b) != "hi" {
		t.Fatalf("unpadded base64: %q %v", b, err)
	}
	if _, err := webhookauth.Hash("md5"); err == nil {
		t.Fatal("unknown algorithm")
	}
}
