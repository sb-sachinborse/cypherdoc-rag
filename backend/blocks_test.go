package main

import "testing"

const sampleCtx = "Revenue was $1.62M in Q2, up from $1.39M. Net profit reached $0.56M versus $0.45M. Expenses 1,020 and 940."

func TestValidateBlocksDropsInventedNumbers(t *testing.T) {
	in := Blocks{
		Stats: []Stat{
			{Label: "Revenue", Value: "$1.62M", Delta: "+16.5%"},
			{Label: "Invented", Value: "$9.99M"},
			{Label: "", Value: "$0.56M"},
			{Label: "No number", Value: "high"},
		},
		Table: &Table{Columns: []string{"Metric", "Q1", "Q2"}, Rows: [][]string{
			{"Revenue", "$1.39M", "$1.62M"},
			{"Fake", "$7M", "$8M"},
			{"Short", "$1.39M"},
		}},
		Chart: &Chart{Title: "x", Labels: []string{"Q1", "Q2"}, Series: []Series{{Name: "Revenue", Values: []float64{1.39, 1.62}}}},
	}
	out := validateBlocks(in, sampleCtx)
	if len(out.Stats) != 1 || out.Stats[0].Value != "$1.62M" {
		t.Fatalf("stats: %+v", out.Stats)
	}
	if out.Stats[0].Delta != "" {
		t.Errorf("computed delta not in source must be dropped, got %q", out.Stats[0].Delta)
	}
	if out.Table == nil || len(out.Table.Rows) != 1 || out.Table.Rows[0][0] != "Revenue" {
		t.Fatalf("table: %+v", out.Table)
	}
	if out.Chart == nil || len(out.Chart.Series[0].Values) != 2 {
		t.Fatalf("chart: %+v", out.Chart)
	}
}

func TestValidateBlocksRejectsBadChart(t *testing.T) {
	in := Blocks{Chart: &Chart{Labels: []string{"Q1", "Q2"}, Series: []Series{{Name: "A", Values: []float64{1.39, 42}}}}}
	if out := validateBlocks(in, sampleCtx); out.Chart != nil {
		t.Fatalf("chart with an unseen value must be dropped: %+v", out.Chart)
	}
	in = Blocks{Chart: &Chart{Labels: []string{"Q1", "Q2"}, Series: []Series{{Name: "A", Values: []float64{1.39}}}}}
	if out := validateBlocks(in, sampleCtx); out.Chart != nil {
		t.Fatal("series length mismatch must be dropped")
	}
}

func TestNumberNormalisation(t *testing.T) {
	set := numberSet("1,020 and 16.50")
	if !set["1020"] || !set["16.5"] {
		t.Fatalf("unexpected set %v", set)
	}
}

func TestParseBlocks(t *testing.T) {
	if _, ok := parseBlocks("not json"); ok {
		t.Fatal("invalid JSON must fail")
	}
	b, ok := parseBlocks(`{"stats":[{"label":"A","value":"1"}]}`)
	if !ok || len(b.Stats) != 1 {
		t.Fatalf("got %+v %v", b, ok)
	}
}
