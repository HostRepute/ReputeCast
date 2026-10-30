package engine

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/hostrepute/reputecast/internal/providers"
)

// outcomeUnknown reports whether a failed provider call may have reached
// the provider: it timed out or the connection broke after the request was
// written, so the provider may have taken the mail (review GS-1). A refused
// connection, a failed DNS lookup or dial, and any answer the provider gave
// (an HTTP status, an SMTP reply) are known: nothing went out, or the
// provider said what happened.
func outcomeUnknown(err error) bool {
	if err == nil {
		return false
	}
	var op *net.OpError
	if errors.As(err, &op) && op.Op == "dial" {
		return false
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return false
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	var ce *providers.CallError
	if errors.As(err, &ce) {
		if ce.Status != 0 || ce.Auth || ce.Throttled {
			return false
		}
		return unknownText(ce.Message)
	}
	return unknownText(err.Error())
}

// sendErrUnknown is outcomeUnknown for one message's error in a batch the
// provider answered one call per message for (SES, Brevo).
func sendErrUnknown(se *providers.SendError) bool {
	return se != nil && se.Temporary && !se.Throttled && unknownText(se.Code+" "+se.Message)
}

// unknownText reads an error's words when no typed error survived the
// adapter: a timeout, a deadline, a reset or cut connection. A dial or a
// refused connection never reached the provider.
func unknownText(msg string) bool {
	m := strings.ToLower(msg)
	if strings.Contains(m, "dial ") || strings.Contains(m, "connection refused") || strings.Contains(m, "econnrefused") ||
		strings.Contains(m, "no such host") {
		return false
	}
	for _, w := range []string{"deadline exceeded", "timeout", "timed out", "connection reset", "unexpected eof", "broken pipe", "econnreset"} {
		if strings.Contains(m, w) {
			return true
		}
	}
	return strings.HasSuffix(strings.TrimSpace(m), ": eof") || strings.TrimSpace(m) == "eof"
}
