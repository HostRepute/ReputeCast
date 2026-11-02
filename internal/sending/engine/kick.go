package engine

import (
	"context"
	"sync"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// kickRecheck is how often a kick that found a pass running asks again
// until that pass has ended; kickTimeout bounds one insert of the job.
const (
	kickRecheck = 250 * time.Millisecond
	kickTimeout = 5 * time.Second
)

// kicker wakes an outbox's unique job after mail has committed, from one
// goroutine: a burst of wake-ups costs a few inserts, and no sender's
// transaction holds the job's row (review 2026-11 OP-13). A pass that is
// running may have read the queue before the mail committed, so a wake that
// finds one asks again until it has ended.
type kicker struct {
	once  sync.Once
	kicks chan struct{}
	wake  func() bool
}

// Kick asks for a pass and returns at once; wake inserts the job and reports
// whether a pass was running. The first call's wake is the one used.
func (k *kicker) Kick(wake func() bool) {
	k.once.Do(func() {
		k.wake = wake
		k.kicks = make(chan struct{}, 1)
		go k.run()
	})
	select {
	case k.kicks <- struct{}{}:
	default: // a kick is waiting already; it covers this mail too
	}
}

func (k *kicker) run() {
	for range k.kicks {
		for k.wake() {
			time.Sleep(kickRecheck)
		}
	}
}

// wakeOutbox inserts an outbox's unique job. It reports whether a pass was
// running, so the job wasn't queued again.
func (e *Engine) wakeOutbox(args river.JobArgs) bool {
	c := e.jobs()
	if c == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), kickTimeout)
	defer cancel()
	res, err := c.Insert(ctx, args, nil)
	if err != nil {
		e.log().WarnContext(ctx, "wake an outbox", "kind", args.Kind(), "err", err)
		return false
	}
	return res.UniqueSkippedAsDuplicate && res.Job != nil && res.Job.State == rivertype.JobStateRunning
}

// KickAutomationOutbox asks for a pass of the automation outbox for mail
// committed before the call (automation and milestone emails), as
// TransactionalOutbox.Kick does for transactional mail. Mail that a lost
// kick leaves is woken by the sending watch within 30 seconds.
func (e *Engine) KickAutomationOutbox() {
	e.automationKick.Kick(func() bool { return e.wakeOutbox(OutboxArgs{}) })
}
