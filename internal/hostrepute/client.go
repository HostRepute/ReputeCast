// Package hostrepute connects a workspace to HostRepute (hostrepute.com),
// the sibling email-reputation service: the OAuth 2 (PKCE) and API-key
// connection, the encrypted tokens, a River job that syncs blocklist,
// Google Postmaster, Microsoft SNDS and DMARC data plus alerts into
// ReputeCast, "Add to monitoring" and disconnect.
//
// The HostRepute API is reached only through the Client interface. HTTPClient
// implements it against the endpoints documented in
// docs/integrations/hostrepute.md; when the official hostrepute-go client is
// public, an adapter to it replaces HTTPClient without touching the rest.
//
// What crosses the boundary: ReputeCast sends sending domains, organization
// domains, dedicated IPs and server hostnames to monitor. It never sends
// contacts, email content or reports.
package hostrepute

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Scopes ReputeCast asks for, with what each one is used for.
const (
	ScopeMonitorRead = "monitor:read"
	ScopeAlertsRead  = "alerts:read"
	ScopeAssetsWrite = "assets:write"
)

// Scopes in the order they are requested and shown.
var Scopes = []string{ScopeMonitorRead, ScopeAlertsRead, ScopeAssetsWrite}

// ScopeUse explains a scope in the product's words.
var ScopeUse = map[string]string{
	ScopeMonitorRead: "blocklist, Postmaster, SNDS and DMARC data",
	ScopeAlertsRead:  "incidents",
	ScopeAssetsWrite: "add domains and IPs to monitoring",
}

// Asset types HostRepute monitors.
const (
	AssetDomain   = "domain"
	AssetIP       = "ip"
	AssetHostname = "hostname"
)

// Account is the HostRepute account and team behind a token (GET /me).
type Account struct {
	ID     string   `json:"id"`
	Email  string   `json:"email"`
	Team   Team     `json:"team"`
	Scopes []string `json:"scopes"`
	// Sources is how many blocklists HostRepute checks (84).
	Sources              int `json:"sources"`
	CheckIntervalMinutes int `json:"check_interval_minutes"`
	// DMARCRua is the address that receives aggregate reports for the team.
	DMARCRua string `json:"dmarc_rua"`
}

// Team is the HostRepute team, with its plan's monitoring limit.
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Plan string `json:"plan"`
	// MonitorLimit is how many assets the plan monitors; 0 means unlimited.
	MonitorLimit int `json:"monitor_limit"`
	Monitored    int `json:"monitored"`
}

// Asset is a monitored domain, IP or hostname.
type Asset struct {
	ID    string `json:"id"`
	Asset string `json:"asset"`
	Type  string `json:"type"`
	// Status of HostRepute's monitoring: synced, pending (first check
	// running) or error.
	Status string `json:"status"`
	// Score is HostRepute's 0–100 reputation score with its verdict word.
	Score         *int       `json:"score,omitempty"`
	Verdict       string     `json:"verdict,omitempty"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	Listings      int        `json:"listings"`
}

// Blocklists is every source checked for every monitored asset.
type Blocklists struct {
	Sources   int               `json:"sources"`
	CheckedAt time.Time         `json:"checked_at"`
	Results   []BlocklistResult `json:"results"`
}

// BlocklistResult is one asset on one source.
type BlocklistResult struct {
	Asset     string `json:"asset"`
	AssetType string `json:"asset_type"` // domain or ip
	Source    string `json:"source"`
	// Impact on inbox placement when listed: high, medium or low.
	Impact      string     `json:"impact"`
	Rank        int        `json:"rank"`
	Listed      bool       `json:"listed"`
	ListedSince *time.Time `json:"listed_since,omitempty"`
	DelistURL   string     `json:"delist_url"`
	Detail      string     `json:"detail,omitempty"`
	Delisting   *Delisting `json:"delisting,omitempty"`
	CheckedAt   time.Time  `json:"checked_at"`
}

// Delisting is HostRepute's delisting request for a listing.
type Delisting struct {
	ID          string    `json:"id,omitempty"`
	Status      string    `json:"status"` // requested, pending, delisted, denied
	RequestedAt time.Time `json:"requested_at"`
	URL         string    `json:"url,omitempty"`
}

// PostmasterDay is one day of Google Postmaster Tools data for a domain.
type PostmasterDay struct {
	Day              string   `json:"day"` // YYYY-MM-DD
	SpamRate         *float64 `json:"spam_rate,omitempty"`
	DomainReputation string   `json:"domain_reputation,omitempty"` // HIGH, MEDIUM, LOW, BAD
	IPReputation     string   `json:"ip_reputation,omitempty"`
	SPFRate          *float64 `json:"spf_rate,omitempty"`
	DKIMRate         *float64 `json:"dkim_rate,omitempty"`
	DMARCRate        *float64 `json:"dmarc_rate,omitempty"`
	TLSRate          *float64 `json:"tls_rate,omitempty"`
	DeliveryErrors   *float64 `json:"delivery_errors,omitempty"`
	FBLFlagged       *int     `json:"fbl_flagged,omitempty"`
}

// SNDSDay is one day of Microsoft SNDS data for an IP.
type SNDSDay struct {
	Day           string   `json:"day"`
	FilterResult  string   `json:"filter_result"` // green, yellow, red
	ComplaintRate *float64 `json:"complaint_rate,omitempty"`
	TrapHits      int      `json:"trap_hits"`
	Messages      int64    `json:"messages"`
	JMRPReports   int      `json:"jmrp_reports"`
}

// DMARCReport is one aggregate (rua) report.
type DMARCReport struct {
	Domain    string            `json:"domain"`
	OrgName   string            `json:"org_name"`
	ReportID  string            `json:"report_id"`
	DateBegin time.Time         `json:"date_begin"`
	DateEnd   time.Time         `json:"date_end"`
	Policy    map[string]string `json:"policy"`
	Records   []DMARCRecord     `json:"records"`
}

// DMARCRecord is one row of an aggregate report: a source IP and what
// happened to its messages.
type DMARCRecord struct {
	SourceIP    string `json:"source_ip"`
	PTR         string `json:"ptr,omitempty"`
	Count       int64  `json:"count"`
	Disposition string `json:"disposition"` // none, quarantine, reject
	DKIM        string `json:"dkim"`        // DMARC-evaluated: pass or fail
	SPF         string `json:"spf"`
	// DKIMResult / SPFResult are the raw auth results: pass, fail, none…
	DKIMResult string `json:"dkim_result,omitempty"`
	SPFResult  string `json:"spf_result,omitempty"`
	HeaderFrom string `json:"header_from"`
	DKIMDomain string `json:"dkim_domain,omitempty"`
	SPFDomain  string `json:"spf_domain,omitempty"`
}

// Alert is something HostRepute noticed: a new listing, a delisting, a
// reputation drop, a DMARC failure spike.
type Alert struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	// Kind: listed, delisted, reputation_drop, dmarc_failure, auth_change.
	Kind      string `json:"kind"`
	Severity  string `json:"severity"` // info, ok, warn, bad
	Asset     string `json:"asset"`
	Source    string `json:"source,omitempty"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	DelistURL string `json:"delist_url,omitempty"`
}

// Check is an on-demand re-check of monitored assets.
type Check struct {
	ID         string     `json:"id"`
	Status     string     `json:"status"` // running, done, failed
	Total      int        `json:"total"`
	Answered   int        `json:"answered"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// Client is everything ReputeCast asks HostRepute. Implementations must be
// safe for concurrent use.
type Client interface {
	Account(ctx context.Context) (Account, error)
	Assets(ctx context.Context) ([]Asset, error)
	// AddAsset starts monitoring an asset; it fails with *LimitError when the
	// team's plan is full.
	AddAsset(ctx context.Context, asset, typ, label string) (Asset, error)
	RemoveAsset(ctx context.Context, id string) error
	Blocklists(ctx context.Context) (Blocklists, error)
	Postmaster(ctx context.Context, domain string, from, to time.Time) ([]PostmasterDay, error)
	SNDS(ctx context.Context, ip string, from, to time.Time) ([]SNDSDay, error)
	DMARCReports(ctx context.Context, domain string, from, to time.Time) ([]DMARCReport, error)
	Alerts(ctx context.Context, since time.Time) ([]Alert, error)
	StartCheck(ctx context.Context, assets []string) (Check, error)
	Check(ctx context.Context, id string) (Check, error)
	RequestDelisting(ctx context.Context, asset, source string) (Delisting, error)
	// Revoke invalidates the token at HostRepute (disconnect).
	Revoke(ctx context.Context) error
}

// Errors returned by clients.
var (
	// ErrUnauthorized: HostRepute rejected the token (revoked or expired).
	ErrUnauthorized = errors.New("hostrepute: token rejected")
	// ErrForbidden: the token lacks a scope the call needs.
	ErrForbidden = errors.New("hostrepute: missing scope")
	// ErrUnavailable: HostRepute couldn't be reached (network, timeouts, 5xx).
	ErrUnavailable = errors.New("hostrepute: unreachable")
	// ErrNotFound: the asset, check or report doesn't exist at HostRepute.
	ErrNotFound = errors.New("hostrepute: not found")
)

// LimitError means the team's plan monitors as many assets as it allows.
// It's a HostRepute limit; ReputeCast stays fully usable.
type LimitError struct {
	Limit int
	Used  int
	Team  string
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("hostrepute: monitoring limit reached (%d of %d)", e.Used, e.Limit)
}

// APIError is any other non-2xx answer.
type APIError struct {
	Status int
	Code   string
	Detail string
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("hostrepute: %d %s: %s", e.Status, e.Code, e.Detail)
	}
	return fmt.Sprintf("hostrepute: %d %s", e.Status, e.Code)
}
