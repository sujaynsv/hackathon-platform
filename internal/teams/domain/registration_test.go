package domain

import (
	"errors"
	"testing"
	"time"
)

func TestCheckCanRegister_OpenEventWithinWindow(t *testing.T) {
	closesAt := time.Now().Add(time.Hour)
	if err := CheckCanRegister("registration_open", &closesAt); err != nil {
		t.Fatalf("expected registration to be allowed, got %v", err)
	}
}

func TestCheckCanRegister_EventNotOpen(t *testing.T) {
	err := CheckCanRegister("draft", nil)
	if !errors.Is(err, ErrEventNotOpen) {
		t.Fatalf("expected ErrEventNotOpen, got %v", err)
	}
}

func TestCheckCanRegister_PastDeadline(t *testing.T) {
	closesAt := time.Now().Add(-time.Hour)
	err := CheckCanRegister("registration_open", &closesAt)
	if !errors.Is(err, ErrDeadlinePassed) {
		t.Fatalf("expected ErrDeadlinePassed, got %v", err)
	}
}
