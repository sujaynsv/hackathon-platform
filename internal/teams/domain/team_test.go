package domain

import (
	"regexp"
	"testing"
)

func TestGenerateInviteCode_Returns8CharUpperAlphanumeric(t *testing.T) {
	code, err := GenerateInviteCode()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(code) != 8 {
		t.Errorf("expected length 8, got %d", len(code))
	}

	matched, err := regexp.MatchString(`^[A-Z0-9]{8}$`, code)
	if err != nil || !matched {
		t.Errorf("expected 8 char uppercase alphanumeric, got %q", code)
	}
}

func TestGenerateInviteCode_IsUniqueBetweenCalls(t *testing.T) {
	codes := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		code, err := GenerateInviteCode()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if codes[code] {
			t.Fatalf("collision detected on iteration %d: %s", i, code)
		}
		codes[code] = true
	}
}
