package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Delta is a piece of an answer as it arrives.
type Delta struct {
	Text          string    // a piece of the visible answer
	ToolCallStart *ToolCall // a tool call's ID and name, before its arguments
	ToolArgs      string    // a piece of the current tool call's JSON arguments
}

// Streamer is a Provider that shows its answer as it arrives. Every
// adapter in this package is one; callers that don't stream keep calling
// Generate.
//
// Stream sends req like Generate and calls emit, in order and on the
// caller's goroutine, with each piece of visible text and of each tool
// call. Thought parts (Gemini's) stay in the Message and are never
// emitted. It returns the Response Generate would return: the text (what
// was emitted), the replayable Message with tool calls in order and
// checked, the usage of the provider's last event, the stop reason.
//
// When emit returns an error, reading stops, the connection is closed and
// Stream returns that error. When the provider sends an error event, goes
// silent for Config.IdleTimeout or the connection is cut, Stream returns
// an *Error and the answer so far, whose usage is unknown unless the
// provider had already reported it. A request is retried (Config.Retries)
// only before the stream begins: once the provider answered 2xx it may
// bill for the request, so asking again could bill twice. An error after
// that is never one of the kinds that mean the provider refused the
// request before running it (auth, rate limit, bad request).
type Streamer interface {
	Stream(ctx context.Context, req Request, emit func(Delta) error) (Response, error)
}

var (
	_ Streamer = (*anthropic)(nil)
	_ Streamer = (*openAI)(nil)
	_ Streamer = (*gemini)(nil)
)

// StreamOrGenerate streams req when p can, and otherwise generates the
// answer and emits its whole text as one delta.
func StreamOrGenerate(ctx context.Context, p Provider, req Request, emit func(Delta) error) (Response, error) {
	if s, ok := p.(Streamer); ok {
		return s.Stream(ctx, req, emit)
	}
	res, err := p.Generate(ctx, req)
	if err != nil || res.Text == "" {
		return res, err
	}
	return res, emit(Delta{Text: res.Text})
}

// MaxStreamLineBytes bounds one line of a stream, its line break included.
// A provider's event is one line of JSON; the largest is a tool call's
// arguments (200 KB at most) or a Gemini chunk with a thought signature.
const MaxStreamLineBytes = 1 << 20

// maxStray bounds what is kept of lines that aren't part of an event.
const maxStray = 64 << 10

var (
	// errStreamEnd from a handler stops reading: the provider's last event
	// arrived.
	errStreamEnd = errors.New("end of stream")
	// errIdle is the cause of a stream cancelled for silence.
	errIdle           = errors.New("the provider went silent")
	errStreamTooLarge = errors.New("the stream is over the size limit")
	errLineTooLong    = errors.New("a line of the stream is over the size limit")
)

// sseEvent is one server-sent event: its name (Anthropic's "event:"
// lines) and its data lines, joined.
type sseEvent struct {
	name string
	data []byte
}

// sseReader reads server-sent events
// (https://html.spec.whatwg.org/multipage/server-sent-events.html): lines
// end with LF or CRLF, a blank line ends an event, ":" starts a comment,
// "id" and "retry" are ignored, and an event cut by the end of the stream
// is dropped. Lines that aren't fields are kept in stray (bounded): a
// provider may write an error there as plain JSON (Gemini does).
type sseReader struct {
	br    *bufio.Reader
	line  []byte
	stray []byte
}

func newSSEReader(r io.Reader) *sseReader {
	return &sseReader{br: bufio.NewReaderSize(r, 32<<10)}
}

// readLine returns the next line without its line break.
func (r *sseReader) readLine() ([]byte, error) {
	r.line = r.line[:0]
	for {
		frag, err := r.br.ReadSlice('\n')
		if len(r.line)+len(frag) > MaxStreamLineBytes {
			return nil, errLineTooLong
		}
		r.line = append(r.line, frag...)
		switch {
		case err == nil:
			line := bytes.TrimSuffix(r.line, []byte("\n"))
			return bytes.TrimSuffix(line, []byte("\r")), nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF) && len(r.line) > 0:
			r.keepStray(r.line)
		}
		return nil, err
	}
}

func (r *sseReader) keepStray(line []byte) {
	if len(r.stray)+len(line)+1 > maxStray {
		return
	}
	if len(r.stray) > 0 {
		r.stray = append(r.stray, '\n')
	}
	r.stray = append(r.stray, line...)
}

// next returns the next event, or the error that ended the stream (io.EOF
// at its end).
func (r *sseReader) next() (sseEvent, error) {
	var ev sseEvent
	hasData := false
	for {
		line, err := r.readLine()
		if err != nil {
			return sseEvent{}, err
		}
		if len(line) == 0 {
			if hasData {
				return ev, nil
			}
			ev = sseEvent{}
			continue
		}
		if line[0] == ':' {
			continue
		}
		field, value, ok := bytes.Cut(line, []byte(":"))
		if ok {
			value = bytes.TrimPrefix(value, []byte(" "))
		}
		switch string(field) {
		case "event":
			ev.name = string(value)
		case "data":
			if hasData {
				ev.data = append(ev.data, '\n')
			}
			ev.data = append(ev.data, value...)
			hasData = true
		case "id", "retry":
		default:
			r.keepStray(line)
		}
	}
}

// streamBody bounds a stream's size and notes each byte that arrives.
type streamBody struct {
	r      io.Reader
	n, max int64
	idle   *time.Timer
	wait   time.Duration
}

func (b *streamBody) Read(p []byte) (int, error) {
	if rem := b.max + 1 - b.n; int64(len(p)) > rem {
		p = p[:rem]
	}
	n, err := b.r.Read(p)
	if n > 0 {
		b.idle.Reset(b.wait)
	}
	b.n += int64(n)
	if b.n > b.max {
		return 0, errStreamTooLarge
	}
	return n, err
}

// streamHandler is an adapter's reading of its provider's events.
type streamHandler interface {
	// event reads one event: errStreamEnd at the provider's last one, an
	// *Error for an error event, or emit's error.
	event(ev sseEvent) error
	// complete reports whether the answer ended as the provider ends one.
	complete() bool
}

// stream sends the call and hands each event of a 2xx answer to h until
// the provider's last event, an error or the end of the stream. Overloaded
// and unavailable answers before the stream are retried like do's; after
// it began nothing is.
func (c call) stream(ctx context.Context, h streamHandler) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()
	payload, err := json.Marshal(c.body)
	if err != nil {
		return &Error{Kind: ErrInvalid, Err: err}
	}
	attempts := 0
	for {
		attempts++
		started, err := c.streamOnce(ctx, payload, h)
		e, _ := AsError(err)
		if e != nil {
			e.Attempts = attempts
			c.redact(e)
		}
		retry := !started && e != nil && (e.Kind == ErrOverloaded || e.Kind == ErrUnavailable) && attempts <= c.cfg.retries()
		if !retry {
			return err
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(c.cfg.RetryWait):
		}
	}
}

// streamOnce is one attempt; it reports whether a 2xx answer arrived.
func (c call) streamOnce(ctx context.Context, payload []byte, h streamHandler) (bool, error) {
	ctx, stop := context.WithCancelCause(ctx)
	defer stop(nil)
	idle := time.AfterFunc(c.cfg.IdleTimeout, func() { stop(errIdle) })
	defer idle.Stop()
	res, err := c.open(ctx, payload, "text/event-stream")
	if err != nil {
		if errors.Is(context.Cause(ctx), errIdle) {
			return false, c.idleError()
		}
		return false, err
	}
	defer res.Body.Close() //nolint:errcheck
	idle.Reset(c.cfg.IdleTimeout)
	r := newSSEReader(&streamBody{r: res.Body, max: c.cfg.MaxResponseBytes, idle: idle, wait: c.cfg.IdleTimeout})
	events := 0
	for {
		ev, err := r.next()
		switch {
		case errors.Is(err, io.EOF):
			if h.complete() {
				return true, nil
			}
			return true, c.incomplete(events, r.stray)
		case errors.Is(err, errStreamTooLarge):
			return true, &Error{Kind: ErrTooLarge, Message: fmt.Sprintf("the answer is over %d KB", c.cfg.MaxResponseBytes>>10)}
		case errors.Is(err, errLineTooLong):
			return true, &Error{Kind: ErrTooLarge, Message: fmt.Sprintf("an event of the answer is over %d KB", MaxStreamLineBytes>>10)}
		case err != nil && errors.Is(context.Cause(ctx), errIdle):
			return true, c.idleError()
		case err != nil:
			return true, transportError(ctx, err)
		}
		events++
		if err := h.event(ev); err != nil {
			if errors.Is(err, errStreamEnd) {
				return true, nil
			}
			return true, err
		}
	}
}

func (c call) idleError() *Error {
	return &Error{Kind: ErrTimeout, Err: errIdle, Message: fmt.Sprintf("nothing arrived for %s", c.cfg.IdleTimeout)}
}

// incomplete is the error of a stream that ended before the provider's
// last event: an error the provider wrote outside the event format, an
// answer that isn't a stream at all, or a connection cut short.
func (c call) incomplete(events int, stray []byte) *Error {
	if raw, ok := FirstJSON(string(stray)); ok {
		if e := c.classify(0, http.Header{}, raw); e.Message != "" || e.Code != "" {
			return midStream(e)
		}
	}
	if events == 0 {
		return &Error{Kind: ErrInvalid, Message: "the answer isn't an event stream"}
	}
	return &Error{Kind: ErrNetwork, Message: "the answer stopped before it ended"}
}

// midStream is an error the provider reported after the answer began. It
// may have billed for the request, so the error never reads as one it
// refused before running (which accounting takes as free): those kinds
// become ErrUnavailable, keeping the provider's code and message.
func midStream(e *Error) *Error {
	e.Status = 0
	switch e.Kind {
	case ErrAuth, ErrRateLimit, ErrBadRequest:
		e.Kind = ErrUnavailable
	}
	return e
}

// decodeEvent reads an event's JSON data.
func decodeEvent(name string, ev sseEvent, v any) error {
	if err := json.Unmarshal(ev.data, v); err != nil {
		return &Error{Kind: ErrInvalid, Err: err, Message: name + "'s answer couldn't be read"}
	}
	return nil
}
