package secrets

import (
	"encoding/base64"
	"path/filepath"
	"testing"
)

func mustKey(t *testing.T) []byte {
	t.Helper()
	k, err := base64.StdEncoding.DecodeString(GenerateKey())
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestSealOpenRoundTrip(t *testing.T) {
	b, err := New(mustKey(t))
	if err != nil {
		t.Fatal(err)
	}
	ct, err := b.SealString("AKIA-secret", "provider:1:credentials")
	if err != nil {
		t.Fatal(err)
	}
	got, err := b.OpenString(ct, "provider:1:credentials")
	if err != nil || got != "AKIA-secret" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := b.Open(ct, "provider:2:credentials"); err == nil {
		t.Fatal("ciphertext must not open under other associated data")
	}
}

func TestRotation(t *testing.T) {
	oldKey, newKey := mustKey(t), mustKey(t)
	oldBox, _ := New(oldKey)
	ct, _ := oldBox.SealString("v", "ad")
	rotated, _ := New(newKey, oldKey)
	if got, err := rotated.OpenString(ct, "ad"); err != nil || got != "v" {
		t.Fatalf("previous key must still open: %q %v", got, err)
	}
	if !rotated.NeedsRotation(ct) {
		t.Fatal("old ciphertext should need rotation")
	}
	fresh, _ := New(newKey)
	if _, err := fresh.Open(ct, "ad"); err == nil {
		t.Fatal("unknown key must fail")
	}
}

func TestKeyFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "secret.key")
	k1, created, err := LoadOrCreateKeyFile(p)
	if err != nil || !created {
		t.Fatalf("create: %v %v", created, err)
	}
	k2, created, err := LoadOrCreateKeyFile(p)
	if err != nil || created || k1 != k2 {
		t.Fatalf("reload: %v %v", created, err)
	}
}
