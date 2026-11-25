package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

const (
	cookieName = "cd_session"
	sessionTTL = 72 * time.Hour
	bcryptCost = 12
)

type ctxKey int

const userCtxKey ctxKey = 1

type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// dummyHash makes unknown-email logins cost the same as real ones.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcryptCost)

func validateEmail(email string) (string, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) < 3 || len(email) > 254 {
		return "", false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", false
	}
	return email, true
}

// validatePassword returns an error message, or "" when the password is acceptable.
func validatePassword(p string) string {
	if len(p) < 10 {
		return "Password must be at least 10 characters."
	}
	if len(p) > 72 {
		return "Password must be at most 72 bytes."
	}
	return ""
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(t string) []byte {
	sum := sha256.Sum256([]byte(t))
	return sum[:]
}

func (s *Server) startSession(ctx context.Context, w http.ResponseWriter, userID int64) error {
	token, err := newToken()
	if err != nil {
		return err
	}
	exp := time.Now().Add(sessionTTL)
	if _, err := s.db.Exec(ctx, `INSERT INTO sessions(token_hash, user_id, expires_at) VALUES($1,$2,$3)`, hashToken(token), userID, exp); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: token, Path: "/", Expires: exp, MaxAge: int(sessionTTL.Seconds()),
		HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteStrictMode,
	})
	return nil
}

func (s *Server) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(0, 0),
		HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteStrictMode,
	})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.AllowSignup {
		writeError(w, http.StatusForbidden, "Sign-ups are disabled.")
		return
	}
	if !s.authRL.allow("auth:" + clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "Too many attempts. Try again in a minute.")
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	email, ok := validateEmail(in.Email)
	if !ok {
		writeError(w, http.StatusBadRequest, "Enter a valid email address.")
		return
	}
	if msg := validatePassword(in.Password); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	name := truncateRunes(strings.TrimSpace(in.Name), 60)
	if name == "" {
		name = strings.SplitN(email, "@", 2)[0]
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create account.")
		return
	}
	var id int64
	err = s.db.QueryRow(r.Context(),
		`INSERT INTO users(email, name, password_hash) VALUES($1,$2,$3) RETURNING id`,
		email, name, string(hash)).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		writeError(w, http.StatusConflict, "That email is already registered.")
		return
	}
	if err != nil {
		s.log.Error("register", "err", err)
		writeError(w, http.StatusInternalServerError, "Could not create account.")
		return
	}
	if err := s.startSession(r.Context(), w, id); err != nil {
		s.log.Error("session", "err", err)
		writeError(w, http.StatusInternalServerError, "Could not start session.")
		return
	}
	writeJSON(w, http.StatusCreated, User{ID: id, Email: email, Name: name})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.authRL.allow("auth:" + clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "Too many attempts. Try again in a minute.")
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	var u User
	var hash string
	err := s.db.QueryRow(r.Context(), `SELECT id, email, name, password_hash FROM users WHERE email=$1`, email).
		Scan(&u.ID, &u.Email, &u.Name, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(in.Password))
		writeError(w, http.StatusUnauthorized, "Invalid email or password.")
		return
	}
	if err != nil {
		s.log.Error("login", "err", err)
		writeError(w, http.StatusInternalServerError, "Could not log in.")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "Invalid email or password.")
		return
	}
	if err := s.startSession(r.Context(), w, u.ID); err != nil {
		s.log.Error("session", "err", err)
		writeError(w, http.StatusInternalServerError, "Could not start session.")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		_, _ = s.db.Exec(r.Context(), `DELETE FROM sessions WHERE token_hash=$1`, hashToken(c.Value))
	}
	s.clearCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, userFrom(r))
}

func userFrom(r *http.Request) User {
	u, _ := r.Context().Value(userCtxKey).(User)
	return u
}

// auth wraps a handler so it only runs for a valid session.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil || c.Value == "" {
			writeError(w, http.StatusUnauthorized, "Please log in.")
			return
		}
		var u User
		err = s.db.QueryRow(r.Context(),
			`SELECT u.id, u.email, u.name FROM sessions s JOIN users u ON u.id = s.user_id
			 WHERE s.token_hash=$1 AND s.expires_at > now()`, hashToken(c.Value)).Scan(&u.ID, &u.Email, &u.Name)
		if err != nil {
			s.clearCookie(w)
			writeError(w, http.StatusUnauthorized, "Please log in.")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userCtxKey, u)))
	}
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func (s *Server) cleanupLoop(ctx context.Context) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_, _ = s.db.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
		}
	}
}
