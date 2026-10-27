package mjml

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

// The composer compiles MJML in the browser (mjml-browser) and sends the
// HTML along with the MJML; checks, previews and readiness read the MJML,
// and sends use the HTML. A client of the API could store clean MJML with
// different HTML, or HTML without the markers that keep a conditional
// block from everyone it isn't for (CO-3). Verify compares what the HTML
// shows and does with what the server's own compile of the MJML shows and
// does; a mismatch is recompiled by the caller.

// Fingerprint is what an email shows and does, read from its HTML: its
// links, its images, its words and its conditional markers. The HTML of
// one MJML document from two compilers has one fingerprint even where the
// markup differs (gomjml and mjml-browser, checked by the parity tests).
type Fingerprint struct {
	// Links are the href of every <a>, in document order.
	Links []string
	// Images are the src of every <img>, in document order.
	Images []string
	// Words are the visible words of the body, in document order.
	Words []string
	// Markers are the conditional blocks' markers, in document order
	// ("if a1b2c3", "fi a1b2c3").
	Markers []string
}

// HTMLFingerprint reads the fingerprint of compiled HTML. The head, styles,
// scripts and comments (Outlook's conditional comments included) carry no
// words; markers are comments of their own.
func HTMLFingerprint(src string) Fingerprint {
	fp := Fingerprint{Links: []string{}, Images: []string{}, Words: []string{}, Markers: []string{}}
	z := html.NewTokenizer(strings.NewReader(src))
	skip := 0 // inside <head>, <style>, <script> or <title>
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			// The end of the HTML, or HTML too broken to read further.
			return fp
		case html.CommentToken:
			c := strings.TrimSpace(string(z.Text()))
			if rest, ok := strings.CutPrefix(c, "rc:"); ok && (strings.HasPrefix(rest, "if ") || strings.HasPrefix(rest, "fi ")) {
				fp.Markers = append(fp.Markers, rest)
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			tag := string(name)
			switch tag {
			case "head", "style", "script", "title":
				if tt == html.StartTagToken {
					skip++
				}
				continue
			}
			for hasAttr {
				var k, v []byte
				k, v, hasAttr = z.TagAttr()
				switch {
				case tag == "a" && string(k) == "href":
					fp.Links = append(fp.Links, strings.TrimSpace(string(v)))
				case tag == "img" && string(k) == "src":
					fp.Images = append(fp.Images, strings.TrimSpace(string(v)))
				}
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			switch string(name) {
			case "head", "style", "script", "title":
				if skip > 0 {
					skip--
				}
			}
		case html.TextToken:
			if skip > 0 {
				continue
			}
			fp.Words = append(fp.Words, words(string(z.Text()))...)
		}
	}
}

// words splits visible text into words. Invisible characters compilers add
// (the preview text's padding: U+034F, zero-width joiners and spaces) and
// non-breaking spaces are separators.
func words(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		switch r {
		case '\u034f', '\u200b', '\u200c', '\u200d', '\u2060', '\ufeff', '\u00a0', '\u00ad':
			return true
		}
		return unicode.IsSpace(r)
	})
}

// Mismatch says how two fingerprints differ ("links", "images", "text",
// "conditional markers"), or "" when they agree.
func (a Fingerprint) Mismatch(b Fingerprint) string {
	switch {
	case !slices.Equal(a.Markers, b.Markers):
		return "conditional markers"
	case !slices.Equal(a.Links, b.Links):
		return "links"
	case !slices.Equal(a.Images, b.Images):
		return "images"
	case !slices.Equal(a.Words, b.Words):
		return "text"
	}
	return ""
}

// Verdict is the outcome of checking a client's HTML against its MJML.
type Verdict struct {
	// OK is true when the HTML shows and does what the MJML says.
	OK bool
	// Mismatch names what differs ("links", "text", "conditional markers").
	Mismatch string
	// HTML is the server's own compile of the MJML, to store instead when
	// the HTML doesn't match; empty when the server couldn't compile it.
	HTML string
	// Errors are the server compile's errors, if any.
	Errors []MJMLProblem
}

// Verify checks HTML a client compiled from src against the server's own
// compile of src (gomjml, with the conditional markers PrepareCompile
// adds). When gomjml can't compile a document that parses, the markers the
// MJML needs are still checked; one that doesn't parse is refused.
func Verify(src, htmlSrc string) Verdict {
	res := Compile(src)
	got := HTMLFingerprint(htmlSrc)
	if len(res.Errors) > 0 {
		// A document that doesn't parse can't be vouched for: nothing says
		// which blocks are conditional.
		if _, err := Parse(src); err != nil {
			return Verdict{Mismatch: "mjml", Errors: res.Errors}
		}
		if want := ExpectedMarkers(src); !slices.Equal(want, got.Markers) {
			return Verdict{Mismatch: "conditional markers", Errors: res.Errors}
		}
		return Verdict{OK: true, Errors: res.Errors}
	}
	if m := HTMLFingerprint(res.HTML).Mismatch(got); m != "" {
		return Verdict{Mismatch: m, HTML: res.HTML}
	}
	return Verdict{OK: true, HTML: res.HTML}
}

// ExpectedMarkers lists the markers compiled HTML of src must carry, in
// order: one pair per conditional block PrepareCompile wraps.
func ExpectedMarkers(src string) []string {
	return HTMLFingerprint(markersOnly(PrepareCompile(src))).Markers
}

// markersOnly keeps the marker comments of a prepared document (they sit
// in <mj-raw> elements, in document order).
func markersOnly(prepared string) string {
	var b strings.Builder
	rest := prepared
	for {
		i := strings.Index(rest, "<!--rc:")
		if i < 0 {
			return b.String()
		}
		j := strings.Index(rest[i:], "-->")
		if j < 0 {
			return b.String()
		}
		b.WriteString(rest[i : i+j+3])
		rest = rest[i+j+3:]
	}
}

// String writes a fingerprint for test failures.
func (a Fingerprint) String() string {
	return fmt.Sprintf("links %q\nimages %q\nmarkers %q\nwords %q", a.Links, a.Images, a.Markers, a.Words)
}

// Checked returns the HTML to store for an MJML document and the HTML a
// client compiled from it: the client's when it shows and does what the
// MJML says, the server's own compile otherwise (mismatch names what
// differed: links, images, text or conditional markers). Without HTML the
// server compiles; without MJML (an HTML-only email) the HTML is the
// source. errs is set when nothing trustworthy can be stored: the MJML
// doesn't compile and there is no HTML, or the HTML lacks the markers.
func Checked(src, htmlSrc string) (out, mismatch string, errs []MJMLProblem) {
	switch {
	case strings.TrimSpace(src) == "":
		return htmlSrc, "", nil
	case htmlSrc == "":
		res := Compile(src)
		if len(res.Errors) > 0 {
			return "", "", res.Errors
		}
		return res.HTML, "", nil
	}
	v := Verify(src, htmlSrc)
	switch {
	case v.OK:
		return htmlSrc, "", nil
	case v.HTML != "":
		return v.HTML, v.Mismatch, nil
	}
	return "", v.Mismatch, v.Errors
}
