package forms

import (
	"fmt"
	"testing"
	"time"
)

// Without Turnstile, plain posts have a budget of their own: three from
// one address, then one every 20 minutes; thirty to one form from every
// address, then one every two minutes.
func TestPlainBudget(t *testing.T) {
	now := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	t.Run("per address", func(t *testing.T) {
		l := &limiter{buckets: map[string]*bucket{}}
		for i := range plainPerAddress {
			if !l.plainAllowed("ws1", "frm_a", "203.0.113.7", now) {
				t.Fatalf("post %d from one address refused", i+1)
			}
		}
		if l.plainAllowed("ws1", "frm_b", "203.0.113.7", now) {
			t.Fatal("a fourth post from the address went through, to another form of the workspace")
		}
		if !l.plainAllowed("ws2", "frm_c", "203.0.113.7", now) {
			t.Fatal("another workspace's form counts the address's posts here")
		}
		// Idle buckets are swept after ten minutes, but one that takes an
		// hour to fill up is kept until then.
		l.allow("other", 5, now.Add(11*time.Minute))
		if l.plainAllowed("ws1", "frm_a", "203.0.113.7", now.Add(15*time.Minute)) {
			t.Fatal("the address's budget came back after 15 minutes")
		}
		if !l.plainAllowed("ws1", "frm_a", "203.0.113.7", now.Add(20*time.Minute)) {
			t.Fatal("no post 20 minutes later")
		}
	})
	t.Run("per form", func(t *testing.T) {
		l := &limiter{buckets: map[string]*bucket{}}
		for i := range plainPerForm {
			if !l.plainAllowed("ws1", "frm_a", fmt.Sprintf("203.0.113.%d", i), now) {
				t.Fatalf("post %d to the form refused", i+1)
			}
		}
		if l.plainAllowed("ws1", "frm_a", "198.51.100.1", now.Add(time.Minute)) {
			t.Fatal("post 31 to the form went through from a new address")
		}
		if !l.plainAllowed("ws1", "frm_a", "198.51.100.2", now.Add(2*time.Minute)) {
			t.Fatal("no post to the form two minutes later")
		}
		if !l.plainAllowed("ws1", "frm_b", "198.51.100.3", now.Add(2*time.Minute)) {
			t.Fatal("another form shares the budget")
		}
	})
}
