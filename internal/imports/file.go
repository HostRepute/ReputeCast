package imports

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/emailaddr"
)

// MaxFileRows bounds an uploaded file ("Up to 500,000 rows").
const MaxFileRows = 500000

// FileListID is the pseudo-list of a file import.
const FileListID = "file"

// FileSource reads a CSV or XLSX upload. The first row holds the headers.
type FileSource struct {
	Name string
	Kind string // csv, xlsx
	Data []byte
	// SkipRows are data rows (line numbers) left out because their column
	// count doesn't match the header.
	SkipRows map[int]bool

	parsed bool
	header []string
	rows   []fileRow
	bad    []ImportBadRow
}

type fileRow struct {
	line   int
	values []string
}

// ImportBadRow is a row whose column count doesn't match the header.
type ImportBadRow struct {
	Line    int    `json:"line"`
	Columns int    `json:"columns"`
	Raw     string `json:"raw"`
}

// FileError explains a file that can't be imported as is.
type FileError struct {
	Message string
	// BadRows are rows with the wrong number of columns ("row 1,204 has 9
	// columns, the header has 8"); they can be skipped.
	BadRows []ImportBadRow
	Header  int
}

func (e *FileError) Error() string { return e.Message }

func (f *FileSource) parse() error {
	if f.parsed {
		return nil
	}
	f.parsed = true
	var table [][]string
	var lines []int
	switch f.Kind {
	case "xlsx":
		rows, err := readXLSX(f.Data, MaxFileRows+1)
		if err != nil {
			return &FileError{Message: fmt.Sprintf("%s can't be read: %v.", f.Name, err)}
		}
		for i, r := range rows {
			table = append(table, r)
			lines = append(lines, i+1)
		}
	default:
		data := bytes.TrimPrefix(f.Data, []byte("\xef\xbb\xbf"))
		if !utf8.Valid(data) {
			data = latin1ToUTF8(data)
		}
		r := csv.NewReader(bytes.NewReader(data))
		r.Comma = sniffDelimiter(data)
		r.FieldsPerRecord = -1
		r.LazyQuotes = true
		for {
			rec, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				var pe *csv.ParseError
				if errors.As(err, &pe) {
					return &FileError{Message: fmt.Sprintf("%s: row %d can't be read (%v). Fix the file and upload it again.", f.Name, pe.Line, pe.Err)}
				}
				return &FileError{Message: fmt.Sprintf("%s can't be read: %v.", f.Name, err)}
			}
			line, _ := r.FieldPos(0)
			table = append(table, rec)
			lines = append(lines, line)
			if len(table) > MaxFileRows+1 {
				return &FileError{Message: fmt.Sprintf("%s has more than %d rows. Split it into smaller files.", f.Name, MaxFileRows)}
			}
		}
	}
	// Drop empty rows and find the header.
	start := -1
	for i, r := range table {
		if !emptyRecord(r) {
			start = i
			break
		}
	}
	if start < 0 {
		return &FileError{Message: fmt.Sprintf("%s is empty.", f.Name)}
	}
	f.header = make([]string, len(table[start]))
	for i, h := range table[start] {
		f.header[i] = strings.TrimSpace(h)
		if f.header[i] == "" {
			f.header[i] = fmt.Sprintf("Column %d", i+1)
		}
	}
	for i := start + 1; i < len(table); i++ {
		rec := table[i]
		if emptyRecord(rec) {
			continue
		}
		if f.Kind != "xlsx" && len(rec) != len(f.header) {
			if !f.SkipRows[lines[i]] {
				f.bad = append(f.bad, ImportBadRow{Line: lines[i], Columns: len(rec), Raw: strings.Join(rec, ",")})
			}
			continue
		}
		for len(rec) < len(f.header) {
			rec = append(rec, "")
		}
		f.rows = append(f.rows, fileRow{line: lines[i], values: rec[:len(f.header)]})
	}
	if len(f.bad) > 0 {
		first := f.bad[0]
		msg := fmt.Sprintf("%s: row %s has %d columns, the header has %d", f.Name, thousands(first.Line), first.Columns, len(f.header))
		return &FileError{Message: msg, BadRows: f.bad, Header: len(f.header)}
	}
	return nil
}

func emptyRecord(r []string) bool {
	for _, v := range r {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

func sniffDelimiter(data []byte) rune {
	line := data
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		line = data[:i]
	}
	best, n := ',', bytes.Count(line, []byte(","))
	for _, d := range []rune{';', '\t', '|'} {
		if c := bytes.Count(line, []byte(string(d))); c > n {
			best, n = d, c
		}
	}
	return best
}

func latin1ToUTF8(b []byte) []byte {
	out := make([]rune, len(b))
	for i, c := range b {
		out[i] = rune(c)
	}
	return []byte(string(out))
}

func thousands(n int) string {
	s := fmt.Sprint(n)
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	return string(out)
}

// Header returns the column names.
func (f *FileSource) Header() ([]string, error) {
	if err := f.parse(); err != nil {
		return nil, err
	}
	return f.header, nil
}

// Rows returns the number of data rows.
func (f *FileSource) Rows() int { return len(f.rows) }

// Connect implements Source.
func (f *FileSource) Connect(context.Context) (ImportConnection, error) {
	if err := f.parse(); err != nil {
		return ImportConnection{}, err
	}
	return ImportConnection{Account: f.Name, CheckedAt: time.Now(), Lists: []ImportSourceList{{ID: FileListID, Name: f.Name, Subscribed: int64(len(f.rows)), Fields: len(f.header), TagExamples: []string{}}}}, nil
}

var (
	emailHeader  = regexp.MustCompile(`(?i)^(e-?mail|email ?address|e-mail address|mail)$`)
	statusHeader = regexp.MustCompile(`(?i)^(status|consent|subscription ?status|subscribed|opt-?in status|email status)$`)
	tagsHeader   = regexp.MustCompile(`(?i)^(tags?|labels?)$`)
)

// Fields implements Source: one field per column, typed from its values.
func (f *FileSource) Fields(context.Context, []string) ([]SourceField, error) {
	if err := f.parse(); err != nil {
		return nil, err
	}
	out := make([]SourceField, 0, len(f.header))
	email := f.emailColumn()
	for i, h := range f.header {
		typ := guessType(f.column(i, 50))
		switch {
		case i == email:
			typ = "email"
		case statusHeader.MatchString(h):
			typ = "status"
		case tagsHeader.MatchString(h):
			typ = "tags"
		}
		out = append(out, SourceField{Key: h, Label: h, Type: typ})
	}
	return out, nil
}

func (f *FileSource) column(i, n int) []string {
	var out []string
	for _, r := range f.rows {
		if v := strings.TrimSpace(r.values[i]); v != "" {
			out = append(out, v)
			if len(out) == n {
				break
			}
		}
	}
	return out
}

// emailColumn finds the email column by header, then by content.
func (f *FileSource) emailColumn() int {
	for i, h := range f.header {
		if emailHeader.MatchString(h) {
			return i
		}
	}
	best, bestN := 0, -1
	for i := range f.header {
		n := 0
		for _, v := range f.column(i, 20) {
			if emailaddr.Valid(v) {
				n++
			}
		}
		if n > bestN {
			best, bestN = i, n
		}
	}
	return best
}

var (
	numberRe = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	dateRe   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}([ T]\d{2}:\d{2}(:\d{2})?)?`)
)

// guessType looks at sample values: number, date, boolean or text.
func guessType(samples []string) string {
	if len(samples) == 0 {
		return "text"
	}
	num, date, boolean := 0, 0, 0
	for _, s := range samples {
		switch {
		case numberRe.MatchString(s):
			num++
		case dateRe.MatchString(s):
			date++
		}
		switch strings.ToLower(s) {
		case "true", "false", "yes", "no":
			boolean++
		}
	}
	switch n := len(samples); {
	case num == n:
		return "number"
	case date == n:
		return "date"
	case boolean == n:
		return "boolean"
	}
	return "text"
}

// NormalizeStatus maps a status column's value to a source status.
func NormalizeStatus(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "subscribed", "active", "yes", "true", "1", "confirmed", "opted in", "opt-in", "enabled":
		return StatusSubscribed
	case "pending", "unconfirmed", "not confirmed":
		return StatusPending
	case "bounced", "cleaned", "hard bounce", "invalid", "undeliverable":
		return StatusCleaned
	case "complained", "spam", "abuse", "junk":
		return StatusComplained
	case "transactional":
		return StatusTransactional
	}
	return StatusUnsubscribed
}

// Fetch implements Source. Email and status come from the columns that
// look like them; the mapping step can pick others.
func (f *FileSource) Fetch(_ context.Context, _ []string, emit func(Row) error) error {
	if err := f.parse(); err != nil {
		return err
	}
	email := f.emailColumn()
	status, tags := -1, -1
	for i, h := range f.header {
		if statusHeader.MatchString(h) {
			status = i
		}
		if tagsHeader.MatchString(h) {
			tags = i
		}
	}
	for _, r := range f.rows {
		row := Row{Email: strings.TrimSpace(r.values[email]), List: FileListID, Status: StatusSubscribed, Data: map[string]string{}, Tags: []string{}}
		for i, h := range f.header {
			row.Data[h] = strings.TrimSpace(r.values[i])
		}
		if status >= 0 {
			row.Status = NormalizeStatus(r.values[status])
		}
		if tags >= 0 {
			row.Tags = splitTags(r.values[tags])
		}
		if err := emit(row); err != nil {
			return err
		}
	}
	return nil
}

func splitTags(s string) []string {
	out := []string{}
	for _, t := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == '|' }) {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}
