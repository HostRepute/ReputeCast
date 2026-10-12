package imports

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"strconv"
	"strings"
	"time"
)

// readXLSX returns the cells of the first worksheet of an .xlsx file as
// text. It reads the Office Open XML parts directly (archive/zip and
// encoding/xml) instead of pulling in a spreadsheet library: shared and
// inline strings, numbers, booleans and dates (by number format) are
// enough for contact exports.
func readXLSX(data []byte, maxRows int) ([][]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, errors.New("this isn't an .xlsx file (it can't be opened as a spreadsheet)")
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	sheet, err := firstSheetPath(files)
	if err != nil {
		return nil, err
	}
	shared, err := sharedStrings(files["xl/sharedStrings.xml"])
	if err != nil {
		return nil, err
	}
	dateStyles, err := dateStyleIndexes(files["xl/styles.xml"])
	if err != nil {
		return nil, err
	}
	f, ok := files[sheet]
	if !ok {
		return nil, fmt.Errorf("the workbook points to %s, which is missing", sheet)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return sheetRows(io.LimitReader(rc, 512<<20), shared, dateStyles, maxRows)
}

func readZipXML(f *zip.File, v any) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	return xml.NewDecoder(io.LimitReader(rc, 64<<20)).Decode(v)
}

// firstSheetPath resolves the first sheet of the workbook through its
// relationships ("xl/worksheets/sheet1.xml" in most files).
func firstSheetPath(files map[string]*zip.File) (string, error) {
	wb, ok := files["xl/workbook.xml"]
	if !ok {
		return "", errors.New("this .xlsx file has no workbook")
	}
	var book struct {
		Sheets []struct {
			ID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := readZipXML(wb, &book); err != nil {
		return "", fmt.Errorf("the workbook can't be read: %w", err)
	}
	if len(book.Sheets) == 0 {
		return "", errors.New("the workbook has no sheets")
	}
	if relsFile, ok := files["xl/_rels/workbook.xml.rels"]; ok {
		var rels struct {
			Rels []struct {
				ID     string `xml:"Id,attr"`
				Target string `xml:"Target,attr"`
			} `xml:"Relationship"`
		}
		if err := readZipXML(relsFile, &rels); err == nil {
			for _, r := range rels.Rels {
				if r.ID == book.Sheets[0].ID {
					t := r.Target
					if strings.HasPrefix(t, "/") {
						return strings.TrimPrefix(t, "/"), nil
					}
					return path.Clean("xl/" + t), nil
				}
			}
		}
	}
	return "xl/worksheets/sheet1.xml", nil
}

func sharedStrings(f *zip.File) ([]string, error) {
	if f == nil {
		return nil, nil
	}
	var sst struct {
		Items []struct {
			T string `xml:"t"`
			R []struct {
				T string `xml:"t"`
			} `xml:"r"`
		} `xml:"si"`
	}
	if err := readZipXML(f, &sst); err != nil {
		return nil, fmt.Errorf("the shared strings can't be read: %w", err)
	}
	out := make([]string, len(sst.Items))
	for i, si := range sst.Items {
		if len(si.R) > 0 {
			var sb strings.Builder
			for _, r := range si.R {
				sb.WriteString(r.T)
			}
			out[i] = sb.String()
		} else {
			out[i] = si.T
		}
	}
	return out, nil
}

// dateStyleIndexes returns the cell style indexes whose number format is a date.
func dateStyleIndexes(f *zip.File) (map[int]bool, error) {
	out := map[int]bool{}
	if f == nil {
		return out, nil
	}
	var st struct {
		NumFmts []struct {
			ID   int    `xml:"numFmtId,attr"`
			Code string `xml:"formatCode,attr"`
		} `xml:"numFmts>numFmt"`
		Xfs []struct {
			NumFmtID int `xml:"numFmtId,attr"`
		} `xml:"cellXfs>xf"`
	}
	if err := readZipXML(f, &st); err != nil {
		return out, nil // styles are optional for reading values
	}
	custom := map[int]bool{}
	for _, nf := range st.NumFmts {
		custom[nf.ID] = isDateFormat(nf.Code)
	}
	for i, xf := range st.Xfs {
		id := xf.NumFmtID
		if (id >= 14 && id <= 22) || (id >= 45 && id <= 47) || custom[id] {
			out[i] = true
		}
	}
	return out, nil
}

func isDateFormat(code string) bool {
	c := strings.ToLower(code)
	// Drop quoted text and escaped characters before looking for d, m, y.
	var sb strings.Builder
	quoted := false
	for i := 0; i < len(c); i++ {
		switch {
		case c[i] == '"':
			quoted = !quoted
		case quoted:
		case c[i] == '\\':
			i++
		default:
			sb.WriteByte(c[i])
		}
	}
	s := sb.String()
	return strings.ContainsAny(s, "dy") || (strings.Contains(s, "m") && !strings.Contains(s, "0"))
}

// sheetRows streams <row>/<c> elements into a grid.
func sheetRows(r io.Reader, shared []string, dateStyles map[int]bool, maxRows int) ([][]string, error) {
	dec := xml.NewDecoder(r)
	var rows [][]string
	var cur []string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("the sheet can't be read: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			if ee, ok := tok.(xml.EndElement); ok && ee.Name.Local == "row" {
				rows = append(rows, cur)
				cur = nil
				if maxRows > 0 && len(rows) > maxRows {
					return nil, fmt.Errorf("the sheet has more than %d rows", maxRows)
				}
			}
			continue
		}
		switch se.Name.Local {
		case "row":
			cur = []string{}
			// Rows can be sparse: pad up to the declared row number.
			if n := attr(se, "r"); n != "" {
				if idx, err := strconv.Atoi(n); err == nil {
					for len(rows) < idx-1 {
						rows = append(rows, []string{})
					}
				}
			}
		case "c":
			var c struct {
				R  string `xml:"r,attr"`
				T  string `xml:"t,attr"`
				S  string `xml:"s,attr"`
				V  string `xml:"v"`
				Is struct {
					T string `xml:"t"`
					R []struct {
						T string `xml:"t"`
					} `xml:"r"`
				} `xml:"is"`
			}
			if err := dec.DecodeElement(&c, &se); err != nil {
				return nil, fmt.Errorf("a cell can't be read: %w", err)
			}
			col := columnIndex(c.R)
			if col < 0 {
				col = len(cur)
			}
			for len(cur) < col {
				cur = append(cur, "")
			}
			val := c.V
			switch c.T {
			case "s":
				i, err := strconv.Atoi(strings.TrimSpace(c.V))
				if err == nil && i >= 0 && i < len(shared) {
					val = shared[i]
				}
			case "inlineStr":
				if len(c.Is.R) > 0 {
					var sb strings.Builder
					for _, r := range c.Is.R {
						sb.WriteString(r.T)
					}
					val = sb.String()
				} else {
					val = c.Is.T
				}
			case "b":
				val = map[string]string{"1": "true", "0": "false"}[c.V]
			case "str", "e":
			default:
				style, _ := strconv.Atoi(c.S)
				if c.S != "" && dateStyles[style] {
					if f, err := strconv.ParseFloat(c.V, 64); err == nil {
						val = excelDate(f)
					}
				} else if f, err := strconv.ParseFloat(c.V, 64); err == nil && f == math.Trunc(f) && math.Abs(f) < 1e15 {
					val = strconv.FormatInt(int64(f), 10)
				}
			}
			cur = append(cur, val)
		}
	}
	return rows, nil
}

func attr(se xml.StartElement, name string) string {
	for _, a := range se.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// columnIndex turns "C7" into 2.
func columnIndex(ref string) int {
	n := 0
	i := 0
	for ; i < len(ref); i++ {
		ch := ref[i]
		if ch < 'A' || ch > 'Z' {
			break
		}
		n = n*26 + int(ch-'A'+1)
	}
	if i == 0 {
		return -1
	}
	return n - 1
}

// excelDate converts an Excel serial date (1900 system) to ISO.
func excelDate(serial float64) string {
	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	days := math.Floor(serial)
	secs := math.Round((serial - days) * 86400)
	t := base.AddDate(0, 0, int(days)).Add(time.Duration(secs) * time.Second)
	if secs == 0 {
		return t.Format("2006-01-02")
	}
	return t.Format("2006-01-02 15:04")
}
