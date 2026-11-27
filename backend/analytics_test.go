package main

import (
	"reflect"
	"testing"
)

func testDS() (*datasetInfo, [][]string) {
	ds := &datasetInfo{ID: 7, Doc: "sales.xlsx", Sheet: "Q1", NRows: 4,
		Columns: []Column{{"Region", false}, {"Revenue", true}, {"Units", true}}}
	rows := [][]string{
		{"North", "100", "1"},
		{"South", "200", "2"},
		{"North", "50", "3"},
		{"East", "$1,000", "4"},
	}
	return ds, rows
}

func TestRunQuerySumGroupBy(t *testing.T) {
	ds, rows := testDS()
	res := runQuery(ds, rows, QuerySpec{GroupBy: "Region", Metrics: []Metric{{"Revenue", "sum"}}, Limit: 12})
	if !reflect.DeepEqual(res.Labels, []string{"North", "South", "East"}) {
		t.Fatalf("labels keep first-seen order: %v", res.Labels)
	}
	if !reflect.DeepEqual(res.Values[0], []float64{150, 200, 1000}) {
		t.Fatalf("sums: %v", res.Values[0])
	}
}

func TestRunQueryAggregations(t *testing.T) {
	ds, rows := testDS()
	cases := map[string]float64{"sum": 1350, "avg": 337.5, "min": 50, "max": 1000, "count": 4}
	for agg, want := range cases {
		res := runQuery(ds, rows, QuerySpec{Metrics: []Metric{{"Revenue", agg}}, Limit: 12})
		if len(res.Values[0]) != 1 || res.Values[0][0] != want {
			t.Errorf("%s: got %v want %v", agg, res.Values[0], want)
		}
	}
}

func TestRunQueryFiltersSortAndLimit(t *testing.T) {
	ds, rows := testDS()
	res := runQuery(ds, rows, QuerySpec{Metrics: []Metric{{"Revenue", "sum"}}, Filters: []Filter{{"Revenue", "gt", "99"}}, Limit: 12})
	if res.Values[0][0] != 1300 || res.RowsUsed != 3 {
		t.Fatalf("filter gt: %+v", res)
	}
	res = runQuery(ds, rows, QuerySpec{Metrics: []Metric{{"Revenue", "count"}}, Filters: []Filter{{"Region", "eq", "north"}}, Limit: 12})
	if res.Values[0][0] != 2 {
		t.Fatalf("eq is case-insensitive: %+v", res)
	}
	res = runQuery(ds, rows, QuerySpec{GroupBy: "Region", Metrics: []Metric{{"Revenue", "sum"}}, SortDesc: true, Limit: 2})
	if !reflect.DeepEqual(res.Labels, []string{"East", "South"}) || !res.Truncated || res.Groups != 3 {
		t.Fatalf("top 2: %+v", res)
	}
}

func TestValidateSpec(t *testing.T) {
	ds, _ := testDS()
	dss := []datasetInfo{*ds}
	good := QuerySpec{DatasetID: 999, GroupBy: "region", Metrics: []Metric{{"revenue", "SUM"}}, Chart: "PIE"}
	got, err := validateSpec(&good, dss)
	if err != nil || got.ID != 7 {
		t.Fatalf("single dataset should be assumed: %v %v", got, err)
	}
	if good.GroupBy != "Region" || good.Metrics[0].Column != "Revenue" || good.Metrics[0].Agg != "sum" || good.Chart != "pie" || good.Limit != 12 {
		t.Fatalf("names must be canonicalised: %+v", good)
	}
	bad := []QuerySpec{
		{DatasetID: 7, Metrics: []Metric{{"Nope", "sum"}}},
		{DatasetID: 7, Metrics: []Metric{{"Region", "sum"}}},
		{DatasetID: 7, GroupBy: "Missing", Metrics: []Metric{{"Revenue", "sum"}}},
		{DatasetID: 7, Metrics: []Metric{{"Revenue", "drop table"}}},
		{DatasetID: 7},
		{DatasetID: 7, Metrics: []Metric{{"Revenue", "sum"}}, Filters: []Filter{{"Revenue", "gt", "abc"}}},
		{DatasetID: 7, Metrics: []Metric{{"Revenue", "sum"}}, Filters: []Filter{{"Revenue", "like", "1"}}},
	}
	for i, spec := range bad {
		if _, err := validateSpec(&spec, dss); err == nil {
			t.Errorf("bad spec %d should be rejected: %+v", i, spec)
		}
	}
	two := []datasetInfo{*ds, {ID: 8, Columns: ds.Columns}}
	s := QuerySpec{DatasetID: 99, Metrics: []Metric{{"Revenue", "sum"}}}
	if _, err := validateSpec(&s, two); err == nil {
		t.Error("unknown dataset id with several datasets must be rejected")
	}
}

func TestResultBlocks(t *testing.T) {
	ds, rows := testDS()
	spec := QuerySpec{GroupBy: "Region", Metrics: []Metric{{"Revenue", "sum"}}, Chart: "bar", Limit: 12}
	b := resultBlocks(runQuery(ds, rows, spec), spec)
	if b.Table == nil || b.Chart == nil || b.Chart.Kind != "bar" || len(b.Chart.Labels) != 3 {
		t.Fatalf("expected table and bar chart: %+v", b)
	}
	if b.Table.Rows[2][1] != "1,000" {
		t.Fatalf("numbers are formatted: %v", b.Table.Rows)
	}
	total := QuerySpec{Metrics: []Metric{{"Revenue", "sum"}, {"", "count"}}, Limit: 12}
	b = resultBlocks(runQuery(ds, rows, total), total)
	if len(b.Stats) != 2 || b.Stats[0].Value != "1,350" || b.Table != nil {
		t.Fatalf("ungrouped result becomes stat cards: %+v", b)
	}
}

func TestFmtNum(t *testing.T) {
	for in, want := range map[float64]string{5: "5", 0.5: "0.5", 1234567.891: "1,234,567.89", -1234.5: "-1,234.5", 0: "0", 1000: "1,000"} {
		if got := fmtNum(in); got != want {
			t.Errorf("fmtNum(%v) = %q want %q", in, got, want)
		}
	}
}

func TestChartFromMarkdownTable(t *testing.T) {
	answer := "Here is the comparison:\n\n| Month | Revenue | Expenses | Net Profit |\n| --- | --- | --- | --- |\n| January | $420K | $300K | $120K |\n| February | $460K | $310K | $150K |\n| March | $510K | $330K | $180K |\n\nNote: data may be incomplete."
	ch := chartFromMarkdownTable(answer, "Show a chart of revenue by month")
	if ch == nil || len(ch.Labels) != 3 || len(ch.Series) != 3 || ch.Series[0].Values[0] != 420 || ch.Kind != "bar" {
		t.Fatalf("unexpected chart: %+v", ch)
	}
	source := "January: revenue was $420K, expenses were $300K, and net profit was $120K. February: revenue $460K, expenses $310K, net $150K. March: revenue $510K, expenses $330K, net $180K."
	if v := validateBlocks(Blocks{Chart: ch}, source, false); v.Chart == nil {
		t.Fatal("a chart built from a correct table must pass validation")
	}
	if v := validateBlocks(Blocks{Chart: ch}, "unrelated text 1 2 3", false); v.Chart != nil {
		t.Fatal("numbers missing from the source must be rejected")
	}
	if chartFromMarkdownTable("no table here", "chart") != nil {
		t.Fatal("no table means no chart")
	}
	if chartFromMarkdownTable("| A | B |\n| --- | --- |\n| x | text |\n| y | more |", "chart") != nil {
		t.Fatal("non-numeric columns must not become charts")
	}
}

func TestKindFromQuestion(t *testing.T) {
	cases := []struct {
		q      string
		series int
		want   string
	}{
		{"show a pie chart of revenue", 1, "pie"},
		{"show a pie chart of revenue", 2, "bar"},
		{"revenue trend over time", 1, "line"},
		{"make a line chart", 2, "line"},
		{"bar chart of revenue trend", 1, "bar"},
		{"chart revenue by month", 1, "bar"},
	}
	for _, c := range cases {
		if got := kindFromQuestion(c.q, c.series); got != c.want {
			t.Errorf("kindFromQuestion(%q,%d) = %q want %q", c.q, c.series, got, c.want)
		}
	}
}
