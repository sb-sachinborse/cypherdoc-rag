package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ingestJob struct {
	DocID, UserID int64
	Ext           string
	Data          []byte
}

// userError carries a message that is safe to show in the UI.
type userError struct {
	msg string
	err error
}

func (e *userError) Error() string { return e.msg }
func (e *userError) Unwrap() error { return e.err }

var collectionRe = regexp.MustCompile(`^[\p{L}\p{N} _.&-]+$`)

// sanitizeCollection returns a safe collection name or "" when invalid/empty.
func sanitizeCollection(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || len([]rune(s)) > 60 || !collectionRe.MatchString(s) {
		return ""
	}
	return s
}

type docOut struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Collection string    `json:"collection"`
	Size       int64     `json:"size"`
	Status     string    `json:"status"`
	Error      string    `json:"error"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Server) handleListDocs(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	rows, err := s.db.Query(r.Context(),
		`SELECT id, name, collection, size_bytes, status, error, created_at
		 FROM documents WHERE user_id=$1 ORDER BY created_at DESC LIMIT 500`, u.ID)
	if err != nil {
		s.log.Error("list docs", "err", err)
		writeError(w, http.StatusInternalServerError, "Could not load documents.")
		return
	}
	defer rows.Close()
	docs := []docOut{}
	for rows.Next() {
		var d docOut
		if err := rows.Scan(&d.ID, &d.Name, &d.Collection, &d.Size, &d.Status, &d.Error, &d.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load documents.")
			return
		}
		docs = append(docs, d)
	}
	writeJSON(w, http.StatusOK, docs)
}

func (s *Server) handleDeleteDoc(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id.")
		return
	}
	tag, err := s.db.Exec(r.Context(), `DELETE FROM documents WHERE id=$1 AND user_id=$2`, id, userFrom(r).ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not delete.")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Document not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleUpload streams the multipart body (no temp files) and queues ingestion.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	if !s.uploadRL.allow("upload:" + strconv.FormatInt(u.ID, 10)) {
		writeError(w, http.StatusTooManyRequests, "Too many uploads. Try again later.")
		return
	}
	var count int
	if err := s.db.QueryRow(r.Context(), `SELECT count(*) FROM documents WHERE user_id=$1`, u.ID).Scan(&count); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not upload.")
		return
	}
	if count >= s.cfg.MaxDocsPerUser {
		writeError(w, http.StatusConflict, "Document limit reached. Delete some documents first.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes+(1<<20))
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "Expected a multipart upload.")
		return
	}
	var (
		data       []byte
		filename   string
		collection = "General"
		gotFile    bool
	)
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "Upload failed or file is too large.")
			return
		}
		switch part.FormName() {
		case "collection":
			b, _ := io.ReadAll(io.LimitReader(part, 256))
			if c := sanitizeCollection(string(b)); c != "" {
				collection = c
			}
		case "file":
			filename = cleanFilename(part.FileName())
			data, err = io.ReadAll(io.LimitReader(part, s.cfg.MaxUploadBytes+1))
			if err != nil {
				writeError(w, http.StatusBadRequest, "Upload failed or file is too large.")
				return
			}
			gotFile = true
		}
		part.Close()
	}
	if !gotFile {
		writeError(w, http.StatusBadRequest, "No file provided.")
		return
	}
	if int64(len(data)) > s.cfg.MaxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "File is larger than the limit.")
		return
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedExt[ext] {
		writeError(w, http.StatusUnsupportedMediaType, "Unsupported file type. Allowed: "+allowedExtList+".")
		return
	}
	if err := validateContent(ext, data); err != nil {
		writeError(w, http.StatusUnsupportedMediaType, "The file content does not match its type.")
		return
	}

	var d docOut
	err = s.db.QueryRow(r.Context(),
		`INSERT INTO documents(user_id, name, collection, size_bytes) VALUES($1,$2,$3,$4)
		 RETURNING id, name, collection, size_bytes, status, error, created_at`,
		u.ID, filename, collection, int64(len(data))).
		Scan(&d.ID, &d.Name, &d.Collection, &d.Size, &d.Status, &d.Error, &d.CreatedAt)
	if err != nil {
		s.log.Error("insert doc", "err", err)
		writeError(w, http.StatusInternalServerError, "Could not upload.")
		return
	}
	select {
	case s.jobs <- ingestJob{DocID: d.ID, UserID: u.ID, Ext: ext, Data: data}:
		writeJSON(w, http.StatusAccepted, d)
	default:
		_, _ = s.db.Exec(r.Context(), `DELETE FROM documents WHERE id=$1`, d.ID)
		writeError(w, http.StatusServiceUnavailable, "Server is busy. Try again shortly.")
	}
}

func (s *Server) startWorkers(ctx context.Context) {
	_, _ = s.db.Exec(ctx, `UPDATE documents SET status='failed', error='Interrupted by a restart. Please upload again.' WHERE status='processing'`)
	for i := 0; i < 2; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case j := <-s.jobs:
					s.processJob(ctx, j)
				}
			}
		}()
	}
	go s.cleanupLoop(ctx)
}

func (s *Server) processJob(ctx context.Context, j ingestJob) {
	jctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	err := s.ingest(jctx, j)

	uctx, ucancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer ucancel()
	if err != nil {
		s.log.Error("ingest failed", "doc", j.DocID, "err", err)
		msg := "Processing failed. Please try again."
		var ue *userError
		if errors.As(err, &ue) {
			msg = ue.msg
		}
		_, _ = s.db.Exec(uctx, `UPDATE documents SET status='failed', error=$2 WHERE id=$1`, j.DocID, msg)
		return
	}
	_, _ = s.db.Exec(uctx, `UPDATE documents SET status='indexed', error='' WHERE id=$1`, j.DocID)
}

// waitForModels blocks until Ollama has both models (first run downloads them).
func (s *Server) waitForModels(ctx context.Context) error {
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		if chatOK, embedOK := s.modelsReady(ctx); chatOK && embedOK {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
	return errors.New("models not ready in time")
}

func (s *Server) ingest(ctx context.Context, j ingestJob) error {
	var pages []Page
	var sheets []Sheet
	var err error
	if isSheetExt(j.Ext) {
		sheets, err = parseSpreadsheet(j.Ext, j.Data)
		if err == nil {
			pages = sheetPages(sheets)
		}
	} else {
		pages, err = extractText(j.Ext, j.Data)
	}
	if err != nil {
		return &userError{"Could not read this file.", err}
	}
	chunks := chunkPages(pages, 200, 40)
	if len(chunks) == 0 {
		return &userError{msg: "No readable text found. Scanned PDFs are not supported."}
	}
	if err := s.waitForModels(ctx); err != nil {
		return &userError{"The AI models are not ready yet. Please upload again later.", err}
	}

	vectors := make([][]float32, 0, len(chunks))
	const batch = 16
	for i := 0; i < len(chunks); i += batch {
		end := i + batch
		if end > len(chunks) {
			end = len(chunks)
		}
		inputs := make([]string, 0, end-i)
		for _, c := range chunks[i:end] {
			inputs = append(inputs, "search_document: "+c.Text)
		}
		vs, err := s.ol.Embed(ctx, inputs)
		if err != nil {
			return &userError{"Embedding failed. Please try again.", err}
		}
		vectors = append(vectors, vs...)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	for i, c := range chunks {
		if _, err := tx.Exec(ctx,
			`INSERT INTO chunks(document_id, user_id, page, idx, content, embedding) VALUES($1,$2,$3,$4,$5,$6::text::vector)`,
			j.DocID, j.UserID, c.Page, i, c.Text, vecString(vectors[i])); err != nil {
			return err
		}
	}
	for _, sh := range sheets {
		colsJSON, _ := json.Marshal(sh.Columns)
		bodyJSON, _ := json.Marshal(sh.Rows)
		sample := sh.Rows
		if len(sample) > 3 {
			sample = sample[:3]
		}
		sampleJSON, _ := json.Marshal(sample)
		if _, err := tx.Exec(ctx,
			`INSERT INTO datasets(document_id, user_id, sheet_name, cols, n_rows, sample, body)
			 VALUES($1,$2,$3,$4::text::jsonb,$5,$6::text::jsonb,$7::text::jsonb)`,
			j.DocID, j.UserID, sh.Name, string(colsJSON), len(sh.Rows), string(sampleJSON), string(bodyJSON)); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
