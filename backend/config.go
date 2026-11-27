package main

import (
	"net/url"
	"os"
	"strconv"
)

const embedDim = 768

type Config struct {
	Port           string
	DatabaseURL    string
	OllamaURL      string
	ChatModel      string
	EmbedModel     string
	CookieSecure   bool
	AllowSignup    bool
	RichBlocks     bool
	MaxUploadBytes int64
	MaxDistance    float64
	TopK           int
	MaxDocsPerUser int
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envFloat(key string, def float64) float64 {
	if f, err := strconv.ParseFloat(os.Getenv(key), 64); err == nil {
		return f
	}
	return def
}

func envInt(key string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil && n > 0 {
		return n
	}
	return def
}

// databaseURL builds the connection string from separate parts so that special
// characters in the password (@ : / # % ...) are escaped correctly.
// DATABASE_URL, if set, wins.
func databaseURL() string {
	if u := os.Getenv("DATABASE_URL"); u != "" {
		return u
	}
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(env("DB_USER", "cypherdoc"), env("POSTGRES_PASSWORD", "change-me-local-only")),
		Host:     env("DB_HOST", "db") + ":" + env("DB_PORT", "5432"),
		Path:     "/" + env("DB_NAME", "cypherdoc"),
		RawQuery: "sslmode=disable",
	}
	return u.String()
}

func loadConfig() Config {
	return Config{
		Port:           env("PORT", "8080"),
		DatabaseURL:    databaseURL(),
		OllamaURL:      env("OLLAMA_URL", "http://ollama:11434"),
		ChatModel:      env("CHAT_MODEL", "llama3.2:3b"),
		EmbedModel:     env("EMBED_MODEL", "nomic-embed-text"),
		CookieSecure:   envBool("COOKIE_SECURE", false),
		AllowSignup:    envBool("ALLOW_SIGNUP", true),
		RichBlocks:     envBool("RICH_BLOCKS", true),
		MaxUploadBytes: int64(envInt("MAX_UPLOAD_MB", 10)) << 20,
		MaxDistance:    envFloat("MAX_DISTANCE", 0.75),
		TopK:           envInt("TOP_K", 8),
		MaxDocsPerUser: envInt("MAX_DOCS_PER_USER", 100),
	}
}
