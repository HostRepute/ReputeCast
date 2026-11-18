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

// Tool calls whose fragments interleave: index 0 starts, index 1 starts
// (growing the builders' slice), then index 0 goes on. Each call keeps its
// own arguments.
func TestOpenAIStreamInterleavedToolCalls(t *testing.T) {
	s := &openAIStream{name: "OpenAI", emit: func(Delta) error { return nil }}
	chunks := []string{
		`{"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"a","type":"function","function":{"name":"one","arguments":"{\"x\""}}]}}]}`,
		`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":1,"id":"b","type":"function","function":{"name":"two","arguments":"{\"y\":2}"}}]}}]}`,
		`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":":1}"}}]},"finish_reason":"tool_calls"}]}`,
	}
	for _, c := range chunks {
		if err := s.event(sseEvent{data: []byte(c)}); err != nil {
			t.Fatal(err)
		}
	}
	var got []string
	for _, p := range s.response().Message.Parts {
		if p.ToolCall != nil {
			got = append(got, p.ToolCall.Name+" "+string(p.ToolCall.Arguments))
		}
	}
	if want := []string{`one {"x":1}`, `two {"y":2}`}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls: %q, want %q", got, want)
	}
}
