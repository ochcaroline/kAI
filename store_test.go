package main

import (
	"errors"
	"path/filepath"
	"testing"
)

func ptr(s string) *string { return &s }

func TestStore(t *testing.T) {
	s, err := NewStore(filepath.Join(t.TempDir(), "b.json"))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := s.Create(Patch{Title: ptr("Fix login bug"), Priority: ptr("high")})
	s.Create(Patch{Title: ptr("Fix logout bug")})
	if _, err := s.Create(Patch{Title: ptr(" ")}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
	if got, err := s.Get("login"); err != nil || got.ID != a.ID {
		t.Fatalf("text ref: %v %v", got, err)
	}
	if _, err := s.Get("fix"); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("want ErrAmbiguous, got %v", err)
	}
	if got, err := s.Update("#1", Patch{Status: ptr("done"), Plan: ptr("p")}); err != nil || got.Status != "done" || got.Plan != "p" {
		t.Fatalf("update: %v %v", got, err)
	}
	if _, err := s.Update("1", Patch{Status: ptr("nope")}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
	if err := s.Delete("1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
