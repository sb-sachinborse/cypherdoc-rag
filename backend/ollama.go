package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Msg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Ollama struct {
	base, chat, embed string
	hc                *http.Client
}

func newOllama(base, chat, embed string) *Ollama {
	return &Ollama{base: strings.TrimRight(base, "/"), chat: chat, embed: embed, hc: &http.Client{}}
}

func (o *Ollama) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, o.base+path, rd)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := o.hc.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		return nil, fmt.Errorf("ollama %s: %d %s", path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return resp, nil
}

func normModel(n string) string { return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(n)), ":latest") }

// Models returns the set of normalised model names installed in Ollama.
func (o *Ollama) Models(ctx context.Context) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := o.do(ctx, http.MethodGet, "/api/tags", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(out.Models))
	for _, m := range out.Models {
		set[normModel(m.Name)] = true
	}
	return set, nil
}

func (s *Server) modelsReady(ctx context.Context) (chatOK, embedOK bool) {
	set, err := s.ol.Models(ctx)
	if err != nil {
		return false, false
	}
	return set[normModel(s.ol.chat)], set[normModel(s.ol.embed)]
}

// Embed returns one 768-dim vector per input.
func (o *Ollama) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	resp, err := o.do(ctx, http.MethodPost, "/api/embed", map[string]any{"model": o.embed, "input": inputs})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<20)).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embeddings) != len(inputs) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(inputs), len(out.Embeddings))
	}
	for _, e := range out.Embeddings {
		if len(e) != embedDim {
			return nil, fmt.Errorf("embedding has %d dimensions, schema expects %d", len(e), embedDim)
		}
	}
	return out.Embeddings, nil
}

// vecString formats a vector as a pgvector literal like [0.1,0.2].
func vecString(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

// ChatStream streams tokens from the chat model to onToken.
func (o *Ollama) ChatStream(ctx context.Context, msgs []Msg, onToken func(string) error) error {
	resp, err := o.do(ctx, http.MethodPost, "/api/chat", map[string]any{
		"model": o.chat, "messages": msgs, "stream": true,
		"options": map[string]any{"temperature": 0.2, "num_ctx": 4096},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		var ev struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done  bool   `json:"done"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Error != "" {
			return errors.New(ev.Error)
		}
		if ev.Message.Content != "" {
			if err := onToken(ev.Message.Content); err != nil {
				return err
			}
		}
		if ev.Done {
			return nil
		}
	}
	return sc.Err()
}

// ChatJSON asks for a single JSON object that follows schema.
func (o *Ollama) ChatJSON(ctx context.Context, msgs []Msg, schema any) (string, error) {
	resp, err := o.do(ctx, http.MethodPost, "/api/chat", map[string]any{
		"model": o.chat, "messages": msgs, "stream": false, "format": schema,
		"options": map[string]any{"temperature": 0, "num_ctx": 4096},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return "", err
	}
	return out.Message.Content, nil
}
