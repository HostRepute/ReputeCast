package webpushtest

import (
	"context"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/push/webpush"
)

func TestFakeService(t *testing.T) {
	svc := New()
	k, _ := webpush.GenerateVAPIDKey()
	s := &webpush.Sender{Key: k, Contact: "mailto:support@acme.io", HTTP: svc.Client()}
	ep, p256dh, auth := svc.Subscribe("firefox")
	sub, err := webpush.ParseSubscription(ep, p256dh, auth)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Send(context.Background(), sub, webpush.Message{Payload: []byte(`{"title":"Hello"}`), TTL: time.Hour})
	if err != nil || res.Outcome != webpush.Accepted || res.Location == "" {
		t.Fatalf("send: %+v %v", res, err)
	}
	got := svc.Received()
	if len(got) != 1 || string(got[0].Payload) != `{"title":"Hello"}` || got[0].Claims.Sub != "mailto:support@acme.io" ||
		got[0].Claims.Aud != "https://updates.push.services.mozilla.com" {
		t.Fatalf("received %+v", got)
	}
	// A subscription that ended answers 410; the sender reads Gone.
	svc.Answer(ep, 410)
	res, _ = s.Send(context.Background(), sub, webpush.Message{Payload: []byte("x")})
	if res.Outcome != webpush.Gone || res.Detail != "410 Gone · Unregistered" {
		t.Fatalf("gone: %+v", res)
	}
	// A token signed by another key for the wrong service is refused.
	other := &webpush.Sender{Key: k, Contact: "mailto:x@example.com", HTTP: svc.Client(), Now: func() time.Time { return time.Now().Add(-48 * time.Hour) }}
	ep2, p2, a2 := svc.Subscribe("chrome")
	sub2, _ := webpush.ParseSubscription(ep2, p2, a2)
	res, _ = other.Send(context.Background(), sub2, webpush.Message{Payload: []byte("x")})
	if res.Outcome != webpush.Rejected || res.Status != 403 {
		t.Fatalf("expired token: %+v", res)
	}
}
