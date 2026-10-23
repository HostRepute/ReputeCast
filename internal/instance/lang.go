package instance

import (
	"context"
	"embed"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Instance admin reads in the reader's language (i18n.LangFrom): its
// problems, notices and the notifications it sends. The audit entries it
// writes keep their summaries' messages beside the English
// (audit.KeepSummary), and the notifications theirs (notify's TitleText…).

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "instance")

// say writes one of the package's messages in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// keep is one of the package's messages kept for later readers.
func keep(key string, args i18n.Args) i18n.Text { return msgs.M(key).Keep(args) }

// duration splits a length in minutes into the unit the policy's audit
// lines name it in, as before: whole days, whole hours, or minutes.
func duration(minutes int) (string, int) {
	switch {
	case minutes%(24*60) == 0:
		return "day", minutes / (24 * 60)
	case minutes%60 == 0:
		return "hour", minutes / 60
	}
	return "minute", minutes
}

// durationArgs are a message's arguments for two lengths in minutes, now
// and was: nowUnit, nowN, wasUnit, wasN.
func durationArgs(now, was int) i18n.Args {
	nu, nn := duration(now)
	wu, wn := duration(was)
	return i18n.Args{"nowUnit": nu, "nowN": nn, "wasUnit": wu, "wasN": wn}
}

// location is a workspace's timezone, where its members read its dates.
func location(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.UTC
}

// onOff is the select value of a switch.
func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}
