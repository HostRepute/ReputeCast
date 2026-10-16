// Package config loads ReputeCast configuration from defaults, an optional
// YAML file, REPUTECAST_* environment variables and command-line flags, in
// that order of precedence (later wins).
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

// EnvPrefix is the prefix of every environment variable ReputeCast reads.
// REPUTECAST_HTTP__ADDR maps to http.addr: a double underscore separates
// levels so that single underscores can stay inside key names.
const EnvPrefix = "REPUTECAST_"

// Role names for --role. The default runs all of them in one process.
const (
	RoleWeb    = "web"
	RoleWorker = "worker"
	RoleIngest = "ingest"
)

// Config is the full runtime configuration.
type Config struct {
	Env        string           `koanf:"env"`
	Roles      []string         `koanf:"roles"`
	HTTP       HTTPConfig       `koanf:"http"`
	Database   DatabaseConfig   `koanf:"database"`
	Secrets    SecretsConfig    `koanf:"secrets"`
	Log        LogConfig        `koanf:"log"`
	Auth       AuthConfig       `koanf:"auth"`
	Worker     WorkerConfig     `koanf:"worker"`
	Updates    UpdatesConfig    `koanf:"updates"`
	Metrics    MetricsConfig    `koanf:"metrics"`
	Webhooks   WebhooksConfig   `koanf:"webhooks"`
	Providers  ProvidersConfig  `koanf:"providers"`
	HostRepute HostReputeConfig `koanf:"hostrepute"`
	DNS        DNSConfig        `koanf:"dns"`
	Mail       MailConfig       `koanf:"mail"`
	Connect    ConnectConfig    `koanf:"connect"`
	I18n       I18nConfig       `koanf:"i18n"`
}

// I18nConfig configures the app languages.
type I18nConfig struct {
	// PseudoLocale offers "Pseudo (+40%)" in Language & region outside
	// development, to check layouts on a staging instance: every string
	// accented, about 40% longer and wrapped in ⟦ ⟧.
	PseudoLocale bool `koanf:"pseudo_locale"`
}

// PseudoLocale reports whether members can pick the pseudo-locale: always
// in development, elsewhere when i18n.pseudo_locale is on.
func (c *Config) PseudoLocale() bool { return c.IsDevelopment() || c.I18n.PseudoLocale }

// MailConfig configures system mail (password resets, invitations,
// notifications), which goes out through the System stream's provider.
type MailConfig struct {
	// DevSMTP is where, in development only, mail for the demo's sample
	// providers is delivered instead of a real provider (Mailpit).
	DevSMTP string `koanf:"dev_smtp"`
}

// ConnectConfig configures "Connect a sending provider".
type ConnectConfig struct {
	// SESTemplateURL is the published CloudFormation template the "Launch
	// stack in AWS" link opens. AWS reads quick-create templates from S3, so
	// instances that publish their own copy point this at it.
	SESTemplateURL string `koanf:"ses_template_url"`
}

// HostReputeConfig configures the optional HostRepute integration. ReputeCast
// is a public OAuth client of HostRepute (PKCE, no secret); instances that
// registered their own client can set its id and secret.
type HostReputeConfig struct {
	APIURL       string `koanf:"api_url"`
	AuthURL      string `koanf:"auth_url"`
	TokenURL     string `koanf:"token_url"`
	RevokeURL    string `koanf:"revoke_url"`
	ClientID     string `koanf:"client_id"`
	ClientSecret string `koanf:"client_secret"`
	// Demo serves a built-in, offline HostRepute (sample data, instant OAuth
	// approval) for development and screenshots.
	Demo bool `koanf:"demo"`
}

// DNSConfig configures the resolvers used for authentication checks and
// blocklist lookups. Empty uses the system resolvers; DNSBLs such as
// Spamhaus refuse queries that come through public resolvers.
type DNSConfig struct {
	Servers []string      `koanf:"servers"`
	Timeout time.Duration `koanf:"timeout"`
}

// HTTPConfig configures the HTTP listener.
type HTTPConfig struct {
	Addr string `koanf:"addr"`
	// PublicURL is the externally reachable base URL. It is used for provider
	// webhooks, SSO redirects, passkeys (relying party) and public pages.
	PublicURL string `koanf:"public_url"`
	// TrustedProxies lists CIDRs whose X-Forwarded-For header is trusted.
	TrustedProxies []string `koanf:"trusted_proxies"`
	// RateLimitAuth and RateLimitAPI are requests per minute: sign-in
	// endpoints per client IP, and the API per key (or IP for sessions).
	RateLimitAuth   int           `koanf:"rate_limit_auth"`
	RateLimitAPI    int           `koanf:"rate_limit_api"`
	ReadTimeout     time.Duration `koanf:"read_timeout"`
	WriteTimeout    time.Duration `koanf:"write_timeout"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
}

// DatabaseConfig configures Postgres.
type DatabaseConfig struct {
	URL      string `koanf:"url"`
	MaxConns int32  `koanf:"max_conns"`
	// SegmentRole is the SELECT-only role user-written SQL segments run as.
	SegmentRole string `koanf:"segment_role"`
}

// SecretsConfig holds the master key used to encrypt provider credentials,
// OIDC client secrets and other secrets at rest.
type SecretsConfig struct {
	// Key is a base64-encoded 32-byte key. Rotate by moving the current key
	// to PreviousKeys and setting a new Key.
	Key          string   `koanf:"key"`
	PreviousKeys []string `koanf:"previous_keys"`
	// KeyFile is used when Key is empty. The file is created with a fresh
	// key on first start, so a one-command install works without extra setup.
	KeyFile string `koanf:"key_file"`
}

// LogConfig configures log/slog.
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"` // "text" or "json"
}

// AuthConfig holds sign-in defaults. Instance admins can change most of these
// at runtime in Instance admin › Authentication; these are the bootstrap values.
type AuthConfig struct {
	SessionLifetime   time.Duration `koanf:"session_lifetime"`
	SessionIdle       time.Duration `koanf:"session_idle"`
	CookieName        string        `koanf:"cookie_name"`
	CookieSecure      bool          `koanf:"cookie_secure"`
	BreachedCheck     bool          `koanf:"breached_check"`
	PasswordMinLength int           `koanf:"password_min_length"`
}

// WorkerConfig configures background jobs.
type WorkerConfig struct {
	Concurrency int `koanf:"concurrency"`
}

// UpdatesConfig configures the release check against GitHub.
type UpdatesConfig struct {
	Check bool   `koanf:"check"`
	Repo  string `koanf:"repo"`
}

// MetricsConfig configures the Prometheus endpoint.
type MetricsConfig struct {
	Enabled bool   `koanf:"enabled"`
	Addr    string `koanf:"addr"` // empty serves /metrics on the main listener
}

// ProvidersConfig configures calls to sending-provider APIs.
type ProvidersConfig struct {
	// AllowPrivateNetworks lets provider URLs a workspace enters resolve to
	// loopback and private (RFC 1918) addresses. Off by default so workspace
	// admins can't reach internal services (SSRF); link-local and cloud
	// metadata addresses are always blocked. It also decides whether a
	// community adapter's process may listen on such an address (a sidecar);
	// plain http:// only reaches those.
	AllowPrivateNetworks bool `koanf:"allow_private_networks"`
}

// WebhooksConfig configures outgoing webhooks and the audit stream.
type WebhooksConfig struct {
	// AllowPrivateNetworks lets endpoints resolve to loopback and private
	// (RFC 1918) addresses, for receivers on the same network. Off by
	// default: user-supplied URLs must not reach internal services (SSRF).
	// Link-local and cloud metadata addresses are always blocked.
	AllowPrivateNetworks bool `koanf:"allow_private_networks"`
}

// Defaults returns the configuration used when nothing else is set.
func Defaults() Config {
	return Config{
		Env:   "production",
		Roles: []string{RoleWeb, RoleWorker, RoleIngest},
		HTTP: HTTPConfig{
			Addr:            ":8080",
			PublicURL:       "http://localhost:8080",
			RateLimitAuth:   30,
			RateLimitAPI:    600,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    60 * time.Second,
			ShutdownTimeout: 20 * time.Second,
		},
		// The docker-compose development database; production sets database.url.
		Database: DatabaseConfig{ //nolint:gosec // development default, not a secret
			URL:         "postgres://reputecast:reputecast@localhost:5432/reputecast?sslmode=disable",
			MaxConns:    20,
			SegmentRole: "reputecast_segments",
		},
		Secrets: SecretsConfig{KeyFile: "data/secret.key"},
		Log:     LogConfig{Level: "info", Format: "text"},
		Auth:    AuthConfig{SessionLifetime: 14 * 24 * time.Hour, SessionIdle: 12 * time.Hour, CookieName: "rc_session", CookieSecure: true, BreachedCheck: false, PasswordMinLength: 12},
		Worker:  WorkerConfig{Concurrency: 50},
		Updates: UpdatesConfig{Check: true, Repo: "hostrepute/reputecast"},
		Metrics: MetricsConfig{Enabled: true},
		HostRepute: HostReputeConfig{ //nolint:gosec // G101: public endpoint URLs and client ID, not credentials
			APIURL:    "https://api.hostrepute.com/v1",
			AuthURL:   "https://app.hostrepute.com/oauth/authorize",
			TokenURL:  "https://api.hostrepute.com/oauth/token",
			RevokeURL: "https://api.hostrepute.com/oauth/revoke",
			ClientID:  "reputecast",
		},
		DNS:     DNSConfig{Timeout: 4 * time.Second},
		Mail:    MailConfig{DevSMTP: "localhost:1025"},
		Connect: ConnectConfig{SESTemplateURL: "https://reputecast.com/cf/reputecast-ses.yaml"},
	}
}

// Load builds the configuration. configFile may be empty; flags may be nil.
func Load(configFile string, flags *pflag.FlagSet) (*Config, error) {
	k := koanf.New(".")
	if err := k.Load(structs.Provider(Defaults(), "koanf"), nil); err != nil {
		return nil, fmt.Errorf("config defaults: %w", err)
	}
	if configFile == "" {
		configFile = os.Getenv(EnvPrefix + "CONFIG")
	}
	if configFile != "" {
		if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("config file %s: %w", configFile, err)
		}
	}
	envProvider := env.Provider(".", env.Opt{
		Prefix: EnvPrefix,
		TransformFunc: func(key, value string) (string, any) {
			key = strings.ToLower(strings.TrimPrefix(key, EnvPrefix))
			key = strings.ReplaceAll(key, "__", ".")
			if strings.Contains(value, ",") && listKeys[key] {
				return key, strings.Split(value, ",")
			}
			return key, value
		},
	})
	if err := k.Load(envProvider, nil); err != nil {
		return nil, fmt.Errorf("config env: %w", err)
	}
	// DATABASE_URL is the de-facto standard; honour it when the specific key is unset.
	if v := os.Getenv("DATABASE_URL"); v != "" && os.Getenv(EnvPrefix+"DATABASE__URL") == "" {
		_ = k.Set("database.url", v)
	}
	if flags != nil {
		if err := k.Load(posflag.ProviderWithFlag(flags, ".", k, flagKey), nil); err != nil {
			return nil, fmt.Errorf("config flags: %w", err)
		}
	}
	var c Config
	if err := k.UnmarshalWithConf("", &c, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// listKeys are keys whose environment value is a comma-separated list.
var listKeys = map[string]bool{
	"roles":                 true,
	"http.trusted_proxies":  true,
	"secrets.previous_keys": true,
	"dns.servers":           true,
}

// flagKey maps a command-line flag to a config key. Only flags that were
// explicitly set override lower layers.
func flagKey(f *pflag.Flag) (string, any) {
	if !f.Changed {
		return "", nil
	}
	switch f.Name {
	case "role":
		return "roles", strings.Split(f.Value.String(), ",")
	case "addr":
		return "http.addr", f.Value.String()
	case "database-url":
		return "database.url", f.Value.String()
	case "log-level":
		return "log.level", f.Value.String()
	}
	return "", nil
}

// Validate checks the configuration for mistakes that would otherwise fail late.
func (c *Config) Validate() error {
	var errs []error
	for _, r := range c.Roles {
		if !slices.Contains([]string{RoleWeb, RoleWorker, RoleIngest}, r) {
			errs = append(errs, fmt.Errorf("unknown role %q (want web, worker or ingest)", r))
		}
	}
	if u, err := url.Parse(c.HTTP.PublicURL); err != nil || u.Scheme == "" || u.Host == "" {
		errs = append(errs, fmt.Errorf("http.public_url must be an absolute URL, got %q", c.HTTP.PublicURL))
	}
	if c.Secrets.Key != "" {
		if _, err := DecodeKey(c.Secrets.Key); err != nil {
			errs = append(errs, fmt.Errorf("secrets.key: %w", err))
		}
	}
	if c.Auth.PasswordMinLength < 8 {
		errs = append(errs, errors.New("auth.password_min_length must be at least 8"))
	}
	return errors.Join(errs...)
}

// HasRole reports whether this process runs the given role.
func (c *Config) HasRole(role string) bool { return slices.Contains(c.Roles, role) }

// IsDevelopment reports whether the instance runs in development mode.
func (c *Config) IsDevelopment() bool { return c.Env == "development" }

// DecodeKey decodes a base64 (standard or URL) 32-byte key.
func DecodeKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil {
			if len(b) != 32 {
				return nil, fmt.Errorf("key must be 32 bytes, got %d", len(b))
			}
			return b, nil
		}
	}
	return nil, errors.New("key must be base64-encoded")
}
