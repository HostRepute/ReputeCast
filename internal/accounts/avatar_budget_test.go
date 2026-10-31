package accounts_test

import (
	"bytes"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// progressiveJPEG is a progressive JPEG of w×h grey pixels in three
// components: one DC scan, every block 0. The file is small, but decoding
// it takes about 15 bytes a pixel.
func progressiveJPEG(w, h int) []byte {
	seg := func(m byte, p ...byte) []byte {
		return append([]byte{0xFF, m, byte((len(p) + 2) >> 8), byte(len(p) + 2)}, p...)
	}
	out := append([]byte{0xFF, 0xD8}, seg(0xDB, append([]byte{0}, bytes.Repeat([]byte{1}, 64)...)...)...)
	out = append(out, seg(0xC2, 8, byte(h>>8), byte(h), byte(w>>8), byte(w), 3, 1, 0x11, 0, 2, 0x11, 0, 3, 0x11, 0)...)
	out = append(out, seg(0xC4, append([]byte{0, 1}, make([]byte, 16)...)...)...) // one code: every DC difference is 0
	out = append(out, seg(0xDA, 3, 1, 0, 2, 0, 3, 0, 0, 0, 0)...)
	blocks := (w + 7) / 8 * ((h + 7) / 8) * 3
	scan := make([]byte, (blocks+7)/8)
	if r := blocks % 8; r != 0 {
		scan[len(scan)-1] = 0xFF >> r
	}
	return append(append(out, scan...), 0xFF, 0xD9)
}

// A photo within 6,000 px a side is decoded only when decoding it and its
// square crop fit the memory an upload may take (review 2026-10): a
// progressive JPEG of 146 KB can take 475 MB.
func TestAvatarWithinDecodeBudget(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh"})
	c := env.Client()
	c.SignIn("linh@acme.io", "")

	p := c.Do("PUT", "/me/avatar", map[string]any{"data": progressiveJPEG(5000, 5000)}).MustStatus(t, 422).Problem(t)
	if p["detail"] != "The photo couldn't be read. Try another file." {
		t.Fatalf("over the budget: %v", p)
	}
	c.Do("PUT", "/me/avatar", map[string]any{"data": progressiveJPEG(600, 400)}).MustStatus(t, 200)
}
