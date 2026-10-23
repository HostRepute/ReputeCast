package feed

import (
	"embed"
	"errors"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
)

// Why a feed can't be read, as people read it: each error keeps its English
// (Error, FetchError.Detail) and the same sentence as a message of
// messages/<locale>.json (an i18n.Text), which a screen writes in its
// reader's language and a digest stores for whoever reads it later.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "digests.feed")

var (
	mAddressEmpty    = msgs.M("address.empty")
	mAddressInvalid  = msgs.M("address.invalid")
	mAddressScheme   = msgs.M("address.scheme")
	mAddressRedirect = msgs.M("address.redirect")

	mStatusNoLocation = msgs.M("status.noLocation")
	mStatusNotFound   = msgs.M("status.notFound")
	mStatusSignIn     = msgs.M("status.signIn")
	mStatusTooMany    = msgs.M("status.tooMany")
	mStatusOther      = msgs.M("status.other")
	mRedirects        = msgs.M("redirects")
	mTooLarge         = msgs.M("tooLarge")

	mBlockedPrivate   = msgs.M("blocked.private")
	mBlockedSpecial   = msgs.M("blocked.special")
	mBlockedLocalName = msgs.M("blocked.localName")
	mBlockedOther     = msgs.M("blocked.other")

	mTimeout     = msgs.M("unreachable.timeout")
	mDNS         = msgs.M("unreachable.dns")
	mTLS         = msgs.M("unreachable.tls")
	mRefused     = msgs.M("unreachable.refused")
	mReset       = msgs.M("unreachable.reset")
	mUnreachable = msgs.M("unreachable.other")

	mNoFeedLink = msgs.M("noFeedLink")
)

// Codes of a NotFeedError: what answered instead of a feed.
const (
	notFeedHTML        = "html"
	notFeedEmpty       = "empty"
	notFeedNeither     = "neither"
	notFeedXML         = "xml"
	notFeedDeep        = "deep"
	notFeedNoElements  = "noElements"
	notFeedNoChannel   = "noChannel"
	notFeedRoot        = "root"
	notFeedJSON        = "json"
	notFeedNotJSONFeed = "notJSONFeed"
)

// AddressError is an address that can't be a feed's: empty, not an
// address, or not http(s).
type AddressError struct {
	msg string
	// Text is the same sentence, to write in a reader's language.
	Text i18n.Text
}

func (e *AddressError) Error() string { return e.msg }

func addressError(m i18n.Msg) *AddressError {
	return &AddressError{msg: m.Source(), Text: m.Keep(nil)}
}

// Text is why the address isn't a feed, as the sentence "blog.acme.io/feed/
// isn't a feed: the document is empty", for an address written without its
// scheme.
func (e *NotFeedError) Text(address string) i18n.Text {
	code := e.Code
	if e.HTML {
		code = notFeedHTML
	}
	if code == "" {
		code = "plain"
	}
	m, err := msgs.Lookup("notFeed." + code)
	if err != nil {
		m = msgs.M("notFeed.plain")
	}
	args := i18n.Args{"address": address}
	switch code {
	case notFeedXML, notFeedJSON:
		args["detail"] = e.Arg
	case notFeedRoot:
		args["root"] = e.Arg
	}
	return m.Keep(args)
}

// Text is "blog.acme.io is a web page that links to no feed".
func (e *NoFeedLinkError) Text() i18n.Text {
	return mNoFeedLink.Keep(i18n.Args{"address": shortURL(e.Page)})
}

// TextOf is why a read failed, as a message, when err knows it: a
// FetchError, an AddressError, or (for the address it was read at) a page
// that isn't a feed or links to none. It is zero for other errors.
func TextOf(err error, address string) i18n.Text {
	var ae *AddressError
	var nf *NotFeedError
	var nl *NoFeedLinkError
	switch {
	case errors.As(err, &ae):
		return ae.Text
	case errors.As(err, &nf):
		return nf.Text(shortURL(address))
	case errors.As(err, &nl):
		return nl.Text()
	}
	if fe, ok := AsFetchError(err); ok {
		return fe.Text
	}
	return i18n.Text{}
}

// GuardText is why the guard refused an address it checked before
// reading, when it refused it for where it points (a private address):
// false for its other answers, which keep their own words.
func GuardText(address string, err error) (i18n.Text, bool) {
	if !netguard.IsBlocked(err) {
		return i18n.Text{}, false
	}
	_, t := blockedText(hostOf(address), blockedReason(err))
	return t, true
}

// blockedText words an address the guard refused: "blog.acme.io can't be
// read: 10.0.0.4 is a private or loopback address. An instance admin can
// allow private addresses for self-hosted feeds."
func blockedText(host, reason string) (string, i18n.Text) {
	args := i18n.Args{"host": host}
	m := mBlockedOther
	switch {
	case strings.HasSuffix(reason, " is a private or loopback address"):
		m, args["ip"] = mBlockedPrivate, strings.TrimSuffix(reason, " is a private or loopback address")
	case strings.HasSuffix(reason, " is a link-local, metadata or multicast address"):
		m, args["ip"] = mBlockedSpecial, strings.TrimSuffix(reason, " is a link-local, metadata or multicast address")
	case strings.HasSuffix(reason, " is a local hostname"):
		m, args["name"] = mBlockedLocalName, strings.TrimSuffix(reason, " is a local hostname")
	default:
		args["reason"] = reason
	}
	return m.Text(i18n.Default, args), m.Keep(args)
}

// blockedReason is what the guard said, without its "address not allowed: ".
func blockedReason(err error) string {
	msg := err.Error()
	if i := strings.Index(msg, netguard.ErrBlocked.Error()+": "); i >= 0 {
		msg = msg[i+len(netguard.ErrBlocked.Error())+2:]
	}
	return msg
}

// english writes a message's English, the Detail every error keeps.
func english(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}
