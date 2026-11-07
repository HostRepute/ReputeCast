package connect_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// A saved secret is sent only to the address it was entered for: a
// connection form that points the provider somewhere else needs the secret
// typed again, and nothing is sent before it is.

// problemAt returns the problem's errors by location.
func problemAt(t *testing.T, r *apptest.Response) map[string]string {
	t.Helper()
	var p struct {
		Detail string
		Errors []struct{ Location, Message string }
	}
	r.JSON(t, &p)
	out := map[string]string{}
	for _, e := range p.Errors {
		out[e.Location] = e.Message
	}
	return out
}

// fakePostal answers Postal's credential check and keeps the keys it got.
type fakePostal struct {
	*httptest.Server
	mu   sync.Mutex
	keys []string
}

func newFakePostal(t *testing.T) *fakePostal {
	f := &fakePostal{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.keys = append(f.keys, r.Header.Get("X-Server-API-Key"))
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "error", "time": 0.01, "flags": map[string]any{},
			"data": map[string]any{"code": "MessageNotFound", "message": "No message found matching provided ID", "id": 0}})
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakePostal) got() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.keys)
}

func TestSavedAPIKeyStaysWithItsServerURL(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	mine, theirs := newFakePostal(t), newFakePostal(t)
	var st setupT
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "postal"}).MustStatus(t, 201).JSON(t, &st)
	base := ws + "/connect/setups/" + st.ID
	c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"server_url": mine.URL, "api_key": "postal-key-1"}}).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.Status != "connected" || !slices.Contains(mine.got(), "postal-key-1") {
		t.Fatalf("connected: %+v, keys %v", st.Connection, mine.got())
	}

	// Another server URL with the key left blank: refused, next to the key,
	// and the other server hears nothing.
	r := c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"server_url": theirs.URL, "api_key": ""}}).MustStatus(t, 422)
	if got := problemAt(t, r)["body.values.api_key"]; got != "The address changed: enter the API key again. ReputeCast doesn't send a saved secret to a new address." {
		t.Fatalf("problem: %s", r.Body)
	}
	if len(theirs.got()) != 0 {
		t.Fatalf("the saved key reached another server: %v", theirs.got())
	}
	var cfg string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT config->>'server_url' FROM providers WHERE id = $1`, st.ProviderID).Scan(&cfg); err != nil || cfg != mine.URL {
		t.Fatalf("server URL %q (%v)", cfg, err)
	}

	// The same server (written with a final slash) keeps the saved key.
	before := len(mine.got())
	c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"server_url": mine.URL + "/"}}).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.Status != "connected" || len(mine.got()) == before || mine.got()[len(mine.got())-1] != "postal-key-1" {
		t.Fatalf("same server: %+v, keys %v", st.Connection, mine.got())
	}

	// A new key for the new server goes there, and only it does.
	c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"server_url": theirs.URL, "api_key": "postal-key-2"}}).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.Status != "connected" || slices.Contains(theirs.got(), "postal-key-1") || !slices.Contains(theirs.got(), "postal-key-2") {
		t.Fatalf("new server: %+v, keys %v", st.Connection, theirs.got())
	}
}

func TestSavedSMTPPasswordStaysWithItsServer(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	mine, theirs := apptest.NewSMTPSink(t), apptest.NewSMTPSink(t)
	host, port, _ := net.SplitHostPort(mine.Addr)
	myPort, _ := strconv.Atoi(port)
	_, port, _ = net.SplitHostPort(theirs.Addr)
	theirPort, _ := strconv.Atoi(port)
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var st setupT
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "smtp"}).MustStatus(t, 201).JSON(t, &st)
	base := ws + "/connect/setups/" + st.ID
	smtp := func(host string, port int, password string) map[string]any {
		return map[string]any{"smtp": map[string]any{"host": host, "port": port, "tls": "none", "username": "acme", "password": password}}
	}
	c.Do("POST", base+"/connection", smtp(host, myPort, "smtp-pass-1")).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.Status != "connected" || !slices.Equal(mine.Passwords(), []string{"smtp-pass-1"}) {
		t.Fatalf("connected: %+v, passwords %v", st.Connection, mine.Passwords())
	}

	// Another host, or another port on the same host, with the password
	// left blank: refused before anything is dialed.
	for _, to := range []struct {
		host string
		port int
	}{{host, theirPort}, {"localhost", myPort}, {"smtp.elsewhere.example", 587}} {
		r := c.Do("POST", base+"/connection", smtp(to.host, to.port, "")).MustStatus(t, 422)
		if got := problemAt(t, r)["body.smtp.password"]; got != "The server's address changed: enter the password again. ReputeCast doesn't send a saved password to a new address." {
			t.Fatalf("%s:%d: %s", to.host, to.port, r.Body)
		}
	}
	if len(theirs.Passwords()) != 0 || len(mine.Passwords()) != 1 {
		t.Fatalf("a saved password was sent: theirs %v, mine %v", theirs.Passwords(), mine.Passwords())
	}

	// The same server keeps the saved password.
	c.Do("POST", base+"/connection", smtp(strings.ToUpper(host), myPort, "")).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.Status != "connected" || !slices.Equal(mine.Passwords(), []string{"smtp-pass-1", "smtp-pass-1"}) {
		t.Fatalf("same server: %+v, passwords %v", st.Connection, mine.Passwords())
	}
	// A password typed for the new server goes there.
	c.Do("POST", base+"/connection", smtp(host, theirPort, "smtp-pass-2")).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.Status != "connected" || !slices.Equal(theirs.Passwords(), []string{"smtp-pass-2"}) {
		t.Fatalf("new server: %+v, passwords %v", st.Connection, theirs.Passwords())
	}
}

// pop3Box is a POP3 server that takes any login, keeps the passwords and
// has no messages.
type pop3Box struct {
	addr      string
	mu        sync.Mutex
	passwords []string
}

func newPOP3Box(t *testing.T) *pop3Box {
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	b := &pop3Box{addr: ln.Addr().String()}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				r := bufio.NewReader(conn)
				_, _ = conn.Write([]byte("+OK ready\r\n"))
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					cmd, arg, _ := strings.Cut(strings.TrimSpace(line), " ")
					switch strings.ToUpper(cmd) {
					case "PASS":
						b.mu.Lock()
						b.passwords = append(b.passwords, arg)
						b.mu.Unlock()
						_, _ = conn.Write([]byte("+OK logged in\r\n"))
					case "STAT":
						_, _ = conn.Write([]byte("+OK 0 0\r\n"))
					case "QUIT":
						_, _ = conn.Write([]byte("+OK bye\r\n"))
						return
					default:
						_, _ = conn.Write([]byte("+OK\r\n"))
					}
				}
			}()
		}
	}()
	return b
}

func (b *pop3Box) got() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.Clone(b.passwords)
}

func TestSavedMailboxPasswordStaysWithItsServer(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	sink := apptest.NewSMTPSink(t)
	host, port, _ := net.SplitHostPort(sink.Addr)
	n, _ := strconv.Atoi(port)
	mine, theirs := newPOP3Box(t), newPOP3Box(t)
	_, port, _ = net.SplitHostPort(mine.addr)
	myPort, _ := strconv.Atoi(port)
	_, port, _ = net.SplitHostPort(theirs.addr)
	theirPort, _ := strconv.Atoi(port)
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var st setupT
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "smtp"}).MustStatus(t, 201).JSON(t, &st)
	base := ws + "/connect/setups/" + st.ID
	c.Do("POST", base+"/connection", map[string]any{"smtp": map[string]any{"host": host, "port": n, "tls": "none", "username": "acme", "password": "smtp-pass"}}).MustStatus(t, 200)
	type mailbox struct {
		Connection struct {
			Mailbox *struct {
				OK    bool
				Error string
			}
		}
	}
	box := func(host string, port int, password string) map[string]any {
		return map[string]any{"address": "bounces@news.acme.io", "protocol": "pop3", "host": host, "port": port, "tls": "none",
			"username": "bounces@news.acme.io", "password": password, "pollEvery": "1m", "afterProcessing": "leave"}
	}
	var mb mailbox
	c.Do("POST", base+"/mailbox", box("127.0.0.1", myPort, "box-pass-1")).MustStatus(t, 200).JSON(t, &mb)
	if mb.Connection.Mailbox == nil || !mb.Connection.Mailbox.OK || !slices.Equal(mine.got(), []string{"box-pass-1"}) {
		t.Fatalf("mailbox: %+v, passwords %v", mb.Connection.Mailbox, mine.got())
	}

	// Another mailbox server with the password left blank: refused.
	for _, to := range []struct {
		host string
		port int
	}{{"127.0.0.1", theirPort}, {"localhost", myPort}} {
		r := c.Do("POST", base+"/mailbox", box(to.host, to.port, "")).MustStatus(t, 422)
		if got := problemAt(t, r)["body.password"]; got != "The mailbox server's address changed: enter the mailbox password again. ReputeCast doesn't send a saved password to a new address." {
			t.Fatalf("%s:%d: %s", to.host, to.port, r.Body)
		}
	}
	if len(theirs.got()) != 0 {
		t.Fatalf("the saved password reached another server: %v", theirs.got())
	}

	// The same server keeps it; a new password goes to the new server.
	c.Do("POST", base+"/mailbox", box("127.0.0.1", myPort, "")).MustStatus(t, 200).JSON(t, &mb)
	if !mb.Connection.Mailbox.OK || !slices.Equal(mine.got(), []string{"box-pass-1", "box-pass-1"}) {
		t.Fatalf("same server: %+v, passwords %v", mb.Connection.Mailbox, mine.got())
	}
	c.Do("POST", base+"/mailbox", box("127.0.0.1", theirPort, "box-pass-2")).MustStatus(t, 200).JSON(t, &mb)
	if !mb.Connection.Mailbox.OK || !slices.Equal(theirs.got(), []string{"box-pass-2"}) {
		t.Fatalf("new server: %+v, passwords %v", mb.Connection.Mailbox, theirs.got())
	}
}
