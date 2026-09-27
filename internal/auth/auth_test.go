package auth

import (
	"strings"
	"testing"
)

func TestNewToken(t *testing.T) {
	a, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 64 {
		t.Errorf("token len = %d, want 64 hex chars", len(a))
	}
	if a == b {
		t.Error("two tokens equal, want unique")
	}
	if !a.Check(string(a)) {
		t.Error("Check(own) = false, want true")
	}
	if a.Check(string(b)) {
		t.Error("Check(other) = true, want false")
	}
	if a.Check("") {
		t.Error("Check(empty) = true, want false")
	}
}

func TestBearer(t *testing.T) {
	if got := Bearer("abc", ""); got != "abc" {
		t.Errorf("query token = %q", got)
	}
	if got := Bearer("", "Bearer xyz"); got != "xyz" {
		t.Errorf("header token = %q", got)
	}
	if got := Bearer("abc", "Bearer xyz"); got != "abc" {
		t.Errorf("precedence = %q, want query", got)
	}
	if got := Bearer("", ""); got != "" {
		t.Errorf("empty = %q", got)
	}
	if got := Bearer("", "Basic xyz"); got != "" {
		t.Errorf("non-bearer = %q", got)
	}
}

func TestNewPIN(t *testing.T) {
	p, err := NewPIN()
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 6 || strings.Trim(p, "0123456789") != "" {
		t.Errorf("PIN = %q, want 6 digits", p)
	}
}
