package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/version"
)

// MaxInFlight is how many calls may run against one adapter at a time.
const MaxInFlight = 64

// Installation is one installed community adapter as the runtime sees it:
// its manifest, where its process listens and the secrets that sign the
// calls. Every connection made through the adapter shares it, with its HTTP
// client, its concurrency limit and its circuit breaker.
type Installation struct {
	// ID is the installed_adapters row.
	ID       uuid.UUID
	Manifest Manifest
	// Endpoint is the adapter process's base URL ("" until it is set).
	Endpoint string
	// PublicURL is ReputeCast's public base URL, used to rebuild the URL of
	// forwarded webhooks.
	PublicURL string

	secrets []string
	// endpointErr explains why the stored address can't be used.
	endpointErr string
	client      *http.Client
	breaker     *breaker
	sem         chan struct{}
	now         func() time.Time
	log         *slog.Logger
}

// Options configure an installation.
type Options struct {
	ID       uuid.UUID
	Manifest Manifest
	Endpoint string
	// Secrets sign requests and verify responses: the current secret first,
	// then the previous one while it is still valid.
	Secrets []string
	// AllowPrivate lets the endpoint resolve to loopback and private
	// addresses (adapters.allow_private_networks).
	AllowPrivate bool
	PublicURL    string
	// HTTP overrides the client (tests); by default calls go through a
	// netguard client built for the endpoint.
	HTTP *http.Client
	Now  func() time.Time
	Log  *slog.Logger

	breaker *breaker
}

// NewInstallation builds an installation. The endpoint is checked again at
// every connection: its address must be allowed by the network policy, and
// plain http:// only reaches loopback and private addresses.
func NewInstallation(o Options) (*Installation, error) {
	in := &Installation{ID: o.ID, Manifest: o.Manifest, PublicURL: strings.TrimRight(o.PublicURL, "/"), now: o.Now, log: o.Log,
		breaker: o.breaker, sem: make(chan struct{}, MaxInFlight)}
	if in.now == nil {
		in.now = time.Now
	}
	if in.log == nil {
		in.log = slog.Default()
	}
	if in.breaker == nil {
		in.breaker = newBreaker(in.now)
	}
	for _, s := range o.Secrets {
		if s != "" {
			in.secrets = append(in.secrets, s)
		}
	}
	if o.Endpoint != "" {
		u, err := CheckEndpoint(o.Endpoint, o.AllowPrivate)
		if err != nil {
			return nil, err
		}
		in.Endpoint = u
	}
	in.client = o.HTTP
	if in.client == nil {
		in.client = endpointClient(o.AllowPrivate, strings.HasPrefix(in.Endpoint, "http://"))
	}
	return in, nil
}

// CheckEndpoint validates an adapter address: an absolute http(s) URL
// without credentials, query or fragment. Without allowPrivate it must be
// https:// and public; with it, http:// is accepted for addresses that turn
// out to be loopback or private when connecting.
func CheckEndpoint(raw string, allowPrivate bool) (string, error) {
	raw = strings.TrimSpace(raw)
	if p, err := url.Parse(raw); err != nil || p.Host == "" || (p.Scheme != "http" && p.Scheme != "https") {
		return "", errors.New("enter the adapter's address as a full URL, like https://example-esp.internal:4600")
	} else if p.User != nil {
		return "", errors.New("the adapter's address can't contain credentials: ReputeCast signs every call instead")
	}
	u, err := netguard.Policy{AllowPrivate: allowPrivate}.ValidateURL(raw)
	if err != nil {
		msg := strings.TrimPrefix(err.Error(), netguard.ErrBlocked.Error()+": ")
		if netguard.IsBlocked(err) && !allowPrivate {
			msg += "; loopback and private addresses need adapters.allow_private_networks"
		}
		return "", errors.New(msg)
	}
	switch {
	case u.RawQuery != "" || u.Fragment != "":
		return "", errors.New("the adapter's address can't have a query or a fragment")
	case u.Scheme == "http" && !allowPrivate:
		return "", errors.New("use https://: plain http:// is only allowed for loopback and private addresses, with adapters.allow_private_networks")
	}
	if ip, err := netip.ParseAddr(strings.Trim(u.Hostname(), "[]")); err == nil && u.Scheme == "http" && !privateAddr(ip) {
		return "", errors.New("use https:// for a public address: credentials travel with every call")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String(), nil
}

var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// privateAddr reports a loopback or private-network address.
func privateAddr(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsLoopback() || ip.IsPrivate() || cgnat.Contains(ip)
}

// endpointClient is a netguard-style client: the address is checked at
// connect time, redirects are never followed, no proxy is used, and with
// plain http only private addresses can be reached.
func endpointClient(allowPrivate, plainHTTP bool) *http.Client {
	policy := netguard.Policy{AllowPrivate: allowPrivate}
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip, err := netip.ParseAddr(host)
			if err != nil {
				return fmt.Errorf("%w: %s", netguard.ErrBlocked, host)
			}
			if err := policy.Check(ip); err != nil {
				return err
			}
			if plainHTTP && !privateAddr(ip) {
				return fmt.Errorf("%w: %s is public and plain http:// is only used on private networks", netguard.ErrBlocked, ip)
			}
			return nil
		}}
	return &http.Client{
		Transport: &http.Transport{
			Proxy:               nil,
			DialContext:         dialer.DialContext,
			ForceAttemptHTTP2:   true,
			MaxIdleConns:        64,
			MaxIdleConnsPerHost: 16,
			MaxConnsPerHost:     MaxInFlight,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// adapterError marks failures of the adapter process itself (not of the
// provider), which count toward the circuit breaker.
func (in *Installation) adapterError(op, code, msg string, cause error) error {
	in.breaker.failure(msg)
	in.log.Warn("community adapter call failed", "adapter", in.Manifest.Key(), "op", op, "code", code, "err", msg)
	return &providers.CallError{Op: op, Code: code, Message: msg, Temporary: true, Cause: cause}
}

// invalidAnswer reports a response that doesn't follow the protocol.
func (in *Installation) invalidAnswer(op string, err error) error {
	return in.adapterError(op, "adapter_invalid_response", "The "+in.Manifest.Name+" adapter answered "+op+" outside the protocol: "+err.Error()+".", err)
}

// call sends one operation and decodes its result into out. Provider
// failures the adapter reports become providers errors (CallError,
// ErrBadSignature, ErrNotSupported); failures of the adapter itself become
// temporary CallErrors and count toward the circuit breaker.
func (in *Installation) call(ctx context.Context, op string, conn *Connection, key string, input, out any) error {
	spec, ok := ops[op]
	if !ok || !spec.Allowed(in.Manifest.Capabilities) {
		return fmt.Errorf("%w: the %s adapter doesn't declare %s", providers.ErrNotSupported, in.Manifest.Name, op)
	}
	switch {
	case in.endpointErr != "":
		return &providers.CallError{Op: op, Code: "adapter_not_configured", Temporary: true,
			Message: fmt.Sprintf("The address set for the %s adapter is refused: %s.", in.Manifest.Name, in.endpointErr)}
	case in.Endpoint == "" || len(in.secrets) == 0:
		return &providers.CallError{Op: op, Code: "adapter_not_configured", Temporary: true,
			Message: fmt.Sprintf("The %s adapter has no address yet. An instance admin sets it with `reputecast adapter set-url %s <url>`.", in.Manifest.Name, in.Manifest.ID)}
	}
	if key == "" {
		key = uuid.Must(uuid.NewV7()).String()
	}
	if ok, until := in.breaker.allow(); !ok && !isProbe(ctx) {
		return &providers.CallError{Op: op, Code: "adapter_unavailable", Temporary: true,
			Message: fmt.Sprintf("The %s adapter isn't answering, so ReputeCast pauses calls to it until %s.", in.Manifest.Name, until.UTC().Format("15:04:05 UTC"))}
	}
	cctx, cancel := context.WithTimeout(ctx, spec.Timeout)
	defer cancel()
	deadline, _ := cctx.Deadline()
	body, err := json.Marshal(Envelope{Protocol: ProtocolVersion, Operation: op, Adapter: in.Manifest.ID, IdempotencyKey: key,
		Deadline: deadline.UTC().Format("2006-01-02T15:04:05.000Z07:00"), Connection: conn, Input: input})
	if err != nil {
		in.breaker.release()
		return &providers.CallError{Op: op, Message: "the request can't be encoded: " + err.Error(), Cause: err}
	}
	req, err := http.NewRequestWithContext(cctx, http.MethodPost, in.Endpoint+"/v1/"+op, bytes.NewReader(body))
	if err != nil {
		in.breaker.release()
		return &providers.CallError{Op: op, Message: err.Error(), Cause: err}
	}
	ts := in.now().Unix()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ReputeCast/"+strings.TrimPrefix(version.Version, "v")+" (adapter-protocol 1)")
	req.Header.Set(HeaderProtocol, strconv.Itoa(ProtocolVersion))
	req.Header.Set(HeaderOperation, op)
	req.Header.Set(HeaderIdempotency, key)
	req.Header.Set(HeaderTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(HeaderSignature, SignatureHeader(in.secrets, ts, body))

	select {
	case in.sem <- struct{}{}:
	case <-cctx.Done():
		in.breaker.release()
		return &providers.CallError{Op: op, Code: "adapter_busy", Temporary: true, Message: fmt.Sprintf("%d calls to the %s adapter were already waiting.", MaxInFlight, in.Manifest.Name), Cause: cctx.Err()}
	}
	defer func() { <-in.sem }()
	resp, err := in.client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			in.breaker.release()
			return &providers.CallError{Op: op, Message: "cancelled", Temporary: true, Cause: err}
		}
		return in.adapterError(op, "adapter_unreachable", "The "+in.Manifest.Name+" adapter can't be reached: "+describe(cctx, err)+".", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, spec.MaxResponse+1))
	if err != nil {
		return in.adapterError(op, "adapter_unreachable", "The "+in.Manifest.Name+" adapter's answer was cut off: "+describe(cctx, err)+".", err)
	}
	if int64(len(raw)) > spec.MaxResponse {
		return in.invalidAnswer(op, invalidf("the answer is larger than %d bytes", spec.MaxResponse))
	}
	if err := Verify(in.secrets, resp.Header.Get(HeaderTimestamp), resp.Header.Get(HeaderSignature), raw, in.now()); err != nil {
		if resp.StatusCode == http.StatusUnauthorized {
			return in.adapterError(op, "adapter_unauthenticated", "The "+in.Manifest.Name+" adapter rejected ReputeCast's signature. Issue a new secret with `reputecast adapter secret "+in.Manifest.ID+"`, set it on the adapter process, and check both clocks.", nil)
		}
		return in.invalidAnswer(op, invalidf("HTTP %d not signed with this installation's secret (%v)", resp.StatusCode, err))
	}
	if mt, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); mt != "application/json" {
		return in.invalidAnswer(op, invalidf("HTTP %d with content type %q instead of application/json", resp.StatusCode, resp.Header.Get("Content-Type")))
	}
	var r Response
	if err := json.Unmarshal(raw, &r); err != nil {
		return in.invalidAnswer(op, invalid("the body is not valid JSON"))
	}
	switch {
	case r.Protocol != ProtocolVersion:
		return in.invalidAnswer(op, invalidf("protocol %d instead of %d", r.Protocol, ProtocolVersion))
	case r.IdempotencyKey != key:
		return in.invalidAnswer(op, invalid("it answers another request (idempotencyKey differs)"))
	case r.Error != nil:
		return in.wireError(op, r.Error)
	case resp.StatusCode != http.StatusOK:
		return in.invalidAnswer(op, invalidf("HTTP %d without an error", resp.StatusCode))
	}
	if out != nil {
		if len(r.Result) == 0 || string(r.Result) == "null" {
			return in.invalidAnswer(op, invalid("the answer has no result"))
		}
		if err := json.Unmarshal(r.Result, out); err != nil {
			return in.invalidAnswer(op, invalidf("the result doesn't match the protocol: %v", err))
		}
	}
	in.breaker.success()
	return nil
}

// wireError maps an error the adapter reported.
func (in *Installation) wireError(op string, e *WireError) error {
	msg := cut(e.Message)
	pe := &providers.CallError{Op: op, Status: e.Status, Code: cut(e.ProviderCode), Message: msg}
	switch e.Code {
	case CodeProviderAuth:
		pe.Auth = true
	case CodeProviderUnavailable:
		pe.Temporary = true
	case CodeProviderThrottled:
		pe.Temporary, pe.Throttled = true, true
	case CodeProviderError:
	case CodeBadSignature:
		in.breaker.success()
		return fmt.Errorf("%w: %s", providers.ErrBadSignature, msg)
	case CodeNotSupported:
		in.breaker.success()
		return fmt.Errorf("%w: %s", providers.ErrNotSupported, msg)
	case CodeUnauthenticated:
		return in.adapterError(op, "adapter_unauthenticated", "The "+in.Manifest.Name+" adapter rejected ReputeCast's signature. Issue a new secret with `reputecast adapter secret "+in.Manifest.ID+"`, set it on the adapter process, and check both clocks.", nil)
	case CodeInvalidRequest, CodeTooLarge:
		in.breaker.failure(msg)
		return &providers.CallError{Op: op, Code: "adapter_" + e.Code, Message: "The " + in.Manifest.Name + " adapter refused the request: " + msg}
	default:
		return in.adapterError(op, "adapter_internal", "The "+in.Manifest.Name+" adapter failed: "+msg, nil)
	}
	// The adapter answered properly; the provider failed.
	in.breaker.success()
	if pe.Message == "" {
		pe.Message = "the provider refused the request"
	}
	return pe
}

// describe turns a transport error into a short phrase.
func describe(ctx context.Context, err error) string {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "timed out"
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	return strings.TrimSuffix(netguard.Describe(ctx, err), ".")
}

type probeKey struct{}

// isProbe reports a call made on purpose to test the adapter: it goes out
// even while the circuit breaker is open, and its outcome closes or reopens it.
func isProbe(ctx context.Context) bool {
	v, _ := ctx.Value(probeKey{}).(bool)
	return v
}

// Health calls the adapter's health operation and checks it is the adapter
// that was installed. It goes out even while the circuit breaker is open.
func (in *Installation) Health(ctx context.Context) (HealthOutput, error) {
	var out HealthOutput
	if err := in.call(context.WithValue(ctx, probeKey{}, true), OpHealth, nil, "", struct{}{}, &out); err != nil {
		return out, err
	}
	switch {
	case out.Adapter != in.Manifest.ID:
		return out, in.invalidAnswer(OpHealth, invalidf("the process at %s is the %q adapter, not %q", in.Endpoint, out.Adapter, in.Manifest.ID))
	case out.Protocol != ProtocolVersion:
		return out, in.invalidAnswer(OpHealth, invalidf("the process speaks protocol %d, not %d", out.Protocol, ProtocolVersion))
	}
	out.Version = cut(out.Version)
	return out, nil
}

// BreakerState describes the circuit breaker: closed, open (until when) or
// half_open, and the last adapter failure.
func (in *Installation) BreakerState() (string, time.Time, string) { return in.breaker.state() }
