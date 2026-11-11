package inbound

import (
	"context"
	"embed"
	"encoding/json"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What Settings › Senders' Receiving mail card, the setup dialog and the
// replies inbox read from the server follows the reader's app language
// (messages/<locale>.json): statuses, problems with their fixes, the test
// message, the notifications and the timeline's line. Provider names, DNS
// records, addresses and what people wrote stay as they are. Notifications
// keep their messages (notify's TitleText…), so each member reads them in
// their own language.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "inbound")

// say writes a message in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// text writes a message in a language.
func text(l i18n.Lang, key string, args i18n.Args) string {
	return msgs.M(key).Text(l, args)
}

// Sentences stored for later readers.
var (
	mUnsubscribedByEmail   = msgs.M("note.unsubscribedByEmail")
	mUnsubscribedFromReply = msgs.M("note.unsubscribedFromReply")
	mDroppedReports        = msgs.M("status.droppedReports")
)

// keepDetail stores a route's status sentence as its message, so each
// reader reads it in their language (inbound_routes.status_detail).
func keepDetail(t i18n.Text) string {
	b, err := json.Marshal(t)
	if err != nil {
		return ""
	}
	return string(b)
}

// detailIn writes a stored status sentence in a reader's language; a
// sentence stored as text stays as it is.
func detailIn(raw string, l i18n.Lang) string {
	if !strings.HasPrefix(raw, "{") {
		return raw
	}
	var t i18n.Text
	if err := json.Unmarshal([]byte(raw), &t); err != nil || t.IsZero() {
		return raw
	}
	if s := t.In(l); s != "" {
		return s
	}
	return ""
}
