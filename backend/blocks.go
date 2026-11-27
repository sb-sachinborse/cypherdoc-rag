package main

import (
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type Stat struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Delta string `json:"delta,omitempty"`
}

type Table struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

type Series struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
}

type Chart struct {
	Kind   string   `json:"kind"` // bar, line or pie
	Title  string   `json:"title"`
	Labels []string `json:"labels"`
	Series []Series `json:"series"`
}

// Blocks is the optional structured part of an answer.
type Blocks struct {
	Stats []Stat `json:"stats,omitempty"`
	Table *Table `json:"table,omitempty"`
	Chart *Chart `json:"chart,omitempty"`
	// Derived is true when some figures were counted by the model rather than
	// copied from the documents. The UI shows a "verify" note.
	Derived bool `json:"derived,omitempty"`
}

func (b Blocks) empty() bool { return len(b.Stats) == 0 && b.Table == nil && b.Chart == nil }

var blocksSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"stats": map[string]any{"type": "array", "items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"label": map[string]any{"type": "string"},
				"value": map[string]any{"type": "string"},
				"delta": map[string]any{"type": "string"},
			},
			"required": []string{"label", "value"},
		}},
		"table": map[string]any{"type": "object", "properties": map[string]any{
			"columns": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"rows":    map[string]any{"type": "array", "items": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}},
		}},
		"chart": map[string]any{"type": "object", "properties": map[string]any{
			"kind":   map[string]any{"type": "string", "enum": []string{"bar", "line", "pie"}},
			"title":  map[string]any{"type": "string"},
			"labels": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"series": map[string]any{"type": "array", "items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":   map[string]any{"type": "string"},
					"values": map[string]any{"type": "array", "items": map[string]any{"type": "number"}},
				},
			}},
		}},
	},
}

var numRe = regexp.MustCompile(`\d[\d,]*(?:\.\d+)?`)

func normNum(tok string) (string, bool) {
	f, err := strconv.ParseFloat(strings.ReplaceAll(tok, ",", ""), 64)
	if err != nil {
		return "", false
	}
	return strconv.FormatFloat(f, 'f', -1, 64), true
}

func numberSet(text string) map[string]bool {
	set := map[string]bool{}
	for _, tok := range numRe.FindAllString(text, -1) {
		if n, ok := normNum(tok); ok {
			set[n] = true
		}
	}
	return set
}

// isCount reports whether a normalised number is a small whole number, the kind
// a model can legitimately count (items, topics, steps).
func isCount(n string) bool {
	f, err := strconv.ParseFloat(n, 64)
	return err == nil && f == math.Trunc(f) && f >= 0 && f <= 1000
}

// grounded reports whether every number in s appears in the source set.
// needNumber demands at least one number. In lenient mode (the user explicitly
// asked for a visual) small whole numbers are also accepted, and derived is set.
func grounded(s string, set map[string]bool, needNumber, lenient bool) (ok, derived bool) {
	toks := numRe.FindAllString(s, -1)
	if needNumber && len(toks) == 0 {
		return false, false
	}
	for _, tok := range toks {
		n, parsed := normNum(tok)
		if !parsed {
			return false, false
		}
		if set[n] {
			continue
		}
		if lenient && isCount(n) {
			derived = true
			continue
		}
		return false, false
	}
	return true, derived
}

// validateBlocks drops anything whose numbers are not literally present in the
// retrieved context (small models invent figures). With lenient=true, small
// whole-number counts are kept too and the result is flagged Derived.
func validateBlocks(in Blocks, source string, lenient bool) Blocks {
	set := numberSet(source)
	var out Blocks
	for _, st := range in.Stats {
		st.Label = truncateRunes(strings.TrimSpace(st.Label), 40)
		st.Value = truncateRunes(strings.TrimSpace(st.Value), 24)
		st.Delta = truncateRunes(strings.TrimSpace(st.Delta), 24)
		ok, d := grounded(st.Value, set, true, lenient)
		if st.Label == "" || !ok {
			continue
		}
		if dok, dd := grounded(st.Delta, set, false, false); !dok {
			st.Delta = ""
		} else if dd {
			st.Delta = ""
		}
		if d {
			out.Derived = true
		}
		out.Stats = append(out.Stats, st)
		if len(out.Stats) == 4 {
			break
		}
	}
	if t := in.Table; t != nil && len(t.Columns) >= 2 && len(t.Columns) <= 5 {
		nt := &Table{}
		for _, c := range t.Columns {
			nt.Columns = append(nt.Columns, truncateRunes(strings.TrimSpace(c), 30))
		}
		for _, row := range t.Rows {
			if len(row) != len(nt.Columns) || len(nt.Rows) == 8 {
				continue
			}
			rowOK, rowDerived := true, false
			clean := make([]string, len(row))
			for i, cell := range row {
				clean[i] = truncateRunes(strings.TrimSpace(cell), 40)
				ok, d := grounded(clean[i], set, false, lenient)
				if !ok {
					rowOK = false
				}
				if d {
					rowDerived = true
				}
			}
			if rowOK {
				nt.Rows = append(nt.Rows, clean)
				if rowDerived {
					out.Derived = true
				}
			}
		}
		if len(nt.Rows) > 0 {
			out.Table = nt
		}
	}
	if c := in.Chart; c != nil && len(c.Labels) >= 2 && len(c.Labels) <= 12 && len(c.Series) >= 1 && len(c.Series) <= 3 {
		ok, derived := true, false
		nc := &Chart{Kind: normKind(c.Kind, len(c.Series)), Title: truncateRunes(strings.TrimSpace(c.Title), 60)}
		for _, l := range c.Labels {
			nc.Labels = append(nc.Labels, truncateRunes(strings.TrimSpace(l), 20))
		}
		for _, sr := range c.Series {
			if len(sr.Values) != len(c.Labels) {
				ok = false
				break
			}
			for _, v := range sr.Values {
				key := strconv.FormatFloat(v, 'f', -1, 64)
				switch {
				case v < 0:
					ok = false
				case set[key]:
				case lenient && isCount(key):
					derived = true
				default:
					ok = false
				}
			}
			nc.Series = append(nc.Series, Series{Name: truncateRunes(strings.TrimSpace(sr.Name), 30), Values: sr.Values})
		}
		if ok {
			out.Chart = nc
			if derived {
				out.Derived = true
			}
		}
	}
	return out
}

func parseBlocks(raw string) (Blocks, bool) {
	var b Blocks
	if err := json.Unmarshal([]byte(raw), &b); err != nil {
		return Blocks{}, false
	}
	return b, true
}

func normKind(k string, seriesCount int) string {
	switch strings.ToLower(strings.TrimSpace(k)) {
	case "line":
		return "line"
	case "pie":
		if seriesCount == 1 {
			return "pie"
		}
	}
	return "bar"
}

var (
	pieRe  = regexp.MustCompile(`(?i)\b(pie|donut|doughnut|proportion|share)\b`)
	lineRe = regexp.MustCompile(`(?i)\b(line|trend|over time|timeline)\b`)
	barRe  = regexp.MustCompile(`(?i)\bbar\b`)
	mdSep  = regexp.MustCompile(`^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)*\|?\s*$`)
)

// kindFromQuestion picks a chart type from the wording; bar is the default.
func kindFromQuestion(q string, seriesCount int) string {
	switch {
	case barRe.MatchString(q):
		return "bar"
	case pieRe.MatchString(q) && seriesCount == 1:
		return "pie"
	case lineRe.MatchString(q):
		return "line"
	}
	return "bar"
}

func splitMDRow(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")
	parts := strings.Split(s, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// numericColumn reads one number per cell ("$420K" -> 420). Units are ignored on
// purpose so the values can be checked against the numbers in the source text.
func numericColumn(cells []string) ([]float64, bool) {
	out := make([]float64, 0, len(cells))
	for _, c := range cells {
		toks := numRe.FindAllString(c, -1)
		if len(toks) != 1 {
			return nil, false
		}
		n, ok := normNum(toks[0])
		if !ok {
			return nil, false
		}
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return nil, false
		}
		out = append(out, f)
	}
	return out, true
}

// chartFromMarkdownTable builds a chart from the first markdown table in the
// model's own answer. The model usually writes a correct table even when it
// cannot produce strict JSON, so this is far more reliable than a second call.
func chartFromMarkdownTable(answer, question string) *Chart {
	lines := strings.Split(strings.ReplaceAll(answer, "\r", ""), "\n")
	for i := 0; i+1 < len(lines); i++ {
		if !strings.Contains(lines[i], "|") || !mdSep.MatchString(lines[i+1]) {
			continue
		}
		head := splitMDRow(lines[i])
		var rows [][]string
		for j := i + 2; j < len(lines) && strings.Contains(lines[j], "|") && strings.TrimSpace(lines[j]) != ""; j++ {
			if r := splitMDRow(lines[j]); len(r) == len(head) {
				rows = append(rows, r)
			}
		}
		if len(head) < 2 || len(rows) < 2 {
			continue
		}
		ch := &Chart{}
		for _, r := range rows {
			ch.Labels = append(ch.Labels, r[0])
		}
		var names []string
		for c := 1; c < len(head) && len(ch.Series) < 3; c++ {
			col := make([]string, len(rows))
			for k, r := range rows {
				col[k] = r[c]
			}
			if vals, ok := numericColumn(col); ok {
				ch.Series = append(ch.Series, Series{Name: head[c], Values: vals})
				names = append(names, head[c])
			}
		}
		if len(ch.Series) == 0 {
			continue
		}
		ch.Title = strings.Join(names, ", ")
		ch.Kind = kindFromQuestion(question, len(ch.Series))
		return ch
	}
	return nil
}
