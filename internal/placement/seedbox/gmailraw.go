package seedbox

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// gmailTabs are Gmail's inbox categories and the tab names they show as.
var gmailTabs = []struct{ query, tab string }{
	{"category:promotions", "Promotions"},
	{"category:updates", "Updates"},
	{"category:social", "Social"},
	{"category:forums", "Forums"},
}

// gmailCategories runs Gmail's X-GM-RAW search ("UID SEARCH UID 4,7
// X-GM-RAW "category:promotions"") for each category on a short connection
// of its own (go-imap can't send the extension) and returns the tab of each
// uid found in one. It reads only search results: no message data.
func gmailCategories(ctx context.Context, cfg Config, token, mailbox string, uids []uint32) (map[uint32]string, error) {
	conn, err := dialRaw(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	defer func() { _ = conn.Close() }()
	s := &rawSession{r: bufio.NewReader(conn), w: conn}
	if err := conn.SetDeadline(time.Now().Add(cfg.Timeout)); err != nil {
		return nil, err
	}
	if _, err := s.line(); err != nil { // greeting
		return nil, err
	}
	if cfg.Auth == AuthOAuth {
		ir := base64.StdEncoding.EncodeToString([]byte("user=" + cfg.username() + "\x01auth=Bearer " + token + "\x01\x01"))
		if _, err := s.cmd("AUTHENTICATE XOAUTH2 " + ir); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrAuthFailed, err)
		}
	} else {
		if _, err := s.cmd("LOGIN " + quote(cfg.username()) + " " + quote(cfg.Password)); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrAuthFailed, err)
		}
	}
	if _, err := s.cmd("EXAMINE " + quote(mailbox)); err != nil {
		return nil, err
	}
	set := make([]string, len(uids))
	for i, u := range uids {
		set[i] = strconv.FormatUint(uint64(u), 10)
	}
	out := map[uint32]string{}
	for _, c := range gmailTabs {
		lines, err := s.cmd("UID SEARCH UID " + strings.Join(set, ",") + " X-GM-RAW " + quote(c.query))
		if err != nil {
			return nil, err
		}
		for _, l := range lines {
			if !strings.HasPrefix(l, "* SEARCH") {
				continue
			}
			for _, f := range strings.Fields(strings.TrimPrefix(l, "* SEARCH")) {
				if n, err := strconv.ParseUint(f, 10, 32); err == nil {
					if _, seen := out[uint32(n)]; !seen {
						out[uint32(n)] = c.tab
					}
				}
			}
		}
	}
	_, _ = s.cmd("LOGOUT")
	return out, nil
}

func dialRaw(ctx context.Context, cfg Config) (net.Conn, error) {
	d := cfg.dialer()
	if cfg.InsecureIMAP {
		return d.DialContext(ctx, "tcp", cfg.addr())
	}
	td := &tls.Dialer{NetDialer: d, Config: &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}}
	return td.DialContext(ctx, "tcp", cfg.addr())
}

// rawSession speaks just enough IMAP for the search: tagged commands and
// line responses (search results carry no literals).
type rawSession struct {
	r   *bufio.Reader
	w   io.Writer
	tag int
}

func (s *rawSession) line() (string, error) {
	l, err := s.r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(l, "\r\n"), nil
}

// cmd sends a command and returns its untagged lines; a NO or BAD answer
// is an error with the server's text.
func (s *rawSession) cmd(c string) ([]string, error) {
	s.tag++
	tag := "x" + strconv.Itoa(s.tag)
	if _, err := io.WriteString(s.w, tag+" "+c+"\r\n"); err != nil {
		return nil, err
	}
	var lines []string
	for {
		l, err := s.line()
		if err != nil {
			return lines, err
		}
		if strings.HasPrefix(l, "+") {
			// A SASL challenge (an error description): answer empty.
			if _, err := io.WriteString(s.w, "\r\n"); err != nil {
				return lines, err
			}
			continue
		}
		if !strings.HasPrefix(l, tag+" ") {
			lines = append(lines, l)
			continue
		}
		status := strings.TrimPrefix(l, tag+" ")
		if strings.HasPrefix(status, "OK") {
			return lines, nil
		}
		return lines, fmt.Errorf("%s", status)
	}
}

// quote writes an IMAP quoted string.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
