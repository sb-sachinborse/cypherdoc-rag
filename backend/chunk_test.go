package main

import (
	"strings"
	"testing"
)

func TestChunkPagesWindowsAndOverlap(t *testing.T) {
	words := make([]string, 450)
	for i := range words {
		words[i] = "w"
	}
	chunks := chunkPages([]Page{{Num: 3, Text: strings.Join(words, " ")}}, 200, 40)
	if len(chunks) != 3 {
		t.Fatalf("want 3 chunks, got %d", len(chunks))
	}
	for _, c := range chunks {
		if c.Page != 3 {
			t.Fatalf("page not preserved: %d", c.Page)
		}
		if n := len(strings.Fields(c.Text)); n > 200 {
			t.Fatalf("chunk too large: %d", n)
		}
	}
}

func TestChunkPagesSmallAndEmpty(t *testing.T) {
	if got := chunkPages([]Page{{Num: 1, Text: "   "}}, 200, 40); len(got) != 0 {
		t.Fatalf("blank page should yield no chunks, got %d", len(got))
	}
	got := chunkPages([]Page{{Num: 1, Text: "hello world"}}, 200, 40)
	if len(got) != 1 || got[0].Text != "hello world" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestChunkPagesBadOverlapDoesNotLoopForever(t *testing.T) {
	text := strings.Repeat("a ", 50)
	got := chunkPages([]Page{{Num: 1, Text: text}}, 10, 10)
	if len(got) == 0 || len(got) > 50 {
		t.Fatalf("unexpected chunk count %d", len(got))
	}
}
