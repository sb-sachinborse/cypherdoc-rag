package main

import (
	"net/url"
	"testing"
)

func TestDatabaseURLEscapesPassword(t *testing.T) {
	for _, pw := range []string{"plain", "p@ss:w/rd#1%2", "xyz@123", "a b?c=d&e"} {
		t.Setenv("DATABASE_URL", "")
		t.Setenv("POSTGRES_PASSWORD", pw)
		t.Setenv("DB_HOST", "db")
		u, err := url.Parse(databaseURL())
		if err != nil {
			t.Fatalf("password %q: URL does not parse: %v", pw, err)
		}
		if u.Hostname() != "db" {
			t.Fatalf("password %q: host became %q", pw, u.Hostname())
		}
		got, _ := u.User.Password()
		if got != pw {
			t.Fatalf("password round trip: got %q want %q", got, pw)
		}
	}
}

func TestDatabaseURLOverride(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@h:1/d")
	if databaseURL() != "postgres://u:p@h:1/d" {
		t.Fatal("DATABASE_URL must take precedence")
	}
}
