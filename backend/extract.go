package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
)

const (
	maxPDFPages    = 500
	maxTextChars   = 8_000_000
	maxXMLBytes    = 50 << 20
	allowedExtList = ".pdf, .docx, .xlsx, .csv, .txt, .md"
)

var allowedExt = map[string]bool{".pdf": true, ".txt": true, ".md": true, ".docx": true, ".xlsx": true, ".csv": true}

// cleanFilename keeps only the base name and removes control characters.
func cleanFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	name = truncateRunes(strings.TrimSpace(name), 200)
	if name == "" || name == "." || name == "/" {
		return "document"
	}
	return name
}

// validateContent checks that the bytes match the claimed type.
func validateContent(ext string, data []byte) error {
	if len(data) == 0 {
		return errors.New("empty file")
	}
	switch ext {
	case ".pdf":
		if !bytes.HasPrefix(data, []byte("%PDF-")) {
			return errors.New("not a PDF")
		}
	case ".docx", ".xlsx":
		if !bytes.HasPrefix(data, []byte("PK\x03\x04")) {
			return errors.New("not an Office document")
		}
	case ".txt", ".md", ".csv":
		if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
			return errors.New("not UTF-8 text")
		}
	default:
		return fmt.Errorf("unsupported type %q (allowed: %s)", ext, allowedExtList)
	}
	return nil
}

// extractText never panics: third-party parsers can on malformed input.
func extractText(ext string, data []byte) (pages []Page, err error) {
	defer func() {
		if v := recover(); v != nil {
			pages, err = nil, fmt.Errorf("parser panic: %v", v)
		}
	}()
	switch ext {
	case ".pdf":
		pages, err = extractPDF(data)
	case ".docx":
		pages, err = extractDOCX(data)
	case ".txt", ".md":
		text := strings.ToValidUTF8(strings.ReplaceAll(string(data), "\x00", ""), "")
		pages = []Page{{Num: 1, Text: text}}
	default:
		return nil, fmt.Errorf("unsupported type %q", ext)
	}
	if err != nil {
		return nil, err
	}
	return capText(pages), nil
}

func capText(pages []Page) []Page {
	total := 0
	var out []Page
	for _, p := range pages {
		if strings.TrimSpace(p.Text) == "" {
			continue
		}
		if total+len(p.Text) > maxTextChars {
			p.Text = truncateRunes(p.Text, maxTextChars-total)
		}
		total += len(p.Text)
		out = append(out, p)
		if total >= maxTextChars {
			break
		}
	}
	return out
}

func extractPDF(data []byte) ([]Page, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	n := r.NumPage()
	if n > maxPDFPages {
		n = maxPDFPages
	}
	var pages []Page
	for i := 1; i <= n; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		pages = append(pages, Page{Num: i, Text: text})
	}
	return pages, nil
}

func extractDOCX(data []byte) ([]Page, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		if f.UncompressedSize64 > maxXMLBytes {
			return nil, errors.New("document too large")
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return []Page{{Num: 1, Text: docxText(io.LimitReader(rc, maxXMLBytes))}}, nil
	}
	return nil, errors.New("word/document.xml not found")
}

func docxText(r io.Reader) string {
	dec := xml.NewDecoder(r)
	var b strings.Builder
	inText := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				inText = true
			case "tab":
				b.WriteByte('\t')
			case "br":
				b.WriteByte('\n')
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inText = false
			case "p":
				b.WriteByte('\n')
			}
		case xml.CharData:
			if inText {
				b.Write(t)
			}
		}
	}
	return b.String()
}
