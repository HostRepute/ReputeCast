package llm

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

// What no provider's recording exercises: several data lines, a field
// without its space, id and retry, an event without data, and an event
// cut by the end of the stream.
func TestSSEReader(t *testing.T) {
	r := newSSEReader(strings.NewReader("id: 7\nretry: 100\nevent: ping\n\n: a comment\nevent: two\ndata:a\ndata: b\r\n\r\n{\"error\":\ndata: [DONE]\n\ndata: {\"cut\""))
	var got []sseEvent
	for {
		ev, err := r.next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, ev)
	}
	want := []sseEvent{{name: "two", data: []byte("a\nb")}, {data: []byte("[DONE]")}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events: %q", got)
	}
	if string(r.stray) != "{\"error\":\ndata: {\"cut\"" {
		t.Fatalf("stray: %q", r.stray)
	}
}

type generator struct{ res Response }

func (g generator) Generate(context.Context, Request) (Response, error) { return g.res, nil }
func (generator) Models(context.Context) ([]Model, error)               { return nil, nil }

func TestStreamOrGenerate(t *testing.T) {
	want := Response{Text: "Ready.", StopReason: StopEnd, UsageKnown: true}
	var shown []Delta
	res, err := StreamOrGenerate(context.Background(), generator{want}, Request{}, func(d Delta) error {
		shown = append(shown, d)
		return nil
	})
	if err != nil || !reflect.DeepEqual(res, want) || !reflect.DeepEqual(shown, []Delta{{Text: "Ready."}}) {
		t.Fatalf("a provider that doesn't stream: %+v %v %+v", res, err, shown)
	}
	stop := errors.New("stop")
	if _, err := StreamOrGenerate(context.Background(), generator{want}, Request{}, func(Delta) error { return stop }); !errors.Is(err, stop) {
		t.Fatalf("emit's error: %v", err)
	}
}
