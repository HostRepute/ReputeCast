package bouncebox

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

type imapBox struct {
	c      *imapclient.Client
	cfg    Config
	cancel context.CancelFunc
}

func openIMAP(ctx context.Context, cfg Config) (Box, error) {
	opts := &imapclient.Options{
		TLSConfig: &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12},
		Dialer:    &net.Dialer{Timeout: cfg.Timeout},
	}
	var (
		c   *imapclient.Client
		err error
	)
	switch cfg.TLS {
	case "starttls":
		c, err = imapclient.DialStartTLS(cfg.addr(), opts)
	case "none":
		c, err = imapclient.DialInsecure(cfg.addr(), opts)
	default:
		c, err = imapclient.DialTLS(cfg.addr(), opts)
	}
	if err != nil {
		return nil, err
	}
	// go-imap commands don't take a context: close the connection when the
	// caller's context ends so nothing blocks forever.
	cctx, cancel := context.WithCancel(ctx)
	go func() {
		<-cctx.Done()
		_ = c.Close()
	}()
	b := &imapBox{c: c, cfg: cfg, cancel: cancel}
	if err := c.Login(cfg.Username, cfg.Password).Wait(); err != nil {
		b.Close()
		var ie *imap.Error
		if errors.As(err, &ie) && (ie.Type == imap.StatusResponseTypeNo || ie.Code == imap.ResponseCodeAuthenticationFailed) {
			return nil, ErrAuth
		}
		return nil, err
	}
	folder := cfg.Folder
	if folder == "" {
		folder = "INBOX"
	}
	if _, err := c.Select(folder, nil).Wait(); err != nil {
		b.Close()
		return nil, err
	}
	return b, nil
}

func (b *imapBox) Fetch(_ context.Context, max int) ([]Message, error) {
	data, err := b.c.UIDSearch(&imap.SearchCriteria{NotFlag: []imap.Flag{imap.FlagSeen}}, nil).Wait()
	if err != nil {
		return nil, err
	}
	uids := data.AllUIDs()
	if len(uids) == 0 {
		return []Message{}, nil
	}
	if max > 0 && len(uids) > max {
		uids = uids[:max]
	}
	section := &imap.FetchItemBodySection{Peek: true}
	msgs, err := b.c.Fetch(imap.UIDSetNum(uids...), &imap.FetchOptions{UID: true, BodySection: []*imap.FetchItemBodySection{section}}).Collect()
	if err != nil {
		return nil, err
	}
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		raw := m.FindBodySection(section)
		if len(raw) == 0 {
			continue
		}
		out = append(out, Message{ID: strconv.FormatUint(uint64(m.UID), 10), Raw: raw})
	}
	return out, nil
}

func (b *imapBox) Ack(_ context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	var set imap.UIDSet
	for _, id := range ids {
		n, err := strconv.ParseUint(strings.TrimSpace(id), 10, 32)
		if err != nil {
			continue
		}
		set.AddNum(imap.UID(n))
	}
	return b.c.Store(set, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagSeen}}, nil).Close()
}

func (b *imapBox) Close() error {
	_ = b.c.Logout().Wait()
	b.cancel()
	return b.c.Close()
}
