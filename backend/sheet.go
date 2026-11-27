package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	maxSheetRows = 50000
	maxSheetCols = 60
	maxSheets    = 10
	maxEmbedRows = 1500 // rows turned into searchable text; all rows stay available for analytics
)

type Column struct {
	Name    string `json:"name"`
	Numeric bool   `json:"numeric"`
}

type Sheet struct {
	Name    string
	Columns []Column
	Rows    [][]string
}

func isSheetExt(ext string) bool { return ext == ".csv" || ext == ".xlsx" }

// parseSpreadsheet reads CSV or XLSX into typed sheets. It never panics.
func parseSpreadsheet(ext string, data []byte) (sheets []Sheet, err error) {
	defer func() {
		if v := recover(); v != nil {
			sheets, err = nil, fmt.Errorf("spreadsheet parser panic: %v", v)
		}
	}()
	switch ext {
	case ".csv":
		rows, e := readCSV(data)
		if e != nil {
			return nil, e
		}
		if sh, ok := buildSheet("Sheet1", rows); ok {
			sheets = append(sheets, sh)
		}
	case ".xlsx":
		f, e := excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: 200 << 20, UnzipXMLSizeLimit: 64 << 20})
		if e != nil {
			return nil, e
		}
		defer f.Close()
		for i, name := range f.GetSheetList() {
			if i >= maxSheets {
				break
			}
			rows, e := f.GetRows(name)
			if e != nil {
				continue
			}
			if sh, ok := buildSheet(name, rows); ok {
				sheets = append(sheets, sh)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported spreadsheet type %q", ext)
	}
	if len(sheets) == 0 {
		return nil, errors.New("no data found")
	}
	return sheets, nil
}

func readCSV(data []byte) ([][]string, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = detectDelimiter(data)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	return r.ReadAll()
}

func detectDelimiter(data []byte) rune {
	line := data
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		line = data[:i]
	}
	best, bestN := ',', 0
	for _, d := range []rune{',', ';', '\t'} {
		if n := bytes.Count(line, []byte(string(d))); n > bestN {
			best, bestN = d, n
		}
	}
	return best
}

func rowEmpty(r []string) bool {
	for _, c := range r {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// buildSheet turns raw rows into a sheet: first non-empty row is the header.
func buildSheet(name string, raw [][]string) (Sheet, bool) {
	start := 0
	for start < len(raw) && rowEmpty(raw[start]) {
		start++
	}
	if start >= len(raw) {
		return Sheet{}, false
	}
	header := raw[start]
	n := len(header)
	for n > 0 && strings.TrimSpace(header[n-1]) == "" {
		n--
	}
	if n > maxSheetCols {
		n = maxSheetCols
	}
	if n == 0 {
		return Sheet{}, false
	}
	seen := map[string]int{}
	cols := make([]Column, n)
	for i := 0; i < n; i++ {
		nm := truncateRunes(strings.TrimSpace(header[i]), 60)
		if nm == "" {
			nm = "Column " + strconv.Itoa(i+1)
		}
		key := strings.ToLower(nm)
		seen[key]++
		if seen[key] > 1 {
			nm = fmt.Sprintf("%s (%d)", nm, seen[key])
		}
		cols[i].Name = nm
	}
	var rows [][]string
	for _, r := range raw[start+1:] {
		if rowEmpty(r) {
			continue
		}
		row := make([]string, n)
		for i := 0; i < n && i < len(r); i++ {
			row[i] = strings.TrimSpace(r[i])
		}
		rows = append(rows, row)
		if len(rows) >= maxSheetRows {
			break
		}
	}
	if len(rows) == 0 {
		return Sheet{}, false
	}
	for i := range cols {
		cols[i].Numeric = columnIsNumeric(rows, i)
	}
	return Sheet{Name: truncateRunes(name, 60), Columns: cols, Rows: rows}, true
}

func columnIsNumeric(rows [][]string, c int) bool {
	filled, nums := 0, 0
	for _, r := range rows {
		if r[c] == "" {
			continue
		}
		filled++
		if _, ok := parseNumber(r[c]); ok {
			nums++
		}
	}
	return filled > 0 && nums*100 >= filled*80
}

var numCellRe = regexp.MustCompile(`^[-+]?\d[\d,]*(?:\.\d+)?[kKmMbB]?%?$`)

// parseNumber understands $1,200  (300)  12%  1.5K  2.4M. Percent signs are
// dropped (12% -> 12), K/M/B scale the value.
func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	neg := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		neg = true
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	s = strings.TrimLeft(s, "$€£₹ ")
	if !numCellRe.MatchString(s) {
		return 0, false
	}
	s = strings.TrimSuffix(s, "%")
	mult := 1.0
	switch s[len(s)-1] {
	case 'k', 'K':
		mult, s = 1e3, s[:len(s)-1]
	case 'm', 'M':
		mult, s = 1e6, s[:len(s)-1]
	case 'b', 'B':
		mult, s = 1e9, s[:len(s)-1]
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	if err != nil {
		return 0, false
	}
	f *= mult
	if neg {
		f = -f
	}
	return f, true
}

// sheetPages renders sheets as searchable text: a summary plus "Row n: Col: value" lines.
func sheetPages(sheets []Sheet) []Page {
	var pages []Page
	for i, sh := range sheets {
		var b strings.Builder
		names := make([]string, len(sh.Columns))
		for j, c := range sh.Columns {
			kind := "text"
			if c.Numeric {
				kind = "number"
			}
			names[j] = c.Name + " (" + kind + ")"
		}
		fmt.Fprintf(&b, "Spreadsheet sheet %q has %d rows and these columns: %s.\n", sh.Name, len(sh.Rows), strings.Join(names, ", "))
		for r, row := range sh.Rows {
			if r >= maxEmbedRows {
				break
			}
			b.WriteString("Row " + strconv.Itoa(r+1) + ": ")
			first := true
			for j, v := range row {
				if v == "" {
					continue
				}
				if !first {
					b.WriteString("; ")
				}
				first = false
				b.WriteString(sh.Columns[j].Name + ": " + v)
			}
			b.WriteString(".\n")
		}
		pages = append(pages, Page{Num: i + 1, Text: b.String()})
	}
	return capText(pages)
}
