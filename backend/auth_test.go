package main

import (
	"bytes"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	good := map[string]string{"A@Example.com ": "a@example.com", "user@company.io": "user@company.io"}
	for in, want := range good {
		got, ok := validateEmail(in)
		if !ok || got != want {
			t.Errorf("validateEmail(%q) = %q,%v want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "nope", "Name <a@b.co>", "a@b.co, c@d.co", "@x.com"} {
		if _, ok := validateEmail(in); ok {
			t.Errorf("validateEmail(%q) should fail", in)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if validatePassword("short") == "" {
		t.Error("short password accepted")
	}
	if validatePassword("this-is-long-enough") != "" {
		t.Error("good password rejected")
	}
	long := make([]byte, 73)
	for i := range long {
		long[i] = 'a'
	}
	if validatePassword(string(long)) == "" {
		t.Error("73-byte password accepted (bcrypt limit is 72)")
	}
}

func TestTokens(t *testing.T) {
	a, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newToken()
	if a == b || len(a) < 40 {
		t.Fatalf("tokens must be unique and long: %q %q", a, b)
	}
	if !bytes.Equal(hashToken(a), hashToken(a)) || bytes.Equal(hashToken(a), hashToken(b)) {
		t.Fatal("hashToken must be deterministic and collision-free for different input")
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("héllo wörld", 5); got != "héllo" {
		t.Fatalf("got %q", got)
	}
	if got := truncateRunes("abc", 10); got != "abc" {
		t.Fatalf("got %q", got)
	}
}
