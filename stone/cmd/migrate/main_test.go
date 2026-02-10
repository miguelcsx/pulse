package main

import (
	"testing"

	"github.com/pulse/stone/internal/db"
)

func TestMigrationsEmbed(t *testing.T) {
	entries, err := db.Migrations.ReadDir("migrations")
	if err != nil {
		t.Fatalf("failed to read embedded migrations: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one embedded migration file")
	}

	hasUp := false
	for _, e := range entries {
		if !e.IsDir() {
			hasUp = true
			break
		}
	}
	if !hasUp {
		t.Fatal("no SQL files found in embedded migrations")
	}
}
