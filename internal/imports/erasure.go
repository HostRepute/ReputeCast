package imports

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
	"github.com/hostrepute/reputecast/internal/store"
)

// ReportKeepDays is how long a finished import keeps its report and
// skipped-row CSVs, which list every address the import read (review
// 2026-11 DP-8). Maintenance clears them after that; a done import drops its
// uploaded file as soon as it finishes.
const ReportKeepDays = 30

// ReportKeptUntil is when a finished import's reports are cleared.
func ReportKeptUntil(finished time.Time) time.Time {
	return finished.AddDate(0, 0, ReportKeepDays)
}

// Erasure is the contact's rows in imports that haven't finished (a done
// import drops its rows): the address, its fields and tags (review 2026-11
// DP-1). An import's kept file and reports are read too (DP-8): a report
// loses the rows that name an erased address, and an uploaded file that
// names one is removed, with a line in the import's log saying why.
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "imports",
		Tables: []erasure.Table{
			{Name: "import_rows", Action: "clear", Columns: []string{"email", "data", "tags", "engagement"}},
			{Name: "imports", Action: "clear", Columns: []string{"file_data", "report_csv", "skipped_csv"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			if err := erasure.Exec(ctx, tx, b,
				`UPDATE import_rows SET email = '', data = '{}', tags = '{}', engagement = '{}'
				 WHERE lower(email) = ANY ($2) AND import_id IN (SELECT id FROM imports WHERE workspace_id = $4)`); err != nil {
				return err
			}
			return stripErased(ctx, tx, b)
		},
	}
}

// stripErased removes the batch's addresses from every import of the
// workspace that still keeps a file or a report. Reports go after
// ReportKeepDays and done imports keep no file, so few imports are read.
func stripErased(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
	emails := b.Emails()
	if len(emails) == 0 {
		return nil
	}
	erased := make(map[string]bool, len(emails))
	for _, e := range emails {
		erased[e] = true
	}
	rows, err := tx.Query(ctx, `SELECT id FROM imports WHERE workspace_id = $1
		AND (file_data IS NOT NULL OR report_csv IS NOT NULL OR skipped_csv IS NOT NULL) ORDER BY id`, b.WorkspaceID)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return err
	}
	q := store.New(tx)
	for _, id := range ids {
		var source, status string
		var staged *time.Time
		var file, report, skipped []byte
		// One import at a time, locked, so a big file is in memory alone.
		if err := tx.QueryRow(ctx, `SELECT source, status, staged_at, file_data, report_csv, skipped_csv FROM imports WHERE id = $1 FOR UPDATE`, id).
			Scan(&source, &status, &staged, &file, &report, &skipped); err != nil {
			return err
		}
		newReport, reportCut := dropRows(report, erased)
		newSkipped, skippedCut := dropRows(skipped, erased)
		fileNames := len(file) > 0 && fileNames(source, file, erased)
		if !reportCut && !skippedCut && !fileNames {
			continue
		}
		if reportCut || skippedCut {
			if _, err := tx.Exec(ctx, `UPDATE imports SET report_csv = $2, skipped_csv = $3 WHERE id = $1`, id, newReport, newSkipped); err != nil {
				return err
			}
		}
		if !fileNames {
			continue
		}
		// The file goes. Staged rows hold what the import still needs (the
		// erased ones are blanked above); a draft not staged yet asks for
		// the file again.
		if staged == nil && status == "draft" {
			_, err = tx.Exec(ctx, `UPDATE imports SET file_data = NULL, file_name = '', connection = '{}', updated_at = now() WHERE id = $1`, id)
		} else {
			_, err = tx.Exec(ctx, `UPDATE imports SET file_data = NULL, updated_at = now() WHERE id = $1`, id)
		}
		if err != nil {
			return err
		}
		// The log line is timed by the erase, not the import's clock.
		at := b.At
		if at.IsZero() {
			at = time.Now()
		}
		if err := (&Service{Now: func() time.Time { return at }}).logMsg(ctx, q, id, "info", mLogFileErased, nil); err != nil {
			return err
		}
	}
	return nil
}

// dropRows removes the rows of a report CSV whose address (its second
// column, as the report and skipped-row files write it) is erased. It says
// whether any row went.
func dropRows(data []byte, erased map[string]bool) ([]byte, bool) {
	if len(data) == 0 {
		return data, false
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	var out bytes.Buffer
	w := csv.NewWriter(&out)
	cut := false
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			// Not a report this code wrote: keep nothing rather than an
			// address that can't be told apart.
			return nil, true
		}
		if len(rec) > 1 && erased[normalAddress(rec[1])] {
			cut = true
			continue
		}
		_ = w.Write(rec)
	}
	w.Flush()
	if !cut {
		return data, false
	}
	return out.Bytes(), true
}

// fileNames says whether an uploaded file has a cell holding an erased
// address. A file that can't be read is taken to name them.
func fileNames(kind string, data []byte, erased map[string]bool) bool {
	if kind == "xlsx" {
		rows, err := readXLSX(data, MaxFileRows+1)
		if err != nil {
			return true
		}
		for _, r := range rows {
			for _, v := range r {
				if erased[normalAddress(v)] {
					return true
				}
			}
		}
		return false
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		data = latin1ToUTF8(data)
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = sniffDelimiter(data)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			return false
		}
		if err != nil {
			return true
		}
		for _, v := range rec {
			if erased[normalAddress(v)] {
				return true
			}
		}
	}
}

// normalAddress is a cell as an address to compare: trimmed, lower-cased,
// without the quote formulaSafe puts before a leading "=+-@".
func normalAddress(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "'") {
		v = v[1:]
	}
	return strings.ToLower(v)
}
