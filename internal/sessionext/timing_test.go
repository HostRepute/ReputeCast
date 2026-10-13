package sessionext

import (
	"testing"
	"time"
)

func TestTiming(t *testing.T) {
	now := time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)
	signedIn := now.Add(-48 * time.Hour)
	idle := 12 * time.Hour

	t.Run("idle timeout first", func(t *testing.T) {
		last := now.Add(-(11*time.Hour + 58*time.Minute))
		got := timing(signedIn, last, signedIn.Add(14*24*time.Hour), idle, now)
		if got.EndsBy != "idle" || !got.EndsAt.Equal(last.Add(idle)) {
			t.Fatalf("ends %s by %s", got.EndsAt, got.EndsBy)
		}
		if !got.Extendable || got.ExtendsTo == nil || !got.ExtendsTo.Equal(now.Add(idle)) {
			t.Fatalf("extend: %v to %v", got.Extendable, got.ExtendsTo)
		}
		if got.IdleTimeoutSeconds != 43200 {
			t.Fatalf("idle seconds %d", got.IdleTimeoutSeconds)
		}
	})

	t.Run("lifetime first", func(t *testing.T) {
		expires := now.Add(90 * time.Second)
		got := timing(signedIn, now, expires, idle, now)
		if got.EndsBy != "lifetime" || !got.EndsAt.Equal(expires) || got.Extendable {
			t.Fatalf("got %+v", got)
		}
		if got.ExtendsTo == nil || !got.ExtendsTo.Equal(expires) {
			t.Fatalf("extending can't pass the lifetime: %v", got.ExtendsTo)
		}
	})

	t.Run("extending gains less than the full idle period near the lifetime", func(t *testing.T) {
		last := now.Add(-11 * time.Hour)
		expires := now.Add(4 * time.Hour)
		got := timing(signedIn, last, expires, idle, now)
		// Idle ends in 1 h, the lifetime in 4 h: extending buys 3 hours.
		if got.EndsBy != "idle" || !got.Extendable || !got.ExtendsTo.Equal(expires) {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("no idle timeout", func(t *testing.T) {
		expires := now.Add(24 * time.Hour)
		got := timing(signedIn, now, expires, 0, now)
		if got.EndsBy != "lifetime" || got.IdleExpiresAt != nil || got.ExtendsTo != nil || got.Extendable {
			t.Fatalf("got %+v", got)
		}
	})
}
