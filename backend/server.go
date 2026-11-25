package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	cfg      Config
	db       *pgxpool.Pool
	ol       *Ollama
	log      *slog.Logger
	jobs     chan ingestJob
	authRL   *limiter
	chatRL   *limiter
	uploadRL *limiter
}

func newServer(cfg Config, db *pgxpool.Pool, log *slog.Logger) *Server {
	return &Server{
		cfg:      cfg,
		db:       db,
		ol:       newOllama(cfg.OllamaURL, cfg.ChatModel, cfg.EmbedModel),
		log:      log,
		jobs:     make(chan ingestJob, 32),
		authRL:   newLimiter(10, time.Minute),
		chatRL:   newLimiter(20, time.Minute),
		uploadRL: newLimiter(20, 10*time.Minute),
	}
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.auth(s.handleMe))
	mux.HandleFunc("GET /api/documents", s.auth(s.handleListDocs))
	mux.HandleFunc("POST /api/documents", s.auth(s.handleUpload))
	mux.HandleFunc("DELETE /api/documents/{id}", s.auth(s.handleDeleteDoc))
	mux.HandleFunc("POST /api/chat", s.auth(s.handleChat))
	return chain(mux, logMW(s.log), recoverMW(s.log), securityHeaders, csrf)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return false
	}
	return true
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	chatOK, embedOK := s.modelsReady(r.Context())
	writeJSON(w, http.StatusOK, map[string]bool{
		"chat_model":  chatOK,
		"embed_model": embedOK,
		"ready":       chatOK && embedOK,
		"signup":      s.cfg.AllowSignup,
	})
}

func runHealthcheck(cfg Config) int {
	c := http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://127.0.0.1:" + cfg.Port + "/api/health")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

