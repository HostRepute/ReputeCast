package webhookauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
)

// Refetch is the least time between two reads of one entry: a flood of
// badly signed requests causes at most one provider API call a minute.
const Refetch = time.Minute

// MaxAge is how long material read from a provider is trusted before it is
// read again, so a secret or key the provider rotated out (after a leak,
// say) stops verifying requests.
const MaxAge = time.Hour

// Material caches signing material adapters read from their provider's API
// (a webhook's signing secret or public key, an identity provider's key
// set), shared by every adapter instance: the factory builds one per
// request. Entries are keyed by a hash (see Key), so no credential is kept
// as a map key. The zero value is ready to use.
type Material struct {
	// Now is the clock, replaced in tests.
	Now func() time.Time

	mu      sync.Mutex
	entries map[string]*entry
}

type entry struct {
	mu      sync.Mutex
	value   string
	fetched time.Time // when value was read
	tried   time.Time // when a read was last attempted
	err     error     // that read's failure
}

// Key identifies an entry by what selects the material: the API's base
// URL, the API key, the webhook's path.
func Key(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Material) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Material) entry(key string) *entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]*entry{}
	}
	e, ok := c.entries[key]
	if !ok {
		e = &entry{}
		c.entries[key] = e
	}
	return e
}

// Set stores material the provider just returned (when the webhook is
// registered), saving a read on the first request.
func (c *Material) Set(key, value string) {
	if value == "" {
		return
	}
	e := c.entry(key)
	now := c.now()
	e.mu.Lock()
	e.value, e.fetched, e.tried, e.err = value, now, now, nil
	e.mu.Unlock()
}

// get returns the material for key, read with fetch when there is none or
// it is older than MaxAge (keeping the old one while the provider can't be
// read), or when again is set because a signature didn't verify with it.
func (c *Material) get(ctx context.Context, key string, again bool, fetch func(context.Context) (string, error)) (string, error) {
	e := c.entry(key)
	e.mu.Lock()
	defer e.mu.Unlock()
	now := c.now()
	recent := !e.tried.IsZero() && now.Sub(e.tried) < Refetch
	switch {
	case e.value != "" && (recent || (!again && now.Sub(e.fetched) < MaxAge)):
		return e.value, nil
	case e.value == "" && recent && e.err != nil:
		return "", e.err
	}
	v, err := fetch(ctx)
	if err == nil && v == "" {
		err = errors.New("the provider's answer has no signing material")
	}
	if ctx.Err() == nil {
		e.tried = now
	}
	if err != nil {
		e.err = err
		if e.value != "" && !again {
			return e.value, nil
		}
		return "", err
	}
	e.value, e.fetched, e.err = v, now, nil
	return v, nil
}

// Verify runs check with the material cached under key, read with fetch
// when needed. When check fails, the material is read again (at most once
// per Refetch) in case the provider rotated it, and checked once more.
// Every failure wraps providers.ErrBadSignature, a failed read included:
// a request that could not be verified is never trusted.
func (c *Material) Verify(ctx context.Context, key string, fetch func(context.Context) (string, error), check func(material string) error) error {
	m, err := c.get(ctx, key, false, fetch)
	if err != nil {
		return bad(err)
	}
	err = check(m)
	if err == nil || errors.Is(err, ErrStale) {
		return bad(err)
	}
	fresh, ferr := c.get(ctx, key, true, fetch)
	if ferr != nil || fresh == m {
		return bad(err)
	}
	return bad(check(fresh))
}

// bad makes sure err wraps providers.ErrBadSignature.
func bad(err error) error {
	if err == nil || errors.Is(err, providers.ErrBadSignature) {
		return err
	}
	return fmt.Errorf("%w: %w", providers.ErrBadSignature, err)
}
