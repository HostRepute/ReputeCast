package inbound

import (
	"net/textproto"
	"strings"
	"unicode"
)

// unsubscribeWords are what a reply asking to be removed says, in the app's
// languages: the subject or the first line that isn't quoted equals or
// starts with one (case and punctuation aside). They mark a reply "Asks to
// unsubscribe"; nobody is unsubscribed by them.
var unsubscribeWords = []string{
	// English
	"unsubscribe", "unsubscribe me", "remove me", "please remove me", "please unsubscribe", "please unsubscribe me",
	"take me off", "please take me off", "stop", "stop emailing me", "stop sending", "opt out", "opt-out",
	// Vietnamese
	"hủy đăng ký", "huỷ đăng ký", "huy dang ky", "dừng gửi", "ngừng gửi", "xóa tôi", "xoá tôi",
	// German
	"abmelden", "abbestellen", "bitte abmelden", "austragen", "bitte austragen", "keine emails mehr", "keine e-mails mehr",
	// Spanish
	"darse de baja", "dar de baja", "darme de baja", "baja", "cancelar suscripción", "cancelar suscripcion", "no más correos", "no mas correos",
	// French
	"se désabonner", "se desabonner", "désabonner", "desabonner", "désinscrire", "desinscrire", "me désinscrire", "me desinscrire",
	"désabonnement", "desabonnement",
}

// asksUnsubscribe reports whether a reply's subject (without Re:) or first
// line of its own text asks to be removed.
func asksUnsubscribe(subject, text string) bool {
	if startsWithWord(stripReply(subject)) {
		return true
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ">") {
			continue
		}
		return startsWithWord(line)
	}
	return false
}

func startsWithWord(s string) bool {
	s = normalizeWords(s)
	if s == "" {
		return false
	}
	for _, w := range unsubscribeWords {
		// A short word ("stop", "baja") counts only alone: "Stop by our
		// booth" isn't a request.
		if s == w || len(w) > 5 && strings.HasPrefix(s, w+" ") {
			return true
		}
	}
	return false
}

// normalizeWords lower-cases s and turns punctuation into single spaces.
func normalizeWords(s string) string {
	var b strings.Builder
	space := true
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			b.WriteRune(r)
			space = false
			continue
		}
		if !space {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(b.String())
}

// replyPrefixes are subject prefixes of replies and forwards.
var replyPrefixes = []string{"re:", "aw:", "sv:", "vs:", "antw:", "rif:", "réf:", "ref:", "tr:", "fw:", "fwd:", "wg:", "rv:"}

func stripReply(s string) string {
	for {
		t := strings.TrimSpace(s)
		l := strings.ToLower(t)
		cut := false
		for _, p := range replyPrefixes {
			if strings.HasPrefix(l, p) {
				t, cut = t[len(p):], true
				break
			}
		}
		if !cut {
			return t
		}
		s = t
	}
}

// isAutoReply reports whether headers mark a message as automatic: an
// out-of-office answer, a vacation responder, a list or bulk mailer, or a
// bounce (an empty return path).
func isAutoReply(h textproto.MIMEHeader) bool {
	if v := strings.ToLower(strings.TrimSpace(h.Get("Auto-Submitted"))); v != "" && v != "no" {
		return true
	}
	if h.Get("X-Autoreply") != "" || h.Get("X-Autorespond") != "" || h.Get("X-Autoresponder") != "" {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(h.Get("Precedence"))) {
	case "auto_reply", "bulk", "junk", "list":
		return true
	}
	if rp, ok := h["Return-Path"]; ok && len(rp) > 0 && strings.TrimSpace(rp[0]) == "<>" {
		return true
	}
	return false
}

// spamVerdict reads a spam verdict from headers when the provider gave none.
func spamVerdict(provider string, h textproto.MIMEHeader) bool {
	switch strings.ToLower(provider) {
	case "spam":
		return true
	case "pass":
		return false
	}
	if v := strings.ToLower(h.Get("X-Spam-Flag")); v == "yes" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(h.Get("X-Spam-Status"))), "yes")
}
