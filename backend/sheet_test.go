package main

import (
	"strings"
	"testing"
)

func TestParseNumber(t *testing.T) {
	good := map[string]float64{"42": 42, "$1,200": 1200, "(300)": -300, "12%": 12, "1.5K": 1500, "2.4M": 2.4e6, "-7.5": -7.5, "€ 10": 10}
	for in, want := range good {
		got, ok := parseNumber(in)
		if !ok || got != want {
			t.Errorf("parseNumber(%q) = %v,%v want %v", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "abc", "2026-01", "12 apples", "1.2.3", "N/A"} {
		if _, ok := parseNumber(in); ok {
			t.Errorf("parseNumber(%q) should fail", in)
		}
	}
}

func TestBuildSheetTypesAndHeaders(t *testing.T) {
	raw := [][]string{
		{},
		{"Region", "Revenue", "Region", ""},
		{"North", "$100", "x", ""},
		{"South", "200", "y", ""},
		{"", "", "", ""},
		{"East", "300", "z"},
	}
	sh, ok := buildSheet("Q1", raw)
	if !ok {
		t.Fatal("sheet should build")
	}
	if len(sh.Columns) != 3 || sh.Columns[2].Name != "Region (2)" {
		t.Fatalf("duplicate header not renamed: %+v", sh.Columns)
	}
	if len(sh.Rows) != 3 {
		t.Fatalf("blank rows must be skipped, got %d rows", len(sh.Rows))
	}
	if sh.Columns[0].Numeric {
		t.Error("Region must be text")
	}
	if !sh.Columns[1].Numeric {
		t.Error("Revenue must be numeric")
	}
}

func TestBuildSheetNumericThreshold(t *testing.T) {
	raw := [][]string{{"Amount"}, {"1"}, {"2"}, {"3"}, {"4"}, {"oops"}}
	sh, _ := buildSheet("S", raw)
	if !sh.Columns[0].Numeric {
		t.Error("80 percent numeric cells should make the column numeric")
	}
	raw = [][]string{{"Amount"}, {"1"}, {"x"}, {"y"}}
	sh, _ = buildSheet("S", raw)
	if sh.Columns[0].Numeric {
		t.Error("mostly text must stay text")
	}
}

func TestParseCSVDelimiters(t *testing.T) {
	for name, data := range map[string]string{
		"comma":     "Name,Score\nAnn,10\nBob,20\n",
		"semicolon": "Name;Score\nAnn;10\nBob;20\n",
		"tab":       "Name\tScore\nAnn\t10\nBob\t20\n",
		"bom":       "\xef\xbb\xbfName,Score\nAnn,10\nBob,20\n",
	} {
		sheets, err := parseSpreadsheet(".csv", []byte(data))
		if err != nil || len(sheets) != 1 || len(sheets[0].Rows) != 2 || sheets[0].Columns[0].Name != "Name" || !sheets[0].Columns[1].Numeric {
			t.Errorf("%s: unexpected result %+v err=%v", name, sheets, err)
		}
	}
}

func TestParseSpreadsheetGarbageDoesNotPanic(t *testing.T) {
	_, _ = parseSpreadsheet(".xlsx", []byte("PK\x03\x04 not really a workbook"))
	if _, err := parseSpreadsheet(".csv", []byte("")); err == nil {
		t.Error("empty CSV should be an error")
	}
}

func TestSheetPagesMentionsColumnsAndRows(t *testing.T) {
	sh, _ := buildSheet("Sales", [][]string{{"Region", "Revenue"}, {"North", "100"}})
	pages := sheetPages([]Sheet{sh})
	if len(pages) != 1 || !strings.Contains(pages[0].Text, "Revenue (number)") || !strings.Contains(pages[0].Text, "Row 1: Region: North; Revenue: 100") {
		t.Fatalf("unexpected text: %q", pages[0].Text)
	}
}
