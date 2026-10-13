package webhookauth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

// source is a provider API holding the current signing secret.
type source struct {
	secret string
	down   bool
	reads  int
}

func (s *source) fetch(context.Context) (string, error) {
	s.reads++
	if s.down {
		return "", errors.New("provider unavailable")
	}
	return s.secret, nil
}

func signedWith(secret string) func(string) error {
	return func(m string) error {
		if m != secret {
			return providers.ErrBadSignature
		}
		return nil
	}
}

func TestMaterialReadsOnceAndFollowsRotation(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	c := &webhookauth.Material{Now: func() time.Time { return clock }}
	src := &source{secret: "one"}
	key := webhookauth.Key("https://api.example", "key", "/hooks/x/tok")
	for range 3 {
		if err := c.Verify(ctx, key, src.fetch, signedWith("one")); err != nil {
			t.Fatal(err)
		}
	}
	if src.reads != 1 {
		t.Fatalf("the secret must be read once, then cached; read %d times", src.reads)
	}
	// Rotated: the first request signed with the new secret reads it again.
	src.secret = "two"
	clock = clock.Add(2 * time.Minute)
	if err := c.Verify(ctx, key, src.fetch, signedWith("two")); err != nil {
		t.Fatalf("a rotated secret must be picked up: %v", err)
	}
	// Forged requests don't make every request call the provider.
	for range 5 {
		if err := c.Verify(ctx, key, src.fetch, signedWith("forged")); !errors.Is(err, providers.ErrBadSignature) {
			t.Fatalf("a forged request must fail: %v", err)
		}
	}
	if src.reads != 2 {
		t.Fatalf("at most one read a minute; read %d times", src.reads)
	}
}

func TestMaterialExpiresRotatedOutSecrets(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	c := &webhookauth.Material{Now: func() time.Time { return clock }}
	src := &source{secret: "leaked"}
	key := webhookauth.Key("k")
	if err := c.Verify(ctx, key, src.fetch, signedWith("leaked")); err != nil {
		t.Fatal(err)
	}
	src.secret = "replacement"
	clock = clock.Add(webhookauth.MaxAge + time.Second)
	if err := c.Verify(ctx, key, src.fetch, signedWith("leaked")); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("a secret the provider replaced must stop verifying after MaxAge: %v", err)
	}
}

func TestMaterialKeepsOldSecretWhileProviderIsDown(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	c := &webhookauth.Material{Now: func() time.Time { return clock }}
	src := &source{secret: "one"}
	key := webhookauth.Key("k")
	if err := c.Verify(ctx, key, src.fetch, signedWith("one")); err != nil {
		t.Fatal(err)
	}
	src.down = true
	clock = clock.Add(webhookauth.MaxAge + time.Second)
	if err := c.Verify(ctx, key, src.fetch, signedWith("one")); err != nil {
		t.Fatalf("an outage must not reject correctly signed requests: %v", err)
	}
}

func TestMaterialReadFailures(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	c := &webhookauth.Material{Now: func() time.Time { return clock }}
	src := &source{down: true}
	key := webhookauth.Key("k")
	for range 4 {
		if err := c.Verify(ctx, key, src.fetch, signedWith("one")); !errors.Is(err, providers.ErrBadSignature) {
			t.Fatalf("a request that can't be verified is never trusted: %v", err)
		}
	}
	if src.reads != 1 {
		t.Fatalf("a failed read must not be repeated on every request; read %d times", src.reads)
	}
	src.down, src.secret = false, "one"
	clock = clock.Add(webhookauth.Refetch)
	if err := c.Verify(ctx, key, src.fetch, signedWith("one")); err != nil {
		t.Fatalf("the read must be retried after a minute: %v", err)
	}
	// Material handed over at registration saves the read.
	c.Set(webhookauth.Key("seeded"), "two")
	if err := c.Verify(ctx, webhookauth.Key("seeded"), src.fetch, signedWith("two")); err != nil || src.reads != 2 {
		t.Fatalf("seeded material must be used: %v, %d reads", err, src.reads)
	}
}
