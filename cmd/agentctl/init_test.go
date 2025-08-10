package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitProjectVariants(t *testing.T) {
	dir := t.TempDir()
	// minimal
	if err := initProject(filepath.Join(dir, "min"), "minimal"); err != nil {
		t.Fatalf("minimal: %v", err)
	}
	// basic
	if err := initProject(filepath.Join(dir, "basic"), "basic"); err != nil {
		t.Fatalf("basic: %v", err)
	}
	// rag
	if err := initProject(filepath.Join(dir, "rag"), "rag"); err != nil {
		t.Fatalf("rag: %v", err)
	}
	// multi-agent
	if err := initProject(filepath.Join(dir, "multi"), "multi-agent"); err != nil {
		t.Fatalf("multi: %v", err)
	}
	// unknown
	if err := initProject(filepath.Join(dir, "bad"), "nope"); err == nil {
		t.Fatalf("expected error for unknown type")
	}
	// ensure files written
	if _, err := os.Stat(filepath.Join(dir, "min", "main.go")); err != nil {
		t.Fatalf("file missing: %v", err)
	}
}
