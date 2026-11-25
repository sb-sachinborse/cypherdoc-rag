package main

import "strings"

type Page struct {
	Num  int
	Text string
}

type Chunk struct {
	Page int
	Text string
}

// chunkPages splits text into overlapping word windows, keeping page numbers.
func chunkPages(pages []Page, size, overlap int) []Chunk {
	if size <= 0 {
		size = 200
	}
	if overlap < 0 || overlap >= size {
		overlap = size / 5
	}
	var out []Chunk
	for _, p := range pages {
		words := strings.Fields(p.Text)
		for start := 0; start < len(words); {
			end := start + size
			if end > len(words) {
				end = len(words)
			}
			out = append(out, Chunk{Page: p.Num, Text: strings.Join(words[start:end], " ")})
			if end == len(words) {
				break
			}
			start = end - overlap
		}
	}
	return out
}
