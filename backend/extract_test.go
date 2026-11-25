package main

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func makeDocx(t *testing.T, body string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte(body))
	zw.Close()
	return buf.Bytes()
}

func TestExtractDOCX(t *testing.T) {
	xml := `<?xml version="1.0"?><w:document xmlns:w="x"><w:body>` +
		`<w:p><w:r><w:t>Leave policy</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>24 days per year</w:t></w:r></w:p></w:body></w:document>`
	pages, err := extractText(".docx", makeDocx(t, xml))
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 || !strings.Contains(pages[0].Text, "Leave policy") || !strings.Contains(pages[0].Text, "24 days per year") {
		t.Fatalf("unexpected: %+v", pages)
	}
}

func TestExtractTextAndMarkdown(t *testing.T) {
	pages, err := extractText(".md", []byte("# Title\n\nSome text"))
	if err != nil || len(pages) != 1 || !strings.Contains(pages[0].Text, "Some text") {
		t.Fatalf("unexpected: %+v %v", pages, err)
	}
}

func TestExtractGarbageDoesNotPanic(t *testing.T) {
	// Malformed input must be handled (error or empty result), never crash the server.
	for _, ext := range []string{".pdf", ".docx"} {
		_, _ = extractText(ext, []byte("%PDF-1.4 garbage PK\x03\x04 garbage"))
	}
}

func TestValidateContent(t *testing.T) {
	cases := []struct {
		ext  string
		data []byte
		ok   bool
	}{
		{".pdf", []byte("%PDF-1.7 ..."), true},
		{".pdf", []byte("MZ executable"), false},
		{".docx", []byte("PK\x03\x04rest"), true},
		{".docx", []byte("%PDF-"), false},
		{".txt", []byte("plain text"), true},
		{".txt", []byte("bin\x00ary"), false},
		{".md", []byte{0xff, 0xfe, 0xfd}, false},
		{".exe", []byte("x"), false},
		{".txt", nil, false},
	}
	for _, c := range cases {
		if err := validateContent(c.ext, c.data); (err == nil) != c.ok {
			t.Errorf("validateContent(%q, %q): err=%v want ok=%v", c.ext, c.data, err, c.ok)
		}
	}
}

func TestCleanFilename(t *testing.T) {
	cases := map[string]string{
		"../../etc/passwd":     "passwd",
		`C:\docs\report.pdf`:   "report.pdf",
		"na\x00me\n.txt":       "name.txt",
		"":                     "document",
		strings.Repeat("a", 300) + ".pdf": strings.Repeat("a", 200),
	}
	for in, want := range cases {
		if got := cleanFilename(in); got != want {
			t.Errorf("cleanFilename(%q) = %q want %q", in, got, want)
		}
	}
}

func TestSanitizeCollection(t *testing.T) {
	for in, want := range map[string]string{"HR Policies": "HR Policies", "  Q3_Assets ": "Q3_Assets", "<script>": "", "": "", "a/b": ""} {
		if got := sanitizeCollection(in); got != want {
			t.Errorf("sanitizeCollection(%q) = %q want %q", in, got, want)
		}
	}
}
