package seedbox

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"golang.org/x/oauth2"
)

type imapBox struct {
	c      *imapclient.Client
	cfg    Config
	cancel context.CancelFunc
	// token is the access token used to sign in (OAuth), for the Gmail
	// search connection.
	token   string
	folders []Folder
}

func (c Config) addr() string {
	port := c.Port
	if port == 0 {
		port = 993
	}
	return net.JoinHostPort(c.Host, strconv.Itoa(port))
}

func (c Config) dialer() *net.Dialer {
	if c.Dialer == nil {
		return &net.Dialer{Timeout: c.Timeout}
	}
	d := *c.Dialer
	d.Timeout = c.Timeout
	return &d
}

func (c Config) username() string {
	if c.Username != "" {
		return c.Username
	}
	return c.Email
}

// accessToken gets a fresh OAuth access token. A refresh the provider
// refuses (invalid_grant: revoked, expired, password changed) is
// ErrAuthExpired.
func accessToken(ts oauth2.TokenSource) (string, error) {
	if ts == nil {
		return "", fmt.Errorf("%w: no OAuth token", ErrAuthExpired)
	}
	tok, err := ts.Token()
	if err != nil {
		var re *oauth2.RetrieveError
		if errors.As(err, &re) && (re.ErrorCode == "invalid_grant" || re.ErrorCode == "unauthorized_client" || re.Response != nil && re.Response.StatusCode == http.StatusUnauthorized) {
			return "", fmt.Errorf("%w: %s", ErrAuthExpired, firstNonEmpty(re.ErrorDescription, re.ErrorCode))
		}
		if errors.As(err, &re) && re.Response != nil && re.Response.StatusCode == http.StatusBadRequest {
			return "", fmt.Errorf("%w: %s", ErrAuthExpired, firstNonEmpty(re.ErrorDescription, re.ErrorCode))
		}
		return "", fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	return tok.AccessToken, nil
}

func dialIMAP(cfg Config) (*imapclient.Client, error) {
	opts := &imapclient.Options{
		TLSConfig: &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12},
		Dialer:    cfg.dialer(),
	}
	if cfg.InsecureIMAP {
		return imapclient.DialInsecure(cfg.addr(), opts)
	}
	return imapclient.DialTLS(cfg.addr(), opts)
}

func openIMAP(ctx context.Context, cfg Config) (Box, error) {
	if cfg.Host == "" {
		return nil, errors.New("seed mailbox: the IMAP server is missing")
	}
	var token string
	if cfg.Auth == AuthOAuth {
		t, err := accessToken(cfg.Token)
		if err != nil {
			return nil, err
		}
		token = t
	}
	c, err := dialIMAP(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	cctx, cancel := context.WithCancel(ctx)
	go func() {
		<-cctx.Done()
		_ = c.Close()
	}()
	b := &imapBox{c: c, cfg: cfg, cancel: cancel, token: token}
	if cfg.Auth == AuthOAuth {
		err = c.Authenticate(&xoauth2{user: cfg.username(), token: token})
	} else {
		err = c.Login(cfg.username(), cfg.Password).Wait()
	}
	if err != nil {
		_ = b.Close()
		var ie *imap.Error
		if errors.As(err, &ie) {
			return nil, fmt.Errorf("%w: %s", ErrAuthFailed, ie.Text)
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	return b, nil
}

// spamNames are the spam folders' names when the server doesn't mark them
// \Junk.
var spamNames = []string{"[gmail]/spam", "[google mail]/spam", "spam", "junk", "junk email", "junk e-mail", "bulk", "bulk mail", "spamverdacht", "junk-e-mail"}

func (b *imapBox) Folders(_ context.Context) ([]Folder, error) {
	if b.folders != nil {
		return b.folders, nil
	}
	caps := b.c.Caps()
	var opts *imap.ListOptions
	if caps.Has(imap.CapSpecialUse) && (caps.Has(imap.CapListExtended) || caps.Has(imap.CapIMAP4rev2)) {
		opts = &imap.ListOptions{ReturnSpecialUse: true}
	}
	list, err := b.c.List("", "*", opts).Collect()
	if err != nil {
		return nil, err
	}
	p, _ := ProviderOf(b.cfg.Provider)
	inbox := Folder{Role: RoleInbox, Name: "INBOX", Label: firstNonEmpty(p.InboxName, "Inbox")}
	var spam *Folder
	for _, m := range list {
		for _, a := range m.Attrs {
			if a == imap.MailboxAttrJunk && spam == nil {
				spam = &Folder{Role: RoleSpam, Name: m.Mailbox, Label: spamLabel(p, m.Mailbox)}
			}
		}
	}
	if spam == nil {
		for _, want := range spamNames {
			for _, m := range list {
				if strings.EqualFold(m.Mailbox, want) {
					spam = &Folder{Role: RoleSpam, Name: m.Mailbox, Label: spamLabel(p, m.Mailbox)}
					break
				}
			}
			if spam != nil {
				break
			}
		}
	}
	out := []Folder{inbox}
	if b.gmail() {
		// Primary is the inbox itself; the other four are its tabs.
		for _, t := range gmailTabs {
			out = append(out, Folder{Role: RoleTab, Name: t.query, Label: t.tab})
		}
	}
	if spam != nil {
		out = append(out, *spam)
	}
	b.folders = out
	return out, nil
}

// spamLabel is the provider's word for its spam folder, from the mailbox
// name when the provider isn't known.
func spamLabel(p Provider, name string) string {
	if p.SpamName != "" {
		return p.SpamName
	}
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

// gmail reports whether the server is Gmail's (its search extension reads tabs).
func (b *imapBox) gmail() bool {
	return b.c.Caps().Has(imap.Cap("X-GM-EXT-1"))
}

// found is one match in a folder.
type found struct {
	uid  imap.UID
	tok  string
	date time.Time
	hdr  Header
}

// scan finds the messages of mailbox carrying a wanted token. It selects
// the mailbox read-only unless write is set.
func (b *imapBox) scan(mailbox string, want map[string]bool, since time.Time, write bool) ([]found, uint32, error) {
	sel, err := b.c.Select(mailbox, &imap.SelectOptions{ReadOnly: !write}).Wait()
	if err != nil {
		return nil, 0, err
	}
	if sel.NumMessages == 0 {
		return nil, sel.UIDValidity, nil
	}
	crit := &imap.SearchCriteria{}
	if !since.IsZero() {
		// SINCE compares dates only, in the server's zone: a day earlier is safe.
		crit.Since = since.Add(-24 * time.Hour)
	}
	data, err := b.c.UIDSearch(crit, nil).Wait()
	if err != nil {
		return nil, sel.UIDValidity, err
	}
	uids := data.AllUIDs()
	if len(uids) == 0 {
		return nil, sel.UIDValidity, nil
	}
	// First the two header fields that carry tokens, then whole headers of
	// the matches only.
	fields := &imap.FetchItemBodySection{Specifier: imap.PartSpecifierHeader, HeaderFields: []string{TestHeader, "Message-ID"}, Peek: true}
	msgs, err := b.c.Fetch(imap.UIDSetNum(uids...), &imap.FetchOptions{UID: true, BodySection: []*imap.FetchItemBodySection{fields}}).Collect()
	if err != nil {
		return nil, sel.UIDValidity, err
	}
	var hits []imap.UID
	toks := map[imap.UID]string{}
	for _, m := range msgs {
		if t := matchToken(ParseHeader(m.FindBodySection(fields)), want); t != "" {
			hits = append(hits, m.UID)
			toks[m.UID] = t
		}
	}
	if len(hits) == 0 {
		return nil, sel.UIDValidity, nil
	}
	full := &imap.FetchItemBodySection{Specifier: imap.PartSpecifierHeader, Peek: true}
	msgs, err = b.c.Fetch(imap.UIDSetNum(hits...), &imap.FetchOptions{UID: true, InternalDate: true, BodySection: []*imap.FetchItemBodySection{full}}).Collect()
	if err != nil {
		return nil, sel.UIDValidity, err
	}
	out := make([]found, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, found{uid: m.UID, tok: toks[m.UID], date: m.InternalDate, hdr: ParseHeader(m.FindBodySection(full))})
	}
	return out, sel.UIDValidity, nil
}

func (b *imapBox) Find(ctx context.Context, tokens []string, since time.Time) ([]Found, error) {
	folders, err := b.Folders(ctx)
	if err != nil {
		return nil, err
	}
	want := tokenSet(tokens)
	var out []Found
	for _, f := range folders {
		if f.Role == RoleTab {
			continue
		}
		hits, validity, err := b.scan(f.Name, want, since, false)
		if err != nil {
			return out, err
		}
		var tabs map[imap.UID]string
		if f.Role == RoleInbox && b.gmail() && len(hits) > 0 {
			uids := make([]imap.UID, len(hits))
			for i, h := range hits {
				uids[i] = h.uid
			}
			tabs, err = b.gmailTabs(ctx, f.Name, uids)
			if err != nil {
				return out, err
			}
		}
		for _, h := range hits {
			fd := Found{Token: h.tok, Role: f.Role, Folder: f.Label, ReceivedAt: h.date, Header: h.hdr,
				Ref: Ref{Folder: f.Name, UID: uint32(h.uid), UIDValidity: validity}}
			if tabs != nil {
				fd.Tab = tabs[h.uid]
				if fd.Tab != "Primary" {
					fd.Role = RoleTab
				}
			}
			out = append(out, fd)
			delete(want, h.tok)
		}
		if len(want) == 0 {
			break
		}
	}
	return out, nil
}

// gmailTabs reads which tab each inbox message is in through Gmail's
// search extension; messages in none of the categories are in Primary.
func (b *imapBox) gmailTabs(ctx context.Context, mailbox string, uids []imap.UID) (map[imap.UID]string, error) {
	nums := make([]uint32, len(uids))
	for i, u := range uids {
		nums[i] = uint32(u)
	}
	cats, err := gmailCategories(ctx, b.cfg, b.token, mailbox, nums)
	if err != nil {
		return nil, err
	}
	out := make(map[imap.UID]string, len(uids))
	for _, u := range uids {
		out[u] = "Primary"
		if c, ok := cats[uint32(u)]; ok {
			out[u] = c
		}
	}
	return out, nil
}

func (b *imapBox) Delete(ctx context.Context, tokens []string, since time.Time) (int, error) {
	folders, err := b.Folders(ctx)
	if err != nil {
		return 0, err
	}
	want := tokenSet(tokens)
	trash := ""
	if b.gmail() {
		// Gmail archives a message that is only expunged from the inbox:
		// move it to the trash and expunge it there.
		list, err := b.c.List("", "*", nil).Collect()
		if err == nil {
			for _, m := range list {
				for _, a := range m.Attrs {
					if a == imap.MailboxAttrTrash {
						trash = m.Mailbox
					}
				}
			}
		}
		if trash == "" {
			trash = "[Gmail]/Trash"
		}
	}
	n := 0
	for _, f := range folders {
		if f.Role == RoleTab {
			continue
		}
		hits, _, err := b.scan(f.Name, want, since, true)
		if err != nil {
			return n, err
		}
		if len(hits) == 0 {
			continue
		}
		var set imap.UIDSet
		for _, h := range hits {
			set.AddNum(h.uid)
		}
		if trash != "" && !strings.EqualFold(f.Name, trash) {
			if _, err := b.c.Move(set, trash).Wait(); err != nil {
				return n, err
			}
		} else if err := b.expunge(set); err != nil {
			return n, err
		}
		n += len(hits)
	}
	if trash != "" && n > 0 {
		hits, _, err := b.scan(trash, want, since, true)
		if err == nil && len(hits) > 0 {
			var set imap.UIDSet
			for _, h := range hits {
				set.AddNum(h.uid)
			}
			_ = b.expunge(set)
		}
	}
	return n, nil
}

// expunge removes set from the selected mailbox.
func (b *imapBox) expunge(set imap.UIDSet) error {
	if err := b.c.Store(set, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagDeleted}}, nil).Close(); err != nil {
		return err
	}
	if b.c.Caps().Has(imap.CapUIDPlus) || b.c.Caps().Has(imap.CapIMAP4rev2) {
		return b.c.UIDExpunge(set).Close()
	}
	return b.c.Expunge().Close()
}

func (b *imapBox) Close() error {
	_ = b.c.Logout().Wait()
	b.cancel()
	return b.c.Close()
}

// xoauth2 is the SASL XOAUTH2 mechanism Gmail and Outlook take: the
// address and a bearer token. On failure the server sends a JSON error as a
// challenge, answered with an empty response to get the final NO.
type xoauth2 struct {
	user, token string
}

func (x *xoauth2) Start() (string, []byte, error) {
	return "XOAUTH2", []byte("user=" + x.user + "\x01auth=Bearer " + x.token + "\x01\x01"), nil
}

func (x *xoauth2) Next([]byte) ([]byte, error) { return []byte{}, nil }
