package main

import (
	"encoding/json"
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
	Title  string   `json:"title"`
	Labels []string `json:"labels"`
	Series []Series `json:"series"`
}

// Blocks is the optional structured part of an answer.
type Blocks struct {
	Stats []Stat `json:"stats,omitempty"`
	Table *Table `json:"table,omitempty"`
	Chart *Chart `json:"chart,omitempty"`
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

// grounded reports whether every number in s appears in the source set.
// needNumber demands at least one number.
func grounded(s string, set map[string]bool, needNumber bool) bool {
	toks := numRe.FindAllString(s, -1)
	if needNumber && len(toks) == 0 {
		return false
	}
	for _, tok := range toks {
		n, ok := normNum(tok)
		if !ok || !set[n] {
			return false
		}
	}
	return true
}

// validateBlocks drops anything whose numbers are not literally present in the
// retrieved context. Small models invent figures; this keeps the UI honest.
func validateBlocks(in Blocks, source string) Blocks {
	set := numberSet(source)
	var out Blocks
	for _, st := range in.Stats {
		st.Label = truncateRunes(strings.TrimSpace(st.Label), 40)
		st.Value = truncateRunes(strings.TrimSpace(st.Value), 24)
		st.Delta = truncateRunes(strings.TrimSpace(st.Delta), 24)
		if st.Label == "" || !grounded(st.Value, set, true) {
			continue
		}
		if !grounded(st.Delta, set, false) {
			st.Delta = ""
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
			ok := true
			clean := make([]string, len(row))
			for i, cell := range row {
				clean[i] = truncateRunes(strings.TrimSpace(cell), 40)
				if !grounded(clean[i], set, false) {
					ok = false
				}
			}
			if ok {
				nt.Rows = append(nt.Rows, clean)
			}
		}
		if len(nt.Rows) > 0 {
			out.Table = nt
		}
	}
	if c := in.Chart; c != nil && len(c.Labels) >= 2 && len(c.Labels) <= 12 && len(c.Series) >= 1 && len(c.Series) <= 3 {
		ok := true
		nc := &Chart{Title: truncateRunes(strings.TrimSpace(c.Title), 60)}
		for _, l := range c.Labels {
			nc.Labels = append(nc.Labels, truncateRunes(strings.TrimSpace(l), 20))
		}
		for _, s := range c.Series {
			if len(s.Values) != len(c.Labels) {
				ok = false
				break
			}
			for _, v := range s.Values {
				if v < 0 || !set[strconv.FormatFloat(v, 'f', -1, 64)] {
					ok = false
				}
			}
			nc.Series = append(nc.Series, Series{Name: truncateRunes(strings.TrimSpace(s.Name), 30), Values: s.Values})
		}
		if ok {
			out.Chart = nc
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
