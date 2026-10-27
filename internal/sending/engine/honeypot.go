package engine

import (
	"context"
	"time"
)

// Bot clicks (review AN-2): every email with tracked links carries one
// more link that no person can see or reach with a keyboard or a screen
// reader. Scanners that follow every link follow it too, and the clicks
// around it settle as the scanner's (internal/tracking).

// HoneypotVar is the Liquid variable holding the hidden link's URL.
const HoneypotVar = "rc_honeypot_url"

// honeypot is the hidden link: no text, hidden from layout, assistive
// technology and the keyboard, so only a machine can follow it.
const honeypot = `<a href="{{ ` + HoneypotVar + ` }}" aria-hidden="true" tabindex="-1" style="display:none;font-size:0;line-height:0;max-height:0;max-width:0;overflow:hidden;mso-hide:all"></a>`

// addHoneypot puts the hidden link at the end of the body.
func addHoneypot(src string) string {
	if loc := bodyCloseR.FindStringIndex(src); loc != nil {
		return src[:loc[0]] + honeypot + src[loc[0]:]
	}
	return src + honeypot
}

// settler settles the clicks the demo transport's opens and clicks left
// pending (the tracking recorder implements it).
type settler interface {
	SettleEngagement(ctx context.Context, now time.Time) (*time.Time, error)
}

// settleClicks settles the clicks due at now and returns when the next is
// due, so the demo transport's pump (and the virtual clock of tests)
// advances to it.
func (e *Engine) settleClicks(ctx context.Context, now time.Time) *time.Time {
	st, ok := e.Recorder.(settler)
	if !ok {
		return nil
	}
	next, err := st.SettleEngagement(ctx, now)
	if err != nil {
		e.log().WarnContext(ctx, "settle clicks", "err", err)
		return nil
	}
	return next
}

func earliest(a, b *time.Time) *time.Time {
	switch {
	case a == nil:
		return b
	case b == nil || a.Before(*b):
		return a
	}
	return b
}
