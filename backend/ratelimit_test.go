package main

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	l := newLimiter(3, 50*time.Millisecond)
	for i := 0; i < 3; i++ {
		if !l.allow("k") {
			t.Fatalf("request %d should pass", i)
		}
	}
	if l.allow("k") {
		t.Fatal("4th request should be blocked")
	}
	if !l.allow("other") {
		t.Fatal("other keys are independent")
	}
	time.Sleep(60 * time.Millisecond)
	if !l.allow("k") {
		t.Fatal("window should have expired")
	}
}
