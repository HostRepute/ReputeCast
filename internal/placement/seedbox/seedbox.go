// Package seedbox reads the seed mailboxes of placement tests: it signs in
// (an IMAP password, IMAP with XOAUTH2, or Microsoft Graph with OAuth),
// finds the folders a copy can land in, finds copies by their test token,
// tells which folder or tab each one is in, and deletes them afterwards.
//
// IMAP goes through github.com/emersion/go-imap/v2, already in the stack
// for bounce mailboxes. It covers LOGIN, AUTHENTICATE (with the XOAUTH2
// client in this package), LIST with special-use attributes, UID SEARCH,
// FETCH of header fields and INTERNALDATE, STORE, MOVE and EXPUNGE, and its
// imapserver package gives the tests an in-process server. It has no way
// to send Gmail's X-GM-RAW search, which is how Gmail's tabs are read over
// IMAP, so gmailraw.go runs that one search on a short connection of its
// own. Outlook's Focused and Other exist only in Microsoft Graph
// (inferenceClassification), so Microsoft seeds connected with OAuth are
// read through Graph.
package seedbox

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// Mailbox providers a seed can be at.
const (
	Gmail           = "gmail"
	GoogleWorkspace = "google_workspace"
	Outlook         = "outlook"
	Microsoft365    = "microsoft_365"
	Yahoo           = "yahoo"
	AOL             = "aol"
	ICloud          = "icloud"
	GMX             = "gmx"
	WebDE           = "webde"
)

// Ways to connect.
const (
	AuthPassword = "password"
	AuthOAuth    = "oauth"
)

// Provider describes a mailbox provider for seeds.
type Provider struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	// IMAP server, filled in for the person.
	Host string `json:"host"`
	Port int    `json:"port"`
	// OAuth is the sign-in app OAuth uses ("google", "microsoft"); empty
	// when only a password works.
	OAuth string `json:"oauth,omitempty"`
	// Password is what the provider calls the IMAP password ("App
	// password", "App-specific password", "Password").
	Password string `json:"password"`
	// UsernameLabel is how the provider names the sign-in ("Apple ID").
	UsernameLabel string `json:"usernameLabel"`
	// PasswordHelp is the provider's page about its app passwords, and
	// IMAPHelp where IMAP access is switched on (GMX, Web.de).
	PasswordHelp string `json:"passwordHelp,omitempty"`
	IMAPHelp     string `json:"imapHelp,omitempty"`
	// InboxName and SpamName are the provider's own names ("Posteingang",
	// "Junk Email", "Bulk", "Spamverdacht").
	InboxName string `json:"inboxName"`
	SpamName  string `json:"spamName"`
	// Tabs are the inbox tabs the provider sorts mail into.
	Tabs []string `json:"tabs,omitempty"`
}

// Providers in the add dialog's order. "Password" holds what each provider
// calls its IMAP password, not a credential.
//
//nolint:gosec // G101: labels, not credentials
var Providers = []Provider{
	{Key: Gmail, Name: "Gmail", Host: "imap.gmail.com", Port: 993, OAuth: "google", Password: "App password", UsernameLabel: "Email address",
		PasswordHelp: "https://support.google.com/accounts/answer/185833", InboxName: "Inbox", SpamName: "Spam", Tabs: []string{"Primary", "Promotions", "Updates", "Social", "Forums"}},
	{Key: GoogleWorkspace, Name: "Google Workspace", Host: "imap.gmail.com", Port: 993, OAuth: "google", Password: "App password", UsernameLabel: "Email address",
		PasswordHelp: "https://support.google.com/accounts/answer/185833", InboxName: "Inbox", SpamName: "Spam", Tabs: []string{"Primary", "Promotions", "Updates", "Social", "Forums"}},
	{Key: Outlook, Name: "Outlook.com", Host: "outlook.office365.com", Port: 993, OAuth: "microsoft", Password: "App password", UsernameLabel: "Email address",
		PasswordHelp: "https://support.microsoft.com/account-billing/how-to-get-and-use-app-passwords-5896ed9b-4263-e681-128a-a6f2979a7944", InboxName: "Inbox", SpamName: "Junk Email", Tabs: []string{"Focused", "Other"}},
	{Key: Microsoft365, Name: "Microsoft 365", Host: "outlook.office365.com", Port: 993, OAuth: "microsoft", Password: "App password", UsernameLabel: "Email address",
		PasswordHelp: "https://support.microsoft.com/account-billing/how-to-get-and-use-app-passwords-5896ed9b-4263-e681-128a-a6f2979a7944", InboxName: "Inbox", SpamName: "Junk Email", Tabs: []string{"Focused", "Other"}},
	{Key: Yahoo, Name: "Yahoo", Host: "imap.mail.yahoo.com", Port: 993, Password: "App password", UsernameLabel: "Email address",
		PasswordHelp: "https://help.yahoo.com/kb/SLN15241.html", InboxName: "Inbox", SpamName: "Bulk"},
	{Key: AOL, Name: "AOL", Host: "imap.aol.com", Port: 993, Password: "App password", UsernameLabel: "Email address",
		PasswordHelp: "https://help.aol.com/articles/Create-and-manage-app-password", InboxName: "Inbox", SpamName: "Bulk"},
	{Key: ICloud, Name: "iCloud", Host: "imap.mail.me.com", Port: 993, Password: "App-specific password", UsernameLabel: "Apple ID",
		PasswordHelp: "https://support.apple.com/102654", InboxName: "Inbox", SpamName: "Junk"},
	{Key: GMX, Name: "GMX", Host: "imap.gmx.net", Port: 993, Password: "Password", UsernameLabel: "Email address",
		IMAPHelp: "https://support.gmx.com/pop-imap/toggle.html", InboxName: "Posteingang", SpamName: "Spamverdacht"},
	{Key: WebDE, Name: "Web.de", Host: "imap.web.de", Port: 993, Password: "Password", UsernameLabel: "Email address",
		IMAPHelp: "https://hilfe.web.de/pop-imap/einschalten.html", InboxName: "Posteingang", SpamName: "Spamverdacht"},
}

// ProviderOf returns a mailbox provider by key.
func ProviderOf(key string) (Provider, bool) {
	for _, p := range Providers {
		if p.Key == key {
			return p, true
		}
	}
	return Provider{}, false
}

// IsGoogle reports whether a provider's mail is Gmail's (tabs over X-GM-RAW).
func IsGoogle(key string) bool { return key == Gmail || key == GoogleWorkspace }

// IsMicrosoft reports whether a provider's mail is Exchange's.
func IsMicrosoft(key string) bool { return key == Outlook || key == Microsoft365 }

// Folder roles.
const (
	RoleInbox = "inbox"
	RoleTab   = "tab"
	RoleSpam  = "spam"
)

// Folder is a place a copy can land in, as the mailbox names it.
type Folder struct {
	Role string `json:"role"`
	// Name is the IMAP mailbox name or the Graph folder id.
	Name string `json:"name"`
	// Label is the provider's own word ("Inbox", "Junk Email", "Promotions").
	Label string `json:"label"`
}

// Ref locates a found copy so it can be deleted.
type Ref struct {
	Folder      string `json:"folder"`
	UID         uint32 `json:"uid,omitempty"`
	UIDValidity uint32 `json:"uidValidity,omitempty"`
	GraphID     string `json:"graphId,omitempty"`
}

// Found is a copy found in a seed.
type Found struct {
	Token string
	// Role is inbox, tab or spam; Folder the mailbox's name for it, Path
	// the path as the mailbox reports it ("acme.seed@outlook.com › Junk Email").
	Role   string
	Folder string
	// Tab is the inbox tab (Primary, Promotions, Focused, Other), when the
	// mailbox has tabs and says.
	Tab string
	// ReceivedAt is the mailbox's own received time.
	ReceivedAt time.Time
	Header     Header
	Ref        Ref
}

// Errors when signing in.
var (
	// ErrAuthFailed means the mailbox rejected the password or token.
	ErrAuthFailed = errors.New("the mailbox refused the sign-in")
	// ErrAuthExpired means an OAuth grant ended (revoked, expired, password changed).
	ErrAuthExpired = errors.New("the mailbox ended ReputeCast's access")
	// ErrUnreachable means the mailbox server didn't answer.
	ErrUnreachable = errors.New("the mailbox server didn't answer")
)

// Config opens a seed.
type Config struct {
	Provider string
	Email    string
	// Auth is AuthPassword or AuthOAuth.
	Auth string
	// IMAP server and sign-in.
	Host     string
	Port     int
	Username string
	Password string
	// Token supplies OAuth access tokens (refreshing them as needed).
	Token oauth2.TokenSource
	// Dialer dials IMAP through the address guard; HTTP calls Graph.
	Dialer *net.Dialer
	HTTP   *http.Client
	// GraphURL replaces https://graph.microsoft.com/v1.0 (tests).
	GraphURL string
	// InsecureIMAP connects without TLS (in-process test servers only).
	InsecureIMAP bool
	Timeout      time.Duration
}

// Box is an open seed mailbox.
type Box interface {
	// Folders lists where copies can land: the inbox, the tabs the mailbox
	// can tell apart, and the spam folder.
	Folders(ctx context.Context) ([]Folder, error)
	// Find returns the copies carrying any of tokens, received since since.
	Find(ctx context.Context, tokens []string, since time.Time) ([]Found, error)
	// Delete removes the copies carrying any of tokens (found again by
	// token, so only test copies are touched) and says how many.
	Delete(ctx context.Context, tokens []string, since time.Time) (int, error)
	Close() error
}

// Open signs in to a seed.
func Open(ctx context.Context, c Config) (Box, error) {
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.Auth == AuthOAuth && IsMicrosoft(c.Provider) {
		return openGraph(ctx, c)
	}
	return openIMAP(ctx, c)
}

// tokenSet is a lower-cased set of tokens.
func tokenSet(tokens []string) map[string]bool {
	m := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		m[strings.ToLower(t)] = true
	}
	return m
}

// matchToken returns the first token of h that is in want.
func matchToken(h Header, want map[string]bool) string {
	for _, t := range h.Tokens() {
		if want[t] {
			return t
		}
	}
	return ""
}

// FolderPath is how a copy's folder reads in the copy detail: the
// mailbox's own path ("[Gmail]/Spam", "Junk Email").
func FolderPath(f Found) string { return f.Folder }
