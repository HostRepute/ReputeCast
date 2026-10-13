package apiclient

import (
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
	sem := make(chan struct{}, parallel)
	for i := range msgs {
		results[i].MessageID = msgs[i].ID
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
