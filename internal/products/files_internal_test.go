package products

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

// sniff writes data to a sniffer in writes of step bytes, as a stream of
// chunks arrives, and reads its kind with the file at hand only through
// io.ReaderAt (no path).
func sniff(data []byte, step int) Detected {
	var sn sniffer
	for len(data[int(sn.n):]) > 0 {
		rest := data[int(sn.n):]
		_, _ = sn.Write(rest[:min(step, len(rest))])
	}
	return sn.detected(bytes.NewReader(data), int64(len(data)))
}

func zipOf(t *testing.T, files ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for i := 0; i < len(files); i += 2 {
		f, err := w.Create(files[i])
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.Write([]byte(files[i+1]))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// A PDF's pages are counted as its bytes stream past, a page object cut
// by the end of a write included; an archive is told from its directory,
// read at any offset; past 64 MB nothing more is counted.
func TestSniffer(t *testing.T) {
	pdf := []byte("%PDF-1.7\n" + strings.Repeat("x", 5000) + "1 0 obj << /Type /Page >> endobj\n" + strings.Repeat("y", 3000) +
		"2 0 obj << /Type\n/Page /Parent 3 0 R >>\n3 0 obj << /Type /Pages >>\n" + strings.Repeat("z", 9000) + "4 0 obj <</Type/Page>>\n")
	for _, step := range []int{1, 7, 1000, 4096, 1 << 20} {
		if d := sniff(pdf, step); d.Kind != "pdf" || d.Pages != 3 || d.ContentType != "application/pdf" {
			t.Errorf("a PDF in writes of %d: %+v", step, d)
		}
	}
	big := make([]byte, pdfScanBytes+100)
	copy(big, "%PDF-1.4\n<< /Type /Page >>")
	copy(big[pdfScanBytes+10:], "<< /Type /Page >>")
	if d := sniff(big, 1<<20); d.Pages != 1 {
		t.Errorf("past 64 MB: %d pages", d.Pages)
	}
	for _, c := range []struct {
		data []byte
		kind string
	}{
		{zipOf(t, "a.txt", "hello"), "zip"},
		{zipOf(t, "[Content_Types].xml", "<x/>", "word/document.xml", "<w/>"), "docx"},
		{zipOf(t, "[Content_Types].xml", "<x/>", "xl/workbook.xml", "<w/>"), "xlsx"},
		{zipOf(t, "mimetype", "application/epub+zip", "OEBPS/a.xhtml", "<p/>"), "epub"},
		{[]byte("MZ\x90\x00 a Windows program"), KindProgram},
		{[]byte("email,name\nreader@example.com,Reader\n"), "text"},
		{[]byte{0, 1, 2, 3}, ""},
	} {
		if d := sniff(c.data, 3); d.Kind != c.kind {
			t.Errorf("%q…: %q, want %q", c.data[:min(len(c.data), 8)], d.Kind, c.kind)
		}
	}
}

// The page counter keeps a short tail between writes, not the file.
func TestPageCounterKeepsATail(t *testing.T) {
	var c pageCounter
	chunk := bytes.Repeat([]byte("<< /Type /Page >> filler "), 2000)
	for range 50 {
		c.write(chunk)
	}
	if len(c.buf) > pageTail {
		t.Errorf("kept %d bytes", len(c.buf))
	}
	if got := c.count(); got != 100_000 {
		t.Errorf("counted %d pages", got)
	}
}
