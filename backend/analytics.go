package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Analytics: the model only PLANS a query (which column, which aggregation).
// Go runs it over the real rows, so totals and averages are exact.

var analyticRe = regexp.MustCompile(`(?i)\b(sum|total|totals|average|avg|mean|count|how many|max|maximum|min|minimum|highest|lowest|top|bottom|best|worst|per|by|group|breakdown|distribution|trend|compare|comparison|chart|graph|plot|table|visuali[sz]e|growth|rank|share)\b`)

type Metric struct {
	Column string `json:"column"`
	Agg    string `json:"agg"`
}

type Filter struct {
	Column string `json:"column"`
	Op     string `json:"op"`
	Value  string `json:"value"`
}

type QuerySpec struct {
	DatasetID int64    `json:"dataset_id"`
	GroupBy   string   `json:"group_by"`
	Metrics   []Metric `json:"metrics"`
	Filters   []Filter `json:"filters"`
	SortDesc  bool     `json:"sort_desc"`
	Limit     int      `json:"limit"`
	Chart     string   `json:"chart"`
}

type datasetInfo struct {
	ID      int64
	Doc     string
	Sheet   string
	Columns []Column
	NRows   int
	Sample  [][]string
}

type QueryResult struct {
	Columns   []string
	Labels    []string
	Values    [][]float64 // [metric][row]
	RowsUsed  int
	Groups    int
	Truncated bool
}

var specSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"dataset_id": map[string]any{"type": "integer"},
		"group_by":   map[string]any{"type": "string"},
		"metrics": map[string]any{"type": "array", "items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"column": map[string]any{"type": "string"},
				"agg":    map[string]any{"type": "string", "enum": []string{"sum", "avg", "min", "max", "count"}},
			},
			"required": []string{"agg"},
		}},
		"filters": map[string]any{"type": "array", "items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"column": map[string]any{"type": "string"},
				"op":     map[string]any{"type": "string", "enum": []string{"eq", "neq", "contains", "gt", "gte", "lt", "lte"}},
				"value":  map[string]any{"type": "string"},
			},
			"required": []string{"column", "op", "value"},
		}},
		"sort_desc": map[string]any{"type": "boolean"},
		"limit":     map[string]any{"type": "integer"},
		"chart":     map[string]any{"type": "string", "enum": []string{"bar", "line", "pie", "none"}},
	},
	"required": []string{"dataset_id", "metrics", "chart"},
}

var (
	validAggs = map[string]bool{"sum": true, "avg": true, "min": true, "max": true, "count": true}
	validOps  = map[string]bool{"eq": true, "neq": true, "contains": true, "gt": true, "gte": true, "lt": true, "lte": true}
)

func findCol(cols []Column, name string) int {
	name = strings.ToLower(strings.TrimSpace(name))
	for i, c := range cols {
		if strings.ToLower(c.Name) == name {
			return i
		}
	}
	return -1
}

// validateSpec checks the model's plan against the real columns and rewrites
// names to their canonical spelling. Anything unknown is rejected.
func validateSpec(spec *QuerySpec, dss []datasetInfo) (*datasetInfo, error) {
	var ds *datasetInfo
	for i := range dss {
		if dss[i].ID == spec.DatasetID {
			ds = &dss[i]
			break
		}
	}
	if ds == nil && len(dss) == 1 {
		ds = &dss[0]
	}
	if ds == nil {
		return nil, errors.New("unknown dataset")
	}
	spec.DatasetID = ds.ID

	if g := strings.TrimSpace(spec.GroupBy); g != "" {
		i := findCol(ds.Columns, g)
		if i < 0 {
			return nil, fmt.Errorf("unknown group_by %q", g)
		}
		spec.GroupBy = ds.Columns[i].Name
	} else {
		spec.GroupBy = ""
	}
	if len(spec.Metrics) == 0 || len(spec.Metrics) > 3 {
		return nil, errors.New("1 to 3 metrics required")
	}
	for i := range spec.Metrics {
		m := &spec.Metrics[i]
		m.Agg = strings.ToLower(strings.TrimSpace(m.Agg))
		if !validAggs[m.Agg] {
			return nil, fmt.Errorf("bad aggregation %q", m.Agg)
		}
		if strings.TrimSpace(m.Column) == "" {
			if m.Agg != "count" {
				return nil, errors.New("metric column required")
			}
			m.Column = ""
			continue
		}
		j := findCol(ds.Columns, m.Column)
		if j < 0 {
			return nil, fmt.Errorf("unknown metric column %q", m.Column)
		}
		if m.Agg != "count" && !ds.Columns[j].Numeric {
			return nil, fmt.Errorf("column %q is not numeric", m.Column)
		}
		m.Column = ds.Columns[j].Name
	}
	if len(spec.Filters) > 4 {
		spec.Filters = spec.Filters[:4]
	}
	for i := range spec.Filters {
		f := &spec.Filters[i]
		f.Op = strings.ToLower(strings.TrimSpace(f.Op))
		if !validOps[f.Op] {
			return nil, fmt.Errorf("bad filter op %q", f.Op)
		}
		j := findCol(ds.Columns, f.Column)
		if j < 0 {
			return nil, fmt.Errorf("unknown filter column %q", f.Column)
		}
		f.Column = ds.Columns[j].Name
		if f.Op != "eq" && f.Op != "neq" && f.Op != "contains" {
			if _, ok := parseNumber(f.Value); !ok {
				return nil, errors.New("numeric filter value required")
			}
		}
	}
	if spec.Limit < 1 || spec.Limit > 50 {
		spec.Limit = 12
	}
	spec.Chart = strings.ToLower(strings.TrimSpace(spec.Chart))
	switch spec.Chart {
	case "bar", "line", "pie", "none":
	default:
		spec.Chart = "bar"
	}
	if spec.GroupBy == "" {
		spec.Chart = "none"
	}
	return ds, nil
}

func matchFilters(cols []Column, row []string, fs []Filter) bool {
	for _, f := range fs {
		i := findCol(cols, f.Column)
		if i < 0 || i >= len(row) {
			return false
		}
		cell := row[i]
		switch f.Op {
		case "eq":
			if !strings.EqualFold(strings.TrimSpace(cell), strings.TrimSpace(f.Value)) {
				return false
			}
		case "neq":
			if strings.EqualFold(strings.TrimSpace(cell), strings.TrimSpace(f.Value)) {
				return false
			}
		case "contains":
			if !strings.Contains(strings.ToLower(cell), strings.ToLower(f.Value)) {
				return false
			}
		default:
			a, ok1 := parseNumber(cell)
			b, ok2 := parseNumber(f.Value)
			if !ok1 || !ok2 {
				return false
			}
			pass := false
			switch f.Op {
			case "gt":
				pass = a > b
			case "gte":
				pass = a >= b
			case "lt":
				pass = a < b
			case "lte":
				pass = a <= b
			}
			if !pass {
				return false
			}
		}
	}
	return true
}

type accum struct {
	sum, min, max float64
	n             int
	seen          bool
}

func (a accum) value(agg string) float64 {
	switch agg {
	case "sum":
		return a.sum
	case "avg":
		if a.n == 0 {
			return 0
		}
		return a.sum / float64(a.n)
	case "min":
		return a.min
	case "max":
		return a.max
	default:
		return float64(a.n)
	}
}

func metricLabel(m Metric) string {
	names := map[string]string{"sum": "Sum of", "avg": "Average of", "min": "Min of", "max": "Max of", "count": "Count of"}
	if m.Column == "" {
		return "Count"
	}
	return names[m.Agg] + " " + m.Column
}

// runQuery executes a validated spec over the rows. Pure Go, exact arithmetic.
func runQuery(ds *datasetInfo, rows [][]string, spec QuerySpec) QueryResult {
	gIdx := -1
	if spec.GroupBy != "" {
		gIdx = findCol(ds.Columns, spec.GroupBy)
	}
	mIdx := make([]int, len(spec.Metrics))
	res := QueryResult{}
	for i, m := range spec.Metrics {
		mIdx[i] = -1
		if m.Column != "" {
			mIdx[i] = findCol(ds.Columns, m.Column)
		}
		res.Columns = append(res.Columns, metricLabel(m))
	}

	var order []string
	groups := map[string][]accum{}
	for _, r := range rows {
		if !matchFilters(ds.Columns, r, spec.Filters) {
			continue
		}
		res.RowsUsed++
		key := ""
		if gIdx >= 0 {
			key = r[gIdx]
			if key == "" {
				key = "(blank)"
			}
		}
		a, ok := groups[key]
		if !ok {
			a = make([]accum, len(spec.Metrics))
			groups[key] = a
			order = append(order, key)
		}
		for m, met := range spec.Metrics {
			if mIdx[m] < 0 {
				a[m].n++
				continue
			}
			cell := r[mIdx[m]]
			if met.Agg == "count" {
				if cell != "" {
					a[m].n++
				}
				continue
			}
			v, ok := parseNumber(cell)
			if !ok {
				continue
			}
			x := &a[m]
			x.sum += v
			x.n++
			if !x.seen || v < x.min {
				x.min = v
			}
			if !x.seen || v > x.max {
				x.max = v
			}
			x.seen = true
		}
	}
	if len(order) == 0 && gIdx < 0 {
		order = []string{""}
		groups[""] = make([]accum, len(spec.Metrics))
	}
	res.Groups = len(order)

	vals := make([][]float64, len(spec.Metrics))
	for m, met := range spec.Metrics {
		vals[m] = make([]float64, len(order))
		for g, key := range order {
			vals[m][g] = math.Round(groups[key][m].value(met.Agg)*100) / 100
		}
	}
	idx := make([]int, len(order))
	for i := range idx {
		idx[i] = i
	}
	if gIdx >= 0 && len(vals) > 0 && (spec.SortDesc || len(order) > spec.Limit) {
		sort.SliceStable(idx, func(a, b int) bool { return vals[0][idx[a]] > vals[0][idx[b]] })
	}
	if len(idx) > spec.Limit {
		idx = idx[:spec.Limit]
		res.Truncated = true
	}
	res.Values = make([][]float64, len(vals))
	for _, g := range idx {
		res.Labels = append(res.Labels, order[g])
	}
	for m := range vals {
		for _, g := range idx {
			res.Values[m] = append(res.Values[m], vals[m][g])
		}
	}
	return res
}

func fmtNum(f float64) string {
	s := strconv.FormatFloat(math.Round(f*100)/100, 'f', 2, 64)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], strings.TrimRight(s[i+1:], "0")
	}
	var b strings.Builder
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}
	out := b.String()
	if frac != "" {
		out += "." + frac
	}
	if neg && out != "0" {
		out = "-" + out
	}
	return out
}

func hasNegative(v []float64) bool {
	for _, x := range v {
		if x < 0 {
			return true
		}
	}
	return false
}

// resultBlocks turns an exact result into stat cards, a table and a chart.
func resultBlocks(res QueryResult, spec QuerySpec) Blocks {
	var b Blocks
	if spec.GroupBy == "" {
		for m := range spec.Metrics {
			if len(b.Stats) == 4 || len(res.Values[m]) == 0 {
				break
			}
			b.Stats = append(b.Stats, Stat{Label: truncateRunes(res.Columns[m], 40), Value: fmtNum(res.Values[m][0])})
		}
		return b
	}
	t := &Table{Columns: append([]string{spec.GroupBy}, res.Columns...)}
	for k, l := range res.Labels {
		row := []string{truncateRunes(l, 40)}
		for m := range res.Values {
			row = append(row, fmtNum(res.Values[m][k]))
		}
		t.Rows = append(t.Rows, row)
	}
	b.Table = t
	if spec.Chart != "none" && len(res.Labels) >= 2 {
		ch := &Chart{Title: truncateRunes(strings.Join(res.Columns, ", ")+" by "+spec.GroupBy, 60), Kind: spec.Chart}
		for k, l := range res.Labels {
			if k >= 12 {
				break
			}
			ch.Labels = append(ch.Labels, truncateRunes(l, 20))
		}
		negative := false
		for m, name := range res.Columns {
			v := res.Values[m]
			if len(v) > len(ch.Labels) {
				v = v[:len(ch.Labels)]
			}
			if hasNegative(v) {
				negative = true
			}
			ch.Series = append(ch.Series, Series{Name: truncateRunes(name, 30), Values: v})
		}
		if ch.Kind == "pie" && len(ch.Series) != 1 {
			ch.Kind = "bar"
		}
		if !negative {
			b.Chart = ch
		}
	}
	return b
}

func resultText(res QueryResult, spec QuerySpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Rows analysed: %d\n", res.RowsUsed)
	if spec.GroupBy == "" {
		for m, c := range res.Columns {
			fmt.Fprintf(&b, "%s = %s\n", c, fmtNum(res.Values[m][0]))
		}
		return b.String()
	}
	b.WriteString(spec.GroupBy + " | " + strings.Join(res.Columns, " | ") + "\n")
	for k, l := range res.Labels {
		cells := []string{l}
		for m := range res.Values {
			cells = append(cells, fmtNum(res.Values[m][k]))
		}
		b.WriteString(strings.Join(cells, " | ") + "\n")
	}
	if res.Truncated {
		fmt.Fprintf(&b, "(showing the top %d of %d groups)\n", len(res.Labels), res.Groups)
	}
	return b.String()
}

func narrativeMessages(question string, res QueryResult, spec QuerySpec) []Msg {
	return []Msg{
		{Role: "system", Content: "You are a data analyst. The results below were computed exactly from the user's spreadsheet. Answer the question in 2 to 4 short sentences using only these results. Never change or recalculate the numbers. Mention the highest and lowest values when useful. Do not add citation markers."},
		{Role: "user", Content: "Results:\n" + resultText(res, spec) + "\nQuestion: " + question},
	}
}

func describeDatasets(dss []datasetInfo) string {
	var b strings.Builder
	for _, d := range dss {
		cols := make([]string, len(d.Columns))
		for i, c := range d.Columns {
			kind := "text"
			if c.Numeric {
				kind = "number"
			}
			cols[i] = c.Name + " (" + kind + ")"
		}
		fmt.Fprintf(&b, "Dataset %d: file %q, sheet %q, %d rows\nColumns: %s\n", d.ID, d.Doc, d.Sheet, d.NRows, strings.Join(cols, ", "))
		for _, r := range d.Sample {
			b.WriteString("Sample row: " + strings.Join(r, " | ") + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (s *Server) listDatasets(ctx context.Context, userID int64, collection string) ([]datasetInfo, error) {
	rows, err := s.db.Query(ctx,
		`SELECT ds.id, d.name, ds.sheet_name, ds.cols, ds.n_rows, ds.sample
		 FROM datasets ds JOIN documents d ON d.id = ds.document_id
		 WHERE ds.user_id = $1 AND d.user_id = $1 AND d.status = 'indexed' AND ($2 = '' OR d.collection = $2)
		 ORDER BY d.created_at DESC, ds.id LIMIT 8`, userID, collection)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []datasetInfo
	for rows.Next() {
		var di datasetInfo
		var colsJSON, sampleJSON []byte
		if err := rows.Scan(&di.ID, &di.Doc, &di.Sheet, &colsJSON, &di.NRows, &sampleJSON); err != nil {
			return nil, err
		}
		if json.Unmarshal(colsJSON, &di.Columns) != nil || json.Unmarshal(sampleJSON, &di.Sample) != nil {
			continue
		}
		out = append(out, di)
	}
	return out, rows.Err()
}

func (s *Server) loadRows(ctx context.Context, userID, datasetID int64) ([][]string, error) {
	var raw []byte
	if err := s.db.QueryRow(ctx, `SELECT body FROM datasets WHERE id=$1 AND user_id=$2`, datasetID, userID).Scan(&raw); err != nil {
		return nil, err
	}
	var rows [][]string
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Server) planQuery(ctx context.Context, question string, dss []datasetInfo) (QuerySpec, *datasetInfo, bool) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	msgs := []Msg{
		{Role: "system", Content: "You translate a user's analytics question into a JSON query over the tables listed. Use ONLY the dataset ids and column names that are listed. metrics[].agg is one of sum, avg, min, max, count. Set group_by to a column name to break results down by category or time, or \"\" for one overall figure. filters narrow the rows (op is one of eq, neq, contains, gt, gte, lt, lte). chart is bar, line, pie or none: line for trends over time, pie for shares of a total, bar otherwise, none if no visual is wanted. Return JSON only."},
		{Role: "user", Content: describeDatasets(dss) + "Question: " + question},
	}
	raw, err := s.ol.ChatJSON(ctx, msgs, specSchema)
	if err != nil {
		s.log.Warn("analytics plan failed", "err", err)
		return QuerySpec{}, nil, false
	}
	var spec QuerySpec
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		s.log.Warn("analytics plan not JSON", "raw", truncateRunes(raw, 300))
		return QuerySpec{}, nil, false
	}
	ds, err := validateSpec(&spec, dss)
	if err != nil {
		s.log.Warn("analytics plan rejected", "err", err, "raw", truncateRunes(raw, 300))
		return QuerySpec{}, nil, false
	}
	return spec, ds, true
}

// tryAnalytics answers from spreadsheet data when the question is analytical.
// It returns true when it produced the answer (so normal document search is skipped).
func (s *Server) tryAnalytics(ctx context.Context, u User, question, collection string, send func(string, any) error) bool {
	if !analyticRe.MatchString(question) {
		return false
	}
	dss, err := s.listDatasets(ctx, u.ID, collection)
	if err != nil || len(dss) == 0 {
		return false
	}
	_ = send("status", map[string]string{"text": "Analyzing your data…"})
	spec, ds, ok := s.planQuery(ctx, question, dss)
	if !ok {
		return false
	}
	rows, err := s.loadRows(ctx, u.ID, ds.ID)
	if err != nil {
		s.log.Error("load dataset rows", "err", err)
		return false
	}
	res := runQuery(ds, rows, spec)
	if res.RowsUsed == 0 || len(res.Labels) == 0 && spec.GroupBy != "" {
		return false
	}
	if b := resultBlocks(res, spec); !b.empty() {
		_ = send("blocks", b)
	}
	_ = send("status", map[string]string{"text": "Writing the summary…"})
	err = s.ol.ChatStream(ctx, narrativeMessages(question, res, spec), func(tok string) error {
		return send("token", map[string]string{"text": tok})
	})
	if err != nil {
		if ctx.Err() == nil {
			s.log.Error("analytics narrative", "err", err)
			_ = send("error", map[string]string{"message": "The model failed to write the summary. The figures above are still correct."})
		}
		return true
	}
	_ = send("done", map[string]bool{"ok": true})
	return true
}
