package apiclient

import (
	"cmp"
	"context"
	"errors"
	"strconv"
	"sync"

	"github.com/hostrepute/reputecast/internal/providers"
)

// SendOne sends one message. It returns the message's result (with Err set
// for a refusal that concerns this message only), or an error when the call
// failed for a reason that concerns the account or the service: rejected
// credentials, an outage, throttling.
type SendOne func(ctx context.Context, m providers.Message) (providers.SendResult, error)

// MessageError is the permanent SendError for a send call refused because
// of the message itself (a 4xx that is neither rejected credentials nor
// throttling), or nil when err concerns the account or the service.
func MessageError(err error) *providers.SendError {
	var ce *providers.CallError
	if !errors.As(err, &ce) || ce.Auth || ce.Temporary || ce.Status < 400 || ce.Status >= 500 {
		return nil
	}
	return &providers.SendError{Code: cmp.Or(ce.Code, "HTTP "+strconv.Itoa(ce.Status)), Message: ce.Message}
}

// Failed is what a SendOne returns when the call for m failed with err:
// the message's own permanent error (see MessageError), or err for the
// account or the service.
func Failed(m providers.Message, err error) (providers.SendResult, error) {
	if se := MessageError(err); se != nil {
		return providers.SendResult{MessageID: m.ID, Err: se}, nil //nolint:nilerr // the refusal is reported on the message's result
	}
	return providers.SendResult{}, err
}

// Fanout sends msgs with one API call each, at most parallel at a time,
// for providers whose API takes one personalized message per call. Results
// keep the input order.
//
// After an account- or service-level failure no new call starts (calls in
// flight finish, so nothing is sent twice when the rest is retried) and
// the messages not sent get a temporary error. When no message got an
// outcome of its own, the whole call fails with the first such error, so
// errors.Is finds providers.ErrUnauthorized or providers.ErrUnavailable.
func Fanout(ctx context.Context, msgs []providers.Message, parallel int, send SendOne) ([]providers.SendResult, error) {
	results := make([]providers.SendResult, len(msgs))
	if len(msgs) == 0 {
		return results, nil
	}
	parallel = max(1, min(parallel, len(msgs)))
	callErrs := make([]error, len(msgs))
	attempted := make([]bool, len(msgs))
	var (
		mu      sync.Mutex
		stopped bool
		wg      sync.WaitGroup
	)
	// Every result names its message, including the ones never attempted
	// after a stop: the caller matches results to messages by MessageID.
	for i := range msgs {
		results[i].MessageID = msgs[i].ID
	}
	sem := make(chan struct{}, parallel)
	for i := range msgs {
		sem <- struct{}{}
		mu.Lock()
		stop := stopped || ctx.Err() != nil
		mu.Unlock()
		if stop {
			<-sem
			break
		}
		attempted[i] = true
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			res, err := send(ctx, msgs[i])
			if err != nil {
				callErrs[i] = err
				mu.Lock()
				stopped = true
				mu.Unlock()
				return
			}
			res.MessageID = msgs[i].ID
			results[i] = res
		}(i)
	}
	wg.Wait()
	var first error
	outcome := false
	for i := range msgs {
		if attempted[i] && callErrs[i] == nil {
			outcome = true
		}
		if first == nil && callErrs[i] != nil {
			first = callErrs[i]
		}
	}
	if first == nil && ctx.Err() != nil {
		first = &providers.CallError{Op: "send", Message: "canceled before every message was sent", Temporary: true, Cause: ctx.Err()}
	}
	if first == nil {
		return results, nil
	}
	if !outcome {
		return nil, first
	}
	var ce *providers.CallError
	errors.As(first, &ce)
	for i := range msgs {
		if attempted[i] && callErrs[i] == nil {
			continue
		}
		err := callErrs[i]
		if err == nil {
			err = first
		}
		var own *providers.CallError
		if !errors.As(err, &own) {
			own = ce
		}
		se := &providers.SendError{Temporary: true, Message: err.Error()}
		if own != nil {
			se.Throttled, se.Code = own.Throttled, own.Code
			if se.Code == "" && own.Status != 0 {
				se.Code = "HTTP " + strconv.Itoa(own.Status)
			}
		}
		if se.Code == "" {
			se.Code = "Unavailable"
		}
		results[i].Err = se
	}
	return results, nil
}
