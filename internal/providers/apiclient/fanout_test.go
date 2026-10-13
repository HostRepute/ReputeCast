package apiclient_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
)

func msgs(n int) []providers.Message {
	out := make([]providers.Message, n)
	for i := range out {
		out[i] = providers.Message{ID: uuid.Must(uuid.NewV7())}
	}
	return out
}

func TestFanoutKeepsOrderAndMessageErrors(t *testing.T) {
	in := msgs(25)
	res, err := apiclient.Fanout(context.Background(), in, 4, func(_ context.Context, m providers.Message) (providers.SendResult, error) {
		if m.ID == in[3].ID {
			return providers.SendResult{Err: &providers.SendError{Code: "invalid", Message: "bad address"}}, nil
		}
		return providers.SendResult{ProviderMessageID: "p-" + m.ID.String()}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range res {
		if r.MessageID != in[i].ID {
			t.Fatalf("result %d is out of order", i)
		}
		if i == 3 {
			if r.Err == nil || r.Err.Temporary {
				t.Fatalf("message 3 must keep its permanent error: %+v", r)
			}
			continue
		}
		if r.Err != nil || r.ProviderMessageID != "p-"+in[i].ID.String() {
			t.Fatalf("result %d: %+v", i, r)
		}
	}
}

func TestFanoutAccountFailure(t *testing.T) {
	auth := &providers.CallError{Op: "send", Status: 401, Auth: true, Message: "bad key"}
	res, err := apiclient.Fanout(context.Background(), msgs(5), 2, func(context.Context, providers.Message) (providers.SendResult, error) {
		return providers.SendResult{}, auth
	})
	if res != nil || !errors.Is(err, providers.ErrUnauthorized) {
		t.Fatalf("no outcome at all must fail the call: %v, %v", res, err)
	}
}

func TestFanoutStopsAfterOutage(t *testing.T) {
	in := msgs(40)
	var calls atomic.Int32
	outage := &providers.CallError{Op: "send", Status: 503, Temporary: true, Message: "down"}
	res, err := apiclient.Fanout(context.Background(), in, 1, func(_ context.Context, m providers.Message) (providers.SendResult, error) {
		if calls.Add(1) > 2 {
			return providers.SendResult{}, outage
		}
		return providers.SendResult{ProviderMessageID: "ok"}, nil
	})
	if err != nil {
		t.Fatalf("accepted messages must be reported, not lost: %v", err)
	}
	if n := calls.Load(); n != 3 {
		t.Fatalf("no call may start after the outage; got %d calls", n)
	}
	for i, r := range res {
		if r.MessageID != in[i].ID {
			t.Fatalf("result %d must name its message, sent or not: %+v", i, r)
		}
		if i < 2 {
			if r.Err != nil {
				t.Fatalf("message %d was accepted: %+v", i, r)
			}
			continue
		}
		if r.Err == nil || !r.Err.Temporary || r.Err.Code != "HTTP 503" {
			t.Fatalf("message %d must be retried later: %+v", i, r.Err)
		}
	}
}

func TestMessageError(t *testing.T) {
	for _, c := range []struct {
		err  error
		code string // "" when the error isn't the message's
	}{
		{&providers.CallError{Status: 422, Code: "validation_error", Message: "bad to"}, "validation_error"},
		{&providers.CallError{Status: 400, Message: "bad"}, "HTTP 400"},
		{&providers.CallError{Status: 401, Auth: true}, ""},
		{&providers.CallError{Status: 429, Temporary: true, Throttled: true}, ""},
		{&providers.CallError{Status: 503, Temporary: true}, ""},
		{errors.New("network"), ""},
	} {
		se := apiclient.MessageError(c.err)
		if (se == nil) != (c.code == "") || (se != nil && (se.Code != c.code || se.Temporary)) {
			t.Fatalf("%v: %+v", c.err, se)
		}
	}
}
