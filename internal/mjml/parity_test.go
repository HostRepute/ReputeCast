package mjml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gomjml ↔ mjml-browser parity (CO-3). testdata/parity holds MJML
// documents (the starters as a workspace gets them, and edge cases) and,
// in browser/, the HTML the composer's mjml-browser makes of each
// (web/src/features/composer/editor/parity.test.ts writes them). The
// server's compile must show and do the same: same links, images, words
// and conditional markers, so Verify accepts every honest composer save.
func TestParityWithMJMLBrowser(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "parity", "*.mjml"))
	if err != nil || len(files) < 6 {
		t.Fatalf("parity fixtures: %v %v", files, err)
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".mjml")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			browser, err := os.ReadFile(filepath.Join("testdata", "parity", "browser", name+".html"))
			if err != nil {
				t.Fatalf("no mjml-browser output for %s (run the composer's parity test with UPDATE_PARITY=1): %v", name, err)
			}
			res := Compile(string(src))
			if len(res.Errors) > 0 {
				t.Fatalf("gomjml: %+v", res.Errors)
			}
			ours, theirs := HTMLFingerprint(res.HTML), HTMLFingerprint(string(browser))
			if m := ours.Mismatch(theirs); m != "" {
				t.Fatalf("%s differ\ngomjml:\n%s\n\nmjml-browser:\n%s", m, ours, theirs)
			}
			if v := Verify(string(src), string(browser)); !v.OK {
				t.Fatalf("the composer's own HTML is refused: %s", v.Mismatch)
			}
		})
	}
}

// Verify refuses HTML that doesn't show or do what its MJML says, and
// Checked stores the server's own compile instead.
func TestVerifyRefusesMismatches(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "parity", "conditional.mjml"))
	if err != nil {
		t.Fatal(err)
	}
	honest, err := os.ReadFile(filepath.Join("testdata", "parity", "browser", "conditional.html"))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct{ html, want string }{
		// A link swapped for another one.
		"links": {strings.Replace(string(honest), "https://acme.io/beta", "https://evil.example/login", 1), "links"},
		// The markers dropped: the beta corner would reach everyone.
		"markers": {strings.NewReplacer("<!--rc:if beta1-->", "", "<!--rc:fi beta1-->", "").Replace(string(honest)), "conditional markers"},
		// Words added that the MJML doesn't have.
		"text": {strings.Replace(string(honest), "Left column for everyone.", "Left column for everyone. Your account is locked.", 1), "text"},
		// An image added.
		"images": {strings.Replace(string(honest), "<body", `<body><img src="https://evil.example/pixel.gif">`, 1), "images"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			v := Verify(string(src), c.html)
			if v.OK || v.Mismatch != c.want {
				t.Fatalf("verdict %+v, want mismatch %q", v, c.want)
			}
			out, mismatch, errs := Checked(string(src), c.html)
			if errs != nil || mismatch != c.want || !strings.Contains(out, "<!--rc:if beta1-->") || strings.Contains(out, "evil.example") || strings.Contains(out, "locked") {
				t.Fatalf("checked: %q %v", mismatch, errs)
			}
		})
	}
	// The composer's own HTML is kept as it is.
	if out, mismatch, errs := Checked(string(src), string(honest)); out != string(honest) || mismatch != "" || errs != nil {
		t.Fatalf("honest HTML: %q %v", mismatch, errs)
	}
	// MJML that doesn't parse: no HTML can be vouched for, and none compiles.
	broken := "<mjml><mj-body><mj-section rc-visible='{\"id\":\"x1\",\"show\":\"match\"}'><mj-column><mj-text>Hi</mj-section></mj-body></mjml>"
	for _, h := range []string{"<p>Hi</p>", "<!--rc:if x1--><p>Hi</p><!--rc:fi x1-->", ""} {
		if _, _, errs := Checked(broken, h); errs == nil {
			t.Fatalf("broken MJML with HTML %q must be refused", h)
		}
	}
	// The markers a document needs, in order.
	if got := ExpectedMarkers(string(src)); strings.Join(got, ",") != "if beta1,fi beta1,if col2,fi col2" {
		t.Fatalf("expected markers: %v", got)
	}
	// HTML-only emails have no MJML to check against.
	if out, _, errs := Checked("", "<p>Raw</p>"); out != "<p>Raw</p>" || errs != nil {
		t.Fatal("HTML-only")
	}
}
