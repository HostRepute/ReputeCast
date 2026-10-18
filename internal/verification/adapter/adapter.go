// Package adapter is the contract every address verification service
// implements (ZeroBounce, NeverBounce, Kickbox, Bouncer, Emailable), like
// internal/providers is for sending providers: a descriptor with the
// service's capabilities and the codes it answers, plain HTTP calls with
// the workspace's own API key, and answers mapped to ReputeCast's four
// results with the service's own code kept beside them.
//
// Adapters live in internal/verification/adapter/<name>, register
// themselves from init and pass the shared suite in
// internal/verification/adapter/contracttest. Core code finds them through
// Lookup and never imports one.
package adapter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

// Result is one of the four outcomes every service's answer maps to.
type Result string

// Results.
const (
	Valid   Result = "valid"
	Invalid Result = "invalid"
	Risky   Result = "risky"
	Unknown Result = "unknown"
)

// Reasons. Invalid answers are either an address that can't receive mail or
// a (possible) spam trap; risky ones say why. A service's own reason that
// fits none of these is kept as it is ("abuse", "toxic").
const (
	ReasonInvalidAddress = "invalid_address"
	ReasonSpamTrap       = "spam_trap"
	ReasonCatchAll       = "catch_all"
	ReasonRole           = "role"
	ReasonDisposable     = "disposable"
)

// Answer is a service's verdict on one address.
type Answer struct {
	Email  string
	Result Result
	// Reason is set for invalid and risky answers (see the Reason constants).
	Reason string
	// Code is the service's own answer, as the service words it
	// ("invalid / mailbox_not_found", "catch-all"), kept beside the result
	// the way bounce codes are.
	Code string
}

// Tag is the contact tag a risky or unknown answer gets ("verify:catch-all").
func (a Answer) Tag() string {
	switch a.Result {
	case Unknown:
		return "verify:unknown"
	case Risky:
		r := a.Reason
		if r == "" {
			r = "risky"
		}
		return "verify:" + tagWord(r)
	}
	return ""
}

// tagWord writes a reason as tags show it: catch_all → catch-all.
func tagWord(r string) string {
	b := []byte(r)
	for i, c := range b {
		if c == '_' || c == ' ' {
			b[i] = '-'
		}
	}
	return string(b)
}

// Errors an adapter returns, so jobs know whether to stop, pause or retry.
var (
	// ErrAuth means the service rejected the API key.
	ErrAuth = errors.New("the service rejected the API key")
	// ErrNoCredits means the account has no credits left.
	ErrNoCredits = errors.New("the service has no credits left")
	// ErrRateLimited means the service asked to slow down (429 and the like).
	ErrRateLimited = errors.New("the service asked to slow down")
	// ErrUnavailable means the service didn't answer or answered 5xx.
	ErrUnavailable = errors.New("the service isn't answering")
)

// CallError carries the HTTP status and the service's message with one of
// the errors above.
type CallError struct {
	Kind    error
	Status  int
	Message string
	// RetryAfter is the wait the service asked for (Retry-After), when it did.
	RetryAfter time.Duration
}

func (e *CallError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("%v (%d): %s", e.Kind, e.Status, e.Message)
	}
	return fmt.Sprintf("%v: %s", e.Kind, e.Message)
}

func (e *CallError) Unwrap() error { return e.Kind }

// Temporary reports whether a later retry may succeed (rate limits, outages).
func Temporary(err error) bool {
	return errors.Is(err, ErrRateLimited) || errors.Is(err, ErrUnavailable)
}

// Status is the HTTP status in err, or 0.
func Status(err error) int {
	var ce *CallError
	if errors.As(err, &ce) {
		return ce.Status
	}
	return 0
}

// Capabilities are the badges the catalog shows for a service.
type Capabilities struct {
	BulkLists     bool `json:"bulkLists"`
	RealTime      bool `json:"realTime"`
	CatchAll      bool `json:"catchAll"`
	CreditBalance bool `json:"creditBalance"`
	// DataRegion is true when the account can choose where addresses are
	// processed (ZeroBounce: United States or the EU).
	DataRegion bool `json:"dataRegion"`
}

// Region is a data region a service offers.
type Region struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Mapping is one row of "How <service>'s answers map": the service's codes,
// the result and reason they become.
type Mapping struct {
	Codes  []string `json:"codes"`
	Result Result   `json:"result"`
	Reason string   `json:"reason,omitempty"`
}

// Config opens an adapter for one connected service.
type Config struct {
	APIKey string
	// Region selects the data region where the service has several ("us",
	// "eu"); empty uses its default endpoint.
	Region string
	// BaseURL replaces the service's API address (tests and recordings).
	BaseURL    string
	HTTPClient *http.Client
}

// Client returns the HTTP client to use (a default with a timeout when unset).
func (c Config) Client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 70 * time.Second}
}

// Adapter calls one service with one account's key.
type Adapter interface {
	// Balance reads the account's credit balance. It sends no address.
	Balance(ctx context.Context) (int64, error)
	// Verify checks one address within timeout (real-time checks, one
	// contact, and bulk jobs one by one).
	Verify(ctx context.Context, email string, timeout time.Duration) (Answer, error)
	// VerifyBatch checks up to Descriptor.MaxBatch addresses in one go and
	// returns one answer per address, in order. Services without a batch
	// endpoint check them one by one inside.
	VerifyBatch(ctx context.Context, emails []string) ([]Answer, error)
}

// Descriptor describes a service in the catalog.
type Descriptor struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	// Initials are the logo tile's letters ("ZB").
	Initials     string       `json:"initials"`
	Capabilities Capabilities `json:"capabilities"`
	Regions      []Region     `json:"regions"`
	// DefaultRegion is where addresses go when no region is chosen
	// ("United States"); empty when the service doesn't say.
	DefaultRegion string `json:"defaultRegion"`
	// Website is the service's own site (Add credits, Open ZeroBounce) and
	// KeyHelp where its API keys are created.
	Website string `json:"website"`
	KeyHelp string `json:"keyHelp"`
	// MaxBatch is how many addresses VerifyBatch takes at once.
	MaxBatch int `json:"maxBatch"`
	// PerSecond paces bulk jobs below the service's rate limits.
	PerSecond float64   `json:"perSecond"`
	Mappings  []Mapping `json:"mappings"`
	New       func(Config) (Adapter, error)
}

var (
	mu       sync.RWMutex
	registry = map[string]Descriptor{}
)

// Register adds a service. Adapters call it from init.
func Register(d Descriptor) {
	mu.Lock()
	defer mu.Unlock()
	if _, dup := registry[d.Key]; dup {
		panic("verification: duplicate service " + d.Key)
	}
	registry[d.Key] = d
}

// Lookup returns a registered service.
func Lookup(key string) (Descriptor, bool) {
	mu.RLock()
	defer mu.RUnlock()
	d, ok := registry[key]
	return d, ok
}

// Order is the catalog's order of the services the brief names.
var Order = []string{"zerobounce", "kickbox", "neverbounce", "bouncer", "emailable"}

// Catalog returns every registered service: the brief's in its order, then
// any other alphabetically.
func Catalog() []Descriptor {
	mu.RLock()
	defer mu.RUnlock()
	rank := map[string]int{}
	for i, k := range Order {
		rank[k] = i + 1
	}
	out := make([]Descriptor, 0, len(registry))
	for _, d := range registry {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		ri, rj := rank[out[i].Key], rank[out[j].Key]
		if ri == 0 {
			ri = 1 << 20
		}
		if rj == 0 {
			rj = 1 << 20
		}
		if ri != rj {
			return ri < rj
		}
		return out[i].Key < out[j].Key
	})
	return out
}
